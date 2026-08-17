package main

import (
	"fmt"
	"os"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type text struct {
	value string
	style cell.Style
}

func (t text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	buf.SetString(ctx.Area.X, ctx.Area.Y, t.value, ctx.Style.Merge(t.style))
}

func (t text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return uint16(len(t.value)), 1
}

// ─────────────────────────────────────────────────────────────────────────────
// DIAL KNOB CUSTOM WIDGET
// ─────────────────────────────────────────────────────────────────────────────

type DialKnob struct {
	ID           string
	Value        *int
	Min          int
	Max          int
	Style        cell.Style
	FocusedStyle cell.Style
}

func (dk DialKnob) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if dk.ID == "" || dk.Value == nil || ctx.Area.Width < 7 || ctx.Area.Height < 3 {
		return
	}

	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(dk.ID)
	}

	isFocused := ctx.FocusedID == dk.ID

	if ctx.RegisterClick != nil && ctx.SetFocus != nil {
		ctx.RegisterClick(ctx.Area, func() {
			ctx.SetFocus(dk.ID)
		})
	}

	activeStyle := ctx.Style.Merge(dk.Style)
	if isFocused {
		activeStyle = activeStyle.Merge(dk.FocusedStyle)
	}

	// Clear area
	for y := ctx.Area.Y; y < ctx.Area.Y+ctx.Area.Height; y++ {
		for x := ctx.Area.X; x < ctx.Area.X+ctx.Area.Width; x++ {
			if c := buf.Get(x, y); c != nil {
				c.Content = ' '
				c.Style = activeStyle
			}
		}
	}

	// Calculate pointer direction based on value
	percent := float64(*dk.Value-dk.Min) / float64(dk.Max-dk.Min)
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	// 8 pointer directions from 0 to 360 degrees
	directions := []rune{'─', '\\', '│', '/', '─', '\\', '│', '/'}
	dirIdx := int(percent * float64(len(directions)-1))

	centerX := ctx.Area.X + ctx.Area.Width/2
	centerY := ctx.Area.Y + ctx.Area.Height/2 - 1

	buf.SetString(centerX-2, centerY-1, "╭───╮", activeStyle)

	buf.SetString(centerX-2, centerY, "│", activeStyle)
	if c := buf.Get(centerX, centerY); c != nil {
		c.Content = directions[dirIdx]
		c.Style = activeStyle
		if isFocused {
			c.Style.Modifier |= cell.ModifierBold
		}
	}
	buf.SetString(centerX+2, centerY, "│", activeStyle)

	buf.SetString(centerX-2, centerY+1, "╰───╯", activeStyle)

	valStr := fmt.Sprintf("%d", *dk.Value)
	buf.SetString(centerX-uint16(len(valStr))/2, centerY+2, valStr, activeStyle)
}

func (dk DialKnob) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return 8, 4
}

// ─────────────────────────────────────────────────────────────────────────────
// COLOR PALETTE GRID CUSTOM WIDGET
// ─────────────────────────────────────────────────────────────────────────────

type ColorPaletteGrid struct {
	ID            string
	SelectedColor *cell.Color
	HoverIndexX   int
	HoverIndexY   int
	Style         cell.Style
	FocusedStyle  cell.Style
}

var paletteColors = [][]cell.Color{
	{
		cell.NewColorRGB(255, 0, 0),     // Red
		cell.NewColorRGB(255, 127, 0),   // Orange
		cell.NewColorRGB(255, 255, 0),   // Yellow
		cell.NewColorRGB(127, 255, 0),   // Lime
	},
	{
		cell.NewColorRGB(0, 255, 0),     // Green
		cell.NewColorRGB(0, 255, 255),   // Cyan
		cell.NewColorRGB(0, 127, 255),   // Sky Blue
		cell.NewColorRGB(0, 0, 255),     // Blue
	},
	{
		cell.NewColorRGB(127, 0, 255),   // Purple
		cell.NewColorRGB(255, 0, 255),   // Magenta
		cell.NewColorRGB(255, 0, 127),   // Pink
		cell.NewColorRGB(255, 255, 255), // White
	},
}

