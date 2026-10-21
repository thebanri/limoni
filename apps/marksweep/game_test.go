package main

import (
	"math"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

const tick = 1.0 / 30

// Every glyph the game draws must be one column wide, or a row shifts.
func TestEveryGlyphIsOneColumn(t *testing.T) {
	check := func(where string, r rune) {
		if r != ' ' && cell.RuneWidth(r) != 1 {
			t.Errorf("%s: %q is %d columns", where, r, cell.RuneWidth(r))
		}
	}
	for k := range kinds {
		for _, a := range kinds[k].art {
			for _, r := range a.glyph {
				check(kinds[k].name, r)
			}
		}
	}
	for _, r := range gunArt.glyph {
		check("gun", r)
	}
	for tile, tex := range wallTextures {
		for y := range tex.glyph {
			for _, r := range tex.glyph[y] {
				check("wall "+string(tile), r)
			}
		}
	}
	for _, r := range ramp {
		check("ramp", r)
	}
	for _, r := range "•∙*▮o●◆01·~✸─│┼◀▶▀▮▯○●◉▏▎▍▌▋▊▉█" {
		check("hud", r)
	}
}

func TestReloadIsManual(t *testing.T) {
	g := playing()
	g.ammo = 0
	g.press(actFire)
	g.step(tick)
	if g.ammo != 0 || g.dryT <= 0 {
		t.Fatalf("an empty heap fired or said nothing: ammo %d, dry %.2f", g.ammo, g.dryT)
	}
	for i := 0; i < 60; i++ {
		g.step(tick)
	}
	if g.ammo != 0 {
		t.Fatal("the heap refilled by itself; collecting is the player's job")
	}
	g.release(actFire)
	g.reload()
	if g.reloadT <= 0 {
		t.Fatal("R did not start the collector")
	}
	for g.reloadT > 0 {
		g.step(tick)
	}
	if g.ammo != g.st.mag {
		t.Fatalf("after collecting the heap holds %d, want %d", g.ammo, g.st.mag)
	}
}

// In a terminal that never reports a release, a held mouse button fires
// until the release the mouse does report; F, a key, stops on its own.
func TestTheTriggerLetsGoInTerminalsWithoutReleases(t *testing.T) {
	g := playing()
	g.press(actFire) // the mouse
	for i := 0; i < 60; i++ {
		g.step(tick)
	}
	if !g.held[actFire] {
		t.Fatal("a held mouse button let go by itself")
	}
	g.release(actFire)
	g.fireKey = true // F
	g.press(actFire)
	for i := 0; i < 60; i++ {
		g.step(tick)
	}
	if g.held[actFire] {
		t.Fatal("F kept firing two seconds after its last press, in a terminal without releases")
	}
}

func TestFiringSpendsRoundsAtTheRate(t *testing.T) {
	g := playing()
	g.exact = true
	g.press(actFire)
	start := g.ammo
	for i := 0; i < 30; i++ { // one second
		g.step(tick)
	}
	spent := start - g.ammo
	if want := int(g.st.fireRate); spent < want-1 || spent > want+1 {
		t.Fatalf("spent %d rounds in a second at %.1f a second", spent, g.st.fireRate)
	}
}

func TestCardsAreDifferentAndFollowTheirNeeds(t *testing.T) {
	for seed := uint64(1); seed < 300; seed++ {
		g := newGame(seed)
		g.reset()
		g.pending = 1
		g.openCards()
		if g.nCards != 3 {
			t.Fatalf("seed %d: %d cards", seed, g.nCards)
		}
		if g.cards[0] == g.cards[1] || g.cards[1] == g.cards[2] || g.cards[0] == g.cards[2] {
			t.Fatalf("seed %d: the same card twice: %v", seed, g.cards)
		}
		for _, a := range g.cards {
			if n := augs[a].needs; n != a && g.rank[n] == 0 {
				t.Fatalf("seed %d: %s offered without %s", seed, augs[a].name, augs[n].name)
			}
		}
	}
}

func TestPastTheLastRankAugmentsOverclock(t *testing.T) {
	g := playing()
	pickN(g, augHotPath, augs[augHotPath].max)
	if g.rank[augHotPath] != augs[augHotPath].max || g.over[augHotPath] != 0 {
		t.Fatalf("rank %d over %d", g.rank[augHotPath], g.over[augHotPath])
	}
	before := g.st.dmg
	pickN(g, augHotPath, 1)
	if g.over[augHotPath] != 1 || g.rank[augHotPath] != augs[augHotPath].max {
		t.Fatalf("the pick past the last rank did not overclock: rank %d over %d", g.rank[augHotPath], g.over[augHotPath])
	}
	if got := g.st.dmg / before; math.Abs(got-1.12) > 1e-9 {
		t.Fatalf("an overclock multiplied damage by %.3f, want 1.12", got)
	}
	// It keeps going, without end.
	pickN(g, augHotPath, 20)
	if g.over[augHotPath] != 21 {
		t.Fatalf("over %d after 21 overclocks", g.over[augHotPath])
	}
}

// pickN takes augment a n times, as if each were a card chosen.
func pickN(g *game, a augID, n int) {
	for i := 0; i < n; i++ {
		g.phase = phCards
		g.cards[0], g.nCards = a, 1
		g.take(0)
	}
}

func TestPressureHoldsThenRunsAway(t *testing.T) {
	g := playing()
	for _, s := range []float64{0, 60, 300, 599} {
		g.runTime = s
		if p := g.pressure(); p != 1 {
			t.Fatalf("pressure %.2f at %.0f s, want 1 before ten minutes", p, s)
		}
	}
	prev := 1.0
	for s := 600.0; s <= 1500; s += 30 {
		g.runTime = s
		p := g.pressure()
		if p < prev {
			t.Fatalf("pressure fell from %.2f to %.2f at %.0f s", prev, p, s)
		}
		prev = p
	}
	g.runTime = 900
	if p := g.pressure(); p < 3.5 {
		t.Fatalf("pressure %.2f at 15 minutes: too gentle to end a broken build", p)
	}
}

// brokenBuild has every multiplying augment at its last rank.
func brokenBuild(g *game) {
	for _, a := range []augID{augREP, augScroll, augDiff, augFork, augPanic, augChain, augLoop, augEscape} {
		pickN(g, a, augs[a].max)
	}
	for _, a := range []augID{augRecursion, augCascade, augOverflow, augKill9} {
		pickN(g, a, augs[a].max)
	}
	pickN(g, augSync, 1)
}

// A build as broken as the cards allow fills the pools and stays inside
// them: no panic, no shot past the pool, and a frame still allocates nothing.
func TestTheMostBrokenBuildStaysInsideThePools(t *testing.T) {
	g := playing()
	brokenBuild(g)
	g.ammo = g.st.mag
	g.exact = true
	g.press(actFire)
	g.hp = 1e9
	fillArena(g, 200, true)
	for i := 0; i < 300; i++ {
		g.step(tick)
		g.ammo = g.st.mag
		g.hp = 1e9
		if g.nShots > maxShots {
			t.Fatalf("%d shots alive in a pool of %d", g.nShots, maxShots)
		}
	}
	if g.peakShot != maxShots {
		t.Errorf("the broken build peaked at %d shots; against foes that do not die it should fill all %d", g.peakShot, maxShots)
	}
	b := buffer.NewBuffer(cell.Rect{Width: 160, Height: 46})
	g.render(b)
	if avg := testing.AllocsPerRun(30, func() { g.step(tick); g.ammo = g.st.mag; g.hp = 1e9; g.render(b) }); avg != 0 {
		t.Errorf("%.2f allocations per frame with the pools full, want 0", avg)
	}
}

// fillArena puts n enemies of every ordinary kind in rows; tough ones do
// not die, so a build's shots keep finding something to split on.
func fillArena(g *game, n int, tough bool) {
	for i := 0; i < n; i++ {
		j := g.addEnemy(kindID(i%int(kBobaPrime)), vec{4 + float64(i%24), 2 + float64(i/24)*0.6})
		if tough && j >= 0 {
			g.enemies[j].hp, g.enemies[j].max = 1e12, 1e12
		}
	}
}

// BenchmarkFullPools is the worst frame: the most broken build, its pool of
// shots full, 200 enemies in the arena.
func BenchmarkFullPools(b *testing.B) {
	g := playing()
	brokenBuild(g)
	g.exact = true
	g.press(actFire)
	fillArena(g, 200, true)
	for i := 0; i < 200; i++ {
		g.step(tick)
		g.ammo, g.hp = g.st.mag, 1e9
	}
	buf := buffer.NewBuffer(cell.Rect{Width: 160, Height: 46})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.step(tick)
		g.ammo, g.hp = g.st.mag, 1e9
		g.render(buf)
	}
	b.ReportMetric(float64(g.nShots), "shots")
}

