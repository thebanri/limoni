package main

import (
	"math"
	"testing"
)

// The bosses come in their order, one every ten levels, and round again,
// tougher.
func TestTheBossesComeInTurn(t *testing.T) {
	g := playing()
	var got []kindID
	var hp []float64
	for lv := 10; lv <= 60; lv += 10 {
		for i := range g.enemies {
			g.enemies[i].alive = false
		}
		g.bossNext = lv
		g.spawnBoss()
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && kinds[e.k].boss {
				got, hp = append(got, e.k), append(hp, e.max/g.hpScale())
			}
		}
	}
	want := []kindID{kBobaPrime, kFerris, kQueen, kReconciler, kTextual, kBobaPrime}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("bosses %v, want %v", got, want)
		}
	}
	if hp[5] <= hp[0]*1.5 {
		t.Fatalf("Boba Prime the second time has %.0f health, the first %.0f", hp[5], hp[0])
	}
}

// By level ten most kinds have come in, each new one more often until it
// has, and the first of each is announced.
func TestNewKindsAreMetEarly(t *testing.T) {
	g := playing()
	g.level = 10
	open := 0
	for k := kindID(0); k < nKinds; k++ {
		if g.kindWeight(k) > 0 {
			open++
		}
	}
	if open < 9 {
		t.Fatalf("%d kinds come by level 10", open)
	}
	if g.kindWeight(kBomb) != 4*kinds[kBomb].weight {
		t.Fatal("a kind not yet met does not come more often")
	}
	for i := 0; i < 2000 && !g.seen[kCursor]; i++ {
		g.spawnAcc = 40
		g.spawn(0)
	}
	if !g.seen[kCursor] || g.kindWeight(kCursor) != kinds[kCursor].weight {
		t.Fatal("termbox never came, or still comes as if new")
	}
}

// A notcurses bomb lights its fuse beside the player, marks its blast on
// the floor, and goes off with it; a dash gets out of it.
func TestTheBombGoesOff(t *testing.T) {
	g := playing()
	g.hp = g.st.maxHP
	i := stillEnemy(g, kBomb, g.pos.add(vec{1.2, 0}), 1e6, 0)
	g.stepEnemies(tick)
	if g.enemies[i].windT <= 0 || !g.hazards[0].alive {
		t.Fatal("no fuse, or no mark on the floor")
	}
	hp := g.hp
	for k := 0; k < 40; k++ {
		g.stepEnemies(tick)
		g.stepHazards(tick)
	}
	if g.enemies[i].alive || g.hp >= hp || g.hurtBy != kBomb {
		t.Fatalf("after the fuse: bomb alive %v, health %.0f → %.0f", g.enemies[i].alive, hp, g.hp)
	}
	// Killed before its fuse: it goes off at once, and hurts its own.
	g = playing()
	b := stillEnemy(g, kBomb, vec{16, 10}, 5, 0)
	o := stillEnemy(g, kElder, vec{16.8, 10}, 1e6, 0)
	g.damage(b, 100, false)
	g.stepHazards(tick)
	if g.enemies[o].hp >= 1e6 {
		t.Fatal("a bomb shot dead did not go off")
	}
}

