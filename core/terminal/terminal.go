package terminal

import (
	"fmt"
	"time"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
)

// Terminal is the main controller of the TUI engine.
// It coordinates double buffering (front/back buffer), screen size changes,
// the synchronised update protocol (?2026) and routing mouse events to the right targets.
type Terminal struct {
	// driver is the layer that handles low-level TTY raw mode and I/O.
	driver *driver.Driver

	// cast records the frames as asciicast while it is set (RecordCast,
	// LIMONI_CAST); nil otherwise, which costs the draw path one check.
	cast *castRecorder

	// front is the active buffer written in the current frame.
	front *buffer.Buffer

	// back holds the cells currently on screen (used for the diff).
	back *buffer.Buffer

	// inline, when greater than zero, draws the application in a band of that many
	// rows on the normal screen instead of the alternate screen.
	inline uint16

	// frame is the context handed to widgets during drawing, for drawing and registering click areas.
	frame *Frame

	// writeBuf is the byte slice reused every frame so that the diff's ANSI escape codes
	// can be written without heap allocation.
	writeBuf []byte

	// lastImageCount is the number of images drawn in the previous frame.
	lastImageCount int

	// lastDrawnImages is the list of images drawn in the previous frame.
	lastDrawnImages []ImageRegion

	// Dither transition state
	transitionActive   bool
	transitionProgress float64
	transitionOldBuf   *buffer.Buffer

	// Debug (layout inspector) state
	debugMode bool

	// mouseCaptureHandler is the active mouse drag (capture) handler.
	mouseCaptureHandler func(ev driver.MouseEvent)

	// lastLayersHash summarises the state of the previous frame's layers (modals/layers).
	lastLayersHash string

	// Profiling metrics
	lastFrameDuration time.Duration
	lastWidgetStats   []WidgetStat

	// Terminal capabilities: caps is what Draw uses; detected is the
	// environment's guess that the handshake refines into caps.
	caps     CapabilityProfile
	detected CapabilityProfile
	// capsPinned is set by SetCapabilities: the application's word beats the
	// handshake.
	capsPinned bool
	// reportVersion is the TerminalReport counter last folded into caps.
	reportVersion uint64
	// drawn is set once a frame has been written to the terminal.
	drawn bool
	// pointerShape is the OSC 22 shape last written; empty is the default.
	pointerShape string
	// kittyPushed is set while the kitty keyboard flags pushed by
	// syncKeyboardMode are on the terminal's stack.
	kittyPushed bool
	// keyReleases asks for the kitty protocol's event types as well, so key
	// repeats and releases are reported (SetKeyReleases).
	keyReleases bool

	// backdrop is the scene drawn behind the application (SetBackdrop).
	backdrop Backdrop
	// bgStart is when the backdrop was set: its clock's zero.
	bgStart time.Time
	// bgClock replaces time.Now for the backdrop in tests.
	bgClock func() time.Time
	// bgOff is set when the user turned backdrops off (LIMONI_BACKDROP).
	bgOff bool
	// bgLayer holds the scene, appLayer the cells the application drew in its
	// last frame, so DrawBackdrop can compose a frame without it.
	bgLayer, appLayer *buffer.Buffer
}

// New returns a Terminal that uses the given Backend, and allocates the first buffers.
func New(b *driver.Backend) (*Terminal, error) {
	// Read the terminal's initial rows and columns
	w, h, err := b.Size()
	if err != nil {
		return nil, err
	}
	if w == 0 || h == 0 {
		w, h = 80, 24
	}

	area := cell.NewRect(0, 0, w, h)
	front := buffer.NewBuffer(area)
	back := buffer.NewBuffer(area)

	focusMgr := NewFocusManager()

	detected := DetectCapabilities()
	t := &Terminal{
		driver:   b,
		front:    front,
		back:     back,
		frame:    NewFrame(front, focusMgr),
		writeBuf: make([]byte, 0, 8192), // Allocate an 8 KB write buffer up front
		caps:     detected,
		detected: detected,
		bgOff:    backdropOff(),
	}
	if err := t.recordCastFromEnv(); err != nil {
		return nil, err
	}
	return t, nil
}

// RestoreModes undoes what the application changed on the terminal beyond
// the screen: the pointer shape and the kitty keyboard flags. Close and
// Suspend call it; so does anything that hands the terminal back without them.
func (t *Terminal) RestoreModes() {
	if t == nil || t.driver == nil {
		return
	}
	t.ResetPointerShape()
	if t.kittyPushed {
		_, _ = t.driver.Write([]byte("\x1b[<u"))
		t.kittyPushed = false
	}
}

