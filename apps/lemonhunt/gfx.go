package main

import "math"

// ── colour ───────────────────────────────────────────────────────────────

// rgb is a linear colour, 0…1 a channel, and more for light.
type rgb struct{ r, g, b float32 }

func (c rgb) mul(k float32) rgb { return rgb{c.r * k, c.g * k, c.b * k} }
func (c rgb) add(o rgb) rgb     { return rgb{c.r + o.r, c.g + o.g, c.b + o.b} }
func (c rgb) times(o rgb) rgb   { return rgb{c.r * o.r, c.g * o.g, c.b * o.b} }
func (c rgb) mix(o rgb, t float32) rgb {
	return rgb{c.r + (o.r-c.r)*t, c.g + (o.g-c.g)*t, c.b + (o.b-c.b)*t}
}

// px is a colour as the terminal gets it.
type px struct{ r, g, b uint8 }

func to8(v float32) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	}
	return uint8(v*255 + 0.5)
}

func (c rgb) px() px { return px{to8(c.r), to8(c.g), to8(c.b)} }

func hex(v uint32) rgb {
	return rgb{float32(v>>16&255) / 255, float32(v>>8&255) / 255, float32(v&255) / 255}
}

// hash2 is a repeatable pseudo-random number in 0…1 for a pair of ints.
func hash2(a, b int) float32 {
	h := uint32(a)*374761393 + uint32(b)*668265263
	h = (h ^ h>>13) * 1274126177
	h ^= h >> 16
	return float32(h&0xffff) / 0xffff
}

func hash3(a, b, c int) float32 { return hash2(a*31+c*7919, b) }

// ── textures ─────────────────────────────────────────────────────────────

const texSize = 32

// texture is a 32×32 surface: its colour, and how much of that colour it
// gives off by itself, unlit.
type texture struct {
	c    [texSize * texSize]rgb
	glow [texSize * texSize]float32
}

func (t *texture) set(x, y int, c rgb) {
	if x >= 0 && y >= 0 && x < texSize && y < texSize {
		t.c[y*texSize+x] = c
	}
}

func (t *texture) sample(u, v float64) (rgb, float32) {
	x := int(u * texSize)
	y := int(v * texSize)
	x &= texSize - 1
	if y < 0 {
		y = 0
	} else if y >= texSize {
		y = texSize - 1
	}
	i := y*texSize + x
	return t.c[i], t.glow[i]
}

const (
	texBrick = iota
	texStone
	texPipe
	texLair
	texGate
	texFloor
	texCeil
	nTextures
)

var textures [nTextures]texture

// wallTexture maps a level character to its texture.
func wallTexture(tile byte) *texture {
	switch tile {
	case 'S':
		return &textures[texStone]
	case 'P':
		return &textures[texPipe]
	case 'L':
		return &textures[texLair]
	case 'G':
		return &textures[texGate]
	}
	return &textures[texBrick]
}

func init() {
	makeBrick(&textures[texBrick])
	makeStone(&textures[texStone])
	makePipe(&textures[texPipe])
	makeLair(&textures[texLair])
	makeGate(&textures[texGate])
	makeFloor(&textures[texFloor])
	makeCeil(&textures[texCeil])
}

func makeBrick(t *texture) {
	mortar := hex(0x4a423c)
	for y := 0; y < texSize; y++ {
		course := y / 8
		off := (course % 2) * 8
		for x := 0; x < texSize; x++ {
			bx := (x + off) / 16
			if y%8 == 7 || (x+off)%16 == 15 {
				t.set(x, y, mortar.mul(0.8+0.3*hash2(x, y)))
				continue
			}
			base := hex(0x8c4128).mix(hex(0xa8603a), hash2(bx, course))
			k := 0.88 + 0.2*hash2(x*7, y*3)
			switch y % 8 {
			case 0:
				k += 0.18 // the top of a brick catches the light
			case 6:
				k -= 0.2
			}
			t.set(x, y, base.mul(k))
		}
	}
}

func makeStone(t *texture) {
	for y := 0; y < texSize; y++ {
		row := y / 6
		for x := 0; x < texSize; x++ {
			// Stones of uneven width in each row.
			w := 9 + int(hash2(row, 1)*6)
			off := int(hash2(row, 2) * 9)
			sx := (x + off) / w
			if y%6 == 5 || (x+off)%w == w-1 {
				t.set(x, y, hex(0x1e241e))
				continue
			}
			base := hex(0x4a5446).mix(hex(0x5e6450), hash2(sx, row))
			t.set(x, y, base.mul(0.85+0.25*hash2(x*5, y*9)))
		}
	}
	// Moss creeping up from the foot of the wall.
	for x := 0; x < texSize; x++ {
		h := 3 + int(hash2(x, 77)*7)
		for y := texSize - h; y < texSize; y++ {
			if hash2(x*3, y*5) < 0.7 {
				t.set(x, y, hex(0x3d6a2a).mul(0.8+0.4*hash2(x, y)))
			}
		}
	}
}

