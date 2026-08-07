package widgets

import (
	"image"

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
				tr, tg, tb, _ := topCol.RGBA()
				bgColor := cell.NewColorRGB(uint8(tr>>8), uint8(tg>>8), uint8(tb>>8))

				// Alt piksel (Foreground rengi olacak)
				botCol := resized.At(int(cx), int(2*cy+1))
				br, bg, bb, _ := botCol.RGBA()
				fgColor := cell.NewColorRGB(uint8(br>>8), uint8(bg>>8), uint8(bb>>8))

				// Hücreyi güncelle
				cellX := ctx.Area.X + cx
				cellY := ctx.Area.Y + cy
				if c := buf.Get(cellX, cellY); c != nil {
					c.Content = '▄'
					c.Style.Fg = fgColor
					c.Style.Bg = bgColor
					c.Style.Modifier = cell.ModifierReset
				}
			}
		}
		return
	}

	// Yerel grafik protokol modları: Metinlerin resmin arkasından taşmasını önlemek için alanı boşlukla temizle
	for y := ctx.Area.Y; y < ctx.Area.Y+ctx.Area.Height; y++ {
		for x := ctx.Area.X; x < ctx.Area.X+ctx.Area.Width; x++ {
			if c := buf.Get(x, y); c != nil {
				c.Content = ' '
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
