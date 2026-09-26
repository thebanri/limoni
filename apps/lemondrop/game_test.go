package main

import (
	"testing"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// playing is a run in progress on a 4-grain board, with nothing on it but
// the first piece.
func playing() *game {
	g := newGame(42)
	g.lastW, g.lastH = 100, 40
	g.start()
	if g.b != 4 {
		panic("a 100×40 window should give a 4-grain board")
	}
	return g
}

func grains(g *game) (n int) {
	for _, v := range g.sand[:g.gw*g.gh] {
		if v != 0 {
			n++
		}
	}
	return n
}

func (g *game) at(x, y int) uint8 { return g.sand[y*g.gw+x] }

// settle runs the sand alone until it stops, or fails.
func settle(t *testing.T, g *game) {
	t.Helper()
	g.wakeAll()
	for i := 0; i < 2000; i++ {
		g.sandTick()
		if g.settled {
			return
		}
	}
	t.Fatal("the sand never settled")
}

// ── the sand ─────────────────────────────────────────────────────────────

func TestSandFallsAndIsConserved(t *testing.T) {
	g := playing()
	g.phase = phTitle // no piece, no scoring
	// A tower of mixed colours, two grains wide, in the middle of the air.
	for y := 5; y < 25; y++ {
		g.sand[y*g.gw+20] = uint8(1 + y%3)
		g.sand[y*g.gw+21] = uint8(1 + (y+1)%3)
	}
	before := grains(g)
	settle(t, g)
	if n := grains(g); n != before {
		t.Fatalf("%d grains before the fall, %d after", before, n)
	}
	// It has spread into a heap on the floor: wider than two, and nothing
	// hangs over an empty cell.
	wide := 0
	for x := 0; x < g.gw; x++ {
		if g.at(x, g.gh-1) != 0 {
			wide++
		}
	}
	if wide <= 2 {
		t.Errorf("the tower did not spread: %d grains on the floor", wide)
	}
	for y := 0; y < g.gh-1; y++ {
		for x := 0; x < g.gw; x++ {
			if g.at(x, y) != 0 && g.at(x, y+1) == 0 {
				t.Fatalf("a grain at (%d,%d) rests on nothing", x, y)
			}
		}
	}
}

// A heap of sand leans at most one grain a column: a grain with room
// below and to a side moves on.
func TestSandSlidesOffAStep(t *testing.T) {
	g := playing()
	g.phase = phTitle
	y := g.gh - 1
	g.sand[y*g.gw+10] = 1
	g.sand[(y-1)*g.gw+10] = 2
	settle(t, g)
	if g.at(10, y-1) != 0 {
		t.Fatal("a grain stayed balanced on another with room on both sides")
	}
	if grains(g) != 2 {
		t.Fatalf("%d grains, want 2", grains(g))
	}
}

// A landed piece is sand at once: its grains do not keep the block's
// bevel, lit on the top and left and dark on the bottom and right.
func TestALandedBlockLosesItsBlockShading(t *testing.T) {
	g := playing()
	g.cur.kind, g.cur.rot = 3, 0 // the O: four blocks, all flat on the floor
	g.hardDrop()
	bevel, n := 0, 0
	for y := 0; y < g.gh; y++ {
		for x := 0; x < g.gw; x++ {
			v := g.at(x, y)
			if v == 0 {
				continue
			}
			n++
			// Where the bevel would have put a shade, and whether it did.
			lx, ly := (x-g.cur.x)%g.b, y%g.b
			if lx < 0 {
				lx += g.b
			}
			if v>>shadeShift&3 == shadeAt(lx, ly, g.b) {
				bevel++
			}
		}
	}
	if n == 0 || bevel*10 > n*6 {
		t.Fatalf("%d of %d grains still carry the block's shading", bevel, n)
	}
}

// An upright I, four blocks tall and one wide, does not stay a pillar: it
// slumps into a heap wider than it was.
func TestAPillarOfSandSlumps(t *testing.T) {
	g := playing()
	g.phase = phTitle
	g.cur = piece{kind: 0, rot: 1, colour: 2, x: 16}
	g.cur.y = g.dropY()
	g.lock()
	settle(t, g)
	wide := 0
	for x := 0; x < g.gw; x++ {
		if g.at(x, g.gh-1) != 0 {
			wide++
		}
	}
	if wide <= 2*g.b {
		t.Fatalf("the pillar spread to %d grains on the floor, want more than %d", wide, 2*g.b)
	}
	if n := grains(g); n != 4*g.b*g.b {
		t.Fatalf("%d grains after the slump, want %d", n, 4*g.b*g.b)
	}
}

// Sand speeds up as it falls, and it takes a visible while to cross the
// board: a grain let go at the top reaches the floor in more than half a
// second and less than two.
func TestSandFallsGathersSpeed(t *testing.T) {
	g := playing()
	g.phase = phTitle
	g.sand[10] = 1
	g.wakeAll()
	var at [3]int // steps to fall a third, two thirds, all the way
	for step := 1; step < 1000 && at[2] == 0; step++ {
		g.sandTick()
		for y := 0; y < g.gh; y++ {
			if g.at(10, y) == 0 {
				continue
			}
			for k := range at {
				if at[k] == 0 && y >= (k+1)*(g.gh-1)/3 {
					at[k] = step
				}
			}
		}
	}
	secs := float64(at[2]) / hz
	if secs < 0.5 || secs > 2 {
		t.Fatalf("a grain fell the board in %.2f s", secs)
	}
	if first, last := at[0], at[2]-at[1]; last >= first {
		t.Errorf("the first third took %d steps and the last %d: no acceleration", first, last)
	}
}

// ── clears ───────────────────────────────────────────────────────────────

func stripe(g *game, y, x0, x1 int, v uint8) {
	for x := x0; x < x1; x++ {
		g.sand[y*g.gw+x] = v
	}
}

func TestAWallToWallRunClears(t *testing.T) {
	g := playing()
	y := g.gh - 1
	// One colour along the floor, but not straight: it steps up a row in
	// the middle, touching only at a corner, which still joins.
	stripe(g, y, 0, 20, 2)
	stripe(g, y-1, 20, g.gw, 2)
	stripe(g, y, 20, g.gw, 3)
	g.wakeAll()
	g.findClear()
	if g.flashT == 0 {
		t.Fatal("a run from wall to wall did not start to clear")
	}
	if g.score == 0 || g.clears != 1 {
		t.Fatalf("score %d, clears %d after a clear", g.score, g.clears)
	}
	for i := 0; i < int(flashTime*hz)+2; i++ {
		g.sandTick()
	}
	for x := 0; x < g.gw; x++ {
		if v := g.at(x, y); v&colourMask == 2 {
			t.Fatalf("a grain of the run is still at (%d,%d)", x, y)
		}
	}
	// The other colour is untouched.
	n := 0
	for _, v := range g.sand[:g.gw*g.gh] {
		if v&colourMask == 3 {
			n++
		}
	}
	if n != g.gw-20 {
		t.Fatalf("%d grains of the other colour, want %d", n, g.gw-20)
	}
}

func TestARunThatStopsShortDoesNotClear(t *testing.T) {
	g := playing()
	y := g.gh - 1
	stripe(g, y, 0, g.gw-1, 1) // one grain short of the right wall
	stripe(g, y, g.gw-1, g.gw, 2)
	g.findClear()
	if g.flashT != 0 || g.score != 0 {
		t.Fatal("a run that does not reach the right wall cleared")
	}
	// Nor does a full row of two colours.
	g = playing()
	stripe(g, y, 0, 20, 1)
	stripe(g, y, 20, g.gw, 2)
	g.findClear()
	if g.flashT != 0 {
		t.Fatal("a row of two colours cleared")
	}
}

// What rested on a clear falls, and a second clear before the next piece
// lands scores as a chain.
func TestAChainScoresMore(t *testing.T) {
	g := playing()
	y := g.gh - 1
	stripe(g, y, 0, g.gw, 1)    // clears first
	stripe(g, y-1, 0, 20, 2)    // falls on the floor, with a gap…
	stripe(g, y-1, 21, g.gw, 2) //
	g.sand[20] = 2              // …that this grain, from the top, fills
	g.wakeAll()
	for i := 0; i < 600 && g.clears < 2; i++ {
		g.sandTick()
	}
	if g.clears != 2 || g.combo != 2 {
		t.Fatalf("clears %d, combo %d, want a chain of 2", g.clears, g.combo)
	}
	if g.gainCombo != 2 {
		t.Errorf("the second clear was scored as combo %d", g.gainCombo)
	}
}

// ── pieces ───────────────────────────────────────────────────────────────

func TestALandedPieceBecomesSand(t *testing.T) {
	g := playing()
	c := g.cur.colour
	g.hardDrop()
	if n := grains(g); n != 4*g.b*g.b {
		t.Fatalf("%d grains after one piece, want %d", n, 4*g.b*g.b)
	}
	for _, v := range g.sand[:g.gw*g.gh] {
		if v != 0 && v&colourMask != c {
			t.Fatalf("a grain of colour %d from a piece of colour %d", v&colourMask, c)
		}
	}
	// It landed on the floor.
	floor := 0
	for x := 0; x < g.gw; x++ {
		if g.at(x, g.gh-1) != 0 {
			floor++
		}
	}
	if floor == 0 {
		t.Fatal("a dropped piece did not reach the floor")
	}
}

func TestPiecesFallAndLockByThemselves(t *testing.T) {
	g := playing()
	for i := 0; i < 60*20 && grains(g) == 0; i++ {
		g.tick()
	}
	if grains(g) != 4*g.b*g.b {
		t.Fatalf("after falling by itself, %d grains", grains(g))
	}
}

func TestTurningOffAWallKicks(t *testing.T) {
	g := playing()
	g.cur.kind, g.cur.rot = 0, 1 // the I, upright
	for g.shift(-1) {
	}
	if !g.turn(1) {
		t.Fatal("an I against the left wall could not turn")
	}
	if !g.fits(g.cur.kind, g.cur.rot, g.cur.x, g.cur.y) {
		t.Fatal("the turn left the piece where it does not fit")
	}
}

func TestMovingStopsAtTheWalls(t *testing.T) {
	g := playing()
	for g.shift(-g.step()) {
	}
	for _, c := range shapes[g.cur.kind][g.cur.rot] {
		if g.cur.x+int(c[0])*g.b < 0 {
			t.Fatal("the piece went through the left wall")
		}
	}
	x := g.cur.x
	if g.shift(-1) || g.cur.x != x {
		t.Fatal("moved into the wall")
	}
}

func TestAFullBoardEndsTheRun(t *testing.T) {
	g := playing()
	for i := range g.sand[:g.gw*g.gh] {
		if i/g.gw > 2*g.b { // leave a little room at the top
			g.sand[i] = uint8(1 + (i/7)%4)
		}
	}
	for i := 0; i < 50 && g.phase == phPlay; i++ {
		g.hardDrop()
	}
	if g.phase != phOver {
		t.Fatal("the run did not end on a full board")
	}
}

func TestBagDealsEveryShape(t *testing.T) {
	g := playing()
	g.nbag = 0 // start from a fresh bag
	var seen [7]int
	for i := 0; i < 70; i++ {
		p := g.draw()
		seen[p.kind]++
		if p.colour < 1 || p.colour > colours {
			t.Fatalf("colour %d", p.colour)
		}
	}
	for k, n := range seen {
		if n != 10 {
			t.Errorf("shape %d dealt %d times in ten bags", k, n)
		}
	}
}

func TestShapesTurnAboutTheirBox(t *testing.T) {
	for k := range shapes {
		for r := range shapes[k] {
			for _, c := range shapes[k][r] {
				if c[0] < 0 || c[0] > 3 || c[1] < 0 || c[1] > 3 {
					t.Fatalf("shape %d rotation %d leaves the box: %v", k, r, c)
				}
			}
		}
		if masks[k][0] == 0 {
			t.Fatalf("shape %d has no mask", k)
		}
	}
	// Four turns come back to the start, and a T turns into a new shape.
	if masks[5][1] == masks[5][0] {
		t.Error("a T turned into itself")
	}
}

// ── keys ─────────────────────────────────────────────────────────────────

func TestHeldArrowsRepeatWhereReleasesAreReported(t *testing.T) {
	g := playing()
	g.exact = true
	x := g.cur.x
	g.key(limoni.KeyEvent{Type: limoni.KeyLeft})
	if g.cur.x != x-g.step() {
		t.Fatal("← did not move the piece")
	}
	// The browser page sends the browser's auto-repeat as more presses of a
	// key it has not released: the game's own repeat stands in for them.
	g.key(limoni.KeyEvent{Type: limoni.KeyLeft})
	g.key(limoni.KeyEvent{Type: limoni.KeyLeft, Repeat: true})
	if g.cur.x != x-g.step() {
		t.Fatal("a repeat of a held ← moved the piece as well")
	}
	for i := 0; i < 20; i++ {
		g.tick()
	}
	held := g.cur.x
	if held >= x-g.step() {
		t.Fatal("a held ← did not repeat")
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyLeft, Release: true})
	for i := 0; i < 20; i++ {
		g.tick()
	}
	if g.cur.x != held {
		t.Fatal("the piece kept moving after ← was let go")
	}
}