// syncKeyboardMode pushes the kitty keyboard flags once the profile says the
// terminal has the protocol. Pushing, not setting, is what lets RestoreModes
// give back exactly the flags the shell had.
func (t *Terminal) syncKeyboardMode() {
	if t.caps.KittyKeyboard && !t.kittyPushed && t.driver != nil {
		if t.keyReleases {
			_, _ = t.driver.Write([]byte("\x1b[>3u"))
		} else {
			_, _ = t.driver.Write([]byte("\x1b[>1u"))
		}
		t.kittyPushed = true
	}
}

// SetKeyReleases asks the terminal to report key repeats and releases as
// well as presses: KeyEvent.Repeat and KeyEvent.Release. Games want this —
// a key can be held down for as long as it is, rather than guessed at from
// auto-repeat. It takes effect in terminals with the kitty keyboard protocol
// (kitty, Ghostty, WezTerm, foot, recent Konsole); in the rest nothing
// changes and every key event stays a press.
//
// An application that turns it on sees every key twice, and must ignore
// releases wherever it acts on presses — including widgets it hands key
// events to, which do not tell the two apart.
func (t *Terminal) SetKeyReleases(on bool) {
	if t == nil || t.keyReleases == on {
		return
	}
	t.keyReleases = on
	if t.kittyPushed && t.driver != nil {
		// Our entry is on top of the terminal's stack: change it in place.
		if on {
			_, _ = t.driver.Write([]byte("\x1b[=3;1u"))
		} else {
			_, _ = t.driver.Write([]byte("\x1b[=1;1u"))
		}
	}
}

// Close restores the terminal state and closes the underlying driver.
func (t *Terminal) Close() error {
	var castErr error
	if t.cast != nil {
		castErr = t.cast.close()
		t.cast = nil
	}
	if t.driver != nil {
		t.RestoreModes()
		if err := t.driver.Close(); err != nil {
			return err
		}
	}
	return castErr
}

// Driver returns the underlying driver instance.
func (t *Terminal) Driver() *driver.Driver {
	return t.driver
}

// Backend returns the underlying driver instance (backward compatibility alias).
func (t *Terminal) Backend() *driver.Driver {
	return t.driver
}

// Events returns the channel of incoming events from the driver.
func (t *Terminal) Events() <-chan driver.Event {
	if t.driver == nil {
		return nil
	}
	return t.driver.Events()
}

// StartEventLoop starts the underlying driver event loop if not already started.
func (t *Terminal) StartEventLoop() {
	if t.driver != nil {
		t.driver.StartEventLoop()
	}
}

// PollEvent waits for and returns the next event from the driver.
func (t *Terminal) PollEvent() driver.Event {
	if t.driver == nil {
		return driver.Event{}
	}
	return <-t.driver.Events()
}

// LastFrameDuration returns the rendering and draw duration of the last frame.
func (t *Terminal) LastFrameDuration() time.Duration {
	return t.lastFrameDuration
}

// LastWidgetStats returns the individual render durations of all widgets rendered in the last frame.
func (t *Terminal) LastWidgetStats() []WidgetStat {
	return t.lastWidgetStats
}

// SetCapabilities overrides the auto-detected terminal capability profile.
//
// Detection is environment-variable based and therefore a guess: it is wrong
// inside tmux and screen, over SSH with an unhelpful TERM, and anywhere the
// emulator does not advertise itself. Applications that know better — because
// they negotiated with the terminal, read a config file, or are driving a
// backend they control — can say so here.
//
// Call it before the first Draw; the profile is read on every flush.
func (t *Terminal) SetCapabilities(profile CapabilityProfile) {
	if t == nil {
		return
	}
	t.caps = profile
	t.capsPinned = true
}

// refreshCapabilities folds new answers from the capability handshake into
// the profile. The check is one atomic load, so it runs on every frame: the
// answers arrive asynchronously, usually before the first frame, but on a slow
// link possibly after it.
func (t *Terminal) refreshCapabilities() {
	if t.capsPinned || t.driver == nil {
		return
	}
	if t.driver.TerminalReportVersion() == t.reportVersion {
		return
	}
	report, version := t.driver.TerminalReport()
	t.reportVersion = version
	caps := t.detected.WithReport(report)
	if caps != t.caps && t.drawn {
		// What is on screen was encoded for the old profile — with REP the
		// terminal may not have, or cursor positions that assumed other
		// cluster widths. Only a full repaint puts it right.
		t.ForceFullRedraw()
	}
	t.caps = caps
}

