package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
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
func clampDialogOffset(screen cell.Rect, width, height uint16, offsetX, offsetY int) (int, int) {
	centered := terminal.CenterRect(screen, width, height)
	minX := -int(centered.X)
	maxX := int(screen.Width) - int(centered.X) - int(width)
	minY := -int(centered.Y)
	maxY := int(screen.Height) - int(centered.Y) - int(height)
	if maxX < minX {
		maxX = minX
	}
	if maxY < minY {
		maxY = minY
	}
	if offsetX < minX {
		offsetX = minX
	}
	if offsetX > maxX {
		offsetX = maxX
	}
	if offsetY < minY {
		offsetY = minY
	}
	if offsetY > maxY {
		offsetY = maxY
	}
	return offsetX, offsetY
}

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
	ThemeSelected      string // "Koyu", "Açık", "Renkli", "Yüksek Kontrast"

	// Çıkış onay diyalog durumu
	ShowExitDialog     bool
	ExitDialogFinished bool
	ExitDialogAnim     *animation.Float

	// Giriş sekmesindeki interaktif tablo durumu
	TableState          *widgets.TableState
	TableFilterState    *widgets.TextInputState
	DemoSliderState     *widgets.SliderState
	PlayDirectionState  *widgets.SelectState
	PlayModeState       *widgets.SelectState
	PlayBorderState     *widgets.SelectState
	PlayRatioState      *widgets.SliderState
	AvatarOpacityState  *widgets.SliderState
	ShowcaseSelected    string
	ShowcaseSelectState *widgets.SelectState
	DemoMarkdown        string
	MarkdownOffset      int
	MarkdownHeight      int
	Processes           []ProcessInfo
	ProcessSamples      map[string]processSample
	LastProcessRead     time.Time
	FormProgress        *animation.Float

	// Açılır menü durumu
	NotificationMode string
	NotifPopupState  *widgets.PopupState

	// Oyun alanı (Playground) durumları
	PlaygroundDir    layout.Direction
	PlaygroundRatio  int
	PlaygroundBorder string
	PlayShowGrid     bool
	ProfileFrame     string

	// Dither geçiş durumları
	IsTransitioning     bool
	TransitionStartTime time.Time

	// Oyun alanı ek özellikleri (Matrix ve Sparkline)
	PlaygroundMode   string
	VirtualListState *widgets.ListState
	MatrixStreams    []MatrixStream
	CPUHistory       []float64

	// Sürükleme ve Yardım Modali özellikleri
	ShowHelpDialog  bool
	HelpDialogAnim  *animation.Float
	IsDraggingModal bool
	DragMouseStartX int
	DragMouseStartY int
	ModalOffsetX    int
	ModalOffsetY    int
	ModalDragBaseX  int
	ModalDragBaseY  int

	// Hata ayıklama modu
	DebugMode bool

	// 3D Grafik Motoru özellikleri
	RotX         float64
	RotY         float64
	RotZ         float64
	IsDragging3D bool
	Drag3DLastX  int
	Drag3DLastY  int
	AppleImg     image.Image
	ProfileImg   image.Image
	OBJModel     *graphics.Model3D
	OBJPath      string
	ThreeDModel  string // "Küp", "Piramit", "Dörtyüzlü", "OBJ"
	ThreeDStyle  string // "Dokulu", "Dolu Renkli", "Kafes"

	// Pencere boyutlandırma (Resizing) özellikleri
	IsResizingModal  bool
	ModalResizeBaseW int
	ModalResizeBaseH int
	HelpDialogW      int
	HelpDialogH      int

	// Komut Paleti ve Kısayol Yöneticisi
	CmdPalette *widgets.CommandPaletteState
	KeyManager *widgets.KeybindingManager

	// Referans sekmesi etkileşim sayaçları
	ReferenceRuntimeMessages      int
	ReferenceInteractionLast      string
	ReferenceLayoutPass           int
	ReferenceSelectedRow          int
	ReferenceBenchmarkRuns        int
	ReferenceDataOffset           int
	ReferenceDataState            *widgets.VirtualDataState
	ReferenceInteractionHover     string
	ReferenceInteractionEvents    int
	ReferenceInteractionLastRoute string
	ReferenceInteractionPointerX  uint16
	ReferenceInteractionPointerY  uint16
	ReferenceInteractionHistory   []string
	ReferenceLayoutLastAction     string
	ReferenceLayoutAllocated      cell.Rect
	ReferenceAccessibilityASCII   bool
	ScreenReaderMode              bool
	LastScreenReaderTree          string
}

func recordReferenceInteraction(state *AppState, event string) {
	state.ReferenceInteractionEvents++
	state.ReferenceInteractionLast = event
	state.ReferenceInteractionHistory = append(state.ReferenceInteractionHistory, event)
	if len(state.ReferenceInteractionHistory) > 4 {
		state.ReferenceInteractionHistory = state.ReferenceInteractionHistory[len(state.ReferenceInteractionHistory)-4:]
	}
}

