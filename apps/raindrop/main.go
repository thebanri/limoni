// Raindrop: rain on a puddle, falling where an entropy pool says.
//
//	go run .              # watch it
//	go run . -fps 30      # fewer frames, for a slow terminal or link
//	go run . -theme dusk  # ocean, lemon, dusk, moss or ink; t cycles them
//
// Every frame, every key and every mouse event is stirred into a SHA-256
// pool with the nanosecond it arrived at; the pool decides whether a drop
// falls and where. It is the idea behind Cloudflare's wall of lava lamps,
// shown rather than used: a demonstration, not a source of keys — those
// come from crypto/rand.
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
	name := flag.String("theme", "ocean", "colours: ocean, lemon, dusk, moss or ink")
	flag.Parse()
	th := themeNamed(*name)
	if th < 0 {
		fmt.Fprintf(os.Stderr, "raindrop: no theme %q; there are ocean, lemon, dusk, moss and ink\n", *name)
		os.Exit(2)
	}

	term, err := limoni.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p := newPond(time.Now().UnixNano())
	p.theme = th
	app := limoni.NewApp(term, limoni.WithFPS(*fps), limoni.WithTitle("Raindrop"))
	err = app.Run(context.Background(), func(f *limoni.Frame, ev *limoni.Event) bool {
		now := time.Now().UnixNano()
		switch {
		case ev == nil:
			p.stir(now, 0, 0)
			p.step()
		case ev.Type == limoni.EventKey:
			if ev.Key.Ch == 'q' || ev.Key.Type == limoni.KeyEsc {
				return false
			}
			if ev.Key.Ch == 't' {
				p.theme = (p.theme + 1) % len(themes)
			}
			p.stir(now, uint16(ev.Key.Type), uint16(ev.Key.Ch))
		case ev.Type == limoni.EventMouse:
			p.stir(now, ev.Mouse.X, ev.Mouse.Y)
			if ev.Mouse.Button == limoni.MouseLeft {
				p.drop(int(ev.Mouse.X), 2*int(ev.Mouse.Y), 6)
			}
		}
		p.render(f.Buffer)
		return true
	})
	term.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
