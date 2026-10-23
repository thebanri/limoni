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
	if g.onTitle() || c.w < 24 || c.h < 8 {
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

	// The build: in a panel on the right when there is room, else listed
	// under the level.
	y := 1
	rows := max(0, h/2-3)
	if g.drawSlots(c) {
		rows = -1
	}
	for _, act := range [2]bool{true, false} {
		if g.holds(act) == 0 {
			continue
		}
		if y > rows {
			break
		}
		label := "PASSIVE "
		if act {
			label = "ACTIVE "
		}
		shadePanel(c, 0, y, width(label)+5, 1)
		x := c.fg(1, y, label, hudDim)
		x = c.fgNumber(x, y, g.holds(act), hudCyan)
		c.fg(x, y, "/6", hudDim)
		y++
		y = g.drawBuild(c, y, rows, act)
	}
	// Top right: the map, what has been swept, and what comes next: the
	// warden, the pressure, or the pressure itself.
	p := g.pressure()
	next, at := "PRESSURE ", g.pressureStart()
	if !g.endless && g.stageT < wardenAt {
		next, at = "WARDEN ", wardenAt
	}
	rw := 1 + width(cur.name) + 3 + 6 + digits(g.kills) + 3 + width(next) + 9
	if !g.endless {
		rw += 4
	}
	rx := w - rw
	shadePanel(c, rx, 0, rw, 1)
	x = c.fg(rx+1, 0, cur.name, hudPink)
	if !g.endless {
		x = c.fgNumber(x+1, 0, g.stage+1, hudDim)
		x = c.fg(x, 0, "/3", hudDim)
	}
	x = c.fg(x+3, 0, "SWEPT ", hudDim)
	x = c.fgNumber(x, 0, g.kills, hudText) + 3
	x = c.fg(x, 0, next, hudDim)
	if p <= 1 {
		x = c.fg(x, 0, "in ", hudDim)
		c.fgClock(x, 0, max(0, int(at-g.stageT)), hudText)
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
	if g.lockHint && g.phase == phPlay {
		const lock = "click to take the mouse"
		lw := width(lock) + 4
		shadePanel(c, (w-lw)/2, h/3, lw, 1)
		c.fg((w-lw)/2+2, h/3, lock, hudCyan)
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

	// Over it, the dashes ready, and what hit last.
	dw := 6 + 2*g.st.dashN
	shadePanel(c, 0, py-1, dw, 1)
	x = c.fg(1, py-1, "DASH ", hudDim)
	for i := 0; i < g.st.dashN; i++ {
		r, dc := "◆", hex(elementCol[elDash])
		if i >= g.dashN {
			r, dc = "◇", hudDim
		}
		x = c.fg(x, py-1, r, dc) + 1
	}
	if g.hurtShow > 0 && g.hurtHow != "" {
		name := kinds[g.hurtBy].name
		hw := 2 + digits(int(g.hurtAmt+0.5)) + 1 + width(name) + 1 + width(g.hurtHow) + 1
		shadePanel(c, 0, py-2, hw, 1)
		x = c.fg(1, py-2, "-", hudBad)
		x = c.fgNumber(x, py-2, int(g.hurtAmt+0.5), hudBad) + 1
		x = c.fg(x, py-2, name, hex(kinds[g.hurtBy].accent)) + 1
		c.fg(x, py-2, g.hurtHow, hudText)
	}

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
		hint := "WASD move · mouse aim · click sweep · R collect · Space dash (2) · Esc pause"
		if hw := width(hint) + 2; hw < w-hpW-amW-2 {
			shadePanel(c, (w-hw)/2, h-1, hw, 1)
			c.fg((w-hw)/2+1, h-1, hint, hudDim)
		}
	}
}

const (
	slotsW   = 30 // the widest row: a long name, five pips, the mark
	slotsTop = 4  // under the boss bar and its warning
)

// drawSlots lists the build down the right edge, right-aligned and
// unframed: a dim heading for the actives and one for the passives, then
// a row for each held: its name, its ranks, and a mark of its element's
// colour — an active's brightening as it readies to act. It reports false,
// drawing nothing, when the window is too small or the cards would cover
// it (the build is then listed under the level instead).
func (g *game) drawSlots(c canvas) bool {
	if c.w < 100 || c.h < slotsTop+2*(slots+1)+6 {
		return false
	}
	if g.phase == phCards {
		if x, _, w, _ := g.cardRect(c, 2); x+w+1 > c.w-slotsW {
			return false
		}
	}
	y := slotsTop
	for _, act := range [2]bool{true, false} {
		n := g.holds(act)
		if n == 0 {
			continue
		}
		label := "PASSIVE "
		if act {
			label = "ACTIVE "
		}
		hw := width(label) + 3
		shadePanel(c, c.w-hw-1, y, hw+1, 1)
		x := c.fg(c.w-hw, y, label, hudDim)
		x = c.fgNumber(x, y, n, hudCyan)
		c.fg(x, y, "/6", hudDim)
		y++
		for a := augID(0); a < nAug; a++ {
			if g.rank[a] > 0 && augs[a].act == act {
				g.slotRow(c, y, a)
				y++
			}
		}
		y++
	}
	return true
}

// slotRow is one held augment, ending at the right edge.
func (g *game) slotRow(c canvas, y int, a augID) {
	au := &augs[a]
	el := hex(elementCol[au.el])
	if au.el == elNone {
		el = hex(rarityCol[au.rar])
	}
	w := width(au.name) + 1 + au.max + 2
	x := c.w - w - 1
	shadePanel(c, x-1, y, w+2, 1)
	x = c.fg(x, y, au.name, hex(rarityCol[au.rar]).mix(rgb{1, 1, 1}, 0.5)) + 1
	for k := 0; k < au.max; k++ {
		pip, pc := "•", hex(0x39405a)
		if k < g.rank[a] {
			pc = el
		}
		x = c.fg(x, y, pip, pc)
	}
	// The mark: an active's glows up as it readies, and flashes as it acts.
	mark, mc := "■", el.mul(0.8)
	if au.act {
		mark = "◆"
		ready := 1.0
		if g.actP[a] > 0 {
			ready = 1 - math.Max(0, g.actT[a])/g.actP[a]
		}
		mc = el.mul(float32(0.3 + 0.7*ready*ready))
		if g.actP[a]-g.actT[a] < 0.12 {
			mc = el.mix(rgb{1, 1, 1}, 0.6)
		}
	}
	c.fg(x+1, y, mark, mc)
}

// drawBuild lists the actives or the passives the build holds from row y,
// and returns the row after them.
func (g *game) drawBuild(c canvas, y, rows int, act bool) int {
	for a := augID(0); a < nAug; a++ {
		if g.rank[a] == 0 || augs[a].act != act {
			continue
		}
		if y > rows {
			shadePanel(c, 0, y, 7, 1)
			c.fg(1, y, "+ more", hudDim)
			return y + 1
		}
		name := augs[a].name
		n := width(name) + 3 + digits(g.rank[a])
		shadePanel(c, 0, y, n+1, 1)
		x := c.fg(2, y, name, hex(rarityCol[augs[a].rar]))
		c.fgNumber(x+1, y, g.rank[a], hudText)
		y++
	}
	return y
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
		if !e.alive || !kinds[e.k].boss || e.decoy {
			continue
		}
		name := kinds[e.k].name
		w := c.w / 2
		x0 := (c.w - w) / 2
		col := hex(kinds[e.k].accent)
		shadePanel(c, x0-1, 2, w+2, 1)
		c.fg(x0, 2, name, col)
		g.bar(c, x0+width(name)+1, 2, w-width(name)-1, e.hp/e.max, col, hex(0x1d1020))
		what, warn := "", true
		switch {
		case e.under > 0:
			what, warn = "UNDERGROUND: it comes up where the floor glows", false
		case e.guards > 0:
			what, warn = "MODAL OPEN: break [Y] and [N] to hurt her", false
		case g.pullT > 0 && g.pullBy == i:
			what = "STRAW: walk away from it, or dash"
		}
		if e.windT > 0 {
			// Its attack, named while it winds up.
			what, warn = "CLAW SALVO: get behind cover", true
			switch {
			case e.sig && e.k == kBobaPrime:
				what = "STRAW: it is going to pull you in"
			case e.sig && e.k == kFerris:
				what = "BURROW: it goes under the floor"
			case e.sig && e.k == kQueen:
				what = "MODAL: break her buttons, or she mends"
			case e.sig && e.k == kTextual:
				what = "CSS GRID: step across, then step back"
			case e.k == kQueen:
				what = "DIALOG WALL: find the gap in it"
			case e.k == kReconciler && e.slam:
				what = "RE-RENDER: it lands by you, and leaves a copy"
			case e.k == kReconciler:
				what = "PROPS DRILL: they follow you, dash"
			case e.k == kTextual && e.slam:
				what = "WIDGET STORM: prompts and cursors"
			case e.k == kTextual:
				what = "CSS CASCADE: leave the marked floor"
			case e.slam:
				what = "CLAW SLAM: dash through the wave"
			case e.k == kBobaPrime:
				what = "PEARL RING: find the gap"
			}
		}
		if what != "" {
			ww := width(what) + 2
			shadePanel(c, (c.w-ww)/2, 3, ww, 1)
			wc := hudCyan
			if warn {
				wc = hudWarn
				if int(g.clock*8)%2 == 0 {
					wc = hudBad
				}
			}
			c.fg((c.w-ww)/2+1, 3, what, wc)
		}
		return
	}
}

