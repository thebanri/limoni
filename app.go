package limoni

import (
	"context"
	"errors"
	"sync"
	"time"
)

// App is one immediate-mode application running on one terminal. Several can
// run in the same process — an SSH server gives each session its own — and
// each has its own wakeup signal.
//
// Run and RunWithContext create an App on the process's own terminal; use
// NewApp to run one on a terminal you created, such as one over an SSH channel.
type App struct {
	term   *Terminal
	cfg    appConfig
	wakeup chan struct{}
}

// NewApp prepares an application on term. The caller owns term: App.Run does
// not close it.
func NewApp(term *Terminal, opts ...AppOption) *App {
	a := &App{term: term, wakeup: make(chan struct{}, 1)}
	for _, opt := range opts {
		if opt != nil {
			opt(&a.cfg)
		}
	}
	return a
}

// Wakeup asks this application to draw a frame without waiting for input.
// It is safe to call from any goroutine; calls made while a wakeup is already
// pending coalesce into one frame.
func (a *App) Wakeup() {
	select {
	case a.wakeup <- struct{}{}:
	default:
	}
}

// Run runs the application until appFn returns false, the terminal's input
// ends, or ctx is cancelled, in which case it returns ctx.Err().
func (a *App) Run(ctx context.Context, appFn func(f *Frame, ev *Event) bool) error {
	running.add(a)
	defer running.remove(a)
	if a.cfg.hasTitle {
		// Push the title the user had, so exiting does not leave the
		// application's name on their terminal.
		a.term.SaveTitle()
		defer a.term.RestoreTitle()
		a.term.SetTitle(a.cfg.title)
	}
	return runLoop(ctx, a.term, appFn, a.cfg, a.wakeup)
}

// Notify shows a desktop notification in terminals that support one (kitty,
// iTerm2, WezTerm, Ghostty, foot, or LIMONI_NOTIFY) and reports whether it
// was sent. Call it from the application function, which runs on the loop
// that draws, never from another goroutine.
func (a *App) Notify(title, body string) bool { return a.term.Notify(title, body) }

// running tracks the Apps that are running, so that the package-level Wakeup
// can reach them.
var running = &appSet{apps: map[*App]struct{}{}}

type appSet struct {
	mu   sync.RWMutex
	apps map[*App]struct{}
}

func (s *appSet) add(a *App) {
	s.mu.Lock()
	s.apps[a] = struct{}{}
	s.mu.Unlock()
}

func (s *appSet) remove(a *App) {
	s.mu.Lock()
	delete(s.apps, a)
	s.mu.Unlock()
}

// Wakeup asks every running application in the process to draw a frame
// without waiting for input. It is safe to call from any goroutine.
//
// With one application — the usual case, and the only one before App
// existed — that is the application. Where several run, prefer App.Wakeup,
// which wakes only the one whose state changed.
func Wakeup() {
	running.mu.RLock()
	for a := range running.apps {
		a.Wakeup()
	}
	running.mu.RUnlock()
}

// AppOption configures the application lifecycle in Run.
type AppOption func(*appConfig)

type appConfig struct {
	catchCtrlC       bool
	fps              int
	automationPath   string
	automationPolicy AutomationPolicy
	inlineHeight     uint16
	suspendOnCtrlZ   bool
	title            string
	hasTitle         bool
	keyReleases      bool
}

// AutomationPolicy decides what an application's automation socket lets out.
// Every field defaults to closed: with the zero value a client can see roles,
// labels, positions and bounds, and nothing else.
//
// It mirrors automation.Policy field for field. It is declared here, in the
// root package, so that a build without the limoni_debug tag does not import
// the automation package at all.
type AutomationPolicy struct {
	// ExposeInputValues sends the values of text fields. Fields marked Secret
	// are never sent, whatever this says.
	ExposeInputValues bool
	// ExposeScreen allows the snapshot of the rendered grid.
	ExposeScreen bool
	// AllowInput permits key, text and click synthesis.
	AllowInput bool
	// AllowUnverifiedPeers accepts connections on platforms that cannot report
	// the connecting user. Without it, those platforms refuse every connection.
	AllowUnverifiedPeers bool
}

