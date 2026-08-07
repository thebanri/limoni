package graphics

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"os"
)

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

	// Sixel modern Linux terminallerinde (foot, alacritty vb.) yaygın bir standarttır.
	return ProtocolSixel
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

// ResizeImage, resmi nearest-neighbor (en yakın komşu) algoritmasıyla hedef piksel boyutuna ölçekler.
// Sıfır dış bağımlılık ve yüksek hız sunar.
func ResizeImage(img image.Image, w, h int) image.Image {
	if w <= 0 || h <= 0 {
		return img
	}
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

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
func EncodeKitty(img image.Image, cols, rows uint16, cellW, cellH uint16, imageID uint32, zIndex int) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)

	resized := ResizeImage(img, targetW, targetH)
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, resized); err != nil {
		return ""
	}
	pngBytes := pngBuf.Bytes()
	b64Data := base64.StdEncoding.EncodeToString(pngBytes)

	controlKeys := fmt.Sprintf("f=100,a=T,t=d,s=%d,v=%d,c=%d,r=%d,z=%d", targetW, targetH, cols, rows, zIndex)
	return chunkKittyPayload(controlKeys, b64Data)
}

// EncodeIterm2, resmi iTerm2 Inline Image Protocol formatında kodlar.
func EncodeIterm2(img image.Image, cols, rows uint16, cellW, cellH uint16) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)

	resized := ResizeImage(img, targetW, targetH)
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, resized); err != nil {
		return ""
	}
	pngBytes := pngBuf.Bytes()
	b64Data := base64.StdEncoding.EncodeToString(pngBytes)

	return fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;size=%d:%s\a", cols, rows, len(pngBytes), b64Data)
}

// EncodeSixel, resmi Sixel Graphics formatında kodlar.
func EncodeSixel(img image.Image, cols, rows uint16, cellW, cellH uint16) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)

	resized := ResizeImage(img, targetW, targetH)
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
	Img    image.Image
	Cols   uint16
	Rows   uint16
	CellW  uint16
	CellH  uint16
	Proto  Protocol
	ZIndex int
}

var escapeSequenceCache = make(map[ImageCacheKey]string)

// GetCachedEscapeSequence, önbellekten veya yeni nesil olarak resmin escape sequence çıktısını döner.
func GetCachedEscapeSequence(img image.Image, cols, rows uint16, cellW, cellH uint16, proto Protocol, zIndex int) string {
	key := ImageCacheKey{
		Img:    img,
		Cols:   cols,
		Rows:   rows,
		CellW:  cellW,
		CellH:  cellH,
		Proto:  proto,
		ZIndex: zIndex,
	}

	if seq, ok := escapeSequenceCache[key]; ok {
		return seq
	}

	var seq string
	switch proto {
	case ProtocolKitty:
		imageID := GetImageID(img)
		seq = EncodeKitty(img, cols, rows, cellW, cellH, imageID, zIndex)
	case ProtocolIterm2:
		seq = EncodeIterm2(img, cols, rows, cellW, cellH)
	case ProtocolSixel:
		seq = EncodeSixel(img, cols, rows, cellW, cellH)
	}

	escapeSequenceCache[key] = seq
	return seq
}
