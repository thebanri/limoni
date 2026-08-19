package graphics

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// CropImage returns a pixel-exact crop of img. The returned image uses a
// zero-based RGBA canvas so it can be safely passed to native image encoders.
func CropImage(img image.Image, crop image.Rectangle) image.Image {
	if img == nil {
		return nil
	}
	crop = crop.Intersect(img.Bounds())
	if crop.Empty() {
		return nil
	}
	out := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(out, out.Bounds(), img, crop.Min, draw.Src)
	return out
}

// Protocol, terminalin desteklediği grafik protokol türünü temsil eder.
type Protocol int

const (
	ProtocolAuto Protocol = iota
	ProtocolKitty
	ProtocolSixel
	ProtocolIterm2
	ProtocolHalfBlock
)

// transferredKittyImages, Kitty protokolüyle terminal belleğine zaten aktarılmış olan
// resim ID'lerini saklar. Bu sayede aynı resmi her karede tekrar göndermek yerine
// sadece konumlandırma komutu gönderilir (performans optimizasyonu).
var transferredKittyImages = make(map[uint32]bool)

// DetectProtocol, terminal ortam değişkenlerini inceleyerek en uygun resim protokolünü otomatik seçer.
func DetectProtocol() Protocol {
	switch os.Getenv("LIMONI_GRAPHICS") {
	case "kitty":
		return ProtocolKitty
	case "sixel":
		return ProtocolSixel
	case "iterm2":
		return ProtocolIterm2
	case "halfblock":
		return ProtocolHalfBlock
	}
	termProg := os.Getenv("TERM_PROGRAM")
	switch termProg {
	case "Ghostty", "kitty", "WezTerm":
		return ProtocolKitty
	case "iTerm.app":
		return ProtocolIterm2
	case "Alacritty":
		return ProtocolHalfBlock
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("WEZTERM_PANE") != "" || os.Getenv("GHOSTTY_BIN_DIR") != "" {
		return ProtocolKitty
	}

	if os.Getenv("ALACRITTY_WINDOW_ID") != "" {
		return ProtocolHalfBlock
	}

	term := os.Getenv("TERM")
	if term == "xterm-kitty" {
		return ProtocolKitty
	}

	// Bilinmeyen terminallerde escape sequence basıp ekranı bozmak yerine
	// güvenli hücre tabanlı fallback kullanılır. Native protocol açıkça
	// LIMONI_GRAPHICS veya bilinen terminal env ile seçilebilir.
	return ProtocolHalfBlock
}

// GetImageID, resim piksellerinden FNV-1a hash algoritmasıyla 32-bit benzersiz bir ID üretir.
func GetImageID(img image.Image) uint32 {
	if img == nil {
		return 0
	}
	h := fnv.New32a()
	bounds := img.Bounds()
	// Performans için hızlıca tüm pikselleri hash'le
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			h.Write([]byte{
				byte(r), byte(r >> 8),
				byte(g), byte(g >> 8),
				byte(b), byte(b >> 8),
				byte(a), byte(a >> 8),
			})
		}
	}
	return h.Sum32()
}

