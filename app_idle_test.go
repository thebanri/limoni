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

// Without WithFPS an App draws only when something happens, so it costs
// nothing while idle. One of those things is the terminal answering the
// capability probe after the first frame, over SSH especially: the frame on
// screen may have been encoded for another terminal, and without input
// nothing else would draw it again.
func TestRunIdlesAndDrawsWhenTheProbeIsAnswered(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "")
	in := make(chan []byte, 4)
	b := driver.NewPortableBackend(&feedIO{in: in})
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	var frames atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- NewApp(term).Run(ctx, func(f *Frame, ev *Event) bool {
			frames.Add(1)
			return true
		})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Run = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Run did not stop")
		}
		_ = term.Close()
	})

	waitFor(t, func() bool { return frames.Load() == 1 })
	time.Sleep(150 * time.Millisecond)
	if n := frames.Load(); n != 1 {
		t.Fatalf("an idle App drew %d frames; want only the first", n)
	}
	in <- []byte("\x1b[?62;22c") // DA1, the sentinel
	waitFor(t, func() bool { return frames.Load() == 2 })
}

// feedIO delivers input when the test sends it.
type feedIO struct {
	in      chan []byte
	pending []byte
}

func (f *feedIO) Read(p []byte) (int, error) {
	if len(f.pending) == 0 {
		f.pending = <-f.in
	}
	n := copy(p, f.pending)
	f.pending = f.pending[n:]
	return n, nil
}

func (f *feedIO) Write(p []byte) (int, error)   { return len(p), nil }
func (f *feedIO) Size() (uint16, uint16, error) { return 40, 10, nil }
