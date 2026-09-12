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
// structurally unfair. This runner fixes that for v2: it drives Ultraviolet's
// TerminalRenderer — the cell-based, ncurses-derived diffing layer that Bubble
// Tea v2 and Lip Gloss v2 are both built on — so the measured pipeline is
// cells in, diffed ANSI bytes out, exactly like Limoni's runner.
//
// The renderer writes into an in-memory buffer, not a PTY, so no terminal is
// required and no I/O latency is included on either side.
//
//	go run . -output ../../../benchmark-results/bubbletea-v2.json
//
// # STATUS: results are NOT yet publishable
//
// This runner builds and runs against real Bubble Tea v2 / Ultraviolet, but two
// harness-level questions are unresolved, and until they are, no number
// produced here belongs in the README:
//
//  1. Touch tracking. Ultraviolet's renderer early-returns when no line is
//     marked touched, and the marks are reset by TerminalScreen — a layer this
//     harness bypasses. Resetting them here (see resetTouched below) takes an
//     unchanged frame from ~195us to ~42ns in isolation, which is the same
//     order as Limoni's clean-frame fast path. So the first version of this
//     runner overstated v2's cost by roughly four orders of magnitude on the
//     sparse workloads.
//
//  2. Clear() semantics. This harness clears the buffer every frame, mirroring
//     the Limoni runner. Limoni's Buffer.Clear short-circuits via a clean flag;
//     Ultraviolet's does not, so clearing re-dirties every line and defeats the
//     early return. Whether that is a genuine architectural difference or an
//     artifact of driving TerminalRenderer instead of TerminalScreen is not yet
//     established. A Bubble Tea v2 application does not clear a RenderBuffer by
//     hand, so the pattern may simply be non-idiomatic.
//
// Resolving this most likely means rewriting the harness against
// uv.TerminalScreen, which is the layer Bubble Tea v2 actually drives.
//
// Until then: treat the output as a work in progress, not as evidence.
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
	"sort"
	"strings"
	"time"

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
type scene func(buf *uv.RenderBuffer, step int)

// writeString writes s starting at (x, y) with the given style.
func writeString(buf *uv.RenderBuffer, x, y int, s string, style uv.Style) {
	col := x
	for _, r := range s {
		cell := uv.NewCell(ansi.GraphemeWidth, string(r))
		cell.Style = style
		buf.SetCell(col, y, cell)
		col += cell.Width
	}
}

