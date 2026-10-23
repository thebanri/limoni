package main

import "math"

// The effects the elements and the actives are seen by, in pixels rather
// than characters:
//
//   - puffs: particles with a radius, drawn as soft blobs that grow and
//     fade — a gas cloud, a flame, frost mist, a fireball;
//   - rings and zones painted on the floor itself, under everything — a
//     nova spreading, a field round the player, a vortex, a burning or a
//     poisoned patch;
//   - arcs: jagged lines of lightning with a glow, between two points in
//     the room, at their heights.
//
// All of them live in fixed pools, like the shots.

const (
	maxRings = 24
	maxArcs  = 40
	maxPuff  = 22.0 // pixels: a puff close to the eye is drawn no bigger
)

// ringFX is a ring on the floor: from r it grows by grow a second, w wide,
// fading over its life. A filled one is a disc.
type ringFX struct {
	alive        bool
	pos          vec
	r, grow, w   float64
	life, max    float64
	col          rgb
	fill, inward bool
}

func (g *game) addRing(p vec, r, grow, w, life float64, col rgb, fill bool) {
	for i := range g.rings {
		if !g.rings[i].alive {
			g.rings[i] = ringFX{alive: true, pos: p, r: r, grow: grow, w: w, life: life, max: life, col: col, fill: fill}
			return
		}
	}
}

// arcFX is lightning from a to b, at heights za and zb.
type arcFX struct {
	alive     bool
	a, b      vec
	za, zb    float64
	life, max float64
	col       rgb
	seed      uint32
	straight  bool // a laser, not lightning
}

func (g *game) addArc(a, b vec, za, zb, life float64, col rgb) {
	for i := range g.arcs {
		if !g.arcs[i].alive {
			g.arcs[i] = arcFX{alive: true, a: a, b: b, za: za, zb: zb, life: life, max: life, col: col, seed: uint32(g.rnd.next())}
			return
		}
	}
}

// addLaser is a straight line of light for a frame: a sniper's aim, a
// shield's thread.
func (g *game) addLaser(a, b vec, col rgb) {
	for i := range g.arcs {
		if !g.arcs[i].alive {
			g.arcs[i] = arcFX{alive: true, a: a, b: b, za: 0.4, zb: 0.4, life: 0.05, max: 0.05, col: col, straight: true}
			return
		}
	}
}

// puff adds a soft blob: radius r in the room, growing by grow a second,
// rising by rise.
func (g *game) puff(p vec, z, r, grow, rise, life float64, col rgb) {
	g.addPart(particle{pos: p, z: z, vz: rise, life: life, col: col, puff: r, grow: grow})
}

func (g *game) stepFX(dt float64) {
	for i := range g.rings {
		r := &g.rings[i]
		if !r.alive {
			continue
		}
		if r.life -= dt; r.life <= 0 || r.r < 0 {
			r.alive = false
			continue
		}
		r.r += r.grow * dt
	}
	for i := range g.arcs {
		a := &g.arcs[i]
		if a.alive {
			if a.life -= dt; a.life <= 0 {
				a.alive = false
			}
		}
	}
}

// ── on the floor ─────────────────────────────────────────────────────────

// decal is a ring or a zone as the floor sees it this frame.
type decal struct {
	pos      vec
	r, inner float64 // a disc out to r; a ring has the inside up to inner clear
	col      rgb
	k        float32 // how strong
	noise    bool    // a zone: blotched, and moving
	hw, hh   float64 // a stripe: half its width and height, instead of r
}

// gatherDecals collects this frame's rings and zones near enough to see.
func (g *game) gatherDecals(pos vec) {
	ds := sc.decals[:0]
	for i := range g.zones {
		z := &g.zones[i]
		if !z.alive || len(ds) == cap(ds) || z.pos.dist2(pos) > 30*30 {
			continue
		}
		ds = append(ds, decal{pos: z.pos, r: z.r, col: hex(elementTint[z.el]), k: float32(0.4 * math.Min(1, z.life*2)), noise: true})
	}
	for i := range g.rings {
		r := &g.rings[i]
		if !r.alive || len(ds) == cap(ds) || r.r <= 0 {
			continue
		}
		inner := math.Max(0, r.r-r.w)
		if r.fill {
			inner = 0
		}
		ds = append(ds, decal{pos: r.pos, r: r.r, inner: inner, col: r.col, k: float32(0.9 * r.life / r.max)})
	}
	ds = g.hazardDecals(ds)
	if w := &g.wave; w.alive && len(ds) < cap(ds) {
		ds = append(ds, decal{pos: w.pos, r: w.r + 0.15, inner: w.r - 0.15, col: hex(0xff7a2a), k: 1})
	}
	sc.decals = ds
}

