package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
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
	OutputMode string `json:"output_mode"`
	ColorMode  string `json:"color_mode"`
	Iterations int    `json:"iterations"`
}

type summary struct {
	Frames        int     `json:"Frames"`
	P50NS         int64   `json:"P50NS"`
	P95NS         int64   `json:"P95NS"`
	P99NS         int64   `json:"P99NS"`
	MinNS         int64   `json:"MinNS"`
	MaxNS         int64   `json:"MaxNS"`
	MeanNS        int64   `json:"MeanNS"`
	StdDevNS      int64   `json:"StdDevNS"`
	BytesPerFrame float64 `json:"BytesPerFrame"`
	AllocBytes    uint64  `json:"AllocBytes"`
	Allocs        uint64  `json:"Allocs"`
}

type workload struct {
	Spec    spec    `json:"spec"`
	Summary summary `json:"summary"`
}

type report struct {
	Implementation string     `json:"implementation"`
	Environment    envMetadata `json:"environment"`
	Valid          bool       `json:"valid"`
	Workloads      []workload `json:"workloads"`
}

type envMetadata struct {
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

type model struct {
	specName string
	text     string
	toggle   bool
	idx      int
	width    int
	height   int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if val, ok := msg.(int); ok {
		m.idx = val
	}
	switch m.specName {
	case "single-cell-update":
		m.toggle = !m.toggle
	case "resize":
		if rmsg, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = rmsg.Width
			m.height = rmsg.Height
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.specName {
	case "empty-frame":
		return ""
	case "full-redraw-120x40":
		buf := make([]byte, 120*40+40)
		idx := 0
		for y := 0; y < 40; y++ {
			for x := 0; x < 120; x++ {
				buf[idx] = 'A'
				idx++
			}
			buf[idx] = '\n'
			idx++
		}
		return string(buf)
	case "single-cell-update":
		if m.toggle {
			return "X"
		}
		return "Y"
	case "text-heavy-120x40":
		style := lipgloss.NewStyle().
			Width(120).
			Height(40).
			Border(lipgloss.DoubleBorder()).
			Align(lipgloss.Center)
		return style.Render("Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis.")
	case "unicode-emoji":
		style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(80)
		return style.Render("Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols verification.")
	case "table-10000":
		offset := m.idx % 9900
		t := table.New().
			Border(lipgloss.NormalBorder()).
			Headers("ID", "Name", "Status")
		for i := 0; i < 40; i++ {
			rowIdx := offset + i
			t.Row(fmt.Sprintf("%d", rowIdx), "process", "running")
		}
		return t.Render()
	case "virtual-1000000":
		var b bytes.Buffer
		offset := m.idx % 990000
		for i := 0; i < 40; i++ {
			rowIdx := offset + i
			fmt.Fprintf(&b, "#%06d | örnek kayıt %d | viewport cache\n", rowIdx, rowIdx)
		}
		return b.String()
	case "mouse-hit-test":
		style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
		return style.Render("Mouse Target Area")
	case "hundred-layers":
		res := "Base Content"
		for i := 0; i < 100; i++ {
			res = fmt.Sprintf("Layer %d\n%s", i, res)
		}
		return res
	case "resize":
		return fmt.Sprintf("Size: %dx%d", m.width, m.height)
	case "async-update-burst":
		return fmt.Sprintf("Value: %d", m.idx)
	case "native-image-capability":
		var b strings.Builder
		for y := 0; y < 24; y++ {
			for x := 0; x < 80; x++ {
				b.WriteString("\x1b[38;2;100;150;200;48;2;50;100;150m▄")
			}
			b.WriteByte('\n')
		}
		return b.String()
	default:
		return m.text
	}
}

func main() {
	output := flag.String("output", "bubbletea.json", "dashboard report path")
	flag.Parse()

	// Load specs from workload manifest
	manifestPath := "benchmarks/workloads.json"
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = "../../workloads.json"
	}
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = "../../../workloads.json"
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		panic(err)
	}
	var items []spec
	if err := json.Unmarshal(data, &items); err != nil {
		panic(err)
	}

	hash := sha256.Sum256(data)
	manifestHash := hex.EncodeToString(hash[:])

	gitCommit := "unknown"
	cmd := exec.Command("git", "rev-parse", "HEAD")
	if out, err := cmd.Output(); err == nil {
		gitCommit = strings.TrimSpace(string(out))
	}

	result := report{
		Implementation: "bubbletea",
		Environment: envMetadata{
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
			Go:            runtime.Version(),
			Output:        "memory",
			ManifestHash:  manifestHash,
			GitCommit:     gitCommit,
			RunnerVersion: "v1.0.0",
			WarmupCount:   10,
			BuildMode:     "release",
		},
		Valid: true,
	}

	for _, item := range items {
		m := model{specName: item.Name, text: "Limoni benchmark ✓ 日本語"}

		// Warmup (10 runs)
		for i := 0; i < 10; i++ {
			var err error
			mTemp, _ := m.Update(i)
			m = mTemp.(model)
			_ = m.View()
			_ = err
		}

		// Timing and allocation tracking
		durations := make([]int64, item.Iterations)
		var totalBytes int64

		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)

		for i := 0; i < item.Iterations; i++ {
			start := time.Now()

			var msg tea.Msg
			if item.Name == "resize" {
				msg = tea.WindowSizeMsg{Width: 100 + i, Height: 30 + i}
			} else if item.Name == "async-update-burst" {
				msg = i
			} else {
				msg = i
			}

			mTemp, _ := m.Update(msg)
			m = mTemp.(model)
			viewStr := m.View()

			durations[i] = time.Since(start).Nanoseconds()
			totalBytes += int64(len(viewStr))
		}

		runtime.ReadMemStats(&after)

		// Calculate statistics
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		p50 := durations[len(durations)*50/100]
		p95 := durations[len(durations)*95/100]
		p99 := durations[len(durations)*99/100]

		sum := int64(0)
		for _, val := range durations {
			sum += val
		}
		mean := sum / int64(len(durations))

		var varianceSum float64
		for _, val := range durations {
			diff := float64(val - mean)
			varianceSum += diff * diff
		}
		stdDev := int64(math.Sqrt(varianceSum / float64(len(durations))))

		result.Workloads = append(result.Workloads, workload{
			Spec: item,
			Summary: summary{
				Frames:        item.Iterations,
				P50NS:         p50,
				P95NS:         p95,
				P99NS:         p99,
				MinNS:         durations[0],
				MaxNS:         durations[len(durations)-1],
				MeanNS:        mean,
				StdDevNS:      stdDev,
				BytesPerFrame: float64(totalBytes) / float64(item.Iterations),
				AllocBytes:    after.TotalAlloc - before.TotalAlloc,
				Allocs:        after.Mallocs - before.Mallocs,
			},
		})
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
