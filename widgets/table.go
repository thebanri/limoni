package widgets

import (
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/layout"
)

// ConstraintType is the kind of rule that sets a column's width.
type ConstraintType int

const (
	ConstraintFixed      ConstraintType = iota // Fixed column width (in characters)
	ConstraintPercentage                       // Percentage column width (% of the total width)
	ConstraintFill                             // Fills the space left over by the other columns
)

// TableConstraint is a column's width rule.
type TableConstraint struct {
	Type  ConstraintType
	Value int
}

// TableCell is the text and style of a single table cell.
type TableCell struct {
	Text    string
	Style   cell.Style
	ColSpan int // Number of columns to span (0 or 1 means a single column)
	RowSpan int // Number of rows to span (0 or 1 means a single row)
}

// TableRow is a table row's list of cells and its style.
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

// NewRow returns a TableRow with the standard style from the given list of strings.
func NewRow(cells ...string) TableRow {
	rowCells := make([]TableCell, len(cells))
	for i, c := range cells {
		rowCells[i] = TableCell{Text: c}
	}
	return TableRow{Cells: rowCells}
}

// TableState manages a table's row selection, vertical scrolling and column widths.
type TableState struct {
	Selected         int      // Selected row index (-1 means no selection)
	Offset           int      // Vertical scroll offset
	HorizontalOffset int      // Column cell offset, for horizontal scrolling
	ColumnWidths     []uint16 // Column widths, resized by dragging or solved automatically
	SortColumn       int      // The sorted column; -1 means sorting is off
	SortDescending   bool
	SelectedRows     map[int]struct{} // Multiple row selection
	selectionDirty   bool             // Set when the selection changed and the view needs adjusting to keep it visible.

	rowsHandler    func(driver.MouseEvent)
	scrollHandler  func(driver.MouseEvent)
	lastStartY     uint16
	lastDrawOffset int
	lastTotalRows  int
	lastRowCount   int
	lastViewportH  int
	lastTableID    string
	lastFocusFn    func(string)

	// Column resizing and sorting: one handler per column, built once, and
	// what they need from the last frame.
	resizeHandlers []func(driver.MouseEvent)
	resizeDrag     func(driver.MouseEvent)
	resizeCol      int
	resizeStartX   int
	resizeStartW   int
	lastCapture    func(func(driver.MouseEvent))
	sortHandlers   []func()
	lastRows       []TableRow

	// The sorted column's title with its arrow, made once per change.
	sortTitle, sortTitleFrom string
	sortTitleDesc            bool

	// rowNodes and cellNodes hold the visible rows' semantic nodes, written
	// during Draw (the only place that knows which data row, after filtering
	// and sorting, lands on which screen row) and reused every frame.
	rowNodes  []accessibility.AccessibilityNode
	cellNodes []accessibility.AccessibilityNode

	scratch *tableDrawScratch // Draw's working buffers, kept across frames
}

func (ts *TableState) initHandlers() {
	if ts.rowsHandler == nil {
		ts.rowsHandler = func(ev driver.MouseEvent) {
			if ev.Button == driver.MouseScrollUp || ev.Button == driver.MouseScrollDown {
				ts.handleScroll(ev, ts.lastRowCount, ts.lastViewportH)
				return
			}
			if ev.Button != driver.MouseLeft || ev.Y < ts.lastStartY {
				return
			}
			targetIdx := ts.lastDrawOffset + int(ev.Y-ts.lastStartY)
			if targetIdx < 0 || targetIdx >= ts.lastTotalRows {
				return
			}
			ts.Select(targetIdx)
			if ts.lastTableID != "" && ts.lastFocusFn != nil {
				ts.lastFocusFn(ts.lastTableID)
			}
		}
	}
	if ts.scrollHandler == nil {
		ts.scrollHandler = func(ev driver.MouseEvent) {
			ts.handleScroll(ev, ts.lastRowCount, ts.lastViewportH)
		}
	}
}

func (ts *TableState) handleScroll(ev driver.MouseEvent, rowCount, viewportHeight int) {
	switch ev.Button {
	case driver.MouseScrollUp:
		if ev.Shift {
			ts.ScrollHorizontal(-2)
		} else {
			ts.Scroll(-3, rowCount, viewportHeight)
		}
	case driver.MouseScrollDown:
		if ev.Shift {
			ts.ScrollHorizontal(2)
		} else {
			ts.Scroll(3, rowCount, viewportHeight)
		}
	}
}

