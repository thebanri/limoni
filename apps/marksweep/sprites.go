package main

import "math"

// The enemies are pixel creatures, drawn in half blocks like the room:
// each framework's idea turned into something alive. Bubble Tea's spinner
// is an orb its dots orbit, its cup a jelly with a straw and legs; Ink's
// </> a creature whose jaws are the brackets; Ratatui a crab with a rat's
// ears and tail; tview's modal a mimic whose title bar opens like a lid and
// whose buttons are its eyes; Bubbles' progress bar a serpent; ncurses an
// old phosphor monitor on legs. They are painted in code, once, and every
// one gets a dark outline so it reads against any wall.

// sprite is one frame: a pixel picture, transparent where off.
type sprite struct {
	w, h int
	px   []rgb
	on   []bool
}

// paint is a sprite being drawn.
type paint struct{ sprite }

func newPaint(w, h int) *paint {
	return &paint{sprite{w: w, h: h, px: make([]rgb, w*h), on: make([]bool, w*h)}}
}

func (p *paint) dot(x, y int, c rgb) {
	if x >= 0 && y >= 0 && x < p.w && y < p.h {
		p.px[y*p.w+x], p.on[y*p.w+x] = c, true
	}
}

func (p *paint) rect(x0, y0, x1, y1 int, c rgb) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			p.dot(x, y, c)
		}
	}
}

// disc fills an ellipse, shading it from a lighter top left.
func (p *paint) disc(cx, cy, rx, ry float64, c rgb) {
	for y := 0; y < p.h; y++ {
		for x := 0; x < p.w; x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 {
				p.dot(x, y, c.mul(float32(1.05-0.25*(dx+dy+1)/2)))
			}
		}
	}
}

// line draws from (x0, y0) to (x1, y1), t pixels thick.
func (p *paint) line(x0, y0, x1, y1 float64, t int, c rgb) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= n; i++ {
		k := float64(i) / float64(n)
		x, y := int(math.Round(x0+(x1-x0)*k)), int(math.Round(y0+(y1-y0)*k))
		p.rect(x, y, x+t-1, y+t-1, c)
	}
}

// glyph draws r in the 3×5 font.
func (p *paint) glyph(x, y int, r rune, c rgb) {
	for gy := 0; gy < fontH; gy++ {
		for gx := 0; gx < fontW; gx++ {
			if fontPixel(r, gx, gy) {
				p.dot(x+gx, y+gy, c)
			}
		}
	}
}

// eye draws a 2×2 eye looking at the viewer, or a closed one.
func (p *paint) eye(x, y int, open bool) {
	if !open {
		p.rect(x, y+1, x+1, y+1, hex(0x1a1020))
		return
	}
	p.rect(x, y, x+1, y+1, hex(0xffffff))
	p.dot(x+1, y+1, hex(0x14101c))
}

// done rings the picture with a dark outline and returns it.
func (p *paint) done() *sprite {
	edge := make([]bool, len(p.on))
	for y := 0; y < p.h; y++ {
		for x := 0; x < p.w; x++ {
			if p.on[y*p.w+x] {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= 0 && ny >= 0 && nx < p.w && ny < p.h && p.on[ny*p.w+nx] {
					edge[y*p.w+x] = true
				}
			}
		}
	}
	for i, e := range edge {
		if e {
			p.px[i], p.on[i] = hex(0x0c0814), true
		}
	}
	s := p.sprite
	return &s
}

// ── the creatures ────────────────────────────────────────────────────────

// dotling is Bubble Tea's spinner: a pink orb with one eye, its dots going
// round it like a spinner's. Frame f puts the bright dot at f eighths.
func dotling(f int) *sprite {
	p := newPaint(13, 13)
	p.disc(6.5, 6.5, 3, 3, hex(0xff5fd2))
	p.dot(5, 5, hex(0xffc6ef))
	p.rect(6, 6, 7, 7, hex(0xffffff))
	p.dot(7, 7, hex(0x200818))
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		k := float32((i-f+8)%8) / 8
		x, y := 6.5+5.2*math.Cos(a), 6.5+5.2*math.Sin(a)
		p.dot(int(x), int(y), hex(0xffe6f7).mix(hex(0x7a1e60), k))
	}
	return p.done()
}

