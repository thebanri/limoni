package main

import "math"

// The elements live on the enemies as stacks: burn and poison hurt every
// half second (poison through armour), chill slows and at six stacks
// freezes, a shock makes every hit land harder. Cards add the stacks, make
// them spread as their carriers die, and set them off against each other.
// Fire, frost and poison can also lie on the floor for a while (zones).

const (
	maxBurn   = 20
	maxPoison = 12
	freezeAt  = 5
	maxZones  = 64
	dotTick   = 0.5
)

// elementTint is how an element shows: on a sprite, a number, a particle.
var elementTint = [nEl]uint32{0xf4f0ff, 0xff8a3a, 0x8fe9ff, 0xffe14a, 0x9cf05a, 0xc07bff}

// vuln is what a hit on e is multiplied by.
func (g *game) vuln(e *enemy) float64 {
	k := 1.0
	if e.shockT > 0 {
		k += g.st.shockAmp
	}
	if e.frozenT > 0 {
		k += g.st.shatter
	}
	if e.burn > 0 {
		k += g.st.overheat
	}
	if e.elite || kinds[e.k].boss {
		k += g.st.genGC
	}
	return k
}

func (g *game) addBurn(i, n int) {
	e := &g.enemies[i]
	if !e.alive || n <= 0 {
		return
	}
	if e.frozenT > 0 && g.st.thermal {
		e.frozenT = 0
		g.thermalBlast(i)
		if !e.alive {
			return
		}
	}
	e.burn = min(e.burn+n, maxBurn)
	e.burnT = g.st.burnFor
	if g.st.incinerate > 0 && e.burn >= 10 {
		e.burn = 0
		r := 1.3 + 0.35*float64(g.st.incinerate)
		g.blast(e.pos, r, g.st.dmg*(1.5+float64(g.st.incinerate)), elFire)
	}
}

func (g *game) addChill(i, n int) {
	e := &g.enemies[i]
	if !e.alive || n <= 0 || e.frozenT > 0 {
		return
	}
	e.chill += n
	e.chillT = g.st.chillFor
	if e.chill >= freezeAt {
		g.freeze(i)
	}
}

func (g *game) freeze(i int) {
	e := &g.enemies[i]
	e.chill = 0
	e.frozenT = math.Max(g.st.freezeFor, 1.2)
	if kinds[e.k].boss {
		e.frozenT *= 0.3 // a boss shakes it off
	}
	g.soundFrom(sfxFreeze, e.pos, 0.35)
	if g.st.thermal && e.burn > 0 {
		e.burn = 0
		e.frozenT = 0
		g.thermalBlast(i)
	}
}

func (g *game) addPoison(i, n int) {
	e := &g.enemies[i]
	if !e.alive || n <= 0 {
		return
	}
	e.poison = min(e.poison+n, maxPoison)
	e.poisonT = 5
}

// thermalBlast is a frozen foe set burning, or a burning one frozen: 400%
// on it, half that around it.
func (g *game) thermalBlast(i int) {
	at := g.enemies[i].pos
	g.say("thermal shock")
	g.hit(i, g.st.dmg*4, true, 1, elFire)
	g.blast(at, 1.4, g.st.dmg*2, elIce)
}

// blast hurts everything within r of at, with a ring of its element's
// colour; unlike panic() it throws nothing.
func (g *game) blast(at vec, r, dmg float64, el element) {
	n := int(8 + r*6)
	col := hex(elementTint[el])
	for k := 0; k < n; k++ {
		a := float64(k) / float64(n) * 2 * math.Pi
		g.addPart(particle{pos: at, vel: dir(a).scale(r * 5), z: 0.3, life: 0.25, glyph: '*', col: col})
	}
	g.soundFrom(sfxBoom, at, 0.4)
	r2 := r * r
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.alive && e.pos.dist2(at) <= r2 {
			g.hit(i, dmg, false, 1, el)
		}
	}
}

