package main

import (
	"math"
	"testing"
)

// Every map is closed, the player starts on floor, bosses arrive on floor,
// and all of it can be walked to from the start: nothing spawns sealed in.
func TestEveryMapIsWhole(t *testing.T) {
	defer useArena(0)
	for i := range arenas {
		a := &arenas[i]
		for y, row := range a.rows {
			if len(row) != mapW {
				t.Fatalf("%s row %d is %d wide", a.name, y, len(row))
			}
		}
		useArena(i)
		for k := 0; k < mapW; k++ {
			for _, p := range [4][2]int{{k, 0}, {k, mapH - 1}, {0, k}, {mapW - 1, k}} {
				if tile(p[0], p[1]) == '.' {
					t.Fatalf("%s is open at %v", a.name, p)
				}
			}
		}
		for _, p := range []vec{{16, 16.5}, {16, 4}, {16, 28}} {
			if !free(p.x, p.y, 0.6) {
				t.Fatalf("%s: %v is not floor", a.name, p)
			}
		}
		if len(portals) < 6 {
			t.Fatalf("%s has %d spawn points", a.name, len(portals))
		}
		// Flood the floor from the start; every floor tile must be reached.
		var seen [mapW * mapH]bool
		todo := [][2]int{{16, 16}}
		seen[16*mapW+16] = true
		for len(todo) > 0 {
			p := todo[len(todo)-1]
			todo = todo[:len(todo)-1]
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, y := p[0]+d[0], p[1]+d[1]
				if tile(x, y) == '.' && !seen[y*mapW+x] {
					seen[y*mapW+x] = true
					todo = append(todo, [2]int{x, y})
				}
			}
		}
		for y := 0; y < mapH; y++ {
			for x := 0; x < mapW; x++ {
				if tile(x, y) == '.' && !seen[y*mapW+x] {
					t.Fatalf("%s: floor at (%d, %d) cannot be reached", a.name, x, y)
				}
			}
		}
	}
}

// Ten minutes on a map bring its warden; sweeping it moves the run to the
// next map with the build and full health, and nothing of the last map;
// the third warden ends the run as a win.
func TestThreeMapsThenSweptClean(t *testing.T) {
	g := playing()
	defer useArena(0)
	for stage := 0; stage < nStages; stage++ {
		if g.stage != stage || cur != &arenas[stage] {
			t.Fatalf("on map %d (%s), want %d", g.stage, cur.name, stage)
		}
		g.stageT = wardenAt - tick/2
		g.step(tick)
		w := -1
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && e.warden {
				w = i
			}
		}
		if w < 0 || g.enemies[w].k != wardens[stage] {
			t.Fatalf("map %d: no warden at ten minutes", stage)
		}
		g.kill(w)
		if g.clearT <= 0 {
			t.Fatalf("map %d: the warden swept, but no next map coming", stage)
		}
		g.addEnemy(kSpinner, vec{16, 10})
		g.hp = 1
		for g.clearT > 0 && g.phase != phDead {
			if g.phase == phCards {
				g.take(0) // the warden's bytes are worth a level or two
			}
			g.step(tick)
		}
		if stage == nStages-1 {
			break
		}
		for i := range g.enemies {
			if g.enemies[i].alive {
				t.Fatalf("map %d: an enemy came along to the next map", stage)
			}
		}
		if g.hp != g.st.maxHP || g.stageT != 0 || g.pos != (vec{16, 16.5}) {
			t.Fatalf("map %d: next map starts at hp %.0f, clock %.1f, %v", stage, g.hp, g.stageT, g.pos)
		}
	}
	if !g.won || g.phase != phDead || g.best.runs != 1 || g.best.endless.runs != 0 {
		t.Fatalf("after the last warden: won %v, phase %v, runs %d/%d", g.won, g.phase, g.best.runs, g.best.endless.runs)
	}
	if g.heat() < stageLead*2 {
		t.Fatal("the last map started cold")
	}
}

// ENDLESS on the title is /dev/zero: no warden, no next map, its own
// records, and the next run from the death card is endless again.
func TestEndlessIsDevZero(t *testing.T) {
	g := newGame(3)
	defer useArena(0)
	g.titlePick(1)
	if !g.endless || cur != &arenas[len(arenas)-1] || g.phase != phPlay {
		t.Fatalf("ENDLESS gave %s, endless %v", cur.name, g.endless)
	}
	g.stageT = wardenAt - tick/2
	g.step(tick)
	for i := range g.enemies {
		if g.enemies[i].alive && g.enemies[i].warden {
			t.Fatal("a warden came to /dev/zero")
		}
	}
	g.die()
	if g.best.endless.runs != 1 || g.best.runs != 0 {
		t.Fatalf("endless run counted as %d/%d", g.best.runs, g.best.endless.runs)
	}
	g.reset()
	if !g.endless || cur != &arenas[len(arenas)-1] {
		t.Fatal("run again left /dev/zero")
	}
}

// An enemy with a wall across its line walks round it: on THE STACK a
// boba straight above a frame's wall reaches a player straight below it.
func TestEnemiesWalkRoundWalls(t *testing.T) {
	g := playing()
	defer useArena(0)
	g.enter(1)
	g.pos = vec{8.5, 9.5}
	i := g.addEnemy(kBoba, vec{8.5, 3.5})
	if g.sees(g.enemies[i].pos) {
		t.Fatal("the wall is not between them")
	}
	for s := 0; s < 30*20 && g.enemies[i].pos.dist2(g.pos) > 1; s++ {
		g.hurtT = 1 // it may bite; the test is whether it arrives
		g.stepEnemies(tick)
	}
	if d := g.enemies[i].pos.dist2(g.pos); d > 1 {
		t.Fatalf("the boba is still %.1f away, at %v", math.Sqrt(d), g.enemies[i].pos)
	}
}
