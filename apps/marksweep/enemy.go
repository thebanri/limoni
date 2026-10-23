package main

import "math"

// The enemies are other people's terminal frameworks, each a creature made
// from its own idea (sprites.go). When one dies it comes apart into its
// framework's characters.

type kindID uint8

const (
	kSpinner    kindID = iota // Bubble Tea's spinner: swarms
	kBoba                     // Bubble Tea: a cup with pearls
	kJSX                      // Ink: </>, hops, splits in two
	kTag                      // half of a </>
	kCrab                     // Ratatui: keeps its distance and fires bars
	kModal                    // tview: a slow modal that splits into buttons
	kButton                   // one of a modal's buttons
	kBar                      // Bubbles' progress bar: weaves at speed
	kElder                    // ncurses: old, armoured, slow
	kBeetle                   // crossterm: winds up, then charges in a line
	kCaret                    // prompt_toolkit: a sniper, its aim a laser
	kBomb                     // notcurses: runs in, and blows up
	kGloss                    // Lip Gloss: shields the others
	kCursor                   // termbox: blinks closer
	kBobaPrime                // boss
	kFerris                   // boss
	kQueen                    // boss: tview's Modal Queen
	kReconciler               // boss: Ink's Reconciler
	kTextual                  // boss: Textual's Cascade
	nKinds
)

type behaviour uint8

const (
	bChase behaviour = iota
	bHop
	bWeave
	bKeepAway
	bCharge  // stops, shows its line, and charges along it
	bSnipe   // keeps far, aims a laser, fires one hard shot
	bBomb    // runs in; close, it lights its fuse
	bSupport // keeps behind the others and shields them
	bBlink   // walks, and every so often blinks nearer
)

// bosses come in this order, one every ten levels, then again, tougher.
var bosses = [...]kindID{kBobaPrime, kFerris, kQueen, kReconciler, kTextual}

type kind struct {
	name     string
	from     string // the framework
	hp       float64
	speed    float64
	r        float64 // body radius
	height   float64 // drawn height, in world units
	dmg      float64 // on contact
	xp       float64
	armor    float64
	act      behaviour
	minLevel int     // first level it spawns at; 0: only from others
	weight   float64 // how often it is picked
	cost     float64 // spawn budget
	group    int     // how many come at once
	splitTo  kindID
	splitN   int
	ranged   bool
	spr      []*sprite // animation frames
	bits     []rune    // the characters it comes apart into
	accent   uint32    // its colour, for particles and the HUD
	boss     bool
	intro    string // said the first time it comes in a run
}

var kinds [nKinds]kind

const maxEnemies = 240

type enemy struct {
	alive    bool
	k        kindID
	pos      vec
	hp, max  float64
	armor    float64
	elite    bool
	immune   bool // insulated: explosions and arcs do nothing
	t        float64
	cd       float64 // until it may fire
	flash    float64 // hit flash
	knock    vec
	lastShot int32 // the shot that last hit it, so a piercing one hits once
	phaseOff float64

	// the elements' stacks (element.go)
	burn, poison, chill                           int
	burnT, poisonT, chillT, frozenT, shockT, dotT float64
	ramHit                                        int32 // the dash that last ran through it

	windT   float64 // winding up an attack: it fires when this runs out
	attacks int     // attacks made, for a boss that takes turns
	slam    bool    // the attack wound up is Ferris's slam, or a boss's other one
	dashT   float64 // charging while above zero
	dashDir vec     // and which way
	aim     vec     // where a sniper's laser points
	shieldT float64 // shielded by a Lip Gloss while above zero
}

// windUp is how long a ranged enemy shows its attack coming.
func windUp(k kindID, slam bool) float64 {
	switch {
	case k == kCaret:
		return 1.1
	case k == kBeetle:
		return 0.75
	case k == kCursor:
		return 0.35
	case k == kQueen, k == kReconciler, k == kTextual:
		return 1.0
	case slam:
		return 1.1
	case k == kFerris:
		return 0.75
	case k == kBobaPrime:
		return 0.6
	}
	return 0.45
}

