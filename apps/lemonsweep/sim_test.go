package main

import (
	"math"
	"os"
	"slices"
	"testing"
)

// bot plays: it aims at a boss in sight, else the nearest enemy (a better aim than a
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
	// A boss in sight is the target, as it would be a player's, unless
	// something is about to bite.
	if nd > 2.5*2.5 {
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && kinds[e.k].boss && g.sees(e.pos) {
				near, nd = i, e.pos.dist2(g.pos)
				break
			}
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
		// What the warnings say: a wind-up means step aside, a shot about
		// to land means dash.
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && e.windT > 0 && !e.slam {
				side := int(g.runTime/3)%2 == 0
				g.held[actLeft], g.held[actRight] = side, !side
				g.held[actFwd], g.held[actBack] = false, false
				break
			}
		}
		for i := 0; i < g.shotTop; i++ {
			if s := &g.shots[i]; s.alive && s.hostile() && s.pos.dist2(g.pos) < 1.2*1.2 {
				if to := g.pos.sub(s.pos); to.x*s.vel.x+to.y*s.vel.y > 0 && g.dashN > 1 {
					g.dash()
				}
			}
		}
		// A marked floor: step off it, and dash if it is about to go.
		for k := range g.hazards {
			h := &g.hazards[k]
			if !h.alive || g.pos.dist2(h.pos) > (h.r+0.3)*(h.r+0.3) {
				continue
			}
			out := g.pos.sub(h.pos)
			f, r := dir(g.ang), dir(g.ang+math.Pi/2)
			fw, rt := out.x*f.x+out.y*f.y, out.x*r.x+out.y*r.y
			g.held[actFwd], g.held[actBack] = fw > 0, fw < 0
			g.held[actRight], g.held[actLeft] = rt > 0, rt < 0
			if h.t < 0.3 {
				g.dash()
			}
			break
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
// SIM unset it plays one short run for crashes only (SIMBUILD=name plays
// one build). The assertion that matters: every run ends, won on the three
// maps or ended by the pressure, broken or not.
func TestSimulatedRuns(t *testing.T) {
	// Six actives and six passives each, the order the bot prefers them.
	builds := map[string][]augID{
		"broken":   {augRoundTrip, augFork, augCron, augPanic, augSegfault, augRecursion, augStorm, augCascade, augTesla, augScroll, augCompaction, augREP},
		"steady":   {augCron, augHotPath, augTesla, augLoop, augColdPath, augHeap, augDaemon, augShutdown, augIRQ, augRecover, augAbsZero, augEscape},
		"anything": nil,
		"fire":     {augSegfault, augIgnite, augBurnIn, augFirewall, augCron, augOverheat, augRoundTrip, augWildfire, augTesla, augIncinerate, augDaemon, augHotPath},
		"frost":    {augColdPath, augChill, augAbsZero, augShatter, augCron, augPermafrost, augSegfault, augShards, augTesla, augThermal, augRoundTrip, augIgnite},
		"storm":    {augTesla, augChain, augStorm, augStatic, augIRQ, augConductor, augCron, augThunder, augRoundTrip, augShort, augDaemon, augSuper},
		"plague":   {augGasLeak, augLeak, augCompaction, augPlague, augCron, augDangling, augTesla, augCorrode, augRoundTrip, augNeuro, augDaemon, augHotPath},
		"machine":  {augCron, augHotPath, augDaemon, augLoop, augRoundTrip, augRetry, augColdPath, augBatch, augStorm, augBigO, augTesla, augEscape},
	}
	limit, seeds := 30.0, uint64(3)
	if os.Getenv("SIM") != "" {
		limit, seeds = 60*60, 10
	}
	for _, endless := range []bool{false, true} {
		for name, taste := range builds {
			if os.Getenv("SIMBUILD") != "" && os.Getenv("SIMBUILD") != name {
				continue
			}
			var lasted []float64
			won := 0
			for seed := uint64(1); seed <= seeds; seed++ {
				g := newGame(seed * 7919)
				g.endless = endless
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
				end := "died"
				switch {
				case g.won:
					end, won = "WON ", won+1
				case g.phase != phDead:
					end = "alive"
				}
				t.Logf("%-7s %-8s seed %2d: %s at %2d:%02d on %-12s (map at %2d:%02d), level %2d at 3:00, %3d at the end, %5d swept, biggest hit %6.0f, pressure x%.1f, by %s %s",
					map[bool]string{true: "endless", false: "maps"}[endless], name, seed, end,
					int(g.runTime)/60, int(g.runTime)%60, cur.name, int(g.stageT)/60, int(g.stageT)%60, at3, g.level, g.kills, g.bigHit, g.pressure(), kinds[g.killedBy].name, g.killedHow)
				if g.phase != phDead {
					t.Errorf("%s seed %d still alive after %d minutes: the pressure does not end runs", name, seed, int(limit/60))
				}
			}
			if len(lasted) > 0 {
				slices.Sort(lasted)
				m := lasted[len(lasted)/2]
				t.Logf("MEDIAN %-7s %-8s %2d:%02d (shortest %2d:%02d, longest %2d:%02d), %d of %d won", map[bool]string{true: "endless", false: "maps"}[endless], name, int(m)/60, int(m)%60,
					int(lasted[0])/60, int(lasted[0])%60, int(lasted[len(lasted)-1])/60, int(lasted[len(lasted)-1])%60, won, len(lasted))
			}
		}
	}
}
