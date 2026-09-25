package main

import "math"

// The sewer. '#' brick, 'S' slime-covered stone, 'P' pipes, 'L' the lair,
// 'G' the lair's gate, which lifts once ten lemons are found. '@' is where
// the player starts, facing east; 'l' a lemon, 'r' a rat, 'f' a fat rat,
// 'R' Ratatui, '*' a lamp hanging from the ceiling.
var level = [...]string{
	"##############################",
	"#....#.......l#.......#......#",
	"#..*.#........#...r...#..l*..#",
	"#....#...##...#...*...#......#",
	"#@........##..SSSS.SSSS...r..#",
	"###.###...........l..........#",
	"#.....#..r...PPPP.PPPPP..#####",
	"#..l*.#......P...*...l..#....#",
	"#.....SSSS.SSS...f......#.l*.#",
	"#..............##...##..#....#",
	"####.#####..l..##...##..##.###",
	"#........#.....r.....*......l#",
	"#.l...r..#..PPPPPPPPPP.......#",
	"#...*....#..P.............f..#",
	"#..SSSS..#..P...l....P.......#",
	"#........####...*...rP...l...#",
	"LLLLLLLLLLLLLLLLLLLLLLLLLLLGLL",
	"L............................L",
	"L..*..LL............LL...*...L",
	"L.............R..............L",
	"L.....LL.....*......LL.......L",
	"LLLLLLLLLLLLLLLLLLLLLLLLLLLLLL",
}

const (
	mapW = 30
	mapH = 22

	lemonsNeeded = 10
	maxEnts      = 64
	maxParts     = 384
	maxLights    = 16
	maxCols      = 512
	maxRows      = 200 // cell rows; the picture is twice as tall in pixels

	moveSpeed   = 3.0 // tiles a second
	strafeSpeed = 2.6
	turnSpeed   = 2.5 // radians a second
	fov         = 0.66
	radius      = 0.22 // the player's, against walls

	ratHP      = 2
	fatHP      = 5
	bossHP     = 48
	ratBite    = 8
	fatBite    = 13
	bossBite   = 20
	cheeseHurt = 12
	lemonHeal  = 8

	// The squirter holds eight squirts. It never runs out of juice, but
	// refilling it takes R and a second and a half with no squirting.
	fireDelay  = 0.3
	magSize    = 8
	reloadTime = 1.5

	boardSize = 10 // scores kept on the leaderboard
	nameMax   = 12 // characters in a player's name
)

type phase uint8

const (
	phTitle phase = iota
	phPlay
	phWon
	phDead
	phName // typing the player's name, before the title menu
)

type kind uint8

const (
	kRat kind = iota + 1
	kFat
	kLemon
	kBoss
	kCheese
)

type state uint8

const (
	stGone state = iota
	stAlive
	stDying
)

// Ratatui's moods.
const (
	bossChase = iota
	bossWindup
	bossCharge
	bossThrow
)

type ent struct {
	kind   kind
	state  state
	x, y   float64
	vx, vy float64 // heading, a unit vector
	hp     int
	dying  float64 // seconds left of the death animation
	hurt   float64 // seconds left of the flash after a hit
	think  float64 // seconds until the next wandering decision
	bite   float64 // seconds until it can bite again
	lunge  float64 // seconds left of a bite's lunge
	awake  bool
	anim   float64 // walk cycle, advanced by distance walked
	phase  float64 // free-running, for bobbing and breathing
	mood   int     // Ratatui only
	moodT  float64
}

type particle struct {
	x, y, z    float64 // z is height: 0 floor, 1 ceiling
	vx, vy, vz float64
	life, max  float64
	c          rgb
	glow       float32
	size       float64 // world units
	gravity    bool
}

type light struct {
	x, y  float64
	c     rgb
	power float32
	seed  float64
}

type action uint8

const (
	actFwd action = iota
	actBack
	actTurnL
	actTurnR
	actStrafeL
	actStrafeR
	actFire
	nActions
)

