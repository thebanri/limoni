package engine

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// idleModel counts frames and x presses. Update never asks for a redraw.
type idleModel struct {
	views   atomic.Int32
	presses atomic.Int32 // x keys Update has seen
	drawn   atomic.Int32 // presses the last frame showed
}

func (m *idleModel) Init() []Cmd { return nil }

func (m *idleModel) Update(msg Msg) UpdateResult {
	if k, ok := msg.(KeyPressMsg); ok {
		switch k.Key.Ch {
		case 'q':
			return UpdateResult{Quit: true}
		case 'x':
			m.presses.Add(1)
		}
	}
	return UpdateResult{}
}

func (m *idleModel) View(f *terminal.Frame) {
	m.views.Add(1)
	n := m.presses.Load()
	m.drawn.Store(n)
	f.Buffer.SetString(0, 0, strconv.Itoa(int(n)), cell.Style{})
}

// startIdle runs a Program on a terminal whose input the test feeds, and
// waits for its first frame.
func startIdle(t *testing.T, opts ...Option) (*idleModel, *chanTerminalIO) {
	t.Helper()
	io := newChanTerminalIO(20, 5)
	b := driver.NewPortableBackend(io)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	m := &idleModel{}
	p := New(append([]Option{WithModel(m)}, opts...)...)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	done := make(chan error, 1)
	go func() { done <- p.RunTerminal(ctx, term, b) }()
	t.Cleanup(func() {
		io.in <- []byte("q")
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("RunTerminal: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("RunTerminal did not return after q")
		}
		cancel()
	})
	waitUntil(t, func() bool { return m.views.Load() >= 1 })
	return m, io
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 2s")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// Without a frame rate, a Program with nothing to do draws nothing. It used to
// draw thirty frames a second of the same screen, which is what an idle
// Program cost in CPU (~0.7% of a core, 200 wakeups a second, measured).
func TestRunTerminalIdlesWithoutAFrameRate(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	m, _ := startIdle(t)
	time.Sleep(300 * time.Millisecond)
	if n := m.views.Load(); n != 1 {
		t.Fatalf("an idle Program drew %d frames in 300ms; want only the first", n)
	}
}

// An Update that changes the model but does not ask for a redraw still reaches
// the screen: the 30 fps loop used to draw it, and a model that relied on
// that must keep working.
func TestRunTerminalDrawsAChangeThatDidNotAskForRedraw(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	m, io := startIdle(t)
	io.in <- []byte("x")
	waitUntil(t, func() bool { return m.drawn.Load() == 1 })
	// One frame for it, not a loop.
	time.Sleep(200 * time.Millisecond)
	if n := m.views.Load(); n != 2 {
		t.Fatalf("drew %d frames; want the first and one for the change", n)
	}
}

// Such a change is drawn at once when the screen has been still for a frame,
// which is what makes a key feel immediate; a stream of them is held to one
// frame per idleFrame. The frame is stretched to a second here, so the two
// cases differ by hundreds of milliseconds rather than by scheduler jitter.
func TestRunTerminalPacesChangesThatDidNotAskForRedraw(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	frame := idleFrame
	t.Cleanup(func() { idleFrame = frame }) // after RunTerminal has returned
	idleFrame = time.Second
	m, io := startIdle(t)
	time.Sleep(idleFrame + 100*time.Millisecond)

	sent := time.Now()
	io.in <- []byte("x")
	waitUntil(t, func() bool { return m.drawn.Load() == 1 })
	if d := time.Since(sent); d > idleFrame/2 {
		t.Fatalf("a change after a still second took %v to draw; want it at once", d)
	}

	io.in <- []byte("xxx") // within a frame of the last one
	time.Sleep(idleFrame / 2)
	if n := m.drawn.Load(); n != 1 {
		t.Fatalf("changes within a frame of the last were drawn at once (showing %d)", n)
	}
	waitUntil(t, func() bool { return m.drawn.Load() == 4 })
	time.Sleep(100 * time.Millisecond)
	if n := m.views.Load(); n != 3 {
		t.Fatalf("drew %d frames; want the first, one for x, one for xxx", n)
	}
}

// WithFPS still means what it says: continuous frames, for a View that
// changes on its own.
func TestRunTerminalKeepsTheFrameRateItWasGiven(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	m, _ := startIdle(t, WithFPS(100))
	time.Sleep(300 * time.Millisecond)
	if n := m.views.Load(); n < 10 {
		t.Fatalf("drew %d frames in 300ms at 100 fps", n)
	}
}

// Answers to the capability probe can land after the first frame, over SSH
// especially. The 30 fps loop used to redraw with them within a frame; a
// Program that draws only on events must draw once more when they arrive.
func TestRunTerminalDrawsWhenTheProbeIsAnswered(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "")
	m, io := startIdle(t)
	time.Sleep(100 * time.Millisecond)
	if n := m.views.Load(); n != 1 {
		t.Fatalf("drew %d frames before the answer; want 1", n)
	}
	io.in <- []byte("\x1b[?62;22c") // DA1, the sentinel
	waitUntil(t, func() bool { return m.views.Load() == 2 })
}