func init() {
	tap := func(f int) *sprite { return tapioca(f, false) }
	prime := func(f int) *sprite { return tapioca(f, true) }
	crab := func(f int) *sprite { return ratacrab(f, false) }
	king := func(f int) *sprite { return ratacrab(f, true) }
	tag := []*sprite{halfChevron(false), halfChevron(true)}
	btn := []*sprite{button(false), button(true)}

	kinds = [nKinds]kind{
		kSpinner: {name: "spinner", from: "Bubble Tea", hp: 5, speed: 3.4, r: 0.18, height: 0.32, dmg: 5, xp: 0.6,
			act: bChase, minLevel: 1, weight: 3.5, cost: 0.6, group: 5, spr: frames(8, dotling), bits: []rune("⠋⠙⠹⠸⠼⠴"), accent: 0xff5fd2},
		kBoba: {name: "boba", from: "Bubble Tea", hp: 20, speed: 2.0, r: 0.32, height: 0.75, dmg: 11, xp: 2,
			act: bChase, minLevel: 1, weight: 4.5, cost: 2, group: 1, spr: frames(6, tap), bits: []rune("°o●"), accent: 0xff5fd2},
		kJSX: {name: "</>", from: "Ink", hp: 14, speed: 2.5, r: 0.26, height: 0.4, dmg: 9, xp: 1.5,
			act: bHop, minLevel: 3, weight: 4, cost: 1.6, group: 2, splitTo: kTag, splitN: 2, spr: frames(2, chevron), bits: []rune("</>"), accent: 0x61dafb},
		kTag: {name: "<", from: "Ink", hp: 5, speed: 3.3, r: 0.18, height: 0.3, dmg: 6, xp: 0.4,
			act: bHop, spr: tag, bits: []rune("<>"), accent: 0x61dafb},
		kCrab: {name: "crab", from: "Ratatui", hp: 28, speed: 1.6, r: 0.36, height: 0.6, dmg: 10, xp: 3,
			act: bKeepAway, minLevel: 5, weight: 3, cost: 3, group: 1, ranged: true, spr: frames(2, crab), bits: []rune("()/\\"), accent: 0xf74c00},
		kModal: {name: "modal", from: "tview", hp: 85, speed: 1.05, r: 0.48, height: 0.9, dmg: 20, xp: 6,
			act: bChase, minLevel: 8, weight: 2, cost: 6, group: 1, splitTo: kButton, splitN: 2, spr: frames(2, mimic), bits: []rune("[]YN═║"), accent: 0x3d7bff},
		kButton: {name: "button", from: "tview", hp: 10, speed: 2.7, r: 0.22, height: 0.32, dmg: 7, xp: 0.8,
			act: bChase, spr: btn, bits: []rune("[]YN"), accent: 0xffe14a},
		kBar: {name: "progress", from: "Bubbles", hp: 36, speed: 3.0, r: 0.3, height: 0.3, dmg: 12, xp: 3,
			act: bWeave, minLevel: 11, weight: 3, cost: 3.5, group: 2, spr: frames(8, serpent), bits: []rune("█░▕▏"), accent: 0xc06ef3},
		kElder: {name: "elder", from: "ncurses", hp: 210, speed: 0.95, r: 0.52, height: 0.85, dmg: 28, xp: 12, armor: 3,
			act: bChase, minLevel: 15, weight: 1.4, cost: 11, group: 1, spr: frames(5, phosphor), bits: []rune("+-|@="), accent: 0x33ff66},
		kBeetle: {name: "crossterm", from: "crossterm", hp: 42, speed: 1.8, r: 0.34, height: 0.45, dmg: 18, xp: 3.5, armor: 1,
			act: bCharge, minLevel: 6, weight: 2.5, cost: 3, group: 1, ranged: true, spr: frames(2, beetle), bits: []rune("+=#"), accent: 0x6f8fb8},
		kCaret: {name: "caret", from: "prompt_toolkit", hp: 22, speed: 1.6, r: 0.26, height: 0.45, dmg: 9, xp: 3,
			act: bSnipe, minLevel: 10, weight: 2, cost: 3, group: 1, ranged: true, spr: frames(2, caret), bits: []rune(">_$"), accent: 0xffd43b},
		kBomb: {name: "bomb", from: "notcurses", hp: 10, speed: 3.1, r: 0.27, height: 0.45, dmg: 30, xp: 1.6,
			act: bBomb, minLevel: 4, weight: 2.5, cost: 1.8, group: 1, spr: frames(4, bomb), bits: []rune("*#@"), accent: 0xff6a1a},
		kGloss: {name: "lip gloss", from: "Lip Gloss", hp: 30, speed: 1.9, r: 0.24, height: 0.55, dmg: 6, xp: 3.5,
			act: bSupport, minLevel: 9, weight: 1.5, cost: 4, group: 1, spr: frames(4, gloss), bits: []rune("✧·+"), accent: 0xff6ad5},
		kCursor: {name: "cursor", from: "termbox", hp: 13, speed: 1.6, r: 0.22, height: 0.42, dmg: 8, xp: 1.2,
			act: bBlink, minLevel: 2, weight: 3, cost: 1.4, group: 2, ranged: true, spr: frames(2, cursorBox), bits: []rune("█▄_"), accent: 0x00add8},
		kBobaPrime: {name: "BOBA PRIME", from: "Bubble Tea", hp: 1400, speed: 1.35, r: 0.95, height: 1.6, dmg: 30, xp: 60,
			act: bChase, ranged: true, spr: frames(6, prime), bits: []rune("°o●◕"), accent: 0xff5fd2, boss: true},
		kFerris: {name: "FERRIS, CRAB KING", from: "Ratatui", hp: 2300, speed: 1.1, r: 1.0, height: 1.5, dmg: 34, xp: 90, armor: 4,
			act: bKeepAway, ranged: true, spr: frames(2, king), bits: []rune("()/\\▲"), accent: 0xf74c00, boss: true},
		kQueen: {name: "THE MODAL QUEEN", from: "tview", hp: 3200, speed: 0.9, r: 1.1, height: 1.7, dmg: 36, xp: 120, armor: 5,
			act: bKeepAway, ranged: true, spr: frames(2, queen), bits: []rune("[]YN?═"), accent: 0x3d7bff, boss: true},
		kReconciler: {name: "INK, THE RECONCILER", from: "Ink", hp: 4000, speed: 1.4, r: 1.0, height: 1.6, dmg: 38, xp: 150, armor: 4,
			act: bKeepAway, ranged: true, spr: frames(6, atom), bits: []rune("</>{}"), accent: 0x61dafb, boss: true},
		kTextual: {name: "TEXTUAL, THE CASCADE", from: "Textual", hp: 5200, speed: 0.8, r: 1.1, height: 1.8, dmg: 42, xp: 180, armor: 6,
			act: bKeepAway, ranged: true, spr: frames(2, cascade), bits: []rune("┃━▔▁"), accent: 0xffa62b, boss: true},
	}
	for k := range kinds {
		if kd := &kinds[k]; kd.minLevel > 0 {
			kd.intro = kd.from + "'s " + kd.name + " has come for the heap"
			if kd.from == kd.name {
				kd.intro = kd.name + " has come for the heap"
			}
		}
	}
}

