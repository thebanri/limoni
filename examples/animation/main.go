package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type AppState struct {
	SidebarWidth *animation.Float
	SidebarOpen  bool

	ButtonColor *animation.Color
	ColorIndex  int

	BounceVal *animation.Float
	Bouncing  bool

	LastKey   string
	LastMouse string
	FPS       float64
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing backend: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing terminal: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	state := &AppState{
		SidebarWidth: animation.NewFloat(6),
		SidebarOpen:  false,
		ButtonColor:  animation.NewColor(cell.NewColorRGB(0, 100, 255)), // Start with blue
		ColorIndex:   0,
		BounceVal:    animation.NewFloat(0),
		Bouncing:     false,
		LastKey:      "None",
		LastMouse:    "None",
	}

	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	frameCount := 0
	lastFpsCalc := time.Now()

	drawApp(t, state)

	for {
		select {
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
					return
				}

				if ev.Key.Type == backend.KeySpace {
					state.SidebarOpen = !state.SidebarOpen
					target := 6.0
					if state.SidebarOpen {
						target = 24.0
					}
					state.SidebarWidth.AnimateTo(target, 400*time.Millisecond, animation.EaseInOutCubic)
				}

				if ev.Key.Type == backend.KeyEnter {
					state.ColorIndex = (state.ColorIndex + 1) % 4
					var targetColor cell.Color
					switch state.ColorIndex {
					case 0:
						targetColor = cell.NewColorRGB(0, 100, 255) // Blue
					case 1:
						targetColor = cell.NewColorRGB(255, 0, 100) // Pink
					case 2:
						targetColor = cell.NewColorRGB(0, 255, 100) // Green
					case 3:
						targetColor = cell.NewColorRGB(255, 180, 0) // Orange
					}
					state.ButtonColor.AnimateTo(targetColor, 500*time.Millisecond, animation.EaseInOutQuad)
				}

				if ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'b' {
					state.BounceVal.SetValue(0)
					state.BounceVal.AnimateTo(10, 1000*time.Millisecond, animation.EaseOutBounce)
				}

				state.LastKey = fmt.Sprintf("Code: %d, Char: %q", ev.Key.Type, string(ev.Key.Ch))

			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)
				state.LastMouse = fmt.Sprintf("Btn: %d, Pos: (%d, %d)", ev.Mouse.Button, ev.Mouse.X, ev.Mouse.Y)

			case backend.EventResize:
				// Automatically redrawn on next frame
			}

		case <-ticker.C:
			now := time.Now()
			state.SidebarWidth.Update(now)
			state.ButtonColor.Update(now)
			state.BounceVal.Update(now)

			drawApp(t, state)

			frameCount++
			if time.Since(lastFpsCalc) >= 1*time.Second {
				state.FPS = float64(frameCount) / time.Since(lastFpsCalc).Seconds()
				frameCount = 0
				lastFpsCalc = time.Now()
			}
		}
	}
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		rootLay := layout.NewFlexLayout(
			layout.Vertical,
			0,
			layout.Fixed(3),
			layout.Fill(),
			layout.Fixed(1),
		)
		chunks := rootLay.Split(f.Buffer.Area)

		// 1. Header
		f.RenderWidget(widgets.Block{
			Title:          " LIMONI TUI - ANIMATION ENGINE SHOWCASE ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 255)},
			Child:          label{text: " Time-based interpolation, fluid transitions, and easing curves ", style: cell.Style{Fg: cell.NewColorRGB(255, 255, 255)}},
		}, chunks[0])

		// 2. Body
		sidebarW := uint16(math.Round(state.SidebarWidth.Value()))
		bodyLay := layout.NewFlexLayout(
			layout.Horizontal,
			1,
			layout.Fixed(sidebarW),
			layout.Fill(),
		)
		bodyChunks := bodyLay.Split(chunks[1])

		// Left Panel (Sidebar)
		sidebarTitle := "MENU"
		if sidebarW >= 10 {
			sidebarTitle = " QUICK ACCESS "
		}
		sidebarText := "...\n...\n..."
		if sidebarW >= 15 {
			sidebarText = "System Active\nSolid 60 FPS\nRGB Interpolation\nFluid Animations"
		}

		sidebarBlock := widgets.Block{
			Title:         sidebarTitle,
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 120, 120)},
			PaddingLeft:   1,
			PaddingTop:    1,
			Child:         label{text: sidebarText, style: cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}},
		}
		f.RenderWidget(sidebarBlock, bodyChunks[0])

		// Right Panel
		rightLay := layout.NewFlexLayout(
			layout.Vertical,
			1,
			layout.Percentage(50),
			layout.Percentage(50),
		)
		rightChunks := rightLay.Split(bodyChunks[1])

		// Top: Color Animation
		btnCol := state.ButtonColor.Value()
		colorBox := widgets.Block{
			Title:          " COLOR TRANSITION (Blend / Fade) ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: btnCol},
			PaddingLeft:    2,
			PaddingTop:     1,
			Child: &widgets.Paragraph{
				Text:  fmt.Sprintf("The border color dynamically transitions using quadratic easing.\n\nPress [Enter] or click this card to trigger color change.\nActive Color (RGB): %+v\nLast Events - Key: %s | Mouse: %s", btnCol, state.LastKey, state.LastMouse),
				Style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
				Wrap:  true,
			},
		}
		f.RenderWidget(colorBox, rightChunks[0])

		// Bottom: Bounce Animation
		bounceOffset := int(math.Round(state.BounceVal.Value()))
		bounceText := ""
		for i := 0; i < bounceOffset; i++ {
			bounceText += "\n"
		}
		bounceText += "  BOUNCING BOX (Press 'b' or click this card to trigger bounce)"

		bounceBox := widgets.Block{
			Title:          " BOUNCE EFFECT (EaseOutBounce) ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 0, 255)},
			PaddingLeft:    2,
			Child:          label{text: bounceText, style: cell.Style{Fg: cell.NewColorRGB(0, 255, 255)}},
		}
		f.RenderWidget(bounceBox, rightChunks[1])

		// Click Handlers
		f.RegisterClickHandler(bodyChunks[0], func(ev backend.MouseEvent) {
			state.SidebarOpen = !state.SidebarOpen
			target := 6.0
			if state.SidebarOpen {
				target = 24.0
			}
			state.SidebarWidth.AnimateTo(target, 400*time.Millisecond, animation.EaseInOutCubic)
		})

		f.RegisterClickHandler(rightChunks[0], func(ev backend.MouseEvent) {
			state.ColorIndex = (state.ColorIndex + 1) % 4
			var targetColor cell.Color
			switch state.ColorIndex {
			case 0:
				targetColor = cell.NewColorRGB(0, 100, 255)
			case 1:
				targetColor = cell.NewColorRGB(255, 0, 100)
			case 2:
				targetColor = cell.NewColorRGB(0, 255, 100)
			case 3:
				targetColor = cell.NewColorRGB(255, 180, 0)
			}
			state.ButtonColor.AnimateTo(targetColor, 500*time.Millisecond, animation.EaseInOutQuad)
		})

		f.RegisterClickHandler(rightChunks[1], func(ev backend.MouseEvent) {
			state.BounceVal.SetValue(0)
			state.BounceVal.AnimateTo(10, 1000*time.Millisecond, animation.EaseOutBounce)
		})

		// 3. Footer
		footerText := fmt.Sprintf(" [Space] Toggle Sidebar | [Enter] Cycle Color | [b] Bounce | [q] Quit | FPS: %.1f", state.FPS)
		footerBlock := widgets.Block{
			Borders: widgets.BorderNone,
			Style:   cell.Style{Fg: cell.NewColorRGB(140, 140, 140), Bg: cell.NewColorRGB(30, 30, 30)},
			Child:   label{text: footerText, style: cell.Style{Fg: cell.NewColorRGB(140, 140, 140), Bg: cell.NewColorRGB(30, 30, 30)}},
		}
		f.RenderWidget(footerBlock, chunks[2])
	})
}

type label struct {
	text  string
	style cell.Style
}

func (l label) Draw(ctx cell.Context, buf *buffer.Buffer) {
	mergedStyle := ctx.Style.Merge(l.style)
	currY := ctx.Area.Y
	lineStart := 0
	for i := 0; i < len(l.text); i++ {
		if l.text[i] == '\n' {
			if currY < ctx.Area.Y+ctx.Area.Height {
				buf.SetString(ctx.Area.X, currY, l.text[lineStart:i], mergedStyle)
				currY++
			}
			lineStart = i + 1
		}
	}
	if lineStart < len(l.text) && currY < ctx.Area.Y+ctx.Area.Height {
		buf.SetString(ctx.Area.X, currY, l.text[lineStart:], mergedStyle)
	}
}

func (l label) SizeHint(maxArea cell.Rect) (width, height uint16) {
	lines := 1
	maxW := 0
	currW := 0
	for i := 0; i < len(l.text); i++ {
		if l.text[i] == '\n' {
			lines++
			if currW > maxW {
				maxW = currW
			}
			currW = 0
		} else {
			currW++
		}
	}
	if currW > maxW {
		maxW = currW
	}
	return uint16(maxW), uint16(lines)
}
