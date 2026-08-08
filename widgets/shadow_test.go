package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestDrawShadow(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 12, 8))
	area := cell.NewRect(2, 1, 4, 3)
	original := cell.Style{Fg: cell.NewColorRGB(200, 200, 200), Bg: cell.NewColorRGB(30, 30, 30)}
	for i := range buf.Content {
		buf.Content[i].Content = 'x'
		buf.Content[i].Style = original
	}

	DrawShadow(buf, area, 2, 1)

	shadow := cell.NewColorRGB(6, 7, 9)
	for _, point := range []struct{ x, y uint16 }{
		{6, 2}, {7, 2}, {6, 3}, {7, 3},
		{4, 4}, {5, 4}, {6, 4}, {7, 4},
	} {
		got := buf.Get(point.x, point.y)
		if got.Content != ' ' || got.Style.Fg != shadow || got.Style.Bg != shadow {
			t.Errorf("shadow at (%d,%d) = %+v, want blank cell with shadow color", point.x, point.y, *got)
		}
	}

	for _, point := range []struct{ x, y uint16 }{
		{2, 1}, {3, 1}, {2, 2}, {5, 3}, {2, 4}, {3, 4},
	} {
		got := buf.Get(point.x, point.y)
		if got.Content != 'x' || got.Style != original {
			t.Errorf("cell at (%d,%d) was modified unexpectedly: %+v", point.x, point.y, *got)
		}
	}
}

func TestDrawShadowClipsToBuffer(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 5, 4))
	DrawShadow(buf, cell.NewRect(3, 2, 3, 2), 4, 3)
}