// ── spawning ─────────────────────────────────────────────────────────────

// hpScale is how much tougher everything is now: a little with every
// level and minute, and then the pressure on top.
func (g *game) hpScale() float64 {
	return (1 + 0.02*float64(g.level-1)) * (1 + g.runTime/200) * g.pressure()
}

// budget is how much enemy arrives per second.
func (g *game) budget() float64 {
	// Time alone sets it: tied to the level, more enemies meant more
	// experience meant more enemies, and a good run never stopped growing.
	return (0.8 + g.runTime/200) * math.Sqrt(g.pressure())
}

func (g *game) spawn(dt float64) {
	g.spawnAcc += g.budget() * dt
	alive := 0
	for i := range g.enemies {
		if g.enemies[i].alive {
			alive++
		}
	}
	for g.spawnAcc > 0 && alive < maxEnemies-12 {
		k := g.pickKind()
		cost := kinds[k].cost * float64(kinds[k].group)
		if g.spawnAcc < cost {
			break
		}
		g.spawnAcc -= cost
		if !g.seen[k] {
			g.seen[k] = true
			if k != kSpinner && k != kBoba {
				g.say(kinds[k].intro) // the first of its kind this run
			}
		}
		p := g.portal()
		for i := 0; i < kinds[k].group; i++ {
			off := vec{g.rnd.between(-0.6, 0.6), g.rnd.between(-0.6, 0.6)}
			q := p.add(off)
			if !free(q.x, q.y, kinds[k].r) {
				q = p
			}
			if g.addEnemy(k, q) >= 0 {
				alive++
			}
		}
	}
	if g.spawnAcc > 40 {
		g.spawnAcc = 40
	}
}

