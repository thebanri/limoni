package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// Options only set fields, which is the kind of wiring that breaks silently.
func TestOptionsSetWhatTheyName(t *testing.T) {
	var opts programOptions
	for _, o := range []Option{
		WithMessageQueue(8), WithCommandQueue(9), WithFPS(45),
		WithCatchCtrlC(true), WithAltScreen(),
	} {
		o(&opts)
	}
	if opts.messageQueue != 8 || opts.commandQueue != 9 {
		t.Errorf("queues = %d/%d", opts.messageQueue, opts.commandQueue)
	}
	if opts.fps != 45 {
		t.Errorf("fps = %d", opts.fps)
	}
	if !opts.catchCtrlC || !opts.altScreen {
		t.Errorf("flags = %+v", opts)
	}

	// Sizes that make no sense are ignored rather than accepted: a zero
	// queue would deadlock the runtime, a zero FPS would divide by zero.
	var zero programOptions
	zero.messageQueue, zero.commandQueue, zero.fps = 64, 64, 30
	for _, o := range []Option{WithMessageQueue(0), WithCommandQueue(-1), WithFPS(0)} {
		o(&zero)
	}
	if zero.messageQueue != 64 || zero.commandQueue != 64 || zero.fps != 30 {
		t.Errorf("a nonsense size was accepted: %+v", zero)
	}

	var quitKeys programOptions
	WithoutDefaultQuitKeys()(&quitKeys)
	if !quitKeys.catchCtrlC {
		t.Error("WithoutDefaultQuitKeys should hand Ctrl+C to the model")
	}

	obs := &recordingObserver{}
	var withObs programOptions
	WithObserver(obs)(&withObs)
	if withObs.observer != obs {
		t.Error("WithObserver did not attach the observer")
	}
}

type recordingObserver struct {
	mu       sync.Mutex
	messages []Msg
	frames   int
	panics   []any
}

func (o *recordingObserver) Message(step uint64, msg Msg) {
	o.mu.Lock()
	o.messages = append(o.messages, msg)
	o.mu.Unlock()
}

func (o *recordingObserver) Frame(step uint64, tree []accessibility.AccessibilityNode) {
	o.mu.Lock()
	o.frames++
	o.mu.Unlock()
}

func (o *recordingObserver) Panic(step uint64, value any) {
	o.mu.Lock()
	o.panics = append(o.panics, value)
	o.mu.Unlock()
}

func (o *recordingObserver) seen() []Msg {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]Msg(nil), o.messages...)
}

// NowCmd exists so that a model never reads the clock itself: the time
// arrives as a message, and messages are what a recording replays.
// waitForCond polls until cond holds, so a test never sleeps a fixed time.
func waitForCond(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition never became true")
}

func TestNowCmdDeliversTheTimeAsAMessage(t *testing.T) {
	before := time.Now()
	msg := NowCmd()(context.Background())
	tm, ok := msg.(TimeMsg)
	if !ok {
		t.Fatalf("NowCmd produced %T, want TimeMsg", msg)
	}
	if tm.Time.Before(before) || tm.Time.After(time.Now()) {
		t.Errorf("TimeMsg carries %v, outside the window it was created in", tm.Time)
	}
}

// Context is what a command is given: it must reach the program that made it,
// and must not panic when it belongs to no program at all.
func TestRuntimeContextSendAndRedraw(t *testing.T) {
	model := &testModel{}
	p := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()

	rc := Context{Context: ctx, program: p}
	if err := rc.Send("hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	rc.RequestRedraw()

	waitForCond(t, func() bool {
		for _, m := range model.messages() {
			if m == Msg("hello") {
				return true
			}
		}
		return false
	})

	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}

	// A zero Context belongs to no program. It reports that rather than
	// panicking, because commands outlive the program that made them.
	var orphan Context
	if err := orphan.Send("x"); !errors.Is(err, context.Canceled) {
		t.Errorf("orphan Send = %v, want context.Canceled", err)
	}
	orphan.RequestRedraw() // must not panic
}

func TestDrawRefusesMissingPiecesAndDrawsOtherwise(t *testing.T) {
	if err := (&Program{}).Draw(nil); err == nil || !strings.Contains(err.Error(), "model") {
		t.Errorf("a program with no model should say so: %v", err)
	}
	p := New(WithModel(&testModel{}))
	if err := p.Draw(nil); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Errorf("Draw with no terminal should say so: %v", err)
	}

	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 20, 5)
	b := driver.NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = term.Close() }()

	if err := p.Draw(term); err != nil {
		t.Fatalf("Draw: %v", err)
	}
}

func TestRunTerminalRefusesMissingPieces(t *testing.T) {
	ctx := context.Background()
	if err := (&Program{}).RunTerminal(ctx, nil, nil); err == nil || !strings.Contains(err.Error(), "model") {
		t.Errorf("no model: %v", err)
	}
	p := New(WithModel(&testModel{}))
	if err := p.RunTerminal(ctx, nil, nil); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Errorf("no terminal or backend: %v", err)
	}
}

