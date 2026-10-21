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
			g.cards[0], g.cards[1], g.cards[2] = augRecursion, augHotPath, augScroll
			g.cardHov = 1
			return g
		},
		"dead": func() *game {
			g := scene()
			g.runTime, g.level, g.kills, g.peakShot, g.bigHit = 754, 31, 2041, 612, 1880
			g.best = records{runs: 4, secs: 700, level: 35, kills: 1900, peak: 500, bigHit: 2000}
			g.die()
			return g
		},
		"title": func() *game {
			g := newGame(42)
			g.best = records{runs: 4, secs: 754, level: 35, kills: 2041}
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
