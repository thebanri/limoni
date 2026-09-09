package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

type hDividerComponent struct {
	symbol rune
	style  cell.Style
	title  string
}

// Divider creates a horizontal rule filling 100% of the available width with a single row height.
// By default, it uses the horizontal line rune '─'.
func Divider(style ...cell.Style) Component {
	st := cell.NewStyle()
	if len(style) > 0 {
		st = style[0]
	}
	return &hDividerComponent{symbol: '─', style: st}
}

// DividerCustom creates a horizontal rule with a custom line rune.
func DividerCustom(symbol rune, style ...cell.Style) Component {
	st := cell.NewStyle()
	if len(style) > 0 {
		st = style[0]
	}
	return &hDividerComponent{symbol: symbol, style: st}
}

// DividerWithTitle creates a horizontal rule with a title centered in the divider.
func DividerWithTitle(title string, symbols widgets.BorderSymbols, style cell.Style) Component {
	sym := symbols.Horizontal
	if sym == 0 {
		sym = '─'
	}
	return &hDividerComponent{symbol: sym, style: style, title: title}
}

func (d *hDividerComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	finalStyle := ctx.Style.Merge(d.style)
	y := area.Y

	if d.title == "" {
		for x := area.X; x < area.X+area.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: d.symbol, Style: finalStyle})
		}
		return
	}

	// Title divider: "── Title ──────"
	titleWidth := uint16(cell.StringWidth(d.title))
	totalSpace := area.Width
	if totalSpace <= titleWidth+2 {
		// Not enough space for decoration, just draw title
		col := uint16(0)
		for _, r := range d.title {
			rw := uint16(cell.RuneWidth(r))
			if col+rw > area.Width {
				break
			}
			buf.SetCell(area.X+col, y, cell.Cell{Content: r, Style: finalStyle})
			col += rw
		}
		return
	}

	leftBarLen := uint16(2)
	rightBarStart := area.X + leftBarLen + titleWidth + 2

	// Left bar: "── "
	for x := area.X; x < area.X+leftBarLen; x++ {
		buf.SetCell(x, y, cell.Cell{Content: d.symbol, Style: finalStyle})
	}
	buf.SetCell(area.X+leftBarLen, y, cell.Cell{Content: ' ', Style: finalStyle})

	// Title
	col := uint16(0)
	titleStart := area.X + leftBarLen + 1
	for _, r := range d.title {
		rw := uint16(cell.RuneWidth(r))
		buf.SetCell(titleStart+col, y, cell.Cell{Content: r, Style: finalStyle})
		col += rw
	}
	buf.SetCell(titleStart+col, y, cell.Cell{Content: ' ', Style: finalStyle})

	// Right bar: " ──────"
	for x := rightBarStart; x < area.X+area.Width; x++ {
		buf.SetCell(x, y, cell.Cell{Content: d.symbol, Style: finalStyle})
	}
}

func (d *hDividerComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	minW := uint16(1)
	if d.title != "" {
		minW = uint16(cell.StringWidth(d.title)) + 4
	}
	return LayoutProps{
		MinWidth:  minW,
		MinHeight: 1,
		MaxHeight: 1,
		Flex:      1, // Stretches across horizontal axis
	}
}

func (d *hDividerComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return d.LayoutInfo(maxArea).MinWidth, 1
}

func (d *hDividerComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return false
}

type vDividerComponent struct {
	symbol rune
	style  cell.Style
}

// VDivider creates a vertical rule filling 100% of the available height with a single column width.
// By default, it uses the vertical line rune '│'.
func VDivider(style ...cell.Style) Component {
	st := cell.NewStyle()
	if len(style) > 0 {
		st = style[0]
	}
	return &vDividerComponent{symbol: '│', style: st}
}

// VDividerCustom creates a vertical rule with a custom line rune.
func VDividerCustom(symbol rune, style ...cell.Style) Component {
	st := cell.NewStyle()
	if len(style) > 0 {
		st = style[0]
	}
	return &vDividerComponent{symbol: symbol, style: st}
}

func (v *vDividerComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	finalStyle := ctx.Style.Merge(v.style)
	x := area.X
	for y := area.Y; y < area.Y+area.Height; y++ {
		buf.SetCell(x, y, cell.Cell{Content: v.symbol, Style: finalStyle})
	}
}

func (v *vDividerComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	return LayoutProps{
		MinWidth:  1,
		MinHeight: 1,
		MaxWidth:  1,
		Flex:      1, // Stretches across vertical axis
	}
}

func (v *vDividerComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return 1, 1
}

func (v *vDividerComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return false
}
