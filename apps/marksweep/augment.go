package main

// Augments are the cards offered at every level. Each is named after
// something Limoni or Go does, takes up to max ranks, and past its last
// rank keeps coming back as an overclock: +12% to what it does, without
// end. The broken builds come from the ones that multiply each other —
// shots that split, splits that bounce, bounces that explode, explosions
// that throw shrapnel — and the pressure (game.pressure) is what ends them.

type augID uint8

const (
	augREP       augID = iota // multishot
	augScroll                 // ricochet
	augDiff                   // pierce
	augFork                   // split on hit
	augPanic                  // explode on hit
	augChain                  // chain lightning
	augHotPath                // damage
	augLoop                   // fire rate
	augHeap                   // magazine
	augZeroAlloc              // free shots
	augConcGC                 // reload speed
	augEscape                 // critical hits
	augShutdown               // health
	augRecover                // heal on kill
	augHotReload              // speed and dash
	augGOGC                   // experience
	augCascade                // explosions throw shrapnel
	augRecursion              // splits split again
	augOverflow               // momentum
	augSync                   // kills slow time
	augKill9                  // execute
	nAug
)

type rarity uint8

const (
	common rarity = iota
	rare
	legendary
)

var rarityName = [...]string{"COMMON", "RARE", "LEGENDARY"}

// rarityCol is each rarity's colour: silver, blue, gold.
var rarityCol = [...]uint32{0xb8c4d6, 0x4aa3ff, 0xffb020}

type augment struct {
	name  string
	tag   string
	rar   rarity
	max   int
	what  string // one rank
	what2 string // the second line, if any
	needs augID  // offered only once this has a rank (or itself, for none)
}

var augs = [nAug]augment{
	augREP:       {"REP", "PROJECTILE", common, 5, "+1 shot per trigger pull", "repeat the last glyph", augREP},
	augScroll:    {"Scroll Region", "BOUNCE", common, 5, "shots bounce off walls", "+1 bounce per rank", augScroll},
	augDiff:      {"Diff Engine", "PIERCE", common, 5, "shots pass through", "+1 enemy per rank", augDiff},
	augFork:      {"fork()", "SPLIT", rare, 5, "a hit splits the shot", "+1 child per rank", augFork},
	augPanic:     {"panic()", "EXPLODE", rare, 5, "hits explode", "bigger and harder per rank", augPanic},
	augChain:     {"Goroutines", "CHAIN", rare, 5, "hits arc to nearby foes", "+1 arc per rank", augChain},
	augHotPath:   {"Hot Path", "DAMAGE", common, 5, "+22% damage", "", augHotPath},
	augLoop:      {"Tight Loop", "RATE", common, 5, "+16% fire rate", "", augLoop},
	augHeap:      {"Bigger Heap", "AMMO", common, 5, "+5 rounds a magazine", "", augHeap},
	augZeroAlloc: {"Zero-Alloc", "AMMO", rare, 5, "+9% chance a shot", "costs no ammo", augZeroAlloc},
	augConcGC:    {"Concurrent GC", "RELOAD", common, 5, "-15% reload time", "", augConcGC},
	augEscape:    {"Escape Analysis", "CRIT", rare, 5, "+8% critical chance", "crits hit 2.5x, +0.5x a rank", augEscape},
	augShutdown:  {"Graceful Shutdown", "HEALTH", common, 5, "+25 max health", "and heals it", augShutdown},
	augRecover:   {"recover()", "HEALTH", rare, 5, "+1.5 health a kill", "", augRecover},
	augHotReload: {"Hot Reload", "SPEED", common, 5, "+9% speed", "-12% dash cooldown", augHotReload},
	augGOGC:      {"GOGC=200", "GROWTH", common, 5, "+12% experience", "from every freed byte", augGOGC},
	augCascade:   {"Cascading Diff", "EXPLODE", legendary, 3, "explosions throw shrapnel", "+3 pieces per rank", augPanic},
	augRecursion: {"Recursion", "SPLIT", legendary, 3, "children split again", "+1 generation per rank", augFork},
	augOverflow:  {"Stack Overflow", "DAMAGE", legendary, 3, "each hit in a row adds", "+3% damage, to +90%", augOverflow},
	augSync:      {"Mode 2026", "TIME", legendary, 1, "a kill can stop the clock", "the world slows for 2s", augSync},
	augKill9:     {"kill -9", "EXECUTE", legendary, 3, "foes under 8% health die", "+4% per rank", augKill9},
}

// stats are what the ranks add up to.
type stats struct {
	dmg, fireRate, reload, speed, dashCD float64
	mag, multishot                       int
	bounces, pierce                      int
	splitN, splitGen                     int
	explodeR, explodeK                   float64
	chainN                               int
	chainK                               float64
	crit, critMul                        float64
	freeShot                             float64
	maxHP, onKill                        float64
	xpK                                  float64
	shrapnel                             int
	overflow                             float64
	sync                                 bool
	execute                              float64
	bounceK                              float64 // damage a bounce adds
}

// ov is the overclock multiplier of an augment: +12% a pick past its last
// rank.
func (g *game) ov(a augID) float64 { return 1 + 0.12*float64(g.over[a]) }

