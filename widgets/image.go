package widgets

import (
	"image"
	"image/color"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// imageProtocol is the image protocol the terminal drawing ctx settled on,
// which may come from the capability handshake (a Sixel terminal found by
// its DA1 answer). Outside a terminal it is what the environment suggests.
func imageProtocol(ctx cell.Context) graphics.Protocol {
	if ctx.ImageProtocol != 0 {
		return graphics.Protocol(ctx.ImageProtocol)
	}
	return graphics.DetectProtocol()
}

// Image draws real pictures such as PNG/JPG in the terminal, using the
// native image protocols (Kitty, Sixel, iTerm2).
type Image struct {
	// ID is the widget's focus ID.
	ID string
	// Img is the raw image to show.
	Img image.Image
	// ZIndex is the image's layer order. Negative values (e.g. -1)
	// draw the image underneath the text cells.
	ZIndex int
	// ForceHalfBlock, when set, forces the cell-based half-block method instead of the hardware protocols.
	ForceHalfBlock bool
	// CircleMask crops the image to a circle (for avatars).
	CircleMask bool
	// OpaqueBackground composites transparency over Background before native rendering.
	OpaqueBackground bool
	Background       cell.Color
	// Transparent reports whether the image's transparent pixels are kept.
	Transparent bool
	// Opacity is the image's opacity (between 0.0 and 1.0).
	Opacity float64
	// OpacitySet reports that Opacity was set on purpose,
	// so that 0.0 can be told apart from the default (unset) value.
	OpacitySet bool
	// FocusedStyle is the border/highlight style applied when focused.
	FocusedStyle cell.Style

	// Cache fields
	lastImg       image.Image
	lastArea      cell.Rect
	cachedCells   []cell.Cell
	lastSrcImg    image.Image
	lastCircle    bool
	lastMaskedImg image.Image
}

// Draw blanks the cells of the drawing area with spaces and registers
// the image with the Frame.
// If the terminal supports no image protocol, it draws the image straight
// into the cell buffer with half blocks (U+2584), at 1x2 resolution.
func (im *Image) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if im.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(im.ID)
	}
	// A click focuses the widget; registered as data, so it does not allocate.
	if ctx.RegisterClickAction != nil && im.ID != "" {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: im.ID})
	} else if im.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(im.ID)
			}
		})
	}
	if im.Img == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	img := im.Img
	if im.CircleMask {
		if im.Img == im.lastSrcImg && im.lastCircle && im.lastMaskedImg != nil {
			img = im.lastMaskedImg
		} else {
			img = graphics.ApplyCircleMask(im.Img)
			im.lastSrcImg = im.Img
			im.lastCircle = true
			im.lastMaskedImg = img
		}
	} else {
		im.lastCircle = false
		im.lastMaskedImg = nil
	}
	if im.OpacitySet && im.Opacity < 1.0 {
		img = graphics.ApplyOpacity(img, im.Opacity)
	}
	bgCol := im.Background
	if bgCol.Type() == cell.ColorDefault {
		bgCol = ctx.Style.Bg
	}
	if im.OpaqueBackground && bgCol.Type() != cell.ColorDefault {
		r, g, b := bgCol.RGB()
		img = graphics.FlattenImage(img, color.RGBA{R: r, G: g, B: b, A: 255})
	}

	proto := imageProtocol(ctx)
	if !im.ForceHalfBlock && !im.OpaqueBackground && !im.Transparent && proto != graphics.ProtocolHalfBlock {
		// Native image protocols leave transparent pixels to the terminal's default
		// background. In most terminals that is black, which makes a rectangle or
		// band that differs from the widget's background.
		r, g, b := ctx.Style.Bg.RGB()
		img = graphics.FlattenImage(img, color.RGBA{R: r, G: g, B: b, A: 255})
	}
	if im.ForceHalfBlock || proto == graphics.ProtocolHalfBlock {
		im.drawHalfBlock(ctx, buf, img)
		return
	}

	registered := false
	if ctx.RegisterImage != nil {
		registered = ctx.RegisterImage(ctx.Area, img, im.ZIndex, im.Transparent)
	}

	if registered {
		// Native image protocol modes: fill the area with image markers so text does not show through from behind the image
		for y := ctx.Area.Y; y < ctx.Area.Y+ctx.Area.Height; y++ {
			for x := ctx.Area.X; x < ctx.Area.X+ctx.Area.Width; x++ {
				if c := buf.Get(x, y); c != nil {
					c.Content = cell.RuneImage
					c.Style.Reset()
					if im.Transparent {
						// Native image protocols composite transparent pixels over the
						// terminal cell background. Resetting the style here would turn
						// that background into the terminal default (usually black).
						c.Style.Bg = ctx.Style.Bg
					}
				}
			}
		}
	} else {
		// Without an image protocol, fall back to half-block mode
		im.drawHalfBlock(ctx, buf, img)
	}
}

