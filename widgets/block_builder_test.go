package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// drawBlock draws b into a fresh buffer of the given size.
func drawBlock(b *Block, w, h uint16) *buffer.Buffer {
	area := cell.NewRect(0, 0, w, h)
	buf := buffer.NewBuffer(area)
	b.Draw(cell.NewContext(area, cell.Style{}), buf)
	return buf
}

func TestBlockBorderSymbolBuilders(t *testing.T) {
	cases := []struct {
		name  string
		apply func(*Block) *Block
		want  BorderSymbols
	}{
		{"Single", (*Block).Single, SymbolsSingle},
		{"Double", (*Block).Double, SymbolsDouble},
		{"Thick", (*Block).Thick, SymbolsThick},
		{"BlockBorder", (*Block).BlockBorder, SymbolsBlock},
		{"Rounded", (*Block).Rounded, SymbolsRounded},
	}
	// The sets must differ from one another, or a builder wired to the wrong
	// one could still pass below.
	for i := range cases {
		for j := i + 1; j < len(cases); j++ {
			if cases[i].want == cases[j].want {
				t.Fatalf("%s and %s share a symbol set", cases[i].name, cases[j].name)
			}
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Start from a different set, so a builder that does nothing fails.
			b := &Block{Borders: BorderAll, BorderSymbols: SymbolsInnerHalfBlock}
			if got := tc.apply(b); got != b {
				t.Fatal("builder did not return its receiver")
			}
			if b.BorderSymbols != tc.want {
				t.Fatalf("BorderSymbols = %+v, want %+v", b.BorderSymbols, tc.want)
			}
			if b.Borders != BorderAll {
				t.Fatalf("Borders changed to %04b", b.Borders)
			}

			buf := drawBlock(b, 6, 3)
			for _, c := range []struct {
				x, y uint16
				want rune
			}{
				{0, 0, tc.want.TopLeft},
				{5, 0, tc.want.TopRight},
				{0, 2, tc.want.BottomLeft},
				{5, 2, tc.want.BottomRight},
				{2, 0, tc.want.Horizontal},
				{2, 2, tc.want.Horizontal},
				{0, 1, tc.want.Vertical},
				{5, 1, tc.want.Vertical},
			} {
				if got := buf.Get(c.x, c.y).Content; got != c.want {
					t.Errorf("(%d,%d) = %q, want %q", c.x, c.y, got, c.want)
				}
			}
		})
	}

	if got := NewBlock().Double().Thick().BorderSymbols; got != SymbolsThick {
		t.Fatalf("the last border builder should win, got %+v", got)
	}
}

func TestBlockWithBorders(t *testing.T) {
	b := NewBlock()
	if got := b.WithBorders(BorderTop | BorderBottom); got != b {
		t.Fatal("WithBorders did not return its receiver")
	}
	if b.Borders != BorderTop|BorderBottom {
		t.Fatalf("Borders = %04b, want %04b", b.Borders, BorderTop|BorderBottom)
	}
	if b.BorderSymbols != SymbolsRounded {
		t.Fatal("WithBorders changed the symbol set")
	}

	// Only the top and bottom lines are drawn: no sides, so no corners either.
	buf := drawBlock(b, 6, 3)
	if got := buf.Get(2, 0).Content; got != '─' {
		t.Errorf("top line = %q, want '─'", got)
	}
	if got := buf.Get(2, 2).Content; got != '─' {
		t.Errorf("bottom line = %q, want '─'", got)
	}
	for _, p := range [][2]uint16{{0, 0}, {5, 0}, {0, 1}, {5, 1}, {0, 2}, {5, 2}} {
		if got := buf.Get(p[0], p[1]).Content; got != ' ' {
			t.Errorf("(%d,%d) = %q, want blank: the sides are off", p[0], p[1], got)
		}
	}
	if got, want := b.Inner(cell.NewRect(0, 0, 6, 3)), cell.NewRect(0, 1, 6, 1); got != want {
		t.Errorf("Inner = %+v, want %+v", got, want)
	}

	b.WithBorders(BorderNone)
	area := cell.NewRect(0, 0, 6, 3)
	if got := b.Inner(area); got != area {
		t.Errorf("Inner with no borders = %+v, want the whole area", got)
	}
	if w, h := b.SizeHint(area); w != 0 || h != 0 {
		t.Errorf("SizeHint with no borders = %dx%d, want 0x0", w, h)
	}
}