// Capabilities returns the capability profile of the active terminal.
func (t *Terminal) Capabilities() CapabilityProfile {
	return t.caps
}

// SetTitle sets the terminal window title with OSC 2 (`ESC ] 2 ; <title> BEL`).
//
// Control characters are stripped first so a title containing ESC or BEL
// cannot inject further escape sequences. An empty title still writes OSC 2
// (some emulators treat that as "clear the title").
func (t *Terminal) SetTitle(title string) {
	if t == nil || t.driver == nil {
		return
	}
	clean := sanitizeWindowTitle(title)
	seq := make([]byte, 0, 4+len(clean)+1)
	seq = append(seq, 0x1b, ']', '2', ';')
	seq = append(seq, clean...)
	seq = append(seq, 0x07)
	_, _ = t.driver.Write(seq)
}

// SaveTitle asks the terminal to push the current window title onto its own
// stack (`CSI 22 ; 2 t`), so RestoreTitle can put it back on exit. Terminals
// that do not implement XTWINOPS ignore it, and RestoreTitle then does
// nothing visible — the title simply stays as the application set it.
func (t *Terminal) SaveTitle() {
	if t == nil || t.driver == nil {
		return
	}
	_, _ = t.driver.Write([]byte("\x1b[22;2t"))
}

// RestoreTitle pops the title saved by SaveTitle (`CSI 23 ; 2 t`).
func (t *Terminal) RestoreTitle() {
	if t == nil || t.driver == nil {
		return
	}
	_, _ = t.driver.Write([]byte("\x1b[23;2t"))
}

// sanitizeWindowTitle drops C0 controls and DEL, and the C1 controls a UTF-8
// terminal reads as ST, so OSC 2 cannot be nested or ended from inside the
// payload.
func sanitizeWindowTitle(title string) string { return sanitizeOSCText(title) }

// Suspend hands the terminal back to the shell and stops the process, as
// Ctrl+Z does in any other program. It returns when the shell resumes the
// application, with raw mode and the screen set up again and the next frame
// forced to repaint in full — the shell has written over the screen, and the
// terminal may even be a different one.
//
// It returns driver.ErrSuspendUnsupported on a backend with no controlling
// terminal to give back: a remote or in-memory one, the browser, Windows.
func (t *Terminal) Suspend() error {
	if t == nil || t.driver == nil {
		return nil
	}
	// The shell should not inherit a resize arrow or the kitty keyboard
	// flags; the next mouse move and the next frame set them again.
	t.RestoreModes()
	if err := t.driver.Suspend(); err != nil {
		return err
	}
	// Answers to the fresh handshake land in the report; take them next frame.
	t.reportVersion = 0
	t.ForceFullRedraw()
	return nil
}

// Draw initiates a frame drawing pass. It detects terminal resize, clears the front buffer,
// executes the user draw callback fn, computes the differential ANSI stream, and writes changes
// in a single synchronized I/O pass.
// Performance: Employs a zero-allocation design on steady-state redraw passes.
func (t *Terminal) Draw(fn func(f *Frame)) error {
	t0 := time.Now()
	t.refreshCapabilities()
	t.syncKeyboardMode()
	// Query the current screen size
	w, h, err := t.driver.Size()
	if t.inline > 0 {
		// An inline application owns a fixed band of rows, not the screen.
		h = t.inline
	}
	if err != nil {
		return err
	}
	if w == 0 || h == 0 {
		w, h = t.front.Area.Width, t.front.Area.Height
		if w == 0 || h == 0 {
			w, h = 80, 24
		}
	}

	// If the window size changed, resize only the front buffer.
	// The back buffer keeps its size, so buffer.Diff can see the resize,
	// clear the screen and redraw the whole frame onto the cleared screen.
	if w != t.front.Area.Width || h != t.front.Area.Height {
		t.front.Resize(cell.NewRect(0, 0, w, h))
		if t.cast != nil {
			t.cast.resize(w, h)
		}
	}

	// Clear the active drawing buffer
	t.front.Clear()
	// Reset the registered click regions
	t.frame.Reset()
	t.frame.Hyperlinks = t.caps.Hyperlinks
	t.frame.ImageProtocol = uint8(t.caps.GraphicsProto)
	if t.frame.FocusManager != nil {
		t.frame.FocusManager.Clear()
	}

	// Hand the frame context (Frame) to the application to draw its components
	if fn != nil {
		fn(t.frame)
	}
	if t.backdropActive() {
		t.keepAppLayer()
		t.composeBackdrop()
	}
	return t.present(t0)
}

