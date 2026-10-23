package main

import "math"

// Shots are bodies in the arena, not rays, so they can bounce, pierce,
// split and set each other off. The pool is fixed: a build broken enough
// to want more shots than it holds simply gets the most it can have, and
// the frame stays the same size.

type shotKind uint8

const (
	shotPlayer shotKind = iota
	shotShrapnel
	shotBar    // Ratatui's ▮
	shotPearl  // Boba Prime's pearls
	shotShard  // Ice Shards: a frozen foe's pieces, which chill
	shotRang   // Round Trip: out, and back to the player
	shotHoming // Ink's props: they turn after the player
)

const (
	maxShots   = 900
	maxParts   = 700
	maxBytes   = 400
	shotSpeed  = 32.0 // fast enough to feel like a shot, slow enough to see
	enemySpeed = 7.5
)

type shot struct {
	alive   bool
	kind    shotKind
	pos     vec
	vel     vec
	dmg     float64
	life    float64
	bounces int
	pierce  int
	gen     int // how many splits made it
	crit    bool
	id      int32
	lastHit int16
	slug    bool    // Batch Job's tenth
	back    bool    // a boomerang on its way home
	src     kindID  // who fired a hostile one
	age     float64 // seconds in flight, for Big O
}

var shotSerial int32

func (s *shot) hostile() bool {
	return s.kind == shotBar || s.kind == shotPearl || s.kind == shotHoming
}

// shoot fires a shot from p at angle a and returns its index, or -1.
func (g *game) shoot(p vec, a, dmg float64, gen int, k shotKind) int {
	i := -1
	for j := range g.shots {
		if !g.shots[j].alive {
			i = j
			break
		}
	}
	if i < 0 {
		return -1 // the pool is full: the build has broken the heap
	}
	sp := shotSpeed
	life := 1.05
	if k == shotBar || k == shotPearl {
		sp, life = enemySpeed, 3
	}
	if k == shotHoming {
		sp, life = 6, 4.5
	}
	if k == shotShrapnel || k == shotShard {
		sp, life = shotSpeed*0.8, 0.5
	}
	if k == shotRang {
		sp, life = 15, 1.8
	}
	s := &g.shots[i]
	shotSerial++
	*s = shot{alive: true, kind: k, pos: p, vel: dir(a).scale(sp), dmg: dmg, life: life, gen: gen, id: shotSerial, lastHit: -1}
	if !s.hostile() {
		s.bounces = g.st.bounces
		s.pierce = g.st.pierce
		if k == shotRang {
			s.pierce = 1 << 30 // it goes through everything, both ways
		}
		if g.rnd.f() < g.st.crit {
			s.crit = true
			s.dmg *= g.st.critMul
		}
	}
	if i >= g.shotTop {
		g.shotTop = i + 1
	}
	return i
}

func (g *game) stepShots(dt float64) {
	alive := 0
	top := 0
	for i := 0; i < g.shotTop; i++ {
		s := &g.shots[i]
		if !s.alive {
			continue
		}
		s.life -= dt
		s.age += dt
		if s.life <= 0 {
			s.alive = false
			continue
		}
		if s.kind == shotHoming {
			g.steerHoming(s, dt)
		}
		if s.kind == shotRang && s.age > 0.38 {
			// Home: straight back to the player, able to hit again.
			to := g.pos.sub(s.pos)
			if l := to.len(); l < 0.5 {
				s.alive = false
				continue
			} else {
				s.vel = to.scale(15 / l)
			}
			if !s.back {
				s.back = true
				shotSerial++
				s.id, s.lastHit = shotSerial, -1
			}
		}
		// Sub-steps of a quarter of a unit, so a fast shot does not skip
		// through a thin enemy or a wall's corner.
		move := s.vel.scale(dt)
		steps := int(move.len()/0.25) + 1
		part := move.scale(1 / float64(steps))
		for st := 0; st < steps && s.alive; st++ {
			g.advance(i, s, part)
		}
		if s.alive {
			alive++
			top = i + 1
		}
	}
	g.shotTop = top
	g.nShots = alive
	if alive > g.peakShot {
		g.peakShot = alive
	}
}

