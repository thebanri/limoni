package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/layout"
)

// ListState is the state of a scrollable, selectable list.
type ListState struct {
	// Selected is the index of the selected item, or -1 if none is selected.
	Selected int
	// Offset is the index of the first item shown at the top of the list (the scroll offset).
	Offset int

	// rows is reused for the visible rows' semantic nodes on every frame, so
	// exposing them allocates only while the list grows taller.
	rows []accessibility.AccessibilityNode
}

// NewListState returns a new ListState. By default nothing is selected.
func NewListState() *ListState {
	return &ListState{
		Selected: -1,
		Offset:   0,
	}
}

// Select selects the given index.
func (s *ListState) Select(index int) {
	s.Selected = index
}

// Next moves the selection to the next item.
func (s *ListState) Next() {
	s.Selected++
}

// Previous moves the selection to the previous item.
func (s *ListState) Previous() {
	if s.Selected > 0 {
		s.Selected--
	} else {
		s.Selected = 0
	}
}

// ScrollTo keeps the selected item (Selected) always visible within the list's visible
// height. If the selected item goes off screen, it scrolls Offset to follow it.
//
// Parameters:
//   - height: the number of rows of the list visible on screen.
//   - total: the total number of items in the list.
func (s *ListState) ScrollTo(height int, total int) {
	if s.Selected < 0 || total == 0 || height <= 0 {
		s.Offset = 0
		return
	}

	// Clamp a selection that is out of range
	if s.Selected >= total {
		s.Selected = total - 1
	}

	// If the selection is above the view, scroll up
	if s.Selected < s.Offset {
		s.Offset = s.Selected
	}

	// If the selection is below the view, scroll down
	if s.Selected >= s.Offset+height {
		s.Offset = s.Selected - height + 1
	}

	// Bounds guards
	if s.Offset < 0 {
		s.Offset = 0
	}
	maxOffset := total - height
	if s.Offset > maxOffset {
		s.Offset = maxOffset
	}
	if s.Offset < 0 {
		s.Offset = 0
	}
}

// List is an interactive widget that draws items as a vertical list.
type List struct {
	// ID is the list's focus and identity ID.
	ID string
	// Label names the list in the semantic tree — what a screen reader
	// announces and what an automation selector matches. Defaults to "List".
	Label string
	// Items are the strings shown in the list.
	Items []string
	// Provider supplies the data for a virtual (virtually scrolled) list.
	// If set, it is used instead of Items.
	Provider ListProvider
	// Scrollbar, when set, draws a vertical scrollbar on the right edge of the list.
	Scrollbar bool
	// ScrollbarTrackStyle is the style of the scrollbar track.
	ScrollbarTrackStyle cell.Style
	// ScrollbarThumbStyle is the style of the scrollbar thumb.
	ScrollbarThumbStyle cell.Style
	// Style is the list's overall colour and text style.
	Style cell.Style
	// FocusedStyle is the style applied when the list has the focus.
	FocusedStyle cell.Style
	// SelectedStyle is the style the selected item is highlighted with.
	SelectedStyle cell.Style
	// HighlightSymbol is the symbol placed left of the selected item (e.g. "> ").
	HighlightSymbol string
	// HighlightSpacing keeps the symbol's width free on every row, so the
	// text of the selected row lines up with the rest instead of moving
	// right when it is selected.
	HighlightSpacing bool

	// State points to the list's selected index and scroll state.
	State *ListState
}

// NewList creates a new List widget with the given items.
func NewList(items ...string) *List {
	return &List{
		Items: items,
	}
}

// WithID sets the widget focus and event ID.
func (l *List) WithID(id string) *List {
	l.ID = id
	return l
}

// WithItems sets the items of the list.
func (l *List) WithItems(items ...string) *List {
	l.Items = items
	return l
}

// WithProvider sets a virtual data provider.
func (l *List) WithProvider(p ListProvider) *List {
	l.Provider = p
	return l
}

// WithState sets the ListState pointer.
func (l *List) WithState(state *ListState) *List {
	l.State = state
	return l
}

// WithHighlightSymbol sets the prefix symbol for the selected item.
func (l *List) WithHighlightSymbol(sym string) *List {
	l.HighlightSymbol = sym
	return l
}

// WithScrollbar enables or disables the scrollbar.
func (l *List) WithScrollbar(enable bool) *List {
	l.Scrollbar = enable
	return l
}

// WithStyle sets the default list style.
func (l *List) WithStyle(style cell.Style) *List {
	l.Style = style
	return l
}

// WithSelectedStyle sets the selected item style.
func (l *List) WithSelectedStyle(style cell.Style) *List {
	l.SelectedStyle = style
	return l
}

// WithFocusedStyle sets the focused list style.
func (l *List) WithFocusedStyle(style cell.Style) *List {
	l.FocusedStyle = style
	return l
}

