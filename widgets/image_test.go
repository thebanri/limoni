package widgets

import (
	"image"
	"image/color"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

var (
	red           = color.RGBA{R: 255, A: 255}
	green         = color.RGBA{G: 255, A: 255}
	blue          = color.RGBA{B: 255, A: 255}
	white         = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	transparentPx = color.RGBA{}
)

// pixelColumn builds a 1-pixel-wide image from top to bottom.
func pixelColumn(pixels ...color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 1, len(pixels)))
	for y, p := range pixels {
		img.SetRGBA(0, y, p)
	}
	return img
}

func rgb(c color.RGBA) cell.Color { return cell.NewColorRGB(c.R, c.G, c.B) }

// Half blocks: each cell is ▄ with the lower pixel as foreground and the
// upper as background, so one cell shows two pixels.
func TestImageHalfBlocksShowTwoPixelsACell(t *testing.T) {
	area := cell.NewRect(0, 0, 1, 2)
	buf := buffer.NewBuffer(area)
	im := &Image{Img: pixelColumn(red, green, blue, white), ForceHalfBlock: true}
	im.Draw(cell.NewContext(area, cell.Style{}), buf)

	for y, want := range [][2]color.RGBA{{red, green}, {blue, white}} {
		c := buf.CellAt(0, uint16(y))
		if c.Content != '▄' || c.Style.Bg != rgb(want[0]) || c.Style.Fg != rgb(want[1]) {
			t.Errorf("row %d: %q fg %v bg %v, want ▄ with %v over %v", y, c.Content, c.Style.Fg, c.Style.Bg, want[1], want[0])
		}
	}

	// Drawn again unchanged, the cached cells are the same cells.
	before := buf.CellAt(0, 1)
	im.Draw(cell.NewContext(area, cell.Style{}), buf)
	if buf.CellAt(0, 1) != before {
		t.Error("the cached redraw differs from the first")
	}
}

// Transparent pixels show the widget's background; half-transparent ones are
// blended into it.
func TestImageHalfBlocksTransparency(t *testing.T) {
	bg := cell.NewColorRGB(0, 0, 100)
	area := cell.NewRect(0, 0, 1, 3)
	buf := buffer.NewBuffer(area)
	half := color.RGBA{R: 128, A: 128} // premultiplied: red at half opacity
	(&Image{Img: pixelColumn(transparentPx, transparentPx, transparentPx, green, half, half), ForceHalfBlock: true, Background: bg}).
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if c := buf.CellAt(0, 0); c.Content != ' ' || c.Style.Bg != bg {
		t.Errorf("two transparentPx pixels: %q bg %v, want a blank on the background", c.Content, c.Style.Bg)
	}
	if c := buf.CellAt(0, 1); c.Content != '▄' || c.Style.Fg != rgb(green) || c.Style.Bg != bg {
		t.Errorf("transparentPx over green: fg %v bg %v", c.Style.Fg, c.Style.Bg)
	}
	c := buf.CellAt(0, 2)
	r, g, b := c.Style.Fg.RGB()
	if r < 100 || r > 160 || g != 0 || b < 30 || b > 70 {
		t.Errorf("half-transparent red over (0,0,100) gave (%d,%d,%d), want about (128,0,50)", r, g, b)
	}
}

// With an image protocol the picture is registered with the frame and the
// cells under it are marked, so text does not show through; when the frame
// declines, the image falls back to half blocks.
func TestImageUsesTheProtocolAndFallsBack(t *testing.T) {
	area := cell.NewRect(0, 0, 2, 2)
	ctx := cell.NewContext(area, cell.Style{Bg: cell.NewColorRGB(9, 9, 9)})
	ctx.ImageProtocol = uint8(graphics.ProtocolKitty)
	var gotArea cell.Rect
	var gotZ int
	accept := true
	ctx.RegisterImage = func(a cell.Rect, img image.Image, z int, transparent bool) bool {
		gotArea, gotZ = a, z
		return accept
	}

	buf := buffer.NewBuffer(area)
	(&Image{Img: pixelColumn(red, green), ZIndex: -1, Transparent: true}).Draw(ctx, buf)
	if gotArea != area || gotZ != -1 {
		t.Fatalf("registered %v at z %d", gotArea, gotZ)
	}
	if c := buf.CellAt(1, 1); c.Content != cell.RuneImage || c.Style.Bg != ctx.Style.Bg {
		t.Errorf("cell under the picture: %q bg %v", c.Content, c.Style.Bg)
	}

	accept = false
	buf = buffer.NewBuffer(area)
	(&Image{Img: pixelColumn(red, green)}).Draw(ctx, buf)
	if c := buf.CellAt(0, 0); c.Content != '▄' {
		t.Errorf("a declined picture drew %q, want half blocks", c.Content)
	}
}

// A circle mask clears the corners; opacity fades the picture.
func TestImageMaskAndOpacity(t *testing.T) {
	solid := image.NewRGBA(image.Rect(0, 0, 8, 16))
	for i := range solid.Pix {
		solid.Pix[i] = 255
	}
	bg := cell.NewColorRGB(0, 0, 0)
	area := cell.NewRect(0, 0, 8, 8)

	buf := buffer.NewBuffer(area)
	im := &Image{Img: solid, ForceHalfBlock: true, CircleMask: true, Background: bg}
	im.Draw(cell.NewContext(area, cell.Style{}), buf)
	if c := buf.CellAt(0, 0); c.Content != ' ' {
		t.Errorf("the masked corner drew %q", c.Content)
	}
	if c := buf.CellAt(4, 4); c.Style.Fg != cell.NewColorRGB(255, 255, 255) {
		t.Errorf("the middle of the circle is %v, want white", c.Style.Fg)
	}

	buf = buffer.NewBuffer(area)
	(&Image{Img: solid, ForceHalfBlock: true, Opacity: 0.5, OpacitySet: true, Background: bg}).
		Draw(cell.NewContext(area, cell.Style{}), buf)
	if r, _, _ := buf.CellAt(4, 4).Style.Fg.RGB(); r < 100 || r > 160 {
		t.Errorf("white at half opacity over black gave red %d, want about 128", r)
	}
}
