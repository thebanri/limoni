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
	// Img, gösterilecek olan ham resim nesnesidir.
	Img image.Image
	// ZIndex, resmin dikey katman yerleşim sırasıdır. Negatif değerler (örneğin -1)
	// resmin metin hücrelerinin arkasına (underneath text) çizilmesini sağlar.
	ZIndex int
	// ForceHalfBlock, aktif edilirse donanımsal protokoller yerine hücre tabanlı half-block yöntemini zorlar.
	ForceHalfBlock bool
	// CircleMask, resmi daire şeklinde kırpar (avatar).
	CircleMask bool
}

// Draw, çizim alanındaki hücrelerin içeriğini boşluk karakteriyle temizler
// ve resmi çizim çerçevesine (Frame) kaydeder.
// Eğer hedef terminal görsel protokollerini desteklemiyorsa, Half-Block (U+2584)
// yöntemiyle doğrudan hücre tamponu üzerine 1x2 çözünürlüklü resim çizer.
func (im Image) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if im.Img == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	img := im.Img
	if im.CircleMask {
		img = graphics.ApplyCircleMask(img)
	}

	proto := graphics.DetectProtocol()
	if im.ForceHalfBlock || proto == graphics.ProtocolHalfBlock {
		// Half-Block modu: Resmi doğrudan buffer.Buffer hücrelerine karakterler halinde çiz.
		// Genişlik = Hücre Genişliği, Yükseklik = Hücre Yüksekliği * 2 (Her hücre dikeyde 2 piksel barındırır).
		targetW := int(ctx.Area.Width)
		targetH := int(ctx.Area.Height) * 2

		resized := graphics.ResizeImage(img, targetW, targetH)

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
				}
			}
		}
		return
	}

	// Yerel grafik protokol modları: Metinlerin resmin arkasından taşmasını önlemek için alanı resim işaretiyle doldur
	for y := ctx.Area.Y; y < ctx.Area.Y+ctx.Area.Height; y++ {
		for x := ctx.Area.X; x < ctx.Area.X+ctx.Area.Width; x++ {
			if c := buf.Get(x, y); c != nil {
				c.Content = cell.RuneImage
				c.Style.Reset()
			}
		}
	}

	// Çizim bağlamında tanımlıysa resmi yerel protokole kaydet
	if ctx.RegisterImage != nil {
		ctx.RegisterImage(ctx.Area, img, im.ZIndex)
	}
}

// SizeHint, resmin kaplayacağı alanı belirler. Varsayılan olarak kendisine tahsis
// edilmek istenen maksimum alanı dolduracak şekilde maksimum satır ve sütun boyutunu döner.
func (im Image) SizeHint(maxArea cell.Rect) (width, height uint16) {
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
