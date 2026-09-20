package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
)

// Button is a labelled push button: "[ Save ]". A click calls OnPress and
// focuses the button. Keyboard activation (Enter or Space while focused) is
// the application's to handle, since only it knows which key events reach
// it; Focused tells it whether this button has focus.
//
// OnPress is registered as it is, not wrapped, so drawing a button does not
// allocate — keep it a function built once, not a closure made in the draw
// function, or that closure is the allocation.
type Button struct {
	ID      string
	Label   string
	OnPress func()
	// Disabled buttons draw dimmed, ignore clicks and say so in the tree.
	Disabled bool

	Style        cell.Style
	FocusedStyle cell.Style // defaults to reverse video
}

// Draw draws the button and registers its click.
func (b Button) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	if b.ID != "" && ctx.RegisterFocus != nil && !b.Disabled {
		ctx.RegisterFocus(b.ID)
	}
	if !b.Disabled {
		if b.OnPress != nil && ctx.RegisterClick != nil {
			ctx.RegisterClick(area, b.OnPress)
		}
		if b.ID != "" && ctx.RegisterClickAction != nil {
			ctx.RegisterClickAction(area, cell.ClickAction{Focus: b.ID})
		}
	}

	style := ctx.Style.Merge(b.Style)
	switch {
	case b.Disabled:
		style.Modifier |= cell.ModifierDim
	case ctx.IsFocused(b.ID):
		fs := b.FocusedStyle
		if fs == (cell.Style{}) {
			fs = cell.Style{Modifier: cell.ModifierReverse | cell.ModifierBold}
		}
		style = style.Merge(fs)
	}
	n := buf.SetStringWithin(area.X, area.Y, "[ ", style, area.Width)
	n += buf.SetStringWithin(area.X+n, area.Y, b.Label, style, area.Width-min(n, area.Width))
	if n < area.Width {
		buf.SetStringWithin(area.X+n, area.Y, " ]", style, area.Width-n)
	}
}

// SizeHint is the label plus its brackets, one row high.
func (b Button) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	w := uint16(cell.StringWidth(b.Label) + 4)
	if w > maxArea.Width {
		w = maxArea.Width
	}
	return w, 1
}

// Measure reports the button's natural size.
func (b Button) Measure(maxArea cell.Rect) layout.Measure {
	w, h := b.SizeHint(maxArea)
	return layout.Measure{MinWidth: w, IdealWidth: w, MaxWidth: w, MinHeight: h, IdealHeight: h, MaxHeight: h}
}

// AccessibilityNode describes the button as a button with its label.
func (b Button) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	if b.Disabled {
		state |= accessibility.StateDisabled
	}
	return accessibility.AccessibilityNode{ID: b.ID, Role: accessibility.RoleButton, Label: b.Label, State: state, Bounds: bounds}
}