func (cp ColorPaletteGrid) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if cp.ID == "" || cp.SelectedColor == nil || ctx.Area.Width < 12 || ctx.Area.Height < 3 {
		return
	}

	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(cp.ID)
	}

	isFocused := ctx.FocusedID == cp.ID

	if ctx.RegisterClick != nil && ctx.SetFocus != nil {
		ctx.RegisterClick(ctx.Area, func() {
			ctx.SetFocus(cp.ID)
		})
	}

	cellW := uint16(3)
	cellH := uint16(1)

	for yIdx, row := range paletteColors {
		for xIdx, col := range row {
			cellArea := cell.NewRect(
				ctx.Area.X+uint16(xIdx)*4,
				ctx.Area.Y+uint16(yIdx),
				cellW,
				cellH,
			)

			sr, sg, sb := cp.SelectedColor.RGB()
			cr, cg, cb := col.RGB()
			isSelected := sr == cr && sg == cg && sb == cb

			isHovered := isFocused && cp.HoverIndexX == xIdx && cp.HoverIndexY == yIdx

			cellStyle := cell.Style{Bg: col, Fg: cell.NewColorRGB(0, 0, 0)}
			if isSelected {
				cellStyle.Modifier |= cell.ModifierBold
			}

			for y := cellArea.Y; y < cellArea.Y+cellArea.Height; y++ {
				for x := cellArea.X; x < cellArea.X+cellArea.Width; x++ {
					if c := buf.Get(x, y); c != nil {
						c.Style = cellStyle
						if isSelected {
							c.Content = '✔'
						} else {
							c.Content = ' '
						}
					}
				}
			}

			if isHovered {
				for x := cellArea.X; x < cellArea.X+cellArea.Width; x++ {
					if c := buf.Get(x, cellArea.Y); c != nil {
						c.Style.Modifier |= cell.ModifierReverse
					}
				}
			}

			if ctx.RegisterClick != nil {
				targetCol := col
				ctx.RegisterClick(cellArea, func() {
					if ctx.SetFocus != nil {
						ctx.SetFocus(cp.ID)
					}
					*cp.SelectedColor = targetCol
				})
			}
		}
	}
}

func (cp ColorPaletteGrid) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return 16, 3
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO MAIN LOGIC
// ─────────────────────────────────────────────────────────────────────────────