// tapioca is Bubble Tea's cup: a pink jelly with a straw, pearls in its
// belly, and two stubby legs. big is Boba Prime: taller, three straws for
// a crown, and brows.
func tapioca(f int, big bool) *sprite {
	w, h := 16, 20
	if big {
		w, h = 24, 30
	}
	p := newPaint(w, h)
	top, bot := 5, h-3
	if big {
		top = 8
	}
	cup := hex(0xff7fd6)
	for y := top; y <= bot; y++ {
		t := float64(y-top) / float64(bot-top)
		half := (float64(w)/2 - 1.5) * (1 - 0.3*t)
		x0, x1 := int(float64(w)/2-half), int(float64(w)/2+half)-1
		for x := x0; x <= x1; x++ {
			c := cup
			if x <= x0+1 {
				c = hex(0xffc6ef)
			}
			p.dot(x, y, c)
		}
	}
	// The lid and the straws.
	p.rect(1, top-1, w-2, top, hex(0xffe0f6))
	straws := []int{w/2 + 2}
	if big {
		straws = []int{w/2 - 5, w/2 + 1, w/2 + 6}
	}
	for _, sx := range straws {
		p.line(float64(sx), float64(top-2), float64(sx+2), 0, 1, hex(0x8a4dff))
	}
	// Pearls in the belly, drifting with the frame.
	n := 5
	if big {
		n = 11
	}
	for i := 0; i < n; i++ {
		x := 3 + int(hash2(i, 7)*float32(w-6))
		y := bot - 1 - int(hash2(i, 3)*float32((bot-top)/3)) - (f+i)%2
		p.rect(x, y, x+1, y+1, hex(0x3a1a10))
		p.dot(x, y, hex(0x8a5a40))
	}
	// The face.
	ey := top + 3
	if big {
		ey = top + 5
		p.line(float64(w/2-7), float64(ey-2), float64(w/2-3), float64(ey-1), 1, hex(0x5a0b3a))
		p.line(float64(w/2+6), float64(ey-2), float64(w/2+2), float64(ey-1), 1, hex(0x5a0b3a))
	}
	p.eye(w/2-4, ey, f%6 != 5)
	p.eye(w/2+2, ey, f%6 != 5)
	p.rect(w/2-1, ey+3, w/2, ey+3, hex(0x5a0b3a))
	// Legs, stepping.
	lx := []int{w/2 - 4, w/2 + 3}
	for i, x := range lx {
		lift := 0
		if (f+i)%2 == 0 {
			lift = 1
		}
		p.rect(x, bot+1, x, h-1-lift, hex(0xc04aa0))
	}
	return p.done()
}

// chevron is Ink's </>: the brackets are its jaws and the slash its glowing
// spine. On frame 1 the jaws close in.
func chevron(f int) *sprite {
	p := newPaint(17, 13)
	c := hex(0x61dafb)
	in := float64(f % 2)
	p.line(6+in, 0, 1, 6, 2, c)
	p.line(1, 6, 6+in, 11, 2, c)
	p.line(10-in, 0, 15, 6, 2, c)
	p.line(15, 6, 10-in, 11, 2, c)
	p.line(10, 1, 6, 11, 1, hex(0xeaf6ff))
	p.eye(3, 4, true)
	p.eye(12, 4, true)
	return p.done()
}

// halfChevron is one bracket after a </> splits: '<' or '>'.
func halfChevron(right bool) *sprite {
	p := newPaint(8, 13)
	c := hex(0x61dafb)
	if right {
		p.line(1, 0, 6, 6, 2, c)
		p.line(6, 6, 1, 11, 2, c)
		p.eye(3, 5, true)
	} else {
		p.line(6, 0, 1, 6, 2, c)
		p.line(1, 6, 6, 11, 2, c)
		p.eye(3, 5, true)
	}
	return p.done()
}