// ── cards ────────────────────────────────────────────────────────────────

// cardRect is where card i is drawn, for the mouse as much as the picture.
// A window with the room gets tall cards with their pictures.
func (g *game) cardRect(c canvas, i int) (x, y, w, h int) {
	w = 28
	if room := (c.w - 4) / 3; room-2 < w {
		w = room - 2
	}
	h = 13
	if c.h >= 28 && w >= iconW+6 {
		h = 19
	}
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
	// The slots, under it: six actives, six passives.
	slotsW := 28
	x := (c.w - slotsW) / 2
	x = c.text(x, y0-2, "ACTIVE ", style(hudDim, hex(0x05060a)))
	x = c.number(x, y0-2, g.holds(true), style(hudCyan, hex(0x05060a)))
	x = c.text(x, y0-2, "/6   PASSIVE ", style(hudDim, hex(0x05060a)))
	x = c.number(x, y0-2, g.holds(false), style(hudCyan, hex(0x05060a)))
	c.text(x, y0-2, "/6", style(hudDim, hex(0x05060a)))
	for i := 0; i < g.nCards; i++ {
		g.drawCard(c, i)
	}
	hint := "click a card, or press 1 2 3"
	if g.locked {
		hint = "move the mouse to choose, click to take, or press 1 2 3"
	}
	if g.pending > 0 {
		hint = "click a card, or press 1 2 3   ·   more cards waiting"
	}
	_, y, _, h := g.cardRect(c, 0)
	c.fill(0, y+h+1, c.w, 1, ' ', title)
	c.centered(y+h+1, hint, style(hudDim, hex(0x05060a)))
}

