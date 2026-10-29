// Lemon Sweep: a first-person roguelite in the terminal, drawn in
// half-block pixels.
//
//	go run .            # play, at up to 144 frames a second
//	go run . -showfps   # and show how many frames a second are drawn
//	go run . -mute      # no sound
//
// You are Limoni's garbage collector. Bubble Tea, Ratatui, tview, Ink and
// ncurses have come for the heap, each drawn the way its own apps look.
// Aim with the mouse, sweep with the button, collect (reload) with R or the
// right button. Every level brings three augment cards; once every card
// held is at its last rank, a random stat. After ten minutes the pressure
// builds, and no build holds it for long.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

func main() {
	fps := flag.Int("fps", 144, "frames per second to draw; the terminal may show fewer (kitty: repaint_delay)")
	showFPS := flag.Bool("showfps", false, "show the frames drawn each second")
	mute := flag.Bool("mute", false, "no sound")
	seed := flag.Uint64("seed", 0, "a seed, for a run that goes the same way twice")
	gui := flag.Bool("gui", false, "play in a window the game draws itself: keys that never stick and a locked mouse (macOS and Windows)")
	window := flag.Bool("window", false, "open in a window of its own: kitty full screen with no mouse pointer, or in Apple's Terminal one set to draw without stripes")
	flag.Parse()
	if *window {
		var rest []string
		flag.Visit(func(f *flag.Flag) {
			if f.Name != "window" {
				rest = append(rest, "-"+f.Name+"="+f.Value.String())
			}
		})
		if err := openWindow(rest); err != nil {
			fmt.Fprintln(os.Stderr, "-window needs kitty or Apple's Terminal:", err)
			os.Exit(1)
		}
		return
	}

	noWindow := "" // why -gui did not open, for the title screen
	if *gui || autoGUI() && !flagSet("gui") {
		g, mix := newSession(*seed, *mute)
		g.showFPS = *showFPS
		err := runGUI(g)
		if mix != nil {
			mix.close()
		}
		if err == nil {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		if *gui {
			os.Exit(1)
		}
		// No window to be had: play in the terminal, stripes and all, and
		// say why on the title, where stderr is hidden under the game.
		noWindow = "no window (" + err.Error() + "): playing in the terminal"
	}

	// Apple's Terminal has no synchronized output and falls behind a fast
	// full-screen picture, so frames show half drawn; fewer frames tear less.
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" && !flagSet("fps") {
		*fps = 30
	}
	// A browser shows at most 60 a second, and the game shares its one
	// thread with xterm.js: more frames only starve the drawing.
	if runtime.GOOS == "js" && !flagSet("fps") {
		*fps = 60
	}

	term, err := limoni.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	g, mix := newSession(*seed, *mute)
	g.showFPS = *showFPS
	ptr := newPointer()
	app := limoni.NewApp(term,
		limoni.WithFPS(*fps),
		limoni.WithTitle("Lemon Sweep"),
		limoni.WithKeyReleases(),
		limoni.WithMouseMode(limoni.MouseMotion),
	)
	last := time.Now()
	var shown []cell.Cell // the last frame drawn, in the browser
	err = app.Run(context.Background(), func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev == nil && behind() && len(shown) == len(f.Buffer.Content) {
			// xterm.js has yet to show the last frame: the same again sends
			// nothing, and the step waits for the next tick.
			copy(f.Buffer.Content, shown)
		} else if ev == nil {
			now := time.Now()
			ptr.drain(g)
			g.hint = ptr.problem()
			if noWindow != "" {
				g.hint = noWindow
			}
			g.step(now.Sub(last).Seconds())
			last = now
			g.render(f.Buffer)
			g.countFrame(time.Now())
		} else {
			g.event(ev)
			g.render(f.Buffer)
			g.countFrame(time.Now())
		}
		if runtime.GOOS == "js" {
			shown = append(shown[:0], f.Buffer.Content...)
		}
		if g.phase == phPlay {
			f.RenderWidget(crosshairPointer{}, f.Buffer.Area)
		}
		ptr.want(g)
		return !g.quit
	})
	ptr.close()
	term.Close()
	if mix != nil {
		mix.close()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newSession is a game with the player's records and settings loaded and
// the sound on, for either front end: the terminal or -gui's window.
func newSession(seed uint64, mute bool) (*game, *mixer) {
	var mix *mixer
	if !mute {
		mix = newMixer()
	}
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	g := newGame(seed)
	g.store = newStore()
	if g.store != nil {
		g.best = g.store.load()
		if g.best.hasRet {
			g.ret = g.best.ret
		}
		if g.best.sens > 0 {
			g.sens = g.best.sens
		}
		g.mapOff = g.best.mapOff
		if g.best.volume >= 0 {
			g.volume = g.best.volume
			if g.volume > 0 {
				g.lastVol = g.volume
			}
		}
	}
	switch {
	case mix != nil:
		g.audio = mix
		g.setVolume(g.volume)
	case mute:
		g.noSound = "off (-mute)"
	default:
		g.noSound = "no audio output"
	}
	return g, mix
}

// crosshairPointer asks for the crosshair pointer over the picture in the
// terminals that can change it (kitty, foot, Ghostty), so the arrow does not
// cover what is being aimed at.
type crosshairPointer struct{}

func (crosshairPointer) Draw(ctx cell.Context, _ *buffer.Buffer) {
	if ctx.RegisterClickAction != nil {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Pointer: "crosshair"})
	}
}

