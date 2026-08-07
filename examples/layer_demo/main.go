// layer_demo, Limoni Faz 10'un yeni özelliklerini gösterir:
// - Katmanlı Render (Layered Rendering) ile z-index bazlı çizim
// - Modal Pencere ile odak kilitlenmesi (Focus Trapping)
// - Açılır Menü (Popup) widget'ı
// - Click-outside close mekanizması
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

// AppState, demo uygulamasının durumunu temsil eder.
type AppState struct {
	// Modal durumu
	ShowDialog bool
	DialogMsg  string

	// Popup durumu
	FileMenuState *widgets.PopupState
	EditMenuState *widgets.PopupState
	HelpMenuState *widgets.PopupState
	LastAction    string
	StatusBar     string
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	state := &AppState{
		DialogMsg:     "Bu bir modal pencere örneğidir. Dışarı tıklayarak kapatın.",
		FileMenuState: widgets.NewPopupState(),
		EditMenuState: widgets.NewPopupState(),
		HelpMenuState: widgets.NewPopupState(),
		StatusBar:     "Hazır — F10 menü, Ctrl+N modal, Esc kapat",
	}

	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	// İlk kareyi çiz
	drawApp(t, state)

	for {
		select {
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				// Modal açıksa klavye girdisini ona yönlendir
				if state.ShowDialog {
					if ev.Key.Type == backend.KeyEsc {
						state.ShowDialog = false
						state.StatusBar = "Modal kapatıldı"
					}
					break
				}

				// Genel kısayollar
				switch {
				case ev.Key.Type == backend.KeyEsc:
					// Açık menü varsa kapat, yoksa çıkış
					if state.FileMenuState.IsOpen {
						state.FileMenuState.Close()
					} else if state.EditMenuState.IsOpen {
						state.EditMenuState.Close()
					} else if state.HelpMenuState.IsOpen {
						state.HelpMenuState.Close()
					}
					state.StatusBar = "Menüler kapatıldı"

				case ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'n' && ev.Key.Ctrl:
					// Ctrl+N: Modal aç
					state.ShowDialog = true
					state.StatusBar = "Modal pencere açıldı — Esc ile kapatın"

				case ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q':
					b.Close()
					fmt.Println("\nLimoni Faz 10 Demo'dan çıkış yapıldı. Görüşmek üzere!")
					os.Exit(0)

				case ev.Key.Type == backend.KeyF10:
					// Dosya menüsünü aç/kapat
					state.FileMenuState.Toggle()
					if state.FileMenuState.IsOpen {
						state.EditMenuState.Close()
						state.HelpMenuState.Close()
						state.StatusBar = "Dosya menüsü açıldı"
					} else {
						state.StatusBar = "Dosya menüsü kapatıldı"
					}

				case ev.Key.Type == backend.KeyF11:
					// Düzenleme menüsünü aç/kapat
					state.EditMenuState.Toggle()
					if state.EditMenuState.IsOpen {
						state.FileMenuState.Close()
						state.HelpMenuState.Close()
						state.StatusBar = "Düzenleme menüsü açıldı"
					} else {
						state.StatusBar = "Düzenleme menüsü kapatıldı"
					}

				case ev.Key.Type == backend.KeyF12:
					// Yardım menüsünü aç/kapat
					state.HelpMenuState.Toggle()
					if state.HelpMenuState.IsOpen {
						state.FileMenuState.Close()
						state.EditMenuState.Close()
						state.StatusBar = "Yardım menüsü açıldı"
					} else {
						state.StatusBar = "Yardım menüsü kapatıldı"
					}
				}

			case backend.EventMouse:
				if ev.Mouse.Button == backend.MouseLeft && !ev.Mouse.Drag {
					t.RouteMouseEvent(ev.Mouse)
				}

			case backend.EventResize:
				// Otomatik yeniden çizilecek
			}

		case <-ticker.C:
			drawApp(t, state)
		}
	}
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		area := f.Buffer.Area

		// Ana yerleşim: Header + Body + Footer
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

		// ── HEADER: Menü Butonları ──
		headerBlock := widgets.Block{
			Borders:       widgets.BorderBottom,
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
			BorderSymbols: widgets.SymbolsSingle,
		}
		f.RenderWidget(headerBlock, headerArea)

		// Menü butonları (Popup widget'ları)
		menuLay := layout.NewFlexLayout(
			layout.Horizontal, 0,
			layout.Fixed(16),
			layout.Fixed(16),
			layout.Fixed(16),
			layout.Fill(),
		)
		menuChunks := menuLay.Split(cell.NewRect(headerArea.X, headerArea.Y, headerArea.Width, 1))

		// Dosya Menüsü (F10)
		filePopup := widgets.Popup{
			ID:    "file_menu",
			Label: "Dosya (F10)",
			Items: []widgets.PopupItem{
				{Text: "Yeni", Handler: func() {
					state.LastAction = "Yeni dosya oluşturuldu"
					state.StatusBar = state.LastAction
				}},
				{Text: "Aç...", Handler: func() {
					state.LastAction = "Dosya açma dialogu"
					state.StatusBar = state.LastAction
				}},
				{Text: "Kaydet", Handler: func() {
					state.LastAction = "Dosya kaydedildi"
					state.StatusBar = state.LastAction
				}},
				{Text: "", Disabled: true}, // Ayırıcı
				{Text: "Çıkış", Handler: func() {
					state.StatusBar = "Çıkış isteği alındı (Ctrl+Q)"
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

		// Düzenleme Menüsü (F11)
		editPopup := widgets.Popup{
			ID:    "edit_menu",
			Label: "Düzenle (F11)",
			Items: []widgets.PopupItem{
				{Text: "Geri Al", Handler: func() {
					state.LastAction = "Son işlem geri alındı"
					state.StatusBar = state.LastAction
				}},
				{Text: "Yinele", Handler: func() {
					state.LastAction = "Son işlem yinelendi"
					state.StatusBar = state.LastAction
				}},
				{Text: "", Disabled: true},
				{Text: "Kopyala", Handler: func() {
					state.LastAction = "Panoya kopyalandı"
					state.StatusBar = state.LastAction
				}},
				{Text: "Yapıştır", Handler: func() {
					state.LastAction = "Panodan yapıştırıldı"
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

		// Yardım Menüsü (F12)
		helpPopup := widgets.Popup{
			ID:    "help_menu",
			Label: "Yardım (F12)",
			Items: []widgets.PopupItem{
				{Text: "Hakkında", Handler: func() {
					state.ShowDialog = true
					state.DialogMsg = "Limoni TUI Motoru — Faz 10: Katmanlı Render Demo"
					state.StatusBar = "Hakkında dialogu açıldı"
				}},
				{Text: "Kısayollar", Handler: func() {
					state.ShowDialog = true
					state.DialogMsg = "F10: Dosya | F11: Düzenle | F12: Yardım | Ctrl+N: Modal | Esc: Kapat"
					state.StatusBar = "Kısayollar dialogu açıldı"
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

		// ── BODY: İçerik Alanı ──
		contentBlock := widgets.Block{
			Title:          " Limoni Faz 10 — Katmanlı Render Demo ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
			Style:          cell.Style{Bg: cell.NewColorRGB(18, 18, 24)},
			Child: widgets.Paragraph{
				Text: "Hoş geldiniz!\n\n" +
					"Faz 10 Yenilikleri:\n" +
					"  • Katmanlı Render: z-index ile üst üste binen katmanlar\n" +
					"  • Modal Pencere: Odak kilitleme ve click-outside close\n" +
					"  • Açılır Menü (Popup): Dropdown menü desteği\n" +
					"  • Multi-layer olay yönlendirme\n\n" +
					"Menü butonlarına tıklayın veya kısayolları kullanın:\n" +
					"  F10: Dosya | F11: Düzenle | F12: Yardım | Ctrl+N: Modal",
				Style: cell.Style{Fg: cell.NewColorRGB(180, 200, 220)},
			},
		}
		f.RenderWidget(contentBlock, bodyArea)

		// ── FOOTER: Durum Çubuğu ──
		footerBlock := widgets.Block{
			Borders:      widgets.BorderTop,
			BorderStyle:  cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
			PaddingLeft:  1,
			PaddingRight: 1,
			Child: widgets.Paragraph{
				Text: state.StatusBar,
				Style: cell.Style{
					Fg: cell.NewColorRGB(140, 160, 180),
				},
			},
		}
		f.RenderWidget(footerBlock, footerArea)

		// ── MODAL KATMAN (Ctrl+N ile açılır) ──
		if state.ShowDialog {
			dialogW := uint16(50)
			dialogH := uint16(7)
			dialogArea := terminal.CenterRect(area, dialogW, dialogH)

			// Modal katmanını kaydet: z-index=2000 (menülerden üstün)
			f.RegisterLayer("main_dialog", terminal.LayerModal, dialogArea, 2000, func() {
				state.ShowDialog = false
				state.StatusBar = "Modal dışarı tıklama ile kapatıldı"
			})

			// Modal içindeki çizimi BeginLayer ile yap
			f.BeginLayer("main_dialog")

			dialog := widgets.Dialog{
				ID:      "info_dialog",
				Title:   "Bilgi",
				Message: state.DialogMsg,
				Buttons: []widgets.DialogButton{
					{
						Text: "Tamam",
						Handler: func() {
							state.ShowDialog = false
							state.StatusBar = "Modal Tamam ile kapatıldı"
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