type game struct {
	grid     [mapH][mapW]byte
	gateOpen bool
	gateLift float64 // 0 shut … 1 open

	phase phase
	now   float64

	px, py      float64
	dirX, dirY  float64
	planeX      float64
	planeY      float64
	vf, vs, vt  float64 // forward, strafe and turn speed
	walked      float64 // distance walked: head bob and footsteps
	stepAt      float64 // walked distance of the next footstep
	hp          int
	lemons      int
	kills       int
	shots, hits int
	finished    float64 // when the run ended
	ammo        int     // squirts left before a reload
	reloadT     float64 // seconds left of a reload, or 0
	fireCD      float64
	flash       float64 // the squirter's muzzle flash
	hitMark     float64
	hurt        float64
	pickup      float64 // the yellow glow of a lemon found
	shake       float64
	msg         string
	msgT        float64
	boss        int // index of Ratatui in ents
	bossSeen    bool
	summoned    int
	showMap     bool
	skipToBoss  bool
	quit        bool

	// The player's name, the scores kept, and where they are kept (nil in
	// tests: nothing is written to the disk or the browser there).
	name   string
	typing string // the name as it is being typed
	board  [boardSize]scoreEntry
	nBoard int
	last   result // the run that just ended
	store  store

	// The shared leaderboard, when there is a server for it (remote.go).
	remote     *remote
	world      [boardSize]scoreEntry
	nWorld     int
	worldState int
	worldRank  int // the last run's place: -1 while it is being sent

	// The title menu, and the sound: volume is 0 (off) … 10.
	menu    int
	volume  int
	lastVol int    // what switching the sound back on restores
	noSound string // why there is no sound, when there is none
	dripT   float64

	// Keys. A terminal with the kitty keyboard protocol reports releases,
	// and then a key is held exactly as long as it is. Elsewhere a key is
	// taken to be held for a moment after each press or auto-repeat.
	held, pushed [nActions]float64
	exact        bool

	ents   [maxEnts]ent
	nEnts  int
	parts  [maxParts]particle
	lights [maxLights]light
	nLight int
	rng    uint64

	audio speaker

	// flow holds each tile's distance in steps from the player's tile, or
	// -1 for walls; rats walk downhill on it, which takes them round corners
	// and pillars. It is rebuilt when the player changes tile.
	flow         [mapH][mapW]int16
	flowX, flowY int
	flowGate     bool
	queue        [mapW * mapH]int16

	// Scratch for the renderer, kept here so that a frame allocates nothing.
	cols   [maxCols]column
	fb     [maxCols * maxRows * 2]rgb
	order  [maxEnts]int
	depth  [maxEnts]float64
	near   [maxLights]int
	nNear  int
	vigX   [maxCols]float32
	vigY   [maxRows * 2]float32
	vigW   int
	vigH   int
	hudBuf [16]byte
	mm     [mmMax * mmMax]rgb // the map's pixels, and how opaque each is
	mmA    [mmMax * mmMax]float32
}

func newGame() *game {
	g := &game{volume: 7, lastVol: 7, showMap: true}
	g.reset()
	g.phase = phTitle
	return g
}

// setVolume sets the sound from 0 (off) to 10, heard as a curve: each step
// sounds about as big as the last.
func (g *game) setVolume(v int) {
	g.volume = max(0, min(10, v))
	if g.volume > 0 {
		g.lastVol = g.volume
	}
	if g.audio != nil {
		k := float64(g.volume) / 10
		g.audio.setVolume(float32(k * k))
	}
}

// reset lays the level out again. It allocates nothing, so restarting is as
// cheap as a frame.
func (g *game) reset() {
	keep := struct {
		showMap, skipToBoss, exact bool
		audio                      speaker
		volume, lastVol            int
		noSound                    string
	}{g.showMap, g.skipToBoss, g.exact, g.audio, g.volume, g.lastVol, g.noSound}
	// The name, the leaderboard and the store live in fields clearState
	// does not touch.
	// Clear the state but keep the large scratch arrays where they are.
	g.clearState()
	g.showMap, g.skipToBoss, g.exact, g.audio = keep.showMap, keep.skipToBoss, keep.exact, keep.audio
	g.volume, g.lastVol, g.noSound = keep.volume, keep.lastVol, keep.noSound
	g.rng = 0x9e3779b97f4a7c15
	g.hp = 100
	g.ammo = magSize
	g.boss = -1
	g.flowX = -1
	for y, row := range level {
		for x := 0; x < mapW; x++ {
			c := row[x]
			fx, fy := float64(x)+0.5, float64(y)+0.5
			switch c {
			case '@':
				g.px, g.py = fx, fy
			case 'l':
				g.spawn(kLemon, fx, fy)
			case 'r':
				g.spawn(kRat, fx, fy)
			case 'f':
				g.spawn(kFat, fx, fy)
			case 'R':
				g.boss = g.spawn(kBoss, fx, fy)
			case '*':
				if g.nLight < maxLights {
					l := light{x: fx, y: fy, c: hex(0xffc070), power: 1.3, seed: float64(x*7 + y)}
					if y > 16 {
						l.c, l.power = hex(0xff4060), 1.5 // the lair's lamps are red
					}
					g.lights[g.nLight] = l
					g.nLight++
				}
			}
			if c != '#' && c != 'S' && c != 'P' && c != 'L' && c != 'G' {
				c = '.'
			}
			g.grid[y][x] = c
		}
	}
	g.dirX, g.dirY = 1, 0
	g.planeX, g.planeY = 0, fov
	g.phase = phPlay
	g.stepAt = 0.8
	if g.skipToBoss {
		// Stand at the gate with the lemons already found.
		for i := 0; i < g.nEnts; i++ {
			if e := &g.ents[i]; e.kind == kLemon && g.lemons < lemonsNeeded {
				e.state = stGone
				g.lemons++
			}
		}
		g.openGate()
		g.gateLift = 1
		g.msgT = 0
		g.px, g.py = 27.5, 15.5
		g.dirX, g.dirY = 0, 1
		g.planeX, g.planeY = -fov, 0
	}
}