func TestBlockStyleBuilders(t *testing.T) {
	titleStyle := cell.Style{Fg: cell.NewColorANSI(2)}
	borderStyle := cell.Style{Fg: cell.NewColorANSI(1)}
	blockStyle := cell.Style{Bg: cell.NewColorANSI(4)}

	// Each builder sets its own field and leaves the other two alone.
	if b := NewBlock(); b.WithTitleStyle(titleStyle) != b || b.TitleStyle != titleStyle ||
		b.BorderStyle != (cell.Style{}) || b.Style != (cell.Style{}) {
		t.Fatalf("WithTitleStyle: %+v", b)
	}
	if b := NewBlock(); b.WithBorderStyle(borderStyle) != b || b.BorderStyle != borderStyle ||
		b.TitleStyle != (cell.Style{}) || b.Style != (cell.Style{}) {
		t.Fatalf("WithBorderStyle: %+v", b)
	}
	if b := NewBlock(); b.WithStyle(blockStyle) != b || b.Style != blockStyle ||
		b.TitleStyle != (cell.Style{}) || b.BorderStyle != (cell.Style{}) {
		t.Fatalf("WithStyle: %+v", b)
	}

	// And each one reaches the cells it is meant for.
	b := NewBlock().WithTitle("T").WithTitleStyle(titleStyle).
		WithBorderStyle(borderStyle).WithStyle(blockStyle)
	buf := drawBlock(b, 10, 3)

	// " T " starts at x=2; the title glyph is at x=3.
	if c := buf.Get(3, 0); c.Content != 'T' || c.Style.Fg != titleStyle.Fg || c.Style.Bg != blockStyle.Bg {
		t.Errorf("title cell %q %+v, want 'T' with the title Fg on the block Bg", c.Content, c.Style)
	}
	if c := buf.Get(7, 0); c.Content != '─' || c.Style.Fg != borderStyle.Fg || c.Style.Bg != blockStyle.Bg {
		t.Errorf("border cell %q %+v, want '─' with the border Fg on the block Bg", c.Content, c.Style)
	}
	if c := buf.Get(4, 1); c.Style.Bg != blockStyle.Bg || c.Style.Fg == borderStyle.Fg || c.Style.Fg == titleStyle.Fg {
		t.Errorf("inner cell %+v, want the block Bg and neither the border nor the title Fg", c.Style)
	}
}

func TestUniformInsets(t *testing.T) {
	if got, want := UniformInsets(3), (Insets{Top: 3, Right: 3, Bottom: 3, Left: 3}); got != want {
		t.Fatalf("UniformInsets(3) = %+v, want %+v", got, want)
	}
	if got := UniformInsets(0); got != (Insets{}) {
		t.Fatalf("UniformInsets(0) = %+v, want zero insets", got)
	}

	// border 1 + padding 1 on every side, then margin 1 on top of that.
	area := cell.NewRect(0, 0, 10, 8)
	b := Block{Borders: BorderAll, Padding: UniformInsets(1)}
	if got, want := b.Inner(area), cell.NewRect(2, 2, 6, 4); got != want {
		t.Errorf("Inner with padding = %+v, want %+v", got, want)
	}
	if w, h := b.SizeHint(area); w != 4 || h != 4 {
		t.Errorf("SizeHint with padding = %dx%d, want 4x4", w, h)
	}
	b.Margin = UniformInsets(1)
	if got, want := b.Inner(area), cell.NewRect(3, 3, 4, 2); got != want {
		t.Errorf("Inner with padding and margin = %+v, want %+v", got, want)
	}
	if w, h := b.SizeHint(area); w != 6 || h != 6 {
		t.Errorf("SizeHint with padding and margin = %dx%d, want 6x6", w, h)
	}
}

func TestBlockSizeHintTitleAndBounds(t *testing.T) {
	// With no child the title sets the width: " Logs " plus two corners.
	b := NewBlock().WithTitle("Logs")
	if w, h := b.SizeHint(cell.NewRect(0, 0, 40, 10)); w != 8 || h != 2 {
		t.Errorf("SizeHint = %dx%d, want 8x2", w, h)
	}
	// The hint never exceeds the space on offer.
	if w, h := b.SizeHint(cell.NewRect(0, 0, 5, 1)); w != 5 || h != 1 {
		t.Errorf("SizeHint in a 5x1 area = %dx%d, want 5x1", w, h)
	}
	// A child wider than the title wins, plus the border.
	b.WithChild(dummyWidget{width: 12, height: 3})
	if w, h := b.SizeHint(cell.NewRect(0, 0, 40, 10)); w != 14 || h != 5 {
		t.Errorf("SizeHint with a child = %dx%d, want 14x5", w, h)
	}
	m := b.Measure(cell.NewRect(0, 0, 40, 10))
	if m.IdealWidth != 14 || m.IdealHeight != 5 || m.MaxWidth != 40 || m.MaxHeight != 10 {
		t.Errorf("Measure = %+v, want ideal 14x5 within 40x10", m)
	}
}
