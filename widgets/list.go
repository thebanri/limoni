package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
)

// ListState, kaydırılabilir ve seçilebilir listenin durumunu (state) temsil eder.
type ListState struct {
	// Selected, listede o an seçilmiş olan öğenin indeksidir. Seçili öğe yoksa -1'dir.
	Selected int
	// Offset, ekranda listenin en üstünde gösterilen ilk öğenin indeksidir (Scroll kayma mesafesi).
	Offset int
}

// NewListState yeni bir ListState örneği oluşturur. Varsayılan olarak hiçbir öğe seçili değildir.
func NewListState() *ListState {
	return &ListState{
		Selected: -1,
		Offset:   0,
	}
}

// Select, belirtilen indeksi seçili hale getirir.
func (s *ListState) Select(index int) {
	s.Selected = index
}

// ScrollTo, seçili olan öğenin (Selected) listenin görünür yüksekliği (height) içerisinde
// her zaman görünür kalmasını garanti eder. Seçilen öğe ekran dışına taşarsa, Offset değerini otomatik kaydırır.
//
// Parametreler:
//   - height: Listenin ekrandaki görünür satır yüksekliği.
//   - total: Listedeki toplam öğe sayısı.
func (s *ListState) ScrollTo(height int, total int) {
	if s.Selected < 0 || total == 0 || height <= 0 {
		s.Offset = 0
		return
	}

	// Seçim sınır dışıysa sınırla
	if s.Selected >= total {
		s.Selected = total - 1
	}

	// Seçim ekranın yukarısında kalıyorsa görünümü yukarı kaydır
	if s.Selected < s.Offset {
		s.Offset = s.Selected
	}

	// Seçim ekranın aşağısında kalıyorsa görünümü aşağı kaydır
	if s.Selected >= s.Offset+height {
		s.Offset = s.Selected - height + 1
	}

	// Sınır korumaları
	if s.Offset < 0 {
		s.Offset = 0
	}
	maxOffset := total - height
	if s.Offset > maxOffset {
		s.Offset = maxOffset
	}
	if s.Offset < 0 {
		s.Offset = 0
	}
}

// List, terminal ekranında liste şeklinde dikey öğeler çizen interaktif widget'tır.
type List struct {
	// ID, listenin odaklanma ve kimlik belirleme kimliğidir.
	ID string
	// Items, listede gösterilecek olan metin dizilimleridir.
	Items []string
	// Provider, sanal liste (virtual scrolling) için veri sağlayıcıdır.
	// Eğer belirtilirse Items dizisi yerine bu kullanılır.
	Provider ListProvider
	// Scrollbar, aktif edilirse listenin sağ kenarında bir dikey kaydırma çubuğu çizer.
	Scrollbar bool
	// ScrollbarTrackStyle, kaydırma çubuğu rayının (track) stilidir.
	ScrollbarTrackStyle cell.Style
	// ScrollbarThumbStyle, kaydırma çubuğu kaydırıcısının (thumb) stilidir.
	ScrollbarThumbStyle cell.Style
	// Style, listenin genel rengini ve yazı stilini belirtir.
	Style cell.Style
	// FocusedStyle, liste odağa sahip olduğunda uygulanacak stildir.
	FocusedStyle cell.Style
	// SelectedStyle, seçili olan öğenin vurgulanacağı stildir.
	SelectedStyle cell.Style
	// HighlightSymbol, seçili olan öğenin soluna yerleştirilecek semboldür (örn: "> ").
	HighlightSymbol string

	// State, listenin seçili indeksi ve kaydırma durumunu tutan işaretçidir (pointer).
	State *ListState
}

