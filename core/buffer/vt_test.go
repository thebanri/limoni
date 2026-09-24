package buffer

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/cell"
)

// vt is a small model of a terminal: enough of VT100/ECMA-48 to replay
// what the diff emits — cursor moves, SGR, ECH, REP, EL, scroll regions and
// SU/SD — so a test can check that the bytes rebuild the frame instead of
// trusting their shape.
type vt struct {
	t          *testing.T
	w, h       int
	cells      []rune
	styled     []bool // cell written while an SGR other than reset was active
	x, y       int
	top, bot   int
	sgrActive  bool
	last       rune
	scrollUsed int
	shiftsUsed int
}

func newVT(t *testing.T, w, h int) *vt {
	v := &vt{t: t, w: w, h: h, cells: make([]rune, w*h), styled: make([]bool, w*h), bot: h - 1}
	for i := range v.cells {
		v.cells[i] = ' '
	}
	return v
}

// load puts a buffer's content on the screen, as if a previous frame had
// drawn it.
func (v *vt) load(b *Buffer) {
	for i, c := range b.Content {
		r := c.Content
		if r == 0 || r == cell.RuneContinuation {
			r = ' '
		}
		v.cells[i] = r
	}
}

func (v *vt) put(r rune) {
	if v.x >= v.w {
		return // the diff never relies on autowrap
	}
	v.cells[v.y*v.w+v.x] = r
	v.styled[v.y*v.w+v.x] = v.sgrActive
	v.last = r
	v.x += max(1, cell.RuneWidth(r))
}

func (v *vt) scroll(n int) { // positive: up (SU)
	w := v.w
	for i := 0; i < abs(n); i++ {
		if n > 0 {
			copy(v.cells[v.top*w:v.bot*w], v.cells[(v.top+1)*w:(v.bot+1)*w])
			copy(v.styled[v.top*w:v.bot*w], v.styled[(v.top+1)*w:(v.bot+1)*w])
			v.fillRow(v.bot)
		} else {
			copy(v.cells[(v.top+1)*w:(v.bot+1)*w], v.cells[v.top*w:v.bot*w])
			copy(v.styled[(v.top+1)*w:(v.bot+1)*w], v.styled[v.top*w:v.bot*w])
			v.fillRow(v.top)
		}
	}
}

func (v *vt) fillRow(y int) {
	for x := 0; x < v.w; x++ {
		v.cells[y*v.w+x] = ' '
		v.styled[y*v.w+x] = v.sgrActive
	}
}

func (v *vt) feed(out []byte) {
	for i := 0; i < len(out); {
		switch out[i] {
		case '\r':
			v.x = 0
			i++
			continue
		case '\n':
			if v.y == v.bot {
				v.scroll(1)
			} else if v.y < v.h-1 {
				v.y++
			}
			i++
			continue
		}
		if out[i] != 0x1b {
			r, size := utf8.DecodeRune(out[i:])
			v.put(r)
			i += size
			continue
		}
		if i+1 < len(out) && out[i+1] == ']' { // OSC: skip to ST or BEL
			j := i + 2
			for j < len(out) && out[j] != 0x07 && !(out[j] == 0x1b && j+1 < len(out) && out[j+1] == '\\') {
				j++
			}
			if j < len(out) && out[j] == 0x1b {
				j++
			}
			i = j + 1
			continue
		}
		if i+1 >= len(out) || out[i+1] != '[' {
			v.t.Fatalf("unmodelled escape at %d: %q", i, out[i:min(len(out), i+8)])
		}
		j := i + 2
		for j < len(out) && (out[j] < 0x40 || out[j] > 0x7E) {
			j++
		}
		params, final := string(out[i+2:j]), out[j]
		i = j + 1
		if strings.HasPrefix(params, "?") {
			continue // private modes: ?2026 and friends
		}
		nums := strings.Split(params, ";")
		arg := func(k, def int) int {
			if k < len(nums) && nums[k] != "" {
				n, _ := strconv.Atoi(nums[k])
				return n
			}
			return def
		}
		switch final {
		case 'H':
			v.y, v.x = arg(0, 1)-1, arg(1, 1)-1
		case 'm':
			v.sgrActive = !(params == "" || params == "0")
		case 'X':
			for k := 0; k < arg(0, 1) && v.x+k < v.w; k++ {
				v.cells[v.y*v.w+v.x+k] = ' '
				v.styled[v.y*v.w+v.x+k] = v.sgrActive
			}
		case 'K':
			for x := v.x; x < v.w; x++ {
				v.cells[v.y*v.w+x] = ' '
			}
		case 'b':
			for k := 0; k < arg(0, 1); k++ {
				v.put(v.last)
			}
		case '@', 'P': // ICH, DCH: shift the rest of the row, open blanks
			if v.sgrActive {
				v.t.Error("inserted or deleted with a style active: the opened cells take its background")
			}
			n, row := arg(0, 1), v.cells[v.y*v.w:(v.y+1)*v.w]
			if final == '@' {
				copy(row[v.x+n:], row[v.x:v.w-n])
				for x := v.x; x < v.x+n; x++ {
					row[x] = ' '
				}
			} else {
				copy(row[v.x:], row[v.x+n:])
				for x := v.w - n; x < v.w; x++ {
					row[x] = ' '
				}
			}
			v.shiftsUsed++
		case 'r':
			v.top, v.bot = arg(0, 1)-1, arg(1, v.h)-1
			v.x, v.y = 0, 0
		case 'S', 'T':
			if v.sgrActive {
				v.t.Error("scrolled with a style active: the new rows take its background")
			}
			n := arg(0, 1)
			if final == 'T' {
				n = -n
			}
			v.scroll(n)
			v.scrollUsed++
		default:
			v.t.Fatalf("unmodelled CSI %q%c", params, final)
		}
	}
}

