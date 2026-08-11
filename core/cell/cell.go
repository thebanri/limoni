package cell

import "unicode/utf8"

// ColorType temsilcisi, terminal renk türünü tanımlar.
type ColorType uint8

const (
	// ColorDefault terminalin varsayılan rengine sıfırlar.
	ColorDefault ColorType = iota
	// ColorANSI 8-bitlik standart ANSI renk kodunu temsil eder (0-255).
	ColorANSI
	// ColorRGB 24-bitlik gerçek renk (TrueColor) değerini temsil eder.
	ColorRGB
)

// Color terminal rengini temsil eden, bellek dostu bir uint32 yapısıdır.
// İlk byte (bit 24-31) ColorType'ı, kalan 3 byte ise rengin değerini (RGB veya ANSI) tutar.
type Color uint32

// NewColorDefault varsayılan terminal rengini döndürür.
func NewColorDefault() Color {
	return Color(ColorDefault) << 24
}

// NewColorANSI 8-bit ANSI rengi (0-255) oluşturur.
func NewColorANSI(code uint8) Color {
	return (Color(ColorANSI) << 24) | Color(code)
}

// NewColorRGB 24-bit TrueColor RGB rengi oluşturur.
func NewColorRGB(r, g, b uint8) Color {
	return (Color(ColorRGB) << 24) | (Color(r) << 16) | (Color(g) << 8) | Color(b)
}

// Type rengin türünü (Default, ANSI, RGB) döndürür.
func (c Color) Type() ColorType {
	return ColorType(c >> 24)
}

// ANSI rengin ANSI kodunu döndürür. Sadece Type() == ColorANSI durumunda geçerlidir.
func (c Color) ANSI() uint8 {
	return uint8(c & 0xFF)
}

// RGB rengin R, G, B değerlerini döndürür. Sadece Type() == ColorRGB durumunda geçerlidir.
func (c Color) RGB() (r, g, b uint8) {
	r = uint8((c >> 16) & 0xFF)
	g = uint8((c >> 8) & 0xFF)
	b = uint8(c & 0xFF)
	return
}

// Modifier terminal stili modifikatörlerini temsil eden bit maskesidir.
type Modifier uint16

const (
	ModifierReset           Modifier = 0
	ModifierBold            Modifier = 1 << 0
	ModifierDim             Modifier = 1 << 1
	ModifierItalic          Modifier = 1 << 2
	ModifierUnderline       Modifier = 1 << 3
	ModifierBlink           Modifier = 1 << 4
	ModifierReverse         Modifier = 1 << 5
	ModifierHidden          Modifier = 1 << 6
	ModifierStrikethrough   Modifier = 1 << 7
	ModifierDoubleUnderline Modifier = 1 << 8
	ModifierUndercurl       Modifier = 1 << 9
)

// Style terminal hücresinin stilini ve rengini tanımlar.
// Bellek Hizalaması: 4 (Fg) + 4 (Bg) + 2 (Modifier) = 10 byte.
// Go derleyicisi bunu 12 byte sınırına hizalar.
type Style struct {
	Fg       Color    // 4 byte
	Bg       Color    // 4 byte
	Modifier Modifier // 2 byte
}

// Reset stili varsayılan ayarlara getirir.
func (s *Style) Reset() {
	s.Fg = NewColorDefault()
	s.Bg = NewColorDefault()
	s.Modifier = ModifierReset
}

// AddModifier stile yeni bir özellik ekler (akıcı API/fluet API için değer döndürür).
func (s Style) AddModifier(m Modifier) Style {
	s.Modifier |= m
	return s
}

// RemoveModifier stilden bir özelliği kaldırır.
func (s Style) RemoveModifier(m Modifier) Style {
	s.Modifier &= ^m
	return s
}

// HasModifier stilde ilgili özelliğin olup olmadığını kontrol eder.
func (s Style) HasModifier(m Modifier) bool {
	return (s.Modifier & m) != 0
}

// Cell terminal ızgarasındaki tek bir hücreyi temsil eder.
// Bellek Hizalaması: 4 (Content) + 12 (Style) = 16 byte.
// 64-bit mimaride tam bir kelime sınırına (word boundary) hizalandığı için L2 cache dostudur.
type Cell struct {
	Content rune  // 4 byte (UTF-8 karakterler için)
	Style   Style // 12 byte (Go hizalaması dahil)
}

// Reset hücreyi varsayılan durumuna getirir (boşluk karakteri, varsayılan stil).
func (c *Cell) Reset() {
	c.Content = ' '
	c.Style.Reset()
}

// RuneContinuation, 2 sütun genişliğindeki karakterlerin ikinci yarısını işaretlemek için kullanılan özel Unicode değeridir.
const RuneContinuation rune = 0xFFFE

// RuneImage, yerel Sixel/Kitty resimleri tarafından kaplanan hücreleri işaretlemek için kullanılan özel Unicode değeridir.
const RuneImage rune = 0xFFFF

