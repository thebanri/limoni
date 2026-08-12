package widgets

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
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
	Text    string
	Style   cell.Style
	ColSpan int // Birleştirilecek sütun sayısı (varsayılan veya 0/1 ise tek sütun)
	RowSpan int // Birleştirilecek satır sayısı (varsayılan veya 0/1 ise tek satır)
}

// TableRow, tablodaki bir satırın hücre listesi ve satır stilidir.
type TableRow struct {
	Cells []TableCell
	Style cell.Style
}

// SearchText returns the searchable text of all cells in the row.
func (r TableRow) SearchText() string {
	parts := make([]string, 0, len(r.Cells))
	for _, c := range r.Cells {
		parts = append(parts, c.Text)
	}
	return strings.Join(parts, " ")
}

// NewRow, verilen kelime/metin listesinden standart stilli bir satır (TableRow) oluşturur.
func NewRow(cells ...string) TableRow {
	rowCells := make([]TableCell, len(cells))
	for i, c := range cells {
		rowCells[i] = TableCell{Text: c}
	}
	return TableRow{Cells: rowCells}
}

// TableState, tablodaki satır seçimini, dikey kaydırma (scrolling) ve sütun genişliklerini yönetir.
type TableState struct {
	Selected         int      // Seçili satır indeksi (-1 ise seçim yok)
	Offset           int      // Dikey kaydırma (scroll offset)
	HorizontalOffset int      // Yatay kaydırma için hazırlanan sütun hücre offset'i miktarı
	ColumnWidths     []uint16 // Sürüklenerek yeniden boyutlandırılan veya otomatik çözülen sütun genişlikleri
	SortColumn       int      // Sıralanan sütun; -1 ise sıralama kapalı
	SortDescending   bool
	SelectedRows     map[int]struct{} // Çoklu satır seçimi
	selectionDirty   bool             // Seçim değiştiğinde görünürlük ayarı gerektiğini belirtir.
}

type tableDrawScratch struct {
	widths    []uint16
	owner     map[[2]int][2]int
	cells     map[[2]int]TableCell
	filtered  []TableRow
}

var tableDrawScratchPool = sync.Pool{
	New: func() any {
		return &tableDrawScratch{}
	},
}

// NewTableState, yeni bir TableState nesnesi oluşturur.
func NewTableState() *TableState {
	return &TableState{
		Selected:       -1,
		Offset:         0,
		ColumnWidths:   nil,
		SortColumn:     -1,
		SortDescending: false,
		SelectedRows:   make(map[int]struct{}),
	}
}

// Select, belirli bir satırı seçer.
func (ts *TableState) Select(index int) {
	ts.Selected = index
	ts.selectionDirty = true
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
	ts.selectionDirty = true
}

// Prev, seçimi bir önceki satıra taşır.
func (ts *TableState) Prev() {
	if ts.Selected > 0 {
		ts.Selected--
		ts.selectionDirty = true
	}
}

// Scroll moves the vertical viewport while clamping it to the row count.
func (ts *TableState) Scroll(delta, totalRows, visibleRows int) {
	if ts == nil || visibleRows <= 0 {
		return
	}
	maxOffset := totalRows - visibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	ts.Offset += delta
	if ts.Offset < 0 {
		ts.Offset = 0
	}
	if ts.Offset > maxOffset {
		ts.Offset = maxOffset
	}
}

func (ts *TableState) ScrollHorizontal(delta int) {
	if ts == nil {
		return
	}
	ts.HorizontalOffset += delta
	if ts.HorizontalOffset < 0 {
		ts.HorizontalOffset = 0
	}
}

// ToggleRow toggles a row in the multi-selection set.
func (ts *TableState) ToggleRow(index int) {
	if ts == nil || index < 0 {
		return
	}
	if ts.SelectedRows == nil {
		ts.SelectedRows = make(map[int]struct{})
	}
	if _, exists := ts.SelectedRows[index]; exists {
		delete(ts.SelectedRows, index)
	} else {
		ts.SelectedRows[index] = struct{}{}
	}
}

func (ts *TableState) IsRowSelected(index int) bool {
	if ts == nil {
		return false
	}
	_, selected := ts.SelectedRows[index]
	return selected
}

func (ts *TableState) ClearSelectedRows() {
	if ts != nil {
		ts.SelectedRows = make(map[int]struct{})
	}
}