// drawCard draws a card in its rarity's colour: a tinted face, a frame
// that brightens under the mouse (a legendary one has a glint running
// round it), its picture on a glow, and what it does.
func (g *game) drawCard(c canvas, i int) {
	x, y, w, h := g.cardRect(c, i)
	a := g.cards[i]
	au := &augs[a]
	rc := hex(rarityCol[au.rar])
	hov := i == g.cardHov
	tintK := float32(0.1)
	if hov {
		tintK = 0.2
	}
	bg := hex(0x0b0e1a).mix(rc, tintK)
	st := style(hudText, bg)
	c.fill(x, y, w, h, ' ', st)
	border := rc.mul(0.55)
	if hov {
		border = rc
	}
	c.box(x, y, w, h, style(border, bg))
	if au.rar == legendary {
		// A glint running round the frame.
		per := 2 * (w + h)
		at := int(g.clock*24) % per
		for d := -2; d <= 2; d++ {
			k := (at + d + per) % per
			bx, by := x+k, y
			switch {
			case k >= w+h+w:
				bx, by = x, y+h-1-(k-w-h-w)
			case k >= w+h:
				bx, by = x+w-1-(k-w-h), y+h-1
			case k >= w:
				bx, by = x+w-1, y+(k-w)
			}
			if p := c.at(bx, by); p != nil {
				p.Style.Fg = rc.mix(rgb{1, 1, 1}, 0.7-0.2*float32(abs(d))).color()
			}
		}
	}
	c.text(x+2, y+1, rarityName[au.rar], style(rc, bg))
	c.text(x+w-2-width(au.tag), y+1, au.tag, style(hex(elementCol[au.el]), bg))
	yy := y + 3
	if h > 13 {
		// The picture, on a glow of the rarity's colour.
		top, rows := 2*(y+2), 7
		for py := top; py < top+2*rows; py++ {
			for px := x + 2; px < x+w-2; px++ {
				dx := float64(px-(x+w/2)) / float64(w/2-2)
				dy := float64(py-(top+rows)) / float64(rows)
				k := float32(math.Max(0, 1-math.Sqrt(dx*dx+dy*dy)))
				c.px(px, py, bg.mix(rc, k*(0.15+tintK)))
			}
		}
		ic := icons[a]
		ix, iy := x+(w-ic.w)/2, top+1
		for py := 0; py < ic.h; py++ {
			for px := 0; px < ic.w; px++ {
				if k := py*ic.w + px; ic.on[k] {
					c.px(ix+px, iy+py, ic.px[k])
				}
			}
		}
		yy = y + 10
	}
	name := au.name
	c.text(x+(w-width(name))/2, yy, name, style(rc.mix(rgb{1, 1, 1}, 0.3), bg))
	c.text(x+2, yy+2, au.what, st)
	if au.what2 != "" {
		c.text(x+2, yy+3, au.what2, style(hudDim, bg))
	}
	if rc, ok := g.comboHint(a); ok && w >= 20 {
		// A recipe this card goes on with: its name, and how far it gets.
		n := g.has(rc)
		if g.rank[a] == 0 {
			n++
		}
		hw := 2 + width(rc.name) + 1 + digits(n) + 1 + digits(len(rc.cards))
		hx := c.text(x+(w-hw)/2, yy+4, "★ ", style(hudWarn, bg))
		hx = c.text(hx, yy+4, rc.name, style(hudWarn, bg))
		hx = c.number(hx+1, yy+4, n, style(hudGood, bg))
		hx = c.text(hx, yy+4, "/", style(hudDim, bg))
		c.number(hx, yy+4, len(rc.cards), style(hudDim, bg))
	}
	r := g.rank[a]
	ry := y + h - 4
	xx := x + 2
	for k := 0; k < au.max; k++ {
		pip, col := '○', hudDim
		if k < r {
			pip, col = '●', rc
		} else if k == r {
			pip, col = '◉', hudWarn
		}
		c.set(xx, ry, pip, style(col, bg))
		xx += 2
	}
	xx = c.text(xx, ry, "rank ", style(hudDim, bg))
	c.number(xx, ry, r+1, st)
	kind, kc := "PASSIVE", hudDim
	switch {
	case au.act && g.rank[a] == 0:
		kind, kc = "ACTIVE · NEW", hudCyan
	case au.act:
		kind, kc = "ACTIVE", hudCyan
	case g.rank[a] == 0:
		kind = "PASSIVE · NEW"
	}
	c.text(x+(w-width(kind))/2, y+h-3, kind, style(kc, bg))
	k := style(hudDim, bg)
	c.text(x+(w-5)/2, y+h-2, "[ ]", k)
	c.set(x+(w-5)/2+1, y+h-2, rune('1'+i), style(hudWarn, bg))
}

