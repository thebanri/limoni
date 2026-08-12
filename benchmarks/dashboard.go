package benchmarks

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
)

type ComparisonRow struct {
	WorkloadName   string
	LimoniP50      string
	BubbleTeaP50   string
	RatatuiP50     string
	LimoniAlloc    string
	BubbleTeaAlloc string
	RatatuiAlloc   string
}

type TemplateData struct {
	Report         DashboardReport
	ComparisonRows []ComparisonRow
}

func WriteJSON(w io.Writer, report DashboardReport) error { return json.NewEncoder(w).Encode(report) }

func WriteHTML(w io.Writer, report DashboardReport) error {
	// Build unique workload list in order of appearance
	var workloadNames []string
	seen := make(map[string]bool)
	for _, w := range report.Workloads {
		name := w.Spec.Name
		if !seen[name] {
			seen[name] = true
			workloadNames = append(workloadNames, name)
		}
	}

	formatDuration := func(ns int64) string {
		if ns == 0 {
			return "N/A"
		}
		if ns < 1000 {
			return fmt.Sprintf("%d ns", ns)
		}
		if ns < 1000000 {
			return fmt.Sprintf("%.2f µs", float64(ns)/1000.0)
		}
		return fmt.Sprintf("%.2f ms", float64(ns)/1000000.0)
	}

	formatAlloc := func(allocs uint64, bytes uint64) string {
		if allocs == 0 && bytes == 0 {
			return "0"
		}
		if bytes < 1024 {
			return fmt.Sprintf("%d (%d B)", allocs, bytes)
		}
		if bytes < 1024*1024 {
			return fmt.Sprintf("%d (%.1f KB)", allocs, float64(bytes)/1024.0)
		}
		return fmt.Sprintf("%d (%.1f MB)", allocs, float64(bytes)/(1024.0*1024.0))
	}

	var comparisonRows []ComparisonRow
	for _, name := range workloadNames {
		row := ComparisonRow{
			WorkloadName:   name,
			LimoniP50:      "N/A",
			BubbleTeaP50:   "N/A",
			RatatuiP50:     "N/A",
			LimoniAlloc:    "N/A",
			BubbleTeaAlloc: "N/A",
			RatatuiAlloc:   "N/A",
		}
		for _, w := range report.Workloads {
			if w.Spec.Name == name {
				impl := w.Implementation
				p50Str := fmt.Sprintf("%s (±%s)", formatDuration(w.Summary.P50NS), formatDuration(w.Summary.StdDevNS))
				allocStr := formatAlloc(w.Summary.Allocs, w.Summary.AllocBytes)
				switch impl {
				case "limoni":
					row.LimoniP50 = p50Str
					row.LimoniAlloc = allocStr
				case "bubbletea":
					row.BubbleTeaP50 = p50Str
					row.BubbleTeaAlloc = allocStr
				case "ratatui":
					row.RatatuiP50 = p50Str
					row.RatatuiAlloc = allocStr
				}
			}
		}
		comparisonRows = append(comparisonRows, row)
	}

	const page = `<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <title>Limoni Benchmark Dashboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body {
            font-family: 'Inter', sans-serif;
            background-color: #0f172a;
            color: #f8fafc;
            margin: 0;
            padding: 2rem;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        h1 {
            font-size: 2.5rem;
            font-weight: 700;
            background: linear-gradient(135deg, #a855f7 0%, #3b82f6 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 2rem;
            text-align: center;
        }
        .card {
            background-color: #1e293b;
            border-radius: 12px;
            padding: 1.5rem;
            margin-bottom: 2rem;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
            border: 1px solid #334155;
        }
        .card-title {
            font-size: 1.25rem;
            font-weight: 600;
            margin-bottom: 1rem;
            border-bottom: 1px solid #334155;
            padding-bottom: 0.5rem;
            color: #38bdf8;
        }
        .badge {
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.875rem;
            font-weight: 600;
            display: inline-block;
        }
        .badge-success {
            background-color: #059669;
            color: #ecfdf5;
        }
        .badge-error {
            background-color: #dc2626;
            color: #fef2f2;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 1rem;
        }
        th, td {
            padding: 0.75rem 1rem;
            text-align: left;
            border-bottom: 1px solid #334155;
        }
        th {
            background-color: #334155;
            color: #f1f5f9;
            font-weight: 600;
        }
        tr:hover {
            background-color: #334155;
        }
        .highlight-limoni {
            color: #a855f7;
            font-weight: 600;
        }
        .env-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
        }
        .env-item {
            background-color: #0f172a;
            padding: 0.75rem;
            border-radius: 6px;
            border: 1px solid #334155;
        }
        .env-label {
            font-size: 0.75rem;
            color: #94a3b8;
            margin-bottom: 0.25rem;
        }
        .env-val {
            font-size: 1rem;
            font-weight: 500;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Limoni Benchmark Dashboard</h1>
        
        <!-- Environment & Validation Info -->
        <div class="card">
            <div class="card-title">Test Environment & Validation Status</div>
            <div class="env-grid">
                <div class="env-item">
                    <div class="env-label">Validation Status</div>
                    <div class="env-val">
                        {{if .Report.Valid}}
                        <span class="badge badge-success">VALID COMPARISON</span>
                        {{else}}
                        <span class="badge badge-error">INVALID COMPARISON</span>
                        {{end}}
                    </div>
                </div>
                <div class="env-item">
                    <div class="env-label">Host OS</div>
                    <div class="env-val">{{.Report.Environment.OS}}</div>
                </div>
                <div class="env-item">
                    <div class="env-label">Arch</div>
                    <div class="env-val">{{.Report.Environment.Arch}}</div>
                </div>
                <div class="env-item">
                    <div class="env-label">Output Mode</div>
                    <div class="env-val">{{.Report.Environment.Output}}</div>
                </div>
            </div>
            {{if .Report.Warnings}}
            <div style="margin-top: 1rem; color: #facc15; font-size: 0.875rem;">
                <strong>Warnings:</strong>
                <ul>
                    {{range .Report.Warnings}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
            </div>
            {{end}}
        </div>

        <!-- Comparison Matrix Card -->
        <div class="card">
            <div class="card-title">Side-by-Side p50 Latency & Allocation Comparison Matrix</div>
            <table>
                <thead>
                    <tr>
                        <th>Workload</th>
                        <th>Limoni p50 (StdDev)</th>
                        <th>Limoni Allocs (Bytes)</th>
                        <th>Bubble Tea p50 (StdDev)</th>
                        <th>Bubble Tea Allocs (Bytes)</th>
                        <th>Ratatui p50 (StdDev)</th>
                        <th>Ratatui Allocs (Bytes)</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .ComparisonRows}}
                    <tr>
                        <td><strong>{{.WorkloadName}}</strong></td>
                        <td class="highlight-limoni">{{.LimoniP50}}</td>
                        <td class="highlight-limoni">{{.LimoniAlloc}}</td>
                        <td>{{.BubbleTeaP50}}</td>
                        <td>{{.BubbleTeaAlloc}}</td>
                        <td>{{.RatatuiP50}}</td>
                        <td>{{.RatatuiAlloc}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>

        <!-- Detailed Workload Raw Metrics -->
        <div class="card">
            <div class="card-title">Detailed Execution Raw Metrics</div>
            <table>
                <thead>
                    <tr>
                        <th>Implementation</th>
                        <th>Workload</th>
                        <th>Frames</th>
                        <th>p50 Latency</th>
                        <th>Mean Latency</th>
                        <th>StdDev Latency</th>
                        <th>Min Latency</th>
                        <th>Max Latency</th>
                        <th>Bytes / Frame</th>
                        <th>AllocBytes</th>
                        <th>Allocs</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Report.Workloads}}
                    <tr>
                        <td><span style="text-transform: capitalize;">{{if .Implementation}}{{.Implementation}}{{else}}{{$.Report.Implementation}}{{end}}</span></td>
                        <td>{{.Spec.Name}}</td>
                        <td>{{.Summary.Frames}}</td>
                        <td>{{formatDuration .Summary.P50NS}}</td>
                        <td>{{formatDuration .Summary.MeanNS}}</td>
                        <td>{{formatDuration .Summary.StdDevNS}}</td>
                        <td>{{formatDuration .Summary.MinNS}}</td>
                        <td>{{formatDuration .Summary.MaxNS}}</td>
                        <td>{{printf "%.2f" .Summary.BytesPerFrame}}</td>
                        <td>{{.Summary.AllocBytes}}</td>
                        <td>{{.Summary.Allocs}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>`

	funcMap := template.FuncMap{
		"formatDuration": formatDuration,
	}

	tmpl, err := template.New("dashboard").Funcs(funcMap).Parse(page)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, TemplateData{
		Report:         report,
		ComparisonRows: comparisonRows,
	})
}
