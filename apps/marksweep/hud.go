package main

import (
	"math"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

var (
	hudBg   = hex(0x0b0e16)
	hudText = hex(0xd8dcea)
	hudDim  = hex(0x6c7390)
	hudPink = hex(0xff5fd2)
	hudCyan = hex(0x5ad1ff)
	hudGood = hex(0x6ee07a)
	hudWarn = hex(0xffc94a)
	hudBad  = hex(0xff5a5a)
)

func (g *game) drawHUD(c canvas, v *view) {
	st := style(hudText, hudBg)
	dim := style(hudDim, hudBg)
	// Top row: the level and the bytes to the next one.
	c.fill(0, 0, c.w, 1, ' ', st)
	if g.phase == phTitle {
		c.fill(0, c.h-2, c.w, 2, ' ', st)
		return
	}
	x := c.text(1, 0, "LV ", dim)
	x = c.number(x, 0, g.level, style(hudCyan, hudBg))
	x++
	barW := c.w - x - 14
	if barW > 4 && g.phase != phTitle {
		frac := g.xp / need(g.level)
		g.bar(c, x, 0, barW, frac, hudGood, hex(0x1a2a1e))
		x += barW + 1
	}
	if g.phase != phTitle {
		x = c.clock(x+1, 0, int(g.runTime), st)
	}

	// Bottom rows.
	y := c.h - 2
	c.fill(0, y, c.w, 2, ' ', st)
	if g.phase == phTitle {
		return
	}
	x = c.text(1, y, "HP ", dim)
	hpCol := hudGood
	if g.hp < g.st.maxHP*0.3 {
		hpCol = hudBad
	}
	g.bar(c, x, y, 16, g.hp/g.st.maxHP, hpCol, hex(0x2a1418))
	x += 17
	x = c.number(x, y, int(math.Ceil(g.hp)), st)
	c.set(x, y, '/', dim)
	x = c.number(x+1, y, int(g.st.maxHP), dim)

	x += 3
	x = c.text(x, y, "HEAP ", dim)
	switch {
	case g.reloadT > 0:
		frac := 1 - g.reloadT/g.st.reload
		x = c.text(x, y, "GC PAUSE ", style(hudCyan, hudBg))
		g.bar(c, x, y, 10, frac, hudCyan, hex(0x10202a))
		x += 11
	default:
		pips := g.st.mag
		if pips > 20 {
			pips = 20
		}
		on := int(math.Ceil(float64(g.ammo) / float64(g.st.mag) * float64(pips)))
		for i := 0; i < pips; i++ {
			r, col := '▮', hudWarn
			if i >= on {
				r, col = '▯', hudDim
			}
			c.set(x, y, r, style(col, hudBg))
			x++
		}
		x++
		x = c.number(x, y, g.ammo, st)
		if g.ammo == 0 && int(g.clock*3)%2 == 0 {
			x = c.text(x+1, y, " R: COLLECT", style(hudBad, hudBg))
		}
	}

	x += 3
	x = c.text(x, y, "SWEPT ", dim)
	x = c.number(x, y, g.kills, st)
	x += 3
	x = c.text(x, y, "PRESSURE ", dim)
	p := g.pressure()
	if p <= 1 {
		left := 600 - int(g.runTime)
		if left < 0 {
			left = 0
		}
		x = c.text(x, y, "builds in ", dim)
		c.clock(x, y, left, st)
	} else {
		col := hudWarn
		if p > 4 {
			col = hudBad
		}
		x = c.text(x, y, "x", style(col, hudBg))
		x = c.number(x, y, int(p), style(col, hudBg))
		c.set(x, y, '.', style(col, hudBg))
		c.number(x+1, y, int(p*10)%10, style(col, hudBg))
	}

	// Second row: the build, or a message.
	y++
	if g.msgT > 0 {
		c.text(1, y, g.message, style(hudPink, hudBg))
		return
	}
	x = 1
	any := false
	for a := augID(0); a < nAug; a++ {
		if g.rank[a] == 0 {
			continue
		}
		any = true
		if x+width(augs[a].name)+5 > c.w {
			break
		}
		x = c.text(x, y, augs[a].name, style(hex(rarityCol[augs[a].rar]), hudBg))
		x = c.number(x+1, y, g.rank[a], st)
		if g.over[a] > 0 {
			x = c.text(x, y, "+", style(hudWarn, hudBg))
			x = c.number(x, y, g.over[a], style(hudWarn, hudBg))
		}
		x += 2
	}
	if !any {
		c.text(1, y, "WASD move · mouse aim · click sweep · R collect · Space dash · Esc pause", dim)
	}
	if g.bossLive {
		g.drawBossBar(c)
	}
}

var eighths = []rune("▏▎▍▌▋▊▉")

func (g *game) bar(c canvas, x, y, w int, frac float64, on, off rgb) {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	full := frac * float64(w)
	for i := 0; i < w; i++ {
		r, col := '█', on
		switch {
		case float64(i+1) <= full:
		case float64(i) < full:
			r = eighths[int((full-float64(i))*7)%7]
		default:
			r, col = ' ', off
		}
		c.set(x+i, y, r, style(col, off))
	}
}

func (g *game) drawBossBar(c canvas) {
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive || !kinds[e.k].boss {
			continue
		}
		name := kinds[e.k].name
		w := c.w / 2
		x0 := (c.w - w) / 2
		col := hex(kinds[e.k].accent)
		c.fill(x0-1, 1, w+2, 1, ' ', style(col, hudBg))
		c.text(x0, 1, name, style(col, hudBg))
		g.bar(c, x0+width(name)+1, 1, w-width(name)-1, e.hp/e.max, col, hex(0x1d1020))
		return
	}
}

