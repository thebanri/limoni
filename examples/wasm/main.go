// Limoni's WebAssembly playground.
//
// One binary hosts every scene, switched with the number keys or Tab, so the
// browser downloads a single ~3 MB module instead of one per demo.
//
//	GOOS=js GOARCH=wasm go build -o limoni.wasm ./examples/wasm
//
// The page in index.html wires xterm.js to the JS bridge in
// core/driver/backend_wasm.go: Limoni writes ANSI, xterm.js interprets it, and
// keystrokes come back through window.__limoni_input.
package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/engine"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

type tickMsg time.Time

// scene is one switchable demo.
type scene struct {
	title string
	view  func(m *playground, frame *terminal.Frame, area cell.Rect)
}

type playground struct {
	active int
	scenes []scene

	// Animation and interaction state, shared across scenes.
	angle    float64
	count    int
	frame    int
	started  time.Time
	series   []float64
	logState *widgets.ViewportState
	logLines []string
}

var (
	accent = cell.Style{Fg: cell.NewColorRGB(0, 255, 170)}
	text   = cell.Style{Fg: cell.NewColorRGB(201, 209, 217)}
	muted  = cell.Style{Fg: cell.NewColorRGB(139, 148, 158)}
	cyan   = cell.Style{Fg: cell.NewColorRGB(0, 220, 255)}
)

func newPlayground() *playground {
	m := &playground{
		started:  time.Now(),
		logState: widgets.NewViewportState(),
		series:   make([]float64, 0, 120),
		logLines: make([]string, 0, 200),
	}
	for i := 0; i < 200; i++ {
		m.logLines = append(m.logLines, fmt.Sprintf(
			"%04d  %s  limoni/core/buffer: diffed %d dirty cells in %d ns",
			i+1,
			[]string{"INFO ", "DEBUG", "WARN "}[i%3],
			(i*37)%4800,
			1900+(i*13)%6000,
		))
	}
	m.scenes = []scene{
		{title: "3D", view: (*playground).viewCube},
		{title: "Charts", view: (*playground).viewCharts},
		{title: "Scroll", view: (*playground).viewScroll},
		{title: "About", view: (*playground).viewAbout},
	}
	return m
}

func (m *playground) Init() []engine.Cmd {
	return []engine.Cmd{tick}
}

func tick(ctx context.Context) engine.Msg {
	select {
	case <-ctx.Done():
		return nil
	case t := <-time.After(50 * time.Millisecond):
		return tickMsg(t)
	}
}

func (m *playground) Update(msg engine.Msg) engine.UpdateResult {
	switch msg := msg.(type) {
	case tickMsg:
		m.frame++
		m.angle = math.Mod(m.angle+3, 360)

		// Feed the chart with a slowly drifting signal.
		t := float64(m.frame) * 0.08
		value := 50 + 35*math.Sin(t) + 10*math.Sin(t*2.7)
		m.series = append(m.series, value)
		if len(m.series) > 120 {
			m.series = m.series[len(m.series)-120:]
		}

		return engine.UpdateResult{Redraw: true, Commands: []engine.Cmd{tick}}

	case engine.KeyPressMsg:
		return m.handleKey(msg.Key)

	case engine.MouseWheelMsg:
		if m.scenes[m.active].title == "Scroll" {
			m.logState.ScrollBy(msg.DeltaY * 3)
			return engine.UpdateResult{Redraw: true}
		}
	}
	return engine.UpdateResult{}
}

func (m *playground) handleKey(key driver.KeyEvent) engine.UpdateResult {
	switch key.Type {
	case driver.KeyTab:
		if key.Shift {
			m.active = (m.active - 1 + len(m.scenes)) % len(m.scenes)
		} else {
			m.active = (m.active + 1) % len(m.scenes)
		}
		return engine.UpdateResult{Redraw: true}

	case driver.KeyArrowUp:
		m.logState.ScrollBy(-1)
		return engine.UpdateResult{Redraw: true}
	case driver.KeyArrowDown:
		m.logState.ScrollBy(1)
		return engine.UpdateResult{Redraw: true}
	case driver.KeyPageUp:
		m.logState.ScrollBy(-10)
		return engine.UpdateResult{Redraw: true}
	case driver.KeyPageDown:
		m.logState.ScrollBy(10)
		return engine.UpdateResult{Redraw: true}
	case driver.KeyHome:
		m.logState.GotoTop()
		return engine.UpdateResult{Redraw: true}
	case driver.KeyEnd:
		m.logState.GotoBottom()
		return engine.UpdateResult{Redraw: true}

	case driver.KeyEnter, driver.KeySpace:
		m.count++
		return engine.UpdateResult{Redraw: true}

	case driver.KeyRune:
		if key.Ch >= '1' && key.Ch <= '9' {
			if index := int(key.Ch - '1'); index < len(m.scenes) {
				m.active = index
				return engine.UpdateResult{Redraw: true}
			}
		}
		if key.Ch == ' ' {
			m.count++
			return engine.UpdateResult{Redraw: true}
		}
	}
	return engine.UpdateResult{}
}