// present finishes a frame whose cells are in the front buffer: it applies
// the transition and the debug overlay, places images, and writes the diff.
func (t *Terminal) present(t0 time.Time) error {
	// If a dither transition is active, blend the screen buffer first.
	// The debug HUD is drawn after it, so the transition does not fade or
	// garble the debug lines and labels.
	if t.transitionActive && t.transitionOldBuf != nil {
		animation.ApplyDitherFade(t.front, t.transitionOldBuf, t.transitionProgress)
	}

	// In debug mode, draw the layout bounds over the transition.
	if t.debugMode {
		// Show every widget's debug region. Regions cover each other by their
		// z-index and drawing order; no widget is left out.
		t.drawDebugOverlay()
	}

	// Detect a change in the layer or modal structure. Native images have to
	// be placed again when a modal opens or closes.
	currentLayersHash := t.layersHash()
	layersChanged := currentLayersHash != t.lastLayersHash
	t.lastLayersHash = currentLayersHash

	// If no cell changed and there is no native image, transition or debug layer,
	// skip the synchronised update, the image pass and the diff altogether.
	sizeChanged := t.front.Area.Width != t.back.Area.Width || t.front.Area.Height != t.back.Area.Height
	if !sizeChanged && !t.front.IsDirty && !t.transitionActive && !t.debugMode &&
		len(t.frame.ImageRegions) == 0 && t.lastImageCount == 0 {
		t.lastFrameDuration = time.Since(t0)
		t.copyWidgetStats()
		return nil
	}

	// ── Single-write batching ──
	// All drawing escape codes (synchronised update, images and the cell diff) are
	// collected in the preallocated t.writeBuf and written with a single t.driver.Write.
	t.writeBuf = t.writeBuf[:0]

	// Synchronised update protocol (?2026)
	syncWrapped := false
	if t.caps.SyncOutput {
		t.writeBuf = append(t.writeBuf, "\x1b[?2026h"...)
		syncWrapped = true
	}
	// A redraw whose cells all match the previous frame produces no body.
	// Remember where the body starts so such a frame can be dropped whole
	// instead of sending an empty ?2026h/?2026l pair on every tick.
	bodyStart := len(t.writeBuf)

	// On a full redraw, ESC[2J must not erase native images already sent.
	// Sizes are matched here, and the clear is moved ahead of the image pass.
	needsFullClear := sizeChanged
	if needsFullClear {
		t.back.Resize(t.front.Area)
		// ESC[2J clears the whole screen, which in inline mode means the user's
		// scrollback. An inline frame owns only its own band, and DiffInline
		// already erases each of its rows with EL.
		if t.inline == 0 {
			t.writeBuf = append(t.writeBuf, "\x1b[2J"...)
		}
	}

	// ── STEP 1: add Kitty/Sixel images to the buffer (the pixel layer at the back) ──
	// The protocol detected when the terminal was created (or set with
	// SetCapabilities). Detecting it again here read a dozen environment
	// variables on every frame, and on Windows each read allocates.
	proto := t.caps.GraphicsProto
	if proto != graphics.ProtocolHalfBlock {
		imageRegions := t.clippedImageRegions()
		if len(imageRegions) > 0 {
			cellW, cellH, _ := t.driver.CellPixelSize()

			imagesChanged := needsFullClear || layersChanged
			if !imagesChanged {
				if len(imageRegions) != len(t.lastDrawnImages) {
					imagesChanged = true
				} else {
					for i, reg := range imageRegions {
						prev := t.lastDrawnImages[i]
						if reg.Img != prev.Img || reg.Area != prev.Area || reg.ZIndex != prev.ZIndex {
							imagesChanged = true
							break
						}
					}
				}
			}

			if imagesChanged {
				if proto == graphics.ProtocolKitty {
					t.writeBuf = append(t.writeBuf, "\x1b_Ga=d,d=A,q=2\x1b\\"...)
				}

				for _, reg := range imageRegions {
					zIndex := reg.ZIndex
					if proto == graphics.ProtocolKitty && zIndex == 0 {
						zIndex = -1
					}
					escSeq := graphics.GetCachedEscapeSequence(reg.Img, reg.Area.Width, reg.Area.Height, cellW, cellH, proto, zIndex, reg.Transparent)
					if escSeq != "" {
						t.writeBuf = buffer.AppendCursor(t.writeBuf, reg.Area.X, reg.Area.Y)
						t.writeBuf = append(t.writeBuf, escSeq...)
					}
				}

				if cap(t.lastDrawnImages) >= len(imageRegions) {
					t.lastDrawnImages = t.lastDrawnImages[:len(imageRegions)]
				} else {
					t.lastDrawnImages = make([]ImageRegion, len(imageRegions))
				}
				copy(t.lastDrawnImages, imageRegions)
			}
			t.lastImageCount = len(imageRegions)
		} else {
			if t.lastImageCount > 0 {
				if proto == graphics.ProtocolKitty {
					t.writeBuf = append(t.writeBuf, "\x1b_Ga=d,d=A,q=2\x1b\\"...)
				}
				t.lastImageCount = 0
				t.lastDrawnImages = nil
			}
		}
	}

	// ── STEP 2: draw the ASCII buffer (ON TOP of the pixel layer) ──
	var diffErr error
	diffOpts := buffer.DiffOptions{
		TrueColor:  t.caps.TrueColor,
		Colors256:  t.caps.Colors256,
		EraseChar:  t.caps.EraseChar,
		RepeatChar: t.caps.RepeatChar,
		Hyperlinks: t.caps.Hyperlinks,
		// A terminal that confirmed mode 2027 needs no cursor re-anchoring
		// after each grapheme cluster.
		ClusterWidths: t.caps.ClusterWidths,
		ScrollRegions: t.caps.ScrollRegions,
		InsertDelete:  t.caps.ScrollRegions,
		// Draw already wrapped the frame in ?2026 above; wrapping again inside
		// the encoder would nest the sequence.
		SyncOutput: false,
	}
	if t.inline > 0 {
		// Inline frames are emitted relative to the cursor, because the row the
		// application starts on moves whenever the terminal scrolls.
		t.writeBuf, diffErr = buffer.DiffInline(t.front, t.back, t.writeBuf, diffOpts)
	} else {
		t.writeBuf, diffErr = buffer.DiffWithOptions(t.front, t.back, t.writeBuf, diffOpts)
	}
	if diffErr != nil {
		return diffErr
	}

	if len(t.writeBuf) == bodyStart {
		t.writeBuf = t.writeBuf[:0]
	}

	// Close the synchronized update.
	if syncWrapped && len(t.writeBuf) > 0 {
		t.writeBuf = append(t.writeBuf, "\x1b[?2026l"...)
	}

	// Send the whole frame to stdout in a single I/O call
	if len(t.writeBuf) > 0 {
		if _, err := t.driver.Write(t.writeBuf); err != nil {
			return err
		}
		if t.cast != nil {
			t.cast.output(t.writeBuf)
		}
		t.drawn = true
	}

	dur := time.Since(t0)
	t.lastFrameDuration = dur
	t.copyWidgetStats()

	return nil
}