type AppState struct {
	DialValue     int
	SelectedColor cell.Color
	PaletteHoverX int
	PaletteHoverY int
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
		DialValue:     20,
		SelectedColor: cell.NewColorRGB(0, 255, 255),
		PaletteHoverX: 0,
		PaletteHoverY: 0,
	}

	t.FocusManager().SetFocused("knob_dial")

	draw := func() {
		drawApp(t, state)
	}

	draw()

	for ev := range b.Events() {
		switch ev.Type {
		case backend.EventKey:
			if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
				return
			}

			if ev.Key.Type == backend.KeyTab {
				t.FocusManager().Next()
				draw()
				break
			}

			focused := t.FocusManager().Focused()
			switch focused {
			case "knob_dial":
				if ev.Key.Type == backend.KeyArrowUp || ev.Key.Type == backend.KeyArrowRight {
					if state.DialValue < 100 {
						state.DialValue += 5
					}
				}
				if ev.Key.Type == backend.KeyArrowDown || ev.Key.Type == backend.KeyArrowLeft {
					if state.DialValue > 0 {
						state.DialValue -= 5
					}
				}

			case "color_grid":
				if ev.Key.Type == backend.KeyArrowUp {
					if state.PaletteHoverY > 0 {
						state.PaletteHoverY--
					}
				}
				if ev.Key.Type == backend.KeyArrowDown {
					if state.PaletteHoverY < 2 {
						state.PaletteHoverY++
					}
				}
				if ev.Key.Type == backend.KeyArrowLeft {
					if state.PaletteHoverX > 0 {
						state.PaletteHoverX--
					}
				}
				if ev.Key.Type == backend.KeyArrowRight {
					if state.PaletteHoverX < 3 {
						state.PaletteHoverX++
					}
				}
				if ev.Key.Type == backend.KeyEnter || ev.Key.Type == backend.KeySpace {
					state.SelectedColor = paletteColors[state.PaletteHoverY][state.PaletteHoverX]
				}
			}

			draw()

		case backend.EventMouse:
			t.RouteMouseEvent(ev.Mouse)

			focused := t.FocusManager().Focused()
			if focused == "knob_dial" {
				if ev.Mouse.Button == backend.MouseScrollUp {
					if state.DialValue < 100 {
						state.DialValue += 5
					}
				}
				if ev.Mouse.Button == backend.MouseScrollDown {
					if state.DialValue > 0 {
						state.DialValue -= 5
					}
				}
			}

			draw()

		case backend.EventResize:
			draw()
		}
	}
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		area := f.Buffer.Area

		rootLay := layout.NewFlexLayout(
			layout.Vertical,
			0,
			layout.Fixed(3), // Header
			layout.Fill(),   // Content
			layout.Fixed(1), // Footer
		)
		chunks := rootLay.Split(area)

		// Header
		f.RenderWidget(widgets.Block{
			Title:          " LIMONI - CUSTOM WIDGET DEVELOPMENT GUIDE ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 180, 0)},
			Child:          text{value: " Build custom widget components, implement rendering, focus, and input handlers ", style: cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}},
		}, chunks[0])

		// Content area split: Left (DialKnob) + Right (ColorPaletteGrid)
		contentLay := layout.NewFlexLayout(
			layout.Horizontal,
			1,
			layout.Percentage(50),
			layout.Percentage(50),
		)
		contentChunks := contentLay.Split(chunks[1])

		// Left: DialKnob Demo Block
		dialBorderCol := cell.NewColorRGB(60, 65, 80)
		if t.FocusManager().Focused() == "knob_dial" {
			dialBorderCol = cell.NewColorRGB(255, 180, 0)
		}

		dialBlock := widgets.Block{
			Title:          " DIAL KNOB (Rotary Control) ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: dialBorderCol},
			PaddingTop:     2,
		}

		f.RenderWidget(dialBlock, contentChunks[0])
		dialInnerArea := cell.NewRect(
			contentChunks[0].X+2,
			contentChunks[0].Y+2,
			contentChunks[0].Width-4,
			contentChunks[0].Height-4,
		)
		f.RenderWidget(DialKnob{
			ID:           "knob_dial",
			Value:        &state.DialValue,
			Min:          0,
			Max:          100,
			Style:        cell.Style{Fg: cell.NewColorRGB(180, 180, 190)},
			FocusedStyle: cell.Style{Fg: cell.NewColorRGB(255, 180, 0)},
		}, dialInnerArea)

		// Right: ColorPaletteGrid Demo Block
		paletteBorderCol := cell.NewColorRGB(60, 65, 80)
		if t.FocusManager().Focused() == "color_grid" {
			paletteBorderCol = cell.NewColorRGB(255, 180, 0)
		}

		paletteBlock := widgets.Block{
			Title:          " COLOR PALETTE (RGB Grid Selector) ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: paletteBorderCol},
			PaddingTop:     2,
		}

		f.RenderWidget(paletteBlock, contentChunks[1])
		paletteInnerArea := cell.NewRect(
			contentChunks[1].X+2,
			contentChunks[1].Y+2,
			contentChunks[1].Width-4,
			contentChunks[1].Height-4,
		)
		f.RenderWidget(ColorPaletteGrid{
			ID:            "color_grid",
			SelectedColor: &state.SelectedColor,
			HoverIndexX:   state.PaletteHoverX,
			HoverIndexY:   state.PaletteHoverY,
			Style:         cell.Style{Fg: cell.NewColorRGB(180, 180, 190)},
			FocusedStyle:  cell.Style{Fg: cell.NewColorRGB(255, 180, 0)},
		}, paletteInnerArea)

		summaryArea := cell.NewRect(
			chunks[1].X+10,
			chunks[1].Y+chunks[1].Height-3,
			chunks[1].Width-20,
			2,
		)
		r, g, bVal := state.SelectedColor.RGB()
		sumStr := fmt.Sprintf("Selected Values -> Dial Knob: %%%d | RGB Color: (%d, %d, %d)", state.DialValue, r, g, bVal)
		f.RenderWidget(widgets.Block{
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsSingle,
			BorderStyle:   cell.Style{Fg: state.SelectedColor},
			PaddingLeft:   1,
			Child:         text{value: sumStr, style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220)}},
		}, summaryArea)

		// Footer
		footerText := " [Tab] Focus | [Dial Knob] Arrow Keys / Mouse Wheel | [Palette] Arrows + Enter | [q] Quit"
		f.RenderWidget(widgets.Block{
			Borders: widgets.BorderNone,
			Style:   cell.Style{Fg: cell.NewColorRGB(130, 130, 130), Bg: cell.NewColorRGB(20, 20, 25)},
			Child:   text{value: footerText, style: cell.Style{Fg: cell.NewColorRGB(130, 130, 130)}},
		}, chunks[2])
	})
}