// MoveSortColumn selects the next/previous sortable column.
func (ts *TableState) MoveSortColumn(delta, columnCount int) {
	if ts == nil || columnCount <= 0 {
		return
	}
	if ts.SortColumn < 0 {
		ts.SortColumn = 0
		return
	}
	ts.SortColumn = (ts.SortColumn + delta + columnCount) % columnCount
}

// ResizeColumn changes a column width while preserving the table's total width.
// Growing a column shrinks columns to its right; shrinking it gives the freed
// space to the last column. Every column keeps at least two cells.
func (ts *TableState) ResizeColumn(index, delta int) bool {
	if ts == nil || index < 0 || index >= len(ts.ColumnWidths)-1 || delta == 0 {
		return false
	}

	const minWidth = 2
	if delta > 0 {
		remaining := delta
		for i := len(ts.ColumnWidths) - 1; i > index && remaining > 0; i-- {
			available := int(ts.ColumnWidths[i]) - minWidth
			if available <= 0 {
				continue
			}
			shrink := available
			if shrink > remaining {
				shrink = remaining
			}
			ts.ColumnWidths[i] -= uint16(shrink)
			remaining -= shrink
		}
		actual := delta - remaining
		if actual > 0 {
			ts.ColumnWidths[index] += uint16(actual)
		}
		return actual > 0
	}

	requested := int(ts.ColumnWidths[index]) + delta
	if requested < minWidth {
		requested = minWidth
	}
	freed := int(ts.ColumnWidths[index]) - requested
	if freed == 0 {
		return false
	}
	ts.ColumnWidths[index] = uint16(requested)
	ts.ColumnWidths[len(ts.ColumnWidths)-1] += uint16(freed)
	return true
}

// TableDataSource provides rows lazily for large tables.
type TableDataSource interface {
	RowCount() int
	RowAt(index int) TableRow
}

// Table, interaktif, esnek sütunlu, dikey kaydırılabilir ve hücre birleştirme destekli tablo bileşenidir.
type Table struct {
	ID            string
	Header        *TableRow
	Rows          []TableRow
	DataSource    TableDataSource
	Constraints   []TableConstraint
	State         *TableState
	GridStyle     cell.Style
	SelectedStyle cell.Style
	DrawGrid      bool
	SortEnabled   bool                                              // Başlık hücrelerine tıklayarak satır sıralamayı etkinleştirir.
	MultiSelect   bool                                              // Space ile birden fazla satırın seçilmesini etkinleştirir.
	FilterQuery   string                                            // Fuzzy filtre sorgusu; boşsa tüm satırlar çizilir.
	CellStyle     func(row, column int, value TableCell) cell.Style // Hücre bazlı stil kuralı.
	StickyColumns int                                               // Soldan sabit kalacak sütun sayısı.
	Scrollbar     bool                                              // Sağ kenarda dikey kaydırma çubuğu çizer.
}

func (t Table) columnX(area cell.Rect, widths []uint16, column int) uint16 {
	sticky := t.StickyColumns
	if sticky < 0 {
		sticky = 0
	}
	if sticky > len(widths) {
		sticky = len(widths)
	}
	stickyWidth := uint16(0)
	for i := 0; i < sticky; i++ {
		stickyWidth += widths[i]
		if t.DrawGrid && i < len(widths)-1 {
			stickyWidth++
		}
	}
	if column < sticky {
		x := area.X
		for i := 0; i < column; i++ {
			x += widths[i]
			if t.DrawGrid {
				x++
			}
		}
		return x
	}
	x := area.X + stickyWidth
	for i := sticky; i < column; i++ {
		x += widths[i]
		if t.DrawGrid {
			x++
		}
	}
	offset := uint16(0)
	if t.State != nil && t.State.HorizontalOffset > 0 {
		offset = uint16(t.State.HorizontalOffset)
	}
	available := x - area.X - stickyWidth
	if offset > available {
		offset = available
	}
	return x - offset
}

// SolveWidths, toplam kullanılabilir tablo genişliğini sütun kurallarına göre çözerek genişlikleri belirler.
func SolveWidths(totalWidth uint16, constraints []TableConstraint) []uint16 {
	return solveWidthsInto(nil, totalWidth, constraints)
}

