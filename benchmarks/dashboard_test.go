package benchmarks

import (
	"bytes"
	"strings"
	"testing"
)

func TestDashboardWriters(t *testing.T) {
	report := DashboardReport{Implementation: "limoni", Workloads: []WorkloadReport{{Spec: WorkloadSpec{Name: "empty"}, Summary: Summary{Frames: 1}}}}
	var jsonOut, htmlOut bytes.Buffer
	if err := WriteJSON(&jsonOut, report); err != nil {
		t.Fatal(err)
	}
	if err := WriteHTML(&htmlOut, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonOut.String(), "limoni") || !strings.Contains(htmlOut.String(), "Benchmark Dashboard") {
		t.Fatalf("dashboard output missing: %q / %q", jsonOut.String(), htmlOut.String())
	}
}
