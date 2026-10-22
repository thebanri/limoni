package main

import "math"

// Every augment card has a picture, 20×12 pixels, painted once at start-up
// with the creatures' brushes.

const iconW, iconH = 20, 12

var icons [nAug]*sprite

func init() {
	for a := augID(0); a < nAug; a++ {
		icons[a] = paintIcon(a)
	}
}

var (
	brass  = hex(0xffc94a)
	cyanC  = hex(0x5ad1ff)
	steel  = hex(0x8e9bb8)
	ember  = hex(0xff6a1a)
	flameY = hex(0xffe14a)
	blood  = hex(0xff3a5a)
	leaf   = hex(0x6ee07a)
	violet = hex(0xb07bff)
	bone   = hex(0xf0ead8)
)

func (p *paint) heart(cx, cy int, c rgb) {
	p.disc(float64(cx)-1.5, float64(cy)-1, 2.2, 2, c)
	p.disc(float64(cx)+2.5, float64(cy)-1, 2.2, 2, c)
	for y := 0; y < 4; y++ {
		p.rect(cx-3+y, cy+y, cx+4-y, cy+y, c)
	}
	p.dot(cx-2, cy-2, hex(0xffffff))
}

func (p *paint) boom(cx, cy, r float64) {
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		p.line(cx, cy, cx+math.Cos(a)*r*1.4, cy+math.Sin(a)*r*1.1, 1, ember)
	}
	p.disc(cx, cy, r, r*0.8, ember)
	p.disc(cx, cy, r*0.55, r*0.45, flameY)
}

func (p *paint) arrowHead(x, y int, c rgb) {
	p.dot(x-1, y-1, c)
	p.dot(x-1, y+1, c)
	p.dot(x-2, y-2, c)
	p.dot(x-2, y+2, c)
}

