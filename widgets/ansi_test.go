package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func ansiLog(lines ...string) *testLog {
	l := &testLog{}
	for _, s := range lines {
		l.add(s, LevelInfo)
	}
	return l
}

func fgAt(buf *buffer.Buffer, x, y uint16) cell.Color { return buf.Get(x, y).Style.Fg }

// Colours a program printed are drawn, and the escapes themselves are not.
func TestLogViewDrawsANSIColours(t *testing.T) {
	lv := &LogView{Source: ansiLog("\x1b[31mred\x1b[0m plain \x1b[1;38;5;208mhot\x1b[22m \x1b[38;2;10;20;30mrgb\x1b[39m!")}
	buf := drawLog(lv, 40, 1)
	if got := rowAt(buf, 0); got != "red plain hot rgb!" {
		t.Fatalf("row %q", got)
	}
	if fgAt(buf, 0, 0) != cell.NewColorANSI(1) || fgAt(buf, 2, 0) != cell.NewColorANSI(1) {
		t.Fatalf("red: %v", fgAt(buf, 0, 0))
	}
	if fgAt(buf, 4, 0) != (cell.Style{}).Fg {
		t.Fatalf("reset kept colour %v", fgAt(buf, 4, 0))
	}
	hot := buf.Get(10, 0).Style
	if hot.Fg != cell.NewColorANSI(208) || hot.Modifier&cell.ModifierBold == 0 {
		t.Fatalf("256-colour bold: %+v", hot)
	}
	if buf.Get(13, 0).Style.Modifier&cell.ModifierBold != 0 {
		t.Fatal("22 left bold on")
	}
	if fgAt(buf, 14, 0) != cell.NewColorRGB(10, 20, 30) || fgAt(buf, 17, 0) != (cell.Style{}).Fg {
		t.Fatalf("truecolour %v, after 39 %v", fgAt(buf, 14, 0), fgAt(buf, 17, 0))
	}
}

// A reset returns to the line's own style (its level), not to the terminal's.
func TestLogViewANSIResetKeepsTheLevelStyle(t *testing.T) {
	src := &testLog{}
	src.add("\x1b[32mok\x1b[0m failed", LevelError)
	buf := drawLog(&LogView{Source: src}, 20, 1)
	if fgAt(buf, 0, 0) != cell.NewColorANSI(2) {
		t.Fatalf("green %v", fgAt(buf, 0, 0))
	}
	if fgAt(buf, 3, 0) != defaultLevelStyles[LevelError].Fg {
		t.Fatalf("after reset %v, want the error colour", fgAt(buf, 3, 0))
	}
}

func TestSGRForms(t *testing.T) {
	base := cell.Style{Modifier: cell.ModifierItalic}
	cases := []struct {
		params string
		want   cell.Style
	}{
		{"38:2::1:2:3", cell.Style{Fg: cell.NewColorRGB(1, 2, 3), Modifier: cell.ModifierItalic}},
		{"38:2:1:2:3", cell.Style{Fg: cell.NewColorRGB(1, 2, 3), Modifier: cell.ModifierItalic}},
		{"48:5:17", cell.Style{Bg: cell.NewColorANSI(17), Modifier: cell.ModifierItalic}},
		{"4:3", cell.Style{Modifier: cell.ModifierItalic | cell.ModifierUndercurl}},
		{"4;24", cell.Style{Modifier: cell.ModifierItalic}},
		{"23", cell.Style{Modifier: cell.ModifierItalic}}, // "off" returns to base
		{"3;9;97;104", cell.Style{Fg: cell.NewColorANSI(15), Bg: cell.NewColorANSI(12), Modifier: cell.ModifierItalic | cell.ModifierStrikethrough}},
		{"38;5", base},     // truncated: ignored
		{"38;2;1;2", base}, // too short for ;2;r;g;b
		{"38;5;999", cell.Style{Fg: cell.NewColorANSI(255), Modifier: cell.ModifierItalic}},
		{"", base},
	}
	for _, c := range cases {
		if got := applySGR(c.params, base, base); got != c.want {
			t.Errorf("SGR %q = %+v, want %+v", c.params, got, c.want)
		}
	}
}