// WithTitle sets the terminal window title with OSC 2 when the application
// starts. Control characters in the title are stripped; see Terminal.SetTitle.
func WithTitle(title string) AppOption {
	return func(c *appConfig) {
		c.title = title
		c.hasTitle = true
	}
}

// WithKeyReleases reports key repeats and releases as well as presses, in
// terminals with the kitty keyboard protocol: Event.Key.Repeat and
// Event.Key.Release. It is for games and anything else that needs to know a
// key is still held. Every key then arrives at least twice, so an
// application that turns it on must skip releases wherever it acts on a
// press. See Terminal.SetKeyReleases.
func WithKeyReleases() AppOption {
	return func(c *appConfig) {
		c.keyReleases = true
	}
}

// WithSuspend makes Ctrl+Z hand the terminal back to the shell and stop the
// application, as it does in vim or less; `fg` brings it back and the screen
// is repainted. Without it, Ctrl+Z reaches the application as an ordinary key.
//
// It has no effect where there is no shell to return to — a remote backend,
// the browser, Windows — and the key is delivered as usual there.
func WithSuspend() AppOption {
	return func(c *appConfig) {
		c.suspendOnCtrlZ = true
	}
}

// WithInline renders the application in place, in a band of the given height,
// instead of taking over the screen.
//
// This is the mode `gum`, CI progress renderers and shell prompts use: the
// scrollback above stays intact, and what the application drew is still on
// screen after it exits. Full-screen mode is the default.
func WithInline(height uint16) AppOption {
	return func(c *appConfig) {
		c.inlineHeight = height
	}
}

// WithAutomation serves the application's semantic tree on a Unix socket, so
// tests and agents can drive it by selector instead of by screen coordinate.
// See the automation package for the protocol and its security caveats.
//
// It is off unless you call this. The socket accepts commands that synthesise
// input into the running application, so treat enabling it the way you would
// treat enabling a debug console: a development and CI facility, not something
// to ship on by default.
func WithAutomation(socketPath string, policy AutomationPolicy) AppOption {
	return func(c *appConfig) {
		c.automationPath = socketPath
		c.automationPolicy = policy
	}
}

// WithFPS configures a continuous animation frame rate (e.g. 60, 120, 240 FPS).
// When configured, the render loop continuously invokes the draw function at the target rate.
// Without it the draw function runs only when something happens (input, a
// resize, a Wakeup), and an idle application uses no CPU.
func WithFPS(fps int) AppOption {
	return func(c *appConfig) {
		if fps > 0 {
			c.fps = fps
		}
	}
}

// WithCatchCtrlC configures whether Ctrl+C is forwarded to the application
// as a normal key event instead of automatically terminating the process.
func WithCatchCtrlC(catch bool) AppOption {
	return func(c *appConfig) {
		c.catchCtrlC = catch
	}
}

// WithoutDefaultQuitKeys disables automatic termination on Ctrl+C.
// When enabled, Ctrl+C is forwarded to the application's event handler.
func WithoutDefaultQuitKeys() AppOption {
	return WithCatchCtrlC(true)
}

// Run starts an event-driven Limoni application loop.
// appFn is invoked with the active Frame and the triggering Event.
// Returning false from appFn cleanly exits the application.
// On the first invocation, ev is nil for the initial render.
// By default, Ctrl+C automatically terminates the application gracefully,
// unless WithCatchCtrlC(true) or WithoutDefaultQuitKeys() is supplied.
func Run(appFn func(f *Frame, ev *Event) bool, opts ...AppOption) error {
	return RunWithContext(context.Background(), appFn, opts...)
}

// RunWithContext is Run that also stops when ctx is cancelled, restoring the
// terminal and returning ctx.Err().
func RunWithContext(ctx context.Context, appFn func(f *Frame, ev *Event) bool, opts ...AppOption) error {
	var cfg appConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	term, err := NewInline(cfg.inlineHeight)
	if err != nil {
		return err
	}
	defer term.Close()
	app := &App{term: term, cfg: cfg, wakeup: make(chan struct{}, 1)}
	return app.Run(ctx, appFn)
}

// ErrAutomationNotCompiled is returned by Run when WithAutomation is used in a
// binary built without the limoni_debug tag.
//
// This is the point of the tag. A release binary does not merely have
// automation switched off; the socket server, the policy and the input
// injector are not in it, so no configuration mistake can switch them on.
var ErrAutomationNotCompiled = errors.New("limoni: automation is not compiled into this binary; build with -tags limoni_debug")