// UpdateAnimations, zaman tabanlı animasyonları bir kare ileriye taşır.
func (state *AppState) UpdateAnimations(now time.Time) {
	// Giriş sekmesindeki progress bar 0 -> 100 -> 0 döngüsü
	if state.FormProgress != nil {
		if !state.FormProgress.IsAnimating() {
			if state.FormProgress.Value() >= 99.9 {
				state.FormProgress.AnimateTo(0, 4*time.Second, animation.EaseInOutSine)
			} else {
				state.FormProgress.AnimateTo(100, 4*time.Second, animation.EaseInOutSine)
			}
		}
		state.FormProgress.Update(now)
	}

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

	// Matrix/Particle Rain Stream Animasyonu
	if state.PlaygroundMode == "Matrix" || state.PlaygroundMode == "Particle" {
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
	case "Yüksek Kontrast":
		accentColor = cell.NewColorRGB(255, 255, 0)
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

	// Yardım diyalog animasyonu güncellemesi
	if state.HelpDialogAnim != nil {
		state.HelpDialogAnim.Update(now)
	}

	// 3D otomatik rotasyon güncellemesi
	if !state.IsDragging3D {
		state.RotX = math.Mod(state.RotX+1.0, 360.0)
		state.RotY = math.Mod(state.RotY+1.5, 360.0)
		state.RotZ = math.Mod(state.RotZ+0.5, 360.0)
	}
}

func main() {
	screenReaderMode := slices.Contains(os.Args[1:], "--screen-reader")
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
		ScreenReaderMode:  screenReaderMode,
		LastKey:           "Yok",
		LastMouse:         "Yok",
		SettingsListState: widgets.NewListState(),
		PulseVal:          animation.NewFloat(0),
		FormProgress:      animation.NewFloat(0),
		TabColors: map[string]*animation.Color{
			"Giriş":      animation.NewColor(cell.NewColorRGB(0, 255, 0)),
			"Ayarlar":    animation.NewColor(cell.NewColorRGB(120, 120, 120)),
			"Grafik":     animation.NewColor(cell.NewColorRGB(120, 120, 120)),
			"Playground": animation.NewColor(cell.NewColorRGB(120, 120, 120)),
			"Referans":   animation.NewColor(cell.NewColorRGB(120, 120, 120)),
			"Çıkış":      animation.NewColor(cell.NewColorRGB(120, 120, 120)),
		},
		UsernameInputState:  widgets.NewTextInputState(),
		ExitDialogAnim:      animation.NewFloat(0.0),
		HelpDialogAnim:      animation.NewFloat(0.0),
		NotificationMode:    "Normal Mod",
		NotifPopupState:     widgets.NewPopupState(),
		PlaygroundDir:       layout.Horizontal,
		PlaygroundRatio:     50,
		PlaygroundBorder:    "Rounded",
		PlaygroundMode:      "Vector",
		VirtualListState:    widgets.NewListState(),
		MouseModeChecked:    true,
		ThemeSelected:       "Koyu",
		ProfileFrame:        "Rounded",
		DebugMode:           false,
		RotX:                30.0,
		RotY:                45.0,
		RotZ:                0.0,
		IsResizingModal:     false,
		HelpDialogW:         64,
		HelpDialogH:         16,
		LastImageToggle:     time.Now(),
		TableState:          widgets.NewTableState(),
		TableFilterState:    widgets.NewTextInputState(),
		DemoSliderState:     widgets.NewSliderState(50),
		PlayDirectionState:  widgets.NewSelectState(),
		PlayModeState:       widgets.NewSelectState(),
		PlayBorderState:     widgets.NewSelectState(),
		PlayRatioState:      widgets.NewSliderState(50),
		AvatarOpacityState:  widgets.NewSliderState(100),
		ShowcaseSelected:    "Paragraph",
		ShowcaseSelectState: widgets.NewSelectState(),
		MarkdownHeight:      6,
		ProcessSamples:      make(map[string]processSample),
	}
	state.UsernameInputState.SetValue("LimoniGelistirici")
	state.DemoMarkdown = loadDemoMarkdown()
	state.Processes, state.ProcessSamples = readLiveProcesses(state.ProcessSamples, time.Now())
	state.TableState.Select(0) // Tabloda ilk satırı seçili başlat
	state.ReferenceDataState = widgets.NewVirtualDataState()

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

	// 3. apple.png dokusunu dosyadan yükle (hem root hem de examples/demo dizininden çalıştırılabilmesi için fallback'li)
	appleFile, err := os.Open("examples/demo/apple.png")
	if err != nil {
		appleFile, err = os.Open("apple.png")
	}
	if err == nil {
		defer appleFile.Close()
		appleImg, _, err := image.Decode(appleFile)
		if err == nil {
			state.AppleImg = appleImg
		} else {
			fmt.Fprintf(os.Stderr, "Limoni Doku Cozumleme Hatasi: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Limoni Doku Dosyasi Acilamadi (Cift Yol Denendi): %v\n", err)
	}

	state.ProfileImg = loadProfileImage()
	state.ThreeDModel = "Küp"
	state.ThreeDStyle = "Dokulu"

	// İsteğe bağlı 3D modeli: LIMONI_MODEL=/path/model.stl go run ./examples/demo
	modelPath := os.Getenv("LIMONI_MODEL")
	if modelPath == "" {
		modelPath = os.Getenv("LIMONI_OBJ")
	} // geriye dönük uyumluluk
	if modelPath != "" {
		var model graphics.Model3D
		var modelErr error
		if strings.HasSuffix(strings.ToLower(modelPath), ".stl") {
			model, modelErr = graphics.LoadSTL(modelPath)
		} else if strings.HasSuffix(strings.ToLower(modelPath), ".ply") {
			model, modelErr = graphics.LoadPLY(modelPath)
		} else {
			model, modelErr = graphics.LoadOBJ(modelPath)
		}
		if modelErr != nil {
			fmt.Fprintf(os.Stderr, "3D modeli yüklenemedi: %v\n", modelErr)
		} else {
			model.Normalize(2.4)
			state.OBJModel = &model
			state.OBJPath = modelPath
			state.ThreeDModel = "OBJ"
		}
	}

	// Komut Paleti ve Kısayol Yöneticisi başlat
	state.CmdPalette = widgets.NewCommandPaletteState()
	state.KeyManager = widgets.NewKeybindingManager()

	// Navigasyon kısayolları kaydet
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: 'p', Ctrl: true,
		Label: "Komut Paletini Aç/Kapa", Category: "Genel",
		Handler: func() {
			state.CmdPalette.Toggle()
			if state.CmdPalette.IsOpen {
				t.FocusManager().SetFocused("command_palette")
			} else {
				t.FocusManager().SetFocused("")
			}
		},
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: 'd', Ctrl: true,
		Label: "Hata Ayıklama Modunu Aç/Kapa", Category: "Görünüm",
		Handler: func() { state.DebugMode = !state.DebugMode },
	})
	canHandleGlobalCommand := func() bool {
		focused := t.FocusManager().Focused()
		return !state.ShowExitDialog && !state.ShowHelpDialog && !state.NotifPopupState.IsOpen &&
			focused != "username_input" && focused != "showcase_input" && focused != "table_filter"
	}
	openHelp := func() {
		state.ShowHelpDialog = true
		state.ModalOffsetX = 0
		state.ModalOffsetY = 0
		state.HelpDialogW = 66
		state.HelpDialogH = 12
		state.HelpDialogAnim.AnimateTo(1.0, 250*time.Millisecond, animation.EaseOutCubic)
		state.LastKey = "Yardım Paneli Açıldı"
	}
	openExitConfirmation := func() {
		state.ShowExitDialog = true
		state.ModalOffsetX = 0
		state.ModalOffsetY = 0
		state.ExitDialogAnim.AnimateTo(1.0, 250*time.Millisecond, animation.EaseOutCubic)
		t.FocusManager().SetFocused("exit_dialog_btn_1")
		state.LastKey = "Çıkış Onay Modali Açıldı"
		t.ForceFullRedraw()
	}
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: '?', Label: "Yardım Panelini Aç", Category: "Görünüm",
		When: canHandleGlobalCommand, Handler: openHelp,
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: 'q', Label: "Çıkış Onayını Aç", Category: "Genel",
		When: canHandleGlobalCommand, Handler: openExitConfirmation,
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyEsc, Label: "Çıkış Onayını Aç", Category: "Genel",
		When: canHandleGlobalCommand, Handler: openExitConfirmation,
	})
	closeExitDialog := func() {
		state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
		t.FocusManager().SetFocused("")
		t.ForceFullRedraw()
	}
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyEsc, Scope: "exit_dialog", Label: "Çıkış Diyaloğunu Kapat", Category: "Modal",
		Handler: closeExitDialog,
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyEsc, Scope: "help_dialog", Label: "Yardım Panelini Kapat", Category: "Modal",
		Handler: func() {
			state.ShowHelpDialog = false
			t.FocusManager().SetFocused("")
		},
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyEsc, Label: "Açılır Menüyü Kapat", Category: "Modal",
		When:    func() bool { return state.NotifPopupState.IsOpen },
		Handler: func() { state.NotifPopupState.Close() },
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyEsc, Label: "Aktif Kontrolden Çık", Category: "Navigasyon",
		When: func() bool {
			switch t.FocusManager().Focused() {
			case "username_input", "showcase_input", "demo_markdown", "table_filter":
				return true
			default:
				return false
			}
		},
		Handler: func() { t.FocusManager().SetFocused("") },
	})
	registerGraphicKey := func(ch rune, label string, handler func()) {
		state.KeyManager.Register(widgets.Keybinding{
			Key: backend.KeyRune, Ch: ch, Label: label, Category: "3D Grafik",
			When: func() bool { return state.ActiveTab == "Grafik" }, Handler: handler,
		})
	}
	registerGraphicKey('1', "3D Model: Küp", func() { state.ThreeDModel = "Küp" })
	registerGraphicKey('2', "3D Model: Piramit", func() { state.ThreeDModel = "Piramit" })
	registerGraphicKey('3', "3D Model: Dörtyüzlü", func() { state.ThreeDModel = "Dörtyüzlü" })
	registerGraphicKey('4', "Render Stili: Dokulu", func() { state.ThreeDStyle = "Dokulu" })
	registerGraphicKey('5', "Render Stili: Dolu Renkli", func() { state.ThreeDStyle = "Dolu Renkli" })
	registerGraphicKey('6', "Render Stili: Kafes", func() { state.ThreeDStyle = "Kafes" })
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: '+', Scope: "playground",
		Label: "Playground Oranını Artır", Category: "Playground",
		When: func() bool {
			return state.ActiveTab == "Playground" && !state.ShowExitDialog && !state.ShowHelpDialog && !state.NotifPopupState.IsOpen
		},
		Handler: func() {
			state.PlaygroundRatio += 5
			if state.PlaygroundRatio > 90 {
				state.PlaygroundRatio = 90
			}
			state.PlayRatioState.Set(state.PlaygroundRatio, 10, 90)
		},
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: '-', Scope: "playground",
		Label: "Playground Oranını Azalt", Category: "Playground",
		When: func() bool {
			return state.ActiveTab == "Playground" && !state.ShowExitDialog && !state.ShowHelpDialog && !state.NotifPopupState.IsOpen
		},
		Handler: func() {
			state.PlaygroundRatio -= 5
			if state.PlaygroundRatio < 10 {
				state.PlaygroundRatio = 10
			}
			state.PlayRatioState.Set(state.PlaygroundRatio, 10, 90)
		},
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: 'j', Scope: "playground_virtual_list",
		Label: "Sanal Listede Aşağı Git", Category: "Playground",
		When: func() bool {
			return state.PlaygroundMode == "VirtualList" && !state.ShowExitDialog && !state.ShowHelpDialog && !state.NotifPopupState.IsOpen
		},
		Handler: func() { moveVirtualListSelection(state, 1) },
	})
	state.KeyManager.Register(widgets.Keybinding{
		Key: backend.KeyRune, Ch: 'k', Scope: "playground_virtual_list",
		Label: "Sanal Listede Yukarı Git", Category: "Playground",
		When: func() bool {
			return state.PlaygroundMode == "VirtualList" && !state.ShowExitDialog && !state.ShowHelpDialog && !state.NotifPopupState.IsOpen
		},
		Handler: func() { moveVirtualListSelection(state, -1) },
	})

	// Sekme navigasyon komutlarını Command Palette'e kaydet
	cmdItems := []widgets.CommandItem{
		{Label: "Giriş Sekmesine Git", Detail: "", Category: "Navigasyon",
			Handler: func() {
				if state.ActiveTab != "Giriş" {
					state.ActiveTab = "Giriş"
					t.FocusManager().SetFocused("")
					state.TransitionStartTime = time.Now()
					state.IsTransitioning = true
				}
			}},
		{Label: "Ayarlar Sekmesine Git", Detail: "", Category: "Navigasyon",
			Handler: func() {
				if state.ActiveTab != "Ayarlar" {
					state.ActiveTab = "Ayarlar"
					t.FocusManager().SetFocused("")
					state.TransitionStartTime = time.Now()
					state.IsTransitioning = true
				}
			}},
		{Label: "Grafik Sekmesine Git", Detail: "", Category: "Navigasyon",
			Handler: func() {
				if state.ActiveTab != "Grafik" {
					state.ActiveTab = "Grafik"
					t.FocusManager().SetFocused("")
					state.TransitionStartTime = time.Now()
					state.IsTransitioning = true
				}
			}},
		{Label: "Playground Sekmesine Git", Detail: "", Category: "Navigasyon",
			Handler: func() {
				if state.ActiveTab != "Playground" {
					state.ActiveTab = "Playground"
					t.FocusManager().SetFocused("")
					state.TransitionStartTime = time.Now()
					state.IsTransitioning = true
				}
			}},

		{Label: "3D Model: Küp", Detail: "1", Category: "3D Grafik",
			Handler: func() { state.ActiveTab = "Grafik"; state.ThreeDModel = "Küp" }},
		{Label: "3D Model: Piramit", Detail: "2", Category: "3D Grafik",
			Handler: func() { state.ActiveTab = "Grafik"; state.ThreeDModel = "Piramit" }},
		{Label: "3D Model: Dörtyüzlü", Detail: "3", Category: "3D Grafik",
			Handler: func() { state.ActiveTab = "Grafik"; state.ThreeDModel = "Dörtyüzlü" }},
	}
	if state.OBJModel != nil {
		cmdItems = append(cmdItems, widgets.CommandItem{Label: "3D Model: OBJ Dosyası", Detail: "7", Category: "3D Grafik",
			Handler: func() { state.ActiveTab = "Grafik"; state.ThreeDModel = "OBJ" }})
	}
	cmdItems = append(cmdItems,
		widgets.CommandItem{Label: "Render Stili: Dokulu", Detail: "4", Category: "3D Grafik",
			Handler: func() { state.ActiveTab = "Grafik"; state.ThreeDStyle = "Dokulu" }},
		widgets.CommandItem{Label: "Render Stili: Dolu Renkli", Detail: "5", Category: "3D Grafik",
			Handler: func() { state.ActiveTab = "Grafik"; state.ThreeDStyle = "Dolu Renkli" }},
		widgets.CommandItem{Label: "Render Stili: Kafes", Detail: "6", Category: "3D Grafik",
			Handler: func() { state.ActiveTab = "Grafik"; state.ThreeDStyle = "Kafes" }},
	)

	// KeybindingManager'dan otomatik olarak kısayol komutlarını da ekle
	cmdItems = append(cmdItems, state.KeyManager.ToCommandItems()...)
	state.CmdPalette.AllItems = cmdItems
	state.CmdPalette.Filtered = widgets.FuzzyFilter("", cmdItems)

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
				recordReferenceInteraction(state, fmt.Sprintf("key type=%d rune=%q ctrl=%t alt=%t shift=%t", ev.Key.Type, ev.Key.Ch, ev.Key.Ctrl, ev.Key.Alt, ev.Key.Shift))
				focused := t.FocusManager().Focused()

				// Palet açıksa tüm tuşları ona yönlendir. Ctrl+P burada
				// paleti kapatır; kapalıyken aşağıdaki KeybindingManager açar.
				paletteWasOpen := state.CmdPalette.IsOpen
				if state.CmdPalette.HandleKey(ev.Key) {
					if paletteWasOpen && !state.CmdPalette.IsOpen {
						t.FocusManager().SetFocused("")
					}
					break
				}
				if state.ActiveTab == "Referans" && ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'a' &&
					!state.ShowExitDialog && !state.ShowHelpDialog {
					state.ReferenceAccessibilityASCII = !state.ReferenceAccessibilityASCII
					state.LastKey = "Accessibility ASCII modu değişti"
					break
				}
				// Markdown alanı odaktayken ok tuşları global focus kısayollarına
				// gitmemeli; doğrudan içeriği kaydırmalıdır.
				// Giriş sekmesinde Bilgilendirme alanı bir metin editörü değildir;
				// odak başka bir widget'a geçmiş olsa bile ok tuşları scroll'u
				// doğrudan bu viewport'a yönlendirilir. Aksi halde bir redraw
				// sonrasında focus değişince alan "çalışmıyor" gibi görünür.
				markdownKey := ev.Key.Type == backend.KeyArrowUp || ev.Key.Type == backend.KeyArrowDown || (ev.Key.Type == backend.KeyRune && (ev.Key.Ch == '+' || ev.Key.Ch == '-'))
				if state.ActiveTab == "Giriş" && markdownKey && (focused == "demo_markdown" || focused == "" || focused[:minInt(len(focused), len("tab_"))] == "tab_") {
					switch {
					case ev.Key.Type == backend.KeyArrowUp && state.MarkdownOffset > 0:
						state.MarkdownOffset--
					case ev.Key.Type == backend.KeyArrowDown:
						state.MarkdownOffset++
					case ev.Key.Type == backend.KeyRune && ev.Key.Ch == '+' && state.MarkdownHeight < 12:
						state.MarkdownHeight++
					case ev.Key.Type == backend.KeyRune && ev.Key.Ch == '-' && state.MarkdownHeight > 4:
						state.MarkdownHeight--
					}
					break
				}
				// Bazı klavye düzenlerinde soru işareti terminale '?' yerine
				// Shift+/ üretiminin ham '/' rune'u olarak ulaşabilir. Metin
				// alanlarında slash normal karakter olarak kalmalı; yalnızca
				// global kısayol kullanılabilir durumdaysa yardım panelini aç.
				if state.KeyManager != nil && canHandleGlobalCommand() && ev.Key.Type == backend.KeyRune &&
					(ev.Key.Ch == '?' || (ev.Key.Ch == '/' && ev.Key.Shift)) && !ev.Key.Ctrl && !ev.Key.Alt {
					openHelp()
					break
				}
				// Declarative kısayolların merkezi yönlendiricisi.
				if state.KeyManager.Handle(ev.Key, t.FocusManager().ActiveScopes()...) {
					break
				}

				// Playground yönü, bir kontrol widget'ı odakta değilken ok tuşlarıyla değişir.
				playgroundControlFocused := focused == "play_direction" || focused == "play_ratio" || focused == "play_mode" || focused == "border_rounded" || focused == "border_double" || focused == "border_thick" || focused == "play_grid_cb" || focused == "avatar_opacity"
				if state.ActiveTab == "Playground" && !playgroundControlFocused && !state.ShowExitDialog && !state.ShowHelpDialog && !state.NotifPopupState.IsOpen {
					if ev.Key.Type == backend.KeyArrowLeft || ev.Key.Type == backend.KeyArrowRight || ev.Key.Type == backend.KeyArrowUp || ev.Key.Type == backend.KeyArrowDown {
						if state.PlaygroundDir == layout.Horizontal {
							state.PlaygroundDir = layout.Vertical
						} else {
							state.PlaygroundDir = layout.Horizontal
						}
						state.LastKey = "Playground Yön Değiştir"
						break
					}
				}

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
							t.ForceFullRedraw()
						}
					}
					if ev.Key.Type == backend.KeyEsc {
						state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
						t.FocusManager().SetFocused("")
						t.ForceFullRedraw()
					}
					state.LastKey = fmt.Sprintf("Çıkış Diyalog Tuşu: %d", ev.Key.Type)
					break // Diğer klavye olaylarını yut!
				}

				// Eğer Yardım Modali açıksa, sadece Esc ile kapat
				if state.ShowHelpDialog {
					if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == '?') {
						state.ShowHelpDialog = false
						t.FocusManager().SetFocused("")
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

				// Yön tuşları veya Vim tuşları (h/j/k/l) ile 2B spatial odak navigasyonu
				var spatialDir terminal.FocusDirection
				isSpatialKey := false

				// Hangi widget'ların yön tuşlarını yutacağını kontrol et
				consumesArrow := false
				consumesVim := false

				switch focused {
				case "username_input", "showcase_input":
					// Text inputlar sol/sağ yön tuşlarını ve tüm karakter tuşlarını (Vim harfleri) yutar
					consumesVim = true
					if ev.Key.Type == backend.KeyArrowLeft || ev.Key.Type == backend.KeyArrowRight {
						consumesArrow = true
					}
				case "demo_slider", "showcase_slider", "avatar_opacity":
					// Slider tüm yön tuşlarını yutar
					if ev.Key.Type == backend.KeyArrowLeft || ev.Key.Type == backend.KeyArrowRight || ev.Key.Type == backend.KeyArrowUp || ev.Key.Type == backend.KeyArrowDown {
						consumesArrow = true
					}
				case "play_direction", "play_mode", "play_border", "play_showcase_select":
					// Select bileşeni yukarı/aşağı yön tuşlarını yutar
					if ev.Key.Type == backend.KeyArrowUp || ev.Key.Type == backend.KeyArrowDown {
						consumesArrow = true
					}
				case "table_filter":
					// Tablo arama alanında yön tuşları sıralama kontrolüne
					// aittir; spatial focus navigasyonuna kaçmamalıdır.
					// Ctrl+Sol/Sağ aşağıda TextInput cursor hareketine aktarılır.
					if ev.Key.Type == backend.KeyArrowUp || ev.Key.Type == backend.KeyArrowDown ||
						ev.Key.Type == backend.KeyArrowLeft || ev.Key.Type == backend.KeyArrowRight {
						consumesArrow = true
					}
				}

				switch ev.Key.Type {
				case backend.KeyArrowUp:
					if !consumesArrow {
						spatialDir = terminal.DirUp
						isSpatialKey = true
					}
				case backend.KeyArrowDown:
					if !consumesArrow {
						spatialDir = terminal.DirDown
						isSpatialKey = true
					}
				case backend.KeyArrowLeft:
					if !consumesArrow {
						spatialDir = terminal.DirLeft
						isSpatialKey = true
					}
				case backend.KeyArrowRight:
					if !consumesArrow {
						spatialDir = terminal.DirRight
						isSpatialKey = true
					}
				case backend.KeyRune:
					if !consumesVim {
						switch ev.Key.Ch {
						case 'k':
							spatialDir = terminal.DirUp
							isSpatialKey = true
						case 'j':
							spatialDir = terminal.DirDown
							isSpatialKey = true
						case 'h':
							spatialDir = terminal.DirLeft
							isSpatialKey = true
						case 'l':
							spatialDir = terminal.DirRight
							isSpatialKey = true
						}
					}
				}

				if isSpatialKey {
					if t.FocusManager().MoveFocus2D(spatialDir) {
						state.LastKey = fmt.Sprintf("Yön Odaklanma (%v)", ev.Key.Type)
						break
					}
				}

				// Tab ve Shift+Tab tuşlarıyla form elemanları arası odak geçişi
				if ev.Key.Type == backend.KeyTab {
					if ev.Key.Shift {
						navigateDemoTab(state, t.FocusManager(), -1)
						state.LastKey = "Shift+Tab (Sekme Önceki)"
					} else {
						t.FocusManager().NextExcluding("tab_")
						state.LastKey = "Tab (Aktif Widget)"
					}
					break
				}

				// Sekme menüsü focus'taysa Enter/Space ile sekmeyi aç.
				if strings.HasPrefix(focused, "tab_") && (ev.Key.Type == backend.KeyEnter || ev.Key.Type == backend.KeySpace) {
					tabName := strings.TrimPrefix(focused, "tab_")
					if tabName != "Çıkış" {
						state.ActiveTab = tabName
						state.IsTransitioning = false
						t.SetTransitionActive(false)
					}
					break
				}

				// Eğer bir TextInput aktif odaklıysa, klavye girdilerini ona yönlendir
				if focused == "username_input" {
					if state.UsernameInputState.HandleKey(ev.Key) {
						// TextInput durumu güncellendi
					}

				} else if focused == "demo_slider" {
					state.DemoSliderState.HandleKey(ev.Key, 0, 100)
				} else if focused == "avatar_opacity" {
					state.AvatarOpacityState.HandleKey(ev.Key, 0, 100)
				} else if focused == "play_direction" {
					state.PlayDirectionState.Open = true
					state.PlayDirectionState.HandleKey(ev.Key, 2)
					if state.PlayDirectionState.Selected == 0 {
						state.PlaygroundDir = layout.Horizontal
					} else {
						state.PlaygroundDir = layout.Vertical
					}
				} else if focused == "border_rounded" || focused == "border_double" || focused == "border_thick" {
					borderIDs := []string{"border_rounded", "border_double", "border_thick"}
					borderValues := []string{"Rounded", "Double", "Thick"}
					index := 0
					for i, id := range borderIDs {
						if id == focused {
							index = i
							break
						}
					}
					if ev.Key.Type == backend.KeyArrowUp {
						index = (index + 2) % 3
					}
					if ev.Key.Type == backend.KeyArrowDown {
						index = (index + 1) % 3
					}
					state.PlaygroundBorder = borderValues[index]
					t.FocusManager().SetFocused(borderIDs[index])
				} else if focused == "play_grid_cb" {
					if ev.Key.Type == backend.KeySpace || ev.Key.Type == backend.KeyEnter {
						state.PlayShowGrid = !state.PlayShowGrid
					}
				} else if focused == "play_mode" {
					state.PlayModeState.Open = true
					state.PlayModeState.HandleKey(ev.Key, 8)
					switch state.PlayModeState.Selected {
					case 0:
						state.PlaygroundMode = "Vector"
					case 1:
						state.PlaygroundMode = "Matrix"
					case 2:
						state.PlaygroundMode = "Chart"
					case 3:
						state.PlaygroundMode = "ChartTable"
					case 4:
						state.PlaygroundMode = "Particle"
					case 5:
						state.PlaygroundMode = "Dither"
					case 6:
						state.PlaygroundMode = "Profiler"
					case 7:
						state.PlaygroundMode = "VirtualList"
					}

				} else if focused == "play_showcase_select" {
					state.ShowcaseSelectState.Open = true
					state.ShowcaseSelectState.HandleKey(ev.Key, 4)
					switch state.ShowcaseSelectState.Selected {
					case 0:
						state.ShowcaseSelected = "Paragraph"
					case 1:
						state.ShowcaseSelected = "Table"
					case 2:
						state.ShowcaseSelected = "Forms"
					case 3:
						state.ShowcaseSelected = "Vector"
					}
				} else if focused == "play_ratio" {
					state.PlayRatioState.HandleKey(ev.Key, 10, 90)
					state.PlaygroundRatio = state.PlayRatioState.Value
				} else if focused == "table_filter" {
					switch ev.Key.Type {
					case backend.KeyArrowLeft:
						if ev.Key.Ctrl {
							state.TableFilterState.HandleKey(ev.Key)
							break
						}
						state.TableState.MoveSortColumn(-1, 5)
						state.LastKey = "Sıralama sütunu önceki"
					case backend.KeyArrowRight:
						if ev.Key.Ctrl {
							state.TableFilterState.HandleKey(ev.Key)
							break
						}
						state.TableState.MoveSortColumn(1, 5)
						state.LastKey = "Sıralama sütunu sonraki"
					case backend.KeyArrowUp, backend.KeyArrowDown:
						if state.TableState.SortColumn < 0 {
							state.TableState.SortColumn = 2
						} // Varsayılan: CPU
						state.TableState.SortDescending = ev.Key.Type == backend.KeyArrowDown
						state.LastKey = "Tablo sıralama yönü değişti"
					default:
						state.TableFilterState.HandleKey(ev.Key)
					}

				} else if focused == "process_table" {
					if ev.Key.Type == backend.KeyArrowDown {
						state.TableState.Next(len(state.Processes))
						state.LastKey = "Tablo Aşağı (Ok Tuşu)"
					} else if ev.Key.Type == backend.KeyArrowUp {
						state.TableState.Prev()
						state.LastKey = "Tablo Yukarı (Ok Tuşu)"
					} else if ev.Key.Type == backend.KeyArrowLeft {
						state.TableState.ScrollHorizontal(-2)
						state.LastKey = "Tablo Sola Kaydır (Sol Ok)"
					} else if ev.Key.Type == backend.KeyArrowRight {
						state.TableState.ScrollHorizontal(2)
						state.LastKey = "Tablo Sağa Kaydır (Sağ Ok)"
					} else if ev.Key.Type == backend.KeySpace && state.TableState.Selected >= 0 {
						state.TableState.ToggleRow(state.TableState.Selected)
						state.LastKey = "Tablo satır seçimi değişti"
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
					case "theme_contrast_rb":
						state.ThemeSelected = "Yüksek Kontrast"
					case "notif_popup":
						state.NotifPopupState.Toggle()
					}
				}

				state.LastKey = fmt.Sprintf("Kod: %d, Karakter: %q, Ctrl: %v", ev.Key.Type, string(ev.Key.Ch), ev.Key.Ctrl)

			case backend.EventMouse:
				handled := t.RouteMouseEvent(ev.Mouse)
				state.ReferenceInteractionPointerX = ev.Mouse.X
				state.ReferenceInteractionPointerY = ev.Mouse.Y
				state.ReferenceInteractionHover = t.HoveredRegionID()
				if state.ReferenceInteractionHover == "" {
					state.ReferenceInteractionHover = "no semantic target"
				}
				state.ReferenceInteractionLastRoute = fmt.Sprintf("handled=%t", handled)
				recordReferenceInteraction(state, fmt.Sprintf("mouse button=%d pos=(%d,%d) drag=%t route=%t", ev.Mouse.Button, ev.Mouse.X, ev.Mouse.Y, ev.Mouse.Drag, handled))
				if !handled {
					if ev.Mouse.Drag {
						if state.IsDraggingModal {
							dx := int(ev.Mouse.X) - state.DragMouseStartX
							dy := int(ev.Mouse.Y) - state.DragMouseStartY
							state.ModalOffsetX = state.ModalDragBaseX + dx
							state.ModalOffsetY = state.ModalDragBaseY + dy
						} else if state.IsResizingModal {
							dx := int(ev.Mouse.X) - state.DragMouseStartX
							dy := int(ev.Mouse.Y) - state.DragMouseStartY
							newW := state.ModalResizeBaseW + dx
							newH := state.ModalResizeBaseH + dy
							if newW < 40 {
								newW = 40
							}
							if newW > 100 {
								newW = 100
							}
							if newH < 10 {
								newH = 10
							}
							if newH > 30 {
								newH = 30
							}
							state.HelpDialogW = newW
							state.HelpDialogH = newH
						} else if state.IsDragging3D {
							dx := int(ev.Mouse.X) - state.Drag3DLastX
							dy := int(ev.Mouse.Y) - state.Drag3DLastY
							state.RotX = math.Mod(state.RotX+float64(dx)*1.5, 360.0)
							state.RotY = math.Mod(state.RotY-float64(dy)*1.5, 360.0)
							state.Drag3DLastX = int(ev.Mouse.X)
							state.Drag3DLastY = int(ev.Mouse.Y)
						}
					} else if ev.Mouse.Button == backend.MouseRelease {
						state.IsDraggingModal = false
						state.IsResizingModal = false
						state.IsDragging3D = false
					}
					state.LastMouse = fmt.Sprintf("Buton: %d, Pozisyon: (%d, %d), Sürükleme: %v", ev.Mouse.Button, ev.Mouse.X, ev.Mouse.Y, ev.Mouse.Drag)
				} else {
					if ev.Mouse.Button == backend.MouseRelease {
						state.IsDraggingModal = false
						state.IsResizingModal = false
						state.IsDragging3D = false
					}
				}

			case backend.EventResize:
				recordReferenceInteraction(state, fmt.Sprintf("resize %dx%d", ev.Resize.Width, ev.Resize.Height))
			case backend.EventFocus:
				recordReferenceInteraction(state, fmt.Sprintf("focus gained=%t", ev.Focus.Gained))
			case backend.EventPaste:
				recordReferenceInteraction(state, fmt.Sprintf("paste %d chars", len(ev.Paste.Text)))
			}
			// Input state is visible immediately; do not wait for the animation tick.
			drawApp(t, b, state, fps)

		case <-ticker.C:
			// Animasyonları güncelle
			now := time.Now()
			state.UpdateAnimations(now)
			if state.LastProcessRead.IsZero() || now.Sub(state.LastProcessRead) >= 500*time.Millisecond {
				state.Processes, state.ProcessSamples = readLiveProcesses(state.ProcessSamples, now)
				state.LastProcessRead = now
			}

			// Dither geçiş ilerlemesini güncelle
			if state.IsTransitioning {
				elapsed := time.Since(state.TransitionStartTime)
				progress := float64(elapsed) / float64(250*time.Millisecond)
				if progress >= 1.0 {
					progress = 1.0
					state.IsTransitioning = false
					t.SetTransitionActive(false)
				} else {
					t.SetTransitionActive(true)
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// drawApp, uygulamanın durumunu okur ve ekranın yerleşimini çizdirir.
func drawApp(t *terminal.Terminal, b *backend.Backend, state *AppState, fps float64) {
	t.SetDebugMode(state.DebugMode)
	// Modal açılışı, sekme dither'ından bağımsız bir animasyondur. Önceki
	// sekme geçişinin old-frame'i modalın üzerine taşınırsa aynı panel iki
	// farklı konumda görünür; modal açıkken terminal geçişini iptal et.
	if state.ShowHelpDialog || state.ShowExitDialog {
		t.SetTransitionActive(false)
	}
	var accessibilityFrame *terminal.Frame
	t.Draw(func(f *terminal.Frame) {
		accessibilityFrame = f
		demoTheme := themeForSelection(state.ThemeSelected)
		f.SetTheme(demoTheme)
		f.RegisterAccessibility(accessibility.AccessibilityNode{
			ID: "limoni-demo", Role: accessibility.RoleDialog,
			Label: "Limoni TUI demo", Value: state.ActiveTab,
			Bounds: f.Buffer.Area,
			Children: []accessibility.AccessibilityNode{{
				ID: "active-tab", Role: accessibility.RoleGeneric,
				Label: "Aktif sekme", Value: state.ActiveTab,
				Bounds: f.Buffer.Area,
			}},
		})
		// Tüm ana UI renkleri semantic theme token'larından gelir.
		mainColor := demoTheme.Colors.Primary
		accentColor := demoTheme.Colors.Success

		// Eğer çıkış veya yardım diyalogu açık olacaksa, en baştan modalı kaydet ki çizilen arka plan widget'ları olay alamasın!
		if state.ShowExitDialog {
			dialogW, dialogH := uint16(46), uint16(9)
			dialogArea := terminal.CenterRect(f.Buffer.Area, dialogW, dialogH)
			dialogArea.X = uint16(int(dialogArea.X) + state.ModalOffsetX)
			dialogArea.Y = uint16(int(dialogArea.Y) + state.ModalOffsetY)
			// Modal alanı sabit kalır. Resimlerin native yerleşimi bu alana göre
			// yeniden ölçeklenmez veya yeniden konumlandırılmaz. Görsel dialog
			// aşağıda ayrıca animasyonlu olarak çizilir.
			f.RegisterModal("exit_dialog", dialogArea, func() {
				state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
				t.ForceFullRedraw()
			})
		}
		if state.ShowHelpDialog {
			helpW := uint16(state.HelpDialogW)
			helpH := uint16(state.HelpDialogH)
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

		// Sol Panel (Menü Bölmesi) - Dikeyde 6 adet buton ve esnek boşluk içerir
		menuLay := layout.NewFlexLayout(
			layout.Vertical,
			1,               // Butonlar arasında 1 satır boşluk
			layout.Fixed(3), // Giriş Butonu
			layout.Fixed(3), // Ayarlar Butonu
			layout.Fixed(3), // Grafik Butonu
			layout.Fixed(3), // Playground Butonu
			layout.Fixed(3), // Referans Butonu
			layout.Fixed(3), // Çıkış Butonu
			layout.Fill(),
		)
		menuChunks := menuLay.Split(bodyChunks[0])

		// drawButton, sol menüdeki tıklanabilir butonları ve focus callback'lerini çizer
		drawButton := func(area cell.Rect, title string, tabName string) {
			focusID := "tab_" + tabName
			t.FocusManager().Register(focusID)
			borderCol := state.TabColors[tabName].Value()
			if t.FocusManager().IsFocused(focusID) {
				borderCol = demoTheme.Colors.Primary
			}
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
			registerTargetClick(f, area, func(ev backend.MouseEvent) {
				if tabName == "Çıkış" {
					state.ShowExitDialog = true
					state.ExitDialogAnim.AnimateTo(1.0, 250*time.Millisecond, animation.EaseOutCubic)
					t.FocusManager().SetFocused("exit_dialog_btn_1")
					t.ForceFullRedraw()
				} else {
					t.FocusManager().SetFocused(focusID)
					if state.ActiveTab != tabName {
						state.ActiveTab = tabName
						// Sekme geçişinde eski frame'i hücre hücre harmanlamak,
						// özellikle metin ve canvas alanlarında eski panel parçaları
						// bırakabiliyor. Yeni sekmeyi temiz frame olarak çiz.
						state.IsTransitioning = false
						t.SetTransitionActive(false)
					}
				}
			})
		}

		drawButton(menuChunks[0], "1. Giris", "Giriş")
		drawButton(menuChunks[1], "2. Ayarlar", "Ayarlar")
		drawButton(menuChunks[2], "3. Grafik", "Grafik")
		drawButton(menuChunks[3], "4. OyunAlani", "Playground")
		drawButton(menuChunks[4], "5. Referans", "Referans")
		drawButton(menuChunks[5], "6. Cikis", "Çıkış")

		// Çıkış buton alanı koordinatını kaydet
		state.ExitButtonArea = menuChunks[5]

		// Profiler & Capabilities HUD (Çizim if Height >= 6)
		if menuChunks[6].Height >= 6 {
			caps := terminal.DetectCapabilities()
			lastFrameTime := t.LastFrameDuration()

			var lines []string
			lines = append(lines, fmt.Sprintf("Kare: %5.2f ms", float64(lastFrameTime.Microseconds())/1000.0))

			trueColorText := "TrueColor: [✕]"
			if caps.TrueColor {
				trueColorText = "TrueColor: [✓]"
			}
			lines = append(lines, trueColorText)

			var protoName string
			switch caps.GraphicsProto {
			case graphics.ProtocolKitty:
				protoName = "Kitty"
			case graphics.ProtocolSixel:
				protoName = "Sixel"
			case graphics.ProtocolIterm2:
				protoName = "iTerm2"
			default:
				protoName = "HalfBlock"
			}
			lines = append(lines, "Grafik: "+protoName)

			if menuChunks[6].Height >= 8 && len(t.LastWidgetStats()) > 0 {
				var slowestType string
				var slowestDur time.Duration
				for _, stat := range t.LastWidgetStats() {
					if stat.Duration > slowestDur {
						slowestDur = stat.Duration
						slowestType = stat.Type
					}
				}
				lines = append(lines, fmt.Sprintf("Yavas: %s", slowestType))
				lines = append(lines, fmt.Sprintf("  %5.2f ms", float64(slowestDur.Microseconds())/1000.0))
			}

			linesText := strings.Join(lines, "\n")
			f.RenderWidget(widgets.Block{
				Title:         " PROFILER ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 120, 120)},
				PaddingLeft:   1,
				PaddingRight:  1,
				Child:         label{text: linesText, style: cell.Style{Fg: cell.NewColorRGB(180, 180, 180)}},
			}, menuChunks[6])
		}

		// Sekme geçişi Terminal seviyesindeki dither motoru tarafından uygulanır.
		// Böylece eski frame ile yeni frame tüm hücrelerde doğru şekilde harmanlanır;
		// gövdeyi ayrı bir geçici buffer'a çizip gösterilmeyen hücreleri temiz bırakmayız.

		// Sağ Panel (İçerik Paneli) Çizimi
		switch state.ActiveTab {
		case "Giriş":
			drawHome(t, f, state, demoTheme, mainColor, accentColor, bodyChunks[1])
		case "Ayarlar":
			drawSettings(t, f, state, demoTheme, mainColor, accentColor, bodyChunks[1])
		case "Referans":
			drawReference(t, f, state, demoTheme, mainColor, accentColor, bodyChunks[1])

		case "Grafik":
			// Grafik sekmesini yatayda iki eşit bölüme ayır: Sol tarafta Canvas, Sağ tarafta Resim ve Kontroller
			grafikLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Percentage(50),
				layout.Percentage(50),
			)
			grafikChunks := grafikLay.Split(bodyChunks[1])

			// Sağ tarafı dikey olarak ikiye böl: Üstte Gerçek Resim, Altta 3D Model Kontrolleri
			sağLay := layout.NewFlexLayout(
				layout.Vertical,
				1,
				layout.Percentage(50),
				layout.Percentage(50),
			)
			sağChunks := sağLay.Split(grafikChunks[1])

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

			virtualW := int(w) * 2
			virtualH := int(h) * 4

			if virtualW > 2 && virtualH > 2 {
				// 3D rotasyon sürüklemesi için tıklama alanını kaydet
				registerTargetClick(f, grafikChunks[0], func(ev backend.MouseEvent) {
					state.IsDragging3D = true
					state.Drag3DLastX = int(ev.X)
					state.Drag3DLastY = int(ev.Y)
				})

				// 3D Model Tanımları (Köşeler ve Yüzler)
				var vertices []graphics.Vertex3D
				var faces [][]int

				switch state.ThreeDModel {
				case "Piramit":
					// Kare tabanlı Piramit (Apex tepe noktası)
					vertices = []graphics.Vertex3D{
						{X: -1.0, Y: 0.6, Z: -1.0}, // 0: sol-ön
						{X: 1.0, Y: 0.6, Z: -1.0},  // 1: sağ-ön
						{X: 1.0, Y: 0.6, Z: 1.0},   // 2: sağ-arka
						{X: -1.0, Y: 0.6, Z: 1.0},  // 3: sol-arka
						{X: 0.0, Y: -1.2, Z: 0.0},  // 4: tepe (apex)
					}
					faces = [][]int{
						{3, 2, 1, 0}, // Taban
						{0, 1, 4},    // Ön yüz
						{1, 2, 4},    // Sağ yüz
						{2, 3, 4},    // Arka yüz
						{3, 0, 4},    // Sol yüz
					}

				case "Dörtyüzlü":
					// Düzgün Dörtyüzlü (Üçgen Piramit)
					vertices = []graphics.Vertex3D{
						{X: 0.0, Y: -1.2, Z: 0.0},  // 0: tepe
						{X: -1.0, Y: 0.8, Z: -0.8}, // 1: sol-ön
						{X: 1.0, Y: 0.8, Z: -0.8},  // 2: sağ-ön
						{X: 0.0, Y: 0.8, Z: 1.2},   // 3: arka
					}
					faces = [][]int{
						{1, 2, 3}, // Taban
						{0, 2, 1}, // Ön-Sol
						{0, 3, 2}, // Ön-Sağ
						{0, 1, 3}, // Arka
					}
				case "OBJ":
					if state.OBJModel != nil {
						vertices = state.OBJModel.Vertices
						faces = state.OBJModel.Faces
					}
				default: // "Küp"
					vertices = []graphics.Vertex3D{
						{X: -1.0, Y: -1.0, Z: -1.0},
						{X: 1.0, Y: -1.0, Z: -1.0},
						{X: 1.0, Y: 1.0, Z: -1.0},
						{X: -1.0, Y: 1.0, Z: -1.0},
						{X: -1.0, Y: -1.0, Z: 1.0},
						{X: 1.0, Y: -1.0, Z: 1.0},
						{X: 1.0, Y: 1.0, Z: 1.0},
						{X: -1.0, Y: 1.0, Z: 1.0},
					}
					faces = [][]int{
						{0, 1, 2, 3}, // Front (Z = -1)
						{5, 4, 7, 6}, // Back (Z = 1)
						{1, 5, 6, 2}, // Right (X = 1)
						{4, 0, 3, 7}, // Left (X = -1)
						{3, 2, 6, 7}, // Top (Y = 1)
						{4, 5, 1, 0}, // Bottom (Y = -1)
					}
				}

				projected := make([]struct {
					x, y    int
					z       float64
					visible bool
				}, len(vertices))

				canvasW := float64(virtualW)
				canvasH := float64(virtualH)

				for i, v := range vertices {
					// Eksen rotasyonları uygula (RotateY first, then RotateX!)
					v = v.RotateY(state.RotY)
					v = v.RotateX(state.RotX)
					v = v.RotateZ(state.RotZ)

					// Projeksiyon (Mesafe: 3.5, Ölçek: canvas yüksekliğinin %40'ı)
					scale := canvasH * 0.40
					px, py, visible := graphics.Project(v, canvasW, canvasH, 3.5, scale)
					projected[i] = struct {
						x, y    int
						z       float64
						visible bool
					}{x: int(px), y: int(py), z: v.Z, visible: visible}
				}

				// Yüzey renkleri (Dolu Renkli mod için prizmatik renk geçişleri)
				faceColors := []cell.Color{
					cell.NewColorRGB(0, 255, 255), // Neon Turkuaz
					cell.NewColorRGB(255, 0, 255), // Neon Pembe
					cell.NewColorRGB(255, 255, 0), // Neon Sarı
					cell.NewColorRGB(0, 255, 0),   // Neon Yeşil
					cell.NewColorRGB(255, 128, 0), // Neon Turuncu
					cell.NewColorRGB(0, 128, 255), // Neon Mavi
				}

				textureImg := state.AppleImg

				// Yüzeyleri kapla ve kenarlıkları çiz.
				wireStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 255)}
				if state.ThreeDStyle == "Dokulu" {
					wireStyle = cell.Style{Fg: cell.NewColorRGB(70, 75, 80)} // İnce ve parlamayan koyu gri kenar stili
				}
				getFaceUV := func(faceIndex, corner int, fallback graphics.UV) graphics.UV {
					if state.OBJModel == nil || faceIndex >= len(state.OBJModel.FaceUVs) || corner >= len(state.OBJModel.FaceUVs[faceIndex]) {
						return fallback
					}
					uvIndex := state.OBJModel.FaceUVs[faceIndex][corner]
					if uvIndex < 0 || uvIndex >= len(state.OBJModel.UVs) {
						return fallback
					}
					uv := state.OBJModel.UVs[uvIndex]
					return graphics.UV{U: uv.U, V: 1 - uv.V}
				}
				for faceIdx, face := range faces {
					if len(face) < 3 {
						continue
					}

					p0 := projected[face[0]]
					p1 := projected[face[1]]
					p2 := projected[face[2]]

					if !p0.visible || !p1.visible || !p2.visible {
						continue
					}

					var p3 struct {
						x, y    int
						z       float64
						visible bool
					}
					isQuad := len(face) == 4
					if isQuad {
						p3 = projected[face[3]]
						if !p3.visible {
							continue
						}
					}

					// 2D Winding / Back-face Culling Testi
					cross := (float64(p1.x-p0.x) * float64(p2.y-p0.y)) - (float64(p1.y-p0.y) * float64(p2.x-p0.x))
					if cross < 0 {
						if state.ThreeDStyle == "Dokulu" && textureImg != nil {
							if isQuad {
								// Default UV coordinates (Full image mapping)
								uMin, uMax, vMin, vMax := 0.0, 1.0, 0.0, 1.0

								uv0 := getFaceUV(faceIdx, 0, graphics.UV{U: uMin, V: vMax})
								uv1 := getFaceUV(faceIdx, 1, graphics.UV{U: uMax, V: vMax})
								uv2 := getFaceUV(faceIdx, 2, graphics.UV{U: uMax, V: vMin})
								uv3 := getFaceUV(faceIdx, 3, graphics.UV{U: uMin, V: vMin})

								canvas.DrawTexturedTriangle(
									graphics.Vertex2D{X: float64(p0.x), Y: float64(p0.y)},
									graphics.Vertex2D{X: float64(p1.x), Y: float64(p1.y)},
									graphics.Vertex2D{X: float64(p2.x), Y: float64(p2.y)},
									uv0, uv1, uv2, textureImg,
								)
								canvas.DrawTexturedTriangle(
									graphics.Vertex2D{X: float64(p0.x), Y: float64(p0.y)},
									graphics.Vertex2D{X: float64(p2.x), Y: float64(p2.y)},
									graphics.Vertex2D{X: float64(p3.x), Y: float64(p3.y)},
									uv0, uv2, uv3, textureImg,
								)
							} else {
								uv0 := getFaceUV(faceIdx, 0, graphics.UV{U: 0.0, V: 1.0})
								uv1 := getFaceUV(faceIdx, 1, graphics.UV{U: 1.0, V: 1.0})
								uv2 := getFaceUV(faceIdx, 2, graphics.UV{U: 0.5, V: 0.0})

								canvas.DrawTexturedTriangle(
									graphics.Vertex2D{X: float64(p0.x), Y: float64(p0.y)},
									graphics.Vertex2D{X: float64(p1.x), Y: float64(p1.y)},
									graphics.Vertex2D{X: float64(p2.x), Y: float64(p2.y)},
									uv0, uv1, uv2, textureImg,
								)
							}
						} else if state.ThreeDStyle == "Dolu Renkli" {
							col := faceColors[faceIdx%len(faceColors)]
							if state.OBJModel != nil && faceIdx < len(state.OBJModel.FaceMaterials) {
								if material, ok := state.OBJModel.Materials[state.OBJModel.FaceMaterials[faceIdx]]; ok {
									col = cell.NewColorRGB(material.R, material.G, material.B)
								}
							}
							faceStyle := cell.Style{Fg: col}
							if isQuad {
								canvas.DrawFilledTriangleDepth(
									graphics.Vertex2D{X: float64(p0.x), Y: float64(p0.y)},
									graphics.Vertex2D{X: float64(p1.x), Y: float64(p1.y)},
									graphics.Vertex2D{X: float64(p2.x), Y: float64(p2.y)},
									p0.z, p1.z, p2.z, faceStyle,
								)
								canvas.DrawFilledTriangleDepth(
									graphics.Vertex2D{X: float64(p0.x), Y: float64(p0.y)},
									graphics.Vertex2D{X: float64(p2.x), Y: float64(p2.y)},
									graphics.Vertex2D{X: float64(p3.x), Y: float64(p3.y)},
									p0.z, p2.z, p3.z, faceStyle,
								)
							} else {
								canvas.DrawFilledTriangleDepth(
									graphics.Vertex2D{X: float64(p0.x), Y: float64(p0.y)},
									graphics.Vertex2D{X: float64(p1.x), Y: float64(p1.y)},
									graphics.Vertex2D{X: float64(p2.x), Y: float64(p2.y)},
									p0.z, p1.z, p2.z, faceStyle,
								)
							}
						}

						// Sadece ön yüze ait olan kenarlıkları çiz (Arka köşelerin görünmesini engeller)
						canvas.DrawLine(p0.x, p0.y, p1.x, p1.y, wireStyle)
						canvas.DrawLine(p1.x, p1.y, p2.x, p2.y, wireStyle)
						if isQuad {
							canvas.DrawLine(p2.x, p2.y, p3.x, p3.y, wireStyle)
							canvas.DrawLine(p3.x, p3.y, p0.x, p0.y, wireStyle)
						} else {
							canvas.DrawLine(p2.x, p2.y, p0.x, p0.y, wireStyle)
						}
					}
				}
			}

			canvasBlock := widgets.Block{
				Title:          fmt.Sprintf(" 📦 3D %s (%s) ", state.ThreeDModel, state.ThreeDStyle),
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 255)},
				Child:          canvas,
			}
			f.RenderWidget(canvasBlock, grafikChunks[0])

			// 2. SAĞ ÜST TARAF: Gerçek Görsel Gösterimi (Native Image)
			imageBlock := widgets.Block{
				Title:          " GERÇEK RESİM GÖSTERİMİ ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 0, 255)},
				Child:          widgets.Image{Img: state.ActiveImg, ForceHalfBlock: true},
			}
			f.RenderWidget(imageBlock, sağChunks[0])

			// 3. SAĞ ALT TARAF: 3D Model Kontrol Paneli
			modelLabel := " [1] Küp (PNG Görsel) "
			if state.ThreeDModel == "Küp" {
				modelLabel = " 🔴 [1] Küp (Aktif) "
			}
			piramitLabel := " [2] Piramit "
			if state.ThreeDModel == "Piramit" {
				piramitLabel = " 🔴 [2] Piramit (Aktif) "
			}
			dortyuzluLabel := " [3] Dörtyüzlü "
			if state.ThreeDModel == "Dörtyüzlü" {
				dortyuzluLabel = " 🔴 [3] Dörtyüzlü (Aktif) "
			}
			dokuluLabel := " [4] Dokulu (PNG Texture) "
			if state.ThreeDStyle == "Dokulu" {
				dokuluLabel = " 🟢 [4] Dokulu (Aktif) "
			}
			doluLabel := " [5] Dolu Renkli (Prizmatik) "
			if state.ThreeDStyle == "Dolu Renkli" {
				doluLabel = " 🟢 [5] Dolu Renkli (Aktif) "
			}
			kafesLabel := " [6] Kafes (Tel Kafes) "
			if state.ThreeDStyle == "Kafes" {
				kafesLabel = " 🟢 [6] Kafes (Aktif) "
			}
			var ctrlLines []string
			ctrlLines = append(ctrlLines, "Model Seçimi (Klavye 1-3):")
			ctrlLines = append(ctrlLines, "  "+modelLabel)
			ctrlLines = append(ctrlLines, "  "+piramitLabel)
			ctrlLines = append(ctrlLines, "  "+dortyuzluLabel)
			ctrlLines = append(ctrlLines, "")
			ctrlLines = append(ctrlLines, "Render Stili (Klavye 4-6):")
			ctrlLines = append(ctrlLines, "  "+dokuluLabel)
			ctrlLines = append(ctrlLines, "  "+doluLabel)
			ctrlLines = append(ctrlLines, "  "+kafesLabel)
			ctrlLines = append(ctrlLines, "")
			ctrlLines = append(ctrlLines, "💡 Sürükleyerek 3D uzayda döndürün.")

			ctrlBlock := widgets.Block{
				Title:          " 3D MODEL KONTROLLERİ ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 128)},
				Child:          widgets.List{Items: ctrlLines},
			}
			f.RenderWidget(ctrlBlock, sağChunks[1])

		case "Playground":
			drawPlayground(t, b, f, state, mainColor, accentColor, bodyChunks[1])
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

			// Dialog sabit boyutlu bir overlay'dir. Açılış/kapanış animasyonu
			// görsel alanı küçültür/büyütür; modal ve resim alanına dokunmaz.
			progress := state.ExitDialogAnim.Value()
			animatedArea := terminal.ScaleRect(dialogArea, progress)

			// Animasyonun bitip bitmediğini denetle
			if progress <= 0.001 && !state.ExitDialogAnim.IsAnimating() {
				state.ShowExitDialog = false
				t.FocusManager().SetFocused("")
				t.ForceFullRedraw()
			} else {
				if progress >= 0.999 && !state.ExitDialogAnim.IsAnimating() && !state.ExitDialogFinished {
					state.ExitDialogFinished = true
					t.ForceFullRedraw()
				} else if progress < 0.999 {
					state.ExitDialogFinished = false
				}
				// Başlık çubuğu sürükleme tıklama alanını tanımla
				titleBarArea := cell.NewRect(animatedArea.X, animatedArea.Y, animatedArea.Width, 1)
				registerTargetClick(f, titleBarArea, func(ev backend.MouseEvent) {
					if ev.Button != backend.MouseLeft {
						return
					}
					state.IsDraggingModal = true
					state.DragMouseStartX = int(ev.X)
					state.DragMouseStartY = int(ev.Y)
					state.ModalDragBaseX = state.ModalOffsetX
					state.ModalDragBaseY = state.ModalOffsetY
					f.CaptureMouse(func(dragEv backend.MouseEvent) {
						if dragEv.Button == backend.MouseRelease {
							state.IsDraggingModal = false
							return
						}
						if dragEv.Drag {
							state.ModalOffsetX, state.ModalOffsetY = clampDialogOffset(f.Buffer.Area, 46, 9,
								state.ModalDragBaseX+int(dragEv.X)-state.DragMouseStartX,
								state.ModalDragBaseY+int(dragEv.Y)-state.DragMouseStartY)
						}
					})
				})

				// Native profil resmi hücre tamponundan bağımsız çizildiği için
				// yalnızca Dialog.Draw içindeki hücre arka planı resmi örtemez.
				// Sabit opak katman resmi değiştirmeden dialogun arkasını kapatır;
				// Dialog'un ASCII içeriği bunun üstünde çizilir.
				// Native kaplama da dialogla aynı animasyonlu alanı takip eder.
				// Böylece açılışta arka plan/gölge dialogdan önce görünmez.
				if animatedArea.Width > 0 && animatedArea.Height > 0 {
					shadowBackdrop := cell.NewRect(animatedArea.X, animatedArea.Y, animatedArea.Width+2, animatedArea.Height+1)
					f.RenderWidget(widgets.Block{
						Style:  cell.Style{Bg: cell.NewColorRGB(18, 20, 24)},
						Opaque: true,
					}, shadowBackdrop)
				}

				exitDialog := widgets.Dialog{
					ID:          "exit_dialog",
					Title:       " ⚠️ SİSTEMDEN ÇIKIŞ ",
					Message:     "Uygulamadan çıkmak istediğinize emin misiniz?",
					SubMessage:  "Oturum ve kaydedilmemiş tüm veriler sonlandırılacaktır.",
					Style:       cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(25, 25, 25)},
					HeaderStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(220, 60, 60)},
					BorderStyle: cell.Style{Fg: cell.NewColorRGB(220, 60, 60)},
					ButtonStyle: cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(45, 45, 45)},
					ButtonFocusedStyle: cell.Style{
						Fg:       cell.NewColorRGB(255, 255, 255),
						Bg:       accentColor,
						Modifier: cell.ModifierBold,
					},
					Shadow: true,
					Buttons: []widgets.DialogButton{
						{
							Text: "Evet",
							Handler: func() {
								b.Close()
								fmt.Println("\nLimoni TUI uygulamasından çıkış yapıldı. Görüşmek üzere!")
								os.Exit(0)
							},
						},
						{
							Text: "Hayır",
							Handler: func() {
								state.ExitDialogAnim.AnimateTo(0.0, 200*time.Millisecond, animation.EaseInCubic)
								t.ForceFullRedraw()
							},
						},
					},
				}

				f.BeginFocusScope("exit_dialog")
				f.RenderWidget(exitDialog, animatedArea)
			} // else
		} // if state.ShowExitDialog

		// 6. KISAYOL YARDIM MODAL DIALOG ÇİZİMİ
		if state.ShowHelpDialog {
			helpW := uint16(state.HelpDialogW)
			helpH := uint16(state.HelpDialogH)
			helpArea := terminal.CenterRect(f.Buffer.Area, helpW, helpH)
			state.ModalOffsetX, state.ModalOffsetY = clampDialogOffset(f.Buffer.Area, helpW, helpH, state.ModalOffsetX, state.ModalOffsetY)

			// Sürükleme offsetlerini uygula
			helpArea.X = uint16(int(helpArea.X) + state.ModalOffsetX)
			helpArea.Y = uint16(int(helpArea.Y) + state.ModalOffsetY)

			progress := state.HelpDialogAnim.Value()
			if progress <= 0.001 && !state.HelpDialogAnim.IsAnimating() {
				state.ShowHelpDialog = false
				t.FocusManager().SetFocused("")
			} else {
				// Animasyonlu Y koordinat kaydırmasını hesapla (SlideDown)
				offsetY := int(float64(f.Buffer.Area.Height) * (1.0 - progress))
				animatedHelpArea := helpArea
				animatedHelpArea.Y = uint16(int(animatedHelpArea.Y) + offsetY)

				// Başlık çubuğu sürükleme tıklama alanını tanımla
				titleBarArea := cell.NewRect(animatedHelpArea.X, animatedHelpArea.Y, helpW, 1)
				registerTargetClick(f, titleBarArea, func(ev backend.MouseEvent) {
					if ev.Button != backend.MouseLeft {
						return
					}
					state.IsDraggingModal = true
					state.DragMouseStartX = int(ev.X)
					state.DragMouseStartY = int(ev.Y)
					state.ModalDragBaseX = state.ModalOffsetX
					state.ModalDragBaseY = state.ModalOffsetY
					f.CaptureMouse(func(dragEv backend.MouseEvent) {
						if dragEv.Button == backend.MouseRelease {
							state.IsDraggingModal = false
							return
						}
						if dragEv.Drag {
							state.ModalOffsetX, state.ModalOffsetY = clampDialogOffset(f.Buffer.Area, helpW, helpH,
								state.ModalDragBaseX+int(dragEv.X)-state.DragMouseStartX,
								state.ModalDragBaseY+int(dragEv.Y)-state.DragMouseStartY)
						}
					})
				})

				// Diyalog kutusu arka plan bloğu
				helpBlock := widgets.Block{
					Title:          " ⌨ KISAYOL YARDIMI (Sürükle / Köşeden Boyutlandır) ",
					TitleAlignment: widgets.AlignLeft,
					Borders:        widgets.BorderAll,
					BorderSymbols:  widgets.SymbolsRounded,
					BorderStyle:    cell.Style{Fg: accentColor},
					Style:          cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(25, 25, 25)},
					Opaque:         true,
				}
				f.BeginFocusScope("help_dialog")
				f.RenderWidget(helpBlock, animatedHelpArea)

				// Sağ alt köşe yeniden boyutlandırma tutamacı çizimi
				cornerX := animatedHelpArea.X + animatedHelpArea.Width - 1
				cornerY := animatedHelpArea.Y + animatedHelpArea.Height - 1
				if c := f.Buffer.Get(cornerX, cornerY); c != nil {
					c.Content = '◢'
					c.Style = cell.Style{Fg: accentColor, Modifier: cell.ModifierBold}
				}

				// Sağ alt köşe yeniden boyutlandırma tıklama alanını tanımla
				resizeHandleArea := cell.NewRect(cornerX, cornerY, 1, 1)
				registerTargetClick(f, resizeHandleArea, func(ev backend.MouseEvent) {
					if ev.Button != backend.MouseLeft {
						return
					}
					state.IsResizingModal = true
					state.DragMouseStartX = int(ev.X)
					state.DragMouseStartY = int(ev.Y)
					state.ModalResizeBaseW = state.HelpDialogW
					state.ModalResizeBaseH = state.HelpDialogH
					f.CaptureMouse(func(dragEv backend.MouseEvent) {
						if dragEv.Button == backend.MouseRelease {
							state.IsResizingModal = false
							return
						}
						if !dragEv.Drag {
							return
						}
						newW := state.ModalResizeBaseW + int(dragEv.X) - state.DragMouseStartX
						newH := state.ModalResizeBaseH + int(dragEv.Y) - state.DragMouseStartY
						if newW < 40 {
							newW = 40
						}
						if newW > 100 {
							newW = 100
						}
						if newH < 10 {
							newH = 10
						}
						if newH > 30 {
							newH = 30
						}
						state.HelpDialogW = newW
						state.HelpDialogH = newH
						state.ModalOffsetX, state.ModalOffsetY = clampDialogOffset(f.Buffer.Area, uint16(newW), uint16(newH), state.ModalOffsetX, state.ModalOffsetY)
					})
				})

				helpInner := cell.Rect{
					X:      animatedHelpArea.X + 2,
					Y:      animatedHelpArea.Y + 1,
					Width:  animatedHelpArea.Width - 4,
					Height: animatedHelpArea.Height - 2,
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
				mdHelp := &widgets.Markdown{
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
					Child:   widgets.Image{Img: state.ActiveImg, CircleMask: true},
				}
				f.RenderWidget(avatarBlock, helpChunks[1])
			}
		}

		// 7. KOMUT PALETİ OVERLAY ÇİZİMİ (En üst katman)
		if state.CmdPalette.IsOpen {
			palette := widgets.CommandPalette{
				ID:    "command_palette",
				State: state.CmdPalette,
				// CSS benzeri konumlandırma: panel ekranın altından 2 satır yukarıda açılır.
				Position: &widgets.CommandPalettePosition{Bottom: 2},
			}
			f.RenderWidget(palette, f.Buffer.Area)
		}
	})
	if state.ScreenReaderMode {
		lineMode := accessibilityFrame.AccessibilityLineMode(accessibility.Mode{ScreenReader: true})
		if lineMode != "" && lineMode != state.LastScreenReaderTree {
			fmt.Fprintf(os.Stderr, "\n[limoni screen-reader]\n%s\n", lineMode)
			state.LastScreenReaderTree = lineMode
		}
	}
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

