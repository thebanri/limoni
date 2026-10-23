package main

import "math"

// Every boss has a move that is its own, wound up like its others and
// named over its health bar, and a second phase under half health:
//
//	Boba Prime   STRAW     it drinks: the player is pulled in, and has to
//	                       walk away or dash
//	Ferris       BURROW    under the floor, unseen and unhurt; it comes up
//	                       where the player stood, the floor marked first
//	Modal Queen  MODAL     a dialog opens: while its two buttons stand,
//	                       nothing hurts her, and if it is left open she
//	                       mends
//	Reconciler   RE-RENDER it lands by the player and leaves a virtual
//	                       copy behind, which looks the same and is nothing
//	Textual      CSS GRID  the floor in stripes, then the stripes between

const (
	burrowTime = 2.2 // Ferris's time under the floor
	pullSpeed  = 2.6 // the straw, against a walk of baseMove
	modalTime  = 12  // until a modal left open closes, and she mends
)

// sigs reports whether e's next attack is its own move.
func sigs(e *enemy) bool {
	switch e.k {
	case kBobaPrime, kQueen:
		return e.attacks%3 == 2
	case kFerris:
		return e.attacks%4 == 3
	case kTextual:
		return e.attacks%3 == 1
	}
	return false
}

// signature is boss i's own move, as its wind-up runs out.
func (g *game) signature(i int, e *enemy, n vec) {
	kd := &kinds[e.k]
	switch e.k {
	case kBobaPrime:
		e.cd = 2.6
		g.pullT, g.pullBy = 1.8, i
		g.soundFrom(sfxRoar, e.pos, 0.8)
	case kFerris:
		e.cd = 1.2
		e.under = burrowTime
		g.fireball(e.pos, kd.r, hex(kd.accent).mul(0.6))
		g.soundFrom(sfxBoom, e.pos, 0.8)
	case kQueen:
		e.cd = 3
		g.openModal(i, e, n)
	case kTextual:
		e.cd = 3.2
		g.cssGrid(e)
	}
}

// stepBoss is what a boss does every frame besides walking: its second
// phase, and the modal it holds open.
func (g *game) stepBoss(e *enemy, dt float64) {
	kd := &kinds[e.k]
	if !e.phase2 && e.hp < e.max/2 {
		e.phase2 = true
		g.say(kd.name + ": phase 2")
		g.soundFrom(sfxRoar, e.pos, 1)
		g.addRing(e.pos, 0.5, 9, 0.6, 0.6, hex(kd.accent), false)
		g.shake = math.Max(g.shake, 0.8)
	}
	if e.guards > 0 {
		if e.guardT -= dt; e.guardT <= 0 {
			g.closeModal(e)
		}
	}
}

// ── Boba Prime: the straw ────────────────────────────────────────────────

// stepPull draws the player towards Boba Prime while it drinks, a straw
// of light between them. A dash is not pulled.
func (g *game) stepPull(dt float64, dashing bool) {
	if g.pullT <= 0 {
		return
	}
	g.pullT -= dt
	e := &g.enemies[g.pullBy]
	if !e.alive || e.k != kBobaPrime || e.frozenT > 0 {
		g.pullT = 0
		return
	}
	g.addLaser(e.pos, g.pos, hex(0xff5fd2))
	to := e.pos.sub(g.pos)
	if l := to.len(); l > kinds[e.k].r+0.5 && !dashing {
		slide(&g.pos, to.x/l*pullSpeed*dt, to.y/l*pullSpeed*dt, playerR)
	}
}

// ── Ferris: the burrow ───────────────────────────────────────────────────

// stepBurrow moves Ferris under the floor towards the player, dust
// over it, and marks where it comes up a moment before it does.
func (g *game) stepBurrow(i int, e *enemy, dt float64) {
	kd := &kinds[e.k]
	was := e.under
	e.under -= dt
	to := g.pos.sub(e.pos)
	if l := to.len(); l > 0.3 && e.under > 0.7 {
		slide(&e.pos, to.x/l*5*dt, to.y/l*5*dt, kd.r*0.8)
	}
	if int(was*10) != int(e.under*10) {
		g.puff(e.pos, 0.05, 0.35, 0.8, 0.3, 0.5, hex(0x8a6a4a))
	}
	if was > 0.7 && e.under <= 0.7 {
		at, ok := roomFor(g.pos, kd.r*0.8)
		if !ok {
			at = e.pos
		}
		g.addHazard(hazard{pos: at, r: 1.6, t: 0.7, dmg: g.contact(e) * 0.9, src: e.k, how: "burrow", owner: i, ownerK: e.k, land: true})
	}
	if e.under <= 0 {
		e.under = 0
		g.soundFrom(sfxBoom, e.pos, 1)
	}
}