type tableDrawScratch struct {
	constraints []TableConstraint // the even split used when none are given
	widths      []uint16
	owner       map[[2]int][2]int
	cells       map[[2]int]TableCell
	filtered    []TableRow
}

var tableDrawScratchPool = sync.Pool{
	New: func() any {
		return &tableDrawScratch{}
	},
}

// NewTableState returns a new TableState.
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

// Select selects the given row.
func (ts *TableState) Select(index int) {
	ts.Selected = index
	ts.selectionDirty = true
}

// Next moves the selection to the next row.
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

// Prev moves the selection to the previous row.
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

// Table is an interactive table with flexible columns, vertical scrolling and cell spanning.
type Table struct {
	ID            string
	Header        *TableRow
	Rows          []TableRow
	DataSource    TableDataSource
	Constraints   []TableConstraint
	State         *TableState
	GridStyle     cell.Style
	SelectedStyle cell.Style
	FocusedStyle  cell.Style
	DrawGrid      bool
	SortEnabled   bool                                              // Enables sorting rows by clicking the header cells.
	MultiSelect   bool                                              // Enables selecting several rows with Space.
	FilterQuery   string                                            // Fuzzy filter query; when empty, every row is drawn.
	CellStyle     func(row, column int, value TableCell) cell.Style // Per-cell style rule.
	StickyColumns int                                               // Number of columns frozen on the left.
	Scrollbar     bool                                              // Draws a vertical scrollbar on the right edge.
}

// NewTable creates a new Table with default grid enabled.
func NewTable() *Table {
	return &Table{
		DrawGrid: true,
	}
}

// WithID sets the table's focus ID.
func (t *Table) WithID(id string) *Table {
	t.ID = id
	return t
}

// WithHeaders sets the table headers from string slices.
func (t *Table) WithHeaders(headers ...string) *Table {
	row := NewRow(headers...)
	t.Header = &row
	return t
}

// WithRow appends a single row to the table.
func (t *Table) WithRow(cells ...string) *Table {
	t.Rows = append(t.Rows, NewRow(cells...))
	return t
}

// WithRows sets the rows of the table.
func (t *Table) WithRows(rows ...TableRow) *Table {
	t.Rows = rows
	return t
}

// WithConstraints sets column width constraints.
func (t *Table) WithConstraints(constraints ...TableConstraint) *Table {
	t.Constraints = constraints
	return t
}

// WithDrawGrid enables or disables grid lines.
func (t *Table) WithDrawGrid(drawGrid bool) *Table {
	t.DrawGrid = drawGrid
	return t
}

// WithState sets the TableState.
func (t *Table) WithState(state *TableState) *Table {
	t.State = state
	return t
}

// WithSelectedStyle sets the style for the selected row.
func (t *Table) WithSelectedStyle(style cell.Style) *Table {
	t.SelectedStyle = style
	return t
}

// WithGridStyle sets the style for grid lines.
func (t *Table) WithGridStyle(style cell.Style) *Table {
	t.GridStyle = style
	return t
}

// WithStickyColumns sets the number of sticky frozen columns.
func (t *Table) WithStickyColumns(n int) *Table {
	t.StickyColumns = n
	return t
}