// RuneWidth, verilen karakterin terminalde kaç sütun genişliğinde çizileceğini hesaplar.
func RuneWidth(r rune) int {
	// Zero-width / combining characters
	if (r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
		(r >= 0x1F00 && r <= 0x1F1F) || // Combining diacritical marks
		(r >= 0x0300 && r <= 0x036F) || // Combining Diacritical Marks
		r == 0x200D || // Zero Width Joiner
		r == 0x200B || // Zero Width Space
		r == 0x200C || // Zero Width Non-Joiner
		r == 0x00AD { // Soft Hyphen
		return 0
	}
	if r >= 0x1F000 && r <= 0x1FFFF {
		return 2
	}
	// Yaygın emojiler ve CJK (Uzak Doğu) karakter grupları
	if (r >= 0x2E80 && r <= 0x9FFF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFF00 && r <= 0xFFEF) {
		return 2
	}
	return 1
}

// StringWidth returns the terminal-cell width of UTF-8 text.
func StringWidth(text string) int {
	width := 0
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		width += RuneWidth(r)
		text = text[size:]
	}
	return width
}

// Downsample maps the color to a compatible representation based on the capabilities.
func (c Color) Downsample(trueColor, colors256 bool) Color {
	t := c.Type()
	if t == ColorDefault {
		return c
	}

	if trueColor {
		// All colors (RGB, ANSI) are supported directly
		return c
	}

	if colors256 {
		// Terminal supports 256 colors but not TrueColor.
		if t == ColorANSI {
			return c
		}
		// Convert RGB to 256-color ANSI index
		r, g, b := c.RGB()
		return NewColorANSI(RGBToANSI256(r, g, b))
	}

	// Terminal only supports 16 colors (standard/bright ANSI)
	if t == ColorANSI {
		// Map 256-color ANSI to 16-color ANSI
		return NewColorANSI(ANSI256To16(c.ANSI()))
	}

	// Convert RGB to 16-color ANSI
	r, g, b := c.RGB()
	return NewColorANSI(RGBToANSI16(r, g, b))
}

// Downsample downsamples Fg and Bg colors in the style.
func (s Style) Downsample(trueColor, colors256 bool) Style {
	s.Fg = s.Fg.Downsample(trueColor, colors256)
	s.Bg = s.Bg.Downsample(trueColor, colors256)
	return s
}

// RGBToANSI256 maps an RGB color to the closest 256-color ANSI index.
func RGBToANSI256(r, g, b uint8) uint8 {
	// 1. Check if it's grayscale
	const grayThreshold = 8
	absDiff := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}

	if absDiff(r, g) <= grayThreshold && absDiff(g, b) <= grayThreshold && absDiff(r, b) <= grayThreshold {
		// Grayscale ramp from 232 to 255. Grayscale values are 8 + 10*i (8, 18, 28, ... 238)
		avg := (int(r) + int(g) + int(b)) / 3
		if avg < 8 {
			return 16 // Cube black
		}
		if avg > 238 {
			return 231 // Cube white
		}
		grayIdx := (avg - 8) / 10
		return uint8(232 + grayIdx)
	}

	// 2. Otherwise map to the 6x6x6 color cube: 16 + 36*cr + 6*cg + cb
	componentToCube := func(c uint8) int {
		if c < 48 {
			return 0
		}
		if c < 115 {
			return 1
		}
		if c < 155 {
			return 2
		}
		if c < 195 {
			return 3
		}
		if c < 235 {
			return 4
		}
		return 5
	}

	cr := componentToCube(r)
	cg := componentToCube(g)
	cb := componentToCube(b)
	return uint8(16 + 36*cr + 6*cg + cb)
}

var ansi16Colors = []struct {
	r, g, b uint8
	ansi    uint8
}{
	{0, 0, 0, 0},        // Black
	{128, 0, 0, 1},      // Red
	{0, 128, 0, 2},      // Green
	{128, 128, 0, 3},    // Yellow
	{0, 0, 128, 4},      // Blue
	{128, 0, 128, 5},    // Magenta
	{0, 128, 128, 6},    // Cyan
	{192, 192, 192, 7},  // White
	{128, 128, 128, 8},  // Bright Black
	{255, 0, 0, 9},      // Bright Red
	{0, 255, 0, 10},     // Bright Green
	{255, 255, 0, 11},   // Bright Yellow
	{0, 0, 255, 12},     // Bright Blue
	{255, 0, 255, 13},   // Bright Magenta
	{0, 255, 255, 14},   // Bright Cyan
	{255, 255, 255, 15}, // Bright White
}

// RGBToANSI16 maps an RGB color to the closest 16-color ANSI index.
func RGBToANSI16(r, g, b uint8) uint8 {
	minDist := int64(1 << 30)
	var bestAnsi uint8
	for _, c := range ansi16Colors {
		dr := int64(r) - int64(c.r)
		dg := int64(g) - int64(c.g)
		db := int64(b) - int64(c.b)
		dist := dr*dr + dg*dg + db*db
		if dist < minDist {
			minDist = dist
			bestAnsi = c.ansi
		}
	}
	return bestAnsi
}

// ANSI256To16 maps a 256-color ANSI index to the closest 16-color ANSI index.
func ANSI256To16(ansi uint8) uint8 {
	if ansi < 16 {
		return ansi
	}

	var r, g, b uint8
	if ansi >= 232 {
		gray := 8 + 10*(ansi-232)
		r, g, b = gray, gray, gray
	} else {
		cubeVal := []uint8{0, 95, 135, 175, 215, 255}
		idx := ansi - 16
		cb := idx % 6
		idx /= 6
		cg := idx % 6
		cr := idx / 6
		r = cubeVal[cr]
		g = cubeVal[cg]
		b = cubeVal[cb]
	}

	return RGBToANSI16(r, g, b)
}