// copyWidgetStats copies the frame profiling statistics into a reused slice.
func (t *Terminal) copyWidgetStats() {
	if cap(t.lastWidgetStats) >= len(t.frame.WidgetStats) {
		t.lastWidgetStats = t.lastWidgetStats[:len(t.frame.WidgetStats)]
		copy(t.lastWidgetStats, t.frame.WidgetStats)
		return
	}
	t.lastWidgetStats = make([]WidgetStat, len(t.frame.WidgetStats))
	copy(t.lastWidgetStats, t.frame.WidgetStats)
}

// clippedImageRegions intentionally preserves native image placements.
// Modals are drawn in the cell layer above the images; cropping or
// re-encoding an image while a modal moves would make it look as if the
// image had moved or rescaled, and cause needless expensive redraws.
func (t *Terminal) clippedImageRegions() []ImageRegion {
	if t == nil || t.frame == nil {
		return nil
	}
	return t.frame.ImageRegions
}

// LastImageRegions returns a copy of image regions registered during the last frame.
func (t *Terminal) LastImageRegions() []ImageRegion {
	if t == nil || t.frame == nil {
		return nil
	}
	return t.frame.ImageRegionsSnapshot()
}

// SetTransitionProgress sets the dither-fade transition progress (0.0 - 1.0).
func (t *Terminal) SetTransitionProgress(p float64) {
	t.transitionProgress = p
}

