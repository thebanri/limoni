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
	"sort"
	"strconv"
	"strings"
	"sync"
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

// Protocol represents the graphics protocol supported by the terminal.
type Protocol int

const (
	ProtocolAuto Protocol = iota
	ProtocolKitty
	ProtocolSixel
	ProtocolIterm2
	ProtocolHalfBlock
)

// DetectProtocol automatically selects the most suitable image protocol by examining terminal environment variables.
func DetectProtocol() Protocol {
	switch strings.ToLower(os.Getenv("LIMONI_GRAPHICS")) {
	case "kitty":
		return ProtocolKitty
	case "sixel":
		return ProtocolSixel
	case "iterm2":
		return ProtocolIterm2
	case "halfblock":
		return ProtocolHalfBlock
	}

	termProg := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch termProg {
	case "ghostty", "kitty", "wezterm", "rio":
		return ProtocolKitty
	case "iterm.app", "iterm":
		return ProtocolIterm2
	case "foot", "mlterm":
		return ProtocolSixel
	case "alacritty":
		return ProtocolHalfBlock
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("KITTY_PID") != "" || os.Getenv("KITTY_INSTALLATION_DIR") != "" {
		return ProtocolKitty
	}
	if os.Getenv("WEZTERM_PANE") != "" {
		return ProtocolKitty
	}
	if os.Getenv("GHOSTTY_BIN_DIR") != "" || os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return ProtocolKitty
	}

	term := strings.ToLower(os.Getenv("TERM"))
	if term == "xterm-kitty" || term == "xterm-ghostty" || term == "wezterm" {
		return ProtocolKitty
	}
	if strings.HasPrefix(term, "foot") || term == "mlterm" || strings.Contains(term, "sixel") {
		return ProtocolSixel
	}

	if os.Getenv("ALACRITTY_WINDOW_ID") != "" {
		return ProtocolHalfBlock
	}

	// Instead of corrupting the screen with escape sequences on unknown terminals,
	// use a safe cell-based fallback. Native protocols can be explicitly selected
	// via LIMONI_GRAPHICS or known terminal environments.
	return ProtocolHalfBlock
}

