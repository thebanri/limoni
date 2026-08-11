// Package accessibility contains semantic UI metadata independent of rendering.
package accessibility

import "github.com/thebanri/limoni/core/cell"

type Role uint8

const (
	RoleGeneric Role = iota
	RoleButton
	RoleCheckbox
	RoleInput
	RoleList
	RoleListItem
	RoleTable
	RoleDialog
	RoleProgress
	RoleImage
)

type NodeState uint32

const (
	StateFocused NodeState = 1 << iota
	StateDisabled
	StateSelected
	StateExpanded
	StateChecked
	StateBusy
	StateInvalid
)

type AccessibilityNode struct {
	ID          string
	Role        Role
	Label       string
	Description string
	Value       string
	State       NodeState
	Bounds      cell.Rect
	Children    []AccessibilityNode
}

// Provider is an optional widget capability for automatic semantic node
// registration during rendering.
type Provider interface {
	AccessibilityNode(bounds cell.Rect, focused bool) AccessibilityNode
}

func (n *AccessibilityNode) AddChild(child AccessibilityNode) { n.Children = append(n.Children, child) }

func (n AccessibilityNode) Find(id string) *AccessibilityNode {
	if n.ID == id {
		return &n
	}
	for _, child := range n.Children {
		if found := child.Find(id); found != nil {
			return found
		}
	}
	return nil
}

type Mode struct {
	HighContrast  bool
	NoColor       bool
	ASCIIOnly     bool
	ReducedMotion bool
	ScreenReader  bool
	NoMouse       bool
}

func (m Mode) Normalize() Mode {
	if m.ScreenReader {
		m.NoMouse = true
	}
	return m
}

// ShouldAnimate reports whether time-based transitions may run.
func (m Mode) ShouldAnimate() bool { return !m.Normalize().ReducedMotion }

// AllowsMouse reports whether pointer interaction is available.
func (m Mode) AllowsMouse() bool { return !m.Normalize().NoMouse }

// TextFallback returns an ASCII-safe representation when ASCIIOnly is enabled.
func (m Mode) TextFallback(text string) string {
	if !m.Normalize().ASCIIOnly {
		return text
	}
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if r < 128 {
			result = append(result, r)
		} else {
			result = append(result, '?')
		}
	}
	return string(result)
}