func (crosshairPointer) SizeHint(cell.Rect) (uint16, uint16) { return 0, 0 }

func (g *game) event(ev *limoni.Event) {
	switch ev.Type {
	case limoni.EventKey:
		g.key(ev.Key)
	case limoni.EventMouse:
		if !g.locked {
			g.mouse(ev.Mouse)
		}
	case driver.EventFocus:
		// Another window has the keyboard: let the mouse go with it.
		if !ev.Focus.Gained && g.phase == phPlay && !g.grab {
			g.pauseRun()
		}
	}
}

func (g *game) key(k limoni.KeyEvent) {
	if k.Release {
		g.exact = true // the terminal says when keys are let go
	}
	ch := k.Ch
	if ch >= 'A' && ch <= 'Z' {
		ch += 'a' - 'A'
	}
	isRune := k.Type == limoni.KeyRune
	down := !k.Release

	switch g.phase {
	case phTitle:
		if down {
			switch {
			case isRune && ch == 'a' || k.Type == limoni.KeyLeft:
				g.mapSel = (g.mapSel + len(arenas) - 1) % len(arenas)
			case isRune && ch == 'd' || k.Type == limoni.KeyRight:
				g.mapSel = (g.mapSel + 1) % len(arenas)
			default:
				g.menuKey(k, isRune, ch, &g.menu, len(titleItems), g.titlePick)
			}
		}
		return
	case phPause:
		if down {
			// The Esc that let the browser's pointer go may arrive after
			// the pause it caused; it must not undo it.
			if (k.Type == limoni.KeyEsc && g.clock-g.pausedAt > 0.3) || isRune && ch == 'p' {
				g.resume()
				return
			}
			g.menuKey(k, isRune, ch, &g.pause, len(pauseItems), g.pausePick)
		}
		return
	case phSettings:
		if down {
			g.settingsKey(k, isRune, ch)
		}
		return
	case phRecipes:
		if down {
			g.bookKey(k, isRune, ch)
		}
		return
	case phDead:
		if down && (isRune && ch == 'r' || k.Type == limoni.KeyEnter) {
			g.reset()
		} else if down && k.Type == limoni.KeyEsc {
			g.quit = true
		}
		return
	case phCards:
		// Movement keys are followed while the cards are up, so the run
		// goes on with the keys as they are: one let go here would stay
		// held, and one still held after a pick on 1, 2 or 3 sends no
		// more repeats to say so.
		if a, ok := moveAction(k, isRune, ch); ok {
			if down {
				g.press(a)
			} else {
				g.release(a)
			}
			return
		}
		if down && isRune && ch >= '1' && ch <= '3' {
			g.take(int(ch - '1'))
		} else if down && k.Type == limoni.KeyEnter && g.cardHov >= 0 {
			g.take(g.cardHov)
		}
		return
	}

	a, ok := moveAction(k, isRune, ch)
	switch {
	case ok:
	case isRune && ch == 'f':
		a = actFire // a keyboard trigger, for terminals without a mouse
	default:
		if !down {
			return
		}
		switch {
		case (k.Type == limoni.KeyEsc || isRune && ch == 'p') && (g.resumedAt < 0 || g.clock-g.resumedAt > 0.3):
			// Not the key that just resumed: the game's window on Wayland,
			// shown again, is sent the keys still held as presses.
			g.pauseRun()
		case isRune && ch == 'r':
			g.reload()
		case k.Type == limoni.KeySpace:
			g.dash()
		case isRune && ch == 'm':
			g.mapOff = !g.mapOff
			g.persist()
		case isRune && ch == 'n':
			if g.volume > 0 {
				g.setVolume(0)
			} else {
				g.setVolume(g.lastVol)
			}
			g.persist()
		}
		return
	}
	if a == actFire {
		g.fireKey = true
	}
	if down {
		g.press(a)
	} else {
		g.release(a)
	}
}

