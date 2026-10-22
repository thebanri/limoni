package main

import (
	"math"
	"slices"

	"github.com/thebanri/limoni/core/buffer"
)

// The picture is a raycaster in half-block pixels, two to a cell: a column
// per ray, walls textured with terminal windows, a polished floor and a
// ceiling. The enemies are pixel sprites over it (sprites.go), scaled by
// picking the nearest pixel, and the gun is drawn by viewmodel.go over them. A
// frame allocates nothing; its scratch is sized once.

const maxCols = 512

// Walls are taller than the eye is high: windows, not fences.
const (
	wallH = 1.6
	eye   = 0.5
)

// view is one frame's camera.
type view struct {
	c       canvas
	w, top  int // columns, and the first row of the 3D view
	h       int // rows of the 3D view
	horizon float64
	f       float64 // focal length, in columns
	dirX    float64
	dirY    float64
	plX     float64
	plY     float64
	invDet  float64
	grey    float32 // the collector's pause drains colour
	dim     float32 // 1, or less behind the title
}

type billboard struct {
	depth float64
	kind  uint8
	i     int
}

const (
	bbEnemy uint8 = iota
	bbShot
	bbPart
	bbByte
)

type scratch struct {
	zbuf   [maxCols]float64
	wallTx [maxCols]int
	wallID [maxCols]byte
	bbs    []billboard
}

var sc = scratch{bbs: make([]billboard, 0, maxEnemies+maxShots+maxParts+maxBytes)}

func (g *game) render(b *buffer.Buffer) {
	w, h := int(b.Area.Width), int(b.Area.Height)
	g.scrW, g.scrH = w, h
	c := canvas{b, w, h}
	b.Invalidate()
	if w < 20 || h < 8 {
		c.fill(0, 0, w, h, ' ', style(hex(0xffffff), hex(0)))
		c.text(0, 0, "make the window bigger", style(hex(0xffffff), hex(0)))
		return
	}
	if w > maxCols {
		w = maxCols
	}
	// The picture takes the whole window; the HUD floats over it.
	v := view{c: c, w: w, h: h}
	g.viewW, g.viewH = w, v.h
	ang := g.ang
	if g.phase == phTitle {
		ang = g.clock * 0.12
	}
	v.dirX, v.dirY = math.Cos(ang), math.Sin(ang)
	pl := math.Tan(fov / 2)
	v.plX, v.plY = -v.dirY*pl, v.dirX*pl
	v.invDet = 1 / (v.plX*v.dirY - v.dirX*v.plY)
	v.f = float64(w) / 2 / pl
	bob := 0.0
	if g.phase == phPlay {
		bob = math.Sin(g.bob) * 0.35
	}
	v.horizon = float64(v.top) + float64(v.h)/2 + bob
	v.dim = 1
	if g.phase == phTitle {
		v.dim = 0.3
	}
	if g.reloadT > 0 && g.st.reload > 0 {
		v.grey = float32(0.55 * math.Sin(math.Pi*(1-g.reloadT/g.st.reload)))
	}
	pos := g.pos
	if g.phase == phTitle {
		pos = vec{16, 16}
	}
	g.drawWorld(&v, pos)
	if g.phase != phTitle {
		g.drawSprites(&v, pos)
		g.drawGun(&v)
		g.drawCrosshair(&v)
		g.drawThreats(&v, pos)
		g.drawHurt(&v)
	}
	g.drawHUD(c, &v)
	switch g.phase {
	case phTitle:
		g.drawTitle(c)
	case phCards:
		g.drawCards(c)
	case phPause:
		g.drawPause(c)
	case phDead:
		g.drawDead(c)
	}
}

func fog(d float64) float32 { return float32(1 / (1 + d*d*0.018)) }

var fogCol = hex(0x05070e)

// fogged is c seen through d world units of the room's haze.
func fogged(c rgb, d float64) rgb { return fogCol.mix(c, fog(d)) }

// column is what one ray of the picture hit.
type column struct {
	h        hit
	tex      *wallTex
	u        float64
	top, bot float64 // the wall's rows, as fractions
	lineH    float64
	level    int // the mip level its texture is read from
	rx, ry   float64
	pulse    rgb // a spawn window's glow
}

