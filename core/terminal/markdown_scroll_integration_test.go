package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

func TestMarkdownScrollThroughTerminalMouseRouter(t *testing.T) {
	offset := 0
	frame := NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 30, 4)), NewFocusManager())
	term := &Terminal{frame: frame}
	frame.RenderWidget(&widgets.Markdown{
		ID: "demo_markdown", Content: "one\ntwo\nthree\nfour\nfive\nsix", ScrollOffset: &offset,
	}, cell.NewRect(1, 1, 20, 3))

	if !term.RouteMouseEvent(backend.MouseEvent{X: 4, Y: 2, Button: backend.MouseLeft}) {
		t.Fatal("markdown click was not routed")
	}
	if term.FocusManager().Focused() != "demo_markdown" {
		t.Fatalf("focused widget = %q, want demo_markdown", term.FocusManager().Focused())
	}
	if !term.RouteMouseEvent(backend.MouseEvent{X: 4, Y: 0, Button: backend.MouseLeft, Drag: true}) {
		t.Fatal("markdown drag was not captured")
	}
	if offset == 0 {
		t.Fatal("markdown drag did not change scroll offset")
	}
	term.RouteMouseEvent(backend.MouseEvent{X: 4, Y: 0, Button: backend.MouseRelease})

	term.RouteMouseEvent(backend.MouseEvent{X: 4, Y: 2, Button: backend.MouseScrollDown})
	if offset == 0 {
		t.Fatal("markdown wheel did not change scroll offset")
	}
}