func (g *game) advance(i int, s *shot, d vec) {
	nx, ny := s.pos.x+d.x, s.pos.y+d.y
	if solid(nx, ny) {
		if s.bounces <= 0 {
			s.alive = false
			g.sparks(s.pos, 3, 0xc8d0e8)
			if !s.hostile() && s.gen == 0 {
				g.soundFrom(sfxPing, s.pos, 0.25)
			}
			return
		}
		// Reflect on the axis that was crossed.
		if solid(nx, s.pos.y) {
			s.vel.x = -s.vel.x
		}
		if solid(s.pos.x, ny) {
			s.vel.y = -s.vel.y
		}
		if !solid(nx, s.pos.y) && !solid(s.pos.x, ny) {
			s.vel = s.vel.scale(-1) // straight into a corner
		}
		s.bounces--
		s.dmg *= 1 + g.st.bounceK
		s.lastHit = -1
		s.life += 0.3
		g.sparks(s.pos, 2, 0x5ad1ff)
		return
	}
	s.pos.x, s.pos.y = nx, ny
	if s.hostile() {
		if s.pos.dist2(g.pos) < (playerR+0.15)*(playerR+0.15) {
			s.alive = false
			g.hurt(s.dmg, s.pos.sub(s.vel), s.src, attackName(s.src, s.kind))
		}
		return
	}
	// Enemies in this cell and the ones around it.
	cx, cy := int(s.pos.x), int(s.pos.y)
	for yy := cy - 1; yy <= cy+1; yy++ {
		for xx := cx - 1; xx <= cx+1; xx++ {
			if xx < 0 || yy < 0 || xx >= mapW || yy >= mapH {
				continue
			}
			for j := g.bucket[yy*mapW+xx]; j >= 0; j = g.nextInCel[j] {
				e := &g.enemies[j]
				if !e.alive || j == s.lastHit || e.lastShot == s.id {
					continue
				}
				r := kinds[e.k].r + 0.07
				if s.pos.dist2(e.pos) > r*r {
					continue
				}
				g.impact(s, int(j))
				if !s.alive {
					return
				}
			}
		}
	}
}

// impact is a shot meeting enemy j: damage, then everything the build
// adds to a hit.
func (g *game) impact(s *shot, j int) {
	e := &g.enemies[j]
	e.lastShot = s.id
	s.lastHit = int16(j)
	dmg := s.dmg * (1 + g.momentum) * (1 + g.st.bigO*s.age)
	if g.st.overflow > 0 {
		g.momentum = math.Min(0.9, g.momentum+g.st.overflow)
	}
	at := e.pos
	e.knock = e.knock.add(s.vel.scale(0.08 / (1 + kinds[e.k].r)))
	g.damage(j, dmg, s.crit)
	g.onHit(s, j)
	if s.kind == shotShard {
		g.addChill(j, 1)
	}
	if s.gen == 0 {
		g.soundFrom(sfxHit, at, 0.35)
	}
	if s.gen == 0 {
		g.sparks(s.pos, 5, 0xffd36b) // a hit throws sparks the way the shot went
	} else {
		g.sparks(s.pos, 2, 0xffffff)
	}

	// panic(): an explosion around the hit.
	if g.st.explodeR > 0 {
		g.explode(at, g.st.explodeR, s.dmg*g.st.explodeK, s.gen, j)
	}
	// goroutines: arcs to the nearest others.
	if g.st.chainN > 0 {
		g.chain(at, j, g.st.chainN, s.dmg*g.st.chainK)
	}
	// fork(): the shot splits into children, which may split again.
	if g.st.splitN > 0 && s.gen < g.st.splitGen {
		a := math.Atan2(s.vel.y, s.vel.x)
		for c := 0; c < g.st.splitN; c++ {
			off := (float64(c) - float64(g.st.splitN-1)/2) * 0.7
			g.shoot(s.pos, a+off+g.rnd.between(-0.15, 0.15), s.dmg*0.5, s.gen+1, shotPlayer)
		}
	}
	if s.pierce > 0 {
		s.pierce--
		return
	}
	s.alive = false
}