func (g *game) clearState() {
	g.grid = [mapH][mapW]byte{}
	g.gateOpen, g.gateLift = false, 0
	g.phase, g.now = phPlay, 0
	g.px, g.py, g.dirX, g.dirY, g.planeX, g.planeY = 0, 0, 0, 0, 0, 0
	g.vf, g.vs, g.vt, g.walked, g.stepAt = 0, 0, 0, 0, 0
	g.hp, g.lemons, g.kills, g.shots, g.hits = 0, 0, 0, 0, 0
	g.finished, g.fireCD, g.flash, g.hitMark, g.hurt, g.pickup, g.shake = 0, 0, 0, 0, 0, 0, 0
	g.ammo, g.reloadT, g.last = 0, 0, result{}
	g.msg, g.msgT = "", 0
	g.boss, g.bossSeen, g.summoned = -1, false, 0
	g.quit, g.dripT = false, 0
	g.held = [nActions]float64{}
	for i := range g.pushed {
		g.pushed[i] = math.Inf(-1) // never pressed: the first press is not a repeat
	}
	g.ents, g.nEnts = [maxEnts]ent{}, 0
	g.parts = [maxParts]particle{}
	g.nLight = 0
}

func (g *game) spawn(k kind, x, y float64) int {
	i := -1
	for j := 0; j < g.nEnts; j++ {
		if g.ents[j].state == stGone && g.ents[j].kind != kBoss {
			i = j
			break
		}
	}
	if i < 0 {
		if g.nEnts == maxEnts {
			return -1
		}
		i = g.nEnts
		g.nEnts++
	}
	e := ent{kind: k, state: stAlive, x: x, y: y, vx: 1, phase: g.rand() * 6, think: g.rand()}
	switch k {
	case kRat:
		e.hp = ratHP
	case kFat:
		e.hp = fatHP
	case kBoss:
		e.hp = bossHP
	}
	g.ents[i] = e
	return i
}

// rand is xorshift64*: repeatable, and free of the allocation and lock in
// math/rand's global source.
func (g *game) rand() float64 {
	g.rng ^= g.rng >> 12
	g.rng ^= g.rng << 25
	g.rng ^= g.rng >> 27
	return float64((g.rng*2685821657736338717)>>11) / (1 << 53)
}

func (g *game) solid(x, y int) bool {
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return true
	}
	c := g.grid[y][x]
	if c == 'G' && g.gateOpen && g.gateLift >= 1 {
		return false
	}
	return c != '.'
}

func (g *game) tile(x, y int) byte {
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return '#'
	}
	return g.grid[y][x]
}

// cast walks the grid from (x, y) along (dx, dy) and reports the first wall:
// its distance in units of the ray's length (the perpendicular distance for
// a camera ray), the cell, which face was hit, and where on the face, 0…1.
func (g *game) cast(x, y, dx, dy float64) (dist float64, mx, my int, side uint8, u float64) {
	mx, my = int(x), int(y)
	ddx, ddy := math.Inf(1), math.Inf(1)
	if dx != 0 {
		ddx = math.Abs(1 / dx)
	}
	if dy != 0 {
		ddy = math.Abs(1 / dy)
	}
	var stepX, stepY int
	var sdx, sdy float64
	if dx < 0 {
		stepX, sdx = -1, (x-float64(mx))*ddx
	} else {
		stepX, sdx = 1, (float64(mx)+1-x)*ddx
	}
	if dy < 0 {
		stepY, sdy = -1, (y-float64(my))*ddy
	} else {
		stepY, sdy = 1, (float64(my)+1-y)*ddy
	}
	for i := 0; i < 96; i++ {
		if sdx < sdy {
			sdx += ddx
			mx += stepX
			side = 0
		} else {
			sdy += ddy
			my += stepY
			side = 1
		}
		if g.solid(mx, my) {
			break
		}
	}
	if side == 0 {
		dist = sdx - ddx
		u = y + dist*dy
	} else {
		dist = sdy - ddy
		u = x + dist*dx
	}
	u -= math.Floor(u)
	// Read every face left to right, so textures are not mirrored.
	if (side == 0 && dx < 0) || (side == 1 && dy > 0) {
		u = 1 - u
	}
	return dist, mx, my, side, u
}

