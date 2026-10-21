package main

import (
	"runtime"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// ── colour ───────────────────────────────────────────────────────────────

// rgb is a colour, 0…1 a channel.
type rgb struct{ r, g, b float32 }

func (c rgb) mul(k float32) rgb { return rgb{c.r * k, c.g * k, c.b * k} }
func (c rgb) add(o rgb) rgb     { return rgb{c.r + o.r, c.g + o.g, c.b + o.b} }
func (c rgb) mix(o rgb, t float32) rgb {
	return rgb{c.r + (o.r-c.r)*t, c.g + (o.g-c.g)*t, c.b + (o.b-c.b)*t}
}

// grey is the colour with its saturation taken out by t: the collector's
// pause drains the picture.
func (c rgb) grey(t float32) rgb {
	l := c.r*0.3 + c.g*0.59 + c.b*0.11
	return c.mix(rgb{l, l, l}, t)
}

func hex(v uint32) rgb {
	return rgb{float32(v>>16&255) / 255, float32(v>>8&255) / 255, float32(v&255) / 255}
}

func to8(v float32) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return uint8(v*255 + 0.5)
}

// coarse rounds colours harder in the browser, as Castle Lemonstein does:
// xterm.js's WebGL renderer keeps a glyph per pair of colours in a texture
// atlas, and free colours fill it.
const coarse = runtime.GOOS == "js"

func round(c uint8) uint8 {
	switch {
	case !coarse || c < 48:
		return c &^ 7
	case c < 112:
		return c &^ 15
	}
	return c &^ 31
}

func (c rgb) color() cell.Color {
	return cell.NewColorRGB(round(to8(c.r)), round(to8(c.g)), round(to8(c.b)))
}

func style(fg, bg rgb) cell.Style { return cell.Style{Fg: fg.color(), Bg: bg.color()} }

// hash2 is a repeatable pseudo-random number in 0…1 for a pair of ints.
func hash2(a, b int) float32 {
	h := uint32(a)*374761393 + uint32(b)*668265263
	h = (h ^ h>>13) * 1274126177
	h ^= h >> 16
	return float32(h&0xffff) / 0xffff
}

// ── cells ────────────────────────────────────────────────────────────────

// canvas writes cells straight into the frame's buffer. Nothing on it is
// bold or dim: a modifier switched off makes the diff build a style through
// its cache, and free colours would fill the cache (Castle Lemonstein found
// that out).
type canvas struct {
	b    *buffer.Buffer
	w, h int
}

func (c canvas) set(x, y int, r rune, s cell.Style) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.b.Content[y*c.w+x] = cell.Cell{Content: r, Style: s}
}

func (c canvas) at(x, y int) *cell.Cell {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return nil
	}
	return &c.b.Content[y*c.w+x]
}

// text writes s from (x, y) and returns the column after it. Ranging over a
// string decodes it in place: nothing is allocated.
func (c canvas) text(x, y int, s string, st cell.Style) int {
	for _, r := range s {
		c.set(x, y, r, st)
		x++
	}
	return x
}

// fg writes s keeping the background already there.
func (c canvas) fg(x, y int, s string, fg rgb) int {
	col := fg.color()
	for _, r := range s {
		if p := c.at(x, y); p != nil {
			p.Content, p.Style.Fg = r, col
		}
		x++
	}
	return x
}

// number writes n in decimal without strconv, which would allocate.
func (c canvas) number(x, y, n int, st cell.Style) int {
	if n < 0 {
		c.set(x, y, '-', st)
		x++
		n = -n
	}
	d := digits(n)
	for i := d - 1; i >= 0; i-- {
		c.set(x+i, y, rune('0'+n%10), st)
		n /= 10
	}
	return x + d
}

func digits(n int) int {
	d := 1
	for p := 10; p <= n; p *= 10 {
		d++
	}
	return d
}

// clock writes seconds as m:ss.
func (c canvas) clock(x, y, secs int, st cell.Style) int {
	x = c.number(x, y, secs/60, st)
	c.set(x, y, ':', st)
	c.set(x+1, y, rune('0'+secs%60/10), st)
	c.set(x+2, y, rune('0'+secs%10), st)
	return x + 3
}

func (c canvas) fill(x0, y0, w, h int, r rune, st cell.Style) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			c.set(x, y, r, st)
		}
	}
}

func width(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func (c canvas) centered(y int, s string, st cell.Style) {
	c.text((c.w-width(s))/2, y, s, st)
}

// box draws a rounded frame, the way a Bubble Tea app would.
func (c canvas) box(x0, y0, w, h int, st cell.Style) {
	for x := x0 + 1; x < x0+w-1; x++ {
		c.set(x, y0, '─', st)
		c.set(x, y0+h-1, '─', st)
	}
	for y := y0 + 1; y < y0+h-1; y++ {
		c.set(x0, y, '│', st)
		c.set(x0+w-1, y, '│', st)
	}
	c.set(x0, y0, '╭', st)
	c.set(x0+w-1, y0, '╮', st)
	c.set(x0, y0+h-1, '╰', st)
	c.set(x0+w-1, y0+h-1, '╯', st)
}

// ── art ──────────────────────────────────────────────────────────────────

// art is a picture made of characters: every enemy is drawn in the idiom
// of the framework it comes from, and scaled with the distance by picking
// the nearest character.
type art struct {
	w, h  int
	glyph []rune
	col   []rgb
	solid []bool // inside the outline: filled with bg even where blank
	bg    rgb    // a panel colour; black leaves the art see-through
}

// newArt builds art from rows of text and, row for row, a key naming each
// character's colour in pal. A space is transparent.
func newArt(rows, key []string, pal map[byte]uint32) *art {
	a := &art{h: len(rows)}
	for _, r := range rows {
		if n := width(r); n > a.w {
			a.w = n
		}
	}
	a.glyph = make([]rune, a.w*a.h)
	a.col = make([]rgb, a.w*a.h)
	for y, row := range rows {
		x := 0
		for _, r := range row {
			a.glyph[y*a.w+x] = r
			k := byte(' ')
			if y < len(key) && x < len(key[y]) {
				k = key[y][x]
			}
			a.col[y*a.w+x] = hex(pal[k])
			x++
		}
		for ; x < a.w; x++ {
			a.glyph[y*a.w+x] = ' '
		}
	}
	a.solid = make([]bool, a.w*a.h)
	for y := 0; y < a.h; y++ {
		first, last := -1, -1
		for x := 0; x < a.w; x++ {
			if a.glyph[y*a.w+x] != ' ' {
				if first < 0 {
					first = x
				}
				last = x
			}
		}
		for x := first; x >= 0 && x <= last; x++ {
			a.solid[y*a.w+x] = true
		}
	}
	return a
}

// panel gives every frame of an animation a background colour.
func panel(frames []*art, bg uint32) []*art {
	for _, a := range frames {
		a.bg = hex(bg)
	}
	return frames
}
