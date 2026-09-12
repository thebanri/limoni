package limoni_test

import (
	"context"
	"testing"

	"github.com/thebanri/limoni"
)

// countingModel exercises the declarative runtime purely through the root
// package aliases, so the test fails if a re-exported type ever drifts from
// core/engine.
type countingModel struct {
	seen    []limoni.Msg
	quitOn  limoni.Msg
	initCmd []limoni.Cmd
}

func (m *countingModel) Init() []limoni.Cmd { return m.initCmd }

func (m *countingModel) Update(msg limoni.Msg) limoni.UpdateResult {
	m.seen = append(m.seen, msg)
	return limoni.UpdateResult{Redraw: true, Quit: m.quitOn != nil && msg == m.quitOn}
}

func (m *countingModel) View(*limoni.Frame) {}

func TestNewProgramRunsModelThroughRootAliases(t *testing.T) {
	model := &countingModel{quitOn: "done"}
	model.initCmd = []limoni.Cmd{
		func(context.Context) limoni.Msg { return "first" },
		func(context.Context) limoni.Msg { return "done" },
	}

	program := limoni.NewProgram(model)
	if program == nil {
		t.Fatal("NewProgram returned nil")
	}
	if err := program.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(model.seen) != 2 || model.seen[0] != "first" || model.seen[1] != "done" {
		t.Fatalf("model.seen = %v, want [first done]", model.seen)
	}
}

func TestNewProgramAcceptsProgramOptions(t *testing.T) {
	model := &countingModel{quitOn: "stop"}
	model.initCmd = []limoni.Cmd{func(context.Context) limoni.Msg { return "stop" }}

	program := limoni.NewProgram(model,
		limoni.WithMessageQueue(8),
		limoni.WithCommandQueue(8),
		limoni.WithProgramFPS(60),
		limoni.WithProgramCatchCtrlC(true),
		limoni.WithoutProgramQuitKeys(),
		limoni.WithPanicHandler(func(any) {}),
	)
	if err := program.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(model.seen) != 1 {
		t.Fatalf("model.seen = %v, want one message", model.seen)
	}
}

func TestMessageFromEventMapsKeyPress(t *testing.T) {
	event := limoni.Event{Type: limoni.EventKey}
	event.Key.Ch = 'a'

	msg := limoni.MessageFromEvent(event)
	press, ok := msg.(limoni.KeyPressMsg)
	if !ok {
		t.Fatalf("MessageFromEvent returned %T, want limoni.KeyPressMsg", msg)
	}
	if press.Key.Ch != 'a' {
		t.Fatalf("press.Key.Ch = %q, want 'a'", press.Key.Ch)
	}
}

// TestProgramRunCancelsWithContext documents that the declarative runtime is
// already context-aware, unlike immediate-mode Run.
func TestProgramRunCancelsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model := &countingModel{}
	if err := limoni.NewProgram(model).Run(ctx); err != nil && err != context.Canceled {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
}
