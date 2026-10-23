package main

import "math"

// The actives are the build's core: abilities that act by themselves, each
// on its own clock, for a share of the gun's damage — so whatever makes the
// gun stronger makes them stronger too. A rank makes one sooner, bigger or
// more of it.

// activeK is how much harder an active hits than the gun's share says:
// they are the core, and six passives are all the rest a build gets.
// (1.25 of a shot of 9; the shot is 13.5 since it came slower.)
const activeK = 1.25 * 9 / baseDmg

// Six actives each on a short clock filled the screen with effects. They
// act actPace times as seldom, never sooner than actFloor, and hit
// actPunch times as hard: far less damage in all, and each one a moment.
// Daemons and Burn-in are not paced — they are touch and trail, not casts.
const (
	actPace  = 3.0
	actFloor = 1.5 // seconds: the shortest an active's clock gets
	actPunch = 1.3
	actRetry = 0.25 // an active that found nothing to act on tries again soon
)

func (g *game) stepActives(dt float64) {
	for a := augID(0); a < nAug; a++ {
		if !augs[a].act || g.rank[a] == 0 {
			continue
		}
		if g.actT[a] -= dt; g.actT[a] <= 0 {
			paced := a != augDaemon && a != augBurnIn
			d := g.st.dmg * activeK
			if paced {
				d *= actPunch
			}
			t := g.activate(a, g.rank[a], d)
			if paced && t > actRetry {
				t = math.Max(t*actPace, actFloor)
			}
			g.actT[a], g.actP[a] = t, t
		}
	}
	if n := g.rank[augDaemon]; n > 0 {
		// Each daemon an orb of light, its trail behind it.
		for k := 0; k < n; k++ {
			g.puff(g.daemonPos(k, n), 0.55, 0.13, -0.3, 0, 0.18, hex(0xc07bff).mix(rgb{1, 1, 1}, 0.35))
		}
	}
	g.stepVortex(dt)
}

