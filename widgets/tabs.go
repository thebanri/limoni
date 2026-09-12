package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// DefaultTabDivider separates adjacent tab titles.
const DefaultTabDivider = "│"

// Tabs renders a single-line horizontal tab bar and reports clicks.
//
// It owns no selection state: pass the selected index in and update it from
// OnSelect, so the tab bar composes with whatever state model the application
// already uses.
//
//	widgets.Tabs{
//		Titles:   []string{"Home", "Charts", "Settings"},
//		Selected: app.tab,
//		OnSelect: func(i int) { app.tab = i },
//	}
type Tabs struct {
	// ID registers the tab bar with the focus manager when set.
	ID string

	// Titles are the tab labels, rendered left to right.
	Titles []string

	// Selected is the index of the active tab. Out-of-range values select none.
	Selected int

	// Style applies to inactive tabs and the bar background.
	Style cell.Style
	// SelectedStyle applies to the active tab. Defaults to the theme accent.
	SelectedStyle cell.Style
	// DividerStyle applies to the separator between tabs.
	DividerStyle cell.Style

	// Divider separates adjacent titles. Defaults to DefaultTabDivider.
	// Set it to an empty string to remove separators entirely.
	Divider *string

	// Padding is the number of spaces added on each side of a title.
	// Defaults to one; set PaddingZero to remove it.
	Padding uint16
	// PaddingZero renders titles with no surrounding spaces.
	PaddingZero bool

	// OnSelect is invoked with the tab index when a tab is clicked.
	OnSelect func(index int)
}

// divider resolves the configured separator.
func (t Tabs) divider() string {
	if t.Divider != nil {
		return *t.Divider
	}
	return DefaultTabDivider
}

// padding resolves the configured horizontal padding.
func (t Tabs) padding() uint16 {
	if t.PaddingZero {
		return 0
	}
	if t.Padding == 0 {
		return 1
	}
	return t.Padding
}

// Width reports the total rendered width of the tab bar, including padding and
// dividers. Use it to size a fixed-width column or to decide whether the bar
// fits before rendering it.
func (t Tabs) Width() uint16 {
	if len(t.Titles) == 0 {
		return 0
	}
	pad := int(t.padding())
	dividerWidth := cell.StringWidth(t.divider())

	total := 0
	for i, title := range t.Titles {
		if i > 0 {
			total += dividerWidth
		}
		total += pad + cell.StringWidth(title) + pad
	}
	if total > int(^uint16(0)) {
		return ^uint16(0)
	}
	return uint16(total)
}

// Draw renders the tab bar on the first row of the context area.
func (t Tabs) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 || len(t.Titles) == 0 {
		return
	}

	if t.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(t.ID)
	}

	base := ctx.Style.Merge(t.Style)

	selectedStyle := base.Merge(t.SelectedStyle)
	if t.SelectedStyle == (cell.Style{}) {
		if ctx.ThemeStyle != nil {
			selectedStyle = base.Merge(ctx.ThemeStyle("focus"))
		}
		selectedStyle.Modifier |= cell.ModifierBold
	}

	dividerStyle := base.Merge(t.DividerStyle)
	if t.DividerStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
		dividerStyle = base.Merge(ctx.ThemeStyle("muted"))
	}

	pad := t.padding()
	divider := t.divider()
	dividerWidth := uint16(cell.StringWidth(divider))

	x := area.X
	limit := area.X + area.Width

	for i, title := range t.Titles {
		if i > 0 && dividerWidth > 0 {
			if x+dividerWidth > limit {
				return
			}
			buf.SetStringWithin(x, area.Y, divider, dividerStyle, limit-x)
			x += dividerWidth
		}

		titleWidth := uint16(cell.StringWidth(title))
		tabWidth := pad + titleWidth + pad
		if x >= limit {
			return
		}

		style := base
		if i == t.Selected {
			style = selectedStyle
		}

		// Fill the whole tab cell range first so the selected background covers
		// the padding, not just the glyphs.
		for fill := x; fill < x+tabWidth && fill < limit; fill++ {
			if c := buf.Get(fill, area.Y); c != nil {
				c.Content = ' '
				c.Style = style
			}
		}

		textX := x + pad
		if textX < limit {
			buf.SetStringWithin(textX, area.Y, title, style, limit-textX)
		}

		if t.OnSelect != nil && ctx.RegisterClick != nil {
			clickWidth := tabWidth
			if x+clickWidth > limit {
				clickWidth = limit - x
			}
			index := i
			onSelect := t.OnSelect
			ctx.RegisterClick(cell.NewRect(x, area.Y, clickWidth, 1), func() {
				onSelect(index)
			})
		}

		x += tabWidth
	}
}

// SizeHint reports the bar's natural width and a height of one row.
func (t Tabs) SizeHint(maxArea cell.Rect) (width, height uint16) {
	w := t.Width()
	if w > maxArea.Width {
		w = maxArea.Width
	}
	return w, 1
}

// AccessibilityNode returns the semantic node description for Tabs.
func (t Tabs) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}

	value, position := "", 0
	if t.Selected >= 0 && t.Selected < len(t.Titles) {
		state |= accessibility.StateSelected
		value = t.Titles[t.Selected]
		position = t.Selected + 1
	}

	return accessibility.AccessibilityNode{
		ID:       t.ID,
		Role:     accessibility.RoleList,
		Label:    "Tabs",
		Value:    value,
		State:    state,
		Bounds:   bounds,
		Position: position,
		SetSize:  len(t.Titles),
	}
}
