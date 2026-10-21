package main

import (
	"math"
	"slices"

	"github.com/thebanri/limoni/core/buffer"
)

// The picture is a raycaster drawn in characters: a column of cells per
// ray, a wall's texture sampled glyph by glyph, near walls readable and far
// ones shaded with a ramp, and every sprite is art scaled by picking the
// nearest character. A frame allocates nothing; its scratch is sized once.

const maxCols = 512

// Walls are taller than the eye is high: windows, not fences.
const (
	wallH = 1.6
	eye   = 0.5
)

var ramp = []rune(" .:-=+*#%@")

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
	hud := 3
	v := view{c: c, w: w, top: 1, h: h - hud}
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
		v.grey = float32(0.75 * math.Sin(math.Pi*(1-g.reloadT/g.st.reload)))
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

// stretches reports whether a glyph keeps going when it is drawn wider or
// taller than a cell: lines and blocks do, letters do not.
func stretches(r rune) (horizontal, vertical bool) {
	switch r {
	case '─', '═', '▔', '▁', '▀', '▄', '━', '~':
		return true, false
	case '│', '║', '▕', '▏', '▌', '▐', '┃':
		return false, true
	case '█', '▓', '▒', '░':
		return true, true
	}
	return false, false
}

func fog(d float64) float32 { return float32(1 / (1 + d*d*0.018)) }

func (g *game) drawWorld(v *view, pos vec) {
	c := v.c
	for x := 0; x < v.w; x++ {
		camX := 2*(float64(x)+0.5)/float64(v.w) - 1
		rx, ry := v.dirX+v.plX*camX, v.dirY+v.plY*camX
		h := cast(pos.x, pos.y, rx, ry)
		if h.dist < 0.05 {
			h.dist = 0.05
		}
		sc.zbuf[x] = h.dist
		unit := v.f * 0.5 / h.dist // rows a world unit takes at this distance
		lineH := unit * wallH
		bot := v.horizon + unit*eye
		top := bot - lineH
		tex := wallTexture(h.tile)
		u := h.along/texSpan - math.Floor(h.along/texSpan)
		// Text on a wall reads left to right from where it is seen.
		if h.side == 0 && rx < 0 || h.side == 1 && ry > 0 {
			u = 1 - u
		}
		tx := int(u * texW)
		if tx >= texW {
			tx = texW - 1
		}
		// A character wider than a column is drawn once, in its first
		// column: near text reads spaced out instead of doubled.
		repeatX := x > 0 && sc.wallTx[x-1] == tx && sc.wallID[x-1] == h.tile && math.Abs(sc.zbuf[x-1]-h.dist) < 0.5
		sc.wallTx[x], sc.wallID[x] = tx, h.tile
		shade := fog(h.dist) * v.dim
		if h.side == 1 {
			shade *= 0.82
		}
		bg := tex.bg
		if h.tile == 'P' {
			bg = bg.mix(hex(0x5a0b3a), float32(0.5+0.5*math.Sin(g.clock*4)))
		}
		near := lineH >= texH*0.7
		prevTy := -1
		for y := v.top; y < v.top+v.h; y++ {
			fy := float64(y) + 0.5
			switch {
			case fy < top:
				g.ceiling(v, x, y, fy)
			case fy < bot:
				vv := (fy - top) / lineH
				ty := int(vv * texH)
				if ty >= texH {
					ty = texH - 1
				}
				r := tex.glyph[ty][tx]
				fg := tex.col[ty][tx]
				repeatY := ty == prevTy
				prevTy = ty
				if near {
					sh, sv := stretches(r)
					if repeatX && !sh || repeatY && !sv {
						r = ' '
					}
				} else {
					// Far away, letters would alias into noise: shade by
					// how much ink the texture has there instead.
					k := tex.ink[ty][tx]
					if ty == 0 || ty == texH-1 {
						r = '─'
					} else {
						r = ramp[int(k*float32(len(ramp)-1)*0.7)]
					}
				}
				c.set(x, y, r, style(fg.mul(shade).grey(v.grey), bg.mul(shade*0.9).grey(v.grey)))
			default:
				g.floor(v, x, y, fy, rx, ry, pos)
			}
		}
	}
}

func (g *game) ceiling(v *view, x, y int, fy float64) {
	d := (v.horizon - fy) / float64(v.h) * 2 // 1 at the top, 0 at the horizon
	col := hex(0x0b0e18).mix(hex(0x03040a), float32(d)).mul(v.dim)
	v.c.set(x, y, ' ', style(col, col.grey(v.grey)))
}

