package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawPlayground(t *terminal.Terminal, b *backend.Backend, f *terminal.Frame, state *AppState, mainColor, accentColor cell.Color, bodyArea cell.Rect) {
	// ─── ANA İKİ SÜTUN: Sol Kontroller (34 sütun) | Sağ Önizleme (Fill) ───
	mainLay := layout.NewFlexLayout(
		layout.Horizontal,
		1,
		layout.Fixed(34),
		layout.Fill(),
	)
	mainChunks := mainLay.Split(bodyArea)

	// ═══════════════════════════════════════════════
	//  SOL PANEL: KONTROL PANELİ
	// ═══════════════════════════════════════════════
	drawPlaygroundControls(t, f, state, mainColor, accentColor, mainChunks[0])

	// ═══════════════════════════════════════════════
	//  SAĞ PANEL: CANLI ÖNİZLEME
	// ═══════════════════════════════════════════════
	drawPlaygroundPreview(f, state, mainColor, accentColor, mainChunks[1])
}

// ── Sol Panel: Gruplandırılmış Kontroller ──────────────────────────────────
func drawPlaygroundControls(t *terminal.Terminal, f *terminal.Frame, state *AppState, mainColor, accentColor cell.Color, area cell.Rect) {
	focused := t.FocusManager().Focused()

	// Dış çerçeve
	ctrlBlock := widgets.Block{
		Title:          " ⚙ KONTROL PANELİ ",
		TitleAlignment: widgets.AlignCenter,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: mainColor},
	}
	f.RenderWidget(ctrlBlock, area)

	innerArea := cell.Rect{
		X:      area.X + 1,
		Y:      area.Y + 1,
		Width:  area.Width - 2,
		Height: area.Height - 2,
	}

	// Kontrol elemanlarını dikey olarak grupla
	ctrlLay := layout.NewFlexLayout(
		layout.Vertical,
		1,
		layout.Fixed(1), // 0: Açıklama
		layout.Fixed(5), // 1: Yön Select + açılır seçenekler
		layout.Fixed(5), // 2: Oran Slider
		layout.Fixed(5), // 3: Kenarlık Stili (3 RadioButton)
		layout.Fixed(6), // 4: Render Mode + açılır seçenekler
		layout.Fixed(1), // 5: Checkbox
		layout.Fill(),   // 6: Bilgi kutusu
	)
	ctrlRows := ctrlLay.Split(innerArea)

	// ── 0. Kılavuz satırı ──
	f.RenderWidget(label{
		text:  "Fare ve klavye ile kontrol edin",
		style: cell.Style{Fg: cell.NewColorRGB(130, 140, 160), Modifier: cell.ModifierItalic},
	}, ctrlRows[0])

	// ── 1. Yön Seçimi ──
	// tab_home.go pattern'i: fokusta border rengi değişir, DrawFocusRing kullanılmaz
	dirBorderCol := cell.NewColorRGB(60, 65, 80)
	if focused == "play_direction" {
		dirBorderCol = accentColor
	}
	// State sync: state -> widget (readonly, her karede güncel tutulur)
	if state.PlaygroundDir == layout.Vertical {
		state.PlayDirectionState.Selected = 1
	} else {
		state.PlayDirectionState.Selected = 0
	}
	directionField := cell.NewRect(ctrlRows[1].X+1, ctrlRows[1].Y+1, ctrlRows[1].Width-2, 1)
	registerTargetClick(f, directionField, func(backend.MouseEvent) {
		if state.PlaygroundDir == layout.Horizontal {
			state.PlaygroundDir = layout.Vertical
			state.PlayDirectionState.Selected = 1
		} else {
			state.PlaygroundDir = layout.Horizontal
			state.PlayDirectionState.Selected = 0
		}
	})
	f.RenderWidget(widgets.Block{
		Title: " YÖN ", TitleAlignment: widgets.AlignLeft,
		Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: dirBorderCol},
		Child: widgets.Select{
			ID:      "play_direction",
			Options: []string{"Horizontal", "Vertical"},
			State:   state.PlayDirectionState,
			OnChange: func(index int, _ string) {
				if index == 0 {
					state.PlaygroundDir = layout.Horizontal
				} else {
					state.PlaygroundDir = layout.Vertical
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 210), Bg: cell.NewColorRGB(30, 33, 42)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
			HoverStyle:    cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: mainColor},
		},
	}, ctrlRows[1])

	// ── 2. Oran Slider ──
	// tab_home.go pattern'i takip ediliyor: Block + Child = Slider, padding ile slider'a alan verilir.
	// ÖNEMLİ: state.PlayRatioState.Set() HER KAREDE çağrılMAMALI! Slider kendi state'ini yönetir.
	// Sadece PlaygroundRatio'yu Slider state'inden güncelle.
	sliderBorderCol := cell.NewColorRGB(60, 65, 80)
	if focused == "play_ratio" {
		sliderBorderCol = accentColor
	}
	f.RenderWidget(widgets.Block{
		Title: fmt.Sprintf(" ORAN: %%%d ", state.PlayRatioState.Value), TitleAlignment: widgets.AlignLeft,
		Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: sliderBorderCol},
		PaddingLeft: 1, PaddingRight: 1,
		Child: widgets.Slider{
			ID: "play_ratio", State: state.PlayRatioState,
			Min: 10, Max: 90,
			TrackStyle:  cell.Style{Fg: cell.NewColorRGB(60, 65, 80)},
			FilledStyle: cell.Style{Fg: accentColor},
			ThumbStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
		},
	}, ctrlRows[2])
	// Slider state'den ana state'i güncelle (tek yönlü akış: widget -> app state)
	state.PlaygroundRatio = state.PlayRatioState.Value

	// ── 3. Kenarlık Stili (RadioButton grubu) ──
	borderBorderCol := cell.NewColorRGB(60, 65, 80)
	if focused == "border_rounded" || focused == "border_double" || focused == "border_thick" {
		borderBorderCol = accentColor
	}

	// RadioButton'ları Block'un child'ı olarak değil, elle iç alana çizeceğiz
	// (Block Child'ı tek widget alır, 3 radio button var)
	borderBlock := widgets.Block{
		Title: " KENARLIK ", TitleAlignment: widgets.AlignLeft,
		Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: borderBorderCol},
	}
	f.RenderWidget(borderBlock, ctrlRows[3])

	borderInner := cell.Rect{
		X: ctrlRows[3].X + 1, Y: ctrlRows[3].Y + 1,
		Width: ctrlRows[3].Width - 2, Height: ctrlRows[3].Height - 2,
	}
	if borderInner.Height >= 3 {
		borderLay := layout.NewFlexLayout(layout.Vertical, 0,
			layout.Fixed(1),
			layout.Fixed(1),
			layout.Fixed(1),
		)
		borderChunks := borderLay.Split(borderInner)

		rbStyle := cell.Style{Fg: cell.NewColorRGB(190, 195, 210)}
		rbFocus := cell.Style{Fg: accentColor, Modifier: cell.ModifierBold}

		f.RenderWidget(widgets.RadioButton{
			ID: "border_rounded", Selected: &state.PlaygroundBorder, Value: "Rounded",
			Label: "╭─╮ Oval", Style: rbStyle, FocusedStyle: rbFocus,
		}, borderChunks[0])
		f.RenderWidget(widgets.RadioButton{
			ID: "border_double", Selected: &state.PlaygroundBorder, Value: "Double",
			Label: "╔═╗ Çift", Style: rbStyle, FocusedStyle: rbFocus,
		}, borderChunks[1])
		f.RenderWidget(widgets.RadioButton{
			ID: "border_thick", Selected: &state.PlaygroundBorder, Value: "Thick",
			Label: "┏━┓ Kalın", Style: rbStyle, FocusedStyle: rbFocus,
		}, borderChunks[2])
	}

	modeField := cell.NewRect(ctrlRows[4].X+1, ctrlRows[4].Y+1, ctrlRows[4].Width-2, 1)
	registerTargetClick(f, modeField, func(backend.MouseEvent) {
		state.PlayModeState.Open = !state.PlayModeState.Open
	})

	// ── 4. Render Modu ──
	modeBorderCol := cell.NewColorRGB(60, 65, 80)
	if focused == "play_mode" {
		modeBorderCol = accentColor
	}
	modeIndex := 0
	switch state.PlaygroundMode {
	case "Matrix":
		modeIndex = 1
	case "Chart":
		modeIndex = 2
	}
	state.PlayModeState.Selected = modeIndex
	f.RenderWidget(widgets.Block{
		Title: " RENDER MODU ", TitleAlignment: widgets.AlignLeft,
		Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: modeBorderCol},
		Child: widgets.Select{
			ID:      "play_mode",
			Options: []string{"Vector Canvas", "Matrix Rain", "Sparkline"},
			State:   state.PlayModeState,
			OnChange: func(index int, _ string) {
				switch index {
				case 0:
					state.PlaygroundMode = "Vector"
				case 1:
					state.PlaygroundMode = "Matrix"
				case 2:
					state.PlaygroundMode = "Chart"
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 210), Bg: cell.NewColorRGB(30, 33, 42)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
			HoverStyle:    cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: mainColor},
		},
	}, ctrlRows[4])

	// ── 5. Checkbox ──
	f.RenderWidget(widgets.Checkbox{
		ID: "play_grid_cb", Checked: &state.PlayShowGrid,
		Label:        "Izgara çizgileri",
		Style:        cell.Style{Fg: cell.NewColorRGB(190, 195, 210)},
		FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
	}, ctrlRows[5])

	// ── 6. Bilgi kutusu (Fill alanı) ──
	if ctrlRows[6].Height >= 3 {
		infoLines := []widgets.Line{
			widgets.NewLine(
				widgets.Span{Text: "Mod: ", Style: cell.Style{Fg: cell.NewColorRGB(120, 125, 140)}},
				widgets.Span{Text: state.PlaygroundMode, Style: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold}},
			),
			widgets.NewLine(
				widgets.Span{Text: "Oran: ", Style: cell.Style{Fg: cell.NewColorRGB(120, 125, 140)}},
				widgets.Span{Text: fmt.Sprintf("%d / %d", state.PlaygroundRatio, 100-state.PlaygroundRatio), Style: cell.Style{Fg: cell.NewColorRGB(255, 200, 80)}},
			),
			widgets.NewLine(
				widgets.Span{Text: "Border: ", Style: cell.Style{Fg: cell.NewColorRGB(120, 125, 140)}},
				widgets.Span{Text: state.PlaygroundBorder, Style: cell.Style{Fg: cell.NewColorRGB(180, 255, 180)}},
			),
		}
		f.RenderWidget(widgets.Text{Lines: infoLines, Wrap: true}, ctrlRows[6])
	}
}

