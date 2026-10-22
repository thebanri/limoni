package main

import (
	"math"
	"testing"
)

// stillEnemy puts an enemy of kind k at p that keeps its health in the
// test's hands: no portal, no spawning around it.
func stillEnemy(g *game, k kindID, p vec, hp, armor float64) int {
	i := g.addEnemy(k, p)
	e := &g.enemies[i]
	e.hp, e.max, e.armor = hp, hp, armor
	return i
}

func run(g *game, secs float64, each func()) {
	for t := 0.0; t < secs; t += tick {
		g.stepEnemies(tick)
		if each != nil {
			each()
		}
	}
}

// Burn and poison hurt every half second; poison goes through armour that
// all but stops the burn.
func TestBurnAndPoisonHurtOverTime(t *testing.T) {
	g := playing()
	b := stillEnemy(g, kElder, vec{6, 4}, 1e6, 1000)
	p := stillEnemy(g, kElder, vec{26, 4}, 1e6, 1000)
	g.addBurn(b, 10)
	g.addPoison(p, 10)
	run(g, 1.05, func() { g.enemies[b].pos, g.enemies[p].pos = vec{6, 4}, vec{26, 4} })
	burnt, poisoned := 1e6-g.enemies[b].hp, 1e6-g.enemies[p].hp
	if want := 2 * dotTick * 0.18 * g.st.dmg * 10; math.Abs(poisoned-want) > 0.01*want {
		t.Fatalf("poison took %.1f through 1000 armour in a second, want %.1f", poisoned, want)
	}
	if burnt <= 0 || burnt > poisoned/5 {
		t.Fatalf("burn took %.1f through 1000 armour (poison %.1f): armour should all but stop it", burnt, poisoned)
	}
}

// Enough chill freezes; the frozen do not move or bite, and thaw.
func TestChillFreezesAndTheFrozenStandStill(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kBoba, vec{16, 12}, 1e6, 0)
	g.addChill(i, freezeAt-1)
	if g.enemies[i].frozenT > 0 {
		t.Fatalf("frozen at %d chills, want %d", freezeAt-1, freezeAt)
	}
	g.addChill(i, 1)
	if g.enemies[i].frozenT <= 0 {
		t.Fatal("not frozen at", freezeAt, "chills")
	}
	at := g.enemies[i].pos
	run(g, 1, nil)
	if g.enemies[i].pos != at {
		t.Fatalf("a frozen boba moved from %v to %v", at, g.enemies[i].pos)
	}
	run(g, 1, nil)
	if g.enemies[i].pos == at {
		t.Fatal("still standing after the freeze wore off")
	}
}

// Thermal Shock: fire on the frozen is a blast, and ends the freeze.
func TestThermalShock(t *testing.T) {
	g := playing()
	pickN(g, augChill, 1)
	pickN(g, augIgnite, 1)
	pickN(g, augThermal, 1)
	i := stillEnemy(g, kBoba, vec{16, 12}, 1e6, 0)
	g.freeze(i)
	g.addBurn(i, 1)
	if lost := 1e6 - g.enemies[i].hp; lost < 4*g.st.dmg {
		t.Fatalf("thermal shock took %.1f, want at least %.1f", lost, 4*g.st.dmg)
	}
	if g.enemies[i].frozenT > 0 {
		t.Fatal("still frozen after the blast")
	}
}

// Wildfire and Plague: a carrier's stacks go to its neighbours as it dies,
// and not to anything out of reach.
func TestStacksSpreadAsTheyDie(t *testing.T) {
	g := playing()
	pickN(g, augIgnite, 1)
	pickN(g, augWildfire, 1)
	pickN(g, augLeak, 1)
	pickN(g, augPlague, 1)
	dying := stillEnemy(g, kBoba, vec{10, 10}, 10, 0)
	near := stillEnemy(g, kBoba, vec{11, 10}, 1e6, 0)
	far := stillEnemy(g, kBoba, vec{20, 10}, 1e6, 0)
	g.enemies[dying].burn, g.enemies[dying].poison = 8, 6
	g.damage(dying, 1e6, false)
	if e := g.enemies[near]; e.burn != 4 || e.poison != 3 {
		t.Fatalf("the neighbour got %d burns and %d poisons, want half: 4 and 3", e.burn, e.poison)
	}
	if e := g.enemies[far]; e.burn != 0 || e.poison != 0 {
		t.Fatal("stacks went 10m away")
	}
}