// Draw draws the list into the area. It works out the visible items, highlights the selected
// one and registers a mouse click region (RegisterClick) for each item.
func (l List) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	totalItems := len(l.Items)
	if l.Provider != nil {
		totalItems = l.Provider.Len()
	}
	if area.Width == 0 || area.Height == 0 || totalItems == 0 {
		return
	}

	if l.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(l.ID)
	}

	listStyle := ctx.Style.Merge(l.Style)
	if listStyle.Bg.Type() == cell.ColorDefault && ctx.ThemeStyle != nil {
		listStyle = listStyle.Merge(ctx.ThemeStyle("surface"))
	}
	if ctx.IsFocused(l.ID) {
		listStyle = listStyle.Merge(l.FocusedStyle)
	}
	styler, _ := l.Provider.(ListStyler)

	selected := -1
	offset := 0
	if l.State != nil {
		l.State.ScrollTo(int(area.Height), totalItems)
		selected = l.State.Selected
		offset = l.State.Offset
	}

	// The mouse wheel scrolls the list.
	if ctx.RegisterScroll != nil && l.State != nil {
		maxOffset := totalItems - int(ctx.Area.Height)
		if maxOffset < 0 {
			maxOffset = 0
		}
		ctx.RegisterScroll(ctx.Area, &l.State.Offset, maxOffset)
	} else if ctx.RegisterMouse != nil && l.State != nil {
		st := l.State
		viewHeight := int(ctx.Area.Height)
		ctx.RegisterMouse(ctx.Area, func(ev driver.MouseEvent) {
			if ev.Button == driver.MouseScrollUp {
				st.Offset--
				if st.Offset < 0 {
					st.Offset = 0
				}
			} else if ev.Button == driver.MouseScrollDown {
				st.Offset++
				maxOffset := totalItems - viewHeight
				if maxOffset < 0 {
					maxOffset = 0
				}
				if st.Offset > maxOffset {
					st.Offset = maxOffset
				}
			}
		})
	}

	// Draw the vertical scrollbar
	visibleHeight := int(area.Height)
	if l.Scrollbar && totalItems > visibleHeight && area.Width > 1 {
		scrollbarX := area.X + area.Width - 1
		thumbH := (visibleHeight * visibleHeight) / totalItems
		if thumbH < 1 {
			thumbH = 1
		}
		maxOffset := totalItems - visibleHeight
		thumbY := 0
		if maxOffset > 0 {
			thumbY = (offset * (visibleHeight - thumbH)) / maxOffset
		}

		trackStyle := listStyle.Merge(l.ScrollbarTrackStyle)
		if l.ScrollbarTrackStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
			trackStyle = listStyle.Merge(ctx.ThemeStyle("muted"))
		}
		thumbStyle := listStyle.Merge(l.ScrollbarThumbStyle)
		if l.ScrollbarThumbStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
			thumbStyle = listStyle.Merge(ctx.ThemeStyle("focus"))
		}

		for y := 0; y < visibleHeight; y++ {
			c := buf.Get(scrollbarX, area.Y+uint16(y))
			if c != nil {
				c.Content = '░'
				c.Style = c.Style.Merge(trackStyle)
				if y >= thumbY && y < thumbY+thumbH {
					c.Content = '█'
					c.Style = c.Style.Merge(thumbStyle)
				}
			}
		}
		// Take the track out of the text area
		area.Width--
	}

	for i := 0; i < int(area.Height); i++ {
		itemIdx := offset + i
		if itemIdx >= totalItems {
			break
		}

		currY := area.Y + uint16(i)
		itemText := ""
		if l.Provider != nil {
			itemText = l.Provider.ItemAt(itemIdx)
		} else {
			itemText = l.Items[itemIdx]
		}

		isSel := itemIdx == selected
		itemStyle := listStyle
		if styler != nil {
			itemStyle = itemStyle.Merge(styler.StyleAt(itemIdx))
		}
		if isSel {
			itemStyle = itemStyle.Merge(l.SelectedStyle)
		}

		// Clear and fill the row's background
		for x := area.X; x < area.X+area.Width; x++ {
			c := buf.Get(x, currY)
			if c == nil {
				continue
			}
			rowStyle := itemStyle
			if rowStyle.Bg.Type() == cell.ColorDefault {
				if ctx.Style.Bg.Type() != cell.ColorDefault {
					rowStyle.Bg = ctx.Style.Bg
				} else if c.Style.Bg.Type() != cell.ColorDefault {
					rowStyle.Bg = c.Style.Bg
				}
			}
			c.Content = ' '
			c.Style = rowStyle
		}

		// Draw the text (allocation-free string rendering)
		textX := area.X
		rightLimit := area.X + area.Width
		if l.HighlightSymbol != "" && (isSel || l.HighlightSpacing) {
			symWidth := uint16(cell.StringWidth(l.HighlightSymbol))
			if isSel && textX < rightLimit {
				buf.SetStringWithin(textX, currY, l.HighlightSymbol, itemStyle, rightLimit-textX)
			}
			textX += symWidth
		}
		if textX < rightLimit {
			buf.SetStringWithin(textX, currY, itemText, itemStyle, rightLimit-textX)
		}

		// Hook up the automatic mouse routing
		if ctx.RegisterClickAction != nil && l.State != nil {
			// Clicking a row selects it and focuses the list, as data.
			ctx.RegisterClickAction(cell.Rect{X: area.X, Y: currY, Width: area.Width, Height: 1},
				cell.ClickAction{Focus: l.ID, Select: &l.State.Selected, Index: itemIdx})
		} else if ctx.RegisterClick != nil && l.State != nil {
			st := l.State
			id := l.ID
			setFocus := ctx.SetFocus
			targetIdx := itemIdx
			itemRect := cell.Rect{
				X:      area.X,
				Y:      currY,
				Width:  area.Width,
				Height: 1,
			}
			// When the item is clicked, select this index (Selected) and take the focus
			ctx.RegisterClick(itemRect, func() {
				st.Selected = targetIdx
				if id != "" && setFocus != nil {
					setFocus(id)
				}
			})
		}
	}

	// Fill the remaining empty rows with the background colour
	for y := totalItems - offset; y < int(area.Height); y++ {
		if y < 0 {
			continue
		}
		currY := area.Y + uint16(y)
		for x := area.X; x < area.X+area.Width; x++ {
			c := buf.Get(x, currY)
			if c != nil {
				bg := listStyle.Bg
				if bg.Type() == cell.ColorDefault {
					if ctx.Style.Bg.Type() != cell.ColorDefault {
						bg = ctx.Style.Bg
					} else if c.Style.Bg.Type() != cell.ColorDefault {
						bg = c.Style.Bg
					}
				}
				if bg.Type() != cell.ColorDefault {
					c.Content = ' '
					c.Style.Bg = bg
				}
			}
		}
	}
}