func (g *game) explode(at vec, r, dmg float64, gen, skip int) {
	g.ring(at, r)
	if gen == 0 && at.dist2(g.pos) < 64 {
		g.shake = math.Max(g.shake, 0.3)
	}
	if gen == 0 {
		g.soundFrom(sfxBoom, at, 0.45)
	}
	r2 := r * r
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive || i == skip || e.immune || e.pos.dist2(at) > r2 {
			continue
		}
		g.hit(i, dmg, false, 1, elFire)
		g.addBurn(i, g.st.napalm)
	}
	// Cascading Diff: the blast throws shrapnel, and shrapnel is a shot
	// like any other.
	if g.st.shrapnel > 0 && gen < 2 {
		for k := 0; k < g.st.shrapnel; k++ {
			g.shoot(at, g.rnd.f()*2*math.Pi, dmg*0.6, gen+1, shotShrapnel)
		}
	}
}

func (g *game) chain(from vec, skip, n int, dmg float64) {
	last := from
	done := [8]int{skip, -1, -1, -1, -1, -1, -1, -1}
	if n > 7 {
		n = 7
	}
	for k := 0; k < n; k++ {
		best, bd := -1, g.st.chainReach*g.st.chainReach
		for i := range g.enemies {
			e := &g.enemies[i]
			if !e.alive || e.immune {
				continue
			}
			seen := false
			for _, d := range done {
				if d == i {
					seen = true
				}
			}
			if seen {
				continue
			}
			if dd := e.pos.dist2(last); dd < bd {
				best, bd = i, dd
			}
		}
		if best < 0 {
			return
		}
		done[k+1] = best
		to := g.enemies[best].pos
		g.arc(last, to)
		last = to
		g.hit(best, dmg, false, 1, elShock)
		g.arcCarries(best)
	}
	g.soundFrom(sfxZap, from, 0.3)
}

// ── particles ────────────────────────────────────────────────────────────

type particle struct {
	alive bool
	pos   vec
	vel   vec
	z, vz float64
	life  float64
	max   float64
	glyph rune
	col   rgb
	num   int32 // a damage number, when not zero
	crit  bool
	tiny  bool
	puff  float64 // a soft blob this wide in the room, not a glyph (fx.go)
	grow  float64 // and how fast it widens
}

// addPart puts p in the pool, looking for a free slot from where the last
// one was found; a full pool turns it away at once.
func (g *game) addPart(p particle) {
	if g.nParts >= maxParts {
		return
	}
	for k := 0; k < maxParts; k++ {
		i := (g.partAt + k) % maxParts
		if !g.parts[i].alive {
			p.alive = true
			p.max = p.life
			g.parts[i] = p
			g.partAt = i + 1
			g.nParts++
			return
		}
	}
}

func (g *game) sparks(p vec, n int, c uint32) {
	for i := 0; i < n; i++ {
		a := g.rnd.f() * 2 * math.Pi
		g.addPart(particle{pos: p, vel: dir(a).scale(g.rnd.between(1, 3)), z: 0.45, vz: g.rnd.between(0, 2),
			life: g.rnd.between(0.15, 0.35), glyph: '·', col: hex(c), tiny: true})
	}
}

// burst breaks an enemy into the characters it was drawn with.
func (g *game) burst(e *enemy) {
	kd := &kinds[e.k]
	bits := kd.bits
	n := 6
	if kd.boss {
		n = 40
	}
	for i := 0; i < n; i++ {
		r := bits[g.rnd.intn(len(bits))]
		ang := g.rnd.f() * 2 * math.Pi
		g.addPart(particle{pos: e.pos, vel: dir(ang).scale(g.rnd.between(1, 4)), z: kd.height * 0.5, vz: g.rnd.between(1, 4),
			life: g.rnd.between(0.5, 1.0), glyph: r, col: hex(kd.accent)})
	}
}

