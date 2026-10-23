package main

import "math"

// A hazard is a place about to be hit: marked on the floor first — a ring
// with a disc filling it as the moment nears — then it goes off, hurting
// the player if they are still inside and not dashing. A notcurses bomb's
// blast, a block Textual drops, the spot Ink's Reconciler is about to land
// on.

const maxHazards = 24

type hazard struct {
	alive  bool
	pos    vec
	r      float64
	t, max float64
	dmg    float64
	src    kindID
	how    string
	owner  int    // the enemy it belongs to, or -1
	ownerK kindID // and its kind, so a slot taken since is not it
	foes   bool   // it hurts the enemies too (a bomb)
	land   bool   // its owner lands there when it goes off (a re-render)
}

func (g *game) addHazard(h hazard) {
	h.alive, h.max = true, h.t
	for i := range g.hazards {
		if !g.hazards[i].alive {
			g.hazards[i] = h
			return
		}
	}
}

// owns reports whether hazard h's owner is still the enemy that made it.
func (g *game) owns(h *hazard) (*enemy, bool) {
	if h.owner < 0 {
		return nil, false
	}
	e := &g.enemies[h.owner]
	return e, e.alive && e.k == h.ownerK
}

func (g *game) stepHazards(dt float64) {
	for i := range g.hazards {
		h := &g.hazards[i]
		if !h.alive {
			continue
		}
		if h.t -= dt; h.t > 0 {
			continue
		}
		h.alive = false
		g.goOff(h)
	}
}

func (g *game) goOff(h *hazard) {
	col := hex(kinds[h.src].accent)
	g.fireball(h.pos, h.r, col)
	g.soundFrom(sfxBoom, h.pos, 0.6)
	if e, ok := g.owns(h); ok && h.land {
		e.pos = h.pos
		g.addRing(h.pos, 0.3, 6, 0.4, 0.3, col, false)
	}
	if g.pos.dist2(h.pos) < h.r*h.r {
		g.hurt(h.dmg, h.pos, h.src, h.how)
	}
	if h.foes {
		r2 := h.r * h.r
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && i != h.owner && e.pos.dist2(h.pos) < r2 {
				g.hit(i, h.dmg, false, 1, elFire)
			}
		}
	}
	if e, ok := g.owns(h); ok && kinds[e.k].act == bBomb {
		g.kill(h.owner) // a bomb goes with its blast
	}
}

// bombGoesOff is a notcurses bomb killed: its fuse, lit or not, burns out
// at once.
func (g *game) bombGoesOff(i int) {
	e := &g.enemies[i]
	for k := range g.hazards {
		if h := &g.hazards[k]; h.alive && h.owner == i && h.ownerK == e.k {
			h.t, h.owner = 0.001, -1
			return
		}
	}
	g.addHazard(hazard{pos: e.pos, r: 1.8, t: 0.001, dmg: g.contact(e) * 0.6, src: e.k, how: "blast", owner: -1, foes: true})
}

// contact is what enemy e's bite is worth now.
func (g *game) contact(e *enemy) float64 {
	return kinds[e.k].dmg * (1 + 0.03*float64(g.level)) * math.Sqrt(g.pressure())
}

// hazardDecals puts each hazard on the floor: its ring, bright, and a disc
// filling it as the moment nears.
func (g *game) hazardDecals(ds []decal) []decal {
	for i := range g.hazards {
		h := &g.hazards[i]
		if !h.alive || len(ds)+2 > cap(ds) {
			continue
		}
		col := hex(kinds[h.src].accent).mix(hex(0xff3a3a), 0.4)
		pulse := float32(0.7 + 0.3*math.Sin(g.clock*20))
		ds = append(ds, decal{pos: h.pos, r: h.r, inner: h.r - 0.14, col: col, k: pulse})
		if fill := h.r * (1 - h.t/h.max); fill > 0.05 {
			ds = append(ds, decal{pos: h.pos, r: fill, col: col, k: 0.35})
		}
	}
	return ds
}

// marked reports whether the player stands where a hazard will go off.
func (g *game) marked() bool {
	for i := range g.hazards {
		if h := &g.hazards[i]; h.alive && g.pos.dist2(h.pos) < h.r*h.r {
			return true
		}
	}
	return false
}

// steerHoming turns a homing shot towards the player, a little at a time:
// a turn, or a dash, shakes it.
func (g *game) steerHoming(s *shot, dt float64) {
	to := g.pos.sub(s.pos)
	l := to.len()
	if l < 0.01 {
		return
	}
	speed := s.vel.len()
	want := to.scale(speed / l)
	k := math.Min(1, dt*1.6)
	s.vel = s.vel.add(want.sub(s.vel).scale(k))
	if v := s.vel.len(); v > 0 {
		s.vel = s.vel.scale(speed / v)
	}
}
