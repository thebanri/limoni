package cell

import "image"

// Context, alt bileşenlerin (widget) çizim yaparken kullandığı stack-allocated bağlam yapısıdır.
// Değer kopyalaması (pass-by-value) ile aktarıldığı için heap allocation oluşturmaz ve bellek dostudur.
// Bu sayede alt bileşenler, üst bileşenlerinin çizim sınırlarını ve stil kararlarını otomatik olarak devralır.
type Context struct {
	// Area, ilgili widget'ın çizim yapabileceği sınırlandırılmış alanı (koordinat ve boyut) belirtir.
	// Alt bileşenler bu alanın dışına çizim yapmamalıdır.
	Area Rect

	// Style, üst bileşenlerden devralınan (miras kalan) stil özelliklerini (renkler, kalınlık, eğiklik vb.) taşır.
	Style Style

	// RegisterClick, widget'ların çizim sırasında tıklanabilir bölgeler (click areas)
	// kaydetmesini sağlayan, terminal katmanı tarafından doldurulan callback köprüsüdür.
	RegisterClick func(area Rect, handler func())

	// RegisterImage, widget'ların çizim sırasında resim çizdirme taleplerini
	// kaydetmesini sağlayan, terminal katmanı tarafından doldurulan callback köprüsüdür.
	RegisterImage func(area Rect, img image.Image, zIndex int)

	// RegisterFocus, widget'ların çizim sırasında odaklanabilir (focusable) olduklarını
	// bildirmesini sağlayan, terminal odak yöneticisi tarafından doldurulan callback köprüsüdür.
	RegisterFocus func(id string)

	// SetFocus, widget'ların tıklandıklarında veya bir olay anında odağı kendi üzerlerine
	// almalarını sağlayan, odak yöneticisi tarafından doldurulan callback köprüsüdür.
	SetFocus func(id string)

	// FocusedID, aktif olarak odaklanmış olan bileşenin ID'sini taşır.
	FocusedID string
}

// NewContext yeni bir Context örneği oluşturup döndürür.
func NewContext(area Rect, style Style) Context {
	return Context{
		Area:  area,
		Style: style,
	}
}

// Merge, iki stili cascading (stil mirası) kurallarına göre birleştirir ve yeni bir Style yapısı döndürür.
// 'other' stildeki tanımlı renkler, mevcut (üst) stilin renklerini ezer.
// Eğer 'other' stildeki renkler varsayılan (ColorDefault) ise, üst stilden miras kalan renkler korunur.
// Modifikatörler (Bold, Italic vb.) bit düzeyinde OR (veya) işlemine tabi tutularak birleştirilir.
func (s Style) Merge(other Style) Style {
	merged := s

	// Eğer hedef stilde ön plan rengi belirtilmişse (varsayılan değilse), miras kalan rengi ez
	if other.Fg.Type() != ColorDefault {
		merged.Fg = other.Fg
	}

	// Eğer hedef stilde arka plan rengi belirtilmişse (varsayılan değilse), miras kalan rengi ez
	if other.Bg.Type() != ColorDefault {
		merged.Bg = other.Bg
	}

	// Modifikatörleri birleştir
	merged.Modifier |= other.Modifier

	return merged
}
