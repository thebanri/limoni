package main

import (
	"math"
	"strings"
	"testing"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

const tick = 1.0 / 30

func playing() *game {
	g := newGame()
	g.reset()
	return g
}

func screen(b *buffer.Buffer) string {
	var sb strings.Builder
	w := int(b.Area.Width)
	for i, c := range b.Content {
		if i > 0 && i%w == 0 {
			sb.WriteByte('\n')
		}
		sb.WriteRune(c.Content)
	}
	return sb.String()
}

// face points the player at (x, y).
func (g *game) face(x, y float64) {
	dx, dy := x-g.px, y-g.py
	d := math.Hypot(dx, dy)
	g.dirX, g.dirY = dx/d, dy/d
	g.planeX, g.planeY = -g.dirY*fov, g.dirX*fov
}

func (g *game) find(k kind) *ent {
	for i := 0; i < g.nEnts; i++ {
		if g.ents[i].kind == k && g.ents[i].state == stAlive {
			return &g.ents[i]
		}
	}
	return nil
}

// ── the level ────────────────────────────────────────────────────────────

func TestLevelIsWellFormed(t *testing.T) {
	if len(level) != mapH {
		t.Fatalf("%d rows, want %d", len(level), mapH)
	}
	count := map[byte]int{}
	for y, row := range level {
		if len(row) != mapW {
			t.Fatalf("row %d is %d wide, want %d", y, len(row), mapW)
		}
		for x := 0; x < mapW; x++ {
			c := row[x]
			count[c]++
			edge := x == 0 || y == 0 || x == mapW-1 || y == mapH-1
			if edge && strings.IndexByte("#SPL", c) < 0 {
				t.Errorf("(%d,%d) = %q: the border must be wall", x, y, c)
			}
		}
	}
	if count['@'] != 1 || count['R'] != 1 || count['G'] != 1 {
		t.Errorf("want one start, one Ratatui and one gate; got %d, %d, %d", count['@'], count['R'], count['G'])
	}
	if count['l'] < lemonsNeeded {
		t.Errorf("%d lemons, the gate needs %d", count['l'], lemonsNeeded)
	}

	// Every lemon can be reached with the gate shut, and Ratatui only once
	// it opens.
	g := playing()
	reach := func() [mapH][mapW]bool {
		var seen [mapH][mapW]bool
		queue := [][2]int{{int(g.px), int(g.py)}}
		seen[int(g.py)][int(g.px)] = true
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, y := p[0]+d[0], p[1]+d[1]
				if !g.solid(x, y) && !seen[y][x] {
					seen[y][x] = true
					queue = append(queue, [2]int{x, y})
				}
			}
		}
		return seen
	}
	shut := reach()
	boss := g.find(kBoss)
	for i := 0; i < g.nEnts; i++ {
		if e := g.ents[i]; e.kind == kLemon && !shut[int(e.y)][int(e.x)] {
			t.Errorf("the lemon at (%.1f,%.1f) cannot be reached", e.x, e.y)
		}
	}
	if shut[int(boss.y)][int(boss.x)] {
		t.Error("Ratatui can be reached before the gate opens")
	}
	g.gateOpen, g.gateLift = true, 1
	if open := reach(); !open[int(boss.y)][int(boss.x)] {
		t.Error("Ratatui cannot be reached once the gate opens")
	}
}

// ── zero allocations ─────────────────────────────────────────────────────

// frames runs the game the way main does — keys, a tick, a render — and
// sends each frame through Limoni's diff, as the terminal would.
type frames struct {
	g         *game
	cur, prev *buffer.Buffer
	out       []byte
	n         int
	releases  bool // the terminal reports key releases
}

func newFrames(g *game, w, h uint16) *frames {
	r := cell.Rect{Width: w, Height: h}
	return &frames{g: g, cur: buffer.NewBuffer(r), prev: buffer.NewBuffer(r), out: make([]byte, 0, 1<<16)}
}

func (f *frames) frame() {
	g := f.g
	// A scripted player: walks, turns, strafes and squirts. With releases
	// it holds keys down; without, it leans on auto-repeat.
	w := limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'w'}
	right := limoni.KeyEvent{Type: limoni.KeyRight}
	a := limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'a'}
	space := limoni.KeyEvent{Type: limoni.KeySpace}
	let := func(k limoni.KeyEvent) limoni.KeyEvent { k.Release = true; return k }
	if f.releases {
		switch f.n % 90 {
		case 0:
			g.key(w)
		case 20:
			g.key(let(w))
		case 30:
			g.key(right)
		case 38:
			g.key(let(right))
		case 45:
			g.key(a)
		case 52:
			g.key(let(a))
		case 60:
			g.key(space)
		case 75:
			g.key(let(space))
		case 85:
			g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'm'})
		}
	} else {
		switch f.n % 90 {
		case 0, 3, 6, 9, 12, 15:
			g.key(w)
		case 30, 33, 36:
			g.key(right)
		case 45, 48:
			g.key(a)
		case 60, 70, 80:
			g.key(space)
		case 85:
			g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'm'})
		}
	}
	f.n++
	g.step(tick)
	g.render(f.cur)
	f.out, _ = buffer.DiffWithOptions(f.cur, f.prev, f.out[:0], buffer.DiffOptions{TrueColor: true, EraseChar: true, RepeatChar: true})
	f.cur, f.prev = f.prev, f.cur
}