func makePipe(t *texture) {
	for y := 0; y < texSize; y++ {
		for x := 0; x < texSize; x++ {
			c := hex(0x3c434c).mul(0.85 + 0.2*hash2(x*3, y*7))
			if x%16 == 0 {
				c = hex(0x23272c)
			}
			if (x%16 == 3 || x%16 == 12) && (y == 2 || y == 29) {
				c = hex(0x8a939c) // rivets
			}
			t.set(x, y, c)
		}
	}
	pipe := func(y0, y1 int, col rgb) {
		for y := y0; y <= y1; y++ {
			// Shade the pipe as a cylinder: bright along the top.
			a := (float64(y-y0) + 0.5) / float64(y1-y0+1)
			k := float32(0.35 + 0.75*math.Sin(a*math.Pi)*(1.1-a*0.6))
			for x := 0; x < texSize; x++ {
				c := col.mul(k)
				if x%10 == 0 {
					c = c.mul(0.7) // a joint
				}
				t.set(x, y, c)
			}
		}
	}
	pipe(7, 13, hex(0x6f8aa0))
	pipe(19, 22, hex(0x8a6a4a))
	// Rust running down from the pipes.
	for x := 0; x < texSize; x++ {
		if hash2(x, 5) < 0.3 {
			n := 3 + int(hash2(x, 6)*8)
			for y := 14; y < 14+n && y < 19; y++ {
				t.set(x, y, hex(0x6b3a1c).mul(0.9-float32(y-14)*0.08))
			}
		}
	}
}

func makeLair(t *texture) {
	for y := 0; y < texSize; y++ {
		for x := 0; x < texSize; x++ {
			t.set(x, y, hex(0x1c1024).mul(0.8+0.3*hash2(x*3, y*5)))
		}
	}
	// Two terminal panels, glowing: a border, a title bar and a bar chart.
	glow := func(x, y int, c rgb, k float32) {
		t.set(x, y, c)
		t.glow[y*texSize+x] = k
	}
	for _, p := range [2][2]int{{2, 3}, {17, 3}} {
		x0, y0 := p[0], p[1]
		x1, y1 := x0+12, y0+24
		for x := x0; x <= x1; x++ {
			glow(x, y0, hex(0xe060c8), 1)
			glow(x, y1, hex(0xe060c8), 1)
			glow(x, y0+3, hex(0x80305f), 0.6)
		}
		for y := y0; y <= y1; y++ {
			glow(x0, y, hex(0xe060c8), 1)
			glow(x1, y, hex(0xe060c8), 1)
		}
		for x := x0 + 2; x <= x1-2; x += 2 {
			h := 3 + int(hash2(x, y0)*14)
			for y := y1 - 2; y > y1-2-h; y-- {
				glow(x, y, hex(0x48d8e8), 0.9)
			}
		}
	}
}

func makeGate(t *texture) {
	for y := 0; y < texSize; y++ {
		for x := 0; x < texSize; x++ {
			// Red light from the lair behind the bars.
			k := float32(0.5 + 0.5*float64(y)/texSize)
			t.set(x, y, hex(0x3a0a10).mul(k))
			t.glow[y*texSize+x] = 0.8
			bar := x%8 < 3 || y == 4 || y == 5 || y == 26 || y == 27
			if bar {
				c := hex(0x5a5550)
				if x%8 == 1 {
					c = hex(0x8c857c)
				}
				t.set(x, y, c.mul(0.8+0.3*hash2(x, y)))
				t.glow[y*texSize+x] = 0
			}
		}
	}
}

func makeFloor(t *texture) {
	for y := 0; y < texSize; y++ {
		for x := 0; x < texSize; x++ {
			c := hex(0x3a3630).mix(hex(0x48423a), hash2(x/16, y/16))
			c = c.mul(0.82 + 0.3*hash2(x*7, y*11))
			if x%16 == 0 || y%16 == 0 {
				c = hex(0x1c1a17)
			}
			t.set(x, y, c)
		}
	}
	// A crack or two.
	x := 5
	for y := 3; y < 14; y++ {
		t.set(x, y, hex(0x201d19))
		if hash2(y, 3) < 0.5 {
			x++
		}
	}
}

