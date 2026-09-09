package main

import (
	"fmt"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawTab3D(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	// Split into: Left Viewport (58%), Right Controls & Mesh Info (42%)
	cols := layout.HBoxWithGap(area, 1,
		layout.Percentage(58),
		layout.Percentage(42),
	)

	// ==========================================
	// 1. LEFT: 3D GLB VIEWPORT
	// ==========================================
	modeNames := []string{"ASCII Typography", "Half-Block 2x TrueColor", "Braille 8x Dot-Matrix"}
	if state.RenderModeIndex >= len(modeNames) {
		state.RenderModeIndex = 0
	}
	currMode := modeNames[state.RenderModeIndex]

	block3D := widgets.Block{
		Title:          fmt.Sprintf(" 3D GO GOPHER & LEMON MASCOT • %s ", currMode),
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Primary},
		TitleStyle:     cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(block3D, cols[0])

	inner3D := layout.Padded(cols[0], 1, 1, 1, 1)
	if inner3D.Width >= 4 && inner3D.Height >= 4 {
		highlightCol := cell.NewColorRGB(255, 255, 220)
		if !state.Specular {
			highlightCol = cell.NewColorRGB(200, 180, 20)
		}

		bodyCol := cell.NewColorRGB(255, 220, 30) // Vibrant natural lemon yellow
		if !state.NaturalColors {
			bodyCol = cell.NewColorRGB(200, 200, 200) // Monochrome wireframe
		}

		// In ASCII mode, scale up significantly to maximize letter count and visual density (~800+ glyphs)
		effectiveScale := state.Scale
		if state.AsciiMode == widgets.ModeASCII {
			effectiveScale = state.Scale * 2.45
		}

		asciiWidget := widgets.Ascii3D{
			Model:           state.Model,
			Mode:            state.AsciiMode,
			Scale:           effectiveScale,
			AutoRotate:      state.AutoRotate,
			AutoRotateSpeed: state.RotateSpeed,
			Time:            state.ElapsedSecs,
			RotX:            state.RotX + 15.0,
			RotY:            state.RotY,
			FloatIntensity:  0.16,
			FloatSpeed:      1.5,
			Roughness:       state.Roughness,
			Exposure:        state.Exposure,
			Contrast:        1.25,
			Colored:         state.NaturalColors,
			Ascii:           state.AsciiMode == widgets.ModeASCII,
			Color:           bodyCol,
			Highlight:       highlightCol,
			LightDirection:  graphics.Vector3D{X: -0.5, Y: 0.8, Z: 0.6},
		}
		f.RenderWidget(asciiWidget, inner3D)

		// Bottom Viewport Status Pill
		hudY := inner3D.Y + inner3D.Height - 1
		hudText := fmt.Sprintf(" Verts: %d │ Faces: %d │ Rot: %3.0f° │ Scale: %.1fx ", len(state.Model.Vertices), len(state.Model.Faces), state.RotY, effectiveScale)
		f.Buffer.SetString(inner3D.X+1, hudY, hudText, cell.Style{Fg: theme.Text, Bg: theme.BgCard, Modifier: cell.ModifierBold})
	}

	// ==========================================
	// 2. RIGHT: 3D CONTROLS & PROPERTIES
	// ==========================================
	blockCtrl := widgets.Block{
		Title:          " 3D CONTROLS & PROPERTIES ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Secondary},
		TitleStyle:     cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(blockCtrl, cols[1])

	innerCtrl := layout.Padded(cols[1], 1, 1, 1, 1)
	draw3DControls(f, innerCtrl, state, theme)
}

func draw3DControls(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	if area.Width < 20 || area.Height < 10 {
		return
	}

	// Vertical layout inside right panel:
	// - Header guide (1)
	// - Checkbox 1 (1)
	// - Checkbox 2 (1)
	// - Checkbox 3 (1)
	// - Mode Title (1)
	// - 3 Mode Buttons (2)
	// - Stats Block (Fill)
	vChunks := layout.VBoxWithGap(area, 0,
		layout.Fixed(1), // Guide
		layout.Fixed(1), // AutoRotate Checkbox
		layout.Fixed(1), // Specular Checkbox
		layout.Fixed(1), // NaturalColors Checkbox
		layout.Fixed(1), // Spacer / Mode label
		layout.Fixed(2), // 3 Mode buttons row
		layout.Fill(),   // Model Info summary
	)

	// Guide text
	f.Buffer.SetString(vChunks[0].X, vChunks[0].Y, "Click checkboxes or buttons to interact:", cell.Style{Fg: theme.Muted, Modifier: cell.ModifierItalic})

	// 1. Checkboxes
	cbRotate := widgets.Checkbox{
		ID:           "cb_3d_rotate",
		Checked:      &state.AutoRotate,
		Label:        "Auto-Rotate Mesh [r]",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbRotate, vChunks[1])

	cbSpecular := widgets.Checkbox{
		ID:           "cb_3d_specular",
		Checked:      &state.Specular,
		Label:        "Blinn-Phong Specular Highlights",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbSpecular, vChunks[2])

	cbColors := widgets.Checkbox{
		ID:           "cb_3d_colors",
		Checked:      &state.NaturalColors,
		Label:        "Mascot TrueColor (Blue Gopher & Yellow Lemon)",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbColors, vChunks[3])

	// 2. Mode selection header
	f.Buffer.SetString(vChunks[4].X, vChunks[4].Y, "Render Mode Selection [m]:", cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold})

	// 3 Mode buttons in one clean row
	modeCols := layout.HBoxWithGap(vChunks[5], 1, layout.Percentage(34), layout.Percentage(33), layout.Percentage(33))
	renderClickButton(f, modeCols[0], "1. ASCII", state.RenderModeIndex == 0, theme.Primary, theme.Muted, func() {
		state.RenderModeIndex = 0
		state.AsciiMode = widgets.ModeASCII
		addLog(state, "Mode: ASCII Typography")
	})
	renderClickButton(f, modeCols[1], "2. Block", state.RenderModeIndex == 1, theme.Primary, theme.Muted, func() {
		state.RenderModeIndex = 1
		state.AsciiMode = widgets.ModeBlock
		addLog(state, "Mode: Half-Block 2x TrueColor")
	})
	renderClickButton(f, modeCols[2], "3. Braille", state.RenderModeIndex == 2, theme.Primary, theme.Muted, func() {
		state.RenderModeIndex = 2
		state.AsciiMode = widgets.ModeBraille
		addLog(state, "Mode: Braille 8x Dot-Matrix")
	})

	// 3. Model Info Card
	infoArea := vChunks[6]
	if infoArea.Height >= 4 {
		infoBlock := widgets.Block{
			Title:          " MESH ASSET SPECS ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: theme.Muted},
			TitleStyle:     cell.Style{Fg: theme.Text, Modifier: cell.ModifierBold},
		}
		f.RenderWidget(infoBlock, infoArea)

		innerInfo := layout.Padded(infoArea, 1, 1, 1, 1)
		lines := []string{
			"• Model Asset: examples/demo/limoni.glb",
			"• Mascot:      Go Gopher (Blue) & Lemon (Yellow)",
			fmt.Sprintf("• Total Faces: %d Triangles", len(state.Model.Faces)),
			"• Gopher Body: 14,318 Faces (Go Cyan Blue #38B6FF)",
			"• Lemon Mesh:  7,880 Faces (Citrus Yellow #FFE024)",
			"• Leaf & Stem: Leaf Green #4ADE80 & White Snout",
			"• Controls:    [m] Switch Mode, [r] Rotate, [+/-] Zoom",
		}
		for i, line := range lines {
			if uint16(i) >= innerInfo.Height {
				break
			}
			f.Buffer.SetString(innerInfo.X+1, innerInfo.Y+uint16(i), line, cell.Style{Fg: theme.Text})
		}
	}
}