// SizeHint returns the ideal size from the longest item and the number of items.
func (l List) SizeHint(maxArea cell.Rect) (width, height uint16) {
	totalItems := len(l.Items)
	if l.Provider != nil {
		totalItems = l.Provider.Len()
	}
	if totalItems == 0 {
		return 0, 0
	}

	symbolLen := cell.StringWidth(l.HighlightSymbol)
	maxW := 0

	if l.Provider != nil {
		limit := totalItems
		if limit > 100 {
			limit = 100
		}
		for i := 0; i < limit; i++ {
			w := cell.StringWidth(l.Provider.ItemAt(i)) + symbolLen
			if w > maxW {
				maxW = w
			}
		}
	} else {
		for _, item := range l.Items {
			w := cell.StringWidth(item) + symbolLen
			if w > maxW {
				maxW = w
			}
		}
	}

	if l.Scrollbar && totalItems > int(maxArea.Height) {
		maxW++
	}

	w := uint16(maxW)
	h := uint16(totalItems)

	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}

	return w, h
}

// Measure provides explicit size negotiation for List.
func (l List) Measure(maxArea cell.Rect) layout.Measure {
	w, h := l.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}

// AccessibilityNode returns the semantic node description for List.
//
// The node carries the selected item as its value and one child per visible
// row, so a screen reader can announce "3 of 20" and an agent or a test can
// address a row by its label and click it. Only visible rows are included: a
// virtual list of a million items exposes the dozen on screen.
//
// Row nodes are written into a buffer owned by State and reused on every
// frame, so building them does not allocate once the list has been drawn at
// its height. A list without State has no scroll position to report rows
// against and stays flat. Consumers that keep a tree past the frame must copy
// it; Frame.AccessibilityTree does.
func (l List) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}

	count := len(l.Items)
	if l.Provider != nil {
		count = l.Provider.Len()
	}

	selected := -1
	if l.State != nil {
		selected = l.State.Selected
	}

	value := ""
	position := 0
	if selected >= 0 && selected < count {
		state |= accessibility.StateSelected
		value = l.itemAt(selected)
		position = selected + 1
	}

	label := l.Label
	if label == "" {
		label = "List"
	}

	var rows []accessibility.AccessibilityNode
	if l.State != nil && bounds.Width > 0 {
		rowWidth := bounds.Width
		// Draw takes the scrollbar column from the text area; the rows do too.
		if l.Scrollbar && count > int(bounds.Height) && bounds.Width > 1 {
			rowWidth--
		}
		rows = l.State.rows[:0]
		for i := 0; i < int(bounds.Height); i++ {
			index := l.State.Offset + i
			if index < 0 || index >= count {
				break
			}
			rowState := accessibility.NodeState(0)
			if index == selected {
				rowState = accessibility.StateSelected
			}
			rows = append(rows, accessibility.AccessibilityNode{
				Role:     accessibility.RoleListItem,
				Label:    l.itemAt(index),
				State:    rowState,
				Bounds:   cell.Rect{X: bounds.X, Y: bounds.Y + uint16(i), Width: rowWidth, Height: 1},
				Position: index + 1,
				SetSize:  count,
			})
		}
		l.State.rows = rows
	}

	return accessibility.AccessibilityNode{
		ID:       l.ID,
		Role:     accessibility.RoleList,
		Label:    label,
		Value:    value,
		State:    state,
		Bounds:   bounds,
		Position: position,
		SetSize:  count,
		Children: rows,
	}
}

func (l List) itemAt(index int) string {
	if l.Provider != nil {
		return l.Provider.ItemAt(index)
	}
	return l.Items[index]
}
