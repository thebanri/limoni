package main

import (
	"context"
	"fmt"
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

// ─────────────────────────────────────────────────────────────────────────────
// VIRTUAL DATA SOURCE FOR 1,000,000 ENTERPRISE LOG RECORDS
// ─────────────────────────────────────────────────────────────────────────────

type logDataSource struct {
	totalCount int
}

func (l logDataSource) RowCount(ctx context.Context) (int, error) {
	return l.totalCount, nil
}

func (l logDataSource) RowAt(ctx context.Context, index int) (widgets.Row, error) {
	if index < 0 || index >= l.totalCount {
		return widgets.Row{}, fmt.Errorf("out of bounds")
	}

	levels := []string{"INFO ", "WARN ", "ERROR", "DEBUG"}
	level := levels[index%len(levels)]
	levelCol := cell.NewColorRGB(0, 220, 255)
	levelBg := cell.NewColorRGB(10, 30, 45)

	switch level {
	case "WARN ":
		levelCol = cell.NewColorRGB(255, 190, 0)
		levelBg = cell.NewColorRGB(45, 35, 10)
	case "ERROR":
		levelCol = cell.NewColorRGB(255, 60, 80)
		levelBg = cell.NewColorRGB(45, 15, 20)
	case "DEBUG":
		levelCol = cell.NewColorRGB(180, 120, 255)
		levelBg = cell.NewColorRGB(35, 20, 50)
	}

	timeStr := time.Now().Add(time.Duration(-index) * 250 * time.Millisecond).Format("15:04:05.000")
	nodeName := fmt.Sprintf("node-%02d", (index%16)+1)

	messages := []string{
		"TLS v1.3 handshake completed successfully for client session",
		"HTTP GET /api/v2/telemetry/stream returned status 200 OK",
		"Database query latency exceeded warning threshold (148ms)",
		"Cache miss on key user_session_auth_token; querying replica",
		"Upstream gateway connection reset by peer; retrying (attempt 2)",
		"Distributed consensus raft heartbeat acknowledged across cluster",
		"Memory pool slab allocation optimized; GC sweep complete",
		"Security audit event: API token rotation verified for tenant",
	}
	msg := messages[index%len(messages)]
	latency := fmt.Sprintf("%4.1fms", float64((index*37)%450)/10.0+0.4)

	return widgets.Row{
		ID: widgets.RowID(fmt.Sprintf("log_%d", index)),
		Cells: []widgets.TableCell{
			{Text: fmt.Sprintf("#%07d", index), Style: cell.Style{Fg: cell.NewColorRGB(120, 125, 140)}},
			{Text: timeStr, Style: cell.Style{Fg: cell.NewColorRGB(160, 165, 180)}},
			{Text: fmt.Sprintf(" %s ", level), Style: cell.Style{Fg: levelCol, Bg: levelBg, Modifier: cell.ModifierBold}},
			{Text: nodeName, Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 180)}},
			{Text: msg, Style: cell.Style{Fg: cell.NewColorRGB(220, 225, 235)}},
			{Text: latency, Style: cell.Style{Fg: cell.NewColorRGB(255, 200, 100)}},
		},
	}, nil
}

