package main

import (
	"fmt"
	"os"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

// AppState, interaktif demo uygulamasının durumunu (state) temsil eder.
type AppState struct {
	// ActiveTab, sol menüde hangi sekmenin aktif olduğunu belirtir (örn. "Giriş", "Ayarlar").
	ActiveTab string
	// LastKey, klavyeden basılan son tuş bilgisini ekranda göstermek için saklar.
	LastKey string
	// LastMouse, fare ile yapılan son eylemin (tıklama, hareket) bilgisini saklar.
	LastMouse string
	// ExitButtonArea, Çıkış butonunun ekrandaki koordinatlarını tutar.
	ExitButtonArea cell.Rect
	// SettingsListState, Ayarlar sekmesindeki listenin durumunu saklar.
	SettingsListState *widgets.ListState
}

func main() {
	// Standard I/O kullanarak terminal backend'ini oluştur
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}
	// Program bittiğinde terminal ayarlarını (Raw mode, ekran temizleme vb.) restore et
	defer b.Close()

	// Terminal yöneticisini (Double-buffer ve çizim motoru) oluştur
	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}

	// Asenkron olay okuyucu Event Loop'u başlat
	b.StartEventLoop()

	// Başlangıç uygulama durumunu ata
	state := &AppState{
		ActiveTab:         "Giriş",
		LastKey:           "Yok",
		LastMouse:         "Yok",
		SettingsListState: widgets.NewListState(),
	}

	// İlk kareyi (frame) çiz
	drawApp(t, b, state)

	// Olay dinleme döngüsü (Event Loop)
	for ev := range b.Events() {
		switch ev.Type {
		case backend.EventKey:
			// ESC veya 'q' basılırsa programdan çık
			if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
				return
			}
			state.LastKey = fmt.Sprintf("Kod: %d, Karakter: %q, Ctrl: %v", ev.Key.Type, string(ev.Key.Ch), ev.Key.Ctrl)
			drawApp(t, b, state)

		case backend.EventMouse:
			// Çıkış butonunun üstüne gelindiyse veya tıklandıysa çıkış yap
			if state.ExitButtonArea.Contains(ev.Mouse.X, ev.Mouse.Y) {
				b.Close()
				fmt.Println("\nLimoni TUI uygulamasından (Çıkış butonunun üzerine gelindiği/tıklandığı için) çıkış yapıldı. Görüşmek üzere!")
				os.Exit(0)
			}

			// Fare olayını otomatik tıklama yönlendiriciye ilet (RouteMouseEvent).
			// Eğer tıklama kayıtlı bir bölgeye isabet ettiyse callback çalışır ve RouteMouseEvent 'true' döner.
			if t.RouteMouseEvent(ev.Mouse) {
				// Eşleşen bir buton/öğe tıklandı, ekranı yenile
				drawApp(t, b, state)
			} else {
				// Eşleşen bir buton yoksa, son fare koordinat ve eylemini genel ekrana yazmak için kaydet
				state.LastMouse = fmt.Sprintf("Buton: %d, Pozisyon: (%d, %d), Sürükleme: %v", ev.Mouse.Button, ev.Mouse.X, ev.Mouse.Y, ev.Mouse.Drag)
				drawApp(t, b, state)
			}

		case backend.EventResize:
			// Pencere boyutu değiştikçe ekranı yeniden çiz
			drawApp(t, b, state)
		}
	}
}