// sees reports whether nothing solid stands between (x, y) and the player.
func (g *game) sees(x, y float64) bool {
	dx, dy := g.px-x, g.py-y
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		return true
	}
	wall, _, _, _, _ := g.cast(x, y, dx/d, dy/d)
	return wall > d
}

// ── sound ────────────────────────────────────────────────────────────────

// emit plays a sound as heard from where the player stands: quieter with
// distance, and panned to the side it comes from.
func (g *game) emit(s sfx, x, y float64) {
	if g.audio == nil || g.volume == 0 {
		return
	}
	dx, dy := x-g.px, y-g.py
	d := math.Hypot(dx, dy)
	vol := math.Max(0, 1-d/14)
	if vol <= 0 {
		return
	}
	pan := 0.0
	if d > 0.3 {
		pan = (dx*-g.dirY + dy*g.dirX) / d
	}
	g.audio.play(s, float32(vol*vol), float32(pan))
}

func (g *game) sound(s sfx) {
	if g.audio != nil && g.volume > 0 {
		g.audio.play(s, 1, 0)
	}
}

// ── input ────────────────────────────────────────────────────────────────

func (g *game) press(a action) {
	if g.exact {
		g.held[a] = math.Inf(1)
	} else {
		// A key counts as held long enough to bridge the pause before
		// auto-repeat starts (600 ms on many desktops); each repeat keeps it
		// going a little longer.
		hold := 0.5
		if g.now-g.pushed[a] < 0.7 {
			hold = 0.12
		}
		g.held[a] = g.now + hold
	}
	g.pushed[a] = g.now
	// The opposite direction lets go of this one.
	if opp := a ^ 1; a < actFire && opp < actFire {
		g.held[opp] = 0
	}
	if a == actFire {
		g.fire()
	}
}

// release is a key let go. The first one seen proves the terminal reports
// them, and from then on keys are held exactly.
func (g *game) release(a action) {
	g.exact = true
	g.held[a] = 0
}

func (g *game) holding(a action) float64 {
	if g.held[a] > g.now {
		return 1
	}
	return 0
}

// ── combat ───────────────────────────────────────────────────────────────

// fire squirts the lemon: a hitscan along the view, stopped by the first
// wall or the first rat, with droplets flying for show.
func (g *game) fire() {
	if g.phase != phPlay || g.fireCD > 0 || g.reloadT > 0 {
		return
	}
	if g.ammo <= 0 {
		g.fireCD = fireDelay
		g.sound(sfxDry)
		g.say("Empty! Press R to reload")
		return
	}
	g.ammo--
	g.fireCD = fireDelay
	g.flash = 0.1
	g.shots++
	g.sound(sfxSquirt)
	for i := 0; i < 7; i++ {
		spread := (g.rand() - 0.5) * 0.12
		dx := g.dirX - g.dirY*spread
		dy := g.dirY + g.dirX*spread
		sp := 7 + g.rand()*3
		g.particle(g.px+g.dirX*0.3, g.py+g.dirY*0.3, 0.42,
			dx*sp, dy*sp, 0.6+g.rand()*0.8, 0.5, hex(0xffe040), 0.6, 0.035, true)
	}

	wall, _, _, _, _ := g.cast(g.px, g.py, g.dirX, g.dirY)
	best, bestD := -1, wall
	for i := 0; i < g.nEnts; i++ {
		e := &g.ents[i]
		if e.state != stAlive || e.kind == kLemon {
			continue
		}
		dx, dy := e.x-g.px, e.y-g.py
		along := dx*g.dirX + dy*g.dirY
		if along <= 0 || along >= bestD {
			continue
		}
		across := math.Abs(-dx*g.dirY + dy*g.dirX)
		// About the width of the body, not of the picture: the squirt has
		// to be aimed.
		reach := 0.26
		switch e.kind {
		case kFat:
			reach = 0.32
		case kBoss:
			reach = 0.62
		case kCheese:
			reach = 0.26
		}
		if across < reach {
			best, bestD = i, along
		}
	}
	if best < 0 {
		// A splash on the wall.
		hx, hy := g.px+g.dirX*(wall-0.05), g.py+g.dirY*(wall-0.05)
		g.splat(hx, hy, 0.45, hex(0xffe040), 6)
		return
	}
	g.hits++
	g.hitMark = 0.15
	e := &g.ents[best]
	if e.kind == kCheese {
		e.state = stGone
		g.splat(e.x, e.y, 0.4, hex(0xf0c040), 10)
		g.emit(sfxSplat, e.x, e.y)
		return
	}
	e.hurt = 0.12
	e.awake = true
	e.hp--
	// Knock it back a little.
	g.move(&e.x, &e.y, g.dirX*0.12, g.dirY*0.12, 0.2)
	z := 0.5
	if e.kind == kBoss {
		z = 0.6
		g.bossSeen = true
		g.emit(sfxBossHit, e.x, e.y)
		// It calls for help at three quarters, half and a quarter of its
		// health.
		for g.summoned < 3 && e.hp <= bossHP*(3-g.summoned)/4 {
			g.summoned++
			g.summon(e)
		}
	} else {
		g.emit(sfxSplat, e.x, e.y)
	}
	g.splat(e.x, e.y, z, hex(0xffe040), 8)
	g.splat(e.x, e.y, z, hex(0xa01818), 5)
	if e.hp <= 0 {
		e.state = stDying
		e.dying = 0.8
		if e.kind == kBoss {
			e.dying = 2.2
			g.say("RATATUI FALLS!")
			g.shake = 1
			g.emit(sfxRoar, e.x, e.y)
		} else {
			g.kills++
			g.emit(sfxRatDie, e.x, e.y)
		}
	}
}

