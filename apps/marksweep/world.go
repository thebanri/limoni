package main

import (
	"math"
	"strings"
)

// The arena is the inside of a terminal: its walls are windows — a shell,
// a log viewer, some Go, htop — and the frameworks come in through the
// spawn windows (P).

const mapW, mapH = 32, 32

// arena is a map: its walls, and the colours of its floor and ceiling.
type arena struct {
	name                           string
	rows                           [mapH]string
	floor, seam, ceil, tray, light uint32
}

// The run goes through the first nStages arenas in turn; the last is
// /dev/zero, the endless one, open and with spawn windows all round.
const nStages = 3

var arenas = [...]arena{
	{
		name: "THE HEAP", floor: 0x0a0d18, seam: 0x24365e, ceil: 0x07090f, tray: 0x111727, light: 0x2a3654,
		rows: [mapH]string{
			"zzzzPzzzllllllllPzcccccccczPzzzz",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"P..............................P",
			"z..............................z",
			"z......cc..............hh......z",
			"h......cc..............hh......l",
			"h..............................l",
			"h...........llllllll...........l",
			"h..............................l",
			"h..............................l",
			"h...c......................c...l",
			"z...c......................c...z",
			"z...c......................c...z",
			"z...c......................c...z",
			"z...c......................c...z",
			"c...c......................c...h",
			"c..............................h",
			"c..............................h",
			"c...........llllllll...........h",
			"c..............................h",
			"c......hh..............cc......h",
			"c......hh..............cc......h",
			"c..............................h",
			"z..............................z",
			"P..............................P",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"zzzzPzzzccccccccPzllllllllzPzzzz",
		},
	},
	{
		name: "THE STACK", floor: 0x110c08, seam: 0x5a3a1c, ceil: 0x0c0806, tray: 0x22170d, light: 0x5a4024,
		rows: [mapH]string{
			"zzzzPzzzcccccccPzcccccccczzzPzzz",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"P..............................P",
			"z..ccccccccccc....ccccccccccc..z",
			"z..............................z",
			"z......l................l......z",
			"z......l................l......z",
			"z..............................z",
			"z..ccccccccccc....ccccccccccc..z",
			"z..............................z",
			"z..............hh..............z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............hh..............z",
			"z..............................z",
			"z..ccccccccccc....ccccccccccc..z",
			"z..............................z",
			"z......l................l......z",
			"z......l................l......z",
			"z..............................z",
			"P..ccccccccccc....ccccccccccc..P",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"zzzPzzzccccccccPccccccczzzzPzzzz",
		},
	},
	{
		name: "KERNEL SPACE", floor: 0x140709, seam: 0x5c1c28, ceil: 0x0e0507, tray: 0x26101a, light: 0x5a2232,
		rows: [mapH]string{
			"hhhhPhhhhhhhhhhPhhhhhhhhhhhPhhhh",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"h....ll....ll......cc....cc....h",
			"P....ll....ll......cc....cc....P",
			"h..............................h",
			"h............zzzzzz............h",
			"h..............................h",
			"h..............................h",
			"h....ll....ll......cc....cc....h",
			"h....ll....ll......cc....cc....h",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"h....cc....cc......ll....ll....h",
			"h....cc....cc......ll....ll....h",
			"h..............................h",
			"h..............................h",
			"h............zzzzzz............h",
			"h..............................h",
			"h....cc....cc......ll....ll....h",
			"P....cc....cc......ll....ll....P",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"h..............................h",
			"hhhhPhhhhhhhhhhhPhhhhhhhhhhPhhhh",
		},
	},
	{
		name: "/dev/zero", floor: 0x070b08, seam: 0x1e4a2c, ceil: 0x050806, tray: 0x0f1a12, light: 0x24503a,
		rows: [mapH]string{
			"zzzPzzzzzPzzzzzzzzzzzzPzzzzzPzzz",
			"z..............................z",
			"z..............................z",
			"P..............................P",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z.......cc............cc.......z",
			"P.......cc............cc.......P",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"P.......cc............cc.......P",
			"z.......cc............cc.......z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"z..............................z",
			"P..............................P",
			"z..............................z",
			"z..............................z",
			"zzzPzzzzzPzzzzzzzzzzzzPzzzzzPzzz",
		},
	},
}