// onHit is what the elements add to a shot's hit on enemy j.
func (g *game) onHit(s *shot, j int) {
	e := &g.enemies[j]
	if g.st.thunderK > 0 && s.gen == 0 && s.kind == shotPlayer {
		g.hits++
		if g.hits%7 == 0 {
			g.bolt(j)
		}
	}
	if s.crit && g.st.vampire > 0 {
		g.hp = math.Min(g.st.maxHP, g.hp+g.st.vampire)
	}
	if !e.alive {
		return
	}
	if g.st.shockAmp > 0 {
		e.shockT = 2
	}
	if g.st.burnP > 0 && g.rnd.f() < g.st.burnP {
		g.addBurn(j, 1)
	}
	if g.st.chillP > 0 && g.rnd.f() < g.st.chillP {
		g.addChill(j, 1)
	}
	if g.st.poisonP > 0 && g.rnd.f() < g.st.poisonP {
		g.addPoison(j, 1)
	}
}

// arcCarries is Superconductor: an arc brings the burn and chill the build
// has with it.
func (g *game) arcCarries(j int) {
	if !g.st.superArc {
		return
	}
	if g.rank[augIgnite] > 0 {
		g.addBurn(j, 1)
	}
	if g.rank[augChill] > 0 {
		g.addChill(j, 1)
	}
}

// bolt is Thunderstrike: lightning from above onto enemy j.
func (g *game) bolt(j int) {
	e := &g.enemies[j]
	for z := 0.0; z < 2.6; z += 0.2 {
		g.addPart(particle{pos: e.pos.add(vec{g.rnd.between(-0.06, 0.06), g.rnd.between(-0.06, 0.06)}), z: z, life: 0.22, glyph: '│', col: hex(0xfff3a0)})
	}
	g.soundFrom(sfxZap, e.pos, 0.5)
	g.hit(j, g.st.dmg*g.st.thunderK, true, 1, elShock)
}

// stepStatus wears e's stacks down and lets the burn and poison hurt.
func (g *game) stepStatus(i int, e *enemy, dt float64) {
	e.shockT -= dt
	if e.frozenT > 0 {
		e.frozenT -= dt
	}
	if e.chill > 0 {
		if e.chillT -= dt; e.chillT <= 0 {
			e.chill--
			e.chillT = g.st.chillFor * 0.4
		}
	}
	if e.burn > 0 {
		if e.burnT -= dt; e.burnT <= 0 {
			e.burn = 0
		}
		if g.rnd.f() < dt*float64(e.burn)*1.5 {
			g.addPart(particle{pos: e.pos.add(vec{g.rnd.between(-0.2, 0.2), g.rnd.between(-0.2, 0.2)}), z: kinds[e.k].height * g.rnd.between(0.2, 0.8),
				vz: 1.2, life: 0.4, glyph: '^', col: hex(0xff8a3a), tiny: true})
		}
	}
	if e.poison > 0 {
		if e.poisonT -= dt; e.poisonT <= 0 {
			e.poison = 0
		}
		if g.rnd.f() < dt*float64(e.poison)*0.8 {
			g.addPart(particle{pos: e.pos, z: kinds[e.k].height * 0.6, vz: 0.6, life: 0.5, glyph: '°', col: hex(0x9cf05a), tiny: true})
		}
	}
	if e.dotT += dt; e.dotT < dotTick {
		return
	}
	e.dotT -= dotTick
	if e.burn > 0 {
		g.hit(i, dotTick*0.2*g.st.dmg*g.st.burnK*float64(e.burn), false, 0.5, elFire)
	}
	if e.alive && e.poison > 0 {
		d := dotTick * 0.18 * g.st.dmg * g.st.poisonK * float64(e.poison)
		if g.st.dangling {
			share := 0.01
			if kinds[e.k].boss {
				share = 0.001
			}
			d += dotTick * share * e.max * float64(e.poison)
		}
		g.hit(i, d, false, 0, elPoison)
	}
	if e.alive && e.chill > 0 && g.st.frostbite > 0 {
		g.hit(i, dotTick*g.st.frostbite*g.st.dmg*float64(e.chill), false, 0.5, elIce)
	}
}