// WithScrollbar enables or disables the vertical scrollbar.
func (t *Table) WithScrollbar(enabled bool) *Table {
	t.Scrollbar = enabled
	return t
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

// SolveWidths solves the column rules against the total usable table width and sets the widths.
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

	// Pass 1: solve the Fixed and Percentage columns
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

	// Pass 2: share the remaining space among the Fill columns
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

// Draw renders the table: it writes the header, lays out the rows by the scroll offset and draws the grid lines.
func (t Table) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	colCount := len(t.Constraints)
	if colCount == 0 {
		if t.Header != nil && len(t.Header.Cells) > 0 {
			colCount = len(t.Header.Cells)
		} else if len(t.Rows) > 0 {
			colCount = len(t.Rows[0].Cells)
		}
		if colCount == 0 {
			return
		}
	}
	// A table with State keeps its scratch buffers there. A sync.Pool is
	// emptied by every garbage collection, so drawing from one allocated a
	// fresh scratch and two maps after each GC.
	var scratch *tableDrawScratch
	if t.State != nil {
		if t.State.scratch == nil {
			t.State.scratch = &tableDrawScratch{}
		}
		scratch = t.State.scratch
	} else {
		scratch = tableDrawScratchPool.Get().(*tableDrawScratch)
		defer tableDrawScratchPool.Put(scratch)
	}
	if len(t.Constraints) == 0 {
		// An even split, kept in the scratch so it is not rebuilt each frame.
		if cap(scratch.constraints) < colCount {
			scratch.constraints = make([]TableConstraint, colCount)
		}
		t.Constraints = scratch.constraints[:colCount]
		pct := 100 / colCount
		for i := range t.Constraints {
			t.Constraints[i] = TableConstraint{Type: ConstraintPercentage, Value: pct}
		}
	}
	if scratch.owner == nil {
		scratch.owner = make(map[[2]int][2]int)
	}
	if scratch.cells == nil {
		scratch.cells = make(map[[2]int]TableCell)
	}
	if t.State != nil && t.SortEnabled && t.State.SortColumn >= 0 && t.DataSource == nil {
		sortTableRows(t.Rows, t.State.SortColumn, t.State.SortDescending)
	}

	// Register as focusable
	if t.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(t.ID)
	}

	// 1. WORK OUT THE ROW COUNT AND THE FILTER
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

	// 2. WORK OUT WHETHER A SCROLLBAR IS NEEDED AND THE AREA WIDTH
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

	// 3. WORK OUT AND INITIALISE THE COLUMN WIDTHS
	colsCount := len(t.Constraints)
	netWidth := ctx.Area.Width
	// If grid lines are drawn, subtract 1 character between each pair of columns
	if t.DrawGrid && colsCount > 1 {
		if netWidth > uint16(colsCount-1) {
			netWidth -= uint16(colsCount - 1)
		} else {
			netWidth = 1
		}
	}

	var widths []uint16
	if t.State != nil {
		// Keep the column widths, and solve them again if the screen size changed
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

	if ctx.RegisterMouse != nil && t.State != nil {
		t.registerScrollHandlers(ctx, rowCount)
		t.registerResizeHandlers(ctx, widths, colsCount, sticky, stickyWidth)
	}
	if ctx.RegisterClick != nil && t.SortEnabled && t.Header != nil && t.State != nil {
		t.registerSortHandlers(ctx, widths, colsCount)
	}

	// 3. WORK OUT THE OWNERSHIP MATRIX (COLSPAN / ROWSPAN)
	owner := scratch.owner
	clear(owner)
	cellsMap := scratch.cells
	clear(cellsMap)

	// Put the header row (row -1) into the matrix
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
				cVal.Text = t.State.sortedHeader(cVal.Text)
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

	// Put the body rows into the matrix (skipping rows out of view)
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

	// 4. DRAW THE HEADER
	currY := ctx.Area.Y
	gridStyle := ctx.Style.Merge(t.GridStyle)

	if t.Header != nil {
		t.drawSpanRow(ctx, buf, currY, -1, widths, false, owner, cellsMap, gridStyle, t.Header.Style)
		currY++

		// Separator line under the header
		if currY < ctx.Area.Y+ctx.Area.Height {
			targetBodyRow := 0
			if t.State != nil {
				targetBodyRow = t.State.Offset
			}

			for i, w := range widths {
				// Check whether a spanned cell covers the horizontal line
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
					// Pick the junction character where vertical and horizontal lines meet
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

	// 5. ROW SCROLL CALCULATIONS
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

	// 6. DRAW THE ROWS
	drawOffset := 0
	if t.State != nil {
		drawOffset = t.State.Offset
	}
	rowsStartY := currY
	drawnRows := uint16(0)
	// Registering a closure per row would allocate on the heap for every visible
	// row. Instead the whole block of rows is registered as one mouse region, and
	// the target row index is worked out from the event's coordinates.
	perRowClick := ctx.RegisterMouse == nil && ctx.RegisterClick != nil
	if t.State != nil {
		t.State.rowNodes = t.State.rowNodes[:0]
		// Sized before the loop: rows keep sub-slices of cellNodes, which a
		// later append must not move.
		if need := visibleRows * len(widths); cap(t.State.cellNodes) < need {
			t.State.cellNodes = make([]accessibility.AccessibilityNode, 0, need)
		}
		t.State.cellNodes = t.State.cellNodes[:0]
	}
	for rIdx := 0; rIdx < visibleRows; rIdx++ {
		offset := drawOffset
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

		if perRowClick {
			t.registerRowClickHandler(ctx, cell.NewRect(ctx.Area.X, currY, ctx.Area.Width, 1), actualRowIdx)
		}

		t.drawSpanRow(ctx, buf, currY, actualRowIdx, widths, isSelected, owner, cellsMap, gridStyle, row.Style)
		if t.State != nil {
			t.State.appendRowNode(t, ctx.Area, row, currY, actualRowIdx, rowCount, widths, isSelected)
		}
		currY++
		drawnRows++
	}

	if !perRowClick && drawnRows > 0 && ctx.RegisterMouse != nil && (t.State != nil || (t.ID != "" && ctx.SetFocus != nil)) {
		t.registerRowsBlockHandler(ctx, cell.NewRect(ctx.Area.X, rowsStartY, ctx.Area.Width, drawnRows), rowsStartY, drawOffset, totalRows, rowCount)
	}

	// Draw the scrollbar
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
				c.Style = c.Style.Merge(gridStyle)
				if y >= thumbY && y < thumbY+thumbH {
					c.Content = '█'
					c.Style = c.Style.Merge(thumbStyle)
				}
			}
		}
	}
}

