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
	boxW := ctx.Area.Width
	boxH := ctx.Area.Height
	x := ctx.Area.X
	y := ctx.Area.Y

	if di.ID == "" || boxW < 6 || boxH < 3 {
		return
	}

	if di.Shadow && boxW >= 4 && boxH >= 3 {
		DrawShadow(buf, ctx.Area, 2, 1)
	}

	// 1. GLASSMORPHISM BODY (Solid dark slate background fill strictly within ctx.Area)
	bgCol := cell.NewColorRGB(18, 20, 24)
	if di.Style.Bg.Type() != cell.ColorDefault {
		bgCol = di.Style.Bg
	}
	fgCol := cell.NewColorRGB(220, 225, 235)
	if di.Style.Fg.Type() != cell.ColorDefault {
		fgCol = di.Style.Fg
	}
	baseStyle := cell.Style{Fg: fgCol, Bg: bgCol}

	for dy := uint16(0); dy < boxH; dy++ {
		by := y + dy
		for dx := uint16(0); dx < boxW; dx++ {
			bx := x + dx
			if c := buf.Get(bx, by); c != nil {
				c.Content = ' '
				c.Style = baseStyle
			}
		}
	}

	// 2. GLOWING GRADIENT BORDERS
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

	getGradientColor := func(factor float64) cell.Color {
		if factor < 0 {
			factor = 0
		} else if factor > 1.0 {
			factor = 1.0
		}
		r1, g1, b1 := startCol.RGB()
		r2, g2, b2 := endCol.RGB()
		r := uint8(float64(r1) + float64(int(r2)-int(r1))*factor)
		g := uint8(float64(g1) + float64(int(g2)-int(g1))*factor)
		b := uint8(float64(b1) + float64(int(b2)-int(b1))*factor)
		return cell.NewColorRGB(r, g, b)
	}

	// Top border
	for dx := uint16(0); dx < boxW; dx++ {
		col := x + dx
		factor := 0.0
		if boxW > 1 {
			factor = float64(dx) / float64(boxW-1)
		}
		gColor := getGradientColor(factor)
		if c := buf.Get(col, y); c != nil {
			if dx == 0 {
				c.Content = sym.TopLeft
			} else if dx == boxW-1 {
				c.Content = sym.TopRight
			} else {
				c.Content = sym.Horizontal
			}
			c.Style.Fg = gColor
			c.Style.Bg = bgCol
		}
	}

	// Bottom border (if boxH >= 2)
	if boxH >= 2 {
		for dx := uint16(0); dx < boxW; dx++ {
			col := x + dx
			factor := 0.0
			if boxW > 1 {
				factor = float64(dx) / float64(boxW-1)
			}
			gColor := getGradientColor(factor)
			if c := buf.Get(col, y+boxH-1); c != nil {
				if dx == 0 {
					c.Content = sym.BottomLeft
				} else if dx == boxW-1 {
					c.Content = sym.BottomRight
				} else {
					c.Content = sym.Horizontal
				}
				c.Style.Fg = gColor
				c.Style.Bg = bgCol
			}
		}
	}

	// Side borders (if boxH >= 3)
	if boxH >= 3 {
		for dy := uint16(1); dy < boxH-1; dy++ {
			row := y + dy
			factor := float64(dy) / float64(boxH-1)
			gColor := getGradientColor(factor)
			if c := buf.Get(x, row); c != nil {
				c.Content = sym.Vertical
				c.Style.Fg = gColor
				c.Style.Bg = bgCol
			}
			if boxW >= 2 {
				if c := buf.Get(x+boxW-1, row); c != nil {
					c.Content = sym.Vertical
					c.Style.Fg = gColor
					c.Style.Bg = bgCol
				}
			}
		}
	}

	innerW := int(boxW) - 2
	innerH := int(boxH) - 2
	if innerW <= 0 || innerH <= 0 {
		return
	}

	// 3. HEADER TITLE
	if di.Title != "" && innerH >= 1 {
		titleText := di.Title
		if cell.StringWidth(titleText) > innerW {
			titleText = clipString(titleText, innerW)
		}
		titleLen := cell.StringWidth(titleText)
		titleX := x + 1 + uint16((innerW-titleLen)/2)
		headerTitleStyle := cell.Style{
			Fg:       cell.NewColorRGB(255, 255, 255),
			Bg:       bgCol,
			Modifier: cell.ModifierBold,
		}
		if di.HeaderStyle.Fg.Type() != cell.ColorDefault {
			headerTitleStyle.Fg = di.HeaderStyle.Fg
		}
		if di.HeaderStyle.Bg.Type() != cell.ColorDefault {
			headerTitleStyle.Bg = di.HeaderStyle.Bg
		}
		buf.SetString(titleX, y+1, titleText, headerTitleStyle)
	}

	// 4. HEADER SEPARATOR LINE
	hasSeparator := false
	sepY := y + 3
	if boxH >= 8 && innerW > 0 {
		hasSeparator = true
		for dx := uint16(1); dx < boxW-1; dx++ {
			col := x + dx
			factor := float64(dx) / float64(boxW)
			gColor := getGradientColor(factor)
			if c := buf.Get(col, sepY); c != nil {
				c.Content = '─'
				c.Style.Fg = blendWithColor(gColor, bgCol, 0.5)
				c.Style.Bg = bgCol
			}
		}
	}

	// 5. BUTTONS (Positioned strictly at y + boxH - 2)
	hasButtons := len(di.Buttons) > 0 && boxH >= 5
	btnY := y + boxH - 2

	if hasButtons {
		spacing := 4
		if innerW < 30 {
			spacing = 2
		}
		if innerW < 20 {
			spacing = 1
		}

		type btnLayout struct {
			text    string
			width   int
			btn     DialogButton
			btnID   string
			style   cell.Style
		}

		btnList := make([]btnLayout, len(di.Buttons))
		totalBtnsW := 0

		for i, btn := range di.Buttons {
			btnID := fmt.Sprintf("%s_btn_%d", di.ID, i)
			if ctx.RegisterFocus != nil {
				ctx.RegisterFocus(btnID)
			}
			isFocused := (ctx.FocusedID == btnID)
			btnText := fmt.Sprintf(" [ %s ] ", btn.Text)
			btnW := displayWidth(btnText)

			bStyle := cell.Style{
				Fg: cell.NewColorRGB(180, 185, 200),
				Bg: cell.NewColorRGB(35, 40, 50),
			}
			if di.ButtonStyle.Fg.Type() != cell.ColorDefault {
				bStyle.Fg = di.ButtonStyle.Fg
			}
			if di.ButtonStyle.Bg.Type() != cell.ColorDefault {
				bStyle.Bg = di.ButtonStyle.Bg
			}
			if isFocused {
				factor := 0.5
				btnGlow := getGradientColor(factor)
				bStyle = cell.Style{
					Fg:       cell.NewColorRGB(255, 255, 255),
					Bg:       btnGlow,
					Modifier: cell.ModifierBold,
				}
				if di.ButtonFocusedStyle.Fg.Type() != cell.ColorDefault {
					bStyle.Fg = di.ButtonFocusedStyle.Fg
				}
				if di.ButtonFocusedStyle.Bg.Type() != cell.ColorDefault {
					bStyle.Bg = di.ButtonFocusedStyle.Bg
				}
			}

			btnList[i] = btnLayout{
				text:  btnText,
				width: btnW,
				btn:   btn,
				btnID: btnID,
				style: bStyle,
			}
			totalBtnsW += btnW
			if i > 0 {
				totalBtnsW += spacing
			}
		}

		curBtnX := int(x) + 1
		if totalBtnsW < innerW {
			curBtnX = int(x) + 1 + (innerW-totalBtnsW)/2
		}

		for _, item := range btnList {
			if curBtnX >= int(x+boxW-1) {
				break
			}
			maxW := int(x+boxW-1) - curBtnX
			if maxW <= 0 {
				break
			}
			textToDraw := item.text
			if item.width > maxW {
				textToDraw = clipString(textToDraw, maxW)
			}
			drawnW := cell.StringWidth(textToDraw)
			buf.SetString(uint16(curBtnX), btnY, textToDraw, item.style)

			// Register click handler strictly within dialog inner bounds
			if ctx.RegisterClick != nil && drawnW > 0 {
				btnArea := cell.NewRect(uint16(curBtnX), btnY, uint16(drawnW), 1)
				handler := item.btn.Handler
				btnID := item.btnID
				ctx.RegisterClick(btnArea, func() {
					if ctx.SetFocus != nil {
						ctx.SetFocus(btnID)
					}
					if handler != nil {
						handler()
					}
				})
			}

			curBtnX += item.width + spacing
		}
	}

	// 6. MESSAGE & SUBMESSAGE (Vertically clipped between header and buttons)
	topMsgY := y + 1
	if hasSeparator {
		topMsgY = sepY + 1
	} else if di.Title != "" && innerH >= 2 {
		topMsgY = y + 2
	}

	bottomMsgY := y + boxH - 1
	if hasButtons {
		if btnY > topMsgY+1 {
			bottomMsgY = btnY - 1
		} else {
			bottomMsgY = btnY
		}
	}

	if topMsgY >= bottomMsgY {
		return
	}

	bodyStyle := cell.Style{
		Fg: cell.NewColorRGB(240, 245, 255),
		Bg: bgCol,
	}
	if di.Style.Fg.Type() != cell.ColorDefault {
		bodyStyle.Fg = di.Style.Fg
	}

	maxMsgW := innerW - 2
	if maxMsgW < 1 {
		maxMsgW = innerW
	}

	var msgLines []string
	if di.Message != "" {
		msgLines = splitMessage(di.Message, maxMsgW)
	}

	var subLines []string
	if di.SubMessage != "" {
		subLines = splitMessage(di.SubMessage, maxMsgW)
	}

	curY := topMsgY
	for _, line := range msgLines {
		if curY >= bottomMsgY {
			break
		}
		lineText := line
		if cell.StringWidth(lineText) > innerW {
			lineText = clipString(lineText, innerW)
		}
		lineW := cell.StringWidth(lineText)
		lineX := x + 1 + uint16((innerW-lineW)/2)
		buf.SetString(lineX, curY, lineText, bodyStyle.AddModifier(cell.ModifierBold))
		curY++
	}

	if len(subLines) > 0 && curY < bottomMsgY {
		if int(bottomMsgY)-int(curY) > len(subLines) {
			curY++
		}
		subStyle := cell.Style{
			Fg:       cell.NewColorRGB(140, 145, 160),
			Bg:       bgCol,
			Modifier: cell.ModifierItalic,
		}
		for _, line := range subLines {
			if curY >= bottomMsgY {
				break
			}
			lineText := line
			if cell.StringWidth(lineText) > innerW {
				lineText = clipString(lineText, innerW)
			}
			lineW := cell.StringWidth(lineText)
			lineX := x + 1 + uint16((innerW-lineW)/2)
			buf.SetString(lineX, curY, lineText, subStyle)
			curY++
		}
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

