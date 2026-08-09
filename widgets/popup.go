package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// PopupItem, açılır menüdeki bir öğeyi temsil eder.
type PopupItem struct {
	// Text, menü öğesinin gösterilecek metnidir.
	Text string
	// Disabled, bu öğenin seçilemez (gri) durumda olup olmadığını belirtir.
	Disabled bool
	// Handler, bu öğe seçildiğinde çalıştırılacak callback fonksiyonudur.
	Handler func()
}

// PopupState, açılır menünün açılıp kapanma durumunu ve seçim indeksini yönetir.
type PopupState struct {
	// IsOpen, menünün açık olup olmadığını belirtir.
	IsOpen bool
	// Selected, mevcut fare sobre kalan (hover) veya klavye ile seçili öğe indeksi.
	Selected int
}

// NewPopupState, yeni bir PopupState nesnesi oluşturur.
func NewPopupState() *PopupState {
	return &PopupState{
		IsOpen:   false,
		Selected: -1,
	}
}

// Open, menüyü açar.
func (ps *PopupState) Open() {
	ps.IsOpen = true
	ps.Selected = -1
}

// Close, menüyü kapatır.
func (ps *PopupState) Close() {
	ps.IsOpen = false
	ps.Selected = -1
}

// Toggle, menünün açık/kapalı durumunu değiştirir.
func (ps *PopupState) Toggle() {
	if ps.IsOpen {
		ps.Close()
	} else {
		ps.Open()
	}
}

// Next, seçimi bir sonraki öğeye taşır (disabled öğeleri atlar).
func (ps *PopupState) Next(totalItems int) {
	if totalItems <= 0 {
		return
	}
	ps.Selected++
	if ps.Selected >= totalItems {
		ps.Selected = 0
	}
}

// Prev, seçimi bir önceki öğeye taşır (disabled öğeleri atlar).
func (ps *PopupState) Prev() {
	if ps.Selected > 0 {
		ps.Selected--
	}
}

// Popup, açılır menü (dropdown) widget'ıdır.
// Başlangıç butonuna tıklandığında aşağı doğru bir menü listesi açılır.
// Her öğe tıklanabilir ve odaklanabilir. Menü alanı dışına tıklandığında kapanır.
type Popup struct {
	// ID, popup'ın benzersiz tanımlayıcısıdır.
	ID string
	// Label, buton üzerindeki başlangıç metnidir.
	Label string
	// Items, menüdeki öğelerin listesidir.
	Items []PopupItem
	// State, popup'ın açık/kapalı ve seçim durumunu yönetir.
	State *PopupState
	// Style, buton ve menü arka plan stilini belirler.
	Style cell.Style
	// ItemStyle, menü öğelerinin normal stilini belirler.
	ItemStyle cell.Style
	// SelectedStyle, menü öğesinin fare sobre kaldığında/klavye ile seçili olduğundaki stilidir.
	SelectedStyle cell.Style
	// DisabledStyle, devre dışı bırakılmış menü öğelerinin stilidir.
	DisabledStyle cell.Style
	// BorderStyle, menü kenarlığının stilini belirler.
	BorderStyle cell.Style
	// BorderSymbols, menü kenarlık sembollerini belirler.
	BorderSymbols BorderSymbols
}

