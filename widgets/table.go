package widgets

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/backend"
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
	Selected       int      // Seçili satır indeksi (-1 ise seçim yok)
	Offset         int      // Dikey kaydırma (scroll offset) miktarı
	ColumnWidths   []uint16 // Sürüklenerek yeniden boyutlandırılan veya otomatik çözülen sütun genişlikleri
	SortColumn     int      // Sıralanan sütun; -1 ise sıralama kapalı
	SortDescending bool
}

// NewTableState, yeni bir TableState nesnesi oluşturur.
func NewTableState() *TableState {
	return &TableState{
		Selected:       -1,
		Offset:         0,
		ColumnWidths:   nil,
		SortColumn:     -1,
		SortDescending: false,
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

// Table, interaktif, esnek sütunlu, dikey kaydırılabilir ve hücre birleştirme destekli tablo bileşenidir.
type Table struct {
	ID            string
	Header        *TableRow
	Rows          []TableRow
	Constraints   []TableConstraint
	State         *TableState
	GridStyle     cell.Style
	SelectedStyle cell.Style
	DrawGrid      bool
	SortEnabled   bool   // Başlık hücrelerine tıklayarak satır sıralamayı etkinleştirir.
	FilterQuery   string // Fuzzy filtre sorgusu; boşsa tüm satırlar çizilir.
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
	if t.State != nil && t.SortEnabled && t.State.SortColumn >= 0 {
		sortTableRows(t.Rows, t.State.SortColumn, t.State.SortDescending)
	}

	// Odaklanabilir olarak kaydet
	if t.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(t.ID)
	}

	// 1. SÜTUN GENİŞLİKLERİNİN HESAPLANMASI VE İLKLENDİRİLMESİ
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
			t.State.ColumnWidths = SolveWidths(netWidth, t.Constraints)
		}
		widths = t.State.ColumnWidths
	} else {
		widths = SolveWidths(netWidth, t.Constraints)
	}

	// 2. İNTERAKTİF SÜTUN BOYUTLANDIRICI SÜRÜKLEME ALANLARI
	if t.State != nil && ctx.RegisterMouse != nil {
		sepX := ctx.Area.X
		for i := 0; i < colsCount-1; i++ {
			sepX += widths[i]

			// Her sütun sınırı için 1 genişliğinde dikey bir sürükleme tetikleyici alan tanımla
			handleArea := cell.NewRect(sepX, ctx.Area.Y, 1, ctx.Area.Height)
			colIdx := i

			ctx.RegisterMouse(handleArea, func(ev backend.MouseEvent) {
				if ev.Button == backend.MouseLeft && !ev.Drag {
					// Sürükleme başlangıcı: Mevcut X koordinatı ve tüm sütun genişliklerini yakala
					startMouseX := int(ev.X)
					startColW := int(t.State.ColumnWidths[colIdx])

					ctx.CaptureMouse(func(dragEv backend.MouseEvent) {
						if dragEv.Button == backend.MouseRelease {
							return
						}
						// Fare hareketi farkına göre yeni genişliği hesapla
						dx := int(dragEv.X) - startMouseX
						requestedNewW := startColW + dx
						if requestedNewW < 2 {
							requestedNewW = 2
						}

						// ResizeColumn toplam genişliği koruyarak sağdaki sütunları
						// gerektiğinde daraltır; drag döngüsünde tahsisat yapmaz.
						delta := requestedNewW - int(t.State.ColumnWidths[colIdx])
						t.State.ResizeColumn(colIdx, delta)
					})
				}
			})

			if t.DrawGrid {
				sepX++
			}
		}
	}

	rows := t.Rows
	if t.FilterQuery != "" {
		rows = FuzzyFilterByStable(t.FilterQuery, rows, func(row TableRow) string { return row.SearchText() })
	}

	// 3. SAHİPLİK MATRİSİNİN (COLSPAN / ROWSPAN) HESAPLANMASI
	owner := make(map[[2]int][2]int)
	cellsMap := make(map[[2]int]TableCell)

	getOwner := func(r, c int) [2]int {
		if val, exists := owner[[2]int{r, c}]; exists {
			return val
		}
		return [2]int{r, c}
	}

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

	// Body satırlarını (row 0..len(Rows)-1) matrise işle
	for rIdx := 0; rIdx < len(rows); rIdx++ {
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

	// Başlığa tıklama ile sütun sıralama kaydı.
	if t.SortEnabled && t.Header != nil && ctx.RegisterClick != nil {
		currX := ctx.Area.X
		for colIdx, width := range widths {
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
			currX += width
			if t.DrawGrid && colIdx < colsCount-1 {
				currX++
			}
		}
	}

	// 4. BAŞLIK ÇİZİMİ
	currY := ctx.Area.Y
	gridStyle := ctx.Style.Merge(t.GridStyle)

	if t.Header != nil {
		t.drawSpanRow(ctx, buf, currY, -1, widths, false, getOwner, cellsMap, gridStyle, t.Header.Style)
		currY++

		// Başlık altı ayırıcı çizgi
		if currY < ctx.Area.Y+ctx.Area.Height {
			targetBodyRow := 0
			if t.State != nil {
				targetBodyRow = t.State.Offset
			}

			currX := ctx.Area.X
			for i, w := range widths {
				// Yatay çizginin birleştirilmiş hücre tarafından örtülüp örtülmediğini denetle
				sepCovered := getOwner(-1, i) == getOwner(targetBodyRow, i)

				for col := uint16(0); col < w; col++ {
					if !sepCovered {
						buf.SetCell(currX, currY, cell.Cell{Content: '─', Style: gridStyle})
					}
					currX++
				}

				if t.DrawGrid && i < colsCount-1 {
					// Dikey ve yatay çizgilerin birleştiği kesişim karakterini seç
					up := getOwner(-1, i) != getOwner(-1, i+1)
					down := getOwner(targetBodyRow, i) != getOwner(targetBodyRow, i+1)
					left := getOwner(-1, i) != getOwner(targetBodyRow, i)
					right := getOwner(-1, i+1) != getOwner(targetBodyRow, i+1)

					ch := getIntersectionChar(up, down, left, right)
					if ch != ' ' {
						buf.SetCell(currX, currY, cell.Cell{Content: ch, Style: gridStyle})
					}
					currX++
				}
			}
			currY++
		}
	}

	// 5. SATIR SCROLL HESAPLAMALARI
	if currY >= ctx.Area.Y+ctx.Area.Height {
		return
	}
	visibleRows := int(ctx.Area.Y + ctx.Area.Height - currY)
	if visibleRows <= 0 {
		return
	}

	totalRows := len(rows)
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
		if t.State.Selected != -1 && t.State.Selected < t.State.Offset {
			t.State.Offset = t.State.Selected
		}
		if t.State.Selected != -1 && t.State.Selected >= t.State.Offset+visibleRows {
			t.State.Offset = t.State.Selected - visibleRows + 1
		}
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

		row := rows[actualRowIdx]
		isSelected := (t.State != nil && t.State.Selected == actualRowIdx)

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

		t.drawSpanRow(ctx, buf, currY, actualRowIdx, widths, isSelected, getOwner, cellsMap, gridStyle, row.Style)
		currY++
	}
}

