package runtime

import (
	"context"
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/terminal"
)

func TestProgramBackendMessageInjection(t *testing.T) {
	model := &testModel{quitOn: KeyPressMsg{Key: backend.KeyEvent{Type: backend.KeyEsc}}}
	program := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- program.Run(ctx) }()
	if err := program.Send(ctx, MessageFromBackend(backend.Event{Type: backend.EventKey, Key: backend.KeyEvent{Type: backend.KeyEsc}})); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestProgramViewDelegation(t *testing.T) {
	model := &viewModel{}
	program := New(WithModel(model))
	program.View(terminal.NewFrame(nil, terminal.NewFocusManager()))
	if !model.viewed {
		t.Fatal("expected Program.View to delegate to the model")
	}
}

type viewModel struct{ viewed bool }

func (m *viewModel) Init() []Cmd             { return nil }
func (m *viewModel) Update(Msg) UpdateResult { return UpdateResult{} }
func (m *viewModel) View(*terminal.Frame)    { m.viewed = true }