// GetImageID generates a unique 32-bit ID from image pixels using the FNV-1a hash algorithm.
func GetImageID(img image.Image) uint32 {
	if img == nil {
		return 0
	}
	h := fnv.New32a()
	if rgba, ok := img.(*image.RGBA); ok {
		h.Write(rgba.Pix)
		return h.Sum32()
	}
	if nrgba, ok := img.(*image.NRGBA); ok {
		h.Write(nrgba.Pix)
		return h.Sum32()
	}
	bounds := img.Bounds()
	var pixel [8]byte
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := rgbaAt(img, x, y)
			pixel[0] = byte(r)
			pixel[1] = byte(r >> 8)
			pixel[2] = byte(g)
			pixel[3] = byte(g >> 8)
			pixel[4] = byte(b)
			pixel[5] = byte(b >> 8)
			pixel[6] = byte(a)
			pixel[7] = byte(a >> 8)
			h.Write(pixel[:])
		}
	}
	return h.Sum32()
}

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
						r, g, b, a := rgbaAt(img, sx, sy)
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
		var srcY float64
		if h > 1 && srcH > 1 {
			srcY = float64(y) * float64(srcH-1) / float64(h-1)
		} else if srcH > 1 {
			srcY = float64(srcH-1) / 2.0
		}
		y0 := int(srcY)
		y1 := y0 + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		fy := srcY - float64(y0)

		for x := 0; x < w; x++ {
			var srcX float64
			if w > 1 && srcW > 1 {
				srcX = float64(x) * float64(srcW-1) / float64(w-1)
			} else if srcW > 1 {
				srcX = float64(srcW-1) / 2.0
			}
			x0 := int(srcX)
			x1 := x0 + 1
			if x1 >= srcW {
				x1 = srcW - 1
			}
			fx := srcX - float64(x0)

			r00, g00, b00, a00 := rgbaAt(img, srcBounds.Min.X+x0, srcBounds.Min.Y+y0)
			r10, g10, b10, a10 := rgbaAt(img, srcBounds.Min.X+x1, srcBounds.Min.Y+y0)
			r01, g01, b01, a01 := rgbaAt(img, srcBounds.Min.X+x0, srcBounds.Min.Y+y1)
			r11, g11, b11, a11 := rgbaAt(img, srcBounds.Min.X+x1, srcBounds.Min.Y+y1)

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

// ResizeImageContain fits the image into the target area while preserving its aspect ratio.
// The target canvas is full-sized; unused areas are filled with the top-left pixel
// of the source image. This prevents native protocols from stretching the image.
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

// buildPalette creates a dynamic color palette from image pixels with a maximum of maxColors.
func buildPalette(img image.Image, maxColors int) color.Palette {
	bounds := img.Bounds()
	var pal color.Palette
	colorMap := make(map[color.RGBA]bool)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := rgbaAt(img, x, y)
			c := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
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

// chunkKittyPayload encodes base64 data for the Kitty protocol by splitting it into 4096-byte chunks.
// Kitty terminal does not accept single chunks larger than 4096 bytes per the protocol.
func chunkKittyPayload(controlKeys string, b64Data string) string {
	chunkSize := 4096
	totalLen := len(b64Data)

	if totalLen <= chunkSize {
		return fmt.Sprintf("\x1b_G%s;%s\x1b\\", controlKeys, b64Data)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("\x1b_G%s,m=1;%s\x1b\\", controlKeys, b64Data[:chunkSize]))

	offset := chunkSize
	for offset+chunkSize < totalLen {
		buf.WriteString(fmt.Sprintf("\x1b_Gm=1;%s\x1b\\", b64Data[offset:offset+chunkSize]))
		offset += chunkSize
	}

	buf.WriteString(fmt.Sprintf("\x1b_Gm=0;%s\x1b\\", b64Data[offset:]))

	return buf.String()
}

// KittyCanvas is the size, in pixels, of the picture kitty is sent for img
// shown over cols×rows cells. It is the cells' size in pixels, unless img is
// smaller than that: then it is the cells' shape at img's own resolution,
// and kitty scales the picture up to the cells itself. Scaling it up here
// only made the PNG larger and slower to encode, and the picture no sharper.
func KittyCanvas(img image.Image, cols, rows, cellW, cellH uint16) (w, h int) {
	w, h = int(cols)*int(cellW), int(rows)*int(cellH)
	if img == nil || w == 0 || h == 0 {
		return w, h
	}
	if _, ok := img.(*image.Uniform); ok {
		return w, h
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx() > w || b.Dy() > h {
		return w, h
	}
	// The larger of the two ratios, so img fits whole at its own size.
	if b.Dx()*h >= b.Dy()*w {
		return b.Dx(), max(1, (h*b.Dx()+w-1)/w)
	}
	return max(1, (w*b.Dy()+h-1)/h), b.Dy()
}

// kittyEncoder favours speed: a moving picture is sent many times a second,
// and BestSpeed encodes one in about half the time for a few more bytes.
var kittyEncoder = png.Encoder{CompressionLevel: png.BestSpeed}

// kittyPNG is img as kitty is sent it — fitted to KittyCanvas, as PNG, in
// base64 — and the picture's size. It is "" when img cannot be encoded.
func kittyPNG(img image.Image, cols, rows, cellW, cellH uint16, transparent bool) (string, int, int) {
	w, h := KittyCanvas(img, cols, rows, cellW, cellH)
	fitted := img
	if b := img.Bounds(); b.Dx() != w || b.Dy() != h {
		fitted = ResizeImageContain(img, w, h, transparent)
	}
	var pngBuf bytes.Buffer
	if err := kittyEncoder.Encode(&pngBuf, fitted); err != nil {
		return "", 0, 0
	}
	return base64.StdEncoding.EncodeToString(pngBuf.Bytes()), w, h
}

// EncodeKitty encodes the image in the Kitty Graphics Protocol format.
func EncodeKitty(img image.Image, cols, rows uint16, cellW, cellH uint16, imageID uint32, zIndex int, transparent bool) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	b64Data, w, h := kittyPNG(img, cols, rows, cellW, cellH, transparent)
	if b64Data == "" {
		return ""
	}

	// C=1 keeps the cursor where it is. Without it kitty moves the cursor
	// below the picture, and a picture that reaches the last row scrolls the
	// whole screen up a line — under a diff that does not know it moved.
	controlKeys := fmt.Sprintf("q=2,f=100,a=T,t=d,C=1,i=%d,s=%d,v=%d,c=%d,r=%d,z=%d", imageID, w, h, cols, rows, zIndex)
	return chunkKittyPayload(controlKeys, b64Data)
}

// EncodeIterm2 encodes the image in the iTerm2 Inline Image Protocol format.
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

	// doNotMoveCursor, as kitty's C=1: a picture reaching the last row must
	// not scroll the screen (iTerm2 3.5; older versions ignore the key).
	return fmt.Sprintf("\x1b]1337;File=inline=1;doNotMoveCursor=1;width=%d;height=%d;size=%d:%s\a", cols, rows, len(pngBytes), b64Data)
}

// EncodeSixel encodes the image in the Sixel Graphics format.
//
// A picture of at most 256 colours keeps them exactly. One with more — any
// photograph, anything drawn with soft edges — gets the 256 most common
// colours at five bits a channel, and every other colour the nearest of
// them, looked up once for each of the 32768 five-bit colours it falls in.
// It used to search the palette for every distinct colour, which on a
// record with soft edges was most of a 30 ms encode; and the palette was the
// first 256 colours met from the top, which left the rest of the picture
// to be matched against the colours of its first rows.
func EncodeSixel(img image.Image, cols, rows uint16, cellW, cellH uint16, transparent bool) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)

	resized := ResizeImageContain(img, targetW, targetH, transparent)
	q := quantPool.Get().(*sixelQuant)
	defer quantPool.Put(q)
	q.build(resized, transparent)
	pal := q.pal

	var buf bytes.Buffer
	buf.Grow(64 << 10)
	num := make([]byte, 0, 16)
	writeInt := func(n int) {
		num = strconv.AppendInt(num[:0], int64(n), 10)
		buf.Write(num)
	}
	// Sixel initialization sequence
	buf.WriteString("\x1bPq\"1;1;")

	for idx, col := range pal {
		buf.WriteByte('#')
		writeInt(idx)
		buf.WriteString(";2;")
		writeInt(int(col.R) * 100 / 255)
		buf.WriteByte(';')
		writeInt(int(col.G) * 100 / 255)
		buf.WriteByte(';')
		writeInt(int(col.B) * 100 / 255)
	}

	width := resized.Bounds().Dx()
	height := resized.Bounds().Dy()
	minX, minY := resized.Bounds().Min.X, resized.Bounds().Min.Y

	bandIndices := make([][6]int16, width)
	colorsInBand := make([]bool, len(pal))

	// Sixel encodes in 6-pixel vertical bands
	for bandY := 0; bandY < height; bandY += 6 {
		for i := range colorsInBand {
			colorsInBand[i] = false
		}

		// Quantize only the current 6-line band once: O(6 * width)
		for x := 0; x < width; x++ {
			for dy := 0; dy < 6; dy++ {
				y := bandY + dy
				if y >= height {
					bandIndices[x][dy] = -1
					continue
				}
				c, a := q.at(resized, minX+x, minY+y)
				if transparent && a < 128 {
					bandIndices[x][dy] = -1 // Transparent pixel
					continue
				}
				if q.many {
					c = dither(c, x, y)
				}
				colIdx := q.index(c)
				bandIndices[x][dy] = colIdx
				colorsInBand[colIdx] = true
			}
		}

		for colorIdx := range pal {
			if !colorsInBand[colorIdx] {
				continue
			}
			buf.WriteByte('#')
			writeInt(colorIdx)

			targetIdx := int16(colorIdx)
			repeatCount := 0
			var lastChar byte

			flushRepeat := func() {
				if repeatCount > 3 {
					buf.WriteByte('!')
					writeInt(repeatCount)
					buf.WriteByte(lastChar)
				} else {
					for k := 0; k < repeatCount; k++ {
						buf.WriteByte(lastChar)
					}
				}
				repeatCount = 0
			}

			for x := 0; x < width; x++ {
				var mask byte
				for dy := 0; dy < 6; dy++ {
					if bandIndices[x][dy] == targetIdx {
						mask |= 1 << dy
					}
				}
				char := mask + 63
				if repeatCount > 0 && char != lastChar {
					flushRepeat()
				}
				lastChar = char
				repeatCount++
			}
			flushRepeat()

			// Carriage return
			buf.WriteByte('$')
		}
		// Move to the next band (newline)
		buf.WriteByte('-')
	}

	// Sixel exit sequence
	buf.WriteString("\x1b\\")
	return buf.String()
}