// ── Sağ Panel: Canlı Önizleme ──────────────────────────────────────────────
func drawPlaygroundPreview(f *terminal.Frame, state *AppState, mainColor, accentColor cell.Color, area cell.Rect) {
	// Dış çerçeve
	directionLabel := "HORIZONTAL"
	if state.PlaygroundDir == layout.Vertical {
		directionLabel = "VERTICAL"
	}
	previewBlock := widgets.Block{
		Title:          fmt.Sprintf(" CANLI ÖNİZLEME · %s ", directionLabel),
		TitleAlignment: widgets.AlignCenter,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: mainColor},
	}
	f.RenderWidget(previewBlock, area)

	previewInner := cell.Rect{
		X:      area.X + 1,
		Y:      area.Y + 1,
		Width:  area.Width - 2,
		Height: area.Height - 2,
	}
	if previewInner.Width < 4 || previewInner.Height < 4 {
		return
	}

	// Üst durum çubuğu (3 satır) + Alt grid önizleme alanı
	prevLay := layout.NewFlexLayout(
		layout.Vertical,
		1,
		layout.Fixed(3), // Durum çubuğu
		layout.Fill(),   // Grid / Canvas alanı
	)
	prevChunks := prevLay.Split(previewInner)

	// ─── ÜST: Durum Çubuğu ───
	drawPlaygroundStatusBar(f, state, accentColor, prevChunks[0])

	// ─── ALT: Direction-aware preview ───
	if state.PlaygroundDir == layout.Vertical {
		drawPlaygroundVertical(f, state, accentColor, prevChunks[1])
	} else {
		drawPlaygroundGrid(f, state, accentColor, prevChunks[1])
	}
}