func TestFramesAllocateNothing(t *testing.T) {
	scenes := []struct {
		name  string
		setup func(g *game)
	}{
		{"title", func(g *game) { g.phase = phTitle }},
		{"sewer", func(g *game) {}},
		{"sewer with releases", func(g *game) {}},
		{"lair", func(g *game) {
			g.gateOpen = true
			g.px, g.py = 10.5, 17.5
			g.face(14.5, 19.5)
		}},
		{"won", func(g *game) { g.phase, g.bossSeen = phWon, true }},
		{"dead", func(g *game) { g.phase = phDead }},
	}
	for _, sc := range scenes {
		t.Run(sc.name, func(t *testing.T) {
			g := playing()
			g.hp = 1 << 20 // the scripted player must survive the measurement
			sc.setup(g)
			f := newFrames(g, 140, 44)
			f.releases = strings.HasSuffix(sc.name, "releases")
			// The first frames fill the diff's style cache: one entry per
			// style ever drawn, and never more than the palette.
			for i := 0; i < 400; i++ {
				f.frame()
			}
			if avg := testing.AllocsPerRun(300, f.frame); avg != 0 {
				t.Errorf("%.2f allocations per frame, want 0", avg)
			}
		})
	}
}

func TestNoCellCarriesAModifier(t *testing.T) {
	// The diff builds a style from scratch, through its cache, only when a
	// modifier such as bold has to be switched off. With a picture of free
	// colours that cache would fill with styles never seen again, and
	// allocate. So nothing on screen is bold, dim or italic.
	g := playing()
	g.gateOpen, g.bossSeen, g.showMap = true, true, true
	g.say("a message")
	b := buffer.NewBuffer(cell.Rect{Width: 140, Height: 44})
	for _, ph := range []phase{phTitle, phPlay, phWon, phDead} {
		g.phase, g.finished = ph, g.now-5
		g.render(b)
		for i, c := range b.Content {
			if c.Style.Modifier != 0 {
				t.Fatalf("phase %d: cell %d (%q) has modifier %v", ph, i, c.Content, c.Style.Modifier)
			}
		}
	}
}

// ── play ─────────────────────────────────────────────────────────────────

func TestTenLemonsOpenTheGate(t *testing.T) {
	g := playing()
	gx, gy := -1, -1
	for y := 0; y < mapH; y++ {
		if x := strings.IndexByte(level[y], 'G'); x >= 0 {
			gx, gy = x, y
		}
	}
	for n := 0; n < lemonsNeeded; n++ {
		if g.gateOpen {
			t.Fatalf("the gate opened after %d lemons", n)
		}
		l := g.find(kLemon)
		g.px, g.py = l.x, l.y
		g.step(tick)
	}
	if g.lemons != lemonsNeeded || !g.gateOpen {
		t.Fatalf("lemons %d, gate open %v", g.lemons, g.gateOpen)
	}
	if !g.solid(gx, gy) {
		t.Error("the gate let the player through before it had lifted")
	}
	for i := 0; i < 60; i++ {
		g.step(tick)
	}
	if g.solid(gx, gy) {
		t.Error("the gate is still solid two seconds after it opened")
	}
}

func TestSquirtKillsARatInTwo(t *testing.T) {
	g := playing()
	rat := g.find(kRat)
	g.px, g.py = rat.x-2, rat.y
	g.face(rat.x, rat.y)
	for i := 0; i < ratHP; i++ {
		if rat.state != stAlive {
			t.Fatalf("the rat died after %d squirts", i)
		}
		g.fireCD = 0
		g.fire()
	}
	if rat.state != stDying {
		t.Fatalf("rat state %d after %d squirts", rat.state, ratHP)
	}
	for i := 0; i < 30; i++ {
		g.step(tick)
	}
	if rat.state != stGone || g.kills != 1 {
		t.Errorf("state %d, kills %d", rat.state, g.kills)
	}
}