func (g *game) drawWorld(v *view, pos vec) {
	c := v.c
	prevAlong, prevTile, prevSide := 0.0, byte(0), -1
	for x := 0; x < v.w; x++ {
		camX := 2*(float64(x)+0.5)/float64(v.w) - 1
		var col column
		col.rx, col.ry = v.dirX+v.plX*camX, v.dirY+v.plY*camX
		h := cast(pos.x, pos.y, col.rx, col.ry)
		if h.dist < 0.05 {
			h.dist = 0.05
		}
		col.h = h
		sc.zbuf[x] = h.dist
		unit := v.f * 0.5 / h.dist // rows a world unit takes at this distance
		col.lineH = unit * wallH
		col.bot = v.horizon + unit*eye
		col.top = col.bot - col.lineH
		tex := wallTexture(h.tile)
		col.tex = tex
		u := h.along/tex.span - math.Floor(h.along/tex.span)
		// Text on a wall reads left to right from where it is seen.
		if h.side == 0 && col.rx < 0 || h.side == 1 && col.ry > 0 {
			u = 1 - u
		}
		col.u = u
		// How many texels fall on a pixel, across and down: past one, read
		// a smaller level so the far walls are averaged instead of noise.
		across := h.dist / v.f
		if h.tile == prevTile && h.side == prevSide {
			across = math.Max(math.Abs(h.along-prevAlong), across)
		}
		prevAlong, prevTile, prevSide = h.along, h.tile, h.side
		tpp := math.Max(across*float64(tex.w)/tex.span, float64(tex.h)/(2*col.lineH))
		if tpp > 1.05 { // text only where a letter's pixels are pixels
			col.level, tpp = 1, tpp/2
			for tpp > 1.4 && col.level < nMip-1 {
				tpp /= 2
				col.level++
			}
		}
		if h.tile == 'P' {
			col.pulse = hex(0x5a0b3a).mul(float32(0.25 + 0.25*math.Sin(g.clock*4)))
		}
		for y := v.top; y < v.top+v.h; y++ {
			top := g.pixel(v, &col, float64(y)+0.25, pos)
			bot := g.pixel(v, &col, float64(y)+0.75, pos)
			if p := c.at(x, y); p != nil {
				setHalves(p, top.mul(v.dim).grey(v.grey), bot.mul(v.dim).grey(v.grey))
			}
		}
	}
}

// pixel is the colour at row fy of a column: ceiling, wall or floor.
func (g *game) pixel(v *view, col *column, fy float64, pos vec) rgb {
	switch {
	case fy < col.top:
		return g.ceilingAt(v, fy, col.rx, col.ry, pos)
	case fy < col.bot:
		return g.wallAt(col, (fy-col.top)/col.lineH)
	}
	c := g.floorAt(v, fy, col.rx, col.ry, pos)
	// The floor is polished: it shows the wall standing on it, fading
	// with the distance from its foot.
	if m := 2*col.bot - fy; m > col.top {
		k := float32(1-(fy-col.bot)/col.lineH) * 0.16
		w := col.tex.at(col.u, (m-col.top)/col.lineH, min(col.level+2, nMip-1))
		c = c.add(fogged(w, col.h.dist).mul(k))
	}
	return c
}

func (g *game) wallAt(col *column, vv float64) rgb {
	t := col.tex
	if t.scroll1 > t.scroll0 {
		py := vv * float64(t.h)
		if s0, s1 := float64(t.scroll0), float64(t.scroll1); py >= s0 && py < s1 {
			py = s0 + math.Mod(py-s0+g.clock*9, s1-s0)
			vv = py / float64(t.h)
		}
	}
	c := t.at(col.u, vv, col.level).add(col.pulse)
	if col.h.side == 1 {
		c = c.mul(0.8)
	}
	return fogged(c, col.h.dist)
}

func (g *game) ceilingAt(v *view, fy, rx, ry float64, pos vec) rgb {
	o := math.Max(v.horizon-fy, 0.25)
	d := v.f * 0.5 * (wallH - eye) / o
	wx, wy := pos.x+rx*d, pos.y+ry*d
	// Cable trays every second tile, and a light panel every fourth.
	fx, fyy := footprint(v, d, d*d/(v.f*(wallH-eye)), rx, ry)
	k := math.Max(seam(wx/2, fx/2, 0.012), seam(wy/2, fyy/2, 0.012)) / (1 + d*0.2)
	c := hex(0x07090f).mix(hex(0x111727), float32(k))
	ix, iy := int(math.Floor(wx)), int(math.Floor(wy))
	if ix%4 == 2 && iy%4 == 2 {
		fx, fyy := wx-float64(ix), wy-float64(iy)
		if fx > 0.2 && fx < 0.8 && fyy > 0.2 && fyy < 0.8 {
			c = hex(0x2a3654)
		}
	}
	return fogged(c, d)
}

