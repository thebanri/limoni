package main

import (
	"math"
	"time"
)

// Lemon Sweep: you are Limoni's garbage collector, and the frameworks have
// come for the heap. Mark them with the crosshair, sweep them with the
// trigger, take a card every level, and build something broken enough to
// get through three maps: ten minutes on each, then its warden, and the
// pressure if the warden is not swept soon — which nothing outlasts for
// long. Or /dev/zero, the endless map, where the pressure comes slower and
// never stops.

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
	phSettings // over the title or the pause, whichever opened it
	phRecipes  // the recipe book, the same way
)

// onTitle reports whether the title is what is on the screen, the settings
// over it or not.
func (g *game) onTitle() bool {
	return g.phase == phTitle || (g.phase == phSettings || g.phase == phRecipes) && g.setFrom == phTitle
}

const (
	fov      = 66 * math.Pi / 180
	playerR  = 0.25
	maxDT    = 0.1
	baseMove = 4.2
	regen    = 0.8 // health a second, before Watchdog
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
	pos        vec
	ang        float64
	hp         float64
	hurtT      float64 // invulnerable while above zero
	hurtFX     float64 // red edge, fading
	dashT      float64 // dashing while above zero
	dashN      int     // dashes ready
	dashCharge float64 // towards the next one
	dashDir    vec
	dashes     int32 // dashes made, so one runs through a foe once
	trail      vec   // where Afterburner last left fire
	bob        float64

	// what hurt the player last, for the marker and the line over health
	hurtAng  float64
	hurtBy   kindID
	hurtHow  string
	hurtAmt  float64
	hurtShow float64 // shown while above zero

	// the sweeper
	ammo    int
	reloadT float64 // collecting while above zero
	fireCD  float64
	recoil  float64
	swayX   float64 // the gun lagging behind the aim, sideways and up
	swayY   float64
	prevAim float64
	hitT    float64 // a shot landed: the crosshair's marker
	shake   float64 // the screen shaking: a kill, a blast, a hit taken
	aimOn   bool    // an enemy under the crosshair, this frame
	killT   float64 // ... and killed

	// input
	held     [nAct]bool
	heldT    [nAct]float64 // when a press without release support lets go
	exact    bool          // the terminal reports key releases
	fireKey  bool          // the trigger is held by F, not by a button
	mouseX   int
	mouseY   int
	mouseIn  bool    // a mouse position has been seen
	locked   bool    // the browser or the desktop has the pointer: look by movement
	lockHint bool    // the game's window is up but has no lock yet: a click gives it
	grab     bool    // the game's window holds the keyboard: the terminal losing it is no pause
	lookDX   float64 // movement to turn by, from the browser
	lookPend float64 // turn still to come from the terminal's mouse, in radians
	viewW    int
	viewH    int
	scrW     int // the last frame's size, for clicks on cards
	scrH     int
	sens     float64 // radians per pixel of pointer movement

	// the run
	endless   bool    // /dev/zero, not the three maps
	stage     int     // which of the three maps
	stageT    float64 // seconds on this map
	clearT    float64 // the warden is swept: the next map when this runs out
	wardenUp  bool    // the warden is in the arena
	mapKills  int     // kills when this map began
	won       bool    // all three maps swept
	level     int
	xp        float64
	kills     int
	pending   int // level-ups whose cards are still to be chosen
	rank      [nAug]int
	bonus     [nBonus]int // random stats taken when no card was left (bonus.go)
	st        stats
	cards     [3]augID
	nCards    int
	cardHov   int
	cardAcc   float64 // a locked mouse's movement towards the next card
	cardsAt   float64 // when the cards came up
	clickOK   bool    // the button was let go since: a click now is meant for a card
	bossNext  int     // the level at which the next boss comes
	bossLive  bool    // a boss is in the arena
	spawnAcc  float64 // spawn budget not yet spent
	momentum  float64 // stack overflow: damage built by hits in a row
	pullT     float64 // Boba Prime's straw: pulling while above zero (bosses.go)
	pullBy    int     // and which enemy drinks
	hits      int     // shots' hits, for Thunderstrike's seventh
	volleys   int     // trigger pulls, for Batch Job's tenth
	deferT    float64 // defer: shots hit harder while above zero
	retryT    float64 // Retry: the volley again when this runs out
	phoenixT  float64 // Phoenix Restart: ready when not above zero
	gasT      float64
	actT      [nAug]float64 // each active's clock: it acts when this runs out
	actP      [nAug]float64 // and how long that clock was set to, for the panel
	vortex    vortex
	burnAt    vec  // where Burn-in last left fire
	slamSeen  bool // the claw slam has been explained
	zones     [maxZones]zone
	hazards   [maxHazards]hazard
	seen      [nKinds]bool // the kinds that have come this run
	rings     [maxRings]ringFX
	arcs      [maxArcs]arcFX
	wave      wave
	slowT     float64 // mode 2026: time runs slow while above zero
	peakShot  int     // most shots alive at once, this run
	bigHit    float64 // the largest single hit, this run
	message   string
	msgT      float64
	newBest   bool
	deadAt    float64
	killedBy  kindID
	killedHow string
	pausedAt  float64

	// pools
	enemies   [maxEnemies]enemy
	shots     [maxShots]shot
	parts     [maxParts]particle
	nParts    int // alive in parts
	partAt    int // where to look for a free one
	bytes     [maxBytes]orb
	nShots    int
	shotTop   int // shots above this index are all dead
	bucket    [mapW * mapH]int16
	flow      [2][mapW * mapH]uint8 // steps to the player round the walls, for small and big bodies (flow.go)
	flowAt    int                   // the player's tile when it was counted
	flowArena *arena                // and the map
	nextInCel [maxEnemies]int16

	// screens
	menu    int
	pause   int
	setSel  int     // the settings row chosen
	setFrom phase   // where the settings or the recipe book go back to
	bookTop int     // the first recipe shown
	menuY   int     // the row of the menu's first button, for the mouse
	setY    int     // the row of the first setting, for the mouse
	ret     reticle // the crosshair, as chosen

	// outside
	audio   speaker
	volume  int
	lastVol int
	store   store
	best    records
	noSound string
	hint    string // a line for the title screen

	// -showfps: the frames drawn in the last whole second
	showFPS bool
	fps     int
	frames  int
	fpsAt   time.Time
}

