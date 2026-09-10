package component

import (
	"unicode"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// transformComponent applies a per-rune transformation function to all cells
// written by its child. This is a zero-allocation post-draw cell mutator:
// the child draws normally, then we sweep the bounding rect and mutate
// cell.Content in-place.
//
// Equivalent to Lipgloss's Transform(func(string) string), but operates
// at the cell level instead of string level for zero-allocation.
type transformComponent struct {
	child     Component
	transform func(rune) rune
}

// Transform wraps a child component and applies fn to every non-zero rune
// in its bounding area after the child has drawn. The transform runs
// in-place on the buffer cells with no heap allocation.
//
// Example:
//
//	limoni.Transform(limoni.Label("hello"), unicode.ToUpper) // renders "HELLO"
func Transform(child Component, fn func(rune) rune) Component {
	return &transformComponent{child: child, transform: fn}
}

// Uppercase wraps a child component and transforms all text to uppercase.
func Uppercase(child Component) Component {
	return Transform(child, unicode.ToUpper)
}

// Lowercase wraps a child component and transforms all text to lowercase.
func Lowercase(child Component) Component {
	return Transform(child, unicode.ToLower)
}

// Mask wraps a child component and replaces all visible text with a mask rune (e.g. '•' for passwords).
func Mask(child Component, mask rune) Component {
	return Transform(child, func(r rune) rune {
		if r > ' ' { // only mask visible characters, leave whitespace
			return mask
		}
		return r
	})
}

func (t *transformComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	t.child.Draw(ctx, buf)

	// In-place sweep: mutate cell content within the bounding area.
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			c := buf.Get(x, y)
			if c != nil && c.Content != 0 {
				c.Content = t.transform(c.Content)
			}
		}
	}
}

func (t *transformComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	return t.child.LayoutInfo(maxArea)
}

func (t *transformComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return t.child.SizeHint(maxArea)
}

func (t *transformComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return DispatchEvent(t.child, ctx, ev)
}