// ── screens ──────────────────────────────────────────────────────────────
//
// The menus share one look: the picture behind veiled, a card with a
// heading in the pixel font, and the choices as buttons, the chosen one a
// lit bar.

const (
	cardBg     = 0x0a0d18
	cardBorder = 0x2a3350
	buttonW    = 26
)

// veil darkens the whole screen towards the night by k.
func veil(c canvas, k float32, tint rgb) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			p := c.at(x, y)
			if p == nil {
				continue
			}
			top, bot := halves(p)
			if p.Content == '▀' || p.Content == ' ' {
				setHalves(p, top.mix(tint, k), bot.mix(tint, k))
				continue
			}
			p.Style.Fg = fromColor(p.Style.Fg).mix(tint, k).color()
			p.Style.Bg = bot.mix(tint, k).color()
		}
	}
}

var (
	titleItems = [...]string{"SWEEP: THREE MAPS", "ENDLESS: /dev/zero", "RECIPES", "SETTINGS", "QUIT"}
	pauseItems = [...]string{"RESUME", "RECIPES", "SETTINGS", "QUIT"}
)

// card draws a framed card w by h at (x, y), in the colour of its frame.
func card(c canvas, x, y, w, h int, frame rgb) {
	c.fill(x, y, w, h, ' ', style(hudText, hex(cardBg)))
	c.box(x, y, w, h, style(frame, hex(cardBg)))
}