// Two dashes, then none until one recharges; a dash is untouchable all the
// way.
func TestTheDashHasCharges(t *testing.T) {
	g := playing()
	g.hp = 1e9
	for k := 0; k < 2; k++ {
		g.dash()
		if g.dashT <= 0 || g.hurtT < g.dashT {
			t.Fatalf("dash %d: dashing %.2f, untouchable %.2f", k+1, g.dashT, g.hurtT)
		}
		for g.dashT > 0 {
			g.step(tick)
		}
	}
	g.dash()
	if g.dashT > 0 {
		t.Fatal("a third dash with no charge left")
	}
	for t0 := g.runTime; g.runTime-t0 < 1.6*g.st.dashCD+0.1; {
		g.step(tick)
	}
	if g.dashN != 1 {
		t.Fatalf("%d charges after one recharge time, want 1", g.dashN)
	}
}

// Ferris shows an attack before it: it winds up, fires its salvo when the
// wind-up runs out, and every third attack is a slam whose wave hurts once,
// names itself, and is dashed through.
func TestFerrisWindsUpAndTheSlamCanBeDashed(t *testing.T) {
	g := playing()
	g.hp = 1e6
	var spot vec
	for y := 12.0; y > 2; y -= 0.5 {
		if p := (vec{g.pos.x, y}); !solid(p.x, p.y) && g.sees(p) && g.pos.dist2(p) > 25 {
			spot = p
			break
		}
	}
	if spot == (vec{}) {
		t.Fatal("no spot in sight of the player")
	}
	f := stillEnemy(g, kFerris, spot, 1e9, 0)
	g.enemies[f].cd = 0
	hostile := func() int {
		n := 0
		for i := 0; i < g.shotTop; i++ {
			if s := &g.shots[i]; s.alive && s.hostile() && s.src == kFerris {
				n++
			}
		}
		return n
	}
	g.stepEnemies(tick)
	if g.enemies[f].windT <= 0 || hostile() != 0 {
		t.Fatalf("no wind-up first: wind %.2f, %d shots already", g.enemies[f].windT, hostile())
	}
	for g.enemies[f].windT > 0 {
		g.stepEnemies(tick)
	}
	if n := hostile(); n != 5 {
		t.Fatalf("%d shots in the salvo, want 5", n)
	}

	g.enemies[f].attacks, g.enemies[f].cd = 2, 0
	run(g, windUp(kFerris, true)+0.1, nil)
	if !g.wave.alive {
		t.Fatal("the third attack was no slam")
	}
	hp := g.hp
	for g.wave.alive {
		g.stepWave(tick)
	}
	if g.hp >= hp || g.hurtBy != kFerris || g.hurtHow != "claw slam" {
		t.Fatalf("the wave: %.0f → %.0f, by %s %q", hp, g.hp, kinds[g.hurtBy].name, g.hurtHow)
	}

	g.hurtT, g.hurtHow = 0, ""
	g.wave = wave{alive: true, pos: g.enemies[f].pos, r: 1, dmg: 50}
	for g.wave.alive {
		if gap := math.Sqrt(g.pos.dist2(g.wave.pos)) - g.wave.r; gap < 0.6 && g.dashT <= 0 && g.hurtT <= 0 {
			g.dash()
		}
		g.stepWave(tick)
		g.stepPlayer(tick)
	}
	if g.hurtHow != "" || !g.wave.hit {
		t.Fatalf("dashed through the wave: hurt by %q, passed %v", g.hurtHow, g.wave.hit)
	}
}

// Every card has a name, a line, a picture of its own, and a need that is
// itself an entry card, so no card waits on one that can never come.
func TestEveryCardIsComplete(t *testing.T) {
	seen := map[string]augID{}
	for a := augID(0); a < nAug; a++ {
		au := augs[a]
		if au.name == "" || au.what == "" || au.tag == "" || au.max < 1 {
			t.Errorf("card %d is missing a name, line, tag or rank", a)
		}
		// A card is 28 columns at most: 25 between its margin and its frame.
		for _, line := range []string{au.name, au.what, au.what2} {
			if width(line) > 25 {
				t.Errorf("%s: %q is %d columns, past the card's frame", au.name, line, width(line))
			}
		}
		if n := augs[au.needs].needs; n != au.needs {
			t.Errorf("%s needs %s, which needs %s", au.name, augs[au.needs].name, augs[n].name)
		}
		on, key := 0, make([]byte, 0, iconW*iconH)
		for k, b := range icons[a].on {
			if b {
				on++
				key = append(key, byte(k), byte(icons[a].px[k].r*255))
			}
		}
		if on < 15 {
			t.Errorf("%s's picture has %d pixels", au.name, on)
		}
		if b, dup := seen[string(key)]; dup {
			t.Errorf("%s has %s's picture", au.name, augs[b].name)
		}
		seen[string(key)] = a
	}
	if nAug < 70 {
		t.Errorf("%d cards", nAug)
	}
}
