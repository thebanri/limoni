package main

import (
	"context"
	"math/rand/v2"
	"os"
	"testing"
	"time"
)

func demoLines(n int) [][]byte {
	r := rand.New(rand.NewPCG(1, 2))
	t := time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC)
	out := make([][]byte, n)
	for i := range out {
		out[i] = demoLine(nil, r, t)
	}
	return out
}

func BenchmarkDetectLevel(b *testing.B) {
	lines := demoLines(1000)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		detectLevel(lines[i%len(lines)])
	}
}

func BenchmarkStoreAdd(b *testing.B) {
	lines := demoLines(1000)
	var s store
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.add(lines[i%len(lines)])
	}
}

func BenchmarkDemoLine(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 2))
	t := time.Now()
	var buf []byte
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf = demoLine(buf[:0], r, t)
	}
}

// TestPreloadSpeed is a measurement, not an assertion: run it with -v to see
// how long a million demo lines take to generate and store.
func TestPreloadSpeed(t *testing.T) {
	if os.Getenv("ZEST_MEASURE") == "" {
		t.Skip("a measurement; set ZEST_MEASURE=1 to run it")
	}
	var s store
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	go generateDemo(ctx, &s, 1_000_000, func() {})
	for s.Len() < 1_000_000 {
		time.Sleep(time.Millisecond)
	}
	t.Logf("1,000,000 lines in %v (%d MiB of text)", time.Since(start).Round(time.Millisecond), s.bytes()>>20)
}