// SetTransitionActive turns the dither-fade transition on or off.
func (t *Terminal) SetTransitionActive(active bool) {
	if !active {
		if t.transitionActive {
			t.transitionActive = false
			t.transitionProgress = 1.0
			t.transitionOldBuf = nil
			t.ForceFullRedraw()
		}
		return
	}
	if t.transitionActive {
		return
	}
	t.transitionActive = true
	w := t.back.Area.Width
	h := t.back.Area.Height
	if t.transitionOldBuf == nil || t.transitionOldBuf.Area.Width != w || t.transitionOldBuf.Area.Height != h {
		t.transitionOldBuf = buffer.NewBuffer(cell.NewRect(0, 0, w, h))
	}
	// Copy the back buffer into oldBuf
	if len(t.transitionOldBuf.Content) == len(t.back.Content) {
		copy(t.transitionOldBuf.Content, t.back.Content)
	}
}

// IsTransitionActive reports whether the dither-fade transition is active.
func (t *Terminal) IsTransitionActive() bool {
	return t.transitionActive
}

// RouteMouseEvent matches a mouse click, drag or wheel event from the terminal against
// the click regions registered in the last frame drawn, and calls the matching callback.
// With layered rendering, regions in the topmost layer take precedence.
// It returns true if the event matched a region and was handled, false otherwise.
func (t *Terminal) RouteMouseEvent(ev driver.MouseEvent) bool {
	t.updatePointer(ev)
	// 0. Mouse capture is checked first; drag and release events go to the
	// capture handler regardless of the propagation regions.
	if t.mouseCaptureHandler != nil {
		t.mouseCaptureHandler(ev)
		if ev.Button == driver.MouseRelease {
			t.mouseCaptureHandler = nil
		}
		return true
	}

	// MouseRelease is consumed by the capture above. Plain click regions only
	// receive left button presses, and mouse regions hover (MouseNone) events.
	if ev.Button != driver.MouseLeft && ev.Button != driver.MouseNone && ev.Button != driver.MouseScrollUp && ev.Button != driver.MouseScrollDown {
		return false
	}
	if ev.Button == driver.MouseLeft && ev.Drag {
		return false
	}
	if ev.Button == driver.MouseNone {
		t.frame.DispatchPointerMove(ev)
	}

	// Reset the frame's capture requests before normal routing
	t.frame.mouseCaptureRequest = nil
	propagationHandled := t.dispatchEventRegions(ev)
	if t.frame.mouseCaptureRequest != nil {
		t.mouseCaptureHandler = t.frame.mouseCaptureRequest
		t.frame.mouseCaptureRequest = nil
	}
	if propagationHandled {
		return true
	}

	// 1. Layer system: search from the topmost layer down
	if len(t.frame.Layers) > 0 {
		topLayer := t.frame.TopLayer()
		if topLayer != nil {
			if topLayer.Area.Contains(ev.X, ev.Y) {
				// The click is inside the top layer: check only that layer's regions
				for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
					reg := t.frame.ClickRegions[i]
					if reg.LayerID == topLayer.ID && reg.Area.Contains(ev.X, ev.Y) && (reg.MouseOnly && (ev.Button == driver.MouseNone || ev.Button == driver.MouseScrollUp || ev.Button == driver.MouseScrollDown) || ev.Button == driver.MouseLeft) {
						reg.Fire(ev, t.frame)
						if t.frame.mouseCaptureRequest != nil {
							t.mouseCaptureHandler = t.frame.mouseCaptureRequest
							t.frame.mouseCaptureRequest = nil
						}
						return true
					}
				}
				// Inside the top layer but no click area of that layer → swallow the event (never leak it to the layers below)
				return true
			} else {
				// Clicked outside the top layer → fire ClickOutside (on left button presses only)
				if ev.Button == driver.MouseLeft && !ev.Drag && topLayer.ClickOutside != nil {
					topLayer.ClickOutside()
				}
				return true // Swallow the click
			}
		}
	}

	// 2. Backwards compatibility: ActiveModal (may have been set with the old RegisterModal API)
	if t.frame.ActiveModal != nil {
		modal := t.frame.ActiveModal
		if modal.Area.Contains(ev.X, ev.Y) {
			// Inside the modal: search only the regions with the modal's ID
			for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
				reg := t.frame.ClickRegions[i]
				if reg.LayerID == modal.ID && reg.Area.Contains(ev.X, ev.Y) && (reg.MouseOnly && (ev.Button == driver.MouseNone || ev.Button == driver.MouseScrollUp || ev.Button == driver.MouseScrollDown) || ev.Button == driver.MouseLeft) {
					reg.Fire(ev, t.frame)
					if t.frame.mouseCaptureRequest != nil {
						t.mouseCaptureHandler = t.frame.mouseCaptureRequest
						t.frame.mouseCaptureRequest = nil
					}
					return true
				}
			}
			return true // Inside the modal but on empty space: swallow the event
		} else {
			// Click outside the modal (on left button presses only)
			if ev.Button == driver.MouseLeft && !ev.Drag && modal.ClickOutside != nil {
				modal.ClickOutside()
			}
			return true
		}
	}

	// 3. The normal (layerless) click routing loop
	for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
		reg := t.frame.ClickRegions[i]
		if reg.LayerID == "" && reg.Area.Contains(ev.X, ev.Y) && (reg.MouseOnly && (ev.Button == driver.MouseNone || ev.Button == driver.MouseScrollUp || ev.Button == driver.MouseScrollDown) || ev.Button == driver.MouseLeft) {
			reg.Fire(ev, t.frame)
			if t.frame.mouseCaptureRequest != nil {
				t.mouseCaptureHandler = t.frame.mouseCaptureRequest
				t.frame.mouseCaptureRequest = nil
			}
			return true
		}
	}
	return false
}

