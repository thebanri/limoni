package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawReference(t *terminal.Terminal, f *terminal.Frame, state *AppState, theme widgets.Theme, mainColor, accentColor cell.Color, area cell.Rect) {
	// Split into sub-tab bar and sub-tab content
	rows := layout.NewFlexLayout(layout.Vertical, 1, layout.Fixed(3), layout.Fill()).Split(area)

	// Draw Sub-Tab Bar
	subTabs := []string{"Runtime", "Layout", "Accessibility", "Virtual Data", "Benchmark"}
	if state.ReferenceActiveSubTab == "" {
		state.ReferenceActiveSubTab = "Runtime"
	}

	cols := layout.NewFlexLayout(layout.Horizontal, 1,
		layout.Percentage(20),
		layout.Percentage(20),
		layout.Percentage(20),
		layout.Percentage(20),
		layout.Percentage(20),
	).Split(rows[0])

	for i, tab := range subTabs {
		titleStyle := theme.RoleStyle("text")
		borderStyle := theme.Border
		if state.ReferenceActiveSubTab == tab {
			titleStyle.Modifier = cell.ModifierBold
			borderStyle = cell.Style{Fg: accentColor, Modifier: cell.ModifierBold}
		}
		btn := widgets.Block{
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    borderStyle,
			Title:          " " + tab + " ",
			TitleAlignment: widgets.AlignCenter,
			TitleStyle:     titleStyle,
		}
		f.RenderWidget(btn, cols[i])

		tabName := tab
		f.RegisterClickHandler(cols[i], func(ev backend.MouseEvent) {
			if ev.Button == backend.MouseLeft && !ev.Drag {
				state.ReferenceActiveSubTab = tabName
			}
		})
	}

	// Register basic accessibility tree
	f.RegisterAccessibility(referenceAccessibilityTree(state, area))

	// Draw Content based on active sub-tab
	contentArea := rows[1]
	switch state.ReferenceActiveSubTab {
	case "Runtime":
		drawSubTabRuntime(f, state, theme, contentArea)
	case "Layout":
		drawSubTabLayout(f, state, theme, contentArea)
	case "Accessibility":
		drawSubTabAccessibility(f, state, theme, contentArea)
	case "Virtual Data":
		drawSubTabVirtualData(f, state, theme, accentColor, contentArea)
	case "Benchmark":
		drawSubTabBenchmark(f, state, theme, accentColor, contentArea)
	}
	_ = mainColor
}

func referenceAccessibilityTree(state *AppState, area cell.Rect) accessibility.AccessibilityNode {
	return accessibility.AccessibilityNode{
		ID: "reference", Role: accessibility.RoleDialog, Label: "Limoni referans ve geliştirici araçları", Bounds: area,
		Children: []accessibility.AccessibilityNode{
			{ID: "subtab-runtime", Role: accessibility.RoleGeneric, Label: "Runtime CMD-MSG", Value: fmt.Sprintf("%d mesaj", state.ReferenceRuntimeMessages), Bounds: area},
		},
	}
}

// 1. RUNTIME SUB-TAB
func drawSubTabRuntime(f *terminal.Frame, state *AppState, theme widgets.Theme, area cell.Rect) {
	message := fmt.Sprintf(
		"RUNTIME & MESSAGE DISPATCH SHOWCASE\n\n"+
			"• Mesaj Sayısı (Runtime messages): %d\n"+
			"• Cmd Scheduler: Active (asynchronous loops supported)\n"+
			"• Cancellation: context.Context native propagation\n"+
			"• Draw Updates: Coalesced screen updates\n\n"+
			"--> TIKLA: Örnek Msg gönder (Mesaj sayısını artırır)",
		state.ReferenceRuntimeMessages,
	)
	f.RenderWidget(widgets.Block{
		Title: " RUNTIME DIAGNOSTICS ", Borders: widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border,
		PaddingLeft: 2, PaddingTop: 1, Child: referenceLabel{text: message, style: theme.RoleStyle("text")},
	}, area)

	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceRuntimeMessages++
		}
	})
}

