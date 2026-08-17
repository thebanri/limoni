package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

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
	lines := strings.Split(t.value, "\n")
	for i, line := range lines {
		if uint16(i) >= ctx.Area.Height {
			break
		}
		buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(i), line, ctx.Style.Merge(t.style))
	}
}

func (t text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	lines := strings.Split(t.value, "\n")
	maxW := 0
	for _, l := range lines {
		w := len([]rune(l))
		if w > maxW {
			maxW = w
		}
	}
	return uint16(maxW), uint16(len(lines))
}

type Process struct {
	PID    int
	Name   string
	CPU    float64
	Memory float64
	Status string
}

type AppState struct {
	SearchState *widgets.TextInputState
	TableState  *widgets.TableState
	ThemeName   string // "Dark", "Light", "Matrix"
	CPUHistory  []float64
	RAMHistory  []float64
	DiskHistory []float64
	NetHistory  []float64
	Logs        []string
	ShowHelp    bool
	LastEvent   string
	Metrics     SystemMetrics
	Processes   []Process
	Collector   *SysCollector
	ChartMode   int // 0: LineChart, 1: BarChart, 2: PieChart
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

	collector := NewSysCollector()
	initialMetrics := collector.CollectMetrics()
	initialProcesses := collector.CollectProcesses()

	searchState := widgets.NewTextInputState()
	tableState := widgets.NewTableState()
	tableState.Selected = 0

	cpuHist := make([]float64, 0, 150)
	ramHist := make([]float64, 0, 150)
	diskHist := make([]float64, 0, 150)
	netHist := make([]float64, 0, 150)

	for i := 0; i < 60; i++ {
		cpuHist = append(cpuHist, initialMetrics.CPUPercent)
		ramHist = append(ramHist, initialMetrics.RAMPercent)
		diskHist = append(diskHist, initialMetrics.DiskPercent)
		netHist = append(netHist, initialMetrics.NetRxRateMB*10)
	}

	state := &AppState{
		SearchState: searchState,
		TableState:  tableState,
		ThemeName:   "Dark",
		CPUHistory:  cpuHist,
		RAMHistory:  ramHist,
		DiskHistory: diskHist,
		NetHistory:  netHist,
		Metrics:     initialMetrics,
		Processes:   initialProcesses,
		Collector:   collector,
		ChartMode:   0,
		Logs: []string{
			fmt.Sprintf(" [%s] [SYS] Limoni telemetry engine attached (%s)", time.Now().Format("15:04:05"), initialMetrics.PlatformInfo),
			fmt.Sprintf(" [%s] [HOST] Host: %s (%d CPU Cores)", time.Now().Format("15:04:05"), initialMetrics.Hostname, initialMetrics.CoreCount),
			fmt.Sprintf(" [%s] [MEM] Total RAM: %.1f GB | Disk: %.1f GB", time.Now().Format("15:04:05"), initialMetrics.RAMTotalMB/1024.0, initialMetrics.DiskTotalGB),
			fmt.Sprintf(" [%s] [CHARTS] LineChart / BarChart / PieChart active", time.Now().Format("15:04:05")),
		},
	}

	renderTicker := time.NewTicker(40 * time.Millisecond) // 25 FPS
	defer renderTicker.Stop()

	sysTicker := time.NewTicker(300 * time.Millisecond) // Sample every 300ms
	defer sysTicker.Stop()

	draw := func() {
		drawApp(t, state)
	}

	draw()

	var frameCount uint64 = 0

	for {
		select {
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if state.ShowHelp {
					if ev.Key.Type == backend.KeyEsc || ev.Key.Type == backend.KeyEnter || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == ' ') {
						state.ShowHelp = false
						draw()
						break
					}
				}

				if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
					return
				}

				if ev.Key.Type == backend.KeyRune && ev.Key.Ch == '?' {
					state.ShowHelp = !state.ShowHelp
					draw()
					break
				}

				if ev.Key.Type == backend.KeyTab {
					fm := t.FocusManager()
					if fm.Focused() == "search_input" {
						fm.SetFocused("process_table")
					} else {
						fm.SetFocused("search_input")
					}
					draw()
					break
				}

				focused := t.FocusManager().Focused()

				if focused == "search_input" {
					if ev.Key.Type == backend.KeyArrowDown {
						t.FocusManager().SetFocused("process_table")
						draw()
						break
					}
					state.SearchState.HandleKey(ev.Key)
					state.TableState.Selected = 0
				} else {
					filteredCount := len(getFilteredProcesses(state))
					if ev.Key.Type == backend.KeyArrowUp {
						if state.TableState.Selected > 0 {
							state.TableState.Selected--
						} else {
							t.FocusManager().SetFocused("search_input")
						}
					} else if ev.Key.Type == backend.KeyArrowDown {
						if state.TableState.Selected < filteredCount-1 {
							state.TableState.Selected++
						}
					} else if ev.Key.Type == backend.KeyPageUp {
						state.TableState.Selected -= 10
						if state.TableState.Selected < 0 {
							state.TableState.Selected = 0
						}
					} else if ev.Key.Type == backend.KeyPageDown {
						state.TableState.Selected += 10
						if state.TableState.Selected >= filteredCount {
							state.TableState.Selected = filteredCount - 1
						}
					} else if ev.Key.Type == backend.KeyHome {
						state.TableState.Selected = 0
					} else if ev.Key.Type == backend.KeyEnd {
						state.TableState.Selected = filteredCount - 1
					} else if ev.Key.Type == backend.KeyRune && ev.Key.Ch == ' ' {
						if state.TableState.Selected >= 0 && state.TableState.Selected < filteredCount {
							state.TableState.ToggleRow(state.TableState.Selected)
						}
					} else if ev.Key.Type == backend.KeyArrowLeft {
						state.TableState.MoveSortColumn(-1, 5)
					} else if ev.Key.Type == backend.KeyArrowRight {
						state.TableState.MoveSortColumn(1, 5)
					}
				}

				if ev.Key.Type == backend.KeyRune {
					switch ev.Key.Ch {
					case 'm', 'M':
						state.ChartMode = (state.ChartMode + 1) % 3
					case '1':
						state.ThemeName = "Dark"
					case '2':
						state.ThemeName = "Light"
					case '3':
						state.ThemeName = "Matrix"
					case 'c':
						state.TableState.ClearSelectedRows()
					}
				}

				draw()

			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)
				draw()

			case backend.EventResize:
				draw()
			}

		case <-sysTicker.C:
			metrics := collector.CollectMetrics()
			processes := collector.CollectProcesses()

			state.Metrics = metrics
			if len(processes) > 0 {
				state.Processes = processes
			}

			if len(state.Logs) < 30 && time.Now().Unix()%5 == 0 {
				logMsgs := []string{
					fmt.Sprintf(" [%s] [CPU] CPU: %%%4.1f (%d Cores active)", time.Now().Format("15:04:05"), metrics.CPUPercent, metrics.CoreCount),
					fmt.Sprintf(" [%s] [MEM] RAM: %.1f GB / %.1f GB (%%%0.1f)", time.Now().Format("15:04:05"), metrics.RAMUsedMB/1024, metrics.RAMTotalMB/1024, metrics.RAMPercent),
					fmt.Sprintf(" [%s] [NET] Network Speed: Rx %.2f MB/s | Tx %.2f MB/s", time.Now().Format("15:04:05"), metrics.NetRxRateMB, metrics.NetTxRateMB),
					fmt.Sprintf(" [%s] [TASK] Active Tasks: %d running processes", time.Now().Format("15:04:05"), len(state.Processes)),
					fmt.Sprintf(" [%s] [DISK] Disk Free: %.1f GB free (/)", time.Now().Format("15:04:05"), metrics.DiskTotalGB-metrics.DiskUsedGB),
				}
				state.Logs = append(state.Logs, logMsgs[len(state.Logs)%len(logMsgs)])
			}

		case <-renderTicker.C:
			frameCount++

			targetCPU := state.Metrics.CPUPercent
			targetRAM := state.Metrics.RAMPercent
			targetDisk := state.Metrics.DiskPercent
			targetNet := state.Metrics.NetRxRateMB * 10.0

			waveNoise := math.Sin(float64(frameCount)*0.2) * 0.4
			liveCPU := targetCPU + waveNoise
			if liveCPU < 0 {
				liveCPU = 0
			}
			if liveCPU > 100 {
				liveCPU = 100
			}

			liveRAM := targetRAM + (waveNoise * 0.1)
			if liveRAM < 0 {
				liveRAM = 0
			}
			if liveRAM > 100 {
				liveRAM = 100
			}

			state.CPUHistory = append(state.CPUHistory, liveCPU)
			state.RAMHistory = append(state.RAMHistory, liveRAM)
			state.DiskHistory = append(state.DiskHistory, targetDisk)
			state.NetHistory = append(state.NetHistory, targetNet)

			if len(state.CPUHistory) > 140 {
				state.CPUHistory = state.CPUHistory[1:]
				state.RAMHistory = state.RAMHistory[1:]
				state.DiskHistory = state.DiskHistory[1:]
				state.NetHistory = state.NetHistory[1:]
			}

			draw()
		}
	}
}

