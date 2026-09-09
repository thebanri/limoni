package component

import (
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// dynamicComponent resolves its underlying component on-the-fly via a supplier function.
type dynamicComponent struct {
	supplier func() Component
}

// Dynamic constructs a reactive component whose child subtree is resolved at render time.
// This allows static component hierarchies to embed stateful, changing subviews without
// rebuilding the entire tree on every frame.
//
// Example:
//
//	limoni.Dynamic(func() limoni.Component {
//	    if m.loading {
//	        return limoni.Text("Loading...")
//	    }
//	    return dashboardView
//	})
func Dynamic(supplier func() Component) Component {
	return &dynamicComponent{supplier: supplier}
}

func (d *dynamicComponent) resolve() Component {
	if d.supplier == nil {
		return Empty()
	}
	comp := d.supplier()
	if comp == nil {
		return Empty()
	}
	return comp
}

func (d *dynamicComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	d.resolve().Draw(ctx, buf)
}

func (d *dynamicComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	return d.resolve().LayoutInfo(maxArea)
}

func (d *dynamicComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return d.resolve().SizeHint(maxArea)
}

func (d *dynamicComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return DispatchEvent(d.resolve(), ctx, ev)
}

// textRefComponent binds directly to a pointer to a string.
type textRefComponent struct {
	ptr   *string
	style cell.Style
}

// TextRef creates a dynamic, data-driven text component bound directly to a string variable pointer.
// On every frame, the text is read directly from the pointer with zero heap allocations.
//
// Example:
//
//	status := "Ready"
//	view := limoni.TextRef(&status)
//	// Later in an event or goroutine:
//	status = "Processing..." // view automatically renders the updated value!
func TextRef(ptr *string, style ...cell.Style) Component {
	st := cell.NewStyle()
	if len(style) > 0 {
		st = style[0]
	}
	return &textRefComponent{ptr: ptr, style: st}
}

func (t *textRefComponent) content() string {
	if t.ptr == nil {
		return ""
	}
	return *t.ptr
}

func (t *textRefComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	str := t.content()
	if str == "" {
		return
	}
	area := ctx.Area
	finalStyle := ctx.Style.Merge(t.style)

	lines := strings.Split(str, "\n")
	for row, line := range lines {
		if uint16(row) >= area.Height {
			break
		}
		col := uint16(0)
		for _, r := range line {
			rw := cell.RuneWidth(r)
			if col+uint16(rw) > area.Width {
				break
			}
			buf.SetCell(area.X+col, area.Y+uint16(row), cell.Cell{
				Content: r,
				Style:   finalStyle,
			})
			col += uint16(rw)
		}
	}
}

func (t *textRefComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	str := t.content()
	if str == "" {
		return LayoutProps{MinWidth: 0, MinHeight: 0, Flex: 0}
	}
	lines := strings.Split(str, "\n")
	var maxW uint16
	for _, l := range lines {
		w := uint16(cell.StringWidth(l))
		if w > maxW {
			maxW = w
		}
	}
	return LayoutProps{
		MinWidth:  maxW,
		MinHeight: uint16(len(lines)),
		Flex:      0,
	}
}

func (t *textRefComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := t.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

func (t *textRefComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return false
}

// textFnComponent evaluates a string getter dynamically on render.
type textFnComponent struct {
	getter func() string
	style  cell.Style
}

// TextFn creates a dynamic text component evaluated via a getter function on each frame.
func TextFn(getter func() string, style ...cell.Style) Component {
	st := cell.NewStyle()
	if len(style) > 0 {
		st = style[0]
	}
	return &textFnComponent{getter: getter, style: st}
}

func (t *textFnComponent) content() string {
	if t.getter == nil {
		return ""
	}
	return t.getter()
}

func (t *textFnComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	str := t.content()
	if str == "" {
		return
	}
	area := ctx.Area
	finalStyle := ctx.Style.Merge(t.style)

	lines := strings.Split(str, "\n")
	for row, line := range lines {
		if uint16(row) >= area.Height {
			break
		}
		col := uint16(0)
		for _, r := range line {
			rw := cell.RuneWidth(r)
			if col+uint16(rw) > area.Width {
				break
			}
			buf.SetCell(area.X+col, area.Y+uint16(row), cell.Cell{
				Content: r,
				Style:   finalStyle,
			})
			col += uint16(rw)
		}
	}
}

func (t *textFnComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	str := t.content()
	if str == "" {
		return LayoutProps{MinWidth: 0, MinHeight: 0, Flex: 0}
	}
	lines := strings.Split(str, "\n")
	var maxW uint16
	for _, l := range lines {
		w := uint16(cell.StringWidth(l))
		if w > maxW {
			maxW = w
		}
	}
	return LayoutProps{
		MinWidth:  maxW,
		MinHeight: uint16(len(lines)),
		Flex:      0,
	}
}

func (t *textFnComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := t.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

func (t *textFnComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return false
}