// crossterm stops, shows its line, then charges along it; prompt_toolkit
// aims a laser and fires hard; termbox blinks nearer.
func TestTheNewMinionsDoWhatTheySay(t *testing.T) {
	g := playing()
	g.hp = 1e6
	c := stillEnemy(g, kBeetle, inSight(t, g, 5), 1e6, 0)
	g.enemies[c].cd = 0
	g.stepEnemies(tick)
	if g.enemies[c].windT <= 0 {
		t.Fatal("crossterm did not wind up")
	}
	at := g.enemies[c].pos
	for g.enemies[c].windT > 0 {
		g.stepEnemies(tick)
	}
	if g.enemies[c].pos.dist2(at) > 0.01 {
		t.Fatal("crossterm moved while showing its line")
	}
	for k := 0; k < 9; k++ {
		g.stepEnemies(tick)
	}
	if moved := math.Sqrt(g.enemies[c].pos.dist2(at)); moved < 2 {
		t.Fatalf("its charge took it %.1f in 0.3 s", moved)
	}

	g = playing()
	s := stillEnemy(g, kCaret, inSight(t, g, 8), 1e6, 0)
	g.enemies[s].cd = 0
	g.stepEnemies(tick)
	if g.enemies[s].windT <= 0 {
		t.Fatal("prompt_toolkit did not take aim")
	}
	g.stepEnemies(tick) // the laser is drawn from the frame after
	lasers := 0
	for _, a := range g.arcs {
		if a.alive && a.straight {
			lasers++
		}
	}
	for g.enemies[s].windT > 0 {
		g.stepEnemies(tick)
	}
	fast := false
	for i := 0; i < g.shotTop; i++ {
		if sh := &g.shots[i]; sh.alive && sh.src == kCaret && sh.vel.len() > enemySpeed*2 {
			fast = true
		}
	}
	if lasers == 0 || !fast {
		t.Fatalf("laser %d, fast shot %v", lasers, fast)
	}

	g = playing()
	b := stillEnemy(g, kCursor, g.pos.add(vec{0, -6}), 1e6, 0)
	g.enemies[b].cd = 0
	before := math.Sqrt(g.enemies[b].pos.dist2(g.pos))
	for k := 0; k < 15; k++ {
		g.stepEnemies(tick)
	}
	if after := math.Sqrt(g.enemies[b].pos.dist2(g.pos)); before-after < 2 {
		t.Fatalf("termbox from %.1f to %.1f in half a second: no blink", before, after)
	}
}

// Lip Gloss shields those near it: they take half.
func TestLipGlossShields(t *testing.T) {
	g := playing()
	l := stillEnemy(g, kGloss, vec{16, 10}, 1e6, 0)
	o := stillEnemy(g, kElder, vec{17, 10}, 1e6, 0)
	g.enemies[l].cd = 0
	g.stepEnemies(tick)
	if g.enemies[o].shieldT <= 0 {
		t.Fatal("not shielded")
	}
	g.damage(o, 100, false)
	if lost := 1e6 - g.enemies[o].hp; math.Abs(lost-50) > 0.5 {
		t.Fatalf("a shielded foe took %.1f of 100", lost)
	}
}

// Textual's cascade marks the floor round the player and hurts whoever is
// still there; Ink's re-render lands the Reconciler on its mark.
func TestTheNewBossesMarkTheirBlows(t *testing.T) {
	g := playing()
	g.hp = 1e6
	x := stillEnemy(g, kTextual, g.pos.add(vec{0, -6}), 1e9, 0)
	g.enemies[x].attacks = 0
	g.enemyFire(x, &g.enemies[x], vec{0, 1}, 6)
	marks := 0
	for _, h := range g.hazards {
		if h.alive && h.src == kTextual {
			marks++
		}
	}
	hp := g.hp
	for k := 0; k < 60; k++ {
		g.stepHazards(tick)
	}
	if marks != 5 || g.hp >= hp || g.hurtHow != "css cascade" {
		t.Fatalf("%d marks; health %.0f → %.0f by %q", marks, hp, g.hp, g.hurtHow)
	}

	g = playing()
	r := stillEnemy(g, kReconciler, g.pos.add(vec{0, -8}), 1e9, 0)
	g.enemies[r].attacks = 1
	g.enemies[r].slam = slams(&g.enemies[r])
	g.enemyFire(r, &g.enemies[r], vec{0, 1}, 8)
	var mark vec
	for _, h := range g.hazards {
		if h.alive && h.land {
			mark = h.pos
		}
	}
	for k := 0; k < 40; k++ {
		g.stepHazards(tick)
	}
	if mark == (vec{}) || g.enemies[r].pos != mark {
		t.Fatalf("the Reconciler is at %v, its mark was %v", g.enemies[r].pos, mark)
	}
}

// inSight is a free spot about d from the player with nothing between.
func inSight(t *testing.T, g *game, d float64) vec {
	t.Helper()
	for a := 0.0; a < 2*math.Pi; a += 0.1 {
		p := g.pos.add(dir(a - math.Pi/2).scale(d))
		if free(p.x, p.y, 0.4) && g.sees(p) {
			return p
		}
	}
	t.Fatal("no spot in sight")
	return vec{}
}