func (g *game) pickKind() kindID {
	total := 0.0
	for k := range kinds {
		total += g.kindWeight(kindID(k))
	}
	pick := g.rnd.f() * total
	for k := range kinds {
		if w := g.kindWeight(kindID(k)); pick < w {
			return kindID(k)
		} else {
			pick -= w
		}
	}
	return kBoba
}

// kindWeight is how often kind k comes now: not before its level, and much
// more often until it has come once, so a new one is met soon.
func (g *game) kindWeight(k kindID) float64 {
	kd := &kinds[k]
	if kd.minLevel == 0 || kd.minLevel > g.level {
		return 0
	}
	if !g.seen[k] {
		return kd.weight * 4
	}
	return kd.weight
}

// portal picks a spawn window that is not right next to the player, and
// returns the floor just in front of it.
func (g *game) portal() vec {
	var best vec
	for try := 0; try < 6; try++ {
		i := g.rnd.intn(len(portals))
		p := portals[i]
		best = p
		if p.dist2(g.pos) > 36 {
			return p
		}
	}
	return best
}

var portals []vec

func init() {
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			if tile(x, y) != 'P' {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if tile(x+d[0], y+d[1]) == '.' {
					portals = append(portals, vec{float64(x+d[0]) + 0.5 + 0.3*float64(d[0]), float64(y+d[1]) + 0.5 + 0.3*float64(d[1])})
				}
			}
		}
	}
}

// addEnemy puts one in the arena and returns its index, or -1 if full.
func (g *game) addEnemy(k kindID, p vec) int {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.alive {
			continue
		}
		kd := &kinds[k]
		hp := kd.hp * g.hpScale()
		*e = enemy{alive: true, k: k, pos: p, hp: hp, max: hp, armor: kd.armor, lastShot: -1,
			phaseOff: g.rnd.f() * 10, cd: g.rnd.between(1, 2.5)}
		// Elites from level 12; after eight minutes they are armoured, and
		// after twelve everything is.
		if !kd.boss && g.level >= 12 && g.rnd.f() < math.Min(0.35, float64(g.level-11)*0.015) {
			e.elite = true
			e.hp *= 3
			e.max = e.hp
			e.immune = g.rnd.f() < 0.4
		}
		// Past ten minutes the heap fights the build itself: armour that
		// shrugs off shrapnel and children, and more and more of them
		// insulated against explosions and arcs.
		if late := g.runTime - 540; late > 0 {
			e.armor += late / 60 * 8
			if e.elite {
				e.armor += 3
			}
			if g.rnd.f() < late/240 {
				e.immune = true
			}
		}
		return i
	}
	return -1
}

func (g *game) spawnBoss() {
	k := bosses[(g.bossNext/10-1)%len(bosses)]
	// Each boss in the order is tougher than the last by its own health;
	// each time the order comes round, all of them again.
	round := float64((g.bossNext/10 - 1) / len(bosses))
	p := vec{16, 4}
	if g.pos.y < 16 {
		p = vec{16, 28}
	}
	if i := g.addEnemy(k, p); i >= 0 {
		g.enemies[i].hp *= 1 + 0.8*round
		g.enemies[i].max = g.enemies[i].hp
		g.bossLive = true
		g.say(kinds[k].name + " has entered the heap")
		g.sound(sfxRoar)
	}
	g.bossNext += 10
}

// ── behaviour ────────────────────────────────────────────────────────────

func (g *game) stepEnemies(dt float64) {
	for i := range g.bucket {
		g.bucket[i] = -1
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		cx, cy := int(e.pos.x), int(e.pos.y)
		if cx >= 0 && cy >= 0 && cx < mapW && cy < mapH {
			c := cy*mapW + cx
			g.nextInCel[i] = g.bucket[c]
			g.bucket[c] = int16(i)
		}
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		g.moveEnemy(i, e, dt)
	}
}

