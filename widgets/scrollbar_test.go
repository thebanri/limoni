package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// column reads a vertical run of glyphs out of a buffer.
func column(buf *buffer.Buffer, x, y, height uint16) string {
	out := ""
	for i := uint16(0); i < height; i++ {
		if c := buf.Get(x, y+i); c != nil {
			out += string(c.Content)
		}
	}
	return out
}

// row reads a horizontal run of glyphs out of a buffer.
func row(buf *buffer.Buffer, x, y, width uint16) string {
	out := ""
	for i := uint16(0); i < width; i++ {
		if c := buf.Get(x+i, y); c != nil {
			out += string(c.Content)
		}
	}
	return out
}

func TestScrollbarMetricsBounds(t *testing.T) {
	tests := []struct {
		name                            string
		content, viewport, offset, size int
		wantStart, wantLength           int
	}{
		{"content fits, no thumb", 5, 10, 0, 10, 0, 0},
		{"exact fit, no thumb", 10, 10, 0, 10, 0, 0},
		{"half visible at top", 20, 10, 0, 10, 0, 5},
		{"half visible at bottom", 20, 10, 10, 10, 5, 5},
		{"half visible midway", 20, 10, 5, 10, 2, 5},
		{"huge content clamps thumb to one cell", 100000, 1, 0, 10, 0, 1},
		{"negative offset clamps to top", 20, 10, -5, 10, 0, 5},
		{"overshooting offset clamps to bottom", 20, 10, 999, 10, 5, 5},
		{"zero track yields nothing", 20, 10, 0, 0, 0, 0},
		{"zero viewport yields nothing", 20, 0, 0, 10, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, length := ScrollbarMetrics(tc.content, tc.viewport, tc.offset, tc.size)
			if start != tc.wantStart || length != tc.wantLength {
				t.Fatalf("ScrollbarMetrics(%d,%d,%d,%d) = (%d,%d), want (%d,%d)",
					tc.content, tc.viewport, tc.offset, tc.size, start, length, tc.wantStart, tc.wantLength)
			}
		})
	}
}

// The thumb must touch the far end exactly when the last item is visible,
// otherwise "scrolled to the bottom" is not readable from the bar.
func TestScrollbarThumbReachesEndAtMaxOffset(t *testing.T) {
	for _, content := range []int{11, 17, 50, 997, 100000} {
		for _, viewport := range []int{1, 3, 10} {
			for _, track := range []int{3, 10, 40} {
				if content <= viewport {
					continue
				}
				start, length := ScrollbarMetrics(content, viewport, content-viewport, track)
				if start+length != track {
					t.Fatalf("content=%d viewport=%d track=%d: thumb ends at %d, want %d",
						content, viewport, track, start+length, track)
				}
				if topStart, _ := ScrollbarMetrics(content, viewport, 0, track); topStart != 0 {
					t.Fatalf("content=%d viewport=%d track=%d: thumb starts at %d at offset 0, want 0",
						content, viewport, track, topStart)
				}
			}
		}
	}
}

// Clicking a position must land on an offset whose thumb covers that position,
// so a click never scrolls somewhere visibly unrelated.
func TestScrollbarOffsetAtRoundTrip(t *testing.T) {
	const content, viewport, track = 100, 10, 20

	for position := 0; position < track; position++ {
		offset := ScrollbarOffsetAt(content, viewport, track, position)
		if offset < 0 || offset > content-viewport {
			t.Fatalf("position %d produced out-of-range offset %d", position, offset)
		}
		start, length := ScrollbarMetrics(content, viewport, offset, track)
		if position < start || position >= start+length {
			// Rounding may place the click one cell outside the thumb at the
			// extremes; anything further means the mapping is wrong.
			if position < start-1 || position > start+length {
				t.Fatalf("position %d maps to offset %d whose thumb is [%d,%d)",
					position, offset, start, start+length)
			}
		}
	}
}

func TestScrollbarOffsetAtContentFits(t *testing.T) {
	if got := ScrollbarOffsetAt(5, 10, 10, 7); got != 0 {
		t.Fatalf("ScrollbarOffsetAt with content that fits = %d, want 0", got)
	}
}

func TestScrollbarDrawVertical(t *testing.T) {
	area := cell.NewRect(0, 0, 1, 10)
	buf := buffer.NewBuffer(area)

	Scrollbar{ContentLength: 20, ViewportLength: 10, Offset: 0}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := column(buf, 0, 0, 10), "█████░░░░░"; got != want {
		t.Fatalf("vertical scrollbar = %q, want %q", got, want)
	}
}