// ResizeImage scales an image to w x h using area-averaging (box filtering) for downscaling
// ResizeImage scales an image to w x h using area-averaging (box filtering) for downscaling
// and bilinear interpolation for upscaling, producing crisp, anti-aliased images with zero external dependencies.
func ResizeImage(img image.Image, w, h int) image.Image {
	if img == nil || w <= 0 || h <= 0 {
		return img
	}

	if uniform, ok := img.(*image.Uniform); ok {
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		c := color.RGBAModel.Convert(uniform.C).(color.RGBA)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetRGBA(x, y, c)
			}
		}
		return dst
	}

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return img
	}

	if srcW > 4096 || srcH > 4096 {
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				srcX := int(float64(x) / float64(w) * float64(srcW))
				srcY := int(float64(y) / float64(h) * float64(srcH))
				dst.Set(x, y, img.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
			}
		}
		return dst
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// Downscaling: Area-averaging box filter
	if w <= srcW && h <= srcH {
		for y := 0; y < h; y++ {
			srcY0 := srcBounds.Min.Y + int(float64(y)*float64(srcH)/float64(h))
			srcY1 := srcBounds.Min.Y + int(float64(y+1)*float64(srcH)/float64(h))
			if srcY1 <= srcY0 {
				srcY1 = srcY0 + 1
			}
			if srcY1 > srcBounds.Max.Y {
				srcY1 = srcBounds.Max.Y
			}

			for x := 0; x < w; x++ {
				srcX0 := srcBounds.Min.X + int(float64(x)*float64(srcW)/float64(w))
				srcX1 := srcBounds.Min.X + int(float64(x+1)*float64(srcW)/float64(w))
				if srcX1 <= srcX0 {
					srcX1 = srcX0 + 1
				}
				if srcX1 > srcBounds.Max.X {
					srcX1 = srcBounds.Max.X
				}

				var totalR, totalG, totalB, totalA uint64
				var count uint64

				for sy := srcY0; sy < srcY1; sy++ {
					for sx := srcX0; sx < srcX1; sx++ {
						r, g, b, a := img.At(sx, sy).RGBA()
						totalR += uint64(r)
						totalG += uint64(g)
						totalB += uint64(b)
						totalA += uint64(a)
						count++
					}
				}

				if count > 0 {
					avgR := uint8((totalR / count) >> 8)
					avgG := uint8((totalG / count) >> 8)
					avgB := uint8((totalB / count) >> 8)
					avgA := uint8((totalA / count) >> 8)
					dst.SetRGBA(x, y, color.RGBA{R: avgR, G: avgG, B: avgB, A: avgA})
				}
			}
		}
		return dst
	}

	// Upscaling: Bilinear interpolation
	for y := 0; y < h; y++ {
		srcY := float64(y) * float64(srcH-1) / float64(h)
		y0 := int(srcY)
		y1 := y0 + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		fy := srcY - float64(y0)

		for x := 0; x < w; x++ {
			srcX := float64(x) * float64(srcW-1) / float64(w)
			x0 := int(srcX)
			x1 := x0 + 1
			if x1 >= srcW {
				x1 = srcW - 1
			}
			fx := srcX - float64(x0)

			r00, g00, b00, a00 := img.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y0).RGBA()
			r10, g10, b10, a10 := img.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y0).RGBA()
			r01, g01, b01, a01 := img.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y1).RGBA()
			r11, g11, b11, a11 := img.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y1).RGBA()

			topR := float64(r00)*(1-fx) + float64(r10)*fx
			topG := float64(g00)*(1-fx) + float64(g10)*fx
			topB := float64(b00)*(1-fx) + float64(b10)*fx
			topA := float64(a00)*(1-fx) + float64(a10)*fx

			botR := float64(r01)*(1-fx) + float64(r11)*fx
			botG := float64(g01)*(1-fx) + float64(g11)*fx
			botB := float64(b01)*(1-fx) + float64(b11)*fx
			botA := float64(a01)*(1-fx) + float64(a11)*fx

			finR := uint8(int(topR*(1-fy)+botR*fy) >> 8)
			finG := uint8(int(topG*(1-fy)+botG*fy) >> 8)
			finB := uint8(int(topB*(1-fy)+botB*fy) >> 8)
			finA := uint8(int(topA*(1-fy)+botA*fy) >> 8)

			dst.SetRGBA(x, y, color.RGBA{R: finR, G: finG, B: finB, A: finA})
		}
	}
	return dst
}

