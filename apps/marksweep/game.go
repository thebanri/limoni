package main

import "math"

// Mark & Sweep: you are Limoni's garbage collector, and the frameworks have
// come for the heap. Mark them with the crosshair, sweep them with the
// trigger, take a card every level, and build something broken enough to
// outlast the pressure — which, after ten minutes, nothing does for long.

type vec struct{ x, y float64 }

func (a vec) add(b vec) vec       { return vec{a.x + b.x, a.y + b.y} }
func (a vec) sub(b vec) vec       { return vec{a.x - b.x, a.y - b.y} }
func (a vec) scale(k float64) vec { return vec{a.x * k, a.y * k} }
func (a vec) len() float64        { return math.Hypot(a.x, a.y) }
func (a vec) dist2(b vec) float64 { dx, dy := a.x-b.x, a.y-b.y; return dx*dx + dy*dy }

func dir(a float64) vec { return vec{math.Cos(a), math.Sin(a)} }

// rng is xorshift: repeatable in tests, and nothing allocated per call.
type rng uint64

func (r *rng) next() uint64 {
	x := uint64(*r)
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	*r = rng(x)
	return x
}

func (r *rng) f() float64                   { return float64(r.next()>>11) / (1 << 53) }
func (r *rng) intn(n int) int               { return int(r.next() % uint64(n)) }
func (r *rng) between(a, b float64) float64 { return a + (b-a)*r.f() }

type phase uint8

const (
	phTitle phase = iota
	phPlay
	phCards // the run waits while a card is chosen
	phPause
	phDead
)

const (
	fov      = 66 * math.Pi / 180
	playerR  = 0.25
	maxDT    = 0.1
	baseMove = 4.2
	regen    = 1.0 // health a second
)

// action is something held down.
type action uint8

const (
	actFwd action = iota
	actBack
	actLeft
	actRight
	actTurnL
	actTurnR
	actFire
	nAct
)

type game struct {
	rnd   rng
	phase phase
	quit  bool

	// time
	clock   float64 // seconds since start-up, for animation
	runTime float64 // seconds of play in this run

	// the player
	pos     vec
	ang     float64
	hp      float64
	hurtT   float64 // invulnerable while above zero
	hurtFX  float64 // red edge, fading
	dashT   float64 // dashing while above zero
	dashCD  float64
	dashDir vec
	bob     float64

	// the sweeper
	ammo    int
	reloadT float64 // collecting while above zero
	fireCD  float64
	recoil  float64
	dryT    float64 // "the heap is full" hint

	// input
	held    [nAct]bool
	heldT   [nAct]float64 // when a press without release support lets go
	exact   bool          // the terminal reports key releases
	fireKey bool          // the trigger is held by F, not by a button
	mouseX  int
	mouseY  int
	mouseIn bool    // a mouse position has been seen
	locked  bool    // the browser has the pointer: look by movement
	lookDX  float64 // movement to turn by, from the browser
	viewW   int
	viewH   int
	scrW    int // the last frame's size, for clicks on cards
	scrH    int
	sens    float64 // radians per pixel of pointer movement

	// the run
	level    int
	xp       float64
	kills    int
	pending  int // level-ups whose cards are still to be chosen
	rank     [nAug]int
	over     [nAug]int // picks past an augment's last rank
	st       stats
	cards    [3]augID
	nCards   int
	cardHov  int
	bossNext int     // the level at which the next boss comes
	bossLive bool    // a boss is in the arena
	spawnAcc float64 // spawn budget not yet spent
	momentum float64 // stack overflow: damage built by hits in a row
	slowT    float64 // mode 2026: time runs slow while above zero
	peakShot int     // most shots alive at once, this run
	bigHit   float64 // the largest single hit, this run
	message  string
	msgT     float64
	newBest  bool
	deadAt   float64
	pausedAt float64

	// pools
	enemies   [maxEnemies]enemy
	shots     [maxShots]shot
	parts     [maxParts]particle
	bytes     [maxBytes]orb
	nShots    int
	shotTop   int // shots above this index are all dead
	bucket    [mapW * mapH]int16
	nextInCel [maxEnemies]int16

	// screens
	menu  int
	pause int

	// outside
	audio   speaker
	volume  int
	lastVol int
	store   store
	best    records
	noSound string
}

func newGame(seed uint64) *game {
	g := &game{rnd: rng(seed | 1), volume: 7, lastVol: 7, sens: 0.0032}
	g.reset()
	g.phase = phTitle
	return g
}

