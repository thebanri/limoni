package limoni

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// Two applications in one process — two SSH sessions, say — each with its
// own terminal. Waking one must not redraw the other, and cancelling one must
// leave the other running. With the package-level wakeup channel both of
// these failed: a wakeup went to whichever loop read it first.
func TestAppsInOneProcessAreIsolated(t *testing.T) {
	type session struct {
		app    *App
		frames atomic.Int32
		cancel context.CancelFunc
		done   chan error
	}
	start := func() *session {
		t.Setenv("LIMONI_PROBE", "0")
		b := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 40, 10))
		if err := b.Setup(); err != nil {
			t.Fatal(err)
		}
		term, err := terminal.New(b)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = term.Close() })
		s := &session{app: NewApp(term), done: make(chan error, 1)}
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		go func() {
			s.done <- s.app.Run(ctx, func(f *Frame, ev *Event) bool {
				s.frames.Add(1)
				return true
			})
		}()
		waitFor(t, func() bool { return s.frames.Load() == 1 }) // initial frame
		return s
	}
	a, b := start(), start()

	// Alternate, so that a shared channel cannot pass by luck: the goroutine
	// that started waiting first would take every wakeup.
	for round := 0; round < 10; round++ {
		target, other := a, b
		if round%2 == 0 {
			target, other = b, a
		}
		wantTarget, wantOther := target.frames.Load()+1, other.frames.Load()
		target.app.Wakeup()
		waitFor(t, func() bool { return target.frames.Load() == wantTarget })
		time.Sleep(5 * time.Millisecond)
		if n := other.frames.Load(); n != wantOther {
			t.Fatalf("round %d: waking one app drew %d frames in the other", round, n-wantOther)
		}
	}

	wantA, wantB := a.frames.Load()+1, b.frames.Load()+1
	Wakeup() // the package-level call reaches every running app
	waitFor(t, func() bool { return a.frames.Load() == wantA && b.frames.Load() == wantB })

	a.cancel()
	select {
	case err := <-a.done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled app returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled app did not stop")
	}
	wantB = b.frames.Load() + 1
	b.app.Wakeup()
	waitFor(t, func() bool { return b.frames.Load() == wantB })

	b.cancel()
	<-b.done
	running.mu.RLock()
	n := len(running.apps)
	running.mu.RUnlock()
	if n != 0 {
		t.Fatalf("%d apps still registered after both returned", n)
	}
}

// A context that is already cancelled stops before drawing anything.
func TestAppRunWithCancelledContext(t *testing.T) {
	b := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 20, 5))
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	drew := false
	err = NewApp(term).Run(ctx, func(*Frame, *Event) bool { drew = true; return true })
	if !errors.Is(err, context.Canceled) || drew {
		t.Fatalf("Run = %v, drew = %v", err, drew)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(2 * time.Millisecond)
	}
}
