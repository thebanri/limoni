package main

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// label implements widgets.Widget for simple text display inside Blocks.
type label struct {
	text  string
	style cell.Style
}

func (l label) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	buf.SetStringWithin(ctx.Area.X, ctx.Area.Y, l.text, l.style, ctx.Area.Width)
}

func (l label) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return uint16(cell.StringWidth(l.text)), 1
}

// registerTargetClick registers a click handler for area.
// This guarantees that mouse clicks are dispatched on MouseLeft, without falsely capturing MouseNone hover events.
func registerTargetClick(f *terminal.Frame, area cell.Rect, handler func(backend.MouseEvent)) {
	if handler == nil || area.Width == 0 || area.Height == 0 {
		return
	}

	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button != backend.MouseLeft || ev.Drag {
			return
		}
		handler(ev)
	})
}

// renderClickButton renders a stylish rounded clickable button with hover/active state and registers click handler.
func renderClickButton(f *terminal.Frame, area cell.Rect, title string, active bool, primaryCol, mutedCol cell.Color, onClick func()) {
	borderCol := mutedCol
	titleStyle := cell.Style{Fg: mutedCol}

	if active {
		borderCol = primaryCol
		titleStyle = cell.Style{Fg: primaryCol, Modifier: cell.ModifierBold}
	}

	btn := widgets.Block{
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: borderCol},
		Title:          title,
		TitleAlignment: widgets.AlignCenter,
		TitleStyle:     titleStyle,
	}
	f.RenderWidget(btn, area)

	registerTargetClick(f, area, func(ev backend.MouseEvent) {
		if onClick != nil {
			onClick()
		}
	})
}

// drawBadge renders a colored badge at (x, y).
func drawBadge(buf *buffer.Buffer, x, y uint16, text string, fg, bg cell.Color) uint16 {
	style := cell.Style{Fg: fg, Bg: bg, Modifier: cell.ModifierBold}
	buf.SetString(x, y, text, style)
	return uint16(utf8.RuneCountInString(text))
}