func TestAFrameAllocatesNothing(t *testing.T) {
	for _, ph := range []phase{phPlay, phCards, phPause, phDead, phTitle} {
		g := playing()
		for i := 0; i < 40; i++ {
			g.addEnemy(kindID(i%int(kBobaPrime)), vec{6 + float64(i%20), 4 + float64(i/20)})
		}
		g.press(actFire)
		for i := 0; i < 60; i++ {
			g.step(tick)
		}
		g.phase = ph
		if ph == phCards {
			g.pending = 1
			g.phase = phPlay
			g.openCards()
		}
		b := buffer.NewBuffer(cell.Rect{Width: 140, Height: 44})
		g.render(b)
		frame := func() {
			g.step(tick)
			g.render(b)
		}
		if avg := testing.AllocsPerRun(50, frame); avg != 0 {
			t.Errorf("phase %d: %.2f allocations per frame, want 0", ph, avg)
		}
	}
}

func TestEveryWindowSizeDraws(t *testing.T) {
	g := playing()
	for i := 0; i < 30; i++ {
		g.addEnemy(kindID(i%int(nKinds)), vec{6 + float64(i%20), 4 + float64(i/20)})
	}
	for _, sz := range [][2]uint16{{10, 4}, {20, 8}, {40, 12}, {80, 24}, {140, 44}, {300, 90}, {600, 40}} {
		for _, ph := range []phase{phTitle, phPlay, phCards, phPause, phDead} {
			g.phase = ph
			if ph == phCards {
				g.nCards = 3
			}
			b := buffer.NewBuffer(cell.Rect{Width: sz[0], Height: sz[1]})
			g.render(b) // must not panic
		}
	}
}