func (v *vt) matches(b *Buffer) (int, int, bool) {
	for i, c := range b.Content {
		r := c.Content
		if r == 0 || r == cell.RuneContinuation {
			r = ' '
		}
		if v.cells[i] != r {
			return i % v.w, i / v.w, false
		}
	}
	return 0, 0, true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func randomRow(rng *rand.Rand, b *Buffer, y int) {
	w := int(b.Area.Width)
	for x := 0; x < w; x++ {
		c := cell.Cell{Content: rune('a' + rng.Intn(26))}
		c.Style.Reset()
		if rng.Intn(5) == 0 {
			c.Style.Fg = cell.NewColorRGB(uint8(rng.Intn(255)), 0, 0)
		}
		b.Content[y*w+x] = c
	}
}

// Whatever shape the change takes — a scroll up or down of part of the
// screen, with or without edits around it — replaying the diff over the old
// screen must give the new one.
func TestScrollDiffRebuildsTheFrame(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	scrolled := 0
	defer func() {
		// The trials are only worth something if they exercise the scroll.
		if scrolled < 150 {
			t.Errorf("only %d of 400 trials scrolled", scrolled)
		}
	}()
	for trial := 0; trial < 400; trial++ {
		w, h := 5+rng.Intn(30), 3+rng.Intn(25)
		area := cell.NewRect(0, 0, uint16(w), uint16(h))
		back, front := NewBuffer(area), NewBuffer(area)
		for y := 0; y < h; y++ {
			if rng.Intn(6) != 0 { // leave some rows blank
				randomRow(rng, back, y)
			}
		}
		copy(front.Content, back.Content)

		// Scroll a band by k, as a list or log would.
		top := rng.Intn(h - 1)
		bot := top + 1 + rng.Intn(h-top-1)
		k := 1 + rng.Intn(bot-top)
		if rng.Intn(2) == 0 {
			copy(front.Content[top*w:(bot+1-k)*w], back.Content[(top+k)*w:(bot+1)*w])
			for y := bot + 1 - k; y <= bot; y++ {
				randomRow(rng, front, y)
			}
		} else {
			copy(front.Content[(top+k)*w:(bot+1)*w], back.Content[top*w:(bot+1-k)*w])
			for y := top; y < top+k; y++ {
				randomRow(rng, front, y)
			}
		}
		for e := rng.Intn(4); e > 0; e-- { // and some edits anywhere
			front.Content[rng.Intn(w*h)] = cell.Cell{Content: '#'}
		}
		front.IsDirty = true

		screen := newVT(t, w, h)
		screen.load(back)
		out, err := DiffWithOptions(front, back, nil, DiffOptions{TrueColor: true, EraseChar: true, RepeatChar: true, ScrollRegions: true})
		if err != nil {
			t.Fatal(err)
		}
		screen.feed(out)
		scrolled += min(screen.scrollUsed, 1)
		if x, y, ok := screen.matches(front); !ok {
			t.Fatalf("trial %d (%dx%d, band %d-%d by %d): screen differs from the frame at %d,%d\noutput %q", trial, w, h, top, bot, k, x, y, out)
		}
		if x, y, ok := screen.matches(back); !ok {
			t.Fatalf("trial %d: back buffer does not describe the screen at %d,%d", trial, x, y)
		}
	}
}

// A log that gained a line at the bottom scrolls instead of redrawing, and
// costs a fraction of the bytes.
func TestScrollDiffSavesBytes(t *testing.T) {
	w, h := 80, 24
	area := cell.NewRect(0, 0, uint16(w), uint16(h))
	// Every line different, as log lines are.
	paths := []string{"/api/items", "/login", "/static/app.js", "/api/orders/17", "/health", "/api/users"}
	line := func(b *Buffer, y, n int) {
		b.SetString(0, uint16(y), "2026-09-24 12:00:"+strconv.Itoa(10+n%50)+" INFO GET "+paths[n%len(paths)]+" "+strconv.Itoa(200+n*7%300)+" in "+strconv.Itoa(n*13%97)+"ms", cell.Style{})
	}
	sizes := map[bool]int{}
	for _, scroll := range []bool{false, true} {
		back, front := NewBuffer(area), NewBuffer(area)
		for y := 0; y < h; y++ {
			line(back, y, y)
			line(front, y, y+1)
		}
		screen := newVT(t, w, h)
		screen.load(back)
		out, _ := DiffWithOptions(front, back, nil, DiffOptions{TrueColor: true, EraseChar: true, ScrollRegions: scroll})
		screen.feed(out)
		if _, _, ok := screen.matches(front); !ok {
			t.Fatalf("scroll=%v: wrong screen", scroll)
		}
		if scroll && screen.scrollUsed != 1 {
			t.Errorf("did not scroll")
		}
		sizes[scroll] = len(out)
	}
	if sizes[true]*5 > sizes[false] {
		t.Errorf("scrolling a log by one line: %d bytes with scroll regions, %d without", sizes[true], sizes[false])
	}
	t.Logf("one line of log: %d bytes with scroll regions, %d without", sizes[true], sizes[false])
}

func TestScrollDiffDoesNotAllocate(t *testing.T) {
	area := cell.NewRect(0, 0, 80, 24)
	back, front := NewBuffer(area), NewBuffer(area)
	rng := rand.New(rand.NewSource(2))
	for y := 0; y < 24; y++ {
		randomRow(rng, back, y)
	}
	out := make([]byte, 0, 64<<10)
	opts := DiffOptions{TrueColor: true, EraseChar: true, ScrollRegions: true}
	shift := func() {
		copy(front.Content, back.Content[80:])
		randomRow(rng, front, 23)
		front.IsDirty = true
	}
	shift()
	out, _ = DiffWithOptions(front, back, out[:0], opts)
	if n := testing.AllocsPerRun(20, func() {
		shift()
		out, _ = DiffWithOptions(front, back, out[:0], opts)
	}); n != 0 {
		t.Errorf("%.0f allocs per scrolled diff", n)
	}
}

// Text typed into or deleted from the middle of a line: replaying the diff
// must give the new line, whether or not ICH or DCH was used for it.
func TestInsertDeleteDiffRebuildsTheLine(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	shifted := 0
	for trial := 0; trial < 500; trial++ {
		w, h := 20+rng.Intn(60), 1+rng.Intn(4)
		area := cell.NewRect(0, 0, uint16(w), uint16(h))
		back, front := NewBuffer(area), NewBuffer(area)
		y := rng.Intn(h)
		textLen := 5 + rng.Intn(w-5)
		for x := 0; x < textLen; x++ {
			back.Content[y*w+x].Content = rune('a' + rng.Intn(26))
		}
		copy(front.Content, back.Content)
		row := front.Content[y*w : (y+1)*w]
		at, k := rng.Intn(textLen), 1+rng.Intn(10)
		if rng.Intn(2) == 0 { // insert k characters at at
			copy(row[min(w, at+k):], row[at:max(at, w-k)])
			for x := at; x < min(w, at+k); x++ {
				row[x].Content = 'X'
			}
		} else { // delete k characters at at
			copy(row[at:], row[min(w, at+k):])
			for x := max(at, w-k); x < w; x++ {
				row[x].Reset()
				if rng.Intn(3) == 0 {
					// Text scrolled in from the right, as in a scrolled
					// input: DCH would blank these, so it must not be used.
					row[x].Content = 'Z'
				}
			}
		}
		if rng.Intn(4) == 0 {
			row[rng.Intn(w)].Style.Fg = cell.NewColorRGB(9, 9, 9) // a style change too
		}
		front.IsDirty = true

		screen := newVT(t, w, h)
		screen.load(back)
		out, _ := DiffWithOptions(front, back, nil, DiffOptions{TrueColor: true, EraseChar: true, RepeatChar: true, InsertDelete: true})
		screen.feed(out)
		shifted += min(screen.shiftsUsed, 1)
		if x, yy, ok := screen.matches(front); !ok {
			t.Fatalf("trial %d: screen differs at %d,%d\nback  %q\nfront %q\nout %q", trial, x, yy, rowText(back, y), rowText(front, y), out)
		}
		if x, yy, ok := screen.matches(back); !ok {
			t.Fatalf("trial %d: back buffer does not describe the screen at %d,%d", trial, x, yy)
		}
	}
	if shifted < 100 {
		t.Errorf("only %d of 500 trials used ICH or DCH", shifted)
	}
}

func rowText(b *Buffer, y int) string {
	w := int(b.Area.Width)
	var s []rune
	for _, c := range b.Content[y*w : (y+1)*w] {
		s = append(s, c.Content)
	}
	return string(s)
}

// Typing one character at the start of a long line writes the character,
// not the line. (On a one-row screen that line is most of the frame and the
// diff streams the whole frame instead, so this is a full-size screen.)
func TestInsertDeleteSavesBytes(t *testing.T) {
	area := cell.NewRect(0, 0, 80, 24)
	text := "the quick brown fox jumps over the lazy dog, again and again and again"
	sizes := map[bool]int{}
	for _, on := range []bool{false, true} {
		back, front := NewBuffer(area), NewBuffer(area)
		back.SetString(0, 5, text, cell.Style{})
		front.SetString(0, 5, "A"+text, cell.Style{})
		screen := newVT(t, 80, 24)
		screen.load(back)
		out, _ := DiffWithOptions(front, back, nil, DiffOptions{TrueColor: true, EraseChar: true, InsertDelete: on})
		screen.feed(out)
		if _, _, ok := screen.matches(front); !ok {
			t.Fatalf("insert=%v: wrong screen", on)
		}
		sizes[on] = len(out)
	}
	if sizes[true]*4 > sizes[false] {
		t.Errorf("one typed character: %d bytes with ICH, %d without", sizes[true], sizes[false])
	}
	t.Logf("one typed character: %d bytes with ICH, %d without", sizes[true], sizes[false])
}

// Delete under a text cursor: the cursor stays on its column and highlights
// the next character, so the first cell differs from the shifted row. DCH is
// still used, and the text that scrolls in at the right edge is written.
func TestDeleteUnderCursorUsesDCH(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 10)
	back, front := NewBuffer(area), NewBuffer(area)
	text := "ABCthe quick brown fox jumps over the lazy dog"
	cursor := cell.Style{}
	cursor.Reset()
	cursor = cursor.Reverse()
	back.SetString(0, 3, text, cell.Style{})
	back.Get(3, 3).Style = cursor
	front.SetString(0, 3, "ABChe quick brown fox jumps over the lazy dog", cell.Style{})
	front.Get(3, 3).Style = cursor
	screen := newVT(t, 40, 10)
	screen.load(back)
	out, _ := DiffWithOptions(front, back, nil, DiffOptions{TrueColor: true, EraseChar: true, InsertDelete: true})
	screen.feed(out)
	if _, _, ok := screen.matches(front); !ok {
		t.Fatalf("wrong screen: %q", out)
	}
	if screen.shiftsUsed != 1 {
		t.Errorf("DCH not used: %q", out)
	}
}