func (g *game) moveEnemy(i int, e *enemy, dt float64) {
	kd := &kinds[e.k]
	if e.flash > 0 {
		e.flash -= dt * 6
	}
	if e.burn > 0 || e.poison > 0 || e.chill > 0 || e.frozenT > 0 || e.shockT > 0 {
		g.stepStatus(i, e, dt)
		if !e.alive {
			return
		}
	}
	if e.frozenT > 0 {
		e.knock = e.knock.scale(math.Max(0, 1-dt*8))
		slide(&e.pos, e.knock.x*dt, e.knock.y*dt, kd.r*0.8)
		return // frozen: no step, no bite, no attack, and its frame held
	}
	e.t += dt
	to := g.pos.sub(e.pos)
	d := to.len()
	if d < 0.001 {
		d = 0.001
	}
	n := to.scale(1 / d)
	speed := kd.speed * (1 + 0.15*math.Log2(g.pressure()))
	if e.elite {
		speed *= 1.15
	}
	speed *= math.Max(0.3, 1-0.1*float64(e.chill))
	if e.poison > 0 {
		speed *= 1 - g.st.neuro
	}
	if e.shieldT > 0 {
		e.shieldT -= dt
	}
	var m vec
	switch kd.act {
	case bChase, bHop:
		m = n
	case bCharge:
		switch {
		case e.dashT > 0:
			e.dashT -= dt
			m, speed = e.dashDir, 10
		case e.windT > 0:
			// standing, showing its line
		default:
			m = n
		}
	case bSnipe:
		switch {
		case e.windT > 0:
		case d > 11:
			m = n
		case d < 7:
			m = n.scale(-0.8)
		default:
			m = vec{-n.y, n.x}.scale(0.5 * math.Sin(e.phaseOff+e.t*0.5))
		}
	case bBomb:
		if e.windT > 0 {
			break // the fuse is lit: it stands and fizzes
		}
		m = n
		if d < 1.6 {
			e.windT = 0.9
			g.addHazard(hazard{pos: e.pos, r: 1.8, t: 0.9, dmg: g.contact(e), src: e.k, how: "blast", owner: i, ownerK: e.k, foes: true})
			g.soundFrom(sfxCharge, e.pos, 0.8)
		}
	case bSupport:
		switch {
		case d > 9:
			m = n
		case d < 6:
			m = n.scale(-0.9)
		}
		if e.cd -= dt; e.cd <= 0 {
			e.cd = 1.2
			g.shieldNear(i, e)
		}
	case bBlink:
		if e.windT <= 0 {
			m = n
		}
	case bWeave:
		side := vec{-n.y, n.x}
		m = n.add(side.scale(math.Sin(e.t*4+e.phaseOff) * 1.2))
	case bKeepAway:
		switch {
		case d > 7:
			m = n
		case d < 4.5:
			m = n.scale(-0.8)
		default:
			m = vec{-n.y, n.x}.scale(0.6 * math.Sin(e.phaseOff))
		}
	}
	// Keep apart from the ones in the same cell, or a swarm becomes a dot.
	cx, cy := int(e.pos.x), int(e.pos.y)
	if cx >= 0 && cy >= 0 && cx < mapW && cy < mapH {
		for j := g.bucket[cy*mapW+cx]; j >= 0; j = g.nextInCel[j] {
			if int(j) == i {
				continue
			}
			o := &g.enemies[j]
			dd := e.pos.sub(o.pos)
			l := dd.len()
			rr := kd.r + kinds[o.k].r
			if l > 0.0001 && l < rr {
				m = m.add(dd.scale((rr - l) / l * 2.5))
			}
		}
	}
	if l := m.len(); l > 1 {
		m = m.scale(1 / l)
	}
	step := m.scale(speed * dt).add(e.knock.scale(dt))
	e.knock = e.knock.scale(math.Max(0, 1-dt*8))
	slide(&e.pos, step.x, step.y, kd.r*0.8)

	// contact
	if d < kd.r+playerR && g.hurtT <= 0 {
		bite := kd.dmg * (1 + 0.03*float64(g.level)) * math.Sqrt(g.pressure())
		if e.poison > 0 {
			bite *= 1 - g.st.neuro/2
		}
		how := "bite"
		if e.dashT > 0 {
			how = "charge"
		}
		g.hurt(bite, e.pos, e.k, how)
		e.knock = n.scale(-6)
	}
	if kd.act == bBomb && e.windT > 0 {
		e.windT -= dt // its hazard goes off, and it with it
	}
	// A sniper's laser: it follows the player until just before the shot,
	// then holds still, so a step aside is the way out.
	if kd.act == bSnipe && e.windT > 0 {
		if e.windT > 0.3 {
			e.aim = n
		}
		g.addLaser(e.pos, e.pos.add(e.aim.scale(math.Min(d+2, 14))), hex(0xff3a3a))
	}
	// Ranged: a wind-up first, long enough to see and to dash.
	if kd.ranged {
		switch {
		case e.windT > 0:
			if e.windT -= dt; e.windT <= 0 {
				g.enemyFire(i, e, n, d)
			}
		default:
			e.cd -= dt
			reach := 13.0
			switch kd.act {
			case bCharge:
				reach = 7
			case bSnipe:
				reach = 14
			}
			if e.cd <= 0 && d < reach && (kd.act == bBlink || g.sees(e.pos)) {
				e.slam = slams(e)
				if kd.act == bCharge {
					e.dashDir = n
					for k := 1; k <= 6; k++ { // its line, on the floor
						g.addRing(e.pos.add(n.scale(float64(k)*1.0)), 0.22, 0, 0.22, windUp(e.k, false), hex(0x6f8fb8), true)
					}
				}
				e.windT = windUp(e.k, e.slam)
				vol := float32(0.5)
				if kd.boss {
					vol = 1
				}
				g.soundFrom(sfxCharge, e.pos, vol)
				if e.slam && !g.slamSeen {
					g.slamSeen = true
					g.say("CLAW SLAM coming: dash through the wave")
				}
			}
		}
	}
}

