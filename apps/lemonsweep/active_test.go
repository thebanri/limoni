package main

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Six actives and six passives: once a kind's slots are full, its cards
// are only upgrades of what is held, and the other kind still comes.
func TestSixActivesAndSixPassives(t *testing.T) {
	g := playing()
	var actives, passives []augID
	for a := augID(0); a < nAug; a++ {
		switch {
		case augs[a].act && len(actives) < slots:
			actives = append(actives, a)
		case !augs[a].act && augs[a].needs == a && augs[a].max > 1 && len(passives) < slots:
			passives = append(passives, a)
		}
	}
	for _, a := range append(actives, passives...) {
		pickN(g, a, 1)
	}
	if g.holds(true) != slots || g.holds(false) != slots {
		t.Fatalf("holds %d actives and %d passives", g.holds(true), g.holds(false))
	}
	for seed := uint64(1); seed <= 200; seed++ {
		g.rnd = rng(seed)
		g.pending = 1
		g.openCards()
		if g.nCards == 0 {
			t.Fatal("no cards with the slots full but upgrades left")
		}
		for i := 0; i < g.nCards; i++ {
			if a := g.cards[i]; g.rank[a] == 0 {
				t.Fatalf("seed %d: %s offered with its slots full", seed, augs[a].name)
			}
		}
		g.phase = phPlay
	}

	g = playing()
	for _, a := range actives {
		pickN(g, a, 1)
	}
	newPassive := false
	for seed := uint64(1); seed <= 50; seed++ {
		g.rnd = rng(seed)
		g.pending = 1
		g.openCards()
		for i := 0; i < g.nCards; i++ {
			if a := g.cards[i]; g.rank[a] == 0 && augs[a].act {
				t.Fatalf("a new active, %s, with six held", augs[a].name)
			}
			if !augs[g.cards[i]].act {
				newPassive = true
			}
		}
		g.phase = phPlay
	}
	if !newPassive {
		t.Fatal("full actives also stopped the passives")
	}
}

// Every active, alone, hurts what is near without the trigger pulled, by
// more than a few shots.
func TestEveryActiveActsByItself(t *testing.T) {
	for a := augID(0); a < nAug; a++ {
		if !augs[a].act {
			continue
		}
		g := playing()
		g.bossNext = 99
		pickN(g, a, 3)
		var foes []int
		for k := 0; k < 6; k++ {
			p := g.pos.add(dir(float64(k) * math.Pi / 3).scale(1.2 + 0.25*float64(k)))
			if !solid(p.x, p.y) {
				foes = append(foes, stillEnemy(g, kElder, p, 1e6, 0))
			}
		}
		for i := 0; i < 8*30; i++ {
			for _, f := range foes {
				g.enemies[f].knock = vec{}
			}
			g.ang += 0.05 // the boomerang goes where the player looks
			g.pos = vec{16, 16.5}
			g.step(tick)
			g.hp = g.st.maxHP
			for i := range g.enemies {
				if !slicesContains(foes, i) {
					g.enemies[i].alive = false // only the foes placed
				}
			}
		}
		hurt := 0.0
		for _, f := range foes {
			hurt += 1e6 - g.enemies[f].hp
		}
		// A core ability: at least fifteen of the gun's shots' worth.
		t.Logf("%-16s %6.0f (%.0f a shot)", augs[a].name, hurt, g.st.dmg)
		if hurt < 15*g.st.dmg {
			t.Errorf("%s did %.1f damage in 8 s, under 15 shots' worth", augs[a].name, hurt)
		}
	}
}

func slicesContains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Round Trip's boomerang comes back to the player and hits on the way
// home too.
func TestTheBoomerangComesBack(t *testing.T) {
	g := playing()
	j := g.shoot(g.pos, g.ang, 10, 1, shotRang)
	far := 0.0
	for i := 0; i < 60 && g.shots[j].alive; i++ {
		g.stepShots(tick)
		far = math.Max(far, math.Sqrt(g.shots[j].pos.dist2(g.pos)))
	}
	if g.shots[j].alive || far < 3 || !g.shots[j].back {
		t.Fatalf("alive %v after 2s, went %.1f, came back %v", g.shots[j].alive, far, g.shots[j].back)
	}
}

// A kill pays its experience at once: the level, and its cards, come the
// moment the enemy dies, not when the bytes arrive.
func TestAKillPaysAtOnce(t *testing.T) {
	g := playing()
	g.xp = need(1) - 0.1
	i := stillEnemy(g, kBoba, vec{16, 10}, 1, 0)
	g.damage(i, 10, false)
	if g.level != 2 || g.pending != 1 {
		t.Fatalf("level %d, %d cards waiting, right after the kill", g.level, g.pending)
	}
}