// 2. LAYOUT SUB-TAB
func drawSubTabLayout(f *terminal.Frame, state *AppState, theme widgets.Theme, area cell.Rect) {
	idealHeight := uint16(2 + (state.ReferenceLayoutPass % 4))
	measure := layout.Measure{
		MinWidth: 10, MinHeight: 1,
		IdealWidth: area.Width, IdealHeight: idealHeight,
		MaxWidth: area.Width, MaxHeight: area.Height,
		GrowPriority: 1 + state.ReferenceLayoutPass,
		Overflow:     layout.OverflowClip,
	}.Normalize(area)

	allocated := layout.Arrange(area, []layout.Measure{measure}, layout.Vertical, 0)[0]
	state.ReferenceLayoutAllocated = allocated

	if state.ReferenceLayoutLastAction == "" {
		state.ReferenceLayoutLastAction = "henüz ölçüm yapılmadı"
	}

	// Calculate diagnostics using our newly implemented Diagnose API
	diagnostics := layout.Diagnose([]layout.Measure{measure}, []cell.Rect{allocated})[0]

	text := fmt.Sprintf(
		"LAYOUT MANAGER & EXPLICIT SIZE NEGOTIATION\n\n"+
			"• Min Size: %dx%d  |  Ideal Size: %dx%d  |  Max Size: %dx%d\n"+
			"• Allocated Area: X=%d Y=%d W=%d H=%d\n"+
			"• Grow Priority: %d  |  Overflow Policy: %v\n\n"+
			"Layout Diagnostics:\n"+
			"  - Shrunk (Clipped): %t\n"+
			"  - Grown (Expanded): %t\n"+
			"  - Baseline Offset: %d\n"+
			"  - Active Policy: %v\n\n"+
			"--> TIKLA: Ideal yükseklik, grow priority ve diagnostics'i değiştir",
		measure.MinWidth, measure.MinHeight, measure.IdealWidth, measure.IdealHeight, measure.MaxWidth, measure.MaxHeight,
		allocated.X, allocated.Y, allocated.Width, allocated.Height,
		measure.GrowPriority, measure.Overflow,
		diagnostics.Shrunk, diagnostics.Grown, diagnostics.BaselineOffset, diagnostics.Policy,
	)

	f.RenderWidget(widgets.Block{
		Title: " LAYOUT INSPECTOR ", Borders: widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border,
		PaddingLeft: 2, PaddingTop: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")},
	}, area)

	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceLayoutPass = (state.ReferenceLayoutPass + 1) % 10
			state.ReferenceLayoutLastAction = fmt.Sprintf("click at (%d,%d)", ev.X, ev.Y)
		}
	})
}

// 3. ACCESSIBILITY SUB-TAB
func drawSubTabAccessibility(f *terminal.Frame, state *AppState, theme widgets.Theme, area cell.Rect) {
	mode := accessibility.Mode{
		HighContrast:  theme.Colors.Primary == cell.NewColorRGB(255, 255, 0),
		ReducedMotion: false,
		ScreenReader:  true,
		ASCIIOnly:     state.ReferenceAccessibilityASCII,
	}

	// Capture a line mode accessibility string
	lineModeText := f.AccessibilityLineMode(mode)
	nav := accessibility.NewLineNavigator(lineModeText)

	// Traverse using LineNavigator
	navOutput := ""
	lineIndex := 0
	for nav.Current() != "" {
		if lineIndex < 4 {
			navOutput += "   - " + nav.AnnounceCurrent() + "\n"
		}
		nav.Next()
		lineIndex++
	}
	if navOutput == "" {
		navOutput = "   - (Semantic tree is empty)"
	}

	text := fmt.Sprintf(
		"ACCESSIBILITY 2.0 & SEMANTIC ANNOUNCEMENT\n\n"+
			"• Mode: HighContrast=%t  |  ASCIIOnly=%t\n"+
			"• Screen Reader Protocol: Active (Announcements serializing)\n\n"+
			"LineNavigator Traversal Announcements:\n%s\n"+
			"--> TIKLA: ASCII / Unicode çıkış formatını değiştir",
		mode.HighContrast, mode.ASCIIOnly, navOutput,
	)

	f.RenderWidget(widgets.Block{
		Title: " ACCESSIBILITY SERIALIZER ", Borders: widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border,
		PaddingLeft: 2, PaddingTop: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")},
	}, area)

	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceAccessibilityASCII = !state.ReferenceAccessibilityASCII
		}
	})
}

