package main

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

const (
	damping  = 0.95 // how much of a wave survives a step
	speed    = 0.25 // the wave speed squared: half a pixel a step; at 0.5, the stable limit, the water rings like a bell
	calm     = 0.05 // a height closer to zero than this is still water
	rainRate = 25   // a drop falls on a step when the pool's first byte is below this: ~10%
	shine    = 4    // palette steps per unit of slope
)

// pond is a puddle under rain. The water is a height field stepped with the
// two-buffer wave equation, drawn in half blocks: a pixel per column, two
// per row. Drops land where the entropy pool says, so the rain is the pool
// made visible.
type pond struct {
	w, h         int // in pixels
	cur, prev    []float32
	pool         [32]byte // SHA-256 of everything stirred in so far
	stirs, drops uint64
	theme        int
	in           [32 + 8 + 4]byte
}

func newPond(seed int64) *pond {
	p := &pond{}
	p.stir(seed, 0, 0)
	return p
}

// resize allocates, and only when the window changes size.
func (p *pond) resize(w, h int) {
	if w == p.w && h == p.h {
		return
	}
	p.w, p.h = w, h
	p.cur, p.prev = make([]float32, w*h), make([]float32, w*h)
}

// stir mixes one observation into the pool: a timestamp, whose low bits are
// scheduling jitter no one can predict, and where it happened. The water is
// never fed back in: it is computed from the pool, so it adds nothing.
func (p *pond) stir(ns int64, a, b uint16) {
	copy(p.in[:32], p.pool[:])
	binary.LittleEndian.PutUint64(p.in[32:], uint64(ns))
	binary.LittleEndian.PutUint16(p.in[40:], a)
	binary.LittleEndian.PutUint16(p.in[42:], b)
	p.pool = sha256.Sum256(p.in[:])
	p.stirs++
}

// drop pushes the water down in a 3×3 patch around (x, y).
func (p *pond) drop(x, y int, depth float32) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if xx, yy := x+dx, y+dy; xx > 0 && yy > 0 && xx < p.w-1 && yy < p.h-1 {
				p.cur[yy*p.w+xx] -= depth
			}
		}
	}
	p.drops++
}

// step lets the pool decide whether and where a drop falls, then moves the
// waves on by one step.
func (p *pond) step() {
	if p.w < 3 || p.h < 3 {
		return
	}
	if p.pool[0] < rainRate {
		x := int(binary.LittleEndian.Uint16(p.pool[1:]))%(p.w-2) + 1
		y := int(binary.LittleEndian.Uint16(p.pool[3:]))%(p.h-2) + 1
		p.drop(x, y, 2+float32(p.pool[5]%5))
	}
	w, cur, prev := p.w, p.cur, p.prev
	for y := 1; y < p.h-1; y++ {
		for i := y*w + 1; i < y*w+w-1; i++ {
			v := (2*cur[i] - prev[i] + speed*(cur[i-1]+cur[i+1]+cur[i-w]+cur[i+w]-4*cur[i])) * damping
			if v < calm && v > -calm {
				v = 0 // still water stays still, and the diff has nothing to send
			}
			prev[i] = v
		}
	}
	p.cur, p.prev = prev, cur
}

// A theme is four colours, deep water to sky reflection to glint, spread
// over a palette of 40 at init.
type theme struct {
	name  string
	stops [4]uint32
}

var themes = [...]theme{
	{"ocean", [4]uint32{0x06121f, 0x1d4e6b, 0x6fb6d6, 0xeaf8ff}},
	{"lemon", [4]uint32{0x1f1800, 0x8a6d00, 0xf5d742, 0xfffbe0}},
	{"dusk", [4]uint32{0x120a24, 0x4a2560, 0xe0709a, 0xfff0e8}},
	{"moss", [4]uint32{0x07140c, 0x1f4d2c, 0x7fd08a, 0xf0fff2}},
	{"ink", [4]uint32{0x0a0a0a, 0x363636, 0xa8a8a8, 0xffffff}},
}

var palettes [len(themes)][40]cell.Color

