package main

// The rules: falling blocks, sand, and the clear.
//
// A piece is four blocks, and a block is b×b grains. While it falls it is a
// rigid piece, moved and turned like any falling-block game. When it lands,
// every grain of it becomes loose sand of the piece's colour and runs down
// the pile. A run of one colour, grains touching (diagonals count), that
// reaches from the left wall to the right wall flashes and is gone: that is
// the line clear. What was resting on it falls, and may make another.
//
// Everything lives in fixed arrays in one game value made at start-up, so a
// step allocates nothing, and neither does a frame.

import "github.com/thebanri/limoni/core/cell"

const (
	cols = 10 // board width, in blocks
	rows = 18 // board height, in blocks

	minB = 2 // grains per block edge, in the smallest window
	maxB = 6 // … and the largest

	maxGW    = cols * maxB
	maxGH    = rows * maxB
	maxCells = maxGW * maxGH

	colours = 4

	hz = 60 // simulation steps a second
	dt = 1.0 / hz

	lockDelay = 0.4  // seconds a piece rests before it becomes sand
	maxResets = 12   // moves that may restart that wait, per piece
	flashTime = 0.45 // seconds a clear flashes before it goes
	dasDelay  = 0.16 // a held key starts repeating after this…
	dasRate   = 1.0 / 40
	softSpeed = 24.0 // blocks a second, with ↓ held
)

// A grain is one byte: the colour in the low three bits (0 is empty), the
// shade in the next two, and clearBit while it flashes.
const (
	colourMask = 0x07
	shadeShift = 3
	clearBit   = 0x80
)

type phase uint8

const (
	phTitle phase = iota
	phPlay
	phOver
	phName // typing the player's name
)

// shapes holds each piece's four blocks in every rotation, as (x, y) in a
// 4×4 box; made by init from the spawn orientation.
var shapes [7][4][4][2]int8

// spawn orientations (x, y), y downwards, in a box of the given size.
var spawnShapes = [7]struct {
	n     int8
	cells [4][2]int8
}{
	{4, [4][2]int8{{0, 1}, {1, 1}, {2, 1}, {3, 1}}}, // I
	{3, [4][2]int8{{0, 0}, {0, 1}, {1, 1}, {2, 1}}}, // J
	{3, [4][2]int8{{2, 0}, {0, 1}, {1, 1}, {2, 1}}}, // L
	{2, [4][2]int8{{0, 0}, {1, 0}, {0, 1}, {1, 1}}}, // O
	{3, [4][2]int8{{1, 0}, {2, 0}, {0, 1}, {1, 1}}}, // S
	{3, [4][2]int8{{1, 0}, {0, 1}, {1, 1}, {2, 1}}}, // T
	{3, [4][2]int8{{0, 0}, {1, 0}, {1, 1}, {2, 1}}}, // Z
}

// masks is shapes as bits, 4×4, bit y*4+x: what the renderer asks.
var masks [7][4]uint16

func init() {
	for k, s := range spawnShapes {
		cells := s.cells
		for r := 0; r < 4; r++ {
			shapes[k][r] = cells
			for _, c := range cells {
				masks[k][r] |= 1 << (c[1]*4 + c[0])
			}
			for i, c := range cells { // a quarter turn clockwise
				cells[i] = [2]int8{s.n - 1 - c[1], c[0]}
			}
		}
	}
}

// kicks are the nudges a turn tries when the piece does not fit where it
// is, in half blocks: sideways first, then up.
var kicks = [...][2]int8{
	{0, 0}, {-1, 0}, {1, 0}, {-2, 0}, {2, 0}, {-4, 0}, {4, 0},
	{0, -1}, {-1, -1}, {1, -1}, {0, -2},
}

type piece struct {
	kind, rot int
	colour    uint8
	x, y      int     // top-left of the 4×4 box, in grains; y may be negative
	fall      float64 // fraction of a grain fallen since the last whole one
}

