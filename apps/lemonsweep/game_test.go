package main

import (
	"math"
	"testing"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
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
		for _, r := range kinds[k].bits {
			check(kinds[k].name, r)
		}
	}
	for _, r := range "•∙*▮o●◆◇01·~─│◀▶▀▄▮▯○●◉▏▎▍▌▋▊▉█╲╱«»^°▲◥◢▼◣◤!o" {
		check("hud", r)
	}
}

func TestEmptyHeapReloadsItself(t *testing.T) {
	g := playing()
	g.ammo = 1
	g.press(actFire)
	g.step(tick)
	if g.ammo != 0 || g.reloadT <= 0 {
		t.Fatalf("the last round left ammo %d, collector %.2f; want 0 and collecting", g.ammo, g.reloadT)
	}
	for g.reloadT > 0 {
		g.step(tick)
	}
	if g.ammo != g.st.mag {
		t.Fatalf("after collecting the heap holds %d, want %d", g.ammo, g.st.mag)
	}
	g.release(actFire)
	g.ammo = 0
	g.press(actFire) // an empty heap from elsewhere: the trigger collects
	g.step(tick)
	if g.reloadT <= 0 {
		t.Fatal("the trigger on an empty heap did not start the collector")
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

// A card at its last rank is not offered again; with every slot full and
// every card held at its last rank, a level is a random stat, at once.
func TestMaxedBuildsTakeRandomStats(t *testing.T) {
	g := playing()
	pickN(g, augHotPath, augs[augHotPath].max)
	if g.offerable(augHotPath) {
		t.Fatal("a card at its last rank is offered again")
	}
	held := 0
	for a := augID(0); a < nAug && held < 2*slots; a++ {
		if g.offerable(a) {
			pickN(g, a, augs[a].max)
			held++
		}
	}
	for a := augID(0); a < nAug; a++ {
		if g.offerable(a) {
			t.Fatalf("%s is still offered to a full, maxed build", augs[a].name)
		}
	}
	dmg, hp := g.st.dmg, g.st.maxHP
	g.pending = 1
	g.openCards()
	if g.phase != phPlay || g.pending != 0 {
		t.Fatalf("phase %v, %d pending: want the run going on", g.phase, g.pending)
	}
	n := 0
	for _, b := range g.bonus {
		n += b
	}
	if n != 1 || g.st.dmg == dmg && g.st.maxHP == hp && (g.bonus[bonDamage] > 0 || g.bonus[bonHealth] > 0) {
		t.Fatalf("%d random stats taken, or not applied: %v", n, g.bonus)
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
	for _, s := range []float64{0, 60, 300, 599, pressureAt} {
		g.stageT = s
		if p := g.pressure(); p != 1 {
			t.Fatalf("pressure %.2f at %.0f s, want 1 before it starts", p, s)
		}
	}
	prev := 1.0
	for s := pressureAt; s <= 1500; s += 30 {
		g.stageT = s
		p := g.pressure()
		if p < prev {
			t.Fatalf("pressure fell from %.2f to %.2f at %.0f s", prev, p, s)
		}
		prev = p
	}
	g.stageT = pressureAt + 300
	if p := g.pressure(); p < 3.5 {
		t.Fatalf("pressure %.2f five minutes on: too gentle to end a broken build", p)
	}
	// /dev/zero's comes later and climbs slower, but climbs.
	g.endless, g.stageT = true, endlessAt-1
	if p := g.pressure(); p != 1 {
		t.Fatalf("endless pressure %.2f at %.0f s, want none yet", p, g.stageT)
	}
	g.stageT = endlessAt + 300
	if p := g.pressure(); p < 1.2 || p > 2 {
		t.Fatalf("endless pressure %.2f five minutes on, want gentler", p)
	}
	g.stageT = endlessAt + 900
	if p := g.pressure(); p < 3.5 {
		t.Fatalf("endless pressure %.2f fifteen minutes on: it must still end a run", p)
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
	// Shots die as fast as they split once the pool is full, so the count
	// at the end of a step sits just under it.
	if g.peakShot < maxShots*95/100 {
		t.Errorf("the broken build peaked at %d shots; against foes that do not die it should fill the %d", g.peakShot, maxShots)
	}
	b := buffer.NewBuffer(cell.Rect{Width: 160, Height: 46})
	g.render(b)
	if avg := testing.AllocsPerRun(30, func() { g.step(tick); g.ammo = g.st.mag; g.hp = 1e9; g.render(b) }); avg != 0 {
		t.Errorf("%.2f allocations per frame with the pools full, want 0", avg)
	}
}

// fillArena puts n enemies of every ordinary kind in rows; tough ones do
// not die, so a build's shots keep finding something to split on. Bombs
// are left out: they go off, and go with it, however tough.
func fillArena(g *game, n int, tough bool) {
	var ordinary []kindID
	for k := kindID(0); k < kBobaPrime; k++ {
		if kinds[k].act != bBomb {
			ordinary = append(ordinary, k)
		}
	}
	for i := 0; i < n; i++ {
		j := g.addEnemy(ordinary[i%len(ordinary)], vec{4 + float64(i%24), 2 + float64(i/24)*0.6})
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

// The mouse turns the view by how far it moves, as in any first-person
// game, and shots still leave through the middle.
func TestTheMouseLooks(t *testing.T) {
	g := playing()
	g.viewW, g.viewH = 140, 44
	g.mouse(limoni.MouseEvent{X: 70, Y: 20})
	before := g.ang
	g.mouse(limoni.MouseEvent{X: 84, Y: 26}) // 14 columns right
	for i := 0; i < 30; i++ {
		g.step(tick)
	}
	want := before + 14*lookPerCell(140)
	if math.Abs(g.ang-want) > 1e-6 {
		t.Fatalf("turned to %.4f, want %.4f", g.ang, want)
	}
	if g.aim() != g.ang {
		t.Fatal("shots must leave through the middle")
	}
	if x, y := g.crosshair(); x != 70 || y != 22 {
		t.Fatalf("crosshair at %d,%d, want the middle", x, y)
	}
	// Outside a run the mouse does not turn the view.
	g.phase = phPause
	g.mouse(limoni.MouseEvent{X: 10, Y: 20})
	g.phase = phPlay
	g.step(tick)
	if math.Abs(g.ang-want) > 1e-6 {
		t.Fatal("moving the mouse on the pause screen turned the view")
	}
	// The pointer leaving the window on one side and coming back on the
	// other is not a half turn.
	g.mouse(limoni.MouseEvent{X: 130, Y: 20})
	for i := 0; i < 30; i++ {
		g.step(tick)
	}
	if math.Abs(g.ang-want) > 1e-6 {
		t.Fatalf("the pointer coming back in turned the view by %.2f", g.ang-want)
	}
}

// A grid line on the floor fades as its pixels grow past it, rather than
// breaking into dots: full where a pixel is narrower than the line, faint
// where it is far wider, and nothing between lines.
func TestSeamsFadeWithDistance(t *testing.T) {
	if seam(3, 0.01, 0.03) != 1 {
		t.Fatal("a near seam is not drawn in full")
	}
	if k := seam(3, 1, 0.03); k <= 0 || k > 0.03 {
		t.Fatalf("a far seam covers %.3f of its pixel, want a faint trace", k)
	}
	if seam(3.5, 0.01, 0.03) != 0 {
		t.Fatal("a seam drawn between lines")
	}
}

// At the window's edge, where a terminal's pointer can go no further, the
// view keeps turning; anywhere else it does not.
func TestEdgeTurn(t *testing.T) {
	const w = 120
	if edgeTurn(0, w) != -1 || edgeTurn(w-1, w) != 1 {
		t.Fatal("the edges do not turn")
	}
	for x := 2; x < w-2; x++ {
		if edgeTurn(x, w) != 0 {
			t.Fatalf("column %d turns", x)
		}
	}
}

// The gun's barrel, on the screen, runs through the crosshair.
func TestTheGunPointsAtTheCrosshair(t *testing.T) {
	g := playing()
	g.viewW, g.viewH = 140, 44
	v := view{w: 140, h: 44, horizon: 22, f: 107}
	p := g.gunPose(&v, 70, 44, 105)
	ax, ay, _ := p.screen(v3{0, 0.004, 0.1}, 70, 44, 105)
	bx, by, _ := p.screen(muzzleTip, 70, 44, 105)
	cx, cy := 70.5, 45.0
	d := math.Abs((bx-ax)*(cy-ay)-(by-ay)*(cx-ax)) / math.Hypot(bx-ax, by-ay)
	if d > 1 {
		t.Errorf("the barrel misses the crosshair by %.1f pixels", d)
	}
}

// The gun is drawn into a layer kept between frames: a frame must show
// nothing of the one before it — a tracer once stayed on the wall.
func TestTheGunLeavesNothingBehind(t *testing.T) {
	// Window heights move the gun's box between even and odd half rows.
	for h := uint16(30); h <= 46; h++ {
		scene := func(recoil float64) *game {
			g := playing()
			g.recoil = recoil
			return g
		}
		want := buffer.NewBuffer(cell.Rect{Width: 140, Height: h})
		scene(0).render(want)
		got := buffer.NewBuffer(cell.Rect{Width: 140, Height: h})
		scene(0.95).render(got)
		scene(0).render(got)
		for i := range want.Content {
			if got.Content[i] != want.Content[i] {
				t.Fatalf("height %d: cell %d,%d differs after a frame with a shot in it", h, i%140, i/140)
			}
		}
	}
}

// Every character on the walls has a glyph in the pixel font.
func TestWallTextIsInTheFont(t *testing.T) {
	for _, w := range windows {
		for _, line := range append([]string{w.title}, w.lines...) {
			for _, r := range line {
				if r == ' ' {
					continue
				}
				on := false
				for y := 0; y < fontH; y++ {
					for x := 0; x < fontW; x++ {
						on = on || fontPixel(r, x, y)
					}
				}
				if !on {
					t.Errorf("%s: %q has no glyph", w.title, r)
				}
			}
		}
	}
}

func TestRecordsRoundTrip(t *testing.T) {
	r := records{bests: bests{runs: 3, secs: 754, level: 31, kills: 2041, peak: 612, bigHit: 1880}, endless: bests{runs: 2, secs: 1500, level: 50, kills: 9000, peak: 700, bigHit: 3000}, volume: 4,
		maps: [nStages]mapBest{{runs: 3, swept: 2, fastest: 41, kills: 820}, {runs: 2, swept: 1, fastest: 77, kills: 1100}}}
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

// kitty sends these for w, 1 and the left button; feed parses them the way
// the terminal's input is parsed and hands them to the game.
func feed(g *game, s string) {
	b := []byte(s)
	for len(b) > 0 {
		ev, n := driver.ParseEvent(b)
		if n == 0 {
			return
		}
		b = b[n:]
		e := limoni.Event(ev)
		g.event(&e)
	}
}

// The keys held after a card are the keys held on the keyboard: one let go
// while the cards were up is let go (it once stayed held, walking on its
// own), and one still held when the pick was made on 1 is still held, though
// the terminal sends no more repeats of it once 1 was pressed.
func TestTheKeysAfterACardAreTheKeysHeld(t *testing.T) {
	const wDown, wUp, oneDown, oneUp = "w", "\x1b[119;1:3u", "1", "\x1b[49;1:3u"

	g := playing()
	feed(g, wDown)
	g.pending = 1
	g.step(tick)
	if g.phase != phCards {
		t.Fatal("no cards")
	}
	feed(g, wUp+oneDown+oneUp)
	g.step(tick)
	if g.phase != phPlay || g.held[actFwd] {
		t.Fatal("w let go during the cards is still held after the pick")
	}

	g = playing()
	feed(g, wDown+wUp+wDown) // the release tells the game this terminal reports them
	g.pending = 1
	g.step(tick)
	feed(g, oneDown+oneUp)
	for i := 0; i < 30; i++ {
		g.step(tick)
	}
	if !g.held[actFwd] {
		t.Fatal("w still held on the keyboard was let go by the pick")
	}

	g = playing()
	feed(g, "\x1b[<0;70;20M") // the button down: firing
	g.pending = 1
	g.step(tick)
	feed(g, "\x1b[<0;70;20m")
	if g.take(0); g.held[actFire] {
		t.Fatal("still firing after the cards")
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

// A trigger held down as the cards come up picks nothing: the button must
// come up first, and the cards stay a moment before a click counts.
func TestHeldTriggerDoesNotPickACard(t *testing.T) {
	g := playing()
	g.pointerButton(0, true) // firing
	g.pending = 1
	g.step(tick)
	if g.phase != phCards {
		t.Fatal("no cards")
	}
	g.cardHov = 0
	g.pointerButton(0, true) // a second press of the burst, still firing
	if g.phase != phCards {
		t.Fatal("the held trigger took a card")
	}
	g.pointerButton(0, false)
	g.pointerButton(0, true) // a click at once: still the burst
	if g.phase != phCards {
		t.Fatal("a click the moment the cards came took one")
	}
	g.pointerButton(0, false)
	for i := 0; i < 15; i++ {
		g.step(tick)
	}
	g.pointerButton(0, true)
	if g.phase != phPlay {
		t.Fatal("a click after the cards had settled took nothing")
	}

	g = playing() // not firing: a click takes after the moment, no release needed
	g.pending = 1
	g.step(tick)
	g.cardHov = 1
	for i := 0; i < 15; i++ {
		g.step(tick)
	}
	g.pointerButton(0, true)
	if g.phase != phPlay {
		t.Fatal("a fresh click took nothing")
	}
}
