package main

import (
	"math"
	"testing"
)

// sig fires boss i's own move at once, as its wind-up would.
func sig(g *game, i int) {
	e := &g.enemies[i]
	e.sig = true
	to := g.pos.sub(e.pos)
	g.enemyFire(i, e, to.scale(1/to.len()), to.len())
}

// Every boss's own move comes in its turn, and each is a different one.
func TestEveryBossHasItsOwnMove(t *testing.T) {
	for _, k := range bosses {
		e := enemy{k: k}
		n := 0
		for e.attacks = 0; e.attacks < 12; e.attacks++ {
			if sigs(&e) {
				n++
				if slams(&e) {
					t.Errorf("%s: its own move and its slam on the same turn", kinds[k].name)
				}
			}
		}
		if k != kReconciler && n == 0 { // the Reconciler's is in its re-render
			t.Errorf("%s never makes its own move", kinds[k].name)
		}
	}
}

// Boba Prime's straw pulls a player who stands still, and not one who
// dashes.
func TestTheStrawPulls(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kBobaPrime, inSight(t, g, 6), 1e6, 0)
	sig(g, i)
	if g.pullT <= 0 {
		t.Fatal("the straw did not start")
	}
	d0 := g.pos.dist2(g.enemies[i].pos)
	for k := 0; k < 15; k++ {
		g.stepPlayer(tick)
	}
	if d := g.pos.dist2(g.enemies[i].pos); d >= d0-0.5 {
		t.Fatalf("the player was not pulled: %.2f → %.2f", d0, d)
	}
}

// Ferris under the floor is not seen and not hurt, and comes up by the
// player where the floor was marked.
func TestFerrisBurrows(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kFerris, inSight(t, g, 8), 1e6, 0)
	e := &g.enemies[i]
	sig(g, i)
	if e.under <= 0 {
		t.Fatal("it did not go under")
	}
	if g.hit(i, 500, false, 1, elNone); e.hp != 1e6 {
		t.Fatal("hurt under the floor")
	}
	var near [1]int16
	if g.nearest(g.pos, 20, -1, near[:]) != 0 {
		t.Fatal("an active found it under the floor")
	}
	marked := false
	for s := 0; s < 30*3 && e.under > 0; s++ {
		g.hurtT = 1
		g.stepEnemies(tick)
		g.stepHazards(tick)
		marked = marked || g.marked()
	}
	g.stepHazards(tick)
	if !marked {
		t.Error("the floor was not marked before it came up")
	}
	if d := e.pos.dist2(g.pos); d > 2.5*2.5 {
		t.Errorf("it came up %.1f from the player", d)
	}
}

// The Modal Queen's modal: nothing hurts her while its buttons stand;
// break both and she can be hurt. Left open, it closes and she mends.
func TestTheModalGuardsHer(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kQueen, inSight(t, g, 7), 1000, 0)
	q := &g.enemies[i]
	sig(g, i)
	if q.guards != 2 {
		t.Fatalf("%d buttons guard her, want 2", q.guards)
	}
	if g.hit(i, 100, false, 1, elNone); q.hp != 1000 {
		t.Fatal("hurt through her modal")
	}
	for j := range g.enemies {
		if b := &g.enemies[j]; b.alive && b.guardOf > 0 {
			g.kill(j)
		}
	}
	if g.hit(i, 100, false, 1, elNone); q.guards != 0 || q.hp >= 1000 {
		t.Fatalf("buttons broken, still guarded: %d guards, health %.0f", q.guards, q.hp)
	}

	q.hp = 500
	sig(g, i)
	q.guardT = 0.01
	g.stepEnemies(tick)
	if q.guards != 0 || q.hp != 600 {
		t.Fatalf("a modal left open: %d guards, health %.0f, want 0 and 600", q.guards, q.hp)
	}
}