// drawSpanRow, birleştirilmiş hücrelere duyarlı olarak tek bir tablo satırını çizdirir.
func (t Table) drawSpanRow(
	ctx cell.Context,
	buf *buffer.Buffer,
	y uint16,
	r int,
	widths []uint16,
	isSelected bool,
	getOwner func(r, c int) [2]int,
	cellsMap map[[2]int]TableCell,
	gridStyle cell.Style,
	baseRowStyle cell.Style,
) {
	currX := ctx.Area.X
	rowStyle := ctx.Style.Merge(baseRowStyle)
	if isSelected {
		rowStyle = rowStyle.Merge(t.SelectedStyle)
	}

	colsCount := len(widths)

	for colIdx := 0; colIdx < colsCount; colIdx++ {
		ownerCoords := getOwner(r, colIdx)

		// Eğer bu hücre üstteki veya soldaki birleştirilmiş bir hücrenin alt parçasıysa çizimi atla
		if ownerCoords != [2]int{r, colIdx} {
			currX += widths[colIdx]
			if t.DrawGrid && colIdx < colsCount-1 {
				// Sınır çizgisi hücre birleştirme alanı içinde kalmıyorsa çiz
				if getOwner(r, colIdx) != getOwner(r, colIdx+1) {
					buf.SetCell(currX, y, cell.Cell{Content: '│', Style: gridStyle})
				}
				currX++
			}
			continue
		}

		// Bu hücre birleştirilmiş alanın başlangıç (ana) hücresidir
		cellVal := cellsMap[[2]int{r, colIdx}]
		cellStyle := rowStyle.Merge(cellVal.Style)

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
				if currX+dx < ctx.Area.X+ctx.Area.Width {
					buf.SetCell(currX+dx, drawY, cell.Cell{Content: ' ', Style: cellStyle})
				}
			}
		}

		// Metni keserek sadece ilk satıra yazdır (top-left)
		clipped := clipString(cellVal.Text, int(cellW))
		buf.SetString(currX, y, clipped, cellStyle)

		currX += widths[colIdx]

		// Sütunlar arası dikey ızgara çizgisini çiz (birleştirilmiş alanın dışındaysa)
		if t.DrawGrid && colIdx < colsCount-1 {
			if getOwner(r, colIdx) != getOwner(r, colIdx+1) {
				buf.SetCell(currX, y, cell.Cell{Content: '│', Style: gridStyle})
			}
			currX++
		}
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
