package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type AppState struct {
	SidebarWidth *animation.Float
	SidebarOpen  bool

	ButtonColor *animation.Color
	ColorIndex  int

	BounceVal *animation.Float
	Bouncing  bool

	LastKey   string
	LastMouse string
	FPS       float64
}

func main() {
	// Standard I/O kullanarak terminal backend'ini oluştur
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	// Terminal yöneticisini oluştur
	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}

	// Olay dinleme döngüsünü asenkron olarak başlat
	b.StartEventLoop()

	state := &AppState{
		SidebarWidth: animation.NewFloat(6),
		SidebarOpen:  false,
		ButtonColor:  animation.NewColor(cell.NewColorRGB(0, 100, 255)), // Başlangıçta mavi
		ColorIndex:   0,
		BounceVal:    animation.NewFloat(0),
		Bouncing:     false,
		LastKey:      "Yok",
		LastMouse:    "Yok",
	}

	// 30 FPS zamanlayıcısı (~33ms)
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	frameCount := 0
	lastFpsCalc := time.Now()

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
				if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
					return
				}

				if ev.Key.Type == backend.KeySpace {
					// Sidebar genişliğini anime et (Slide-in / Slide-out)
					state.SidebarOpen = !state.SidebarOpen
					target := 6.0
					if state.SidebarOpen {
						target = 24.0
					}
					state.SidebarWidth.AnimateTo(target, 400*time.Millisecond, animation.EaseInOutCubic)
				}

				if ev.Key.Type == backend.KeyEnter {
					// Buton rengini anime et (Color Blend/Fade)
					state.ColorIndex = (state.ColorIndex + 1) % 4
					var targetColor cell.Color
					switch state.ColorIndex {
					case 0:
						targetColor = cell.NewColorRGB(0, 100, 255) // Mavi
					case 1:
						targetColor = cell.NewColorRGB(255, 0, 100) // Pembe
					case 2:
						targetColor = cell.NewColorRGB(0, 255, 100) // Yeşil
					case 3:
						targetColor = cell.NewColorRGB(255, 180, 0) // Turuncu
					}
					state.ButtonColor.AnimateTo(targetColor, 500*time.Millisecond, animation.EaseInOutQuad)
				}

				if ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'b' {
					// Zıplama animasyonunu başlat
					state.BounceVal.SetValue(0)
					state.BounceVal.AnimateTo(10, 1000*time.Millisecond, animation.EaseOutBounce)
				}

				state.LastKey = fmt.Sprintf("Kod: %d, Karakter: %q", ev.Key.Type, string(ev.Key.Ch))

			case backend.EventMouse:
				t.RouteMouseEvent(ev.Mouse)
				state.LastMouse = fmt.Sprintf("Buton: %d, Pozisyon: (%d, %d)", ev.Mouse.Button, ev.Mouse.X, ev.Mouse.Y)

			case backend.EventResize:
				// Boyut değişiminde çizimi tetikle
			}

		case <-ticker.C:
			// Zaman tabanlı animasyon nesnelerini güncelle
			now := time.Now()
			state.SidebarWidth.Update(now)
			state.ButtonColor.Update(now)
			state.BounceVal.Update(now)

			// Ekranı yeniden çiz
			drawApp(t, state)

			// FPS hesaplama
			frameCount++
			if time.Since(lastFpsCalc) >= 1*time.Second {
				state.FPS = float64(frameCount) / time.Since(lastFpsCalc).Seconds()
				frameCount = 0
				lastFpsCalc = time.Now()
			}
		}
	}
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		// Dikey bölümlendirme
		rootLay := layout.NewFlexLayout(
			layout.Vertical,
			0,
			layout.Fixed(3),
			layout.Fill(),
			layout.Fixed(1),
		)
		chunks := rootLay.Split(f.Buffer.Area)

		// 1. Header
		f.RenderWidget(widgets.Block{
			Title:          " LİMONİ TUI - FAZ 8: ANİMASYON MOTORU DEMOSU ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 255)},
			Child:          label{text: " Zaman tabanlı interpolasyon, yumuşak geçişler ve easing efektleri ", style: cell.Style{Fg: cell.NewColorRGB(255, 255, 255)}},
		}, chunks[0])

		// 2. Body
		sidebarW := uint16(math.Round(state.SidebarWidth.Value()))
		bodyLay := layout.NewFlexLayout(
			layout.Horizontal,
			1,
			layout.Fixed(sidebarW),
			layout.Fill(),
		)
		bodyChunks := bodyLay.Split(chunks[1])

		// Sol Panel (Sidebar)
		sidebarTitle := "MENÜ"
		if sidebarW >= 10 {
			sidebarTitle = " HIZLI ERİŞİM "
		}
		sidebarText := "...\n...\n..."
		if sidebarW >= 15 {
			sidebarText = "🟢 Sistem Aktif\n⏱️ FPS Kararlı\n🎨 RGB Renkler\n🚀 Faz 8 Başarılı"
		}

		sidebarBlock := widgets.Block{
			Title:         sidebarTitle,
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 120, 120)},
			PaddingLeft:   1,
			PaddingTop:    1,
			Child:         label{text: sidebarText, style: cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}},
		}
		f.RenderWidget(sidebarBlock, bodyChunks[0])

		// Sağ Panel (İçerik Alanı)
		rightLay := layout.NewFlexLayout(
			layout.Vertical,
			1,
			layout.Percentage(50),
			layout.Percentage(50),
		)
		rightChunks := rightLay.Split(bodyChunks[1])

		// Üst Bölüm: Renk Animasyonu
		btnCol := state.ButtonColor.Value()
		colorBox := widgets.Block{
			Title:          " RENK GEÇİŞİ (Blend / Fade) ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: btnCol},
			PaddingLeft:    2,
			PaddingTop:     1,
			Child: &widgets.Paragraph{
				Text:  fmt.Sprintf("Kutunun çerçeve rengi yumuşak bir şekilde değişmektedir.\n\n[Enter] tuşuna basarak veya bu kutuya tıklayarak renk geçişini tetikleyin.\nAktif Renk (RGB): %+v\nSon Olaylar - Tuş: %s | Fare: %s", btnCol, state.LastKey, state.LastMouse),
				Style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
				Wrap:  true,
			},
		}
		f.RenderWidget(colorBox, rightChunks[0])

		// Alt Bölüm: Bouncing Animasyonu
		bounceOffset := int(math.Round(state.BounceVal.Value()))
		bounceText := ""
		for i := 0; i < bounceOffset; i++ {
			bounceText += "\n"
		}
		bounceText += "⚽ ZIPLAYAN KUTU (Zıplatmak için 'b' tuşuna basın veya bu kutuya tıklayın)"

		bounceBox := widgets.Block{
			Title:          " ZIPLAMA EFEKTİ (EaseOutBounce) ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 0, 255)},
			PaddingLeft:    2,
			Child:          label{text: bounceText, style: cell.Style{Fg: cell.NewColorRGB(0, 255, 255)}},
		}
		f.RenderWidget(bounceBox, rightChunks[1])

		// Tıklama Olay Köprülerinin Kaydı
		
		// 1. Sol Panel (Sidebar) Tıklama: Sidebar'ı aç/kapat
		f.RegisterClickHandler(bodyChunks[0], func(ev backend.MouseEvent) {
			state.SidebarOpen = !state.SidebarOpen
			target := 6.0
			if state.SidebarOpen {
				target = 24.0
			}
			state.SidebarWidth.AnimateTo(target, 400*time.Millisecond, animation.EaseInOutCubic)
		})

		// 2. Renk Kutusu Tıklama: Rengi değiştir
		f.RegisterClickHandler(rightChunks[0], func(ev backend.MouseEvent) {
			state.ColorIndex = (state.ColorIndex + 1) % 4
			var targetColor cell.Color
			switch state.ColorIndex {
			case 0:
				targetColor = cell.NewColorRGB(0, 100, 255)
			case 1:
				targetColor = cell.NewColorRGB(255, 0, 100)
			case 2:
				targetColor = cell.NewColorRGB(0, 255, 100)
			case 3:
				targetColor = cell.NewColorRGB(255, 180, 0)
			}
			state.ButtonColor.AnimateTo(targetColor, 500*time.Millisecond, animation.EaseInOutQuad)
		})

		// 3. Zıplama Kutusu Tıklama: Topu zıplat
		f.RegisterClickHandler(rightChunks[1], func(ev backend.MouseEvent) {
			state.BounceVal.SetValue(0)
			state.BounceVal.AnimateTo(10, 1000*time.Millisecond, animation.EaseOutBounce)
		})

		// 3. Footer
		footerText := fmt.Sprintf(" [Boşluk] Menü Aç/Kapa | [Enter] Renk Değiştir | [b] Zıplat | [q] Çıkış | FPS: %.1f", state.FPS)
		footerBlock := widgets.Block{
			Borders: widgets.BorderNone,
			Style:   cell.Style{Fg: cell.NewColorRGB(140, 140, 140), Bg: cell.NewColorRGB(30, 30, 30)},
			Child:   label{text: footerText, style: cell.Style{Fg: cell.NewColorRGB(140, 140, 140), Bg: cell.NewColorRGB(30, 30, 30)}},
		}
		f.RenderWidget(footerBlock, chunks[2])
	})
}

type label struct {
	text  string
	style cell.Style
}

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
