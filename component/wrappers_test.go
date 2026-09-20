package component

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

// probe is a leaf that records the area it was drawn into and whether an
// event reached it. Wrapping it is how a wrapper's geometry is checked:
// what a wrapper does to its child is the whole of what it does.
type probe struct {
	w, h     uint16
	drawnIn  cell.Rect
	gotEvent bool
}

func (p *probe) Draw(ctx cell.Context, buf *buffer.Buffer) {
	p.drawnIn = ctx.Area
	if buf != nil && ctx.Area.Width > 0 && ctx.Area.Height > 0 {
		buf.SetCellDirect(ctx.Area.X, ctx.Area.Y, cell.Cell{Content: 'x'})
	}
}

func (p *probe) LayoutInfo(maxArea cell.Rect) LayoutProps {
	return LayoutProps{MinWidth: p.w, MinHeight: p.h}
}

func (p *probe) SizeHint(cell.Rect) (uint16, uint16) { return p.w, p.h }

func (p *probe) HandleEvent(cell.Context, *driver.Event) bool {
	p.gotEvent = true
	return true
}

func drawInto(c Component, area cell.Rect) *buffer.Buffer {
	buf := buffer.NewBuffer(area)
	c.Draw(cell.NewContext(area, cell.Style{}), buf)
	return buf
}

// Every wrapper reports a size that includes what it adds around its child,
// or a layout built from SizeHint allocates too little room and the child is
// clipped.
func TestWrapperSizeHintsIncludeWhatTheyAdd(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 20)
	cases := []struct {
		name         string
		make         func(Component) Component
		wantW, wantH uint16
	}{
		{"Pad", func(c Component) Component { return Pad(c, 1, 2, 3, 4) }, 10 + 6, 4 + 4},
		{"PadAll", func(c Component) Component { return PadAll(c, 2) }, 10 + 4, 4 + 4},
		{"PadAxis", func(c Component) Component { return PadAxis(c, 3, 1) }, 10 + 6, 4 + 2},
		{"Margin", func(c Component) Component { return Margin(c, 1, 1, 1, 1) }, 12, 6},
		{"MarginAll", func(c Component) Component { return MarginAll(c, 2) }, 14, 8},
		{"MarginAxis", func(c Component) Component { return MarginAxis(c, 2, 1) }, 14, 6},
		{"Border", func(c Component) Component { return Border(c, widgets.SymbolsSingle, cell.Style{}) }, 12, 6},
	}
	for _, c := range cases {
		child := &probe{w: 10, h: 4}
		w, h := c.make(child).SizeHint(area)
		if w != c.wantW || h != c.wantH {
			t.Errorf("%s.SizeHint = %dx%d, want %dx%d", c.name, w, h, c.wantW, c.wantH)
		}
	}
}

// And each one draws its child inside the space it kept for itself.
func TestWrappersDrawTheirChildInset(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 10)
	cases := []struct {
		name string
		make func(Component) Component
		want cell.Rect
	}{
		{"Pad", func(c Component) Component { return Pad(c, 1, 2, 3, 4) }, cell.NewRect(4, 1, 14, 6)},
		{"MarginAll", func(c Component) Component { return MarginAll(c, 2) }, cell.NewRect(2, 2, 16, 6)},
		{"Border", func(c Component) Component { return Border(c, widgets.SymbolsSingle, cell.Style{}) }, cell.NewRect(1, 1, 18, 8)},
	}
	for _, c := range cases {
		child := &probe{w: 1, h: 1}
		drawInto(c.make(child), area)
		if child.drawnIn != c.want {
			t.Errorf("%s drew its child in %+v, want %+v", c.name, child.drawnIn, c.want)
		}
	}
}

// An event must find its way through the wrappers to the leaf, or keys never
// reach the widget the user is looking at.
func TestEventsReachTheChildThroughEveryWrapper(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 10)
	ev := &driver.Event{Type: driver.EventKey, Key: driver.KeyEvent{Type: driver.KeyRune, Ch: 'k'}}

	wrappers := map[string]func(Component) Component{
		"Pad":       func(c Component) Component { return PadAll(c, 1) },
		"Margin":    func(c Component) Component { return MarginAll(c, 1) },
		"Border":    func(c Component) Component { return Border(c, widgets.SymbolsSingle, cell.Style{}) },
		"Constrain": func(c Component) Component { return Constrain(0, 18, 0, 8, c) },
		"VStack":    func(c Component) Component { return VStack(c) },
		"HStack":    func(c Component) Component { return HStack(c) },
	}
	for name, wrap := range wrappers {
		child := &probe{w: 4, h: 2}
		root := wrap(child)
		ctx := cell.NewContext(area, cell.Style{})
		DispatchEvent(root, ctx, ev)
		if !child.gotEvent {
			t.Errorf("%s did not pass the event to its child", name)
		}
	}

	// Dispatch is defensive: neither a nil tree nor a nil event is a panic.
	if DispatchEvent(nil, cell.NewContext(area, cell.Style{}), ev) {
		t.Error("a nil component handled an event")
	}
	if DispatchEvent(&probe{}, cell.NewContext(area, cell.Style{}), nil) {
		t.Error("a nil event was handled")
	}
}

// A wrapper with no room left for its child must draw nothing rather than
// wrap into a negative size.
func TestWrappersTolerateAnAreaSmallerThanTheirInset(t *testing.T) {
	tiny := cell.NewRect(0, 0, 1, 1)
	for name, c := range map[string]Component{
		"Pad":    PadAll(&probe{w: 4, h: 2}, 3),
		"Margin": MarginAll(&probe{w: 4, h: 2}, 3),
		"Border": Border(&probe{w: 4, h: 2}, widgets.SymbolsSingle, cell.Style{}),
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked in a 1x1 area: %v", name, r)
				}
			}()
			drawInto(c, tiny)
		}()
	}
}

func TestEmptyAndWhenChooseABranch(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 3)

	e := Empty()
	w, h := e.SizeHint(area)
	if w != 0 || h != 0 {
		t.Errorf("Empty asks for %dx%d, want nothing", w, h)
	}
	buf := drawInto(e, area)
	if c := buf.Get(0, 0); c != nil && c.Content != 0 && c.Content != ' ' {
		t.Errorf("Empty drew %q", c.Content)
	}
	if DispatchEvent(e, cell.NewContext(area, cell.Style{}), &driver.Event{Type: driver.EventKey}) {
		t.Error("Empty handled an event")
	}

	yes, no := &probe{w: 1, h: 1}, &probe{w: 1, h: 1}
	drawInto(When(true, yes, no), area)
	if yes.drawnIn.Width == 0 || no.drawnIn.Width != 0 {
		t.Error("When(true) drew the wrong branch")
	}

	yes, no = &probe{w: 1, h: 1}, &probe{w: 1, h: 1}
	drawInto(When(false, yes, no), area)
	if no.drawnIn.Width == 0 || yes.drawnIn.Width != 0 {
		t.Error("When(false) drew the wrong branch")
	}

	// With no else branch, a false condition draws nothing at all.
	yes = &probe{w: 1, h: 1}
	drawInto(When(false, yes), area)
	if yes.drawnIn.Width != 0 {
		t.Error("When(false) with no alternative still drew the then-branch")
	}
}

func TestDividerCustomUsesTheGivenRune(t *testing.T) {
	area := cell.NewRect(0, 0, 6, 1)
	buf := drawInto(DividerCustom('='), area)
	for x := uint16(0); x < 6; x++ {
		if c := buf.Get(x, 0); c == nil || c.Content != '=' {
			t.Fatalf("column %d = %q, want '='", x, c.Content)
		}
	}
}