// footprint is how far one pixel reaches across the floor or ceiling at
// distance d, along x and along y: a column steps along the camera plane,
// a half row (dd world units further) along the ray.
func footprint(v *view, d, dd, rx, ry float64) (float64, float64) {
	col := 2 * d / float64(v.w)
	return math.Max(math.Abs(rx)*dd, math.Abs(v.plX)*col), math.Max(math.Abs(ry)*dd, math.Abs(v.plY)*col)
}

// seam is how much of a pixel a grid line covers: w is the coordinate in
// grid units, foot how far it moves across the pixel, lw the line's width.
// A line thinner than its pixel fades by how little of it it fills, rather
// than breaking into dots that crawl as the view turns.
func seam(w, foot, lw float64) float64 {
	e := math.Abs(w - math.Round(w))
	reach := math.Max(lw, foot)
	return math.Max(0, 1-e/reach) * math.Min(1, lw/foot)
}

func (g *game) floorAt(v *view, fy, rx, ry float64, pos vec) rgb {
	o := math.Max(fy-v.horizon, 0.25)
	d := v.f * 0.25 / o
	wx, wy := pos.x+rx*d, pos.y+ry*d
	fx, fyy := footprint(v, d, 2*d*d/v.f, rx, ry)
	k := math.Max(seam(wx, fx, 0.03), seam(wy, fyy, 0.03)) / (1 + d*0.12)
	c := hex(0x0a0d18).mix(hex(0x24365e), float32(k))
	if w := &g.wave; w.alive {
		// Ferris's slam: a ring of fire on the floor, as wide as a pixel's
		// footprint at least, so it holds together far away.
		dx, dy := wx-w.pos.x, wy-w.pos.y
		edge := math.Abs(math.Sqrt(dx*dx+dy*dy) - w.r)
		if wide := math.Max(0.12, 1.5*math.Max(fx, fyy)); edge < wide {
			c = c.mix(hex(0xff7a2a), float32(1-edge/wide))
			return c // no fog: it must be seen
		}
	}
	return fogged(c, d)
}

// ── sprites ──────────────────────────────────────────────────────────────

func byDepth(a, b billboard) int {
	switch {
	case a.depth > b.depth:
		return -1
	case a.depth < b.depth:
		return 1
	}
	return 0
}

// project returns where world point p is on screen: its column, its depth
// and how many rows a world unit takes there.
func (v *view) project(p, pos vec) (sx, depth, rows float64, ok bool) {
	rx, ry := p.x-pos.x, p.y-pos.y
	tx := v.invDet * (v.dirY*rx - v.dirX*ry)
	ty := v.invDet * (-v.plY*rx + v.plX*ry)
	if ty < 0.15 {
		return 0, 0, 0, false
	}
	return float64(v.w) / 2 * (1 + tx/ty), ty, v.f * 0.5 / ty, true
}

func (g *game) drawSprites(v *view, pos vec) {
	bbs := sc.bbs[:0]
	add := func(p vec, k uint8, i int) {
		_, d, _, ok := v.project(p, pos)
		if ok && d < 40 {
			bbs = append(bbs, billboard{d, k, i})
		}
	}
	for i := range g.enemies {
		if g.enemies[i].alive {
			add(g.enemies[i].pos, bbEnemy, i)
		}
	}
	for i := 0; i < g.shotTop; i++ {
		if g.shots[i].alive {
			add(g.shots[i].pos, bbShot, i)
		}
	}
	for i := range g.parts {
		if g.parts[i].alive {
			add(g.parts[i].pos, bbPart, i)
		}
	}
	for i := range g.bytes {
		if g.bytes[i].alive {
			add(g.bytes[i].pos, bbByte, i)
		}
	}
	slices.SortFunc(bbs, byDepth)
	sc.bbs = bbs
	for _, bb := range bbs {
		switch bb.kind {
		case bbEnemy:
			g.drawEnemy(v, pos, &g.enemies[bb.i])
		case bbShot:
			g.drawShot(v, pos, &g.shots[bb.i])
		case bbPart:
			g.drawPart(v, pos, &g.parts[bb.i])
		case bbByte:
			g.drawByte(v, pos, &g.bytes[bb.i])
		}
	}
}