// Ink's Reconciler leaves a virtual copy when it re-renders: the copy is
// soft, its death is not the boss's, and the boss's ends the copies.
func TestTheReconcilerLeavesACopy(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kReconciler, inSight(t, g, 7), 1e5, 0)
	g.bossLive = true
	e := &g.enemies[i]
	e.attacks = 1 // its re-render's turn
	e.slam = true
	to := g.pos.sub(e.pos)
	g.enemyFire(i, e, to.scale(1/to.len()), to.len())
	copies := func() (n, at int) {
		for j := range g.enemies {
			if d := &g.enemies[j]; d.alive && d.decoy {
				n, at = n+1, j
			}
		}
		return
	}
	n, c := copies()
	if n != 1 || g.enemies[c].hp > e.max*0.1 {
		t.Fatalf("%d copies after a re-render", n)
	}
	kills := g.kills
	g.kill(c)
	if !g.bossLive || g.kills != kills {
		t.Fatal("the copy's death counted as the boss's")
	}
	g.enemyFire(i, e, to.scale(1/to.len()), to.len())
	g.kill(i)
	if n, _ := copies(); n != 0 {
		t.Fatal("a copy outlived its Reconciler")
	}
}

// Textual's grid: stripes over the player that go first, and the stripes
// between that go after.
func TestTextualsGrid(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kTextual, inSight(t, g, 7), 1e6, 0)
	sig(g, i)
	first, later := 0, 0
	for k := range g.hazards {
		h := &g.hazards[k]
		if !h.alive || h.hw == 0 {
			continue
		}
		if h.t < 2 {
			first++
		} else {
			later++
		}
	}
	if first != 3 || later != 2 || !g.marked() {
		t.Fatalf("%d stripes first, %d after, player marked %v", first, later, g.marked())
	}
}

// Under half health a boss enters its second phase, once.
func TestBossesHaveASecondPhase(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kBobaPrime, inSight(t, g, 6), 1000, 0)
	g.enemies[i].hp = 400
	g.stepEnemies(tick)
	if !g.enemies[i].phase2 {
		t.Fatal("no second phase under half health")
	}
	g.enemies[i].cd = 0
	g.enemies[i].sig = false
	to := g.pos.sub(g.enemies[i].pos)
	g.enemyFire(i, &g.enemies[i], to.scale(1/to.len()), to.len())
	if g.enemies[i].cd >= 2.2 {
		t.Fatalf("its attacks come no faster: %.2f", g.enemies[i].cd)
	}
}

// A boss shot, blasted or bitten into is not knocked back: it goes where it
// would have gone untouched.
func TestBossesAreNotKnockedBack(t *testing.T) {
	for _, k := range bosses {
		var at [2]vec
		for n, knock := range []bool{false, true} {
			g := playing()
			i := stillEnemy(g, k, openSpot(t, g, kinds[k].r+0.5), 1e6, 0)
			e := &g.enemies[i]
			if knock {
				e.knock = e.pos.sub(g.pos).scale(10)
			}
			for range 5 {
				g.moveEnemy(i, e, tick)
			}
			at[n] = e.pos
		}
		if at[0].dist2(at[1]) > 1e-12 {
			t.Errorf("%s was pushed from %v to %v", kinds[k].name, at[0], at[1])
		}
	}
}

// openSpot is a spot in sight of the player with r of floor all round it.
func openSpot(t *testing.T, g *game, r float64) vec {
	t.Helper()
	for d := 3.0; d < 9; d++ {
		for a := 0.0; a < 2*math.Pi; a += 0.1 {
			if p := g.pos.add(dir(a).scale(d)); free(p.x, p.y, r) && g.sees(p) {
				return p
			}
		}
	}
	t.Fatal("no roomy spot in sight")
	return vec{}
}

// A boss cannot be frozen, is slowed only a little, and carries half the
// burn and poison, so it is not kited and sniped from range.
func TestBossesShrugOffStatus(t *testing.T) {
	g := playing()
	i := stillEnemy(g, kBobaPrime, openSpot(t, g, kinds[kBobaPrime].r+0.5), 1e6, 0)
	e := &g.enemies[i]
	g.addChill(i, 3*freezeAt)
	g.freeze(i)
	g.addBurn(i, 3*maxBurn)
	g.addPoison(i, 3*maxPoison)
	if e.frozenT > 0 || e.chill != freezeAt-1 || e.burn != maxBurn/2 || e.poison != maxPoison/2 {
		t.Fatalf("frozen %.1f, chill %d, burn %d, poison %d", e.frozenT, e.chill, e.burn, e.poison)
	}
}
