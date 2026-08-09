package main

import (
	"fmt"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawPlayground(t *terminal.Terminal, b *backend.Backend, f *terminal.Frame, state *AppState, mainColor, accentColor cell.Color, bodyArea cell.Rect) {
	// Playground sekmesini sol (ayarlar - 30 sütun) ve sağ (canlı önizleme - Fill) olarak ikiye böl
	playLay := layout.NewFlexLayout(
		layout.Horizontal,
		1,
		layout.Fixed(30),
		layout.Fill(),
	)
	playChunks := playLay.Split(bodyArea)

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
	horizClickArea := cell.NewRect(ctrlRows[2].X+1, ctrlRows[2].Y, 9, 1)
	registerTargetClick(f, horizClickArea, func(ev backend.MouseEvent) {
		state.PlaygroundDir = layout.Horizontal
	})
	vertClickArea := cell.NewRect(ctrlRows[2].X+12, ctrlRows[2].Y, 9, 1)
	registerTargetClick(f, vertClickArea, func(ev backend.MouseEvent) {
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
	registerTargetClick(f, ctrlRows[4], func(ev backend.MouseEvent) {
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
	ovalArea := cell.NewRect(ctrlRows[6].X+1, ctrlRows[6].Y, 7, 1)
	registerTargetClick(f, ovalArea, func(ev backend.MouseEvent) {
		state.PlaygroundBorder = "Rounded"
	})
	doubleArea := cell.NewRect(ctrlRows[6].X+10, ctrlRows[6].Y, 7, 1)
	registerTargetClick(f, doubleArea, func(ev backend.MouseEvent) {
		state.PlaygroundBorder = "Double"
	})
	thickArea := cell.NewRect(ctrlRows[6].X+19, ctrlRows[6].Y, 8, 1)
	registerTargetClick(f, thickArea, func(ev backend.MouseEvent) {
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
	circleModeArea := cell.NewRect(ctrlRows[8].X+1, ctrlRows[8].Y, 9, 1)
	registerTargetClick(f, circleModeArea, func(ev backend.MouseEvent) {
		state.PlaygroundMode = "Vector"
	})
	matrixModeArea := cell.NewRect(ctrlRows[8].X+12, ctrlRows[8].Y, 9, 1)
	registerTargetClick(f, matrixModeArea, func(ev backend.MouseEvent) {
		state.PlaygroundMode = "Matrix"
	})
	chartModeArea := cell.NewRect(ctrlRows[8].X+23, ctrlRows[8].Y, 9, 1)
	registerTargetClick(f, chartModeArea, func(ev backend.MouseEvent) {
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
		Child: &widgets.Markdown{
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