// countFrame counts a frame drawn, and every second makes the count the
// frame rate shown.
func (g *game) countFrame(now time.Time) {
	g.frames++
	if d := now.Sub(g.fpsAt); d >= time.Second {
		g.fps = int(float64(g.frames)/d.Seconds() + 0.5)
		g.frames, g.fpsAt = 0, now
	}
}

func newGame(seed uint64) *game {
	g := &game{rnd: rng(seed | 1), volume: 7, lastVol: 7, sens: 0.0032, ret: reticle{size: 1, outline: true}}
	g.reset()
	g.phase = phTitle
	return g
}

// reset starts a new run.
func (g *game) reset() {
	best, store, audio, vol, last, sens, exact, seed := g.best, g.store, g.audio, g.volume, g.lastVol, g.sens, g.exact, g.rnd
	ret, endless := g.ret, g.endless
	showFPS, fps, frames, fpsAt := g.showFPS, g.fps, g.frames, g.fpsAt
	noSound, clock, locked, mx, my, in, hint := g.noSound, g.clock, g.locked, g.mouseX, g.mouseY, g.mouseIn, g.hint
	*g = game{}
	g.best, g.store, g.audio, g.volume, g.lastVol, g.sens, g.exact, g.rnd = best, store, audio, vol, last, sens, exact, seed
	g.noSound, g.clock, g.locked, g.mouseX, g.mouseY, g.mouseIn, g.hint = noSound, clock, locked, mx, my, in, hint
	g.showFPS, g.fps, g.frames, g.fpsAt = showFPS, fps, frames, fpsAt
	g.ret, g.endless = ret, endless
	g.enter(0)
	g.level = 1
	g.bossNext = 10
	g.recalc()
	g.hp = g.st.maxHP
	g.ammo = g.st.mag
	g.dashN = g.st.dashN
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
	if !g.endless && g.stageT < wardenAt && g.stageT+dt >= wardenAt {
		g.spawnWarden()
	}
	before := g.overtime()
	g.runTime += dt
	g.stageT += dt
	if before < 0 && g.overtime() >= 0 {
		g.say("GC pressure: the heap fights back")
		g.sound(sfxRoar)
	}
	if g.clearT > 0 {
		if g.clearT -= dt; g.clearT <= 0 {
			g.nextStage()
			return
		}
	}
	g.stepPlayer(dt)
	g.stepWeapon(dt)
	g.spawn(world)
	g.stepEnemies(world)
	g.stepZones(world)
	g.stepHazards(world)
	g.stepWave(world)
	g.stepPowers(dt)
	g.stepShots(world)
	g.stepParts(dt)
	g.stepFX(dt)
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
	g.hp = math.Min(g.st.maxHP, g.hp+g.st.regen*dt)
	if g.hurtFX > 0 {
		g.hurtFX -= dt * 2
	}
	g.hurtShow -= dt
	// Looking: the mouse turns the view by how far it moves, as in any
	// first-person game. The browser's pointer is locked; a terminal's moves
	// by whole cells, so each step is eased in, and at the window's edge,
	// where it can go no further, the view keeps turning.
	if g.locked {
		g.ang += g.lookDX * g.sens
	}
	g.lookDX = 0
	k := math.Min(1, dt*25)
	g.ang += g.lookPend * k
	g.lookPend -= g.lookPend * k
	turn := 0.0
	if g.held[actTurnL] {
		turn -= 2.6
	}
	if g.held[actTurnR] {
		turn += 2.6
	}
	if !g.locked && g.mouseIn && g.viewW > 0 {
		turn += edgeTurn(g.mouseX, g.viewW) * 2.4
	}
	g.ang += turn * dt
	g.ang = math.Remainder(g.ang, 2*math.Pi)
	// The gun lags behind the aim and springs back.
	a := g.aim()
	da := math.Remainder(a-g.prevAim, 2*math.Pi)
	g.prevAim = a
	spring := math.Min(1, dt*10)
	g.swayX += (math.Max(-0.25, math.Min(0.25, -da*2.5)) - g.swayX) * spring
	g.swayY += (-g.swayY) * spring

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
	dashing := g.dashT > 0
	if dashing {
		g.dashT -= dt
		m, speed = g.dashDir, dashSpeed*g.st.dashDist
	}
	slide(&g.pos, m.x*speed*dt, m.y*speed*dt, playerR)
	g.stepPull(dt, dashing)
	if dashing {
		g.dashStep()
		if g.dashT <= 0 {
			g.dashEnd()
		}
	}
}