func (im *Image) drawHalfBlock(ctx cell.Context, buf *buffer.Buffer, img image.Image) {
	targetW := int(ctx.Area.Width)
	targetH := int(ctx.Area.Height) * 2

	bgCol := im.Background
	if bgCol.Type() == cell.ColorDefault {
		bgCol = ctx.Style.Bg
	}
	if bgCol.Type() == cell.ColorDefault && ctx.ThemeStyle != nil {
		bgCol = ctx.ThemeStyle("surface").Bg
	}

	if im.lastImg == img && im.lastArea == ctx.Area && len(im.cachedCells) == int(ctx.Area.Width)*int(ctx.Area.Height) {
		idx := 0
		for cy := uint16(0); cy < ctx.Area.Height; cy++ {
			for cx := uint16(0); cx < ctx.Area.Width; cx++ {
				cellX := ctx.Area.X + cx
				cellY := ctx.Area.Y + cy
				if c := buf.Get(cellX, cellY); c != nil {
					*c = im.cachedCells[idx]
				}
				idx++
			}
		}
		return
	}

	resized := graphics.ResizeImageContain(img, targetW, targetH, true)

	im.lastImg = img
	im.lastArea = ctx.Area
	im.cachedCells = make([]cell.Cell, int(ctx.Area.Width)*int(ctx.Area.Height))

	idx := 0
	for cy := uint16(0); cy < ctx.Area.Height; cy++ {
		for cx := uint16(0); cx < ctx.Area.Width; cx++ {
			cellX := ctx.Area.X + cx
			cellY := ctx.Area.Y + cy

			// Fallback to existing cell background if bgCol is still default
			effCellBg := bgCol
			if effCellBg.Type() == cell.ColorDefault {
				if cur := buf.Get(cellX, cellY); cur != nil && cur.Style.Bg.Type() != cell.ColorDefault {
					effCellBg = cur.Style.Bg
				}
			}

			// Top pixel (becomes the background colour)
			topCol := resized.At(int(cx), int(2*cy))
			_, _, _, ta := topCol.RGBA()

			// Bottom pixel (becomes the foreground colour)
			botCol := resized.At(int(cx), int(2*cy+1))
			_, _, _, ba := botCol.RGBA()

			const alphaMin = 4000 // ~6% alpha threshold to filter transparent compression noise
			topOpaque := ta >= alphaMin
			botOpaque := ba >= alphaMin

			bgColor := effCellBg
			if topOpaque {
				bgColor = blendColor(topCol, effCellBg)
			}
			fgColor := effCellBg
			if botOpaque {
				fgColor = blendColor(botCol, effCellBg)
			}

			// Update the cell
			if c := buf.Get(cellX, cellY); c != nil {
				c.Style.Modifier = cell.ModifierReset

				if !topOpaque && !botOpaque {
					// Both pixels transparent -> a space
					c.Content = ' '
					c.Style.Fg = cell.NewColorDefault()
					c.Style.Bg = effCellBg
				} else if topOpaque && !botOpaque {
					// Top filled, bottom transparent -> lower half block (▄) with Bg the top pixel, Fg the background
					effBg := effCellBg
					if effBg.Type() == cell.ColorDefault {
						effBg = cell.NewColorRGB(0, 0, 0)
					}
					c.Content = '▄'
					c.Style.Fg = effBg
					c.Style.Bg = bgColor
				} else if !topOpaque && botOpaque {
					// Top transparent, bottom filled -> lower half block (▄) with Fg the bottom pixel, Bg the background
					effBg := effCellBg
					if effBg.Type() == cell.ColorDefault {
						effBg = cell.NewColorRGB(0, 0, 0)
					}
					c.Content = '▄'
					c.Style.Fg = fgColor
					c.Style.Bg = effBg
				} else {
					// Both filled -> lower half block (▄) with Fg the bottom pixel, Bg the top pixel
					c.Content = '▄'
					c.Style.Fg = fgColor
					c.Style.Bg = bgColor
				}
				im.cachedCells[idx] = *c
			}
			idx++
		}
	}
}

// SizeHint sets the area the image covers. By default it returns the largest rows and
// columns, filling the maximum area offered to it.
func (im *Image) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

// blendColor combines semi-transparent or fully transparent image pixels with the
// container's background colour by alpha blending.
func blendColor(fgColor color.Color, bg cell.Color) cell.Color {
	// RGBA returns colour already multiplied by alpha, so the background is
	// what is added, not a second weighting of the foreground: multiplying
	// by alpha again drew a half-transparent edge at a quarter strength, a
	// dark fringe around every anti-aliased picture.
	r, g, b, a := fgColor.RGBA()
	if a < 4000 {
		return bg
	}
	if a >= 65000 {
		return cell.NewColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
	if bg.Type() == cell.ColorDefault {
		// The background is unknown: show the colour itself.
		return cell.NewColorRGB(uint8(r*0xFFFF/a>>8), uint8(g*0xFFFF/a>>8), uint8(b*0xFFFF/a>>8))
	}
	rest := 1 - float64(a)/0xFFFF
	bgR, bgG, bgB := bg.RGB()
	mix := func(premultiplied uint32, back uint8) uint8 {
		return uint8(min(255, float64(premultiplied)/257+float64(back)*rest+0.5))
	}
	return cell.NewColorRGB(mix(r, bgR), mix(g, bgG), mix(b, bgB))
}
