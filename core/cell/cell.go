package cell

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
	ModifierReset          Modifier = 0
	ModifierBold           Modifier = 1 << 0
	ModifierDim            Modifier = 1 << 1
	ModifierItalic         Modifier = 1 << 2
	ModifierUnderline      Modifier = 1 << 3
	ModifierBlink          Modifier = 1 << 4
	ModifierReverse        Modifier = 1 << 5
	ModifierHidden         Modifier = 1 << 6
	ModifierStrikethrough  Modifier = 1 << 7
	ModifierDoubleUnderline Modifier = 1 << 8
	ModifierUndercurl      Modifier = 1 << 9
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
		r == 0x00AD {  // Soft Hyphen
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

