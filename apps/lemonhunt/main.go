// Lemon Hunt: a short first-person game in the terminal.
//
//	go run .            # play
//	go run . -fps 60    # a smoother frame rate, where the terminal keeps up
//	go run . -boss      # start at the lair's gate with ten lemons, to record the fight
//	go run . -mute      # no sound
//
// The rats have hidden the city's lemons in the sewer. Find ten, and the gate
// to the lair lifts: Ratatui, king of the rats, is waiting there with a
// health bar of its own. Squirt it until the bar is empty.
//
// It is a raycaster drawn in half blocks, two pixels to a cell, with lit
// textures, sprites and particles. A frame allocates nothing: the game's
// state and the renderer's scratch live in one value made at start-up. The
// tests hold it to that, through Limoni's diff as well.
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
	boss := flag.Bool("boss", false, "start at the lair's gate with ten lemons")
	mute := flag.Bool("mute", false, "no sound")
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

	g := newGame()
	g.skipToBoss = *boss
	switch {
	case mix != nil:
		g.audio = mix
		g.setVolume(g.volume)
	case *mute:
		g.noSound = "off (-mute)"
	default:
		g.noSound = "no player found"
	}
	last := time.Now()
	app := limoni.NewApp(term,
		limoni.WithFPS(*fps),
		limoni.WithTitle("Lemon Hunt"),
		limoni.WithKeyReleases(),
	)
	err = app.Run(context.Background(), func(f *limoni.Frame, ev *limoni.Event) bool {
		// Time moves on the ticks alone; a key only changes what is held.
		if ev == nil {
			now := time.Now()
			g.step(now.Sub(last).Seconds())
			last = now
		} else if ev.Type == limoni.EventKey {
			g.key(ev.Key)
		}
		g.render(f.Buffer)
		return !g.quit
	})
	term.Close()
	if mix != nil {
		mix.close()
	} else if !*mute {
		fmt.Fprintln(os.Stderr, "lemonhunt: no sound — install pw-play, pacat, aplay or sox to hear it")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (g *game) key(k limoni.KeyEvent) {
	if k.Type == limoni.KeyEsc {
		if !k.Release {
			g.quit = true
		}
		return
	}
	ch := k.Ch
	if ch >= 'A' && ch <= 'Z' {
		ch += 'a' - 'A'
	}
	isRune := k.Type == limoni.KeyRune

	if g.phase == phTitle {
		if !k.Release {
			g.titleKey(k, isRune, ch)
		}
		return
	}
	if g.phase != phPlay {
		if !k.Release && isRune && ch == 'r' {
			g.reset()
		}
		return
	}

	var a action
	switch {
	case k.Type == limoni.KeyUp || isRune && ch == 'w':
		a = actFwd
	case k.Type == limoni.KeyDown || isRune && ch == 's':
		a = actBack
	case k.Type == limoni.KeyLeft:
		a = actTurnL
	case k.Type == limoni.KeyRight:
		a = actTurnR
	case isRune && ch == 'a':
		a = actStrafeL
	case isRune && ch == 'd':
		a = actStrafeR
	case k.Type == limoni.KeySpace || isRune && ch == 'f':
		a = actFire
	case isRune && ch == 'm':
		if !k.Release {
			g.showMap = !g.showMap
		}
		return
	default:
		return
	}
	if k.Release {
		g.release(a)
	} else {
		g.press(a)
	}
}

// titleKey drives the title menu: START, SOUND and QUIT.
func (g *game) titleKey(k limoni.KeyEvent, isRune bool, ch rune) {
	const items = 3
	switch {
	case k.Type == limoni.KeyUp || isRune && ch == 'w':
		g.menu = (g.menu + items - 1) % items
		g.sound(sfxMenu)
	case k.Type == limoni.KeyDown || isRune && ch == 's':
		g.menu = (g.menu + 1) % items
		g.sound(sfxMenu)
	case g.menu == 1 && (k.Type == limoni.KeyLeft || isRune && ch == 'a'):
		g.setVolume(g.volume - 1)
		g.sound(sfxMenu)
	case g.menu == 1 && (k.Type == limoni.KeyRight || isRune && ch == 'd'):
		g.setVolume(g.volume + 1)
		g.sound(sfxMenu)
	case k.Type == limoni.KeyEnter || k.Type == limoni.KeySpace:
		switch g.menu {
		case 0:
			g.reset()
			g.sound(sfxPickup)
		case 1: // sound on and off
			if g.volume > 0 {
				g.setVolume(0)
			} else {
				g.setVolume(g.lastVol)
				g.sound(sfxMenu)
			}
		case 2:
			g.quit = true
		}
	}
}
