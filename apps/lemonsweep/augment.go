package main

import "math"

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

	// fire
	augIgnite     // hits burn
	augFirewall   // burns hotter and longer
	augWildfire   // burns spread on death
	augOverheat   // more damage to the burning
	augIncinerate // ten burns erupt
	augNapalm     // explosions burn
	augAfterburn  // dashes leave fire
	augPhoenix    // a killing blow restarts you, once in a while

	// ice
	augChill      // hits chill, and enough chill freezes
	augShatter    // the frozen take more
	augPermafrost // freezes last longer
	augShards     // the frozen burst into shards
	augColdSnap   // a dash ends in a frost nova
	augAbsZero    // a nova that freezes everything near, now and then
	augFrostbite  // the chilled take damage
	augCryo       // less damage taken

	// lightning
	augStatic    // hits shock: the shocked take more
	augThunder   // every seventh hit calls a bolt
	augTesla     // zaps the nearest, every second
	augShort     // the shocked discharge as they die
	augSparkGap  // a dash zaps what is near
	augConductor // arcs hit harder and reach further
	augSuper     // arcs carry burn and chill

	// poison
	augLeak     // hits poison, through armour
	augCorrode  // poison eats armour
	augPlague   // poison spreads on death
	augFumes    // kills leave a cloud
	augNeuro    // the poisoned are slower and softer
	augDangling // poison takes a share of health
	augGCGas    // collecting poisons what is near

	// the dash
	augDoubleBuf // a dash charge more
	augGoto      // dashes go further
	augCtxSwitch // a dash refills the heap
	augDefer     // shots after a dash hit harder
	augRam       // dashes hit what they pass

	// the rest
	augRefCount  // kills refund rounds
	augRetry     // a shot fires twice
	augBatch     // every tenth shot a slug
	augHeadroom  // damage at high health
	augLastStand // fire rate at low health
	augWeakRefs  // experience
	augGenGC     // damage to elites and bosses
	augMutex     // less damage taken
	augWatchdog  // regeneration
	augVampire   // crits heal
	augDaemon    // daemons orbit you
	augCron      // rounds fired at the nearest by themselves
	augPolyglot  // damage for every element carried
	augThermal   // burning the frozen is a blast
	augBigO      // shots grow as they fly

	// actives: the build's core, which act by themselves (Tesla Coil,
	// Daemons, Cron Job and Absolute Zero are actives too)
	augStorm      // bolts on the nearest, again and again
	augSegfault   // a fire blast where they crowd
	augGasLeak    // poison clouds thrown at them
	augColdPath   // piercing lances of ice
	augIRQ        // a field round you that shocks
	augRoundTrip  // a boomerang that comes back
	augBurnIn     // fire left wherever you walk
	augCompaction // a vortex that pulls them together, then bursts
	nAug
)

// Slots: a build holds six actives and six passives. Once a kind's slots
// are full, its cards are only upgrades of what is held.
const slots = 6

type rarity uint8

const (
	common rarity = iota
	rare
	legendary
)

var rarityName = [...]string{"COMMON", "RARE", "LEGENDARY"}

// rarityCol is each rarity's colour: silver, blue, gold.
var rarityCol = [...]uint32{0xb8c4d6, 0x4aa3ff, 0xffb020}

// element is what a card belongs to, for its tag's colour and for builds
// that come together: a card of an element already carried comes more
// often.
type element uint8

const (
	elNone element = iota
	elFire
	elIce
	elShock
	elPoison
	elDash
	nEl
)

var elementCol = [nEl]uint32{0x8a93a8, 0xff7a2a, 0x7fd8ff, 0xffe14a, 0x8ce04a, 0xc07bff}

type augment struct {
	name  string
	tag   string
	rar   rarity
	max   int
	what  string // one rank
	what2 string // the second line, if any
	needs augID  // offered only once this has a rank (or itself, for none)
	el    element
	act   bool // an active: a core ability that acts by itself
}

func init() {
	for _, a := range []augID{augTesla, augDaemon, augCron, augAbsZero, augStorm, augSegfault, augGasLeak, augColdPath, augIRQ, augRoundTrip, augBurnIn, augCompaction} {
		augs[a].act = true
	}
}

// holds is how many augments of a kind the build has.
func (g *game) holds(act bool) int {
	n := 0
	for a := augID(0); a < nAug; a++ {
		if g.rank[a] > 0 && augs[a].act == act {
			n++
		}
	}
	return n
}

