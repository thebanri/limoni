package benchmarks

import (
	"encoding/json"
	"runtime"
)

type Environment struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Go     string `json:"go,omitempty"`
	CPU    string `json:"cpu,omitempty"`
	Output string `json:"output,omitempty"`
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

func CurrentEnvironment() Environment {
	return Environment{OS: runtime.GOOS, Arch: runtime.GOARCH, Go: runtime.Version(), Output: "memory"}
}
