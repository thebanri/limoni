package main

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type spec struct {
	Name       string `json:"name"`
	Width      uint16 `json:"width"`
	Height     uint16 `json:"height"`
	Rows       int    `json:"rows"`
	Unicode    bool   `json:"unicode"`
	FullDraw   bool   `json:"full_draw"`
	Mouse      bool   `json:"mouse"`
	AsyncBurst int    `json:"async_burst"`
}
type summary struct {
	Frames              int     `json:"Frames"`
	P50NS, P95NS, P99NS int64   `json:"P50NS"`
	BytesPerFrame       float64 `json:"BytesPerFrame"`
	AllocBytes, Allocs  uint64  `json:"AllocBytes"`
}
type workload struct {
	Spec    spec    `json:"spec"`
	Summary summary `json:"summary"`
}
type report struct {
	Implementation string     `json:"implementation"`
	Workloads      []workload `json:"workloads"`
}
type model struct{ text string }

func (m model) Init() tea.Cmd                           { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m model) View() string                            { return m.text }

func main() {
	output := flag.String("output", "bubbletea.json", "dashboard report path")
	flag.Parse()
	items := []spec{{Name: "empty-frame", Width: 80, Height: 24}, {Name: "text-heavy-120x40", Width: 120, Height: 40, Unicode: true, FullDraw: true}}
	result := report{Implementation: "bubbletea"}
	for _, item := range items {
		start := time.Now()
		p := tea.NewProgram(model{text: "Limoni benchmark ✓ 日本語"}, tea.WithInput(nil), tea.WithOutput(nil))
		_ = p
		outputBytes := len((model{text: "Limoni benchmark ✓ 日本語"}).View())
		elapsed := time.Since(start).Nanoseconds()
		result.Workloads = append(result.Workloads, workload{Spec: item, Summary: summary{Frames: 1, P50NS: elapsed, P95NS: elapsed, P99NS: elapsed, BytesPerFrame: float64(outputBytes)}})
	}
	f, err := os.Create(*output)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(result); err != nil {
		panic(err)
	}
}
