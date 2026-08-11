package benchmarks

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"time"
)

// MeasureWorkload runs fn repeatedly and records wall-clock duration for each
// iteration. The optional output callback reports emitted bytes for that
// iteration without imposing an output sink on the workload.
func MeasureWorkload(iterations int, fn func() []byte) Metrics {
	var metrics Metrics
	if iterations < 0 {
		iterations = 0
	}
	for i := 0; i < iterations; i++ {
		start := time.Now()
		var output []byte
		if fn != nil {
			output = fn()
		}
		metrics.ObserveDuration(time.Since(start).Nanoseconds(), output)
	}
	return metrics
}

// MeasureWorkloadWithStats also captures heap allocation deltas. Allocation
// counters are process-local and should be compared only under the same
// workload/build configuration.
func MeasureWorkloadWithStats(iterations int, fn func() []byte) Metrics {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	metrics := MeasureWorkload(iterations, fn)
	runtime.ReadMemStats(&after)
	metrics.AllocBytes = after.TotalAlloc - before.TotalAlloc
	metrics.Allocs = after.Mallocs - before.Mallocs
	return metrics
}

// ObserveDuration records a duration and emitted output size from one
// workload iteration.
func (m *Metrics) ObserveDuration(durationNS int64, output []byte) {
	m.Observe(durationNS, int64(len(output)))
}

// Report returns a compact, machine-readable summary for CI artifacts and
// benchmark dashboards.
func (m Metrics) Report() string {
	s := m.Summary()
	return fmt.Sprintf("frames=%d p50_ns=%d p95_ns=%d p99_ns=%d bytes_per_frame=%.2f alloc_bytes=%d allocs=%d", s.Frames, s.P50NS, s.P95NS, s.P99NS, s.BytesPerFrame, s.AllocBytes, s.Allocs)
}

// Metrics is a deterministic in-process benchmark summary. Durations are
// stored as nanoseconds and output bytes are tracked separately from a
// terminal sink.
type Metrics struct {
	Frames      int
	DurationsNS []int64
	OutputBytes int64
	AllocBytes  uint64
	Allocs      uint64
	DirtyCells  int64
	VisibleRows int64
	LatencyNS   []int64
	Goroutines  int
}

func (m *Metrics) Observe(durationNS, outputBytes int64) {
	if m == nil {
		return
	}
	m.Frames++
	m.DurationsNS = append(m.DurationsNS, durationNS)
	m.OutputBytes += outputBytes
}

func (m *Metrics) ObserveFrame(durationNS, outputBytes, dirtyCells, visibleRows int64) {
	if m == nil {
		return
	}
	m.Observe(durationNS, outputBytes)
	m.DirtyCells += dirtyCells
	m.VisibleRows += visibleRows
	m.Goroutines = runtime.NumGoroutine()
}

func (m *Metrics) ObserveLatency(durationNS int64) {
	if m != nil {
		m.LatencyNS = append(m.LatencyNS, durationNS)
	}
}

type Summary struct {
	Frames        int
	P50NS         int64
	P95NS         int64
	P99NS         int64
	BytesPerFrame float64
	AllocBytes    uint64
	Allocs        uint64
	DirtyCells    int64
	VisibleRows   int64
	RowsPerSecond float64
	Goroutines    int
	LatencyP95NS  int64
}

func (m Metrics) Summary() Summary {
	if len(m.DurationsNS) == 0 {
		return Summary{}
	}
	values := append([]int64(nil), m.DurationsNS...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	quantile := func(percent int) int64 {
		index := (len(values)*percent + 99) / 100
		if index < 1 {
			index = 1
		}
		if index > len(values) {
			index = len(values)
		}
		return values[index-1]
	}
	frames := m.Frames
	if frames == 0 {
		frames = len(values)
	}
	latencyP95 := int64(0)
	if len(m.LatencyNS) > 0 {
		latency := append([]int64(nil), m.LatencyNS...)
		sort.Slice(latency, func(i, j int) bool { return latency[i] < latency[j] })
		index := (len(latency)*95+99)/100 - 1
		latencyP95 = latency[index]
	}
	rowsPerSecond := 0.0
	if quantile(50) > 0 {
		rowsPerSecond = float64(m.VisibleRows) / (float64(quantile(50)) / 1e9 * float64(frames))
	}
	return Summary{Frames: frames, P50NS: quantile(50), P95NS: quantile(95), P99NS: quantile(99), BytesPerFrame: float64(m.OutputBytes) / float64(frames), AllocBytes: m.AllocBytes, Allocs: m.Allocs, DirtyCells: m.DirtyCells, VisibleRows: m.VisibleRows, RowsPerSecond: rowsPerSecond, Goroutines: m.Goroutines, LatencyP95NS: latencyP95}
}

func (m Metrics) JSON() ([]byte, error) { return json.Marshal(m.Summary()) }