// ── Durum çubuğu: ProgressBar + aktif mod göstergesi ────────────────────────
func drawPlaygroundStatusBar(f *terminal.Frame, state *AppState, accentColor cell.Color, area cell.Rect) {
	statusBlock := widgets.Block{
		Title:         " DURUM ",
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: cell.NewColorRGB(60, 65, 80)},
	}
	f.RenderWidget(statusBlock, area)

	// İç alan: oran progress bar
	if area.Width > 4 && area.Height > 2 {
		barArea := cell.NewRect(area.X+2, area.Y+1, area.Width-4, 1)
		f.RenderWidget(widgets.ProgressBar{
			Value: float64(state.PlaygroundRatio), Min: 0, Max: 100,
			ShowPercent: true,
			FilledStyle: cell.Style{Fg: accentColor},
			EmptyStyle:  cell.Style{Fg: cell.NewColorRGB(40, 43, 55)},
		}, barArea)
	}
}

func drawPlaygroundVertical(f *terminal.Frame, state *AppState, accentColor cell.Color, area cell.Rect) {
	profileImg := state.ProfileImg
	if profileImg == nil {
		profileImg = state.ActiveImg
	}
	parts := layout.NewFlexLayout(layout.Vertical, 1, layout.Percentage(28), layout.Percentage(28), layout.Fill()).Split(area)
	borders := widgets.BorderAll
	if !state.PlayShowGrid {
		borders = widgets.BorderNone
	}
	sym := widgets.SymbolsRounded
	f.RenderWidget(widgets.Image{Img: playgroundSurfaceImage, ForceHalfBlock: true}, area)
	f.RenderWidget(widgets.Block{Title: " MARKDOWN · VERTICAL ", Borders: borders, BorderSymbols: sym, BorderStyle: cell.Style{Fg: accentColor}, Child: &widgets.Markdown{Content: "# Limoni TUI\nVertical layout aktif.\n- Markdown paneli\n- Profil maskesi\n- Canvas / Matrix / Sparkline", Style: cell.Style{Fg: cell.NewColorRGB(210, 215, 225)}}}, parts[0])
	f.RenderWidget(widgets.Block{Title: " PROFİL · VERTICAL ", Borders: borders, BorderSymbols: sym, BorderStyle: cell.Style{Fg: cell.NewColorRGB(255, 165, 0)}, Style: cell.Style{Bg: cell.NewColorRGB(25, 28, 36)}, Child: widgets.Image{Img: profileImg, CircleMask: true, ForceHalfBlock: false, OpaqueBackground: true, Background: cell.NewColorRGB(25, 28, 36)}}, parts[1])
	drawPlaygroundCanvas(f, state, accentColor, sym, parts[2])
}

