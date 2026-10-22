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
	frost  = hex(0x9fe8ff)
	toxic  = hex(0x8ce04a)
	volt   = hex(0xffe14a)
	dashV  = hex(0xc07bff)
	night  = hex(0x14101c)
)

// flame is a fire s high-ish, its base at (cx, cy+2s).
func (p *paint) flame(cx, cy, s float64) {
	p.disc(cx, cy+1.6*s, 3.6*s, 3*s, blood)
	p.disc(cx, cy+0.4*s, 2.6*s, 3.2*s, ember)
	p.disc(cx, cy, 1.5*s, 2.5*s, flameY)
	p.line(cx, cy-3.4*s, cx, cy-2.4*s, 1, flameY)
}

// flake is a snowflake of radius r.
func (p *paint) flake(cx, cy, r float64, c rgb) {
	for k := 0; k < 3; k++ {
		a := float64(k) * math.Pi / 3
		dx, dy := math.Cos(a)*r, math.Sin(a)*r*0.8
		p.line(cx-dx, cy-dy, cx+dx, cy+dy, 1, c)
		for _, sgn := range [2]float64{-1, 1} {
			bx, by := cx+sgn*dx*0.6, cy+sgn*dy*0.6
			p.dot(int(math.Round(bx-dy*0.25)), int(math.Round(by+dx*0.25)), c)
			p.dot(int(math.Round(bx+dy*0.25)), int(math.Round(by-dx*0.25)), c)
		}
	}
	p.dot(int(cx), int(cy), hex(0xffffff))
}

// bolt is a lightning bolt from (x, y), h tall.
func (p *paint) bolt(x, y, h float64, c rgb) {
	p.line(x+2, y, x-1, y+h*0.55, 1, c)
	p.line(x-1, y+h*0.55, x+2, y+h*0.45, 1, c)
	p.line(x+2, y+h*0.45, x-1, y+h, 1, c)
}

// drop is a drop of poison.
func (p *paint) drop(cx, cy, r float64, c rgb) {
	p.disc(cx, cy, r, r, c)
	p.line(cx, cy-r*2, cx-r*0.7, cy-r*0.5, 1, c)
	p.line(cx, cy-r*2, cx+r*0.7, cy-r*0.5, 1, c)
	p.dot(int(cx-r*0.4), int(cy-r*0.3), hex(0xe8ffd0))
}

// cloud is a puff of three discs.
func (p *paint) cloud(cx, cy float64, c rgb) {
	p.disc(cx-3, cy+1, 3, 2.2, c.mul(0.8))
	p.disc(cx+3, cy+1, 3, 2.2, c.mul(0.8))
	p.disc(cx, cy-1, 3.5, 2.8, c)
}

// streaks are speed lines, left of x.
func (p *paint) streaks(x, y int, c rgb) {
	for k, l := range [3]int{6, 9, 5} {
		yy := y - 3 + k*3
		p.rect(x-l, yy, x-1, yy, c.mul(0.5+0.2*float32(k)))
	}
}

// runner is a little figure mid-stride, at (x, y) its head.
func (p *paint) runner(x, y int, c rgb) {
	p.disc(float64(x)+0.5, float64(y)+0.5, 1.4, 1.4, c)
	p.line(float64(x), float64(y)+2, float64(x)-1, float64(y)+6, 1, c)
	p.line(float64(x)-1, float64(y)+6, float64(x)-3, float64(y)+9, 1, c)
	p.line(float64(x)-1, float64(y)+6, float64(x)+2, float64(y)+9, 1, c)
	p.line(float64(x), float64(y)+3, float64(x)+3, float64(y)+4, 1, c)
	p.line(float64(x), float64(y)+3, float64(x)-3, float64(y)+2, 1, c)
}