func (t *Terminal) dispatchEventRegions(ev driver.MouseEvent) bool {
	return t.frame.DispatchEventRegions(ev)
}

// FocusManager returns the terminal's focus manager.
func (t *Terminal) FocusManager() *FocusManager {
	return t.frame.FocusManager
}

// HoveredRegionID returns the semantic event-region ID under the pointer in
// the most recently routed frame. It is intentionally separate from the
// visual mouse coordinates so inspectors can distinguish semantic targets
// from ordinary click regions.
func (t *Terminal) HoveredRegionID() string {
	if t == nil || t.frame == nil {
		return ""
	}
	return t.frame.HoveredRegionID()
}

// SetDebugMode turns debug (layout inspector) mode on or off.
func (t *Terminal) SetDebugMode(active bool) {
	t.debugMode = active
}

// DebugMode reports whether debug mode is on.
func (t *Terminal) DebugMode() bool {
	return t.debugMode
}

// drawDebugOverlay outlines every drawn widget with dashed lines and labels its
// corner with the widget's type, size and z-index layer.
// With z-order clipping, layers on top cover the lines of the ones below.
//
// Debug regions are clipped by z-index and drawing order. The topmost
// widget's own border and label are still drawn; no widget is hidden.
func (t *Terminal) drawDebugOverlay() {
	borderStyle := cell.Style{
		Fg: cell.NewColorRGB(255, 0, 255), // Bright magenta
		Bg: cell.NewColorRGB(35, 20, 35),
	}
	textStyle := cell.Style{
		Fg:       cell.NewColorRGB(255, 255, 255),
		Bg:       cell.NewColorRGB(255, 0, 255),
		Modifier: cell.ModifierBold,
	}

	for regionIndex, reg := range t.frame.DebugRegions {
		area := reg.Area
		if area.Width == 0 || area.Height == 0 {
			continue
		}

		// Horizontal dashed lines
		for col := area.X; col < area.X+area.Width; col++ {
			if !isObscured(col, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(col, area.Y); c != nil {
					c.Content = '╌'
					c.Style = borderStyle
				}
			}
			if !isObscured(col, area.Y+area.Height-1, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(col, area.Y+area.Height-1); c != nil {
					c.Content = '╌'
					c.Style = borderStyle
				}
			}
		}
		// Vertical dashed lines
		for row := area.Y; row < area.Y+area.Height; row++ {
			if !isObscured(area.X, row, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(area.X, row); c != nil {
					c.Content = '╎'
					c.Style = borderStyle
				}
			}
			if !isObscured(area.X+area.Width-1, row, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(area.X+area.Width-1, row); c != nil {
					c.Content = '╎'
					c.Style = borderStyle
				}
			}
		}

		// Join the corners
		if !isObscured(area.X, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X, area.Y); c != nil {
				c.Content = '┌'
				c.Style = borderStyle
			}
		}
		if !isObscured(area.X+area.Width-1, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X+area.Width-1, area.Y); c != nil {
				c.Content = '┐'
				c.Style = borderStyle
			}
		}
		if !isObscured(area.X, area.Y+area.Height-1, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X, area.Y+area.Height-1); c != nil {
				c.Content = '└'
				c.Style = borderStyle
			}
		}
		if !isObscured(area.X+area.Width-1, area.Y+area.Height-1, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X+area.Width-1, area.Y+area.Height-1); c != nil {
				c.Content = '┘'
				c.Style = borderStyle
			}
		}

		// Print the size and type label in the top-left corner
		label := fmt.Sprintf(" %s [%dx%d m:%dx%d z:%d", reg.WidgetType, area.Width, area.Height, reg.Measured.IdealWidth, reg.Measured.IdealHeight, reg.ZIndex)
		if reg.Overflowed {
			label += " !overflow"
		}
		label += " "
		for idx, r := range label {
			col := area.X + 1 + uint16(idx)
			if col < area.X+area.Width-1 {
				if !isObscured(col, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
					if c := t.front.Get(col, area.Y); c != nil {
						c.Content = r
						c.Style = textStyle
					}
				}
			}
		}
	}
}

