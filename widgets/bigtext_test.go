package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func drawBig(b BigText, w, h uint16) []string {
	area := cell.NewRect(0, 0, w, h)
	buf := buffer.NewBuffer(area)
	b.Draw(cell.NewContext(area, cell.Style{}), buf)
	lines := strings.Split(buf.Snapshot(), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return lines
}

func TestBigTextSizes(t *testing.T) {
	// font8x8's "A".
	full := drawBig(BigText{Text: "A"}, 8, 8)
	for i, want := range []string{"  ██", " ████", "██  ██", "██  ██", "██████", "██  ██", "██  ██", ""} {
		if full[i] != want {
			t.Errorf("full row %d: %q, want %q", i, full[i], want)
		}
	}
	half := drawBig(BigText{Text: "A", Size: BigTextHalfHeight}, 8, 4)
	for i, want := range []string{" ▄██▄", "██  ██", "██▀▀██", "▀▀  ▀▀"} {
		if half[i] != want {
			t.Errorf("half-height row %d: %q, want %q", i, half[i], want)
		}
	}
	quad := drawBig(BigText{Text: "A", Size: BigTextQuadrant}, 4, 4)
	for i, want := range []string{"▗█▖", "█ █", "█▀█", "▀ ▀"} {
		if quad[i] != want {
			t.Errorf("quadrant row %d: %q, want %q", i, quad[i], want)
		}
	}
}

func TestBigTextLayout(t *testing.T) {
	// Two lines, centred, at half height: 4 rows each.
	rows := drawBig(BigText{Text: "I\nII", Size: BigTextHalfHeight, Alignment: AlignCenter}, 20, 8)
	// "I" is 8 wide, so it starts at (20-8)/2 = 6; "II" at (20-16)/2 = 2.
	// font8x8's I (0x1E: bits 1-4) has one blank column on its left.
	indent := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	if got := indent(rows[0]); got != 6+1 {
		t.Errorf("first line indented %d:\n%s", got, strings.Join(rows, "\n"))
	}
	if got := indent(rows[4]); got != 2+1 {
		t.Errorf("second line indented %d:\n%s", got, strings.Join(rows, "\n"))
	}
	// A line that does not fit in the height is left out, not cut.
	if rows := drawBig(BigText{Text: "A\nA", Size: BigTextHalfHeight}, 8, 6); strings.TrimSpace(rows[4]+rows[5]) != "" {
		t.Errorf("a half line was drawn:\n%s", strings.Join(rows, "\n"))
	}
	// Turkish letters outside Latin-1 fall back to their base letters.
	if a, b := drawBig(BigText{Text: "ş"}, 8, 8), drawBig(BigText{Text: "s"}, 8, 8); strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Error("ş does not fall back to s")
	}
	if w, h := (BigText{Text: "ab\nc", Size: BigTextQuadrant}).SizeHint(cell.NewRect(0, 0, 100, 100)); w != 8 || h != 8 {
		t.Errorf("SizeHint %dx%d, want 8x8", w, h)
	}
}

func TestBigTextDoesNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	b := BigText{Text: "12:04\nçüö", Size: BigTextHalfHeight, Alignment: AlignCenter}
	if n := testing.AllocsPerRun(50, func() { b.Draw(ctx, buf) }); n != 0 {
		t.Errorf("%.0f allocs per Draw", n)
	}
}

func BenchmarkBigTextDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := BigText{Text: "12:04:59", Size: BigTextHalfHeight}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}
