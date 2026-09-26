package main

import (
	"math"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// The board is drawn in half blocks: a grain is one pixel, and a cell holds
// two of them, '▀' with the upper as the foreground and the lower as the
// background. A cell is about twice as tall as it is wide, so grains come
// out square. Every colour on screen comes from a fixed palette or from the
// background, which is drawn once per board size; a frame writes cells and
// nothing else, and the diff sends only the cells that changed.

const (
	panelW = 20 // the side panel's width in cells
	gap    = 2  // cells between the board and the panel
)

type rgb struct{ r, g, b float64 }

func hex(v uint32) rgb {
	return rgb{float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255}
}

func (c rgb) mix(o rgb, t float64) rgb {
	return rgb{c.r + (o.r-c.r)*t, c.g + (o.g-c.g)*t, c.b + (o.b-c.b)*t}
}

func (c rgb) scale(k float64) rgb { return rgb{c.r * k, c.g * k, c.b * k} }

func to8(v float64) uint8 { return uint8(math.Round(max(0, min(1, v)) * 255)) }

func (c rgb) color() cell.Color { return cell.NewColorRGB(to8(c.r), to8(c.g), to8(c.b)) }

var (
	screenBg  = hex(0x0e0f12)
	boardBg   = hex(0x16181d)
	borderCol = hex(0x6b6035)
	textCol   = hex(0xdcdcd2)
	labelCol  = hex(0x86867c)
	accent    = hex(0xf5d000)

	// The grains' colours: lemon, lime, grapefruit and blueberry.
	baseColours = [colours + 1]rgb{
		{},
		hex(0xf4d21f),
		hex(0x74c043),
		hex(0xea5a5a),
		hex(0x5a7fe0),
	}

	// The lemon behind the board, as a slice seen from the cut face.
	rindCol   = hex(0xf2c200)
	pithCol   = hex(0xfff3c4)
	fleshCol  = hex(0xffd84a)
	lemonAmt  = 0.17 // how much of the lemon shows through: faint, not a picture
	segments  = 10.0
	flashCols = [2]cell.Color{hex(0xfffbe6).color(), hex(0xffe98a).color()}
)

// pal is every grain colour by colour and shade; ghost is the landing
// preview's.
var (
	pal   [colours + 1][4]cell.Color
	ghost [colours + 1]cell.Color
)

func init() {
	shades := [4]float64{0.68, 0.86, 1, 1.12}
	for c := 1; c <= colours; c++ {
		for s, k := range shades {
			col := baseColours[c].scale(k)
			if k > 1 {
				col = baseColours[c].mix(rgb{1, 1, 1}, k-1)
			}
			pal[c][s] = col.color()
		}
		ghost[c] = boardBg.mix(baseColours[c], 0.22).color()
	}
}

// buildBackground draws the board's empty picture — the dark board with a
// faint half lemon in it — for the current size. It runs when the size
// changes, not every frame.
func (g *game) buildBackground() {
	gw, gh := g.gw, g.gh
	cx, cy := float64(gw)/2, float64(gh)*0.56
	R := float64(gw) * 0.44
	const ss = 3 // samples a side, to smooth the edges
	for y := 0; y < gh; y++ {
		for x := 0; x < gw; x++ {
			var sum rgb
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					u := (float64(x) + (float64(sx)+0.5)/ss - cx) / R
					v := (float64(y) + (float64(sy)+0.5)/ss - cy) / R
					c := boardBg
					if l, ok := lemonAt(u, v); ok {
						c = boardBg.mix(l, lemonAmt)
					}
					sum = rgb{sum.r + c.r, sum.g + c.g, sum.b + c.b}
				}
			}
			g.bg[y*gw+x] = sum.scale(1.0 / (ss * ss)).color()
		}
	}
	g.bgFor = g.b
}

// lemonAt is the lemon slice at (u, v), in radii from its centre: rind,
// pith, and flesh in segments parted by pale membranes.
func lemonAt(u, v float64) (rgb, bool) {
	d := math.Hypot(u, v)
	switch {
	case d > 1:
		return rgb{}, false
	case d > 0.9:
		return rindCol, true
	case d > 0.8 || d < 0.1:
		return pithCol, true
	}
	a := (math.Atan2(v, u) + math.Pi) / (2 * math.Pi) * segments
	f := a - math.Floor(a)
	if min(f, 1-f)*2*math.Pi/segments*d < 0.035 {
		return pithCol, true
	}
	return fleshCol.scale(0.86 + 0.14*d), true
}

// canvas writes cells straight into the frame's buffer.
type canvas struct {
	b    *buffer.Buffer
	w, h int
}

func style(fg, bg rgb) cell.Style { return cell.Style{Fg: fg.color(), Bg: bg.color()} }

func (c canvas) set(x, y int, r rune, s cell.Style) {
	if x >= 0 && y >= 0 && x < c.w && y < c.h {
		c.b.Content[y*c.w+x] = cell.Cell{Content: r, Style: s}
	}
}

func (c canvas) text(x, y int, s string, st cell.Style) int {
	for _, r := range s {
		c.set(x, y, r, st)
		x++
	}
	return x
}