type game struct {
	phase  phase
	paused bool
	quit   bool

	b      int // grains per block edge
	gw, gh int // board size in grains

	sand  [maxCells]uint8
	mark  [maxCells]uint32 // flood-fill visits, by stamp
	stamp uint32
	stack [maxCells]int32
	clr   [maxCells]int32 // the grains flashing now
	nclr  int

	vel     [maxCells]uint8 // a grain's speed down, in 1/32 grain a step; 0 at rest
	frac    [maxCells]uint8 // how far it has got towards its next cell, in 1/32
	awake   [maxGH]bool     // rows the next pass visits: see fall
	settled bool            // no row is awake, so the sand has nothing to do
	flashT  float64         // time left on the current flash; 0 when none
	combo   int             // clears since the last piece landed

	cur, next piece
	bag       [7]int
	nbag      int
	resting   float64 // seconds the piece has been unable to fall
	resets    int
	holdL     bool
	holdR     bool
	holdDown  bool
	dasT      float64
	exact     bool // the terminal reports key releases
	dropping  bool // the piece landing now was dropped, not let down

	audio    speaker // nil: no sound
	muted    bool
	noSound  string  // why there is no sound, when there is none
	trickleT float64 // until the running sand's next sound
	grainsMv int     // grains that moved on the last pass

	score, best int
	clears      int
	level       int
	gain        int     // the last clear's points, for the panel
	gainT       float64 // how long ago it was scored
	gainCombo   int

	now, acc float64
	ticks    uint64
	overAt   float64
	attractT float64
	rng      uint64
	store    store

	// The player, and the boards: the game's own and the world's.
	name, typing string
	back         phase // where the name entry goes back to
	local        [boardSize]scoreEntry
	nLocal       int
	localRank    int // the last run's place on the game's own board
	world        [boardSize]scoreEntry
	nWorld       int
	worldState   int
	worldRank    int // the last run's place on the world's: -1 sending
	remote       *remote

	// What the run did, for the leaderboard to score again.
	pieces   int
	dropPts  int
	clearLog [maxLog][2]int32
	nlog     int
	startAt  float64

	// The picture: see render.go.
	bg                   [maxCells]cell.Color
	bgFor                int // the b the background was drawn for
	spin, spinDrawn      float64
	ndyn                 int // the pixels the lemon's turning changes:
	dynIdx               [maxCells]int32
	dynD, dynT           [maxCells]float32 // distance in radii, angle in turns
	dynFlesh             [maxCells][3]float32
	lastW, lastH         int
	bx, by, panelX, topY int
}

func newGame(seed uint64) *game {
	g := &game{rng: seed | 1, level: 1}
	g.setSize(4)
	return g
}

// rnd is xorshift64*.
func (g *game) rnd() uint64 {
	g.rng ^= g.rng >> 12
	g.rng ^= g.rng << 25
	g.rng ^= g.rng >> 27
	return g.rng * 2685821657736338717
}

func (g *game) intn(n int) int { return int(g.rnd() >> 33 % uint64(n)) }

// setSize empties the board and sizes it at b grains to a block.
func (g *game) setSize(b int) {
	g.b, g.gw, g.gh = b, cols*b, rows*b
	clear(g.sand[:])
	clear(g.vel[:])
	clear(g.frac[:])
	clear(g.awake[:])
	g.nclr, g.flashT, g.settled = 0, 0, true
}

// wake has the next pass visit rows y0 to y1, inclusive.
func (g *game) wake(y0, y1 int) {
	for y := max(y0, 0); y <= min(y1, g.gh-1); y++ {
		g.awake[y] = true
	}
	g.settled = false
}

// fitSize is the largest b whose board and panel fit a w×h window, or 0.
func fitSize(w, h int) int {
	for b := maxB; b >= minB; b-- {
		if w >= cols*b+2+gap+panelW && h >= rows*b/2+2 {
			return b
		}
	}
	return 0
}

// start begins a run on an empty board.
func (g *game) start() {
	b := fitSize(g.lastW, g.lastH)
	if b == 0 {
		b = minB
	}
	g.setSize(b)
	g.phase, g.paused = phPlay, false
	g.score, g.clears, g.level, g.combo = 0, 0, 1, 0
	g.gain, g.gainT = 0, 99
	g.pieces, g.dropPts, g.nlog, g.startAt = 0, 0, 0, g.now
	g.localRank, g.worldRank = 0, 0
	g.holdL, g.holdR, g.holdDown = false, false, false
	g.nbag = 0
	g.next = g.draw()
	g.spawn()
	g.sound(sfxStart, 0.8)
}

