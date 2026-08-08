package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Kenarlık yön maskeleri (bitmask).
const (
	BorderNone   uint8 = 0
	BorderLeft   uint8 = 1 << 0
	BorderRight  uint8 = 1 << 1
	BorderTop    uint8 = 1 << 2
	BorderBottom uint8 = 1 << 3
	BorderAll    uint8 = BorderLeft | BorderRight | BorderTop | BorderBottom
)

// BorderSymbols, kenarlık çizgisinde kullanılacak olan glif (rune) grubunu tanımlar.
type BorderSymbols struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
}

var (
	// SymbolsSingle ince/standart kenarlık çizgileridir.
	SymbolsSingle = BorderSymbols{
		Horizontal:  '─',
		Vertical:    '│',
		TopLeft:     '┌',
		TopRight:    '┐',
		BottomLeft:  '└',
		BottomRight: '┘',
	}
	// SymbolsDouble çift çizgili kenarlıktır.
	SymbolsDouble = BorderSymbols{
		Horizontal:  '═',
		Vertical:    '║',
		TopLeft:     '╔',
		TopRight:    '╗',
		BottomLeft:  '╚',
		BottomRight: '╝',
	}
	// SymbolsThick kalın kenarlık çizgileridir.
	SymbolsThick = BorderSymbols{
		Horizontal:  '━',
		Vertical:    '┃',
		TopLeft:     '┏',
		TopRight:    '┓',
		BottomLeft:  '┗',
		BottomRight: '┛',
	}
	// SymbolsRounded köşeleri yuvarlatılmış ince kenarlıktır.
	SymbolsRounded = BorderSymbols{
		Horizontal:  '─',
		Vertical:    '│',
		TopLeft:     '╭',
		TopRight:    '╮',
		BottomLeft:  '╰',
		BottomRight: '╯',
	}
	// SymbolsBlock dolu blok elemanlı kalın kenarlıktır.
	SymbolsBlock = BorderSymbols{
		Horizontal:  '█',
		Vertical:    '█',
		TopLeft:     '█',
		TopRight:    '█',
		BottomLeft:  '█',
		BottomRight: '█',
	}
)

// Alignment, başlık veya metin hizalamasını belirten türdür.
type Alignment uint8

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// Block, terminal ekranında kenarlık çizebilen, arka plan dolgusu yapabilen
// ve üstüne başlık (Title) yerleştirebilen en temel kapsayıcı (container) widget'tır.
// Alt bileşenini (Child) otomatik olarak daraltılmış iç alana yönlendirir ve stil mirasını aktarır.
type Block struct {
	// Title, bloğun üst kenarında gösterilecek olan başlık metnidir.
	Title string
	// TitleAlignment, başlık metninin kenarlık üzerindeki hizasını belirler (Left, Center, Right).
	TitleAlignment Alignment
	// TitleStyle, başlık metninin rengini ve stil özelliklerini belirler.
	TitleStyle cell.Style

	// Borders, hangi kenarların çizileceğini belirleyen maske alanıdır (örn. BorderAll veya BorderTop|BorderBottom).
	Borders uint8
	// BorderSymbols, kenarlık çiziminde kullanılacak olan glif sembolleridir (örn. SymbolsRounded).
	BorderSymbols BorderSymbols
	// BorderStyle, kenarlık çizgilerinin rengini ve stilini belirler.
	BorderStyle cell.Style

	// İçerik ile kenarlık arasındaki boşluk hücre sayıları (padding).
	PaddingLeft   uint16
	PaddingRight  uint16
	PaddingTop    uint16
	PaddingBottom uint16

	// Style, bloğun arka plan dolgu rengini ve varsayılan genel stilini belirler.
	Style cell.Style

	// Child, bloğun içerisine çizilecek olan alt görsel bileşendir.
	Child Widget
}