// Draw, listeyi belirtilen alana çizer. Görünür öğeleri hesaplar, seçili öğeyi vurgular
// ve listedeki her öğe için otomatik fare tıklama bölgeleri (RegisterClick) kaydeder.
func (l List) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	totalItems := len(l.Items)
	if l.Provider != nil {
		totalItems = l.Provider.Len()
	}
	if area.Width == 0 || area.Height == 0 || totalItems == 0 {
		return
	}

	if l.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(l.ID)
	}

	listStyle := ctx.Style.Merge(l.Style)
	if ctx.IsFocused(l.ID) {
		listStyle = listStyle.Merge(l.FocusedStyle)
	}
	selStyle := listStyle.Merge(l.SelectedStyle)

	// Eğer State tanımlanmadıysa geçici yerel bir durum kullan
	state := l.State
	if state == nil {
		state = &ListState{Selected: -1, Offset: 0}
	}

	// Kaydırma (Scroll) sınırlarını otomatik çöz
	state.ScrollTo(int(area.Height), totalItems)

	// Fare tekerleği olaylarını dinle ve kaydır
	if ctx.RegisterMouse != nil {
		ctx.RegisterMouse(ctx.Area, func(ev backend.MouseEvent) {
			if ev.Button == backend.MouseScrollUp {
				state.Offset--
				if state.Offset < 0 {
					state.Offset = 0
				}
			} else if ev.Button == backend.MouseScrollDown {
				state.Offset++
				maxOffset := totalItems - int(ctx.Area.Height)
				if maxOffset < 0 {
					maxOffset = 0
				}
				if state.Offset > maxOffset {
					state.Offset = maxOffset
				}
			}
		})
	}

	// Dikey kaydırma çubuğunu (Scrollbar) çiz
	visibleHeight := int(area.Height)
	if l.Scrollbar && totalItems > visibleHeight && area.Width > 1 {
		scrollbarX := area.X + area.Width - 1
		thumbH := (visibleHeight * visibleHeight) / totalItems
		if thumbH < 1 {
			thumbH = 1
		}
		maxOffset := totalItems - visibleHeight
		thumbY := 0
		if maxOffset > 0 {
			thumbY = (state.Offset * (visibleHeight - thumbH)) / maxOffset
		}

		trackStyle := listStyle.Merge(l.ScrollbarTrackStyle)
		if l.ScrollbarTrackStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
			trackStyle = listStyle.Merge(ctx.ThemeStyle("border"))
		}
		thumbStyle := listStyle.Merge(l.ScrollbarThumbStyle)
		if l.ScrollbarThumbStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
			thumbStyle = listStyle.Merge(ctx.ThemeStyle("focus"))
		}

		for y := 0; y < visibleHeight; y++ {
			c := buf.Get(scrollbarX, area.Y+uint16(y))
			if c != nil {
				c.Content = '░'
				c.Style = trackStyle
				if y >= thumbY && y < thumbY+thumbH {
					c.Content = '█'
					c.Style = thumbStyle
				}
			}
		}
		// Ray alanını metin çizim alanından düş
		area.Width--
	}

	for i := 0; i < int(area.Height); i++ {
		itemIdx := state.Offset + i
		if itemIdx >= totalItems {
			break
		}

		currY := area.Y + uint16(i)
		itemText := ""
		if l.Provider != nil {
			itemText = l.Provider.ItemAt(itemIdx)
		} else {
			itemText = l.Items[itemIdx]
		}

		isSel := itemIdx == state.Selected
		itemStyle := listStyle
		displayText := itemText

		if isSel {
			itemStyle = selStyle
			if l.HighlightSymbol != "" {
				displayText = l.HighlightSymbol + itemText
			}
		}

		// Satırın arka planını temizle ve doldur
		for x := area.X; x < area.X+area.Width; x++ {
			if c := buf.Get(x, currY); c != nil {
				c.Content = ' '
				c.Style = itemStyle
			}
		}

		// Metni çiz
		buf.SetString(area.X, currY, displayText, itemStyle)

		// Otomatik fare yönlendirme köprüsünü bağla
		if ctx.RegisterClick != nil {
			targetIdx := itemIdx
			itemRect := cell.Rect{
				X:      area.X,
				Y:      currY,
				Width:  area.Width,
				Height: 1,
			}
			// Öğeye fareyle tıklandığında listedeki bu indeksi seç (Selected) ve odaklan
			ctx.RegisterClick(itemRect, func() {
				state.Selected = targetIdx
				if l.ID != "" && ctx.SetFocus != nil {
					ctx.SetFocus(l.ID)
				}
			})
		}
	}
}

// SizeHint, listenin en uzun öğesini ve toplam öğe sayısını hesaplayarak ideal boyutları döndürür.
func (l List) SizeHint(maxArea cell.Rect) (width, height uint16) {
	totalItems := len(l.Items)
	if l.Provider != nil {
		totalItems = l.Provider.Len()
	}
	if totalItems == 0 {
		return 0, 0
	}

	symbolLen := utf8.RuneCountInString(l.HighlightSymbol)
	maxW := 0

	if l.Provider != nil {
		limit := totalItems
		if limit > 100 {
			limit = 100
		}
		for i := 0; i < limit; i++ {
			w := utf8.RuneCountInString(l.Provider.ItemAt(i)) + symbolLen
			if w > maxW {
				maxW = w
			}
		}
	} else {
		for _, item := range l.Items {
			w := utf8.RuneCountInString(item) + symbolLen
			if w > maxW {
				maxW = w
			}
		}
	}

	if l.Scrollbar && totalItems > int(maxArea.Height) {
		maxW++
	}

	w := uint16(maxW)
	h := uint16(totalItems)

	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}

	return w, h
}

// Measure provides explicit size negotiation for List.
func (l List) Measure(maxArea cell.Rect) layout.Measure {
	w, h := l.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}