// ratacrab is Ratatui: a crab's shell and claws with a rat's face, ears,
// whiskers and tail. king is Ferris, Crab King: bigger, crowned, no rat
// about it.
func ratacrab(f int, king bool) *sprite {
	w, h := 28, 16
	if king {
		w, h = 38, 24
	}
	p := newPaint(w, h)
	cx, cy := float64(w)/2, float64(h)*0.6
	rx, ry := float64(w)*0.24, float64(h)*0.24
	// Legs under the shell.
	for i := 0; i < 3; i++ {
		dx := 1.5 + float64(i)*rx/3
		step := float64((f + i) % 2)
		p.line(cx-dx, cy+ry*0.6, cx-dx-2, float64(h-1)-step, 1, hex(0xa33200))
		p.line(cx+dx, cy+ry*0.6, cx+dx+2, float64(h-1)-(1-step), 1, hex(0xa33200))
	}
	if !king {
		// The tail, curling away behind.
		p.line(cx+rx-2, cy+ry, cx+rx+2, cy+ry+2, 1, hex(0xff9a9a))
		p.line(cx+rx+2, cy+ry+2, cx+rx+5, cy+ry, 1, hex(0xff9a9a))
	}
	// Arms out to the claws, raised on frame 1.
	up := float64(f % 2 * 2)
	cr := 2.6
	if king {
		cr = 3.6
	}
	for _, s := range []float64{-1, 1} {
		ax := cx + s*(rx-1)
		hx, hy := cx+s*(rx+cr+2.5), cy-2-up
		p.line(ax, cy, hx, hy+1, 1, hex(0xd84000))
		p.disc(hx, hy-cr*0.4, cr, cr*0.8, hex(0xff6a1a))
		p.line(hx, hy-cr*0.4, hx+s*cr, hy-cr*0.4, 1, hex(0x14101c)) // the pincer's gap
	}
	p.disc(cx, cy, rx, ry, hex(0xf74c00))
	p.rect(int(cx-rx*0.5), int(cy+ry*0.55), int(cx+rx*0.5)-1, int(cy+ry*0.55), hex(0xa33200))
	ey := int(cy - ry*0.4)
	if king {
		gold := hex(0xffd23a)
		top := int(cy - ry - 1)
		p.rect(int(cx)-5, top-2, int(cx)+4, top-1, gold)
		for _, x := range []int{-5, -1, 3} {
			p.rect(int(cx)+x, top-4, int(cx)+x+1, top-3, gold)
		}
		p.dot(int(cx)-1, top-2, hex(0xff4fa0))
		p.eye(int(cx)-4, ey, true)
		p.eye(int(cx)+2, ey, true)
		return p.done()
	}
	// The rat: ears on the shell's rim, a face, whiskers.
	p.disc(cx-4, cy-ry, 1.8, 1.8, hex(0xffb08a))
	p.disc(cx+4, cy-ry, 1.8, 1.8, hex(0xffb08a))
	p.dot(int(cx-4), int(cy-ry), hex(0xff6a8a))
	p.dot(int(cx+3), int(cy-ry), hex(0xff6a8a))
	p.eye(int(cx)-3, ey, true)
	p.eye(int(cx)+1, ey, true)
	p.dot(int(cx)-1, ey+2, hex(0xff9ab0))
	p.dot(int(cx), ey+2, hex(0xff9ab0))
	p.line(cx-4, cy+1, cx-7, cy, 1, hex(0xfff4e0))
	p.line(cx+3, cy+1, cx+6, cy, 1, hex(0xfff4e0))
	return p.done()
}

