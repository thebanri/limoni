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

// WantsFocus reports whether a widget built on Accessible takes part in Tab
// navigation: it has an ID, a role a user acts on, and is not disabled. The
// frame registers such a widget for focus itself, so a custom widget that
// embeds Accessible need not call RegisterFocus — forgetting to was how Tab
// used to skip them.
func (a Accessible) WantsFocus() bool {
	if a.ID == "" || a.State&accessibility.StateDisabled != 0 {
		return false
	}
	switch a.Role {
	case accessibility.RoleButton, accessibility.RoleCheckbox, accessibility.RoleInput,
		accessibility.RoleList, accessibility.RoleTable, accessibility.RoleSlider,
		accessibility.RoleRadioButton, accessibility.RoleTree, accessibility.RoleTabList:
		return true
	}
	return false
}
