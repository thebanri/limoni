package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"os"

	"github.com/thebanri/limoni/benchmarks"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

type benchmarkVirtualDataSource struct{}

func (benchmarkVirtualDataSource) RowCount(ctx context.Context) (int, error) { return 1000000, nil }
func (benchmarkVirtualDataSource) RowAt(ctx context.Context, index int) (widgets.Row, error) {
	return widgets.Row{ID: widgets.RowID(fmt.Sprintf("%d", index)), Text: "benchmark virtual data row"}, nil
}
func (benchmarkVirtualDataSource) RowID(index int) widgets.RowID {
	return widgets.RowID(fmt.Sprintf("%d", index))
}

type benchmarkModel struct{ ready chan struct{} }

func (m *benchmarkModel) Init() []runtime.Cmd                   { close(m.ready); return nil }
func (*benchmarkModel) Update(runtime.Msg) runtime.UpdateResult { return runtime.UpdateResult{} }
func (*benchmarkModel) View(*terminal.Frame)                    {}

func main() {
	output := flag.String("output", "limoni.json", "dashboard report path")
	flag.Parse()

	specs := []benchmarks.WorkloadSpec{
		{Name: "empty-frame", Width: 80, Height: 24, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "full-redraw-120x40", Width: 120, Height: 40, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "single-cell-update", Width: 80, Height: 24, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "text-heavy-120x40", Width: 120, Height: 40, Unicode: true, FullDraw: true, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "unicode-emoji", Width: 80, Height: 24, Unicode: true, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "table-10000", Width: 120, Height: 40, Rows: 10000, Unicode: true, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "virtual-1000000", Width: 120, Height: 40, Rows: 1000000, Unicode: true, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "mouse-hit-test", Width: 80, Height: 24, Mouse: true, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "hundred-layers", Width: 80, Height: 24, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "resize", Width: 80, Height: 24, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "async-update-burst", Width: 80, Height: 24, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
		{Name: "native-image-capability", Width: 80, Height: 24, Iterations: 100, OutputMode: "memory", ColorMode: "truecolor"},
	}

	workloads := make([]benchmarks.WorkloadReport, 0, len(specs))
	for _, spec := range specs {
		var runFn func() []byte
		var teardownFn func()

		// Setup phase (outside measurement)
		switch spec.Name {
		case "empty-frame":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			runFn = func() []byte {
				term.Draw(nil)
				return []byte(term.Snapshot())
			}

		case "full-redraw-120x40":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			runFn = func() []byte {
				term.Draw(func(f *terminal.Frame) {
					for y := uint16(0); y < spec.Height; y++ {
						for x := uint16(0); x < spec.Width; x++ {
							f.Buffer.SetCell(x, y, cell.Cell{
								Content: 'A',
								Style:   cell.Style{Fg: cell.Color(x), Bg: cell.Color(y)},
							})
						}
					}
				})
				return []byte(term.Snapshot())
			}

		case "single-cell-update":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			term.Draw(func(f *terminal.Frame) {
				f.Buffer.SetString(0, 0, "Initial frame state", cell.Style{})
			})
			var toggle bool
			runFn = func() []byte {
				term.Draw(func(f *terminal.Frame) {
					if toggle {
						f.Buffer.SetCell(0, 0, cell.Cell{Content: 'X'})
					} else {
						f.Buffer.SetCell(0, 0, cell.Cell{Content: 'Y'})
					}
					toggle = !toggle
				})
				return []byte(term.Snapshot())
			}

		case "text-heavy-120x40":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			p := &widgets.Paragraph{Text: "Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis.", Wrap: true}
			rect := cell.NewRect(0, 0, spec.Width, spec.Height)
			runFn = func() []byte {
				term.Render(p, rect)
				return []byte(term.Snapshot())
			}

		case "unicode-emoji":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			p := &widgets.Paragraph{Text: "Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols verification.", Wrap: true}
			rect := cell.NewRect(0, 0, spec.Width, spec.Height)
			runFn = func() []byte {
				term.Render(p, rect)
				return []byte(term.Snapshot())
			}

		case "table-10000":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			rows := make([]widgets.TableRow, 10000)
			for i := range rows {
				rows[i] = widgets.NewRow(fmt.Sprintf("%d", i), "process", "running")
			}
			table := widgets.Table{
				Rows: rows,
				Constraints: []widgets.TableConstraint{
					{Type: widgets.ConstraintFixed, Value: 8},
					{Type: widgets.ConstraintPercentage, Value: 40},
					{Type: widgets.ConstraintFill},
				},
				DrawGrid: true,
			}
			rect := cell.NewRect(0, 0, spec.Width, spec.Height)
			runFn = func() []byte {
				term.Render(table, rect)
				return []byte(term.Snapshot())
			}

		case "virtual-1000000":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			state := widgets.NewVirtualDataState()
			provider := benchmarkVirtualDataSource{}
			rect := cell.NewRect(0, 0, spec.Width, spec.Height)
			viewportHeight := int(spec.Height)
			_ = state.Refresh(context.Background(), provider, 0, viewportHeight, 2)
			var offset int
			runFn = func() []byte {
				term.Render(widgets.VirtualDataView{
					State:  state,
					Source: provider,
					First:  0,
					Offset: &offset,
					Style:  cell.Style{},
				}, rect)
				return []byte(term.Snapshot())
			}

		case "mouse-hit-test":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			term.Draw(func(frame *terminal.Frame) {
				for i := 0; i < 100; i++ {
					frame.RegisterClickHandler(cell.NewRect(uint16(i), 0, 1, 1), func(backend.MouseEvent) {})
				}
			})
			runFn = func() []byte {
				term.Click(50, 0)
				return []byte(term.Snapshot())
			}

		case "hundred-layers":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			term.Draw(func(frame *terminal.Frame) {
				for i := 0; i < 100; i++ {
					frame.RegisterLayer(fmt.Sprintf("layer-%d", i), terminal.LayerPopup, cell.NewRect(uint16(i%100), uint16(i%30), 10, 5), i, nil)
				}
			})
			runFn = func() []byte {
				term.Click(50, 10)
				return []byte(term.Snapshot())
			}

		case "resize":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			var toggle bool
			runFn = func() []byte {
				if toggle {
					term.Resize(spec.Width+20, spec.Height+10)
				} else {
					term.Resize(spec.Width, spec.Height)
				}
				term.Draw(nil)
				toggle = !toggle
				return []byte(term.Snapshot())
			}

		case "async-update-burst":
			model := &benchmarkModel{}
			program := runtime.New(runtime.WithModel(model), runtime.WithMessageQueue(1024))
			ctx, cancel := context.WithCancel(context.Background())
			ready := make(chan struct{})
			model.ready = ready
			go func() { _ = program.Run(ctx) }()
			<-ready
			var idx int
			runFn = func() []byte {
				_ = program.Send(ctx, idx)
				idx++
				return nil
			}
			teardownFn = func() {
				cancel()
			}

		case "native-image-capability":
			term := testkit.NewTerminal(spec.Width, spec.Height)
			img := image.NewRGBA(image.Rect(0, 0, 2, 2))
			imgWidget := &widgets.Image{Img: img, Transparent: true}
			rect := cell.NewRect(0, 0, spec.Width, spec.Height)
			runFn = func() []byte {
				term.Render(imgWidget, rect)
				return []byte(term.Snapshot())
			}
		}

		// Warmup (10 runs) & Timed (spec.Iterations = 100 runs) measurements
		metrics := benchmarks.MeasureWorkloadWithWarmup(10, spec.Iterations, runFn)
		if teardownFn != nil {
			teardownFn()
		}

		workloads = append(workloads, spec.ReportFor("limoni", metrics))
	}

	report := benchmarks.DashboardReport{
		Implementation: "limoni",
		Environment:    benchmarks.CurrentEnvironment(),
		Valid:          true,
		Workloads:      workloads,
	}

	writeReport(*output, report)
}

func writeReport(path string, report benchmarks.DashboardReport) {
	if err := os.MkdirAll(dir(path), 0o755); err != nil {
		panic(err)
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := benchmarks.WriteJSON(f, report); err != nil {
		panic(err)
	}
}

func dir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