// foe is a little round enemy.
func (p *paint) foe(cx, cy, r float64, c rgb) {
	p.disc(cx, cy, r, r, c)
	p.dot(int(cx-r*0.4), int(cy-r*0.2), hex(0xffffff))
	p.dot(int(cx+r*0.3), int(cy-r*0.2), hex(0xffffff))
}

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

	// ── fire ──
	case augIgnite: // a round setting a foe alight
		p.foe(12, 7, 3.5, steel)
		p.flame(12, 3, 0.7)
		p.line(1, 7, 6, 7, 1, flameY)
	case augFirewall: // a wall of bricks, burning
		for y := 7; y <= 11; y += 2 {
			for x := 1 + (y/2%2)*2; x < 19; x += 4 {
				p.rect(x, y, x+2, y, hex(0xa04a2a))
			}
		}
		for _, x := range []float64{4, 10, 16} {
			p.flame(x, 3, 0.55)
		}
	case augWildfire: // fire leaping foe to foe
		p.foe(3, 9, 2.2, steel)
		p.foe(10, 9, 2.2, steel)
		p.foe(17, 9, 2.2, steel)
		p.flame(3, 3, 0.5)
		p.flame(10, 3, 0.5)
		p.flame(17, 3, 0.5)
		p.line(5, 2, 8, 1, 1, flameY)
		p.line(12, 1, 15, 2, 1, flameY)
	case augOverheat: // a thermometer in the red
		p.rect(9, 0, 11, 8, bone)
		p.rect(10, 3, 10, 8, blood)
		p.disc(10.5, 9.5, 2.5, 2.2, blood)
		p.flame(16, 6, 0.5)
		p.flame(4, 6, 0.5)
	case augIncinerate: // a blast of fire
		p.boom(10, 6, 4.5)
		p.flame(10, 3, 0.6)
	case augNapalm: // a bomb with a burning fuse
		p.disc(9, 7, 4.5, 4, hex(0x2a2a38))
		p.dot(7, 5, hex(0x6a6a80))
		p.line(12, 3, 15, 1, 1, bone)
		p.flame(16, 1, 0.35)
	case augAfterburn: // a runner leaving fire
		p.runner(14, 1, dashV)
		p.flame(4, 6, 0.45)
		p.flame(8, 7, 0.35)
	case augPhoenix: // a burning bird
		p.disc(10, 6, 2, 2.5, ember)
		p.line(9, 5, 1, 1, 2, ember)
		p.line(11, 5, 18, 1, 2, ember)
		p.line(9, 5, 3, 6, 1, flameY)
		p.line(11, 5, 17, 6, 1, flameY)
		p.line(10, 8, 10, 11, 1, flameY)
		p.dot(10, 3, flameY)
		p.dot(9, 5, hex(0xffffff))

	// ── ice ──
	case augChill:
		p.flake(10, 6, 5, frost)
	case augShatter: // a frozen foe in pieces
		p.rect(5, 2, 9, 6, frost)
		p.rect(11, 1, 15, 5, frost.mul(0.8))
		p.rect(6, 8, 10, 11, frost.mul(0.9))
		p.rect(12, 7, 15, 10, frost)
		p.line(3, 0, 17, 11, 1, hex(0xffffff))
	case augPermafrost: // a block of ice, a clock in it
		p.rect(3, 1, 16, 11, frost.mul(0.55))
		p.rect(3, 1, 16, 1, frost)
		p.disc(9.5, 6.5, 3.5, 3.5, hex(0xdfe6f5))
		p.line(9.5, 6.5, 9.5, 4, 1, night)
		p.line(9.5, 6.5, 11.5, 7, 1, night)
	case augShards: // shards flying from a burst
		for k := 0; k < 6; k++ {
			a := float64(k) / 6 * 2 * math.Pi
			p.line(10+math.Cos(a)*2, 6+math.Sin(a)*1.6, 10+math.Cos(a)*8, 6+math.Sin(a)*5, 1, frost)
			p.dot(int(10+math.Cos(a)*8), int(6+math.Sin(a)*5), hex(0xffffff))
		}
		p.disc(10, 6, 1.8, 1.6, hex(0xffffff))
	case augColdSnap: // a runner arriving in frost
		p.runner(7, 1, dashV)
		p.flake(15, 6, 3.5, frost)
	case augAbsZero: // a frozen zero
		p.flake(10, 6, 6, frost)
		for gy := 0; gy < fontH; gy++ {
			for gx := 0; gx < fontW; gx++ {
				if fontPixel('0', gx, gy) {
					p.dot(9+gx, 4+gy, hex(0x1a3050))
				}
			}
		}
	case augFrostbite: // jaws of ice
		for x := 3; x <= 15; x += 3 {
			p.line(float64(x), 1, float64(x)+1.5, 5, 1, frost)
			p.line(float64(x), 11, float64(x)+1.5, 7, 1, frost)
		}
		p.rect(2, 0, 17, 0, frost)
		p.rect(2, 11, 17, 11, frost)
	case augCryo: // a shield of ice
		p.disc(10, 5, 5.5, 5, frost.mul(0.7))
		p.line(5, 7, 10, 11, 2, frost.mul(0.7))
		p.line(15, 7, 10, 11, 2, frost.mul(0.7))
		p.flake(10, 5, 3, hex(0xffffff))

	// ── lightning ──
	case augStatic: // a foe crackling
		p.foe(10, 7, 3.5, steel)
		p.line(4, 2, 6, 4, 1, volt)
		p.line(16, 2, 14, 4, 1, volt)
		p.line(10, 0, 10, 2, 1, volt)
		p.dot(3, 8, volt)
		p.dot(17, 9, volt)
	case augThunder: // a bolt out of a cloud
		p.cloud(10, 2, hex(0x5a6078))
		p.bolt(9, 4, 7, volt)
	case augTesla: // a coil and its spark
		for y := 4; y <= 10; y += 2 {
			p.rect(8, y, 12, y, brass)
		}
		p.rect(9, 3, 11, 11, hex(0x8a6a30))
		p.disc(10, 2, 2, 1.8, steel)
		p.bolt(15, 0, 6, volt)
		p.bolt(3, 0, 6, volt)
	case augShort: // a broken wire, sparking
		p.line(0, 6, 7, 6, 1, steel)
		p.line(13, 6, 19, 6, 1, steel)
		p.line(8, 6, 12, 3, 1, volt)
		p.line(8, 6, 12, 9, 1, volt)
		p.line(8, 6, 13, 6, 1, hex(0xffffff))
	case augSparkGap: // a runner trailing sparks
		p.runner(14, 1, dashV)
		p.bolt(4, 1, 9, volt)
	case augConductor: // a rod and arcs off it
		p.rect(9, 1, 10, 11, brass)
		p.disc(9.5, 1, 1.5, 1.2, steel)
		p.bolt(3, 2, 8, volt)
		p.bolt(15, 2, 8, volt)
	case augSuper: // an arc half fire, half ice
		p.line(1, 6, 6, 2, 1, ember)
		p.line(6, 2, 10, 8, 1, flameY)
		p.line(10, 8, 14, 3, 1, frost)
		p.line(14, 3, 19, 7, 1, frost)
		p.flame(4, 8, 0.4)
		p.flake(16, 9, 2, frost)

	// ── poison ──
	case augLeak: // a pipe dripping
		p.rect(0, 2, 12, 4, steel)
		p.rect(12, 2, 14, 7, steel)
		p.drop(13, 9.5, 1.6, toxic)
		p.dot(5, 5, toxic)
	case augCorrode: // armour eaten through
		p.rect(3, 2, 16, 10, steel)
		p.disc(8, 6, 2.5, 2, toxic)
		p.disc(13, 4, 1.6, 1.4, toxic.mul(0.8))
		p.dot(14, 9, toxic)
	case augPlague: // poison from foe to foe
		p.foe(4, 8, 2.5, toxic.mul(0.7))
		p.foe(15, 8, 2.5, steel)
		p.line(7, 6, 12, 6, 1, toxic)
		p.arrowHead(13, 6, toxic)
		p.drop(10, 2.5, 1.2, toxic)
	case augFumes: // a cloud of poison
		p.cloud(10, 6, toxic.mul(0.85))
		p.dot(8, 5, night)
		p.dot(12, 5, night)
		p.rect(9, 8, 11, 8, night)
	case augNeuro: // a snail, poisoned
		p.disc(12, 6, 4, 3.5, toxic.mul(0.8))
		p.disc(12, 6, 1.6, 1.4, toxic.mul(0.5))
		p.rect(2, 9, 15, 10, bone.mul(0.8))
		p.line(4, 9, 3, 5, 1, bone.mul(0.8))
		p.dot(3, 4, night)
	case augDangling: // a pointer at nothing
		p.disc(4, 6, 2.5, 2.5, toxic)
		p.line(6, 6, 14, 6, 1, toxic)
		p.arrowHead(15, 6, toxic)
		p.glyph(16, 3, '?', blood)
	case augGCGas: // a canister venting
		p.rect(3, 3, 8, 11, steel)
		p.rect(4, 4, 7, 10, cyanC.mul(0.6))
		p.cloud(14, 4, toxic.mul(0.85))

	// ── the dash ──
	case augDoubleBuf: // two runners
		p.runner(7, 1, dashV.mul(0.6))
		p.runner(15, 1, dashV)
	case augGoto: // a long arrow
		p.line(1, 6, 17, 6, 2, dashV)
		p.arrowHead(19, 6, dashV)
		p.arrowHead(18, 7, dashV)
		p.streaks(6, 6, dashV)
	case augCtxSwitch: // a runner with a full magazine
		p.runner(6, 1, dashV)
		for k := 0; k < 3; k++ {
			p.rect(12+k*2, 3, 12+k*2, 10, brass)
		}
	case augDefer: // a runner, then a hard shot
		p.runner(5, 1, dashV)
		p.line(9, 5, 17, 5, 1, flameY)
		p.disc(17.5, 5, 1.6, 1.6, ember)
	case augRam: // a runner through a foe
		p.foe(13, 6, 3, steel)
		p.runner(8, 1, dashV)
		p.streaks(5, 6, dashV)
		p.dot(17, 2, flameY)
		p.dot(18, 9, flameY)

	// ── the rest ──
	case augRefCount: // a skull giving a round back
		p.disc(6, 5, 3.5, 3, bone)
		p.rect(5, 7, 7, 9, bone)
		p.dot(5, 4, night)
		p.dot(7, 4, night)
		p.line(10, 6, 13, 6, 1, leaf)
		p.arrowHead(14, 6, leaf)
		p.rect(16, 3, 17, 10, brass)
	case augRetry: // two arrows round
		p.line(2, 4, 14, 4, 1, cyanC)
		p.arrowHead(16, 4, cyanC)
		p.line(5, 8, 17, 8, 1, cyanC.mul(0.6))
		p.arrowHead(19, 8, cyanC.mul(0.6))
	case augBatch: // a slug among pellets
		for _, x := range []int{2, 5, 8} {
			p.disc(float64(x), 6, 1, 1, brass.mul(0.7))
		}
		p.rect(11, 3, 17, 9, brass)
		p.disc(17, 6, 2, 3, hex(0xd8a040))
		p.rect(11, 3, 17, 3, brass.mul(1.3))
	case augHeadroom: // a full bar and a sword
		p.rect(1, 9, 18, 11, hex(0x14261a))
		p.rect(1, 9, 16, 11, leaf)
		p.line(5, 7, 15, 1, 1, bone)
		p.line(6, 4, 8, 6, 1, brass)
	case augLastStand: // an empty bar, burning
		p.rect(1, 9, 18, 11, hex(0x2a1418))
		p.rect(1, 9, 4, 11, blood)
		p.flame(10, 3, 0.6)
	case augWeakRefs: // a magnet
		p.line(5, 1, 5, 7, 2, blood)
		p.line(14, 1, 14, 7, 2, blood)
		for x := 5; x <= 15; x++ {
			y := 7 + int(3*math.Sin(float64(x-5)/10*math.Pi))
			p.rect(x, y, x, y+1, blood)
		}
		p.rect(5, 1, 6, 2, steel)
		p.rect(14, 1, 15, 2, steel)
		p.glyph(9, 2, '1', leaf)
	case augGenGC: // a crown on a target
		p.disc(10, 7, 4.5, 4, blood)
		p.disc(10, 7, 2.8, 2.5, bone)
		p.disc(10, 7, 1.2, 1.2, blood)
		p.rect(6, 1, 14, 2, brass)
		for _, x := range []int{6, 10, 14} {
			p.dot(x, 0, brass)
		}
	case augMutex: // a padlock
		p.rect(5, 5, 14, 11, brass)
		p.line(7, 5, 7, 2, 1, steel)
		p.line(12, 5, 12, 2, 1, steel)
		p.line(7, 1, 12, 1, 1, steel)
		p.rect(9, 7, 10, 9, night)
	case augWatchdog: // a dog's head with a cross
		p.disc(8, 6, 4, 4, hex(0xb08050))
		p.disc(4.5, 3, 1.5, 2.5, hex(0x805030))
		p.dot(9, 5, night)
		p.dot(11, 7, night)
		p.rect(15, 3, 15, 9, leaf)
		p.rect(12, 6, 18, 6, leaf)
	case augVampire: // fangs and a drop
		p.rect(3, 2, 16, 4, blood.mul(0.6))
		p.line(6, 5, 7, 9, 1, bone)
		p.line(13, 5, 12, 9, 1, bone)
		p.disc(10, 9, 1.2, 1.4, blood)
	case augDaemon: // daemons circling
		for k := 0; k < 24; k++ {
			t := float64(k) / 24 * 2 * math.Pi
			p.dot(int(10+6*math.Cos(t)), int(6+5*math.Sin(t)), dashV.mul(0.5))
		}
		p.disc(10, 6, 1.5, 1.5, hex(0xffffff))
		for _, t := range []float64{0, 2.1, 4.2} {
			p.disc(10+6*math.Cos(t), 6+5*math.Sin(t), 1.5, 1.5, dashV)
		}
	case augCron: // a turret
		p.rect(6, 7, 13, 11, steel)
		p.disc(9.5, 6, 3, 2.5, steel.mul(1.2))
		p.line(11, 5, 18, 2, 1, hex(0x5a6078))
		p.dot(19, 1, flameY)
		p.dot(8, 5, cyanC)
	case augPolyglot: // the four elements
		p.flame(5, 3, 0.4)
		p.flake(15, 3, 2.5, frost)
		p.bolt(4, 6, 5, volt)
		p.drop(15, 9, 1.4, toxic)
	case augThermal: // fire meeting ice
		p.flame(6, 5, 0.6)
		p.flake(14, 6, 4, frost)
		p.line(10, 0, 10, 11, 1, hex(0xffffff))
	case augBigO: // a shot growing
		p.disc(3, 6, 1, 1, cyanC)
		p.disc(8, 6, 1.8, 1.8, cyanC)
		p.disc(15, 6, 3.5, 3.5, cyanC)
		p.dot(14, 4, hex(0xffffff))
	}
	return p.done()
}
