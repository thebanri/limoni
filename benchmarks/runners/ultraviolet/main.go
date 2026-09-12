// Benchmark runner for Bubble Tea v2 / Ultraviolet.
//
// # Why this runner exists, and why it differs from the v1 runner
//
// The v1 runner (benchmarks/runners/bubbletea) measures Model.Update plus
// Model.View — that is, building the output string. It never invokes Bubble
// Tea's renderer, so the diffing and ANSI encoding are excluded. The Limoni
// runner, by contrast, measures widgets drawing into a cell buffer *and* the
// full buffer.Diff that produces the escape sequence stream.
//
// Those are different amounts of work, which makes v1-vs-Limoni numbers
// structurally unfair. This runner closes that gap by measuring the cell-based
// diffing layer directly: cells in, diffed ANSI bytes out, exactly like
// Limoni's runner. The screen writes into an in-memory buffer, not a PTY, so
// no terminal is required and no I/O latency is included on either side.
//
//	go run . -output ../../../benchmark-results/ultraviolet.json
//
// # Why this is labelled Ultraviolet and not Bubble Tea v2
//
// It measures what it links, and it does not link Bubble Tea. Nothing here
// imports charm.land/bubbletea/v2 — the earlier requirement in go.mod was
// unreferenced and `go mod tidy` removed it. What is exercised is
// uv.TerminalScreen from github.com/charmbracelet/ultraviolet, which is the
// layer Bubble Tea v2 and Lip Gloss v2 are both built on and the layer a
// Bubble Tea v2 program actually drives.
//
// That makes it a fair proxy for v2's rendering cost, and an unfair label for
// v2 itself: a Bubble Tea program also pays for its own runtime, message
// dispatch and view construction, none of which is measured here. Quoting
// these numbers as "Bubble Tea v2" would overclaim in one direction and
// underclaim in the other. The report's `target` field carries the exact
// Ultraviolet version, read from the binary's build info.
//
// Driving TerminalScreen rather than TerminalRenderer matters. The renderer
// early-returns when no line is marked touched, and it is TerminalScreen that
// resets those marks — so a harness that bypasses the screen re-scans every
// line every frame. An earlier version of this runner did exactly that and
// overstated the cost by roughly four orders of magnitude on sparse workloads.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"math"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
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
	Measures      string `json:"measures,omitempty"`
	Target        string `json:"target,omitempty"`
}

// ultravioletVersion reports the Ultraviolet module version actually linked
// into this binary, read from the embedded build info. Hand-written version
// labels drift; this one cannot.
func ultravioletVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/charmbracelet/ultraviolet" {
			return dep.Path + " " + dep.Version
		}
	}
	return "unknown"
}

type report struct {
	Implementation string      `json:"implementation"`
	Environment    envMetadata `json:"environment"`
	Valid          bool        `json:"valid"`
	Workloads      []workload  `json:"workloads"`
}

const warmupIterations = 10

// truecolor keeps the color path identical to Limoni's runner, which diffs with
// trueColor and colors256 both enabled.
func truecolor(r, g, b uint8) color.Color { return color.RGBA{R: r, G: g, B: b, A: 0xff} }

// scene renders one frame into buf. Returning is enough; the harness renders
// and diffs afterwards.
type scene func(scr *uv.TerminalScreen, step int)

// writeString writes s starting at (x, y) with the given style.
func writeString(scr *uv.TerminalScreen, x, y int, s string, style uv.Style) {
	col := x
	for _, r := range s {
		cell := uv.NewCell(ansi.GraphemeWidth, string(r))
		cell.Style = style
		scr.SetCell(col, y, cell)
		col += cell.Width
	}
}

func fill(scr *uv.TerminalScreen, w, h int, content string, style uv.Style) {
	cell := uv.NewCell(ansi.GraphemeWidth, content)
	cell.Style = style
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := *cell
			scr.SetCell(x, y, &c)
		}
	}
}