// buttons draws a menu centred at row y: the chosen item a lit
// bar with its marks, the others dim. The mouse finds it by menuY.
func (g *game) buttons(c canvas, y int, items []string, sel int, bg rgb) {
	g.menuY = y
	x0 := (c.w - buttonW) / 2
	for i, it := range items {
		yy := y + i
		st := style(hudDim, bg)
		if i == sel {
			st = style(hudWarn, bg.mix(hudWarn, 0.16))
			c.fill(x0, yy, buttonW, 1, ' ', st)
			c.set(x0, yy, '▐', style(hudWarn, bg))
			c.set(x0+buttonW-1, yy, '▌', style(hudWarn, bg))
		}
		c.text(x0+3, yy, it, st)
	}
}

// parade draws the five frameworks' creatures in a row, rows tall in
// pixels, with their framework under each; it returns the row after them.
func (g *game) parade(c canvas, y, px int) int {
	foes := [...]struct {
		k    kindID
		name string
	}{{kBoba, "Bubble Tea"}, {kCrab, "Ratatui"}, {kModal, "tview"}, {kJSX, "Ink"}, {kElder, "ncurses"}}
	step := 16
	x0 := c.w/2 - step*len(foes)/2
	for i, f := range foes {
		kd := &kinds[f.k]
		s := kd.spr[int(g.clock*5+float64(i))%len(kd.spr)]
		hPx := px
		wPx := hPx * s.w / s.h
		cx := x0 + i*step + step/2
		left, top := cx-wPx/2, 2*y+(px-hPx)
		hop := int(math.Abs(math.Sin(g.clock*3+float64(i))) * 2)
		for yy := 0; yy < hPx; yy++ {
			for xx := 0; xx < wPx; xx++ {
				k := (yy*s.h/hPx)*s.w + xx*s.w/wPx
				if s.on[k] {
					c.px(left+xx, top+yy-hop, s.px[k])
				}
			}
		}
		c.text(cx-width(f.name)/2, y+px/2+1, f.name, style(hex(kd.accent), hex(0x03040a)))
	}
	return y + px/2 + 2
}

func (g *game) drawTitle(c canvas) {
	bg := hex(0x03040a)
	veil(c, 0.35, bg)
	size := widgets.BigTextHalfHeight
	tw, th := 88, 4 // eight columns a letter
	if c.w < 92 {
		size, tw, th = widgets.BigTextQuadrant, 44, 4
	}
	// What fits: the creatures' parade on a tall screen, the maps as cards
	// on a wide one, as lines on a narrower one.
	cards := c.w >= 4*mapCardW+3*2 && c.h >= 30
	need := th + 4 + len(titleItems) + 3 + 3
	switch {
	case cards:
		need += mapCardH + 1
	case c.w >= 56:
		need += len(arenas) + 1
	}
	tall := need+9 <= c.h
	if tall {
		need += 9
	}
	y := max(1, (c.h-need)/2)
	// The name, over its shadow.
	tx := max(0, (c.w-tw)/2)
	shadow := widgets.BigText{Text: "LEMON SWEEP", Size: size, Style: cell.Style{Fg: hex(0x3a2e08).color()}}
	shadow.Draw(cell.Context{Area: cell.Rect{X: uint16(tx + 1), Y: uint16(y + 1), Width: uint16(min(tw, c.w-tx-1)), Height: uint16(th)}}, c.b)
	big := widgets.BigText{Text: "LEMON SWEEP", Size: size, Style: cell.Style{Fg: hex(0xffe14a).color()}}
	big.Draw(cell.Context{Area: cell.Rect{X: uint16(tx), Y: uint16(y), Width: uint16(min(tw, c.w)), Height: uint16(th)}}, c.b)
	y += th + 2
	c.centered(y, "you are the garbage collector. the frameworks have come for the heap.", style(hudText, bg))
	y += 2
	if tall {
		y = g.parade(c, y, 12) + 1
	}
	g.buttons(c, y, titleItems[:], g.menu, bg)
	y += len(titleItems)
	c.centered(y, titleAbout[g.menu], style(hudCyan.mul(0.8), bg))
	y += 2
	switch {
	case cards:
		x0 := (c.w - (4*mapCardW + 3*2)) / 2
		for i := range arenas {
			g.mapCard(c, x0+i*(mapCardW+2), y, i)
		}
		y += mapCardH + 1
	case c.w >= 56:
		for i := range arenas {
			g.mapLine(c, c.w/2-26, y, i)
			y++
		}
		y++
	}
	if b := &g.best.bests; b.runs > 0 {
		dim, val := style(hudDim, bg), style(hudWarn, bg)
		x := c.w/2 - 22
		x = c.text(x, y, "LONGEST RUN ", dim)
		x = c.clock(x, y, b.secs, val)
		x = c.text(x, y, "  ·  LV ", dim)
		x = c.number(x, y, b.level, val)
		x = c.text(x, y, "  ·  ", dim)
		x = c.number(x, y, b.kills, val)
		x = c.text(x, y, " swept", dim)
		if won := g.best.maps[nStages-1].swept; won > 0 {
			x = c.text(x, y, "  ·  won ", dim)
			x = c.number(x, y, won, style(hudGood, bg))
			c.text(x, y, "×", dim)
		}
	}
	y += 2
	c.centered(y, "Enter or click to start · W/S choose · Esc quits", style(hudDim, bg))
	if g.hint != "" {
		c.centered(y+2, g.hint, style(hudCyan, bg))
	}
}