// ── cards ────────────────────────────────────────────────────────────────

// cardRect is where card i is drawn, for the mouse as much as the picture.
func (g *game) cardRect(c canvas, i int) (x, y, w, h int) {
	w = 28
	if room := (c.w - 4) / 3; room-2 < w {
		w = room - 2
	}
	h = 13
	gap := 2
	total := 3*w + 2*gap
	x = (c.w-total)/2 + i*(w+gap)
	y = (c.h - h) / 2
	return
}

func (g *game) cardAt(c canvas, mx, my int) int {
	for i := 0; i < g.nCards; i++ {
		x, y, w, h := g.cardRect(c, i)
		if mx >= x && mx < x+w && my >= y && my < y+h {
			return i
		}
	}
	return -1
}

func (g *game) drawCards(c canvas) {
	_, y0, _, _ := g.cardRect(c, 0)
	title := style(hudText, hex(0x05060a))
	c.fill(0, y0-3, c.w, 1, ' ', title)
	head := "LEVEL UP: CHOOSE AN AUGMENT"
	c.centered(y0-3, head, style(hudWarn, hex(0x05060a)))
	for i := 0; i < g.nCards; i++ {
		g.drawCard(c, i)
	}
	hint := "click a card, or press 1 2 3"
	if g.pending > 0 {
		hint = "click a card, or press 1 2 3   ·   more cards waiting"
	}
	_, y, _, h := g.cardRect(c, 0)
	c.fill(0, y+h+1, c.w, 1, ' ', title)
	c.centered(y+h+1, hint, style(hudDim, hex(0x05060a)))
}

func (g *game) drawCard(c canvas, i int) {
	x, y, w, h := g.cardRect(c, i)
	a := g.cards[i]
	au := &augs[a]
	rc := hex(rarityCol[au.rar])
	bg := hex(0x0d1020)
	if i == g.cardHov {
		bg = hex(0x172040)
	}
	st := style(hudText, bg)
	c.fill(x, y, w, h, ' ', st)
	border := rc.mul(0.6)
	if i == g.cardHov {
		border = rc
	}
	c.box(x, y, w, h, style(border, bg))
	c.text(x+2, y+1, rarityName[au.rar], style(rc, bg))
	c.text(x+w-2-width(au.tag), y+1, au.tag, style(hudDim, bg))
	name := au.name
	c.text(x+(w-width(name))/2, y+3, name, style(rc.mix(rgb{1, 1, 1}, 0.3), bg))
	c.text(x+2, y+5, au.what, st)
	if au.what2 != "" {
		c.text(x+2, y+6, au.what2, style(hudDim, bg))
	}
	r := g.rank[a]
	yy := y + h - 4
	if r < au.max {
		xx := x + 2
		for k := 0; k < au.max; k++ {
			pip, col := '○', hudDim
			if k < r {
				pip, col = '●', rc
			} else if k == r {
				pip, col = '◉', hudWarn
			}
			c.set(xx, yy, pip, style(col, bg))
			xx += 2
		}
		xx = c.text(xx, yy, "rank ", style(hudDim, bg))
		c.number(xx, yy, r+1, st)
	} else {
		xx := c.text(x+2, yy, "OVERCLOCK ", style(hudWarn, bg))
		xx = c.text(xx, yy, "x", style(hudWarn, bg))
		xx = c.number(xx, yy, g.over[a]+1, style(hudWarn, bg))
		c.text(xx+1, yy, "+12%", style(hudGood, bg))
	}
	k := style(hudDim, bg)
	c.text(x+(w-5)/2, y+h-2, "[ ]", k)
	c.set(x+(w-5)/2+1, y+h-2, rune('1'+i), style(hudWarn, bg))
}

// ── screens ──────────────────────────────────────────────────────────────