// moveAction is the movement a key holds down, if it is one.
func moveAction(k limoni.KeyEvent, isRune bool, ch rune) (action, bool) {
	switch {
	case isRune && ch == 'w' || k.Type == limoni.KeyUp:
		return actFwd, true
	case isRune && ch == 's' || k.Type == limoni.KeyDown:
		return actBack, true
	case isRune && ch == 'a':
		return actLeft, true
	case isRune && ch == 'd':
		return actRight, true
	case k.Type == limoni.KeyLeft || isRune && ch == 'q':
		return actTurnL, true
	case k.Type == limoni.KeyRight || isRune && ch == 'e':
		return actTurnR, true
	}
	return 0, false
}

// menuKey moves through a menu of n items.
func (g *game) menuKey(k limoni.KeyEvent, isRune bool, ch rune, sel *int, n int, pick func(int)) {
	switch {
	case isRune && ch == 'w' || k.Type == limoni.KeyUp:
		*sel = (*sel + n - 1) % n
		g.sound(sfxMenu)
	case isRune && ch == 's' || k.Type == limoni.KeyDown:
		*sel = (*sel + 1) % n
		g.sound(sfxMenu)
	case k.Type == limoni.KeyEnter || k.Type == limoni.KeySpace:
		pick(*sel)
	case k.Type == limoni.KeyEsc:
		if g.phase == phTitle {
			g.quit = true
		}
	}
}

func (g *game) titlePick(i int) {
	switch i {
	case 0:
		g.endless = g.mapSel == len(arenas)-1
		g.reset()
		g.sound(sfxPick)
	case 1:
		g.openBook()
	case 2:
		g.openSettings()
	case 3:
		g.quit = true
	}
}

// mapAt is the map on the title under column x of row y, or -1.
func (g *game) mapAt(c canvas, x, y int) int {
	if g.mapCards {
		x0 := (c.w - (4*mapCardW + 3*2)) / 2
		if i := (x - x0) / (mapCardW + 2); y >= g.mapY && y < g.mapY+mapCardH && x >= x0 && i < len(arenas) {
			return i
		}
		return -1
	}
	if i := y - g.mapY; i >= 0 && i < len(arenas) && c.w >= 56 && abs(x-c.w/2) <= 26 {
		return i
	}
	return -1
}

func (g *game) pausePick(i int) {
	switch i {
	case 0:
		g.resume()
	case 1:
		g.openBook()
	case 2:
		g.openSettings()
	case 3:
		// Back to the title: the run is let go, not counted.
		g.reset()
		g.phase, g.menu = phTitle, 0
	case 4:
		g.quit = true
	}
}

