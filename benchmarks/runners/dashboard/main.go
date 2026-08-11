package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/thebanri/limoni/benchmarks"
	"os"
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
	for _, report := range reports[1:] {
		combined.Implementation = combined.Implementation + " vs " + report.Implementation
		combined.Workloads = append(combined.Workloads, report.Workloads...)
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