// mimic is tview's modal: a window whose title bar lifts like a lid on a
// mouth of teeth, and whose [Y] and [N] buttons are its eyes.
func mimic(f int) *sprite {
	p := newPaint(22, 20)
	blue, light := hex(0x2a5bd7), hex(0x8fb4ff)
	open := f%2 == 1
	body := 6
	p.rect(1, body, 20, 18, blue)
	p.rect(1, body, 1, 18, light)
	p.rect(20, body, 20, 18, light)
	p.rect(1, 18, 20, 18, light)
	lid := body - 3
	if open {
		lid = body - 6
		p.rect(2, body-3, 19, body-1, hex(0x4a0014))
		for x := 3; x < 19; x += 3 {
			p.rect(x, body-1, x+1, body-1, hex(0xffffff))
			p.dot(x, body-3, hex(0xffffff))
		}
	}
	p.rect(1, lid, 20, lid+2, light)
	for i, c := range []uint32{0xff5f57, 0xfebc2e, 0x28c840} {
		p.dot(3+i*2, lid+1, hex(c))
	}
	// The buttons, glaring.
	for i, r := range []rune{'Y', 'N'} {
		x := 4 + i*9
		p.rect(x, 9, x+4, 15, hex(0xffe14a))
		p.glyph(x+1, 10, r, hex(0x14101c))
	}
	return p.done()
}

// button is one of a modal's buttons, still angry: Y or N.
func button(n bool) *sprite {
	p := newPaint(9, 10)
	p.rect(1, 1, 7, 8, hex(0xffe14a))
	p.rect(1, 1, 7, 1, hex(0xfff4a0))
	r := 'Y'
	if n {
		r = 'N'
	}
	p.glyph(3, 3, r, hex(0x14101c))
	return p.done()
}

// serpent is Bubbles' progress bar: its filled part a gradient body that
// slithers, its unfilled part a grey tail.
func serpent(f int) *sprite {
	p := newPaint(28, 10)
	cols := []uint32{0x4a4a5e, 0x4a4a5e, 0x5a56e0, 0x7c5ee8, 0x9e66ef, 0xc06ef3, 0xee6ff8}
	for i, c := range cols {
		y := 3 + int(math.Round(1.5*math.Sin(float64(i+f)*0.9)))
		p.rect(i*3, y, i*3+2, y+3, hex(c))
		p.dot(i*3, y, hex(c).mul(1.25))
	}
	hy := 3 + int(math.Round(1.5*math.Sin(float64(len(cols)+f)*0.9)))
	p.disc(23.5, float64(hy)+2, 3.5, 3, hex(0xff7ffb))
	p.eye(23, hy, true)
	p.rect(27, hy+3, 27, hy+3, hex(0xff3a5a))
	return p.done()
}

// phosphor is ncurses: an old monitor walking on two legs, its face drawn
// on the screen in green phosphor, a keyboard for a body.
func phosphor(f int) *sprite {
	p := newPaint(20, 24)
	p.rect(1, 0, 18, 12, hex(0x9a9a84))
	p.rect(1, 12, 18, 12, hex(0x6a6a5a))
	p.rect(3, 2, 16, 10, hex(0x021a08))
	for y := 2; y <= 10; y += 2 { // scanlines
		p.rect(3, y, 16, y, hex(0x04260c))
	}
	g := hex(0x33ff66)
	if f%5 == 4 { // blinking
		p.rect(5, 5, 7, 5, g)
		p.rect(12, 5, 14, 5, g)
	} else {
		p.rect(5, 3, 7, 7, g)
		p.rect(6, 4, 6, 6, hex(0x021a08))
		p.rect(12, 3, 14, 7, g)
		p.rect(13, 4, 13, 6, hex(0x021a08))
	}
	p.rect(7, 9, 12, 9, g)
	p.rect(9, 13, 10, 13, hex(0x6a6a5a))
	p.rect(2, 14, 17, 17, hex(0x8a8a76))
	for x := 3; x < 17; x += 2 {
		p.dot(x, 15, hex(0x4a4a3e))
		p.dot(x+1, 16, hex(0x4a4a3e))
	}
	for i, x := range []int{5, 13} {
		lift := (f + i) % 2
		p.rect(x, 18, x+1, 23-lift, hex(0x6a6a5a))
	}
	return p.done()
}

func frames(n int, draw func(int) *sprite) []*sprite {
	s := make([]*sprite, n)
	for i := range s {
		s[i] = draw(i)
	}
	return s
}