// gateway carries application state out of the process and synthetic input
// back in. See app_gateway_debug.go; release builds have no implementation.
type gateway interface {
	publish(f *Frame)
	events() <-chan Event
	close() error
}

// runLoop is Run's body with the terminal supplied, so the loop — including
// the automation wiring — can be exercised against a headless terminal instead
// of only against a tty.
func runLoop(ctx context.Context, term *Terminal, appFn func(f *Frame, ev *Event) bool, cfg appConfig, wakeup <-chan struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var err error
	if cfg.keyReleases {
		term.SetKeyReleases(true)
		defer term.SetKeyReleases(false)
	}
	term.StartEventLoop()

	// The gateway is everything that carries application state out of the
	// process — the automation socket today. Its implementation only exists in
	// builds tagged limoni_debug; a release build gets a stub that refuses to
	// start one, so the code is not merely disabled but absent from the binary.
	gw, err := openGateway(cfg)
	if err != nil {
		return err
	}
	var injected <-chan Event
	publish := func(*Frame) {}
	if gw != nil {
		defer gw.close()
		injected = gw.events()
		publish = gw.publish
	}

	running := true
	// Initial render
	err = term.Draw(func(f *Frame) {
		running = appFn(f, nil)
		publish(f)
	})
	if err != nil || !running {
		return err
	}

	var tickerChan <-chan time.Time
	if cfg.fps > 0 {
		ticker := time.NewTicker(time.Second / time.Duration(cfg.fps))
		defer ticker.Stop()
		tickerChan = ticker.C
	}

	// handle draws one frame for an event, or for nil on a timer or wakeup.
	handle := func(ev *Event) error {
		if ev != nil && ev.Type == EventMouse {
			// Route mouse event through terminal hit-test router
			term.RouteMouseEvent(ev.Mouse)
		}
		return term.Draw(func(f *Frame) {
			running = appFn(f, ev)
			publish(f)
		})
	}

	var answered <-chan struct{}
	if b := term.Backend(); b != nil {
		answered = b.ProbeAnswered()
	}

	events := term.Events()
	for running {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			// Automatic graceful exit on Ctrl+C unless explicitly caught
			if !cfg.catchCtrlC && ev.Type == EventKey && !ev.Key.Release && ev.Key.Ctrl && (ev.Key.Ch == 'c' || ev.Key.Ch == 'C') {
				return nil
			}
			if cfg.suspendOnCtrlZ && ev.Type == EventKey && !ev.Key.Release && ev.Key.Ctrl && (ev.Key.Ch == 'z' || ev.Key.Ch == 'Z') {
				// Hands the terminal back until the shell resumes us; the
				// frame after it repaints the screen the shell wrote over.
				if err := term.Suspend(); err == nil {
					if err := handle(nil); err != nil {
						return err
					}
					continue
				}
				// Unsupported here: the key is the application's, as usual.
			}
			if err := handle(&ev); err != nil {
				return err
			}

		case ev := <-injected:
			// Synthetic input from the automation socket. Ctrl+C is not given
			// the quit shortcut here: a remote client should not be able to
			// terminate the application by accident.
			if err := handle(&ev); err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()

		case <-wakeup:
			// Wakeup from another goroutine: draw without an event.
			if err := handle(nil); err != nil {
				return err
			}

		case <-tickerChan:
			// Set by WithFPS: fires at the target frame rate.
			if err := handle(nil); err != nil {
				return err
			}

		case <-answered:
			// Late answers to the capability probe: the frame on screen may
			// have been encoded for another terminal, and without input
			// nothing else would draw it again.
			answered = nil
			if err := handle(nil); err != nil {
				return err
			}
		}
	}
	return nil
}

// Start displays a static or single-state frame and cleanly exits when 'q', 'ESC', or Ctrl+C is pressed.
func Start(drawFn func(f *Frame)) error {
	return Run(func(f *Frame, ev *Event) bool {
		drawFn(f)
		if ev != nil && ev.Type == EventKey {
			if ev.Key.Type == KeyEsc || ev.Key.Ch == 'q' || ev.Key.Ch == 'Q' {
				return false
			}
		}
		return true
	})
}
