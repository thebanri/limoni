package main

import (
	"context"
	"fmt"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

type tickMsg time.Time

type wasmAppModel struct {
	angle float64
	count int
}

func (m *wasmAppModel) Init() []runtime.Cmd {
	return []runtime.Cmd{
		func(ctx context.Context) runtime.Msg {
			time.Sleep(50 * time.Millisecond)
			return tickMsg(time.Now())
		},
	}
}

func (m *wasmAppModel) Update(msg runtime.Msg) runtime.UpdateResult {
	switch msg := msg.(type) {
	case tickMsg:
		m.angle += 3.0
		if m.angle >= 360.0 {
			m.angle -= 360.0
		}
		return runtime.UpdateResult{
			Redraw: true,
			Commands: []runtime.Cmd{
				func(ctx context.Context) runtime.Msg {
					time.Sleep(50 * time.Millisecond)
					return tickMsg(time.Now())
				},
			},
		}

	case runtime.KeyPressMsg:
		if msg.Key.Ch == ' ' || msg.Key.Type == backend.KeyEnter {
			m.count++
			return runtime.UpdateResult{Redraw: true}
		}
	}

	return runtime.UpdateResult{}
}

func (m *wasmAppModel) View(frame *terminal.Frame) {
	area := frame.Area()
	if area.Width == 0 || area.Height == 0 {
		return
	}

	// 1. Header Block
	headerBlock := widgets.Block{
		Title:       " Limoni WebAssembly Browser Demo ",
		BorderStyle: cell.Style{Fg: cell.NewColorRGB(120, 220, 100)},
		TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}
	frame.RenderWidget(headerBlock, area)

	// 2. Info Paragraph
	inner := headerBlock.Inner(area)
	infoText := fmt.Sprintf(
		"Running on WebAssembly (GOOS=js GOARCH=wasm)!\n"+
			"• Real-time 3D Braille Canvas rotation: %.0f°\n"+
			"• Space/Enter press count: %d\n"+
			"• Zero-allocation double-buffered diff engine active.",
		m.angle, m.count,
	)
	p := &widgets.Paragraph{
		Text:  infoText,
		Style: cell.Style{Fg: cell.NewColorRGB(200, 200, 255)},
		Wrap:  true,
	}

	infoWidth := uint16(45)
	if inner.Width > 50 {
		frame.RenderWidget(p, cell.NewRect(inner.X+1, inner.Y+1, infoWidth, inner.Height-2))
	}

	// 3. 3D Rotating Cube on Canvas
	canvasX := inner.X + infoWidth + 2
	if canvasX < inner.X+inner.Width {
		canvasW := inner.X + inner.Width - canvasX - 1
		canvasH := inner.Height - 2
		if canvasW > 10 && canvasH > 5 {
			canvas := widgets.NewCanvas(canvasW, canvasH)

			// Simple 3D cube vertices
			v := []graphics.Vertex3D{
				{X: -1, Y: -1, Z: -1}, {X: 1, Y: -1, Z: -1},
				{X: 1, Y: 1, Z: -1}, {X: -1, Y: 1, Z: -1},
				{X: -1, Y: -1, Z: 1}, {X: 1, Y: -1, Z: 1},
				{X: 1, Y: 1, Z: 1}, {X: -1, Y: 1, Z: 1},
			}

			// Rotate vertices
			rotV := make([]graphics.Vertex3D, len(v))
			for i, vert := range v {
				rotV[i] = vert.RotateX(m.angle * 0.7).RotateY(m.angle)
			}

			// Project and draw edges
			screenW := float64(canvasW) * 2.0
			screenH := float64(canvasH) * 4.0
			cubeStyle := cell.Style{Fg: cell.NewColorRGB(0, 220, 255)}

			edges := [][2]int{
				{0, 1}, {1, 2}, {2, 3}, {3, 0},
				{4, 5}, {5, 6}, {6, 7}, {7, 4},
				{0, 4}, {1, 5}, {2, 6}, {3, 7},
			}

			for _, e := range edges {
				x1, y1, v1 := graphics.Project(rotV[e[0]], screenW, screenH, 3.5, screenH*0.8)
				x2, y2, v2 := graphics.Project(rotV[e[1]], screenW, screenH, 3.5, screenH*0.8)
				if v1 && v2 {
					canvas.DrawLine(int(x1), int(y1), int(x2), int(y2), cubeStyle)
				}
			}

			frame.RenderWidget(canvas, cell.NewRect(canvasX, inner.Y+1, canvasW, canvasH))
		}
	}
}

func main() {
	model := &wasmAppModel{}
	app := runtime.New(
		runtime.WithModel(model),
		runtime.WithFPS(30),
	)

	if err := app.Run(context.Background()); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