func (g *game) recalc() {
	r := &g.rank
	s := stats{
		dmg:       10 * (1 + 0.22*float64(r[augHotPath])) * g.ov(augHotPath) * g.ov(augREP),
		fireRate:  5.5 * (1 + 0.16*float64(r[augLoop])) * g.ov(augLoop),
		reload:    1.35 * pow(0.85, r[augConcGC]) / g.ov(augConcGC),
		speed:     (1 + 0.09*float64(r[augHotReload])) * (1 + 0.03*float64(g.over[augHotReload])),
		dashCD:    pow(0.88, r[augHotReload]),
		mag:       14 + 5*r[augHeap] + int(float64(14+5*r[augHeap])*0.12*float64(g.over[augHeap])),
		multishot: 1 + r[augREP],
		bounces:   r[augScroll],
		bounceK:   0.12 * float64(g.over[augScroll]),
		pierce:    r[augDiff],
		splitN:    0,
		splitGen:  0,
		chainN:    r[augChain],
		chainK:    0.35 * g.ov(augChain),
		crit:      0.08 * float64(r[augEscape]),
		critMul:   2 + 0.5*float64(r[augEscape]),
		freeShot:  0.09 * float64(r[augZeroAlloc]),
		maxHP:     100 + 25*float64(r[augShutdown]),
		onKill:    1.5 * float64(r[augRecover]) * g.ov(augRecover),
		xpK:       (1 + 0.12*float64(r[augGOGC])) * g.ov(augGOGC),
		shrapnel:  3 * r[augCascade],
		overflow:  0.03 * float64(r[augOverflow]) * g.ov(augOverflow),
		sync:      r[augSync] > 0,
		execute:   0.04 + 0.04*float64(r[augKill9]),
	}
	if r[augKill9] == 0 {
		s.execute = 0
	}
	if r[augFork] > 0 {
		s.splitN = 1 + r[augFork]
		s.splitGen = 1 + r[augRecursion]
	}
	if r[augPanic] > 0 {
		s.explodeR = (0.9 + 0.25*float64(r[augPanic])) * (1 + 0.05*float64(g.over[augPanic]))
		s.explodeK = (0.35 + 0.1*float64(r[augPanic])) * g.ov(augPanic)
	}
	s.pierce += g.over[augDiff] / 2
	s.multishot += g.over[augREP] / 3
	s.crit += 0.02 * float64(g.over[augEscape])
	s.freeShot = minf(s.freeShot+0.03*float64(g.over[augZeroAlloc]), 0.75)
	s.splitN += g.over[augFork] / 2
	s.chainN += g.over[augChain] / 2
	s.maxHP *= g.ov(augShutdown)
	g.st = s
	if g.ammo > s.mag {
		g.ammo = s.mag
	}
}

func pow(b float64, n int) float64 {
	r := 1.0
	for ; n > 0; n-- {
		r *= b
	}
	return r
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// offerable reports whether a card can be in this draw.
func (g *game) offerable(a augID) bool {
	need := augs[a].needs
	if need != a && g.rank[need] == 0 {
		return false
	}
	if augs[a].max == 1 && g.rank[a] == 1 {
		return false // a one-rank card has nothing to overclock
	}
	return true
}

var rarityWeight = [...]float64{10, 4.5, 1.6}

// openCards draws three different cards, weighted by rarity, the ones the
// player already has a little more often, so builds come together.
func (g *game) openCards() {
	g.phase = phCards
	// Keys let go while the cards are up never reach the run: let go of
	// them all now, or one stays held after the pick.
	g.held = [nAct]bool{}
	g.pending--
	g.nCards = 0
	g.cardHov = -1
	for g.nCards < 3 {
		total := 0.0
		for a := augID(0); a < nAug; a++ {
			total += g.cardWeight(a)
		}
		if total == 0 {
			break
		}
		pick := g.rnd.f() * total
		for a := augID(0); a < nAug; a++ {
			w := g.cardWeight(a)
			if pick < w {
				g.cards[g.nCards] = a
				g.nCards++
				break
			}
			pick -= w
		}
	}
	g.sound(sfxLevel)
}

func (g *game) cardWeight(a augID) float64 {
	if !g.offerable(a) {
		return 0
	}
	for i := 0; i < g.nCards; i++ {
		if g.cards[i] == a {
			return 0
		}
	}
	w := rarityWeight[augs[a].rar]
	if g.rank[a] > 0 {
		w *= 1.35
	}
	if g.rank[a] >= augs[a].max {
		w *= 0.7 // overclocks come, a little less often
	}
	return w
}

// take applies card i and goes back to the run, or to the next card.
func (g *game) take(i int) {
	if g.phase != phCards || i < 0 || i >= g.nCards {
		return
	}
	a := g.cards[i]
	if g.rank[a] < augs[a].max {
		g.rank[a]++
	} else {
		g.over[a]++
	}
	oldMax := g.st.maxHP
	g.recalc()
	if a == augShutdown {
		g.hp += g.st.maxHP - oldMax + 25
	}
	if g.hp > g.st.maxHP {
		g.hp = g.st.maxHP
	}
	g.sound(sfxPick)
	g.phase = phPlay
	if g.level >= g.bossNext && !g.bossLive {
		g.spawnBoss()
	}
	if g.pending > 0 {
		g.openCards()
	}
}