// reset starts a new run.
func (g *game) reset() {
	best, store, audio, vol, last, sens, exact, seed := g.best, g.store, g.audio, g.volume, g.lastVol, g.sens, g.exact, g.rnd
	noSound, clock, locked, mx, my, in := g.noSound, g.clock, g.locked, g.mouseX, g.mouseY, g.mouseIn
	*g = game{}
	g.best, g.store, g.audio, g.volume, g.lastVol, g.sens, g.exact, g.rnd = best, store, audio, vol, last, sens, exact, seed
	g.noSound, g.clock, g.locked, g.mouseX, g.mouseY, g.mouseIn = noSound, clock, locked, mx, my, in
	g.pos = vec{16, 16.5}
	g.ang = -math.Pi / 2
	g.level = 1
	g.bossNext = 10
	g.recalc()
	g.hp = g.st.maxHP
	g.ammo = g.st.mag
	g.phase = phPlay
	for i := range g.bucket {
		g.bucket[i] = -1
	}
}

func (g *game) say(s string) { g.message, g.msgT = s, 2.5 }

func (g *game) sound(s sfx) { g.soundAt(s, 1, 0) }

func (g *game) soundAt(s sfx, vol, pan float32) {
	if g.audio != nil && g.volume > 0 {
		g.audio.play(s, vol, pan)
	}
}

// soundFrom plays s from a point in the arena: quieter far away, panned to
// its side.
func (g *game) soundFrom(s sfx, p vec, vol float32) {
	d := p.sub(g.pos)
	dist := d.len()
	if dist > 18 {
		return
	}
	right := dir(g.ang + math.Pi/2)
	pan := float32(0)
	if dist > 0.01 {
		pan = float32((d.x*right.x + d.y*right.y) / dist)
	}
	g.soundAt(s, vol*float32(1/(1+dist*0.15)), pan*0.8)
}

func (g *game) setVolume(v int) {
	if v < 0 {
		v = 0
	}
	if v > 10 {
		v = 10
	}
	g.volume = v
	if v > 0 {
		g.lastVol = v
	}
	if g.audio != nil {
		g.audio.setVolume(float32(v) / 10)
	}
}

// ── the loop ─────────────────────────────────────────────────────────────

// step moves the world on by dt seconds.
func (g *game) step(dt float64) {
	if dt > maxDT {
		dt = maxDT
	}
	if dt < 0 {
		dt = 0
	}
	g.clock += dt
	if g.msgT > 0 {
		g.msgT -= dt
	}
	g.releaseStale()
	if g.phase != phPlay {
		g.lookDX = 0
		return
	}
	// Mode 2026: the world runs slow; the player does not.
	world := dt
	if g.slowT > 0 {
		g.slowT -= dt
		world = dt * 0.35
	}
	if g.runTime < 600 && g.runTime+dt >= 600 {
		g.say("GC pressure: the heap fights back")
		g.sound(sfxRoar)
	}
	g.runTime += dt
	g.stepPlayer(dt)
	g.stepWeapon(dt)
	g.spawn(world)
	g.stepEnemies(world)
	g.stepShots(world)
	g.stepParts(dt)
	g.stepBytes(dt)
	if g.hp <= 0 {
		g.die()
	} else if g.pending > 0 && g.phase == phPlay {
		g.openCards()
	}
}

func (g *game) stepPlayer(dt float64) {
	if g.hurtT > 0 {
		g.hurtT -= dt
	}
	// A slow mend, so chip damage is forgiven and swarms are not.
	g.hp = math.Min(g.st.maxHP, g.hp+regen*dt)
	if g.hurtFX > 0 {
		g.hurtFX -= dt * 2
	}
	if g.dashCD > 0 {
		g.dashCD -= dt
	}
	// Looking: the browser's pointer turns by movement; in a terminal the
	// crosshair follows the mouse, and near the edges the view turns.
	if g.locked {
		g.ang += g.lookDX * g.sens
	}
	g.lookDX = 0
	turn := 0.0
	if g.held[actTurnL] {
		turn -= 2.6
	}
	if g.held[actTurnR] {
		turn += 2.6
	}
	if !g.locked && g.mouseIn && g.viewW > 0 {
		turn += edgeTurn(g.mouseX, g.viewW) * 2.8
	}
	g.ang += turn * dt
	g.ang = math.Remainder(g.ang, 2*math.Pi)

	f := dir(g.ang)
	r := dir(g.ang + math.Pi/2)
	var m vec
	if g.held[actFwd] {
		m = m.add(f)
	}
	if g.held[actBack] {
		m = m.sub(f)
	}
	if g.held[actRight] {
		m = m.add(r)
	}
	if g.held[actLeft] {
		m = m.sub(r)
	}
	if l := m.len(); l > 0 {
		m = m.scale(1 / l)
		g.bob += dt * 9
	}
	speed := baseMove * g.st.speed
	if g.dashT > 0 {
		g.dashT -= dt
		m, speed = g.dashDir, 15
	}
	slide(&g.pos, m.x*speed*dt, m.y*speed*dt, playerR)
}

