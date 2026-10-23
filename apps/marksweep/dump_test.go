package main

import (
	"fmt"
	"html"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// playing is a run in progress, the same every time.
func playing() *game {
	g := newGame(42)
	g.reset()
	return g
}

func snapshot(b *buffer.Buffer) string {
	var sb strings.Builder
	for y := 0; y < int(b.Area.Height); y++ {
		for x := 0; x < int(b.Area.Width); x++ {
			r := b.Content[y*int(b.Area.Width)+x].Content
			if r == 0 {
				r = ' '
			}
			sb.WriteRune(r)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func TestDumpFrame(t *testing.T) {
	dir := os.Getenv("DUMP")
	if dir == "" {
		t.Skip("set DUMP to a directory to write frames as HTML")
	}
	scene := func() *game {
		g := playing()
		g.pos = vec{16, 26}
		g.ang = -1.5708
		for i, k := range []kindID{kBoba, kCrab, kModal, kJSX, kElder, kBar, kSpinner} {
			g.addEnemy(k, vec{12.5 + float64(i)*1.2, 22.6 + float64(i%2)*0.9})
		}
		g.step(1.0 / 30)
		return g
	}
	shots := map[string]func() *game{
		"play": scene,
		"cards": func() *game {
			g := scene()
			pickN(g, augFork, 2)
			pickN(g, augHotPath, 5)
			g.pending = 1
			g.openCards()
			g.cards[0], g.cards[1], g.cards[2] = augRecursion, augPanic, augHotPath
			g.cardHov = 1
			return g
		},
		"dead": func() *game {
			g := scene()
			g.runTime, g.level, g.kills, g.peakShot, g.bigHit = 754, 31, 2041, 612, 1880
			g.best = records{bests: bests{runs: 4, secs: 700, level: 35, kills: 1900, peak: 500, bigHit: 2000}}
			g.die()
			return g
		},
		"pause": func() *game {
			g := scene()
			pickN(g, augTesla, 2)
			pickN(g, augHotPath, 1)
			g.pauseRun()
			return g
		},
		"title": func() *game {
			g := newGame(42)
			g.best = records{bests: bests{runs: 4, secs: 754, level: 35, kills: 2041}, endless: bests{runs: 2, secs: 1250, level: 58, kills: 2900}}
			g.clock = 3
			return g
		},
		"broken": func() *game {
			g := scene()
			brokenBuild(g)
			g.exact = true
			g.press(actFire)
			fillArena(g, 60, true)
			for i := 0; i < 45; i++ {
				g.step(1.0 / 30)
				g.ammo, g.hp = g.st.mag, 1e9
				g.phase, g.pending = phPlay, 0
			}
			g.hp = g.st.maxHP * 0.6
			return g
		},
	}
	// Each map, a few frameworks in it; /dev/zero as an endless run.
	for i, name := range []string{"heap", "stack", "kernel", "zero"} {
		shots["map-"+name] = func() *game {
			g := playing()
			if i == len(arenas)-1 {
				g.endless = true
				g.reset()
			} else {
				g.enter(i)
			}
			g.ang = -math.Pi / 2
			g.stageT, g.runTime, g.kills = 437, 437+float64(i)*640, 380+i*700
			for k, kd := range []kindID{kBoba, kCrab, kBeetle, kGloss, kSpinner} {
				g.addEnemy(kd, vec{13.5 + float64(k)*1.4, 11.5 + float64(k%2)})
			}
			g.step(1.0 / 30)
			g.msgT = 0
			return g
		}
	}
	shots["won"] = func() *game {
		g := scene()
		g.runTime, g.level, g.kills, g.peakShot, g.bigHit = 1931, 66, 3120, 640, 2400
		g.stage, g.won = nStages-1, true
		g.best = records{bests: bests{runs: 4, secs: 1500, level: 60, kills: 2900, peak: 500, bigHit: 2000}}
		g.die()
		return g
	}
	shots["ferris"] = func() *game {
		g := playing()
		g.pos, g.ang = vec{16, 26}, -math.Pi/2
		g.bossNext = 99
		f := g.addEnemy(kFerris, vec{14.6, 22.9})
		g.bossLive = true
		e := &g.enemies[f]
		e.windT, e.slam = 0.6, true
		for i := -2; i <= 2; i++ {
			g.enemyShot(e, vec{15.2 + float64(i)*0.45, 24.4 + math.Abs(float64(i))*0.15}, 1.3, 20, shotBar)
		}
		g.wave = wave{alive: true, pos: e.pos, r: 2.0}
		g.stepWave(0.001)
		g.hurt(24, vec{19, 22}, kFerris, "claw salvo")
		g.hurtFX = 0.3
		g.dashN = 1
		g.msgT = 0
		return g
	}
	shots["elements"] = func() *game {
		g := playing()
		g.pos, g.ang = vec{16, 26}, -math.Pi/2
		g.bossNext = 99
		pickN(g, augIgnite, 1)
		pickN(g, augChill, 1)
		pickN(g, augLeak, 1)
		pickN(g, augStatic, 1)
		for i, k := range []kindID{kBoba, kCrab, kModal, kElder} {
			j := g.addEnemy(k, vec{13.5 + float64(i)*2, 22.6})
			switch i {
			case 0:
				g.addBurn(j, 6)
			case 1:
				g.freeze(j)
			case 2:
				g.addPoison(j, 8)
			case 3:
				g.enemies[j].shockT = 2
			}
		}
		g.addZone(vec{16, 24}, 1.2, 3, elFire)
		for i := 0; i < 20; i++ {
			g.stepZones(1.0 / 30)
			g.stepParts(1.0 / 30)
			g.stepEnemies(1.0 / 30)
			g.clock += 1.0 / 30
		}
		g.msgT = 0
		return g
	}
	shots["cards2"] = func() *game {
		g := playing()
		for _, a := range []augID{augTesla, augRoundTrip, augCron, augIgnite, augIgnite, augHotPath, augChain} {
			pickN(g, a, 1)
		}
		g.pending = 1
		g.openCards()
		g.cards[0], g.cards[1], g.cards[2] = augStorm, augIgnite, augDangling
		g.cardHov = 0
		return g
	}
	shots["actives"] = func() *game {
		g := playing()
		g.pos, g.ang = vec{16, 26}, -math.Pi/2
		g.bossNext = 99
		for _, a := range []augID{augDaemon, augDaemon, augDaemon, augRoundTrip, augRoundTrip, augIRQ, augCompaction, augSegfault, augColdPath} {
			pickN(g, a, 1)
		}
		for i, k := range []kindID{kBoba, kCrab, kModal, kJSX, kElder, kBar, kSpinner, kBoba} {
			j := g.addEnemy(k, vec{12.5 + float64(i)*1.0, 22.8 + float64(i%2)*0.8})
			g.enemies[j].hp, g.enemies[j].max = 1e6, 1e6
		}
		for i := 0; i < 40; i++ {
			g.step(1.0 / 30)
			g.hp = g.st.maxHP
			g.pos, g.ang = vec{16, 26}, -math.Pi/2
		}
		g.msgT = 0
		return g
	}
	fx := func(acts []augID, steps int, at func(g *game, step int)) func() *game {
		return func() *game {
			g := playing()
			g.pos, g.ang = vec{16, 26}, -math.Pi/2
			g.bossNext = 99
			for _, a := range acts {
				pickN(g, a, 3)
			}
			for i, k := range []kindID{kBoba, kCrab, kModal, kJSX, kElder, kBar, kSpinner, kBoba} {
				j := g.addEnemy(k, vec{12.8 + float64(i)*0.9, 22.9 + float64(i%2)*0.7})
				g.enemies[j].hp, g.enemies[j].max = 1e6, 1e6
			}
			for i := 0; i < steps; i++ {
				if at != nil {
					at(g, i)
				}
				g.step(1.0 / 30)
				g.hp = g.st.maxHP
				g.pos, g.ang = vec{16, 26}, -math.Pi/2
				for j := range g.enemies {
					g.enemies[j].knock = vec{}
				}
			}
			g.msgT = 0
			return g
		}
	}
	shots["fx-gas"] = fx([]augID{augGasLeak, augBurnIn}, 50, func(g *game, i int) {
		g.pos = vec{14.5 + float64(i)*0.06, 25.5} // walk, for Burn-in
	})
	shots["fx-storm"] = func() *game {
		// Stopped as a bolt strikes, while the field's ring is out.
		g := fx([]augID{augStorm, augTesla, augIRQ}, 1, nil)()
		for i := 0; i < 300; i++ {
			g.step(1.0 / 30)
			g.hp, g.pos, g.ang = g.st.maxHP, vec{16, 26}, -math.Pi/2
			bolt := false
			for _, a := range g.arcs {
				bolt = bolt || (a.alive && a.za > 3 && a.life > a.max*0.6)
			}
			if bolt {
				break
			}
		}
		g.msgT = 0
		return g
	}
	shots["fx-vortex"] = fx([]augID{augCompaction, augAbsZero, augSegfault, augDaemon}, 33, nil)
	shots["crosshair"] = func() *game {
		g := playing()
		g.pos, g.ang = vec{16, 26}, -math.Pi/2
		g.bossNext = 99
		g.addEnemy(kBoba, vec{16, 23.5})
		g.hitT, g.killT, g.recoil = 0.1, 0, 0.3
		return g
	}
	shots["settings"] = func() *game {
		g := newGame(42)
		g.clock = 3
		g.openSettings()
		g.setSel = 1
		return g
	}
	shots["settings-pause"] = func() *game {
		g := scene()
		g.pauseRun()
		g.openSettings()
		g.setSel, g.ret = 2, reticle{style: 3, color: 2, size: 2, outline: true}
		return g
	}
	for s := range reticleStyles {
		s := s
		shots[fmt.Sprintf("ret%d", s)] = func() *game {
			g := playing()
			g.pos, g.ang = vec{16, 26}, -math.Pi/2
			g.bossNext = 99
			g.addEnemy(kElder, vec{17.2, 23.2})
			g.ret = reticle{style: s, color: s % len(reticleColors), size: 1, outline: true}
			return g
		}
	}
	shots["minions"] = func() *game {
		g := playing()
		g.pos, g.ang = vec{16, 26}, -math.Pi/2
		g.bossNext = 99
		for i, k := range []kindID{kBeetle, kCaret, kBomb, kGloss, kCursor} {
			j := g.addEnemy(k, vec{13.6 + float64(i)*1.2, 23.6 + float64(i%2)*0.5})
			g.enemies[j].phaseOff = 0
		}
		return g
	}
	boss := func(k kindID, slam bool, steps int) func() *game {
		return func() *game {
			g := playing()
			g.pos, g.ang = vec{16, 26}, -math.Pi/2
			g.bossNext, g.hp = 99, 1e6
			j := g.addEnemy(k, vec{15, 22.4})
			g.bossLive = true
			e := &g.enemies[j]
			if slam {
				e.attacks = 1
				if k == kTextual {
					e.attacks = 2
				}
			}
			e.slam = slams(e)
			to := g.pos.sub(e.pos)
			g.enemyFire(j, e, to.scale(1/to.len()), to.len())
			for i := 0; i < steps; i++ {
				g.stepShots(1.0 / 30)
				g.stepHazards(1.0 / 30)
				g.stepFX(1.0 / 30)
				g.clock += 1.0 / 30
			}
			e.windT, e.slam = 0.5, slams(e)
			g.msgT = 0
			return g
		}
	}
	shots["boss-queen"] = boss(kQueen, false, 6)
	shots["boss-ink"] = boss(kReconciler, true, 12)
	shots["boss-textual"] = boss(kTextual, false, 12)
	shots["poster"] = func() *game {
		g := playing()
		g.pos, g.ang = vec{16.12, 28.2}, -math.Pi/2
		g.level, g.kills, g.runTime = 12, 412, 312
		g.bossNext = 99 // no boss on the poster
		for _, a := range []augID{augREP, augREP, augFork, augPanic, augHotPath, augHotPath, augScroll} {
			pickN(g, a, 1)
		}
		place := []struct {
			k kindID
			p vec
		}{
			{kModal, vec{13.9, 22.7}}, {kCrab, vec{15.95, 22.6}}, {kElder, vec{18.0, 22.7}},
			{kBoba, vec{14.9, 24.4}}, {kBar, vec{17.1, 24.5}},
			{kJSX, vec{15.9, 25.7}}, {kSpinner, vec{14.4, 26.0}}, {kSpinner, vec{14.0, 26.25}}, {kSpinner, vec{17.6, 26.1}},
		}
		for _, e := range place {
			i := g.addEnemy(e.k, e.p)
			g.enemies[i].phaseOff = 0
			if e.k == kElder || e.k == kModal {
				g.enemies[i].hp = g.enemies[i].max * 0.6
			}
		}
		g.ammo = 9
		g.hp = g.st.maxHP * 0.72
		for i, a := range []float64{-1.66, -1.57, -1.5} {
			g.shoot(g.pos.add(vec{0, -1.2 - float64(i)*0.9}), a, 10, 0, shotPlayer)
		}
		g.shoot(vec{15.2, 23.4}, math.Pi/2, 5, 0, shotBar)
		g.recoil = 0.62
		g.mouseIn, g.mouseX, g.mouseY, g.viewW, g.viewH = true, 64, 21, 124, 40
		g.msgT = 0
		return g
	}
	gun := func(mx, my int, reload float64) func() *game {
		return func() *game {
			g := playing()
			g.pos, g.ang = vec{16, 20}, -math.Pi/2
			g.mouseIn, g.mouseX, g.mouseY = true, mx, my
			g.viewW, g.viewH = 140, 44
			g.ammo = 9
			if reload > 0 {
				g.reloadT = g.st.reload * (1 - reload)
			}
			return g
		}
	}
	shots["gun"] = gun(70, 22, 0)
	shots["gun-left"] = gun(20, 12, 0)
	shots["gun-right"] = gun(125, 30, 0)
	shots["gun-reload"] = gun(70, 22, 0.2)
	shots["gunbig"] = gun(100, 28, 0)
	if os.Getenv("DUMP_FRAMES") != "" {
		for _, r := range []float64{0.1, 0.25, 0.35, 0.5, 0.6, 0.75, 0.85, 0.95} {
			shots[fmt.Sprintf("reload%02d", int(r*100))] = gun(70, 22, r)
		}
		for _, k := range []float64{1, 0.8, 0.5} {
			k := k
			shots[fmt.Sprintf("fire%02d", int(k*100))] = func() *game {
				g := gun(70, 22, 0)()
				g.recoil = k
				for i := 0; i < 3; i++ {
					j := g.shoot(g.pos.add(vec{math.Cos(g.ang), math.Sin(g.ang)}.scale(0.4+float64(i)*1.6+(1-k)*3)), g.ang, 10, 0, shotPlayer)
					g.shots[j].age = 0.05
				}
				return g
			}
		}
	}
	for name, make := range shots {
		g := make()
		w, h := uint16(140), uint16(44)
		if name == "poster" {
			w, h = 124, 40 // bigger characters: it is shown small, on a card
		}
		if name == "gunbig" {
			w, h = 200, 56
		}
		b := buffer.NewBuffer(cell.Rect{Width: w, Height: h})
		g.render(b)
		writeHTML(t, b, dir+"/"+name+".html")
	}
}

// writeHTML writes the frame as a page of coloured cells, to look at in a
// browser when DUMP is a path.
func writeHTML(t *testing.T, b *buffer.Buffer, path string) {
	if !strings.HasSuffix(path, ".html") {
		t.Log("\n" + snapshot(b))
		return
	}
	var sb strings.Builder
	// Sized so the frame fills 1280×768, the cards' picture size.
	font := 1280.0 / float64(b.Area.Width) / 0.6
	line := 768.0 / float64(b.Area.Height)
	fmt.Fprintf(&sb, `<html><body style="margin:0;background:#000"><pre style="font:%.1fpx/%.2fpx monospace;margin:0">`, font, line)
	w := int(b.Area.Width)
	for y := 0; y < int(b.Area.Height); y++ {
		for x := 0; x < w; x++ {
			c := b.Content[y*w+x]
			r := c.Content
			if r == 0 {
				r = ' '
			}
			fr, fg, fb := c.Style.Fg.RGB()
			br, bg, bb := c.Style.Bg.RGB()
			fmt.Fprintf(&sb, `<span style="color:#%02x%02x%02x;background:#%02x%02x%02x">%s</span>`, fr, fg, fb, br, bg, bb, html.EscapeString(string(r)))
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("</pre></body></html>")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
