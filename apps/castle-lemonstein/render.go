package main

import (
	"math"
	"runtime"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// The picture is drawn into a framebuffer of pixels twice as tall as the
// terminal has rows, and handed to the terminal as half blocks: '▀' with the
// upper pixel as the foreground and the lower as the background. A cell is
// about twice as tall as it is wide, so the pixels come out square.

const (
	minW = 60
	minH = 20
	hudH = 2
)

// column is what one camera ray found.
type column struct {
	dist   float64
	u      float64
	mx, my int
	side   uint8
	tile   byte
}

var (
	fogColor = rgb{0.015, 0.02, 0.018}
	ambient  = rgb{0.1, 0.1, 0.11}
	lantern  = rgb{0.95, 0.85, 0.7}
)

// view is one frame's camera.
type view struct {
	pw, ph  int     // framebuffer size in pixels
	horizon float64 // pixel row of the horizon
	camZ    float64 // eye height, 0 floor … 1 ceiling
	focal   float64 // pixels per unit of height at distance 1
}

// render draws one frame. It allocates nothing.
func (g *game) render(b *buffer.Buffer) {
	W, H := int(b.Area.Width), int(b.Area.Height)
	if len(b.Content) < W*H {
		return
	}
	b.IsDirty = true
	cv := canvas{b: b, w: W, h: H}
	if W < minW || H < minH {
		cv.fill(0, 0, W, H, ' ', hudStyle(0x000000, 0x000000))
		cv.centered(H/2, "Castle Lemonstein needs a window of 60×20 or more", hudStyle(0xeeeeee, 0x000000))
		return
	}

	viewW, viewH := min(W, maxCols), min(H-hudH, maxRows)
	v := view{pw: viewW, ph: viewH * 2}
	v.focal = float64(v.pw) / 2 / fov
	bob := math.Sin(g.walked * 2.3 * math.Pi / 1.6)
	v.camZ = 0.5 + 0.012*bob
	v.horizon = float64(v.ph) / 2

	g.castColumns(v.pw)
	g.nearLights()
	g.drawWorld(&v)
	g.drawSprites(&v)
	menu := g.phase == phTitle || g.phase == phName
	if menu {
		g.drawTitleScene(&v)
	}
	g.drawParticles(&v)
	if g.phase == phPlay {
		g.drawGun(&v, bob)
		g.drawCrosshair(&v)
	}
	g.post(&v)

	// Shake moves the picture a pixel or two, not the world.
	sx, sy := 0, 0
	if g.shake > 0 {
		sx = int(math.Round(math.Sin(g.now*53) * g.shake * 2.5))
		sy = int(math.Round(math.Cos(g.now*61) * g.shake * 2))
	}
	g.toCells(cv, &v, viewH, sx, sy)
	if viewW < W {
		cv.fill(viewW, 0, W-viewW, viewH, ' ', hudStyle(0x000000, 0x000000))
	}
	if viewH < H-hudH {
		cv.fill(0, viewH, W, H-hudH-viewH, ' ', hudStyle(0x000000, 0x000000))
	}

	if g.showMap && g.phase == phPlay {
		g.drawMinimap(cv, &v, viewW)
	}
	if g.bossSeen && !menu {
		g.drawBossBar(cv, W)
	}
	if menu {
		cv.fill(0, H-hudH, W, hudH, ' ', hudStyle(0, 0))
	} else {
		g.drawHUD(cv, W, H)
	}
	switch g.phase {
	case phTitle:
		g.drawTitle(cv, W, viewH)
	case phName:
		g.drawNameEntry(cv, W, viewH)
	case phWon:
		if g.now-g.finished > 1.2 {
			g.drawEnd(cv, W, viewH, true)
		}
	case phDead:
		if g.now-g.finished > 0.8 {
			g.drawEnd(cv, W, viewH, false)
		}
	}
}

func (g *game) castColumns(pw int) {
	for x := 0; x < pw; x++ {
		camX := 2*(float64(x)+0.5)/float64(pw) - 1
		dx := g.dirX + g.planeX*camX
		dy := g.dirY + g.planeY*camX
		c := &g.cols[x]
		c.dist, c.mx, c.my, c.side, c.u = g.cast(g.px, g.py, dx, dy)
		if c.dist < 0.05 {
			c.dist = 0.05
		}
		c.tile = g.tile(c.mx, c.my)
	}
}

// nearLights picks the lamps close enough to matter this frame, and how
// brightly each is burning: they flicker, and now and then nearly go out.
var lightK [maxLights]float32

func (g *game) nearLights() {
	g.nNear = 0
	for i := 0; i < g.nLight; i++ {
		l := &g.lights[i]
		if math.Hypot(l.x-g.px, l.y-g.py) > 9 {
			continue
		}
		t := g.now*9 + l.seed
		k := 0.85 + 0.1*math.Sin(t) + 0.05*math.Sin(t*2.7)
		if math.Sin(g.now*0.7+l.seed*3) > 0.97 {
			k *= 0.3 + 0.7*math.Abs(math.Sin(g.now*40))
		}
		lightK[i] = float32(k) * l.power
		g.near[g.nNear] = i
		g.nNear++
	}
}

// lightAt is how much light reaches (x, y) at height z.
func (g *game) lightAt(x, y, z float64) rgb {
	l := ambient
	dx, dy := x-g.px, y-g.py
	d2 := dx*dx + dy*dy
	l = l.add(lantern.mul(float32(1.3 / (1 + d2*0.2))))
	if g.flash > 0 {
		l = l.add(rgb{1, 0.85, 0.35}.mul(float32(g.flash * 9 / (1 + d2*1.2))))
	}
	for i := 0; i < g.nNear; i++ {
		li := g.near[i]
		lt := &g.lights[li]
		dx, dy, dz := x-lt.x, y-lt.y, (z-0.85)*1.4
		k := lightK[li] / float32(1+(dx*dx+dy*dy+dz*dz)*1.5)
		l = l.add(lt.c.mul(k))
	}
	return l
}

func fogAt(d float64) float32 {
	return float32(1 - math.Exp(-d*0.075))
}

func (g *game) drawWorld(v *view) {
	pw, ph := v.pw, v.ph
	for x := 0; x < pw; x++ {
		c := &g.cols[x]
		camX := 2*(float64(x)+0.5)/float64(pw) - 1
		rdx := g.dirX + g.planeX*camX
		rdy := g.dirY + g.planeY*camX
		lineH := v.focal / c.dist
		top := v.horizon - lineH*(1-v.camZ)
		bot := v.horizon + lineH*v.camZ
		tex := wallTexture(c.tile)
		wx, wy := g.px+rdx*c.dist, g.py+rdy*c.dist
		fog := fogAt(c.dist)
		shadeK := float32(1)
		if c.side == 1 {
			shadeK = 0.82
		}
		for y := 0; y < ph; y++ {
			yc := float64(y) + 0.5
			var col rgb
			switch {
			case yc < top:
				col = g.ceilingAt(v, yc, rdx, rdy)
			case yc < bot:
				wv := (yc - top) / lineH
				col = g.wallAt(c, tex, wv, wx, wy, shadeK)
				col = col.mix(fogColor, fog)
			default:
				col = g.floorAt(v, yc, rdx, rdy)
			}
			g.fb[y*pw+x] = col
		}
	}
}

func (g *game) wallAt(c *column, tex *texture, wv, wx, wy float64, shadeK float32) rgb {
	u := c.u
	if c.tile == 'G' && g.gateLift > 0 {
		// The gate lifts into the ceiling; below it, the lair's red glow.
		wv += g.gateLift
		if wv >= 1 {
			return rgb{0.35, 0.03, 0.06}
		}
	}
	albedo, glow := tex.sample(u, wv)
	lit := albedo.times(g.lightAt(wx, wy, 1-wv)).mul(shadeK)
	if glow > 0 {
		lit = lit.add(albedo.mul(glow * 1.3))
	}
	if c.tile == 'S' {
		// Rainwater streaks the old stone.
		k := math.Floor(u * 9)
		f := u*9 - k
		seed := float64(hash3(int(k), c.mx, c.my))
		length := 0.1 + 0.55*seed + 0.04*math.Sin(g.now*1.2+k*2)
		if seed > 0.35 && f > 0.3 && f < 0.7 && wv < length {
			s := rgb{0.32, 0.42, 0.52}
			lit = lit.mix(s.mul(0.55), 0.8)
			if wv > length-0.04 {
				lit = s.mul(0.9) // the drop at the end of the streak
			}
		}
	}
	return lit
}

func (g *game) floorAt(v *view, yc, rdx, rdy float64) rgb {
	d := v.camZ * v.focal / (yc - v.horizon)
	fx, fy := g.px+rdx*d, g.py+rdy*d
	tx, ty := math.Floor(fx), math.Floor(fy)
	albedo, _ := textures[texFloor].sample(fx-tx, fy-ty)
	light := g.lightAt(fx, fy, 0)
	col := albedo.times(light)
	if hash2(int(tx)*3+11, int(ty)*5+7) < 0.16 {
		// A puddle: dark water that catches the light and ripples.
		ripple := float32(0.5 + 0.5*math.Sin(fx*9+fy*7+g.now*2.5)*math.Sin(fx*5-fy*8-g.now*1.7))
		water := rgb{0.03, 0.06, 0.08}.add(light.mul(0.12 + 0.18*ripple))
		col = col.mix(water, 0.85)
	}
	return col.mix(fogColor, fogAt(d))
}

func (g *game) ceilingAt(v *view, yc, rdx, rdy float64) rgb {
	d := (1 - v.camZ) * v.focal / (v.horizon - yc)
	fx, fy := g.px+rdx*d, g.py+rdy*d
	albedo, _ := textures[texCeil].sample(fx-math.Floor(fx), fy-math.Floor(fy))
	return albedo.times(g.lightAt(fx, fy, 1)).mix(fogColor, fogAt(d))
}

// ── sprites ──────────────────────────────────────────────────────────────

// billboard is a picture standing in the world.
type billboard struct {
	s              *sprite
	x, y           float64
	height, bottom float64 // world units: how tall, and how far off the floor
	widthK         float64 // squeeze across, for a spinning lemon
	flip           bool
	flash          float32 // white after a hit
	tint           rgb     // multiplied in; {1,1,1} for none
	unlit          bool    // gives its own light, like a lamp
}

func (g *game) drawSprites(v *view) {
	invDet := 1 / (g.planeX*g.dirY - g.dirX*g.planeY)
	n := 0
	for i := 0; i < g.nEnts; i++ {
		e := &g.ents[i]
		if e.state == stGone {
			continue
		}
		sx, sy := e.x-g.px, e.y-g.py
		depth := invDet * (-g.planeY*sx + g.planeX*sy)
		if depth < 0.2 {
			continue
		}
		g.order[n] = i
		g.depth[i] = depth
		n++
	}
	// Farthest first, so nearer sprites cover them. Insertion sort: the order
	// barely changes between frames, and sort.Slice would allocate.
	for i := 1; i < n; i++ {
		for j := i; j > 0 && g.depth[g.order[j]] > g.depth[g.order[j-1]]; j-- {
			g.order[j], g.order[j-1] = g.order[j-1], g.order[j]
		}
	}
	// Lamps hang from the ceiling; they are drawn first, being background.
	for i := 0; i < g.nLight; i++ {
		l := &g.lights[i]
		g.blit(v, invDet, billboard{s: lampArt, x: l.x, y: l.y, height: 0.2, bottom: 0.8, widthK: 1, tint: rgb{1, 1, 1}, unlit: true})
	}
	for k := 0; k < n; k++ {
		e := &g.ents[g.order[k]]
		bb := billboard{x: e.x, y: e.y, widthK: 1, tint: rgb{1, 1, 1}}
		if e.hurt > 0 {
			bb.flash = 0.8
		}
		// Which way it faces on screen: the way it is going.
		bb.flip = invDet*(g.dirY*e.vx-g.dirX*e.vy) < 0
		switch e.kind {
		case kRat, kFat:
			// Upright, a rat stands most of the way to the player's eye
			// (0.5): in sight even when it is close enough to bite.
			frames, lunge, h := &ratWalk, ratLunge, 0.7
			if e.kind == kFat {
				frames, lunge, h = &fatWalk, fatLunge, 0.8
			}
			bb.s = frames[int(e.anim)&3]
			if e.lunge > 0 {
				bb.s = lunge
			}
			bb.height = h
			if e.state == stDying {
				t := e.dying / 0.8
				bb.height = h * math.Max(0.12, t)
				bb.widthK = 1 + (1-t)*0.4
				bb.tint = rgb{1, 0.35 + 0.65*float32(t), 0.35 + 0.65*float32(t)}
			}
		case kBoss:
			bb.s = bossIdle[int(e.phase*1.8)&1]
			if e.mood == bossWindup || e.mood == bossCharge {
				bb.s = bossRage
			}
			bb.height, bb.flip = 1.25, false
			if e.state == stDying {
				t := e.dying / 2.2
				bb.bottom = -(1 - t) * 1.25 // sinks into the floor
				if int(e.dying*12)%2 == 0 {
					bb.flash = 0.5
				}
			}
		case kLemon:
			g.drawLemon(v, e, g.depth[g.order[k]], 0.32+0.04*math.Sin(e.phase*2.4))
			continue
		case kCheese:
			bb.s = cheese
			bb.height = 0.18
			bb.bottom = 0.35 + 0.05*math.Sin(e.phase*20)
		}
		if bb.s != nil {
			g.blit(v, invDet, bb)
		}
	}
}

// The lemons to be found are solid: half a lemon, cut across, turning slowly
// in the air. Every pixel it might cover casts a ray at a half ellipsoid and
// shades what it hits — the dimpled rind of the dome, or the cut face with
// its segments, pith and seeds.
const (
	lemonLong = 0.18  // half-length along the lemon: the depth of the dome
	lemonWide = 0.145 // radius across: the cut face
)

func (g *game) drawLemon(v *view, e *ent, depth, cz float64) {
	g.lemonShadow(v, e, depth)

	// The lemon's own axes in the world. It spins about the vertical, with
	// its cut face tipped up so the player sees into it every half turn.
	spin, tilt := e.phase*1.3, 0.55
	cs, sn := math.Cos(spin), math.Sin(spin)
	ct, st := math.Cos(tilt), math.Sin(tilt)
	ax := [3]float64{cs * ct, sn * ct, st} // out of the cut face
	ay := [3]float64{-sn, cs, 0}
	az := [3]float64{-st * cs, -st * sn, ct}
	dot := func(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

	o := [3]float64{g.px - e.x, g.py - e.y, v.camZ - cz}
	ol := [3]float64{dot(o, ax) / lemonLong, dot(o, ay) / lemonWide, dot(o, az) / lemonWide}
	// Light comes from the lamps around it and from the player's lantern,
	// which is where the eye is too.
	lit := g.lightAt(e.x, e.y, cz).add(rgb{0.12, 0.11, 0.1})
	// Held under white, or the face washes out and its segments vanish.
	if m := max(lit.r, lit.g, lit.b); m > 0.85 {
		lit = lit.mul(0.85 / m)
	}
	toEye := o
	if l := math.Sqrt(dot(o, o)); l > 0 {
		toEye = [3]float64{o[0] / l, o[1] / l, o[2] / l}
	}
	fog := fogAt(depth)

	invDet := 1 / (g.planeX*g.dirY - g.dirX*g.planeY)
	sx, sy := e.x-g.px, e.y-g.py
	across := invDet * (g.dirY*sx - g.dirX*sy)
	cx := float64(v.pw) / 2 * (1 + across/depth)
	cy := v.horizon + (v.camZ-cz)*v.focal/depth
	r := (lemonLong + 0.03) * v.focal / depth
	xa, xb := max(0, int(cx-r)), min(v.pw-1, int(cx+r))
	ya, yb := max(0, int(cy-r)), min(v.ph-1, int(cy+r))
	for x := xa; x <= xb; x++ {
		camX := 2*(float64(x)+0.5)/float64(v.pw) - 1
		rd := [3]float64{g.dirX + g.planeX*camX, g.dirY + g.planeY*camX, 0}
		for y := ya; y <= yb; y++ {
			// A ray whose length along the view is 1, so t is the depth.
			rd[2] = (v.horizon - (float64(y) + 0.5)) / v.focal
			dl := [3]float64{dot(rd, ax) / lemonLong, dot(rd, ay) / lemonWide, dot(rd, az) / lemonWide}
			qa := dot(dl, dl)
			qb := 2 * dot(ol, dl)
			qc := dot(ol, ol) - 1
			disc := qb*qb - 4*qa*qc
			if disc < 0 {
				continue
			}
			sq := math.Sqrt(disc)
			t1, t2 := (-qb-sq)/(2*qa), (-qb+sq)/(2*qa)
			var t float64
			face := false
			switch {
			case t1 > 0.05 && ol[0]+t1*dl[0] <= 0:
				t = t1 // the dome
			case dl[0] != 0:
				tp := -ol[0] / dl[0]
				if tp < 0.05 || tp < t1 || tp > t2 {
					continue
				}
				t, face = tp, true
			default:
				continue
			}
			if t >= g.cols[x].dist {
				continue
			}
			p := [3]float64{ol[0] + t*dl[0], ol[1] + t*dl[1], ol[2] + t*dl[2]}
			var c rgb
			var n [3]float64
			var glow float32
			if face {
				c, glow = lemonFace(p[1], p[2])
				n = ax
			} else {
				c = lemonRind(p)
				// The ellipsoid's normal, back in the world.
				gx, gy, gz := p[0]/lemonLong, p[1]/lemonWide, p[2]/lemonWide
				n = [3]float64{
					ax[0]*gx + ay[0]*gy + az[0]*gz,
					ax[1]*gx + ay[1]*gy + az[1]*gz,
					ax[2]*gx + ay[2]*gy + az[2]*gz,
				}
				l := math.Sqrt(dot(n, n))
				n = [3]float64{n[0] / l, n[1] / l, n[2] / l}
			}
			diffuse := math.Max(0, dot(n, toEye))
			col := c.times(lit).mul(float32(0.3 + 0.75*diffuse))
			if glow > 0 {
				col = col.add(c.mul(glow))
			}
			if !face {
				// A glint where the rind faces the lantern squarely.
				spec := math.Pow(diffuse, 40) * 0.3
				col = col.add(lit.mul(float32(spec)))
			}
			g.fb[y*v.pw+x] = col.mix(fogColor, fog*0.9)
		}
	}
}

// lemonRind is the colour of the rind at p, a point on the unit sphere the
// lemon was squashed from: dimpled all over, turning green towards the tip,
// where the stalk was.
func lemonRind(p [3]float64) rgb {
	base := rgb{1, 0.82, 0.12}
	k := float32(0.88 + 0.14*float64(hash2(int(p[1]*16+40), int(p[2]*16+40+p[0]*11))))
	switch {
	case p[0] < -0.96:
		base = rgb{0.2, 0.45, 0.1} // the stalk end
	case p[0] < -0.72:
		t := float32((-0.72 - p[0]) / 0.24)
		base = base.mix(rgb{0.35, 0.7, 0.15}, t)
	}
	return base.mul(k)
}

// lemonFace is the colour of the cut face at (y, z) on the unit disc, and
// how much it glows: a thin rind, the white pith, nine segments of flesh
// divided by membranes, a white core and a few seeds.
func lemonFace(y, z float64) (rgb, float32) {
	r := math.Hypot(y, z)
	switch {
	case r > 0.93:
		return rgb{1, 0.78, 0.1}, 0.05
	case r > 0.84:
		return rgb{0.98, 0.95, 0.84}, 0.12
	case r < 0.11:
		return rgb{0.98, 0.94, 0.78}, 0.14
	}
	seg := (math.Atan2(z, y)/(2*math.Pi) + 0.5) * 9
	f := seg - math.Floor(seg)
	if f < 0.07 || f > 0.93 {
		return rgb{0.98, 0.93, 0.74}, 0.14 // a membrane
	}
	if hash2(int(seg), 3) < 0.45 && math.Abs(r-0.4) < 0.07 && math.Abs(f-0.5) < 0.12 {
		return rgb{0.95, 0.88, 0.62}, 0.1 // a seed
	}
	// Juice sacs: streaks running out from the middle.
	k := float32(0.84 + 0.16*math.Sin(r*38+seg*2.3))
	return rgb{1, 0.85, 0.22}.mul(k), 0.22
}

// lemonShadow darkens the floor under a lemon, which is what makes it look
// like it hangs in the air rather than being painted on the wall behind.
func (g *game) lemonShadow(v *view, e *ent, depth float64) {
	invDet := 1 / (g.planeX*g.dirY - g.dirX*g.planeY)
	sx, sy := e.x-g.px, e.y-g.py
	across := invDet * (g.dirY*sx - g.dirX*sy)
	cx := float64(v.pw) / 2 * (1 + across/depth)
	fy := v.horizon + v.camZ*v.focal/depth
	r := lemonWide * 1.3 * v.focal / depth
	for y := max(0, int(fy-r*0.6)); y <= min(v.ph-1, int(fy+r*0.6)); y++ {
		yc := float64(y) + 0.5
		if yc <= v.horizon {
			continue
		}
		d := v.camZ * v.focal / (yc - v.horizon)
		for x := max(0, int(cx-r)); x <= min(v.pw-1, int(cx+r)); x++ {
			if g.cols[x].dist < d {
				continue
			}
			camX := 2*(float64(x)+0.5)/float64(v.pw) - 1
			wx := g.px + (g.dirX+g.planeX*camX)*d
			wy := g.py + (g.dirY+g.planeY*camX)*d
			dd := math.Hypot(wx-e.x, wy-e.y) / (lemonWide * 1.2)
			if dd < 1 {
				i := y*v.pw + x
				g.fb[i] = g.fb[i].mul(float32(0.45 + 0.55*dd*dd))
			}
		}
	}
}

func (g *game) blit(v *view, invDet float64, bb billboard) {
	sx, sy := bb.x-g.px, bb.y-g.py
	depth := invDet * (-g.planeY*sx + g.planeX*sy)
	if depth < 0.2 {
		return
	}
	across := invDet * (g.dirY*sx - g.dirX*sy)
	cx := float64(v.pw) / 2 * (1 + across/depth)
	s := bb.s
	unit := v.focal / depth
	sh := bb.height * unit
	sw := sh * float64(s.w) / float64(s.h) * bb.widthK
	bottomY := v.horizon + (v.camZ-bb.bottom)*unit
	topY := bottomY - sh
	x0 := cx - sw/2
	if sw < 0.5 || sh < 0.5 || x0 > float64(v.pw) || x0+sw < 0 {
		return
	}
	light := rgb{1, 1, 1}
	if !bb.unlit {
		// Never quite black: a creature in the dark still has an outline.
		light = g.lightAt(bb.x, bb.y, bb.bottom+bb.height/2).add(rgb{0.18, 0.17, 0.2})
	}
	light = light.times(bb.tint)
	fog := fogAt(depth)
	xa, xb := max(0, int(x0)), min(v.pw-1, int(x0+sw))
	ya, yb := max(0, int(topY)), min(v.ph-1, int(bottomY))
	for x := xa; x <= xb; x++ {
		if g.cols[x].dist < depth {
			continue
		}
		srcX := int((float64(x) + 0.5 - x0) / sw * float64(s.w))
		if srcX < 0 || srcX >= s.w {
			continue
		}
		if bb.flip {
			srcX = s.w - 1 - srcX
		}
		for y := ya; y <= yb; y++ {
			srcY := int((float64(y) + 0.5 - topY) / sh * float64(s.h))
			if srcY < 0 || srcY >= s.h {
				continue
			}
			i := srcY*s.w + srcX
			if s.a[i] == 0 {
				continue
			}
			c := s.c[i].times(light)
			if gl := s.glow[i]; gl > 0 {
				c = c.add(s.c[i].mul(gl))
			}
			if bb.flash > 0 {
				c = c.mix(rgb{1, 1, 1}, bb.flash)
			}
			g.fb[y*v.pw+x] = c.mix(fogColor, fog*0.9)
		}
	}
}

func (g *game) drawParticles(v *view) {
	invDet := 1 / (g.planeX*g.dirY - g.dirX*g.planeY)
	for i := range g.parts {
		p := &g.parts[i]
		if p.life <= 0 {
			continue
		}
		sx, sy := p.x-g.px, p.y-g.py
		depth := invDet * (-g.planeY*sx + g.planeX*sy)
		if depth < 0.15 {
			continue
		}
		across := invDet * (g.dirY*sx - g.dirX*sy)
		unit := v.focal / depth
		cx := int(float64(v.pw) / 2 * (1 + across/depth))
		cy := int(v.horizon + (v.camZ-p.z)*unit)
		r := int(p.size * unit / 2)
		if r > 3 {
			r = 3
		}
		c := p.c.times(g.lightAt(p.x, p.y, p.z)).add(p.c.mul(p.glow))
		fade := float32(math.Min(1, p.life/p.max*2.5))
		for y := cy - r; y <= cy+r; y++ {
			if y < 0 || y >= v.ph {
				continue
			}
			for x := cx - r; x <= cx+r; x++ {
				if x < 0 || x >= v.pw || g.cols[x].dist < depth {
					continue
				}
				i := y*v.pw + x
				g.fb[i] = g.fb[i].mix(c, fade)
			}
		}
	}
}

// drawGun draws the squirter, swaying as the player walks and kicking back
// when it fires.
func (g *game) drawGun(v *view, bob float64) {
	s := gunArt
	if g.flash > 0.04 {
		s = gunFire
	}
	gh := float64(v.ph) * 0.36
	scale := gh / float64(s.h)
	gw := float64(s.w) * scale
	cx := float64(v.pw)*0.56 + math.Sin(g.walked*2.3*math.Pi/3.2)*float64(v.pw)*0.025 + g.vt*float64(v.pw)*0.012
	bottom := float64(v.ph) + math.Abs(bob)*float64(v.ph)*0.025 + g.flash*float64(v.ph)*0.5
	if g.reloadT > 0 {
		// Down out of sight to be refilled, and back up.
		t := 1 - g.reloadT/reloadTime
		bottom += math.Sin(t*math.Pi) * gh * 0.8
	}
	x0, y0 := cx-gw/2, bottom-gh
	light := g.lightAt(g.px, g.py, 0.4).mul(0.9)
	for y := max(0, int(y0)); y < v.ph && float64(y) < bottom; y++ {
		srcY := int((float64(y) + 0.5 - y0) / scale)
		if srcY < 0 || srcY >= s.h {
			continue
		}
		for x := max(0, int(x0)); x < v.pw && float64(x) < x0+gw; x++ {
			srcX := int((float64(x) + 0.5 - x0) / scale)
			if srcX < 0 || srcX >= s.w {
				continue
			}
			i := srcY*s.w + srcX
			if s.a[i] == 0 {
				continue
			}
			c := s.c[i].times(light)
			if gl := s.glow[i]; gl > 0 {
				c = c.add(s.c[i].mul(gl))
			}
			g.fb[y*v.pw+x] = c
		}
	}
}

func (g *game) drawCrosshair(v *view) {
	cx, cy := v.pw/2, v.ph/2
	c := rgb{0.9, 0.9, 0.85}
	if g.hitMark > 0 {
		c = rgb{1, 0.25, 0.2}
	}
	for _, d := range [...][2]int{{-2, 0}, {2, 0}, {0, -2}, {0, 2}, {-3, 0}, {3, 0}} {
		x, y := cx+d[0], cy+d[1]
		if x >= 0 && y >= 0 && x < v.pw && y < v.ph {
			g.fb[y*v.pw+x] = c
		}
	}
}

// post darkens the corners, and tints the picture red when bitten and
// yellow when a lemon is found.
func (g *game) post(v *view) {
	if g.vigW != v.pw || g.vigH != v.ph {
		g.vigW, g.vigH = v.pw, v.ph
		for x := 0; x < v.pw; x++ {
			d := (float64(x)+0.5)/float64(v.pw)*2 - 1
			g.vigX[x] = float32(1 - 0.28*d*d)
		}
		for y := 0; y < v.ph; y++ {
			d := (float64(y)+0.5)/float64(v.ph)*2 - 1
			g.vigY[y] = float32(1 - 0.35*d*d)
		}
	}
	hurt := float32(math.Min(1, g.hurt*2.5))
	glow := float32(math.Min(1, g.pickup*2.5))
	for y := 0; y < v.ph; y++ {
		vy := g.vigY[y]
		for x := 0; x < v.pw; x++ {
			k := g.vigX[x] * vy
			i := y*v.pw + x
			c := g.fb[i].mul(k)
			if hurt > 0 {
				c = c.mix(rgb{0.7, 0.02, 0.02}, hurt*(1-k)*1.6)
			}
			if glow > 0 {
				c = c.add(rgb{0.25, 0.2, 0}.mul(glow * (1.2 - k)))
			}
			g.fb[i] = c
		}
	}
}

// toCells hands the framebuffer to the terminal as half blocks.
func (g *game) toCells(cv canvas, v *view, rows, sx, sy int) {
	w := cv.w
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < v.pw; cx++ {
			top := g.pixel(v, cx+sx, cy*2+sy)
			bot := g.pixel(v, cx+sx, cy*2+1+sy)
			st := cell.Style{Fg: cell.NewColorRGB(top.r, top.g, top.b), Bg: cell.NewColorRGB(bot.r, bot.g, bot.b)}
			r := '▀'
			if top == bot {
				r = ' '
				st.Fg = st.Bg
			}
			cv.b.Content[cy*w+cx] = cell.Cell{Content: r, Style: st}
		}
	}
}

// pixel reads the framebuffer, clamped at the edges, and rounds each
// channel (see round). Neighbouring cells then share colours more often,
// and the diff sends a quarter fewer bytes (107 → 79 KB a frame at 160×48).
func (g *game) pixel(v *view, x, y int) px {
	x = max(0, min(v.pw-1, x))
	y = max(0, min(v.ph-1, y))
	p := g.fb[y*v.pw+x].px()
	return px{round(p.r), round(p.g), round(p.b)}
}

// coarse rounds bright colours harder, for the browser. xterm.js's WebGL
// renderer draws every glyph once per pair of colours into a texture atlas,
// and a full atlas is merged and uploaded again while a frame waits: in
// Ratatui's lair, 10–60 ms. Rounding to eight everywhere gave 33,000 pairs
// over a whole run; to eight in the dark, sixteen in the middle and 32 in
// the bright, 12,400, and the picture looks the same.
const coarse = runtime.GOOS == "js"

// round rounds a channel to a multiple of eight; a multiple of sixteen
// everywhere would save more, but bands in the dark.
func round(c uint8) uint8 {
	switch {
	case !coarse || c < 48:
		return c &^ 7
	case c < 112:
		return c &^ 15
	}
	return c &^ 31
}

// ── text on top ──────────────────────────────────────────────────────────

// canvas writes cells straight into the frame's buffer.
type canvas struct {
	b    *buffer.Buffer
	w, h int
}

func hudStyle(fg, bg uint32) cell.Style {
	return cell.Style{
		Fg: cell.NewColorRGB(uint8(fg>>16), uint8(fg>>8), uint8(fg)),
		Bg: cell.NewColorRGB(uint8(bg>>16), uint8(bg>>8), uint8(bg)),
	}
}

func (c canvas) set(x, y int, r rune, s cell.Style) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.b.Content[y*c.w+x] = cell.Cell{Content: r, Style: s}
}

