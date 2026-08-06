package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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
	// Items, listede gösterilecek olan metin dizilimleridir.
	Items []string
	// Style, listenin genel rengini ve yazı stilini belirtir.
	Style cell.Style
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
	if area.Width == 0 || area.Height == 0 || len(l.Items) == 0 {
		return
	}

	listStyle := ctx.Style.Merge(l.Style)
	selStyle := listStyle.Merge(l.SelectedStyle)

	// Eğer State tanımlanmadıysa geçici yerel bir durum kullan
	state := l.State
	if state == nil {
		state = &ListState{Selected: -1, Offset: 0}
	}

	// Kaydırma (Scroll) sınırlarını otomatik çöz
	state.ScrollTo(int(area.Height), len(l.Items))

	for i := 0; i < int(area.Height); i++ {
		itemIdx := state.Offset + i
		if itemIdx >= len(l.Items) {
			break
		}

		currY := area.Y + uint16(i)
		itemText := l.Items[itemIdx]

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
			// Öğeye fareyle tıklandığında listedeki bu indeksi seç (Selected)
			ctx.RegisterClick(itemRect, func() {
				state.Selected = targetIdx
			})
		}
	}
}

// SizeHint, listenin en uzun öğesini ve toplam öğe sayısını hesaplayarak ideal boyutları döndürür.
func (l List) SizeHint(maxArea cell.Rect) (width, height uint16) {
	if len(l.Items) == 0 {
		return 0, 0
	}

	symbolLen := utf8.RuneCountInString(l.HighlightSymbol)
	maxW := 0
	for _, item := range l.Items {
		w := utf8.RuneCountInString(item) + symbolLen
		if w > maxW {
			maxW = w
		}
	}

	w := uint16(maxW)
	h := uint16(len(l.Items))

	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}

	return w, h
}