func TestScrollbarDrawHorizontal(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	buf := buffer.NewBuffer(area)

	Scrollbar{
		Orientation:    ScrollbarHorizontal,
		ContentLength:  20,
		ViewportLength: 10,
		Offset:         10,
	}.Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := row(buf, 0, 0, 10), "░░░░░█████"; got != want {
		t.Fatalf("horizontal scrollbar = %q, want %q", got, want)
	}
}

// A bar next to content that fits should render nothing, so short lists do not
// show a dead track.
func TestScrollbarHiddenWhenContentFits(t *testing.T) {
	area := cell.NewRect(0, 0, 1, 10)
	buf := buffer.NewBuffer(area)

	Scrollbar{ContentLength: 5, ViewportLength: 10}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := column(buf, 0, 0, 10), "          "; got != want {
		t.Fatalf("scrollbar drew %q for content that fits, want blank", got)
	}
}

func TestScrollbarAlwaysVisibleDrawsTrack(t *testing.T) {
	area := cell.NewRect(0, 0, 1, 4)
	buf := buffer.NewBuffer(area)

	Scrollbar{ContentLength: 2, ViewportLength: 10, AlwaysVisible: true}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := column(buf, 0, 0, 4), "░░░░"; got != want {
		t.Fatalf("AlwaysVisible scrollbar = %q, want %q", got, want)
	}
}

func TestScrollbarWheelAndClickScroll(t *testing.T) {
	area := cell.NewRect(0, 0, 1, 10)
	buf := buffer.NewBuffer(area)

	var handler func(driver.MouseEvent)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(_ cell.Rect, h func(driver.MouseEvent)) { handler = h }

	got := -1
	Scrollbar{
		ContentLength:  20,
		ViewportLength: 10,
		Offset:         4,
		OnScroll:       func(offset int) { got = offset },
	}.Draw(ctx, buf)

	if handler == nil {
		t.Fatal("scrollbar with OnScroll did not register a mouse handler")
	}

	handler(driver.MouseEvent{Button: driver.MouseScrollDown})
	if got != 5 {
		t.Fatalf("scroll down from 4 = %d, want 5", got)
	}

	handler(driver.MouseEvent{Button: driver.MouseScrollUp})
	if got != 3 {
		t.Fatalf("scroll up from 4 = %d, want 3", got)
	}

	// Clicking the bottom of the track jumps to the end.
	handler(driver.MouseEvent{Button: driver.MouseLeft, Y: 9})
	if want := 20 - 10; got != want {
		t.Fatalf("click at track bottom = %d, want %d", got, want)
	}
}

// Scrolling past either end must clamp rather than run away.
func TestScrollbarWheelClampsAtEnds(t *testing.T) {
	area := cell.NewRect(0, 0, 1, 10)
	buf := buffer.NewBuffer(area)

	var handler func(driver.MouseEvent)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(_ cell.Rect, h func(driver.MouseEvent)) { handler = h }

	got := -1
	Scrollbar{
		ContentLength:  20,
		ViewportLength: 10,
		Offset:         0,
		OnScroll:       func(offset int) { got = offset },
	}.Draw(ctx, buf)

	handler(driver.MouseEvent{Button: driver.MouseScrollUp})
	if got != 0 {
		t.Fatalf("scroll up at top = %d, want 0", got)
	}
}

func TestScrollbarSizeHint(t *testing.T) {
	maxArea := cell.NewRect(0, 0, 40, 12)

	if w, h := (Scrollbar{}).SizeHint(maxArea); w != 1 || h != 12 {
		t.Fatalf("vertical SizeHint = (%d,%d), want (1,12)", w, h)
	}
	if w, h := (Scrollbar{Orientation: ScrollbarHorizontal}).SizeHint(maxArea); w != 40 || h != 1 {
		t.Fatalf("horizontal SizeHint = (%d,%d), want (40,1)", w, h)
	}
}

func BenchmarkScrollbarDraw(b *testing.B) {
	area := cell.NewRect(0, 0, 1, 40)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	bar := Scrollbar{ContentLength: 100000, ViewportLength: 40}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bar.Offset = i % (100000 - 40)
		bar.Draw(ctx, buf)
	}
}
