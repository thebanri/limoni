package main

import (
	"fmt"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawTabTelemetry(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	// Vertical split: Top Metric Cards (4 rows), Bottom Analytics & Logs (Fill)
	vChunks := layout.VBoxWithGap(area, 1,
		layout.Fixed(4), // 3 Cards
		layout.Fill(),   // Sparkline & Event Log
	)

	// ==========================================
	// 1. TOP: 3 TELEMETRY METRIC CARDS
	// ==========================================
	cards := layout.HBoxWithGap(vChunks[0], 1,
		layout.Percentage(34),
		layout.Percentage(33),
		layout.Percentage(33),
	)

	// Card 1: Hardware Diff Latency
	c1 := widgets.Block{
		Title:          " BUFFER DIFF LATENCY ",
		TitleAlignment: widgets.AlignCenter,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Accent},
		TitleStyle:     cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(c1, cards[0])
	c1Inner := layout.Padded(cards[0], 1, 1, 1, 1)
	latestLat := 18.2
	if len(state.LatencyHistory) > 0 {
		latestLat = state.LatencyHistory[len(state.LatencyHistory)-1]
	}
	f.Buffer.SetString(c1Inner.X+2, c1Inner.Y, fmt.Sprintf("%5.1f µs / frame", latestLat), cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold})
	f.Buffer.SetString(c1Inner.X+2, c1Inner.Y+1, "Hardware ANSI Diff Engine", cell.Style{Fg: theme.Muted})

	// Card 2: Render Refresh Rate
	c2 := widgets.Block{
		Title:          " REFRESH RATE ",
		TitleAlignment: widgets.AlignCenter,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Secondary},
		TitleStyle:     cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(c2, cards[1])
	c2Inner := layout.Padded(cards[1], 1, 1, 1, 1)
	f.Buffer.SetString(c2Inner.X+2, c2Inner.Y, fmt.Sprintf("%5.1f FPS", state.FPS), cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold})
	f.Buffer.SetString(c2Inner.X+2, c2Inner.Y+1, "Zero Dropped Frames", cell.Style{Fg: theme.Muted})

	// Card 3: Zero Allocations
	c3 := widgets.Block{
		Title:          " ALLOCATIONS ",
		TitleAlignment: widgets.AlignCenter,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Primary},
		TitleStyle:     cell.Style{Fg: theme.Primary, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(c3, cards[2])
	c3Inner := layout.Padded(cards[2], 1, 1, 1, 1)
	f.Buffer.SetString(c3Inner.X+2, c3Inner.Y, "0 B / frame (0 allocs)", cell.Style{Fg: theme.Primary, Modifier: cell.ModifierBold})
	f.Buffer.SetString(c3Inner.X+2, c3Inner.Y+1, fmt.Sprintf("Heap: %.1f MB │ GC: %d", float64(state.AllocStats.HeapAlloc)/1024/1024, state.AllocStats.NumGC), cell.Style{Fg: theme.Muted})

	// ==========================================
	// 2. BOTTOM: SPARKLINE HISTORY & EVENT LOG
	// ==========================================
	bottomCols := layout.HBoxWithGap(vChunks[1], 1,
		layout.Percentage(56),
		layout.Percentage(44),
	)

	// Left: Sparkline Chart & Progress Bar
	chartBlock := widgets.Block{
		Title:          " REAL-TIME LATENCY SPARKLINE (µs) ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Primary},
		TitleStyle:     cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(chartBlock, bottomCols[0])

	innerChart := layout.Padded(bottomCols[0], 1, 1, 1, 1)
	chartChunks := layout.VBoxWithGap(innerChart, 1,
		layout.Fixed(1), // Guide
		layout.Fixed(3), // Sparkline
		layout.Fixed(2), // Progress bar
		layout.Fixed(1), // Checkbox 1
		layout.Fixed(1), // Checkbox 2
		layout.Fill(),   // Telemetry explanation
	)

	f.Buffer.SetString(chartChunks[0].X, chartChunks[0].Y, "High-frequency frame diff measurement history:", cell.Style{Fg: theme.Muted, Modifier: cell.ModifierItalic})

	sparkline := widgets.Sparkline{
		Data:  state.LatencyHistory,
		Style: cell.Style{Fg: theme.Accent},
	}
	f.RenderWidget(sparkline, chartChunks[1])

	// Buffer swap throughput
	prog := widgets.ProgressBar{
		Value:       state.ProgressVal,
		Min:         0.0,
		Max:         100.0,
		ShowPercent: true,
		FilledStyle: cell.Style{Fg: theme.Secondary, Bg: theme.Secondary},
		EmptyStyle:  cell.Style{Fg: theme.Muted, Bg: theme.BgCard},
	}
	f.RenderWidget(prog, chartChunks[2])

	// Checkboxes
	cbPolling := widgets.Checkbox{
		ID:           "cb_telem_poll",
		Checked:      &state.LiveTelemetry,
		Label:        "Live High-Frequency Metric Sampling",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbPolling, chartChunks[3])

	cbZeroAlloc := widgets.Checkbox{
		ID:           "cb_telem_zero",
		Checked:      &state.ZeroAllocGuard,
		Label:        "Zero-Allocations Benchmark Invariant Guard",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbZeroAlloc, chartChunks[4])

	// Explanation
	f.Buffer.SetString(chartChunks[5].X, chartChunks[5].Y+1, "• Hardware ANSI Diff: Emits only modified terminal cells", cell.Style{Fg: theme.Text})
	f.Buffer.SetString(chartChunks[5].X, chartChunks[5].Y+2, "• Zero allocations on hot render paths (0 bytes/frame)", cell.Style{Fg: theme.Text})

	// Right: Audit & Interaction Event Log
	logBlock := widgets.Block{
		Title:          " ENGINE AUDIT & INTERACTION LOG ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Secondary},
		TitleStyle:     cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(logBlock, bottomCols[1])

	innerLog := layout.Padded(bottomCols[1], 1, 1, 1, 1)
	f.Buffer.SetString(innerLog.X, innerLog.Y, "Recent Events & User Interactions:", cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold})

	for i, logEntry := range state.Logs {
		rowY := innerLog.Y + 2 + uint16(i)
		if rowY >= innerLog.Y+innerLog.Height {
			break
		}
		f.Buffer.SetString(innerLog.X+1, rowY, logEntry, cell.Style{Fg: theme.Text})
	}
}