// floorDecals lights floor point (wx, wy) with the decals over it; foot is
// a pixel's footprint there, so a thin ring holds together far away.
func (g *game) floorDecals(c rgb, wx, wy, foot float64) rgb {
	for i := range sc.decals {
		d := &sc.decals[i]
		dx, dy := wx-d.pos.x, wy-d.pos.y
		if d.hw > 0 {
			ex, ey := math.Abs(dx)-d.hw, math.Abs(dy)-d.hh
			if ex > foot || ey > foot {
				continue
			}
			// Filled, its edge drawn brighter.
			edge := float32(math.Max(0, 1-math.Min(-ex, -ey)/0.25))
			c = c.mix(d.col, d.k*(0.5+0.5*edge))
			continue
		}
		r := d.r + foot
		d2 := dx*dx + dy*dy
		if d2 > r*r {
			continue
		}
		dist := math.Sqrt(d2)
		k := d.k
		switch {
		case d.noise:
			// A patch: blotched, the blotches drifting, brightest at its edge.
			n := hash2(int(wx*3+g.clock*2), int(wy*3-g.clock))
			edge := float32(math.Max(0, 1-math.Abs(dist-d.r*0.85)/(d.r*0.3)))
			k *= 0.35 + 0.45*n + 0.4*edge
		case d.inner > 0:
			if dist < d.inner-foot {
				continue
			}
			// The ring's edge fades to both sides.
			mid := (d.r + d.inner) / 2
			half := math.Max((d.r-d.inner)/2, foot)
			k *= float32(math.Max(0, 1-math.Abs(dist-mid)/half))
		default:
			k *= float32(0.4 + 0.6*dist/d.r) // a disc: brighter at its rim
		}
		c = c.mix(d.col, k)
	}
	return c
}

// ── in the air ───────────────────────────────────────────────────────────

// softBlob draws a puff: a disc r pixels across that fades to its edge,
// laid over what is there by alpha.
func (g *game) softBlob(v *view, sx, cy, r, depth float64, col rgb, alpha float32) {
	r = math.Min(r, maxPuff)
	if r < 0.6 || alpha <= 0.01 {
		return
	}
	// A cell at a time: both its pixels read and written once.
	row0, row1 := max(int(cy-r)/2, v.top), min(int(cy+r)/2, v.top+v.h-1)
	for x := max(int(sx-r), 0); x <= min(int(sx+r), v.w-1); x++ {
		if depth >= sc.zbuf[x] {
			continue
		}
		dx := (float64(x) + 0.5 - sx) / r
		dx2 := dx * dx
		if dx2 >= 1 {
			continue
		}
		for row := row0; row <= row1; row++ {
			dt := (float64(2*row) + 0.5 - cy) / r
			db := (float64(2*row) + 1.5 - cy) / r
			kt, kb := float32(1-dx2-dt*dt), float32(1-dx2-db*db)
			if kt <= 0 && kb <= 0 {
				continue
			}
			p := v.c.at(x, row)
			if p == nil {
				continue
			}
			top, bot := halves(p)
			if kt > 0 {
				top = top.mix(col, alpha*kt)
			}
			if kb > 0 {
				bot = bot.mix(col, alpha*kb)
			}
			setHalves(p, top, bot)
		}
	}
}

// blendPx lays col over pixel (x, py) by k.
func blendPx(v *view, x, py int, col rgb, k float32) {
	if x < 0 || x >= v.w || py < 2*v.top || py >= 2*(v.top+v.h) {
		return
	}
	p := v.c.at(x, py/2)
	if p == nil {
		return
	}
	top, bot := halves(p)
	if py%2 == 0 {
		top = top.mix(col, k)
	} else {
		bot = bot.mix(col, k)
	}
	setHalves(p, top, bot)
}

// drawArcs draws the lightning: each arc a jagged line between its ends,
// re-jagged twenty times a second, white at its core and its colour round
// it.
func (g *game) drawArcs(v *view, pos vec) {
	for i := range g.arcs {
		a := &g.arcs[i]
		if !a.alive {
			continue
		}
		ax, ad, arows, ok1 := v.project(a.a, pos)
		bx, bd, brows, ok2 := v.project(a.b, pos)
		if !ok1 || !ok2 {
			continue
		}
		ay := 2 * (v.horizon + 0.5*arows - a.za*arows)
		by := 2 * (v.horizon + 0.5*brows - a.zb*brows)
		depth := math.Min(ad, bd)
		fade := float32(a.life / a.max)
		const segs = 7
		seed := a.seed + uint32(g.clock*20)
		px, py := ax, ay
		span := math.Hypot(bx-ax, by-ay)
		for s := 1; s <= segs; s++ {
			t := float64(s) / segs
			nx, ny := ax+(bx-ax)*t, ay+(by-ay)*t
			if s < segs && !a.straight {
				j := (float64(hash2(int(seed), s)) - 0.5) * span * 0.18
				// across the line
				nx += -(by - ay) / span * j
				ny += (bx - ax) / span * j
			}
			g.glowLine(v, px, py, nx, ny, depth, a.col.mix(rgb{1, 1, 1}, 0.6).mul(fade), a.col.mul(fade*0.5))
			px, py = nx, ny
		}
	}
}

// glowLine draws a line of pixels, core in its colour and a halo above and
// below it.
func (g *game) glowLine(v *view, x0, y0, x1, y1, depth float64, core, halo rgb) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	n = min(n, 160)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, py := int(x0+(x1-x0)*t), int(y0+(y1-y0)*t)
		if x < 0 || x >= v.w || depth >= sc.zbuf[x] || py < 2*v.top || py >= 2*(v.top+v.h) {
			continue
		}
		v.c.px(x, py, core)
		for _, dy := range [2]int{-1, 1} {
			q := py + dy
			if q < 2*v.top || q >= 2*(v.top+v.h) {
				continue
			}
			if p := v.c.at(x, q/2); p != nil {
				top, bot := halves(p)
				if q%2 == 0 {
					top = top.add(halo)
				} else {
					bot = bot.add(halo)
				}
				setHalves(p, top, bot)
			}
		}
	}
}
