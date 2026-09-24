package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// dots builds a cell mask from a picture of the 2×4 dots, top row first.
func dots(rows ...string) byte {
	var mask byte
	for y, row := range rows {
		for x, ch := range row {
			if ch == '#' {
				mask |= brailleOffset[y][x]
			}
		}
	}
	return mask
}

func TestMarkerRune(t *testing.T) {
	for _, tc := range []struct {
		name   string
		marker Marker
		mask   byte
		want   rune
	}{
		{"empty", MarkerSextant, 0, ' '},
		{"braille", MarkerBraille, dots("#.", "..", "..", ".#"), '⢁'},
		{"sextant top left", MarkerSextant, dots("#.", "..", "..", ".."), '\U0001FB00'},
		{"sextant top right", MarkerSextant, dots(".#", "..", "..", ".."), '\U0001FB01'},
		{"sextant middle row from dot row 1", MarkerSextant, dots("..", "##", "..", ".."), '\U0001FB0B'}, // sextant-34
		{"sextant middle row from dot row 2", MarkerSextant, dots("..", "..", "##", ".."), '\U0001FB0B'}, // sextant-34
		{"sextant bottom right", MarkerSextant, dots("..", "..", "..", ".#"), '\U0001FB1E'},              // sextant-6
		{"sextant 23456", MarkerSextant, dots(".#", "##", "..", "##"), '\U0001FB3B'},
		{"sextant left column", MarkerSextant, dots("#.", "#.", "..", "#."), '▌'},
		{"sextant right column", MarkerSextant, dots(".#", "..", ".#", ".#"), '▐'},
		{"sextant full", MarkerSextant, dots("##", "..", "##", "##"), '█'},
		{"quadrant top left", MarkerQuadrant, dots("..", "#.", "..", ".."), '▘'},
		{"quadrant diagonal", MarkerQuadrant, dots("#.", "..", "..", ".#"), '▚'},
		{"quadrant bottom", MarkerQuadrant, dots("..", "..", "#.", ".#"), '▄'},
		{"half block top", MarkerHalfBlock, dots(".#", "..", "..", ".."), '▀'},
		{"half block bottom", MarkerHalfBlock, dots("..", "..", "..", "#."), '▄'},
		{"half block both", MarkerHalfBlock, dots("#.", "..", "..", "#."), '█'},
		{"block", MarkerBlock, dots("..", "..", "#.", ".."), '█'},
	} {
		if got := markerRune(tc.marker, tc.mask); got != tc.want {
			t.Errorf("%s: got %q (%U), want %q (%U)", tc.name, got, got, tc.want, tc.want)
		}
	}
}

// Every sextant pattern gets its own character, and each one is a single
// column wide, or the canvas would shift the rest of the row.
func TestSextantRunesAreDistinctAndNarrow(t *testing.T) {
	seen := map[rune]uint8{}
	for bits := uint8(0); bits < 64; bits++ {
		r := sextantRune(bits)
		if prev, dup := seen[r]; dup {
			t.Errorf("bits %d and %d both map to %q", prev, bits, r)
		}
		seen[r] = bits
		if w := cell.RuneWidth(r); w != 1 {
			t.Errorf("%U is %d columns wide", r, w)
		}
	}
}

func TestCanvasDrawsWithMarker(t *testing.T) {
	area := cell.NewRect(0, 0, 2, 1)
	c := NewCanvas(2, 1)
	c.Marker = MarkerQuadrant
	c.Set(0, 0, cell.Style{})
	c.Set(3, 3, cell.Style{})
	buf := buffer.NewBuffer(area)
	c.Draw(cell.NewContext(area, cell.Style{}), buf)
	if got := buf.CellAt(0, 0).Rune(); got != '▘' {
		t.Errorf("cell 0 = %q, want ▘", got)
	}
	if got := buf.CellAt(1, 0).Rune(); got != '▗' {
		t.Errorf("cell 1 = %q, want ▗", got)
	}
}

func BenchmarkCanvasDrawSextant(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	c := NewCanvas(80, 25)
	c.Marker = MarkerSextant
	for i := 0; i < 160; i++ {
		c.Set(i, i%100, cell.Style{})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Draw(ctx, buf)
	}
}

func chartsForBench() (LineChart, PieChart) {
	lc := LineChart{
		Datasets: []LineDataset{{Name: "in", Data: []float64{10, 25, 18, 45, 60, 52, 78, 90}, Color: cell.NewColorRGB(46, 204, 113)}},
		ShowAxes: true, ShowLegend: true, XLabels: []string{"a", "b", "c"},
	}
	pc := PieChart{Data: []PieSlice{{Label: "a", Value: 3}, {Label: "b", Value: 2}}, ShowLegend: true, DonutHoleRatio: 0.4}
	return lc, pc
}

// The charts draw on a canvas every frame and have value receivers, so they
// borrow one; a fresh canvas per frame was several KB of garbage each time.
func TestChartsDrawWithoutAllocating(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	lc, pc := chartsForBench()
	for _, m := range []Marker{MarkerBraille, MarkerSextant} {
		lc.Marker, pc.Marker = m, m
		lc.Draw(ctx, buf)
		pc.Draw(ctx, buf)
		if a := testing.AllocsPerRun(50, func() { lc.Draw(ctx, buf) }); a != 0 {
			t.Errorf("LineChart marker %d: %.0f allocs per Draw", m, a)
		}
		if a := testing.AllocsPerRun(50, func() { pc.Draw(ctx, buf) }); a != 0 {
			t.Errorf("PieChart marker %d: %.0f allocs per Draw", m, a)
		}
	}
}

// A solid pie drawn with sextants uses sextant or block characters, never Braille.
func TestPieChartSextantMarker(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	_, pc := chartsForBench()
	pc.DonutHoleRatio, pc.ShowLegend, pc.Marker = 0, false, MarkerSextant
	pc.Draw(ctx, buf)
	blocks := 0
	for y := uint16(0); y < ctx.Area.Height; y++ {
		for x := uint16(0); x < ctx.Area.Width; x++ {
			r := buf.CellAt(x, y).Rune()
			if r >= 0x2801 && r <= 0x28FF {
				t.Fatalf("Braille %q at %d,%d with MarkerSextant", r, x, y)
			}
			if r == '█' || (r >= 0x1FB00 && r <= 0x1FB3B) || r == '▌' || r == '▐' {
				blocks++
			}
		}
	}
	if blocks < 50 {
		t.Errorf("only %d block cells in a solid pie", blocks)
	}
}

func BenchmarkLineChartDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	lc, _ := chartsForBench()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lc.Draw(ctx, buf)
	}
}

func BenchmarkPieChartDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	_, pc := chartsForBench()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pc.Draw(ctx, buf)
	}
}

func BenchmarkBarChartDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	bc := BarChart{Data: []BarData{{Label: "a", Value: 3}, {Label: "b", Value: 7}, {Label: "c", Value: 5}}, ShowValues: true}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bc.Draw(ctx, buf)
	}
}
