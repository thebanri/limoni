package testkit

import (
	"fmt"
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

func newMouseTestTable(rowCount int) (widgets.Table, *widgets.TableState) {
	rows := make([]widgets.TableRow, rowCount)
	for i := range rows {
		rows[i] = widgets.NewRow(fmt.Sprintf("row-%d", i), "value")
	}
	state := widgets.NewTableState()
	return widgets.Table{
		Rows:        rows,
		State:       state,
		Constraints: []widgets.TableConstraint{{Type: widgets.ConstraintFixed, Value: 10}, {Type: widgets.ConstraintFill}},
	}, state
}

// TestTableRowBlockClickSelectsRow, satır bloğu tek fare bölgesiyle kaydedildiğinde
// tıklanan satırın doğru indeksle seçildiğini doğrular.
func TestTableRowBlockClickSelectsRow(t *testing.T) {
	table, state := newMouseTestTable(50)
	term := NewTerminal(30, 10)
	area := cell.NewRect(0, 0, 30, 10)

	term.Render(table, area)
	if !term.Click(2, 3) {
		t.Fatalf("row click was not routed")
	}
	if state.Selected != 3 {
		t.Fatalf("expected selected row 3, got %d", state.Selected)
	}

	term.Render(table, area)
	if !term.Click(2, 0) {
		t.Fatalf("first row click was not routed")
	}
	if state.Selected != 0 {
		t.Fatalf("expected selected row 0, got %d", state.Selected)
	}
}

// TestTableRowBlockClickRespectsScrollOffset, kaydırma sonrası tıklamanın
// görünür satır indeksine offset eklediğini doğrular.
func TestTableRowBlockClickRespectsScrollOffset(t *testing.T) {
	table, state := newMouseTestTable(50)
	term := NewTerminal(30, 10)
	area := cell.NewRect(0, 0, 30, 10)

	term.Render(table, area)
	if !term.Mouse(backend.MouseEvent{X: 2, Y: 2, Button: backend.MouseScrollDown}) {
		t.Fatalf("scroll event was not routed")
	}
	if state.Offset != 3 {
		t.Fatalf("expected offset 3 after scroll, got %d", state.Offset)
	}

	term.Render(table, area)
	if !term.Click(2, 1) {
		t.Fatalf("row click was not routed after scroll")
	}
	if state.Selected != state.Offset+1 {
		t.Fatalf("expected selected row %d, got %d", state.Offset+1, state.Selected)
	}
}