func TestWallsStopTheSquirt(t *testing.T) {
	g := playing()
	// The rat in the room east of the start; the player on the other side
	// of the wall at x = 14.
	var rat *ent
	for i := 0; i < g.nEnts; i++ {
		if e := &g.ents[i]; e.kind == kRat && int(e.y) == 2 {
			rat = e
		}
	}
	g.px, g.py = 12.5, rat.y
	g.face(rat.x, rat.y)
	g.fire()
	if rat.hp != ratHP || g.hits != 0 {
		t.Errorf("hit through a wall: hp %d, hits %d", rat.hp, g.hits)
	}
}

func TestWallsStopThePlayer(t *testing.T) {
	g := playing()
	g.px, g.py = 2.5, 2.5
	g.face(g.px+1, g.py) // the start room's east wall is at x = 5
	for i := 0; i < 120; i++ {
		g.press(actFwd)
		g.step(tick)
	}
	if g.px > 5-radius+1e-9 {
		t.Errorf("walked into the wall: x = %.3f", g.px)
	}
	if g.px < 4 {
		t.Errorf("did not walk to the wall: x = %.3f", g.px)
	}
}

func TestATapMovesAndStops(t *testing.T) {
	g := playing()
	x0 := g.px
	g.press(actFwd)
	for i := 0; i < 60; i++ {
		g.step(tick)
	}
	moved := g.px - x0
	if moved < 0.5 || moved > 2.2 {
		t.Errorf("one tap moved %.2f tiles", moved)
	}
	x1 := g.px
	for i := 0; i < 30; i++ {
		g.step(tick)
	}
	if g.px != x1 {
		t.Errorf("still moving two seconds after the tap")
	}
}

func TestRatsBiteAndTheGameEnds(t *testing.T) {
	g := playing()
	rat := g.find(kRat)
	g.px, g.py = rat.x+0.4, rat.y
	for i := 0; i < 30*30 && g.phase == phPlay; i++ {
		g.px, g.py = rat.x+0.4, rat.y // stand still next to it
		g.step(tick)
	}
	if g.phase != phDead || g.hp != 0 {
		t.Fatalf("phase %d, hp %d", g.phase, g.hp)
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'r'})
	if g.phase != phPlay || g.hp != 100 || g.lemons != 0 {
		t.Errorf("after R: phase %d, hp %d, lemons %d", g.phase, g.hp, g.lemons)
	}
}

func TestRatatuiSleepsUntilTheGateOpens(t *testing.T) {
	g := playing()
	boss := g.find(kBoss)
	x, y := boss.x, boss.y
	for i := 0; i < 60; i++ {
		g.step(tick)
	}
	if boss.x != x || boss.y != y || g.bossSeen {
		t.Error("Ratatui stirred before the gate opened")
	}
}

func TestDefeatingRatatuiWins(t *testing.T) {
	g := playing()
	g.hp = 1 << 20
	g.gateOpen = true
	boss := g.find(kBoss)
	// Squirt until it falls; the rats it summons get in the way and take
	// some of the squirts.
	for i := 0; i < 200 && boss.state == stAlive; i++ {
		boss.x, boss.y = 14.5, 19.5 // hold it still
		g.px, g.py = 14.5, 17.5
		g.face(boss.x, boss.y)
		g.fireCD = 0
		g.fire()
		g.step(tick)
	}
	if boss.state != stDying {
		t.Fatalf("Ratatui state %d, hp %d after 200 squirts", boss.state, boss.hp)
	}
	if g.hits < bossHP {
		t.Errorf("Ratatui fell after %d hits, want at least %d", g.hits, bossHP)
	}
	if g.summoned != 2 {
		t.Errorf("Ratatui called for help %d times, want 2", g.summoned)
	}
	for i := 0; i < 150; i++ {
		g.step(tick)
	}
	if g.phase != phWon {
		t.Fatalf("phase %d after Ratatui fell", g.phase)
	}

	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.render(b)
	if s := screen(b); !strings.Contains(s, "RATATUI IS DEFEATED") {
		t.Errorf("no victory screen:\n%s", s)
	}
}

func TestTheBossBarFollowsItsHealth(t *testing.T) {
	g := playing()
	g.gateOpen, g.bossSeen = true, true
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	bar := func() int {
		g.render(b)
		n := 0
		for x := 0; x < 120; x++ {
			if c := b.Content[x]; c.Content == '█' {
				n++
			}
		}
		return n
	}
	full := bar()
	g.ents[g.boss].hp = bossHP / 2
	half := bar()
	if full == 0 || half*2 < full-1 || half*2 > full+1 {
		t.Errorf("bar is %d cells full and %d at half health", full, half)
	}
}

func TestSmallWindowSaysSo(t *testing.T) {
	g := playing()
	b := buffer.NewBuffer(cell.Rect{Width: 40, Height: 12})
	g.render(b)
	if !strings.Contains(screen(b), "60×20") {
		t.Error("a small window does not say how large it must be")
	}
}

// ── benchmarks ───────────────────────────────────────────────────────────