func paintIcon(a augID) *sprite {
	p := newPaint(iconW, iconH)
	switch a {
	case augREP: // three rounds side by side
		for i := 0; i < 3; i++ {
			x := 4 + i*5
			p.rect(x, 4, x+2, 10, brass)
			p.rect(x, 4, x, 10, brass.mul(1.3))
			p.disc(float64(x)+1.5, 3.5, 1.5, 2, hex(0xd8a040))
		}
	case augScroll: // a shot bouncing off a wall
		p.rect(17, 0, 18, 11, steel)
		p.line(1, 2, 16, 6, 1, cyanC.mul(0.6))
		p.line(16, 6, 6, 10, 1, cyanC.mul(0.6))
		p.disc(6, 9.5, 1.6, 1.6, cyanC)
	case augDiff: // straight through a block
		p.rect(8, 1, 11, 10, steel)
		p.line(1, 6, 18, 6, 1, cyanC)
		p.arrowHead(19, 6, cyanC)
	case augFork: // one becomes three
		p.line(1, 6, 8, 6, 1, cyanC)
		for _, y := range []float64{1, 6, 11} {
			p.line(8, 6, 16, y, 1, cyanC)
			p.disc(17, y+0.5, 1.4, 1.4, flameY)
		}
	case augPanic:
		p.boom(10, 6, 4)
	case augChain: // an arc between three goroutines
		pts := [][2]float64{{2, 9}, {10, 3}, {17, 8}}
		for i := 0; i < 2; i++ {
			a, b := pts[i], pts[i+1]
			mx, my := (a[0]+b[0])/2, (a[1]+b[1])/2+2
			p.line(a[0], a[1], mx, my, 1, cyanC)
			p.line(mx, my, b[0], b[1], 1, cyanC)
		}
		for _, q := range pts {
			p.disc(q[0], q[1], 1.8, 1.8, hex(0x61dafb))
		}
	case augHotPath: // a flame
		p.disc(10, 8, 4, 3.5, blood)
		p.disc(10, 6, 3, 3.5, ember)
		p.disc(10, 5, 1.8, 3, flameY)
		p.line(10, 0, 10, 2, 1, flameY)
	case augLoop: // a loop with its arrow
		for i := 0; i < 28; i++ {
			t := float64(i) / 28 * 2 * math.Pi * 0.85
			p.dot(int(10+5*math.Cos(t)), int(6+4.5*math.Sin(t)), cyanC)
		}
		p.arrowHead(16, 4, cyanC)
		p.disc(10, 6, 1.5, 1.5, flameY)
	case augHeap: // the heap's canister, full
		p.rect(7, 0, 12, 1, steel)
		p.rect(7, 10, 12, 11, steel)
		p.rect(8, 2, 11, 9, cyanC)
		p.rect(8, 2, 8, 9, hex(0xbff0ff))
		p.rect(6, 1, 6, 10, steel)
		p.rect(13, 1, 13, 10, steel)
	case augZeroAlloc: // a zero that shines
		for gy := 0; gy < fontH*2; gy++ {
			for gx := 0; gx < fontW*2; gx++ {
				if fontPixel('0', gx/2, gy/2) {
					p.dot(7+gx, 1+gy, leaf)
				}
			}
		}
		p.dot(16, 2, hex(0xffffff))
		p.dot(15, 3, flameY)
		p.dot(17, 3, flameY)
		p.dot(16, 4, flameY)
		p.dot(3, 9, flameY)
	case augConcGC: // a clock, running fast
		p.disc(10, 6, 5, 5, hex(0xdfe6f5))
		p.line(10, 6, 10, 2, 1, hex(0x1a2030))
		p.line(10, 6, 13, 7, 1, blood)
		p.line(3, 2, 1, 4, 1, cyanC)
		p.line(17, 2, 19, 4, 1, cyanC)
	case augEscape: // a crosshair with a star in it
		for i := 0; i < 24; i++ {
			t := float64(i) / 24 * 2 * math.Pi
			p.dot(int(10+5*math.Cos(t)), int(6+5*math.Sin(t)), steel)
		}
		p.line(10, 0, 10, 2, 1, steel)
		p.line(10, 10, 10, 11, 1, steel)
		p.line(3, 6, 5, 6, 1, steel)
		p.line(15, 6, 17, 6, 1, steel)
		p.disc(10, 6, 1.8, 1.8, blood)
		p.dot(10, 6, flameY)
	case augShutdown:
		p.heart(10, 5, blood)
	case augRecover:
		p.heart(8, 5, blood)
		p.rect(15, 2, 15, 8, leaf)
		p.rect(12, 5, 18, 5, leaf)
	case augHotReload: // a bolt
		p.line(12, 0, 7, 6, 2, flameY)
		p.line(7, 6, 13, 6, 1, flameY)
		p.line(13, 6, 8, 11, 2, flameY)
	case augGOGC: // a gem
		for y := 0; y < 10; y++ {
			half := 5 - int(math.Abs(float64(y)-3))
			if y > 3 {
				half = int(5 - float64(y-3)*0.8)
			}
			p.rect(10-half, y+1, 9+half, y+1, leaf.mul(0.85+0.03*float32(y)))
		}
		p.rect(8, 2, 9, 3, hex(0xd8ffe0))
		p.dot(17, 1, hex(0xffffff))
	case augCascade: // one blast throwing smaller ones
		p.boom(6, 7, 3.5)
		p.boom(13, 4, 2.2)
		p.boom(17, 9, 1.5)
	case augRecursion: // squares in squares
		for i, c := range []uint32{0x6a3cff, 0x8a5cff, 0xb07bff, 0xe0c0ff} {
			p.rect(4+2*i, i, 15-2*i, 11-i, hex(c))
		}
		p.rect(9, 5, 10, 6, hex(0x1a0a30))
	case augOverflow: // a stack spilling over
		for i := 0; i < 5; i++ {
			c := steel
			if i == 4 {
				c = blood
			}
			p.rect(5, 10-2*i, 14, 10-2*i, c)
		}
		p.rect(15, 2, 17, 2, blood)
		p.dot(18, 3, blood)
		p.dot(18, 5, blood.mul(0.7))
	case augSync: // an hourglass
		p.rect(5, 0, 14, 0, steel)
		p.rect(5, 11, 14, 11, steel)
		for y := 1; y < 11; y++ {
			half := int(math.Abs(float64(y)-5.5)) + 1
			c := hex(0x2a3040)
			if y > 6 || y < 3 {
				c = flameY
			}
			p.rect(10-half, y, 9+half, y, c)
		}
	case augKill9: // a skull
		p.disc(10, 5, 5, 4.5, bone)
		p.rect(7, 8, 12, 10, bone)
		p.rect(7, 4, 8, 6, hex(0x14101c))
		p.rect(11, 4, 12, 6, hex(0x14101c))
		p.dot(9, 7, hex(0x14101c))
		p.dot(10, 7, hex(0x14101c))
		for x := 7; x <= 12; x += 2 {
			p.dot(x, 10, hex(0x14101c))
		}
	}
	return p.done()
}