// sixelQuant is a sixel picture's palette and the way from a colour to
// its entry. Its tables are kept between pictures (quantPool): an animated
// picture is encoded many times a second.
type sixelQuant struct {
	pal    []color.RGBA
	exact  map[color.RGBA]int16 // the picture's own colours, when there are at most 256
	counts [1 << 15]int32       // how many pixels fall in each five-bit colour
	sums   [1 << 15][3]int64    // and the sum of their channels
	table  [1 << 15]int16       // a five-bit colour's entry; -1 until looked up
	used   []int32              // the five-bit colours seen, to clear after
	ready  bool                 // table has been set to -1
	many   bool                 // more colours than the palette holds: dithered
}

// bayer4 is a 4×4 ordered-dither matrix.
var bayer4 = [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}

// dither nudges c by up to half a five-bit step, by its place in a 4×4
// pattern, so the edge between two palette colours is a fine mix rather
// than a band. The pattern is fixed to the pixels, so an animated picture
// does not shimmer.
func dither(c color.RGBA, x, y int) color.RGBA {
	d := (bayer4[y&3][x&3]*2 - 15) / 4 // −3 … +3
	nudge := func(v uint8) uint8 { return uint8(max(0, min(255, int(v)+d))) }
	return color.RGBA{nudge(c.R), nudge(c.G), nudge(c.B), c.A}
}

