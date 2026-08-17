package main

import (
	"fmt"
	"os"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

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

	toastMgr := widgets.NewToastManager(widgets.ToastTopRight)
	toastMgr.Success("System Initialized", "Telemetry stream online.")
	toastMgr.Info("Welcome", "Press 1-4 to trigger notifications.")

	posNames := []string{"Top-Right", "Top-Left", "Bottom-Right", "Bottom-Left"}
	posIdx := 0

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			area := f.Buffer.Area
			accent := cell.NewColorRGB(52, 152, 219)

			chunks := layout.VBox(area, layout.Fixed(3), layout.Fill(), layout.Fixed(1))

			// Header
			f.RenderWidget(widgets.Block{
				Title:          " 🔔 LIMONI TOAST NOTIFICATION STUDIO ",
				TitleAlignment: widgets.AlignCenter,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accent, Modifier: cell.ModifierBold},
				Child:          &widgets.Paragraph{Text: fmt.Sprintf(" Position: %s | Active Notifications: %d ", posNames[posIdx], len(toastMgr.Toasts))},
			}, chunks[0])

			// Control Panel
			instructions := "# Notification Controls & Keybindings\n\n" +
				"- Press **[1]** → Trigger **Info Toast** (`ToastInfo`)\n" +
				"- Press **[2]** → Trigger **Success Toast** (`ToastSuccess`)\n" +
				"- Press **[3]** → Trigger **Warning Toast** (`ToastWarning`)\n" +
				"- Press **[4]** → Trigger **Error Toast** (`ToastError`)\n\n" +
				"- Press **[p]** → Cycle Screen Corner Position\n" +
				"- Press **[c]** → Dismiss All Active Toasts\n" +
				"- Press **[q]** → Exit Studio\n\n" +
				"> *Toasts automatically expire after their preset duration and can also be dismissed by clicking directly on them.*"

			f.RenderWidget(widgets.Block{
				Title:         " CONTROL PANEL ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 130, 150)},
				PaddingLeft:   2,
				PaddingTop:    1,
				Child: &widgets.Markdown{
					Content: instructions,
				},
			}, chunks[1])

			// Footer
			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Child: &widgets.Paragraph{
					Text: " [1] Info  [2] Success  [3] Warning  [4] Error  [p] Position  [c] Clear  [q] Quit",
					Style: cell.Style{
						Fg: cell.NewColorRGB(140, 150, 165),
					},
				},
			}, chunks[2])

			// Render active toasts with drop shadow
			toastMgr.Update(time.Now())
			toastMgr.Draw(cell.NewContext(area, cell.Style{}), f.Buffer)
		})
	}

	draw()

	renderTicker := time.NewTicker(40 * time.Millisecond)
	defer renderTicker.Stop()

	for {
		select {
		case <-renderTicker.C:
			draw()
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if ev.Key.Type == backend.KeyRune {
					switch ev.Key.Ch {
					case 'q', 'Q':
						return
					case '1':
						toastMgr.Info("Package Update", "New version v2.1.0 available.")
					case '2':
						toastMgr.Success("Build Completed", "Artifacts generated in 142ms.")
					case '3':
						toastMgr.Warning("High Memory Usage", "Heap memory exceeded 85%.")
					case '4':
						toastMgr.Error("Connection Failed", "Connection timeout to host 10.0.0.1.")
					case 'p', 'P':
						posIdx = (posIdx + 1) % 4
						toastMgr.Position = widgets.ToastPosition(posIdx)
					case 'c', 'C':
						toastMgr.Toasts = nil
					}
					draw()
				}
				if ev.Key.Type == backend.KeyEsc {
					return
				}
			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)
				draw()
			case backend.EventResize:
				draw()
			}
		}
	}
}