func (g *game) drawTitle(c canvas) {
	size := widgets.BigTextHalfHeight
	tw, th := 96, 4
	if c.w < 100 {
		size, tw, th = widgets.BigTextQuadrant, 48, 4
	}
	y := c.h/2 - 9
	if y < 2 {
		y = 2
	}
	big := widgets.BigText{Text: "MARK & SWEEP", Size: size, Style: cell.Style{Fg: hudPink.color()}}
	big.Draw(cell.Context{Area: cell.Rect{X: uint16(max(0, (c.w-tw)/2)), Y: uint16(y), Width: uint16(min(tw, c.w)), Height: uint16(th)}}, c.b)
	y += th + 1
	sub := style(hudText, hex(0x03040a))
	c.centered(y, "you are the garbage collector. the frameworks have come for the heap.", sub)
	y += 2
	items := [...]string{"SWEEP", "SOUND", "QUIT"}
	for i, it := range items {
		st := style(hudDim, hex(0x03040a))
		mark := "  "
		if i == g.menu {
			st = style(hudWarn, hex(0x03040a))
			mark = "▶ "
		}
		x := c.w/2 - 8
		x = c.text(x, y+i, mark, st)
		x = c.text(x, y+i, it, st)
		if i == 1 {
			x++
			for k := 0; k < 10; k++ {
				r := '·'
				if k < g.volume {
					r = '▮'
				}
				c.set(x+k, y+i, r, st)
			}
			if g.noSound != "" {
				c.text(x+11, y+i, g.noSound, style(hudDim, hex(0x03040a)))
			}
		}
	}
	y += len(items) + 1
	// Who is coming, each in its own colour.
	foes := [...]struct {
		name string
		col  uint32
	}{{"Bubble Tea", 0xff5fd2}, {"Ratatui", 0xf74c00}, {"tview", 0x3d7bff}, {"Ink", 0x61dafb}, {"ncurses", 0x33ff66}}
	total := 0
	for _, f := range foes {
		total += width(f.name) + 3
	}
	x := (c.w - total + 3) / 2
	for i, f := range foes {
		x = c.text(x, y, f.name, style(hex(f.col), hex(0x03040a)))
		if i < len(foes)-1 {
			x = c.text(x, y, " · ", style(hudDim, hex(0x03040a)))
		}
	}
	y += 2
	if g.best.runs > 0 {
		dim := style(hudDim, hex(0x03040a))
		val := style(hudText, hex(0x03040a))
		x := c.w/2 - 22
		x = c.text(x, y, "BEST  ", dim)
		x = c.clock(x, y, g.best.secs, val)
		x = c.text(x, y, "  level ", dim)
		x = c.number(x, y, g.best.level, val)
		x = c.text(x, y, "  swept ", dim)
		c.number(x, y, g.best.kills, val)
		y++
	}
	c.centered(y+1, "click or Enter to start · W/S choose · Esc quits", style(hudDim, hex(0x03040a)))
}

func (g *game) drawPause(c canvas) {
	y := c.h/2 - 3
	st := style(hudText, hex(0x05060a))
	c.fill(0, y-1, c.w, 7, ' ', st)
	c.centered(y, "SIGSTOP", style(hudWarn, hex(0x05060a)))
	items := [...]string{"RESUME", "SOUND", "QUIT"}
	for i, it := range items {
		s := style(hudDim, hex(0x05060a))
		mark := "  "
		if i == g.pause {
			s = style(hudWarn, hex(0x05060a))
			mark = "▶ "
		}
		x := c.text(c.w/2-6, y+2+i, mark, s)
		x = c.text(x, y+2+i, it, s)
		if i == 1 {
			x++
			for k := 0; k < 10; k++ {
				r := '·'
				if k < g.volume {
					r = '▮'
				}
				c.set(x+k, y+2+i, r, s)
			}
		}
	}
}

func (g *game) drawDead(c canvas) {
	y := c.h/2 - 6
	bg := hex(0x12040a)
	st := style(hudText, bg)
	dim := style(hudDim, bg)
	c.fill(0, y-1, c.w, 14, ' ', st)
	c.centered(y, "fatal error: out of memory", style(hudBad, bg))
	c.centered(y+1, "the heap was overrun", dim)
	rows := [...]struct {
		label string
		val   int
		best  int
		clock bool
	}{
		{"survived", int(g.runTime), g.best.secs, true},
		{"level", g.level, g.best.level, false},
		{"swept", g.kills, g.best.kills, false},
		{"most shots alive", g.peakShot, g.best.peak, false},
		{"biggest hit", int(g.bigHit), g.best.bigHit, false},
	}
	for i, r := range rows {
		yy := y + 3 + i
		x := c.w/2 - 20
		c.text(x, yy, r.label, dim)
		x += 18
		if r.clock {
			x = c.clock(x, yy, r.val, st)
		} else {
			x = c.number(x, yy, r.val, st)
		}
		if r.val >= r.best && r.val > 0 {
			c.text(x+2, yy, "best", style(hudWarn, bg))
		}
	}
	c.centered(y+10, "click or R to run again · Esc quits", dim)
}