// onDeath is what an enemy's stacks do as it dies.
func (g *game) onDeath(i int) {
	e := &g.enemies[i]
	if e.burn > 0 && g.st.wildfire > 0 {
		var near [8]int16
		n := g.nearest(e.pos, 3, i, near[:min(g.st.wildfire, 8)])
		for _, j := range near[:n] {
			g.flame(e.pos, g.enemies[j].pos)
			g.addBurn(int(j), max(1, e.burn/2))
		}
	}
	if e.poison > 0 && g.st.plague > 0 {
		var near [8]int16
		n := g.nearest(e.pos, 3, i, near[:min(g.st.plague, 8)])
		for _, j := range near[:n] {
			g.addPoison(int(j), max(1, e.poison/2))
		}
	}
	if e.shockT > 0 && g.st.short > 0 {
		g.chain(e.pos, i, g.st.short, g.st.dmg*0.8)
	}
	if e.frozenT > 0 && g.st.shards > 0 {
		for k := 0; k < g.st.shards; k++ {
			g.shoot(e.pos, g.rnd.f()*2*math.Pi, g.st.dmg*0.7, 1, shotShard)
		}
	}
	if g.st.fumes > 0 && g.rnd.f() < g.st.fumes {
		g.addZone(e.pos, 1.3, 4, elPoison)
	}
	if g.st.refund > 0 && g.reloadT <= 0 && g.ammo < g.st.mag && g.rnd.f() < g.st.refund {
		g.ammo++
	}
}

// flame draws fire leaping from a to b.
func (g *game) flame(a, b vec) {
	d := b.sub(a)
	for k := 0; k < 5; k++ {
		g.addPart(particle{pos: a.add(d.scale(float64(k) / 5)), z: 0.4, vz: 1, life: 0.3, glyph: '^', col: hex(0xffb347)})
	}
}

// nearest finds up to len(out) living enemies within reach of from,
// closest first, skipping skip; it returns how many. Each caller has its
// own out: a hit can kill, and a death look for its own nearest.
func (g *game) nearest(from vec, reach float64, skip int, out []int16) int {
	n := len(out)
	found := 0
	for found < n {
		best, bd := -1, reach*reach
		for i := range g.enemies {
			e := &g.enemies[i]
			if !e.alive || i == skip {
				continue
			}
			taken := false
			for _, j := range out[:found] {
				if int(j) == i {
					taken = true
					break
				}
			}
			if d := e.pos.dist2(from); !taken && d < bd {
				best, bd = i, d
			}
		}
		if best < 0 {
			break
		}
		out[found] = int16(best)
		found++
	}
	return found
}

// ── zones ────────────────────────────────────────────────────────────────

type zone struct {
	alive   bool
	pos     vec
	r, life float64
	tick    float64
	el      element
}

func (g *game) addZone(p vec, r, life float64, el element) {
	for i := range g.zones {
		if !g.zones[i].alive {
			g.zones[i] = zone{alive: true, pos: p, r: r, life: life, el: el}
			return
		}
	}
}

