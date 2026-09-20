// Command globe is a world you can turn, zoom into and pin places on, drawn
// in the terminal — and an example of what a Limoni application looks like
// from the outside, since this is a module of its own that depends on a
// published Limoni and uses nothing but its public API.
//
//	go install github.com/thebanri/limoni/apps/globe@latest
//
//	globe                          # the whole planet, turning
//	globe -at Türkiye              # open looking at a place
//	globe -at Istanbul -zoom 8     # and closer in
//	globe -ascii                   # shading characters instead of colour
//
// Keys: / find a place · ⏎ fly there · ←→↑↓ turn · +− zoom · space hold ·
// click pin a point · m pin the centre · c clear · p panel · b borders ·
// a ascii · g graticule · r reset · ? keys · q quit.
//
// Everything on screen is in the semantic tree, so a test or an agent can
// drive it by name rather than by pixel:
//
//	go run -tags limoni_debug . -socket "$XDG_RUNTIME_DIR/globe.sock"
//	claude mcp add globe -- limoni-mcp -socket "$XDG_RUNTIME_DIR/globe.sock"
//
// The land is Natural Earth 1:110m, which is in the public domain.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/apps/globe/globe"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "globe:", err)
		os.Exit(1)
	}
}

func run() error {
	at := flag.String("at", "", "open looking at this country or city")
	lat := flag.Float64("lat", 20, "latitude at the centre of the view")
	lon := flag.Float64("lon", 20, "longitude at the centre of the view")
	zoom := flag.Float64("zoom", 1, "1 shows the whole globe; larger zooms in")
	spin := flag.Bool("spin", true, "keep the world turning")
	ascii := flag.Bool("ascii", false, "draw with shading characters instead of colour")
	fps := flag.Int("fps", 24, "frames a second while the world is moving")
	socket := flag.String("socket", "", "serve the UI to agents and tests on this Unix socket (needs a -tags limoni_debug build)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: globe [flags]\n\nA world you can turn, search and pin.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	b := driver.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		return err
	}
	term, err := terminal.New(b)
	if err != nil {
		_ = b.Close()
		return err
	}
	defer term.Close()

	opts := []limoni.AppOption{
		limoni.WithoutDefaultQuitKeys(),
		limoni.WithTitle("globe"),
		limoni.WithSuspend(),
	}
	if *socket != "" {
		opts = append(opts, limoni.WithAutomation(*socket, limoni.AutomationPolicy{
			AllowInput: true, ExposeScreen: true, ExposeInputValues: true,
		}))
	}
	app := limoni.NewApp(term, opts...)

	viewer := globe.New()
	viewer.SetWake(app.Wakeup)
	viewer.Globe().ASCII = *ascii
	viewer.Look(*lat, *lon, *zoom)
	viewer.SetSpinning(*spin)

	if *at != "" {
		p, ok := viewer.Find(*at)
		if !ok {
			return fmt.Errorf("no country or city called %q — try %s", *at, strings.Join(viewer.Suggest(*at, 3), ", "))
		}
		viewer.Look(p.Lat, p.Lon, *zoom)
		if *zoom <= 1 {
			viewer.Look(p.Lat, p.Lon, 3)
		}
		viewer.PinPlace(p, time.Now())
	}

	// The world turns between key presses, so something has to ask for the
	// next frame. A ticker is enough, and it stops when nothing is moving.
	go func() {
		t := time.NewTicker(time.Second / time.Duration(max(1, *fps)))
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if viewer.Moving() {
					app.Wakeup()
				}
			}
		}
	}()

	return app.Run(ctx, viewer.Frame)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