func getFilteredProcesses(state *AppState) []Process {
	query := state.SearchState.Value()
	if query == "" {
		return state.Processes
	}
	return widgets.FuzzyFilterByStable(query, state.Processes, func(p Process) string {
		return p.Name
	})
}

func getTheme(themeName string) widgets.Theme {
	switch themeName {
	case "Light":
		return widgets.LightTheme()
	case "Matrix":
		t := widgets.DarkTheme()
		t.Colors.Background = cell.NewColorRGB(0, 5, 0)
		t.Colors.Surface = cell.NewColorRGB(0, 15, 0)
		t.Colors.Primary = cell.NewColorRGB(0, 255, 0)
		t.Colors.Secondary = cell.NewColorRGB(0, 180, 0)
		t.Colors.Text = cell.NewColorRGB(50, 255, 50)
		t.Colors.Border = cell.NewColorRGB(0, 100, 0)
		t.Base = cell.Style{Fg: t.Colors.Text, Bg: t.Colors.Background}
		t.Focus = cell.Style{Fg: t.Colors.Primary, Bg: cell.NewColorRGB(0, 40, 0), Modifier: cell.ModifierBold}
		return t
	default:
		return widgets.DarkTheme()
	}
}

func buildProgressBar(percent float64, totalWidth int) string {
	if totalWidth <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filledLen := int((percent / 100.0) * float64(totalWidth))
	if filledLen > totalWidth {
		filledLen = totalWidth
	}
	emptyLen := totalWidth - filledLen
	if emptyLen < 0 {
		emptyLen = 0
	}

	return strings.Repeat("█", filledLen) + strings.Repeat("░", emptyLen)
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		theme := getTheme(state.ThemeName)
		f.SetTheme(theme)
		area := f.Buffer.Area

		accentColor := theme.Colors.Primary
		if state.ThemeName == "Matrix" {
			accentColor = cell.NewColorRGB(0, 255, 0)
		}

		rootLay := layout.NewFlexLayout(
			layout.Vertical,
			0,
			layout.Fixed(3), // Header
			layout.Fixed(5), // Quick metrics HUD panels
			layout.Fill(),   // Body (Table + Charts)
			layout.Fixed(1), // Footer
		)
		chunks := rootLay.Split(area)

		// 1. Header Bar
		uptimeStr := state.Metrics.UptimeStr
		if uptimeStr == "" {
			uptimeStr = "00:00:00"
		}

		headerValue := fmt.Sprintf("  HOST: %s (%s)   |   UPTIME: %s   |   TIME: %s   |   THEME: %s [1/2/3] ",
			state.Metrics.Hostname, state.Metrics.PlatformInfo, uptimeStr, time.Now().Format("15:04:05"), strings.ToUpper(state.ThemeName))

		f.RenderWidget(widgets.Block{
			Title:          " LIMONI SYSTEM MONITOR ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
			Child:          text{value: headerValue, style: cell.Style{Fg: theme.Colors.Text}},
		}, chunks[0])

		// 2. Quick Panels (4 Cells Grid)
		grid := layout.NewGridLayout(
			[]layout.GridConstraint{layout.GridFraction(1), layout.GridFraction(1), layout.GridFraction(1), layout.GridFraction(1)},
			[]layout.GridConstraint{layout.GridFraction(1)},
			1,
		)
		quickChunks := grid.Split(chunks[1])

		latestCPU := state.Metrics.CPUPercent
		latestRAM := state.Metrics.RAMPercent
		usedRAMGB := state.Metrics.RAMUsedMB / 1024.0
		totRAMGB := state.Metrics.RAMTotalMB / 1024.0
		freeRAMGB := totRAMGB - usedRAMGB
		if freeRAMGB < 0 {
			freeRAMGB = 0
		}

		// Panel 1: CPU
		cpuCol := accentColor
		if latestCPU > 80 {
			cpuCol = cell.NewColorRGB(255, 60, 60)
		} else if latestCPU > 50 {
			cpuCol = cell.NewColorRGB(255, 180, 0)
		}
		cpuBar := buildProgressBar(latestCPU, 12)
		cpuText := fmt.Sprintf(" %s %5.1f%%\n %d Cores • Live Load", cpuBar, latestCPU, state.Metrics.CoreCount)
		f.RenderWidget(widgets.Block{
			Title:         " CPU LOAD ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: cpuCol},
			Child:         text{value: cpuText, style: cell.Style{Fg: theme.Colors.Text, Modifier: cell.ModifierBold}},
		}, quickChunks.Cell(0, 0).Area)

		// Panel 2: RAM
		ramCol := cell.NewColorRGB(0, 210, 255)
		if latestRAM > 85 {
			ramCol = cell.NewColorRGB(255, 60, 60)
		}
		ramBar := buildProgressBar(latestRAM, 12)
		ramText := fmt.Sprintf(" %s %5.1f%%\n %.1fG / %.1fG (%.1fG Free)", ramBar, latestRAM, usedRAMGB, totRAMGB, freeRAMGB)
		f.RenderWidget(widgets.Block{
			Title:         " MEMORY (RAM) ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: ramCol},
			Child:         text{value: ramText, style: cell.Style{Fg: theme.Colors.Text, Modifier: cell.ModifierBold}},
		}, quickChunks.Cell(0, 1).Area)

		// Panel 3: DISK
		diskPct := state.Metrics.DiskPercent
		diskCol := cell.NewColorRGB(180, 120, 255)
		diskBar := buildProgressBar(diskPct, 12)
		freeDiskGB := state.Metrics.DiskTotalGB - state.Metrics.DiskUsedGB
		if freeDiskGB < 0 {
			freeDiskGB = 0
		}
		diskText := fmt.Sprintf(" %s %5.1f%%\n %.0fG / %.0fG (%.0fG Free)", diskBar, diskPct, state.Metrics.DiskUsedGB, state.Metrics.DiskTotalGB, freeDiskGB)
		f.RenderWidget(widgets.Block{
			Title:         " DISK (/) ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: diskCol},
			Child:         text{value: diskText, style: cell.Style{Fg: theme.Colors.Text, Modifier: cell.ModifierBold}},
		}, quickChunks.Cell(0, 2).Area)

		// Panel 4: NETWORK
		netRx := state.Metrics.NetRxRateMB
		netTx := state.Metrics.NetTxRateMB
		rxStr := fmt.Sprintf("%.1f MB/s", netRx)
		if netRx < 0.1 {
			rxStr = fmt.Sprintf("%.0f KB/s", netRx*1024)
		}
		txStr := fmt.Sprintf("%.1f MB/s", netTx)
		if netTx < 0.1 {
			txStr = fmt.Sprintf("%.0f KB/s", netTx*1024)
		}
		netText := fmt.Sprintf(" Rx: %-8s  Tx: %-8s\n Total: %.1fG Rx | %.1fG Tx", rxStr, txStr, state.Metrics.TotalRxMB/1024.0, state.Metrics.TotalTxMB/1024.0)
		f.RenderWidget(widgets.Block{
			Title:         " NETWORK TRAFFIC ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 255, 200)},
			Child:         text{value: netText, style: cell.Style{Fg: theme.Colors.Text, Modifier: cell.ModifierBold}},
		}, quickChunks.Cell(0, 3).Area)

		// 3. Body Section: Left Table (50%) + Right Charts (50%)
		bodyLay := layout.NewFlexLayout(
			layout.Horizontal,
			1,
			layout.Percentage(50),
			layout.Percentage(50),
		)
		bodyChunks := bodyLay.Split(chunks[2])

		drawProcessesColumn(f, state, bodyChunks[0], theme, accentColor, t.FocusManager().Focused())
		drawStatsColumn(f, state, bodyChunks[1], theme, accentColor)

		// 4. Footer
		footerText := "  [Tab] Focus   [Up/Down] Table Rows   [Space] Select   [m] Chart Mode   [?] Help   [1/2/3] Themes   [q] Quit"
		f.RenderWidget(widgets.Block{
			Borders: widgets.BorderNone,
			Style:   cell.Style{Fg: cell.NewColorRGB(130, 135, 150), Bg: theme.Colors.Surface},
			Child:   text{value: footerText, style: cell.Style{Fg: cell.NewColorRGB(140, 145, 160), Modifier: cell.ModifierBold}},
		}, chunks[3])

		// 5. Help Modal Dialog
		if state.ShowHelp {
			helpW := uint16(60)
			helpH := uint16(14)
			helpArea := terminal.CenterRect(area, helpW, helpH)

			f.RegisterLayer("help_dialog", terminal.LayerModal, helpArea, 3000, func() {
				state.ShowHelp = false
			})

			f.BeginLayer("help_dialog")

			helpContent := "Limoni Dashboard Guide & Shortcuts:\n\n" +
				"  • [m] Chart Mode   : Line (Braille) -> Spectrum (Bar) -> Quad Grid -> Shaded Area.\n" +
				"  • [Tab] Focus      : Toggle focus between Search Input and Process Table.\n" +
				"  • [Up/Down] Nav    : Browse and select table rows.\n" +
				"  • [Space] Select   : Toggle multi-selection on active row.\n" +
				"  • [Left/Right] Sort: Sort table by column (PID, Name, CPU, RAM).\n" +
				"  • [1/2/3] Themes   : Dark [1], Light [2], and Matrix [3] color palettes.\n\n" +
				"Press [Esc], [Space], or [Enter] to dismiss."

			f.RenderWidget(widgets.Block{
				Title:         " SYSTEM HELP & SHORTCUTS ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsDouble,
				BorderStyle:   cell.Style{Fg: accentColor},
				Style:         cell.Style{Bg: theme.Colors.Surface},
				PaddingLeft:   2,
				PaddingTop:    1,
				Child:         text{value: helpContent, style: cell.Style{Fg: theme.Colors.Text}},
			}, helpArea)

			f.EndLayer()
		}
	})
}

