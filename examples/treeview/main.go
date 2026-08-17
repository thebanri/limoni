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

	treeState := widgets.NewTreeViewState()
	treeState.Select("main_go")

	roots := []widgets.TreeNode{
		{
			ID:       "limoni_root",
			Label:    "limoni (v2.0-performance)",
			Icon:     "📁",
			Expanded: true,
			Children: []widgets.TreeNode{
				{
					ID:       "core_dir",
					Label:    "core",
					Icon:     "📁",
					Expanded: true,
					Children: []widgets.TreeNode{
						{ID: "buffer_go", Label: "buffer.go", Icon: "📄", Data: "Zero-allocation diff engine"},
						{ID: "cell_go", Label: "cell.go", Icon: "📄", Data: "TrueColor & packed cell grid"},
						{ID: "terminal_go", Label: "terminal.go", Icon: "📄", Data: "Layering & frame orchestration"},
					},
				},
				{
					ID:       "widgets_dir",
					Label:    "widgets",
					Icon:     "📁",
					Expanded: true,
					Children: []widgets.TreeNode{
						{ID: "treeview_go", Label: "treeview.go", Icon: "🌳", Data: "Hierarchical collapsible tree explorer"},
						{ID: "charts_go", Label: "barchart.go / linechart.go", Icon: "📊", Data: "Braille subpixel data visualization"},
						{ID: "colorpicker_go", Label: "colorpicker.go", Icon: "🎨", Data: "HSV, RGB & Hex interactive picker"},
						{ID: "toast_go", Label: "toast.go", Icon: "🔔", Data: "Timed notification stack overlay"},
					},
				},
				{
					ID:       "examples_dir",
					Label:    "examples",
					Icon:     "📁",
					Expanded: false,
					Children: []widgets.TreeNode{
						{ID: "demo_app", Label: "demo/main.go", Icon: "🚀"},
						{ID: "paint_app", Label: "paint/main.go", Icon: "🎨"},
						{ID: "dash_app", Label: "dashboard/main.go", Icon: "📊"},
					},
				},
				{ID: "gomod", Label: "go.mod", Icon: "📦", Data: "Go module definition"},
				{ID: "readme", Label: "README.md", Icon: "📖", Data: "Limoni project documentation"},
			},
		},
	}

	selectedDetails := "Select a file to inspect details."
	updateDetails := func() {
		if node := widgets.FindNode(roots, treeState.SelectedID); node != nil {
			if node.Data != nil {
				selectedDetails = fmt.Sprintf("ID: %s\nLabel: %s\nDetails: %v", node.ID, node.Label, node.Data)
			} else {
				selectedDetails = fmt.Sprintf("ID: %s\nLabel: %s\n(Directory node)", node.ID, node.Label)
			}
		}
	}
	updateDetails()

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			area := f.Buffer.Area
			accent := cell.NewColorRGB(0, 255, 180)

			chunks := layout.VBox(area, layout.Fixed(3), layout.Fill(), layout.Fixed(1))

			// Header
			f.RenderWidget(widgets.Block{
				Title:          " 🌳 LIMONI TREEVIEW EXPLORER ",
				TitleAlignment: widgets.AlignCenter,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accent, Modifier: cell.ModifierBold},
				Child:          &widgets.Paragraph{Text: " Hierarchical Collapsible Tree Widget with Guide Lines & Keyboard Navigation "},
			}, chunks[0])

			// Body: Left Tree (50%) + Right Details (50%)
			cols := layout.HBox(chunks[1], layout.Percentage(50), layout.Percentage(50))

			treeView := widgets.TreeView{
				ID:         "file_tree",
				Roots:      roots,
				State:      treeState,
				ShowGuides: true,
			}

			f.RenderWidget(widgets.Block{
				Title:         " FILE TREE (Arrows / Enter to expand) ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 135, 160)},
				Child:         treeView,
			}, cols[0])

			f.RenderWidget(widgets.Block{
				Title:         " NODE METADATA & PREVIEW ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 135, 160)},
				Child: &widgets.Paragraph{
					Text: selectedDetails,
					Style: cell.Style{
						Fg: cell.NewColorRGB(0, 220, 255),
					},
				},
			}, cols[1])

			// Footer
			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Child: &widgets.Paragraph{
					Text: " [▲/▼] Navigate  [▶] Expand  [◀] Collapse  [Enter/Space] Toggle  [q] Quit",
					Style: cell.Style{
						Fg: cell.NewColorRGB(140, 150, 165),
					},
				},
			}, chunks[2])
		})
	}

	draw()

	renderTicker := time.NewTicker(50 * time.Millisecond)
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
				if ev.Key.Type == backend.KeyRune && (ev.Key.Ch == 'q' || ev.Key.Ch == 'Q') {
					return
				}
				if ev.Key.Type == backend.KeyEsc {
					return
				}
				if treeState.HandleKey(ev.Key, roots) {
					updateDetails()
					draw()
				}
			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)
				updateDetails()
				draw()
			case backend.EventResize:
				draw()
			}
		}
	}
}