// cur is the arena being played.
var cur = &arenas[0]

// useArena makes arena i the one played, and finds its spawn windows.
func useArena(i int) {
	cur = &arenas[i]
	portals = portals[:0]
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			if tile(x, y) != 'P' {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if tile(x+d[0], y+d[1]) == '.' {
					portals = append(portals, vec{float64(x+d[0]) + 0.5 + 0.3*float64(d[0]), float64(y+d[1]) + 0.5 + 0.3*float64(d[1])})
				}
			}
		}
	}
}

func tile(x, y int) byte {
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return 'z'
	}
	return cur.rows[y][x]
}

func solid(x, y float64) bool { return tile(int(math.Floor(x)), int(math.Floor(y))) != '.' }

// free reports whether a body of radius r fits at (x, y).
func free(x, y, r float64) bool {
	return !solid(x-r, y-r) && !solid(x+r, y-r) && !solid(x-r, y+r) && !solid(x+r, y+r)
}

// slide moves a body by (dx, dy), one axis at a time so it slides along walls.
func slide(p *vec, dx, dy, r float64) {
	if free(p.x+dx, p.y, r) {
		p.x += dx
	}
	if free(p.x, p.y+dy, r) {
		p.y += dy
	}
}

// hit is where a ray meets a wall.
type hit struct {
	dist  float64 // perpendicular, so walls are not bent
	side  int     // 0: an x face, 1: a y face
	tile  byte
	along float64 // position along the wall, in world units
}

// cast walks the grid from (px, py) along (dx, dy) to the first wall.
func cast(px, py, dx, dy float64) hit {
	mx, my := int(px), int(py)
	ddx, ddy := math.Abs(1/dx), math.Abs(1/dy)
	var sx, sy int
	var sdx, sdy float64
	if dx < 0 {
		sx, sdx = -1, (px-float64(mx))*ddx
	} else {
		sx, sdx = 1, (float64(mx)+1-px)*ddx
	}
	if dy < 0 {
		sy, sdy = -1, (py-float64(my))*ddy
	} else {
		sy, sdy = 1, (float64(my)+1-py)*ddy
	}
	side := 0
	for i := 0; i < 128; i++ {
		if sdx < sdy {
			sdx += ddx
			mx += sx
			side = 0
		} else {
			sdy += ddy
			my += sy
			side = 1
		}
		if t := tile(mx, my); t != '.' {
			h := hit{side: side, tile: t}
			if side == 0 {
				h.dist = sdx - ddx
				h.along = py + h.dist*dy
			} else {
				h.dist = sdy - ddy
				h.along = px + h.dist*dx
			}
			return h
		}
	}
	return hit{dist: 64, tile: 'z'}
}

// ── wall textures ────────────────────────────────────────────────────────

// A wall's texture is a terminal window drawn in pixels, with its text in
// the 3×5 font. Pixels are square on screen: a cell is a column wide and
// two pixels tall, drawn as a half block.
const (
	winW, winH = 128, 52
	nMip       = 5 // each half the size of the last, so far walls are averaged, not aliased
)

type wallTex struct {
	w, h   int
	span   float64 // world units one window spans along a wall
	mip    [nMip][]rgb
	mw, mh [nMip]int
	// Rows that scroll upwards with time: the data streaming into a spawn
	// window. Both zero for none.
	scroll0, scroll1 int
}

// at is the texel at (u, v), both 0…1, from mip level l. Level 0 has the
// text; the smaller levels are made from the same window with each
// character a bar of its colour, the way an editor's minimap reads code,
// because letters averaged down are grey noise.
func (t *wallTex) at(u, v float64, l int) rgb {
	w, h := t.mw[l], t.mh[l]
	x, y := int(u*float64(w)), int(v*float64(h))
	x = min(max(x, 0), w-1)
	y = min(max(y, 0), h-1)
	return t.mip[l][y*w+x]
}

var wallTextures [256]*wallTex

func wallTexture(t byte) *wallTex {
	if w := wallTextures[t]; w != nil {
		return w
	}
	return wallTextures['z']
}

