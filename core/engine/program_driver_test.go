package engine

import (
	"context"
	"github.com/thebanri/limoni/core/driver"
	"testing"
)

func TestProgramSendBackend(t *testing.T) {
	model := &testModel{quitOn: KeyPressMsg{Key: driver.KeyEvent{Type: driver.KeyEsc}}}
	p := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()
	if err := p.SendBackend(ctx, driver.Event{Type: driver.EventKey, Key: driver.KeyEvent{Type: driver.KeyEsc}}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
