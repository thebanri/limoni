package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func BenchmarkTableVisibleRows(b *testing.B) {
	rows := make([]TableRow, 10000)
	for i := range rows {
		rows[i] = NewRow("pid", "process", "1.2%", "128 MB", "Running")
	}
	table := Table{Rows: rows, Constraints: []TableConstraint{{Type: ConstraintFixed, Value: 12}, {Type: ConstraintFill}}, State: NewTableState()}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 30))
	ctx := cell.NewContext(buf.Area, cell.Style{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		table.Draw(ctx, buf)
	}
}
