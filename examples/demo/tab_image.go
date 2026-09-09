package main

import (
	"fmt"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawTabImage(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	// Split: Left Viewport (56%), Right Controls (44%)
	cols := layout.HBoxWithGap(area, 1,
		layout.Percentage(56),
		layout.Percentage(44),
	)

	// ==========================================
	// 1. LEFT: IMAGE VIEWPORT
	// ==========================================
	activeName := "No Image Loaded"
	if len(state.ImageNames) > 0 && state.ActiveImageIdx < len(state.ImageNames) {
		activeName = state.ImageNames[state.ActiveImageIdx]
	}

	imgBlock := widgets.Block{
		Title:          fmt.Sprintf(" IMAGE VIEWPORT • %s ", activeName),
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Primary},
		TitleStyle:     cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(imgBlock, cols[0])

	innerLeft := layout.Padded(cols[0], 1, 1, 1, 1)

	if len(state.Images) > 0 && state.ActiveImageIdx < len(state.Images) && state.Images[state.ActiveImageIdx] != nil {
		proto := graphics.DetectProtocol()
		protoName := "Half-Block 2x TrueColor"
		if !state.ImageHalfBlock {
			switch proto {
			case graphics.ProtocolKitty:
				protoName = "Kitty Graphics (GPU Direct)"
			case graphics.ProtocolSixel:
				protoName = "Sixel Graphics (Hardware)"
			case graphics.ProtocolIterm2:
				protoName = "iTerm2 Inline Protocol"
			}
		}

		imgWidget := &widgets.Image{
			Img:            state.Images[state.ActiveImageIdx],
			CircleMask:     state.ImageCircleMask,
			ForceHalfBlock: state.ImageHalfBlock,
			Transparent:    true,
		}
		f.RenderWidget(imgWidget, innerLeft)

		// Bottom Viewport Info Pill
		hudY := innerLeft.Y + innerLeft.Height - 1
		bounds := state.Images[state.ActiveImageIdx].Bounds()
		maskStr := "Square"
		if state.ImageCircleMask {
			maskStr = "Circle Mask (Active)"
		}
		hudText := fmt.Sprintf(" %dx%d px │ %s │ Mask: %s ", bounds.Dx(), bounds.Dy(), protoName, maskStr)
		f.Buffer.SetString(innerLeft.X+1, hudY, hudText, cell.Style{Fg: theme.Text, Bg: theme.BgCard, Modifier: cell.ModifierBold})
	} else {
		f.Buffer.SetString(innerLeft.X+2, innerLeft.Y+2, "No image loaded or image format not recognized.", cell.Style{Fg: theme.Accent})
	}

	// ==========================================
	// 2. RIGHT: IMAGE CONTROLS & PROPERTIES
	// ==========================================
	ctrlBlock := widgets.Block{
		Title:          " IMAGE CONTROLS & PROPERTIES ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Secondary},
		TitleStyle:     cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(ctrlBlock, cols[1])

	innerRight := layout.Padded(cols[1], 1, 1, 1, 1)
	drawImageControls(f, innerRight, state, theme)
}

func drawImageControls(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	if area.Width < 20 || area.Height < 10 {
		return
	}

	vChunks := layout.VBoxWithGap(area, 0,
		layout.Fixed(1), // Guide
		layout.Fixed(1), // Image Switcher label
		layout.Fixed(2), // Button: Apple
		layout.Fixed(2), // Button: Profile
		layout.Fixed(1), // Spacer
		layout.Fixed(1), // Checkbox: Circle Mask
		layout.Fixed(1), // Checkbox: Half-Block TrueColor
		layout.Fixed(1), // Checkbox: Aspect Ratio
		layout.Fill(),   // Metadata info card
	)

	f.Buffer.SetString(vChunks[0].X, vChunks[0].Y, "Click buttons or checkboxes to switch:", cell.Style{Fg: theme.Muted, Modifier: cell.ModifierItalic})
	f.Buffer.SetString(vChunks[1].X, vChunks[1].Y, "Select Sample Image Asset:", cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold})

	// Buttons for image selection
	renderClickButton(f, vChunks[2], "1. Apple Graphic (256x256)", state.ActiveImageIdx == 0, theme.Primary, theme.Muted, func() {
		state.ActiveImageIdx = 0
		addLog(state, "Switched to Apple asset")
	})

	renderClickButton(f, vChunks[3], "2. Profile Avatar (512x512)", state.ActiveImageIdx == 1, theme.Primary, theme.Muted, func() {
		state.ActiveImageIdx = 1
		addLog(state, "Switched to Profile Avatar asset")
	})

	// Checkboxes
	cbMask := widgets.Checkbox{
		ID:           "cb_img_mask",
		Checked:      &state.ImageCircleMask,
		Label:        "Circular Avatar Mask [c]",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbMask, vChunks[5])

	cbHalf := widgets.Checkbox{
		ID:           "cb_img_half",
		Checked:      &state.ImageHalfBlock,
		Label:        "Force Half-Block Mode (Override Kitty/Sixel)",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbHalf, vChunks[6])

	cbAspect := widgets.Checkbox{
		ID:           "cb_img_aspect",
		Checked:      &state.ImageAspectCorrect,
		Label:        "Terminal 1:2 Aspect Ratio Compensation",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbAspect, vChunks[7])

	// Image Specs Card
	metaArea := vChunks[8]
	if metaArea.Height >= 4 {
		metaBlock := widgets.Block{
			Title:          " IMAGE PIPELINE SPECS ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: theme.Muted},
			TitleStyle:     cell.Style{Fg: theme.Text, Modifier: cell.ModifierBold},
		}
		f.RenderWidget(metaBlock, metaArea)

		innerMeta := layout.Padded(metaArea, 1, 1, 1, 1)
		proto := graphics.DetectProtocol()
		activeMode := "Half-Block Cell Matrix (Fallback)"
		if !state.ImageHalfBlock {
			switch proto {
			case graphics.ProtocolKitty:
				activeMode = "Kitty Protocol (1-to-1 GPU Pixel Direct)"
			case graphics.ProtocolSixel:
				activeMode = "Sixel Protocol (1-to-1 Hardware Pixel Direct)"
			case graphics.ProtocolIterm2:
				activeMode = "iTerm2 Protocol (1-to-1 High-DPI Inline)"
			}
		}

		lines := []string{
			fmt.Sprintf("• Active Backend: %s", activeMode),
			"• Hardware Support: Automatic Kitty, Sixel, and iTerm2 detection",
			"• Fallback: Dual-pixel Half-Block ('▀'/'▄') TrueColor ANSI",
			"• Transparency: Full RGBA alpha preservation (Zero black borders)",
			"• Masking: Real-time geometric circle/avatar clipping",
		}
		for i, line := range lines {
			if uint16(i) >= innerMeta.Height {
				break
			}
			f.Buffer.SetString(innerMeta.X+1, innerMeta.Y+uint16(i), line, cell.Style{Fg: theme.Text})
		}
	}
}
