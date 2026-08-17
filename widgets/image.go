package widgets

import (
	"image"
	"image/color"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// Image, terminalde yerel görsel protokolleri (Kitty, Sixel, iTerm2) kullanarak
// PNG/JPG gibi gerçek resimleri çizebilen TUI bileşenidir.
type Image struct {
	// ID, widget odak kimliğidir.
	ID string
	// Img, gösterilecek olan ham resim nesnesidir.
	Img image.Image
	// ZIndex, resmin dikey katman yerleşim sırasıdır. Negatif değerler (örneğin -1)
	// resmin metin hücrelerinin arkasına (underneath text) çizilmesini sağlar.
	ZIndex int
	// ForceHalfBlock, aktif edilirse donanımsal protokoller yerine hücre tabanlı half-block yöntemini zorlar.
	ForceHalfBlock bool
	// CircleMask, resmi daire şeklinde kırpar (avatar).
	CircleMask bool
	// OpaqueBackground composites transparency over Background before native rendering.
	OpaqueBackground bool
	Background       cell.Color
	// Transparent, resmin şeffaf piksellerinin korunup korunmayacağını belirtir.
	Transparent bool
	// Opacity, resmin opaklık değeridir (0.0 ile 1.0 arasında).
	Opacity float64
	// OpacitySet, Opacity alanının bilinçli olarak ayarlandığını belirtir.
	// Böylece 0.0 değeri ile varsayılan (belirtilmemiş) değer ayrıştırılır.
	OpacitySet bool
	// FocusedStyle, odaklandığında uygulanacak kenar/vurgu stilidir.
	FocusedStyle cell.Style

	// Cache fields
	lastImg     image.Image
	lastArea    cell.Rect
	cachedCells []cell.Cell
}

// Draw, çizim alanındaki hücrelerin içeriğini boşluk karakteriyle temizler
// ve resmi çizim çerçevesine (Frame) kaydeder.
// Eğer hedef terminal görsel protokollerini desteklemiyorsa, Half-Block (U+2584)
// yöntemiyle doğrudan hücre tamponu üzerine 1x2 çözünürlüklü resim çizer.
func (im *Image) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if im.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(im.ID)
	}
	if im.ID != "" && ctx.RegisterClick != nil {
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
		img = graphics.ApplyCircleMask(img)
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

	proto := graphics.DetectProtocol()
	if !im.ForceHalfBlock && !im.OpaqueBackground && !im.Transparent && proto != graphics.ProtocolHalfBlock {
		// Native image protocols transparan pikselleri terminalin varsayılan
		// arka planına bırakır. Bu renk çoğu terminalde siyahtır ve widget'ın
		// arka planından farklı bir dikdörtgen/şerit oluşturur.
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
		// Yerel grafik protokol modları: Metinlerin resmin arkasından taşmasını önlemek için alanı resim işaretiyle doldur
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
		// Grafik protokolü yoksa half-block moduna düş
		im.drawHalfBlock(ctx, buf, img)
	}
}

func (im *Image) drawHalfBlock(ctx cell.Context, buf *buffer.Buffer, img image.Image) {
	targetW := int(ctx.Area.Width)
	targetH := int(ctx.Area.Height) * 2

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

	resized := graphics.ResizeImage(img, targetW, targetH)

	im.lastImg = img
	im.lastArea = ctx.Area
	im.cachedCells = make([]cell.Cell, int(ctx.Area.Width)*int(ctx.Area.Height))

	idx := 0
	for cy := uint16(0); cy < ctx.Area.Height; cy++ {
		for cx := uint16(0); cx < ctx.Area.Width; cx++ {
			// Üst piksel (Background rengi olacak)
			topCol := resized.At(int(cx), int(2*cy))
			_, _, _, ta := topCol.RGBA()
			bgColor := blendColor(topCol, ctx.Style.Bg)

			// Alt piksel (Foreground rengi olacak)
			botCol := resized.At(int(cx), int(2*cy+1))
			_, _, _, ba := botCol.RGBA()
			fgColor := blendColor(botCol, ctx.Style.Bg)

			// Hücreyi güncelle
			cellX := ctx.Area.X + cx
			cellY := ctx.Area.Y + cy
			if c := buf.Get(cellX, cellY); c != nil {
				c.Style.Modifier = cell.ModifierReset

				if ta == 0 && ba == 0 {
					// Her iki piksel de şeffaf -> Boşluk karakteri çizerek terminal varsayılan Fg'sinin çizilmesini engelle
					c.Content = ' '
					c.Style.Bg = ctx.Style.Bg
				} else if ta > 0 && ba == 0 {
					// Üst dolu, alt şeffaf -> Üst yarım blok (▀)
					c.Content = '▀'
					c.Style.Fg = bgColor
					c.Style.Bg = ctx.Style.Bg
				} else {
					// İkisi de dolu veya alt dolu -> Alt yarım blok (▄)
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

// SizeHint, resmin kaplayacağı alanı belirler. Varsayılan olarak kendisine tahsis
// edilmek istenen maksimum alanı dolduracak şekilde maksimum satır ve sütun boyutunu döner.
func (im *Image) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

// blendColor, yarı-transparan veya tamamen transparan resim piksellerini
// konteyner arka plan rengiyle alfa-harmanlama (alpha blending) formülüyle birleştirir.
func blendColor(fgColor color.Color, bg cell.Color) cell.Color {
	r, g, b, a := fgColor.RGBA()
	if a == 0 {
		return bg
	}
	if a == 65535 {
		return cell.NewColorRGB(uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}

	alpha := float64(a) / 65535.0

	// Foreground renk kanalları
	fgR := uint8(r >> 8)
	fgG := uint8(g >> 8)
	fgB := uint8(b >> 8)

	// Background renk kanalları
	bgR, bgG, bgB := bg.RGB()

	// Alfa harmanlama formülü: C = C_fg * alpha + C_bg * (1 - alpha)
	blendR := uint8(float64(fgR)*alpha + float64(bgR)*(1.0-alpha))
	blendG := uint8(float64(fgG)*alpha + float64(bgG)*(1.0-alpha))
	blendB := uint8(float64(fgB)*alpha + float64(bgB)*(1.0-alpha))

	return cell.NewColorRGB(blendR, blendG, blendB)
}
