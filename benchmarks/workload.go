package benchmarks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Environment struct {
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Go            string `json:"go,omitempty"`
	CPU           string `json:"cpu,omitempty"`
	Output        string `json:"output,omitempty"`
	ManifestHash  string `json:"manifest_hash,omitempty"`
	GitCommit     string `json:"git_commit,omitempty"`
	RunnerVersion string `json:"runner_version,omitempty"`
	WarmupCount   int    `json:"warmup_count,omitempty"`
	BuildMode     string `json:"build_mode,omitempty"`
}

// WorkloadSpec describes a reproducible benchmark workload independent of the
// implementation being measured.
type WorkloadSpec struct {
	Name       string `json:"name"`
	Width      uint16 `json:"width"`
	Height     uint16 `json:"height"`
	Rows       int    `json:"rows"`
	Unicode    bool   `json:"unicode"`
	FullDraw   bool   `json:"full_draw"`
	Mouse      bool   `json:"mouse"`
	AsyncBurst int    `json:"async_burst"`
	OutputMode string `json:"output_mode"`
	ColorMode  string `json:"color_mode"`
	Iterations int    `json:"iterations"`
}

func (s WorkloadSpec) JSON() ([]byte, error) { return json.Marshal(s) }

type WorkloadReport struct {
	Implementation string       `json:"implementation,omitempty"`
	Spec           WorkloadSpec `json:"spec"`
	Summary        Summary      `json:"summary"`
}

type DashboardReport struct {
	Implementation string           `json:"implementation"`
	Environment    Environment      `json:"environment"`
	Valid          bool             `json:"valid"`
	Warnings       []string         `json:"warnings,omitempty"`
	Workloads      []WorkloadReport `json:"workloads"`
}

func (s WorkloadSpec) Report(metrics Metrics) WorkloadReport {
	return WorkloadReport{Spec: s, Summary: metrics.Summary()}
}

func (s WorkloadSpec) ReportFor(implementation string, metrics Metrics) WorkloadReport {
	return WorkloadReport{Implementation: implementation, Spec: s, Summary: metrics.Summary()}
}

func CurrentEnvironment(manifestData []byte) Environment {
	hash := sha256.Sum256(manifestData)
	manifestHash := hex.EncodeToString(hash[:])

	gitCommit := "unknown"
	cmd := exec.Command("git", "rev-parse", "HEAD")
	if out, err := cmd.Output(); err == nil {
		gitCommit = strings.TrimSpace(string(out))
	}

	buildMode := "release"
	return Environment{
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Go:            runtime.Version(),
		Output:        "memory",
		ManifestHash:  manifestHash,
		GitCommit:     gitCommit,
		RunnerVersion: "v1.0.0",
		WarmupCount:   10,
		BuildMode:     buildMode,
	}
}

// ValidateReport enforces strict JSON schema correctness, environment metadata
// completeness, and summary value consistency.
func ValidateReport(report DashboardReport) error {
	if report.Implementation == "" {
		return fmt.Errorf("missing implementation field")
	}
	if report.Environment.OS == "" {
		return fmt.Errorf("missing environment.os field")
	}
	if report.Environment.Arch == "" {
		return fmt.Errorf("missing environment.arch field")
	}
	if report.Environment.ManifestHash == "" {
		return fmt.Errorf("missing environment.manifest_hash field")
	}
	if report.Environment.GitCommit == "" {
		return fmt.Errorf("missing environment.git_commit field")
	}
	if report.Environment.RunnerVersion == "" {
		return fmt.Errorf("missing environment.runner_version field")
	}
	if len(report.Workloads) != 12 {
		return fmt.Errorf("workload count mismatch: got %d, want 12", len(report.Workloads))
	}
	for i, w := range report.Workloads {
		if w.Spec.Name == "" {
			return fmt.Errorf("workload %d is missing spec.name", i)
		}
		if w.Spec.Iterations <= 0 {
			return fmt.Errorf("workload %q has non-positive spec.iterations: %d", w.Spec.Name, w.Spec.Iterations)
		}
		if w.Summary.Frames < w.Spec.Iterations {
			return fmt.Errorf("workload %q has summary.Frames (%d) < spec.iterations (%d)", w.Spec.Name, w.Summary.Frames, w.Spec.Iterations)
		}
		if w.Summary.P50NS < 0 || w.Summary.P95NS < 0 || w.Summary.P99NS < 0 || w.Summary.MinNS < 0 || w.Summary.MaxNS < 0 || w.Summary.MeanNS < 0 || w.Summary.StdDevNS < 0 {
			return fmt.Errorf("workload %q has negative latency metrics", w.Spec.Name)
		}
		if w.Summary.P50NS > w.Summary.P95NS {
			return fmt.Errorf("workload %q: P50NS (%d) > P95NS (%d)", w.Spec.Name, w.Summary.P50NS, w.Summary.P95NS)
		}
		if w.Summary.P95NS > w.Summary.P99NS {
			return fmt.Errorf("workload %q: P95NS (%d) > P99NS (%d)", w.Spec.Name, w.Summary.P95NS, w.Summary.P99NS)
		}
	}
	return nil
}
