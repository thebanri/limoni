package widgets

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
)

// DialogButton represents a button in the dialog.
type DialogButton struct {
	Text         string
	Handler      func()
	Style        cell.Style
	FocusedStyle cell.Style
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
	FocusedButton      int
	OnButtonHover      func(index int)

	// State keeps the buttons' IDs, labels and mouse handlers between
	// frames. With one the dialog draws without allocating; without one it
	// builds them every frame (about a dozen allocations for two buttons).
	State *DialogState
}

// DialogState is what a Dialog keeps between frames. Keep one per dialog.
type DialogState struct {
	// For each button: its focus ID and drawn label, rebuilt when the
	// dialog's ID or the button's text changes.
	ids, labels  []string
	idsFor       string
	labelsFor    []string
	hover        []func(driver.MouseEvent)
	click        []func()
	lastHandlers []func()
	lastOnHover  func(int)
	lastSetFocus func(string)
}

// prepare brings the cached IDs and labels up to date with di.
func (s *DialogState) prepare(di *Dialog) {
	n := len(di.Buttons)
	if s.idsFor != di.ID || len(s.ids) != n {
		s.ids = s.ids[:0]
		for i := 0; i < n; i++ {
			s.ids = append(s.ids, dialogButtonID(di.ID, i))
		}
		s.idsFor = di.ID
	}
	for len(s.labels) < n {
		s.labels = append(s.labels, "")
		s.labelsFor = append(s.labelsFor, "\x00") // never a button's text
	}
	for i, b := range di.Buttons {
		if s.labelsFor[i] != b.Text {
			s.labels[i], s.labelsFor[i] = dialogButtonLabel(b.Text), b.Text
		}
	}
	for len(s.lastHandlers) < n {
		s.lastHandlers = append(s.lastHandlers, nil)
	}
	for i, b := range di.Buttons {
		s.lastHandlers[i] = b.Handler
	}
	s.lastOnHover = di.OnButtonHover
}

// handlers returns button i's hover and click handlers, built once per state;
// they read the last frame's callbacks.
func (s *DialogState) handlers(i int) (func(driver.MouseEvent), func()) {
	for len(s.hover) <= i {
		b := len(s.hover)
		s.hover = append(s.hover, func(driver.MouseEvent) { s.enter(b) })
		s.click = append(s.click, func() {
			s.enter(b)
			if b < len(s.lastHandlers) && s.lastHandlers[b] != nil {
				s.lastHandlers[b]()
			}
		})
	}
	return s.hover[i], s.click[i]
}

func (s *DialogState) enter(i int) {
	if s.lastSetFocus != nil && i < len(s.ids) {
		s.lastSetFocus(s.ids[i])
	}
	if s.lastOnHover != nil {
		s.lastOnHover(i)
	}
}

func dialogButtonID(id string, i int) string { return id + "_btn_" + strconv.Itoa(i) }

func dialogButtonLabel(text string) string { return " [ " + text + " ] " }

// isDialogButtonID reports whether focused is dialogButtonID(id, i), without
// building it.
func isDialogButtonID(focused, id string, i int) bool {
	rest, ok := strings.CutPrefix(focused, id)
	if !ok {
		return false
	}
	rest, ok = strings.CutPrefix(rest, "_btn_")
	return ok && rest == strconv.Itoa(i)
}

