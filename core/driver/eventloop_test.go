//go:build unix

package driver

import (
	"io"
	"runtime"
	"sync"
	"testing"
	"time"
)

// pipeIO is a portable terminal whose input arrives when the test writes it,
// a write at a time, and whose size the test changes.
type pipeIO struct {
	r  *io.PipeReader
	w  *io.PipeWriter
	mu sync.Mutex
	w_ uint16
	h_ uint16
}

func newPipeIO() *pipeIO {
	r, w := io.Pipe()
	return &pipeIO{r: r, w: w, w_: 80, h_: 24}
}
func (p *pipeIO) Read(b []byte) (int, error)  { return p.r.Read(b) }
func (p *pipeIO) Write(b []byte) (int, error) { return len(b), nil }
func (p *pipeIO) Size() (uint16, uint16, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.w_, p.h_, nil
}
func (p *pipeIO) resize(w, h uint16) {
	p.mu.Lock()
	p.w_, p.h_ = w, h
	p.mu.Unlock()
}

func startPortable(t *testing.T) (*Backend, *pipeIO) {
	t.Helper()
	pio := newPipeIO()
	b := NewPortableBackend(pio)
	b.StartEventLoop()
	t.Cleanup(func() { _ = b.Close(); pio.w.Close() })
	return b, pio
}

// nextEvent waits for an event, generously: CI runners stall.
func nextEvent(t *testing.T, b *Backend) Event {
	t.Helper()
	select {
	case ev := <-b.Events():
		return ev
	case <-time.After(3 * time.Second):
		t.Fatal("no event within 3s")
		return Event{}
	}
}

// noEvent asserts that nothing arrives for d.
func noEvent(t *testing.T, b *Backend, d time.Duration, why string) {
	t.Helper()
	select {
	case ev := <-b.Events():
		t.Fatalf("%s, got %+v", why, ev)
	case <-time.After(d):
	}
}

// eventLoopCases are what both the portable and the TTY loop must do with
// the bytes a person's keys produce, written as separate reads.
func checkEscHandling(t *testing.T, b *Backend, write func(string)) {
	t.Helper()
	// A lone ESC is Esc, but only once the timeout has passed: before it,
	// the rest of a sequence may still come.
	write("\x1b")
	noEvent(t, b, escTimeout/3, "Esc was reported before the timeout")
	if ev := nextEvent(t, b); ev.Type != EventKey || ev.Key.Type != KeyEsc {
		t.Fatalf("lone ESC gave %+v", ev)
	}

	// ESC then "[A" in a separate read, within the timeout, is one arrow key —
	// not Esc, '[' and 'A'.
	write("\x1b")
	write("[A")
	if ev := nextEvent(t, b); ev.Type != EventKey || ev.Key.Type != KeyArrowUp {
		t.Fatalf("a split ESC [ A gave %+v, want the up arrow", ev)
	}
	noEvent(t, b, 4*escTimeout, "a split sequence left something behind")

	// Over a slow link the rest of a sequence can come later than the
	// timeout, but once the introducer has arrived it is not a lone ESC: the
	// loop waits for the rest instead of giving up Esc, '[' and 'A'.
	write("\x1b[")
	noEvent(t, b, 4*escTimeout, "half a sequence was reported")
	write("A")
	if ev := nextEvent(t, b); ev.Type != EventKey || ev.Key.Type != KeyArrowUp {
		t.Fatalf("ESC [ then, later, A gave %+v, want the up arrow", ev)
	}

	// Ordinary keys come through in order.
	write("hi")
	for _, want := range "hi" {
		if ev := nextEvent(t, b); ev.Key.Type != KeyRune || ev.Key.Ch != want {
			t.Fatalf("typed %q, got %+v", want, ev)
		}
	}
}

func TestPortableLoopEscAndSplitSequences(t *testing.T) {
	b, pio := startPortable(t)
	checkEscHandling(t, b, func(s string) {
		if _, err := pio.w.Write([]byte(s)); err != nil {
			t.Fatal(err)
		}
	})
}

// A size change is reported once; an unchanged size is not reported.
func TestPortableLoopReportsAResizeOnce(t *testing.T) {
	b, pio := startPortable(t)
	noEvent(t, b, 600*time.Millisecond, "an unchanged size was reported")
	pio.resize(100, 30)
	if ev := nextEvent(t, b); ev.Type != EventResize || ev.Resize.Width != 100 || ev.Resize.Height != 30 {
		t.Fatalf("got %+v, want a resize to 100x30", ev)
	}
	noEvent(t, b, 600*time.Millisecond, "one size change was reported twice")
	if w, h, _ := b.Size(); w != 100 || h != 30 {
		t.Errorf("Size() = %dx%d after the resize", w, h)
	}
}

// Closing the backend while the loop is blocked sending to a full channel
// ends the loop: nothing is left running once the input is gone too.
func TestPortableLoopEndsWhenClosedWhileBlocked(t *testing.T) {
	before := runtime.NumGoroutine()
	pio := newPipeIO()
	b := NewPortableBackend(pio)
	b.StartEventLoop()
	go func() {
		for i := 0; i < 300; i++ { // more than the 128-event channel holds
			if _, err := pio.w.Write([]byte("x")); err != nil {
				return
			}
		}
	}()
	time.Sleep(100 * time.Millisecond) // let the channel fill and the loop block
	_ = b.Close()
	pio.w.Close() // a reader blocked in Read ends when its input does
	deadline := time.Now().Add(3 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			t.Fatalf("goroutines %d, was %d:\n%s", runtime.NumGoroutine(), before, buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(10 * time.Millisecond)
	}
}