func (g *game) summon(boss *ent) {
	g.say("Ratatui calls the rats!")
	g.emit(sfxRoar, boss.x, boss.y)
	g.shake = 0.6
	for k, off := range [...][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		x, y := boss.x+off[0]*1.2, boss.y+off[1]*1.2
		if g.solid(int(x), int(y)) {
			continue
		}
		// The last call brings a fat one along.
		kind := kRat
		if g.summoned == 3 && k == 0 {
			kind = kFat
		}
		if i := g.spawn(kind, x, y); i >= 0 {
			g.ents[i].awake = true
		}
	}
}

// stage is how far Ratatui's fight has gone: 0 at full health, 1 below two
// thirds, 2 below a third. Each makes it quicker and meaner.
func (g *game) stage() int {
	if g.boss < 0 {
		return 0
	}
	hp := g.ents[g.boss].hp
	switch {
	case hp*3 <= bossHP:
		return 2
	case hp*3 <= bossHP*2:
		return 1
	}
	return 0
}

// reload refills the squirter, if it needs it and is not already at it.
func (g *game) reload() {
	if g.phase != phPlay || g.reloadT > 0 || g.ammo == magSize {
		return
	}
	g.reloadT = reloadTime
	g.sound(sfxReload)
}

func (g *game) say(s string) {
	g.msg = s
	g.msgT = 2.5
}

func (g *game) openGate() {
	g.gateOpen = true
	g.bossSeen = true
	if g.boss >= 0 {
		g.ents[g.boss].awake = true
	}
	g.say("The gate grinds open. Ratatui is waiting.")
	g.sound(sfxGate)
	g.shake = 0.5
}

func (g *game) hurtPlayer(n int, x, y float64) {
	g.hp -= n
	g.hurt = 0.35
	g.shake = math.Max(g.shake, 0.35)
	g.sound(sfxHurt)
	// Pushed back, away from what hit.
	dx, dy := g.px-x, g.py-y
	if d := math.Hypot(dx, dy); d > 1e-6 {
		g.move(&g.px, &g.py, dx/d*0.15, dy/d*0.15, radius)
	}
}

// ── particles ────────────────────────────────────────────────────────────

func (g *game) particle(x, y, z, vx, vy, vz, life float64, c rgb, glow float32, size float64, gravity bool) {
	for i := range g.parts {
		p := &g.parts[i]
		if p.life > 0 {
			continue
		}
		*p = particle{x: x, y: y, z: z, vx: vx, vy: vy, vz: vz, life: life, max: life, c: c, glow: glow, size: size, gravity: gravity}
		return
	}
}

func (g *game) splat(x, y, z float64, c rgb, n int) {
	for i := 0; i < n; i++ {
		a := g.rand() * 2 * math.Pi
		sp := 0.6 + g.rand()*1.6
		g.particle(x, y, z, math.Cos(a)*sp, math.Sin(a)*sp, 0.5+g.rand()*1.8,
			0.4+g.rand()*0.5, c.mul(float32(0.7+g.rand()*0.4)), 0.3, 0.03+g.rand()*0.03, true)
	}
}

