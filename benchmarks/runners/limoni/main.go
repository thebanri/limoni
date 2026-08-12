package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"os"

	"github.com/thebanri/limoni/benchmarks"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
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

	// Load specs from workload manifest
	manifestPath := "benchmarks/workloads.json"
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = "../../workloads.json"
	}
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = "../../../workloads.json"
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		panic(err)
	}
	var specs []benchmarks.WorkloadSpec
	if err := json.Unmarshal(data, &specs); err != nil {
		panic(err)
	}

	workloads := make([]benchmarks.WorkloadReport, 0, len(specs))
	for _, spec := range specs {
		var runFn func() []byte
		var teardownFn func()

		// Setup phase (outside measurement) using production buffers
		switch spec.Name {
		case "empty-frame":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "full-redraw-120x40":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				for y := uint16(0); y < spec.Height; y++ {
					for x := uint16(0); x < spec.Width; x++ {
						front.SetCell(x, y, cell.Cell{
							Content: 'A',
							Style:   cell.Style{Fg: cell.Color(x), Bg: cell.Color(y)},
						})
					}
				}
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "single-cell-update":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			front.SetString(0, 0, "Initial frame state", cell.Style{})
			back.SetString(0, 0, "Initial frame state", cell.Style{})
			var toggle bool
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				front.SetString(0, 0, "Initial frame state", cell.Style{})
				if toggle {
					front.SetCell(0, 0, cell.Cell{Content: 'X'})
				} else {
					front.SetCell(0, 0, cell.Cell{Content: 'Y'})
				}
				toggle = !toggle
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "text-heavy-120x40":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			p := &widgets.Paragraph{Text: "Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis.", Wrap: true}
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				frame.RenderWidget(p, area)
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "unicode-emoji":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			p := &widgets.Paragraph{Text: "Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols verification.", Wrap: true}
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				frame.RenderWidget(p, area)
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "table-10000":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
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
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				frame.RenderWidget(table, area)
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "virtual-1000000":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			state := widgets.NewVirtualDataState()
			provider := benchmarkVirtualDataSource{}
			viewportHeight := int(spec.Height)
			_ = state.Refresh(context.Background(), provider, 0, viewportHeight, 2)
			var offset int
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				frame.RenderWidget(widgets.VirtualDataView{
					State:  state,
					Source: provider,
					First:  0,
					Offset: &offset,
					Style:  cell.Style{},
				}, area)
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "mouse-hit-test":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				for i := 0; i < 100; i++ {
					frame.RegisterClickHandler(cell.NewRect(uint16(i), 0, 1, 1), func(backend.MouseEvent) {})
				}
				frame.DispatchEventRegions(backend.MouseEvent{X: 50, Y: 0, Button: backend.MouseLeft})
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "hundred-layers":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				for i := 0; i < 100; i++ {
					frame.RegisterLayer(fmt.Sprintf("layer-%d", i), terminal.LayerPopup, cell.NewRect(uint16(i%100), uint16(i%30), 10, 5), i, nil)
				}
				frame.DispatchEventRegions(backend.MouseEvent{X: 50, Y: 10, Button: backend.MouseLeft})
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}

		case "resize":
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var toggle bool
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				if toggle {
					front.Resize(cell.NewRect(0, 0, spec.Width+20, spec.Height+10))
				} else {
					front.Resize(cell.NewRect(0, 0, spec.Width, spec.Height))
				}
				toggle = !toggle
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
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
			area := cell.NewRect(0, 0, spec.Width, spec.Height)
			front := buffer.NewBuffer(area)
			back := buffer.NewBuffer(area)
			img := image.NewRGBA(image.Rect(0, 0, 2, 2))
			imgWidget := &widgets.Image{Img: img, Transparent: true}
			focusMgr := terminal.NewFocusManager()
			frame := terminal.NewFrame(front, focusMgr)
			var writeBuf []byte
			runFn = func() []byte {
				front.Clear()
				frame.Reset()
				frame.RenderWidget(imgWidget, area)
				writeBuf, _ = buffer.Diff(front, back, writeBuf[:0], true, true)
				return writeBuf
			}
		}

		// Warmup (10 runs) & Timed (spec.Iterations runs) measurements
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
