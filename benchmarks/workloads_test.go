package benchmarks

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func BenchmarkEmptyFrame(b *testing.B) {
	term := testkit.NewTerminal(80, 24)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term.Draw(nil)
	}
}

func BenchmarkTextHeavyFrame(b *testing.B) {
	term := testkit.NewTerminal(120, 40)
	var sb strings.Builder
	for line := 0; line < 40; line++ {
		sb.WriteString(fmt.Sprintf("Line %02d: Limoni high-performance TUI engine benchmark with unicode ✓, symbols ★ ➔, and long text wrapping across 120 columns.\n", line))
	}
	text := &widgets.Paragraph{Text: sb.String(), Wrap: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term.Render(text, cell.NewRect(0, 0, 120, 40))
	}
}

func BenchmarkTenThousandRowTable(b *testing.B) {
	rows := make([]widgets.TableRow, 10000)
	for i := range rows {
		rows[i] = widgets.NewRow(fmt.Sprintf("%d", i), "process", "running")
	}
	state := widgets.NewTableState()
	table := &widgets.Table{
		Rows: rows,
		Constraints: []widgets.TableConstraint{
			{Type: widgets.ConstraintFixed, Value: 8},
			{Type: widgets.ConstraintPercentage, Value: 40},
			{Type: widgets.ConstraintFill},
		},
		DrawGrid: true,
		State:    state,
	}
	term := testkit.NewTerminal(120, 40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.Select((i * 7) % len(rows))
		term.Render(table, cell.NewRect(0, 0, 120, 40))
	}
}

func BenchmarkOneMillionRowVirtualScroll(b *testing.B) {
	state := widgets.NewVirtualDataState()
	provider := benchmarkVirtualDataSource{}
	term := testkit.NewTerminal(120, 40)
	viewportHeight := 40
	_ = state.Refresh(context.Background(), provider, 0, viewportHeight, 2)
	var offset int
	area := cell.NewRect(0, 0, 120, 40)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		first := (i * 13) % (1000000 - viewportHeight)
		_ = state.Refresh(context.Background(), provider, first, first+viewportHeight, 1)
		view := widgets.VirtualDataView{
			State:  state,
			Source: provider,
			First:  first,
			Offset: &offset,
			Style:  cell.Style{},
		}
		term.Render(view, area)
	}
}

type benchmarkVirtualDataSource struct{}

var benchmarkStaticRow = widgets.Row{ID: "row", Text: "benchmark virtual data row"}

func (benchmarkVirtualDataSource) RowCount(ctx context.Context) (int, error) { return 1000000, nil }
func (benchmarkVirtualDataSource) RowAt(ctx context.Context, index int) (widgets.Row, error) {
	return benchmarkStaticRow, nil
}
func (benchmarkVirtualDataSource) RowID(index int) widgets.RowID {
	return "row"
}

func BenchmarkMouseHitTest(b *testing.B) {
	term := testkit.NewTerminal(120, 40)
	term.Draw(func(frame *terminal.Frame) {
		for i := 0; i < 100; i++ {
			frame.RegisterClickHandler(cell.NewRect(uint16(i), 0, 1, 1), func(backend.MouseEvent) {})
		}
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term.Click(50, 0)
	}
}

func BenchmarkHundredLayers(b *testing.B) {
	term := testkit.NewTerminal(120, 40)
	blocks := make([]widgets.Block, 100)
	for i := range blocks {
		blocks[i] = widgets.Block{
			Title:   fmt.Sprintf("Layer %d", i),
			Borders: widgets.BorderAll,
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term.Draw(func(frame *terminal.Frame) {
			for j := 0; j < 100; j++ {
				area := cell.NewRect(uint16(j%70), uint16(j%20), 10, 3)
				frame.RenderWidget(&blocks[j], area)
			}
		})
	}
}

func BenchmarkAsyncUpdateBurst(b *testing.B) {
	model := &benchmarkModel{}
	program := runtime.New(runtime.WithModel(model), runtime.WithMessageQueue(1024))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	ready := make(chan struct{})
	model.ready = ready
	go func() { _ = program.Run(ctx); close(done) }()
	<-ready
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = program.Send(ctx, i)
	}
	cancel()
	<-done
}

type benchmarkModel struct{ ready chan struct{} }

func (m *benchmarkModel) Init() []runtime.Cmd                   { close(m.ready); return nil }
func (*benchmarkModel) Update(runtime.Msg) runtime.UpdateResult { return runtime.UpdateResult{} }
func (*benchmarkModel) View(*terminal.Frame)                    {}
