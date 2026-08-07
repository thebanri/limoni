package widgets

import (
	"fmt"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// DialogButton, diyalog penceresindeki bir butonu ve tıklandığında çalışacak callback'i temsil eder.
type DialogButton struct {
	Text    string
	Handler func()
}

// Dialog, premium tasarımlı, 3D gölgeli ve sekmeli odağa duyarlı bir modal pencere widget'ıdır.
type Dialog struct {
	ID                 string
	Title              string
	Message            string
	Buttons            []DialogButton
	Style              cell.Style
	BorderStyle        cell.Style
	ButtonStyle        cell.Style
	ButtonFocusedStyle cell.Style
	BorderSymbols      BorderSymbols
}

// Draw, diyalog penceresini çizer, sağ/alt gölgelerini ekler ve butonlar için odak/tıklama alanlarını kaydeder.
func (di Dialog) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if di.ID == "" || ctx.Area.Width < 6 || ctx.Area.Height < 4 {
		return
	}

	boxStyle := ctx.Style.Merge(di.Style)

	// 1. 3D GÖLGE ÇİZİMİ
	shadowStyle := cell.Style{Bg: cell.NewColorRGB(20, 20, 20), Fg: cell.NewColorRGB(20, 20, 20)}
	// Dikey gölge (Sağ kenarda, 2 sütun genişliğinde)
	for dy := 1; dy <= int(ctx.Area.Height); dy++ {
		y := int(ctx.Area.Y) + dy
		for dx := 0; dx < 2; dx++ {
			x := int(ctx.Area.X) + int(ctx.Area.Width) + dx
			if c := buf.Get(uint16(x), uint16(y)); c != nil {
				c.Content = ' '
				c.Style = shadowStyle
			}
		}
	}
	// Yatay gölge (Alt kenarda, 1 satır yüksekliğinde)
	for dx := 2; dx < int(ctx.Area.Width)+2; dx++ {
		x := int(ctx.Area.X) + dx
		y := int(ctx.Area.Y) + int(ctx.Area.Height)
		if c := buf.Get(uint16(x), uint16(y)); c != nil {
			c.Content = ' '
			c.Style = shadowStyle
		}
	}

	// 2. DIALOG ARKA PLAN TEMİZLEME VE DOLDURMA
	for dy := 0; dy < int(ctx.Area.Height); dy++ {
		y := ctx.Area.Y + uint16(dy)
		for dx := 0; dx < int(ctx.Area.Width); dx++ {
			x := ctx.Area.X + uint16(dx)
			if c := buf.Get(x, y); c != nil {
				c.Content = ' '
				c.Style = boxStyle
			}
		}
	}

	// 3. KENARLIKLARIN ÇİZİMİ
	borderStyle := ctx.Style.Merge(di.BorderStyle)
	w, h := ctx.Area.Width, ctx.Area.Height
	x, y := ctx.Area.X, ctx.Area.Y

	sym := di.BorderSymbols
	if sym.TopLeft == 0 {
		sym = SymbolsDouble // Varsayılan olarak çift çizgi kenarlık
	}

	// Köşeler
	buf.SetCell(x, y, cell.Cell{Content: sym.TopLeft, Style: borderStyle})
	buf.SetCell(x+w-1, y, cell.Cell{Content: sym.TopRight, Style: borderStyle})
	buf.SetCell(x, y+h-1, cell.Cell{Content: sym.BottomLeft, Style: borderStyle})
	buf.SetCell(x+w-1, y+h-1, cell.Cell{Content: sym.BottomRight, Style: borderStyle})

	// Yatay çizgiler
	for col := x + 1; col < x+w-1; col++ {
		buf.SetCell(col, y, cell.Cell{Content: sym.Horizontal, Style: borderStyle})
		buf.SetCell(col, y+h-1, cell.Cell{Content: sym.Horizontal, Style: borderStyle})
	}
	// Dikey çizgiler
	for row := y + 1; row < y+h-1; row++ {
		buf.SetCell(x, row, cell.Cell{Content: sym.Vertical, Style: borderStyle})
		buf.SetCell(x+w-1, row, cell.Cell{Content: sym.Vertical, Style: borderStyle})
	}

	// Butonların hemen üstüne kesikli yatay çizgi ayırıcı ekle
	dividerY := y + h - 3
	for col := x + 1; col < x+w-1; col++ {
		buf.SetCell(col, dividerY, cell.Cell{Content: '┄', Style: borderStyle.Merge(cell.Style{Fg: cell.NewColorRGB(90, 90, 90)})})
	}

	// Başlık çizimi (Ortalanmış)
	if di.Title != "" {
		titleLen := uint16(len(di.Title))
		if titleLen+4 < w {
			titleX := x + (w-titleLen)/2
			buf.SetString(titleX, y, di.Title, borderStyle.Merge(cell.Style{Modifier: cell.ModifierBold}))
		}
	}

	// 4. MESAJIN ÇİZİMİ (Metin Kaydırma ve Ortalama)
	msgY := y + 2
	lines := splitMessage(di.Message, int(w)-4)
	for i, line := range lines {
		if msgY+uint16(i) < y+h-2 {
			lineW := uint16(len(line))
			lineX := x + (w-lineW)/2
			buf.SetString(lineX, msgY+uint16(i), line, boxStyle)
		}
	}

	// 5. BUTONLARIN ÇİZİMİ VE KÖPRÜLENMESİ
	if len(di.Buttons) > 0 {
		totalBtnsW := 0
		spacing := 3
		for i, btn := range di.Buttons {
			if i > 0 {
				totalBtnsW += spacing
			}
			totalBtnsW += len(btn.Text) + 4 // Örn: "[ Evet ]"
		}

		btnY := y + h - 2
		btnX := x + (w-uint16(totalBtnsW))/2

		for i, btn := range di.Buttons {
			btnID := fmt.Sprintf("%s_btn_%d", di.ID, i)

			// Odak listesine buton ID'sini kaydet
			if ctx.RegisterFocus != nil {
				ctx.RegisterFocus(btnID)
			}

			isBtnFocused := (ctx.FocusedID == btnID)

			btnText := fmt.Sprintf("[ %s ]", btn.Text)
			btnW := uint16(len(btnText))

			bStyle := boxStyle.Merge(di.ButtonStyle)
			if isBtnFocused {
				bStyle = bStyle.Merge(di.ButtonFocusedStyle)
			}

			// Buton metnini yaz
			buf.SetString(btnX, btnY, btnText, bStyle)

			// Tıklama alanını kaydet
			btnArea := cell.NewRect(btnX, btnY, btnW, 1)
			if ctx.RegisterClick != nil {
				handler := btn.Handler
				ctx.RegisterClick(btnArea, func() {
					if ctx.SetFocus != nil {
						ctx.SetFocus(btnID)
					}
					if handler != nil {
						handler()
					}
				})
			}

			btnX += btnW + uint16(spacing)
		}
	}
}

// SizeHint, diyalog penceresinin esnek boyutlu olduğunu bildirir.
func (di Dialog) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

func splitMessage(msg string, maxW int) []string {
	if maxW <= 0 {
		return []string{msg}
	}
	var lines []string
	words := splitWords(msg)
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= maxW {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

// splitWords, widgets/paragraph.go dosyasında zaten tanımlıdır (aynı pakettedir).