func init() {
	at := [4]int{0, 20, 32, 39}
	for t, th := range themes {
		for s := 1; s < 4; s++ {
			a, b := th.stops[s-1], th.stops[s]
			for i := at[s-1]; i <= at[s]; i++ {
				f := float64(i-at[s-1]) / float64(at[s]-at[s-1])
				lerp := func(shift int) uint8 {
					x, y := float64(a>>shift&0xff), float64(b>>shift&0xff)
					return uint8(x + (y-x)*f)
				}
				palettes[t][i] = cell.NewColorRGB(lerp(16), lerp(8), lerp(0))
			}
		}
	}
}

// themeNamed returns the index of the theme called name, or -1.
func themeNamed(name string) int {
	for i, th := range themes {
		if th.name == name {
			return i
		}
	}
	return -1
}

// colour shades a pixel by its slope, lit from the top left, over water
// that is lighter towards the top, where it reflects more sky.
func (p *pond) colour(x, y int) cell.Color {
	k := 18 - 8*y/p.h
	if x > 0 && y > 0 && x < p.w-1 && y < p.h-1 {
		i, w := y*p.w+x, p.w
		k += int((p.cur[i-1] - p.cur[i+1] + p.cur[i-w] - p.cur[i+w]) * shine)
	}
	pal := &palettes[p.theme]
	return pal[max(0, min(len(pal)-1, k))]
}

var (
	labelStyle = cell.Style{Fg: cell.NewColorRGB(0x8a, 0x9b, 0xa8), Bg: cell.NewColorRGB(0x02, 0x08, 0x10)}
	poolStyle  = cell.Style{Fg: cell.NewColorRGB(0xf5, 0xd7, 0x42), Bg: labelStyle.Bg}
)

// render draws the pond above a status line. It allocates nothing once the
// pond has the window's size.
func (p *pond) render(b *buffer.Buffer) {
	W, H := int(b.Area.Width), int(b.Area.Height)
	if W < 1 || H < 2 || len(b.Content) < W*H {
		return
	}
	b.IsDirty = true
	p.resize(W, 2*(H-1))
	for r := 0; r < H-1; r++ {
		out := b.Content[r*W : r*W+W]
		for x := range out {
			top, bot := p.colour(x, 2*r), p.colour(x, 2*r+1)
			if top == bot {
				out[x] = cell.Cell{Content: ' ', Style: cell.Style{Fg: bot, Bg: bot}}
			} else {
				out[x] = cell.Cell{Content: '▀', Style: cell.Style{Fg: top, Bg: bot}}
			}
		}
	}
	line := b.Content[(H-1)*W : H*W]
	for i := range line {
		line[i] = cell.Cell{Content: ' ', Style: labelStyle}
	}
	x := put(line, 1, "pool ", labelStyle)
	const hexDigits = "0123456789abcdef"
	for _, c := range p.pool {
		x = put(line, x, hexDigits[c>>4:c>>4+1], poolStyle)
		x = put(line, x, hexDigits[c&15:c&15+1], poolStyle)
	}
	x = put(line, x, "  stirs ", labelStyle)
	x = putNumber(line, x, p.stirs, labelStyle)
	x = put(line, x, "  drops ", labelStyle)
	x = putNumber(line, x, p.drops, labelStyle)
	x = put(line, x, "  · ", labelStyle)
	x = put(line, x, themes[p.theme].name, poolStyle)
	put(line, x, " · click: a drop · t: colours · q: quit", labelStyle)
}

func put(line []cell.Cell, x int, s string, st cell.Style) int {
	for _, r := range s {
		if x < len(line) {
			line[x] = cell.Cell{Content: r, Style: st}
		}
		x++
	}
	return x
}

func putNumber(line []cell.Cell, x int, n uint64, st cell.Style) int {
	var d [20]byte
	i := len(d)
	for {
		i--
		d[i] = byte('0' + n%10)
		if n /= 10; n == 0 {
			break
		}
	}
	for _, c := range d[i:] {
		if x < len(line) {
			line[x] = cell.Cell{Content: rune(c), Style: st}
		}
		x++
	}
	return x
}
