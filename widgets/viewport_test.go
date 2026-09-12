package widgets

import (
	"fmt"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// numberedRows draws "row N" on each line of its area, so a test can read the
// scroll offset straight out of the rendered buffer.
type numberedRows struct {
	width, height uint16
}

func (n numberedRows) Draw(ctx cell.Context, buf *buffer.Buffer) {
	for y := uint16(0); y < ctx.Area.Height && y < n.height; y++ {
		buf.SetStringWithin(ctx.Area.X, ctx.Area.Y+y, fmt.Sprintf("row %d", y), ctx.Style, ctx.Area.Width)
	}
}

func (n numberedRows) SizeHint(cell.Rect) (uint16, uint16) { return n.width, n.height }

func TestViewportShowsWindowAtOffset(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 3)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()
	state.OffsetY = 4

	Viewport{Child: numberedRows{width: 10, height: 20}, State: state}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	for i := uint16(0); i < 3; i++ {
		want := fmt.Sprintf("row %d", 4+i)
		if got := row(buf, 0, i, uint16(len(want))); got != want {
			t.Fatalf("viewport line %d = %q, want %q", i, got, want)
		}
	}
}

// Content that fits must bypass the offscreen buffer entirely and still render.
func TestViewportShortContentRendersDirectly(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()

	Viewport{Child: numberedRows{width: 10, height: 2}, State: state}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got := row(buf, 0, 0, 5); got != "row 0" {
		t.Fatalf("first line = %q, want %q", got, "row 0")
	}
	if !state.AtBottom() {
		t.Error("content shorter than the viewport should report AtBottom")
	}
}

func TestViewportClampsOffsetToContent(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 4)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()
	state.OffsetY = 9999

	Viewport{Child: numberedRows{width: 10, height: 10}, State: state}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if want := 10 - 4; state.OffsetY != want {
		t.Fatalf("clamped offset = %d, want %d", state.OffsetY, want)
	}
	if !state.AtBottom() {
		t.Error("clamped-to-end viewport should report AtBottom")
	}
}

func TestViewportStateScrollHelpers(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()
	vp := Viewport{Child: numberedRows{width: 10, height: 25}, State: state}
	ctx := cell.NewContext(area, cell.Style{})

	// The first Draw measures the content so the helpers can clamp.
	vp.Draw(ctx, buf)

	if !state.AtTop() {
		t.Error("fresh viewport should report AtTop")
	}

	state.ScrollBy(10)
	if state.OffsetY != 10 {
		t.Fatalf("ScrollBy(10) = %d, want 10", state.OffsetY)
	}

	state.ScrollBy(-100)
	if state.OffsetY != 0 {
		t.Fatalf("ScrollBy(-100) = %d, want 0 (clamped)", state.OffsetY)
	}

	state.GotoBottom()
	if want := 25 - 5; state.OffsetY != want {
		t.Fatalf("GotoBottom = %d, want %d", state.OffsetY, want)
	}
	if pct := state.ScrollPercent(); pct != 1 {
		t.Fatalf("ScrollPercent at bottom = %v, want 1", pct)
	}

	state.GotoTop()
	if state.OffsetY != 0 || state.ScrollPercent() != 0 {
		t.Fatalf("GotoTop = %d (%.2f), want 0 (0)", state.OffsetY, state.ScrollPercent())
	}

	if w, h := state.ContentSize(); w != 10 || h != 25 {
		t.Fatalf("ContentSize = (%d,%d), want (10,25)", w, h)
	}
}

func TestViewportMouseWheelScrolls(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()

	var handler func(driver.MouseEvent)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(_ cell.Rect, h func(driver.MouseEvent)) { handler = h }

	Viewport{Child: numberedRows{width: 10, height: 40}, State: state, MouseWheelStep: 2}.
		Draw(ctx, buf)

	if handler == nil {
		t.Fatal("viewport did not register a mouse handler")
	}
	handler(driver.MouseEvent{Button: driver.MouseScrollDown})
	if state.OffsetY != 2 {
		t.Fatalf("after one wheel notch offset = %d, want 2", state.OffsetY)
	}
	handler(driver.MouseEvent{Button: driver.MouseScrollUp})
	if state.OffsetY != 0 {
		t.Fatalf("after scrolling back offset = %d, want 0", state.OffsetY)
	}
}

