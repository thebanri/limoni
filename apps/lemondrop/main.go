// Lemon Drop: falling blocks that turn to sand.
//
//	go run .            # play
//	go run . -fps 30    # fewer frames, for a slow terminal or link
//	go run . -mute      # no sound
//	go run . -board off # keep the scores on this machine
//
// Pieces fall as in any falling-block game, but each one that lands
// crumbles into sand of its colour and runs down the pile. A run of one
// colour that reaches from the left wall to the right wall flashes and is
// gone, and whatever rested on it falls — sometimes into another.
//
// The board is drawn in half blocks, a grain to a pixel, over a faint half
// lemon. A step and a frame allocate nothing: the board, the sand and the
// renderer's scratch live in one value made at start-up. The tests hold it
// to that, through Limoni's diff as well.
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
	fps := flag.Int("fps", 60, "frames per second")
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

	g := newGame(uint64(time.Now().UnixNano()))
	g.store = newStore()
	g.restore()
	if g.remote = newRemote(boardURL(*board)); g.remote != nil {
		g.worldState = worldLoading
		g.remote.fetch()
	}
	if g.name == "" {
		g.askName()
	}
	switch {
	case mix != nil:
		g.audio = mix
	case *mute:
		g.noSound = "off (-mute)"
	default:
		g.noSound = "no player found"
	}
	last := time.Now()
	app := limoni.NewApp(term,
		limoni.WithFPS(*fps),
		limoni.WithTitle("Lemon Drop"),
		limoni.WithKeyReleases(),
	)
	err = app.Run(context.Background(), func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev == nil {
			now := time.Now()
			g.update(now.Sub(last).Seconds())
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
		fmt.Fprintln(os.Stderr, "lemondrop: no sound — install pw-play, pacat, aplay or sox to hear it")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// key handles one key event. Where the terminal reports releases (kitty,
// Ghostty, WezTerm, foot), held arrows repeat at the game's own pace and ↓
// drops fast until it is let go; elsewhere the terminal's auto-repeat does
// the repeating, and each ↓ drops the piece a block.
func (g *game) key(k limoni.KeyEvent) {
	if k.Release {
		g.exact = true
	}
	ch := k.Ch
	if ch >= 'A' && ch <= 'Z' {
		ch += 'a' - 'A'
	}
	rn := k.Type == limoni.KeyRune
	press := !k.Release && !k.Repeat

	if g.phase == phName {
		if k.Type == limoni.KeyEsc && press {
			g.phase = g.back // keep the name there was
			if g.name == "" {
				g.quit = true
			}
		} else if press || k.Repeat {
			g.nameKey(k)
		}
		return
	}
	if k.Type == limoni.KeyEsc || rn && ch == 'q' {
		if press {
			g.quit = true
		}
		return
	}
	if press && rn && ch == 'm' {
		g.muted = !g.muted
		return
	}

	switch g.phase {
	case phTitle:
		switch {
		case press && (k.Type == limoni.KeyEnter || k.Type == limoni.KeySpace):
			g.start()
		case press && rn && ch == 'n':
			g.askName()
		}
		return
	case phOver:
		if press && (rn && ch == 'r' || k.Type == limoni.KeyEnter) {
			g.start()
		}
		return
	}

	if g.paused {
		switch {
		case press && rn && ch == 'p':
			g.paused = false
			g.sound(sfxPause, 0.8)
		case press && rn && ch == 'r':
			g.start()
		}
		return
	}

	left := k.Type == limoni.KeyLeft || rn && ch == 'a'
	right := k.Type == limoni.KeyRight || rn && ch == 'd'
	down := k.Type == limoni.KeyDown || rn && ch == 's'
	switch {
	case left || right:
		if k.Release {
			if left {
				g.holdL = false
			} else {
				g.holdR = false
			}
			return
		}
		// Where releases are reported the game repeats held keys itself, at
		// its own pace, so a repeat is ignored: kitty marks it as one, and
		// the browser page sends it as another press of a key still held.
		if g.exact && (k.Repeat || left && g.holdL || right && g.holdR) {
			return
		}
		if left {
			g.holdL = true
			g.shift(-g.step())
		} else {
			g.holdR = true
			g.shift(g.step())
		}
		g.dasT = 0
	case down:
		switch {
		case k.Release:
			g.holdDown = false
		case g.exact:
			g.holdDown = true
		default:
			g.softStep()
		}
	case !press:
	case k.Type == limoni.KeyUp || rn && (ch == 'w' || ch == 'x'):
		g.turn(1)
	case rn && ch == 'z':
		g.turn(-1)
	case k.Type == limoni.KeySpace:
		g.hardDrop()
	case rn && ch == 'p':
		g.sound(sfxPause, 0.8)
		g.paused = true
		g.holdL, g.holdR, g.holdDown = false, false, false
	}
}

// softStep drops the piece a block, or as far as it will go, for a
// terminal that sends ↓ again and again rather than holding it.
func (g *game) softStep() {
	p := &g.cur
	moved := false
	for i := 0; i < g.b && g.fits(p.kind, p.rot, p.x, p.y+1); i++ {
		p.y++
		p.fall = 0
		moved = true
	}
	if moved {
		g.score++
		g.dropPts++
	}
}

// askName opens the name entry, with the name so far to edit.
func (g *game) askName() {
	g.typing = g.name
	g.back = g.phase
	if g.back == phName || g.back == phPlay {
		g.back = phTitle
	}
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
		g.phase = g.back
	case k.Type == limoni.KeyBackspace:
		if r := []rune(g.typing); len(r) > 0 {
			g.typing = string(r[:len(r)-1])
		}
	case k.Type == limoni.KeySpace || k.Type == limoni.KeyRune:
		ch := k.Ch
		if k.Type == limoni.KeySpace {
			ch = ' '
		}
		if nameRune(ch) && len([]rune(g.typing)) < nameMax && (ch != ' ' || g.typing != "") {
			g.typing += string(ch)
		}
	}
}
