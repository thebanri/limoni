package benchmarks

import (
	"context"
	"fmt"
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
	text := &widgets.Paragraph{Text: "Limoni benchmark text with unicode ✓ and wrapping. ", Wrap: true}
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
	table := &widgets.Table{Rows: rows, Constraints: []widgets.TableConstraint{{Type: widgets.ConstraintFixed, Value: 8}, {Type: widgets.ConstraintPercentage, Value: 40}, {Type: widgets.ConstraintFill}}, DrawGrid: true}
	term := testkit.NewTerminal(120, 40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term.Render(table, cell.NewRect(0, 0, 120, 40))
	}
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
	term.Draw(func(frame *terminal.Frame) {
		for i := 0; i < 100; i++ {
			frame.RegisterLayer(fmt.Sprintf("layer-%d", i), terminal.LayerPopup, cell.NewRect(uint16(i%100), uint16(i%30), 10, 5), i, nil)
		}
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		term.Click(50, 10)
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
