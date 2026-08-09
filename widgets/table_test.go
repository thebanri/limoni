package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestTableWidthSolver(t *testing.T) {
	constraints := []TableConstraint{
		{Type: ConstraintFixed, Value: 10},
		{Type: ConstraintPercentage, Value: 50},
		{Type: ConstraintFill, Value: 0},
	}

	widths := SolveWidths(100, constraints)
	if len(widths) != 3 {
		t.Fatalf("SolveWidths length = %d; 3 bekleniyordu", len(widths))
	}

	// 100 genişliğinde:
	// Fixed 10 -> 10
	// Percentage 50 -> 50
	// Fill -> 100 - (10 + 50) = 40
	if widths[0] != 10 {
		t.Errorf("widths[0] = %d; 10 bekleniyordu", widths[0])
	}
	if widths[1] != 50 {
		t.Errorf("widths[1] = %d; 50 bekleniyordu", widths[1])
	}
	if widths[2] != 40 {
		t.Errorf("widths[2] = %d; 40 bekleniyordu", widths[2])
	}
}

func TestTableStateNavigation(t *testing.T) {
	state := NewTableState()
	if state.Selected != -1 {
		t.Errorf("NewTableState.Selected = %d; -1 bekleniyordu", state.Selected)
	}

	state.Next(5)
	if state.Selected != 0 {
		t.Errorf("Selected after Next() = %d; 0 bekleniyordu", state.Selected)
	}

	state.Next(5)
	if state.Selected != 1 {
		t.Errorf("Selected after second Next() = %d; 1 bekleniyordu", state.Selected)
	}

	state.Prev()
	if state.Selected != 0 {
		t.Errorf("Selected after Prev() = %d; 0 bekleniyordu", state.Selected)
	}

	// Sınır koruma (min index 0)
	state.Prev()
	if state.Selected != 0 {
		t.Errorf("Selected after Prev() boundary = %d; 0 bekleniyordu", state.Selected)
	}
}

func TestTableEmptyFilterDoesNotUnderflowOffset(t *testing.T) {
	state := NewTableState()
	state.Selected = 0
	table := Table{
		Rows:        []TableRow{NewRow("visible")},
		FilterQuery: "missing",
		Constraints: []TableConstraint{{Type: ConstraintFixed, Value: 10}},
		State:       state,
	}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 10, 3))
	table.Draw(cell.NewContext(buf.Area, cell.Style{}), buf)
	if state.Selected != -1 || state.Offset != 0 {
		t.Fatalf("empty filter state = selected:%d offset:%d; want -1, 0", state.Selected, state.Offset)
	}
}

func TestTableSorting(t *testing.T) {
	rows := []TableRow{NewRow("b", "Beta"), NewRow("a", "Alpha"), NewRow("c", "Gamma")}
	sortTableRows(rows, 0, false)
	if rows[0].Cells[0].Text != "a" || rows[2].Cells[0].Text != "c" {
		t.Fatalf("ascending rows = %q, %q; want a, c", rows[0].Cells[0].Text, rows[2].Cells[0].Text)
	}
	sortTableRows(rows, 1, true)
	if rows[0].Cells[1].Text != "Gamma" || rows[2].Cells[1].Text != "Alpha" {
		t.Fatalf("descending rows = %q, %q; want Gamma, Alpha", rows[0].Cells[1].Text, rows[2].Cells[1].Text)
	}
}

func TestTableNumericSorting(t *testing.T) {
	rows := []TableRow{NewRow("75.4 MB"), NewRow("105.4 MB"), NewRow("256.1 MB"), NewRow("14.2 MB")}
	sortTableRows(rows, 0, true)
	want := []string{"256.1 MB", "105.4 MB", "75.4 MB", "14.2 MB"}
	for i, row := range rows {
		if row.Cells[0].Text != want[i] {
			t.Fatalf("descending row %d = %q; want %q", i, row.Cells[0].Text, want[i])
		}
	}
}

func TestTableCellStyleCallback(t *testing.T) {
	table := Table{
		Rows:        []TableRow{NewRow("warning")},
		Constraints: []TableConstraint{{Type: ConstraintFixed, Value: 10}},
		CellStyle: func(row, column int, value TableCell) cell.Style {
			if row == 0 && column == 0 && value.Text == "warning" {
				return cell.Style{Fg: cell.NewColorRGB(255, 0, 0)}
			}
			return cell.Style{}
		},
	}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 10, 1))
	table.Draw(cell.NewContext(buf.Area, cell.Style{}), buf)
	if buf.Get(0, 0).Style.Fg != cell.NewColorRGB(255, 0, 0) {
		t.Fatal("cell style callback was not applied")
	}
}