func makeCeil(t *texture) {
	for y := 0; y < texSize; y++ {
		for x := 0; x < texSize; x++ {
			c := hex(0x2a2a30).mul(0.8 + 0.3*hash2(x*3, y*5))
			if y%8 == 0 {
				c = hex(0x141418)
			}
			t.set(x, y, c)
		}
	}
}

// ── sprites ──────────────────────────────────────────────────────────────

// sprite is a small picture drawn once at start-up. Transparent pixels have
// a = 0; glow is light the pixel gives off by itself.
type sprite struct {
	w, h int
	c    []rgb
	a    []uint8
	glow []float32
}

func newSprite(w, h int) *sprite {
	return &sprite{w: w, h: h, c: make([]rgb, w*h), a: make([]uint8, w*h), glow: make([]float32, w*h)}
}

func (s *sprite) put(x, y int, c rgb, glow float32) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	i := y*s.w + x
	s.c[i], s.a[i], s.glow[i] = c, 1, glow
}

// ellipse fills an ellipse shaded as the front of a ball lit from the top
// left, which is what gives the sprites their volume.
func (s *sprite) ellipse(cx, cy, rx, ry float64, c rgb, glow float32) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx := (float64(x) + 0.5 - cx) / rx
			dy := (float64(y) + 0.5 - cy) / ry
			d := dx*dx + dy*dy
			if d > 1 {
				continue
			}
			nz := math.Sqrt(1 - d)
			k := 0.45 + 0.35*nz + 0.3*(-dx*0.5-dy*0.7)
			s.put(x, y, c.mul(float32(k)), glow)
		}
	}
}

func (s *sprite) flat(cx, cy, rx, ry float64, c rgb, glow float32) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx := (float64(x) + 0.5 - cx) / rx
			dy := (float64(y) + 0.5 - cy) / ry
			if dx*dx+dy*dy <= 1 {
				s.put(x, y, c, glow)
			}
		}
	}
}

func (s *sprite) line(x0, y0, x1, y1, thick float64, c rgb, glow float32) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*2) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := x0+(x1-x0)*t, y0+(y1-y0)*t
		if thick <= 1 {
			s.put(int(x), int(y), c, glow)
		} else {
			s.flat(x, y, thick/2, thick/2, c, glow)
		}
	}
}

func (s *sprite) rect(x0, y0, x1, y1 int, c rgb, glow float32) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			s.put(x, y, c, glow)
		}
	}
}

// outline darkens the rim of the drawing, so it reads against any wall.
func (s *sprite) outline() {
	edge := make([]bool, len(s.a))
	for y := 0; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			if s.a[y*s.w+x] == 0 {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= s.w || ny >= s.h || s.a[ny*s.w+nx] == 0 {
					edge[y*s.w+x] = true
				}
			}
		}
	}
	for i, e := range edge {
		if e && s.glow[i] < 0.5 {
			s.c[i] = s.c[i].mul(0.45)
		}
	}
}

// Animation frames, drawn at start-up.
var (
	ratWalk  [4]*sprite
	ratLunge *sprite
	fatWalk  [4]*sprite
	fatLunge *sprite
	bossIdle [2]*sprite
	bossRage *sprite
	cheese   *sprite
	lampArt  *sprite
	gunArt   *sprite
	gunFire  *sprite
)

func init() {
	for i := range ratWalk {
		ratWalk[i] = drawRat(float64(i)*math.Pi/2, false, hex(0x9a9296), 1)
		fatWalk[i] = drawRat(float64(i)*math.Pi/2, false, hex(0xa07852), 1.25)
	}
	ratLunge = drawRat(0, true, hex(0x9a9296), 1)
	fatLunge = drawRat(0, true, hex(0xa07852), 1.25)
	bossIdle[0] = drawBoss(0, false)
	bossIdle[1] = drawBoss(1, false)
	bossRage = drawBoss(0, true)
	cheese = drawCheese()
	lampArt = drawLamp()
	gunArt = drawGun(false)
	gunFire = drawGun(true)
}

