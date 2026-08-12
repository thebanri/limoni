package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/thebanri/limoni/benchmarks"
)

func normalizeOS(osStr string) string {
	osStr = strings.ToLower(osStr)
	if strings.Contains(osStr, "linux") {
		return "linux"
	}
	if strings.Contains(osStr, "darwin") || strings.Contains(osStr, "macos") {
		return "darwin"
	}
	if strings.Contains(osStr, "windows") {
		return "windows"
	}
	return osStr
}

func normalizeArch(archStr string) string {
	archStr = strings.ToLower(archStr)
	if strings.Contains(archStr, "x86_64") || strings.Contains(archStr, "amd64") {
		return "amd64"
	}
	if strings.Contains(archStr, "aarch64") || strings.Contains(archStr, "arm64") {
		return "arm64"
	}
	return archStr
}

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
		if err := benchmarks.ValidateReport(report); err != nil {
			fmt.Printf("Validation error for report %s: %v\n", path, err)
			os.Exit(1)
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
		os1 := normalizeOS(report.Environment.OS)
		os2 := normalizeOS(reports[0].Environment.OS)
		arch1 := normalizeArch(report.Environment.Arch)
		arch2 := normalizeArch(reports[0].Environment.Arch)

		if !report.Valid || os1 != os2 || arch1 != arch2 {
			fmt.Printf("Environment validation failed for %s:\n", report.Implementation)
			fmt.Printf("  Report valid: %t\n", report.Valid)
			fmt.Printf("  OS: %q (normalized: %q), want %q (normalized: %q)\n", report.Environment.OS, os1, reports[0].Environment.OS, os2)
			fmt.Printf("  Arch: %q (normalized: %q), want %q (normalized: %q)\n", report.Environment.Arch, arch1, reports[0].Environment.Arch, arch2)
			panic(fmt.Sprintf("invalid or incompatible environment for %s", report.Implementation))
		}
		if report.Environment.ManifestHash != reports[0].Environment.ManifestHash {
			panic(fmt.Sprintf("workload manifest hash mismatch for %s: got %q, want %q", report.Implementation, report.Environment.ManifestHash, reports[0].Environment.ManifestHash))
		}
		if report.Environment.GitCommit != reports[0].Environment.GitCommit {
			warningMsg := fmt.Sprintf("git commit mismatch for %s: got %q, want %q", report.Implementation, report.Environment.GitCommit, reports[0].Environment.GitCommit)
			fmt.Println("Warning:", warningMsg)
			combined.Warnings = append(combined.Warnings, warningMsg)
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
	var nativeRows []benchmarks.NativeGoRow
	var goOS, goArch, goCPU string
	if rows, osVal, archVal, cpuVal, err := parseGoBenchmark("benchmark-results/go-benchmark.txt"); err == nil {
		nativeRows = rows
		goOS = osVal
		goArch = archVal
		goCPU = cpuVal
	}

	f, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := benchmarks.WriteHTML(f, combined, nativeRows, goOS, goArch, goCPU); err != nil {
		panic(err)
	}
	fmt.Println(*output)
}

func parseGoBenchmark(path string) ([]benchmarks.NativeGoRow, string, string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", "", "", err
	}
	var rows []benchmarks.NativeGoRow
	var goos, goarch, cpu string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "goos:") {
			goos = strings.TrimSpace(strings.TrimPrefix(line, "goos:"))
		} else if strings.HasPrefix(line, "goarch:") {
			goarch = strings.TrimSpace(strings.TrimPrefix(line, "goarch:"))
		} else if strings.HasPrefix(line, "cpu:") {
			cpu = strings.TrimSpace(strings.TrimPrefix(line, "cpu:"))
		} else if strings.HasPrefix(line, "Benchmark") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				name := parts[0]
				if idx := strings.LastIndex(name, "-"); idx != -1 {
					name = name[:idx]
				}
				iters := parts[1]
				speed := parts[2] + " " + parts[3]
				bytes := "0 B/op"
				allocs := "0 allocs/op"
				for i := 4; i < len(parts)-1; i += 2 {
					if parts[i+1] == "B/op" {
						bytes = parts[i] + " B/op"
					} else if parts[i+1] == "allocs/op" {
						allocs = parts[i] + " allocs/op"
					}
				}
				rows = append(rows, benchmarks.NativeGoRow{
					Name:       name,
					Iterations: iters,
					Speed:      speed,
					Bytes:      bytes,
					Allocs:     allocs,
				})
			}
		}
	}
	return rows, goos, goarch, cpu, nil
}
