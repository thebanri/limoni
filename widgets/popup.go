package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// PopupItem is an item in a popup menu.
type PopupItem struct {
	// Text is the menu item's text.
	Text string
	// Disabled reports whether the item cannot be selected (greyed out).
	Disabled bool
	// Handler is the callback run when the item is chosen.
	Handler func()
}

// PopupState manages whether the popup menu is open and the selected index.
type PopupState struct {
	// IsOpen reports whether the menu is open.
	IsOpen bool
	// Selected is the index of the item under the mouse (hover) or selected with the keyboard.
	Selected int

	// Handlers, built once, and the last frame's callbacks they call.
	onButton     func()
	onItem       []func()
	onHover      []func(driver.MouseEvent)
	lastHandlers []func()
	id           string
	setFocus     func(string)
}

func (ps *PopupState) focus() {
	if ps.setFocus != nil {
		ps.setFocus(ps.id)
	}
}

// buttonHandler focuses the popup and opens or closes it. Built once.
func (ps *PopupState) buttonHandler() func() {
	if ps.onButton == nil {
		ps.onButton = func() {
			ps.focus()
			ps.Toggle()
		}
	}
	return ps.onButton
}

// itemHandlers returns item i's click (close, then run its Handler) and hover
// (select it) handlers, built once per state and index.
func (ps *PopupState) itemHandlers(i int) (func(), func(driver.MouseEvent)) {
	for len(ps.onItem) <= i {
		index := len(ps.onItem)
		ps.onItem = append(ps.onItem, func() {
			ps.focus()
			ps.Close()
			if index < len(ps.lastHandlers) && ps.lastHandlers[index] != nil {
				ps.lastHandlers[index]()
			}
		})
		ps.onHover = append(ps.onHover, func(ev driver.MouseEvent) {
			if ev.Button == driver.MouseNone {
				ps.Selected = index
			}
		})
	}
	return ps.onItem[i], ps.onHover[i]
}

// NewPopupState returns a new PopupState.
func NewPopupState() *PopupState {
	return &PopupState{
		IsOpen:   false,
		Selected: -1,
	}
}

// Open opens the menu.
func (ps *PopupState) Open() {
	ps.IsOpen = true
	ps.Selected = -1
}

// Close closes the menu.
func (ps *PopupState) Close() {
	ps.IsOpen = false
	ps.Selected = -1
}

// Toggle opens or closes the menu.
func (ps *PopupState) Toggle() {
	if ps.IsOpen {
		ps.Close()
	} else {
		ps.Open()
	}
}

// Next moves the selection to the next item (skipping disabled ones).
func (ps *PopupState) Next(totalItems int) {
	if totalItems <= 0 {
		return
	}
	ps.Selected++
	if ps.Selected >= totalItems {
		ps.Selected = 0
	}
}

// Prev moves the selection to the previous item (skipping disabled ones).
func (ps *PopupState) Prev() {
	if ps.Selected > 0 {
		ps.Selected--
	}
}

// Popup is a dropdown menu widget.
// Clicking its button opens a menu list below it.
// Every item is clickable and focusable. A click outside the menu closes it.
type Popup struct {
	// ID uniquely identifies the popup.
	ID string
	// Label is the text on the button.
	Label string
	// Items is the list of menu items.
	Items []PopupItem
	// State holds whether the popup is open and what is selected.
	State *PopupState
	// Style sets the button and menu background style.
	Style cell.Style
	// ItemStyle sets the normal style of the menu items.
	ItemStyle cell.Style
	// SelectedStyle is the style of a menu item under the mouse or selected with the keyboard.
	SelectedStyle cell.Style
	// DisabledStyle is the style of disabled menu items.
	DisabledStyle cell.Style
	// BorderStyle sets the style of the menu border.
	BorderStyle cell.Style
	// BorderSymbols sets the menu border symbols.
	BorderSymbols BorderSymbols
}