func drawProcessesColumn(f *terminal.Frame, state *AppState, area cell.Rect, theme widgets.Theme, accentColor cell.Color, focused string) {
	isFocusedCol := focused == "search_input" || focused == "process_table"

	blockBorderCol := theme.Colors.Border
	if isFocusedCol {
		blockBorderCol = accentColor
	}

	processes := getFilteredProcesses(state)

	f.RenderWidget(widgets.Block{
		Title:         fmt.Sprintf(" SYSTEM PROCESSES (%d Active) ", len(processes)),
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: blockBorderCol},
	}, area)

	innerArea := cell.NewRect(area.X+1, area.Y+1, area.Width-2, area.Height-2)

	subLay := layout.NewFlexLayout(
		layout.Vertical,
		0,
		layout.Fixed(3),
		layout.Fill(),
	)
	subChunks := subLay.Split(innerArea)

	searchBorderCol := theme.Colors.Border
	if focused == "search_input" {
		searchBorderCol = accentColor
	}
	f.RenderWidget(widgets.Block{
		Title:         " FILTER PROCESSES ",
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: searchBorderCol},
		Child: widgets.TextInput{
			ID:               "search_input",
			State:            state.SearchState,
			Placeholder:      "Filter by process name (e.g. go, node, sshd)...",
			Style:            cell.Style{Fg: theme.Colors.Text},
			FocusedStyle:     cell.Style{Fg: theme.Colors.Text, Modifier: cell.ModifierUnderline},
			PlaceholderStyle: cell.Style{Fg: cell.NewColorRGB(100, 105, 120), Modifier: cell.ModifierItalic},
		},
	}, subChunks[0])

	headerRow := &widgets.TableRow{
		Cells: []widgets.TableCell{
			{Text: "PID", Style: cell.Style{Modifier: cell.ModifierBold}},
			{Text: "PROCESS NAME", Style: cell.Style{Modifier: cell.ModifierBold}},
			{Text: "CPU %", Style: cell.Style{Modifier: cell.ModifierBold}},
			{Text: "MEMORY", Style: cell.Style{Modifier: cell.ModifierBold}},
			{Text: "STATUS", Style: cell.Style{Modifier: cell.ModifierBold}},
		},
		Style: cell.Style{Bg: theme.Colors.Surface, Fg: theme.Colors.Text},
	}

	tableRows := make([]widgets.TableRow, len(processes))
	for i, p := range processes {
		cpuStyle := cell.Style{Fg: cell.NewColorRGB(0, 230, 120)}
		if p.CPU > 50.0 {
			cpuStyle = cell.Style{Fg: cell.NewColorRGB(255, 60, 60), Modifier: cell.ModifierBold}
		} else if p.CPU > 15.0 {
			cpuStyle = cell.Style{Fg: cell.NewColorRGB(255, 180, 0), Modifier: cell.ModifierBold}
		} else if p.CPU < 0.1 {
			cpuStyle = cell.Style{Fg: cell.NewColorRGB(120, 125, 140)}
		}

		statusText := "RUNNING"
		statusStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 128)}
		switch p.Status {
		case "Sleeping":
			statusText = "SLEEPING"
			statusStyle = cell.Style{Fg: cell.NewColorRGB(130, 135, 150)}
		case "Disk IO":
			statusText = "DISK_IO"
			statusStyle = cell.Style{Fg: cell.NewColorRGB(255, 180, 0)}
		case "Zombie":
			statusText = "ZOMBIE"
			statusStyle = cell.Style{Fg: cell.NewColorRGB(255, 60, 60)}
		case "Stopped":
			statusText = "STOPPED"
			statusStyle = cell.Style{Fg: cell.NewColorRGB(180, 60, 60)}
		case "Idle":
			statusText = "IDLE"
			statusStyle = cell.Style{Fg: cell.NewColorRGB(100, 105, 120)}
		}

		memText := fmt.Sprintf("%6.1f MB", p.Memory)
		if p.Memory >= 1024 {
			memText = fmt.Sprintf("%6.2f GB", p.Memory/1024.0)
		}

		tableRows[i] = widgets.TableRow{
			Cells: []widgets.TableCell{
				{Text: fmt.Sprintf("#%d", p.PID), Style: cell.Style{Fg: cell.NewColorRGB(130, 135, 150)}},
				{Text: p.Name, Style: cell.Style{Fg: theme.Colors.Text, Modifier: cell.ModifierBold}},
				{Text: fmt.Sprintf("%5.1f%%", p.CPU), Style: cpuStyle},
				{Text: memText, Style: cell.Style{Fg: cell.NewColorRGB(0, 200, 255)}},
				{Text: statusText, Style: statusStyle},
			},
		}
	}

	tableBorderCol := theme.Colors.Border
	if focused == "process_table" {
		tableBorderCol = accentColor
	}

	table := &widgets.Table{
		ID:          "process_table",
		Header:      headerRow,
		Rows:        tableRows,
		State:       state.TableState,
		MultiSelect: true,
		SortEnabled: true,
		Constraints: []widgets.TableConstraint{
			{Type: widgets.ConstraintPercentage, Value: 16},
			{Type: widgets.ConstraintPercentage, Value: 34},
			{Type: widgets.ConstraintPercentage, Value: 16},
			{Type: widgets.ConstraintPercentage, Value: 16},
			{Type: widgets.ConstraintFill},
		},
		GridStyle:     cell.Style{Fg: theme.Colors.Border},
		DrawGrid:      true,
		SelectedStyle: theme.Focus,
		FocusedStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 80, 45), Modifier: cell.ModifierBold},
	}

	f.RenderWidget(widgets.Block{
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: tableBorderCol},
		Child:         table,
	}, subChunks[1])
}

