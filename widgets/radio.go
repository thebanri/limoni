package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// RadioButton is a radio button for choosing one option from a group.
type RadioButton struct {
	ID           string
	Selected     *string
	Value        string
	Label        string
	Style        cell.Style
	FocusedStyle cell.Style // unset: reverse bold
}

// Draw draws the radio button as ( ) or (*); when clicked it takes the focus and updates the group's selected value.
func (rb RadioButton) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if rb.ID == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// Register as focusable
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(rb.ID)
	}

	isFocused := (ctx.FocusedID == rb.ID)

	// On click, take the focus and update the selection
	if ctx.RegisterClickAction != nil {
		// Focus and select, registered as data: no allocation per frame.
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: rb.ID, Assign: rb.Selected, Value: rb.Value})
	} else if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(rb.ID)
			}
			if rb.Selected != nil {
				*rb.Selected = rb.Value
			}
		})
	}

	// Merge the styles
	textStyle := ctx.Style.Merge(rb.Style)
	if isFocused {
		textStyle = textStyle.Merge(focusStyle(rb.FocusedStyle))
	}

	// Prepare the ( ) or (*) state text
	prefix := "( ) "
	if rb.Selected != nil && *rb.Selected == rb.Value {
		prefix = "(*) "
	}

	buf.SetStringWithin(ctx.Area.X, ctx.Area.Y, prefix+rb.Label, textStyle, ctx.Area.Width)
}

// SizeHint returns the single-row area the radio button needs, with its width.
func (rb RadioButton) SizeHint(maxArea cell.Rect) (width, height uint16) {
	neededW := uint16(cell.StringWidth(rb.Label) + 4)
	if neededW > maxArea.Width {
		neededW = maxArea.Width
	}
	return neededW, 1
}

// AccessibilityNode returns the semantic node description for RadioButton.
func (rb RadioButton) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.Node {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	isSelected := rb.Selected != nil && *rb.Selected == rb.Value
	if isSelected {
		state |= accessibility.StateSelected
	}
	val := "false"
	if isSelected {
		val = "true"
	}
	return accessibility.Node{
		ID:     rb.ID,
		Role:   accessibility.RoleRadioButton,
		Label:  rb.Label,
		Value:  val,
		State:  state,
		Bounds: bounds,
	}
}