func solveWidthsInto(widths []uint16, totalWidth uint16, constraints []TableConstraint) []uint16 {
	if cap(widths) < len(constraints) {
		widths = make([]uint16, len(constraints))
	} else {
		widths = widths[:len(constraints)]
		for i := range widths {
			widths[i] = 0
		}
	}
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

func getOwnerCell(owner map[[2]int][2]int, r, c int) [2]int {
	if val, exists := owner[[2]int{r, c}]; exists {
		return val
	}
	return [2]int{r, c}
}

// Draw, tabloyu render eder, başlığı yazar, satırları kaydırma offsetine göre dizer ve ızgara çizgilerini çizer.
func (t Table) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if len(t.Constraints) == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	scratch := tableDrawScratchPool.Get().(*tableDrawScratch)
	defer tableDrawScratchPool.Put(scratch)
	if scratch.owner == nil {
		scratch.owner = make(map[[2]int][2]int)
	}
	if scratch.cells == nil {
		scratch.cells = make(map[[2]int]TableCell)
	}
	if t.State != nil && t.SortEnabled && t.State.SortColumn >= 0 && t.DataSource == nil {
		sortTableRows(t.Rows, t.State.SortColumn, t.State.SortDescending)
	}

	// Odaklanabilir olarak kaydet
	if t.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(t.ID)
	}

	// 1. SATIR SAYISININ VE FİLTRENİN HESAPLANMASI
	rows := t.Rows
	rowCount := len(rows)
	if t.DataSource != nil {
		rowCount = t.DataSource.RowCount()
	}
	if t.FilterQuery != "" {
		filtered := scratch.filtered[:0]
		if cap(filtered) < rowCount {
			filtered = make([]TableRow, 0, rowCount)
		}
		for i := 0; i < rowCount; i++ {
			var row TableRow
			if t.DataSource != nil {
				row = t.DataSource.RowAt(i)
			} else {
				row = rows[i]
			}
			if _, matched := FuzzyMatch(t.FilterQuery, row.SearchText()); matched {
				filtered = append(filtered, row)
			}
		}
		scratch.filtered = filtered[:0]
		rows = filtered
		rowCount = len(rows)
	}

	// 2. SCROLLBAR TALEBİNİN VE ALAN GENİŞLİĞİNİN HESAPLANMASI
	visibleRows := int(ctx.Area.Height)
	if t.Header != nil {
		visibleRows--
		if ctx.Area.Height > 1 {
			visibleRows--
		}
	}
	if visibleRows < 0 {
		visibleRows = 0
	}

	drawScrollbar := false
	if t.Scrollbar && t.State != nil && rowCount > visibleRows && ctx.Area.Width > 1 {
		drawScrollbar = true
		ctx.Area.Width--
	}

	// 3. SÜTUN GENİŞLİKLERİNİN HESAPLANMASI VE İLKLENDİRİLMESİ
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

	var widths []uint16
	if t.State != nil {
		// Sütun genişliklerini sakla ve ekran boyutu değiştiyse yeniden hesapla
		var totalStoredWidth uint16
		for _, w := range t.State.ColumnWidths {
			totalStoredWidth += w
		}
		if len(t.State.ColumnWidths) != colsCount || totalStoredWidth != netWidth {
			t.State.ColumnWidths = solveWidthsInto(t.State.ColumnWidths, netWidth, t.Constraints)
		}
		widths = t.State.ColumnWidths
	} else {
		widths = solveWidthsInto(scratch.widths, netWidth, t.Constraints)
		scratch.widths = widths[:0]
	}

	sticky := t.StickyColumns
	if sticky < 0 {
		sticky = 0
	}
	if sticky > colsCount {
		sticky = colsCount
	}
	stickyWidth := uint16(0)
	for i := 0; i < sticky; i++ {
		stickyWidth += widths[i]
		if t.DrawGrid && i < colsCount-1 {
			stickyWidth++
		}
	}

	if ctx.RegisterMouse != nil {
		t.registerScrollHandlers(ctx, rowCount)
		t.registerResizeHandlers(ctx, widths, colsCount, sticky, stickyWidth)
	}
	if ctx.RegisterClick != nil {
		t.registerSortHandlers(ctx, widths, colsCount)
	}

	// 3. SAHİPLİK MATRİSİNİN (COLSPAN / ROWSPAN) HESAPLANMASI
	owner := scratch.owner
	clear(owner)
	cellsMap := scratch.cells
	clear(cellsMap)

	// Header satırını (row -1) matrise işle
	if t.Header != nil {
		cellIdx := 0
		for colIdx := 0; colIdx < colsCount; {
			if _, exists := owner[[2]int{-1, colIdx}]; exists {
				colIdx++
				continue
			}
			if cellIdx >= len(t.Header.Cells) {
				break
			}
			cVal := t.Header.Cells[cellIdx]
			cellIdx++
			if t.State != nil && t.State.SortColumn == colIdx && cVal.ColSpan <= 1 {
				indicator := " ▲"
				if t.State.SortDescending {
					indicator = " ▼"
				}
				cVal.Text += indicator
			}

			colSpan := cVal.ColSpan
			if colSpan < 1 {
				colSpan = 1
			}
			rowSpan := cVal.RowSpan
			if rowSpan < 1 {
				rowSpan = 1
			}

			cellsMap[[2]int{-1, colIdx}] = cVal

			for dr := 0; dr < rowSpan; dr++ {
				for dc := 0; dc < colSpan; dc++ {
					owner[[2]int{-1 + dr, colIdx + dc}] = [2]int{-1, colIdx}
				}
			}
			colIdx += colSpan
		}
	}

	// Body satırlarını matrise işle (satır kırpma optimizasyonu ile)
	offset := 0
	if t.State != nil {
		offset = t.State.Offset
	}
	startRow := offset - 2
	if startRow < 0 {
		startRow = 0
	}
	endRow := offset + int(ctx.Area.Height) + 2
	if endRow > len(rows) {
		endRow = len(rows)
	}

	for rIdx := startRow; rIdx < endRow; rIdx++ {
		row := rows[rIdx]
		cellIdx := 0
		for colIdx := 0; colIdx < colsCount; {
			if _, exists := owner[[2]int{rIdx, colIdx}]; exists {
				colIdx++
				continue
			}
			if cellIdx >= len(row.Cells) {
				break
			}
			cVal := row.Cells[cellIdx]
			cellIdx++

			colSpan := cVal.ColSpan
			if colSpan < 1 {
				colSpan = 1
			}
			rowSpan := cVal.RowSpan
			if rowSpan < 1 {
				rowSpan = 1
			}

			cellsMap[[2]int{rIdx, colIdx}] = cVal

			for dr := 0; dr < rowSpan; dr++ {
				for dc := 0; dc < colSpan; dc++ {
					owner[[2]int{rIdx + dr, colIdx + dc}] = [2]int{rIdx, colIdx}
				}
			}
			colIdx += colSpan
		}
	}

	// 4. BAŞLIK ÇİZİMİ
	currY := ctx.Area.Y
	gridStyle := ctx.Style.Merge(t.GridStyle)

	if t.Header != nil {
		t.drawSpanRow(ctx, buf, currY, -1, widths, false, owner, cellsMap, gridStyle, t.Header.Style)
		currY++

		// Başlık altı ayırıcı çizgi
		if currY < ctx.Area.Y+ctx.Area.Height {
			targetBodyRow := 0
			if t.State != nil {
				targetBodyRow = t.State.Offset
			}

			for i, w := range widths {
				// Yatay çizginin birleştirilmiş hücre tarafından örtülüp örtülmediğini denetle
				sepCovered := getOwnerCell(owner, -1, i) == getOwnerCell(owner, targetBodyRow, i)
				startX := t.columnX(ctx.Area, widths, i)

				// Clip boundaries for this column
				clipLeft := ctx.Area.X
				clipRight := ctx.Area.X + ctx.Area.Width
				if sticky > 0 {
					if i < sticky {
						clipRight = ctx.Area.X + stickyWidth
					} else {
						clipLeft = ctx.Area.X + stickyWidth
					}
				}

				for col := uint16(0); col < w; col++ {
					x := startX + col
					if !sepCovered && x >= clipLeft && x < clipRight {
						buf.SetCell(x, currY, cell.Cell{Content: '─', Style: gridStyle})
					}
				}

				if t.DrawGrid && i < colsCount-1 {
					// Dikey ve yatay çizgilerin birleştiği kesişim karakterini seç
					up := getOwnerCell(owner, -1, i) != getOwnerCell(owner, -1, i+1)
					down := getOwnerCell(owner, targetBodyRow, i) != getOwnerCell(owner, targetBodyRow, i+1)
					left := getOwnerCell(owner, -1, i) != getOwnerCell(owner, targetBodyRow, i)
					right := getOwnerCell(owner, -1, i+1) != getOwnerCell(owner, targetBodyRow, i+1)

					ch := getIntersectionChar(up, down, left, right)
					separatorX := t.columnX(ctx.Area, widths, i+1) - 1

					// Clip boundaries for intersection separator
					clipLeftSep := ctx.Area.X
					clipRightSep := ctx.Area.X + ctx.Area.Width
					if sticky > 0 {
						if i+1 < sticky {
							clipRightSep = ctx.Area.X + stickyWidth
						} else {
							clipLeftSep = ctx.Area.X + stickyWidth
						}
					}

					if ch != ' ' && separatorX >= clipLeftSep && separatorX < clipRightSep {
						buf.SetCell(separatorX, currY, cell.Cell{Content: ch, Style: gridStyle})
					}
				}
			}
			currY++
		}
	}

	// 5. SATIR SCROLL HESAPLAMALARI
	if currY >= ctx.Area.Y+ctx.Area.Height {
		return
	}
	visibleRows = int(ctx.Area.Y + ctx.Area.Height - currY)
	if visibleRows <= 0 {
		return
	}

	totalRows := rowCount
	if t.State != nil {
		if totalRows == 0 {
			t.State.Selected = -1
			t.State.Offset = 0
			return
		}
		if t.State.Offset < 0 {
			t.State.Offset = 0
		}
		if t.State.Selected >= totalRows {
			t.State.Selected = totalRows - 1
		}
		if t.State.selectionDirty && t.State.Selected != -1 && t.State.Selected < t.State.Offset {
			t.State.Offset = t.State.Selected
		}
		if t.State.selectionDirty && t.State.Selected != -1 && t.State.Selected >= t.State.Offset+visibleRows {
			t.State.Offset = t.State.Selected - visibleRows + 1
		}
		t.State.selectionDirty = false
	}

	// 6. SATIRLARIN ÇİZİLMESİ
	for rIdx := 0; rIdx < visibleRows; rIdx++ {
		offset := 0
		if t.State != nil {
			offset = t.State.Offset
		}
		actualRowIdx := rIdx + offset
		if actualRowIdx < 0 || actualRowIdx >= totalRows {
			break
		}

		var row TableRow
		if t.DataSource != nil {
			row = t.DataSource.RowAt(actualRowIdx)
		} else {
			row = rows[actualRowIdx]
		}
		isSelected := t.State != nil && (t.State.Selected == actualRowIdx || (t.MultiSelect && t.State.IsRowSelected(actualRowIdx)))

		if ctx.RegisterClick != nil {
			t.registerRowClickHandler(ctx, cell.NewRect(ctx.Area.X, currY, ctx.Area.Width, 1), actualRowIdx)
		}

		t.drawSpanRow(ctx, buf, currY, actualRowIdx, widths, isSelected, owner, cellsMap, gridStyle, row.Style)
		currY++
	}

	// Scrollbar Çizimi
	if drawScrollbar {
		scrollbarX := ctx.Area.X + ctx.Area.Width
		scrollbarH := int(ctx.Area.Height)
		thumbH := (scrollbarH * scrollbarH) / rowCount
		if thumbH < 1 {
			thumbH = 1
		}
		maxOffset := rowCount - visibleRows
		thumbY := 0
		if maxOffset > 0 {
			thumbY = (t.State.Offset * (scrollbarH - thumbH)) / maxOffset
		}

		if t.GridStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
			gridStyle = gridStyle.Merge(ctx.ThemeStyle("border"))
		}
		thumbStyle := gridStyle
		if ctx.ThemeStyle != nil {
			thumbStyle = thumbStyle.Merge(ctx.ThemeStyle("focus"))
		}

		for y := 0; y < scrollbarH; y++ {
			c := buf.Get(scrollbarX, ctx.Area.Y+uint16(y))
			if c != nil {
				c.Content = '░'
				c.Style = gridStyle
				if y >= thumbY && y < thumbY+thumbH {
					c.Content = '█'
					c.Style = thumbStyle
				}
			}
		}
	}
}