// draw deals the next piece from a shuffled bag of all seven, in one of the
// colours at random.
func (g *game) draw() piece {
	if g.nbag == 0 {
		for i := range g.bag {
			g.bag[i] = i
		}
		for i := len(g.bag) - 1; i > 0; i-- {
			j := g.intn(i + 1)
			g.bag[i], g.bag[j] = g.bag[j], g.bag[i]
		}
		g.nbag = len(g.bag)
	}
	g.nbag--
	return piece{kind: g.bag[g.nbag], colour: uint8(1 + g.intn(colours))}
}

// spawn puts the next piece at the top, centred. If it does not fit, the
// run is over.
func (g *game) spawn() bool {
	p := g.next
	g.next = g.draw()
	top, wide := int8(4), int8(0)
	for _, c := range shapes[p.kind][0] {
		top = min(top, c[1])
		wide = max(wide, c[0]+1)
	}
	p.rot, p.fall = 0, 0
	p.x = (g.gw - int(wide)*g.b) / 2
	p.y = -int(top) * g.b
	g.cur = p
	g.resting, g.resets = 0, 0
	return g.fits(p.kind, p.rot, p.x, p.y)
}

// fits reports whether a piece in that place overlaps neither a wall, the
// floor nor a grain. Above the top is open.
func (g *game) fits(kind, rot, x, y int) bool {
	b := g.b
	for _, c := range shapes[kind][rot] {
		x0, y0 := x+int(c[0])*b, y+int(c[1])*b
		if x0 < 0 || x0+b > g.gw || y0+b > g.gh {
			return false
		}
		for yy := max(y0, 0); yy < y0+b; yy++ {
			row := g.sand[yy*g.gw+x0 : yy*g.gw+x0+b]
			for _, v := range row {
				if v != 0 {
					return false
				}
			}
		}
	}
	return true
}

// dropY is where the piece would land, straight down.
func (g *game) dropY() int {
	p := &g.cur
	y := p.y
	for g.fits(p.kind, p.rot, p.x, y+1) {
		y++
	}
	return y
}

func (g *game) step() int { return max(1, g.b/2) }

// shift moves the piece sideways by dx grains, if it fits.
func (g *game) shift(dx int) bool {
	p := &g.cur
	if !g.fits(p.kind, p.rot, p.x+dx, p.y) {
		return false
	}
	p.x += dx
	g.moved()
	g.sound(sfxMove, 0.5)
	return true
}

// turn rotates the piece a quarter (dir 1 clockwise, -1 the other way),
// nudging it off a wall or a pile when it has to.
func (g *game) turn(dir int) bool {
	p := &g.cur
	r := (p.rot + dir + 4) % 4
	for _, k := range kicks {
		x, y := p.x+int(k[0])*g.b/2, p.y+int(k[1])*g.b/2
		if g.fits(p.kind, r, x, y) {
			p.rot, p.x, p.y = r, x, y
			g.moved()
			g.sound(sfxTurn, 0.7)
			return true
		}
	}
	return false
}

// moved restarts the lock wait, a limited number of times, so a piece can
// be slid along the pile but not held up for ever.
func (g *game) moved() {
	if g.resting > 0 && g.resets < maxResets {
		g.resting = 0
		g.resets++
	}
}

// hardDrop puts the piece where it would land and makes it sand at once.
func (g *game) hardDrop() {
	y := g.dropY()
	pts := (y - g.cur.y) / g.b * 2
	g.score += pts
	g.dropPts += pts
	g.cur.y = y
	g.dropping = true
	g.lock()
}

// shadeAt is a falling block's shading at a grain inside it: lit on the top
// and left edges, dark on the bottom and right, speckled inside. It is for
// the piece in the air only; landed grains take shades at random (lock).
func shadeAt(lx, ly, b int) uint8 {
	switch {
	case lx == 0 || ly == 0:
		return 3
	case lx == b-1 || ly == b-1:
		return 0
	}
	return 1 + uint8((lx*5+ly*3)%3&1)
}