const (
	dashSpeed = 17.0
	dashTime  = 0.16 // about 2.7 units
)

// edgeTurn is -1 or 1 with the mouse in the last columns on either side,
// where a terminal's pointer cannot move any further, and 0 elsewhere.
func edgeTurn(x, w int) float64 {
	switch {
	case x < 2:
		return -1
	case x >= w-2:
		return 1
	}
	return 0
}

// lookPerCell is how far the view turns for the mouse moving one column: a
// half turn across the window.
func lookPerCell(w int) float64 { return math.Pi / float64(w) }

// dash spends a charge: a short burst the way the keys point, untouchable
// all the way and a moment after — through a bite, a bar, a claw slam.
func (g *game) dash() {
	if g.dashN <= 0 || g.dashT > 0 || g.phase != phPlay {
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
	g.dashT = dashTime
	g.dashN--
	g.dashes++
	g.hurtT = math.Max(g.hurtT, dashTime+0.15)
	g.sound(sfxDash)
	g.dashStart()
}

// aim is the direction a shot leaves in: through the crosshair, in the
// middle of the picture.
func (g *game) aim() float64 { return g.ang }

func (g *game) stepWeapon(dt float64) {
	if g.fireCD > 0 {
		g.fireCD -= dt
	}
	if g.recoil > 0 {
		g.recoil -= dt * 6
	}
	g.shake = math.Max(0, g.shake-dt*4)
	g.hitT -= dt
	g.killT -= dt
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
		if g.hp < g.st.maxHP*0.4 {
			g.fireCD /= 1 + g.st.lastStand
		}
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
		g.reload()
		return
	}
	g.fireCD = 1 / g.st.fireRate
	if g.rnd.f() >= g.st.freeShot {
		g.ammo--
	}
	g.recoil = 1
	g.swayY += 0.05
	g.volley(true)
	if g.st.retry > 0 && g.rnd.f() < g.st.retry {
		g.retryT = 0.08
	}
	g.sound(sfxShot)
	if g.ammo <= 0 {
		g.reload() // the last round reloads by itself
	}
}