func (g *game) moveParticles(dt float64) {
	for i := range g.parts {
		p := &g.parts[i]
		if p.life <= 0 {
			continue
		}
		p.life -= dt
		if p.gravity {
			p.vz -= 5 * dt
		}
		nx, ny := p.x+p.vx*dt, p.y+p.vy*dt
		if g.solid(int(nx), int(ny)) {
			p.vx, p.vy = 0, 0 // stuck to the wall, and sliding down it
			if p.gravity {
				p.vz = math.Max(p.vz, -0.6)
			}
		} else {
			p.x, p.y = nx, ny
		}
		p.z += p.vz * dt
		if p.z < 0.01 {
			p.z = 0.01
			p.vx, p.vy, p.vz = p.vx*0.3, p.vy*0.3, 0
		}
	}
}

// ── simulation ───────────────────────────────────────────────────────────

func (g *game) step(dt float64) {
	if dt > 0.1 {
		dt = 0.1 // a stall is not a reason to teleport
	}
	if dt < 0 {
		dt = 0
	}
	g.now += dt
	g.pollRemote()
	for _, t := range [...]*float64{&g.fireCD, &g.flash, &g.hitMark, &g.hurt, &g.msgT, &g.pickup} {
		if *t > 0 {
			*t -= dt
		}
	}
	if g.reloadT > 0 {
		g.reloadT -= dt
		if g.reloadT <= 0 {
			g.reloadT = 0
			g.ammo = magSize
		}
	}
	g.shake = math.Max(0, g.shake-dt*2.2)
	g.moveParticles(dt)
	g.ambient(dt)
	if g.gateOpen && g.gateLift < 1 {
		g.gateLift = math.Min(1, g.gateLift+dt/1.6)
	}

	switch g.phase {
	case phTitle, phName:
		// Attract mode: the camera looks down the first corridor, swaying.
		a := 0.22 * math.Sin(g.now*0.35)
		g.dirX, g.dirY = math.Cos(a), math.Sin(a)
		g.planeX, g.planeY = -g.dirY*fov, g.dirX*fov
		return
	case phWon, phDead:
		g.animate(dt)
		return
	}

	approach := func(v *float64, target, k float64) {
		*v += (target - *v) * math.Min(1, k)
		if target == 0 && math.Abs(*v) < 0.02 {
			*v = 0 // or the player creeps on long after the key
		}
	}
	approach(&g.vf, (g.holding(actFwd)-g.holding(actBack))*moveSpeed, dt*10)
	approach(&g.vs, (g.holding(actStrafeR)-g.holding(actStrafeL))*strafeSpeed, dt*10)
	approach(&g.vt, (g.holding(actTurnR)-g.holding(actTurnL))*turnSpeed, dt*16)
	g.rotate(g.vt * dt)
	mx := (g.dirX*g.vf - g.dirY*g.vs) * dt
	my := (g.dirY*g.vf + g.dirX*g.vs) * dt
	ox, oy := g.px, g.py
	g.move(&g.px, &g.py, mx, my, radius)
	g.walked += math.Hypot(g.px-ox, g.py-oy)
	if g.walked >= g.stepAt {
		g.stepAt = g.walked + 0.8
		g.sound(sfxStep)
	}
	if g.exact && g.holding(actFire) > 0 {
		g.fire() // held down: keeps squirting
	}

	g.animate(dt)
	g.updateFlow()
	g.think(dt)

	if g.hp <= 0 {
		g.hp = 0
		g.phase = phDead
		g.finished = g.now
		g.sound(sfxLose)
		g.record(false)
	}
}

// ambient makes the sewer drip.
func (g *game) ambient(dt float64) {
	g.dripT -= dt
	if g.dripT > 0 {
		return
	}
	g.dripT = 0.25 + g.rand()*0.6
	a := g.rand() * 2 * math.Pi
	r := 1 + g.rand()*5
	x, y := g.px+math.Cos(a)*r, g.py+math.Sin(a)*r
	if g.solid(int(x), int(y)) {
		return
	}
	g.particle(x, y, 0.98, 0, 0, -0.2, 1.4, hex(0x9ac8e8), 0.25, 0.022, true)
	if g.rand() < 0.3 {
		g.emit(sfxDrip, x, y)
	}
}

func (g *game) rotate(a float64) {
	c, s := math.Cos(a), math.Sin(a)
	g.dirX, g.dirY = g.dirX*c-g.dirY*s, g.dirX*s+g.dirY*c
	g.planeX, g.planeY = g.planeX*c-g.planeY*s, g.planeX*s+g.planeY*c
}

// move slides a body of radius r by (dx, dy), one axis at a time, so it
// glides along a wall instead of sticking to it.
func (g *game) move(x, y *float64, dx, dy, r float64) {
	nx := *x + dx
	edge := nx + math.Copysign(r, dx)
	if !g.solid(int(edge), int(*y-r)) && !g.solid(int(edge), int(*y+r)) {
		*x = nx
	}
	ny := *y + dy
	edge = ny + math.Copysign(r, dy)
	if !g.solid(int(*x-r), int(edge)) && !g.solid(int(*x+r), int(edge)) {
		*y = ny
	}
}