var titleAbout = [len(titleItems)]string{
	"three maps, ten minutes on each, then its warden",
	"one open map, no warden and no end: how long?",
	"which cards work together",
	"sound, and the crosshair",
	"back to the shell",
}

const mapCardW, mapCardH = 25, 7

// mapCard draws map i on the title, with what this player has done on it:
// lit when the chosen mode plays it.
func (g *game) mapCard(c canvas, x, y, i int) {
	a := &arenas[i]
	acc := hex(a.accent)
	bg := hex(a.floor).mix(hex(cardBg), 0.4)
	frame := hex(cardBorder)
	if endless := i == len(arenas)-1; g.menu == 0 && !endless || g.menu == 1 && endless {
		frame = acc
	}
	card(c, x, y, mapCardW, mapCardH, frame)
	c.fill(x+1, y+1, mapCardW-2, mapCardH-2, ' ', style(hudText, bg))
	// Its number and name on the top edge.
	tx := c.text(x+2, y, " ", style(acc, hex(cardBg)))
	if i < nStages {
		tx = c.number(tx, y, i+1, style(acc.mul(0.7), hex(cardBg)))
		tx = c.text(tx, y, " ", style(acc, hex(cardBg)))
	}
	tx = c.text(tx, y, a.name, style(acc, hex(cardBg)))
	c.text(tx, y, " ", style(acc, hex(cardBg)))
	c.text(x+2, y+1, a.about, style(hudDim, bg))
	dim, val := style(hudDim, bg), style(hudText, bg)
	vx := x + 15
	if i == len(arenas)-1 {
		e := &g.best.endless
		if e.runs == 0 {
			c.text(x+2, y+3, "not played yet", dim)
			return
		}
		c.text(x+2, y+3, "longest", dim)
		c.clock(vx, y+3, e.secs, style(hudWarn, bg))
		c.text(x+2, y+4, "level", dim)
		c.number(vx, y+4, e.level, val)
		c.text(x+2, y+5, "most swept", dim)
		c.number(vx, y+5, e.kills, val)
		return
	}
	m := &g.best.maps[i]
	if m.runs == 0 {
		c.text(x+2, y+3, "not reached yet", dim)
		return
	}
	c.text(x+2, y+3, "swept", dim)
	sx := c.number(vx, y+3, m.swept, style(hudGood, bg))
	sx = c.text(sx, y+3, "/", dim)
	c.number(sx, y+3, m.runs, val)
	c.text(x+2, y+4, "warden down", dim)
	if m.swept > 0 {
		c.clock(vx, y+4, m.fastest, style(hudWarn, bg))
	} else {
		c.text(vx, y+4, "never", dim)
	}
	c.text(x+2, y+5, "most swept", dim)
	c.number(vx, y+5, m.kills, val)
}

