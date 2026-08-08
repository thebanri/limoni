package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type mockWidget struct {
	text string
}

func (mw mockWidget) Draw(ctx cell.Context, buf *buffer.Buffer) {
	buf.SetString(ctx.Area.X, ctx.Area.Y, mw.text, ctx.Style)
}

func (mw mockWidget) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return uint16(len(mw.text)), 1
}

func TestTransducerFadeColor(t *testing.T) {
	child := mockWidget{text: "A"}
	tr := Transducer{
		Child:    child,
		Type:     TransducerFadeColor,
		Progress: 0.5,
	}

	buf := buffer.NewBuffer(cell.NewRect(0, 0, 5, 5))
	ctx := cell.NewContext(cell.NewRect(0, 0, 5, 5), cell.Style{Fg: cell.NewColorRGB(200, 200, 200)})

	tr.Draw(ctx, buf)

	c := buf.Get(0, 0)
	if c == nil || c.Content != 'A' {
		t.Errorf("Expected content 'A', got %v", c)
	}

	// The color should be interpolated towards target (200,200,200) from black/bg (25,25,25)
	r, g, b := c.Style.Fg.RGB()
	if r == 200 && g == 200 && b == 200 {
		t.Errorf("Expected interpolated Fg color at progress 0.5, got default target %v", c.Style.Fg)
	}
}

func TestTransducerSlide(t *testing.T) {
	child := mockWidget{text: "ABC"}
	tr := Transducer{
		Child:    child,
		Type:     TransducerSlideLeft,
		Progress: 0.66, // Should shift by (1.0 - 0.66) * width = 0.34 * 3 = 1 cell
	}

	buf := buffer.NewBuffer(cell.NewRect(0, 0, 5, 5))
	ctx := cell.NewContext(cell.NewRect(0, 0, 3, 1), cell.Style{})

	tr.Draw(ctx, buf)

	c0 := buf.Get(0, 0)
	c1 := buf.Get(1, 0)
	c2 := buf.Get(2, 0)

	if c0 != nil && c0.Content != ' ' && c0.Content != 0 {
		t.Errorf("Expected column 0 to be empty due to shift, got %c", c0.Content)
	}
	if c1 == nil || c1.Content != 'A' {
		t.Errorf("Expected column 1 to contain 'A', got %v", c1)
	}
	if c2 == nil || c2.Content != 'B' {
		t.Errorf("Expected column 2 to contain 'B', got %v", c2)
	}
}
