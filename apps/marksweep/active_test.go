package main

import (
	"math"
	"testing"
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