// lock turns the piece into sand and deals the next one.
func (g *game) lock() {
	p := &g.cur
	b := g.b
	for _, c := range shapes[p.kind][p.rot] {
		x0, y0 := p.x+int(c[0])*b, p.y+int(c[1])*b
		for ly := 0; ly < b; ly++ {
			y := y0 + ly
			if y < 0 {
				continue
			}
			for lx := 0; lx < b; lx++ {
				i := y*g.gw + x0 + lx
				if g.sand[i] != 0 {
					continue
				}
				// Each grain gets a shade of its own, so the block is sand
				// the moment it lands, not a block drawn in grains; and a
				// start of its own, so its grains do not move in step.
				r := g.rnd()
				g.sand[i] = p.colour | sandShade[r&7]<<shadeShift
				if y < g.gh-1 {
					g.vel[i] = landSpeed
				}
				g.frac[i] = uint8(r>>8) & 31
			}
		}
	}
	g.wake(p.y, p.y+4*b-1)
	g.combo = 0
	if g.phase != phPlay {
		return
	}
	g.pieces++
	if g.dropping {
		g.sound(sfxDrop, 1)
	} else {
		g.sound(sfxLand, 0.8)
	}
	g.dropping = false
	if !g.spawn() {
		g.gameOver()
	}
}

func (g *game) gameOver() {
	g.phase, g.overAt = phOver, g.now
	g.holdL, g.holdR, g.holdDown = false, false, false
	g.sound(sfxOver, 1)
	g.best = max(g.best, g.score)
	e := scoreEntry{Name: g.name, Score: g.score, Clears: g.clears, Level: g.level, Secs: g.runSecs()}
	if e.Name == "" {
		e.Name = "?"
	}
	g.localRank = g.addLocal(e)
	g.persist()
	g.sendRun()
}

// runSecs is how long the run has lasted, in whole seconds, at least one.
func (g *game) runSecs() int { return max(1, int(g.now-g.startAt)) }

// fallSpeed is how fast pieces fall at a level, in blocks a second.
func fallSpeed(level int) float64 {
	s := 1.2
	for i := 1; i < level; i++ {
		s *= 1.2
	}
	return min(s, 20)
}

// update advances the game by a frame's worth of time, in fixed steps.
func (g *game) update(elapsed float64) {
	if g.paused {
		return
	}
	g.acc += min(elapsed, 0.1)
	for g.acc >= dt {
		g.acc -= dt
		g.tick()
	}
}

// tick is one simulation step.
func (g *game) tick() {
	g.now += dt
	g.ticks++
	g.pollRemote()
	g.spin += dt / spinTurn
	g.gainT += dt

	switch g.phase {
	case phPlay:
		g.pieceTick()
	case phTitle:
		g.attract()
	}
	g.sandTick()
}

func (g *game) pieceTick() {
	p := &g.cur
	b := g.b

	// Held arrows repeat, when the terminal says when they are let go.
	if g.exact && g.holdL != g.holdR {
		g.dasT += dt
		for g.dasT >= dasDelay+dasRate {
			g.dasT -= dasRate
			if g.holdL {
				g.shift(-g.step())
			} else {
				g.shift(g.step())
			}
		}
	}

	speed := fallSpeed(g.level)
	if g.holdDown {
		speed = max(speed, softSpeed)
	}
	p.fall += speed * float64(b) * dt
	for p.fall >= 1 {
		if !g.fits(p.kind, p.rot, p.x, p.y+1) {
			p.fall = 0
			break
		}
		p.fall--
		p.y++
		g.resting = 0
		if g.holdDown && p.y%b == 0 {
			g.score++
			g.dropPts++
		}
	}
	if !g.fits(p.kind, p.rot, p.x, p.y+1) {
		g.resting += dt
		if g.resting >= lockDelay || g.holdDown && g.resting >= lockDelay/4 {
			g.lock()
		}
	}
}

// attract plays by itself behind the title: a piece every so often,
// dropped at random, and a fresh board when it fills.
func (g *game) attract() {
	g.attractT -= dt
	if g.attractT > 0 {
		return
	}
	g.attractT = 0.55
	p := g.draw()
	p.rot = g.intn(4)
	g.cur = p
	g.cur.x = g.intn(g.gw - 2*g.b)
	g.cur.y = -4 * g.b
	for !g.fits(p.kind, p.rot, g.cur.x, g.cur.y) && g.cur.x > 0 {
		g.cur.x--
	}
	y := g.dropY()
	if y < 0 {
		g.setSize(g.b)
		return
	}
	g.cur.y = y
	g.lock()
}