// ring is panic()'s blast: a fireball and a scorch spreading on the floor.
func (g *game) ring(at vec, r float64) {
	g.fireball(at, r, hex(0xffb347))
}

// fireball is a blast of colour col, r across: a ring on the floor and a
// ball of puffs.
func (g *game) fireball(at vec, r float64, col rgb) {
	g.addRing(at, r*0.3, r*3, 0.35, 0.3, col, false)
	n := 3 + int(r*2)
	for i := 0; i < n; i++ {
		a := g.rnd.f() * 2 * math.Pi
		p := at.add(dir(a).scale(g.rnd.f() * r * 0.4))
		g.puff(p, g.rnd.between(0.2, 0.5), r*0.25, r*0.9, 0.5, g.rnd.between(0.25, 0.4), col.mix(rgb{1, 1, 1}, 0.25))
	}
}

// arc is lightning between two points, at a body's height.
func (g *game) arc(a, b vec) {
	g.addArc(a, b, 0.45, 0.45, 0.18, hex(0x8fe9ff))
}

func (g *game) damageNumber(p vec, h float64, dmg float64, crit bool, el element) {
	if dmg < 1 {
		return
	}
	g.addPart(particle{pos: p.add(vec{g.rnd.between(-0.2, 0.2), g.rnd.between(-0.2, 0.2)}), z: h + 0.1, vz: 0.9,
		life: 0.6, num: int32(dmg + 0.5), crit: crit, col: hex(elementTint[el])})
}

func (g *game) stepParts(dt float64) {
	for i := range g.parts {
		p := &g.parts[i]
		if !p.alive {
			continue
		}
		p.life -= dt
		if p.life <= 0 {
			p.alive = false
			g.nParts--
			continue
		}
		p.pos = p.pos.add(p.vel.scale(dt))
		p.vel = p.vel.scale(math.Max(0, 1-dt*3))
		p.z += p.vz * dt
		if p.puff > 0 {
			p.puff += p.grow * dt // a puff floats, and spreads
			continue
		}
		if p.num == 0 {
			p.vz -= 9 * dt
			if p.z < 0 {
				p.z, p.vz = 0, -p.vz*0.3
			}
		}
	}
}

// ── freed bytes ──────────────────────────────────────────────────────────

// orb is experience: the bytes an enemy held, freed. The experience is the
// collector's the moment the enemy dies — a level comes at once — and the
// bytes are its picture, streaming home.
type orb struct {
	alive bool
	pos   vec
	xp    float64 // for its size
	t     float64 // for its flicker
	age   float64
}

func (g *game) dropBytes(p vec, xp float64) {
	g.gain(xp * g.st.xpK)
	for i := range g.bytes {
		if !g.bytes[i].alive {
			g.bytes[i] = orb{alive: true, pos: p, xp: xp, t: g.rnd.f() * 6}
			g.bytes[i].pos = g.bytes[i].pos.add(vec{g.rnd.between(-0.3, 0.3), g.rnd.between(-0.3, 0.3)})
			return
		}
	}
}

// stepBytes brings the freed bytes home at once, faster the longer they
// fly.
func (g *game) stepBytes(dt float64) {
	for i := range g.bytes {
		b := &g.bytes[i]
		if !b.alive {
			continue
		}
		b.age += dt
		b.t += dt
		d := g.pos.sub(b.pos)
		l := d.len()
		if l < 0.35 {
			b.alive = false
			g.soundAt(sfxByte, 0.2, 0)
			continue
		}
		speed := 6 + b.age*20
		b.pos = b.pos.add(d.scale(math.Min(1, speed*dt/l)))
	}
}