// volley is the shots of one trigger pull; a Retry's again is not a pull
// of its own.
func (g *game) volley(pull bool) {
	dmg := g.st.dmg
	if g.hp > g.st.maxHP*0.8 {
		dmg *= 1 + g.st.headroom
	}
	if g.deferT > 0 {
		dmg *= 1 + g.st.deferK
	}
	slug := false
	if pull {
		g.volleys++
		slug = g.st.batch > 0 && g.volleys%10 == 0
	}
	n := g.st.multishot
	a := g.aim()
	spread := 0.035 + 0.01*float64(n)
	for i := 0; i < n; i++ {
		off := 0.0
		if n > 1 {
			off = (float64(i)/float64(n-1) - 0.5) * spread * float64(n-1)
		}
		off += g.rnd.between(-0.012, 0.012)
		j := g.shoot(g.pos.add(dir(a).scale(0.3)), a+off, dmg, 0, shotPlayer)
		if slug && j >= 0 {
			g.shots[j].dmg *= float64(g.st.batch)
			g.shots[j].pierce += 4
			g.shots[j].slug = true
		}
	}
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

// die ends the run: out of memory, or every map swept.
func (g *game) die() {
	g.phase = phDead
	if !g.won {
		g.hp = 0
	}
	g.deadAt = g.clock
	g.held = [nAct]bool{}
	if g.won {
		g.sound(sfxPick)
	} else {
		g.sound(sfxDeath)
	}
	g.killedBy, g.killedHow = g.hurtBy, g.hurtHow
	if !g.endless && !g.won && g.clearT <= 0 { // a swept map is counted already
		g.best.maps[g.stage].add(false, 0, g.kills-g.mapKills)
	}
	g.newBest = g.bests().add(runRecord{
		secs: int(g.runTime), level: g.level, kills: g.kills,
		peak: g.peakShot, bigHit: int(g.bigHit),
	})
	g.persist()
}

// bests is the records this run is up against: the three maps', or
// /dev/zero's.
func (g *game) bests() *bests {
	if g.endless {
		return &g.best.endless
	}
	return &g.best.bests
}

// ── difficulty ───────────────────────────────────────────────────────────

const (
	wardenAt   = 600.0 // seconds on a map before its warden comes
	pressureAt = 720.0 // and before the pressure, if the warden still stands
	endlessAt  = 900.0 // /dev/zero's pressure: later, and a third as steep
	stageLead  = 300.0 // each map starts as hard as this far into the last
	heatPace   = 400.0 // seconds of heat that add the enemies' first health and pace again
)

// heat is how far along the slow climb the run is: the time on this map,
// and each map before it counted as five of its minutes, so a new map is a
// breath but not a fresh start. On /dev/zero, the whole run, slower: no
// map ends there to give the health back.
func (g *game) heat() float64 {
	if g.endless {
		return g.stageT * 0.6
	}
	return g.stageT + stageLead*float64(g.stage)
}

// pressureStart is when on this map the pressure begins.
func (g *game) pressureStart() float64 {
	if g.endless {
		return endlessAt
	}
	return pressureAt
}

// overtime is how far past the pressure's start this map is, in the
// pressure's own seconds; below zero it has not started. On /dev/zero a
// run is meant to go on: its pressure climbs at a third of the pace.
func (g *game) overtime() float64 {
	o := g.stageT - g.pressureStart()
	if g.endless {
		o /= 3
	}
	return o
}

// pressure is how hard the heap pushes back: 1 until the map has gone on
// too long, then it doubles about every two minutes. No build outlasts it
// for long.
func (g *game) pressure() float64 {
	if o := g.overtime(); o > 0 {
		return math.Pow(1.2, o/30)
	}
	return 1
}

// ── maps ─────────────────────────────────────────────────────────────────

// enter starts map i of the run, or /dev/zero: the arena, its clock, and
// nothing left of the last one in it.
func (g *game) enter(i int) {
	g.stage = i
	if g.endless {
		useArena(len(arenas) - 1)
	} else {
		useArena(i)
	}
	g.stageT, g.clearT, g.spawnAcc, g.bossLive, g.wardenUp = 0, 0, 0, false, false
	g.mapKills = g.kills
	g.enemies, g.shots, g.bytes = [maxEnemies]enemy{}, [maxShots]shot{}, [maxBytes]orb{}
	g.parts, g.nParts, g.partAt = [maxParts]particle{}, 0, 0
	g.nShots, g.shotTop = 0, 0
	g.zones, g.hazards, g.rings, g.arcs = [maxZones]zone{}, [maxHazards]hazard{}, [maxRings]ringFX{}, [maxArcs]arcFX{}
	g.wave, g.vortex, g.pullT = wave{}, vortex{}, 0
	g.pos, g.ang = vec{16, 16.5}, -math.Pi/2
	g.prevAim = g.ang // or the gun swings at the start of every map
}

// spawnWarden sends the map's warden: the boss that guards the way to the
// next map, tougher than one met by level.
func (g *game) spawnWarden() {
	k := wardens[g.stage]
	p := vec{16, 4}
	if g.pos.y < 16 {
		p = vec{16, 28}
	}
	if i := g.addEnemy(k, p); i >= 0 {
		e := &g.enemies[i]
		e.warden = true
		e.hp *= wardenHP[g.stage]
		e.max = e.hp
		g.bossLive, g.wardenUp = true, true
		g.say(kinds[k].name + " guards the way out of " + cur.name)
		g.sound(sfxRoar)
	}
}

var (
	wardens  = [nStages]kindID{kQueen, kReconciler, kTextual}
	wardenHP = [nStages]float64{1, 1.1, 0.7}
)

// swept is the warden down: the map is done, and the next comes in a
// moment — time to pick up what it dropped.
func (g *game) swept() {
	g.clearT, g.wardenUp = 3, false
	g.best.maps[g.stage].add(true, int(g.stageT-wardenAt), g.kills-g.mapKills)
	g.persist()
	if g.stage+1 < nStages {
		g.say(cur.name + " swept: on to " + arenas[g.stage+1].name)
	} else {
		g.say("the last warden is swept")
	}
}

func (g *game) nextStage() {
	if g.stage+1 >= nStages {
		g.won = true
		g.die()
		return
	}
	g.enter(g.stage + 1)
	g.hp = g.st.maxHP
	g.ammo, g.dashN = g.st.mag, g.st.dashN
	g.sound(sfxPick)
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