var quantPool = sync.Pool{New: func() any {
	return &sixelQuant{exact: make(map[color.RGBA]int16, 257)}
}}

func key15(c color.RGBA) int32 {
	return int32(c.R>>3)<<10 | int32(c.G>>3)<<5 | int32(c.B>>3)
}

// at is the pixel at x, y, opaque-ish colour and alpha.
func (q *sixelQuant) at(img image.Image, x, y int) (color.RGBA, uint8) {
	if m, ok := img.(*image.RGBA); ok {
		i := m.PixOffset(x, y)
		p := m.Pix[i : i+4 : i+4]
		return color.RGBA{p[0], p[1], p[2], p[3]}, p[3]
	}
	r, g, b, a := rgbaAt(img, x, y)
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}, uint8(a >> 8)
}

// build chooses the palette for img.
func (q *sixelQuant) build(img image.Image, transparent bool) {
	for _, k := range q.used {
		q.counts[k], q.sums[k], q.table[k] = 0, [3]int64{}, -1
	}
	q.used = q.used[:0]
	clear(q.exact)
	q.pal = q.pal[:0]
	if !q.ready {
		for i := range q.table {
			q.table[i] = -1
		}
		q.ready = true
	}
	b := img.Bounds()
	many := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c, a := q.at(img, x, y)
			if transparent && a < 128 {
				continue
			}
			if !many {
				if _, ok := q.exact[c]; !ok {
					if len(q.pal) == 256 {
						many = true
					} else {
						q.exact[c] = int16(len(q.pal))
						q.pal = append(q.pal, c)
					}
				}
			}
			k := key15(c)
			if q.counts[k] == 0 {
				q.used = append(q.used, k)
			}
			q.counts[k]++
			q.sums[k][0] += int64(c.R)
			q.sums[k][1] += int64(c.G)
			q.sums[k][2] += int64(c.B)
		}
	}
	q.many = many
	if !many {
		if len(q.pal) == 0 {
			q.pal = append(q.pal, color.RGBA{A: 255})
		}
		return
	}
	// The 256 most common five-bit colours, each the average of its pixels.
	clear(q.exact)
	byCount := append([]int32(nil), q.used...)
	sort.Slice(byCount, func(i, j int) bool {
		if q.counts[byCount[i]] != q.counts[byCount[j]] {
			return q.counts[byCount[i]] > q.counts[byCount[j]]
		}
		return byCount[i] < byCount[j]
	})
	q.pal = q.pal[:0]
	for _, k := range byCount[:min(256, len(byCount))] {
		n := int64(q.counts[k])
		q.table[k] = int16(len(q.pal))
		q.pal = append(q.pal, color.RGBA{uint8(q.sums[k][0] / n), uint8(q.sums[k][1] / n), uint8(q.sums[k][2] / n), 255})
	}
}