func (g *game) drawEnemy(v *view, pos vec, e *enemy) {
	kd := &kinds[e.k]
	sx, depth, rows, ok := v.project(e.pos, pos)
	if !ok {
		return
	}
	n := len(kd.spr)
	s := kd.spr[int(e.t*6+e.phaseOff)%n]
	switch e.k {
	case kTag, kButton:
		s = kd.spr[int(e.phaseOff)%n] // a half keeps its shape
	case kSpinner, kBar:
		s = kd.spr[int(e.t*14+e.phaseOff)%n]
	}
	z := 0.0
	if kd.act == bHop {
		z = math.Abs(math.Sin(e.t*7+e.phaseOff)) * 0.35
	}
	// In pixels: a column wide and half a row tall, so square.
	hPx := 2 * kd.height * rows
	wPx := hPx * float64(s.w) / float64(s.h)
	bottom := 2 * (v.horizon + 0.5*rows - z*rows)
	top := bottom - hPx
	left := sx - wPx/2
	flash := float32(math.Max(0, e.flash))
	tint, tintK := rgb{}, float32(0.3)
	switch {
	case e.frozenT > 0:
		tint, tintK = hex(0x9fe8ff), 0.6
	case e.windT > 0:
		// Winding up: it glows its colour, faster as the attack nears.
		tint = hex(kd.accent)
		flash = max(flash, float32(0.12+0.28*math.Abs(math.Sin(e.windT*18))))
	case e.burn > 0 && int(g.clock*10)%2 == 0:
		tint, tintK = hex(0xff7a2a), 0.5
	case e.poison > 0:
		tint, tintK = hex(0x8ce04a), 0.45
	case e.shockT > 0 && int(g.clock*14)%3 == 0:
		tint, tintK = hex(0xffe14a), 0.55
	case e.chill > 0:
		tint, tintK = hex(0x9fe8ff), 0.35
	case e.elite:
		tint = hex(0xffd23a)
	}
	g.blitSprite(v, s, left, top, wPx, hPx, depth, flash, tint, tintK)
	// A health bar over anything hurt, and every elite.
	if (e.hp < e.max || e.elite) && !kd.boss {
		g.miniBar(v, sx, top/2-1, wPx, depth, e.hp/e.max, e.elite, e.immune)
	}
	g.statusMarks(v, sx, top/2-2, depth, e)
}

// statusMarks writes a mark over an enemy for each element on it: ^ fire,
// * ice, ° poison, ! lightning.
func (g *game) statusMarks(v *view, sx, y, depth float64, e *enemy) {
	var marks [4]byte
	var cols [4]uint32
	n := 0
	if e.burn > 0 {
		marks[n], cols[n] = '^', elementTint[elFire]
		n++
	}
	if e.frozenT > 0 || e.chill > 0 {
		marks[n], cols[n] = '*', elementTint[elIce]
		n++
	}
	if e.poison > 0 {
		marks[n], cols[n] = 'o', elementTint[elPoison]
		n++
	}
	if e.shockT > 0 {
		marks[n], cols[n] = '!', elementTint[elShock]
		n++
	}
	yy := int(y)
	if n == 0 || yy < v.top || yy >= v.top+v.h || depth > 14 {
		return
	}
	x0 := int(sx) - n/2
	for k := 0; k < n; k++ {
		if x := x0 + k; x >= 0 && x < v.w && depth < sc.zbuf[x] {
			if p := v.c.at(x, yy); p != nil {
				p.Content = rune(marks[k])
				p.Style.Fg = hex(cols[k]).color()
			}
		}
	}
}

