package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type testTableSource struct {
	rows  []TableRow
	calls []int
}

func (s *testTableSource) RowCount() int { return len(s.rows) }
func (s *testTableSource) RowAt(index int) TableRow {
	s.calls = append(s.calls, index)
	return s.rows[index]
}

func TestTableDataSourceRendersOnlyViewportRows(t *testing.T) {
	source := &testTableSource{rows: []TableRow{
		NewRow("0"), NewRow("1"), NewRow("2"), NewRow("3"), NewRow("4"),
	}}
	state := NewTableState()
	state.Offset = 2
	table := Table{
		DataSource:  source,
		Constraints: []TableConstraint{{Type: ConstraintFixed, Value: 6}},
		State:       state,
	}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 6, 2))
	table.Draw(cell.NewContext(buf.Area, cell.Style{}), buf)
	if len(source.calls) != 2 || source.calls[0] != 2 || source.calls[1] != 3 {
		t.Fatalf("RowAt calls = %v; want only visible rows [2 3]", source.calls)
	}
}