// Draw draws the popup button and, when open, the menu list.
// When the menu is open, a click area and focus region are registered for each item.
func (p Popup) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if p.ID == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// 1. THE BUTTON (always visible)
	btnStyle := ctx.Style.Merge(p.Style)
	btnText := " " + p.Label + " ▾ "
	btnW := uint16(cell.StringWidth(btnText))
	btnH := uint16(1)

	// Fill the button background
	for dx := uint16(0); dx < ctx.Area.Width; dx++ {
		if c := buf.Get(ctx.Area.X+dx, ctx.Area.Y); c != nil {
			c.Content = ' '
			c.Style = btnStyle
		}
	}

	// Draw button text
	setClipped(buf, ctx.Area.X, ctx.Area.Y, btnText, btnStyle, int(ctx.Area.Width))
	_ = btnW
	_ = btnH

	// Register the button's click area
	if ctx.RegisterClick != nil {
		btnArea := cell.NewRect(ctx.Area.X, ctx.Area.Y, ctx.Area.Width, 1)
		if p.State != nil {
			p.State.id, p.State.setFocus = p.ID, ctx.SetFocus
			p.State.lastHandlers = p.State.lastHandlers[:0]
			for _, item := range p.Items {
				p.State.lastHandlers = append(p.State.lastHandlers, item.Handler)
			}
			ctx.RegisterClick(btnArea, p.State.buttonHandler())
		} else if setFocus, id := ctx.SetFocus, p.ID; setFocus != nil {
			ctx.RegisterClick(btnArea, func() { setFocus(id) })
		}
	}

	// Register as focusable
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(p.ID)
	}

	// 2. THE MENU LIST (only when open)
	if p.State == nil || !p.State.IsOpen {
		return
	}

	// Work out the menu width from the longest item
	menuW := uint16(0)
	for _, item := range p.Items {
		itemLen := uint16(cell.StringWidth(item.Text)) + 2 // " " padding
		if itemLen > menuW {
			menuW = itemLen
		}
	}
	// Minimum width: the button's width
	if menuW < ctx.Area.Width {
		menuW = ctx.Area.Width
	}
	// Menu height: border (2) + items
	menuH := uint16(len(p.Items)) + 2

	// Menu area (right below the button)
	menuX := ctx.Area.X
	menuY := ctx.Area.Y + 1

	// Fill the menu background
	menuBgStyle := ctx.Style.Merge(p.Style)
	for dy := uint16(0); dy < menuH; dy++ {
		for dx := uint16(0); dx < menuW; dx++ {
			x := menuX + dx
			y := menuY + dy
			if c := buf.Get(x, y); c != nil {
				c.Content = ' '
				c.Style = menuBgStyle
			}
		}
	}

	// Draw the menu border
	borderStyle := ctx.Style.Merge(p.BorderStyle)
	sym := p.BorderSymbols
	if sym.TopLeft == 0 {
		sym = SymbolsRounded
	}

	// Corners
	buf.SetCell(menuX, menuY, cell.Cell{Content: sym.TopLeft, Style: borderStyle})
	buf.SetCell(menuX+menuW-1, menuY, cell.Cell{Content: sym.TopRight, Style: borderStyle})
	buf.SetCell(menuX, menuY+menuH-1, cell.Cell{Content: sym.BottomLeft, Style: borderStyle})
	buf.SetCell(menuX+menuW-1, menuY+menuH-1, cell.Cell{Content: sym.BottomRight, Style: borderStyle})

	// Horizontal borders
	for col := menuX + 1; col < menuX+menuW-1; col++ {
		buf.SetCell(col, menuY, cell.Cell{Content: sym.Horizontal, Style: borderStyle})
		buf.SetCell(col, menuY+menuH-1, cell.Cell{Content: sym.Horizontal, Style: borderStyle})
	}

	// Vertical borders
	for row := menuY + 1; row < menuY+menuH-1; row++ {
		buf.SetCell(menuX, row, cell.Cell{Content: sym.Vertical, Style: borderStyle})
		buf.SetCell(menuX+menuW-1, row, cell.Cell{Content: sym.Vertical, Style: borderStyle})
	}

	// Hover regions first: the region registered last wins a left click, and
	// with hover registered after the click regions, a click on an item
	// reached the hover handler, which ignores clicks, and the menu never
	// ran the item.
	for i := range p.Items {
		if !p.Items[i].Disabled && ctx.RegisterMouse != nil {
			_, hover := p.State.itemHandlers(i)
			ctx.RegisterMouse(cell.NewRect(menuX, menuY+uint16(i)+1, menuW, 1), hover)
		}
	}

	// Draw the menu items
	for i, item := range p.Items {
		itemY := menuY + uint16(i) + 1 // Border allowance
		isSelected := p.State.Selected == i

		itemStyle := menuBgStyle.Merge(p.ItemStyle)
		if item.Disabled {
			itemStyle = menuBgStyle.Merge(p.DisabledStyle)
		} else if isSelected {
			itemStyle = menuBgStyle.Merge(p.SelectedStyle)
		}

		// Fill the item background
		for dx := uint16(1); dx < menuW-1; dx++ {
			x := menuX + dx
			if c := buf.Get(x, itemY); c != nil {
				c.Content = ' '
				c.Style = itemStyle
			}
		}

		// Write the item text (inside the border)
		displayText := " " + item.Text
		setClipped(buf, menuX+1, itemY, displayText, itemStyle, int(menuW)-2)

		// Selected item marker
		if isSelected && !item.Disabled {
			checkMark := "▸"
			buf.SetString(menuX+1, itemY, checkMark, itemStyle)
		}

		// Register the click area
		if ctx.RegisterClick != nil && !item.Disabled {
			click, _ := p.State.itemHandlers(i)
			ctx.RegisterClick(cell.NewRect(menuX, itemY, menuW, 1), click)
		}
	}

}

// SizeHint returns the popup button's height and default width.
func (p Popup) SizeHint(maxArea cell.Rect) (width, height uint16) {
	btnW := uint16(cell.StringWidth(p.Label)) + 4 // " ▾ " padding
	if btnW > maxArea.Width {
		btnW = maxArea.Width
	}
	if btnW < 10 {
		btnW = 10
	}
	return btnW, 1
}