func BenchmarkFrame(b *testing.B) {
	g := playing()
	g.hp = 1 << 20
	buf := buffer.NewBuffer(cell.Rect{Width: 160, Height: 48})
	b.ReportAllocs()
	for b.Loop() {
		g.press(actTurnR)
		g.step(tick)
		g.render(buf)
	}
}

func BenchmarkFrameWithDiff(b *testing.B) {
	g := playing()
	g.hp = 1 << 20
	f := newFrames(g, 160, 48)
	for i := 0; i < 400; i++ {
		f.frame()
	}
	b.ReportAllocs()
	sent, n := 0, 0
	for b.Loop() {
		f.frame()
		sent += len(f.out)
		n++
	}
	b.ReportMetric(float64(sent)/float64(n), "bytes/frame")
}

func TestRatatuiFindsItsWayRoundThePillars(t *testing.T) {
	g := playing()
	g.hp = 1 << 20
	g.openGate()
	boss := &g.ents[g.boss]
	// Behind the pillar at (20, 20), the player east of it.
	boss.x, boss.y = 19.4, 19.85
	g.px, g.py = 26.5, 19.5
	for i := 0; i < 30*14; i++ {
		g.step(tick)
		if math.Hypot(boss.x-g.px, boss.y-g.py) < 1.1 {
			return
		}
	}
	t.Errorf("Ratatui is stuck at (%.2f, %.2f)", boss.x, boss.y)
}

// With a terminal that reports releases, a key is held exactly as long as
// it is: no auto-repeat needed to keep walking, and a stop on the release.
func TestHeldKeysWalkUntilReleased(t *testing.T) {
	g := playing()
	w := limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'w'}
	g.key(w)
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'w', Release: true}) // proves releases arrive
	x0 := g.px
	g.key(w)
	for i := 0; i < 45; i++ { // 1.5 s, not a single repeat
		g.step(tick)
	}
	walked := g.px - x0
	if walked < 3.5 {
		t.Fatalf("walked %.2f tiles holding w for 1.5 s, want about 4", walked)
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'w', Release: true})
	x1 := g.px
	for i := 0; i < 30; i++ {
		g.step(tick)
	}
	if coast := g.px - x1; coast > 0.4 {
		t.Errorf("coasted %.2f tiles after the release", coast)
	}
}

func TestHoldingSpaceKeepsSquirting(t *testing.T) {
	g := playing()
	g.key(limoni.KeyEvent{Type: limoni.KeySpace})
	g.key(limoni.KeyEvent{Type: limoni.KeySpace, Release: true})
	shots := g.shots
	g.key(limoni.KeyEvent{Type: limoni.KeySpace})
	for i := 0; i < 30; i++ {
		g.step(tick)
	}
	if n := g.shots - shots; n < 4 {
		t.Errorf("%d squirts in a second of holding space", n)
	}
}

func TestTheTitleMenu(t *testing.T) {
	g := newGame()
	press := func(k limoni.KeyEvent) { g.key(k); g.key(func() limoni.KeyEvent { k.Release = true; return k }()) }
	down := limoni.KeyEvent{Type: limoni.KeyDown}
	enter := limoni.KeyEvent{Type: limoni.KeyEnter}

	press(down) // SOUND
	press(limoni.KeyEvent{Type: limoni.KeyRight})
	press(limoni.KeyEvent{Type: limoni.KeyRight})
	press(limoni.KeyEvent{Type: limoni.KeyRight})
	press(limoni.KeyEvent{Type: limoni.KeyRight})
	if g.volume != 10 {
		t.Errorf("volume %d after turning it up past the top, want 10", g.volume)
	}
	press(enter) // off
	if g.volume != 0 || g.phase != phTitle {
		t.Errorf("ENTER on SOUND: volume %d, phase %d", g.volume, g.phase)
	}
	press(enter) // on again, where it was
	if g.volume != 10 {
		t.Errorf("sound came back at %d, want 10", g.volume)
	}
	press(down) // QUIT
	press(down) // round to START
	press(enter)
	if g.phase != phPlay || g.quit {
		t.Fatalf("START: phase %d, quit %v", g.phase, g.quit)
	}
	if g.volume != 10 {
		t.Errorf("starting the game reset the volume to %d", g.volume)
	}

	g = newGame()
	g.key(limoni.KeyEvent{Type: limoni.KeyUp}) // round to QUIT
	g.key(enter)
	if !g.quit {
		t.Error("ENTER on QUIT did not quit")
	}
}

func TestTheTitleScreenShowsTheMenu(t *testing.T) {
	g := newGame()
	g.noSound = "no player found"
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.render(b)
	s := screen(b)
	for _, want := range []string{"START", "SOUND", "QUIT", "no player found", "ENTER select"} {
		if !strings.Contains(s, want) {
			t.Errorf("the title screen has no %q", want)
		}
	}
}