// dialogButton is one button's layout for a frame.
type dialogButton struct {
	label, id string
	width     int
	style     cell.Style
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

	// Opaque backdrop for native image protocols (Kitty, Sixel, iTerm2):
	if ctx.RegisterImage != nil {
		proto := imageProtocol(ctx)
		if proto != graphics.ProtocolHalfBlock {
			backdropArea := ctx.Area
			if di.Shadow {
				backdropArea.Width += 2
				backdropArea.Height += 1
			}
			solidImg := getSolidImage(bgCol)
			ctx.RegisterImage(backdropArea, solidImg, -2, false)
		}
	}

	for dy := uint16(0); dy < boxH; dy++ {
		by := y + dy
		for dx := uint16(0); dx < boxW; dx++ {
			bx := x + dx
			buf.SetCellDirect(bx, by, cell.Cell{Content: ' ', Style: baseStyle})
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
		var content rune
		if dx == 0 {
			content = sym.TopLeft
		} else if dx == boxW-1 {
			content = sym.TopRight
		} else {
			content = sym.Horizontal
		}
		buf.SetCellDirect(col, y, cell.Cell{
			Content: content,
			Style:   cell.Style{Fg: gColor, Bg: bgCol},
		})
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
			var content rune
			if dx == 0 {
				content = sym.BottomLeft
			} else if dx == boxW-1 {
				content = sym.BottomRight
			} else {
				content = sym.Horizontal
			}
			buf.SetCellDirect(col, y+boxH-1, cell.Cell{
				Content: content,
				Style:   cell.Style{Fg: gColor, Bg: bgCol},
			})
		}
	}

	// Side borders (if boxH >= 3)
	if boxH >= 3 {
		for dy := uint16(1); dy < boxH-1; dy++ {
			row := y + dy
			factor := float64(dy) / float64(boxH-1)
			gColor := getGradientColor(factor)
			st := cell.Style{Fg: gColor, Bg: bgCol}
			buf.SetCellDirect(x, row, cell.Cell{Content: sym.Vertical, Style: st})
			if boxW >= 2 {
				buf.SetCellDirect(x+boxW-1, row, cell.Cell{Content: sym.Vertical, Style: st})
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
			buf.SetCellDirect(col, sepY, cell.Cell{
				Content: '─',
				Style:   cell.Style{Fg: blendWithColor(gColor, bgCol, 0.5), Bg: bgCol},
			})
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

		var st *DialogState
		if di.State != nil {
			st = di.State
			st.prepare(&di)
			st.lastSetFocus = ctx.SetFocus
		}

		// Up to eight buttons are laid out on the stack.
		var layoutStore [8]dialogButton
		btnList := layoutStore[:0]
		if len(di.Buttons) > len(layoutStore) {
			btnList = make([]dialogButton, 0, len(di.Buttons))
		}
		totalBtnsW := 0

		hasDialogFocus := false
		for j := range di.Buttons {
			if isDialogButtonID(ctx.FocusedID, di.ID, j) {
				hasDialogFocus = true
				break
			}
		}

		for i, btn := range di.Buttons {
			var btnID, btnText string
			if st != nil {
				btnID, btnText = st.ids[i], st.labels[i]
			} else {
				btnID, btnText = dialogButtonID(di.ID, i), dialogButtonLabel(btn.Text)
			}
			if ctx.RegisterFocus != nil {
				ctx.RegisterFocus(btnID)
			}
			isFocused := false
			if hasDialogFocus {
				isFocused = (ctx.FocusedID == btnID)
			} else if di.FocusedButton >= 0 && di.FocusedButton < len(di.Buttons) {
				isFocused = (di.FocusedButton == i)
			}
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
			if btn.Style.Fg.Type() != cell.ColorDefault || btn.Style.Bg.Type() != cell.ColorDefault {
				bStyle = btn.Style
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
				if btn.FocusedStyle.Fg.Type() != cell.ColorDefault || btn.FocusedStyle.Bg.Type() != cell.ColorDefault {
					bStyle = btn.FocusedStyle
				}
			}

			btnList = append(btnList, dialogButton{label: btnText, id: btnID, width: btnW, style: bStyle})
			totalBtnsW += btnW
			if i > 0 {
				totalBtnsW += spacing
			}
		}

		curBtnX := int(x) + 1
		if totalBtnsW < innerW {
			curBtnX = int(x) + 1 + (innerW-totalBtnsW)/2
		}

		for i, item := range btnList {
			if curBtnX >= int(x+boxW-1) {
				break
			}
			maxW := int(x+boxW-1) - curBtnX
			if maxW <= 0 {
				break
			}
			drawnW := int(setClipped(buf, uint16(curBtnX), btnY, item.label, item.style, maxW))

			// Register hover and click handlers strictly within dialog inner bounds
			if drawnW > 0 {
				btnArea := cell.NewRect(uint16(curBtnX), btnY, uint16(drawnW), 1)
				if st != nil {
					hover, click := st.handlers(i)
					if ctx.RegisterMouse != nil {
						ctx.RegisterMouse(btnArea, hover)
					}
					if ctx.RegisterClick != nil {
						ctx.RegisterClick(btnArea, click)
					}
				} else {
					registerDialogButton(ctx, btnArea, item.id, i, di.Buttons[i].Handler, di.OnButtonHover)
				}
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

	curY := topMsgY
	if di.Message != "" {
		curY = drawDialogText(buf, di.Message, x+1, innerW, maxMsgW, curY, bottomMsgY, bodyStyle.AddModifier(cell.ModifierBold))
	}

	if di.SubMessage != "" && curY < bottomMsgY {
		if int(bottomMsgY)-int(curY) > countDialogLines(di.SubMessage, maxMsgW) {
			curY++
		}
		subStyle := cell.Style{
			Fg:       cell.NewColorRGB(140, 145, 160),
			Bg:       bgCol,
			Modifier: cell.ModifierItalic,
		}
		drawDialogText(buf, di.SubMessage, x+1, innerW, maxMsgW, curY, bottomMsgY, subStyle)
	}
}

// registerDialogButton registers a button's hover and click for a Dialog
// without a State: two closures, built every frame.
func registerDialogButton(ctx cell.Context, area cell.Rect, id string, index int, handler func(), onHover func(int)) {
	setFocus := ctx.SetFocus
	enter := func() {
		if setFocus != nil {
			setFocus(id)
		}
		if onHover != nil {
			onHover(index)
		}
	}
	if ctx.RegisterMouse != nil {
		ctx.RegisterMouse(area, func(driver.MouseEvent) { enter() })
	}
	if ctx.RegisterClick != nil {
		ctx.RegisterClick(area, func() {
			enter()
			if handler != nil {
				handler()
			}
		})
	}
}

// drawDialogText wraps text at maxW and draws each line centred in the
// innerW columns starting at left, from row y down to (not including)
// bottom. It returns the row after the last one drawn.
func drawDialogText(buf *buffer.Buffer, text string, left uint16, innerW, maxW int, y, bottom uint16, style cell.Style) uint16 {
	for rest := text; y < bottom; y++ {
		var line string
		var width int
		line, width, rest = nextDialogLine(rest, maxW)
		if line == "" {
			break
		}
		if width > innerW {
			width = innerW
		}
		cx := left + uint16((innerW-width)/2)
		limit := int(left) + innerW
		// Words are drawn one by one, a single space apart, as the line was
		// measured.
		for words := line; words != ""; {
			word, after, _ := strings.Cut(words, " ")
			words = strings.TrimLeft(after, " ")
			if word == "" {
				continue
			}
			room := limit - int(cx)
			if room <= 0 {
				break
			}
			cx += setClipped(buf, cx, y, word, style, room)
			if words != "" && int(cx) < limit {
				buf.SetString(cx, y, " ", style)
				cx++
			}
		}
	}
	return y
}

// countDialogLines is how many lines drawDialogText wraps text into.
func countDialogLines(text string, maxW int) int {
	n := 0
	for rest := text; ; n++ {
		var line string
		line, _, rest = nextDialogLine(rest, maxW)
		if line == "" {
			return n
		}
	}
}

// nextDialogLine takes the first line of text wrapped at maxW: the words,
// split at spaces, that fit with one space between them. line runs from the
// first word to the last, width is its width with single spaces, and rest is
// what remains. A word wider than maxW gets a line of its own.
func nextDialogLine(text string, maxW int) (line string, width int, rest string) {
	start, end := -1, 0
	i := 0
	for i < len(text) {
		if text[i] == ' ' {
			i++
			continue
		}
		j := i
		for j < len(text) && text[j] != ' ' {
			j++
		}
		w := cell.StringWidth(text[i:j])
		switch {
		case start < 0:
			start, end, width = i, j, w
		case maxW > 0 && width+1+w > maxW:
			return text[start:end], width, text[i:]
		default:
			end, width = j, width+1+w
		}
		i = j
	}
	if start < 0 {
		return "", 0, ""
	}
	return text[start:end], width, ""
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

// displayWidth works out the width of characters in terminal cells.
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

// SizeHint tells the layout the dialog is drawn at a flexible size.
func (di Dialog) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

// AccessibilityNode returns the semantic node description for Dialog.
func (d Dialog) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	return accessibility.AccessibilityNode{
		ID:          d.ID,
		Role:        accessibility.RoleDialog,
		Label:       d.Title,
		Value:       d.Message,
		Description: d.SubMessage,
		State:       state,
		Bounds:      bounds,
		SetSize:     len(d.Buttons),
	}
}