func fill(buf *uv.RenderBuffer, w, h int, content string, style uv.Style) {
	cell := uv.NewCell(ansi.GraphemeWidth, content)
	cell.Style = style
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := *cell
			buf.SetCell(x, y, &c)
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
		return func(buf *uv.RenderBuffer, step int) {}

	case "full-redraw-120x40":
		return func(buf *uv.RenderBuffer, step int) {
			ch := string(rune('A' + (step % 26)))
			style := uv.Style{
				Fg: truecolor(uint8((step*3)%256), 200, 120),
				Bg: truecolor(20, 20, uint8(step%256)),
			}
			fill(buf, w, h, ch, style)
		}

	case "single-cell-update":
		return func(buf *uv.RenderBuffer, step int) {
			writeString(buf, 0, 0, "Initial frame state", uv.Style{})
			glyph := "Y"
			if step%2 == 0 {
				glyph = "X"
			}
			writeString(buf, 0, 0, glyph, uv.Style{})
		}

	case "text-heavy-120x40":
		line := "Limoni benchmark ✓ 日本語. Heavy text rendering test for performance analysis."
		return func(buf *uv.RenderBuffer, step int) {
			style := uv.Style{Fg: truecolor(200, 200, 255)}
			for y := 0; y < h; y++ {
				writeString(buf, 0, y, line, style)
			}
		}

	case "unicode-emoji":
		line := "Unicode emoji test: 🚀 🍎 🦊 💻 🌟 日本語. Multibyte CJK and complex symbols."
		return func(buf *uv.RenderBuffer, step int) {
			style := uv.Style{Fg: truecolor(255, 200, 120)}
			for y := 0; y < h; y++ {
				writeString(buf, 0, y, line, style)
			}
		}

	case "table-10000":
		return func(buf *uv.RenderBuffer, step int) {
			offset := step % 9900
			header := uv.Style{Fg: truecolor(255, 255, 255), Attrs: 1}
			body := uv.Style{Fg: truecolor(190, 200, 210)}
			writeString(buf, 0, 0, "ID        Name            Status", header)
			for i := 1; i < h; i++ {
				writeString(buf, 0, i,
					fmt.Sprintf("%-9d %-15s %s", offset+i, "process", "running"), body)
			}
		}

	case "virtual-1000000":
		return func(buf *uv.RenderBuffer, step int) {
			offset := step % 990000
			body := uv.Style{Fg: truecolor(190, 200, 210)}
			for i := 0; i < h; i++ {
				writeString(buf, 0, i,
					fmt.Sprintf("#%06d | ornek kayit %d | viewport cache", offset+i, offset+i), body)
			}
		}

	case "mouse-hit-test":
		return func(buf *uv.RenderBuffer, step int) {
			style := uv.Style{Fg: truecolor(120, 220, 100)}
			writeString(buf, 0, 0, "Mouse Target Area", style)
		}

	case "hundred-layers":
		return func(buf *uv.RenderBuffer, step int) {
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
				writeString(buf, x, y, fmt.Sprintf("Layer %d", layer), style)
			}
		}

	case "resize":
		return func(buf *uv.RenderBuffer, step int) {
			writeString(buf, 0, 0,
				fmt.Sprintf("Size: %dx%d", 100+step%40, 30+step%10), uv.Style{})
		}

	case "async-update-burst":
		return func(buf *uv.RenderBuffer, step int) {
			writeString(buf, 0, 0, fmt.Sprintf("Value: %d", step), uv.Style{})
		}

	case "native-image-capability":
		return func(buf *uv.RenderBuffer, step int) {
			style := uv.Style{
				Fg: truecolor(100, 150, 200),
				Bg: truecolor(50, 100, 150),
			}
			fill(buf, w, h, "▄", style)
		}

	default:
		return func(buf *uv.RenderBuffer, step int) {
			writeString(buf, 0, 0, "Limoni benchmark ✓ 日本語", uv.Style{})
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
	output := flag.String("output", "bubbletea-v2.json", "dashboard report path")
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
		Implementation: "bubbletea-v2",
		Environment: envMetadata{
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
			Go:            runtime.Version(),
			Output:        "memory",
			ManifestHash:  hex.EncodeToString(hash[:]),
			GitCommit:     gitCommit,
			RunnerVersion: "v2.0.0",
			WarmupCount:   warmupIterations,
			BuildMode:     "release",
			Measures:      "cells -> ultraviolet TerminalRenderer diff -> ANSI bytes",
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
		// A truecolor-capable environment, matching Limoni's diff settings.
		renderer := uv.NewTerminalRenderer(&sink, []string{"TERM=xterm-256color", "COLORTERM=truecolor"})
		renderer.Resize(w, h)

		buf := uv.NewRenderBuffer(w, h)
		draw := sceneFor(item)

		// Ultraviolet's TerminalRenderer early-returns when no line is marked
		// touched, and RenderBuffer.SetCell only touches a line when the cell
		// actually changes — that is its equivalent of Limoni's clean-frame
		// fast path. The touch marks are reset by the higher-level
		// TerminalScreen, not by Render, so a harness driving the renderer
		// directly must reset them itself. Without this the renderer re-scans
		// every line on every frame and an unchanged frame costs ~195 us
		// instead of ~42 ns, which would overstate v2's cost by four orders of
		// magnitude on the sparse workloads.
		resetTouched := func() {
			for i := range buf.Touched {
				buf.Touched[i] = nil
			}
		}

		render := func(step int) int {
			sink.Reset()
			buf.Clear()
			draw(buf, step)
			renderer.Render(buf)
			_ = renderer.Flush()
			resetTouched()
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
