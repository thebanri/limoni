package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/thebanri/limoni/benchmarks"
)

func main() {
	output := flag.String("output", "dashboard.html", "dashboard output")
	flag.Parse()
	var reports []benchmarks.DashboardReport
	for _, path := range flag.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		var report benchmarks.DashboardReport
		if err := json.Unmarshal(data, &report); err != nil {
			panic(err)
		}
		reports = append(reports, report)
	}
	if len(reports) == 0 {
		panic("no runner reports")
	}

	combined := reports[0]
	combined.Workloads = append([]benchmarks.WorkloadReport(nil), combined.Workloads...)

	// Populate implementation for the first report's workloads
	for i := range combined.Workloads {
		combined.Workloads[i].Implementation = combined.Implementation
	}

	for _, report := range reports[1:] {
		if !report.Valid || report.Environment.OS != reports[0].Environment.OS || report.Environment.Arch != reports[0].Environment.Arch {
			panic(fmt.Sprintf("invalid or incompatible environment for %s", report.Implementation))
		}
		if len(report.Workloads) != len(reports[0].Workloads) {
			panic(fmt.Sprintf("workload count mismatch for %s: got %d, want %d", report.Implementation, len(report.Workloads), len(reports[0].Workloads)))
		}
		for i := range report.Workloads {
			if report.Workloads[i].Spec.Name != reports[0].Workloads[i].Spec.Name {
				panic(fmt.Sprintf("workload name mismatch for %s at index %d: got %s, want %s", report.Implementation, i, report.Workloads[i].Spec.Name, reports[0].Workloads[i].Spec.Name))
			}
			if report.Workloads[i].Spec.Width != reports[0].Workloads[i].Spec.Width || report.Workloads[i].Spec.Height != reports[0].Workloads[i].Spec.Height {
				panic(fmt.Sprintf("workload dimension mismatch for %s: %s", report.Implementation, report.Workloads[i].Spec.Name))
			}
			if report.Workloads[i].Spec.Iterations != reports[0].Workloads[i].Spec.Iterations {
				panic(fmt.Sprintf("workload iterations mismatch for %s: %s", report.Implementation, report.Workloads[i].Spec.Name))
			}
		}
		for _, workload := range report.Workloads {
			workload.Implementation = report.Implementation
			combined.Workloads = append(combined.Workloads, workload)
		}
	}

	f, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := benchmarks.WriteHTML(f, combined); err != nil {
		panic(err)
	}
	fmt.Println(*output)
}