// The observer is how session recording and the debugger see a run: every
// message that reaches Update, and every frame a View produced.
func TestObserverSeesMessagesAndFrames(t *testing.T) {
	obs := &recordingObserver{}
	model := &testModel{}
	p := New(WithModel(model), WithObserver(obs))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()

	if err := p.Send(ctx, "one"); err != nil {
		t.Fatal(err)
	}
	waitForCond(t, func() bool {
		for _, m := range obs.seen() {
			if m == Msg("one") {
				return true
			}
		}
		return false
	})

	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run: %v", err)
	}
}

// Every driver event a model can receive, and the one that maps to nothing.
func TestMessageFromDriverCoversEveryEventKind(t *testing.T) {
	key := driver.KeyEvent{Type: driver.KeyRune, Ch: 'q'}
	cases := []struct {
		name string
		in   driver.Event
		want Msg
	}{
		{"key", driver.Event{Type: driver.EventKey, Key: key}, KeyPressMsg{Key: key}},
		{"resize", driver.Event{Type: driver.EventResize, Resize: driver.ResizeEvent{Width: 80, Height: 24}}, ResizeMsg{Width: 80, Height: 24}},
		{"focus gained", driver.Event{Type: driver.EventFocus, Focus: driver.FocusEvent{Gained: true}}, FocusMsg{Gained: true}},
		{"focus lost", driver.Event{Type: driver.EventFocus}, BlurMsg{}},
		{"paste", driver.Event{Type: driver.EventPaste, Paste: driver.PasteEvent{Text: "hi"}}, PasteMsg{Text: "hi"}},
		{"wheel up", driver.Event{Type: driver.EventMouse, Mouse: driver.MouseEvent{Button: driver.MouseScrollUp}}, MouseWheelMsg{DeltaY: 1}},
		{"wheel down", driver.Event{Type: driver.EventMouse, Mouse: driver.MouseEvent{Button: driver.MouseScrollDown}}, MouseWheelMsg{DeltaY: -1}},
		{"release", driver.Event{Type: driver.EventMouse, Mouse: driver.MouseEvent{Button: driver.MouseRelease, X: 3, Y: 4}}, MouseReleaseMsg{Position: cell.Point{X: 3, Y: 4}, Button: driver.MouseRelease}},
		{"press", driver.Event{Type: driver.EventMouse, Mouse: driver.MouseEvent{Button: driver.MouseLeft, X: 1, Y: 2}}, MousePressMsg{Position: cell.Point{X: 1, Y: 2}, Button: driver.MouseLeft}},
	}
	for _, c := range cases {
		if got := MessageFromDriver(c.in); got != c.want {
			t.Errorf("%s: got %#v, want %#v", c.name, got, c.want)
		}
	}

	// An event kind with no model-level meaning produces no message at all,
	// rather than a zero-valued one the model would have to filter.
	if got := MessageFromDriver(driver.Event{Type: driver.EventType(200)}); got != nil {
		t.Errorf("an unknown event produced %#v", got)
	}
	// The old name is still the same function.
	if got := MessageFromBackend(driver.Event{Type: driver.EventKey, Key: key}); got != (KeyPressMsg{Key: key}) {
		t.Errorf("MessageFromBackend = %#v", got)
	}
}

// RunTerminal is the batteries-included entry point: it owns the backend,
// feeds input in, draws frames out, and returns when the model quits.
func TestRunTerminalDrivesAModelToQuit(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO([]byte("q"), 20, 5)
	b := driver.NewPortableBackend(io)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}

	quitOnQ := &quitKeyModel{key: 'q'}
	p := New(WithModel(quitOnQ), WithFPS(120))

	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { done <- p.RunTerminal(ctx, term, b) }()

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("RunTerminal: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("RunTerminal did not return after the model quit")
	}

	if !quitOnQ.sawKey() {
		t.Error("the key never reached the model")
	}
	if len(io.Output()) == 0 {
		t.Error("RunTerminal drew nothing")
	}
}

type quitKeyModel struct {
	mu   sync.Mutex
	key  rune
	seen bool
}

func (m *quitKeyModel) Init() []Cmd { return nil }

func (m *quitKeyModel) Update(msg Msg) UpdateResult {
	if k, ok := msg.(KeyPressMsg); ok && k.Key.Ch == m.key {
		m.mu.Lock()
		m.seen = true
		m.mu.Unlock()
		return UpdateResult{Quit: true}
	}
	return UpdateResult{Redraw: true}
}

func (m *quitKeyModel) View(f *terminal.Frame) {
	f.Buffer.SetString(0, 0, "running", cell.Style{})
}

func (m *quitKeyModel) sawKey() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seen
}
