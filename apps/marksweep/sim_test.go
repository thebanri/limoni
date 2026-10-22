package main

import (
	"math"
	"os"
	"slices"
	"testing"
)

// bot plays: it aims at the nearest enemy it can see (a better aim than a
// person's, so its runs are an upper bound), fires, collects when the heap
// is empty, backs off from what is close, dashes when cornered, and takes
// cards by a simple taste.
func bot(g *game, taste []augID) {
	switch g.phase {
	case phCards:
		best, score := 0, -1
		for i := 0; i < g.nCards; i++ {
			s := 1
			for k, a := range taste {
				if g.cards[i] == a {
					s = 100 - k
				}
			}
			if s > score {
				best, score = i, s
			}
		}
		g.take(best)
		return
	case phPlay:
	default:
		return
	}
	near, nd := -1, 1e9
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		if d := e.pos.dist2(g.pos); d < nd && g.sees(e.pos) {
			near, nd = i, d
		}
	}
	g.held = [nAct]bool{}
	if near >= 0 {
		to := g.enemies[near].pos.sub(g.pos)
		g.ang = math.Atan2(to.y, to.x)
		g.held[actFire] = g.ammo > 0
		d := math.Sqrt(nd)
		// Away from the crowd: the sum of what is close, weighted by how
		// close, turned into the keys that move that way.
		var away vec
		for i := range g.enemies {
			e := &g.enemies[i]
			if e.alive {
				if dd := e.pos.sub(g.pos); dd.len() < 5 {
					away = away.sub(dd.scale(1 / (dd.len()*dd.len() + 0.1)))
				}
			}
		}
		if away.len() > 0 && d < 5 {
			f, r := dir(g.ang), dir(g.ang+math.Pi/2)
			fw := away.x*f.x + away.y*f.y
			rt := away.x*r.x + away.y*r.y
			g.held[actFwd], g.held[actBack] = fw > 0.05, fw < -0.05
			g.held[actRight], g.held[actLeft] = rt > 0.05, rt < -0.05
		}
		if d < 1.6 && g.hp < g.st.maxHP*0.5 {
			g.dash()
		}
		// What the game says to do about a claw slam: dash through it.
		if w := g.wave; w.alive && !w.hit {
			if gap := math.Sqrt(g.pos.dist2(w.pos)) - w.r; gap > 0 && gap < 0.6 {
				g.dash()
			}
		}
	} else {
		// Nothing in sight: walk back to the middle.
		to := vec{16, 16}.sub(g.pos)
		g.ang = math.Atan2(to.y, to.x)
		g.held[actFwd] = to.len() > 2
	}
	if g.ammo == 0 {
		g.reload()
	}
}

// TestSimulatedRuns prints how long the bot lasts with a few builds; with
// SIM unset it plays one short run for crashes only. The assertion that
// matters: the pressure ends every run, broken or not.
func TestSimulatedRuns(t *testing.T) {
	builds := map[string][]augID{
		"broken":   {augFork, augRecursion, augPanic, augCascade, augScroll, augREP, augChain, augLoop, augHotPath},
		"steady":   {augHotPath, augLoop, augHeap, augConcGC, augShutdown, augRecover, augEscape, augDiff},
		"anything": nil,
		"fire":     {augIgnite, augFirewall, augOverheat, augWildfire, augIncinerate, augAfterburn, augHotPath, augLoop},
		"frost":    {augChill, augShatter, augPermafrost, augShards, augAbsZero, augColdSnap, augThermal, augIgnite, augLoop},
		"storm":    {augChain, augStatic, augConductor, augThunder, augTesla, augShort, augSuper, augLoop},
		"plague":   {augLeak, augPlague, augDangling, augCorrode, augFumes, augNeuro, augLoop, augHotPath},
		"machine":  {augCron, augDaemon, augDoubleBuf, augRam, augRetry, augBatch, augBigO, augHotPath},
	}
	limit, seeds := 30.0, uint64(3)
	if os.Getenv("SIM") != "" {
		limit, seeds = 25*60, 10
	}
	for name, taste := range builds {
		var lasted []float64
		for seed := uint64(1); seed <= seeds; seed++ {
			g := newGame(seed * 7919)
			g.reset()
			g.exact = true
			at3 := 0
			for g.phase != phDead && g.runTime < limit {
				bot(g, taste)
				g.step(tick)
				if at3 == 0 && g.runTime >= 180 {
					at3 = g.level
				}
			}
			if os.Getenv("SIM") == "" {
				continue
			}
			lasted = append(lasted, g.runTime)
			t.Logf("%-8s seed %d: %s at %2d:%02d, level %2d at 3:00, %3d at the end, %5d swept, peak %3d shots, biggest hit %6.0f, pressure x%.1f, by %s %s",
				name, seed, map[bool]string{true: "died", false: "alive"}[g.phase == phDead],
				int(g.runTime)/60, int(g.runTime)%60, at3, g.level, g.kills, g.peakShot, g.bigHit, g.pressure(), kinds[g.killedBy].name, g.killedHow)
			if g.phase != phDead {
				t.Errorf("%s seed %d still alive after %d minutes: the pressure does not end runs", name, seed, int(limit/60))
			}
		}
		if len(lasted) > 0 {
			slices.Sort(lasted)
			m := lasted[len(lasted)/2]
			t.Logf("MEDIAN %-8s %2d:%02d (shortest %2d:%02d, longest %2d:%02d)", name, int(m)/60, int(m)%60,
				int(lasted[0])/60, int(lasted[0])%60, int(lasted[len(lasted)-1])/60, int(lasted[len(lasted)-1])%60)
		}
	}
}