func (g *game) mouse(m limoni.MouseEvent) {
	// A jump of a third of the window is the pointer coming back in from
	// somewhere else, not a turn.
	if dx := int(m.X) - g.mouseX; g.phase == phPlay && g.mouseIn && g.viewW > 0 && 3*abs(dx) < g.viewW {
		g.lookPend += float64(dx) * lookPerCell(g.viewW) * lookScale(g.sens)
	}
	g.mouseX, g.mouseY, g.mouseIn = int(m.X), int(m.Y), true
	c := canvas{w: g.scrW, h: g.scrH}
	press := !m.Drag && (m.Button == limoni.MouseLeft || m.Button == limoni.MouseRight)
	switch g.phase {
	case phPlay:
		switch {
		case m.Button == limoni.MouseLeft && !m.Drag:
			g.fireKey = false
			g.press(actFire)
		case m.Button == limoni.MouseRelease:
			g.release(actFire)
		case m.Button == limoni.MouseRight && !m.Drag:
			g.reload()
		}
	case phCards:
		g.cardHov = g.cardAt(c, g.mouseX, g.mouseY)
		switch {
		case m.Button == limoni.MouseRelease:
			g.clickOK = true
		case m.Button == limoni.MouseLeft && !m.Drag && g.cardHov >= 0:
			g.click(g.cardHov)
			g.release(actFire)
		}
	case phTitle, phPause:
		// The buttons follow the mouse; a click on one picks it, a click
		// anywhere else starts the run or resumes it.
		sel, pick, n := &g.menu, g.titlePick, len(titleItems)
		if g.phase == phPause {
			sel, pick, n = &g.pause, g.pausePick, len(pauseItems)
		}
		if m := g.mapAt(c, g.mouseX, g.mouseY); g.phase == phTitle && m >= 0 {
			if press {
				g.mapSel = m
			}
			break
		}
		row := g.mouseY - g.menuY
		on := row >= 0 && row < n && abs(g.mouseX-c.w/2) <= buttonW/2
		if on {
			*sel = row
		}
		switch {
		case press && on:
			pick(row)
		case press:
			pick(0)
		}
	case phSettings:
		row := g.mouseY - g.setY
		if row < 0 || row >= len(settingRows) {
			break
		}
		g.setSel = row
		switch {
		case press && row == len(settingRows)-1:
			g.closeSettings()
		case m.Button == limoni.MouseLeft && !m.Drag:
			g.adjust(row, 1)
		case m.Button == limoni.MouseRight && !m.Drag:
			g.adjust(row, -1)
		}
	case phRecipes:
		switch m.Button {
		case limoni.MouseScrollUp:
			g.scrollBook(-1)
		case limoni.MouseScrollDown:
			g.scrollBook(1)
		}
	case phDead:
		if press && g.clock-g.deadAt > 1 {
			g.reset()
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// nudgeCard moves the highlight across the cards for a locked mouse, which
// has no pointer to put on one: a card for every 80 pixels of movement.
func (g *game) nudgeCard(dx float64) {
	if g.cardHov < 0 {
		g.cardHov, g.cardAcc = g.nCards/2, 0
	}
	g.cardAcc += dx
	for ; g.cardAcc > 80; g.cardAcc -= 80 {
		g.cardHov = min(g.cardHov+1, g.nCards-1)
	}
	for ; g.cardAcc < -80; g.cardAcc += 80 {
		g.cardHov = max(g.cardHov-1, 0)
	}
}

func (g *game) pauseRun() {
	g.phase, g.pause, g.pausedAt = phPause, 0, g.clock
	g.held = [nAct]bool{}
}

// resume goes back to the run from the pause, and marks when, so the key
// that resumed cannot pause again.
func (g *game) resume() {
	g.phase, g.resumedAt = phPlay, g.clock
}

// pointerButton is a button of the browser's locked pointer.
func (g *game) pointerButton(button int, down bool) {
	switch {
	case button == 0 && down:
		switch {
		case g.phase == phPlay:
			g.press(actFire)
		case g.phase == phCards && g.cardHov >= 0:
			g.click(g.cardHov) // the locked mouse chose it (nudgeCard)
		}
	case button == 0:
		g.release(actFire)
		g.clickOK = true
	case button == 2 && down:
		g.reload()
	}
}

// autoGUI reports whether the game should open its own window unasked:
// on macOS, where Apple's Terminal draws half blocks short, in stripes, and
// is not named inside tmux or some wrappers that run in it, and on
// Windows, whose terminals never say a key was let go and cannot lock the
// mouse. Not inside -window's Terminal window, and not over SSH, where
// there is no desktop to open a window on.
func autoGUI() bool {
	if os.Getenv("LEMONSWEEP_WINDOW") != "" || os.Getenv("SSH_CONNECTION") != "" {
		return false
	}
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows"
}

// flagSet reports whether the flag was given on the command line.
func flagSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}