// number writes n in decimal without allocating.
func (c canvas) number(x, y, n int, st cell.Style) int {
	var d [20]byte
	i := len(d)
	for {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	for _, ch := range d[i:] {
		c.set(x, y, rune(ch), st)
		x++
	}
	return x
}

func digits(n int) int {
	w := 1
	for n >= 10 {
		n /= 10
		w++
	}
	return w
}

func (c canvas) fill(x0, y0, w, h int, st cell.Style) {
	if x0 == 0 && y0 == 0 && w == c.w && h == c.h {
		blank := cell.Cell{Content: ' ', Style: st}
		for i := range c.b.Content[:w*h] {
			c.b.Content[i] = blank
		}
		return
	}
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			c.set(x, y, ' ', st)
		}
	}
}

func textWidth(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// frame is what one render needs to know about the falling piece.
type frame struct {
	piece, ghost bool
	mask         uint16
	colour       uint8
	x, y, gy     int
	flash        cell.Color
	flashOn      bool
}

// render draws one frame. It allocates nothing.
func (g *game) render(b *buffer.Buffer) {
	W, H := int(b.Area.Width), int(b.Area.Height)
	if len(b.Content) < W*H {
		return
	}
	b.IsDirty = true
	g.lastW, g.lastH = W, H
	cv := canvas{b: b, w: W, h: H}
	cv.fill(0, 0, W, H, style(textCol, screenBg))

	fit := fitSize(W, H)
	if g.phase == phTitle && fit != 0 && fit != g.b {
		g.setSize(fit)
	}
	if fit < g.b {
		if g.phase == phPlay {
			g.paused = true
		}
		centre(cv, W/2, H/2-1, "Lemon Drop needs a larger window:", style(textCol, screenBg))
		need := "44×20 or more"
		if g.phase != phTitle && fit != 0 {
			need = "enlarge it to go on, or R to restart"
		}
		centre(cv, W/2, H/2, need, style(labelCol, screenBg))
		return
	}
	if g.bgFor != g.b {
		g.buildBackground()
	}

	rowsOnScreen := g.gh / 2
	total := g.gw + 2 + gap + panelW
	left := (W - total) / 2
	top := (H - rowsOnScreen - 2) / 2
	g.bx, g.by = left+1, top+1
	g.panelX, g.topY = left+g.gw+2+gap, top

	g.drawBorder(cv, left, top, g.gw+2, rowsOnScreen+2)
	g.drawBoard(cv)
	g.drawPanel(cv)
	g.drawOverlay(cv)
}

func (g *game) drawBorder(cv canvas, x, y, w, h int) {
	st := style(borderCol, screenBg)
	cv.set(x, y, '╭', st)
	cv.set(x+w-1, y, '╮', st)
	cv.set(x, y+h-1, '╰', st)
	cv.set(x+w-1, y+h-1, '╯', st)
	for i := x + 1; i < x+w-1; i++ {
		cv.set(i, y, '─', st)
		cv.set(i, y+h-1, '─', st)
	}
	for j := y + 1; j < y+h-1; j++ {
		cv.set(x, j, '│', st)
		cv.set(x+w-1, j, '│', st)
	}
}

func (g *game) drawBoard(cv canvas) {
	var f frame
	if g.phase == phPlay {
		p := &g.cur
		f.piece, f.mask, f.colour, f.x, f.y = true, masks[p.kind][p.rot], p.colour, p.x, p.y
		f.gy = g.dropY()
		f.ghost = f.gy-p.y >= g.b
	}
	if g.flashT > 0 {
		f.flashOn = int(g.flashT/0.075)&1 == 0
		f.flash = flashCols[int(g.flashT/0.15)&1]
	}
	w, b := cv.w, g.b
	for r := 0; r < g.gh/2; r++ {
		y := g.by + r
		if y < 0 || y >= cv.h {
			continue
		}
		out := cv.b.Content[y*w+g.bx : y*w+g.bx+g.gw]
		// Most rows have no piece in them: those read the sand and the
		// background straight.
		y0 := 2 * r
		slow := f.piece && y0+1 >= f.y && y0 < f.y+4*b || f.ghost && y0+1 >= f.gy && y0 < f.gy+4*b
		upper, lower := y0*g.gw, (y0+1)*g.gw
		for x := range out {
			var top, bot cell.Color
			if slow {
				top, bot = g.pixel(&f, x, y0), g.pixel(&f, x, y0+1)
			} else {
				top, bot = g.grain(&f, upper+x), g.grain(&f, lower+x)
			}
			if top == bot {
				out[x] = cell.Cell{Content: ' ', Style: cell.Style{Fg: bot, Bg: bot}}
			} else {
				out[x] = cell.Cell{Content: '▀', Style: cell.Style{Fg: top, Bg: bot}}
			}
		}
	}
}

// grain is the colour at board index i without the piece: a grain of sand,
// or the background.
func (g *game) grain(f *frame, i int) cell.Color {
	v := g.sand[i]
	switch {
	case v == 0:
		return g.bg[i]
	case v&clearBit != 0 && f.flashOn:
		return f.flash
	}
	return pal[v&colourMask][v>>shadeShift&3]
}

// pixel is the colour of one grain's place: the falling piece, a grain of
// sand, the landing preview, or the background.
func (g *game) pixel(f *frame, x, y int) cell.Color {
	b := g.b
	if f.piece {
		if dx, dy := x-f.x, y-f.y; dx >= 0 && dy >= 0 && dx < 4*b && dy < 4*b &&
			f.mask&(1<<((dy/b)*4+dx/b)) != 0 {
			return pal[f.colour][shadeAt(dx%b, dy%b, b)]
		}
	}
	i := y*g.gw + x
	if g.sand[i] != 0 {
		return g.grain(f, i)
	}
	if f.ghost {
		if dx, dy := x-f.x, y-f.gy; dx >= 0 && dy >= 0 && dx < 4*b && dy < 4*b &&
			f.mask&(1<<((dy/b)*4+dx/b)) != 0 {
			return ghost[f.colour]
		}
	}
	return g.bg[i]
}

func (g *game) drawPanel(cv canvas) {
	x, y := g.panelX, g.topY
	bottom := g.topY + g.gh/2 + 2
	lab, val := style(labelCol, screenBg), style(textCol, screenBg)

	cv.text(x, y, "LEMON DROP", style(accent, screenBg))
	y += 2
	cv.text(x, y, "NEXT", lab)
	y++
	if g.phase == phPlay {
		g.drawNext(cv, x, y)
	}
	y += 3

	row := func(label string, n int) {
		cv.text(x, y, label, lab)
		cv.number(x+panelW-1-digits(n), y, n, val)
		y++
	}
	row("SCORE", g.score)
	row("BEST", max(g.best, g.score))
	row("LEVEL", g.level)
	row("CLEARS", g.clears)
	y++

	if g.phase == phPlay && g.gainT < 1.6 && g.gain > 0 {
		st := style(accent, screenBg)
		e := cv.text(x, y, "+", st)
		e = cv.number(e, y, g.gain, st)
		if g.gainCombo > 1 {
			e = cv.text(e+1, y, "chain ×", st)
			cv.number(e, y, g.gainCombo, st)
		}
	}
	y += 2

	help := [...][2]string{
		{"← →", "move"},
		{"↑ X", "turn"},
		{"Z", "turn back"},
		{"↓", "soft drop"},
		{"SPACE", "drop"},
		{"P", "pause"},
		{"ESC", "quit"},
	}
	for _, h := range help {
		if y >= bottom {
			break
		}
		cv.text(x, y, h[0], val)
		cv.text(x+7, y, h[1], lab)
		y++
	}
}

// drawNext shows the next piece, two pixels to a block.
func (g *game) drawNext(cv canvas, x0, y0 int) {
	p := g.next
	m := masks[p.kind][0]
	top := 4
	for _, c := range shapes[p.kind][0] {
		top = min(top, int(c[1]))
	}
	px := func(x, y int) cell.Color {
		bx, by := x/2, y/2+top
		if by < 4 && m&(1<<(by*4+bx)) != 0 {
			return pal[p.colour][shadeAt(x%2, y%2, 2)]
		}
		return screenBg.color()
	}
	for r := 0; r < 2; r++ {
		for x := 0; x < 8; x++ {
			t, b := px(x, 2*r), px(x, 2*r+1)
			if t == b {
				cv.set(x0+x, y0+r, ' ', cell.Style{Fg: b, Bg: b})
			} else {
				cv.set(x0+x, y0+r, '▀', cell.Style{Fg: t, Bg: b})
			}
		}
	}
}

// drawOverlay puts the title, the pause and the end over the board.
func (g *game) drawOverlay(cv canvas) {
	var lines [4]string
	n := 0
	switch {
	case g.phase == phTitle:
		lines, n = [4]string{"LEMON DROP", "", "ENTER  play", "ESC    quit"}, 4
	case g.phase == phOver && g.now-g.overAt > 0.6:
		lines, n = [4]string{"GAME OVER", "", "R      play again", "ESC    quit"}, 4
	case g.paused:
		lines, n = [4]string{"PAUSED", "", "P      go on"}, 3
	default:
		return
	}
	boxW := 0
	for _, l := range lines[:n] {
		boxW = max(boxW, textWidth(l))
	}
	boxW = min(boxW+4, g.gw)
	boxH := n + 2
	cx := g.bx + g.gw/2
	y0 := g.by + (g.gh/2-boxH)/2
	x0 := cx - boxW/2
	bg := style(textCol, screenBg)
	cv.fill(x0, y0, boxW, boxH, bg)
	for i, l := range lines[:n] {
		st := bg
		if i == 0 {
			st = style(accent, screenBg)
		}
		cv.text(x0+2, y0+1+i, l, st)
	}
}

// centre writes s centred on column cx.
func centre(cv canvas, cx, y int, s string, st cell.Style) {
	cv.text(cx-textWidth(s)/2, y, s, st)
}