// windows are the walls' textures: a shell, a log viewer, some Go, htop.
var windows = []struct {
	tile           byte
	title          string
	bg, edge, text uint32
	lines          []string
}{
	{'z', "zsh", 0x0b1020, 0x6d7ba8, 0xc8d0e8, []string{
		"$ go test ./...",
		"ok  limoni          0.412s",
		"ok  limoni/widgets  1.207s",
		"$ ./marksweep --heap 64M",
		"marking... 4242 objects",
		"sweeping... 0 B/op  $ █",
	}},
	{'l', "zest: app.log", 0x0d0f14, 0x7d6b45, 0xb8b0a0, []string{
		"12:01 INFO  frame 0 B/op",
		"12:01 WARN  bubble tea near",
		"12:02 ERROR heap pressure",
		"12:02 WARN  crab in sector 4",
		"12:03 ERROR modal won't close",
		"12:03 INFO  freed 4.2 MB",
	}},
	{'c', "gc.go", 0x10121a, 0x6a5a8a, 0xb9c4d6, []string{
		"func (g *gc) sweep() {",
		"  for _, o := range g.heap {",
		"    if !o.marked {",
		"      free(o)",
		"    }",
		"  } // 0 allocs",
	}},
	{'h', "htop", 0x07120c, 0x3f8a52, 0xa8e0b0, []string{
		"CPU [||||||||||||    62%]",
		"MEM [||||||||||||||| 88%]",
		"PID  USER   COMMAND",
		"1337 charm  bubbletea",
		"4242 ferris ratatui",
		"6060 rivo   tview",
	}},
}

func init() {
	for _, w := range windows {
		wallTextures[w.tile] = makeWindow(w.title, w.bg, w.edge, w.text, w.lines)
	}
	wallTextures['P'] = makeSpawn()
}

// window is a texture being drawn, before its mip levels are made.
type window struct {
	t    *wallTex
	pix  []rgb
	bars []rgb // the window with its text as bars, for the smaller levels
}

func (w window) set(x, y int, c rgb) {
	if x >= 0 && y >= 0 && x < w.t.w && y < w.t.h {
		w.pix[y*w.t.w+x] = c
		w.bars[y*w.t.w+x] = c
	}
}

// text draws s in the 3×5 font from (x, y), each character in its own
// colour, and returns the column after it.
func (w window) text(x, y int, s string, col func(i int) rgb) int {
	i := 0
	for _, r := range s {
		c := col(i)
		for gy := 0; gy < fontH; gy++ {
			for gx := 0; gx < fontW+1; gx++ {
				k := (y+gy)*w.t.w + x + gx
				if x+gx >= w.t.w || y+gy >= w.t.h {
					continue
				}
				if gx < fontW && fontPixel(r, gx, gy) {
					w.pix[k] = c
				}
				if r != ' ' && gy >= 1 && gy <= 3 {
					w.bars[k] = c.mul(0.8)
				}
			}
		}
		x += fontW + 1
		i++
	}
	return x
}

// frame draws a window's border, rounded, and its title bar with the three
// buttons every desktop puts there.
func (w window) frame(title string, bg, edge rgb) {
	tw, th := w.t.w, w.t.h
	for i := range w.pix {
		w.pix[i], w.bars[i] = bg, bg
	}
	bar := bg.mix(edge, 0.28)
	for y := 1; y < 9; y++ {
		for x := 1; x < tw-1; x++ {
			w.set(x, y, bar)
		}
	}
	for x := 1; x < tw-1; x++ {
		w.set(x, 0, edge)
		w.set(x, th-1, edge)
		w.set(x, 9, edge.mul(0.6))
	}
	for y := 1; y < th-1; y++ {
		w.set(0, y, edge)
		w.set(tw-1, y, edge)
	}
	// The corners stay dark: the window is rounded.
	dark := bg.mul(0.4)
	for _, c := range [4][2]int{{0, 0}, {tw - 1, 0}, {0, th - 1}, {tw - 1, th - 1}} {
		w.set(c[0], c[1], dark)
	}
	if tw >= 48 {
		for i, c := range [3]uint32{0xff5f57, 0xfebc2e, 0x28c840} {
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					w.set(4+i*4+dx, 4+dy, hex(c))
				}
			}
		}
	}
	tc := hex(0xffffff).mix(edge, 0.25)
	tx := (tw - width(title)*(fontW+1)) / 2
	w.text(tx, 2, title, func(int) rgb { return tc })
}