type processSample struct {
	ticks uint64
	at    time.Time
}

func readLiveProcesses(previous map[string]processSample, now time.Time) ([]ProcessInfo, map[string]processSample) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, previous
	}
	if previous == nil {
		previous = make(map[string]processSample)
	}
	current := make(map[string]processSample, len(entries))
	processes := make([]ProcessInfo, 0, len(entries))
	for _, entry := range entries {
		pid := entry.Name()
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		statData, err := os.ReadFile(filepath.Join("/proc", pid, "stat"))
		if err != nil {
			continue
		}
		statText := string(statData)
		closeParen := strings.LastIndexByte(statText, ')')
		if closeParen < 0 {
			continue
		}
		name := strings.Trim(statText[strings.Index(statText, " ")+1:closeParen], "()")
		fields := strings.Fields(statText[closeParen+2:])
		if len(fields) <= 19 {
			continue
		}
		utime, err1 := strconv.ParseUint(fields[11], 10, 64)
		stime, err2 := strconv.ParseUint(fields[12], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		ticks := utime + stime
		current[pid] = processSample{ticks: ticks, at: now}
		cpu := 0.0
		if old, ok := previous[pid]; ok && now.After(old.at) && ticks >= old.ticks {
			cpu = float64(ticks-old.ticks) / now.Sub(old.at).Seconds()
		}
		processes = append(processes, ProcessInfo{PID: pid, Name: name, CPU: fmt.Sprintf("%.1f%%", cpu), Memory: fmt.Sprintf("%.1f MB", readProcessMemoryMB(pid)), Status: processState(fields[0])})
	}
	sort.Slice(processes, func(i, j int) bool {
		a, _ := strconv.Atoi(processes[i].PID)
		b, _ := strconv.Atoi(processes[j].PID)
		return a < b
	})
	return processes, current
}

func readProcessMemoryMB(pid string) float64 {
	data, err := os.ReadFile(filepath.Join("/proc", pid, "statm"))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	rss, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return float64(rss*uint64(os.Getpagesize())) / (1024 * 1024)
}

func processState(state string) string {
	switch state {
	case "R":
		return "Çalışıyor"
	case "S", "D", "I":
		return "Beklemede"
	case "Z":
		return "Zombi"
	case "T", "t":
		return "Durduruldu"
	default:
		return state
	}
}

func loadDemoMarkdown() string {
	paths := []string{".agents/skills/limoni_development/skill.md", "../../.agents/skills/limoni_development/skill.md"}
	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	return "# Limoni Demo\nMarkdown dosyası okunamadı.\n\n- `skill.md` bulunamadı.\n- Demo fallback içeriği gösteriliyor."
}