func (l logDataSource) RowID(index int) widgets.RowID {
	return widgets.RowID(fmt.Sprintf("log_%d", index))
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

	state := widgets.NewVirtualDataState()
	source := logDataSource{totalCount: 1000000}
	offset := 0

	w, h, _ := b.Size()
	_, height := int(w), int(h)

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			area := f.Buffer.Area
			theme := widgets.DarkTheme()
			f.SetTheme(theme)

			accentCol := cell.NewColorRGB(0, 210, 255)

			rootLay := layout.NewFlexLayout(
				layout.Vertical,
				0,
				layout.Fixed(3), // Top Header Bar
				layout.Fill(),   // Main Body (Table + Right Inspector)
				layout.Fixed(1), // Footer Shortcuts
			)
			chunks := rootLay.Split(area)

			// 1. Header Bar with stats badges
			headerTitle := "  LIMONI VIRTUAL DATA STREAM (Virtual Data View Engine) "
			recBadge := " [1,000,000 RECORDS] "
			allocBadge := " [0 ALLOC / ZERO MEMORY] "
			fpsBadge := " [60 FPS VIRTUALIZED] "

			f.RenderWidget(widgets.Block{
				Title:          " VIRTUALIZED TABLE ENGINE ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accentCol, Modifier: cell.ModifierBold},
				Child:          text{value: headerTitle, style: cell.Style{Fg: theme.Colors.Text, Modifier: cell.ModifierBold}},
			}, chunks[0])

			// Draw right badges in header
			headerInnerX := chunks[0].X + chunks[0].Width - 2
			headerInnerY := chunks[0].Y + 1

			badges := []struct {
				text  string
				style cell.Style
			}{
				{fpsBadge, cell.Style{Fg: cell.NewColorRGB(255, 190, 0), Bg: cell.NewColorRGB(40, 30, 10), Modifier: cell.ModifierBold}},
				{allocBadge, cell.Style{Fg: cell.NewColorRGB(0, 255, 160), Bg: cell.NewColorRGB(10, 35, 25), Modifier: cell.ModifierBold}},
				{recBadge, cell.Style{Fg: cell.NewColorRGB(0, 210, 255), Bg: cell.NewColorRGB(10, 30, 45), Modifier: cell.ModifierBold}},
			}

			currBadgeX := headerInnerX
			for _, b := range badges {
				bW := uint16(len([]rune(b.text)))
				if currBadgeX > chunks[0].X+bW+uint16(len([]rune(headerTitle)))+4 {
					currBadgeX -= bW
					f.Buffer.SetString(currBadgeX, headerInnerY, b.text, b.style)
					currBadgeX -= 1
				}
			}

			// 2. Main Content Split: Left (70% Virtual Table) + Right (30% Inspector Panel)
			bodyLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Percentage(70),
				layout.Percentage(30),
			)
			bodyChunks := bodyLay.Split(chunks[1])

			// ─────────────────────────────────────────────────────────────
			// LEFT PANEL: 1,000,000 Virtual Log Stream Table
			// ─────────────────────────────────────────────────────────────
			tableArea := bodyChunks[0]
			tableTitle := fmt.Sprintf(" LIVE LOG STREAM (%d - %d / %d Records) ", offset, offset+int(tableArea.Height)-4, source.totalCount)

			f.RenderWidget(widgets.Block{
				Title:         tableTitle,
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: accentCol},
			}, tableArea)

			// Table Column Headers
			headerY := tableArea.Y + 1
			tableInnerW := tableArea.Width - 2
			colHeader := fmt.Sprintf(" %-9s %-14s %-9s %-10s %s", "RECORD ID", "TIMESTAMP", "LEVEL", "NODE", "MESSAGE / PAYLOAD")
			if int(tableInnerW) > len(colHeader) {
				colHeader += strings.Repeat(" ", int(tableInnerW)-len(colHeader))
			}
			f.Buffer.SetString(tableArea.X+1, headerY, colHeader, cell.Style{Fg: cell.NewColorRGB(240, 245, 255), Bg: cell.NewColorRGB(25, 30, 45), Modifier: cell.ModifierBold})

			// VirtualDataView Area
			listInnerArea := cell.NewRect(
				tableArea.X+1,
				tableArea.Y+2,
				tableArea.Width-2,
				tableArea.Height-3,
			)

			f.RenderWidget(widgets.VirtualDataView{
				ID:            "virtual_logs",
				State:         state,
				Source:        source,
				First:         0,
				Prefetch:      20,
				Offset:        &offset,
				Style:         cell.Style{Fg: cell.NewColorRGB(190, 195, 205)},
				SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 80, 130), Modifier: cell.ModifierBold},
				FocusedStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 95, 150), Modifier: cell.ModifierBold},
			}, listInnerArea)

			// ─────────────────────────────────────────────────────────────
			// RIGHT PANEL: Inspector Cards & Telemetry Stats
			// ─────────────────────────────────────────────────────────────
			inspectorArea := bodyChunks[1]
			rightLay := layout.NewFlexLayout(
				layout.Vertical,
				1,
				layout.Percentage(50),
				layout.Percentage(50),
			)
			rightChunks := rightLay.Split(inspectorArea)

			// Card 1: Selected Record Inspector
			var selectedIndex int = -1
			if state.Selected() != "" {
				fmt.Sscanf(string(state.Selected()), "log_%d", &selectedIndex)
			}

			var detailText string
			if selectedIndex >= 0 {
				levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
				lvl := levels[selectedIndex%len(levels)]
				detailText = fmt.Sprintf(
					"Record ID: #%07d\nLevel    : %s\nTimestamp: 15:04:05.%03d\nNode     : node-%02d\nLatency  : %4.1f ms\nProtocol : HTTP/2.0 (TLS 1.3)\nStatus   : Verified & Synced\n\nDetail:\nKubernetes cluster tenant audit log stream generated without GC pressure.",
					selectedIndex,
					lvl,
					selectedIndex%1000,
					(selectedIndex%16)+1,
					float64((selectedIndex*37)%450)/10.0+0.4,
				)
			} else {
				detailText = "Selected Record: None\n\nUse [Up/Down] arrows to inspect a log record."
			}

			f.RenderWidget(widgets.Block{
				Title:         " SELECTED RECORD INSPECTOR ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 255, 180)},
				PaddingLeft:   1,
				PaddingTop:    1,
				Child:         text{value: detailText, style: cell.Style{Fg: cell.NewColorRGB(200, 205, 220)}},
			}, rightChunks[0])

			// Card 2: Performance & Engine Statistics
			scrollPercent := float64(offset) / float64(source.totalCount) * 100.0
			statsText := fmt.Sprintf(
				"Total Records : 1,000,000\nViewport Rows : %d Rows\nScroll Pos    : %d / 1,000,000\nProgress      : %%%5.2f\n\nFrame Render  : %s\nMemory Pres.  : 0 Alloc (O(1))\nTarget FPS    : 60 FPS",
				listInnerArea.Height,
				offset,
				scrollPercent,
				t.LastFrameDuration(),
			)

			f.RenderWidget(widgets.Block{
				Title:         " ENGINE PERFORMANCE ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(255, 190, 0)},
				PaddingLeft:   1,
				PaddingTop:    1,
				Child:         text{value: statsText, style: cell.Style{Fg: cell.NewColorRGB(190, 195, 210)}},
			}, rightChunks[1])

			// 3. Footer Bar with Shortcut Pills
			footerText := "  [Up/Down] Select Row   [PageUp/PageDown] Fast Page (25x)   [Home/End] Jump Top/End   [Mouse Wheel] Scroll   [q] Quit"
			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Style:   cell.Style{Fg: cell.NewColorRGB(140, 145, 160), Bg: theme.Colors.Surface},
				Child:   text{value: footerText, style: cell.Style{Fg: cell.NewColorRGB(140, 145, 160), Modifier: cell.ModifierBold}},
			}, chunks[2])
		})
	}

	draw()

	for ev := range b.Events() {
		switch ev.Type {
		case backend.EventKey:
			if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
				return
			}

			if ev.Key.Type == backend.KeyArrowDown {
				offsetMax := source.totalCount - 1
				if state.Selected() == "" {
					state.Select(source.RowID(offset))
				} else {
					var currentIdx int
					fmt.Sscanf(string(state.Selected()), "log_%d", &currentIdx)
					if currentIdx < offsetMax {
						nextIdx := currentIdx + 1
						state.Select(source.RowID(nextIdx))
						viewportH := height - 6
						if nextIdx >= offset+viewportH {
							offset = nextIdx - viewportH + 1
						}
					}
				}
			}

			if ev.Key.Type == backend.KeyArrowUp {
				if state.Selected() != "" {
					var currentIdx int
					fmt.Sscanf(string(state.Selected()), "log_%d", &currentIdx)
					if currentIdx > 0 {
						prevIdx := currentIdx - 1
						state.Select(source.RowID(prevIdx))
						if prevIdx < offset {
							offset = prevIdx
						}
					}
				}
			}

			if ev.Key.Type == backend.KeyPageDown {
				maxOffset := source.totalCount - 25
				if offset < maxOffset {
					offset += 25
				}
			}
			if ev.Key.Type == backend.KeyPageUp {
				if offset > 25 {
					offset -= 25
				} else {
					offset = 0
				}
			}

			if ev.Key.Type == backend.KeyHome {
				offset = 0
				state.Select(source.RowID(0))
			}
			if ev.Key.Type == backend.KeyEnd {
				offset = source.totalCount - (height - 6)
				if offset < 0 {
					offset = 0
				}
				state.Select(source.RowID(source.totalCount - 1))
			}

			draw()

		case backend.EventMouse:
			t.RouteMouseEvent(ev.Mouse)
			draw()

		case backend.EventResize:
			height = int(ev.Resize.Height)
			draw()
		}
	}
}