// text writes s from (x, y) and returns the column after it. Ranging over a
// string decodes it in place: no conversion, no allocation.
func (c canvas) text(x, y int, s string, st cell.Style) int {
	for _, r := range s {
		c.set(x, y, r, st)
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
	digits := 1
	for p := 10; p <= n; p *= 10 {
		digits++
	}
	for i := digits - 1; i >= 0; i-- {
		c.set(x+i, y, rune('0'+n%10), st)
		n /= 10
	}
	return x + digits
}

func (c canvas) fill(x0, y0, w, h int, r rune, st cell.Style) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			c.set(x, y, r, st)
		}
	}
}

func textWidth(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func (c canvas) centered(y int, s string, st cell.Style) {
	c.text((c.w-textWidth(s))/2, y, s, st)
}

const (
	hudBg    = 0x10131c
	hudText  = 0xece6d8
	hudDim   = 0x9a94a0
	hudLemon = 0xffd82a
	hudRed   = 0xe83a30
	hudGreen = 0x5cc84a
	hudPink  = 0xff84d6
)

// The map is a round window at the top right, turning with the player so
// that ahead is always up, like a car's: a few tiles all round, walls by
// their stone, the lemons, the rats and Ratatui as dots, and the player an
// arrow in the middle with the view ahead lit faintly. It is drawn in its
// own pixels, twice as tall as the rows, over the finished picture.
const (
	mmMax   = 64  // the widest the map's pixel square gets
	mmTiles = 6.5 // tiles from the middle to the rim
)

// mmRadius is the map's radius in pixels for a picture ph pixels tall.
func mmRadius(ph int) int {
	return max(9, min(mmMax/2-1, ph/7))
}

func (g *game) drawMinimap(cv canvas, v *view, viewW int) {
	r := mmRadius(v.ph)
	d := 2*r + 1
	if d+4 > viewW || d/2+3 > v.ph/2 {
		return
	}
	scale := mmTiles / float64(r) // tiles a pixel
	rx, ry := -g.dirY, g.dirX     // right of the view
	for j := 0; j < d; j++ {
		for i := 0; i < d; i++ {
			k := j*d + i
			dx, dy := float64(i-r), float64(j-r)
			dist := math.Hypot(dx, dy)
			switch {
			case dist > float64(r)+0.5:
				g.mmA[k] = 0
				continue
			case dist > float64(r)-0.9:
				g.mm[k], g.mmA[k] = hex(0xb09040), 1 // the rim
				continue
			}
			wx := g.px + (rx*dx-g.dirX*dy)*scale
			wy := g.py + (ry*dx-g.dirY*dy)*scale
			c, a := g.mmTile(int(math.Floor(wx)), int(math.Floor(wy)))
			if a < 1 && dy < 0 && math.Abs(dx) <= -dy*fov {
				c = c.add(rgb{0.06, 0.08, 0.08}) // what the player sees
			}
			g.mm[k], g.mmA[k] = c, a
		}
	}
	// The things in the castle, as dots over the tiles.
	for n := 0; n < g.nEnts; n++ {
		e := &g.ents[n]
		if e.state != stAlive {
			continue
		}
		ox, oy := e.x-g.px, e.y-g.py
		sx := (ox*rx + oy*ry) / scale
		sy := -(ox*g.dirX + oy*g.dirY) / scale
		if math.Hypot(sx, sy) > float64(r)-2.5 {
			continue
		}
		var c rgb
		size := 1
		switch e.kind {
		case kLemon:
			c = hex(0xffd82a)
		case kRat:
			c = hex(0xff3a2a)
		case kFat:
			c, size = hex(0xff7a30), 2
		case kBoss:
			c, size = hex(0xff5ad2), 3
		case kCheese:
			c = hex(0xf0c040)
		}
		cx, cy := r+int(math.Round(sx)), r+int(math.Round(sy))
		for y := cy - size/2; y < cy-size/2+max(size, 2); y++ {
			for x := cx - size/2; x < cx-size/2+max(size, 2); x++ {
				if x >= 0 && y >= 0 && x < d && y < d && g.mmA[y*d+x] > 0 {
					g.mm[y*d+x], g.mmA[y*d+x] = c, 1
				}
			}
		}
	}
	// The player: an arrow pointing up, which is ahead.
	for _, p := range [...][2]int{{0, -2}, {-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {0, 0}, {1, 0}, {-2, 1}, {2, 1}} {
		k := (r+p[1])*d + r + p[0]
		g.mm[k], g.mmA[k] = hex(0x78f0ff), 1
	}

	// Over the picture, a cell at a time: the top pixel is the cell's
	// foreground, the bottom its background.
	x0, row0 := viewW-d-2, 1
	for cy := 0; cy < (d+1)/2; cy++ {
		for i := 0; i < d; i++ {
			x, y := x0+i, row0+cy
			if x < 0 || y >= cv.h {
				continue
			}
			c := &cv.b.Content[y*cv.w+x]
			top, bot := c.Style.Fg, c.Style.Bg
			if c.Content != '▀' {
				top = bot
			}
			t := g.mmBlend(top, cy*2, i, d)
			b := g.mmBlend(bot, cy*2+1, i, d)
			st := cell.Style{Fg: cell.NewColorRGB(t.r, t.g, t.b), Bg: cell.NewColorRGB(b.r, b.g, b.b)}
			if t == b {
				*c = cell.Cell{Content: ' ', Style: cell.Style{Fg: st.Bg, Bg: st.Bg}}
			} else {
				*c = cell.Cell{Content: '▀', Style: st}
			}
		}
	}
}

// mmBlend lays map pixel (i, j) over a colour already on screen.
func (g *game) mmBlend(under cell.Color, j, i, d int) px {
	ur, ug, ub := under.RGB()
	if j >= d || g.mmA[j*d+i] == 0 {
		return px{ur, ug, ub}
	}
	u := rgb{float32(ur) / 255, float32(ug) / 255, float32(ub) / 255}
	p := u.mix(g.mm[j*d+i], g.mmA[j*d+i]).px()
	return px{round(p.r), round(p.g), round(p.b)}
}

// mmTile is the colour of tile (x, y) on the map, and how opaque it is.
func (g *game) mmTile(x, y int) (rgb, float32) {
	if !g.solid(x, y) {
		return rgb{0.05, 0.05, 0.06}, 0.82
	}
	switch g.tile(x, y) {
	case 'S':
		return hex(0x3c5a32), 1
	case 'P':
		return hex(0x465a6e), 1
	case 'L':
		return hex(0x6e2864), 1
	case 'G':
		return hex(0xc82828), 1
	}
	return hex(0x6e3c28), 1
}

func (g *game) drawBossBar(cv canvas, W int) {
	hp := 0
	if b := g.boss; b >= 0 && g.ents[b].state == stAlive {
		hp = g.ents[b].hp
	}
	barW := min(W-24, 50)
	x := (W - barW - 12) / 2
	bg := uint32(0x100808)
	cv.fill(x-2, 0, barW+15, 1, ' ', hudStyle(0, bg))
	x = cv.text(x, 0, "RATATUI ", hudStyle(hudPink, bg))
	filled := (hp*barW*2 + bossHP - 1) / bossHP // half cells
	for i := 0; i < barW; i++ {
		switch {
		case i*2+2 <= filled:
			cv.set(x+i, 0, '█', hudStyle(hudRed, bg))
		case i*2+1 <= filled:
			cv.set(x+i, 0, '▌', hudStyle(hudRed, 0x3a1414))
		default:
			cv.set(x+i, 0, ' ', hudStyle(0, 0x3a1414))
		}
	}
}

func (g *game) drawHUD(cv canvas, W, H int) {
	y := H - 2
	cv.fill(0, y, W, 2, ' ', hudStyle(hudText, hudBg))
	for x := 0; x < W; x++ {
		cv.set(x, y, '▔', hudStyle(0x3a342c, hudBg))
	}
	label := hudStyle(hudDim, hudBg)
	val := hudStyle(hudText, hudBg)

	// In a narrow window the health bar shrinks and the labels go, so the
	// juice still fits.
	hpW, gap, lemonLabel, juiceLabel := 20, 3, "LEMONS ", "JUICE "
	if W < 100 {
		hpW, gap, lemonLabel, juiceLabel = 10, 2, "", ""
	}
	x := cv.text(2, y, "HP ", label)
	hpColor := uint32(hudGreen)
	if g.hp <= 30 {
		hpColor = hudRed
	}
	step := 100 / hpW
	for i := 0; i < hpW; i++ {
		switch {
		case i*step+step <= g.hp:
			cv.set(x+i, y, '█', hudStyle(hpColor, hudBg))
		case i*step < g.hp:
			cv.set(x+i, y, '▌', hudStyle(hpColor, 0x2a2622))
		default:
			cv.set(x+i, y, ' ', hudStyle(0, 0x2a2622))
		}
	}
	x = cv.number(x+hpW+1, y, g.hp, val)
	x = cv.text(x+gap, y, lemonLabel, label)
	for i := 0; i < lemonsNeeded; i++ {
		if i < g.lemons {
			cv.set(x+i, y, '●', hudStyle(hudLemon, hudBg))
		} else {
			cv.set(x+i, y, '○', hudStyle(0x4a463e, hudBg))
		}
	}
	x = cv.number(x+lemonsNeeded+1, y, min(g.lemons, lemonsNeeded), val)
	x = cv.text(x, y, "/10", val)
	x = cv.text(x+gap, y, "RATS ", label)
	x = cv.number(x, y, g.kills, val)

	x = cv.text(x+gap, y, juiceLabel, label)
	if g.reloadT > 0 {
		done := int((1 - g.reloadT/reloadTime) * magSize)
		for i := 0; i < magSize; i++ {
			if i < done {
				cv.set(x+i, y, '▰', hudStyle(0xa08a30, hudBg))
			} else {
				cv.set(x+i, y, '▱', hudStyle(0x4a463e, hudBg))
			}
		}
		cv.text(x+magSize+1, y, "reloading", hudStyle(hudDim, hudBg))
	} else {
		ammo := uint32(hudLemon)
		if g.ammo <= 2 {
			ammo = hudRed
		}
		for i := 0; i < magSize; i++ {
			if i < g.ammo {
				cv.set(x+i, y, '▰', hudStyle(ammo, hudBg))
			} else {
				cv.set(x+i, y, '▱', hudStyle(0x4a463e, hudBg))
			}
		}
		if g.ammo == 0 && int(g.now*3)%2 == 0 {
			cv.text(x+magSize+1, y, "R reload", hudStyle(hudRed, hudBg))
		}
	}
	if g.audio == nil && W >= 110 {
		cv.text(W-9, y, "no sound", hudStyle(0x4a463e, hudBg))
	}

	switch {
	case g.msgT > 0:
		cv.centered(H-1, g.msg, hudStyle(hudLemon, hudBg))
	case g.phase == phPlay:
		cv.centered(H-1, "W/S walk · A/D strafe · ←/→ turn · SPACE squirt · R reload · M map · ESC quit", hudStyle(hudDim, hudBg))
	}
}

// box draws a double-lined panel and clears its inside.
func box(cv canvas, x, y, w, h int, fg uint32) {
	bg := uint32(0x10131c)
	st := hudStyle(fg, bg)
	cv.fill(x, y, w, h, ' ', st)
	for i := 1; i < w-1; i++ {
		cv.set(x+i, y, '═', st)
		cv.set(x+i, y+h-1, '═', st)
	}
	for j := 1; j < h-1; j++ {
		cv.set(x, y+j, '║', st)
		cv.set(x+w-1, y+j, '║', st)
	}
	cv.set(x, y, '╔', st)
	cv.set(x+w-1, y, '╗', st)
	cv.set(x, y+h-1, '╚', st)
	cv.set(x+w-1, y+h-1, '╝', st)
}

// A 5×7 pixel font, just the letters the title needs.
var titleFont = map[rune][7]string{
	'C': {".####", "#....", "#....", "#....", "#....", "#....", ".####"},
	'A': {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'S': {".####", "#....", "#....", ".###.", "....#", "....#", "####."},
	'I': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "#####"},
	'L': {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
	'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'N': {"#...#", "##..#", "#.#.#", "#.#.#", "#..##", "#...#", "#...#"},
	'H': {"#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
}

// drawTitleScene paints the castle's silhouette beneath the gold title.
// The geometry uses the existing framebuffer and allocates nothing per frame.
func (g *game) drawTitleScene(v *view) {
	for y := 0; y < v.ph; y++ {
		for x := 0; x < v.pw; x++ {
			c := hex(0x18243b).mix(hex(0x080b13), float32(y)/float32(v.ph))
			// A pale moon above the eastern tower.
			dx, dy := float64(x-v.pw*5/6), float64(y-v.ph/5)
			r := float64(min(v.pw/12, v.ph/7))
			if dx*dx+dy*dy < r*r {
				c = hex(0xb8bdba).mul(0.8 + 0.15*hash2(x, y))
			}
			g.fb[y*v.pw+x] = c
		}
	}
	fill := func(x, y, w, h int, c rgb) {
		for py := max(0, y); py < min(v.ph, y+h); py++ {
			for px := max(0, x); px < min(v.pw, x+w); px++ {
				g.fb[py*v.pw+px] = c
			}
		}
	}
	wallY := v.ph * 3 / 5
	fill(0, wallY, v.pw, v.ph-wallY, hex(0x202635))
	for x := 0; x < v.pw; x += 8 {
		fill(x, wallY-3, 5, 4, hex(0x202635))
	}
	for _, x := range [2]int{v.pw / 10, v.pw * 8 / 10} {
		w, top := max(8, v.pw/10), v.ph/3
		fill(x, top, w, v.ph-top, hex(0x30394a))
		for bx := x; bx < x+w; bx += 5 {
			fill(bx, top-3, 3, 4, hex(0x41495a))
		}
		for y := top + 5; y < v.ph; y += 9 {
			fill(x+1, y, w-2, 1, hex(0x1c2433))
		}
		fill(x+w/2-1, top+5, 2, 5, hex(0xf0ac4e))
		fill(x+w/2-2, top+14, 4, 10, hex(0x832d43))
		fill(x+w/2-1, top+16, 2, 3, hex(0xf1ca59))
	}
	// Narrow windows use a cell title instead, leaving the menu readable.
	if v.pw < 70 || v.ph < 56 {
		return
	}
	scale := max(1, min(3, (v.pw-10)/59, (v.ph-30)/20))
	top := max(3, v.ph/14)
	g.drawTitleWord(v, "CASTLE", top, scale, hex(0xe8dcc1), hex(0xa99574))
	g.drawTitleWord(v, "LEMONSTEIN", top+9*scale, scale, hex(0xffeb99), hex(0xd59132))
}

func (g *game) drawTitleWord(v *view, word string, y0, scale int, light, dark rgb) {
	x0 := (v.pw - (len(word)*6-1)*scale) / 2
	for pass := 0; pass < 2; pass++ {
		for i, r := range word {
			glyph := titleFont[r]
			for gy := 0; gy < 7; gy++ {
				for gx := 0; gx < 5; gx++ {
					if glyph[gy][gx] != '#' {
						continue
					}
					c := light.mix(dark, float32(gy)/6)
					for sy := 0; sy < scale; sy++ {
						for sx := 0; sx < scale; sx++ {
							px, py := x0+(i*6+gx)*scale+sx, y0+gy*scale+sy
							if pass == 0 {
								px, py = px+scale, py+scale
								c = hex(0x08090e)
							}
							if px >= 0 && py >= 0 && px < v.pw && py < v.ph {
								g.fb[py*v.pw+px] = c
							}
						}
					}
				}
			}
		}
	}
}

// drawTitle writes the story and the menu under the picture, and the best
// scores beside the menu where there is room.
func (g *game) drawTitle(cv canvas, W, viewH int) {
	bg := uint32(0x10131c)
	story := max(0, viewH-13)
	if min(W, maxCols) < 70 || viewH < 28 {
		cv.centered(1, "CASTLE LEMONSTEIN", hudStyle(hudLemon, 0x080b13))
	}
	cv.centered(story, "Storm the castle. Reclaim the stolen lemons.", hudStyle(hudText, 0x000000))
	cv.centered(story+1, "Find ten to open the keep. Defeat the Rat King.", hudStyle(hudText, 0x000000))

	w, h := 46, 9
	boardW := 37
	x, y := (W-w)/2, max(0, viewH-h-1)
	entries, _, world := g.shownBoard()
	showBoard := (len(entries) > 0 || g.worldState != worldNone) && W >= w+boardW+6
	if showBoard {
		x = (W - w - boardW - 2) / 2
	}
	box(cv, x, y, w, h, 0x9b7840)
	item := func(i, row int, label string) int {
		st := hudStyle(hudDim, bg)
		mark := "  "
		if g.menu == i {
			st, mark = hudStyle(hudLemon, bg), "▶ "
		}
		cv.text(x+4, row, mark, st)
		return cv.text(x+6, row, label, st)
	}

	end := item(0, y+2, "ENTER CASTLE")
	if g.menu == 0 && int(g.now*2)%2 == 0 {
		cv.text(end+2, y+2, "ENTER", hudStyle(hudText, bg))
	}

	item(1, y+3, "KNIGHT")
	cv.text(x+16, y+3, g.playerName(), hudStyle(hudText, bg))

	item(2, y+4, "SOUND")
	sx := x + 16
	switch {
	case g.audio == nil:
		cv.text(sx, y+4, g.noSound, hudStyle(0x4a463e, bg))
	case g.volume == 0:
		cv.text(sx, y+4, "off", hudStyle(hudDim, bg))
	default:
		for i := 0; i < 10; i++ {
			if i < g.volume {
				cv.set(sx+i, y+4, '█', hudStyle(hudLemon, bg))
			} else {
				cv.set(sx+i, y+4, '░', hudStyle(0x4a463e, bg))
			}
		}
		cv.number(sx+11, y+4, g.volume*10, hudStyle(hudText, bg))
	}
	if g.menu == 2 && g.audio != nil {
		cv.text(x+w-8, y+4, "◀ ▶", hudStyle(hudDim, bg))
	}

	item(3, y+5, "QUIT")
	hint := "W/S choose · ←→ volume · ENTER select"
	cv.text(x+(w-textWidth(hint))/2, y+7, hint, hudStyle(hudDim, bg))
	if showBoard {
		bx := x + w + 2
		box(cv, bx, y, boardW, h, 0x9b7840)
		g.boardTitle(cv, bx+3, y+1, world, true)
		g.drawBoard(cv, bx+2, y+2, min(5, h-3), 0, entries)
	}
}

// boardTitle names the board shown: the world's, or this player's own, and
// with note, why: the server is being asked or has not answered.
func (g *game) boardTitle(cv canvas, x, y int, world, note bool) {
	bg := uint32(0x10131c)
	if world {
		cv.text(x, y, "WORLD BEST", hudStyle(hudLemon, bg))
		return
	}
	end := cv.text(x, y, "BEST SCORES", hudStyle(hudLemon, bg))
	if !note {
		return
	}
	switch g.worldState {
	case worldLoading:
		cv.text(end+2, y, "connecting…", hudStyle(hudDim, bg))
	case worldOffline:
		cv.text(end+2, y, "offline", hudStyle(hudDim, bg))
	}
}

// drawBoard lists the top n of entries from (x, y), one a row: place, name,
// score, time and aim. The run at place mark (1-based) is picked out.
func (g *game) drawBoard(cv canvas, x, y, n, mark int, entries []scoreEntry) {
	bg := uint32(0x10131c)
	if len(entries) == 0 {
		cv.text(x+1, y, "no scores yet", hudStyle(hudDim, bg))
		return
	}
	for i := 0; i < min(n, len(entries)); i++ {
		e := &entries[i]
		st, dim := hudStyle(hudText, bg), hudStyle(hudDim, bg)
		if i+1 == mark {
			st, dim = hudStyle(hudLemon, bg), hudStyle(hudLemon, bg)
		}
		row := y + i
		if i+1 < 10 {
			cv.number(x+1, row, i+1, dim)
		} else {
			cv.number(x, row, i+1, dim)
		}
		cv.text(x+3, row, e.Name, st)
		cv.numberRight(x+21, row, e.Score, st)
		drawTime(cv, x+23, row, e.Secs, dim)
		cv.text(cv.numberRight(x+32, row, e.Aim, dim), row, "%", dim)
	}
}

// drawTime writes secs as m:ss and returns the column after it.
func drawTime(cv canvas, x, y, secs int, st cell.Style) int {
	x = cv.number(x, y, secs/60, st)
	x = cv.text(x, y, ":", st)
	if secs%60 < 10 {
		x = cv.text(x, y, "0", st)
	}
	return cv.number(x, y, secs%60, st)
}

// numberRight writes n so that it ends just before column end, and returns
// end.
func (c canvas) numberRight(end, y, n int, st cell.Style) int {
	digits := 1
	for p := 10; p <= n; p *= 10 {
		digits++
	}
	c.number(end-digits, y, n, st)
	return end
}

// drawNameEntry asks for the player's name, which goes on the leaderboard.
func (g *game) drawNameEntry(cv canvas, W, viewH int) {
	if min(W, maxCols) < 70 || viewH < 28 {
		cv.centered(1, "CASTLE LEMONSTEIN", hudStyle(hudLemon, 0x080b13))
	}
	bg := uint32(0x10131c)
	w, h := 46, 8
	x, y := (W-w)/2, max(0, viewH-h-1)
	box(cv, x, y, w, h, 0x9b7840)
	cv.centered(y+2, "NAME YOUR KNIGHT", hudStyle(hudLemon, bg))
	fx := x + (w-nameMax-2)/2
	cv.fill(fx, y+4, nameMax+2, 1, ' ', hudStyle(hudText, 0x2a2622))
	end := cv.text(fx+1, y+4, g.typing, hudStyle(hudText, 0x2a2622))
	if int(g.now*2)%2 == 0 {
		cv.set(end, y+4, '▏', hudStyle(hudLemon, 0x2a2622))
	}
	cv.centered(y+6, "type it · ENTER save · it goes on the leaderboard", hudStyle(hudDim, bg))
}

// drawEnd is the screen after a run: how it went, what it scored and where
// that puts it among the best.
func (g *game) drawEnd(cv canvas, W, viewH int, won bool) {
	w, h := 60, 17
	x, y := (W-w)/2, max(0, (viewH-h)/2)
	bg := uint32(0x10131c)
	dim, val := hudStyle(hudDim, bg), hudStyle(hudText, bg)
	if won {
		box(cv, x, y, w, h, 0xf0b830)
		cv.centered(y+1, "_/\\_/\\_/\\_", hudStyle(0xf0b830, bg))
		cv.centered(y+2, "THE CASTLE IS LIBERATED", hudStyle(hudLemon, bg))
		cv.centered(y+3, "Ratatui has fallen. The lemons are yours.", val)
	} else {
		box(cv, x, y, w, h, hudRed)
		cv.centered(y+2, "YOU FELL IN THE CASTLE", hudStyle(hudRed, bg))
		cv.centered(y+3, "The keep still stands. Rise and try again.", dim)
	}

	r := &g.last
	lx, row := x+4, y+5
	cx := cv.text(lx, row, "time ", dim)
	cx = drawTime(cv, cx, row, r.entry.Secs, val)
	cx = cv.text(cx+3, row, "aim ", dim)
	cx = cv.text(cv.number(cx, row, r.entry.Aim, val), row, "%", val)
	cx = cv.text(cx+3, row, "rats ", dim)
	cx = cv.number(cx, row, g.kills, val)
	cx = cv.text(cx+3, row, "lemons ", dim)
	cv.number(cx, row, min(g.lemons, lemonsNeeded), val)

	row++
	cx = cv.text(lx, row, "SCORE ", hudStyle(hudLemon, bg))
	cx = cv.number(cx, row, r.entry.Score, hudStyle(hudLemon, bg))
	cx = cv.text(cx+2, row, "= ", dim)
	cx = cv.number(cx, row, r.aim, val)
	cx = cv.text(cx, row, " aim + ", dim)
	cx = cv.number(cx, row, r.time, val)
	cx = cv.text(cx, row, " time + ", dim)
	cx = cv.number(cx, row, r.rats+r.lemons, val)
	cv.text(cx, row, " finds", dim)

	row++
	entries, mark, world := g.shownBoard()
	where := " on the leaderboard"
	rank := r.rank
	if world {
		where, rank = " in the world", g.worldRank
	}
	switch {
	case g.worldState != worldNone && g.worldRank == -1:
		cv.text(lx, row, "Sending the score to the world's board…", dim)
	case rank == 1 && world:
		cv.text(lx, row, "The best score in the world!", hudStyle(hudLemon, bg))
	case rank == 1:
		cv.text(lx, row, "A new best score!", hudStyle(hudLemon, bg))
	case rank > 0:
		cx = cv.text(lx, row, "Number ", val)
		cx = cv.number(cx, row, rank, hudStyle(hudLemon, bg))
		cv.text(cx, row, where, val)
	case g.worldState == worldOffline:
		cv.text(lx, row, "The world's board did not answer", dim)
	default:
		cv.text(lx, row, "Not on the leaderboard this time", dim)
	}

	g.boardTitle(cv, lx, y+9, world, false) // the line above says how the sending went
	cv.text(lx+16, y+9, "score  time  aim", dim)
	g.drawBoard(cv, lx-1, y+10, 5, mark, entries)
	cv.centered(y+h-2, "R play again · ESC quit", dim)
}
