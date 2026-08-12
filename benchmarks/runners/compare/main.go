package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/thebanri/limoni/benchmarks"
)

func main() {
	baselinePath := flag.String("baseline", "", "path to baseline json")
	currentPath := flag.String("current", "", "path to current json")
	flag.Parse()

	if *baselinePath == "" || *currentPath == "" {
		fmt.Println("Usage: compare -baseline <path> -current <path>")
		os.Exit(1)
	}

	baselineData, err := os.ReadFile(*baselinePath)
	if err != nil {
		fmt.Printf("Warning: baseline file missing: %v\n", err)
		os.Exit(0) // Don't fail if baseline is missing (e.g. first run)
	}

	currentData, err := os.ReadFile(*currentPath)
	if err != nil {
		fmt.Printf("Error: current file missing: %v\n", err)
		os.Exit(1)
	}

	var baselineReport benchmarks.DashboardReport
	if err := json.Unmarshal(baselineData, &baselineReport); err != nil {
		fmt.Printf("Error: failed to parse baseline JSON: %v\n", err)
		os.Exit(1)
	}

	var currentReport benchmarks.DashboardReport
	if err := json.Unmarshal(currentData, &currentReport); err != nil {
		fmt.Printf("Error: failed to parse current JSON: %v\n", err)
		os.Exit(1)
	}

	// Validate metadata & specs
	if currentReport.Environment.ManifestHash != baselineReport.Environment.ManifestHash {
		fmt.Printf("Error: manifest hash mismatch: got %q, want %q\n", currentReport.Environment.ManifestHash, baselineReport.Environment.ManifestHash)
		os.Exit(1)
	}

	if len(currentReport.Workloads) != len(baselineReport.Workloads) {
		fmt.Printf("Error: workload count mismatch: got %d, want %d\n", len(currentReport.Workloads), len(baselineReport.Workloads))
		os.Exit(1)
	}

	// Initialize default regression thresholds
	p50Threshold := 0.05
	p95Threshold := 0.10
	p99Threshold := 0.15
	allocThreshold := 0.10

	if env := os.Getenv("P50_REGRESSION_THRESHOLD"); env != "" {
		if v, err := strconv.ParseFloat(env, 64); err == nil {
			p50Threshold = v
		}
	}
	if env := os.Getenv("P95_REGRESSION_THRESHOLD"); env != "" {
		if v, err := strconv.ParseFloat(env, 64); err == nil {
			p95Threshold = v
		}
	}
	if env := os.Getenv("P99_REGRESSION_THRESHOLD"); env != "" {
		if v, err := strconv.ParseFloat(env, 64); err == nil {
			p99Threshold = v
		}
	}
	if env := os.Getenv("ALLOC_REGRESSION_THRESHOLD"); env != "" {
		if v, err := strconv.ParseFloat(env, 64); err == nil {
			allocThreshold = v
		}
	}

	hasFailure := false

	for i := range currentReport.Workloads {
		currW := currentReport.Workloads[i]
		baseW := baselineReport.Workloads[i]

		if currW.Spec.Name != baseW.Spec.Name {
			fmt.Printf("Error: workload name mismatch at index %d: %q vs %q\n", i, currW.Spec.Name, baseW.Spec.Name)
			os.Exit(1)
		}

		// Calculate p50 change
		p50Diff := float64(currW.Summary.P50NS - baseW.Summary.P50NS) / float64(baseW.Summary.P50NS)
		// Calculate p95 change
		p95Diff := float64(currW.Summary.P95NS - baseW.Summary.P95NS) / float64(baseW.Summary.P95NS)
		// Calculate p99 change
		p99Diff := float64(currW.Summary.P99NS - baseW.Summary.P99NS) / float64(baseW.Summary.P99NS)
		// Calculate allocs change
		allocsDiff := 0.0
		if baseW.Summary.Allocs > 0 {
			allocsDiff = float64(int64(currW.Summary.Allocs) - int64(baseW.Summary.Allocs)) / float64(baseW.Summary.Allocs)
		} else if currW.Summary.Allocs > 0 {
			allocsDiff = 1.0 // 100% regression if we went from 0 to > 0 allocs
		}

		fmt.Printf("Workload %q:\n", currW.Spec.Name)
		fmt.Printf("  p50: %d ns vs %d ns (%.1f%%)\n", currW.Summary.P50NS, baseW.Summary.P50NS, p50Diff*100)
		fmt.Printf("  p95: %d ns vs %d ns (%.1f%%)\n", currW.Summary.P95NS, baseW.Summary.P95NS, p95Diff*100)
		fmt.Printf("  p99: %d ns vs %d ns (%.1f%%)\n", currW.Summary.P99NS, baseW.Summary.P99NS, p99Diff*100)
		fmt.Printf("  allocs: %d vs %d (%.1f%%)\n", currW.Summary.Allocs, baseW.Summary.Allocs, allocsDiff*100)

		// Warning: p50 regression
		if p50Diff > p50Threshold {
			fmt.Printf("  [WARNING] p50 latency regressed by > %.0f%% (%.1f%%)\n", p50Threshold*100, p50Diff*100)
		}

		// Warning: p99 regression
		if p99Diff > p99Threshold {
			fmt.Printf("  [WARNING] p99 latency regressed by > %.0f%% (%.1f%%)\n", p99Threshold*100, p99Diff*100)
		}

		// Failure: p95 regression
		if p95Diff > p95Threshold {
			fmt.Printf("  [FAILURE] p95 latency regressed by > %.0f%% (%.1f%%)\n", p95Threshold*100, p95Diff*100)
			hasFailure = true
		}

		// Failure: allocs regression
		if allocsDiff > allocThreshold {
			fmt.Printf("  [FAILURE] allocations/frame regressed by > %.0f%% (%.1f%%)\n", allocThreshold*100, allocsDiff*100)
			hasFailure = true
		}
	}

	if hasFailure {
		fmt.Println("Error: Benchmark regression validation failed.")
		os.Exit(1)
	}

	fmt.Println("Success: Benchmark regression validation passed.")
}