func (g *game) stepZones(dt float64) {
	glyph := [nEl]rune{'·', '^', '*', '°', '·', '·'}
	for z := range g.zones {
		zn := &g.zones[z]
		if !zn.alive {
			continue
		}
		if zn.life -= dt; zn.life <= 0 {
			zn.alive = false
			continue
		}
		if g.rnd.f() < dt*12*zn.r {
			a, d := g.rnd.f()*2*math.Pi, math.Sqrt(g.rnd.f())*zn.r
			g.addPart(particle{pos: zn.pos.add(dir(a).scale(d)), z: 0.02, vz: 0.8, life: 0.5, glyph: glyph[zn.el], col: hex(elementTint[zn.el]), tiny: true})
		}
		if zn.tick -= dt; zn.tick > 0 {
			continue
		}
		zn.tick = 0.3
		r2 := zn.r * zn.r
		for i := range g.enemies {
			e := &g.enemies[i]
			if !e.alive || e.pos.dist2(zn.pos) > r2 {
				continue
			}
			switch zn.el {
			case elFire:
				g.addBurn(i, 1)
			case elIce:
				g.addChill(i, 1)
			case elPoison:
				g.addPoison(i, 1)
			}
		}
	}
}

// ── powers that act by themselves ────────────────────────────────────────

func (g *game) stepPowers(dt float64) {
	g.phoenixT -= dt
	g.deferT -= dt
	if g.dashN < g.st.dashN {
		if g.dashCharge += dt; g.dashCharge >= 1.6*g.st.dashCD {
			g.dashN++
			g.dashCharge = 0
		}
	}
	if g.retryT > 0 {
		if g.retryT -= dt; g.retryT <= 0 {
			g.volley(false)
		}
	}
	if g.st.tesla > 0 {
		if g.teslaT -= dt; g.teslaT <= 0 {
			g.teslaT = 1
			g.zap(g.pos, 5.5, g.st.tesla, g.st.dmg*1.1)
		}
	}
	if g.st.cron > 0 {
		if g.cronT -= dt; g.cronT <= 0 {
			g.cronT = 0.7
			var near [8]int16
			n := g.nearest(g.pos, 12, -1, near[:min(g.st.cron, 8)])
			for _, j := range near[:n] {
				to := g.enemies[j].pos.sub(g.pos)
				g.shoot(g.pos, math.Atan2(to.y, to.x), g.st.dmg*0.8, 1, shotPlayer)
			}
		}
	}
	if g.st.absZero {
		if g.absT -= dt; g.absT <= 0 {
			g.absT = 8
			g.nova(4, elIce, 0)
			for i := range g.enemies {
				if e := &g.enemies[i]; e.alive && e.frozenT <= 0 && e.pos.dist2(g.pos) < 16 {
					g.freeze(i)
				}
			}
		}
	}
	if g.st.gas > 0 && g.reloadT > 0 {
		if g.gasT -= dt; g.gasT <= 0 {
			g.gasT = 0.4
			g.nova(3, elPoison, 0)
			for i := range g.enemies {
				if e := &g.enemies[i]; e.alive && e.pos.dist2(g.pos) < 9 {
					g.addPoison(i, g.st.gas)
				}
			}
		}
	}
	g.stepDaemons(dt)
}

// zap is an arc from p to the n nearest within reach.
func (g *game) zap(p vec, reach float64, n int, dmg float64) {
	var near [8]int16
	k := g.nearest(p, reach, -1, near[:min(n, 8)])
	for _, j := range near[:k] {
		g.arc(p, g.enemies[j].pos)
		g.hit(int(j), dmg, false, 1, elShock)
		g.arcCarries(int(j))
	}
	if k > 0 {
		g.soundFrom(sfxZap, p, 0.3)
	}
}

// nova is a ring of an element's colour spreading from the player; with
// stacks, it adds them to everything inside.
func (g *game) nova(r float64, el element, stacks int) {
	n := int(10 + r*6)
	col := hex(elementTint[el])
	glyph := [nEl]rune{'*', '^', '*', '°', '~', '·'}[el]
	for k := 0; k < n; k++ {
		a := float64(k) / float64(n) * 2 * math.Pi
		g.addPart(particle{pos: g.pos, vel: dir(a).scale(r * 3.2), z: 0.15, life: 0.35, glyph: glyph, col: col})
	}
	if stacks == 0 {
		return
	}
	r2 := r * r
	for i := range g.enemies {
		if e := &g.enemies[i]; e.alive && e.pos.dist2(g.pos) < r2 {
			switch el {
			case elIce:
				g.addChill(i, stacks)
			case elFire:
				g.addBurn(i, stacks)
			}
		}
	}
}