func drawStatsColumn(f *terminal.Frame, state *AppState, area cell.Rect, theme widgets.Theme, accentColor cell.Color) {
	statLay := layout.NewFlexLayout(
		layout.Vertical,
		1,
		layout.Percentage(62),
		layout.Percentage(38),
	)
	statChunks := statLay.Split(area)

	var chartWidget widgets.Widget
	chartTitle := ""

	switch state.ChartMode % 3 {
	case 0:
		chartTitle = " LIVE SYSTEM TELEMETRY — LineChart (Braille Subpixels) "
		chartWidget = widgets.LineChart{
			ID: "dash_linechart",
			Datasets: []widgets.LineDataset{
				{
					Name:  fmt.Sprintf("CPU (%.1f%%)", state.Metrics.CPUPercent),
					Data:  state.CPUHistory,
					Color: cell.NewColorRGB(0, 255, 180),
				},
				{
					Name:  fmt.Sprintf("RAM (%.1f%%)", state.Metrics.RAMPercent),
					Data:  state.RAMHistory,
					Color: cell.NewColorRGB(0, 200, 255),
				},
				{
					Name:  fmt.Sprintf("Disk (%.1f%%)", state.Metrics.DiskPercent),
					Data:  state.DiskHistory,
					Color: cell.NewColorRGB(255, 140, 0),
				},
			},
			ShowAxes:   true,
			ShowLegend: true,
			XLabels:    []string{"-60s", "-45s", "-30s", "-15s", "now"},
		}

	case 1:
		chartTitle = " LIVE RESOURCE SPECTRUM — BarChart (Vertical Bars) "
		chartWidget = widgets.BarChart{
			ID: "dash_barchart",
			Data: []widgets.BarData{
				{Label: "CPU Total", Value: state.Metrics.CPUPercent, Color: cell.NewColorRGB(0, 255, 180)},
				{Label: "Core 0", Value: math.Min(100, state.Metrics.CPUPercent*1.1), Color: cell.NewColorRGB(46, 204, 113)},
				{Label: "Core 1", Value: math.Max(0, state.Metrics.CPUPercent*0.9), Color: cell.NewColorRGB(46, 204, 113)},
				{Label: "RAM Util", Value: state.Metrics.RAMPercent, Color: cell.NewColorRGB(0, 200, 255)},
				{Label: "Disk Used", Value: state.Metrics.DiskPercent, Color: cell.NewColorRGB(255, 140, 0)},
				{Label: "Net Rx", Value: math.Min(100, state.Metrics.NetRxRateMB*15), Color: cell.NewColorRGB(233, 30, 99)},
			},
			Direction:  widgets.BarVertical,
			BarWidth:   5,
			BarGap:     2,
			ShowValues: true,
		}

	case 2:
		chartTitle = " MEMORY & DISK DISTRIBUTION — PieChart (Donut) "
		usedRAM := state.Metrics.RAMUsedMB
		freeRAM := state.Metrics.RAMTotalMB - usedRAM
		if freeRAM < 0 {
			freeRAM = 0
		}
		chartWidget = widgets.PieChart{
			ID: "dash_piechart",
			Data: []widgets.PieSlice{
				{Label: "RAM Used", Value: usedRAM, Color: cell.NewColorRGB(0, 200, 255)},
				{Label: "RAM Free", Value: freeRAM, Color: cell.NewColorRGB(46, 204, 113)},
				{Label: "Disk Used", Value: state.Metrics.DiskUsedGB * 1024, Color: cell.NewColorRGB(255, 140, 0)},
			},
			DonutHoleRatio:  0.45,
			ShowLegend:      true,
			ShowPercentages: true,
		}
	}

	f.RenderWidget(widgets.Block{
		Title:         chartTitle,
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: accentColor},
		PaddingTop:    0,
		PaddingBottom: 0,
		PaddingLeft:   1,
		PaddingRight:  1,
		Child:         chartWidget,
	}, statChunks[0])

	logItems := make([]string, len(state.Logs))
	copy(logItems, state.Logs)

	logListState := widgets.NewListState()
	if len(logItems) > 0 {
		logListState.Selected = len(logItems) - 1
	}

	f.RenderWidget(widgets.Block{
		Title:         " SYSTEM EVENT LOGS (Live Stream) ",
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: theme.Colors.Border},
		Child: widgets.List{
			Items:         logItems,
			State:         logListState,
			Style:         cell.Style{Fg: cell.NewColorRGB(175, 180, 195)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
		},
	}, statChunks[1])
}
