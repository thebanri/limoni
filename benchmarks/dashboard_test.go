package benchmarks

import (
	"bytes"
	"strings"
	"testing"
)

func TestDashboardWriters(t *testing.T) {
	report := DashboardReport{Implementation: "limoni", Valid: true, Workloads: []WorkloadReport{{Spec: WorkloadSpec{Name: "empty"}, Summary: Summary{Frames: 1}}}}
	var jsonOut, htmlOut bytes.Buffer
	if err := WriteJSON(&jsonOut, report); err != nil {
		t.Fatal(err)
	}
	if err := WriteHTML(&htmlOut, report, nil, "", "", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonOut.String(), "limoni") || !strings.Contains(htmlOut.String(), "Benchmark Dashboard") {
		t.Fatalf("dashboard output missing: %q / %q", jsonOut.String(), htmlOut.String())
	}
	if !strings.Contains(htmlOut.String(), "<th>Implementation</th>") || !strings.Contains(htmlOut.String(), "<th>Allocs</th>") {
		t.Fatal("dashboard is missing implementation/alloc columns")
	}
	if !strings.Contains(htmlOut.String(), "VALID FOR CORE WORKLOADS") {
		t.Fatal("dashboard is missing core-workload validation status")
	}
}