// ── CSS Grid önizleme alanı ─────────────────────────────────────────────────
var playgroundSurfaceImage = image.NewUniform(color.RGBA{R: 25, G: 28, B: 36, A: 255})

func drawPlaygroundGrid(f *terminal.Frame, state *AppState, accentColor cell.Color, area cell.Rect) {
	profileImg := state.ProfileImg
	if profileImg == nil {
		profileImg = state.ActiveImg
	}
	if area.Width < 6 || area.Height < 4 {
		return
	}

	// Grid: Oran bazlı sütunlar + 2 satır
	ratio := state.PlaygroundRatio
	if ratio < 10 {
		ratio = 10
	}
	if ratio > 90 {
		ratio = 90
	}

	columns := []layout.GridConstraint{layout.GridFraction(uint16(ratio)), layout.GridFraction(uint16(100 - ratio))}
	rows := []layout.GridConstraint{layout.GridFraction(1), layout.GridFraction(1)}
	if state.PlaygroundDir == layout.Vertical {
		columns = []layout.GridConstraint{layout.GridFraction(1), layout.GridFraction(1)}
		rows = []layout.GridConstraint{layout.GridFraction(uint16(ratio)), layout.GridFraction(uint16(100 - ratio))}
	}
	gridLayout := layout.NewGridLayout(columns, rows, 1)
	gridAreas := gridLayout.Split(area)

	// Önce tüm preview alanını sabit opak surface ile kapat; native image
	// protocol'ünün önceki geniş placement'larından kalan kenarlar görünmesin.
	f.RenderWidget(widgets.Image{Img: playgroundSurfaceImage, ForceHalfBlock: true}, area)

	// Izgara checkbox kapalıysa hücre iç border'larını kaldır.
	borders := widgets.BorderAll
	if !state.PlayShowGrid {
		borders = widgets.BorderNone
	}

	// Kenarlık sembollerini seç
	var sym widgets.BorderSymbols
	switch state.PlaygroundBorder {
	case "Rounded":
		sym = widgets.SymbolsRounded
	case "Double":
		sym = widgets.SymbolsDouble
	case "Thick":
		sym = widgets.SymbolsThick
	default:
		sym = widgets.SymbolsRounded
	}

	// ── Hücre (0,0): Markdown ──
	mdBlock := widgets.Block{
		Title:          " MARKDOWN ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        borders,
		BorderSymbols:  sym,
		BorderStyle:    cell.Style{Fg: accentColor},
		PaddingLeft:    1,
		PaddingRight:   1,
		Child: &widgets.Markdown{
			Content: "# Limoni TUI\nRatatui'den *daha esnek* ve **performanslı**.\n- CSS Grid yerleşimi\n- Bayer dither geçişleri\n- Dairesel `avatar` maskeleme\n- **Sıfır tahsisatlı** çizim",
			Style:   cell.Style{Fg: cell.NewColorRGB(210, 215, 225)},
		},
	}
	f.RenderWidget(mdBlock, gridAreas.Cell(0, 0).Area)

	// ── Hücre (0,1): Profil Resmi ──
	profileSymbols := widgets.SymbolsRounded
	if state.ProfileFrame == "Full" {
		profileSymbols = widgets.SymbolsSingle
	}
	if state.ProfileFrame == "Stretched" {
		profileSymbols = widgets.SymbolsDouble
	}
	imgBlock := widgets.Block{
		Title:          fmt.Sprintf(" PROFİL · %s ", state.ProfileFrame),
		TitleAlignment: widgets.AlignCenter,
		Borders:        borders,
		BorderSymbols:  profileSymbols,
		BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 165, 0)},
		Style:          cell.Style{Bg: cell.NewColorRGB(25, 28, 36)},
		Child:          widgets.Image{Img: profileImg, CircleMask: state.ProfileFrame == "Rounded", ForceHalfBlock: false, OpaqueBackground: true, Background: cell.NewColorRGB(25, 28, 36)},
	}
	profileArea := gridAreas.Cell(0, 1).Area
	f.RenderWidget(imgBlock, profileArea)
	registerTargetClick(f, profileArea, func(backend.MouseEvent) {
		switch state.ProfileFrame {
		case "Rounded":
			state.ProfileFrame = "Full"
		case "Full":
			state.ProfileFrame = "Stretched"
		default:
			state.ProfileFrame = "Rounded"
		}
	})

	// ── Hücre (1,0) span 1 row, 2 cols: Canvas / Sparkline ──
	canvasArea := gridAreas.Cell(1, 0).Span(1, 2)
	drawPlaygroundCanvas(f, state, accentColor, sym, canvasArea)
}

