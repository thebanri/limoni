package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/terminal"
)

type testModel struct {
	mu       sync.Mutex
	init     []Cmd
	updates  []Msg
	commands []Cmd
	quitOn   Msg
}

func (m *testModel) Init() []Cmd { return m.init }

func (m *testModel) Update(message Msg) UpdateResult {
	m.mu.Lock()
	m.updates = append(m.updates, message)
	commands := m.commands
	m.commands = nil
	quit := m.quitOn != nil && message == m.quitOn
	m.mu.Unlock()
	return UpdateResult{Commands: commands, Redraw: true, Quit: quit}
}

func (m *testModel) View(*terminal.Frame) {}

func (m *testModel) messages() []Msg {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Msg, len(m.updates))
	copy(result, m.updates)
	return result
}

func TestProgramDeterministicCommandOrdering(t *testing.T) {
	model := &testModel{quitOn: "third"}
	model.init = []Cmd{
		func(context.Context) Msg { return "first" },
		func(context.Context) Msg { return "second" },
		func(context.Context) Msg { return "third" },
	}
	program := New(WithModel(model))
	if err := program.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := model.messages()
	if len(got) != 3 || got[0] != "first" || got[1] != "second" || got[2] != "third" {
		t.Fatalf("messages = %v, want ordered command results", got)
	}
}

func TestProgramCommandCancellation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	model := &testModel{
		init: []Cmd{func(ctx context.Context) Msg {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return "late"
		}},
	}
	program := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- program.Run(ctx) }()
	<-started
	cancel()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("command did not observe cancellation")
	}
	select {
	case err := <-runDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancellation")
	}
	if got := model.messages(); len(got) != 0 {
		t.Fatalf("late messages = %v, want none", got)
	}
}

func TestProgramPanicRecoveryAndRedrawCoalescing(t *testing.T) {
	var panicValue any
	model := &testModel{
		quitOn: "quit",
		init: []Cmd{
			func(context.Context) Msg { panic("command panic") },
			func(context.Context) Msg { return "quit" },
		},
	}
	program := New(WithModel(model), WithPanicHandler(func(value any) { panicValue = value }))
	program.RequestRedraw()
	program.RequestRedraw()
	select {
	case <-program.Redraws():
	default:
		t.Fatal("expected a redraw request")
	}
	select {
	case <-program.Redraws():
		t.Fatal("redraw requests should be coalesced")
	default:
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := program.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if panicValue != "command panic" {
		t.Fatalf("panic value = %v, want command panic", panicValue)
	}
}

func TestProgramSendCancellation(t *testing.T) {
	program := New(WithModel(&testModel{}), WithMessageQueue(1))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := program.Send(ctx, "message"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send error = %v, want context.Canceled", err)
	}
}