func (g *game) floor(v *view, x, y int, fy, rx, ry float64, pos vec) {
	o := fy - v.horizon
	if o < 0.5 {
		o = 0.5
	}
	d := v.f * 0.25 / o
	wx, wy := pos.x+rx*d, pos.y+ry*d
	fx, fyy := wx-math.Floor(wx), wy-math.Floor(wy)
	shade := fog(d) * v.dim
	bg := hex(0x0a0d18).mul(0.5 + 0.5*shade)
	r := ' '
	fg := hex(0x26345a).mul(shade)
	// Only the corners of the floor's tiles, so the grid reads as depth
	// rather than as noise.
	edge := 0.06 + 0.01*d
	if d < 12 && (fx < edge || fx > 1-edge) && (fyy < edge || fyy > 1-edge) {
		r = '·'
	}
	v.c.set(x, y, r, style(fg.grey(v.grey), bg.grey(v.grey)))
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
	a := kd.art[int(e.t*6+e.phaseOff)%len(kd.art)]
	if e.k == kTag || e.k == kButton {
		a = kd.art[int(e.phaseOff)%len(kd.art)] // a half keeps its shape
	}
	if e.k == kSpinner {
		a = kd.art[int(e.t*14+e.phaseOff)%len(kd.art)]
	}
	z := 0.0
	if kd.act == bHop {
		z = math.Abs(math.Sin(e.t*7+e.phaseOff)) * 0.35
	}
	hRows := kd.height * rows
	bottom := v.horizon + 0.5*rows - z*rows
	top := bottom - hRows
	wCols := hRows * float64(a.w) / float64(a.h)
	if kd.act == bWeave {
		wCols *= 0.8
	}
	left := sx - wCols/2
	shade := fog(depth)
	flash := float32(math.Max(0, e.flash))
	tint := rgb{}
	if e.elite {
		tint = hex(0xffd23a)
	}
	g.blitArt(v, a, left, top, wCols, hRows, depth, shade, flash, tint)
	// A health bar over anything hurt, and every elite.
	if (e.hp < e.max || e.elite) && !kd.boss {
		g.miniBar(v, sx, top-1, wCols, depth, e.hp/e.max, e.elite, e.immune)
	}
}