// activate makes active a, at rank r, act for damage d, and returns how
// long until it acts again, before actPace.
func (g *game) activate(a augID, r int, d float64) float64 {
	const retry = actRetry
	lv := float64(r)
	var near [8]int16
	switch a {
	case augTesla: // arcs to the nearest
		g.zap(g.pos, 5.5, 1+r, d*1.1)
		return math.Max(0.5, 1.1-0.1*lv)

	case augDaemon: // daemons circling: what they touch is hit
		for k := 0; k < r; k++ {
			p := g.daemonPos(k, r)
			for i := range g.enemies {
				e := &g.enemies[i]
				if rr := kinds[e.k].r + 0.3; e.alive && e.pos.dist2(p) < rr*rr {
					g.hit(i, d*0.8, false, 1, elNone)
				}
			}
		}
		return 0.35

	case augCron: // a round at each of the nearest
		n := g.nearest(g.pos, 12, -1, near[:min(r, 8)])
		for _, j := range near[:n] {
			to := g.enemies[j].pos.sub(g.pos)
			g.shoot(g.pos, math.Atan2(to.y, to.x), d*0.8, 1, shotPlayer)
		}
		if n == 0 {
			return retry
		}
		return 0.7

	case augAbsZero: // a frost nova that hurts and freezes all it reaches
		rad := 3 + 0.3*lv
		if g.nearest(g.pos, rad, -1, near[:1]) == 0 {
			return retry
		}
		g.nova(rad, elIce, 0)
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && e.pos.dist2(g.pos) < rad*rad {
				g.hit(i, d*2.5, false, 1, elIce)
				if e.alive && e.frozenT <= 0 {
					g.freeze(i)
				}
			}
		}
		return 9 - lv

	case augStorm: // bolts from above on the nearest
		n := g.nearest(g.pos, 10, -1, near[:min(r, 8)])
		for _, j := range near[:n] {
			g.bolt(int(j), d*2.2)
		}
		if n == 0 {
			return retry
		}
		return math.Max(0.8, 2.6-0.3*lv)

	case augSegfault: // a fire blast on the nearest, burning what it catches
		if g.nearest(g.pos, 9, -1, near[:1]) == 0 {
			return retry
		}
		at, rad := g.enemies[near[0]].pos, 1.5+0.2*lv
		g.blast(at, rad, d*1.8, elFire)
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && e.pos.dist2(at) < rad*rad {
				g.addBurn(i, 2)
			}
		}
		return math.Max(1.2, 3.2-0.35*lv)

	case augGasLeak: // a cloud of poison thrown onto them
		if g.nearest(g.pos, 9, -1, near[:1]) == 0 {
			return retry
		}
		at, rad := g.enemies[near[0]].pos, 1.3+0.2*lv
		g.addZone(at, rad, 3.5+0.5*lv, elPoison)
		for k := 0; k < 6; k++ { // the canister bursts: a first cloud at once
			a := float64(k) / 6 * 2 * math.Pi
			g.puff(at.add(dir(a).scale(rad*0.4)), 0.2, rad*0.35, 0.5, 0.2, 1.4, hex(0x6fbf3a))
		}
		return math.Max(1.5, 4-0.4*lv)

	case augColdPath: // a lance of ice: it pierces, and chills
		if g.nearest(g.pos, 12, -1, near[:1]) == 0 {
			return retry
		}
		to := g.enemies[near[0]].pos.sub(g.pos)
		if j := g.shoot(g.pos, math.Atan2(to.y, to.x), d*1.4, 1, shotShard); j >= 0 {
			g.shots[j].pierce, g.shots[j].life = 2+r, 0.9
		}
		return math.Max(0.35, 1.25-0.17*lv)

	case augIRQ: // a field round the player: everything inside is shocked
		rad := 1.5 + 0.25*lv
		g.addRing(g.pos, rad, 0, 0.25, 0.3, hex(0xffe14a), false)
		arcs := 0
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && e.pos.dist2(g.pos) < rad*rad {
				e.shockT = 2
				if arcs < 4 {
					g.addArc(g.pos, e.pos, 0.3, 0.45, 0.14, hex(0xffe14a))
					arcs++
				}
				g.hit(i, d*0.45, false, 1, elShock)
			}
		}
		return 0.5

	case augRoundTrip: // boomerangs thrown the way the player looks
		n := 1 + r/2
		for k := 0; k < n; k++ {
			off := (float64(k) - float64(n-1)/2) * 0.35
			g.shoot(g.pos.add(dir(g.ang).scale(0.3)), g.ang+off, d*1.2, 1, shotRang)
		}
		return math.Max(0.6, 1.7-0.22*lv)

	case augBurnIn: // fire left where the player has walked
		if g.pos.dist2(g.burnAt) > 0.6*0.6 {
			g.addZone(g.pos, 0.55+0.08*lv, 1.8+0.5*lv, elFire)
			g.burnAt = g.pos
		}
		return 0.1

	case augCompaction: // a vortex on the nearest: it pulls, then bursts
		if g.vortex.alive || g.nearest(g.pos, 10, -1, near[:1]) == 0 {
			return retry
		}
		g.vortex = vortex{alive: true, pos: g.enemies[near[0]].pos, t: 1.4, r: 3.5 + 0.4*lv, dmg: d * 2.2}
		g.addRing(g.vortex.pos, g.vortex.r, -g.vortex.r/1.4, 0.5, 1.4, hex(0xc07bff), false)
		g.soundFrom(sfxCharge, g.vortex.pos, 0.5)
		return math.Max(3, 7.5-0.8*lv)
	}
	return 1
}

// daemonPos is where daemon k of n is now.
func (g *game) daemonPos(k, n int) vec {
	a := g.clock*2.6 + float64(k)*2*math.Pi/float64(n)
	return g.pos.add(dir(a).scale(1.3))
}

// vortex is Compaction's: it pulls everything within r towards pos while
// t runs, then bursts.
type vortex struct {
	alive     bool
	pos       vec
	t, r, dmg float64
}

func (g *game) stepVortex(dt float64) {
	v := &g.vortex
	if !v.alive {
		return
	}
	for k := 0; k < 2; k++ {
		// Puffs drawn in along a spiral.
		a, d := g.rnd.f()*2*math.Pi, g.rnd.between(0.5, v.r)
		p := v.pos.add(dir(a).scale(d))
		g.addPart(particle{pos: p, vel: v.pos.sub(p).scale(2).add(dir(a + math.Pi/2).scale(3)), z: 0.3, life: 0.4,
			col: hex(0xc07bff).mix(rgb{1, 1, 1}, 0.2), puff: 0.22, grow: -0.3})
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive || e.pos.dist2(v.pos) > v.r*v.r {
			continue
		}
		pull := 7.0
		if kinds[e.k].boss {
			pull = 1.5
		}
		if to := v.pos.sub(e.pos); to.len() > 0.2 {
			e.knock = to.scale(pull / to.len())
		}
	}
	if v.t -= dt; v.t <= 0 {
		v.alive = false
		g.blast(v.pos, 2, v.dmg, elNone)
		g.fireball(v.pos, 2, hex(0xc07bff))
	}
}
