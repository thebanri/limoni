package main

import (
	"flag"
	"os"
	"time"

	"github.com/thebanri/limoni/benchmarks"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	output := flag.String("output", "limoni.json", "dashboard report path")
	flag.Parse()
	specs := []benchmarks.WorkloadSpec{
		{Name: "empty-frame", Width: 80, Height: 24},
		{Name: "text-heavy-120x40", Width: 120, Height: 40, Unicode: true, FullDraw: true},
		{Name: "unicode-table", Width: 120, Height: 40, Rows: 10000, Unicode: true, FullDraw: true},
	}
	workloads := make([]benchmarks.WorkloadReport, 0, len(specs))
	for _, spec := range specs {
		metrics := benchmarks.MeasureWorkloadWithStats(20, func() []byte {
			term := testkit.NewTerminal(spec.Width, spec.Height)
			if spec.Name == "text-heavy-120x40" {
				term.Render(widgets.Paragraph{Text: "Limoni benchmark ✓ 日本語", Wrap: true}, cell.NewRect(0, 0, spec.Width, spec.Height))
			} else {
				term.Draw(nil)
			}
			return []byte(term.Snapshot())
		})
		workloads = append(workloads, spec.Report(metrics))
	}
	writeReport(*output, benchmarks.DashboardReport{Implementation: "limoni", Workloads: workloads})
	_ = time.Now()
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
