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

// defaultBoard is the shared leaderboard the game ships with: the
// scoreboard (apps/scoreboard) on Vercel. Empty until it is up; the
// browser's is set in examples/wasm/index.html.
const defaultBoard = ""

func main() {
	fps := flag.Int("fps", 30, "frames per second")
	boss := flag.Bool("boss", false, "start at the lair's gate with ten lemons")
	mute := flag.Bool("mute", false, "no sound")
	board := flag.String("board", "", `the shared leaderboard's address; "off" keeps scores on this machine`)
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
	g.store = newStore()
	g.restore()
	if g.remote = newRemote(boardURL(*board)); g.remote != nil {
		g.worldState = worldLoading
		g.remote.fetch()
	}
	if g.name == "" {
		g.askName()
	}
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
	// A release of any key, on any screen, proves the terminal reports them.
	// The browser page sends one for the Enter that starts the game, so the
	// first walk is already held exactly.
	if k.Release {
		g.exact = true
	}
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

	if g.phase == phName {
		if !k.Release {
			g.nameKey(k)
		}
		return
	}
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

	// ↑ and ↓ are left unbound, here and in the menu: W and S walk.
	var a action
	switch {
	case isRune && ch == 'w':
		a = actFwd
	case isRune && ch == 's':
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
	case isRune && ch == 'r':
		if !k.Release {
			g.reload()
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

// askName opens the name entry, with the name so far to edit.
func (g *game) askName() {
	g.typing = g.name
	g.phase = phName
}

// nameKey types the player's name: letters, digits and a few marks, up to
// nameMax; Backspace takes one back and Enter keeps it.
func (g *game) nameKey(k limoni.KeyEvent) {
	switch {
	case k.Type == limoni.KeyEnter:
		name := cleanName(g.typing)
		if name == "" {
			return
		}
		g.name = name
		g.persist()
		g.phase = phTitle
		g.sound(sfxPickup)
	case k.Type == limoni.KeyBackspace:
		if r := []rune(g.typing); len(r) > 0 {
			g.typing = string(r[:len(r)-1])
			g.sound(sfxMenu)
		}
	case k.Type == limoni.KeySpace || k.Type == limoni.KeyRune:
		ch := k.Ch
		if k.Type == limoni.KeySpace {
			ch = ' '
		}
		if nameRune(ch) && len([]rune(g.typing)) < nameMax && (ch != ' ' || g.typing != "") {
			g.typing += string(ch)
			g.sound(sfxMenu)
		}
	}
}

// titleKey drives the title menu: START, NAME, SOUND and QUIT.
func (g *game) titleKey(k limoni.KeyEvent, isRune bool, ch rune) {
	const items = 4
	switch {
	case isRune && ch == 'w':
		g.menu = (g.menu + items - 1) % items
		g.sound(sfxMenu)
	case isRune && ch == 's':
		g.menu = (g.menu + 1) % items
		g.sound(sfxMenu)
	case g.menu == 2 && (k.Type == limoni.KeyLeft || isRune && ch == 'a'):
		g.setVolume(g.volume - 1)
		g.sound(sfxMenu)
	case g.menu == 2 && (k.Type == limoni.KeyRight || isRune && ch == 'd'):
		g.setVolume(g.volume + 1)
		g.sound(sfxMenu)
	case k.Type == limoni.KeyEnter || k.Type == limoni.KeySpace:
		switch g.menu {
		case 0:
			g.reset()
			g.sound(sfxPickup)
		case 1:
			g.askName()
			g.sound(sfxMenu)
		case 2: // sound on and off
			if g.volume > 0 {
				g.setVolume(0)
			} else {
				g.setVolume(g.lastVol)
				g.sound(sfxMenu)
			}
		case 3:
			g.quit = true
		}
	}
}