func (m *playground) View(frame *terminal.Frame) {
	area := frame.Area()
	if area.Width < 20 || area.Height < 6 {
		return
	}

	// Tab bar, body, status line.
	tabsArea := cell.NewRect(area.X, area.Y, area.Width, 1)
	statusArea := cell.NewRect(area.X, area.Y+area.Height-1, area.Width, 1)
	bodyArea := cell.NewRect(area.X, area.Y+1, area.Width, area.Height-2)

	titles := make([]string, len(m.scenes))
	for i, s := range m.scenes {
		titles[i] = fmt.Sprintf("%d %s", i+1, s.title)
	}
	frame.RenderWidget(widgets.Tabs{
		Titles:        titles,
		Selected:      m.active,
		SelectedStyle: accent,
		DividerStyle:  muted,
		OnSelect:      func(i int) { m.active = i },
	}, tabsArea)

	m.scenes[m.active].view(m, frame, bodyArea)

	frame.RenderWidget(widgets.Spinner{
		Set:        widgets.SpinnerBraille,
		Since:      m.started,
		Label:      "  [1-4] scene   [Tab] next   [↑↓ PgUp/PgDn] scroll   [Space] count: " + fmt.Sprint(m.count),
		Style:      accent,
		LabelStyle: muted,
	}, statusArea)
}

// viewCube keeps the original rotating wireframe demo.
func (m *playground) viewCube(frame *terminal.Frame, area cell.Rect) {
	block := widgets.Block{
		Title:       " 3D Wireframe — software rasteriser, zero allocations ",
		BorderStyle: accent,
		TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}
	frame.RenderWidget(block, area)
	inner := block.Inner(area)
	if inner.Width < 20 || inner.Height < 4 {
		return
	}

	infoWidth := uint16(44)
	if inner.Width > 56 {
		frame.RenderWidget(&widgets.Paragraph{
			Text: fmt.Sprintf(
				"Compiled with GOOS=js GOARCH=wasm and driven by\n"+
					"xterm.js in this browser tab.\n\n"+
					"• Rotation: %.0f°\n"+
					"• Frames rendered: %d\n"+
					"• Braille canvas: 2×4 subpixels per cell\n"+
					"• Diff engine: double-buffered, 0 B/op\n\n"+
					"Every frame goes through the same ANSI diff\n"+
					"that runs in a real terminal — nothing here is\n"+
					"a browser-specific rendering path.",
				m.angle, m.frame,
			),
			Style: text,
			Wrap:  true,
		}, cell.NewRect(inner.X+1, inner.Y+1, infoWidth, inner.Height-2))
	} else {
		infoWidth = 0
	}

	canvasX := inner.X + infoWidth + 2
	if canvasX >= inner.X+inner.Width {
		return
	}
	canvasW := inner.X + inner.Width - canvasX - 1
	canvasH := inner.Height - 2
	if canvasW <= 10 || canvasH <= 5 {
		return
	}

	canvas := widgets.NewCanvas(canvasW, canvasH)
	vertices := []graphics.Vertex3D{
		{X: -1, Y: -1, Z: -1}, {X: 1, Y: -1, Z: -1},
		{X: 1, Y: 1, Z: -1}, {X: -1, Y: 1, Z: -1},
		{X: -1, Y: -1, Z: 1}, {X: 1, Y: -1, Z: 1},
		{X: 1, Y: 1, Z: 1}, {X: -1, Y: 1, Z: 1},
	}
	rotated := make([]graphics.Vertex3D, len(vertices))
	for i, v := range vertices {
		rotated[i] = v.RotateX(m.angle * 0.7).RotateY(m.angle)
	}

	screenW := float64(canvasW) * 2.0
	screenH := float64(canvasH) * 4.0
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		{4, 5}, {5, 6}, {6, 7}, {7, 4},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	}
	for _, e := range edges {
		x1, y1, ok1 := graphics.Project(rotated[e[0]], screenW, screenH, 3.5, screenH*0.8)
		x2, y2, ok2 := graphics.Project(rotated[e[1]], screenW, screenH, 3.5, screenH*0.8)
		if ok1 && ok2 {
			canvas.DrawLine(int(x1), int(y1), int(x2), int(y2), cyan)
		}
	}
	frame.RenderWidget(canvas, cell.NewRect(canvasX, inner.Y+1, canvasW, canvasH))
}

