package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// shownBg is the background a terminal shows for style: reverse video shows
// the foreground there.
func shownBg(s cell.Style) cell.Color {
	if s.Modifier&cell.ModifierReverse != 0 {
		return s.Fg
	}
	return s.Bg
}

// contrast is how far apart two colours' brightness is, 0 to 255.
func contrast(a, b cell.Color) int {
	lum := func(c cell.Color) int {
		r, g, b := c.RGB()
		return (299*int(r) + 587*int(g) + 114*int(b)) / 1000
	}
	d := lum(a) - lum(b)
	if d < 0 {
		d = -d
	}
	return d
}

// The cursor over a blank cell must stand out from the blank beside it.
// It was drawn black on white in reverse video, which the terminal shows as
// white on black: invisible on a dark input.
func TestTheCursorShowsOverABlank(t *testing.T) {
	dark := cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(20, 20, 24)}
	area := cell.NewRect(0, 0, 10, 1)

	input := NewTextInputState()
	input.SetValue("hi")
	buf := buffer.NewBuffer(area)
	TextInput{ID: "in", State: input, Style: dark, Focused: true}.Draw(cell.NewContext(area, cell.Style{}), buf)
	if c := contrast(shownBg(buf.CellAt(2, 0).Style), shownBg(buf.CellAt(3, 0).Style)); c < 100 {
		t.Errorf("TextInput's cursor stands out from the blank beside it by %d of 255", c)
	}

	text := NewTextAreaState()
	text.SetValue("hi")
	buf = buffer.NewBuffer(area)
	TextArea{ID: "ta", State: text, Style: dark, Focused: true}.Draw(cell.NewContext(area, cell.Style{}), buf)
	if c := contrast(shownBg(buf.CellAt(2, 0).Style), shownBg(buf.CellAt(3, 0).Style)); c < 100 {
		t.Errorf("TextArea's cursor stands out from the blank beside it by %d of 255", c)
	}
}