func TestTableStateMultiSelection(t *testing.T) {
	state := NewTableState()
	state.ToggleRow(2)
	state.ToggleRow(4)
	if !state.IsRowSelected(2) || !state.IsRowSelected(4) {
		t.Fatal("rows 2 and 4 should be selected")
	}
	state.ToggleRow(2)
	if state.IsRowSelected(2) || !state.IsRowSelected(4) {
		t.Fatal("row 2 should be deselected while row 4 remains")
	}
	state.ClearSelectedRows()
	if state.IsRowSelected(4) {
		t.Fatal("clear should remove all selected rows")
	}
}

func TestTableStateMoveSortColumn(t *testing.T) {
	state := NewTableState()
	state.MoveSortColumn(1, 5)
	if state.SortColumn != 0 {
		t.Fatalf("initial sort column = %d; want 0", state.SortColumn)
	}
	state.MoveSortColumn(1, 5)
	if state.SortColumn != 1 {
		t.Fatalf("next sort column = %d; want 1", state.SortColumn)
	}
	state.MoveSortColumn(-1, 5)
	if state.SortColumn != 0 {
		t.Fatalf("previous sort column = %d; want 0", state.SortColumn)
	}
	state.MoveSortColumn(-1, 5)
	if state.SortColumn != 4 {
		t.Fatalf("wrapped sort column = %d; want 4", state.SortColumn)
	}
}

func TestTableStateResizeColumn(t *testing.T) {
	state := &TableState{ColumnWidths: []uint16{10, 10, 10}}
	if !state.ResizeColumn(0, 5) {
		t.Fatal("column resize should report a change")
	}
	if got := state.ColumnWidths; got[0] != 15 || got[1] != 10 || got[2] != 5 {
		t.Fatalf("after growing column: got %v, want [15 10 5]", got)
	}
	if !state.ResizeColumn(0, -7) {
		t.Fatal("column shrink should report a change")
	}
	if got := state.ColumnWidths; got[0] != 8 || got[1] != 10 || got[2] != 12 {
		t.Fatalf("after shrinking column: got %v, want [8 10 12]", got)
	}
}

func TestTableTextClipping(t *testing.T) {
	s1 := clipString("Hello World", 5)
	if s1 != "He..." {
		t.Errorf("clipString = %q; 'He...' bekleniyordu", s1)
	}

	s2 := clipString("Hi", 5)
	if s2 != "Hi" {
		t.Errorf("clipString = %q; 'Hi' bekleniyordu", s2)
	}

	s3 := clipString("Hello", 2)
	if s3 != "He" {
		t.Errorf("clipString = %q; 'He' bekleniyordu", s3)
	}

	wide := clipString("🔍abc", 3)
	if wide != "🔍a" {
		t.Errorf("wide clipString = %q; '🔍a' bekleniyordu", wide)
	}
}

func TestTableSpanning(t *testing.T) {
	// A simple 3x3 table with spans
	tbl := Table{
		Header: &TableRow{
			Cells: []TableCell{
				{Text: "H1", ColSpan: 2},
				{Text: "H3"},
			},
		},
		Rows: []TableRow{
			{
				Cells: []TableCell{
					{Text: "R1C1", RowSpan: 2},
					{Text: "R1C2"},
					{Text: "R1C3"},
				},
			},
			{
				Cells: []TableCell{
					{Text: "R2C2"},
					{Text: "R2C3"},
				},
			},
		},
		Constraints: []TableConstraint{
			{Type: ConstraintFixed, Value: 5},
			{Type: ConstraintFixed, Value: 5},
			{Type: ConstraintFixed, Value: 5},
		},
		DrawGrid: true,
	}

	// Draw on a 20x5 buffer
	area := cell.NewRect(0, 0, 20, 5)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	tbl.Draw(ctx, buf)

	// Verify H1 spans across column 0, column 1, and the separator in between.
	h1Cell := buf.Get(0, 0)
	if h1Cell == nil || h1Cell.Content != 'H' {
		t.Errorf("Expected 'H' at (0,0), got %v", h1Cell)
	}
	h1Cell2 := buf.Get(1, 0)
	if h1Cell2 == nil || h1Cell2.Content != '1' {
		t.Errorf("Expected '1' at (1,0), got %v", h1Cell2)
	}
	// The separator between col 0 and col 1 on row 0 (which is at X = 5) should be space ' ' due to colSpan
	sepCell := buf.Get(5, 0)
	if sepCell == nil || sepCell.Content != ' ' {
		t.Errorf("Expected space at (5,0) due to colSpan, got %q", sepCell.Content)
	}

	// Verify R1C1 rowSpan=2. It occupies column 0 on row 2 and row 3.
	r1c1Row2 := buf.Get(0, 2)
	if r1c1Row2 == nil || r1c1Row2.Content != 'R' {
		t.Errorf("Expected 'R' at (0,2), got %v", r1c1Row2)
	}
	r1c1Row3 := buf.Get(0, 3)
	if r1c1Row3 == nil || r1c1Row3.Content != ' ' {
		t.Errorf("Expected space at (0,3) due to rowSpan background, got %v", r1c1Row3)
	}
}

