package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
)

// Accessible is an opt-in semantic description for widgets that need an
// accessibility node without coupling their core state to the terminal.
type Accessible struct {
	ID          string
	Role        accessibility.Role
	Label       string
	Description string
	Value       string
	State       accessibility.NodeState
}

func (a Accessible) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := a.State
	if focused {
		state |= accessibility.StateFocused
	}
	return accessibility.AccessibilityNode{ID: a.ID, Role: a.Role, Label: a.Label, Description: a.Description, Value: a.Value, State: state, Bounds: bounds}
}
