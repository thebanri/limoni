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

// The HUD floats over the picture in its corners, on panels that darken
// what is under them rather than hide it: the level and the build top left,
// the sweep count and the pressure top right, health bottom left and the
// heap bottom right, their numbers big in the walls' pixel font.

// shadePanel darkens the cells of a rectangle.
func shadePanel(c canvas, x0, y0, w, h int) {
	tint := hex(0x060812)
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			p := c.at(x, y)
			if p == nil {
				continue
			}
			top, bot := halves(p)
			if p.Content == '▀' || p.Content == ' ' {
				setHalves(p, top.mix(tint, 0.72), bot.mix(tint, 0.72))
				continue
			}
			p.Style.Fg = fromColor(p.Style.Fg).mix(tint, 0.6).color()
			p.Style.Bg = bot.mix(tint, 0.72).color()
		}
	}
}

// px sets one pixel of the half-block picture: column x, half row py.
func (c canvas) px(x, py int, col rgb) {
	p := c.at(x, py/2)
	if p == nil {
		return
	}
	top, bot := halves(p)
	if py%2 == 0 {
		top = col
	} else {
		bot = col
	}
	setHalves(p, top, bot)
}

// big writes s in the 3×5 font, s pixels a font pixel, from column x and
// half row py, and returns the column after it.
func (c canvas) big(x, py int, s string, scale int, col rgb) int {
	for _, r := range s {
		x = c.bigRune(x, py, r, scale, col)
	}
	return x
}

