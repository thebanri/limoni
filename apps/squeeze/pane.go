package main

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// pane draws a racer's screen. With cursor set it marks where the agent's
// cursor is, in reverse video, for typing into it.
type pane struct {
	r      *racer
	cursor bool
}

func (p pane) Draw(ctx cell.Context, buf *buffer.Buffer) {
	a := ctx.Area
	w, h := min(int(a.Width), p.r.emu.Width()), min(int(a.Height), p.r.emu.Height())
	for y := range h {
		for x := range w {
			buf.SetCell(a.X+uint16(x), a.Y+uint16(y), convertCell(p.r.emu.CellAt(x, y)))
		}
	}
	if p.cursor {
		pos := p.r.emu.CursorPosition()
		if c := buf.Get(a.X+uint16(pos.X), a.Y+uint16(pos.Y)); c != nil && pos.X < w && pos.Y < h {
			c.Style.Modifier |= cell.ModifierReverse
		}
	}
}

func (p pane) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, maxArea.Height }