// Draw, popup butonunu ve açık durumdaysa menü listesini çizer.
// Menü açıldığında, her öğe için tıklama alanları ve odak bölgeleri kaydedilir.
func (p Popup) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if p.ID == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// 1. BUTON ÇİZİMİ (Her zaman görünür)
	btnStyle := ctx.Style.Merge(p.Style)
	btnText := " " + p.Label + " ▾ "
	btnW := uint16(utf8.RuneCountInString(btnText))
	btnH := uint16(1)

	// Buton arka planını doldur
	for dx := uint16(0); dx < ctx.Area.Width; dx++ {
		if c := buf.Get(ctx.Area.X+dx, ctx.Area.Y); c != nil {
			c.Content = ' '
			c.Style = btnStyle
		}
	}

	// Buton metnini yaz
	buf.SetString(ctx.Area.X, ctx.Area.Y, clipString(btnText, int(ctx.Area.Width)), btnStyle)
	_ = btnW
	_ = btnH

	// Buton tıklama alanını kaydet
	if ctx.RegisterClick != nil {
		btnArea := cell.NewRect(ctx.Area.X, ctx.Area.Y, ctx.Area.Width, 1)
		ctx.RegisterClick(btnArea, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(p.ID)
			}
			if p.State != nil {
				p.State.Toggle()
			}
		})
	}

	// Odaklanabilir olarak kaydet
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(p.ID)
	}

	// 2. MENÜ LİSTESİ ÇİZİMİ (Sadece açıksa)
	if p.State == nil || !p.State.IsOpen {
		return
	}

	// Menü genişliğini en uzun öğeye göre hesapla
	menuW := uint16(0)
	for _, item := range p.Items {
		itemLen := uint16(utf8.RuneCountInString(item.Text)) + 2 // " " padding
		if itemLen > menuW {
			menuW = itemLen
		}
	}
	// Minimum genişlik: buton genişliği
	if menuW < ctx.Area.Width {
		menuW = ctx.Area.Width
	}
	// Menü yüksekliği: kenarlık (2) + öğeler
	menuH := uint16(len(p.Items)) + 2

	// Menü alanı (butonun hemen altında)
	menuX := ctx.Area.X
	menuY := ctx.Area.Y + 1

	// Menü arka planını doldur
	menuBgStyle := ctx.Style.Merge(p.Style)
	for dy := uint16(0); dy < menuH; dy++ {
		for dx := uint16(0); dx < menuW; dx++ {
			x := menuX + dx
			y := menuY + dy
			if c := buf.Get(x, y); c != nil {
				c.Content = ' '
				c.Style = menuBgStyle
			}
		}
	}

	// Menü kenarlığını çiz
	borderStyle := ctx.Style.Merge(p.BorderStyle)
	sym := p.BorderSymbols
	if sym.TopLeft == 0 {
		sym = SymbolsRounded
	}

	// Köşeler
	buf.SetCell(menuX, menuY, cell.Cell{Content: sym.TopLeft, Style: borderStyle})
	buf.SetCell(menuX+menuW-1, menuY, cell.Cell{Content: sym.TopRight, Style: borderStyle})
	buf.SetCell(menuX, menuY+menuH-1, cell.Cell{Content: sym.BottomLeft, Style: borderStyle})
	buf.SetCell(menuX+menuW-1, menuY+menuH-1, cell.Cell{Content: sym.BottomRight, Style: borderStyle})

	// Yatay kenarlıklar
	for col := menuX + 1; col < menuX+menuW-1; col++ {
		buf.SetCell(col, menuY, cell.Cell{Content: sym.Horizontal, Style: borderStyle})
		buf.SetCell(col, menuY+menuH-1, cell.Cell{Content: sym.Horizontal, Style: borderStyle})
	}

	// Dikey kenarlıklar
	for row := menuY + 1; row < menuY+menuH-1; row++ {
		buf.SetCell(menuX, row, cell.Cell{Content: sym.Vertical, Style: borderStyle})
		buf.SetCell(menuX+menuW-1, row, cell.Cell{Content: sym.Vertical, Style: borderStyle})
	}

	// Menü öğelerini çiz
	for i, item := range p.Items {
		itemY := menuY + uint16(i) + 1 // Kenarlık payı
		isSelected := p.State.Selected == i

		itemStyle := menuBgStyle.Merge(p.ItemStyle)
		if item.Disabled {
			itemStyle = menuBgStyle.Merge(p.DisabledStyle)
		} else if isSelected {
			itemStyle = menuBgStyle.Merge(p.SelectedStyle)
		}

		// Öğe arka planını doldur
		for dx := uint16(1); dx < menuW-1; dx++ {
			x := menuX + dx
			if c := buf.Get(x, itemY); c != nil {
				c.Content = ' '
				c.Style = itemStyle
			}
		}

		// Öğe metnini yaz (kenarlık payıyla)
		displayText := " " + item.Text
		buf.SetString(menuX+1, itemY, clipString(displayText, int(menuW)-2), itemStyle)

		// Seçili öğe göstergesi
		if isSelected && !item.Disabled {
			checkMark := "▸"
			buf.SetString(menuX+1, itemY, checkMark, itemStyle)
		}

		// Tıklama alanını kaydet
		if ctx.RegisterClick != nil && !item.Disabled {
			itemArea := cell.NewRect(menuX, itemY, menuW, 1)
			handler := item.Handler
			itemIndex := i
			ctx.RegisterClick(itemArea, func() {
				if ctx.SetFocus != nil {
					ctx.SetFocus(p.ID)
				}
				if p.State != nil {
					p.State.Close()
				}
				if handler != nil {
					handler()
				}
				_ = itemIndex
			})
		}
	}

	// Fare sobre (hover) olaylarını kaydet: Menü öğelerinin üzerinde gezinirken seçimi güncelle
	for i := range p.Items {
		if !p.Items[i].Disabled {
			itemArea := cell.NewRect(menuX, menuY+uint16(i)+1, menuW, 1)
			hoverIdx := i
			if ctx.RegisterMouse != nil {
				ctx.RegisterMouse(itemArea, func(ev backend.MouseEvent) {
					if ev.Button == backend.MouseNone && p.State != nil {
						p.State.Selected = hoverIdx
					}
				})
			}
		}
	}
}

// SizeHint, popup'ın buton yüksekliğini ve varsayılan genişliğini döndürür.
func (p Popup) SizeHint(maxArea cell.Rect) (width, height uint16) {
	btnW := uint16(utf8.RuneCountInString(p.Label)) + 4 // " ▾ " padding
	if btnW > maxArea.Width {
		btnW = maxArea.Width
	}
	if btnW < 10 {
		btnW = 10
	}
	return btnW, 1
}
