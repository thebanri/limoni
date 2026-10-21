package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Checkbox is an interactive box that can be ticked.
type Checkbox struct {
	ID           string
	Checked      *bool
	Label        string
	Style        cell.Style
	FocusedStyle cell.Style // unset: reverse bold
}

// Draw draws the checkbox as [ ] or [x]; when clicked it takes the focus and toggles its state.
func (cb Checkbox) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if cb.ID == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// Register as focusable
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(cb.ID)
	}

	isFocused := (ctx.FocusedID == cb.ID)

	// A click focuses the checkbox and flips it, registered as data so the
	// draw does not allocate a closure.
	if ctx.RegisterClickAction != nil {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: cb.ID, Toggle: cb.Checked})
	} else if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(cb.ID)
			}
			if cb.Checked != nil {
				*cb.Checked = !*cb.Checked
			}
		})
	}

	// Merge the styles
	textStyle := ctx.Style.Merge(cb.Style)
	if isFocused {
		textStyle = textStyle.Merge(focusStyle(cb.FocusedStyle))
	}

	// Prepare the [ ] or [x] state text
	prefix := "[ ] "
	if cb.Checked != nil && *cb.Checked {
		prefix = "[x] "
	}

	// Prefix and label written separately: concatenating them allocated on
	// every frame.
	n := buf.SetStringWithin(ctx.Area.X, ctx.Area.Y, prefix, textStyle, ctx.Area.Width)
	if n < ctx.Area.Width {
		buf.SetStringWithin(ctx.Area.X+n, ctx.Area.Y, cb.Label, textStyle, ctx.Area.Width-n)
	}
}

// SizeHint returns the single-row area the checkbox needs, with its width.
func (cb Checkbox) SizeHint(maxArea cell.Rect) (width, height uint16) {
	neededW := uint16(cell.StringWidth(cb.Label) + 4)
	if neededW > maxArea.Width {
		neededW = maxArea.Width
	}
	return neededW, 1
}

// AccessibilityNode returns the semantic node description for Checkbox.
func (cb Checkbox) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.Node {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	if cb.Checked != nil && *cb.Checked {
		state |= accessibility.StateChecked
	}
	val := "false"
	if cb.Checked != nil && *cb.Checked {
		val = "true"
	}
	return accessibility.Node{
		ID:     cb.ID,
		Role:   accessibility.RoleCheckbox,
		Label:  cb.Label,
		Value:  val,
		State:  state,
		Bounds: bounds,
	}
}