func TestEdgeTurn(t *testing.T) {
	const w = 120
	if edgeTurn(w/2, w) != 0 {
		t.Fatal("the middle turns")
	}
	if edgeTurn(0, w) > -0.9 || edgeTurn(w-1, w) < 0.9 {
		t.Fatalf("the edges turn %.2f and %.2f, want close to -1 and 1", edgeTurn(0, w), edgeTurn(w-1, w))
	}
	if edgeTurn(w/6+1, w) != 0 || edgeTurn(w-w/6-2, w) != 0 {
		t.Fatal("outside the outer sixth the view turns")
	}
}

// A shot leaves through the crosshair, wherever the mouse put it.
func TestShotsLeaveThroughTheCrosshair(t *testing.T) {
	g := playing()
	g.viewW = 120
	g.mouseIn = true
	g.mouseX = 119
	a := g.aim()
	want := g.ang + math.Atan(math.Tan(fov/2)*(2*119.5/120-1))
	if math.Abs(a-want) > 1e-9 {
		t.Fatalf("aim %.4f, want %.4f", a, want)
	}
	g.locked = true
	if g.aim() != g.ang {
		t.Fatal("with the pointer locked, shots must go straight ahead")
	}
}

func TestRecordsRoundTrip(t *testing.T) {
	r := records{runs: 3, secs: 754, level: 31, kills: 2041, peak: 612, bigHit: 1880, volume: 4}
	got := decodeRecords(encodeRecords(r))
	if got != r {
		t.Fatalf("round trip: %+v, want %+v", got, r)
	}
	if bad := decodeRecords([]byte("{nope")); bad.runs != 0 || bad.volume != -1 {
		t.Fatalf("garbage decoded as %+v", bad)
	}
	var best records
	if !best.add(runRecord{secs: 100, level: 5}) || best.add(runRecord{secs: 50, level: 9}) {
		t.Fatal("add reports the longest run wrong")
	}
	if best.secs != 100 || best.level != 9 || best.runs != 2 {
		t.Fatalf("records kept %+v", best)
	}
}

