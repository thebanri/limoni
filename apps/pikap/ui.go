package main

import (
	"math"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// ui is the turntable on screen: the deck, what is playing, the controls
// and the list beside them. It runs on the interface goroutine only.
type ui struct {
	src   source
	np    nowPlaying
	scene scene
	gt    *glyphTable
	disc  discStyle
	glyph glyphStyle
	quit  bool
	local *localSource // set when pikap plays files itself

	// Cover art, loaded off the interface goroutine.
	artKey  string
	cover   *art
	artCh   chan loadedArt
	tint    rgb // eased towards the cover's
	rows    []rgb
	fallbck *art

	// The platter, for sources that do not run one.
	angle, speed float64
	armA         float64 // the arm's angle, eased towards where it belongs
	armLift      float64
	last         time.Time

	// Hands on things.
	handDown bool
	handRev  float64 // where the pointer was, as an angle about the centre
	armDrag  bool
	armDragR float64
	barDrag  bool
	volDrag  bool

	// Where things were drawn, for the mouse.
	dv       deckView
	haveDeck bool
	bar      span
	vol      span
	rpmSpan  span
	buttons  [5]span // shuffle, prev, play, next, repeat
	listTop  int
	listX    int
	listW    int
	listRows int
	listOff  int
	cursor   int
	cursorOn bool // the cursor was moved by keys, so it is shown
	help     bool
	rpm      float64 // how fast the record is shown turning, 1 = 33⅓

	// The layout the person chose.
	full       bool // only the record and what plays: no header, list or help
	listHidden bool // the list put away with Tab
	listWant   int  // the list's width as dragged; 0 until it is
	divDrag    bool // the list's edge is being dragged
	fullBtn    span
	lastW      int // the window's width at the last frame
}

type span struct{ x, y, w int }

func (s span) hit(x, y int) bool { return s.w > 0 && y == s.y && x >= s.x && x < s.x+s.w }

type loadedArt struct {
	key string
	a   *art
}

func newUI(src source) *ui {
	u := &ui{src: src, gt: newGlyphTable(), artCh: make(chan loadedArt, 4), armA: restAngle}
	if l, ok := src.(*localSource); ok {
		u.local = l
	}
	u.fallbck = generatedArt("pikap\x00the turntable")
	u.cover = u.fallbck
	u.tint = u.cover.tint
	u.armLift = 1
	u.glyph = halfBlocks
	u.rpm = 1
	return u
}

// tick advances everything that moves with time: the source's state, the
// platter, the arm, the colours.
func (u *ui) tick(now time.Time) {
	dt := 0.0
	if !u.last.IsZero() {
		dt = max(0, min(0.1, now.Sub(u.last).Seconds()))
	}
	u.last = now
	u.src.update(&u.np)

	if u.np.artKey != u.artKey {
		u.artKey = u.np.artKey
		if u.artKey != "" {
			load, key := u.src.artLoader(&u.np), u.artKey
			go func() { u.artCh <- loadedArt{key, load()} }()
		}
	}
	for {
		select {
		case la := <-u.artCh:
			if la.key == u.artKey && la.a != nil {
				u.cover = la.a
			}
			continue
		default:
		}
		break
	}

	// The platter. A source with its own physics says how fast it goes;
	// otherwise it spins up with play and coasts to a stop with pause.
	// Either way it is shown turning at the speed chosen with the rpm keys,
	// which is only what it looks like: the music is not touched. Under a
	// hand, the drag turns it.
	if u.np.hasPlatter {
		u.speed = u.np.speed
	} else if !u.handDown {
		if u.np.playing {
			u.speed = 1 + (u.speed-1)*math.Exp(-dt/spinUp)
		} else {
			u.speed *= math.Exp(-dt / spinDown)
			if f := 0.6 * dt; u.speed < f {
				u.speed = 0
			} else {
				u.speed -= f
			}
		}
	}
	if !u.handDown {
		u.angle += u.speed * u.rpm * rev33 * dt
	}

	// The arm: on the record where the needle is, or on its rest.
	want := restAngle
	lift := 1.0
	if u.np.title != "" && u.np.status == "" {
		prog := 0.0
		if u.np.length > 0 {
			prog = max(0, min(1, u.np.pos/u.np.length))
		}
		r := grooveOut - (grooveOut-grooveIn(u.disc))*prog
		if u.armDrag {
			r = u.armDragR
		}
		want = armAngle(needleAt(r))
		if !u.armDrag && (u.np.playing || u.speed > 0.05 || u.np.pos > 0.5) {
			lift = 0
		}
	}
	k := 1 - math.Exp(-dt/0.25)
	if u.armDrag {
		k = 1
	}
	u.armA += (want - u.armA) * k
	u.armLift += (lift - u.armLift) * (1 - math.Exp(-dt/0.15))

	u.tint = u.tint.mix(u.cover.tint, 1-math.Exp(-dt/0.6))
}

// Colours of the panel and its text, in linear light.
var (
	textCol  = lin(0xffffff)
	subCol   = lin(0xb3b3b3)
	dimCol   = lin(0x7c7c84)
	trackCol = lin(0x4d4d55)
)

// panelTop and panelBottom are the tint made dark, as a music app darkens
// the colour of the cover behind it.
func (u *ui) panelColours() (top, bottom, accent rgb) {
	t := u.tint
	l := max(1e-4, t.lum())
	top = t.scale(0.028 / l)
	if m := max(top.r, top.g, top.b); m > 0.2 {
		top = top.scale(0.2 / m)
	}
	bottom = top.scale(0.22)
	accent = t.scale(0.42 / l)
	if m := max(accent.r, accent.g, accent.b); m > 1 {
		accent = accent.scale(1 / m)
	}
	accent = accent.mix(textCol, 0.25)
	return
}

func style(fg, bg rgb) cell.Style { return cell.Style{Fg: fg.color(), Bg: bg.color()} }

// render draws the whole screen into b.
func (u *ui) render(b *buffer.Buffer) {
	W, H := int(b.Area.Width), int(b.Area.Height)
	if W*H == 0 || len(b.Content) < W*H {
		return
	}
	b.IsDirty = true
	top, bottom, accent := u.panelColours()
	if cap(u.rows) < H {
		u.rows = make([]rgb, H)
	}
	u.rows = u.rows[:H]
	for y := range u.rows {
		t := float64(y) / float64(max(1, H-1))
		u.rows[y] = top.mix(bottom, smoothstep(0, 1, t))
	}
	for y := 0; y < H; y++ {
		st := style(textCol, u.rows[y])
		for x := 0; x < W; x++ {
			b.Content[y*W+x] = cell.Cell{Content: ' ', Style: st}
		}
	}

	c := canvas{b: b, w: W, h: H}
	if W < 24 || H < 8 {
		c.textMax(0, H/2, "pikap: a bigger window, please", style(subCol, u.rows[H/2]), W)
		u.haveDeck, u.listW, u.bar, u.vol = false, 0, span{}, span{}
		u.buttons = [5]span{}
		u.fullBtn, u.rpmSpan = span{}, span{}
		return
	}

	u.lastW = W
	mainW := W - u.listWidth(W, H)
	u.listW = W - mainW
	if u.full {
		u.renderFull(c, b, W, H, accent)
		return
	}

	// Header.
	hy := 0
	if H >= 24 {
		hy = 1
	}
	x := c.text(2, hy, "◉ pikap", style(accent, u.rows[hy]).Bold())
	if u.np.player != "" && u.local == nil {
		x = c.text(x, hy, "  following ", style(dimCol, u.rows[hy]))
		c.textMax(x, hy, u.np.player, style(subCol, u.rows[hy]), mainW-x-16)
	} else if u.np.album != "" {
		x = c.text(x, hy, "  ", style(dimCol, u.rows[hy]))
		c.textMax(x, hy, u.np.album, style(subCol, u.rows[hy]), mainW-x-16)
	}
	c.set(mainW-3, hy, '⤢', style(subCol, u.rows[hy]))
	u.fullBtn = span{mainW - 4, hy, 3}
	u.drawRPM(c, mainW-6, hy)

	// Below the deck: title, artist, progress, controls, help.
	compact := H < 26
	below := 9
	if compact {
		below = 6
	}
	left, right := u.drawDeck(b, 2, hy+2, mainW-4, H-hy-2-below, mainW)
	y := H - below
	if !compact {
		y++
	}
	u.drawNowPlaying(c, left, right, y, compact, accent)
	u.drawHelp(c, H-1, mainW)

	if u.listW > 0 {
		u.drawList(c, mainW, W, H, accent)
	}
	if u.help {
		u.drawHelpCard(c, mainW, accent)
	}
}

// listWidth is how wide the list beside the deck is: as dragged, or two
// fifths of a wide window, and none in fullscreen, when put away, or when
// the deck would have too little room.
func (u *ui) listWidth(W, H int) int {
	if u.full || u.listHidden || H < 12 {
		return 0
	}
	w := u.listWant
	if w == 0 {
		if W < 100 || H < 16 {
			return 0
		}
		w = min(48, W*2/5)
	}
	w = max(minList, min(w, W-minMain))
	if w < minList {
		return 0
	}
	return w
}

// The narrowest the list and the deck beside it may be dragged.
const (
	minList = 22
	minMain = 40
)

// drawDeck draws the record in the given area, and returns the columns
// the text under it should keep to: the record's own, when it is narrower
// than the area.
func (u *ui) drawDeck(b *buffer.Buffer, x, y, w, h, mainW int) (left, right int) {
	u.dv, u.haveDeck = fitDeck(x, y, w, h)
	if u.haveDeck {
		u.scene.prepare(u.cover, int(u.dv.R), u.disc)
		p := sceneParams{angle: u.angle, arm: u.armA, lifted: u.armLift, style: u.disc, tube: 0.024}
		if u.glyph == asciiGlyphs {
			p.tube = 0.35 / u.dv.R
		}
		u.scene.draw(b, u.dv, p, u.glyph, u.gt, u.rows)
	}
	left, right = 3, mainW-3
	if u.haveDeck && u.dv.w < mainW-8 {
		left = max(3, u.dv.x0+1)
		right = min(mainW-3, u.dv.x0+u.dv.w-1)
	}
	return left, right
}

// renderFull is the fullscreen view: the record as large as the window
// allows, and under it what plays, the progress and the controls — no
// header, list or help line, though ? still opens the keys. A corner
// button, f or Esc goes back.
func (u *ui) renderFull(c canvas, b *buffer.Buffer, W, H int, accent rgb) {
	u.rpmSpan = span{}
	below := 5
	if H >= 30 {
		below = 7
	}
	left, right := u.drawDeck(b, 2, 1, W-4, H-1-below, W)
	u.drawNowPlaying(c, left, right, H-below, below == 5, accent)
	c.set(W-2, 0, '⤡', style(dimCol, u.rows[0]))
	u.fullBtn = span{W - 3, 0, 3}
	if u.help {
		u.drawHelpCard(c, W, accent)
	}
}

// drawNowPlaying writes the title and artist at y, then the progress and
// the controls, with a blank row between them unless compact.
func (u *ui) drawNowPlaying(c canvas, left, right, y int, compact bool, accent rgb) {
	H := c.h
	if u.np.status != "" {
		c.textMax(left, y, u.np.status, style(subCol, u.rows[y]), right-left)
		if u.local == nil {
			c.textMax(left, y+1, "Or play files: pikap ~/Music", style(dimCol, u.rows[y+1]), right-left)
		}
	} else {
		c.textMax(left, y, u.np.title, style(textCol, u.rows[y]).Bold(), right-left)
		artist := u.np.artist
		if artist == "" {
			artist = u.np.album
		}
		c.textMax(left, y+1, artist, style(subCol, u.rows[y+1]), right-left)
	}
	y += 2
	if !compact {
		y++
	}
	u.drawProgress(c, left, right, y, accent)
	y++
	if !compact {
		y++
	}
	u.drawControls(c, left, right, y, accent)
	y++
	if y < H-1 && u.np.note != "" && !u.full {
		c.textMax(left, y, u.np.note, style(dimCol, u.rows[y]), right-left)
	}
}

// drawProgress draws "1:23 ━━━━━●──── 3:05"; it can be clicked and dragged.
func (u *ui) drawProgress(c canvas, left, right, y int, accent rgb) {
	bg := u.rows[y]
	pos, length := u.np.pos, u.np.length
	x := c.clock(left, y, pos, style(subCol, bg))
	x++
	endW := 5
	if length >= 3600 {
		endW = 7
	}
	barW := right - x - endW - 1
	u.bar = span{x, y, max(0, barW)}
	if barW < 4 {
		return
	}
	fill := 0
	if length > 0 {
		fill = int(math.Round(pos / length * float64(barW-1)))
		fill = max(0, min(barW-1, fill))
	}
	for i := 0; i < barW; i++ {
		switch {
		case length > 0 && i == fill:
			c.set(x+i, y, '●', style(textCol, bg))
		case length > 0 && i < fill:
			c.set(x+i, y, '━', style(accent, bg))
		default:
			c.set(x+i, y, '─', style(trackCol, bg))
		}
	}
	if length > 0 {
		c.clock(x+barW+1, y, length, style(subCol, bg))
	} else {
		c.text(x+barW+1, y, "-:--", style(dimCol, bg))
	}
}

// drawControls draws the transport, centred, and the volume to the right.
func (u *ui) drawControls(c canvas, left, right, y int, accent rgb) {
	bg := u.rows[y]
	play := "▶"
	if u.np.playing {
		play = "❚❚"
	}
	repeat := "↻"
	if u.np.repeat == 2 {
		repeat = "↻¹"
	}
	labels := [5]string{"⇄", "|◀◀", play, "▶▶|", repeat}
	gap := 4
	total := -gap
	for _, l := range labels {
		total += cell.StringWidth(l) + gap
	}
	volW := 0
	if u.np.volume >= 0 {
		volW = 16
	}
	x := left + (right-left-volW-total)/2
	x = max(left, x)
	for i, l := range labels {
		fg := subCol
		switch {
		case i == 2:
			fg = textCol
		case i == 0 && u.np.shuffle, i == 4 && u.np.repeat > 0:
			fg = accent
		}
		st := style(fg, bg)
		if i == 2 {
			st = st.Bold()
		}
		w := cell.StringWidth(l)
		u.buttons[i] = span{x - 1, y, w + 2}
		c.text(x, y, l, st)
		x += w + gap
	}
	u.vol = span{}
	if volW > 0 && right-left > total+volW+2 {
		vx := right - 12
		c.text(vx-3, y, "♪ ", style(dimCol, bg))
		n := int(math.Round(u.np.volume * 12))
		for i := 0; i < 12; i++ {
			if i < n {
				c.set(vx+i, y, '━', style(subCol, bg))
			} else {
				c.set(vx+i, y, '─', style(trackCol, bg))
			}
		}
		u.vol = span{vx, y, 12}
	}
}

// The speeds the record can be shown turning at, as a fraction of 33⅓:
// from 8⅓ rpm to 83⅓.
const (
	minRPM  = 0.25
	maxRPM  = 2.5
	speed45 = 45 / (100.0 / 3)
)

// rpmBy changes the speed shown by d revolutions a minute, keeping to
// whole numbers on the way so the steps are even.
func (u *ui) rpmBy(d float64) {
	rpm := u.rpm * 100 / 3
	next := math.Round(rpm) + d
	if math.Abs(rpm-math.Round(rpm)) > 0.01 {
		// From 33⅓ or 16⅔, the first step lands on a whole number.
		if d > 0 {
			next = math.Ceil(rpm)
		} else {
			next = math.Floor(rpm)
		}
	}
	u.setRPM(next * 3 / 100)
}

// setRPM sets how fast the record is shown turning. It is a look only:
// the music, and the player, go on at their own speed.
func (u *ui) setRPM(r float64) { u.rpm = max(minRPM, min(maxRPM, r)) }

// drawRPM writes the speed right-aligned to column right: the named speeds
// as a turntable labels them, anything else in whole rpm and tenths.
func (u *ui) drawRPM(c canvas, right, y int) {
	st := style(dimCol, u.rows[y])
	rpm := u.rpm * 100 / 3
	var name string
	for _, p := range [...]struct {
		rpm  float64
		name string
	}{{50.0 / 3, "16⅔ rpm"}, {100.0 / 3, "33⅓ rpm"}, {45, "45 rpm"}, {78, "78 rpm"}} {
		if math.Abs(rpm-p.rpm) < 0.02 {
			name = p.name
		}
	}
	if name != "" {
		w := cell.StringWidth(name)
		c.text(right-w, y, name, st)
		u.rpmSpan = span{right - w, y, w}
		return
	}
	tenths := int(math.Round(rpm * 10))
	w := digits(tenths/10) + 2 + 4
	x := c.number(right-w, y, tenths/10, st)
	c.set(x, y, '.', st)
	c.set(x+1, y, rune('0'+tenths%10), st)
	c.text(x+2, y, " rpm", st)
	u.rpmSpan = span{right - w, y, w}
}

func digits(n int) int {
	w := 1
	for n >= 10 {
		n /= 10
		w++
	}
	return w
}

func (u *ui) drawHelp(c canvas, y, w int) {
	bg := u.rows[y]
	if u.help {
		c.textMax(2, y, "? or esc closes the keys · q quit", style(dimCol, bg), w-4)
		return
	}
	c.textMax(2, y, "space play · ←→ seek · [ ] track · f fullscreen · tab list · +- volume · ? keys · q quit", style(dimCol, bg), w-4)
}

// helpKeys is the card ? opens: every key and gesture. One line of help
// could not hold them; beside the list it was cut off after "r".
var helpKeys = [...][2]string{
	{"space  k", "play / pause"},
	{"← →  h l", "seek 5 s, with shift 30"},
	{", .", "seek 30 s"},
	{"[ ]  p n", "previous / next"},
	{"↑ ↓  enter", "choose from the list"},
	{"tab", "hide or show the list"},
	{"f", "fullscreen"},
	{"s  r", "shuffle / repeat"},
	{"+ -", "volume"},
	{"1 3 4 7  < >", "rpm, only the look"},
	{"d", "picture disc / black vinyl"},
	{"v", "half blocks / ascii"},
	{"c", "crackle, playing files"},
	{"drag record", "scrub"},
	{"drag arm", "put the needle down"},
	{"q", "quit"},
}

const (
	helpKeyW  = 13 // the widest key column, "1 3 4 7  < >"
	helpDescW = 26 // the widest description, "picture disc / black vinyl"
)

// drawHelpCard draws the keys over the middle of columns [0, w). Rows that
// do not fit are left off the bottom; a window too narrow gets no card.
func (u *ui) drawHelpCard(c canvas, w int, accent rgb) {
	cw := 2 + helpKeyW + 2 + helpDescW + 2
	if w < cw+2 || c.h < 6 {
		return
	}
	rows := min(len(helpKeys), c.h-5)
	ch := rows + 4
	x0, y0 := (w-cw)/2, (c.h-ch)/2
	// The card is the panel behind it, darker, so the cover's colour holds.
	for y := y0; y < y0+ch; y++ {
		st := style(textCol, u.rows[y].scale(0.45))
		for x := x0; x < x0+cw; x++ {
			c.set(x, y, ' ', st)
		}
	}
	c.text(x0+2, y0+1, "keys", style(accent, u.rows[y0+1].scale(0.45)).Bold())
	for i := 0; i < rows; i++ {
		y := y0 + 3 + i
		dark := u.rows[y].scale(0.45)
		c.text(x0+2, y, helpKeys[i][0], style(textCol, dark))
		c.text(x0+2+helpKeyW+2, y, helpKeys[i][1], style(subCol, dark))
	}
}

func (u *ui) drawList(c canvas, x0, W, H int, accent rgb) {
	list := u.src.entries()
	listBg := lin(0x000000)
	// The list's left edge is a line that can be taken and dragged; it is
	// brighter while it is.
	edge := trackCol
	if u.divDrag {
		edge = accent
	}
	for y := 0; y < H; y++ {
		bg := u.rows[y].mix(listBg, 0.45)
		for x := x0; x < W; x++ {
			c.set(x, y, ' ', style(textCol, bg))
		}
		c.set(x0, y, '│', style(edge, bg))
	}
	hy := 0
	if H >= 24 {
		hy = 1
	}
	bgAt := func(y int) rgb { return u.rows[y].mix(listBg, 0.45) }
	x := c.text(x0+2, hy, u.src.listTitle(), style(textCol, bgAt(hy)).Bold())
	if n := len(list); n > 0 {
		x = c.text(x+1, hy, "·", style(dimCol, bgAt(hy)))
		x = c.number(x+1, hy, n, style(dimCol, bgAt(hy)))
	}
	u.listX, u.listTop = x0, hy+2
	u.listRows = max(0, H-u.listTop-1)
	cur := -1
	for i, e := range list {
		if e.current {
			cur = i
		}
	}
	if !u.cursorOn && cur >= 0 {
		u.cursor = cur
	}
	u.cursor = max(0, min(len(list)-1, u.cursor))
	// Keep the cursor in view, with a little room around it.
	if u.cursor < u.listOff+1 {
		u.listOff = max(0, u.cursor-1)
	}
	if u.cursor >= u.listOff+u.listRows-1 {
		u.listOff = u.cursor - u.listRows + 2
	}
	u.listOff = max(0, min(u.listOff, len(list)-u.listRows))
	w := W - x0 - 4
	for row := 0; row < u.listRows; row++ {
		i := u.listOff + row
		if i >= len(list) {
			break
		}
		e := list[i]
		y := u.listTop + row
		bg := bgAt(y)
		if u.cursorOn && i == u.cursor {
			bg = bg.mix(textCol, 0.08)
			for xx := x0; xx < W; xx++ {
				c.set(xx, y, ' ', style(textCol, bg))
			}
		}
		fg, sub := textCol, dimCol
		if e.current {
			fg = accent
		}
		lx := x0 + 2
		switch {
		case e.current && e.playing:
			c.set(lx, y, '♪', style(accent, bg))
		case e.current:
			c.set(lx, y, '·', style(accent, bg))
		case e.num > 0:
			if e.num < 10 {
				lx++
			}
			c.number(lx, y, e.num, style(sub, bg))
		}
		lx = x0 + 5
		durW := 0
		if e.dur > 0 {
			durW = 6
			if e.dur >= 3600 {
				durW = 8
			}
			c.clock(W-2-durW+1, y, e.dur, style(sub, bg))
		}
		room := w - 3 - durW
		n := c.textMax(lx, y, e.label, style(fg, bg), room)
		if e.detail != "" && n+3 < room {
			c.textMax(lx+n+1, y, e.detail, style(sub, bg), room-n-1)
		}
	}
}

// ── input ────────────────────────────────────────────────────────────────

func (u *ui) key(k limoni.KeyEvent) {
	if k.Release {
		return
	}
	ch := k.Ch
	switch k.Type {
	case limoni.KeySpace:
		u.src.toggle()
	case limoni.KeyLeft:
		u.seekBy(-u.seekStep(k))
	case limoni.KeyRight:
		u.seekBy(u.seekStep(k))
	case limoni.KeyUp:
		u.moveCursor(-1)
	case limoni.KeyDown:
		u.moveCursor(1)
	case limoni.KeyPageUp:
		u.moveCursor(-max(1, u.listRows-1))
	case limoni.KeyPageDown:
		u.moveCursor(max(1, u.listRows-1))
	case limoni.KeyHome:
		u.src.seek(0)
	case limoni.KeyEnter:
		if u.cursorOn {
			u.src.choose(u.cursor)
			u.cursorOn = false
		}
	case limoni.KeyTab:
		u.listHidden = !u.listHidden
		u.full = false
	case limoni.KeyEsc:
		if u.help {
			u.help = false
		} else if u.full {
			u.full = false
		} else if u.cursorOn {
			u.cursorOn = false
		} else {
			u.quit = true
		}
	case limoni.KeyRune:
		switch ch {
		case 'q', 'Q':
			u.quit = true
		case 'k', 'K':
			u.src.toggle()
		case 'n', 'N', ']':
			u.src.next()
		case 'p', 'P', '[':
			u.src.prev()
		case 'h', 'H':
			u.seekBy(-5)
		case 'l', 'L':
			u.seekBy(5)
		case ',':
			u.seekBy(-30)
		case '.':
			u.seekBy(30)
		case '+', '=':
			u.volumeBy(0.05)
		case '-', '_':
			u.volumeBy(-0.05)
		case '1':
			u.setRPM(0.5) // 16⅔
		case '3':
			u.setRPM(1)
		case '4':
			u.setRPM(speed45)
		case '7':
			u.setRPM(78 / (100.0 / 3))
		case '<':
			u.rpmBy(-1)
		case '>':
			u.rpmBy(1)
		case 's', 'S':
			u.src.toggleShuffle()
		case 'r', 'R':
			u.src.cycleRepeat()
		case 'd', 'D':
			u.disc = 1 - u.disc
		case 'v', 'V':
			u.glyph = 1 - u.glyph
		case 'c', 'C':
			if u.local != nil {
				u.local.d.crackle.Store(!u.local.d.crackle.Load())
			}
		case '?':
			u.help = !u.help
		case 'f', 'F':
			u.full = !u.full
		case '0':
			u.src.seek(0)
		}
	}
}

func (u *ui) seekStep(k limoni.KeyEvent) float64 {
	if k.Shift {
		return 30
	}
	return 5
}

func (u *ui) seekBy(d float64) {
	if u.np.canSeek {
		// Taken as done at once, so a second press before the next frame
		// counts from the first.
		u.np.pos = max(0, u.np.pos+d)
		if u.np.length > 0 {
			u.np.pos = min(u.np.pos, u.np.length)
		}
		u.src.seek(u.np.pos)
	}
}

func (u *ui) volumeBy(d float64) {
	if u.np.volume >= 0 {
		u.src.setVolume(max(0, min(1, u.np.volume+d)))
	}
}

func (u *ui) moveCursor(d int) {
	n := len(u.src.entries())
	if n == 0 {
		return
	}
	u.cursorOn = true
	u.cursor = max(0, min(n-1, u.cursor+d))
}

func (u *ui) mouse(m limoni.MouseEvent) {
	x, y := int(m.X), int(m.Y)
	fx, fy := float64(m.X)+0.5, float64(m.Y)+0.5
	switch {
	case m.Button == limoni.MouseRelease:
		u.releaseAll(fx, fy)
		return
	case m.Button == limoni.MouseScrollUp || m.Button == limoni.MouseScrollDown:
		d := 1
		if m.Button == limoni.MouseScrollUp {
			d = -1
		}
		// Over the speed, the wheel is the pitch control: up is faster.
		if u.rpmSpan.hit(x, y) {
			u.rpmBy(float64(-d))
			return
		}
		if u.listW > 0 && x >= u.listX {
			u.listOff = max(0, u.listOff+d*3)
			u.cursorOn = true
			u.cursor = max(u.listOff, min(u.cursor, u.listOff+u.listRows-1))
			return
		}
		if u.vol.w > 0 && y == u.vol.y {
			u.volumeBy(float64(-d) * 0.05)
			return
		}
		// The wheel over the record turns it: down is forward.
		u.seekBy(float64(d) * 3)
		return
	case m.Button != limoni.MouseLeft:
		return
	}

	if m.Drag {
		switch {
		case u.handDown:
			a := math.Atan2(u.unitsOf(fx, fy)) / (2 * math.Pi)
			d := a - u.handRev
			d -= math.Round(d)
			u.handRev = a
			u.angle += d
			u.src.scrub(d)
		case u.armDrag:
			uu, vv := u.dv.toUnits(fx, fy)
			u.armDragR = max(grooveIn(u.disc), min(grooveOut, math.Hypot(uu, vv)))
		case u.barDrag:
			u.seekToBar(x)
		case u.volDrag:
			u.setVolumeAt(x)
		case u.divDrag:
			u.listWant = max(minList, min(u.lastW-minMain, u.lastW-x))
		}
		return
	}

	// A press.
	u.releaseAll(fx, fy)
	if u.listW > 0 && (x == u.listX || x == u.listX-1) {
		u.divDrag = true
		u.listWant = u.listW
		return
	}
	if u.fullBtn.hit(x, y) {
		u.full = !u.full
		return
	}
	if u.haveDeck {
		uu, vv := u.dv.toUnits(fx, fy)
		nu, nv := pivotU+math.Cos(u.armA)*armLen, pivotV+math.Sin(u.armA)*armLen
		if d, _ := segDist(uu, vv, pivotU, pivotV, nu, nv); d < 0.12 && u.np.canSeek && u.np.length > 0 {
			u.armDrag = true
			u.armDragR = max(grooveIn(u.disc), min(grooveOut, math.Hypot(uu, vv)))
			return
		}
		if math.Hypot(uu, vv) <= 1.02 {
			u.handDown = true
			u.handRev = math.Atan2(vv, uu) / (2 * math.Pi)
			u.src.grab()
			return
		}
	}
	switch {
	case u.bar.hit(x, y):
		u.barDrag = true
		u.seekToBar(x)
	case u.vol.hit(x, y):
		u.volDrag = true
		u.setVolumeAt(x)
	case u.buttons[0].hit(x, y):
		u.src.toggleShuffle()
	case u.buttons[1].hit(x, y):
		u.src.prev()
	case u.buttons[2].hit(x, y):
		u.src.toggle()
	case u.buttons[3].hit(x, y):
		u.src.next()
	case u.buttons[4].hit(x, y):
		u.src.cycleRepeat()
	case u.listW > 0 && x >= u.listX && y >= u.listTop && y < u.listTop+u.listRows:
		i := u.listOff + y - u.listTop
		if i < len(u.src.entries()) {
			u.cursor = i
			u.cursorOn = false
			u.src.choose(i)
		}
	}
}

func (u *ui) unitsOf(fx, fy float64) (y, x float64) {
	uu, vv := u.dv.toUnits(fx, fy)
	return vv, uu
}

// releaseAll lets go of whatever is held; the arm, let go, puts the needle
// down where it is.
func (u *ui) releaseAll(fx, fy float64) {
	if u.handDown {
		u.handDown = false
		u.src.release()
	}
	if u.armDrag {
		u.armDrag = false
		in := grooveIn(u.disc)
		prog := (grooveOut - u.armDragR) / (grooveOut - in)
		if u.np.length > 0 {
			u.src.seek(max(0, min(1, prog)) * u.np.length)
		}
	}
	u.barDrag, u.volDrag, u.divDrag = false, false, false
}

func (u *ui) seekToBar(x int) {
	if u.bar.w < 2 || u.np.length <= 0 || !u.np.canSeek {
		return
	}
	f := float64(x-u.bar.x) / float64(u.bar.w-1)
	u.src.seek(max(0, min(1, f)) * u.np.length)
}

func (u *ui) setVolumeAt(x int) {
	if u.vol.w == 0 {
		return
	}
	u.src.setVolume(max(0, min(1, float64(x-u.vol.x+1)/float64(u.vol.w))))
}