// Draw, bloğu ve kenarlıklarını çizer, arka planını doldurur ve alt bileşenin (Child) çizimini tetikler.
func (b Block) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	// Bloğun nihai stilini belirle (Miras kalan stil ile bu bloğun stilini birleştir)
	blockStyle := ctx.Style.Merge(b.Style)
	borderStyle := blockStyle.Merge(b.BorderStyle)

	// 1. Aşama: Bloğun arka planını doldur
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			if c := buf.Get(x, y); c != nil {
				c.Content = ' '
				c.Style = blockStyle
			}
		}
	}

	// 2. Aşama: Kenarlıkları çiz
	hasL := (b.Borders & BorderLeft) != 0
	hasR := (b.Borders & BorderRight) != 0
	hasT := (b.Borders & BorderTop) != 0
	hasB := (b.Borders & BorderBottom) != 0

	sym := b.BorderSymbols
	// Eğer kenarlık sembolleri atanmadıysa varsayılan ince çizgiyi kullan
	if sym.Horizontal == 0 {
		sym = SymbolsSingle
	}

	// Yatay çizgileri çiz
	if hasT {
		for x := area.X + 1; x < area.X+area.Width-1; x++ {
			if c := buf.Get(x, area.Y); c != nil {
				c.Content = sym.Horizontal
				c.Style = borderStyle
			}
		}
	}
	if hasB {
		for x := area.X + 1; x < area.X+area.Width-1; x++ {
			if c := buf.Get(x, area.Y+area.Height-1); c != nil {
				c.Content = sym.Horizontal
				c.Style = borderStyle
			}
		}
	}

	// Dikey çizgileri çiz
	if hasL {
		for y := area.Y + 1; y < area.Y+area.Height-1; y++ {
			if c := buf.Get(area.X, y); c != nil {
				c.Content = sym.Vertical
				c.Style = borderStyle
			}
		}
	}
	if hasR {
		for y := area.Y + 1; y < area.Y+area.Height-1; y++ {
			if c := buf.Get(area.X+area.Width-1, y); c != nil {
				c.Content = sym.Vertical
				c.Style = borderStyle
			}
		}
	}

	// Köşe birleşimlerini çiz
	if hasT && hasL {
		if c := buf.Get(area.X, area.Y); c != nil {
			c.Content = sym.TopLeft
			c.Style = borderStyle
		}
	}
	if hasT && hasR {
		if c := buf.Get(area.X+area.Width-1, area.Y); c != nil {
			c.Content = sym.TopRight
			c.Style = borderStyle
		}
	}
	if hasB && hasL {
		if c := buf.Get(area.X, area.Y+area.Height-1); c != nil {
			c.Content = sym.BottomLeft
			c.Style = borderStyle
		}
	}
	if hasB && hasR {
		if c := buf.Get(area.X+area.Width-1, area.Y+area.Height-1); c != nil {
			c.Content = sym.BottomRight
			c.Style = borderStyle
		}
	}

	// 3. Aşama: Başlığı üst kenarlığa çiz
	if b.Title != "" && hasT && area.Width > 4 {
		titleStyle := blockStyle.Merge(b.TitleStyle)
		formattedTitle := " " + b.Title + " "
		titleLen := uint16(utf8.RuneCountInString(formattedTitle))

		// Başlığın sığabileceği maksimum genişlik
		maxTitleLen := area.Width - 4
		if titleLen > maxTitleLen {
			formattedTitle = formattedTitle[:maxTitleLen]
			titleLen = uint16(utf8.RuneCountInString(formattedTitle))
		}

		var titleX uint16
		switch b.TitleAlignment {
		case AlignLeft:
			titleX = area.X + 2
		case AlignCenter:
			titleX = area.X + (area.Width-titleLen)/2
		case AlignRight:
			titleX = area.X + area.Width - 2 - titleLen
		}

		buf.SetString(titleX, area.Y, formattedTitle, titleStyle)
	}

	// 4. Aşama: Alt bileşeni (Child) çiz
	if b.Child != nil {
		var offsetL, offsetR, offsetT, offsetB uint16
		if hasL {
			offsetL = 1
		}
		if hasR {
			offsetR = 1
		}
		if hasT {
			offsetT = 1
		}
		if hasB {
			offsetB = 1
		}

		// Kenarlık ve Padding paylarını ekle
		left := offsetL + b.PaddingLeft
		right := offsetR + b.PaddingRight
		top := offsetT + b.PaddingTop
		bottom := offsetB + b.PaddingBottom

		if area.Width > left+right && area.Height > top+bottom {
			childArea := cell.Rect{
				X:      area.X + left,
				Y:      area.Y + top,
				Width:  area.Width - left - right,
				Height: area.Height - top - bottom,
			}
			// Alt bileşene daraltılmış alan ve birleştirilmiş stil bağlamını aktar
			childCtx := cell.NewContext(childArea, blockStyle)
			childCtx.RegisterClick = ctx.RegisterClick
			childCtx.RegisterMouse = ctx.RegisterMouse
			childCtx.CaptureMouse = ctx.CaptureMouse
			childCtx.RegisterImage = ctx.RegisterImage
			childCtx.RegisterFocus = ctx.RegisterFocus
			childCtx.SetFocus = ctx.SetFocus
			childCtx.FocusedID = ctx.FocusedID
			b.Child.Draw(childCtx, buf)
		}
	}
}

// SizeHint, kenarlık ve dolgu paylarını üstüne koyarak bu bloğun kaplamak istediği en uygun alanı hesaplar.
func (b Block) SizeHint(maxArea cell.Rect) (width, height uint16) {
	var offsetL, offsetR, offsetT, offsetB uint16
	if (b.Borders & BorderLeft) != 0 {
		offsetL = 1
	}
	if (b.Borders & BorderRight) != 0 {
		offsetR = 1
	}
	if (b.Borders & BorderTop) != 0 {
		offsetT = 1
	}
	if (b.Borders & BorderBottom) != 0 {
		offsetB = 1
	}

	overheadW := offsetL + offsetR + b.PaddingLeft + b.PaddingRight
	overheadH := offsetT + offsetB + b.PaddingTop + b.PaddingBottom

	// Başlığın sığması için asgari genişlik sınırı
	titleLen := uint16(0)
	if b.Title != "" {
		titleLen = uint16(utf8.RuneCountInString(b.Title)) + 4 // " Başlık " + köşeler
	}

	if b.Child != nil {
		var childMaxW, childMaxH uint16
		if maxArea.Width > overheadW {
			childMaxW = maxArea.Width - overheadW
		}
		if maxArea.Height > overheadH {
			childMaxH = maxArea.Height - overheadH
		}

		// Alt bileşenin pazarlık boyutunu sorgula
		childW, childH := b.Child.SizeHint(cell.NewRect(maxArea.X, maxArea.Y, childMaxW, childMaxH))
		width = childW + overheadW
		height = childH + overheadH
	} else {
		width = overheadW
		height = overheadH
	}

	// Eğer başlık genişliği daha büyükse genişliği başlığa göre genişlet
	if width < titleLen {
		width = titleLen
	}

	// Sınırların dışına taşmayı engelle
	if width > maxArea.Width {
		width = maxArea.Width
	}
	if height > maxArea.Height {
		height = maxArea.Height
	}

	return width, height
}