func sceneFor(s spec) scene {
	w, h := int(s.Width), int(s.Height)
	if w == 0 {
		w = 120
	}
	if h == 0 {
		h = 40
	}

	switch s.Name {
	case "empty-frame":
		return func(scr *uv.TerminalScreen, step int) {}

	case "full-redraw-120x40":
		return func(scr *uv.TerminalScreen, step int) {
			ch := string(rune('A' + (step % 26)))
			style := uv.Style{
				Fg: truecolor(uint8((step*3)%256), 200, 120),
				Bg: truecolor(20, 20, uint8(step%256)),
			}
			fill(scr, w, h, ch, style)
		}

	case "single-cell-update":
		return func(scr *uv.TerminalScreen, step int) {
			writeString(scr, 0, 0, "Initial frame state", uv.Style{})
			glyph := "Y"
			if step%2 == 0 {
				glyph = "X"
			}
			writeString(scr, 0, 0, glyph, uv.Style{})
		}

	case "text-heavy-120x40":
		line := "Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis."
		return func(scr *uv.TerminalScreen, step int) {
			style := uv.Style{Fg: truecolor(200, 200, 255)}
			for y := 0; y < h; y++ {
				writeString(scr, 0, y, line, style)
			}
		}

	case "unicode-emoji":
		line := "Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols."
		return func(scr *uv.TerminalScreen, step int) {
			style := uv.Style{Fg: truecolor(255, 200, 120)}
			for y := 0; y < h; y++ {
				writeString(scr, 0, y, line, style)
			}
		}

	case "table-10000":
		return func(scr *uv.TerminalScreen, step int) {
			offset := step % 9900
			header := uv.Style{Fg: truecolor(255, 255, 255), Attrs: 1}
			body := uv.Style{Fg: truecolor(190, 200, 210)}
			writeString(scr, 0, 0, "ID        Name            Status", header)
			for i := 1; i < h; i++ {
				writeString(scr, 0, i,
					fmt.Sprintf("%-9d %-15s %s", offset+i, "process", "running"), body)
			}
		}

	case "virtual-1000000":
		return func(scr *uv.TerminalScreen, step int) {
			offset := step % 990000
			body := uv.Style{Fg: truecolor(190, 200, 210)}
			for i := 0; i < h; i++ {
				writeString(scr, 0, i,
					fmt.Sprintf("#%06d | ornek kayit %d | viewport cache", offset+i, offset+i), body)
			}
		}

	case "mouse-hit-test":
		return func(scr *uv.TerminalScreen, step int) {
			style := uv.Style{Fg: truecolor(120, 220, 100)}
			writeString(scr, 0, 0, "Mouse Target Area", style)
		}

	case "hundred-layers":
		return func(scr *uv.TerminalScreen, step int) {
			// Ultraviolet has no layer stack; the closest equivalent is drawing
			// 100 overlapping boxes in order, which is what a Bubble Tea app
			// would do by hand.
			// Positions advance with the step, mirroring the Limoni runner, so
			// the diff has real work to do on every frame rather than
			// short-circuiting on identical content.
			for layer := 0; layer < 100; layer++ {
				style := uv.Style{Fg: truecolor(uint8(layer*2%256), 180, 200)}
				x := (layer + step) % max(1, w-12)
				y := (layer + step) % max(1, h-2)
				writeString(scr, x, y, fmt.Sprintf("Layer %d", layer), style)
			}
		}

	case "resize":
		return func(scr *uv.TerminalScreen, step int) {
			writeString(scr, 0, 0,
				fmt.Sprintf("Size: %dx%d", 100+step%40, 30+step%10), uv.Style{})
		}

	case "async-update-burst":
		return func(scr *uv.TerminalScreen, step int) {
			writeString(scr, 0, 0, fmt.Sprintf("Value: %d", step), uv.Style{})
		}

	case "native-image-capability":
		return func(scr *uv.TerminalScreen, step int) {
			style := uv.Style{
				Fg: truecolor(100, 150, 200),
				Bg: truecolor(50, 100, 150),
			}
			fill(scr, w, h, "▄", style)
		}

	default:
		return func(scr *uv.TerminalScreen, step int) {
			writeString(scr, 0, 0, "Limoni benchmark ✓ 日本語", uv.Style{})
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func locateManifest() (string, []byte, error) {
	candidates := []string{
		"benchmarks/workloads.json",
		"../../workloads.json",
		"../../../workloads.json",
		"../../../benchmarks/workloads.json",
	}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			return path, data, nil
		}
	}
	return "", nil, fmt.Errorf("workloads.json not found in %v", candidates)
}

func main() {
	output := flag.String("output", "ultraviolet.json", "dashboard report path")
	flag.Parse()

	_, data, err := locateManifest()
	if err != nil {
		panic(err)
	}
	var specs []spec
	if err := json.Unmarshal(data, &specs); err != nil {
		panic(err)
	}

	hash := sha256.Sum256(data)
	gitCommit := "unknown"
	if out, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		gitCommit = strings.TrimSpace(string(out))
	}

	result := report{
		Implementation: "ultraviolet",
		Environment: envMetadata{
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
			Go:            runtime.Version(),
			Output:        "memory",
			ManifestHash:  hex.EncodeToString(hash[:]),
			GitCommit:     gitCommit,
			RunnerVersion: "v3.0.0",
			WarmupCount:   warmupIterations,
			BuildMode:     "release",
			Measures:      "cells -> ultraviolet TerminalScreen diff -> ANSI bytes",
			Target:        ultravioletVersion(),
		},
		Valid: true,
	}

	for _, item := range specs {
		w, h := int(item.Width), int(item.Height)
		if w == 0 {
			w = 120
		}
		if h == 0 {
			h = 40
		}

		var sink bytes.Buffer
		// TerminalScreen is the layer Bubble Tea v2 actually drives. Using it
		// rather than TerminalRenderer means the harness never touches
		// Ultraviolet's internal dirty tracking, so what is measured is the
		// library's real behaviour and not the harness's idea of it.
		screen := uv.NewTerminalScreen(&sink, uv.Environ{
			"TERM=xterm-256color",
			"COLORTERM=truecolor",
		})
		screen.Resize(w, h)
		// Limoni's runner diffs with trueColor and colors256 both enabled, so
		// pin the same profile here rather than letting environment sniffing
		// decide. Without this the emitted byte counts are not comparable.
		screen.SetColorProfile(colorprofile.TrueColor)

		draw := sceneFor(item)

		render := func(step int) int {
			sink.Reset()
			draw(screen, step)
			// Render composes the frame and diffs it; Flush is what actually
			// writes the resulting bytes to the writer.
			screen.Render()
			_ = screen.Flush()
			return sink.Len()
		}

		for i := 0; i < warmupIterations; i++ {
			render(i)
		}

		iterations := item.Iterations
		if iterations <= 0 {
			iterations = 1000
		}
		durations := make([]int64, iterations)
		var totalBytes int64

		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			n := render(i)
			durations[i] = time.Since(start).Nanoseconds()
			totalBytes += int64(n)
		}

		runtime.ReadMemStats(&after)

		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		var sum int64
		for _, d := range durations {
			sum += d
		}
		mean := sum / int64(len(durations))

		var variance float64
		for _, d := range durations {
			diff := float64(d - mean)
			variance += diff * diff
		}

		result.Workloads = append(result.Workloads, workload{
			Spec: item,
			Summary: summary{
				Frames:        iterations,
				P50NS:         durations[len(durations)*50/100],
				P95NS:         durations[len(durations)*95/100],
				P99NS:         durations[min(len(durations)*99/100, len(durations)-1)],
				MinNS:         durations[0],
				MaxNS:         durations[len(durations)-1],
				MeanNS:        mean,
				StdDevNS:      int64(math.Sqrt(variance / float64(len(durations)))),
				BytesPerFrame: float64(totalBytes) / float64(iterations),
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
	fmt.Fprintf(os.Stderr, "wrote %s (%d workloads)\n", *output, len(result.Workloads))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