func TestPauseStopsTime(t *testing.T) {
	g := playing()
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'p'})
	y := g.cur.y
	g.update(1)
	if g.cur.y != y {
		t.Fatal("the piece fell while paused")
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'P'})
	for i := 0; i < 30; i++ {
		g.update(dt)
	}
	if g.cur.y == y {
		t.Fatal("the piece did not fall after the pause")
	}
}

// ── the picture ──────────────────────────────────────────────────────────

func TestTheBoardSizeFollowsTheWindow(t *testing.T) {
	for _, c := range []struct{ w, h, b int }{
		{80, 24, 2}, {44, 20, 2}, {43, 20, 0}, {44, 19, 0},
		{100, 40, 4}, {200, 60, 6},
	} {
		if b := fitSize(c.w, c.h); b != c.b {
			t.Errorf("%d×%d: %d grains a block, want %d", c.w, c.h, b, c.b)
		}
	}
}

// The lemon behind the board shows, faintly: the middle of the board is
// yellower than its corner, and still darker than any grain.
func TestTheLemonIsBehindTheBoard(t *testing.T) {
	g := playing()
	g.buildBackground()
	yellow := func(c cell.Color) (float64, uint8) {
		r, gg, b := c.RGB()
		return float64(r) + float64(gg) - 2*float64(b), max(r, gg, b)
	}
	cx, cy, R := g.gw/2, int(float64(g.gh)*0.56), int(float64(g.gw)*0.44)
	var mid, edge float64
	var n int
	var brightest uint8
	for y := cy - R/2; y <= cy+R/2; y++ {
		for x := cx - R/2; x <= cx+R/2; x++ {
			v, l := yellow(g.bg[y*g.gw+x])
			mid += v
			brightest = max(brightest, l)
			n++
		}
	}
	mid /= float64(n)
	edge, _ = yellow(g.bg[0])
	if mid <= edge+10 {
		t.Fatalf("the middle of the board (%.1f) is not yellower than its corner (%.1f)", mid, edge)
	}
	colours := map[cell.Color]bool{}
	for _, c := range g.bg[:g.gw*g.gh] {
		colours[c] = true
	}
	if len(colours) > 64 {
		t.Errorf("the background has %d colours; each is a glyph pair more in xterm.js's atlas", len(colours))
	}
	if brightest >= 90 {
		t.Errorf("the lemon is too strong: a channel at %d", brightest)
	}
}