// blitSprite draws s over the pixels from (left, top), wPx by hPx, picking
// the nearest of its pixels, in front of the walls only.
func (g *game) blitSprite(v *view, s *sprite, left, top, wPx, hPx, depth float64, flash float32, tint rgb, tintK float32) {
	if wPx < 1 || hPx < 1 {
		wPx, hPx = math.Max(wPx, 1), math.Max(hPx, 1)
	}
	y0, y1 := max(int(math.Floor(top)), 2*v.top), min(int(math.Ceil(top+hPx)), 2*(v.top+v.h))
	for x := max(int(math.Floor(left)), 0); x < min(int(math.Ceil(left+wPx)), v.w); x++ {
		if depth >= sc.zbuf[x] {
			continue
		}
		ax := int((float64(x) + 0.5 - left) / wPx * float64(s.w))
		if ax < 0 || ax >= s.w {
			continue
		}
		for py := y0; py < y1; py++ {
			ay := int((float64(py) + 0.5 - top) / hPx * float64(s.h))
			if ay < 0 || ay >= s.h {
				continue
			}
			k := ay*s.w + ax
			if !s.on[k] {
				continue
			}
			c := s.px[k]
			if tint != (rgb{}) {
				c = c.mix(tint, tintK)
			}
			if flash > 0 {
				c = c.mix(rgb{1, 1, 1}, flash)
			}
			v.c.px(x, py, fogged(c, depth).grey(v.grey))
		}
	}
}

func (g *game) put(v *view, x, y int, r rune, col rgb, flash float32, tint rgb) {
	p := v.c.at(x, y)
	if p == nil {
		return
	}
	if tint != (rgb{}) {
		col = col.mix(tint, 0.35)
	}
	if flash > 0 {
		col = col.mix(rgb{1, 1, 1}, flash)
	}
	p.Content = r
	p.Style.Fg = col.grey(v.grey).color()
}

func (g *game) miniBar(v *view, sx, y, wCols, depth, frac float64, elite, immune bool) {
	n := int(math.Max(3, math.Min(wCols, 14)))
	x0 := int(sx) - n/2
	yy := int(y)
	if yy < v.top || yy >= v.top+v.h {
		return
	}
	on := int(math.Ceil(frac * float64(n)))
	full := hex(0x6ee07a)
	if elite {
		full = hex(0xffd23a)
	}
	if immune {
		full = hex(0x8fe9ff)
	}
	for i := 0; i < n; i++ {
		x := x0 + i
		if x < 0 || x >= v.w || depth >= sc.zbuf[x] {
			continue
		}
		col := hex(0x3a3f50)
		if i < on {
			col = full
		}
		if p := v.c.at(x, yy); p != nil {
			p.Content = '▀'
			p.Style.Fg = col.mul(fog(depth)).color()
		}
	}
}

func (g *game) drawShot(v *view, pos vec, s *shot) {
	sx, depth, rows, ok := v.project(s.pos, pos)
	if !ok {
		return
	}
	if s.hostile() {
		// What can hurt the player is the clearest thing on the screen: a
		// glowing ball of its maker's colour, whatever the distance.
		col := hex(kinds[s.src].accent)
		r := math.Min(4, math.Max(1.2, 0.1*2*rows))
		g.glowBall(v, sx, 2*(v.horizon+0.5*rows-0.42*rows), r, depth, col)
		return
	}
	if s.slug {
		g.glowBall(v, sx, 2*(v.horizon+0.5*rows-0.42*rows), math.Max(1, 0.1*2*rows), depth, hex(0xfff3b0))
		return
	}
	x := int(sx)
	y := int(v.horizon + 0.5*rows - 0.42*rows)
	if x < 0 || x >= v.w || y < v.top || y >= v.top+v.h || depth >= sc.zbuf[x] {
		return
	}
	var r rune
	var col rgb
	switch s.kind {
	case shotPlayer:
		r, col = '•', hex(0xfff3b0)
		if s.gen > 0 {
			r, col = '∙', hex(0xffd36b)
		}
		if s.crit {
			col = hex(0xff6a3d)
		}
	case shotShrapnel:
		r, col = '*', hex(0xffb347)
	case shotShard:
		r, col = '◆', hex(0x9fe8ff)
	case shotBar:
		r, col = '▮', hex(0xf74c00)
	case shotPearl:
		r, col = 'o', hex(0xff8fe0)
	}
	if rows > 14 && s.kind == shotPlayer {
		r = '●'
	}
	g.put(v, x, y, r, col.mul(0.5+0.5*fog(depth)), 0, rgb{})
}