// sees reports whether a straight line from p to the player is clear.
func (g *game) sees(p vec) bool {
	to := g.pos.sub(p)
	d := to.len()
	steps := int(d * 3)
	for s := 1; s < steps; s++ {
		q := p.add(to.scale(float64(s) / float64(steps)))
		if solid(q.x, q.y) {
			return false
		}
	}
	return true
}

func (g *game) enemyFire(i int, e *enemy, n vec, d float64) {
	kd := &kinds[e.k]
	a := math.Atan2(n.y, n.x)
	dmg := kd.dmg * 0.7 * (1 + 0.03*float64(g.level)) * math.Sqrt(g.pressure())
	e.attacks++
	switch e.k {
	case kBeetle:
		e.cd, e.dashT = 3, 0.6
		g.soundFrom(sfxDash, e.pos, 0.8)
		return
	case kCaret:
		e.cd = 3.2
		if j := g.enemyShot(e, e.pos.add(e.aim.scale(kd.r)), math.Atan2(e.aim.y, e.aim.x), dmg*2.2, shotBar); j >= 0 {
			g.shots[j].vel = g.shots[j].vel.scale(3) // a prompt's shot is fast
		}
	case kCursor:
		e.cd = 1.8
		step := math.Min(2.6, d-1.2)
		if to := e.pos.add(n.scale(step)); step > 0.3 && free(to.x, to.y, kd.r) {
			g.puff(e.pos, 0.3, 0.25, 0.4, 0, 0.3, hex(0x00add8))
			e.pos = to
			g.puff(e.pos, 0.3, 0.25, 0.4, 0, 0.3, hex(0xe8ecf8))
		}
		return
	case kQueen:
		e.cd = 2.6
		if e.slam { // her buttons, and a modal
			for k := 0; k < 4; k++ {
				g.addEnemy(kButton, e.pos.add(dir(float64(k)*math.Pi/2).scale(1.5)))
			}
			g.addEnemy(kModal, e.pos.add(n.scale(1.5)))
			g.soundFrom(sfxRoar, e.pos, 0.6)
			return
		}
		// A dialog wall: eleven bars side by side, two missing.
		side := vec{-n.y, n.x}
		gap := 1 + g.rnd.intn(8)
		for k := 0; k < 11; k++ {
			if k == gap || k == gap+1 {
				continue
			}
			g.enemyShot(e, e.pos.add(side.scale((float64(k)-5)*0.55)).add(n.scale(kd.r)), a, dmg*0.8, shotBar)
		}
	case kReconciler:
		e.cd = 2.4
		if e.slam { // a re-render: it lands by the player, where the floor glows
			for try := 0; try < 8; try++ {
				to := g.pos.add(dir(g.rnd.f() * 2 * math.Pi).scale(2.5))
				if free(to.x, to.y, kd.r) {
					g.addHazard(hazard{pos: to, r: 1.7, t: 1.0, dmg: g.contact(e) * 0.8, src: e.k, how: "re-render", owner: i, ownerK: e.k, land: true})
					break
				}
			}
			return
		}
		for k := -1; k <= 1; k++ { // props, drilled down to the player
			g.enemyShot(e, e.pos.add(n.scale(kd.r)), a+float64(k)*0.6, dmg*0.7, shotHoming)
		}
	case kTextual:
		e.cd = 2.8
		if e.slam { // widgets of its own
			for k := 0; k < 2; k++ {
				g.addEnemy(kCaret, e.pos.add(dir(g.rnd.f()*2*math.Pi).scale(1.6)))
			}
			for k := 0; k < 3; k++ {
				g.addEnemy(kCursor, e.pos.add(dir(g.rnd.f()*2*math.Pi).scale(1.6)))
			}
			g.soundFrom(sfxRoar, e.pos, 0.6)
			return
		}
		// A cascade: blocks falling on the player and round them.
		for k := 0; k < 5; k++ {
			at := g.pos
			if k > 0 {
				at = at.add(dir(float64(k)*math.Pi/2 + g.rnd.f()).scale(g.rnd.between(1.8, 3)))
			}
			g.addHazard(hazard{pos: at, r: 1.2, t: 1.1 + 0.15*float64(k), dmg: g.contact(e) * 0.7, src: e.k, how: "css cascade", owner: -1})
		}
		g.soundFrom(sfxCharge, g.pos, 0.6)
		return
	case kCrab:
		e.cd = 2.4
		g.enemyShot(e, e.pos.add(n.scale(kd.r)), a, dmg, shotBar)
	case kBobaPrime:
		e.cd = 2.2
		for i := 0; i < 10; i++ {
			g.enemyShot(e, e.pos, a+float64(i)*2*math.Pi/10+e.t, dmg, shotPearl)
		}
		for i := 0; i < 3; i++ {
			g.addEnemy(kSpinner, e.pos.add(vec{g.rnd.between(-1, 1), g.rnd.between(-1, 1)}))
		}
	case kFerris:
		e.cd = 1.4
		if e.slam {
			// The claws hit the floor: a ring of force, to be dashed through.
			g.wave = wave{alive: true, pos: e.pos, r: kd.r, dmg: kd.dmg * 1.1 * (1 + 0.03*float64(g.level)) * math.Sqrt(g.pressure())}
			g.soundFrom(sfxBoom, e.pos, 1)
			return
		}
		for i := -2; i <= 2; i++ { // five bars: each softer than a crab's, it is the fan that hurts
			g.enemyShot(e, e.pos.add(n.scale(kd.r)), a+float64(i)*0.13, dmg*0.6, shotBar)
		}
	}
	g.soundFrom(sfxSpit, e.pos, 0.7)
}