func newWindow(w, h int, span float64) window {
	t := &wallTex{w: w, h: h, span: span}
	return window{t: t, pix: make([]rgb, w*h), bars: make([]rgb, w*h)}
}

// done makes the mip levels, each pixel the average of four.
func (w window) done() *wallTex {
	t := w.t
	t.mip[0], t.mw[0], t.mh[0] = w.pix, t.w, t.h
	for l := 1; l < nMip; l++ {
		pw, ph := t.mw[l-1], t.mh[l-1]
		from := t.mip[l-1]
		if l == 1 {
			from = w.bars
		}
		nw, nh := max(1, (pw+1)/2), max(1, (ph+1)/2)
		m := make([]rgb, nw*nh)
		for y := 0; y < nh; y++ {
			for x := 0; x < nw; x++ {
				var s rgb
				for d := 0; d < 4; d++ {
					sx, sy := min(2*x+d%2, pw-1), min(2*y+d/2, ph-1)
					s = s.add(from[sy*pw+sx])
				}
				m[y*nw+x] = s.mul(0.25)
			}
		}
		t.mip[l], t.mw[l], t.mh[l] = m, nw, nh
	}
	return t
}

// makeWindow draws a terminal window with a title bar and lines of text,
// coloured the way a terminal would colour them.
func makeWindow(title string, bg, edge, text uint32, lines []string) *wallTex {
	w := newWindow(winW, winH, 4)
	w.frame(title, hex(bg), hex(edge))
	for i, line := range lines {
		y := 13 + i*6
		if y+fontH >= winH-1 {
			break
		}
		w.text(4, y, line, func(k int) rgb { return lineColour(line, k, hex(text)) })
	}
	return w.done()
}

// makeSpawn draws a spawn window: a narrow one, a world unit wide, where
// ones and zeroes stream in.
func makeSpawn() *wallTex {
	w := newWindow(32, winH, 1)
	edge := hex(0xff4fa0)
	w.frame("GET", hex(0x1c0716), edge)
	for y := 11; y < winH-2; y++ {
		for x := 2; x < 30; x++ {
			w.set(x, y, hex(0x1c0716).mix(hex(0x5a0b3a), float32(y-11)/float32(winH-13)))
		}
	}
	for row := 0; row < 6; row++ {
		for col := 0; col < 6; col++ {
			r := '0'
			if hash2(row, col) > 0.5 {
				r = '1'
			}
			k := 0.35 + 0.65*hash2(col, row+9)
			c := edge.mix(hex(0xffd0e8), 0.3).mul(k)
			w.text(3+col*5, 12+row*7, string(r), func(int) rgb { return c })
		}
	}
	w.t.scroll0, w.t.scroll1 = 11, winH-2
	return w.done()
}

// lineColour picks a colour for the character at column i of a line, the
// way a shell, a log viewer, an editor or htop would. It runs once, at
// start-up.
func lineColour(line string, i int, text rgb) rgb {
	words := [...]struct {
		w string
		c uint32
	}{
		{"ERROR", 0xff5a5a}, {"WARN", 0xffc94a}, {"INFO", 0x5ad1ff}, {"ok ", 0x6ee07a},
		{"func", 0xd58aff}, {"for", 0xd58aff}, {"range", 0xd58aff}, {"if", 0xd58aff},
		{"return", 0xd58aff}, {"0 B/op", 0x6ee07a}, {"0 allocs", 0x6ee07a},
		{"PID  USER   COMMAND", 0x0b2a14},
	}
	r := []rune(line)
	if i < len(r) && r[i] == '|' && strings.Contains(line, "[") {
		// An htop meter: green, then yellow, then red.
		switch {
		case i < 12:
			return hex(0x6ee07a)
		case i < 17:
			return hex(0xffc94a)
		}
		return hex(0xff5a5a)
	}
	for _, w := range words {
		ww := []rune(w.w)
		for s := 0; s+len(ww) <= len(r); s++ {
			if string(r[s:s+len(ww)]) == w.w && i >= s && i < s+len(ww) {
				return hex(w.c)
			}
		}
	}
	if len(r) > 0 && r[0] == '$' && i == 0 {
		return hex(0x6ee07a)
	}
	if k := strings.Index(line, "//"); k >= 0 && i >= len([]rune(line[:k])) {
		return hex(0x6a7590)
	}
	return text
}
