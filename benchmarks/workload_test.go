package benchmarks

import "testing"

func TestWorkloadSpecReport(t *testing.T) {
	spec := WorkloadSpec{Name: "unicode-table", Width: 120, Height: 40, Rows: 10000, Unicode: true}
	report := spec.Report(MeasureWorkload(2, func() []byte { return []byte("x") }))
	data, err := report.Spec.JSON()
	if err != nil || len(data) == 0 || report.Summary.Frames != 2 {
		t.Fatalf("report=%+v json=%s err=%v", report, data, err)
	}
}
