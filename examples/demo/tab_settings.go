package main

import (
	"fmt"
	"runtime"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawTabSettings(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	settingsBlock := widgets.Block{
		Title:          " ENGINE CONFIGURATION & THEMES ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Primary},
		TitleStyle:     cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(settingsBlock, area)

	innerArea := layout.Padded(area, 1, 1, 1, 1)

	// Split: Left Form & Checkboxes (55%), Right Theme Switcher & Info (45%)
	cols := layout.HBoxWithGap(innerArea, 1,
		layout.Percentage(55),
		layout.Percentage(45),
	)

	// ==========================================
	// 1. LEFT: ENGINE INTERACTIVE SETTINGS
	// ==========================================
	leftChunks := layout.VBoxWithGap(cols[0], 1,
		layout.Fixed(1), // Guide
		layout.Fixed(1), // Checkbox: SGR Mouse
		layout.Fixed(1), // Checkbox: Diff Engine
		layout.Fixed(1), // Checkbox: Smooth 60 FPS
		layout.Fixed(1), // Checkbox: Accessibility
		layout.Fill(),   // Architecture Specs
	)

	f.Buffer.SetString(leftChunks[0].X, leftChunks[0].Y, "Interactive Engine Switches (Click to toggle):", cell.Style{Fg: theme.Muted, Modifier: cell.ModifierItalic})

	cbMouse := widgets.Checkbox{
		ID:           "cb_set_mouse",
		Checked:      &state.SGRMouseTracking,
		Label:        "SGR 1006 Extended Mouse Tracking (Full Point & Click)",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbMouse, leftChunks[1])

	cbDiff := widgets.Checkbox{
		ID:           "cb_set_diff",
		Checked:      &state.DiffEngineActive,
		Label:        "Hardware Double-Buffer Diff Engine (Dirty Cell Emits)",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbDiff, leftChunks[2])

	cbFPS := widgets.Checkbox{
		ID:           "cb_set_fps",
		Checked:      &state.Smooth60FPS,
		Label:        "Target 60 FPS Render Loop with Zero Jitter",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbFPS, leftChunks[3])

	cbA11y := widgets.Checkbox{
		ID:           "cb_set_a11y",
		Checked:      &state.AccessibilityActive,
		Label:        "Semantic Accessibility Tree (WAI-ARIA Screen Reader Export)",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbA11y, leftChunks[4])

	// Architecture summary box
	archArea := leftChunks[5]
	if archArea.Height >= 4 {
		archBlock := widgets.Block{
			Title:          " ZERO-COMPROMISE TUI ENGINE ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: theme.Muted},
			TitleStyle:     cell.Style{Fg: theme.Text, Modifier: cell.ModifierBold},
		}
		f.RenderWidget(archBlock, archArea)

		innerArch := layout.Padded(archArea, 1, 1, 1, 1)
		lines := []string{
			"• Hardware-accelerated memory diffing prevents screen flicker",
			"• In-place slice reuse guarantees zero allocations on redraws",
			"• Layered composite modal & popup system with mouse capture",
			"• First-class 3D graphics pipeline with GLB & Wavefront OBJ",
		}
		for i, line := range lines {
			if uint16(i) >= innerArch.Height {
				break
			}
			f.Buffer.SetString(innerArch.X+1, innerArch.Y+uint16(i), line, cell.Style{Fg: theme.Text})
		}
	}

	// ==========================================
	// 2. RIGHT: COLOR THEMES & SYSTEM INFO
	// ==========================================
	rightChunks := layout.VBoxWithGap(cols[1], 1,
		layout.Fixed(1), // Theme header
		layout.Fixed(2), // Theme button 1
		layout.Fixed(2), // Theme button 2
		layout.Fixed(2), // Theme button 3
		layout.Fixed(2), // Theme button 4
		layout.Fill(),   // System Info Card
	)

	f.Buffer.SetString(rightChunks[0].X, rightChunks[0].Y, "Select Active Color Palette [t]:", cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold})

	renderClickButton(f, rightChunks[1], "1. Limoni Citrus (Default)", state.ThemeIndex == 0, theme.Primary, theme.Muted, func() {
		state.ThemeIndex = 0
		addLog(state, "Theme: Limoni Citrus")
	})

	renderClickButton(f, rightChunks[2], "2. Cyberpunk Neon", state.ThemeIndex == 1, theme.Primary, theme.Muted, func() {
		state.ThemeIndex = 1
		addLog(state, "Theme: Cyberpunk Neon")
	})

	renderClickButton(f, rightChunks[3], "3. Nord Frost", state.ThemeIndex == 2, theme.Primary, theme.Muted, func() {
		state.ThemeIndex = 2
		addLog(state, "Theme: Nord Frost")
	})

	renderClickButton(f, rightChunks[4], "4. Emerald Forest", state.ThemeIndex == 3, theme.Primary, theme.Muted, func() {
		state.ThemeIndex = 3
		addLog(state, "Theme: Emerald Forest")
	})

	// System info card
	sysArea := rightChunks[5]
	if sysArea.Height >= 4 {
		sysBlock := widgets.Block{
			Title:          " SYSTEM RUNTIME TELEMETRY ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: theme.Muted},
			TitleStyle:     cell.Style{Fg: theme.Text, Modifier: cell.ModifierBold},
		}
		f.RenderWidget(sysBlock, sysArea)

		innerSys := layout.Padded(sysArea, 1, 1, 1, 1)
		lines := []string{
			fmt.Sprintf("• Go Runtime:     %s (%s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
			fmt.Sprintf("• Logical CPUs:   %d Cores", runtime.NumCPU()),
			fmt.Sprintf("• Engine Build:   Limoni v0.1.7 (Release)"),
			fmt.Sprintf("• Live Goroutines:%d Active", runtime.NumGoroutine()),
		}
		for i, line := range lines {
			if uint16(i) >= innerSys.Height {
				break
			}
			f.Buffer.SetString(innerSys.X+1, innerSys.Y+uint16(i), line, cell.Style{Fg: theme.Text})
		}
	}
}