func (t Table) registerScrollHandlers(ctx cell.Context, rowCount int) {
	if t.State == nil || ctx.RegisterMouse == nil {
		return
	}
	ctx.RegisterMouse(ctx.Area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseScrollUp {
			if ev.Shift {
				t.State.ScrollHorizontal(-2)
			} else {
				t.State.Scroll(-3, rowCount, int(ctx.Area.Height))
			}
		}
		if ev.Button == backend.MouseScrollDown {
			if ev.Shift {
				t.State.ScrollHorizontal(2)
			} else {
				t.State.Scroll(3, rowCount, int(ctx.Area.Height))
			}
		}
	})
}

func (t Table) registerResizeHandlers(ctx cell.Context, widths []uint16, colsCount int, sticky int, stickyWidth uint16) {
	if t.State == nil || ctx.RegisterMouse == nil {
		return
	}
	for i := 0; i < colsCount-1; i++ {
		sepX := t.columnX(ctx.Area, widths, i+1)
		if t.DrawGrid {
			sepX--
		}

		clipLeftSep := ctx.Area.X
		clipRightSep := ctx.Area.X + ctx.Area.Width
		if sticky > 0 {
			if i+1 < sticky {
				clipRightSep = ctx.Area.X + stickyWidth
			} else {
				clipLeftSep = ctx.Area.X + stickyWidth
			}
		}

		if sepX >= clipLeftSep && sepX < clipRightSep {
			handleArea := cell.NewRect(sepX, ctx.Area.Y, 1, ctx.Area.Height)
			colIdx := i

			ctx.RegisterMouse(handleArea, func(ev backend.MouseEvent) {
				if ev.Button == backend.MouseLeft && !ev.Drag {
					startMouseX := int(ev.X)
					startColW := int(t.State.ColumnWidths[colIdx])

					ctx.CaptureMouse(func(dragEv backend.MouseEvent) {
						if dragEv.Button == backend.MouseRelease {
							return
						}
						dx := int(dragEv.X) - startMouseX
						requestedNewW := startColW + dx
						if requestedNewW < 2 {
							requestedNewW = 2
						}
						delta := requestedNewW - int(t.State.ColumnWidths[colIdx])
						t.State.ResizeColumn(colIdx, delta)
					})
				}
			})
		}
	}
}