// index is the palette entry for c.
func (q *sixelQuant) index(c color.RGBA) int16 {
	if len(q.exact) > 0 {
		if i, ok := q.exact[c]; ok {
			return i
		}
	}
	k := key15(c)
	if i := q.table[k]; i >= 0 {
		return i
	}
	// The nearest entry to this five-bit colour's average, looked up once.
	var ar, ag, ab int64
	if n := int64(q.counts[k]); n > 0 {
		ar, ag, ab = q.sums[k][0]/n, q.sums[k][1]/n, q.sums[k][2]/n
	} else {
		ar, ag, ab = int64(c.R), int64(c.G), int64(c.B)
	}
	best, bestD := int16(0), int64(1<<62)
	for i, p := range q.pal {
		dr, dg, db := ar-int64(p.R), ag-int64(p.G), ab-int64(p.B)
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = int16(i), d
		}
	}
	if q.counts[k] == 0 {
		q.used = append(q.used, k) // so build clears it
	}
	q.table[k] = best
	return best
}

// ImageCacheKey serves as a unique key for the image escape sequence cache.
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

var (
	escapeSequenceCache = make(map[ImageCacheKey]string)
	escapeCacheMu       sync.RWMutex
)

// ForgetImage drops everything cached for img: its escape sequences, and the
// flattened and faded copies Image.Draw made of it. The caches are keyed by
// the image value, so an image whose pixels are rewritten in place — a frame
// buffer reused for animation — must be forgotten before it is shown again,
// or the terminal is sent the old picture. Forgetting an image that is no
// longer shown also releases the memory the caches held for it.
func ForgetImage(img image.Image) {
	escapeCacheMu.Lock()
	for key := range escapeSequenceCache {
		if key.Img == img {
			delete(escapeSequenceCache, key)
		}
	}
	escapeCacheMu.Unlock()
	forgetDerived(img)
	generation.Add(1)
}

// GetCachedEscapeSequence returns the cached escape sequence of the image or generates a new one.

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

	escapeCacheMu.RLock()
	if seq, ok := escapeSequenceCache[key]; ok {
		escapeCacheMu.RUnlock()
		return seq
	}
	escapeCacheMu.RUnlock()

	// A Clip is sent as the pixels of its part; only kitty does better, and
	// the terminal handles that itself.
	if c, ok := img.(*Clip); ok {
		img = c.Image
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

	escapeCacheMu.Lock()
	if len(escapeSequenceCache) > 256 {
		clear(escapeSequenceCache)
	}
	escapeSequenceCache[key] = seq
	escapeCacheMu.Unlock()
	return seq
}

// rgbaAt returns the 16-bit RGBA of one pixel. image.Image.At boxes its
// color.Color, a heap allocation per pixel read; the formats images actually
// come in are read directly instead. Resizing a picture for an image protocol
// reads four pixels per output pixel, which made it 180,000 allocations for a
// 480×384 frame.
func rgbaAt(img image.Image, x, y int) (r, g, b, a uint32) {
	switch m := img.(type) {
	case *image.RGBA:
		return m.RGBAAt(x, y).RGBA()
	case *image.NRGBA:
		return m.NRGBAAt(x, y).RGBA()
	case *image.YCbCr:
		return m.YCbCrAt(x, y).RGBA()
	case *image.Gray:
		return m.GrayAt(x, y).RGBA()
	}
	return img.At(x, y).RGBA()
}