func screen(b *buffer.Buffer) string {
	out := make([]rune, 0, len(b.Content)+int(b.Area.Height))
	for y := 0; y < int(b.Area.Height); y++ {
		for x := 0; x < int(b.Area.Width); x++ {
			r := b.Content[y*int(b.Area.Width)+x].Content
			if r == 0 {
				r = ' '
			}
			out = append(out, r)
		}
		out = append(out, '\n')
	}
	return string(out)
}

func TestScreensSayWhatToDo(t *testing.T) {
	g := newGame(1)
	b := buffer.NewBuffer(cell.Rect{Width: 100, Height: 40})
	g.render(b)
	if s := screen(b); !contains(s, "ENTER  play") || !contains(s, "LEMON DROP") {
		t.Fatalf("the title screen:\n%s", s)
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyEnter})
	if g.phase != phPlay || g.b != 4 {
		t.Fatalf("Enter did not start a 4-grain run: phase %d, b %d", g.phase, g.b)
	}
	small := buffer.NewBuffer(cell.Rect{Width: 60, Height: 22})
	g.render(small)
	if s := screen(small); !contains(s, "enlarge") || !g.paused {
		t.Fatalf("a window too small for the run did not pause it:\n%s", s)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestNoCellCarriesAModifier(t *testing.T) {
	// The diff builds a style through its cache, which allocates, only when
	// a modifier has to be switched off. Nothing here is bold or dim.
	g := playing()
	b := buffer.NewBuffer(cell.Rect{Width: 100, Height: 40})
	for _, ph := range []phase{phTitle, phPlay, phOver} {
		g.phase, g.overAt = ph, g.now-5
		g.render(b)
		for i, c := range b.Content {
			if c.Style.Modifier != 0 {
				t.Fatalf("phase %d: cell %d has a modifier", ph, i)
			}
		}
	}
}

// ── zero allocations ─────────────────────────────────────────────────────

// frames runs the game the way main does — keys, a step, a render — and
// sends each frame through Limoni's diff, as the terminal would.
type frames struct {
	g         *game
	cur, prev *buffer.Buffer
	out       []byte
	n         int
}

func newFrames(g *game, w, h uint16) *frames {
	r := cell.Rect{Width: w, Height: h}
	return &frames{g: g, cur: buffer.NewBuffer(r), prev: buffer.NewBuffer(r), out: make([]byte, 0, 1<<16)}
}

func (f *frames) frame() {
	g := f.g
	switch f.n % 40 {
	case 0:
		g.key(limoni.KeyEvent{Type: limoni.KeyLeft})
	case 5:
		g.key(limoni.KeyEvent{Type: limoni.KeyUp})
	case 10, 12:
		g.key(limoni.KeyEvent{Type: limoni.KeyRight})
	case 20:
		g.key(limoni.KeyEvent{Type: limoni.KeyDown})
	case 30:
		g.key(limoni.KeyEvent{Type: limoni.KeySpace})
	}
	f.n++
	g.update(dt)
	if g.phase == phOver {
		g.start()
	}
	g.render(f.cur)
	f.out, _ = buffer.DiffWithOptions(f.cur, f.prev, f.out[:0], buffer.DiffOptions{TrueColor: true, EraseChar: true, RepeatChar: true})
	f.cur, f.prev = f.prev, f.cur
}

func TestFramesAllocateNothing(t *testing.T) {
	for _, sc := range []struct {
		name  string
		w, h  uint16
		setup func(g *game)
	}{
		{"title", 100, 40, func(g *game) { g.phase = phTitle }},
		{"play", 100, 40, func(g *game) {}},
		{"play, large", 200, 60, func(g *game) {}},
		{"play, small", 80, 24, func(g *game) {}},
		{"paused", 100, 40, func(g *game) { g.paused = true }},
	} {
		t.Run(sc.name, func(t *testing.T) {
			g := newGame(7)
			g.store = store{} // a game over must not write a file here
			g.lastW, g.lastH = int(sc.w), int(sc.h)
			g.start()
			sc.setup(g)
			f := newFrames(g, sc.w, sc.h)
			for i := 0; i < 300; i++ {
				f.frame()
			}
			if avg := testing.AllocsPerRun(300, f.frame); avg != 0 {
				t.Errorf("%.2f allocations per frame, want 0", avg)
			}
		})
	}
}

// BenchmarkFrame is a frame of play: a simulation step, a render and the
// diff, at 100×40 with sand on the board.
func BenchmarkFrame(b *testing.B) {
	g := newGame(3)
	g.store = store{}
	g.lastW, g.lastH = 100, 40
	g.start()
	f := newFrames(g, 100, 40)
	for i := 0; i < 600; i++ {
		f.frame()
	}
	b.ReportAllocs()
	b.ResetTimer()
	sent := 0
	for i := 0; i < b.N; i++ {
		f.frame()
		sent += len(f.out)
	}
	b.ReportMetric(float64(sent)/float64(b.N), "bytes/frame")
}

// BenchmarkSandPass is the worst pass: every row of the largest board
// awake, half of it sand, and the search for a clear.
func BenchmarkSandPass(b *testing.B) {
	g := newGame(5)
	g.setSize(6)
	for i := range g.sand[:g.gw*g.gh/2] {
		g.sand[i] = uint8(1 + i%colours)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.wakeAll()
		g.fall()
		g.findClear()
	}
}