// bigNumber writes n with big, from the left digit, without strconv.
func (c canvas) bigNumber(x, py, n, scale int, col rgb) int {
	d := digits(n)
	var buf [12]byte
	for i := d - 1; i >= 0; i-- {
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	for i := 0; i < d; i++ {
		x = c.bigRune(x, py, rune(buf[i]), scale, col)
	}
	return x
}

func (c canvas) bigRune(x, py int, r rune, scale int, col rgb) int {
	for gy := 0; gy < fontH*scale; gy++ {
		for gx := 0; gx < fontW*scale; gx++ {
			if fontPixel(r, gx/scale, gy/scale) {
				c.px(x+gx, py+gy, col)
			}
		}
	}
	return x + (fontW+1)*scale
}

// fgNumber writes n keeping the backgrounds under it.
func (c canvas) fgNumber(x, y, n int, col rgb) int {
	d := digits(n)
	cc := col.color()
	for i := d - 1; i >= 0; i-- {
		if p := c.at(x+i, y); p != nil {
			p.Content, p.Style.Fg = rune('0'+n%10), cc
		}
		n /= 10
	}
	return x + d
}

func (c canvas) fgClock(x, y, secs int, col rgb) int {
	x = c.fgNumber(x, y, secs/60, col)
	x = c.fg(x, y, ":", col)
	x = c.fgNumber(x, y, secs%60/10, col)
	return c.fgNumber(x, y, secs%10, col)
}

func (g *game) drawHUD(c canvas, v *view) {
	if g.phase == phTitle || c.w < 24 || c.h < 8 {
		return
	}
	w, h := c.w, c.h
	scale := 1
	if h >= 60 && w >= 150 {
		scale = 2
	}

	// Top left: the level, the bytes to the next one, the time.
	barW := min(28, max(6, w/6))
	pw := 4 + digits(g.level) + 1 + barW + 2 + 5
	shadePanel(c, 0, 0, pw, 1)
	x := c.fg(1, 0, "LV ", hudDim)
	x = c.fgNumber(x, 0, g.level, hudCyan) + 1
	g.bar(c, x, 0, barW, g.xp/need(g.level), hudGood, hex(0x14261a))
	x += barW + 1
	c.fgClock(x, 0, int(g.runTime), hudText)

	// Under it, the build: one augment a row.
	y := 1
	rows := max(0, h/2-3)
	for a := augID(0); a < nAug; a++ {
		if g.rank[a] == 0 {
			continue
		}
		if y > rows {
			shadePanel(c, 0, y, 7, 1)
			c.fg(1, y, "+ more", hudDim)
			break
		}
		name := augs[a].name
		n := width(name) + 2 + digits(g.rank[a])
		if g.over[a] > 0 {
			n += 1 + digits(g.over[a])
		}
		shadePanel(c, 0, y, n+1, 1)
		x := c.fg(1, y, name, hex(rarityCol[augs[a].rar]))
		x = c.fgNumber(x+1, y, g.rank[a], hudText)
		if g.over[a] > 0 {
			x = c.fg(x, y, "+", hudWarn)
			c.fgNumber(x, y, g.over[a], hudWarn)
		}
		y++
	}

	// Top right: what has been swept, and the pressure.
	p := g.pressure()
	rw := 6 + digits(g.kills) + 3 + 9 + 9
	rx := w - rw
	shadePanel(c, rx, 0, rw, 1)
	x = c.fg(rx+1, 0, "SWEPT ", hudDim)
	x = c.fgNumber(x, 0, g.kills, hudText) + 3
	x = c.fg(x, 0, "PRESSURE ", hudDim)
	if p <= 1 {
		x = c.fg(x, 0, "in ", hudDim)
		c.fgClock(x, 0, max(0, 600-int(g.runTime)), hudText)
	} else {
		col := hudWarn
		if p > 4 {
			col = hudBad
		}
		x = c.fg(x, 0, "x", col)
		x = c.fgNumber(x, 0, int(p), col)
		x = c.fg(x, 0, ".", col)
		c.fgNumber(x, 0, int(p*10)%10, col)
	}

	if g.bossLive {
		g.drawBossBar(c)
	}
	if g.msgT > 0 {
		mw := width(g.message) + 4
		shadePanel(c, (w-mw)/2, h/4, mw, 1)
		c.fg((w-mw)/2+2, h/4, g.message, hudPink)
	}

	// Bottom left: health, its number big.
	numRows := (fontH*scale + 2) / 2
	ph := numRows + 1
	py := h - ph
	hpCol := hudGood
	if g.hp < g.st.maxHP*0.3 {
		hpCol = hudBad
	}
	hpW := 3 + 4*scale*digits(int(math.Ceil(g.st.maxHP))) + 2 + 1 + digits(int(g.st.maxHP)) + 1
	shadePanel(c, 0, py, hpW, ph)
	x = c.bigNumber(2, 2*py+1, int(math.Ceil(g.hp)), scale, hpCol)
	c.fg(x+1, py, "HP", hudDim)
	x = c.fg(x+1, py+numRows-1, "/", hudDim)
	c.fgNumber(x, py+numRows-1, int(g.st.maxHP), hudDim)
	g.bar(c, 1, h-1, hpW-2, g.hp/g.st.maxHP, hpCol, hex(0x2a1418))

	// Bottom right: the heap.
	amW := max(min(g.st.mag, 24)+2, 3+4*scale*max(2, digits(g.st.mag))+2+5+1)
	ax := w - amW
	shadePanel(c, ax, py, amW, ph)
	if g.reloadT > 0 && g.st.reload > 0 {
		x = c.big(ax+2, 2*py+1, "GC", scale, hudCyan)
		c.fg(x+1, py, "HEAP", hudDim)
		c.fg(x+1, py+numRows-1, "pause", hudDim)
		g.bar(c, ax+1, h-1, amW-2, 1-g.reloadT/g.st.reload, hudCyan, hex(0x10202a))
		return
	}
	col := hudText
	switch {
	case g.ammo == 0:
		col = hudBad
	case g.ammo*4 <= g.st.mag:
		col = hudWarn
	}
	x = c.bigNumber(ax+2, 2*py+1, g.ammo, scale, col)
	c.fg(x+1, py, "HEAP", hudDim)
	x = c.fg(x+1, py+numRows-1, "/", hudDim)
	c.fgNumber(x, py+numRows-1, g.st.mag, hudDim)
	pips := min(g.st.mag, amW-2)
	on := int(math.Ceil(float64(g.ammo) / float64(g.st.mag) * float64(pips)))
	for i := 0; i < pips; i++ {
		r, pc := "▮", hudWarn
		if i >= on {
			r, pc = "▯", hudDim
		}
		c.fg(ax+1+i, h-1, r, pc)
	}

	// The keys, for the first seconds of a first run.
	if g.runTime < 12 && g.best.runs == 0 {
		hint := "WASD move · mouse aim · click sweep · R collect · Space dash · Esc pause"
		if hw := width(hint) + 2; hw < w-hpW-amW-2 {
			shadePanel(c, (w-hw)/2, h-1, hw, 1)
			c.fg((w-hw)/2+1, h-1, hint, hudDim)
		}
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
		shadePanel(c, x0-1, 2, w+2, 1)
		c.fg(x0, 2, name, col)
		g.bar(c, x0+width(name)+1, 2, w-width(name)-1, e.hp/e.max, col, hex(0x1d1020))
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
