package runtime

import (
	"context"
	"github.com/thebanri/limoni/core/backend"
	"testing"
)

func TestProgramSendBackend(t *testing.T) {
	model := &testModel{quitOn: KeyPressMsg{Key: backend.KeyEvent{Type: backend.KeyEsc}}}
	p := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()
	if err := p.SendBackend(ctx, backend.Event{Type: backend.EventKey, Key: backend.KeyEvent{Type: backend.KeyEsc}}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