var augs = [nAug]augment{
	augREP:       {"REP", "PROJECTILE", common, 5, "+1 shot per trigger pull", "repeat the last glyph", augREP, elNone, false},
	augScroll:    {"Scroll Region", "BOUNCE", common, 5, "shots bounce off walls", "+1 bounce per rank", augScroll, elNone, false},
	augDiff:      {"Diff Engine", "PIERCE", common, 5, "shots pass through", "+1 enemy per rank", augDiff, elNone, false},
	augFork:      {"fork()", "SPLIT", rare, 5, "a hit splits the shot", "+1 child per rank", augFork, elNone, false},
	augPanic:     {"panic()", "EXPLODE", rare, 5, "hits explode", "bigger, harder a rank", augPanic, elFire, false},
	augChain:     {"Goroutines", "LIGHTNING", rare, 5, "hits arc to nearby foes", "+1 arc per rank", augChain, elShock, false},
	augHotPath:   {"Hot Path", "DAMAGE", common, 5, "+20% damage", "", augHotPath, elNone, false},
	augLoop:      {"Tight Loop", "RATE", common, 5, "+15% fire rate", "", augLoop, elNone, false},
	augHeap:      {"Bigger Heap", "AMMO", common, 5, "+5 rounds a magazine", "", augHeap, elNone, false},
	augZeroAlloc: {"Zero-Alloc", "AMMO", rare, 5, "+9% chance a shot", "costs no ammo", augZeroAlloc, elNone, false},
	augConcGC:    {"Concurrent GC", "RELOAD", common, 5, "-15% reload time", "", augConcGC, elNone, false},
	augEscape:    {"Escape Analysis", "CRIT", rare, 5, "+8% critical chance", "crits x2.5, +0.5x a rank", augEscape, elNone, false},
	augShutdown:  {"Graceful Shutdown", "HEALTH", common, 5, "+25 max health", "and heals it", augShutdown, elNone, false},
	augRecover:   {"recover()", "HEALTH", rare, 5, "+1.5 health a kill", "", augRecover, elNone, false},
	augHotReload: {"Hot Reload", "SPEED", common, 5, "+9% speed", "-12% dash cooldown", augHotReload, elDash, false},
	augGOGC:      {"GOGC=200", "GROWTH", common, 5, "+12% experience", "from every freed byte", augGOGC, elNone, false},
	augCascade:   {"Cascading Diff", "EXPLODE", legendary, 3, "explosions throw shrapnel", "+3 pieces per rank", augPanic, elFire, false},
	augRecursion: {"Recursion", "SPLIT", legendary, 3, "children split again", "+1 generation per rank", augFork, elNone, false},
	augOverflow:  {"Stack Overflow", "DAMAGE", legendary, 3, "each hit in a row adds", "+3% damage, to +90%", augOverflow, elNone, false},
	augSync:      {"Mode 2026", "TIME", legendary, 1, "a kill can stop the clock", "the world slows for 2s", augSync, elNone, false},
	augKill9:     {"kill -9", "EXECUTE", legendary, 3, "foes under 8% health die", "+4% per rank", augKill9, elNone, false},

	augIgnite:     {"Thermal Throttle", "FIRE", common, 5, "hits can set foes burning", "+15% chance a rank", augIgnite, elFire, false},
	augFirewall:   {"Firewall", "FIRE", common, 5, "+30% burn damage", "burns last +1s", augIgnite, elFire, false},
	augWildfire:   {"Wildfire", "FIRE", rare, 3, "the burning spread it", "on death: +2 foes a rank", augIgnite, elFire, false},
	augOverheat:   {"Overheat", "FIRE", rare, 5, "+14% damage", "to burning foes", augIgnite, elFire, false},
	augIncinerate: {"Incinerate", "FIRE", legendary, 3, "10 burns erupt in a blast", "bigger a rank", augIgnite, elFire, false},
	augNapalm:     {"Napalm", "FIRE", rare, 3, "explosions set fire", "+2 burns a rank", augPanic, elFire, false},
	augAfterburn:  {"Afterburner", "FIRE", common, 3, "dashes leave fire behind", "it burns longer a rank", augAfterburn, elFire, false},
	augPhoenix:    {"Phoenix Restart", "FIRE", legendary, 1, "a death restarts you", "at half health, every 90s", augPhoenix, elFire, false},

	augChill:      {"Cold Start", "ICE", common, 5, "hits can chill foes", "5 chills freeze them", augChill, elIce, false},
	augShatter:    {"Shatter", "ICE", rare, 3, "+35% damage", "to frozen foes", augChill, elIce, false},
	augPermafrost: {"Permafrost", "ICE", common, 5, "freezes last +0.3s", "chill wears off slower", augChill, elIce, false},
	augShards:     {"Ice Shards", "ICE", rare, 3, "the frozen die in shards", "into 3 shards a rank", augChill, elIce, false},
	augColdSnap:   {"Cold Snap", "ICE", common, 3, "dash ends in a frost nova", "+2 chills a rank", augColdSnap, elIce, false},
	augAbsZero:    {"Absolute Zero", "ICE", rare, 5, "a frost nova: 250%, and", "freezes; sooner a rank", augAbsZero, elIce, false},
	augFrostbite:  {"Frostbite", "ICE", rare, 5, "chilled foes take damage", "6% dmg a chill, a second", augChill, elIce, false},
	augCryo:       {"Cryo Plating", "ICE", common, 5, "-5% damage taken", "", augCryo, elIce, false},

	augStatic:    {"Static", "LIGHTNING", common, 5, "hits shock foes for 2s", "shocked take +8% a rank", augStatic, elShock, false},
	augThunder:   {"Thunderstrike", "LIGHTNING", rare, 5, "every 7th hit: a bolt", "250% damage, +60% a rank", augThunder, elShock, false},
	augTesla:     {"Tesla Coil", "LIGHTNING", common, 5, "zaps the 2 nearest foes", "each second, +1 a rank", augTesla, elShock, false},
	augShort:     {"Short Circuit", "LIGHTNING", rare, 3, "the shocked discharge", "on death: +2 arcs a rank", augStatic, elShock, false},
	augSparkGap:  {"Spark Gap", "LIGHTNING", common, 3, "a dash zaps what is near", "+1 arc a rank", augSparkGap, elShock, false},
	augConductor: {"Conductor", "LIGHTNING", rare, 5, "+20% arc damage", "and +0.6m reach a rank", augChain, elShock, false},
	augSuper:     {"Superconductor", "LIGHTNING", legendary, 1, "arcs carry burn and chill", "whichever you have", augChain, elShock, false},

	augLeak:     {"Memory Leak", "POISON", common, 5, "hits can poison foes", "it ignores armour", augLeak, elPoison, false},
	augCorrode:  {"Corrosion", "POISON", rare, 3, "poison eats armour", "-1 a stack, more a rank", augLeak, elPoison, false},
	augPlague:   {"Plague", "POISON", rare, 3, "poison spreads", "on death: +2 foes a rank", augLeak, elPoison, false},
	augFumes:    {"Toxic Fumes", "POISON", rare, 3, "kills can leave a cloud", "+12% chance a rank", augFumes, elPoison, false},
	augNeuro:    {"Neurotoxin", "POISON", rare, 3, "the poisoned are slower", "-10% a rank, bite softer", augLeak, elPoison, false},
	augDangling: {"Dangling Pointer", "POISON", legendary, 1, "poison takes 1% health", "per stack/s; bosses 0.1%", augLeak, elPoison, false},
	augGCGas:    {"Mark Gas", "POISON", common, 3, "collecting poisons", "everything within 3m", augGCGas, elPoison, false},

	augDoubleBuf: {"Double Buffer", "DASH", rare, 2, "+1 dash charge", "", augDoubleBuf, elDash, false},
	augGoto:      {"goto", "DASH", common, 3, "dashes go 30% further", "", augGoto, elDash, false},
	augCtxSwitch: {"Context Switch", "DASH", rare, 1, "a dash refills the heap", "", augCtxSwitch, elDash, false},
	augDefer:     {"defer", "DASH", common, 3, "for 1s after a dash", "+40% damage a rank", augDefer, elDash, false},
	augRam:       {"Bulldozer", "DASH", rare, 3, "dashes hit what they pass", "150% damage a rank", augRam, elDash, false},

	augRefCount:   {"Reference Counting", "AMMO", common, 5, "a kill can refund a round", "+20% chance a rank", augRefCount, elNone, false},
	augRetry:      {"Retry", "PROJECTILE", rare, 3, "+12% chance a shot", "fires twice", augRetry, elNone, false},
	augBatch:      {"Batch Job", "PROJECTILE", rare, 3, "every 10th shot a slug", "x4, pierces 4, +x1 a rank", augBatch, elNone, false},
	augHeadroom:   {"Headroom", "DAMAGE", common, 5, "+10% damage", "above 80% health", augHeadroom, elNone, false},
	augLastStand:  {"Last Resort", "RATE", rare, 3, "+25% fire rate", "under 40% health", augLastStand, elNone, false},
	augWeakRefs:   {"Weak Refs", "GROWTH", common, 3, "+8% experience", "from every kill, a rank", augWeakRefs, elNone, false},
	augGenGC:      {"Generational GC", "DAMAGE", rare, 3, "+15% damage", "to elites and bosses", augGenGC, elNone, false},
	augMutex:      {"Mutex", "HEALTH", common, 5, "-6% damage taken", "", augMutex, elNone, false},
	augWatchdog:   {"Watchdog", "HEALTH", common, 5, "+0.6 health a second", "", augWatchdog, elNone, false},
	augVampire:    {"Vampire Thread", "HEALTH", rare, 3, "crits heal you", "1 health a rank", augEscape, elNone, false},
	augDaemon:     {"Daemons", "ORBIT", rare, 5, "a daemon circles you", "+1 a rank, 80% damage", augDaemon, elNone, false},
	augCron:       {"Cron Job", "AUTO", common, 5, "fires at the nearest", "every 0.7s, +1 a rank", augCron, elNone, false},
	augPolyglot:   {"Polyglot", "ELEMENT", legendary, 1, "+15% damage for every", "element you carry", augPolyglot, elNone, false},
	augThermal:    {"Thermal Shock", "FIRE+ICE", legendary, 1, "burning a frozen foe", "is a 400% blast", augChill, elIce, false},
	augBigO:       {"Big O", "PROJECTILE", rare, 3, "shots grow as they fly", "+35% dmg a second a rank", augBigO, elNone, false},
	augStorm:      {"Broadcast Storm", "LIGHTNING", rare, 5, "bolts strike the nearest", "220%, +1 bolt a rank", augStorm, elShock, false},
	augSegfault:   {"Segfault", "FIRE", common, 5, "a fire blast where", "they crowd; sooner a rank", augSegfault, elFire, false},
	augGasLeak:    {"Gas Leak", "POISON", common, 5, "throws poison clouds", "at them; bigger a rank", augGasLeak, elPoison, false},
	augColdPath:   {"Cold Path", "ICE", common, 5, "fires lances of ice", "they pierce and chill", augColdPath, elIce, false},
	augIRQ:        {"IRQ Field", "LIGHTNING", rare, 5, "a field round you", "shocks all inside", augIRQ, elShock, false},
	augRoundTrip:  {"Round Trip", "BOOMERANG", common, 5, "a boomerang that", "hits going and coming", augRoundTrip, elNone, false},
	augBurnIn:     {"Burn-in", "FIRE", rare, 5, "fire is left wherever", "you walk; longer a rank", augBurnIn, elFire, false},
	augCompaction: {"Compaction", "VORTEX", legendary, 5, "a vortex pulls them in", "then bursts, more a rank", augCompaction, elNone, false},
}

