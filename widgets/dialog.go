package widgets

import (
	"fmt"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// DialogButton represents a button in the dialog.
type DialogButton struct {
	Text    string
	Handler func()
}

// Dialog is a premium, modern glassmorphism dialog widget with glowing gradient borders and blended shadows.
type Dialog struct {
	ID                 string
	Title              string
	Message            string
	SubMessage         string
	Buttons            []DialogButton
	Style              cell.Style
	HeaderStyle        cell.Style
	BorderStyle        cell.Style
	ButtonStyle        cell.Style
	ButtonFocusedStyle cell.Style
	BorderSymbols      BorderSymbols
	Shadow             bool
}

// Draw renders the premium glassmorphism dialog inside ctx.Area.
func (di Dialog) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if di.ID == "" || ctx.Area.Width < 10 || ctx.Area.Height < 6 {
		return
	}

	if di.Shadow {
		DrawShadow(buf, ctx.Area, 2, 1)
	}

	boxW := ctx.Area.Width  // Tam genişlik - gölge widget içinde yok, dışarıda çizilir
	boxH := ctx.Area.Height // Tam yükseklik
	x := ctx.Area.X
	y := ctx.Area.Y

	// 0. FULL AREA WIPE — tüm dialog alanını tamamen temizle.
	// ctx.Area.Width ve ctx.Area.Height, animasyonlu (ScaleRect) alana göre geldiğinden
	// bu wipe her karede sadece o an görünen alanı kapsar.
	fullDark := cell.NewColorRGB(12, 14, 18)
	for dy := uint16(0); dy < ctx.Area.Height; dy++ {
		for dx := uint16(0); dx < ctx.Area.Width; dx++ {
			if c := buf.Get(x+dx, y+dy); c != nil {
				c.Content = ' '
				c.Style = cell.Style{Fg: fullDark, Bg: fullDark}
			}
		}
	}

	// 1. GLASSMORPHISM BODY (Frosted glass / dark slate background)
	for dy := uint16(0); dy < boxH; dy++ {
		by := y + dy
		for dx := uint16(0); dx < boxW; dx++ {
			bx := x + dx
			if c := buf.Get(bx, by); c != nil {
				c.Content = ' '
				c.Style.Bg = cell.NewColorRGB(18, 20, 24)
				c.Style.Fg = cell.NewColorRGB(220, 225, 235)
			}
		}
	}

	// 3. GLOWING GRADIENT BORDERS
	startCol := di.BorderStyle.Fg
	if startCol.Type() == cell.ColorDefault {
		startCol = cell.NewColorRGB(255, 80, 80) // Neon Red/Orange
	}
	endCol := di.ButtonFocusedStyle.Bg
	if endCol.Type() == cell.ColorDefault {
		endCol = cell.NewColorRGB(255, 0, 255) // Neon Magenta
	}

	sym := di.BorderSymbols
	if sym.TopLeft == 0 {
		sym = SymbolsRounded
	}

	// Helper to interpolate colors for gradient borders
	getGradientColor := func(factor float64) cell.Color {
		r1, g1, b1 := startCol.RGB()
		r2, g2, b2 := endCol.RGB()
		r := uint8(float64(r1) + float64(int(r2)-int(r1))*factor)
		g := uint8(float64(g1) + float64(int(g2)-int(g1))*factor)
		b := uint8(float64(b1) + float64(int(b2)-int(b1))*factor)
		return cell.NewColorRGB(r, g, b)
	}

	// Draw top border with gradient
	for col := x; col < x+boxW; col++ {
		factor := float64(col-x) / float64(boxW)
		gColor := getGradientColor(factor)
		if c := buf.Get(col, y); c != nil {
			if col == x {
				c.Content = sym.TopLeft
			} else if col == x+boxW-1 {
				c.Content = sym.TopRight
			} else {
				c.Content = sym.Horizontal
			}
			c.Style.Fg = gColor
		}
	}

	// Draw bottom border with gradient
	for col := x; col < x+boxW; col++ {
		factor := float64(col-x) / float64(boxW)
		gColor := getGradientColor(factor)
		if c := buf.Get(col, y+boxH-1); c != nil {
			if col == x {
				c.Content = sym.BottomLeft
			} else if col == x+boxW-1 {
				c.Content = sym.BottomRight
			} else {
				c.Content = sym.Horizontal
			}
			c.Style.Fg = gColor
		}
	}

	// Draw side borders with vertical gradient
	for row := y + 1; row < y+boxH-1; row++ {
		factor := float64(row-y) / float64(boxH)
		gColor := getGradientColor(factor)
		if c := buf.Get(x, row); c != nil {
			c.Content = sym.Vertical
			c.Style.Fg = gColor
		}
		if c := buf.Get(x+boxW-1, row); c != nil {
			c.Content = sym.Vertical
			c.Style.Fg = gColor
		}
	}

	// Draw header separator line (subtle double line)
	headerH := uint16(3)
	sepY := y + headerH
	for col := x + 1; col < x+boxW-1; col++ {
		factor := float64(col-x) / float64(boxW)
		gColor := getGradientColor(factor)
		if c := buf.Get(col, sepY); c != nil {
			c.Content = '─'
			c.Style.Fg = blendWithColor(gColor, cell.NewColorRGB(18, 20, 24), 0.5) // Semi-glowing separator
		}
	}

	// 4. HEADER TITLE
	if di.Title != "" {
		titleLen := uint16(cell.StringWidth(di.Title))
		if titleLen < boxW {
			titleX := x + (boxW-titleLen)/2
			// Draw title with bold, bright white style
			buf.SetString(titleX, y+1, di.Title, cell.Style{
				Fg:       cell.NewColorRGB(255, 255, 255),
				Bg:       buf.Get(titleX, y+1).Style.Bg,
				Modifier: cell.ModifierBold,
			})
		}
	}

	// 5. MESSAGE & SUBMESSAGE
	msgY := y + headerH + 2
	bodyStyle := cell.Style{
		Fg: cell.NewColorRGB(240, 245, 255),
		Bg: buf.Get(x+2, msgY).Style.Bg,
	}

	if di.Message != "" {
		lines := splitMessage(di.Message, int(boxW)-4)
		for i, line := range lines {
			lineW := uint16(cell.StringWidth(line))
			if lineW < boxW {
				buf.SetString(x+(boxW-lineW)/2, msgY+uint16(i), line, bodyStyle.AddModifier(cell.ModifierBold))
			}
		}
		msgY += uint16(len(lines)) - 1
	}

	if di.SubMessage != "" {
		subY := msgY + 2
		if subY >= y+boxH-2 {
			subY = msgY + 1
		}
		subStyle := cell.Style{
			Fg:       cell.NewColorRGB(140, 145, 160),
			Bg:       buf.Get(x+2, subY).Style.Bg,
			Modifier: cell.ModifierItalic,
		}
		subLen := uint16(cell.StringWidth(di.SubMessage))
		if subLen < boxW-4 {
			buf.SetString(x+(boxW-subLen)/2, subY, di.SubMessage, subStyle)
		}
	}

	// 6. pill BUTTONS (High-Contrast Premium Pill Buttons)
	if len(di.Buttons) > 0 {
		totalBtnsW := 0
		spacing := 4
		for i, btn := range di.Buttons {
			if i > 0 {
				totalBtnsW += spacing
			}
			totalBtnsW += displayWidth(btn.Text) + 6 // " [ Evet ] " format
		}

		btnY := y + boxH - 2
		btnX := x + (boxW-uint16(totalBtnsW))/2

		for i, btn := range di.Buttons {
			btnID := fmt.Sprintf("%s_btn_%d", di.ID, i)
			if ctx.RegisterFocus != nil {
				ctx.RegisterFocus(btnID)
			}

			isFocused := (ctx.FocusedID == btnID)
			btnText := fmt.Sprintf(" [ %s ] ", btn.Text)
			btnW := uint16(displayWidth(btnText))

			bStyle := cell.Style{
				Fg: cell.NewColorRGB(180, 185, 200),
				Bg: cell.NewColorRGB(35, 40, 50),
			}
			if isFocused {
				// Focused button glows neon (interpolated gradient color at button X position)
				factor := float64(btnX-x) / float64(boxW)
				btnGlow := getGradientColor(factor)
				bStyle = cell.Style{
					Fg:       cell.NewColorRGB(255, 255, 255),
					Bg:       btnGlow,
					Modifier: cell.ModifierBold,
				}
			}

			buf.SetString(btnX, btnY, btnText, bStyle)

			// Register click handler
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

// applyShadowMultiplier darkens the background and foreground colors of a cell by a factor.
func applyShadowMultiplier(c *cell.Cell, factor float64) {
	if c == nil {
		return
	}
	if c.Style.Bg.Type() == cell.ColorRGB {
		r, g, b := c.Style.Bg.RGB()
		c.Style.Bg = cell.NewColorRGB(uint8(float64(r)*factor), uint8(float64(g)*factor), uint8(float64(b)*factor))
	} else {
		c.Style.Bg = cell.NewColorRGB(10, 12, 16)
	}

	if c.Style.Fg.Type() == cell.ColorRGB {
		r, g, b := c.Style.Fg.RGB()
		c.Style.Fg = cell.NewColorRGB(uint8(float64(r)*factor), uint8(float64(g)*factor), uint8(float64(b)*factor))
	} else {
		c.Style.Fg = cell.NewColorRGB(25, 30, 40)
	}
}

// blendWithColor blends a cell color with a target solid color by a given alpha.
func blendWithColor(orig cell.Color, target cell.Color, alpha float64) cell.Color {
	r1, g1, b1 := orig.RGB()
	r2, g2, b2 := target.RGB()
	if orig.Type() == cell.ColorDefault {
		return target
	}
	r := uint8(float64(r1)*(1-alpha) + float64(r2)*alpha)
	g := uint8(float64(g1)*(1-alpha) + float64(g2)*alpha)
	b := uint8(float64(b1)*(1-alpha) + float64(b2)*alpha)
	return cell.NewColorRGB(r, g, b)
}

// displayWidth, karakterlerin terminaldeki görsel hücre genişliklerini hesaplar.
func displayWidth(s string) int {
	width := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError {
			break
		}
		width += cell.RuneWidth(r)
		s = s[size:]
	}
	return width
}

// SizeHint, diyalog bileşeninin esnek boyutlu çizilmesini bildirir.
func (di Dialog) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

// splitMessage, uzun mesajları kutu genişliğine göre alt satırlara böler.
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
		} else if cell.StringWidth(currentLine)+1+cell.StringWidth(word) <= maxW {
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