// glowBall draws a disc of pixels centred at (sx, cy), r pixels across,
// white at its heart and its colour at its rim, with a halo.
func (g *game) glowBall(v *view, sx, cy, r, depth float64, col rgb) {
	halo := r * 1.8
	for x := max(int(sx-halo), 0); x <= min(int(sx+halo), v.w-1); x++ {
		if depth >= sc.zbuf[x] {
			continue
		}
		for py := max(int(cy-halo), 2*v.top); py <= min(int(cy+halo), 2*(v.top+v.h)-1); py++ {
			dx, dy := float64(x)+0.5-sx, (float64(py)+0.5-cy)*1.0
			d := math.Sqrt(dx*dx + dy*dy)
			switch {
			case d <= r*0.45:
				v.c.px(x, py, col.mix(rgb{1, 1, 1}, 0.75))
			case d <= r:
				v.c.px(x, py, col)
			case d <= halo:
				if p := v.c.at(x, py/2); p != nil {
					top, bot := halves(p)
					k := float32(0.55 * (1 - (d-r)/(halo-r)))
					if py%2 == 0 {
						top = top.mix(col, k)
					} else {
						bot = bot.mix(col, k)
					}
					setHalves(p, top, bot)
				}
			}
		}
	}
}

func (g *game) drawPart(v *view, pos vec, p *particle) {
	sx, depth, rows, ok := v.project(p.pos, pos)
	if !ok {
		return
	}
	x := int(sx)
	y := int(v.horizon + 0.5*rows - p.z*rows)
	if y < v.top || y >= v.top+v.h {
		return
	}
	fade := float32(p.life / p.max)
	if p.num != 0 {
		// A damage number, centred, over everything but the walls.
		n := int(p.num)
		x -= digits(n) / 2
		if x < 0 || x >= v.w || depth >= sc.zbuf[x] {
			return
		}
		col := p.col
		if p.crit {
			col = hex(0xff6a3d)
		}
		col = col.mul(0.4 + 0.6*fade)
		for i, d := 0, digits(n); i < d; i++ {
			if q := v.c.at(x+d-1-i, y); q != nil {
				q.Content = rune('0' + n%10)
				q.Style.Fg = col.color()
			}
			n /= 10
		}
		return
	}
	if x < 0 || x >= v.w || depth >= sc.zbuf[x] {
		return
	}
	g.put(v, x, y, p.glyph, p.col.mul(fade*(0.4+0.6*fog(depth))), 0, rgb{})
}

func (g *game) drawByte(v *view, pos vec, o *orb) {
	sx, depth, rows, ok := v.project(o.pos, pos)
	if !ok {
		return
	}
	x := int(sx)
	y := int(v.horizon + 0.5*rows - (0.12+0.06*math.Sin(o.t*5))*rows)
	if x < 0 || x >= v.w || y < v.top || y >= v.top+v.h || depth >= sc.zbuf[x] {
		return
	}
	r := '0'
	if int(o.t*3)%2 == 1 {
		r = '1'
	}
	if o.xp >= 5 {
		r = '◆'
	}
	g.put(v, x, y, r, hex(0x6ee07a).mul(0.4+0.6*fog(depth)), 0, rgb{})
}

// ── the player's side ────────────────────────────────────────────────────

// crosshair is the cell the shots go through: the middle of the picture.
func (g *game) crosshair() (int, int) { return g.viewW / 2, g.viewH / 2 }

// drawCrosshair draws the crosshair, opened up by the kick, with a marker
// when a shot lands and arrows while the mouse at the edge turns the view.
func (g *game) drawCrosshair(v *view) {
	x, y := g.crosshair()
	if g.reloadT > 0 && g.st.reload > 0 {
		// The collector at work: a little bar under the crosshair.
		frac := 1 - g.reloadT/g.st.reload
		v.c.fg(x-1, y, "GC", hudCyan)
		for i := 0; i < 5; i++ {
			col := hudDim
			if float64(i) < frac*5 {
				col = hudCyan
			}
			v.c.fg(x-2+i, y+1, "▀", col)
		}
		return
	}
	col := hex(0xffffff)
	if g.ammo == 0 {
		col = hudBad
	}
	gap := 1 + int(math.Max(0, g.recoil)*1.6)
	v.c.fg(x, y, "·", col)
	v.c.fg(x-gap-1, y, "─", col.mul(0.85))
	v.c.fg(x+gap+1, y, "─", col.mul(0.85))
	if g.hitT > 0 {
		hc := hex(0xffffff)
		if g.killT > 0 {
			hc = hudBad
		}
		v.c.fg(x-1, y-1, "╲", hc)
		v.c.fg(x+1, y-1, "╱", hc)
		v.c.fg(x-1, y+1, "╱", hc)
		v.c.fg(x+1, y+1, "╲", hc)
	}
	if !g.locked && g.mouseIn {
		if t := edgeTurn(g.mouseX, g.viewW); t < 0 {
			v.c.fg(x-gap-3, y, "«", hudWarn)
		} else if t > 0 {
			v.c.fg(x+gap+3, y, "»", hudWarn)
		}
	}
	if g.ammo == 0 && int(g.clock*3)%2 == 0 {
		v.c.fg(x-5, y+2, "R: COLLECT", hudBad)
	}
}