// drawRat draws a rat side on, facing right, mid-stride at phase.
func drawRat(phase float64, biting bool, fur rgb, fat float64) *sprite {
	s := newSprite(44, 26)
	pink := hex(0xd88c98)
	gy := 25.0
	// Tail, curling behind.
	for i := 0; i < 14; i++ {
		t := float64(i) / 13
		x := 12 - t*12
		y := 18 - t*4 + math.Sin(t*4+phase)*2
		s.put(int(x), int(y), pink.mul(0.8), 0)
		s.put(int(x), int(y)+1, pink.mul(0.6), 0)
	}
	// Legs, two pairs moving opposite each other.
	for k, lx := range [4]float64{13, 17, 27, 31} {
		off := math.Sin(phase+float64(k%2)*math.Pi) * 2
		s.line(lx, 20, lx+off, gy, 2, pink.mul(0.7), 0)
	}
	bodyY := 16 - (fat-1)*6
	s.ellipse(22, bodyY, 11*fat, 6*fat, fur, 0)
	s.ellipse(22, bodyY+2.5*fat, 8*fat, 3*fat, fur.mix(hex(0xb8aca8), 0.35), 0) // belly
	hx, hy := 33.0, 13.0-(fat-1)*5
	if biting {
		hx, hy = 35, 15-(fat-1)*5
	}
	s.ellipse(hx, hy, 6.2, 5, fur, 0)
	s.ellipse(hx+5, hy+1.6, 3.2, 2.4, fur.mix(hex(0xc8bcb8), 0.4), 0) // snout
	s.flat(hx+8, hy+1.2, 1.3, 1.2, pink, 0)                           // nose
	s.ellipse(hx-3, hy-4.5, 3.2, 3.2, fur, 0)                         // ear
	s.flat(hx-3, hy-4.3, 1.8, 1.8, pink, 0)
	if biting {
		s.rect(int(hx+4), int(hy+4), int(hx+7), int(hy+4), hex(0x300808), 0)
		s.put(int(hx+5), int(hy+4), hex(0xf0f0e8), 0)
		s.put(int(hx+7), int(hy+4), hex(0xf0f0e8), 0)
	}
	s.outline()
	// Eye and whiskers go on after the outline: they are the details that
	// have to stay bright.
	s.flat(hx+1.5, hy-1, 1.3, 1.3, hex(0xff3020), 1)
	s.put(int(hx+2), int(hy-2), hex(0xffffff), 1)
	s.line(hx+6, hy+2, hx+10, hy+0.5, 1, hex(0xc8c8c8), 0)
	s.line(hx+6, hy+3, hx+10, hy+4, 1, hex(0xc8c8c8), 0)
	return s
}

// drawBoss draws Ratatui from the front: a crown, a cape, and eyes that
// burn when it is about to charge.
func drawBoss(breath int, rage bool) *sprite {
	s := newSprite(64, 72)
	fur := hex(0x9a8e92)
	pink := hex(0xe898a4)
	gold := hex(0xffc838)
	b := float64(breath)
	// Tail behind, curling out to the right.
	for i := 0; i < 26; i++ {
		t := float64(i) / 25
		s.line(44+t*16, 64-t*10+math.Sin(t*5)*3, 44+t*16, 65-t*10+math.Sin(t*5)*3, 2, pink.mul(0.7), 0)
	}
	// Cape and body.
	s.ellipse(32, 54+b*0.5, 23, 18, hex(0x8a30a0), 0)
	s.ellipse(32, 56+b*0.5, 13, 13, fur.mix(hex(0xb0a4a0), 0.3), 0) // belly
	for y := 38; y < 72; y++ {                                      // gold trim down the cape
		s.put(19, y, gold.mul(0.8), 0)
		s.put(45, y, gold.mul(0.8), 0)
	}
	s.ellipse(22, 69, 6, 3, pink.mul(0.8), 0) // feet
	s.ellipse(42, 69, 6, 3, pink.mul(0.8), 0)
	s.ellipse(11, 50+b, 5, 7, fur, 0) // arms
	s.ellipse(53, 50+b, 5, 7, fur, 0)
	// Head, ears, snout.
	s.ellipse(15, 15, 8, 8, fur, 0)
	s.ellipse(49, 15, 8, 8, fur, 0)
	s.flat(15, 15, 5, 5, pink, 0)
	s.flat(49, 15, 5, 5, pink, 0)
	s.ellipse(32, 28, 17, 15, fur, 0)
	s.ellipse(32, 37, 9, 6.5, fur.mix(hex(0xc8bcb8), 0.4), 0)
	s.flat(32, 33.5, 3, 2.2, pink, 0)
	if rage {
		s.flat(32, 41.5, 6, 3, hex(0x300808), 0) // mouth open
	}
	s.rect(29, 41, 30, 45, hex(0xf2eee0), 0) // teeth
	s.rect(33, 41, 34, 45, hex(0xf2eee0), 0)
	// Crown.
	s.rect(19, 9, 45, 14, gold, 0)
	for _, x := range [3]float64{22, 32, 42} {
		for y := 0; y < 7; y++ {
			w := float64(6-y) / 2
			s.rect(int(x-w), 2+y, int(x+w), 2+y, gold.mul(1.1-float32(y)*0.05), 0)
		}
	}
	s.outline()
	s.flat(26, 11.5, 1.6, 1.6, hex(0xe02030), 1) // gems
	s.flat(38, 11.5, 1.6, 1.6, hex(0x3060f0), 1)
	eye := hex(0xff3020)
	r := 3.2
	if rage {
		eye, r = hex(0xffd040), 4
	}
	s.flat(25, 25, r, r, eye, 1)
	s.flat(39, 25, r, r, eye, 1)
	s.put(24, 24, hex(0xffffff), 1)
	s.put(38, 24, hex(0xffffff), 1)
	for _, w := range [3]float64{-2, 0, 2} {
		s.line(24, 38+w*0.5, 8, 36+w*2, 1, hex(0xd0d0d0), 0)
		s.line(40, 38+w*0.5, 56, 36+w*2, 1, hex(0xd0d0d0), 0)
	}
	return s
}