// enemyShot fires a hostile shot that remembers who fired it, and returns
// its index or -1.
func (g *game) enemyShot(e *enemy, p vec, a, dmg float64, k shotKind) int {
	i := g.shoot(p, a, dmg, 0, k)
	if i >= 0 {
		g.shots[i].src = e.k
	}
	return i
}

// slams reports whether e's next attack is its other one: Ferris's slam
// every third, the Queen's summons and the Reconciler's re-render every
// second, Textual's widgets every third.
func slams(e *enemy) bool {
	switch e.k {
	case kFerris, kTextual:
		return e.attacks%3 == 2
	case kQueen, kReconciler:
		return e.attacks%2 == 1
	}
	return false
}

// shieldNear is Lip Gloss at work: the others near it are shielded for a
// while, a pink line to each.
func (g *game) shieldNear(i int, e *enemy) {
	for j := range g.enemies {
		o := &g.enemies[j]
		if j == i || !o.alive || kinds[o.k].act == bSupport || o.pos.dist2(e.pos) > 16 {
			continue
		}
		o.shieldT = 2.5
		g.addLaser(e.pos, o.pos, hex(0xff6ad5))
	}
}

// attackName is what a hostile shot is called on the screen.
func attackName(src kindID, k shotKind) string {
	switch {
	case src == kFerris:
		return "claw salvo"
	case src == kCaret:
		return "prompt shot"
	case src == kQueen:
		return "dialog wall"
	case src == kReconciler:
		return "props drill"
	case k == shotPearl:
		return "pearl ring"
	}
	return "bar shot"
}

