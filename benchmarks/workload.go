package benchmarks

import "encoding/json"

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
}

func (s WorkloadSpec) JSON() ([]byte, error) { return json.Marshal(s) }

type WorkloadReport struct {
	Spec    WorkloadSpec `json:"spec"`
	Summary Summary      `json:"summary"`
}

func (s WorkloadSpec) Report(metrics Metrics) WorkloadReport {
	return WorkloadReport{Spec: s, Summary: metrics.Summary()}
}