func (t Table) registerSortHandlers(ctx cell.Context, widths []uint16, colsCount int) {
	if !t.SortEnabled || t.Header == nil || ctx.RegisterClick == nil {
		return
	}
	for colIdx, width := range widths {
		currX := t.columnX(ctx.Area, widths, colIdx)
		clickWidth := width
		if t.DrawGrid && colIdx < colsCount-1 && clickWidth > 0 {
			clickWidth--
		}
		if clickWidth > 0 {
			column := colIdx
			ctx.RegisterClick(cell.NewRect(currX, ctx.Area.Y, clickWidth, 1), func() {
				if t.State == nil {
					return
				}
				if t.State.SortColumn == column {
					t.State.SortDescending = !t.State.SortDescending
				} else {
					t.State.SortColumn = column
					t.State.SortDescending = false
				}
				sortTableRows(t.Rows, column, t.State.SortDescending)
			})
		}
	}
}

func (t Table) registerRowClickHandler(ctx cell.Context, rowArea cell.Rect, targetIdx int) {
	if ctx.RegisterClick == nil {
		return
	}
	ctx.RegisterClick(rowArea, func() {
		if t.State != nil {
			t.State.Select(targetIdx)
		}
		if t.ID != "" && ctx.SetFocus != nil {
			ctx.SetFocus(t.ID)
		}
	})
}

