// Command zest is a terminal log viewer: it opens files or reads a pipe,
// follows them as they grow, colours lines by level, and filters millions of
// lines without freezing — built on Limoni as its flagship application.
//
//	zest app.log                      # open and follow a file
//	kubectl logs -f pod | zest        # read a pipe; keys still work
//	zest -level warn -filter timeout app.log
//	zest -demo 1000000                # a generated log, for trying it out
//
// Keys: / filter · 1–6 minimum level · Enter details · f follow · ↑↓ PgUp PgDn
// Home End scroll · ←→ scroll sideways · ? help · q quit.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/internal/zestapp"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "zest:", err)
		os.Exit(1)
	}
}

func run() error {
	follow := flag.Bool("f", true, "keep reading as the input grows, like tail -f")
	level := flag.String("level", "", "show only lines at this level or above: trace, debug, info, warn, error, fatal")
	filter := flag.String("filter", "", "show only lines containing this text (case-insensitive)")
	demo := flag.Int("demo", 0, "view a generated log of this many lines, still growing, instead of a file")
	socket := flag.String("socket", "", "serve the UI to agents and tests on this Unix socket (needs a -tags limoni_debug build)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: zest [flags] [file]\n\nWith no file, zest reads standard input.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	minLevel, err := zestapp.ParseLevel(*level)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	// Input: a file, a demo, or standard input. When standard input is a pipe
	// the keyboard is the controlling terminal, opened separately.
	name := "stdin"
	keyboard := os.Stdin
	var input io.Reader
	switch {
	case *demo > 0:
		name = fmt.Sprintf("demo (%d lines)", *demo)
	case flag.NArg() > 0:
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			return err
		}
		defer f.Close()
		input, name = f, filepath.Base(flag.Arg(0))
	default:
		if isTerminal(os.Stdin) {
			flag.Usage()
			return errors.New("give a file, pipe something in, or try -demo 1000000")
		}
		input = os.Stdin
		tty, err := openTTY()
		if err != nil {
			return fmt.Errorf("stdin is a pipe and the terminal cannot be opened for keys: %w", err)
		}
		defer tty.Close()
		keyboard = tty
	}

	b := driver.NewBackend(keyboard, os.Stdout)
	if err := b.Setup(); err != nil {
		return err
	}
	term, err := terminal.New(b)
	if err != nil {
		_ = b.Close()
		return err
	}
	defer term.Close()

	opts := []limoni.AppOption{limoni.WithoutDefaultQuitKeys()}
	if *socket != "" {
		opts = append(opts, limoni.WithAutomation(*socket, limoni.AutomationPolicy{AllowInput: true, ExposeScreen: true, ExposeInputValues: true}))
	}
	app := limoni.NewApp(term, opts...)
	viewer := zestapp.NewViewer(name)
	viewer.SetWake(app.Wakeup)
	viewer.SetFilter(*filter, minLevel)

	readErr := make(chan error, 1)
	if *demo > 0 {
		go func() { readErr <- viewer.GenerateDemo(ctx, *demo) }()
	} else {
		go func() { readErr <- viewer.ReadFrom(ctx, input, *follow) }()
	}

	// Redraw now and then while following, so the rate and counts move even
	// when the new lines are filtered out.
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				app.Wakeup()
			}
		}
	}()

	err = app.Run(ctx, viewer.Frame)
	select {
	case rerr := <-readErr:
		if rerr != nil && !errors.Is(rerr, context.Canceled) && err == nil {
			err = rerr
		}
	default:
	}
	return err
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