// sandTick lets the sand fall, finds a clear, and ends one.
func (g *game) sandTick() {
	if g.flashT > 0 {
		g.flashT -= dt
		if g.flashT <= 0 {
			g.flashT = 0
			for _, i := range g.clr[:g.nclr] {
				g.sand[i] = 0
				// What rested on it may fall now.
				y := int(i) / g.gw
				g.wake(y-1, y-1)
			}
			g.nclr = 0
		}
	}
	if g.settled {
		return
	}
	loose := g.fall()
	g.trickle()
	if g.flashT == 0 {
		g.findClear()
	}
	if !loose && g.flashT == 0 {
		g.settled = true
	}
}

// The sand's motion. A grain is either at rest or loose; a loose grain
// gains speed every step, from nothing up to a top speed that grows with the
// grains to a block, so that sand falls about as many blocks a second in
// any window. Speeds are under a cell a step, so a grain moves at most one
// cell a step, and a falling column spreads out a little as it goes, the
// way poured sand does.
const (
	gravity   = 1 // speed gained a step, in 1/32 grain
	landSpeed = 4 // the speed a landed piece's grains start at
)

// sandShade is a grain's shade from three random bits: mostly the middle
// two, now and then a dark or a light one.
var sandShade = [8]uint8{0, 1, 1, 1, 2, 2, 2, 3}

func (g *game) topSpeed() uint8 { return uint8(min(31, 32*g.b/5)) }

// fall moves the sand one step. A loose grain speeds up, and when it has
// gathered a whole cell it moves: down if it can, else down to one side. A
// grain that can go nowhere comes to rest, unless the grain under it is
// still falling, in which case it waits for it. Sliding down the side of a
// pile is slower than falling, and a grain sometimes hesitates before it
// slides, so a pile crumbles rather than collapsing in one step. The pass
// runs bottom-up, so a grain moves at most once, and alternates direction
// by row and by step so piles do not lean.
//
// Most of a board is at rest most of the time, so a pass visits only the
// rows that can hold a loose grain: those that held one after the last
// step, those woken by a landed piece or a clear, and the row above one a
// grain has just left, in the same pass, since the pass is going upwards
// anyway. A settled pile costs nothing. fall reports whether any grain is
// still loose.
func (g *game) fall() bool {
	gw, gh := g.gw, g.gh
	s := g.sand[:gw*gh]
	vel, frac := g.vel[:gw*gh], g.frac[:gw*gh]
	top := g.topSpeed()
	loose, left := false, false // left: a grain left the row below, this pass
	g.grainsMv = 0
	r := g.rng
	for y := gh - 2; y >= 0; y-- {
		if !g.awake[y] && !left {
			continue
		}
		g.awake[y], left = false, false
		base := y * gw
		rev := (uint64(y)+g.ticks)&1 == 0
		for k := 0; k < gw; k++ {
			x := k
			if rev {
				x = gw - 1 - k
			}
			i := base + x
			v := s[i]
			if v == 0 || v&clearBit != 0 {
				continue
			}
			// Where it can go: straight down, or else to a side, at random.
			down, to := i+gw, -1
			if s[down] == 0 {
				to = down
			} else {
				r ^= r << 13
				r ^= r >> 7
				r ^= r << 17
				d := 1
				if r&1 == 0 {
					d = -1
				}
				switch {
				case x+d >= 0 && x+d < gw && s[down+d] == 0:
					to = down + d
				case x-d >= 0 && x-d < gw && s[down-d] == 0:
					to = down - d
				}
			}
			if to < 0 {
				// (The floor row is never visited, so a grain there is at
				// rest whatever its speed says.)
				if y+1 < gh-1 && vel[down] != 0 && s[down]&clearBit == 0 {
					// Waiting on a falling grain: ready to follow it.
					frac[i] = 31
					g.awake[y] = true
					loose = true
				} else {
					vel[i], frac[i] = 0, 0
				}
				continue
			}
			sp := min(vel[i]+gravity, top)
			if to != down {
				// A slide: slower, and not every grain goes at once.
				sp = min(sp, top/2)
				if r&0x30 == 0 {
					vel[i] = max(sp, 1)
					g.awake[y] = true
					loose = true
					continue
				}
			}
			f := frac[i] + sp
			if f < 32 {
				vel[i], frac[i] = sp, f
				g.awake[y] = true
				loose = true
				continue
			}
			s[i], vel[i], frac[i] = 0, 0, 0
			left = true
			g.grainsMv++
			if y+1 == gh-1 {
				// The floor: nothing below it to fall into, and no pass
				// visits it to bring the grain to rest.
				s[to] = v
				continue
			}
			s[to], vel[to], frac[to] = v, sp, f-32
			loose = true
			g.awake[y+1] = true
		}
	}
	g.rng = r | 1
	return loose
}

