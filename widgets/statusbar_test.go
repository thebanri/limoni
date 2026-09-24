package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func statusRow(s StatusBar, width uint16) (*buffer.Buffer, string) {
	area := cell.NewRect(0, 0, width, 1)
	buf := buffer.NewBuffer(area)
	s.Draw(cell.NewContext(area, cell.Style{}), buf)
	snap := buf.Snapshot()
	return buf, strings.SplitN(snap, "\n", 2)[0]
}

func TestStatusBarLayout(t *testing.T) {
	bar := StatusBar{
		Left:   []StatusItem{{Text: "NORMAL"}},
		Center: []StatusItem{{Text: "main.go"}},
		Right:  []StatusItem{{Key: "^Q", Text: "quit"}, {Text: "12:04"}},
	}
	for _, tc := range []struct {
		width uint16
		want  string
	}{
		// The centre is centred on the bar: (40-7)/2 = 16.
		{40, "NORMAL          main.go   ^Q quit  12:04"},
		// No room for the centre between the two sides: it is left out.
		{24, "NORMAL    ^Q quit  12:04"},
		// Not even the whole right group: it loses its left end.
		{16, "NORMAL it  12:04"},
	} {
		_, got := statusRow(bar, tc.width)
		if got != tc.want {
			t.Errorf("width %d:\n got %q\nwant %q", tc.width, got, tc.want)
		}
	}
}

func TestStatusBarKeyStyle(t *testing.T) {
	key := cell.Style{Fg: cell.NewColorRGB(255, 200, 0)}
	buf, _ := statusRow(StatusBar{Left: []StatusItem{{Key: "F1", Text: "help"}}, KeyStyle: key}, 20)
	if got := buf.CellAt(0, 0).Style.Fg; got != key.Fg {
		t.Errorf("key colour %v, want %v", got, key.Fg)
	}
	if got := buf.CellAt(3, 0).Style.Fg; got == key.Fg {
		t.Error("the text after the key took the key style")
	}
}

// Cutting the right group from the left must not split a grapheme cluster:
// a family emoji is one wide character, not four.
func TestStatusBarCutsRightGroupByCluster(t *testing.T) {
	family := "\U0001F468\u200D\U0001F469\u200D\U0001F467"
	bar := StatusBar{Left: []StatusItem{{Text: "ab"}}, Right: []StatusItem{{Text: family + "xyz"}}}
	// "ab" + gap = 3 columns; the right group is 5 wide; 7 columns leave 4,
	// cutting one column into the 2-wide emoji.
	_, got := statusRow(bar, 7)
	if got != "ab  xyz" {
		t.Errorf("got %q, want the emoji's visible half blanked", got)
	}
	_, got = statusRow(bar, 8)
	// Snapshot shows the emoji's second column as a space.
	if got != "ab "+family+" xyz" {
		t.Errorf("got %q, want the whole group", got)
	}
}

func TestStatusBarDoesNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	bar := StatusBar{
		Left:   []StatusItem{{Text: "NORMAL"}},
		Center: []StatusItem{{Text: "main.go"}},
		Right:  []StatusItem{{Key: "^Q", Text: "quit"}, {Text: "12:04"}},
	}
	for _, w := range []uint16{80, 24, 16} {
		ctx.Area.Width = w
		if a := testing.AllocsPerRun(50, func() { bar.Draw(ctx, buf) }); a != 0 {
			t.Errorf("width %d: %.0f allocs per Draw", w, a)
		}
	}
}

func BenchmarkStatusBarDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	bar := StatusBar{
		Left:   []StatusItem{{Text: "NORMAL"}},
		Center: []StatusItem{{Text: "main.go"}},
		Right:  []StatusItem{{Key: "^Q", Text: "quit"}, {Text: "12:04"}},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bar.Draw(ctx, buf)
	}
}
