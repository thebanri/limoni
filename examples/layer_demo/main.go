// layer_demo demonstrates Limoni layered rendering capabilities:
// - Layered Rendering with z-index based compositing
// - Modal Dialog with Focus Trapping
// - Popup / Dropdown menus
// - Click-outside dismissal mechanism
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

// AppState represents the interactive layer demo state.
type AppState struct {
	ShowDialog    bool
	DialogMsg     string
	FileMenuState *widgets.PopupState
	EditMenuState *widgets.PopupState
	HelpMenuState *widgets.PopupState
	LastAction    string
	StatusBar     string
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

	state := &AppState{
		DialogMsg:     "This is a modal dialog demonstration. Click outside to dismiss.",
		FileMenuState: widgets.NewPopupState(),
		EditMenuState: widgets.NewPopupState(),
		HelpMenuState: widgets.NewPopupState(),
		StatusBar:     "Ready — F10 File menu, Ctrl+N Modal, Esc Close",
	}

	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	drawApp(t, state)

	for {
		select {
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if state.ShowDialog {
					if ev.Key.Type == backend.KeyEsc {
						state.ShowDialog = false
						state.StatusBar = "Modal dismissed"
					}
					break
				}

				switch {
				case ev.Key.Type == backend.KeyEsc:
					if state.FileMenuState.IsOpen {
						state.FileMenuState.Close()
					} else if state.EditMenuState.IsOpen {
						state.EditMenuState.Close()
					} else if state.HelpMenuState.IsOpen {
						state.HelpMenuState.Close()
					}
					state.StatusBar = "Menus closed"

				case ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'n' && ev.Key.Ctrl:
					state.ShowDialog = true
					state.StatusBar = "Modal dialog opened — Press Esc to close"

				case ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q':
					b.Close()
					fmt.Println("\nExited Limoni Layer Demo.")
					os.Exit(0)

				case ev.Key.Type == backend.KeyF10:
					state.FileMenuState.Toggle()
					if state.FileMenuState.IsOpen {
						state.EditMenuState.Close()
						state.HelpMenuState.Close()
						state.StatusBar = "File menu opened"
					} else {
						state.StatusBar = "File menu closed"
					}

				case ev.Key.Type == backend.KeyF11:
					state.EditMenuState.Toggle()
					if state.EditMenuState.IsOpen {
						state.FileMenuState.Close()
						state.HelpMenuState.Close()
						state.StatusBar = "Edit menu opened"
					} else {
						state.StatusBar = "Edit menu closed"
					}

				case ev.Key.Type == backend.KeyF12:
					state.HelpMenuState.Toggle()
					if state.HelpMenuState.IsOpen {
						state.FileMenuState.Close()
						state.EditMenuState.Close()
						state.StatusBar = "Help menu opened"
					} else {
						state.StatusBar = "Help menu closed"
					}
				}

			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)

			case backend.EventResize:
				// Automatically redrawn on next tick
			}

		case <-ticker.C:
			drawApp(t, state)
		}
	}
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		area := f.Buffer.Area

		rootLay := layout.NewFlexLayout(
			layout.Vertical, 0,
			layout.Fixed(1), // Header
			layout.Fill(),   // Body
			layout.Fixed(1), // Footer
		)
		chunks := rootLay.Split(area)

		headerArea := chunks[0]
		bodyArea := chunks[1]
		footerArea := chunks[2]

		// ── HEADER: Menu Bar ──
		headerBlock := widgets.Block{
			Borders:       widgets.BorderBottom,
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
			BorderSymbols: widgets.SymbolsSingle,
		}
		f.RenderWidget(headerBlock, headerArea)

		menuLay := layout.NewFlexLayout(
			layout.Horizontal, 0,
			layout.Fixed(16),
			layout.Fixed(16),
			layout.Fixed(16),
			layout.Fill(),
		)
		menuChunks := menuLay.Split(cell.NewRect(headerArea.X, headerArea.Y, headerArea.Width, 1))

		// File Menu (F10)
		filePopup := widgets.Popup{
			ID:    "file_menu",
			Label: "File (F10)",
			Items: []widgets.PopupItem{
				{Text: "New", Handler: func() {
					state.LastAction = "New file created"
					state.StatusBar = state.LastAction
				}},
				{Text: "Open...", Handler: func() {
					state.LastAction = "Open file dialog"
					state.StatusBar = state.LastAction
				}},
				{Text: "Save", Handler: func() {
					state.LastAction = "File saved"
					state.StatusBar = state.LastAction
				}},
				{Text: "", Disabled: true}, // Separator
				{Text: "Quit", Handler: func() {
					state.StatusBar = "Quit requested (Ctrl+Q)"
				}},
			},
			State:         state.FileMenuState,
			Style:         cell.Style{Bg: cell.NewColorRGB(30, 30, 40), Fg: cell.NewColorRGB(220, 220, 220)},
			ItemStyle:     cell.Style{Fg: cell.NewColorRGB(200, 200, 200)},
			SelectedStyle: cell.Style{Bg: cell.NewColorRGB(50, 80, 140), Fg: cell.NewColorRGB(255, 255, 255)},
			DisabledStyle: cell.Style{Fg: cell.NewColorRGB(80, 80, 80)},
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(80, 80, 100)},
		}
		f.RenderWidget(filePopup, menuChunks[0])

		// Edit Menu (F11)
		editPopup := widgets.Popup{
			ID:    "edit_menu",
			Label: "Edit (F11)",
			Items: []widgets.PopupItem{
				{Text: "Undo", Handler: func() {
					state.LastAction = "Action undone"
					state.StatusBar = state.LastAction
				}},
				{Text: "Redo", Handler: func() {
					state.LastAction = "Action redone"
					state.StatusBar = state.LastAction
				}},
				{Text: "", Disabled: true},
				{Text: "Copy", Handler: func() {
					state.LastAction = "Copied to clipboard"
					state.StatusBar = state.LastAction
				}},
				{Text: "Paste", Handler: func() {
					state.LastAction = "Pasted from clipboard"
					state.StatusBar = state.LastAction
				}},
			},
			State:         state.EditMenuState,
			Style:         cell.Style{Bg: cell.NewColorRGB(30, 30, 40), Fg: cell.NewColorRGB(220, 220, 220)},
			ItemStyle:     cell.Style{Fg: cell.NewColorRGB(200, 200, 200)},
			SelectedStyle: cell.Style{Bg: cell.NewColorRGB(50, 80, 140), Fg: cell.NewColorRGB(255, 255, 255)},
			DisabledStyle: cell.Style{Fg: cell.NewColorRGB(80, 80, 80)},
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(80, 80, 100)},
		}
		f.RenderWidget(editPopup, menuChunks[1])

		// Help Menu (F12)
		helpPopup := widgets.Popup{
			ID:    "help_menu",
			Label: "Help (F12)",
			Items: []widgets.PopupItem{
				{Text: "About", Handler: func() {
					state.ShowDialog = true
					state.DialogMsg = "Limoni TUI Engine — Layered Rendering Demo"
					state.StatusBar = "About dialog opened"
				}},
				{Text: "Shortcuts", Handler: func() {
					state.ShowDialog = true
					state.DialogMsg = "F10: File | F11: Edit | F12: Help | Ctrl+N: Modal | Esc: Close"
					state.StatusBar = "Shortcuts dialog opened"
				}},
			},
			State:         state.HelpMenuState,
			Style:         cell.Style{Bg: cell.NewColorRGB(30, 30, 40), Fg: cell.NewColorRGB(220, 220, 220)},
			ItemStyle:     cell.Style{Fg: cell.NewColorRGB(200, 200, 200)},
			SelectedStyle: cell.Style{Bg: cell.NewColorRGB(50, 80, 140), Fg: cell.NewColorRGB(255, 255, 255)},
			DisabledStyle: cell.Style{Fg: cell.NewColorRGB(80, 80, 80)},
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(80, 80, 100)},
		}
		f.RenderWidget(helpPopup, menuChunks[2])

		// ── BODY: Content Area ──
		contentBlock := widgets.Block{
			Title:          " Limoni — Layered Rendering Demo ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
			Style:          cell.Style{Bg: cell.NewColorRGB(18, 18, 24)},
			Child: &widgets.Paragraph{
				Text: "Welcome!\n\n" +
					"Layered Engine Features:\n" +
					"  • Layered Rendering: z-index based compositing\n" +
					"  • Modal Windows: Focus trapping and click-outside dismissal\n" +
					"  • Popup Menus: Fast dropdown support with animations\n" +
					"  • Multi-layer event routing\n\n" +
					"Click menu buttons or use keyboard shortcuts:\n" +
					"  F10: File | F11: Edit | F12: Help | Ctrl+N: Modal",
				Style: cell.Style{Fg: cell.NewColorRGB(180, 200, 220)},
			},
		}
		f.RenderWidget(contentBlock, bodyArea)

		// ── FOOTER: Status Bar ──
		footerBlock := widgets.Block{
			Borders:      widgets.BorderTop,
			BorderStyle:  cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
			PaddingLeft:  1,
			PaddingRight: 1,
			Child: &widgets.Paragraph{
				Text: state.StatusBar,
				Style: cell.Style{
					Fg: cell.NewColorRGB(140, 160, 180),
				},
			},
		}
		f.RenderWidget(footerBlock, footerArea)

		// ── MODAL LAYER (Ctrl+N) ──
		if state.ShowDialog {
			dialogW := uint16(50)
			dialogH := uint16(7)
			dialogArea := terminal.CenterRect(area, dialogW, dialogH)

			f.RegisterLayer("main_dialog", terminal.LayerModal, dialogArea, 2000, func() {
				state.ShowDialog = false
				state.StatusBar = "Modal dismissed by outside click"
			})

			f.BeginLayer("main_dialog")

			dialog := widgets.Dialog{
				ID:      "info_dialog",
				Title:   "Information",
				Message: state.DialogMsg,
				Buttons: []widgets.DialogButton{
					{
						Text: "OK",
						Handler: func() {
							state.ShowDialog = false
							state.StatusBar = "Modal closed with OK"
						},
					},
				},
				Style:       cell.Style{Bg: cell.NewColorRGB(25, 25, 35)},
				BorderStyle: cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
				ButtonStyle: cell.Style{Fg: cell.NewColorRGB(160, 180, 200)},
				ButtonFocusedStyle: cell.Style{
					Bg:       cell.NewColorRGB(50, 80, 140),
					Fg:       cell.NewColorRGB(255, 255, 255),
					Modifier: cell.ModifierBold,
				},
				BorderSymbols: widgets.SymbolsDouble,
			}
			f.RenderWidget(dialog, dialogArea)

			f.EndLayer()
		}
	})
}