// wakeAll has the next pass visit every row: for a board changed by hand.
func (g *game) wakeAll() { g.wake(0, g.gh-1) }

// findClear looks for a run of one colour from wall to wall. Only a run
// that touches the left wall can, so the search starts from that column;
// each grain is visited at most once. A run found starts to flash.
func (g *game) findClear() {
	gw, gh := g.gw, g.gh
	g.stamp++
	if g.stamp == 0 { // wrapped: old marks could pass for new ones
		clear(g.mark[:])
		g.stamp = 1
	}
	st := g.stamp
	found := 0
	for y := 0; y < gh; y++ {
		i := y * gw
		v := g.sand[i]
		if v == 0 || v&clearBit != 0 || g.mark[i] == st {
			continue
		}
		c := v & colourMask
		start := g.nclr
		g.mark[i] = st
		g.stack[0] = int32(i)
		sp := 1
		right := false
		for sp > 0 {
			sp--
			j := int(g.stack[sp])
			g.clr[g.nclr] = int32(j)
			g.nclr++
			x, yy := j%gw, j/gw
			if x == gw-1 {
				right = true
			}
			for ny := max(yy-1, 0); ny <= min(yy+1, gh-1); ny++ {
				for nx := max(x-1, 0); nx <= min(x+1, gw-1); nx++ {
					n := ny*gw + nx
					w := g.sand[n]
					if w&(colourMask|clearBit) != c || g.mark[n] == st {
						continue
					}
					g.mark[n] = st
					g.stack[sp] = int32(n)
					sp++
				}
			}
		}
		if !right {
			g.nclr = start
			continue
		}
		for _, j := range g.clr[start:g.nclr] {
			g.sand[j] |= clearBit
		}
		found += g.nclr - start
	}
	if found == 0 {
		return
	}
	g.flashT = flashTime
	if g.phase != phPlay {
		return
	}
	g.combo++
	g.clears++
	level := g.level
	g.level = 1 + g.clears/4
	if g.combo > 1 {
		g.sound(sfxChain, 1)
	} else {
		g.sound(sfxClear, 1)
	}
	if g.level > level {
		g.sound(sfxLevel, 0.8)
	}
	// Points are per block's worth of grains, so the score does not depend
	// on the size of the window.
	base := (found*10 + g.b*g.b/2) / (g.b * g.b)
	g.logClear(base, g.combo)
	pts := base * g.level * g.combo
	g.score += pts
	g.gain, g.gainT, g.gainCombo = pts, 0, g.combo
}

// ── sound ────────────────────────────────────────────────────────────────

// sound plays s, heard from where the piece is, during a run only: the
// title's pieces play by themselves, and quietly.
func (g *game) sound(s sfx, vol float32) {
	if g.audio == nil || g.muted || g.phase != phPlay && s != sfxOver {
		return
	}
	mid := float32(g.cur.x+2*g.b)/float32(g.gw)*2 - 1
	g.audio.play(s, vol, max(-0.6, min(0.6, mid*0.6)))
}

// trickle is the sound of sand running: a short hiss again and again while
// enough grains move, louder the more of them.
func (g *game) trickle() {
	if g.trickleT > 0 {
		g.trickleT -= dt
	}
	if g.grainsMv < g.b*g.b || g.trickleT > 0 {
		return
	}
	g.trickleT = 0.1
	vol := min(1, float32(g.grainsMv)/float32(g.gw*2))
	if g.audio != nil && !g.muted && g.phase == phPlay {
		g.audio.play(sfxTrickle, 0.25+0.55*vol, 0)
	}
}