// isObscured reports whether a debug region's cell is covered by a region
// above it. At the same z-index, the region drawn later is on top.
func isObscured(x, y uint16, zIndex, regionIndex int, regions []DebugRegion) bool {
	for i := regionIndex + 1; i < len(regions); i++ {
		other := regions[i]
		if other.ZIndex < zIndex {
			continue
		}
		if x >= other.Area.X && x < other.Area.X+other.Area.Width &&
			y >= other.Area.Y && y < other.Area.Y+other.Area.Height {
			return true
		}
	}
	return false
}

// layersHash summarises the position and size of the current layers and modal windows.
// When it changes, images are forced to redraw (to avoid graphics corruption).
func (t *Terminal) layersHash() string {
	res := ""
	for _, l := range t.frame.Layers {
		res += fmt.Sprintf("%s:%d,%d,%d,%d;", l.ID, l.Area.X, l.Area.Y, l.Area.Width, l.Area.Height)
	}
	if t.frame.ActiveModal != nil {
		res += fmt.Sprintf("modal:%s:%d,%d,%d,%d;", t.frame.ActiveModal.ID, t.frame.ActiveModal.Area.X, t.frame.ActiveModal.Area.Y, t.frame.ActiveModal.Area.Width, t.frame.ActiveModal.Area.Height)
	}
	return res
}

// ForceFullRedraw forces every screen cell to be redrawn through the diff.
func (t *Terminal) ForceFullRedraw() {
	if t.back != nil {
		for i := range t.back.Content {
			t.back.Content[i].Content = cell.RuneInvalid
			t.back.Content[i].Style = cell.Style{}
		}
		t.back.Invalidate()
	}
	if t.front != nil {
		t.front.Invalidate()
	}
	t.lastDrawnImages = nil
	t.lastLayersHash = ""
}

// FrontBuffer returns the current front buffer (useful for inspection and testing).
func (t *Terminal) FrontBuffer() *buffer.Buffer {
	return t.front
}

// ClickRegions returns the registered click regions of the current frame.
func (t *Terminal) ClickRegions() []ClickRegion {
	if t.frame == nil {
		return nil
	}
	regions := make([]ClickRegion, len(t.frame.ClickRegions))
	copy(regions, t.frame.ClickRegions)
	return regions
}

// Layers returns the registered layers of the current frame.
func (t *Terminal) Layers() []Layer {
	if t.frame == nil {
		return nil
	}
	layers := make([]Layer, len(t.frame.Layers))
	copy(layers, t.frame.Layers)
	return layers
}

// SetInline switches the terminal to inline rendering in a band of the given
// height, leaving the alternate screen alone. Zero restores full-screen mode.
//
// The driver has to be told too, so it reserves the rows and skips the
// alternate-screen switch; limoni.WithInline wires both.
func (t *Terminal) SetInline(height uint16) {
	t.inline = height
}