// drawSpanRow, birleştirilmiş hücrelere duyarlı olarak tek bir tablo satırını çizdirir.
func (t Table) drawSpanRow(
	ctx cell.Context,
	buf *buffer.Buffer,
	y uint16,
	r int,
	widths []uint16,
	isSelected bool,
	owner map[[2]int][2]int,
	cellsMap map[[2]int]TableCell,
	gridStyle cell.Style,
	baseRowStyle cell.Style,
) {
	rowStyle := ctx.Style.Merge(baseRowStyle)
	colsCount := len(widths)
	currX := ctx.Area.X
	if isSelected {
		rowStyle = rowStyle.Merge(t.SelectedStyle)
	}

	sticky := t.StickyColumns
	if sticky < 0 {
		sticky = 0
	}
	if sticky > colsCount {
		sticky = colsCount
	}
	stickyWidth := uint16(0)
	for i := 0; i < sticky; i++ {
		stickyWidth += widths[i]
		if t.DrawGrid && i < colsCount-1 {
			stickyWidth++
		}
	}

	for colIdx := 0; colIdx < colsCount; colIdx++ {
		currX = t.columnX(ctx.Area, widths, colIdx)
		ownerCoords := getOwnerCell(owner, r, colIdx)

		// Determine horizontal clipping boundaries for this column
		clipLeft := ctx.Area.X
		clipRight := ctx.Area.X + ctx.Area.Width
		if sticky > 0 {
			if colIdx < sticky {
				clipRight = ctx.Area.X + stickyWidth
			} else {
				clipLeft = ctx.Area.X + stickyWidth
			}
		}

		// Eğer bu hücre üstteki veya soldaki birleştirilmiş bir hücrenin alt parçasıysa çizimi atla
		if ownerCoords != [2]int{r, colIdx} {
			if t.DrawGrid && colIdx < colsCount-1 {
				currX = t.columnX(ctx.Area, widths, colIdx+1)
				// Sınır çizgisi hücre birleştirme alanı içinde kalmıyorsa çiz
				if getOwnerCell(owner, r, colIdx) != getOwnerCell(owner, r, colIdx+1) {
					separatorX := currX - 1
					// Clip the vertical grid line separator
					clipLeftSep := ctx.Area.X
					clipRightSep := ctx.Area.X + ctx.Area.Width
					if sticky > 0 {
						if colIdx+1 < sticky {
							clipRightSep = ctx.Area.X + stickyWidth
						} else {
							clipLeftSep = ctx.Area.X + stickyWidth
						}
					}
					if separatorX >= clipLeftSep && separatorX < clipRightSep {
						buf.SetCell(separatorX, y, cell.Cell{Content: '│', Style: gridStyle})
					}
				}
			}
			continue
		}

		// Bu hücre birleştirilmiş alanın başlangıç (ana) hücresidir
		cellVal := cellsMap[[2]int{r, colIdx}]
		cellStyle := rowStyle.Merge(cellVal.Style)
		if t.CellStyle != nil {
			cellStyle = cellStyle.Merge(t.CellStyle(r, colIdx, cellVal))
		}

		colSpan := cellVal.ColSpan
		if colSpan < 1 {
			colSpan = 1
		}
		rowSpan := cellVal.RowSpan
		if rowSpan < 1 {
			rowSpan = 1
		}

		// Birleşik hücrenin toplam karakter genişliğini hesapla (komşu sütunlar + aralarındaki ızgaralar)
		cellW := uint16(0)
		for c := 0; c < colSpan && colIdx+c < colsCount; c++ {
			cellW += widths[colIdx+c]
			if t.DrawGrid && c > 0 {
				cellW++
			}
		}

		// Hücre arka planını doldur (dikey rowSpan kadar satıra ve cellW genişliğine yayılır)
		for dy := 0; dy < rowSpan; dy++ {
			drawY := y + uint16(dy)
			if drawY >= ctx.Area.Y+ctx.Area.Height {
				break
			}
			for dx := uint16(0); dx < cellW; dx++ {
				xPixel := currX + dx
				if xPixel >= clipLeft && xPixel < clipRight {
					buf.SetCell(xPixel, drawY, cell.Cell{Content: ' ', Style: cellStyle})
				}
			}
		}

		// Metni keserek sadece ilk satıra yazdır (top-left) - clipping-aware
		clipped := clipString(cellVal.Text, int(cellW))
		drawTextClipped(buf, currX, y, clipped, cellStyle, clipLeft, clipRight)

		// Sütunlar arası dikey ızgara çizgisini çiz (birleştirilmiş alanın dışındaysa)
		if t.DrawGrid && colIdx < colsCount-1 {
			separatorX := t.columnX(ctx.Area, widths, colIdx+1) - 1
			// Clip the separator
			clipLeftSep := ctx.Area.X
			clipRightSep := ctx.Area.X + ctx.Area.Width
			if sticky > 0 {
				if colIdx+1 < sticky {
					clipRightSep = ctx.Area.X + stickyWidth
				} else {
					clipLeftSep = ctx.Area.X + stickyWidth
				}
			}
			if getOwnerCell(owner, r, colIdx) != getOwnerCell(owner, r, colIdx+1) && separatorX >= clipLeftSep && separatorX < clipRightSep {
				buf.SetCell(separatorX, y, cell.Cell{Content: '│', Style: gridStyle})
			}
		}
	}
}