// ResizeImageContain resmi aspect ratio'sunu koruyarak hedef alana sığdırır.
// Hedef canvas tam boyuttadır; kullanılmayan alan kaynak görselin sol üst
// pikseliyle doldurulur. Böylece native protokoller görseli esnetmez.
func ResizeImageContain(img image.Image, w, h int, transparent bool) image.Image {
	if img == nil || w <= 0 || h <= 0 {
		return img
	}

	if uniform, ok := img.(*image.Uniform); ok {
		return ResizeImage(uniform, w, h)
	}

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return img
	}

	fitW, fitH := w, h
	if float64(srcW)*float64(h) > float64(srcH)*float64(w) {
		fitH = int(float64(w) * float64(srcH) / float64(srcW))
	} else {
		fitW = int(float64(h) * float64(srcW) / float64(srcH))
	}
	if fitW < 1 {
		fitW = 1
	}
	if fitH < 1 {
		fitH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	var background color.Color = color.RGBA{0, 0, 0, 0}
	if !transparent {
		background = img.At(bounds.Min.X, bounds.Min.Y)
	}
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	resized := ResizeImage(img, fitW, fitH)
	offset := image.Pt((w-fitW)/2, (h-fitH)/2)
	draw.Draw(dst, image.Rectangle{Min: offset, Max: offset.Add(image.Pt(fitW, fitH))}, resized, image.Point{}, draw.Over)
	return dst
}

// buildPalette, resimdeki piksellerden maksimum maxColors boyutunda dinamik bir renk paleti oluşturur.
func buildPalette(img image.Image, maxColors int) color.Palette {
	bounds := img.Bounds()
	var pal color.Palette
	colorMap := make(map[color.RGBA]bool)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if !colorMap[c] {
				if len(pal) < maxColors {
					pal = append(pal, c)
					colorMap[c] = true
				}
			}
		}
	}
	return pal
}

// chunkKittyPayload, Kitty protokolü için base64 verisini 4096 byte'lık parçalara ayırarak kodlar.
// Kitty terminali 4096 byte'tan büyük tekil parçaları protokol gereği kabul etmemektedir.
func chunkKittyPayload(controlKeys string, b64Data string) string {
	chunkSize := 4096
	totalLen := len(b64Data)

	if totalLen <= chunkSize {
		return fmt.Sprintf("\x1b_G%s;%s\x1b\\", controlKeys, b64Data)
	}

	var buf bytes.Buffer
	// İlk parça (more chunks: m=1)
	buf.WriteString(fmt.Sprintf("\x1b_G%s,m=1;%s\x1b\\", controlKeys, b64Data[:chunkSize]))

	// Orta parçalar
	offset := chunkSize
	for offset+chunkSize < totalLen {
		buf.WriteString(fmt.Sprintf("\x1b_Gm=1;%s\x1b\\", b64Data[offset:offset+chunkSize]))
		offset += chunkSize
	}

	// Son parça (more chunks: m=0)
	buf.WriteString(fmt.Sprintf("\x1b_Gm=0;%s\x1b\\", b64Data[offset:]))

	return buf.String()
}

// EncodeKitty, resmi Kitty Graphics Protocol formatında kodlar.
func EncodeKitty(img image.Image, cols, rows uint16, cellW, cellH uint16, imageID uint32, zIndex int, transparent bool) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)

	resized := ResizeImageContain(img, targetW, targetH, transparent)
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, resized); err != nil {
		return ""
	}
	pngBytes := pngBuf.Bytes()
	b64Data := base64.StdEncoding.EncodeToString(pngBytes)

	controlKeys := fmt.Sprintf("f=100,a=T,t=d,i=%d,s=%d,v=%d,c=%d,r=%d,z=%d", imageID, targetW, targetH, cols, rows, zIndex)
	return chunkKittyPayload(controlKeys, b64Data)
}

// EncodeIterm2, resmi iTerm2 Inline Image Protocol formatında kodlar.
func EncodeIterm2(img image.Image, cols, rows uint16, cellW, cellH uint16, transparent bool) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)

	resized := ResizeImageContain(img, targetW, targetH, transparent)
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, resized); err != nil {
		return ""
	}
	pngBytes := pngBuf.Bytes()
	b64Data := base64.StdEncoding.EncodeToString(pngBytes)

	return fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;size=%d:%s\a", cols, rows, len(pngBytes), b64Data)
}