// drawApp, uygulamanın durumunu okur ve ekranın yerleşimini çizdirir.
func drawApp(t *terminal.Terminal, b *backend.Backend, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		// 1. Ekranı dikeyde 3 bölgeye ayır:
		// - Header (Sabit 3 satır)
		// - Body (Kalan tüm dikey alan)
		// - Footer (Sabit 1 satır)
		rootLay := layout.NewFlexLayout(
			layout.Vertical,
			0,
			layout.Fixed(3),
			layout.Fill(),
			layout.Fixed(1),
		)
		chunks := rootLay.Split(f.Buffer.Area)

		// 2. Header (Başlık Paneli) Çizimi
		headerBlock := widgets.Block{
			Title:          " LİMONİ TUI MOTORU DEMO ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 255)}, // Cyan çerçeve
			Child:          label{text: " Ratatui'den esinlenilmiş, daha modern ve esnek! ", style: cell.Style{Fg: cell.NewColorRGB(255, 255, 255)}},
		}
		f.RenderWidget(headerBlock, chunks[0])

		// 3. Body (Gövde) Çizimi
		// Yatayda Sol Panel (Sabit 22 sütun menü) ve Sağ Panel (Kalan esnek içerik alanı) olarak böl
		bodyLay := layout.NewFlexLayout(
			layout.Horizontal,
			1, // Aralarında 1 hücre boşluk bırak
			layout.Fixed(22),
			layout.Fill(),
		)
		bodyChunks := bodyLay.Split(chunks[1])

		// Sol Panel (Menü Bölmesi) - Dikeyde 3 adet buton ve esnek boşluk içerir
		menuLay := layout.NewFlexLayout(
			layout.Vertical,
			1, // Butonlar arasında 1 satır boşluk
			layout.Fixed(3), // Giriş Butonu
			layout.Fixed(3), // Ayarlar Butonu
			layout.Fixed(3), // Çıkış Butonu
			layout.Fill(),
		)
		menuChunks := menuLay.Split(bodyChunks[0])

		// drawButton, sol menüdeki tıklanabilir buton kutularını ve event callback'lerini çizer
		drawButton := func(area cell.Rect, title string, tabName string) {
			borderCol := cell.NewColorRGB(120, 120, 120)
			titleStyle := cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}

			// Eğer buton aktif sekmeye aitse yeşil çerçeve ve kalın yazı yap
			if state.ActiveTab == tabName {
				borderCol = cell.NewColorRGB(0, 255, 0)
				titleStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 0), Modifier: cell.ModifierBold}
			}

			btn := widgets.Block{
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: borderCol},
				Title:          title,
				TitleAlignment: widgets.AlignCenter,
				TitleStyle:     titleStyle,
			}
			f.RenderWidget(btn, area)

			// Fare tıklamasını bu bölgeye kaydet (RegisterClickHandler)
			f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
				state.ActiveTab = tabName
				drawApp(t, b, state) // Durum değişti, ekranı yeniden çiz
			})
		}

		drawButton(menuChunks[0], "1. Giris", "Giriş")
		drawButton(menuChunks[1], "2. Ayarlar", "Ayarlar")
		drawButton(menuChunks[2], "3. Cikis", "Çıkış")

		// Çıkış butonu koordinat alanını kaydet
		state.ExitButtonArea = menuChunks[2]

		// Çıkış butonuna özel tıklama olayı
		f.RegisterClickHandler(menuChunks[2], func(ev backend.MouseEvent) {
			b.Close() // Terminal ayarlarını düzelt ve alternatif ekrandan çık
			fmt.Println("\nLimoni TUI uygulamasından güvenle çıkış yapıldı. Görüşmek üzere!")
			os.Exit(0)
		})

		// Sağ Panel (İçerik Paneli) Çizimi
		var contentWidget widgets.Widget
		switch state.ActiveTab {
		case "Giriş":
			contentWidget = widgets.Block{
				Title:          " GİRİŞ GÖRÜNÜMÜ ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 0)},
				PaddingLeft:    2,
				PaddingTop:     1,
				Child: widgets.Paragraph{
					Text:  fmt.Sprintf("Fare ile sol menü sekmelerine tıklayarak geçiş yapabilirsiniz.\n\nSon Basılan Tuş: %s\nSon Fare Hareketi: %s", state.LastKey, state.LastMouse),
					Style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
					Wrap:  true,
				},
			}
		case "Ayarlar":
			contentWidget = widgets.Block{
				Title:          " LİMONİ SİSTEM AYARLARI ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 165, 0)}, // Turuncu çerçeve
				PaddingLeft:    2,
				PaddingRight:   2,
				PaddingTop:     1,
				PaddingBottom:  1,
				Child: widgets.List{
					Items: []string{
						"Tema Seçimi (TrueColor)",
						"Mouse Modu Aktif (SGR)",
						"Senkronize Görüntü Açık (?2026)",
						"Klavye Zaman Aşımı (25ms)",
						"L2 Cache Optimizasyonu (Hızlı)",
						"Double-Buffering Aktif",
						"Düzen Pazarlığı (SizeHint) Devrede",
					},
					Style:           cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
					SelectedStyle:   cell.Style{Fg: cell.NewColorRGB(255, 165, 0), Modifier: cell.ModifierBold},
					HighlightSymbol: "> ",
					State:           state.SettingsListState,
				},
			}
		}
		f.RenderWidget(contentWidget, bodyChunks[1])

		// 4. Footer (Alt Bilgi Satırı) Çizimi
		footerStyle := cell.Style{Fg: cell.NewColorRGB(140, 140, 140), Bg: cell.NewColorRGB(30, 30, 30)}
		footerText := fmt.Sprintf(" Boyut: %d x %d | Çıkış için 'q' veya ESC tuşuna basın", f.Buffer.Area.Width, f.Buffer.Area.Height)

		footerBlock := widgets.Block{
			Borders: widgets.BorderNone,
			Style:   footerStyle,
			Child:   label{text: footerText, style: footerStyle},
		}
		f.RenderWidget(footerBlock, chunks[2])
	})
}

// label, çok satırlı metinleri (\n) çizim sınırlarına uygun olarak alt alta çizen basit bir metin widget'ıdır.
type label struct {
	text  string
	style cell.Style
}

// Draw, metni satır satır bölerek çizim sınırları (Area) içine yazar.
func (l label) Draw(ctx cell.Context, buf *buffer.Buffer) {
	mergedStyle := ctx.Style.Merge(l.style)

	currY := ctx.Area.Y
	lineStart := 0
	for i := 0; i < len(l.text); i++ {
		if l.text[i] == '\n' {
			if currY < ctx.Area.Y+ctx.Area.Height {
				buf.SetString(ctx.Area.X, currY, l.text[lineStart:i], mergedStyle)
				currY++
			}
			lineStart = i + 1
		}
	}
	if lineStart < len(l.text) && currY < ctx.Area.Y+ctx.Area.Height {
		buf.SetString(ctx.Area.X, currY, l.text[lineStart:], mergedStyle)
	}
}

// SizeHint, metnin satır sayısını ve en uzun satırının uzunluğunu bularak ideal boyutları hesaplar.
func (l label) SizeHint(maxArea cell.Rect) (width, height uint16) {
	lines := 1
	maxW := 0
	currW := 0
	for i := 0; i < len(l.text); i++ {
		if l.text[i] == '\n' {
			lines++
			if currW > maxW {
				maxW = currW
			}
			currW = 0
		} else {
			currW++
		}
	}
	if currW > maxW {
		maxW = currW
	}
	return uint16(maxW), uint16(lines)
}
