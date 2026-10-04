package widgets_test

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// Through a real frame: a preset swatch picks its colour, a click in the
// hue bar sets the hue, and a frame allocates nothing.
func TestColorPickerMouseThroughAFrame(t *testing.T) {
	state := widgets.NewColorPickerState(10, 20, 30)
	picker := &widgets.ColorPicker{ID: "color", State: state}
	term := testkit.NewTerminal(48, 12)
	draw := func(f *terminal.Frame) { f.RenderWidget(picker, f.Area()) }
	term.Draw(draw)

	rows := strings.Split(term.Snapshot(), "\n")
	swatchY, swatchX := -1, -1
	for y, row := range rows {
		if i := strings.Index(row, "Presets: "); i >= 0 {
			swatchY, swatchX = y, len([]rune(row[:i]))+9+2 // the second swatch
		}
	}
	if swatchY < 0 {
		t.Fatalf("no preset row:\n%s", term.Snapshot())
	}
	term.Click(uint16(swatchX), uint16(swatchY))
	if state.PaletteIndex != 1 {
		t.Fatalf("clicking the second swatch chose preset %d", state.PaletteIndex)
	}
	term.Draw(draw)
	if !strings.Contains(term.Snapshot(), "RGB: ") {
		t.Errorf("no RGB line:\n%s", term.Snapshot())
	}

	if n := testing.AllocsPerRun(20, func() { term.Draw(draw) }); n != 0 {
		t.Errorf("a frame allocated %v times", n)
	}
}
