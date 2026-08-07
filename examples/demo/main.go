package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
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

type ProcessInfo struct {
	PID    string
	Name   string
	CPU    string
	Memory string
	Status string
}

type MatrixStream struct {
	X     int
	Y     float64
	Speed float64
}

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

	// PulseVal, vektör grafiğindeki dairenin boyutunu animate eder.
	PulseVal *animation.Float
	// TabColors, menü butonlarının çerçeve renklerini anime eder.
	TabColors map[string]*animation.Color
	// Canvas, çizim hafızasını korumak ve her karede yeni bellek ayırmamak için önbelleklenmiş Canvas bileşeni.
	Canvas *widgets.Canvas

	// Çift resim geçişli performans demosu için alanlar
	TestImg1        image.Image
	TestImg2        image.Image
	ActiveImg       image.Image
	LastImageToggle time.Time
	UseImg2         bool

	// Ayarlar sekmesindeki interaktif form durumları
	UsernameInputState *widgets.TextInputState
	MouseModeChecked   bool
	ThemeSelected      string // "Koyu", "Açık", "Renkli"

	// Çıkış onay diyalog durumu
	ShowExitDialog bool
	ExitDialogAnim *animation.Float

	// Giriş sekmesindeki interaktif tablo durumu
	TableState *widgets.TableState
	Processes  []ProcessInfo

	// Açılır menü durumu
	NotificationMode string
	NotifPopupState  *widgets.PopupState

	// Oyun alanı (Playground) durumları
	PlaygroundDir    layout.Direction
	PlaygroundRatio  int
	PlaygroundBorder string

	// Dither geçiş durumları
	IsTransitioning     bool
	TransitionStartTime time.Time

	// Oyun alanı ek özellikleri (Matrix ve Sparkline)
	PlaygroundMode string
	MatrixStreams  []MatrixStream
	CPUHistory     []float64

	// Sürükleme ve Yardım Modali özellikleri
	ShowHelpDialog      bool
	IsDraggingModal     bool
	DragMouseStartX     int
	DragMouseStartY     int
	ModalOffsetX        int
	ModalOffsetY        int
	ModalDragBaseX      int
	ModalDragBaseY      int
}

