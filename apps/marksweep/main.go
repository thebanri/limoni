// Mark & Sweep: a first-person roguelite in the terminal, drawn in
// characters.
//
//	go run .            # play
//	go run . -fps 60    # smoother, where the terminal keeps up
//	go run . -mute      # no sound
//
// You are Limoni's garbage collector. Bubble Tea, Ratatui, tview, Ink and
// ncurses have come for the heap, each drawn the way its own apps look.
// Aim with the mouse, sweep with the button, collect (reload) with R or the
// right button. Every level brings three augment cards; past an augment's
// last rank it overclocks. After ten minutes the pressure builds, and no
// build holds it for long.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/thebanri/limoni"
)

func main() {
	fps := flag.Int("fps", 30, "frames per second")
	mute := flag.Bool("mute", false, "no sound")
	seed := flag.Uint64("seed", 0, "a seed, for a run that goes the same way twice")
	flag.Parse()

	var mix *mixer
	if !*mute {
		mix = newMixer()
	}
	term, err := limoni.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := *seed
	if s == 0 {
		s = uint64(time.Now().UnixNano())
	}
	g := newGame(s)
	g.store = newStore()
	if g.store != nil {
		g.best = g.store.load()
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
	case *mute:
		g.noSound = "off (-mute)"
	default:
		g.noSound = "no audio output"
	}
	ptr := newPointer()
	app := limoni.NewApp(term,
		limoni.WithFPS(*fps),
		limoni.WithTitle("Mark & Sweep"),
		limoni.WithKeyReleases(),
		limoni.WithMouseMode(limoni.MouseMotion),
	)
	last := time.Now()
	err = app.Run(context.Background(), func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev == nil {
			now := time.Now()
			ptr.drain(g)
			g.step(now.Sub(last).Seconds())
			last = now
		} else {
			g.event(ev)
		}
		g.render(f.Buffer)
		ptr.want(g.phase == phPlay)
		return !g.quit
	})
	term.Close()
	if mix != nil {
		mix.close()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (g *game) event(ev *limoni.Event) {
	switch ev.Type {
	case limoni.EventKey:
		g.key(ev.Key)
	case limoni.EventMouse:
		if !g.locked {
			g.mouse(ev.Mouse)
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
			g.menuKey(k, isRune, ch, &g.menu, g.titlePick)
		}
		return
	case phPause:
		if down {
			// The Esc that let the browser's pointer go may arrive after
			// the pause it caused; it must not undo it.
			if (k.Type == limoni.KeyEsc && g.clock-g.pausedAt > 0.3) || isRune && ch == 'p' {
				g.phase = phPlay
				return
			}
			g.menuKey(k, isRune, ch, &g.pause, g.pausePick)
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
		if down && isRune && ch >= '1' && ch <= '3' {
			g.take(int(ch - '1'))
		} else if down && k.Type == limoni.KeyEnter && g.cardHov >= 0 {
			g.take(g.cardHov)
		}
		return
	}

	var a action
	switch {
	case isRune && ch == 'w' || k.Type == limoni.KeyUp:
		a = actFwd
	case isRune && ch == 's' || k.Type == limoni.KeyDown:
		a = actBack
	case isRune && ch == 'a':
		a = actLeft
	case isRune && ch == 'd':
		a = actRight
	case k.Type == limoni.KeyLeft || isRune && ch == 'q':
		a = actTurnL
	case k.Type == limoni.KeyRight || isRune && ch == 'e':
		a = actTurnR
	case isRune && ch == 'f':
		a = actFire // a keyboard trigger, for terminals without a mouse
	default:
		if !down {
			return
		}
		switch {
		case k.Type == limoni.KeyEsc || isRune && ch == 'p':
			g.pauseRun()
		case isRune && ch == 'r':
			g.reload()
		case k.Type == limoni.KeySpace:
			g.dash()
		case isRune && ch == 'm':
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

// menuKey moves through a three-item menu whose middle item is the volume.
func (g *game) menuKey(k limoni.KeyEvent, isRune bool, ch rune, sel *int, pick func(int)) {
	switch {
	case isRune && ch == 'w' || k.Type == limoni.KeyUp:
		*sel = (*sel + 2) % 3
		g.sound(sfxMenu)
	case isRune && ch == 's' || k.Type == limoni.KeyDown:
		*sel = (*sel + 1) % 3
		g.sound(sfxMenu)
	case *sel == 1 && (k.Type == limoni.KeyLeft || isRune && ch == 'a'):
		g.setVolume(g.volume - 1)
		g.sound(sfxMenu)
		g.persist()
	case *sel == 1 && (k.Type == limoni.KeyRight || isRune && ch == 'd'):
		g.setVolume(g.volume + 1)
		g.sound(sfxMenu)
		g.persist()
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
		g.reset()
		g.sound(sfxPick)
	case 1:
		if g.volume > 0 {
			g.setVolume(0)
		} else {
			g.setVolume(g.lastVol)
			g.sound(sfxMenu)
		}
		g.persist()
	case 2:
		g.quit = true
	}
}

func (g *game) pausePick(i int) {
	switch i {
	case 0:
		g.phase = phPlay
	case 1:
		g.titlePick(1)
	case 2:
		g.quit = true
	}
}

func (g *game) mouse(m limoni.MouseEvent) {
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
		if m.Button == limoni.MouseLeft && !m.Drag && g.cardHov >= 0 {
			g.take(g.cardHov)
			g.release(actFire)
		}
	case phTitle:
		if press {
			g.titlePick(0)
		}
	case phPause:
		if press {
			g.phase = phPlay
		}
	case phDead:
		if press && g.clock-g.deadAt > 1 {
			g.reset()
		}
	}
}

func (g *game) pauseRun() {
	g.phase, g.pause, g.pausedAt = phPause, 0, g.clock
	g.held = [nAct]bool{}
}

// pointerButton is a button of the browser's locked pointer.
func (g *game) pointerButton(button int, down bool) {
	switch {
	case button == 0 && down:
		if g.phase == phPlay {
			g.press(actFire)
		}
	case button == 0:
		g.release(actFire)
	case button == 2 && down:
		g.reload()
	}
}