func TestSplitsAndShrapnelEnd(t *testing.T) {
	// fork() and Cascading Diff make shots from shots; generations keep it
	// finite even when nothing else would.
	g := playing()
	brokenBuild(g)
	g.addEnemy(kElder, vec{16, 12})
	g.enemies[0].hp, g.enemies[0].max = 1e12, 1e12
	g.shoot(vec{16, 15}, -math.Pi/2, g.st.dmg, 0, shotPlayer)
	for i := 0; i < 400; i++ {
		g.step(tick)
		g.hp = 1e9
		g.ammo = g.st.mag
	}
	if g.nShots != 0 {
		t.Fatalf("%d shots still alive after 13 seconds from a single trigger pull", g.nShots)
	}
}

func TestLevelUpsQueueCards(t *testing.T) {
	g := playing()
	g.gain(need(1) + need(2) + 0.5)
	if g.level != 3 || g.pending != 2 {
		t.Fatalf("level %d, %d cards pending; want 3 and 2", g.level, g.pending)
	}
	g.step(tick)
	if g.phase != phCards || g.pending != 1 {
		t.Fatalf("phase %d pending %d after the step", g.phase, g.pending)
	}
	g.take(0)
	if g.phase != phCards {
		t.Fatal("the second card did not follow the first")
	}
	g.take(1)
	if g.phase != phPlay || g.pending != 0 {
		t.Fatalf("phase %d pending %d after both cards", g.phase, g.pending)
	}
}

func TestABossComesEveryTenLevels(t *testing.T) {
	g := playing()
	g.level = 10
	pickN(g, augHotPath, 1)
	boss := -1
	for i := range g.enemies {
		if g.enemies[i].alive && kinds[g.enemies[i].k].boss {
			boss = i
		}
	}
	if boss < 0 || !g.bossLive || g.bossNext != 20 {
		t.Fatalf("no boss at level 10 (live %v, next %d)", g.bossLive, g.bossNext)
	}
	g.damage(boss, 1e12, false)
	if g.bossLive {
		t.Fatal("the boss died but is still live")
	}
	g.level = 20
	pickN(g, augHotPath, 1)
	for i := range g.enemies {
		if g.enemies[i].alive && kinds[g.enemies[i].k].boss && g.enemies[i].k != kFerris {
			t.Fatalf("the second boss is %s, want Ferris", kinds[g.enemies[i].k].name)
		}
	}
}

func TestArmourBluntsSmallHits(t *testing.T) {
	g := playing()
	i := g.addEnemy(kElder, vec{16, 10})
	e := &g.enemies[i]
	e.hp, e.max, e.armor = 1000, 1000, 10
	g.damage(i, 4, false) // shrapnel-sized
	if lost := 1000 - e.hp; lost > 0.25 {
		t.Fatalf("a 4-damage hit took %.2f off 10 armour, want only the 5%% floor", lost)
	}
	g.damage(i, 100, false)
	if lost := 1000 - e.hp; lost < 90 {
		t.Fatalf("a 100-damage hit took only %.2f", lost)
	}
}