// viewCharts streams a live signal through the chart widgets.
func (m *playground) viewCharts(frame *terminal.Frame, area cell.Rect) {
	block := widgets.Block{
		Title:       " Live Charts — Braille line, bars, sparkline ",
		BorderStyle: accent,
		TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}
	frame.RenderWidget(block, area)
	inner := block.Inner(area)
	if inner.Width < 24 || inner.Height < 8 || len(m.series) < 2 {
		return
	}

	sparkHeight := uint16(3)
	barHeight := uint16(0)
	if inner.Height > 16 {
		barHeight = 8
	}
	lineHeight := inner.Height - sparkHeight - barHeight - 1
	if lineHeight < 4 {
		return
	}

	y := inner.Y + 1
	frame.RenderWidget(widgets.LineChart{
		Datasets: []widgets.LineDataset{{
			Name:  "signal",
			Data:  m.series,
			Color: cell.NewColorRGB(0, 220, 255),
		}},
		MinY:      0,
		MaxY:      100,
		ShowAxes:  true,
		ShowGrid:  true,
		Style:     text,
		AxisStyle: muted,
	}, cell.NewRect(inner.X+1, y, inner.Width-2, lineHeight))
	y += lineHeight

	frame.RenderWidget(widgets.Sparkline{
		Data:  m.series,
		Color: cell.NewColorRGB(0, 255, 170),
		Style: accent,
	}, cell.NewRect(inner.X+1, y, inner.Width-2, sparkHeight))
	y += sparkHeight

	if barHeight > 0 {
		latest := m.series[len(m.series)-1]
		frame.RenderWidget(widgets.BarChart{
			Data: []widgets.BarData{
				{Label: "now", Value: latest, Color: cell.NewColorRGB(0, 255, 170)},
				{Label: "avg", Value: average(m.series), Color: cell.NewColorRGB(0, 180, 255)},
				{Label: "max", Value: maxOf(m.series), Color: cell.NewColorRGB(255, 200, 0)},
				{Label: "min", Value: minOf(m.series), Color: cell.NewColorRGB(255, 110, 110)},
			},
			Max:        100,
			BarWidth:   6,
			BarGap:     2,
			ShowValues: true,
			Style:      text,
			LabelStyle: muted,
		}, cell.NewRect(inner.X+1, y, inner.Width-2, barHeight))
	}
}

// viewScroll shows Viewport + Scrollbar over a long log.
func (m *playground) viewScroll(frame *terminal.Frame, area cell.Rect) {
	block := widgets.Block{
		Title:       " Viewport + Scrollbar — 200 lines, wheel and keys work ",
		BorderStyle: accent,
		TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}
	frame.RenderWidget(block, area)
	inner := block.Inner(area)
	if inner.Width < 10 || inner.Height < 3 {
		return
	}

	frame.RenderWidget(widgets.Viewport{
		Child: &widgets.Paragraph{
			Text:  strings.Join(m.logLines, "\n"),
			Style: text,
		},
		State:     m.logState,
		Scrollbar: true,
	}, inner)
}

// viewAbout is the pitch: what this is and where to go next.
func (m *playground) viewAbout(frame *terminal.Frame, area cell.Rect) {
	block := widgets.Block{
		Title:       " About Limoni ",
		BorderStyle: accent,
		TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}
	frame.RenderWidget(block, area)
	inner := block.Inner(area)
	if inner.Width < 20 || inner.Height < 4 {
		return
	}

	frame.RenderWidget(&widgets.Paragraph{
		Text: "Limoni is a terminal UI engine for Go.\n\n" +
			"What you are looking at is the same engine that runs in a real\n" +
			"terminal, compiled to WebAssembly and pointed at xterm.js. The\n" +
			"application code is unchanged: one driver swap, no rendering\n" +
			"special-cases.\n\n" +
			"• Flat 1D cell grid with a double-buffered ANSI diff\n" +
			"• Zero heap allocations on the render hot path\n" +
			"• Built-in 3D rasteriser, charts, markdown and image protocols\n" +
			"• Two application models: immediate mode and Elm architecture\n" +
			"• Two external dependencies, both golang.org/x\n\n" +
			"  go get github.com/thebanri/limoni\n" +
			"  github.com/thebanri/limoni",
		Style: text,
		Wrap:  true,
	}, cell.NewRect(inner.X+2, inner.Y+1, inner.Width-4, inner.Height-2))
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func maxOf(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	out := values[0]
	for _, v := range values {
		if v > out {
			out = v
		}
	}
	return out
}

func minOf(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	out := values[0]
	for _, v := range values {
		if v < out {
			out = v
		}
	}
	return out
}

func main() {
	// RunTerminal is the entry point that actually renders: Run alone only
	// drives the message loop and never draws a frame.
	backend := driver.NewBackend(nil, nil)
	term, err := terminal.New(backend)
	if err != nil {
		fmt.Printf("limoni: terminal setup failed: %v\n", err)
		return
	}

	app := engine.New(
		engine.WithModel(newPlayground()),
		engine.WithFPS(30),
	)
	if err := app.RunTerminal(context.Background(), term, backend); err != nil {
		fmt.Printf("limoni: %v\n", err)
	}
}
