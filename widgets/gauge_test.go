package widgets

import (
	"math"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func drawRow(w interface {
	Draw(cell.Context, *buffer.Buffer)
}, width, height uint16) (*buffer.Buffer, string) {
	area := cell.NewRect(0, 0, width, height)
	buf := buffer.NewBuffer(area)
	w.Draw(cell.NewContext(area, cell.Style{}), buf)
	row := make([]rune, width)
	for x := uint16(0); x < width; x++ {
		row[x] = buf.CellAt(x, height/2).Rune()
	}
	return buf, string(row)
}

func TestGaugeFillsInEighths(t *testing.T) {
	for _, tc := range []struct {
		ratio float64
		want  string
	}{
		{0, "          "},
		{1, "██████████"},
		{0.5, "█████     "},
		{0.55, "█████▌    "},   // 5.5 cells
		{0.5125, "█████▏    "}, // 5.125 cells
		{-3, "          "},
		{7, "██████████"},
		{math.NaN(), "          "},
	} {
		_, got := drawRow(Gauge{Ratio: tc.ratio, HideLabel: true}, 10, 1)
		if got != tc.want {
			t.Errorf("ratio %v: %q, want %q", tc.ratio, got, tc.want)
		}
	}
	_, got := drawRow(Gauge{Ratio: 0.55, HideLabel: true, WholeCells: true}, 10, 1)
	if got != "██████    " {
		t.Errorf("WholeCells rounds 5.5 up: %q", got)
	}
}

// The label sits in the middle row, centred; the half over the fill is
// reversed so it stays readable against the fill colour.
func TestGaugeLabel(t *testing.T) {
	g := Gauge{Ratio: 0.5, GaugeStyle: cell.Style{Fg: cell.NewColorRGB(0, 200, 0)}}
	buf, row := drawRow(g, 10, 3)
	if row != "███50%    " { // "50%" centred: (10-3)/2 = 3
		t.Fatalf("middle row %q", row)
	}
	if top := buf.CellAt(4, 0).Rune(); top != '█' {
		t.Errorf("the label belongs to the middle row only; top row has %q", top)
	}
	over, beside := buf.CellAt(4, 1).Style, buf.CellAt(5, 1).Style
	if over.Modifier&cell.ModifierReverse == 0 || over.Fg != g.GaugeStyle.Fg {
		t.Errorf("label over the fill: %+v, want reversed fill colour", over)
	}
	if beside.Modifier&cell.ModifierReverse != 0 {
		t.Errorf("label beside the fill is reversed: %+v", beside)
	}

	// A label is centred by its width in columns, clusters included: 8 of 12.
	buf, _ = drawRow(Gauge{Ratio: 0, Label: "héllo 👋🏽"}, 12, 1)
	if got := buf.CellAt(2, 0).Rune(); got != 'h' {
		t.Errorf("custom label starts with %q at column 2, want h", got)
	}
}

func TestLineGauge(t *testing.T) {
	for _, tc := range []struct {
		g    LineGauge
		want string
	}{
		{LineGauge{Ratio: 0.5}, "50% ━━━━━━━━────────"}, // "50% " then 16 cells of line
		{LineGauge{Ratio: 1, Label: "Up"}, "Up ━━━━━━━━━━━━━━━━━"},
		{LineGauge{Ratio: 0.25, HideLabel: true, FilledRune: '=', UnfilledRune: '.'}, "=====..............."},
		{LineGauge{Ratio: 0.3, Label: "a very long label indeed"}, "a very long label in"},
	} {
		if _, got := drawRow(tc.g, 20, 1); got != tc.want {
			t.Errorf("%+v:\n got %q\nwant %q", tc.g, got, tc.want)
		}
	}
}

func TestGaugesDoNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	for name, w := range map[string]interface {
		Draw(cell.Context, *buffer.Buffer)
	}{
		"gauge":           Gauge{Ratio: 0.37},
		"gauge label":     Gauge{Ratio: 0.37, Label: "Disk"},
		"line gauge":      LineGauge{Ratio: 0.37},
		"line gauge text": LineGauge{Ratio: 0.37, Label: "Disk"},
	} {
		if a := testing.AllocsPerRun(50, func() { w.Draw(ctx, buf) }); a != 0 {
			t.Errorf("%s: %.0f allocs per Draw", name, a)
		}
	}
}

func BenchmarkGaugeDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	g := Gauge{Ratio: 0.37}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g.Draw(ctx, buf)
	}
}

func BenchmarkLineGaugeDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	g := LineGauge{Ratio: 0.37, Label: "Disk"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g.Draw(ctx, buf)
	}
}
