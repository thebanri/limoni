package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type TransducerType uint8

const (
	TransducerFadeColor TransducerType = iota
	TransducerDither
	TransducerSlideLeft
	TransducerSlideRight
	TransducerSlideUp
	TransducerSlideDown
)

var bayer4x4 = [4][4]float64{
	{0.0 / 16.0, 8.0 / 16.0, 2.0 / 16.0, 10.0 / 16.0},
	{12.0 / 16.0, 4.0 / 16.0, 14.0 / 16.0, 6.0 / 16.0},
	{3.0 / 16.0, 11.0 / 16.0, 1.0 / 16.0, 9.0 / 16.0},
	{15.0 / 16.0, 7.0 / 16.0, 13.0 / 16.0, 5.0 / 16.0},
}

type Transducer struct {
	Child    Widget
	Type     TransducerType
	Progress float64 // 0.0 -> 1.0
}

func (t Transducer) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if t.Child == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	if t.Progress >= 1.0 {
		t.Child.Draw(ctx, buf)
		return
	}
	if t.Progress <= 0.0 {
		return
	}

	switch t.Type {
	case TransducerFadeColor:
		tempBuf := buffer.NewBuffer(cell.NewRect(0, 0, ctx.Area.Width, ctx.Area.Height))
		childCtx := cell.NewContext(cell.NewRect(0, 0, ctx.Area.Width, ctx.Area.Height), ctx.Style)
		t.Child.Draw(childCtx, tempBuf)

		for y := uint16(0); y < ctx.Area.Height; y++ {
			for x := uint16(0); x < ctx.Area.Width; x++ {
				cellPtr := tempBuf.Get(x, y)
				if cellPtr == nil || cellPtr.Content == 0 {
					continue
				}
				c := *cellPtr
				// Metin rengini karartarak fade-in uygula
				c.Style.Fg = interpolateColor(cell.NewColorRGB(25, 25, 25), c.Style.Fg, t.Progress)
				c.Style.Bg = interpolateColor(cell.NewColorRGB(25, 25, 25), c.Style.Bg, t.Progress)
				buf.SetCell(ctx.Area.X+x, ctx.Area.Y+y, c)
			}
		}

	case TransducerDither:
		tempBuf := buffer.NewBuffer(cell.NewRect(0, 0, ctx.Area.Width, ctx.Area.Height))
		childCtx := cell.NewContext(cell.NewRect(0, 0, ctx.Area.Width, ctx.Area.Height), ctx.Style)
		t.Child.Draw(childCtx, tempBuf)

		for y := uint16(0); y < ctx.Area.Height; y++ {
			for x := uint16(0); x < ctx.Area.Width; x++ {
				threshold := bayer4x4[y%4][x%4]
				if t.Progress >= threshold {
					cellPtr := tempBuf.Get(x, y)
					if cellPtr != nil {
						buf.SetCell(ctx.Area.X+x, ctx.Area.Y+y, *cellPtr)
					}
				}
			}
		}

	case TransducerSlideLeft, TransducerSlideRight, TransducerSlideUp, TransducerSlideDown:
		var offsetX, offsetY int16
		switch t.Type {
		case TransducerSlideLeft:
			offsetX = int16(float64(ctx.Area.Width) * (1.0 - t.Progress))
		case TransducerSlideRight:
			offsetX = -int16(float64(ctx.Area.Width) * (1.0 - t.Progress))
		case TransducerSlideUp:
			offsetY = int16(float64(ctx.Area.Height) * (1.0 - t.Progress))
		case TransducerSlideDown:
			offsetY = -int16(float64(ctx.Area.Height) * (1.0 - t.Progress))
		}

		tempBuf := buffer.NewBuffer(cell.NewRect(0, 0, ctx.Area.Width, ctx.Area.Height))
		childCtx := cell.NewContext(cell.NewRect(0, 0, ctx.Area.Width, ctx.Area.Height), ctx.Style)
		t.Child.Draw(childCtx, tempBuf)

		for y := uint16(0); y < ctx.Area.Height; y++ {
			for x := uint16(0); x < ctx.Area.Width; x++ {
				srcX := int16(x) - offsetX
				srcY := int16(y) - offsetY
				if srcX >= 0 && srcX < int16(ctx.Area.Width) && srcY >= 0 && srcY < int16(ctx.Area.Height) {
					cellPtr := tempBuf.Get(uint16(srcX), uint16(srcY))
					if cellPtr != nil {
						buf.SetCell(ctx.Area.X+x, ctx.Area.Y+y, *cellPtr)
					}
				}
			}
		}
	}
}

func (t Transducer) SizeHint(maxArea cell.Rect) (width, height uint16) {
	if t.Child == nil {
		return 0, 0
	}
	return t.Child.SizeHint(maxArea)
}

func interpolateColor(from, to cell.Color, progress float64) cell.Color {
	r1, g1, b1 := from.RGB()
	r2, g2, b2 := to.RGB()
	r := uint8(float64(r1) + float64(int(r2)-int(r1))*progress)
	g := uint8(float64(g1) + float64(int(g2)-int(g1))*progress)
	b := uint8(float64(b1) + float64(int(b2)-int(b1))*progress)
	return cell.NewColorRGB(r, g, b)
}