// animate advances timers that run whatever the phase.
func (g *game) animate(dt float64) {
	for i := 0; i < g.nEnts; i++ {
		e := &g.ents[i]
		e.phase += dt
		if e.hurt > 0 {
			e.hurt -= dt
		}
		if e.lunge > 0 {
			e.lunge -= dt
		}
		if e.state == stDying {
			e.dying -= dt
			if e.kind == kBoss && g.rand() < dt*12 {
				g.splat(e.x, e.y, 0.3+g.rand()*0.8, hex(0xa01818), 2)
			}
			if e.dying <= 0 {
				e.state = stGone
				if e.kind == kBoss && g.phase == phPlay {
					g.phase = phWon
					g.finished = g.now
					g.sound(sfxWin)
					g.record(true)
				}
			}
		}
	}
}

// think moves the rats and picks up lemons.
func (g *game) think(dt float64) {
	for i := 0; i < g.nEnts; i++ {
		e := &g.ents[i]
		if e.state != stAlive {
			continue
		}
		dx, dy := g.px-e.x, g.py-e.y
		d := math.Hypot(dx, dy)
		switch e.kind {
		case kLemon:
			if d < 0.6 {
				e.state = stGone
				g.lemons++
				g.hp = min(100, g.hp+lemonHeal)
				g.pickup = 0.3
				g.say("+1 lemon, +8 HP")
				g.sound(sfxPickup)
				for j := 0; j < 14; j++ {
					g.particle(e.x, e.y, 0.35, (g.rand()-0.5)*2, (g.rand()-0.5)*2, 1+g.rand(),
						0.6, hex(0xfff080), 1, 0.025, false)
				}
				if g.lemons == lemonsNeeded {
					g.openGate()
				}
			}
		case kCheese:
			g.flyCheese(e, d, dt)
		case kRat, kFat, kBoss:
			g.beast(e, dx, dy, d, dt)
		}
	}
}

func (g *game) flyCheese(e *ent, d, dt float64) {
	nx, ny := e.x+e.vx*4.5*dt, e.y+e.vy*4.5*dt
	if g.solid(int(nx), int(ny)) {
		e.state = stGone
		g.splat(e.x, e.y, 0.4, hex(0xf0c040), 8)
		g.emit(sfxSplat, e.x, e.y)
		return
	}
	e.x, e.y = nx, ny
	if d < 0.45 {
		e.state = stGone
		g.hurtPlayer(cheeseHurt, e.x, e.y)
		g.splat(e.x, e.y, 0.4, hex(0xf0c040), 10)
	}
}

func (g *game) beast(e *ent, dx, dy, d, dt float64) {
	boss := e.kind == kBoss
	if boss && !g.gateOpen {
		return // asleep on its throne until the gate opens
	}
	if e.bite > 0 {
		e.bite -= dt
	}
	sight, chase, reach, body, bite := 9.0, 2.2, 0.62, 0.22, ratBite
	switch e.kind {
	case kFat:
		sight, chase, reach, body, bite = 9, 1.6, 0.7, 0.28, fatBite
	case kBoss:
		sight, chase, reach, body, bite = 16, 1.35, 1.0, 0.42, bossBite
	}
	if !e.awake && d < sight && g.sees(e.x, e.y) {
		e.awake = true
		g.emit(sfxSqueak, e.x, e.y)
		if boss {
			g.bossSeen = true
		}
	}
	speed := 0.7
	if boss && e.awake {
		speed = g.bossMood(e, dx, dy, d, dt)
		if e.mood == bossCharge {
			chase = speed
		}
	}
	switch {
	case boss && e.awake && e.mood != bossChase:
		// Its mood has already set the heading and speed.
	case e.awake && d < sight*1.6:
		e.vx, e.vy = g.towardPlayer(e.x, e.y, dx, dy, d)
		speed = chase
	default:
		e.think -= dt
		if e.think <= 0 {
			a := g.rand() * 2 * math.Pi
			e.vx, e.vy = math.Cos(a), math.Sin(a)
			e.think = 1 + g.rand()*2
		}
	}
	if d > reach*0.8 && speed > 0 {
		ox, oy := e.x, e.y
		g.move(&e.x, &e.y, e.vx*speed*dt, e.vy*speed*dt, body)
		moved := math.Hypot(e.x-ox, e.y-oy)
		e.anim += moved * 9
		if moved < speed*dt*0.2 {
			e.think = 0 // wandered into a wall: pick another way
			if boss && e.mood == bossCharge {
				e.mood, e.moodT = bossChase, 1 // hit the wall: dazed
				g.shake = 0.8
				g.emit(sfxBossHit, e.x, e.y)
			}
		}
	}
	if e.awake && d < reach && e.bite <= 0 {
		e.bite = 0.85
		e.lunge = 0.25
		if boss {
			e.bite = 1.1 - 0.2*float64(g.stage())
		}
		g.hurtPlayer(bite, e.x, e.y)
		g.emit(sfxBite, e.x, e.y)
	}
}

