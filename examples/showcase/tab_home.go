package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawHome(t *terminal.Terminal, f *terminal.Frame, state *AppState, demoTheme widgets.Theme, mainColor, accentColor cell.Color, bodyArea cell.Rect) {
	gisLay := layout.NewFlexLayout(
		layout.Vertical,
		1,
		layout.Fixed(uint16(state.MarkdownHeight)), // Markdown viewer
		layout.Fixed(5), // Slider bar
		layout.Fixed(1), // Progress bar
		layout.Fill(),   // Process Table
	)
	gisChunks := gisLay.Split(bodyArea)

	// 1. TOP: Markdown info block
	mdBorderColor := accentColor
	if t.FocusManager().Focused() == "demo_markdown" {
		mdBorderColor = demoTheme.Colors.Primary
	}
	mdBlock := widgets.Block{
		Title:          " INFORMATION (↑/↓ scroll, +/- resize) ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: mdBorderColor},
		PaddingLeft:    2,
		PaddingRight:   2,
		Child: &widgets.Markdown{
			ID:           "demo_markdown",
			Content:      state.DemoMarkdown,
			Style:        cell.Style{Fg: demoTheme.Colors.Text},
			FocusedStyle: demoTheme.Focus,
			ScrollOffset: &state.MarkdownOffset,
		},
	}
	f.RenderWidget(mdBlock, gisChunks[0])

	if gisChunks[0].Width > 2 && gisChunks[0].Height > 2 {
		cornerX := gisChunks[0].X + gisChunks[0].Width - 1
		cornerY := gisChunks[0].Y + gisChunks[0].Height - 1
		if c := f.Buffer.Get(cornerX, cornerY); c != nil {
			c.Content = '◢'
			c.Style = demoTheme.Focus
		}
		resizeArea := cell.NewRect(cornerX, cornerY, 1, 1)
		registerTargetClick(f, resizeArea, func(ev backend.MouseEvent) {
			if ev.Button != backend.MouseLeft {
				return
			}
			startY, baseHeight := int(ev.Y), state.MarkdownHeight
			f.CaptureMouse(func(dragEv backend.MouseEvent) {
				if dragEv.Button == backend.MouseRelease {
					return
				}
				if dragEv.Drag {
					next := baseHeight + int(dragEv.Y) - startY
					if next < 4 {
						next = 4
					}
					if next > 12 {
						next = 12
					}
					state.MarkdownHeight = next
				}
			})
		})
	}

	// Live CPU Progress bar
	cpuValue := 0.0
	if state.FormProgress != nil {
		cpuValue = state.FormProgress.Value()
	}
	if gisChunks[2].Height > 0 && gisChunks[2].Width > 6 {
		progressArea := cell.NewRect(gisChunks[2].X+2, gisChunks[2].Y, gisChunks[2].Width-4, 1)
		f.RenderWidget(widgets.ProgressBar{
			Value: cpuValue, Min: 0, Max: 100, ShowPercent: true,
			FilledStyle: cell.Style{Fg: accentColor},
			EmptyStyle:  cell.Style{Fg: demoTheme.Colors.Border},
		}, progressArea)
	}

	sliderBorder := accentColor
	if t.FocusManager().Focused() == "demo_slider" {
		sliderBorder = demoTheme.Colors.Primary
	}
	f.RenderWidget(widgets.Block{
		Title: " LOAD SLIDER (↑/↓) ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: sliderBorder}, PaddingLeft: 1, PaddingRight: 1,
		Child: widgets.Slider{ID: "demo_slider", State: state.DemoSliderState, Min: 0, Max: 100, TrackStyle: cell.Style{Fg: demoTheme.Colors.Border}, FilledStyle: cell.Style{Fg: demoTheme.Colors.Success}, ThumbStyle: demoTheme.Focus},
	}, gisChunks[1])

	// 2. BOTTOM: System Process Table
	tableRows := make([]widgets.TableRow, len(state.Processes)+2)
	for i, p := range state.Processes {
		tableRows[i] = widgets.TableRow{
			Cells: []widgets.TableCell{
				{Text: p.PID},
				{Text: p.Name},
				{Text: p.CPU},
				{Text: p.Memory},
				{Text: p.Status, Style: cell.Style{Fg: demoTheme.Colors.Success}},
			},
		}
		if i%2 == 1 {
			tableRows[i].Style = cell.Style{Bg: cell.NewColorRGB(35, 35, 35)}
		}
	}

	// RowSpan demonstration rows
	reportRowStyle := cell.Style{Bg: cell.NewColorRGB(25, 35, 45)}
	tableRows[len(state.Processes)] = widgets.TableRow{
		Cells: []widgets.TableCell{
			{Text: "REPORT", RowSpan: 2, Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 255), Modifier: cell.ModifierBold}},
			{Text: "Total CPU Load", Style: cell.Style{Modifier: cell.ModifierItalic}},
			{Text: "18.4%"},
			{Text: "Normal"},
			{Text: "Stable", Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}},
		},
		Style: reportRowStyle,
	}
	tableRows[len(state.Processes)+1] = widgets.TableRow{
		Cells: []widgets.TableCell{
			{Text: "Total Memory Load", Style: cell.Style{Modifier: cell.ModifierItalic}},
			{Text: "4.8 GB"},
			{Text: "Low"},
			{Text: "Stable", Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}},
		},
		Style: reportRowStyle,
	}

	tableBorderCol := cell.NewColorRGB(100, 100, 100)
	if t.FocusManager().Focused() == "process_table" {
		tableBorderCol = accentColor
	}

	sysTable := widgets.Table{
		ID: "process_table",
		Header: &widgets.TableRow{
			Cells: []widgets.TableCell{
				{Text: "PID", Style: cell.Style{Modifier: cell.ModifierBold}},
				{Text: "PROCESS NAME", Style: cell.Style{Modifier: cell.ModifierBold}},
				{Text: "CPU", Style: cell.Style{Modifier: cell.ModifierBold}},
				{Text: "MEMORY", Style: cell.Style{Modifier: cell.ModifierBold}},
				{Text: "STATUS", Style: cell.Style{Modifier: cell.ModifierBold}},
			},
			Style: cell.Style{Bg: cell.NewColorRGB(45, 45, 45)},
		},
		Rows: tableRows,
		Constraints: []widgets.TableConstraint{
			{Type: widgets.ConstraintFixed, Value: 6},       // PID
			{Type: widgets.ConstraintPercentage, Value: 30}, // Name
			{Type: widgets.ConstraintFixed, Value: 8},       // CPU
			{Type: widgets.ConstraintFixed, Value: 12},      // Memory
			{Type: widgets.ConstraintFill},                  // Status
		},
		State:     state.TableState,
		GridStyle: cell.Style{Fg: cell.NewColorRGB(70, 70, 70)},
		SelectedStyle: cell.Style{
			Fg:       cell.NewColorRGB(255, 255, 255),
			Bg:       accentColor,
			Modifier: cell.ModifierBold,
		},
		DrawGrid:      true,
		SortEnabled:   true,
		MultiSelect:   true,
		StickyColumns: 1,
		FilterQuery:   state.TableFilterState.Value(),
		CellStyle: func(row, column int, value widgets.TableCell) cell.Style {
			if row < 0 {
				return cell.Style{}
			}
			if column == 2 {
				if cpu, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(value.Text), "%"), 64); err == nil {
					if cpu >= 5 {
						return cell.Style{Fg: cell.NewColorRGB(255, 90, 90), Modifier: cell.ModifierBold}
					}
					if cpu >= 2 {
						return cell.Style{Fg: cell.NewColorRGB(255, 210, 80)}
					}
				}
			}
			if column == 4 && (value.Text == "Sleeping" || value.Text == "Beklemede") {
				return cell.Style{Fg: cell.NewColorRGB(255, 210, 80)}
			}
			return cell.Style{}
		},
	}

	tableBlock := widgets.Block{
		Title:          " SYSTEM PROCESS TABLE ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: tableBorderCol},
		Child:          sysTable,
	}
	tableLay := layout.NewFlexLayout(layout.Vertical, 1, layout.Fixed(3), layout.Fill())
	tableChunks := tableLay.Split(gisChunks[3])
	filterBorderCol := cell.NewColorRGB(100, 100, 100)
	if t.FocusManager().Focused() == "table_filter" {
		filterBorderCol = accentColor
	}
	sortLabel := "Click Header"
	if state.TableState.SortColumn >= 0 && state.TableState.SortColumn < 5 {
		columns := []string{"PID", "PROCESS", "CPU", "MEMORY", "STATUS"}
		direction := "▲ asc"
		if state.TableState.SortDescending {
			direction = "▼ desc"
		}
		sortLabel = fmt.Sprintf("%s %s", columns[state.TableState.SortColumn], direction)
	}
	filterBlock := widgets.Block{
		Title: fmt.Sprintf(" SEARCH TABLE | Sort: %s | ←/→ col ↑/↓ dir ", sortLabel), TitleAlignment: widgets.AlignLeft,
		Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: filterBorderCol}, PaddingLeft: 1, PaddingRight: 1,
		Child: widgets.TextInput{ID: "table_filter", State: state.TableFilterState, Placeholder: "Fuzzy filter: process, cpu, status...", Style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(35, 35, 45)}, FocusedStyle: cell.Style{Fg: accentColor}},
	}
	f.RenderWidget(filterBlock, tableChunks[0])
	f.RenderWidget(tableBlock, tableChunks[1])

	if t.FocusManager().Focused() == "process_table" {
		widgets.DrawFocusRing(f.Buffer, tableChunks[1], cell.Style{Fg: accentColor})
	}
}