// stats are what the ranks add up to.
type stats struct {
	dmg, fireRate, reload, speed, dashCD float64
	mag, multishot                       int
	bounces, pierce                      int
	splitN, splitGen                     int
	explodeR, explodeK                   float64
	chainN                               int
	chainK, chainReach                   float64
	crit, critMul                        float64
	freeShot                             float64
	maxHP, onKill, regen                 float64
	xpK                                  float64
	shrapnel                             int
	overflow                             float64
	sync                                 bool
	execute                              float64
	bounceK                              float64 // damage a bounce adds
	taken                                float64 // what is left of a hit on the player

	// elements: a chance a hit adds a stack, and what the stacks do
	burnP, burnK, burnFor float64
	wildfire, napalm      int
	overheat              float64
	incinerate            int
	afterburn             int
	phoenix               bool

	chillP, shatter, freezeFor, chillFor, frostbite float64
	shards, coldSnap                                int
	thermal                                         bool

	shockAmp, thunderK float64
	short, spark       int
	superArc           bool

	poisonP, poisonK, corrode, fumes, neuro float64
	plague, gas                             int
	dangling                                bool

	// the dash
	dashN            int
	dashDist, deferK float64
	ctxSwitch        bool
	ram              float64

	// the rest
	refund, retry, headroom, lastStand, genGC, vampire, bigO float64
	batch                                                    int
}