// EncodeSixel, resmi Sixel Graphics formatında kodlar.
func EncodeSixel(img image.Image, cols, rows uint16, cellW, cellH uint16, transparent bool) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)

	resized := ResizeImageContain(img, targetW, targetH, transparent)
	pal := buildPalette(resized, 256)

	var buf bytes.Buffer
	// Sixel Giriş ANSI kodu
	buf.WriteString("\x1bPq\"1;1;")

	// Renk tablosunu (Palette) tanımla
	for idx, col := range pal {
		r, g, b, _ := col.RGBA()
		pctR := int(r * 100 / 65535)
		pctG := int(g * 100 / 65535)
		pctB := int(b * 100 / 65535)
		buf.WriteString(fmt.Sprintf("#%d;2;%d;%d;%d", idx, pctR, pctG, pctB))
	}

	width := resized.Bounds().Dx()
	height := resized.Bounds().Dy()

	// Sixel 6 piksellik dikey bantlar halinde kodlama yapar
	for bandY := 0; bandY < height; bandY += 6 {
		for colorIdx, targetColor := range pal {
			// Renk bu bantta var mı kontrol et (gereksiz I/O'yu engeller)
			hasColor := false
			for x := 0; x < width; x++ {
				for dy := 0; dy < 6; dy++ {
					y := bandY + dy
					if y < height {
						c := pal.Convert(resized.At(x, y))
						if c == targetColor {
							hasColor = true
							break
						}
					}
				}
				if hasColor {
					break
				}
			}

			if !hasColor {
				continue
			}

			// Aktif rengi seç
			buf.WriteString(fmt.Sprintf("#%d", colorIdx))

			// Tekrar sıkıştırmasıyla (Repeat Compression) Sixel karakterlerini yaz
			repeatCount := 0
			var lastChar byte = 0

			flushRepeat := func() {
				if repeatCount > 0 {
					if repeatCount > 3 {
						buf.WriteString(fmt.Sprintf("!%d%c", repeatCount, lastChar))
					} else {
						for k := 0; k < repeatCount; k++ {
							buf.WriteByte(lastChar)
						}
					}
					repeatCount = 0
				}
			}

			for x := 0; x < width; x++ {
				var mask byte = 0
				for dy := 0; dy < 6; dy++ {
					y := bandY + dy
					if y < height {
						c := pal.Convert(resized.At(x, y))
						if c == targetColor {
							mask |= 1 << dy
						}
					}
				}

				char := mask + 63
				if repeatCount == 0 {
					lastChar = char
					repeatCount = 1
				} else if char == lastChar {
					repeatCount++
				} else {
					flushRepeat()
					lastChar = char
					repeatCount = 1
				}
			}
			flushRepeat()

			// Satır başına dön (taşıyıcı dönüşü)
			buf.WriteByte('$')
		}
		// Sonraki banda geç (yeni satır)
		buf.WriteByte('-')
	}

	// Sixel Çıkış ANSI kodu
	buf.WriteString("\x1b\\")
	return buf.String()
}

// ImageCacheKey, resim escape sequence önbelleği için benzersiz bir anahtar görevi görür.
type ImageCacheKey struct {
	Img         image.Image
	Cols        uint16
	Rows        uint16
	CellW       uint16
	CellH       uint16
	Proto       Protocol
	ZIndex      int
	Transparent bool
}

var escapeSequenceCache = make(map[ImageCacheKey]string)

// GetCachedEscapeSequence, önbellekten veya yeni nesil olarak resmin escape sequence çıktısını döner.
func GetCachedEscapeSequence(img image.Image, cols, rows uint16, cellW, cellH uint16, proto Protocol, zIndex int, transparent bool) string {
	key := ImageCacheKey{
		Img:         img,
		Cols:        cols,
		Rows:        rows,
		CellW:       cellW,
		CellH:       cellH,
		Proto:       proto,
		ZIndex:      zIndex,
		Transparent: transparent,
	}

	if seq, ok := escapeSequenceCache[key]; ok {
		return seq
	}

	var seq string
	switch proto {
	case ProtocolKitty:
		imageID := GetImageID(img)
		seq = EncodeKitty(img, cols, rows, cellW, cellH, imageID, zIndex, transparent)
	case ProtocolIterm2:
		seq = EncodeIterm2(img, cols, rows, cellW, cellH, transparent)
	case ProtocolSixel:
		seq = EncodeSixel(img, cols, rows, cellW, cellH, transparent)
	}

	escapeSequenceCache[key] = seq
	return seq
}