func TestViewportScrollbarReservesColumn(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 4)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()

	Viewport{Child: numberedRows{width: 9, height: 40}, State: state, Scrollbar: true}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	// The last column belongs to the bar; the thumb sits at the top.
	if c := buf.Get(9, 0); c == nil || c.Content != ScrollbarThumbSymbol {
		t.Fatalf("expected scrollbar thumb in the last column, got %q", contentOf(c))
	}
	if got := row(buf, 0, 0, 5); got != "row 0" {
		t.Fatalf("content column = %q, want %q", got, "row 0")
	}
}

// A nil State must render rather than panic, so a Viewport is safe to build
// before its state is wired up.
func TestViewportNilStateRenders(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 3)
	buf := buffer.NewBuffer(area)

	Viewport{Child: numberedRows{width: 10, height: 20}}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got := row(buf, 0, 0, 5); got != "row 0" {
		t.Fatalf("nil-state viewport = %q, want %q", got, "row 0")
	}
}

func TestViewportNilChildIsNoop(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 3)
	buf := buffer.NewBuffer(area)
	Viewport{State: NewViewportState()}.Draw(cell.NewContext(area, cell.Style{}), buf)
	if got := row(buf, 0, 0, 10); got != "          " {
		t.Fatalf("nil-child viewport drew %q, want blank", got)
	}
}

// The offscreen buffer must be reused between frames, otherwise scrolling
// allocates a full content-sized buffer every frame.
func TestViewportReusesScratchBuffer(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()
	vp := Viewport{Child: numberedRows{width: 10, height: 50}, State: state}
	ctx := cell.NewContext(area, cell.Style{})

	vp.Draw(ctx, buf)
	first := state.scratch
	if first == nil {
		t.Fatal("expected an offscreen buffer after the first draw")
	}

	state.ScrollBy(3)
	vp.Draw(ctx, buf)
	if state.scratch != first {
		t.Error("offscreen buffer was reallocated on a plain scroll")
	}
}

func contentOf(c *cell.Cell) string {
	if c == nil {
		return "<nil>"
	}
	return string(c.Content)
}

// staticRows draws without allocating, so the benchmark measures the viewport
// rather than the child's string formatting.
type staticRows struct {
	width, height uint16
	line          string
}

func (s staticRows) Draw(ctx cell.Context, buf *buffer.Buffer) {
	for y := uint16(0); y < ctx.Area.Height && y < s.height; y++ {
		buf.SetStringWithin(ctx.Area.X, ctx.Area.Y+y, s.line, ctx.Style, ctx.Area.Width)
	}
}

func (s staticRows) SizeHint(cell.Rect) (uint16, uint16) { return s.width, s.height }

func BenchmarkViewportScroll(b *testing.B) {
	area := cell.NewRect(0, 0, 80, 24)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()
	vp := Viewport{Child: staticRows{width: 80, height: 1000, line: "lorem ipsum dolor sit amet"}, State: state}
	ctx := cell.NewContext(area, cell.Style{})
	vp.Draw(ctx, buf) // warm the offscreen buffer

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.ScrollTo(i % (1000 - 24))
		vp.Draw(ctx, buf)
	}
}

// BenchmarkViewportScrollShortContent exercises the fast path where content
// fits and the offscreen buffer is bypassed entirely.
func BenchmarkViewportScrollShortContent(b *testing.B) {
	area := cell.NewRect(0, 0, 80, 24)
	buf := buffer.NewBuffer(area)
	state := NewViewportState()
	vp := Viewport{Child: staticRows{width: 80, height: 20, line: "lorem ipsum"}, State: state}
	ctx := cell.NewContext(area, cell.Style{})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vp.Draw(ctx, buf)
	}
}