// drawTextClipped draws text on a buffer with precise left and right pixel clipping boundaries.
func drawTextClipped(buf *buffer.Buffer, startX, y uint16, s string, style cell.Style, clipLeft, clipRight uint16) {
	if y >= buf.Area.Height || startX >= clipRight {
		return
	}

	currX := startX
	input := s
	for len(input) > 0 {
		r, size := utf8.DecodeRuneInString(input)
		if r == utf8.RuneError {
			break
		}

		w := cell.RuneWidth(r)
		if w == 0 {
			input = input[size:]
			continue
		}
		if currX+uint16(w) > clipRight {
			break // Exceeds right boundary
		}

		// Only write to buffer if it is within the horizontal clipping range
		if currX >= clipLeft {
			idx := y*buf.Area.Width + currX
			buf.Content[idx].Content = r
			buf.Content[idx].Style = style

			if w == 2 {
				if currX+1 < clipRight {
					buf.Content[idx+1].Content = cell.RuneContinuation
					buf.Content[idx+1].Style = style
				}
			}
		}

		currX += uint16(w)
		input = input[size:]
	}
}

func sortTableRows(rows []TableRow, column int, descending bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := "", ""
		if column >= 0 && column < len(rows[i].Cells) {
			left = rows[i].Cells[column].Text
		}
		if column >= 0 && column < len(rows[j].Cells) {
			right = rows[j].Cells[column].Text
		}
		comparison := compareTableValues(left, right)
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareTableValues(left, right string) int {
	leftValue, leftNumeric := numericTableValue(left)
	rightValue, rightNumeric := numericTableValue(right)
	if leftNumeric && rightNumeric {
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
		return 0
	}
	leftLower, rightLower := strings.ToLower(strings.TrimSpace(left)), strings.ToLower(strings.TrimSpace(right))
	if leftLower < rightLower {
		return -1
	}
	if leftLower > rightLower {
		return 1
	}
	return 0
}

func numericTableValue(value string) (float64, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return 0, false
	}
	number := strings.TrimSuffix(fields[0], "%")
	parsed, err := strconv.ParseFloat(number, 64)
	return parsed, err == nil
}