func TestTableInteractiveResizing(t *testing.T) {
	state := NewTableState()
	tbl := Table{
		State: state,
		Constraints: []TableConstraint{
			{Type: ConstraintFixed, Value: 10},
			{Type: ConstraintFixed, Value: 10},
		},
		DrawGrid: true,
	}

	area := cell.NewRect(0, 0, 21, 5)
	buf := buffer.NewBuffer(area)

	var registeredMouse bool
	var capturedMouse func(ev backend.MouseEvent)

	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(regArea cell.Rect, handler func(ev backend.MouseEvent)) {
		registeredMouse = true
		handler(backend.MouseEvent{X: 10, Y: 0, Button: backend.MouseLeft, Drag: false})
	}
	ctx.CaptureMouse = func(handler func(ev backend.MouseEvent)) {
		capturedMouse = handler
	}

	tbl.Draw(ctx, buf)

	if !registeredMouse {
		t.Errorf("Expected RegisterMouse to be called")
	}
	if capturedMouse == nil {
		t.Errorf("Expected CaptureMouse to be called")
	}

	// Simulate dragging the mouse to the right (dx = +3)
	capturedMouse(backend.MouseEvent{X: 13, Y: 0, Button: backend.MouseLeft, Drag: true})

	if state.ColumnWidths[0] != 13 {
		t.Errorf("Expected column 0 width 13, got %d", state.ColumnWidths[0])
	}
	if state.ColumnWidths[1] != 7 {
		t.Errorf("Expected column 1 width 7, got %d", state.ColumnWidths[1])
	}
}

func TestTableInteractiveResizingCascading(t *testing.T) {
	state := NewTableState()
	tbl := Table{
		State: state,
		Constraints: []TableConstraint{
			{Type: ConstraintFixed, Value: 10},
			{Type: ConstraintFixed, Value: 10},
			{Type: ConstraintFixed, Value: 10},
		},
		DrawGrid: true,
	}

	area := cell.NewRect(0, 0, 32, 5)
	buf := buffer.NewBuffer(area)

	var capturedMouse func(ev backend.MouseEvent)

	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(regArea cell.Rect, handler func(ev backend.MouseEvent)) {
		if regArea.X == 10 { // boundary between col 0 and 1
			handler(backend.MouseEvent{X: 10, Y: 0, Button: backend.MouseLeft, Drag: false})
		}
	}
	ctx.CaptureMouse = func(handler func(ev backend.MouseEvent)) {
		capturedMouse = handler
	}

	tbl.Draw(ctx, buf)

	if capturedMouse == nil {
		t.Fatalf("Expected CaptureMouse to be called")
	}

	// Grow column 0 by 5 (from 10 to 15).
	// This should shrink column 2 (last column) by 5 (from 10 to 5).
	// Column 1 should remain 10.
	capturedMouse(backend.MouseEvent{X: 15, Y: 0, Button: backend.MouseLeft, Drag: true})

	if state.ColumnWidths[0] != 15 {
		t.Errorf("Expected column 0 width 15, got %d", state.ColumnWidths[0])
	}
	if state.ColumnWidths[1] != 10 {
		t.Errorf("Expected column 1 width 10, got %d", state.ColumnWidths[1])
	}
	if state.ColumnWidths[2] != 5 {
		t.Errorf("Expected column 2 width 5, got %d", state.ColumnWidths[2])
	}

	// Grow column 0 by 12 (from 10 to 22).
	// Since column 2 can only shrink by 8 (from 10 to 2, min width),
	// the remaining 4 shrink should be absorbed by column 1 (shrinking it from 10 to 6).
	// Column 0 should grow to 22.
	capturedMouse(backend.MouseEvent{X: 22, Y: 0, Button: backend.MouseLeft, Drag: true})

	if state.ColumnWidths[0] != 22 {
		t.Errorf("Expected column 0 width 22, got %d", state.ColumnWidths[0])
	}
	if state.ColumnWidths[1] != 6 {
		t.Errorf("Expected column 1 width 6, got %d", state.ColumnWidths[1])
	}
	if state.ColumnWidths[2] != 2 {
		t.Errorf("Expected column 2 width 2, got %d", state.ColumnWidths[2])
	}
}