// mapLine is a map's card as one line, for a title too narrow for cards.
func (g *game) mapLine(c canvas, x, y, i int) {
	bg := hex(0x03040a)
	a := &arenas[i]
	dim, val := style(hudDim, bg), style(hudText, bg)
	c.text(x, y, a.name, style(hex(a.accent), bg))
	x += 14
	if i == len(arenas)-1 {
		e := &g.best.endless
		if e.runs == 0 {
			c.text(x, y, "not played yet", dim)
			return
		}
		x = c.text(x, y, "longest ", dim)
		x = c.clock(x, y, e.secs, style(hudWarn, bg))
		x = c.text(x, y, "  ·  LV ", dim)
		c.number(x, y, e.level, val)
		return
	}
	m := &g.best.maps[i]
	if m.runs == 0 {
		c.text(x, y, "not reached yet", dim)
		return
	}
	x = c.text(x, y, "swept ", dim)
	x = c.number(x, y, m.swept, style(hudGood, bg))
	x = c.text(x, y, "/", dim)
	x = c.number(x, y, m.runs, val)
	if m.swept > 0 {
		x = c.text(x, y, "  ·  warden down ", dim)
		c.clock(x, y, m.fastest, style(hudWarn, bg))
	}
}

func (g *game) drawPause(c canvas) {
	veil(c, 0.6, hex(0x03040a))
	w, h := 44, 16
	x, y := (c.w-w)/2, (c.h-h)/2
	card(c, x, y, w, h, hex(cardBorder))
	bg := hex(cardBg)
	c.big((c.w-27)/2, 2*(y+1)+1, "SIGSTOP", 1, hudWarn)
	c.centered(y+5, "the run is stopped, not freed", style(hudDim, bg))
	// Where the run stands: the map, and the run.
	dim, val := style(hudDim, bg), style(hudText, bg)
	mx := c.w/2 - 14
	mx = c.text(mx, y+6, cur.name, style(hex(cur.accent), bg))
	if g.endless {
		c.text(mx, y+6, "  ·  endless", dim)
	} else {
		mx = c.text(mx, y+6, "  ·  map ", dim)
		mx = c.number(mx, y+6, g.stage+1, val)
		c.text(mx, y+6, " of 3", dim)
	}
	sx := c.w/2 - 14
	sx = c.clock(sx, y+7, int(g.runTime), val)
	sx = c.text(sx, y+7, "  ·  LV ", dim)
	sx = c.number(sx, y+7, g.level, val)
	sx = c.text(sx, y+7, "  ·  ", dim)
	sx = c.number(sx, y+7, g.kills, val)
	c.text(sx, y+7, " swept", dim)
	g.buttons(c, y+9, pauseItems[:], g.pause, bg)
	c.centered(y+h-2, "Esc or P resumes", style(hex(cardBorder).mix(hudDim, 0.6), bg))
}

func (g *game) drawDead(c canvas) {
	veil(c, 0.55, hex(0x1a0208))
	w, h := 58, 17
	x, y := (c.w-w)/2, (c.h-h)/2
	frame := hudBad.mul(0.6)
	card(c, x, y, w, h, frame)
	bg := hex(cardBg)
	dim := style(hudDim, bg)
	head, headW, headCol := "OUT OF MEMORY", 51, hudBad
	if g.won {
		head, headW, headCol = "SWEPT CLEAN", 43, hudGood
	}
	if c.w >= w+2 {
		c.big((c.w-headW)/2, 2*(y+1)+1, head, 1, headCol)
	} else {
		c.centered(y+2, head, style(headCol, bg))
	}
	switch {
	case g.won:
		c.centered(y+5, "all three maps swept: the heap is yours", dim)
	case g.killedHow != "":
		by := kinds[g.killedBy].name
		tw := 10 + width(by) + 1 + width(g.killedHow)
		xx := (c.w - tw) / 2
		xx = c.text(xx, y+5, "killed by ", dim)
		xx = c.text(xx, y+5, by, style(hex(kinds[g.killedBy].accent), bg))
		c.text(xx+1, y+5, g.killedHow, dim)
	default:
		c.centered(y+5, "the heap was overrun", dim)
	}
	best := g.bests()
	maps := g.stage
	if g.won {
		maps = nStages
	}
	rows := [...]struct {
		label string
		val   int
		best  int
		clock bool
	}{
		{"survived", int(g.runTime), best.secs, true},
		{"level", g.level, best.level, false},
		{"swept", g.kills, best.kills, false},
		{"most shots alive", g.peakShot, best.peak, false},
		{"biggest hit", int(g.bigHit), best.bigHit, false},
		{"maps swept", maps, math.MaxInt, false},
	}
	for i, r := range rows {
		if g.endless && r.label == "maps swept" {
			continue
		}
		yy := y + 7 + i
		lx := c.w/2 - 18
		c.text(lx, yy, r.label, dim)
		vx := c.w/2 + 4
		st := style(hudText, bg)
		if r.clock {
			vx = c.clock(vx, yy, r.val, st)
		} else {
			vx = c.number(vx, yy, r.val, st)
		}
		if r.val >= r.best && r.val > 0 {
			c.text(vx+2, yy, " BEST ", style(hex(0x1a1408), hudWarn))
		}
	}
	c.centered(y+h-3, "R or click: run again", style(hudText, bg))
	c.centered(y+h-2, "Esc quits", style(hex(cardBorder).mix(hudDim, 0.6), bg))
}