// daemonPos is where daemon k of n is now.
func (g *game) daemonPos(k, n int) vec {
	a := g.clock*2.6 + float64(k)*2*math.Pi/float64(n)
	return g.pos.add(dir(a).scale(1.3))
}

func (g *game) stepDaemons(dt float64) {
	n := g.st.daemons
	if n == 0 {
		return
	}
	for k := 0; k < n; k++ {
		g.addPart(particle{pos: g.daemonPos(k, n), z: 0.5, life: 0.1, glyph: '◆', col: hex(0xc07bff)})
	}
	if g.daemonT -= dt; g.daemonT > 0 {
		return
	}
	g.daemonT = 0.35
	for k := 0; k < n; k++ {
		p := g.daemonPos(k, n)
		for i := range g.enemies {
			e := &g.enemies[i]
			if r := kinds[e.k].r + 0.3; e.alive && e.pos.dist2(p) < r*r {
				g.hit(i, g.st.dmg*0.6, false, 1, elNone)
			}
		}
	}
}

// ── the dash's cards ─────────────────────────────────────────────────────

func (g *game) dashStart() {
	if g.st.spark > 0 {
		g.zap(g.pos, 4.5, g.st.spark+1, g.st.dmg*1.2)
	}
	if g.st.ctxSwitch && g.ammo < g.st.mag {
		g.ammo, g.reloadT = g.st.mag, 0
		g.sound(sfxReady)
	}
	if g.st.deferK > 0 {
		g.deferT = 1
	}
	g.trail = g.pos
}

// dashStep is a moment of the dash: fire left behind, foes run through.
func (g *game) dashStep() {
	if g.st.afterburn > 0 && g.pos.dist2(g.trail) > 0.45*0.45 {
		g.addZone(g.pos, 0.7, 1.5+float64(g.st.afterburn), elFire)
		g.trail = g.pos
	}
	if g.st.ram > 0 {
		for i := range g.enemies {
			e := &g.enemies[i]
			if r := kinds[e.k].r + 0.5; e.alive && e.ramHit != g.dashes && e.pos.dist2(g.pos) < r*r {
				e.ramHit = g.dashes
				e.knock = e.pos.sub(g.pos).scale(10)
				g.hit(i, g.st.dmg*g.st.ram, false, 1, elNone)
			}
		}
	}
}

func (g *game) dashEnd() {
	if g.st.coldSnap > 0 {
		g.nova(2.5, elIce, g.st.coldSnap)
	}
}

// ── Ferris's claw slam ───────────────────────────────────────────────────

// wave is a ring of force spreading over the floor: it hurts once, as it
// passes the player, unless the player is dashing through it.
type wave struct {
	alive bool
	pos   vec
	r     float64
	dmg   float64
	hit   bool
}

func (g *game) stepWave(dt float64) {
	w := &g.wave
	if !w.alive {
		return
	}
	w.r += 5.5 * dt
	if w.r > 11 {
		w.alive = false
		return
	}
	if g.rnd.f() < dt*30 {
		// Sparks off the ring; the ring itself is drawn on the floor.
		a := g.rnd.f() * 2 * math.Pi
		g.addPart(particle{pos: w.pos.add(dir(a).scale(w.r)), z: 0.05, vz: 1.5, life: 0.3, glyph: '^', col: hex(0xffb347), tiny: true})
	}
	if d := math.Sqrt(g.pos.dist2(w.pos)); !w.hit && math.Abs(d-w.r) < 0.4 {
		w.hit = true
		g.hurt(w.dmg, w.pos, kFerris, "claw slam")
	}
}