// roomFor is a place for a body of radius r at p or near it.
func roomFor(p vec, r float64) (vec, bool) {
	if free(p.x, p.y, r) {
		return p, true
	}
	for k := 0; k < 8; k++ {
		for _, d := range [2]float64{0.8, 1.6} {
			q := p.add(dir(float64(k) * math.Pi / 4).scale(d))
			if free(q.x, q.y, r) {
				return q, true
			}
		}
	}
	return p, false
}

// ── the Modal Queen: the modal ───────────────────────────────────────────

// openModal puts up a dialog: [Y] and [N] buttons that keep by her side,
// and while either stands, nothing hurts her.
func (g *game) openModal(i int, e *enemy, n vec) {
	side := vec{-n.y, n.x}
	for k := -1; k <= 1; k += 2 {
		j := g.addEnemy(kButton, e.pos.add(side.scale(float64(k)*2.2)))
		if j < 0 {
			continue
		}
		b := &g.enemies[j]
		b.guardOf, b.phaseOff = int16(i+1), float64(k+1)*math.Pi/2
		b.hp = e.max * 0.035
		b.max, b.armor = b.hp, 0
		e.guards++
	}
	if e.guards > 0 {
		e.guardT = modalTime
		g.say("the modal is open: break [Y] and [N]")
	}
	g.addEnemy(kModal, e.pos.add(n.scale(1.5)))
	g.soundFrom(sfxRoar, e.pos, 0.7)
}

// guardDown is one of the Queen's buttons broken.
func (g *game) guardDown(b *enemy) {
	q := &g.enemies[b.guardOf-1]
	if !q.alive || q.k != kQueen || q.guards == 0 {
		return
	}
	if q.guards--; q.guards == 0 {
		g.say("the modal is closed: she can be hurt")
		g.addRing(q.pos, 0.5, 7, 0.4, 0.4, hex(kinds[kQueen].accent), false)
		g.soundFrom(sfxPick, q.pos, 0.8)
	}
}

// closeModal is a modal left open too long: its buttons go, and she mends
// a tenth of her health.
func (g *game) closeModal(q *enemy) {
	for j := range g.enemies {
		if b := &g.enemies[j]; b.alive && b.guardOf > 0 && &g.enemies[b.guardOf-1] == q {
			b.alive = false
			g.burst(b)
		}
	}
	q.guards = 0
	q.hp = math.Min(q.max, q.hp+q.max*0.1)
	g.say("the modal timed out: she mended")
}

// guarding moves a Queen's button round her, and reports whether it is
// one.
func (g *game) guarding(e *enemy) (vec, bool) {
	if e.guardOf == 0 {
		return vec{}, false
	}
	q := &g.enemies[e.guardOf-1]
	if !q.alive || q.k != kQueen || q.guards == 0 {
		e.guardOf = 0 // its modal is gone: a button like any other
		return vec{}, false
	}
	at := q.pos.add(dir(e.t*1.1 + e.phaseOff).scale(2.4))
	to := at.sub(e.pos)
	if l := to.len(); l > 0.05 {
		return to.scale(1 / math.Max(l, 1)), true
	}
	return vec{}, true
}

// ── Ink's Reconciler: the virtual copy ───────────────────────────────────

// leaveDecoy is a re-render leaving the old Reconciler behind, virtual: it
// looks the same and fires softer, and a few hits end it. One at a time,
// two in the second phase.
func (g *game) leaveDecoy(e *enemy) {
	most, n := 1, 0
	if e.phase2 {
		most = 2
	}
	for j := range g.enemies {
		if d := &g.enemies[j]; d.alive && d.decoy {
			n++
		}
	}
	if n >= most {
		return
	}
	if j := g.addEnemy(kReconciler, e.pos); j >= 0 {
		d := &g.enemies[j]
		d.decoy, d.phase2 = true, true
		d.hp = e.max * 0.06
		d.max, d.cd = d.hp, 1.5
	}
}

// unmount ends every virtual copy: the Reconciler they copied is gone.
func (g *game) unmount() {
	for j := range g.enemies {
		if d := &g.enemies[j]; d.alive && d.decoy {
			d.alive = false
			g.burst(d)
		}
	}
}

// ── Textual: the grid ────────────────────────────────────────────────────

// cssGrid marks the floor round the player in stripes, every other one,
// and then the ones between: a step across, then a step back.
func (g *game) cssGrid(e *enemy) {
	const w, length = 0.8, 8.0 // half a stripe's width, and half its length
	alongX := g.rnd.f() < 0.5
	dmg := g.contact(e) * 0.7
	for k := -2; k <= 2; k++ {
		off := float64(k) * 2 * w
		h := hazard{pos: g.pos, src: e.k, how: "css grid", owner: -1, dmg: dmg, t: 1.3}
		if k%2 != 0 {
			h.t = 2.4 // the stripes between, after
		}
		if alongX {
			h.pos.y += off
			h.hw, h.hh = length, w
		} else {
			h.pos.x += off
			h.hw, h.hh = w, length
		}
		g.addHazard(h)
	}
	g.soundFrom(sfxCharge, g.pos, 0.8)
}
