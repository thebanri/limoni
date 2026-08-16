package bubbletea

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
)

type counterModel struct {
	count int
}

func (m *counterModel) Init() Cmd {
	return nil
}

func (m *counterModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case KeyMsg:
		if msg.String() == "+" {
			m.count++
		} else if msg.String() == "ctrl+c" {
			return m, Quit
		}
	}
	return m, nil
}

func (m *counterModel) View() string {
	return "count: " + string(rune('0'+m.count))
}

func TestBubbleTeaAdapterBasic(t *testing.T) {
	model := &counterModel{}
	adapter := &adapterModel{inner: model}

	// 1. Check init
	cmds := adapter.Init()
	if len(cmds) != 0 {
		t.Fatalf("expected 0 init cmds, got %d", len(cmds))
	}

	// 2. Simulate key press '+'
	keyMsg := runtime.KeyPressMsg{
		Key: backend.KeyEvent{
			Type: backend.KeyRune,
			Ch:   '+',
		},
	}
	res := adapter.Update(keyMsg)
	if !res.Redraw {
		t.Fatal("expected redraw = true")
	}
	if model.count != 1 {
		t.Fatalf("expected count = 1, got %d", model.count)
	}

	// 3. Render View to Frame
	frame := terminal.NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 20, 5)), terminal.NewFocusManager())
	adapter.View(frame)

	c := frame.Buffer.Get(0, 0)
	if c == nil || c.Content != 'c' {
		t.Fatalf("expected cell (0,0)='c', got %v", c)
	}
	c7 := frame.Buffer.Get(7, 0)
	if c7 == nil || c7.Content != '1' {
		t.Fatalf("expected cell (7,0)='1', got %v", c7)
	}

	// 4. Test Quit
	quitRes := adapter.Update(QuitMsg{})
	if !quitRes.Quit {
		t.Fatal("expected Quit = true")
	}
}

func TestBubbleTeaProgramExecution(t *testing.T) {
	model := &counterModel{}
	p := NewProgram(model)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := p.Run(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("program run unexpected error: %v", err)
	}
}