// drawSettings is the settings card: a row for each setting, the chosen
// one lit with arrows to either side of its value, and beside them the
// crosshair as it is now, over a patch of wall.
func (g *game) drawSettings(c canvas) {
	veil(c, 0.6, hex(0x03040a))
	w, h := 66, 17
	x, y := (c.w-w)/2, (c.h-h)/2
	card(c, x, y, w, h, hex(cardBorder))
	bg := hex(cardBg)
	c.big(x+4, 2*(y+1)+1, "SETTINGS", 1, hudWarn)
	g.setY = y + 5
	for i, name := range settingRows {
		yy := g.setY + i
		chosen := i == g.setSel
		st := style(hudDim, bg)
		if chosen {
			st = style(hudWarn, bg.mix(hudWarn, 0.16))
			c.fill(x+2, yy, 40, 1, ' ', st)
			c.set(x+2, yy, '▐', style(hudWarn, bg))
		}
		c.text(x+5, yy, name, st)
		vx := x + 20
		if chosen && name != "BACK" {
			c.text(vx-2, yy, "‹", st)
		}
		val := ""
		switch name {
		case "SOUND":
			for k := 0; k < 10; k++ {
				r, col := '▮', hudCyan
				if k >= g.volume {
					r, col = '▯', hex(cardBorder)
				}
				if p := c.at(vx+k, yy); p != nil {
					p.Content, p.Style.Fg = r, col.color()
				}
			}
			vx += 10
		case "CROSSHAIR":
			val = reticleStyles[g.ret.style]
		case "COLOR":
			val = reticleColors[g.ret.color].name
			c.set(vx, yy, '■', style(hex(reticleColors[g.ret.color].col), fromColor(st.Bg)))
			vx += 2
		case "SIZE":
			val = reticleSizes[g.ret.size]
		case "OUTLINE":
			val = "OFF"
			if g.ret.outline {
				val = "ON"
			}
		}
		if val != "" {
			vx = c.text(vx, yy, val, style(hudText, fromColor(st.Bg)))
		}
		if chosen && name != "BACK" {
			c.text(max(vx+1, x+38), yy, "›", st)
		}
	}
	// The preview: a patch of wall, the crosshair on it.
	px, py, pw, ph := x+45, y+4, 17, 9
	for yy := py; yy < py+ph; yy++ {
		for xx := px; xx < px+pw; xx++ {
			k := float32(hash2(xx/3, yy)) * 0.15
			c.set(xx, yy, '▀', style(hex(0x1a2236).mix(hex(0x2c3a5c), k), hex(0x141a2a).mix(hex(0x24365e), k)))
		}
	}
	c.box(px-1, py-1, pw+2, ph+2, style(hex(cardBorder), bg))
	pv := view{c: c, w: c.w, h: c.h}
	drawReticle(&pv, px+pw/2, 2*(py+ph/2)+1, g.ret, 0, hex(reticleColors[g.ret.color].col))
	if g.noSound != "" {
		sx := c.text(x+5, y+h-3, "sound: ", style(hudDim, bg))
		c.text(sx, y+h-3, g.noSound, style(hudDim, bg))
	}
	c.centered(y+h-2, "W/S choose · A/D or click change · Esc back", style(hex(cardBorder).mix(hudDim, 0.6), bg))
}