func drawCheese() *sprite {
	s := newSprite(16, 11)
	c := hex(0xf0c040)
	for y := 0; y < 11; y++ {
		for x := 0; x <= y*15/10; x++ {
			s.put(x, y, c.mul(0.8+0.25*float32(10-y)/10), 0.2)
		}
	}
	s.flat(4, 7, 1.3, 1.3, hex(0xb08020), 0.1)
	s.flat(9, 9, 1.5, 1.1, hex(0xb08020), 0.1)
	s.outline()
	return s
}

func drawLamp() *sprite {
	s := newSprite(12, 16)
	s.line(6, 0, 6, 6, 1, hex(0x404040), 0) // chain
	for y := 6; y < 11; y++ {
		w := (y - 6) + 1
		s.rect(6-w, y, 5+w, y, hex(0x3a3e44).mul(1-float32(y-6)*0.08), 0)
	}
	s.flat(6, 12, 3, 2.4, hex(0xffe0a0), 1)
	return s
}

// drawGun draws the lemon squirter as the player holds it, from behind: a
// pump barrel with a whole lemon clamped on top as the tank.
func drawGun(firing bool) *sprite {
	s := newSprite(72, 46)
	metal := hex(0x7c8894)
	// The barrel narrows towards the middle of the screen.
	for y := 8; y < 46; y++ {
		t := float64(y-8) / 38
		half := 3 + t*8
		for x := int(36 - half); x <= int(36+half); x++ {
			a := (float64(x) - (36 - half)) / (2 * half)
			k := float32(0.4 + 0.65*math.Sin(a*math.Pi)*(1.2-a*0.7))
			c := metal.mul(k)
			if y%7 == 0 {
				c = c.mul(0.6) // rings round the barrel
			}
			s.put(x, y, c, 0)
		}
	}
	// The pump grip, ribbed.
	for y := 34; y < 42; y++ {
		for x := 27; x <= 45; x++ {
			c := hex(0x3a2a22)
			if y%2 == 0 {
				c = hex(0x5a4032)
			}
			s.put(x, y, c, 0)
		}
	}
	// The lemon: pointed at both ends, dimpled, with its leaf.
	lemon := hex(0xffd02a)
	s.ellipse(36, 21, 15, 9, lemon, 0.25)
	s.ellipse(20, 21, 3, 2.6, lemon.mul(0.85), 0.2)
	s.ellipse(52, 21, 3, 2.6, lemon.mul(0.85), 0.2)
	for i := 0; i < 40; i++ {
		x := 24 + int(hash2(i, 1)*24)
		y := 14 + int(hash2(i, 2)*14)
		dx, dy := (float64(x)-36)/15, (float64(y)-21)/9
		if dx*dx+dy*dy < 0.8 {
			s.put(x, y, lemon.mul(0.72), 0.2)
		}
	}
	s.line(40, 13, 46, 8, 2.5, hex(0x4caa38), 0.1)
	// Clamps holding it on.
	s.rect(28, 12, 29, 30, hex(0x505860), 0)
	s.rect(43, 12, 44, 30, hex(0x505860), 0)
	s.rect(33, 0, 39, 7, metal.mul(0.75), 0) // nozzle
	s.rect(34, 0, 38, 1, hex(0x4caa38), 0)
	// A hand on the grip.
	s.ellipse(54, 41, 9, 6, hex(0xd8a080), 0)
	s.ellipse(47, 37, 3.5, 2.5, hex(0xe0aa88), 0)
	s.outline()
	s.flat(30, 17, 4, 1.8, hex(0xfff6c8), 0.7) // shine
	if firing {
		s.flat(36, 1, 6, 3, hex(0xfff080), 1)
		s.flat(36, 1, 3, 1.6, hex(0xffffff), 1)
	}
	return s
}
