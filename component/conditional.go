package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// emptyComponent is a zero-sized, no-op component used as a safe fallback.
type emptyComponent struct{}

var empty = &emptyComponent{}

// Empty returns a zero-sized no-op component.
func Empty() Component {
	return empty
}

func (e *emptyComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {}

func (e *emptyComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	return LayoutProps{MinWidth: 0, MinHeight: 0, Flex: 0}
}

func (e *emptyComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return 0, 0
}

func (e *emptyComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return false
}

// When evaluates condition and returns then if true, or the optional otherwise component if false.
// If the selected component is nil or omitted, Empty() is returned.
// Evaluated immediately with zero wrapping overhead.
func When(condition bool, then Component, otherwise ...Component) Component {
	if condition {
		if then != nil {
			return then
		}
		return empty
	}
	if len(otherwise) > 0 && otherwise[0] != nil {
		return otherwise[0]
	}
	return empty
}

// Match inspects value against cases map and returns the matching component.
// If no case matches, defaultCase is returned (or Empty() if omitted or nil).
// Evaluated immediately with zero wrapping overhead.
func Match[T comparable](value T, cases map[T]Component, defaultCase ...Component) Component {
	if cases != nil {
		if c, ok := cases[value]; ok && c != nil {
			return c
		}
	}
	if len(defaultCase) > 0 && defaultCase[0] != nil {
		return defaultCase[0]
	}
	return empty
}
