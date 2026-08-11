package benchmarks

import "testing"

func TestMetricsSummaryQuantilesAndBytes(t *testing.T) {
	var metrics Metrics
	for i := int64(1); i <= 100; i++ {
		metrics.Observe(i, 10)
	}
	summary := metrics.Summary()
	if summary.Frames != 100 || summary.P50NS != 50 || summary.P95NS != 95 || summary.P99NS != 99 {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.BytesPerFrame != 10 {
		t.Fatalf("bytes/frame = %v, want 10", summary.BytesPerFrame)
	}
	if report := metrics.Report(); report == "" {
		t.Fatal("expected benchmark report")
	}
}

func TestMeasureWorkload(t *testing.T) {
	metrics := MeasureWorkload(3, func() []byte { return []byte("out") })
	if summary := metrics.Summary(); summary.Frames != 3 || summary.BytesPerFrame != 3 {
		t.Fatalf("workload summary = %+v", summary)
	}
}

func TestMeasureWorkloadWithStats(t *testing.T) {
	metrics := MeasureWorkloadWithStats(2, func() []byte { return []byte("x") })
	if metrics.Frames != 2 {
		t.Fatalf("frames = %d", metrics.Frames)
	}
}

func TestMetricsExtendedReport(t *testing.T) {
	var m Metrics
	m.ObserveFrame(100, 20, 3, 5)
	m.ObserveLatency(50)
	if _, err := m.JSON(); err != nil {
		t.Fatal(err)
	}
	if s := m.Summary(); s.DirtyCells != 3 || s.VisibleRows != 5 || s.LatencyP95NS != 50 {
		t.Fatalf("summary=%+v", s)
	}
}