// SizeHint, tablonun esnek yerleşim ihtiyacını belirtir.
func (t Table) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

// Measure provides explicit size negotiation for Table.
func (t Table) Measure(maxArea cell.Rect) layout.Measure {
	w, h := t.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowScroll,
	}
}

func clipString(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}

	width := 0
	for _, r := range s {
		runeWidth := cell.RuneWidth(r)
		if runeWidth == 0 {
			continue
		}
		if width+runeWidth > maxW {
			break
		}
		width += runeWidth
	}
	if width == visualWidth(s) {
		return s
	}
	if maxW <= 3 {
		return clipToWidth(s, maxW)
	}
	return clipToWidth(s, maxW-3) + "..."
}

func visualWidth(s string) int {
	width := 0
	for _, r := range s {
		width += cell.RuneWidth(r)
	}
	return width
}

func clipToWidth(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	width := 0
	end := 0
	for _, r := range s {
		runeWidth := cell.RuneWidth(r)
		if runeWidth == 0 {
			continue
		}
		if width+runeWidth > maxW {
			break
		}
		width += runeWidth
		end += len(string(r))
	}
	return s[:end]
}

// Runes count in string helper
func strLen(s string) int {
	return utf8.RuneCountInString(s)
}

// getIntersectionChar, etrafındaki etkin çizgilerin durumuna göre doğru ızgara kavşak karakterini seçer.
func getIntersectionChar(up, down, left, right bool) rune {
	if up && down && left && right {
		return '┼'
	}
	if !up && down && left && right {
		return '┬'
	}
	if up && !down && left && right {
		return '┴'
	}
	if up && down && !left && right {
		return '├'
	}
	if up && down && left && !right {
		return '┤'
	}
	if up && down {
		return '│'
	}
	if left && right {
		return '─'
	}
	if !up && down && !left && right {
		return '┌'
	}
	if !up && down && left && !right {
		return '┐'
	}
	if up && !down && !left && right {
		return '└'
	}
	if up && !down && left && !right {
		return '┘'
	}
	if left {
		return '─'
	}
	if right {
		return '─'
	}
	if up {
		return '│'
	}
	if down {
		return '│'
	}
	return ' '
}