// drawHurt reddens the edges of the picture, fading inwards, and points
// round the crosshair to where the hit came from.
func (g *game) drawHurt(v *view) {
	if g.hurtShow > 1.3 {
		rel := math.Remainder(g.hurtAng-g.ang, 2*math.Pi) // 0: ahead, > 0: to the right
		cx, cy := g.crosshair()
		col := hudBad.mul(float32(0.5 + 0.5*math.Min(1, (g.hurtShow-1.3)/1.2)))
		for k := -1; k <= 1; k++ {
			a := rel + float64(k)*0.16
			x := cx + int(math.Round(math.Sin(a)*9))
			y := cy - int(math.Round(math.Cos(a)*4.5))
			v.c.fg(x, y, arrowFor(a), col)
		}
	}
	if g.dashT > 0 {
		// Speed lines along the sides while dashing.
		for y := v.top; y < v.top+v.h; y += 2 {
			n := int(g.rnd.f() * float64(v.w) / 5)
			for x := 0; x < n; x++ {
				v.c.fg(x, y, "─", hex(0xdfe6ff).mul(0.5))
				v.c.fg(v.w-1-x, y, "─", hex(0xdfe6ff).mul(0.5))
			}
		}
	}
	if g.hurtFX <= 0 {
		return
	}
	k := float32(g.hurtFX) * 0.6
	red := hex(0xa00018)
	n := max(2, v.w/24)
	for y := v.top; y < v.top+v.h; y++ {
		for i := 0; i < n; i++ {
			a := k * float32(n-i) / float32(n)
			for _, x := range [2]int{i, v.w - 1 - i} {
				if p := v.c.at(x, y); p != nil {
					top, bot := halves(p)
					if p.Content == '▀' || p.Content == ' ' {
						setHalves(p, top.mix(red, a), bot.mix(red, a))
					} else {
						p.Style.Bg = bot.mix(red, a).color()
					}
				}
			}
		}
	}
}

// arrowFor is the arrow that points away from the crosshair at angle a
// (0 up, clockwise).
func arrowFor(a float64) string {
	arrows := [8]string{"▲", "◥", "▶", "◢", "▼", "◣", "◀", "◤"}
	k := int(math.Round(a/(math.Pi/4))) % 8
	if k < 0 {
		k += 8
	}
	return arrows[k]
}

// drawThreats marks, at the edges, what is close and out of sight: in a
// terminal there is no sound to turn towards for most of the world.
func (g *game) drawThreats(v *view, pos vec) {
	var left, right float32
	var lc, rc rgb
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		d := e.pos.sub(pos)
		dist := d.len()
		if dist > 7 {
			continue
		}
		sx, _, _, ok := v.project(e.pos, pos)
		if ok && sx >= 0 && sx < float64(v.w) {
			continue // in view
		}
		k := float32(1 - dist/7)
		side := d.x*v.plX + d.y*v.plY // > 0: to the right
		if side < 0 {
			if k > left {
				left, lc = k, hex(kinds[e.k].accent)
			}
		} else if k > right {
			right, rc = k, hex(kinds[e.k].accent)
		}
	}
	mid := v.top + v.h/2
	if left > 0 {
		for dy := -1; dy <= 1; dy++ {
			v.c.fg(1, mid+dy, "◀", lc.mul(0.4+0.6*left))
		}
	}
	if right > 0 {
		for dy := -1; dy <= 1; dy++ {
			v.c.fg(v.w-2, mid+dy, "▶", rc.mul(0.4+0.6*right))
		}
	}
}