// bossMood steers Ratatui between chasing, winding up a charge, charging
// and throwing cheese, and returns how fast it moves.
//
// Each stage of the fight (see stage) makes it walk faster, wind up
// quicker, charge harder, rest less between attacks, and throw more cheese
// at once: one piece, then three in a fan, then five.
func (g *game) bossMood(e *ent, dx, dy, d, dt float64) float64 {
	st := float64(g.stage())
	walk := 1.35 + 0.25*st
	e.moodT -= dt
	switch e.mood {
	case bossChase:
		if e.moodT <= 0 && d > 2.5 && d < 9 && g.sees(e.x, e.y) {
			if g.rand() < 0.55 {
				e.mood, e.moodT = bossWindup, 0.8-0.15*st
				g.emit(sfxRoar, e.x, e.y)
			} else {
				e.mood, e.moodT = bossThrow, 0.5-0.1*st
			}
			return 0
		}
		return walk
	case bossWindup:
		e.vx, e.vy = dx/d, dy/d // lines up on the player, then commits
		if e.moodT <= 0 {
			e.mood, e.moodT = bossCharge, 0.9
		}
		return 0
	case bossCharge:
		if e.moodT <= 0 {
			e.mood, e.moodT = bossChase, (1.5+g.rand()*1.5)*(1-0.25*st)
		}
		g.shake = math.Max(g.shake, 0.25)
		return 5.5 + 0.75*st
	case bossThrow:
		if e.moodT <= 0 {
			ux, uy := dx/d, dy/d
			n := 1 + 2*int(st)
			for k := 0; k < n; k++ {
				a := (float64(k) - float64(n-1)/2) * 0.22
				ca, sa := math.Cos(a), math.Sin(a)
				vx, vy := ux*ca-uy*sa, ux*sa+uy*ca
				if i := g.spawn(kCheese, e.x+vx*0.6, e.y+vy*0.6); i >= 0 {
					c := &g.ents[i]
					c.vx, c.vy = vx, vy
				}
			}
			g.emit(sfxThrow, e.x, e.y)
			e.mood, e.moodT = bossChase, (1.2+g.rand()*1.5)*(1-0.25*st)
		}
		return 0
	}
	return walk
}

// updateFlow measures, breadth first, how many steps each tile is from the
// player's.
func (g *game) updateFlow() {
	tx, ty := int(g.px), int(g.py)
	open := g.gateOpen && g.gateLift >= 1
	if tx == g.flowX && ty == g.flowY && g.flowGate == open {
		return
	}
	g.flowX, g.flowY, g.flowGate = tx, ty, open
	for y := range g.flow {
		for x := range g.flow[y] {
			g.flow[y][x] = -1
		}
	}
	g.flow[ty][tx] = 0
	g.queue[0] = int16(ty*mapW + tx)
	head, tail := 0, 1
	for head < tail {
		i := int(g.queue[head])
		head++
		x, y := i%mapW, i/mapW
		for _, d := range [...][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if g.solid(nx, ny) || g.flow[ny][nx] >= 0 {
				continue
			}
			g.flow[ny][nx] = g.flow[y][x] + 1
			g.queue[tail] = int16(ny*mapW + nx)
			tail++
		}
	}
}

// towardPlayer is the way from (x, y) to the player: straight at them in
// the same or the next tile, otherwise to the middle of the neighbouring
// tile nearest them on the flow field.
func (g *game) towardPlayer(x, y, dx, dy, d float64) (float64, float64) {
	tx, ty := int(x), int(y)
	here := g.flow[ty][tx]
	if here < 0 || here <= 1 {
		return dx / d, dy / d
	}
	bx, by := tx, ty
	for _, n := range [...][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := tx+n[0], ty+n[1]
		if f := g.flow[ny][nx]; f >= 0 && f < g.flow[by][bx] {
			bx, by = nx, ny
		}
	}
	cx, cy := float64(bx)+0.5-x, float64(by)+0.5-y
	l := math.Hypot(cx, cy)
	if l < 1e-6 {
		return dx / d, dy / d
	}
	return cx / l, cy / l
}