// The reload is a heap change: the spent heap slides out and falls, empty;
// a full one rises back into the socket; the gun ends where it began.
func TestTheReloadChangesTheHeap(t *testing.T) {
	g := playing()
	v := view{w: 140, h: 44, f: 90, horizon: 22}
	at := func(t float64) pose {
		g.reloadT = g.st.reload * (1 - t)
		return g.gunPose(&v, 70, 44, 90)
	}
	rest := at(0.999)
	g.reloadT = 0
	still := g.gunPose(&v, 70, 44, 90)
	if p := at(0.35); p.heapDrop < 0.15 || p.fill != 0 {
		t.Fatalf("a third in: heap down %.2f, glow %.1f; want fallen and empty", p.heapDrop, p.fill)
	}
	if a, b := at(0.5), at(0.65); !(b.heapDrop < a.heapDrop) || a.fill != 1 {
		t.Fatalf("the fresh heap does not rise: %.2f then %.2f, glow %.1f", a.heapDrop, b.heapDrop, a.fill)
	}
	if rest.heapDrop != 0 || math.Abs(rest.at.y-still.at.y) > 0.005 {
		t.Fatalf("the gun does not come back: heap %.2f, height %.3f vs %.3f", rest.heapDrop, rest.at.y, still.at.y)
	}
}

// The build is listed down the right edge when there is room: a heading
// for each kind, then each held with its ranks.
func TestTheBuildShowsInThePanel(t *testing.T) {
	g := playing()
	pickN(g, augTesla, 2)
	pickN(g, augHotPath, 1)
	b := buffer.NewBuffer(cell.Rect{Width: 160, Height: 46})
	g.render(b)
	lines := strings.Split(snapshot(b), "\n")
	find := func(s string) (int, int) {
		for y, l := range lines {
			if x := strings.Index(string([]rune(l)), s); x >= 0 {
				return len([]rune(l[:x])), y
			}
		}
		return -1, -1
	}
	for _, s := range []string{"ACTIVE 1/6", "PASSIVE 1/6", "Tesla Coil •••••", "Hot Path •••••"} {
		if x, y := find(s); x < 160-slotsW || y < slotsTop {
			t.Errorf("%q at %d,%d, want at the right", s, x, y)
		}
	}
	// Two ranks of five: the first two pips lit, the third not.
	x, y := find("Tesla Coil •••••")
	pip := func(k int) cell.Color { return b.Content[y*160+x+11+k].Style.Fg }
	if pip(1) == pip(2) || pip(0) != pip(1) {
		t.Errorf("Tesla Coil's two ranks of five are not shown")
	}
}

// -showfps counts the frames drawn and shows the count a second.
func TestTheFrameRateIsCounted(t *testing.T) {
	g := playing()
	start := g.fpsAt.Add(time.Hour)
	g.countFrame(start) // the first second starts
	for i := 1; i <= 144; i++ {
		g.countFrame(start.Add(time.Duration(i) * time.Second / 144))
	}
	if g.fps != 144 {
		t.Fatalf("144 frames in a second counted as %d fps", g.fps)
	}
}

// everyActive is the heaviest frame the actives make: all twelve at their
// last rank, burn and poison and chill on the hits, 150 enemies that do not
// die, the clouds, rings and lightning all out.
func everyActive() *game {
	g := playing()
	g.bossNext = 99
	for a := augID(0); a < nAug; a++ {
		if augs[a].act {
			pickN(g, a, augs[a].max)
		}
	}
	for _, a := range []augID{augIgnite, augChill, augLeak, augStatic, augChain, augREP} {
		pickN(g, a, augs[a].max)
	}
	g.exact = true
	g.press(actFire)
	fillArena(g, 150, true)
	for i := 0; i < 150; i++ {
		g.step(tick)
		g.ammo, g.hp = g.st.mag, 1e9
		g.phase, g.pending = phPlay, 0
	}
	return g
}

func BenchmarkEveryActive(b *testing.B) {
	g := everyActive()
	buf := buffer.NewBuffer(cell.Rect{Width: 160, Height: 46})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.step(tick)
		g.ammo, g.hp = g.st.mag, 1e9
		g.phase, g.pending = phPlay, 0
		g.render(buf)
	}
}

// The effects live in fixed pools: a frame with every active at work
// allocates nothing.
func TestEveryActiveAllocatesNothing(t *testing.T) {
	g := everyActive()
	b := buffer.NewBuffer(cell.Rect{Width: 160, Height: 46})
	g.render(b)
	n, rings, arcs := 0, 0, 0
	for _, p := range g.parts {
		if p.alive && p.puff > 0 {
			n++
		}
	}
	for _, r := range g.rings {
		if r.alive {
			rings++
		}
	}
	for _, a := range g.arcs {
		if a.alive {
			arcs++
		}
	}
	if n == 0 || rings == 0 {
		t.Fatalf("%d puffs, %d rings, %d arcs out: the scene shows no effects", n, rings, arcs)
	}
	if avg := testing.AllocsPerRun(30, func() {
		g.step(tick)
		g.ammo, g.hp = g.st.mag, 1e9
		g.phase, g.pending = phPlay, 0
		g.render(b)
	}); avg != 0 {
		t.Errorf("%.2f allocations per frame with every active at work, want 0", avg)
	}
}
