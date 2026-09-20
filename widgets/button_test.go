package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestButtonDrawsAndDescribesItself(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 1)
	buf := buffer.NewBuffer(area)
	var clicked func()
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterClick = func(_ cell.Rect, h func()) { clicked = h }
	pressed := 0
	b := Button{ID: "save", Label: "Save", OnPress: func() { pressed++ }}
	b.Draw(ctx, buf)
	if got := rowText(buf, 20); got != "[ Save ]" {
		t.Fatalf("drawn %q", got)
	}
	if clicked == nil {
		t.Fatal("no click registered")
	}
	clicked()
	if pressed != 1 {
		t.Fatalf("OnPress ran %d times", pressed)
	}
	node := b.AccessibilityNode(area, true)
	if node.Role != accessibility.RoleButton || node.Label != "Save" || node.State&accessibility.StateFocused == 0 {
		t.Fatalf("node %+v", node)
	}

	disabled := Button{ID: "go", Label: "Go", OnPress: func() { pressed++ }, Disabled: true}
	clicked = nil
	disabled.Draw(ctx, buffer.NewBuffer(area))
	if clicked != nil {
		t.Fatal("a disabled button registered a click")
	}
	if disabled.AccessibilityNode(area, false).State&accessibility.StateDisabled == 0 {
		t.Fatal("disabled state missing from the tree")
	}
}