// ── Canvas / Sparkline render alanı ─────────────────────────────────────────
func drawPlaygroundCanvas(f *terminal.Frame, state *AppState, accentColor cell.Color, sym widgets.BorderSymbols, canvasArea cell.Rect) {
	borders := widgets.BorderAll
	if !state.PlayShowGrid {
		borders = widgets.BorderNone
	}
	if canvasArea.Width < 4 || canvasArea.Height < 3 {
		return
	}

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

	// Render moduna göre içerik üret
	switch state.PlaygroundMode {
	case "Chart":
		childWidget = widgets.Sparkline{
			Data:  state.CPUHistory,
			Color: accentColor,
		}
	case "Matrix":
		// Matrix parçacık yağmuru
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
		childWidget = canvas
	default: // "Vector"
		yellowStyle := cell.Style{Fg: cell.NewColorRGB(255, 255, 0)}
		cyanStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 255)}
		greenStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}

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
		childWidget = canvas
	}

	// Mod etiketleri
	modeLabel := "CANVAS"
	switch state.PlaygroundMode {
	case "Matrix":
		modeLabel = "MATRIX RAIN"
	case "Chart":
		modeLabel = "SPARKLINE"
	}

	canvasBlock := widgets.Block{
		Title:          fmt.Sprintf(" %s ", modeLabel),
		TitleAlignment: widgets.AlignLeft,
		Borders:        borders,
		BorderSymbols:  sym,
		BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 220, 220)},
		Child:          childWidget,
	}
	f.RenderWidget(canvasBlock, canvasArea)
}