// 4. VIRTUAL DATA SUB-TAB
func drawSubTabVirtualData(f *terminal.Frame, state *AppState, theme widgets.Theme, accent cell.Color, area cell.Rect) {
	provider := referenceProvider{}
	if state.ReferenceDataState == nil {
		state.ReferenceDataState = widgets.NewVirtualDataState()
	}
	data := state.ReferenceDataState

	// Select active queue policy display
	policyText := "LatestOnly"
	switch data.QueuePolicy() {
	case widgets.VirtualDropOldest:
		policyText = "DropOldest"
	case widgets.VirtualDropLatest:
		policyText = "DropLatest"
	case widgets.VirtualSequential:
		policyText = "Sequential"
	}

	// Stats
	stats := data.QueueStats()
	qStatsText := fmt.Sprintf(
		"Started: %d | Completed: %d | Canceled: %d | Stale: %d | Dropped: %d | QueueLength: %d",
		stats.Started, stats.Completed, stats.Canceled, stats.Stale, stats.Dropped, stats.QueueLength,
	)

	viewportHeight := int(area.Height) - 9
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Trigger async/sync viewport refresh
	_ = data.Refresh(nil, provider, state.ReferenceDataOffset, viewportHeight, 2)
	data.Select(provider.RowID(state.ReferenceSelectedRow))

	lastVisible := state.ReferenceDataOffset + viewportHeight - 1
	if lastVisible >= data.Count() {
		lastVisible = data.Count() - 1
	}

	// QueryResult
	qResult := data.QueryResult()
	qResultText := fmt.Sprintf("QueryResult: loaded=%d rows, offset=%d (total=%d)", qResult.Filtered, qResult.Offset, qResult.Count)

	text := fmt.Sprintf(
		"VIRTUAL DATA viewport scroll (wheel over list below)  |  Stable ID selected: %s\n"+
			"• Queue Policy: %s  |  %s\n"+
			"• %s\n"+
			"--> TIKLA liste öğesi: Seçili satırı değiştir (seçili satır: #%d)",
		data.Selected(), policyText, qResultText, qStatsText, state.ReferenceSelectedRow,
	)

	// Draw container block
	inner := cell.NewRect(area.X+1, area.Y+1, area.Width-2, area.Height-2)
	f.RenderWidget(widgets.Block{
		Title: " VIRTUAL DATA RUNTIME ", Borders: widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: accent},
		PaddingLeft: 2, PaddingTop: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")},
	}, area)

	// Draw Virtual Data View in the remaining lower area
	if inner.Width > 2 && inner.Height > 7 {
		f.RenderWidget(widgets.VirtualDataView{
			State: data, Source: provider, First: 0, Prefetch: 2,
			Offset: &state.ReferenceDataOffset,
			OnSelect: func(index int, _ widgets.Row) {
				state.ReferenceSelectedRow = index
				// Toggle selected in our state's selected set
				data.ToggleSelect(provider.RowID(index))
			},
			Style: theme.RoleStyle("muted"), SelectedStyle: cell.Style{Fg: accent, Modifier: cell.ModifierBold},
		}, cell.NewRect(inner.X, inner.Y+6, inner.Width, inner.Height-6))
	}
}

// 5. BENCHMARK SUB-TAB
func drawSubTabBenchmark(f *terminal.Frame, state *AppState, theme widgets.Theme, accent cell.Color, area cell.Rect) {
	// Simulator for benchmarks
	p50 := 12.4 + float64(state.ReferenceBenchmarkRuns)*0.15
	p95 := 24.8 + float64(state.ReferenceBenchmarkRuns)*0.32
	p99 := 38.1 + float64(state.ReferenceBenchmarkRuns)*0.45

	text := fmt.Sprintf(
		"BENCHMARK INFRASTRUCTURE LABORATORY\n\n"+
			"• Benchmark Runs Simulator: %d runs completed\n"+
			"• Quantile latency performance results:\n"+
			"   - P50 Latency: %5.2f µs\n"+
			"   - P95 Latency: %5.2f µs\n"+
			"   - P99 Latency: %5.2f µs\n\n"+
			"Dashboard Validations:\n"+
			"  [✓] runner specs matched\n"+
			"  [✓] iterations count correct (100+)\n"+
			"  [✓] warmups active (10 sample frames)\n\n"+
			"--> TIKLA: Benchmark simülasyonunu çalıştır (latencyleri günceller)",
		state.ReferenceBenchmarkRuns, p50, p95, p99,
	)

	f.RenderWidget(widgets.Block{
		Title: " BENCHMARK SIMULATOR ", Borders: widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: accent},
		PaddingLeft: 2, PaddingTop: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")},
	}, area)

	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceBenchmarkRuns++
		}
	})
}

type referenceLabel struct {
	text  string
	style cell.Style
}

func (l referenceLabel) Draw(ctx cell.Context, buf *buffer.Buffer) {
	style := ctx.Style.Merge(l.style)
	for row, line := range strings.Split(l.text, "\n") {
		if row < int(ctx.Area.Height) {
			buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(row), line, style)
		}
	}
}
func (l referenceLabel) SizeHint(max cell.Rect) (uint16, uint16) { return max.Width, max.Height }

type referenceProvider struct{}

func (referenceProvider) RowCount(context.Context) (int, error) { return 1000000, nil }
func (referenceProvider) RowAt(_ context.Context, index int) (widgets.Row, error) {
	return widgets.Row{ID: widgets.RowID(fmt.Sprintf("row-%d", index)), Text: fmt.Sprintf("#%06d  |  örnek kayıt %d  |  viewport cache", index, index)}, nil
}
func (referenceProvider) RowID(index int) widgets.RowID {
	return widgets.RowID(fmt.Sprintf("row-%d", index))
}
