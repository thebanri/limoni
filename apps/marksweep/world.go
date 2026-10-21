package main

import (
	"math"
	"strings"
)

// The arena is the inside of a terminal: its walls are windows — a shell,
// a log viewer, some Go, htop — and the frameworks come in through the
// spawn windows (P).

const mapW, mapH = 32, 32

var arenaMap = [mapH]string{
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
}

func tile(x, y int) byte {
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return 'z'
	}
	return arenaMap[y][x]
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

const texW, texH = 36, 12

// texSpan is how many world units one window spans along a wall.
const texSpan = 3.0

type wallTex struct {
	glyph [texH][texW]rune
	col   [texH][texW]rgb
	ink   [texH][texW]float32 // how much of the cell the glyph covers
	bg    rgb
	edge  rgb
}

var wallTextures = map[byte]*wallTex{}

func wallTexture(t byte) *wallTex {
	if w := wallTextures[t]; w != nil {
		return w
	}
	return wallTextures['z']
}

func init() {
	wallTextures['z'] = makeWindow("zsh", 0x0b1020, 0x6d7ba8, 0xc8d0e8, []string{
		"$ go build ./...",
		"$ go test ./...",
		"ok   limoni          0.412s",
		"ok   limoni/widgets  1.207s",
		"$ ./marksweep --heap 64M",
		"marking... 4242 objects",
		"sweeping... 0 B/op",
		"$ █",
	})
	wallTextures['l'] = makeWindow("zest: app.log", 0x0d0f14, 0x7d6b45, 0xb8b0a0, []string{
		"12:01:07 INFO  frame 0 B/op",
		"12:01:07 WARN  bubble tea nearby",
		"12:01:08 ERROR heap pressure high",
		"12:01:08 INFO  sweeping...",
		"12:01:09 WARN  crab in sector 4",
		"12:01:09 ERROR modal won't close",
		"12:01:10 INFO  freed 4.2 MB",
		"12:01:11 WARN  </> detected",
	})
	wallTextures['c'] = makeWindow("gc.go", 0x10121a, 0x6a5a8a, 0xb9c4d6, []string{
		"func (g *gc) sweep() {",
		"    for _, o := range g.heap {",
		"        if !o.marked {",
		"            free(o)",
		"        }",
		"    }",
		"    return // 0 allocs",
		"}",
	})
	wallTextures['h'] = makeWindow("htop", 0x07120c, 0x3f8a52, 0xa8e0b0, []string{
		"CPU [||||||||||||     62.1%]",
		"MEM [|||||||||||||||| 88.4%]",
		"PID  USER   COMMAND",
		"1337 charm  bubbletea",
		"4242 ferris ratatui",
		"6060 rivo   tview",
		"7070 vadim  ink",
		"  1  root   ncurses",
	})
	wallTextures['P'] = makeWindow("spawn", 0x1c0716, 0xff4fa0, 0xffd0e8, []string{
		"$ go get charm.land/bubbletea",
		"$ cargo add ratatui",
		"$ npm i ink",
		"$ go get rivo/tview",
		"",
		"  ▓▓▓▓▓▓ INCOMING ▓▓▓▓▓▓",
		"",
		"$ █",
	})
}

// makeWindow draws a terminal window with a title bar and lines of text,
// coloured the way a terminal would colour them.
func makeWindow(title string, bg, edge, text uint32, lines []string) *wallTex {
	t := &wallTex{bg: hex(bg), edge: hex(edge)}
	for y := 0; y < texH; y++ {
		for x := 0; x < texW; x++ {
			t.glyph[y][x] = ' '
			t.col[y][x] = hex(text)
		}
	}
	put := func(x, y int, r rune, c rgb) {
		t.glyph[y][x], t.col[y][x] = r, c
	}
	for x := 1; x < texW-1; x++ {
		put(x, 0, '─', t.edge)
		put(x, texH-1, '─', t.edge)
	}
	for y := 1; y < texH-1; y++ {
		put(0, y, '│', t.edge)
		put(texW-1, y, '│', t.edge)
	}
	put(0, 0, '╭', t.edge)
	put(texW-1, 0, '╮', t.edge)
	put(0, texH-1, '╰', t.edge)
	put(texW-1, texH-1, '╯', t.edge)
	x := 2
	for _, r := range " " + title + " " {
		put(x, 0, r, hex(0xffffff).mix(t.edge, 0.3))
		x++
	}
	for i, line := range lines {
		y := i + 2
		if y >= texH-1 {
			break
		}
		x := 2
		for _, r := range line {
			if x >= texW-2 {
				break
			}
			put(x, y, r, lineColour(line, x-2, hex(text)))
			x++
		}
	}
	for y := 0; y < texH; y++ {
		for x := 0; x < texW; x++ {
			t.ink[y][x] = ink(t.glyph[y][x])
		}
	}
	return t
}

// lineColour picks a colour for the character at column i of a line, the
// way a shell, a log viewer or an editor would. It runs once, at start-up.
func lineColour(line string, i int, text rgb) rgb {
	words := [...]struct {
		w string
		c uint32
	}{
		{"ERROR", 0xff5a5a}, {"WARN", 0xffc94a}, {"INFO", 0x5ad1ff}, {"ok ", 0x6ee07a},
		{"func", 0xd58aff}, {"for", 0xd58aff}, {"range", 0xd58aff}, {"if", 0xd58aff},
		{"return", 0xd58aff}, {"INCOMING", 0xff4fa0}, {"0 B/op", 0x6ee07a}, {"0 allocs", 0x6ee07a},
	}
	r := []rune(line)
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

// ink is how much of its cell a glyph covers, for drawing a far wall as
// shading rather than unreadable letters.
func ink(r rune) float32 {
	switch {
	case r == ' ':
		return 0
	case r == '█' || r == '▓':
		return 1
	case r == '─' || r == '│' || r == '╭' || r == '╮' || r == '╰' || r == '╯':
		return 0.45
	case r == '.' || r == ',' || r == '\'' || r == ':':
		return 0.15
	case r >= 'A' && r <= 'Z' || r >= '0' && r <= '9':
		return 0.6
	}
	return 0.5
}
