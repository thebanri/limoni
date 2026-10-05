package main

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// canvas writes cells straight into a frame's buffer; the frame is filled
// edge to edge every time, so there is nothing underneath to merge with.
type canvas struct {
	b    *buffer.Buffer
	w, h int
}

func (c canvas) set(x, y int, r rune, s cell.Style) {
	if x >= 0 && y >= 0 && x < c.w && y < c.h {
		c.b.Content[y*c.w+x] = cell.Cell{Content: r, Style: s}
	}
}

// text writes s and returns the column after it.
func (c canvas) text(x, y int, s string, st cell.Style) int {
	if x < 0 || y < 0 || y >= c.h || x >= c.w {
		return x + cell.StringWidth(s)
	}
	return x + int(c.b.SetStringWithin(uint16(x), uint16(y), s, st, uint16(c.w-x)))
}

// textMax writes s in at most w columns, ending in '…' when it is cut,
// and returns how many columns it took.
func (c canvas) textMax(x, y int, s string, st cell.Style, w int) int {
	if w <= 0 || x < 0 || y < 0 || y >= c.h || x >= c.w {
		return 0
	}
	w = min(w, c.w-x)
	if cell.StringWidth(s) <= w {
		return int(c.b.SetStringWithin(uint16(x), uint16(y), s, st, uint16(w)))
	}
	n := int(c.b.SetStringWithin(uint16(x), uint16(y), s, st, uint16(w-1)))
	c.set(x+n, y, '…', st)
	return n + 1
}

// number writes n in decimal without allocating.
func (c canvas) number(x, y, n int, st cell.Style) int {
	var d [20]byte
	i := len(d)
	for {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	for _, ch := range d[i:] {
		c.set(x, y, rune(ch), st)
		x++
	}
	return x
}

// clock writes seconds as m:ss, or h:mm:ss from an hour.
func (c canvas) clock(x, y int, sec float64, st cell.Style) int {
	t := int(max(0, sec))
	h, m, s := t/3600, t/60%60, t%60
	if h > 0 {
		x = c.number(x, y, h, st)
		c.set(x, y, ':', st)
		x = c.two(x+1, y, m, st)
	} else {
		x = c.number(x, y, m, st)
	}
	c.set(x, y, ':', st)
	return c.two(x+1, y, s, st)
}

func (c canvas) two(x, y, n int, st cell.Style) int {
	c.set(x, y, rune('0'+n/10), st)
	c.set(x+1, y, rune('0'+n%10), st)
	return x + 2
}
