package benchmarks

import (
	"encoding/json"
	"html/template"
	"io"
)

type DashboardReport struct {
	Implementation string           `json:"implementation"`
	Workloads      []WorkloadReport `json:"workloads"`
}

func WriteJSON(w io.Writer, report DashboardReport) error { return json.NewEncoder(w).Encode(report) }

func WriteHTML(w io.Writer, report DashboardReport) error {
	const page = `<!doctype html><html><body><h1>Limoni Benchmark Dashboard</h1><table border="1"><tr><th>Implementation</th><th>Workload</th><th>p50 ns</th><th>p95 ns</th><th>p99 ns</th><th>B/frame</th></tr>{{range .Workloads}}<tr><td>{{$.Implementation}}</td><td>{{.Spec.Name}}</td><td>{{.Summary.P50NS}}</td><td>{{.Summary.P95NS}}</td><td>{{.Summary.P99NS}}</td><td>{{.Summary.BytesPerFrame}}</td></tr>{{end}}</table></body></html>`
	return template.Must(template.New("dashboard").Parse(page)).Execute(w, report)
}