// blitArt draws art over the cells from (left, top), wCols by hRows, in
// front of the walls only.
func (g *game) blitArt(v *view, a *art, left, top, wCols, hRows, depth float64, shade, flash float32, tint rgb) {
	if wCols < 1 || hRows < 1 {
		// Too far for its shape: one character, the art's most telling.
		x, y := int(left+wCols/2), int(top+hRows/2)
		if x < 0 || x >= v.w || y < v.top || y >= v.top+v.h || depth >= sc.zbuf[x] {
			return
		}
		k := a.w*(a.h/2) + a.w/2
		r := a.glyph[k]
		if r == ' ' {
			r = '•'
		}
		g.put(v, x, y, r, a.col[k].mul(shade), flash, tint)
		return
	}
	x0 := int(math.Floor(left))
	x1 := int(math.Ceil(left + wCols))
	y0 := int(math.Floor(top))
	y1 := int(math.Ceil(top + hRows))
	cw, ch := wCols/float64(a.w), hRows/float64(a.h) // cells per character
	for x := max(x0, 0); x < min(x1, v.w); x++ {
		if depth >= sc.zbuf[x] {
			continue
		}
		fx := (float64(x) + 0.5 - left) / cw
		ax := int(fx)
		if ax < 0 || ax >= a.w {
			continue
		}
		// The column a character is centred on; the others in its span
		// are drawn only by characters that stretch (lines, blocks).
		centreX := cw <= 1 || int(left+(float64(ax)+0.5)*cw) == x
		for y := max(y0, v.top); y < min(y1, v.top+v.h); y++ {
			ay := int((float64(y) + 0.5 - top) / ch)
			if ay < 0 || ay >= a.h {
				continue
			}
			k := ay*a.w + ax
			r := a.glyph[k]
			if a.bg != (rgb{}) && a.solid[k] {
				if p := v.c.at(x, y); p != nil {
					p.Content = ' '
					p.Style.Bg = a.bg.mul(shade).mix(rgb{1, 1, 1}, flash*0.3).grey(v.grey).color()
				}
			}
			if r == ' ' {
				continue
			}
			centreY := ch <= 1 || int(top+(float64(ay)+0.5)*ch) == y
			if !centreX || !centreY {
				sh, sv := stretches(r)
				if !centreX && !sh || !centreY && !sv {
					continue
				}
			}
			g.put(v, x, y, r, a.col[k].mul(shade), flash, tint)
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
	case shotBar:
		r, col = '▮', hex(0xf74c00)
	case shotPearl:
		r, col = 'o', hex(0xff8fe0)
	}
	if rows > 14 && s.kind != shotShrapnel {
		r = '●'
	}
	g.put(v, x, y, r, col.mul(0.5+0.5*fog(depth)), 0, rgb{})
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
		col := hex(0xf4f0ff)
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

var gunArt = newArt([]string{
	"    ▗▄▄▄▄▄▄▄▄▄▄▄▖    ",
	"   ▐█▀▀▀▀▀▀▀▀▀▀▀█▌   ",
	"  ▗▟█           █▙▖  ",
	" ▐██▙▄▄▄▄▄▄▄▄▄▄▄▟██▌ ",
	"▐████▀▀▀▀▀▀▀▀▀▀▀████▌",
}, []string{
	"    hhhhhhhhhhhhh    ",
	"   bbhhhhhhhhhhhbb   ",
	"  bbb           bbb  ",
	" bbbbbbbbbbbbbbbbbbb ",
	"bbbbbbbbbbbbbbbbbbbbb",
}, map[byte]uint32{'b': 0x2a3146, 'h': 0x6b7ba0})

// drawGun draws the sweeper at the bottom, with the heap on its back panel:
// rounds left, or the collector at work.
func (g *game) drawGun(v *view) {
	if v.h < 14 {
		return
	}
	gw, gh := gunArt.w, gunArt.h
	x0 := v.w/2 - gw/2 + int(math.Sin(g.bob*0.5)*1.5)
	y0 := v.top + v.h - gh + int(g.recoil*1.5)
	if g.reloadT > 0 {
		y0 += int(2 * math.Sin(math.Pi*(1-g.reloadT/g.st.reload)))
	}
	bottom := v.top + v.h
	for y := 0; y < gh; y++ {
		for x := 0; x < gw; x++ {
			k := y*gunArt.w + x
			r := gunArt.glyph[k]
			if r == ' ' || y0+y >= bottom {
				continue
			}
			if p := v.c.at(x0+x, y0+y); p != nil {
				p.Content = r
				p.Style.Fg = gunArt.col[k].grey(v.grey).color()
			}
		}
	}
	// The panel: the rounds left, in the colour of how many.
	py := y0 + 2
	if py < bottom {
		panel := style(hudCyan, hex(0x061018))
		v.c.fill(x0+5, py, gw-10, 1, ' ', panel)
		switch {
		case g.reloadT > 0:
			v.c.text(x0+gw/2-1, py, "GC", style(hudDim, hex(0x061018)))
		case g.ammo == 0:
			v.c.text(x0+gw/2-2, py, "FULL", style(hudBad, hex(0x061018)))
		default:
			col := hudCyan
			if g.ammo*4 <= g.st.mag {
				col = hudWarn
			}
			v.c.number(x0+gw/2-digits(g.ammo)/2, py, g.ammo, style(col, hex(0x061018)))
		}
	}
	if g.recoil > 0.6 {
		fx, fy := v.w/2, y0-1
		for _, d := range [5][3]int{{0, 0, '✸'}, {-1, 0, '─'}, {1, 0, '─'}, {0, -1, '│'}, {0, 1, '│'}} {
			if p := v.c.at(fx+d[0], fy+d[1]); p != nil {
				p.Content = rune(d[2])
				p.Style.Fg = hex(0xfff3b0).color()
			}
		}
	}
}

func (g *game) crosshair() (int, int) {
	if g.locked || !g.mouseIn {
		return g.viewW / 2, 1 + g.viewH/2
	}
	return g.mouseX, g.mouseY
}

func (g *game) drawCrosshair(v *view) {
	x, y := g.crosshair()
	col := hex(0xffffff)
	if g.reloadT > 0 {
		v.c.fg(x-1, y, "GC", hex(0x8fe9ff))
		return
	}
	if g.ammo == 0 {
		col = hex(0xff5a5a)
	}
	v.c.fg(x, y, "┼", col)
	v.c.fg(x-2, y, "─", col.mul(0.7))
	v.c.fg(x+2, y, "─", col.mul(0.7))
}

func (g *game) drawHurt(v *view) {
	if g.hurtFX <= 0 {
		return
	}
	k := float32(g.hurtFX) * 0.45
	red := hex(0x8a0010)
	edge := func(x, y int) {
		if p := v.c.at(x, y); p != nil {
			p.Style.Bg = red.mul(k).color()
		}
	}
	for y := v.top; y < v.top+v.h; y++ {
		edge(0, y)
		edge(v.w-1, y)
	}
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