// UpdateAnimations, zaman tabanlı animasyonları bir kare ileriye taşır.
func (state *AppState) UpdateAnimations(now time.Time) {
	// Daire daralma/genişleme pulse animasyonu
	if state.PulseVal != nil {
		if !state.PulseVal.IsAnimating() {
			if state.PulseVal.Value() == 0 {
				state.PulseVal.AnimateTo(1.0, 1500*time.Millisecond, animation.EaseInOutSine)
			} else {
				state.PulseVal.AnimateTo(0.0, 1500*time.Millisecond, animation.EaseInOutSine)
			}
		}
		state.PulseVal.Update(now)
	}

	// Matrix Rain Stream Animasyonu
	if state.PlaygroundMode == "Matrix" {
		if len(state.MatrixStreams) == 0 {
			state.MatrixStreams = make([]MatrixStream, 150)
			for i := range state.MatrixStreams {
				state.MatrixStreams[i] = MatrixStream{
					X:     i,
					Y:     float64(-10 - rand.Intn(40)),
					Speed: 0.5 + rand.Float64()*1.0,
				}
			}
		}

		for i := range state.MatrixStreams {
			state.MatrixStreams[i].Y += state.MatrixStreams[i].Speed
			if state.MatrixStreams[i].Y > 160 { // Sınırı aşanları sıfırla
				state.MatrixStreams[i].Y = float64(-10 - rand.Intn(40))
				state.MatrixStreams[i].Speed = 0.5 + rand.Float64()*1.0
			}
		}
	}

	// Sparkline CPU Geçmiş Verisi üretimi
	if len(state.CPUHistory) == 0 {
		state.CPUHistory = make([]float64, 120)
		for i := range state.CPUHistory {
			state.CPUHistory[i] = 10.0 + rand.Float64()*40.0
		}
	}
	copy(state.CPUHistory, state.CPUHistory[1:])
	lastVal := state.CPUHistory[len(state.CPUHistory)-2]
	newVal := lastVal + (rand.Float64()*12.0 - 6.0)
	if newVal < 10.0 {
		newVal = 10.0
	}
	if newVal > 100.0 {
		newVal = 100.0
	}
	state.CPUHistory[len(state.CPUHistory)-1] = newVal

	// Temaya göre vurgu rengini (accent color) belirle
	var accentColor cell.Color
	switch state.ThemeSelected {
	case "Koyu":
		accentColor = cell.NewColorRGB(0, 255, 0) // Yeşil
	case "Açık":
		accentColor = cell.NewColorRGB(0, 100, 255) // Mavi
	case "Renkli":
		accentColor = cell.NewColorRGB(255, 165, 0) // Turuncu
	}

	// Menü sekme butonları renk geçişleri
	if state.TabColors != nil {
		for name, anim := range state.TabColors {
			if state.ActiveTab == name {
				if anim.Value() != accentColor && !anim.IsAnimating() {
					anim.AnimateTo(accentColor, 250*time.Millisecond, animation.EaseInOutQuad)
				}
			} else {
				inactiveColor := cell.NewColorRGB(120, 120, 120)
				if anim.Value() != inactiveColor && !anim.IsAnimating() {
					anim.AnimateTo(inactiveColor, 250*time.Millisecond, animation.EaseInOutQuad)
				}
			}
			anim.Update(now)
		}
	}

	// Resim geçişi (2 saniyede bir resimleri değiştir)
	if now.Sub(state.LastImageToggle) >= 2*time.Second {
		state.UseImg2 = !state.UseImg2
		if state.UseImg2 {
			state.ActiveImg = state.TestImg2
		} else {
			state.ActiveImg = state.TestImg1
		}
		state.LastImageToggle = now
	}

	// Çıkış diyalog animasyonu güncellemesi
	if state.ExitDialogAnim != nil {
		state.ExitDialogAnim.Update(now)
	}
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
		PulseVal:          animation.NewFloat(0),
		TabColors: map[string]*animation.Color{
			"Giriş":      animation.NewColor(cell.NewColorRGB(0, 255, 0)),
			"Ayarlar":    animation.NewColor(cell.NewColorRGB(120, 120, 120)),
			"Grafik":     animation.NewColor(cell.NewColorRGB(120, 120, 120)),
			"Playground": animation.NewColor(cell.NewColorRGB(120, 120, 120)),
			"Çıkış":      animation.NewColor(cell.NewColorRGB(120, 120, 120)),
		},
		UsernameInputState: widgets.NewTextInputState(),
		ExitDialogAnim:     animation.NewFloat(0.0),
		NotificationMode:   "Normal Mod",
		NotifPopupState:    widgets.NewPopupState(),
		PlaygroundDir:      layout.Horizontal,
		PlaygroundRatio:    50,
		PlaygroundBorder:   "Rounded",
		PlaygroundMode:     "Vector",
		MouseModeChecked:   true,
		ThemeSelected:      "Koyu",
		LastImageToggle:    time.Now(),
		TableState:         widgets.NewTableState(),
		Processes: []ProcessInfo{
			{PID: "1284", Name: "limoni_demo", CPU: "1.2%", Memory: "14.2 MB", Status: "Çalışıyor"},
			{PID: "942", Name: "alacritty", CPU: "0.8%", Memory: "48.1 MB", Status: "Çalışıyor"},
			{PID: "1104", Name: "go_compiler", CPU: "0.0%", Memory: "105.4 MB", Status: "Beklemede"},
			{PID: "3201", Name: "chrome", CPU: "5.4%", Memory: "512.0 MB", Status: "Çalışıyor"},
			{PID: "4509", Name: "docker_daemon", CPU: "0.2%", Memory: "84.5 MB", Status: "Çalışıyor"},
			{PID: "8712", Name: "gnome_shell", CPU: "2.1%", Memory: "256.1 MB", Status: "Çalışıyor"},
			{PID: "6321", Name: "vscode", CPU: "0.5%", Memory: "340.2 MB", Status: "Çalışıyor"},
			{PID: "2204", Name: "spotify", CPU: "1.0%", Memory: "120.5 MB", Status: "Beklemede"},
			{PID: "5011", Name: "discord", CPU: "0.4%", Memory: "98.2 MB", Status: "Çalışıyor"},
			{PID: "7701", Name: "golangci_lint", CPU: "0.0%", Memory: "75.4 MB", Status: "Beklemede"},
			{PID: "1409", Name: "git_kraken", CPU: "0.3%", Memory: "150.1 MB", Status: "Çalışıyor"},
		},
	}
	state.UsernameInputState.SetValue("LimoniGelistirici")
	state.TableState.Select(0) // Tabloda ilk satırı seçili başlat

	// 1. Resmi oluştur (Merkez kırmızı, dışı mavi daire)
	imgW, imgH := 128, 128
	testImg1 := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	for dy := 0; dy < imgH; dy++ {
		for dx := 0; dx < imgW; dx++ {
			distX := float64(dx - imgW/2)
			distY := float64(dy - imgH/2)
			dist := math.Sqrt(distX*distX+distY*distY) / (float64(imgW) / 2)
			if dist > 1.0 {
				dist = 1.0
			}
			r := uint8((1.0 - dist) * 255)
			g := uint8(dist * 128)
			b := uint8(dist * 255)
			testImg1.Set(dx, dy, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	state.TestImg1 = testImg1
	state.ActiveImg = testImg1

	// 2. Resmi oluştur (Köşegen yeşil-mor geçiş gradyanı)
	testImg2 := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	for dy := 0; dy < imgH; dy++ {
		for dx := 0; dx < imgW; dx++ {
			factor := float64(dx+dy) / float64(imgW+imgH)
			r := uint8(factor * 255)
			g := uint8((1.0 - factor) * 255)
			b := uint8(factor * 128)
			testImg2.Set(dx, dy, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	state.TestImg2 = testImg2

	// 30 FPS zamanlayıcısı (~33ms)
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	frameCount := 0
	lastFpsCalc := time.Now()
	var fps float64

	// İlk kareyi (frame) çiz
	drawApp(t, b, state, fps)

	// Olay dinleme döngüsü (Event Loop)
	for {
		select {
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				focused := t.FocusManager().Focused()

				// Eğer Çıkış Onay Modali açıksa, klavye girdilerini sadece onun butonlarına yönlendir
				if state.ShowExitDialog {
					if ev.Key.Type == backend.KeyTab {
						if ev.Key.Shift {
							t.FocusManager().Prev()
						} else {
							t.FocusManager().Next()
						}
					} else if ev.Key.Type == backend.KeyArrowLeft {
						t.FocusManager().Prev()
					} else if ev.Key.Type == backend.KeyArrowRight {
						t.FocusManager().Next()
					}
					if ev.Key.Type == backend.KeySpace || ev.Key.Type == backend.KeyEnter {
						if focused == "exit_dialog_btn_0" {
							// Evet - Çıkış yap
							b.Close()
							fmt.Println("\nLimoni TUI uygulamasından çıkış yapıldı. Görüşmek üzere!")
							os.Exit(0)
						} else if focused == "exit_dialog_btn_1" {
							// Hayır - Kapat
							state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
							t.FocusManager().SetFocused("")
						}
					}
					if ev.Key.Type == backend.KeyEsc {
						state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
						t.FocusManager().SetFocused("")
					}
					state.LastKey = fmt.Sprintf("Çıkış Diyalog Tuşu: %d", ev.Key.Type)
					break // Diğer klavye olaylarını yut!
				}

				// Eğer Yardım Modali açıksa, sadece Esc ile kapat
				if state.ShowHelpDialog {
					if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == '?') {
						state.ShowHelpDialog = false
						state.IsDraggingModal = false
					}
					state.LastKey = "Yardım Paneli Kapatıldı"
					break
				}

				// Eğer Açılır Menü (Popup) açıksa, klavye girdilerini ona yönlendir
				if state.NotifPopupState.IsOpen {
					if ev.Key.Type == backend.KeyArrowDown {
						state.NotifPopupState.Next(4)
					} else if ev.Key.Type == backend.KeyArrowUp {
						state.NotifPopupState.Prev()
					} else if ev.Key.Type == backend.KeyEnter || ev.Key.Type == backend.KeySpace {
						idx := state.NotifPopupState.Selected
						if idx >= 0 && idx < 3 {
							switch idx {
							case 0:
								state.NotificationMode = "Sessiz Mod"
							case 1:
								state.NotificationMode = "Normal Mod"
							case 2:
								state.NotificationMode = "Tümünü Bildir"
							}
							state.NotifPopupState.Close()
						}
					} else if ev.Key.Type == backend.KeyEsc {
						state.NotifPopupState.Close()
					}
					state.LastKey = "Açılır Menü Klavye Navigasyonu"
					break
				}

				// Eğer Oyun Alanı (Playground) sekmesi aktifse, klavye girdilerini ona göre işle
				if state.ActiveTab == "Playground" {
					if ev.Key.Type == backend.KeyRune && ev.Key.Ch == '+' {
						state.PlaygroundRatio += 5
						if state.PlaygroundRatio > 90 {
							state.PlaygroundRatio = 90
						}
						state.LastKey = "Playground Oran Arttır (+)"
						break
					} else if ev.Key.Type == backend.KeyRune && ev.Key.Ch == '-' {
						state.PlaygroundRatio -= 5
						if state.PlaygroundRatio < 10 {
							state.PlaygroundRatio = 10
						}
						state.LastKey = "Playground Oran Azalt (-)"
						break
					} else if ev.Key.Type == backend.KeyArrowLeft || ev.Key.Type == backend.KeyArrowRight || ev.Key.Type == backend.KeyArrowUp || ev.Key.Type == backend.KeyArrowDown {
						if state.PlaygroundDir == layout.Horizontal {
							state.PlaygroundDir = layout.Vertical
						} else {
							state.PlaygroundDir = layout.Horizontal
						}
						state.LastKey = "Playground Yön Değiştir"
						break
					}
				}

				// Tab ve Shift+Tab tuşlarıyla form elemanları arası odak geçişi
				if ev.Key.Type == backend.KeyTab {
					if ev.Key.Shift {
						t.FocusManager().Prev()
					} else {
						t.FocusManager().Next()
					}
					state.LastKey = "Tab (Odak Değişimi)"
					break
				}

				// Eğer bir TextInput aktif odaklıysa, klavye girdilerini ona yönlendir
				if focused == "username_input" {
					if state.UsernameInputState.HandleKey(ev.Key) {
						// TextInput durumu güncellendi
					}
					// ESC tuşu basıldığında metin kutusu odağından çık
					if ev.Key.Type == backend.KeyEsc {
						t.FocusManager().SetFocused("")
					}
				} else if focused == "process_table" {
					if ev.Key.Type == backend.KeyArrowDown {
						state.TableState.Next(len(state.Processes))
						state.LastKey = "Tablo Aşağı (Ok Tuşu)"
					} else if ev.Key.Type == backend.KeyArrowUp {
						state.TableState.Prev()
						state.LastKey = "Tablo Yukarı (Ok Tuşu)"
					} else if ev.Key.Type == backend.KeyEsc {
						t.FocusManager().SetFocused("")
					}
				} else {
					// Genel klavye kontrolleri (Metin kutusu odaklı değilse)
					if ev.Key.Type == backend.KeyRune && ev.Key.Ch == '?' {
						state.ShowHelpDialog = true
						state.ModalOffsetX = 0
						state.ModalOffsetY = 0
						state.LastKey = "Yardım Paneli Açıldı"
						break
					}

					if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
						state.ShowExitDialog = true
						state.ModalOffsetX = 0
						state.ModalOffsetY = 0
						state.ExitDialogAnim.AnimateTo(1.0, 250*time.Millisecond, animation.EaseOutCubic)
						t.FocusManager().SetFocused("exit_dialog_btn_1") // Varsayılan odak güvenli olsun ("Hayır")
						state.LastKey = "Çıkış Onay Modali Açıldı"
						break
					}
				}

				// Checkbox, RadioButton veya Popup odaklıyken Space/Enter ile seçim yapılması
				if focused != "" && focused != "username_input" && (ev.Key.Type == backend.KeySpace || ev.Key.Type == backend.KeyEnter) {
					switch focused {
					case "mouse_mode_cb":
						state.MouseModeChecked = !state.MouseModeChecked
					case "theme_dark_rb":
						state.ThemeSelected = "Koyu"
					case "theme_light_rb":
						state.ThemeSelected = "Açık"
					case "theme_colored_rb":
						state.ThemeSelected = "Renkli"
					case "notif_popup":
						state.NotifPopupState.Toggle()
					}
				}

				state.LastKey = fmt.Sprintf("Kod: %d, Karakter: %q, Ctrl: %v", ev.Key.Type, string(ev.Key.Ch), ev.Key.Ctrl)

			case backend.EventMouse:
				// Sürükleme olaylarını denetle
				if ev.Mouse.Drag {
					if state.IsDraggingModal {
						dx := int(ev.Mouse.X) - state.DragMouseStartX
						dy := int(ev.Mouse.Y) - state.DragMouseStartY
						state.ModalOffsetX = state.ModalDragBaseX + dx
						state.ModalOffsetY = state.ModalDragBaseY + dy
					}
				} else if ev.Mouse.Button == backend.MouseRelease {
					state.IsDraggingModal = false
				}

				// Fare olayını otomatik tıklama yönlendiriciye ilet (RouteMouseEvent) - sadece sol tıklamaları yönlendir
				if ev.Mouse.Button == backend.MouseLeft && !ev.Mouse.Drag {
					t.RouteMouseEvent(ev.Mouse)
				} else {
					// Eşleşen bir buton yoksa veya tıklama dışı hareketse, son fare koordinat ve eylemini genel ekrana yazmak için kaydet
					state.LastMouse = fmt.Sprintf("Buton: %d, Pozisyon: (%d, %d), Sürükleme: %v", ev.Mouse.Button, ev.Mouse.X, ev.Mouse.Y, ev.Mouse.Drag)
				}

			case backend.EventResize:
				// Pencere boyutu değiştikçe ekran otomatik olarak bir sonraki tick'te yeniden çizilecek
			}

		case <-ticker.C:
			// Animasyonları güncelle
			now := time.Now()
			state.UpdateAnimations(now)

			// Dither geçiş ilerlemesini güncelle
			if state.IsTransitioning {
				elapsed := time.Since(state.TransitionStartTime)
				progress := float64(elapsed) / float64(250*time.Millisecond)
				if progress >= 1.0 {
					progress = 1.0
					state.IsTransitioning = false
					t.SetTransitionActive(false)
				}
				t.SetTransitionProgress(progress)
			}

			// Ekranı yeniden çiz
			drawApp(t, b, state, fps)

			// FPS hesaplama
			frameCount++
			if time.Since(lastFpsCalc) >= 1*time.Second {
				fps = float64(frameCount) / time.Since(lastFpsCalc).Seconds()
				frameCount = 0
				lastFpsCalc = time.Now()
			}
		}
	}
}

// drawApp, uygulamanın durumunu okur ve ekranın yerleşimini çizdirir.
func drawApp(t *terminal.Terminal, b *backend.Backend, state *AppState, fps float64) {
	t.Draw(func(f *terminal.Frame) {
		// Dinamik renk teması seçimi
		var mainColor, accentColor cell.Color
		switch state.ThemeSelected {
		case "Koyu":
			mainColor = cell.NewColorRGB(0, 255, 255)   // Cyan
			accentColor = cell.NewColorRGB(0, 255, 0)   // Yeşil
		case "Açık":
			mainColor = cell.NewColorRGB(0, 100, 255)  // Mavi
			accentColor = cell.NewColorRGB(200, 200, 200) // Açık Gri
		case "Renkli":
			mainColor = cell.NewColorRGB(255, 165, 0)  // Turuncu
			accentColor = cell.NewColorRGB(255, 0, 255) // Magenta / Mor
		}

		// Eğer çıkış veya yardım diyalogu açık olacaksa, en baştan modalı kaydet ki çizilen arka plan widget'ları olay alamasın!
		if state.ShowExitDialog {
			dialogW, dialogH := uint16(46), uint16(9)
			dialogArea := terminal.CenterRect(f.Buffer.Area, dialogW, dialogH)
			dialogArea.X = uint16(int(dialogArea.X) + state.ModalOffsetX)
			dialogArea.Y = uint16(int(dialogArea.Y) + state.ModalOffsetY)
			progress := state.ExitDialogAnim.Value()
			animatedArea := terminal.ScaleRect(dialogArea, progress)
			f.RegisterModal("exit_dialog", animatedArea, func() {
				state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
			})
		}
		if state.ShowHelpDialog {
			helpW, helpH := uint16(64), uint16(16)
			helpArea := terminal.CenterRect(f.Buffer.Area, helpW, helpH)
			helpArea.X = uint16(int(helpArea.X) + state.ModalOffsetX)
			helpArea.Y = uint16(int(helpArea.Y) + state.ModalOffsetY)
			f.RegisterModal("help_dialog", helpArea, func() {
				state.ShowHelpDialog = false
			})
		}

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
			BorderStyle:    cell.Style{Fg: mainColor},
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

		// Sol Panel (Menü Bölmesi) - Dikeyde 5 adet buton ve esnek boşluk içerir
		menuLay := layout.NewFlexLayout(
			layout.Vertical,
			1, // Butonlar arasında 1 satır boşluk
			layout.Fixed(3), // Giriş Butonu
			layout.Fixed(3), // Ayarlar Butonu
			layout.Fixed(3), // Grafik Butonu
			layout.Fixed(3), // Playground Butonu
			layout.Fixed(3), // Çıkış Butonu
			layout.Fill(),
		)
		menuChunks := menuLay.Split(bodyChunks[0])

		// drawButton, sol menüdeki tıklanabilir buton kutularını ve event callback'lerini çizer
		drawButton := func(area cell.Rect, title string, tabName string) {
			borderCol := state.TabColors[tabName].Value()
			titleStyle := cell.Style{Fg: borderCol}

			// Eğer buton aktif sekmeye aitse kalın yazı yap
			if state.ActiveTab == tabName {
				titleStyle.Modifier = cell.ModifierBold
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
				if tabName == "Çıkış" {
					state.ShowExitDialog = true
					state.ExitDialogAnim.AnimateTo(1.0, 250*time.Millisecond, animation.EaseOutCubic)
					t.FocusManager().SetFocused("exit_dialog_btn_1")
				} else {
					if state.ActiveTab != tabName {
						state.ActiveTab = tabName
						t.FocusManager().SetFocused("") // Sekme değiştirince önceki odağı sıfırla
						t.SetTransitionActive(true)
						t.SetTransitionProgress(0.0)
						state.TransitionStartTime = time.Now()
						state.IsTransitioning = true
					}
				}
			})
		}

		drawButton(menuChunks[0], "1. Giris", "Giriş")
		drawButton(menuChunks[1], "2. Ayarlar", "Ayarlar")
		drawButton(menuChunks[2], "3. Grafik", "Grafik")
		drawButton(menuChunks[3], "4. OyunAlani", "Playground")
		drawButton(menuChunks[4], "5. Cikis", "Çıkış")

		// Çıkış buton alanı koordinatını kaydet
		state.ExitButtonArea = menuChunks[4]

		// Sağ Panel (İçerik Paneli) Çizimi
		switch state.ActiveTab {
		case "Giriş":
			gisLay := layout.NewFlexLayout(
				layout.Vertical,
				1,
				layout.Fixed(4), // Açıklama bloğu
				layout.Fill(),   // Süreç Tablosu
			)
			gisChunks := gisLay.Split(bodyChunks[1])

			// 1. ÜST TARAF: Açıklama paragrafı
			descBlock := widgets.Block{
				Title:          " BİLGİLENDİRME ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accentColor},
				PaddingLeft:    2,
				PaddingRight:   2,
				Child: widgets.Paragraph{
					Text:  fmt.Sprintf("Fare ile sekmelere tıklayabilir veya klavyeden 'Tab' tuşuyla odak değiştirebilirsiniz. Tablo odaklandığında Ok Tuşları dikeyde gezinmenizi sağlar.\nSeçili Satır: %d | Son Basılan Tuş: %s", state.TableState.Selected+1, state.LastKey),
					Style: cell.Style{Fg: cell.NewColorRGB(200, 200, 200)},
					Wrap:  true,
				},
			}
			f.RenderWidget(descBlock, gisChunks[0])

			// 2. ALT TARAF: Sistem süreç tablosu
			tableRows := make([]widgets.TableRow, len(state.Processes))
			for i, p := range state.Processes {
				tableRows[i] = widgets.TableRow{
					Cells: []widgets.TableCell{
						{Text: p.PID},
						{Text: p.Name},
						{Text: p.CPU},
						{Text: p.Memory},
						{Text: p.Status, Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}},
					},
				}
				// Zebra desen (alternating background colors)
				if i%2 == 1 {
					tableRows[i].Style = cell.Style{Bg: cell.NewColorRGB(35, 35, 35)}
				}
			}

			// Tablo odaklandığında çerçeve rengi parlasın
			tableBorderCol := cell.NewColorRGB(100, 100, 100)
			if t.FocusManager().Focused() == "process_table" {
				tableBorderCol = accentColor
			}

			sysTable := widgets.Table{
				ID: "process_table",
				Header: &widgets.TableRow{
					Cells: []widgets.TableCell{
						{Text: "PID", Style: cell.Style{Modifier: cell.ModifierBold}},
						{Text: "SÜREÇ ADI", Style: cell.Style{Modifier: cell.ModifierBold}},
						{Text: "CPU", Style: cell.Style{Modifier: cell.ModifierBold}},
						{Text: "BELLEK", Style: cell.Style{Modifier: cell.ModifierBold}},
						{Text: "DURUM", Style: cell.Style{Modifier: cell.ModifierBold}},
					},
					Style: cell.Style{Bg: cell.NewColorRGB(45, 45, 45)},
				},
				Rows: tableRows,
				Constraints: []widgets.TableConstraint{
					{Type: widgets.ConstraintFixed, Value: 6},   // PID
					{Type: widgets.ConstraintPercentage, Value: 30}, // Name
					{Type: widgets.ConstraintFixed, Value: 8},   // CPU
					{Type: widgets.ConstraintFixed, Value: 12},  // Memory
					{Type: widgets.ConstraintFill},               // Status
				},
				State:     state.TableState,
				GridStyle: cell.Style{Fg: cell.NewColorRGB(70, 70, 70)},
				SelectedStyle: cell.Style{
					Fg:       cell.NewColorRGB(255, 255, 255),
					Bg:       accentColor,
					Modifier: cell.ModifierBold,
				},
				DrawGrid: true,
			}

			tableBlock := widgets.Block{
				Title:          " SİSTEM SÜREÇLERİ (PROCESS TABLE) ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: tableBorderCol},
				Child:          sysTable,
			}
			f.RenderWidget(tableBlock, gisChunks[1])

			if t.FocusManager().Focused() == "process_table" {
				widgets.DrawFocusRing(f.Buffer, gisChunks[1], cell.Style{Fg: accentColor})
			}

		case "Ayarlar":
			innerArea := cell.Rect{
				X:      bodyChunks[1].X + 1,
				Y:      bodyChunks[1].Y + 1,
				Width:  bodyChunks[1].Width - 2,
				Height: bodyChunks[1].Height - 2,
			}
			formLay := layout.NewFlexLayout(
				layout.Vertical,
				1,               // Elemanlar arasında 1 satır boşluk bırak
				layout.Fixed(1), // Kılavuz / Açıklama satırı
				layout.Fixed(3), // Kullanıcı adı kutusu (Bordered Block height 3)
				layout.Fixed(1), // Checkbox (Mouse modu)
				layout.Fixed(5), // Tema grubu kutusu (Bordered Block height 5)
				layout.Fixed(3), // Bildirim modu açılır kutusu (Bordered Block height 3)
			)
			formChunks := formLay.Split(innerArea)

			// 1. Dış Çerçeveyi Çiz
			settingsBlock := widgets.Block{
				Title:          " LİMONİ SİSTEM AYARLARI ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: mainColor},
			}
			f.RenderWidget(settingsBlock, bodyChunks[1])

			// 2. İç form elemanlarını çiz
			f.RenderWidget(label{text: "Tab / Shift+Tab veya Ok tuşları ile odaklanıp Space ile seçin.", style: cell.Style{Fg: cell.NewColorRGB(160, 160, 160), Modifier: cell.ModifierItalic}}, formChunks[0])
			
			// Kullanıcı adı kutusu
			inputBorderCol := cell.NewColorRGB(100, 100, 100)
			if t.FocusManager().Focused() == "username_input" {
				inputBorderCol = accentColor
			}

			usernameInput := widgets.TextInput{
				ID:           "username_input",
				State:        state.UsernameInputState,
				Placeholder:  "Kullanıcı adınızı girin...",
				Style:        cell.Style{Fg: cell.NewColorRGB(255, 255, 255)},
				FocusedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255)},
			}
			usernameBlock := widgets.Block{
				Title:          " KULLANICI ADI ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: inputBorderCol},
				Child:          usernameInput,
			}
			f.RenderWidget(usernameBlock, formChunks[1])

			if t.FocusManager().Focused() == "username_input" {
				widgets.DrawFocusRing(f.Buffer, formChunks[1], cell.Style{Fg: accentColor})
			}

			// Mouse Modu Checkbox
			mouseModeCb := widgets.Checkbox{
				ID:           "mouse_mode_cb",
				Checked:      &state.MouseModeChecked,
				Label:        "Mouse Modunu Aktif Et (SGR)",
				FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
			}
			f.RenderWidget(mouseModeCb, formChunks[2])

			// Tema Seçim Paneli (Bordered Block + Radio buttons)
			themeBorderCol := cell.NewColorRGB(100, 100, 100)
			focused := t.FocusManager().Focused()
			if focused == "theme_dark_rb" || focused == "theme_light_rb" || focused == "theme_colored_rb" {
				themeBorderCol = accentColor
			}

			themeInnerArea := cell.Rect{
				X:      formChunks[3].X + 2,
				Y:      formChunks[3].Y + 1,
				Width:  formChunks[3].Width - 4,
				Height: formChunks[3].Height - 2,
			}
			themeLay := layout.NewFlexLayout(
				layout.Vertical,
				0,
				layout.Fixed(1),
				layout.Fixed(1),
				layout.Fixed(1),
			)
			themeChunks := themeLay.Split(themeInnerArea)

			darkRb := widgets.RadioButton{
				ID:           "theme_dark_rb",
				Selected:     &state.ThemeSelected,
				Value:        "Koyu",
				Label:        "Koyu Tema (Cyan/Yeşil)",
				FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
			}
			lightRb := widgets.RadioButton{
				ID:           "theme_light_rb",
				Selected:     &state.ThemeSelected,
				Value:        "Açık",
				Label:        "Açık Tema (Mavi/Gri)",
				FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
			}
			coloredRb := widgets.RadioButton{
				ID:           "theme_colored_rb",
				Selected:     &state.ThemeSelected,
				Value:        "Renkli",
				Label:        "Renkli Tema (Turuncu/Mor)",
				FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
			}

			themeBlock := widgets.Block{
				Title:          " ARAYÜZ RENK TEMASI ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: themeBorderCol},
			}
			f.RenderWidget(themeBlock, formChunks[3])

			if t.FocusManager().Focused() == "theme_dark_rb" || t.FocusManager().Focused() == "theme_light_rb" || t.FocusManager().Focused() == "theme_colored_rb" {
				widgets.DrawFocusRing(f.Buffer, formChunks[3], cell.Style{Fg: accentColor})
			}

			f.RenderWidget(darkRb, themeChunks[0])
			f.RenderWidget(lightRb, themeChunks[1])
			f.RenderWidget(coloredRb, themeChunks[2])

			// 3. Bildirim Modu Açılır Menü (Popup) Çizimi
			notifBorderCol := cell.NewColorRGB(100, 100, 100)
			if t.FocusManager().Focused() == "notif_popup" {
				notifBorderCol = accentColor
			}

			popupArea := cell.Rect{
				X:      formChunks[4].X + 2,
				Y:      formChunks[4].Y + 1,
				Width:  formChunks[4].Width - 4,
				Height: 1,
			}

			notificationModePopup := widgets.Popup{
				ID:    "notif_popup",
				Label: state.NotificationMode,
				Items: []widgets.PopupItem{
					{Text: "Sessiz Mod", Handler: func() { state.NotificationMode = "Sessiz Mod" }},
					{Text: "Normal Mod", Handler: func() { state.NotificationMode = "Normal Mod" }},
					{Text: "Tümünü Bildir", Handler: func() { state.NotificationMode = "Tümünü Bildir" }},
					{Text: "Devre Dışı", Disabled: true, Handler: func() {}},
				},
				State:         state.NotifPopupState,
				Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 200), Bg: cell.NewColorRGB(50, 50, 50)},
				ItemStyle:     cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
				SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
				DisabledStyle: cell.Style{Fg: cell.NewColorRGB(100, 100, 100)},
				BorderStyle:   cell.Style{Fg: mainColor},
			}

			notifBlock := widgets.Block{
				Title:          " BİLDİRİM MODU (AÇILIR MENÜ) ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: notifBorderCol},
			}
			f.RenderWidget(notifBlock, formChunks[4])

			if t.FocusManager().Focused() == "notif_popup" {
				widgets.DrawFocusRing(f.Buffer, formChunks[4], cell.Style{Fg: accentColor})
			}

			f.RenderWidget(notificationModePopup, popupArea)

		case "Grafik":
			// Grafik sekmesini yatayda iki eşit bölüme ayır: Sol tarafta Canvas, Sağ tarafta Gerçek Resim (Image)
			grafikLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Percentage(50),
				layout.Percentage(50),
			)
			grafikChunks := grafikLay.Split(bodyChunks[1])

			// 1. SOL TARAF: Braille Vektör Canvas
			w := uint16(0)
			h := uint16(0)
			if grafikChunks[0].Width > 2 {
				w = grafikChunks[0].Width - 2
			}
			if grafikChunks[0].Height > 2 {
				h = grafikChunks[0].Height - 2
			}

			if state.Canvas == nil {
				state.Canvas = widgets.NewCanvas(w, h)
			} else {
				state.Canvas.Reset(w, h)
			}
			canvas := state.Canvas

			cyan := cell.Style{Fg: cell.NewColorRGB(0, 255, 255)}
			magenta := cell.Style{Fg: cell.NewColorRGB(255, 0, 255)}
			yellow := cell.Style{Fg: cell.NewColorRGB(255, 255, 0)}
			green := cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}

			virtualW := int(w) * 2
			virtualH := int(h) * 4

			if virtualW > 2 && virtualH > 2 {
				// Çerçeve (Rectangle)
				canvas.DrawRect(0, 0, virtualW, virtualH, yellow)

				// Merkez Daire (Circle)
				cx := virtualW / 2
				cy := virtualH / 2
				r := virtualH / 4
				if r > virtualW/4 {
					r = virtualW / 4
				}
				if r > 0 {
					// Daire boyutunu PulseVal animasyonuyla dinamik büyüt/küçült
					pulse := state.PulseVal.Value()
					r = int(float64(r) * (0.7 + 0.4*pulse))

					canvas.DrawCircle(cx, cy, r, cyan)
					// Daire içi artı (Cross)
					canvas.DrawLine(cx-r+2, cy, cx+r-2, cy, green)
					canvas.DrawLine(cx, cy-r+2, cx, cy+r-2, green)
				}

				// İkinci dereceden Bezier dalgası
				canvas.DrawBezierQuadratic(2, virtualH-4, virtualW/2, cy+4, virtualW-3, virtualH-4, 50, magenta)

				// Üçüncü dereceden Bezier dalgası
				canvas.DrawBezierCubic(2, 4, virtualW/3, virtualH/3, 2*virtualW/3, 2*virtualH/3, virtualW-3, 4, 50, yellow)
			}

			canvasBlock := widgets.Block{
				Title:          " BRAILLE VEKTÖR GRAFİĞİ ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 255)},
				Child:          canvas,
			}
			f.RenderWidget(canvasBlock, grafikChunks[0])

			// 2. SAĞ TARAF: Gerçek Görsel Gösterimi (Native Image) - 2 saniyede bir değişen resmi çiz
			imageBlock := widgets.Block{
				Title:          " GERÇEK RESİM GÖSTERİMİ ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 0, 255)},
				Child:          widgets.Image{Img: state.ActiveImg, ZIndex: -1, ForceHalfBlock: true},
			}
			f.RenderWidget(imageBlock, grafikChunks[1])

		case "Playground":
			// Playground sekmesini sol (ayarlar - 30 sütun) ve sağ (canlı önizleme - Fill) olarak ikiye böl
			playLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Fixed(30),
				layout.Fill(),
			)
			playChunks := playLay.Split(bodyChunks[1])

			// --- SOL TARAF: KONTROLLER ---
			ctrlBlock := widgets.Block{
				Title:          " OYUN ALANI KONTROLLERİ ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: mainColor},
			}
			f.RenderWidget(ctrlBlock, playChunks[0])

			// Kontrol alanındaki satırları böl
			ctrlInner := cell.Rect{
				X:      playChunks[0].X + 2,
				Y:      playChunks[0].Y + 1,
				Width:  playChunks[0].Width - 4,
				Height: playChunks[0].Height - 2,
			}
			ctrlRowLay := layout.NewFlexLayout(
				layout.Vertical,
				1,
				layout.Fixed(1), // Kılavuz / Bilgi satırı
				layout.Fixed(1), // Düzen Yönü başlığı
				layout.Fixed(1), // Düzen Yönü Horiz / Vert
				layout.Fixed(1), // Oran başlığı
				layout.Fixed(1), // Oran göstergesi / Bar
				layout.Fixed(1), // Kenarlık Başlığı
				layout.Fixed(1), // Kenarlık Seçenekleri
				layout.Fixed(1), // Mod Başlığı
				layout.Fixed(1), // Mod Seçenekleri
			)
			ctrlRows := ctrlRowLay.Split(ctrlInner)

			// 1. Bilgi Satırı
			f.RenderWidget(label{text: "Değişimi klavye/fareyle yapın", style: cell.Style{Fg: cell.NewColorRGB(140, 140, 140), Modifier: cell.ModifierItalic}}, ctrlRows[0])

			// 2-3. Düzen Yönü (Yatay / Dikey)
			f.RenderWidget(label{text: "Düzen Yönü (Yön Tuşları/Click):", style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Modifier: cell.ModifierBold}}, ctrlRows[1])

			dirText := " [▶ Yatay]  [  Dikey] "
			if state.PlaygroundDir == layout.Vertical {
				dirText = " [  Yatay]  [▶ Dikey] "
			}
			f.RenderWidget(label{text: dirText, style: cell.Style{Fg: accentColor}}, ctrlRows[2])

			// Tıklama alanları (Fare ile yön değiştirme)
			horizClickArea := cell.NewRect(ctrlRows[2].X + 1, ctrlRows[2].Y, 9, 1)
			f.RegisterClickHandler(horizClickArea, func(ev backend.MouseEvent) {
				state.PlaygroundDir = layout.Horizontal
			})
			vertClickArea := cell.NewRect(ctrlRows[2].X + 12, ctrlRows[2].Y, 9, 1)
			f.RegisterClickHandler(vertClickArea, func(ev backend.MouseEvent) {
				state.PlaygroundDir = layout.Vertical
			})

			// 4-5. Oran Kontrolü (+ / - Tuşları)
			f.RenderWidget(label{text: "Bölme Oranı (+ ve - tuşları):", style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Modifier: cell.ModifierBold}}, ctrlRows[3])

			// Oran barı / gauge çizimi
			barWidth := int(ctrlRows[4].Width) - 10
			if barWidth < 5 {
				barWidth = 10
			}
			filledWidth := int(float64(barWidth) * (float64(state.PlaygroundRatio) / 100.0))
			barStr := "["
			for i := 0; i < barWidth; i++ {
				if i < filledWidth {
					barStr += "█"
				} else {
					barStr += "░"
				}
			}
			barStr += fmt.Sprintf("] %d%%", state.PlaygroundRatio)
			f.RenderWidget(label{text: barStr, style: cell.Style{Fg: accentColor}}, ctrlRows[4])

			// Tıklamayla oran değiştirme
			f.RegisterClickHandler(ctrlRows[4], func(ev backend.MouseEvent) {
				clickX := int(ev.X) - int(ctrlRows[4].X) - 1
				if clickX >= 0 && clickX < barWidth {
					ratio := int(float64(clickX) / float64(barWidth) * 100.0)
					if ratio < 10 {
						ratio = 10
					}
					if ratio > 90 {
						ratio = 90
					}
					state.PlaygroundRatio = ratio
				}
			})

			// 6-7. Kenarlık Seçenekleri
			f.RenderWidget(label{text: "Kenarlık Stili:", style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Modifier: cell.ModifierBold}}, ctrlRows[5])

			borderText := " [▶ Oval]  [  Çift]  [  Kalın] "
			if state.PlaygroundBorder == "Double" {
				borderText = " [  Oval]  [▶ Çift]  [  Kalın] "
			} else if state.PlaygroundBorder == "Thick" {
				borderText = " [  Oval]  [  Çift]  [▶ Kalın] "
			}
			f.RenderWidget(label{text: borderText, style: cell.Style{Fg: accentColor}}, ctrlRows[6])

			// Tıklama alanları (Kenarlık değiştirme)
			ovalArea := cell.NewRect(ctrlRows[6].X + 1, ctrlRows[6].Y, 7, 1)
			f.RegisterClickHandler(ovalArea, func(ev backend.MouseEvent) {
				state.PlaygroundBorder = "Rounded"
			})
			doubleArea := cell.NewRect(ctrlRows[6].X + 10, ctrlRows[6].Y, 7, 1)
			f.RegisterClickHandler(doubleArea, func(ev backend.MouseEvent) {
				state.PlaygroundBorder = "Double"
			})
			thickArea := cell.NewRect(ctrlRows[6].X + 19, ctrlRows[6].Y, 8, 1)
			f.RegisterClickHandler(thickArea, func(ev backend.MouseEvent) {
				state.PlaygroundBorder = "Thick"
			})

			// 8-9. Mod Seçenekleri (Çember / Matrix / Grafik)
			f.RenderWidget(label{text: "Canvas Gösterim Modu:", style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Modifier: cell.ModifierBold}}, ctrlRows[7])

			modText := " [▶ Çember]  [  Matris]  [  Grafik] "
			if state.PlaygroundMode == "Matrix" {
				modText = " [  Çember]  [▶ Matris]  [  Grafik] "
			} else if state.PlaygroundMode == "Chart" {
				modText = " [  Çember]  [  Matris]  [▶ Grafik] "
			}
			f.RenderWidget(label{text: modText, style: cell.Style{Fg: accentColor}}, ctrlRows[8])

			// Tıklama alanları (Mod değiştirme)
			circleModeArea := cell.NewRect(ctrlRows[8].X + 1, ctrlRows[8].Y, 9, 1)
			f.RegisterClickHandler(circleModeArea, func(ev backend.MouseEvent) {
				state.PlaygroundMode = "Vector"
			})
			matrixModeArea := cell.NewRect(ctrlRows[8].X + 12, ctrlRows[8].Y, 9, 1)
			f.RegisterClickHandler(matrixModeArea, func(ev backend.MouseEvent) {
				state.PlaygroundMode = "Matrix"
			})
			chartModeArea := cell.NewRect(ctrlRows[8].X + 23, ctrlRows[8].Y, 9, 1)
			f.RegisterClickHandler(chartModeArea, func(ev backend.MouseEvent) {
				state.PlaygroundMode = "Chart"
			})

			// --- SAĞ TARAF: CANLI ÖNİZLEME (CSS GRID & MARKDOWN & MASK) ---
			previewBlock := widgets.Block{
				Title:          " CANLI IZGARA DÜZENİ VE BİLEŞENLER (CSS GRID & MARKDOWN) ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: mainColor},
			}
			f.RenderWidget(previewBlock, playChunks[1])

			previewInner := cell.Rect{
				X:      playChunks[1].X + 2,
				Y:      playChunks[1].Y + 1,
				Width:  playChunks[1].Width - 4,
				Height: playChunks[1].Height - 2,
			}

			// Sütunları oran bazında esnek (col 0: ratio fr, col 1: 100-ratio fr)
			// Satırları eşit esnek (row 0: 1fr, row 1: 1fr)
			// Gap: 1 karakter boşluk
			gridLayout := layout.NewGridLayout(
				[]layout.GridConstraint{layout.GridFraction(uint16(state.PlaygroundRatio)), layout.GridFraction(uint16(100 - state.PlaygroundRatio))},
				[]layout.GridConstraint{layout.GridFraction(1), layout.GridFraction(1)},
				1,
			)
			gridAreas := gridLayout.Split(previewInner)

			// Kenarlık sembollerini seç
			var sym widgets.BorderSymbols
			switch state.PlaygroundBorder {
			case "Rounded":
				sym = widgets.SymbolsRounded
			case "Double":
				sym = widgets.SymbolsDouble
			case "Thick":
				sym = widgets.SymbolsThick
			}

			// 1. Hücre (0,0): Markdown Çizimi
			mdBlock := widgets.Block{
				Title:          " 📝 BELDELER (MARKDOWN) ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  sym,
				BorderStyle:    cell.Style{Fg: accentColor},
				Child: widgets.Markdown{
					Content: "# Limoni TUI\nRatatui'den *daha esnek* ve **performanslı**.\n- CSS Grid yerleşimi.\n- Bayer dither geçişleri.\n- Dairesel `avatar` maskeleme.",
					Style:   cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
				},
			}
			f.RenderWidget(mdBlock, gridAreas.Cell(0, 0).Area)

			// 2. Hücre (0,1): Dairesel Maskelenmiş Resim
			imgBlock := widgets.Block{
				Title:          " 👤 PROFİL (MASK) ",
				TitleAlignment: widgets.AlignCenter,
				Borders:        widgets.BorderAll,
				BorderSymbols:  sym,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 165, 0)},
				Child:          widgets.Image{Img: state.ActiveImg, CircleMask: true, ForceHalfBlock: true},
			}
			f.RenderWidget(imgBlock, gridAreas.Cell(0, 1).Area)

			// 3. Alt Satır (1,0) span 1 row, 2 cols: Braille Canvas veya Sparkline
			canvasArea := gridAreas.Cell(1, 0).Span(1, 2)

			canvasW := uint16(0)
			canvasH := uint16(0)
			if canvasArea.Width > 2 {
				canvasW = canvasArea.Width - 2
			}
			if canvasArea.Height > 2 {
				canvasH = canvasArea.Height - 2
			}

			if state.Canvas == nil {
				state.Canvas = widgets.NewCanvas(canvasW, canvasH)
			} else {
				state.Canvas.Reset(canvasW, canvasH)
			}

			canvas := state.Canvas
			virtualW := int(canvasW) * 2
			virtualH := int(canvasH) * 4

			var childWidget widgets.Widget

			if state.PlaygroundMode == "Chart" {
				childWidget = widgets.Sparkline{
					Data:  state.CPUHistory,
					Style: cell.Style{},
					Color: accentColor,
				}
			} else {
				yellowStyle := cell.Style{Fg: cell.NewColorRGB(255, 255, 0)}
				cyanStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 255)}
				greenStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}

				if state.PlaygroundMode == "Vector" {
					if virtualW > 2 && virtualH > 2 {
						canvas.DrawRect(0, 0, virtualW, virtualH, yellowStyle)
						cx := virtualW / 2
						cy := virtualH / 2
						r := virtualH / 4
						if r > virtualW/4 {
							r = virtualW / 4
						}
						if r > 0 {
							pulse := state.PulseVal.Value()
							r = int(float64(r) * (0.7 + 0.4*pulse))
							canvas.DrawCircle(cx, cy, r, cyanStyle)
							canvas.DrawLine(cx-r+2, cy, cx+r-2, cy, greenStyle)
							canvas.DrawLine(cx, cy-r+2, cx, cy+r-2, greenStyle)
						}
					}
				} else if state.PlaygroundMode == "Matrix" {
					// Matrix parçacık yağmuru dikey akışı
					for _, stream := range state.MatrixStreams {
						if stream.X >= virtualW {
							continue
						}
						headY := int(stream.Y)
						for k := 0; k < 12; k++ {
							yIdx := headY - k
							if yIdx < 0 || yIdx >= virtualH {
								continue
							}
							intensity := 255 - (k * 20)
							if intensity < 30 {
								intensity = 30
							}
							var col cell.Style
							if k == 0 {
								col = cell.Style{Fg: cell.NewColorRGB(255, 255, 255)}
							} else {
								col = cell.Style{Fg: cell.NewColorRGB(0, uint8(intensity), 0)}
							}
							canvas.Set(stream.X, yIdx, col)
						}
					}
				}
				childWidget = canvas
			}

			canvasBlock := widgets.Block{
				Title:          " 🌀 CANLI GÖSTERİM ALANI (CANVAS / SPARKLINE) ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  sym,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 255)},
				Child:          childWidget,
			}
			f.RenderWidget(canvasBlock, canvasArea)
		}

		// 4. Footer (Alt Bilgi Satırı) Çizimi
		footerStyle := cell.Style{Fg: cell.NewColorRGB(140, 140, 140), Bg: cell.NewColorRGB(30, 30, 30)}
		footerText := fmt.Sprintf(" Boyut: %d x %d | FPS: %.1f | Kısayollar: ? | Sekmeler: Tab / Shift+Tab | Çıkış: 'q' / ESC", f.Buffer.Area.Width, f.Buffer.Area.Height, fps)

		footerBlock := widgets.Block{
			Borders: widgets.BorderNone,
			Style:   footerStyle,
			Child:   label{text: footerText, style: footerStyle},
		}
		f.RenderWidget(footerBlock, chunks[2])

		// 5. ÇIKIŞ ONAY MODAL DIALOG ÇİZİMİ
		if state.ShowExitDialog {
			dialogW, dialogH := uint16(46), uint16(9)
			dialogArea := terminal.CenterRect(f.Buffer.Area, dialogW, dialogH)

			// Sürükleme offsetlerini uygula
			dialogArea.X = uint16(int(dialogArea.X) + state.ModalOffsetX)
			dialogArea.Y = uint16(int(dialogArea.Y) + state.ModalOffsetY)

			progress := state.ExitDialogAnim.Value()
			animatedArea := terminal.ScaleRect(dialogArea, progress)

			// Animasyonun bitip bitmediğini denetle
			if progress <= 0.001 && !state.ExitDialogAnim.IsAnimating() {
				state.ShowExitDialog = false
				t.FocusManager().SetFocused("")
				return
			}

			// Başlık çubuğu sürükleme tıklama alanını tanımla
			titleBarArea := cell.NewRect(dialogArea.X, dialogArea.Y, dialogW, 1)
			f.RegisterClickHandler(titleBarArea, func(ev backend.MouseEvent) {
				state.IsDraggingModal = true
				state.DragMouseStartX = int(ev.X)
				state.DragMouseStartY = int(ev.Y)
				state.ModalDragBaseX = state.ModalOffsetX
				state.ModalDragBaseY = state.ModalOffsetY
			})

			exitDialog := widgets.Dialog{
				ID:          "exit_dialog",
				Title:       " 🗘 ÇIKIŞ ONAYI (Fareyle Sürükleyin) ",
				Message:     "Uygulamadan çıkmak istediğinize emin misiniz?",
				Style:       cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(30, 30, 30)},
				BorderStyle: cell.Style{Fg: mainColor},
				ButtonStyle: cell.Style{Fg: cell.NewColorRGB(200, 200, 200), Bg: cell.NewColorRGB(50, 50, 50)},
				ButtonFocusedStyle: cell.Style{
					Fg:       cell.NewColorRGB(255, 255, 255),
					Bg:       accentColor,
					Modifier: cell.ModifierBold,
				},
				Buttons: []widgets.DialogButton{
					{
						Text: "✔ Evet",
						Handler: func() {
							b.Close()
							fmt.Println("\nLimoni TUI uygulamasından çıkış yapıldı. Görüşmek üzere!")
							os.Exit(0)
						},
					},
					{
						Text: "✘ Hayır",
						Handler: func() {
							state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
						},
					},
				},
			}

			f.RenderWidget(exitDialog, animatedArea)
		}

		// 6. KISAYOL YARDIM MODAL DIALOG ÇİZİMİ
		if state.ShowHelpDialog {
			helpW, helpH := uint16(64), uint16(16)
			helpArea := terminal.CenterRect(f.Buffer.Area, helpW, helpH)

			// Sürükleme offsetlerini uygula
			helpArea.X = uint16(int(helpArea.X) + state.ModalOffsetX)
			helpArea.Y = uint16(int(helpArea.Y) + state.ModalOffsetY)

			// Başlık çubuğu sürükleme tıklama alanını tanımla
			titleBarArea := cell.NewRect(helpArea.X, helpArea.Y, helpW, 1)
			f.RegisterClickHandler(titleBarArea, func(ev backend.MouseEvent) {
				state.IsDraggingModal = true
				state.DragMouseStartX = int(ev.X)
				state.DragMouseStartY = int(ev.Y)
				state.ModalDragBaseX = state.ModalOffsetX
				state.ModalDragBaseY = state.ModalOffsetY
			})

			// Diyalog kutusu arka plan bloğu
			helpBlock := widgets.Block{
				Title:          " ⌨ KISAYOL YARDIMI (Fareyle Sürükleyin) ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accentColor},
				Style:          cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(25, 25, 25)},
			}
			f.RenderWidget(helpBlock, helpArea)

			helpInner := cell.Rect{
				X:      helpArea.X + 2,
				Y:      helpArea.Y + 1,
				Width:  helpArea.Width - 4,
				Height: helpArea.Height - 2,
			}

			// İçeriği yatay olarak iki kolona böl (Sol: Markdown, Sağ: Avatar Profil)
			helpLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Ratio(65),
				layout.Ratio(35),
			)
			helpChunks := helpLay.Split(helpInner)

			// Sol Taraf: Markdown Metni
			mdHelp := widgets.Markdown{
				Content: `# Limoni Kısayolları
- **Tab / Shift+Tab:** Menüler arası geçiş.
- **Yön Tuşları:** Playground Düzen Yönü.
- **+ / - Tuşları:** Playground Oran kontrolü.
- **? :** Yardım Paneli aç / kapat.
- **q / Esc:** Çıkış onay diyaloğu.`,
				Style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
			}
			f.RenderWidget(mdHelp, helpChunks[0])

			// Sağ Taraf: Profil resmi
			avatarBlock := widgets.Block{
				Borders: widgets.BorderNone,
				Child:   widgets.Image{Img: state.ActiveImg, CircleMask: true, ForceHalfBlock: true},
			}
			f.RenderWidget(avatarBlock, helpChunks[1])
		}
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
