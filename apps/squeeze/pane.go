package main

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// pane draws a racer's screen. With cursor set it marks where the agent's
// cursor is, in reverse video, for typing into it.
type pane struct {
	r      *racer
	cursor bool
	sel    *selection // drawn in reverse video where it is r's
}

func (p pane) Draw(ctx cell.Context, buf *buffer.Buffer) {
	a := ctx.Area
	w, h := min(int(a.Width), p.r.emu.Width()), min(int(a.Height), p.r.emu.Height())
	// Scrolled back: the last scroll lines of the scrollback come first.
	sb := p.r.emu.ScrollbackLen()
	top := sb - min(p.r.scroll, sb)
	for y := range h {
		for x := range w {
			var c *uv.Cell
			if row := top + y; row < sb {
				c = p.r.emu.ScrollbackCellAt(x, row)
			} else {
				c = p.r.emu.CellAt(x, row-sb)
			}
			cl := convertCell(c)
			if p.sel != nil && p.sel.r == p.r && p.sel.has(top+y, x) {
				cl.Style.Modifier |= cell.ModifierReverse
			}
			buf.SetCell(a.X+uint16(x), a.Y+uint16(y), cl)
		}
	}
	if p.cursor && p.r.scroll == 0 {
		pos := p.r.emu.CursorPosition()
		if c := buf.Get(a.X+uint16(pos.X), a.Y+uint16(pos.Y)); c != nil && pos.X < w && pos.Y < h {
			c.Style.Modifier |= cell.ModifierReverse
		}
	}
}

func (p pane) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, maxArea.Height }