// ov is the overclock multiplier of an augment: +12% a pick past its last
// rank.
func (g *game) ov(a augID) float64 { return 1 + 0.12*float64(g.over[a]) }

// rk is an augment's rank as a number, overclocks counted as a share of one.
func (g *game) rk(a augID) float64 { return float64(g.rank[a]) * g.ov(a) }

// The run starts weak on purpose: a plain sweeper and a slow mend, so the
// first cards are felt. Every level then adds a little damage of its own
// (perLevel), so the collector grows with the run whatever the cards were,
// and the build is what carries the late game.
const (
	baseDmg  = 9.0
	baseRate = 5.0
	perLevel = 0.025
)

func (g *game) recalc() {
	r := &g.rank
	s := stats{
		dmg:        baseDmg * (1 + perLevel*float64(g.level-1)) * (1 + 0.2*float64(r[augHotPath])) * g.ov(augHotPath) * g.ov(augREP),
		fireRate:   baseRate * (1 + 0.15*float64(r[augLoop])) * g.ov(augLoop),
		reload:     1.35 * pow(0.85, r[augConcGC]) / g.ov(augConcGC),
		speed:      (1 + 0.09*float64(r[augHotReload])) * (1 + 0.03*float64(g.over[augHotReload])),
		dashCD:     pow(0.88, r[augHotReload]),
		mag:        14 + 5*r[augHeap] + int(float64(14+5*r[augHeap])*0.12*float64(g.over[augHeap])),
		multishot:  1 + r[augREP],
		bounces:    r[augScroll],
		bounceK:    0.12 * float64(g.over[augScroll]),
		pierce:     r[augDiff],
		chainN:     r[augChain],
		chainK:     0.35 * g.ov(augChain) * (1 + 0.2*g.rk(augConductor)),
		chainReach: 4.5 + 0.6*float64(r[augConductor]),
		crit:       0.08 * float64(r[augEscape]),
		critMul:    2 + 0.5*float64(r[augEscape]),
		freeShot:   0.09 * float64(r[augZeroAlloc]),
		maxHP:      100 + 25*float64(r[augShutdown]),
		onKill:     1.5 * g.rk(augRecover),
		regen:      regen + 0.6*g.rk(augWatchdog),
		xpK:        (1 + 0.12*float64(r[augGOGC])) * g.ov(augGOGC) * (1 + 0.08*g.rk(augWeakRefs)),
		shrapnel:   3 * r[augCascade],
		overflow:   0.03 * g.rk(augOverflow),
		sync:       r[augSync] > 0,
		taken:      pow(0.95, r[augCryo]) * pow(0.94, r[augMutex]) / g.ov(augCryo) / g.ov(augMutex),

		burnP:      0.15 + 0.15*float64(r[augIgnite]),
		burnK:      (1 + 0.3*g.rk(augFirewall)) * g.ov(augIgnite),
		burnFor:    3 + float64(r[augFirewall]),
		wildfire:   2 * r[augWildfire],
		napalm:     2 * r[augNapalm],
		overheat:   0.14 * g.rk(augOverheat),
		incinerate: r[augIncinerate],
		afterburn:  r[augAfterburn],
		phoenix:    r[augPhoenix] > 0,

		chillP:    0.25 + 0.15*float64(r[augChill]),
		shatter:   0.35 * g.rk(augShatter),
		freezeFor: (1.2 + 0.3*float64(r[augPermafrost])) * g.ov(augChill),
		chillFor:  2.5 + 0.5*float64(r[augPermafrost]),
		frostbite: 0.06 * g.rk(augFrostbite),
		shards:    3 * r[augShards],
		coldSnap:  2 * r[augColdSnap],
		thermal:   r[augThermal] > 0,

		shockAmp: 0.08 * g.rk(augStatic),
		thunderK: (2.5 + 0.6*float64(r[augThunder])) * g.ov(augThunder),
		short:    2 * r[augShort],
		spark:    r[augSparkGap],
		superArc: r[augSuper] > 0,

		poisonP:  0.2 + 0.12*float64(r[augLeak]),
		poisonK:  g.ov(augLeak),
		corrode:  float64(r[augCorrode]),
		plague:   2 * r[augPlague],
		fumes:    0.12 * g.rk(augFumes),
		neuro:    math.Min(0.5, 0.1*g.rk(augNeuro)),
		gas:      r[augGCGas],
		dangling: r[augDangling] > 0,

		dashN:     2 + r[augDoubleBuf],
		dashDist:  1 + 0.3*g.rk(augGoto),
		ctxSwitch: r[augCtxSwitch] > 0,
		deferK:    0.4 * g.rk(augDefer),
		ram:       1.5 * g.rk(augRam),

		refund:    0.2 * g.rk(augRefCount),
		retry:     math.Min(0.8, 0.12*g.rk(augRetry)),
		batch:     3 + r[augBatch],
		headroom:  0.1 * g.rk(augHeadroom),
		lastStand: 0.25 * g.rk(augLastStand),
		genGC:     0.15 * g.rk(augGenGC),
		vampire:   g.rk(augVampire),
		bigO:      0.35 * g.rk(augBigO),
	}
	if r[augIgnite] == 0 {
		s.burnP = 0
	}
	if r[augChill] == 0 {
		s.chillP = 0
	}
	if r[augLeak] == 0 {
		s.poisonP = 0
	}
	if r[augBatch] == 0 {
		s.batch = 0
	}
	if r[augThunder] == 0 {
		s.thunderK = 0
	}
	if r[augKill9] > 0 {
		s.execute = 0.04 + 0.04*float64(r[augKill9])
	}
	if r[augFork] > 0 {
		s.splitN = 1 + r[augFork]
		s.splitGen = 1 + r[augRecursion]
	}
	if r[augPanic] > 0 {
		s.explodeR = (0.9 + 0.25*float64(r[augPanic])) * (1 + 0.05*float64(g.over[augPanic]))
		s.explodeK = (0.35 + 0.1*float64(r[augPanic])) * g.ov(augPanic)
	}
	// Polyglot: every element carried, the dash's own included.
	if r[augPolyglot] > 0 {
		var has [nEl]bool
		for a := augID(0); a < nAug; a++ {
			if r[a] > 0 {
				has[augs[a].el] = true
			}
		}
		n := 0
		for e := elFire; e < nEl; e++ {
			if has[e] {
				n++
			}
		}
		s.dmg *= 1 + 0.15*float64(n)*g.ov(augPolyglot)
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
	if g.dashN > s.dashN {
		g.dashN = s.dashN
	}
}

// carries reports whether the build has a card of element el.
func (g *game) carries(el element) bool {
	for a := augID(0); a < nAug; a++ {
		if g.rank[a] > 0 && augs[a].el == el {
			return true
		}
	}
	return false
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
	if g.rank[a] == 0 && g.holds(augs[a].act) >= slots {
		return false // its kind's slots are full: upgrades only
	}
	if a == augThermal && g.rank[augIgnite] == 0 {
		return false // it needs both
	}
	if augs[a].max == 1 && g.rank[a] == 1 {
		return false // a one-rank card has nothing to overclock
	}
	return true
}

// rarityWeight is how often a rarity comes at a level: rare and
// legendary cards grow commoner as the run goes on, so the start is plain
// and the end is broken.
func rarityWeight(r rarity, level int) float64 {
	l := float64(level)
	switch r {
	case rare:
		return math.Min(6, 2.8+0.09*l)
	case legendary:
		return math.Min(3.2, 0.35+0.06*l)
	}
	return 10
}

// openCards draws three different cards, weighted by rarity, the ones the
// player already has a little more often, so builds come together.
func (g *game) openCards() {
	g.phase = phCards
	// A trigger held when the cards come up is not a pick: the button has
	// to come up first, and the cards stay a moment before a click counts.
	g.clickOK = !g.held[actFire] || g.fireKey
	g.cardsAt = g.clock
	g.release(actFire) // the click that picks a card must not fire
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
	w := rarityWeight(augs[a].rar, g.level)
	if g.rank[a] > 0 {
		w *= 1.7
	}
	if el := augs[a].el; el != elNone && g.carries(el) {
		w *= 1.6 // an element carried pulls its cards in
	}
	if g.rank[a] >= augs[a].max {
		w *= 0.7 // overclocks come, a little less often
	}
	return w
}

// click takes card i for a mouse button, unless the button is one that
// was firing when the cards came up, or the cards have only just come.
func (g *game) click(i int) {
	if g.clickOK && g.clock-g.cardsAt >= cardGrace {
		g.take(i)
	}
}

const cardGrace = 0.4 // seconds before a click picks a card

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
	if a == augDoubleBuf {
		g.dashN++
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