// edgeTurn is how hard the view turns with the mouse at column x of w:
// nothing in the middle, up to ±1 in the outer sixth on each side.
func edgeTurn(x, w int) float64 {
	zone := float64(w) / 6
	fx := float64(x) + 0.5
	switch {
	case fx < zone:
		return -(1 - fx/zone)
	case fx > float64(w)-zone:
		return (fx - (float64(w) - zone)) / zone
	}
	return 0
}

func (g *game) dash() {
	if g.dashCD > 0 || g.phase != phPlay {
		return
	}
	f := dir(g.ang)
	r := dir(g.ang + math.Pi/2)
	var m vec
	if g.held[actFwd] {
		m = m.add(f)
	}
	if g.held[actBack] {
		m = m.sub(f)
	}
	if g.held[actRight] {
		m = m.add(r)
	}
	if g.held[actLeft] {
		m = m.sub(r)
	}
	if m.len() == 0 {
		m = f
	}
	g.dashDir = m.scale(1 / m.len())
	g.dashT = 0.16
	g.dashCD = 1.6 * g.st.dashCD
	g.hurtT = math.Max(g.hurtT, 0.25)
	g.sound(sfxDash)
}

// aim is the direction a shot leaves in: through the crosshair, which in a
// terminal is wherever the mouse is.
func (g *game) aim() float64 {
	if g.locked || !g.mouseIn || g.viewW <= 0 {
		return g.ang
	}
	camX := 2*(float64(g.mouseX)+0.5)/float64(g.viewW) - 1
	return g.ang + math.Atan(camX*math.Tan(fov/2))
}

func (g *game) stepWeapon(dt float64) {
	if g.fireCD > 0 {
		g.fireCD -= dt
	}
	if g.recoil > 0 {
		g.recoil -= dt * 6
	}
	if g.dryT > 0 {
		g.dryT -= dt
	}
	if g.reloadT > 0 {
		g.reloadT -= dt
		if g.reloadT <= 0 {
			g.reloadT = 0
			g.ammo = g.st.mag
			g.sound(sfxReady)
		}
		return
	}
	if g.held[actFire] && g.fireCD <= 0 {
		g.fire()
	}
}

func (g *game) reload() {
	if g.reloadT > 0 || g.ammo == g.st.mag || g.phase != phPlay {
		return
	}
	g.reloadT = g.st.reload
	g.sound(sfxReload)
}

func (g *game) fire() {
	if g.ammo <= 0 {
		if g.dryT <= 0 {
			g.sound(sfxDry)
		}
		g.dryT = 1.2
		g.fireCD = 0.25
		return
	}
	g.fireCD = 1 / g.st.fireRate
	if g.rnd.f() >= g.st.freeShot {
		g.ammo--
	}
	g.recoil = 1
	n := g.st.multishot
	a := g.aim()
	spread := 0.035 + 0.01*float64(n)
	for i := 0; i < n; i++ {
		off := 0.0
		if n > 1 {
			off = (float64(i)/float64(n-1) - 0.5) * spread * float64(n-1)
		}
		off += g.rnd.between(-0.012, 0.012)
		g.shoot(g.pos.add(dir(a).scale(0.3)), a+off, g.st.dmg, 0, shotPlayer)
	}
	g.sound(sfxShot)
}

// ── levels ───────────────────────────────────────────────────────────────

// need is the experience the level after l takes.
func need(l int) float64 {
	return 5 + 2*float64(l) + math.Pow(1.12, float64(l))
}

func (g *game) gain(xp float64) {
	g.xp += xp
	for g.xp >= need(g.level) {
		g.xp -= need(g.level)
		g.level++
		g.pending++
	}
}

func (g *game) die() {
	g.phase = phDead
	g.hp = 0
	g.deadAt = g.clock
	g.held = [nAct]bool{}
	g.sound(sfxDeath)
	g.newBest = g.best.add(runRecord{
		secs: int(g.runTime), level: g.level, kills: g.kills,
		peak: g.peakShot, bigHit: int(g.bigHit),
	})
	g.persist()
}

// pressure is how hard the heap pushes back: 1 until ten minutes, then it
// doubles about every two minutes. No build outlasts it for long.
func (g *game) pressure() float64 {
	const start = 600
	if g.runTime < start {
		return 1
	}
	return math.Pow(1.2, (g.runTime-start)/30)
}

// ── input ────────────────────────────────────────────────────────────────

func (g *game) press(a action) {
	g.held[a] = true
	g.heldT[a] = g.clock
}

func (g *game) release(a action) { g.held[a] = false }

// releaseStale lets go of keys in a terminal that never says a key was let
// go: a press holds for a moment, and the key's repeats renew it.
func (g *game) releaseStale() {
	if g.exact {
		return
	}
	for a := range g.held {
		if action(a) == actFire && !g.fireKey {
			continue // the mouse says when it is let go
		}
		if g.held[a] && g.clock-g.heldT[a] > 0.55 {
			g.held[a] = false
		}
	}
}