// Sequences that are not SGR — cursor moves, erases, OSC titles and links,
// charset switches, an unterminated tail — draw nothing and break nothing.
func TestLogViewSkipsOtherEscapes(t *testing.T) {
	lines := []string{
		"a\x1b[2Kb\x1b[?25lc\x1b[10;5Hd",
		"\x1b]0;title\x07x\x1b]8;;https://e.x\x1b\\link\x1b]8;;\x1b\\y",
		"\x1b(Bp\x1b7q\x1b[",
		"\x1b[>4;1mz", // a private-marker sequence that ends in m is not SGR
	}
	want := []string{"abcd", "xlinky", "pq", "z"}
	buf := drawLog(&LogView{Source: ansiLog(lines...)}, 20, 4)
	for i, w := range want {
		if got := rowAt(buf, uint16(i)); got != w {
			t.Errorf("row %d = %q, want %q", i, got, w)
		}
		if got := stripANSI(lines[i]); got != w {
			t.Errorf("stripANSI(%q) = %q, want %q", lines[i], got, w)
		}
	}
	if fgAt(buf, 0, 3) != (cell.Style{}).Fg || buf.Get(0, 3).Style.Modifier != 0 {
		t.Fatalf("private sequence changed the style: %+v", buf.Get(0, 3).Style)
	}
}

// Sideways scroll, search highlights and the semantic tree count columns of
// the text a reader sees, not of the escapes.
func TestLogViewANSIScrollHighlightAndTree(t *testing.T) {
	st := NewLogViewState()
	st.HOffset = 2
	lv := &LogView{Source: ansiLog("\x1b[31mabc\x1b[0mdef \x1b[1merror\x1b[0m end"), State: st, Highlight: "error"}
	buf := drawLog(lv, 20, 1)
	if got := rowAt(buf, 0); got != "cdef error end" {
		t.Fatalf("scrolled row %q", got)
	}
	if fgAt(buf, 0, 0) != cell.NewColorANSI(1) {
		t.Fatalf("scrolled-in colour %v", fgAt(buf, 0, 0))
	}
	hl := buf.Get(5, 0).Style
	if hl.Bg != cell.NewColorRGB(250, 210, 60) || buf.Get(4, 0).Style.Bg == hl.Bg || buf.Get(10, 0).Style.Bg == hl.Bg {
		t.Fatalf("highlight not on \"error\": %+v", hl)
	}
	node := lv.AccessibilityNode(cell.NewRect(0, 0, 20, 1), false)
	if node.Children[0].Label != "abcdef error end" {
		t.Fatalf("tree label %q", node.Children[0].Label)
	}
	st.Select(0)
	drawLog(lv, 20, 1)
	if node = lv.AccessibilityNode(cell.NewRect(0, 0, 20, 1), false); node.Value != "abcdef error end" {
		t.Fatalf("tree value %q", node.Value)
	}
}

// Following a log of coloured lines, a frame allocates only for lines that
// just came on screen; redrawing the same screen allocates nothing.
func TestLogViewANSIDrawDoesNotAllocate(t *testing.T) {
	src := &testLog{}
	for i := 0; i < 1000; i++ {
		src.add("\x1b[32mINFO\x1b[0m request \x1b[38;2;200;100;0mserved\x1b[0m in 3ms", LevelInfo)
	}
	lv := &LogView{ID: "log", Source: src, State: NewLogViewState(), LineNumbers: true, Highlight: "served"}
	area := cell.NewRect(0, 0, 60, 10)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	lv.Draw(ctx, buf)
	if a := testing.AllocsPerRun(100, func() {
		lv.Draw(ctx, buf)
		_ = lv.AccessibilityNode(area, false)
	}); a != 0 {
		t.Fatalf("Draw allocates %.0f times", a)
	}
}
