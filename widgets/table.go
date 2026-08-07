package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// ConstraintType, sütun genişlik kuralını belirleyen kısıt türüdür.
type ConstraintType int

const (
	ConstraintFixed      ConstraintType = iota // Sabit sütun genişliği (karakter cinsinden)
	ConstraintPercentage                       // Yüzdesel sütun genişliği (toplam genişliğin %'si)
	ConstraintFill                             // Sütunlardan kalan boş alanı doldurur
)

// TableConstraint, bir sütunun genişlik kuralını tanımlar.
type TableConstraint struct {
	Type  ConstraintType
	Value int
}

// TableCell, tablodaki tek bir hücrenin metin ve stil bilgisidir.
type TableCell struct {
	Text  string
	Style cell.Style
}

// TableRow, tablodaki bir satırın hücre listesi ve satır stilidir.
type TableRow struct {
	Cells []TableCell
	Style cell.Style
}

// NewRow, verilen kelime/metin listesinden standart stilli bir satır (TableRow) oluşturur.
func NewRow(cells ...string) TableRow {
	rowCells := make([]TableCell, len(cells))
	for i, c := range cells {
		rowCells[i] = TableCell{Text: c}
	}
	return TableRow{Cells: rowCells}
}

// TableState, tablodaki satır seçimini ve dikey kaydırma (scrolling) durumunu yönetir.
type TableState struct {
	Selected int // Seçili satır indeksi (-1 ise seçim yok)
	Offset   int // Dikey kaydırma (scroll offset) miktarı
}

// NewTableState, yeni bir TableState nesnesi oluşturur.
func NewTableState() *TableState {
	return &TableState{
		Selected: -1,
		Offset:   0,
	}
}

// Select, belirli bir satırı seçer.
func (ts *TableState) Select(index int) {
	ts.Selected = index
}

// Next, seçimi bir sonraki satıra taşır.
func (ts *TableState) Next(totalRows int) {
	if totalRows <= 0 {
		return
	}
	if ts.Selected == -1 {
		ts.Selected = 0
	} else if ts.Selected < totalRows-1 {
		ts.Selected++
	}
}

// Prev, seçimi bir önceki satıra taşır.
func (ts *TableState) Prev() {
	if ts.Selected > 0 {
		ts.Selected--
	}
}

// Table, interaktif, esnek sütunlu ve dikey kaydırılabilir tablo bileşenidir.
type Table struct {
	ID            string
	Header        *TableRow
	Rows          []TableRow
	Constraints   []TableConstraint
	State         *TableState
	GridStyle     cell.Style
	SelectedStyle cell.Style
	DrawGrid      bool
}

// SolveWidths, toplam kullanılabilir tablo genişliğini sütun kurallarına göre çözerek genişlikleri belirler.
func SolveWidths(totalWidth uint16, constraints []TableConstraint) []uint16 {
	widths := make([]uint16, len(constraints))
	var usedWidth uint16
	var fillCount int

	// 1. Geçiş: Sabit ve Yüzdelik sütunları çöz
	for i, c := range constraints {
		switch c.Type {
		case ConstraintFixed:
			widths[i] = uint16(c.Value)
			usedWidth += widths[i]
		case ConstraintPercentage:
			w := uint16(int(totalWidth) * c.Value / 100)
			widths[i] = w
			usedWidth += w
		case ConstraintFill:
			fillCount++
		}
	}

	// 2. Geçiş: Kalan boşluğu Fill sütunlarına paylaştır
	if fillCount > 0 && totalWidth > usedWidth {
		remaining := totalWidth - usedWidth
		fillW := remaining / uint16(fillCount)
		extra := remaining % uint16(fillCount)

		for i, c := range constraints {
			if c.Type == ConstraintFill {
				widths[i] = fillW
				if extra > 0 {
					widths[i]++
					extra--
				}
			}
		}
	}

	return widths
}