// hurt takes dmg from the player, and remembers where it came from and
// what it was, for the marker round the crosshair and the line over the
// health.
func (g *game) hurt(dmg float64, from vec, by kindID, how string) {
	if g.hurtT > 0 || g.phase != phPlay {
		return
	}
	dmg *= g.st.taken
	if g.hp-dmg <= 0 && g.st.phoenix && g.phoenixT <= 0 {
		g.phoenixT = 90
		g.hp = g.st.maxHP * 0.5
		g.hurtT = 1.5
		g.say("phoenix: restarted")
		g.nova(4, elFire, 6)
		g.sound(sfxLevel)
		return
	}
	g.hp -= dmg
	g.hurtT = 0.55
	g.hurtFX = 1
	g.momentum = 0
	to := from.sub(g.pos)
	g.hurtAng = math.Atan2(to.y, to.x)
	g.hurtBy, g.hurtHow, g.hurtAmt, g.hurtShow = by, how, dmg, 2.5
	g.shake = math.Max(g.shake, 0.7)
	g.sound(sfxHurt)
}

// damage hits enemy i for dmg and reports whether it died.
func (g *game) damage(i int, dmg float64, crit bool) bool { return g.hit(i, dmg, crit, 1, elNone) }

// hit is damage with its element, and how much of the enemy's armour
// counts against it (poison: none).
func (g *game) hit(i int, dmg float64, crit bool, armorK float64, el element) bool {
	e := &g.enemies[i]
	if !e.alive {
		return false
	}
	dmg *= g.vuln(e)
	// Armour takes a flat amount off every hit: a swarm of tiny shrapnel
	// does next to nothing to an elder. Corrosion eats it.
	armor := math.Max(0, e.armor-g.st.corrode*float64(e.poison)) * armorK
	dmg = math.Max(dmg-armor, dmg*0.05)
	e.hp -= dmg
	if el == elNone {
		e.flash = 1
		g.hitT = 0.12
	}
	if dmg > g.bigHit {
		g.bigHit = dmg
	}
	if g.st.execute > 0 && e.hp > 0 && e.hp < e.max*g.st.execute && !kinds[e.k].boss {
		e.hp = 0
	}
	g.damageNumber(e.pos, kinds[e.k].height, dmg, crit, el)
	if e.hp > 0 {
		return false
	}
	g.kill(i)
	return true
}

func (g *game) kill(i int) {
	e := &g.enemies[i]
	kd := &kinds[e.k]
	e.alive = false
	if kd.act == bBomb {
		g.bombGoesOff(i)
	}
	g.onDeath(i)
	g.kills++
	g.shake = math.Max(g.shake, 0.22)
	if kd.boss {
		g.shake = 1.2
	}
	g.hitT, g.killT = 0.2, 0.2
	g.burst(e)
	g.dropBytes(e.pos, 1.5*kd.xp*(1+boolf(e.elite)*2))
	g.soundFrom(sfxPop, e.pos, 0.6)
	if g.st.onKill > 0 {
		g.hp = math.Min(g.st.maxHP, g.hp+g.st.onKill)
	}
	if g.st.sync && g.slowT <= 0 && g.rnd.f() < 0.04 {
		g.slowT = 2
		g.say("mode 2026: synchronized")
	}
	for s := 0; s < kd.splitN; s++ {
		off := vec{g.rnd.between(-0.3, 0.3), g.rnd.between(-0.3, 0.3)}
		if j := g.addEnemy(kd.splitTo, e.pos.add(off)); j >= 0 {
			g.enemies[j].knock = off.scale(12)
			g.enemies[j].phaseOff = float64(s)
		}
	}
	if kd.boss {
		g.bossLive = false
		g.say(kd.name + " was swept")
		g.hp = math.Min(g.st.maxHP, g.hp+g.st.maxHP*0.3)
		for s := 0; s < 30; s++ {
			g.dropBytes(e.pos.add(vec{g.rnd.between(-1.5, 1.5), g.rnd.between(-1.5, 1.5)}), kd.xp/30)
		}
	}
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