func (t Table) registerScrollHandlers(ctx cell.Context, rowCount int) {
	if t.State == nil || ctx.RegisterMouse == nil {
		return
	}
	t.State.initHandlers()
	t.State.lastRowCount = rowCount
	t.State.lastViewportH = int(ctx.Area.Height)
	ctx.RegisterMouse(ctx.Area, t.State.scrollHandler)
}

// applyScroll turns mouse wheel events into vertical/horizontal scrolling.
func (t Table) applyScroll(ev driver.MouseEvent, rowCount, viewportHeight int) {
	if t.State == nil {
		return
	}
	t.State.handleScroll(ev, rowCount, viewportHeight)
}

// registerRowsBlockHandler registers all the visible rows as a single mouse region.
// The target row is worked out from the event's Y coordinate and the scroll position at draw time,
// which removes the closure allocations that grew with the number of rows.
func (t Table) registerRowsBlockHandler(ctx cell.Context, rowsArea cell.Rect, rowsStartY uint16, drawOffset, totalRows, rowCount int) {
	if ctx.RegisterMouse == nil {
		return
	}
	if t.State == nil && (t.ID == "" || ctx.SetFocus == nil) {
		return
	}
	if t.State != nil {
		t.State.initHandlers()
		t.State.lastStartY = rowsStartY
		t.State.lastDrawOffset = drawOffset
		t.State.lastTotalRows = totalRows
		t.State.lastRowCount = rowCount
		t.State.lastViewportH = int(ctx.Area.Height)
		t.State.lastTableID = t.ID
		t.State.lastFocusFn = ctx.SetFocus
		ctx.RegisterMouse(rowsArea, t.State.rowsHandler)
		return
	}
	// Copies, so the closure does not capture t: a Table is larger than a
	// closure captures by value, so capturing it moved t to the heap on
	// every call, State or not.
	id, setFocus := t.ID, ctx.SetFocus
	ctx.RegisterMouse(rowsArea, func(ev driver.MouseEvent) {
		if ev.Button == driver.MouseLeft {
			setFocus(id)
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
			t.State.lastCapture = ctx.CaptureMouse
			ctx.RegisterMouse(handleArea, t.State.resizeHandler(i))
		}
	}
}

// resizeHandler is the handler for the divider after column col, built once
// per state, so registering it each frame does not allocate.
func (ts *TableState) resizeHandler(col int) func(driver.MouseEvent) {
	for len(ts.resizeHandlers) <= col {
		c := len(ts.resizeHandlers)
		ts.resizeHandlers = append(ts.resizeHandlers, func(ev driver.MouseEvent) { ts.startResize(c, ev) })
	}
	return ts.resizeHandlers[col]
}

func (ts *TableState) startResize(col int, ev driver.MouseEvent) {
	if ev.Button != driver.MouseLeft || ev.Drag || ts.lastCapture == nil || col >= len(ts.ColumnWidths) {
		return
	}
	ts.resizeCol, ts.resizeStartX, ts.resizeStartW = col, int(ev.X), int(ts.ColumnWidths[col])
	if ts.resizeDrag == nil {
		ts.resizeDrag = func(ev driver.MouseEvent) {
			if ev.Button == driver.MouseRelease || ts.resizeCol >= len(ts.ColumnWidths) {
				return
			}
			width := ts.resizeStartW + int(ev.X) - ts.resizeStartX
			if width < 2 {
				width = 2
			}
			ts.ResizeColumn(ts.resizeCol, width-int(ts.ColumnWidths[ts.resizeCol]))
		}
	}
	ts.lastCapture(ts.resizeDrag)
}

// sortedHeader is title with the sort arrow after it, built when the title or
// the direction changes rather than on every frame.
func (ts *TableState) sortedHeader(title string) string {
	if ts.sortTitle == "" || ts.sortTitleFrom != title || ts.sortTitleDesc != ts.SortDescending {
		indicator := " ▲"
		if ts.SortDescending {
			indicator = " ▼"
		}
		ts.sortTitle, ts.sortTitleFrom, ts.sortTitleDesc = title+indicator, title, ts.SortDescending
	}
	return ts.sortTitle
}

// sortHandler is the click handler for column col's header, built once per
// state; it sorts the rows the last frame drew.
func (ts *TableState) sortHandler(col int) func() {
	for len(ts.sortHandlers) <= col {
		c := len(ts.sortHandlers)
		ts.sortHandlers = append(ts.sortHandlers, func() {
			if ts.SortColumn == c {
				ts.SortDescending = !ts.SortDescending
			} else {
				ts.SortColumn = c
				ts.SortDescending = false
			}
			sortTableRows(ts.lastRows, c, ts.SortDescending)
		})
	}
	return ts.sortHandlers[col]
}

func (t Table) registerSortHandlers(ctx cell.Context, widths []uint16, colsCount int) {
	if !t.SortEnabled || t.Header == nil || ctx.RegisterClick == nil || t.State == nil {
		return
	}
	t.State.lastRows = t.Rows
	for colIdx, width := range widths {
		currX := t.columnX(ctx.Area, widths, colIdx)
		clickWidth := width
		if t.DrawGrid && colIdx < colsCount-1 && clickWidth > 0 {
			clickWidth--
		}
		if clickWidth > 0 {
			ctx.RegisterClick(cell.NewRect(currX, ctx.Area.Y, clickWidth, 1), t.State.sortHandler(colIdx))
		}
	}
}

func (t Table) registerRowClickHandler(ctx cell.Context, rowArea cell.Rect, targetIdx int) {
	if ctx.RegisterClick == nil {
		return
	}
	state, id, setFocus := t.State, t.ID, ctx.SetFocus
	ctx.RegisterClick(rowArea, func() {
		if state != nil {
			state.Select(targetIdx)
		}
		if id != "" && setFocus != nil {
			setFocus(id)
		}
	})
}

// drawSpanRow draws a single table row, taking spanned cells into account.
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
		if ctx.IsFocused(t.ID) {
			rowStyle = rowStyle.Merge(t.FocusedStyle)
		}
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

		// Skip drawing if this cell is part of a spanned cell above or to the left
		if ownerCoords != [2]int{r, colIdx} {
			if t.DrawGrid && colIdx < colsCount-1 {
				currX = t.columnX(ctx.Area, widths, colIdx+1)
				// Draw the border line unless it lies inside a spanned area
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

		// This cell is the start (owner) cell of a spanned area
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

		// Work out the spanned cell's total width in characters (neighbouring columns + the grid lines between them)
		cellW := uint16(0)
		for c := 0; c < colSpan && colIdx+c < colsCount; c++ {
			cellW += widths[colIdx+c]
			if t.DrawGrid && c > 0 {
				cellW++
			}
		}

		// Fill the cell background (across rowSpan rows and cellW columns)
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

		// Cut the text and write it on the first row only (top-left) - clipping-aware.
		// Cut at a cluster boundary and draw the "..." separately, so a cell
		// that does not fit costs no allocation.
		if text := cellVal.Text; cell.StringWidth(text) <= int(cellW) {
			drawTextClipped(buf, currX, y, text, cellStyle, clipLeft, clipRight)
		} else if cellW <= 3 {
			prefix, _ := cell.Truncate(text, int(cellW))
			drawTextClipped(buf, currX, y, prefix, cellStyle, clipLeft, clipRight)
		} else {
			prefix, w := cell.Truncate(text, int(cellW)-3)
			drawTextClipped(buf, currX, y, prefix, cellStyle, clipLeft, clipRight)
			drawTextClipped(buf, currX+uint16(w), y, "...", cellStyle, clipLeft, clipRight)
		}

		// Draw the vertical grid line between columns (if outside the spanned area)
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
func drawTextClipped(buf *buffer.Buffer, startX, y uint16, s string, style cell.Style, clipLeft, clipRight uint16) uint16 {
	if y >= buf.Area.Height || startX >= clipRight {
		return 0
	}
	if clipRight > buf.Area.Width {
		clipRight = buf.Area.Width
	}

	// One grapheme cluster per cell, as Buffer.SetString does: walking runes
	// dropped combining accents and split flags and emoji sequences.
	currX := startX
	for input := s; input != ""; {
		cluster, w, rest := cell.NextCluster(input)
		input = rest
		if w == 0 {
			continue
		}
		if currX+uint16(w) > clipRight {
			break
		}
		// Only cells inside the horizontal clipping range are written.
		if currX >= clipLeft {
			idx := y*buf.Area.Width + currX
			buf.Invalidate()
			buf.Content[idx].Content = cell.ClusterContent(cluster, w)
			buf.Content[idx].Style = style
			if w == 2 && currX+1 < clipRight {
				buf.Content[idx+1].Content = cell.RuneContinuation
				buf.Content[idx+1].Style = style
			}
		}
		currX += uint16(w)
	}
	return currX - startX
}

// sortTableRows sorts rows by column, stably. A sorted table is sorted again
// every frame, so rows already in order are left alone after one pass, and
// neither path allocates: sort.SliceStable built a reflection swapper and
// strings.ToLower a copy of every compared cell, 20 allocations a frame for a
// three-row table.
func sortTableRows(rows []TableRow, column int, descending bool) {
	compare := func(a, b TableRow) int {
		left, right := "", ""
		if column >= 0 && column < len(a.Cells) {
			left = a.Cells[column].Text
		}
		if column >= 0 && column < len(b.Cells) {
			right = b.Cells[column].Text
		}
		if descending {
			return compareTableValues(right, left)
		}
		return compareTableValues(left, right)
	}
	if slices.IsSortedFunc(rows, compare) {
		return
	}
	slices.SortStableFunc(rows, compare)
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
	return compareFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

// compareFold orders a and b as their lower-case forms would order, without
// making them.
func compareFold(a, b string) int {
	for a != "" && b != "" {
		ra, na := utf8.DecodeRuneInString(a)
		rb, nb := utf8.DecodeRuneInString(b)
		if la, lb := unicode.ToLower(ra), unicode.ToLower(rb); la != lb {
			if la < lb {
				return -1
			}
			return 1
		}
		a, b = a[na:], b[nb:]
	}
	switch {
	case a == b:
		return 0
	case a == "":
		return -1
	}
	return 1
}

// numericTableValue reads the number a cell starts with ("42", "3.5 ms",
// "80%"). Text with no digit is not offered to ParseFloat, whose error is an
// allocation, unless it could be one of the words ParseFloat accepts.
func numericTableValue(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if end := strings.IndexFunc(value, unicode.IsSpace); end >= 0 {
		value = value[:end]
	}
	number := strings.TrimSuffix(value, "%")
	if number == "" || !strings.ContainsAny(number, "0123456789") && !isFloatWord(number) {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(number, 64)
	return parsed, err == nil
}

// isFloatWord reports whether s is a spelling of infinity or NaN that
// strconv.ParseFloat accepts.
func isFloatWord(s string) bool {
	s = strings.TrimLeft(s, "+-")
	return strings.EqualFold(s, "inf") || strings.EqualFold(s, "infinity") || strings.EqualFold(s, "nan")
}

// SizeHint reports the table's flexible layout needs.
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

// clipString fits s into maxW columns, ending it in "..." when it had to be
// cut. It cuts at grapheme cluster boundaries and measures clusters, not code
// points. Drawing code should prefer setClipped, which does the same without
// building a new string.
func clipString(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if cell.StringWidth(s) <= maxW {
		return s
	}
	if maxW <= 3 {
		prefix, _ := cell.Truncate(s, maxW)
		return prefix
	}
	prefix, _ := cell.Truncate(s, maxW-3)
	return prefix + "..."
}

// setClipped draws s at (x, y) the way clipString would cut it, without
// allocating: the kept prefix and the "..." are written separately.
func setClipped(buf *buffer.Buffer, x, y uint16, s string, style cell.Style, maxW int) uint16 {
	return setEllipsized(buf, x, y, s, style, maxW, "...")
}

// setEllipsized draws s within maxW columns. If it does not fit, it is cut at a
// grapheme cluster boundary and followed by suffix. The suffix is dropped when
// there is no room for it and at least one column of text.
func setEllipsized(buf *buffer.Buffer, x, y uint16, s string, style cell.Style, maxW int, suffix string) uint16 {
	if maxW <= 0 {
		return 0
	}
	if cell.StringWidth(s) <= maxW {
		return buf.SetStringWithin(x, y, s, style, uint16(maxW))
	}
	sw := cell.StringWidth(suffix)
	if maxW <= sw {
		prefix, w := cell.Truncate(s, maxW)
		return buf.SetStringWithin(x, y, prefix, style, uint16(w))
	}
	prefix, w := cell.Truncate(s, maxW-sw)
	n := buf.SetStringWithin(x, y, prefix, style, uint16(w))
	return n + buf.SetStringWithin(x+n, y, suffix, style, uint16(sw))
}

// getIntersectionChar picks the right grid junction character from the lines around it.
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

// appendRowNode records the semantic node for one drawn row, with a cell
// child per column. Nothing is allocated once the buffers have grown to the
// table's visible size.
func (ts *TableState) appendRowNode(t Table, area cell.Rect, row TableRow, y uint16, index, count int, widths []uint16, selected bool) {
	start := len(ts.cellNodes)
	for c := 0; c < len(widths) && len(ts.cellNodes) < cap(ts.cellNodes); c++ {
		text := ""
		if c < len(row.Cells) {
			text = row.Cells[c].Text
		}
		ts.cellNodes = append(ts.cellNodes, accessibility.AccessibilityNode{
			Role:   accessibility.RoleCell,
			Label:  text,
			Bounds: cell.Rect{X: t.columnX(area, widths, c), Y: y, Width: widths[c], Height: 1},
		})
	}
	label := ""
	if len(row.Cells) > 0 {
		label = row.Cells[0].Text
	}
	state := accessibility.NodeState(0)
	if selected {
		state = accessibility.StateSelected
	}
	ts.rowNodes = append(ts.rowNodes, accessibility.AccessibilityNode{
		Role:     accessibility.RoleRow,
		Label:    label,
		State:    state,
		Bounds:   cell.Rect{X: area.X, Y: y, Width: area.Width, Height: 1},
		Position: index + 1,
		SetSize:  count,
		Children: ts.cellNodes[start:len(ts.cellNodes):len(ts.cellNodes)],
	})
}

// AccessibilityNode returns the semantic node description for Table.
//
// The node carries the selected row's first cell as its value and one row
// child per visible row, each labelled by its first cell and holding a cell
// child per column. A table of a million rows exposes the ones on screen.
// The rows come from the last Draw, which knows how filtering and sorting
// placed them; a Table without State stays flat.
func (t Table) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}

	count := len(t.Rows)
	if t.DataSource != nil {
		count = t.DataSource.RowCount()
	}

	selected, position, value := -1, 0, ""
	if t.State != nil {
		selected = t.State.Selected
	}
	if selected >= 0 && selected < len(t.Rows) {
		state |= accessibility.StateSelected
		position = selected + 1
		if cells := t.Rows[selected].Cells; len(cells) > 0 {
			value = cells[0].Text
		}
	} else if selected >= 0 && selected < count {
		state |= accessibility.StateSelected
		position = selected + 1
	}

	return accessibility.AccessibilityNode{
		ID:       t.ID,
		Role:     accessibility.RoleTable,
		Label:    "Table",
		Value:    value,
		State:    state,
		Bounds:   bounds,
		Position: position,
		SetSize:  count,
		Children: t.rowNodes(),
	}
}

func (t Table) rowNodes() []accessibility.AccessibilityNode {
	if t.State == nil {
		return nil
	}
	return t.State.rowNodes
}