// Draw, tabloyu render eder, başlığı yazar, satırları kaydırma offsetine göre dizer ve ızgara çizgilerini çizer.
func (t Table) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if len(t.Constraints) == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// Odaklanabilir olarak kaydet
	if t.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(t.ID)
	}

	// 1. SÜTUN GENİŞLİKLERİNİN HESAPLANMASI
	colsCount := len(t.Constraints)
	netWidth := ctx.Area.Width
	// Izgara çizgileri çiziliyorsa, her sütun arası için 1 karakterlik boşluğu düş
	if t.DrawGrid && colsCount > 1 {
		if netWidth > uint16(colsCount-1) {
			netWidth -= uint16(colsCount - 1)
		} else {
			netWidth = 1
		}
	}
	widths := SolveWidths(netWidth, t.Constraints)

	// 2. BAŞLIK ÇİZİMİ
	currY := ctx.Area.Y
	gridStyle := ctx.Style.Merge(t.GridStyle)

	if t.Header != nil {
		t.drawRow(ctx, buf, currY, *t.Header, widths, false)
		currY++

		// Başlık altı ayırıcı çizgi
		if currY < ctx.Area.Y+ctx.Area.Height {
			currX := ctx.Area.X
			for i, w := range widths {
				for col := uint16(0); col < w; col++ {
					buf.SetCell(currX, currY, cell.Cell{Content: '─', Style: gridStyle})
					currX++
				}
				if t.DrawGrid && i < len(widths)-1 {
					buf.SetCell(currX, currY, cell.Cell{Content: '┼', Style: gridStyle})
					currX++
				}
			}
			currY++
		}
	}

	// 3. SATIR SCROLL HESAPLAMALARI
	if currY >= ctx.Area.Y+ctx.Area.Height {
		return // Sadece başlık ekrana sığdı
	}
	visibleRows := int(ctx.Area.Y+ctx.Area.Height - currY)
	if visibleRows <= 0 {
		return
	}

	totalRows := len(t.Rows)
	if t.State != nil && t.State.Selected != -1 {
		// Sınır koruma
		if t.State.Selected >= totalRows {
			t.State.Selected = totalRows - 1
		}

		// Otomatik scroll kaydırma hesaplama (Focus follow)
		if t.State.Selected < t.State.Offset {
			t.State.Offset = t.State.Selected
		}
		if t.State.Selected >= t.State.Offset+visibleRows {
			t.State.Offset = t.State.Selected - visibleRows + 1
		}
	}

	// 4. SATIRLARIN ÇİZİLMESİ
	for rIdx := 0; rIdx < visibleRows; rIdx++ {
		actualRowIdx := rIdx + t.State.Offset
		if actualRowIdx >= totalRows {
			break
		}

		row := t.Rows[actualRowIdx]
		isSelected := (t.State != nil && t.State.Selected == actualRowIdx)

		// Tıklama olayını satır satır kaydet (RegisterClick)
		if ctx.RegisterClick != nil {
			rowArea := cell.NewRect(ctx.Area.X, currY, ctx.Area.Width, 1)
			targetIdx := actualRowIdx
			ctx.RegisterClick(rowArea, func() {
				if t.State != nil {
					t.State.Select(targetIdx)
				}
				if t.ID != "" && ctx.SetFocus != nil {
					ctx.SetFocus(t.ID)
				}
			})
		}

		t.drawRow(ctx, buf, currY, row, widths, isSelected)
		currY++
	}
}

// drawRow, tek bir tablo satırını belirtilen koordinatta render eder.
func (t Table) drawRow(ctx cell.Context, buf *buffer.Buffer, y uint16, row TableRow, widths []uint16, isSelected bool) {
	currX := ctx.Area.X
	gridStyle := ctx.Style.Merge(t.GridStyle)

	// Satır stili (seçiliyse SelectedStyle, değilse kendi stili)
	rowStyle := ctx.Style.Merge(row.Style)
	if isSelected {
		rowStyle = rowStyle.Merge(t.SelectedStyle)
	}

	for i, w := range widths {
		cellText := ""
		cellStyle := rowStyle

		if i < len(row.Cells) {
			c := row.Cells[i]
			cellText = c.Text
			cellStyle = rowStyle.Merge(c.Style)
		}

		// Hücre arka planını doldur (özellikle satır seçiliyken tamamı boyansın)
		for dx := uint16(0); dx < w; dx++ {
			buf.SetCell(currX+dx, y, cell.Cell{Content: ' ', Style: cellStyle})
		}

		// Metni kırp (clip) ve yaz
		clipped := clipString(cellText, int(w))
		buf.SetString(currX, y, clipped, cellStyle)

		currX += w

		// Sütunlar arası dikey çizgi
		if t.DrawGrid && i < len(widths)-1 {
			buf.SetCell(currX, y, cell.Cell{Content: '│', Style: gridStyle})
			currX++
		}
	}
}

// SizeHint, tablonun esnek yerleşim ihtiyacını belirtir.
func (t Table) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

func clipString(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxW {
		return s
	}
	if maxW <= 3 {
		// Üç nokta ekleyecek kadar yer yoksa doğrudan kes
		return string(runes[:maxW])
	}
	return string(runes[:maxW-3]) + "..."
}

// Runes count in string helper
func strLen(s string) int {
	return utf8.RuneCountInString(s)
}
