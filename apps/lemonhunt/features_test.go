package main

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// ── the squirter ─────────────────────────────────────────────────────────

func TestTheSquirterRunsDryAndReloads(t *testing.T) {
	g := playing()
	g.face(g.px+1, g.py) // down the empty corridor
	for i := 0; i < magSize; i++ {
		g.fireCD = 0
		g.fire()
	}
	if g.shots != magSize || g.ammo != 0 {
		t.Fatalf("%d squirts, %d left, want %d and 0", g.shots, g.ammo, magSize)
	}
	g.fireCD = 0
	g.fire()
	if g.shots != magSize {
		t.Fatal("squirted with an empty squirter")
	}
	if !strings.Contains(g.msg, "R to reload") {
		t.Errorf("an empty squirter says %q", g.msg)
	}

	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'r'})
	if g.reloadT <= 0 {
		t.Fatal("R did not start a reload")
	}
	// No squirting while it is being refilled.
	for g.now < reloadTime-0.1 {
		g.fireCD = 0
		g.fire()
		g.step(tick)
	}
	if g.shots != magSize || g.ammo != 0 {
		t.Fatalf("mid-reload: %d squirts, %d left", g.shots, g.ammo)
	}
	for i := 0; i < 10; i++ {
		g.step(tick)
	}
	if g.ammo != magSize || g.reloadT != 0 {
		t.Fatalf("after the reload: %d left, %.2f s to go", g.ammo, g.reloadT)
	}
	g.fireCD = 0
	g.fire()
	if g.shots != magSize+1 {
		t.Error("could not squirt after the reload")
	}
}

func TestAFullSquirterDoesNotReload(t *testing.T) {
	g := playing()
	g.reload()
	if g.reloadT != 0 {
		t.Error("a full squirter started a reload")
	}
}

// ── the rats ─────────────────────────────────────────────────────────────

// A rat close enough to bite is in sight: its head reaches above the gun,
// at the height of the player's eye.
func TestARatInBitingRangeIsInSight(t *testing.T) {
	const w, h = 120, 40
	draw := func(withRat bool) *buffer.Buffer {
		g := playing()
		g.showMap = false
		rat := g.find(kRat)
		g.px, g.py = rat.x-0.6, rat.y
		g.face(rat.x, rat.y)
		if !withRat {
			rat.state = stGone
		}
		b := buffer.NewBuffer(cell.Rect{Width: w, Height: h})
		g.render(b)
		return b
	}
	with, without := draw(true), draw(false)
	// The rows between the horizon and a little above it, across the
	// middle of the view, where the gun never reaches.
	viewH := h - hudH
	changed := 0
	for y := viewH/2 - 4; y < viewH/2; y++ {
		for x := w/2 - 6; x < w/2+6; x++ {
			if with.Content[y*w+x] != without.Content[y*w+x] {
				changed++
			}
		}
	}
	if changed < 20 {
		t.Errorf("a rat 0.6 tiles away changed %d cells at eye level, want most of 48", changed)
	}
}

// ── Ratatui ──────────────────────────────────────────────────────────────

func TestRatatuiGetsMeanerAsItWeakens(t *testing.T) {
	g := playing()
	g.gateOpen, g.gateLift = true, 1
	boss := &g.ents[g.boss]
	for _, c := range []struct{ hp, stage, cheese int }{
		{bossHP, 0, 1}, {bossHP * 2 / 3, 1, 3}, {bossHP / 3, 2, 5},
	} {
		boss.hp = c.hp
		if st := g.stage(); st != c.stage {
			t.Errorf("hp %d: stage %d, want %d", c.hp, st, c.stage)
		}
		// Make it throw, from across the room, and count the cheese.
		for i := 0; i < g.nEnts; i++ {
			if g.ents[i].kind == kCheese {
				g.ents[i].state = stGone
			}
		}
		boss.x, boss.y = 14.5, 19.5
		boss.mood, boss.moodT = bossThrow, 0
		g.bossMood(boss, 0, -4, 4, tick)
		n := 0
		for i := 0; i < g.nEnts; i++ {
			if e := &g.ents[i]; e.kind == kCheese && e.state == stAlive {
				n++
			}
		}
		if n != c.cheese {
			t.Errorf("stage %d threw %d pieces of cheese, want %d", c.stage, n, c.cheese)
		}
	}
}

// ── scores ───────────────────────────────────────────────────────────────

func TestTheScoreRewardsAimAndTime(t *testing.T) {
	run := func(shots, hits int, secs float64, won bool) result {
		g := playing()
		g.shots, g.hits, g.finished, g.kills, g.lemons = shots, hits, secs, 10, 10
		return g.score(won)
	}
	careful, sloppy := run(40, 36, 200, true), run(80, 36, 200, true)
	if careful.entry.Score <= sloppy.entry.Score {
		t.Errorf("90%% aim scored %d, 45%% scored %d", careful.entry.Score, sloppy.entry.Score)
	}
	fast, slow := run(40, 36, 200, true), run(40, 36, 400, true)
	if fast.entry.Score-slow.entry.Score != 200*timePoints {
		t.Errorf("200 s faster scored %d more, want %d", fast.entry.Score-slow.entry.Score, 200*timePoints)
	}
	if lost := run(40, 36, 200, false); lost.time != 0 || lost.entry.Score >= fast.entry.Score {
		t.Errorf("a loss scored %d time points and %d in all", lost.time, lost.entry.Score)
	}
	if r := run(0, 0, 900, true); r.aim != 0 || r.time != 0 {
		t.Errorf("no squirts and fifteen minutes: aim %d, time %d", r.aim, r.time)
	}
	if careful.entry.Aim != 90 || careful.aim != 4500 {
		t.Errorf("36 of 40: aim %d%%, %d points", careful.entry.Aim, careful.aim)
	}
}

func TestTheLeaderboardKeepsTheBest(t *testing.T) {
	g := newGame()
	for _, s := range []int{300, 900, 100, 600} {
		g.insert(scoreEntry{Name: "x", Score: s})
	}
	for i, want := range []int{900, 600, 300, 100} {
		if g.board[i].Score != want {
			t.Fatalf("place %d holds %d, want %d", i+1, g.board[i].Score, want)
		}
	}
	if rank := g.insert(scoreEntry{Score: 600}); rank != 3 {
		t.Errorf("a tie with second placed %d, want 3 (below the one already there)", rank)
	}
	for i := 0; i < 20; i++ {
		g.insert(scoreEntry{Score: 1000 + i})
	}
	if g.nBoard != boardSize || g.board[0].Score != 1019 || g.board[boardSize-1].Score != 1010 {
		t.Errorf("%d kept, best %d, last %d", g.nBoard, g.board[0].Score, g.board[boardSize-1].Score)
	}
	if rank := g.insert(scoreEntry{Score: 5}); rank != 0 {
		t.Errorf("a score below the whole board placed %d", rank)
	}
}

// memStore is a store in memory, standing in for the disk and the browser.
type memStore struct {
	data  []byte
	saves int
}

func (m *memStore) load() (string, []scoreEntry) { return decodeSaved(m.data) }
func (m *memStore) save(name string, board []scoreEntry) {
	m.data = encodeSaved(name, board)
	m.saves++
}

func TestTheNameAndTheScoresAreKept(t *testing.T) {
	mem := &memStore{}
	g := newGame()
	g.store = mem
	g.restore()
	g.askName()
	for _, r := range "Ayşe K" {
		if r == ' ' {
			g.key(limoni.KeyEvent{Type: limoni.KeySpace})
		} else {
			g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: r})
		}
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyBackspace})
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: '!'}) // not allowed
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'Z'})
	g.key(limoni.KeyEvent{Type: limoni.KeyEnter})
	if g.name != "Ayşe Z" || g.phase != phTitle {
		t.Fatalf("name %q, phase %d", g.name, g.phase)
	}

	// Win a run, and it goes on the board under that name, and is saved.
	g.reset()
	g.shots, g.hits, g.kills, g.lemons = 10, 8, 3, 10
	g.now = 120
	boss := &g.ents[g.boss]
	boss.state, boss.dying = stDying, 0.01
	g.step(tick)
	if g.phase != phWon || g.last.rank != 1 || g.nBoard != 1 {
		t.Fatalf("phase %d, rank %d, %d on the board", g.phase, g.last.rank, g.nBoard)
	}
	if mem.saves < 2 {
		t.Errorf("saved %d times, want the name and then the score", mem.saves)
	}

	// A new game, as the next time the page is opened, finds both.
	h := newGame()
	h.store = mem
	h.restore()
	if h.name != "Ayşe Z" || h.nBoard != 1 || h.board[0] != g.board[0] {
		t.Errorf("restored %q and %d scores: %+v", h.name, h.nBoard, h.board[0])
	}
}

func TestStoredNamesAreCleaned(t *testing.T) {
	name, board := decodeSaved([]byte(`{"name":" \u001b[31mEve\u0007 ","board":[{"name":"a very long name indeed","score":5}]}`))
	if got := cleanName(name); got != "31mEve" { // the escape and the bell are gone
		t.Errorf("cleaned name %q", got)
	}
	if got := cleanName(board[0].Name); got != "a very long" {
		t.Errorf("cleaned long name %q, want it cut at %d and trimmed", got, nameMax)
	}
	if n, b := decodeSaved([]byte("not json")); n != "" || b != nil {
		t.Error("garbage decoded to something")
	}
}

// ── the map ──────────────────────────────────────────────────────────────

func TestTheMapIsRoundAndOpenFromTheStart(t *testing.T) {
	const w, h = 120, 40
	g := newGame()
	if !g.showMap {
		t.Fatal("the map starts closed")
	}
	g.reset()
	draw := func(show bool) *buffer.Buffer {
		g.showMap = show
		b := buffer.NewBuffer(cell.Rect{Width: w, Height: h})
		g.render(b)
		return b
	}
	on, off := draw(true), draw(false)
	r := mmRadius((h - hudH) * 2)
	d := 2*r + 1
	x0 := w - d - 2
	at := func(b *buffer.Buffer, x, y int) cell.Cell { return b.Content[y*w+x] }
	// Its middle is covered, its top right corner is not: it is a disc.
	if at(on, x0+r, 1+r/2) == at(off, x0+r, 1+r/2) {
		t.Error("the middle of the map is not drawn")
	}
	if at(on, x0+d-1, 1) != at(off, x0+d-1, 1) {
		t.Error("the map covers its corner: it is not round")
	}
	// The player's arrow, at the centre.
	c := at(on, x0+r, 1+r/2)
	if c.Style.Fg != cell.NewColorRGB(round(0x78), round(0xf0), round(0xff)) &&
		c.Style.Bg != cell.NewColorRGB(round(0x78), round(0xf0), round(0xff)) {
		t.Errorf("no arrow at the centre of the map: %+v", c)
	}
}

// ── screens ──────────────────────────────────────────────────────────────

func TestTheEndScreenShowsTheScore(t *testing.T) {
	g := playing()
	g.name = "Deniz"
	g.insert(scoreEntry{Name: "Old", Score: 99999, Secs: 61, Aim: 80, Won: true})
	g.shots, g.hits, g.kills, g.lemons = 20, 15, 4, 10
	g.now = 95
	g.phase = phWon
	g.finished = g.now
	g.record(true)
	g.now += 2
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.render(b)
	s := screen(b)
	for _, want := range []string{"RATATUI IS DEFEATED", "SCORE", "Number 2 on the leaderboard", "Deniz", "Old", "99999", "1:01", "75%"} {
		if !strings.Contains(s, want) {
			t.Errorf("the end screen lacks %q:\n%s", want, s)
		}
	}
}

func TestTheTitleShowsTheBestScores(t *testing.T) {
	g := newGame()
	g.name = "Can"
	g.insert(scoreEntry{Name: "Mert", Score: 7777, Secs: 200, Aim: 64, Won: true})
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.render(b)
	s := screen(b)
	for _, want := range []string{"BEST SCORES", "Mert", "7777", "NAME", "Can"} {
		if !strings.Contains(s, want) {
			t.Errorf("the title lacks %q", want)
		}
	}
}

func TestTheNameEntryAsksForAName(t *testing.T) {
	g := newGame()
	g.askName()
	g.key(limoni.KeyEvent{Type: limoni.KeyEnter})
	if g.phase != phName {
		t.Error("ENTER with no name left the name entry")
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'w'}) // a letter, not the menu
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.render(b)
	if s := screen(b); !strings.Contains(s, "WHAT IS YOUR NAME?") || g.typing != "w" {
		t.Errorf("typing %q on:\n%s", g.typing, s)
	}
}

func TestMenuAndNameEntryAllocateNothing(t *testing.T) {
	for _, sc := range []struct {
		name  string
		setup func(g *game)
	}{
		{"name entry", func(g *game) { g.askName(); g.typing = "Ayşe" }},
		{"title with scores", func(g *game) {
			for i := 0; i < boardSize; i++ {
				g.insert(scoreEntry{Name: "Someone", Score: 1000 * i, Secs: 100 + i, Aim: 50})
			}
		}},
		{"reloading", func(g *game) { g.reset(); g.ammo = 0; g.reload() }},
		{"won with scores", func(g *game) {
			g.reset()
			g.insert(scoreEntry{Name: "Someone", Score: 5000})
			g.finished = 100
			g.record(true)
			g.phase = phWon
			g.now = 200
		}},
	} {
		t.Run(sc.name, func(t *testing.T) {
			g := newGame()
			sc.setup(g)
			b := buffer.NewBuffer(cell.Rect{Width: 140, Height: 44})
			g.render(b) // warm up
			if avg := testing.AllocsPerRun(50, func() { g.step(tick); g.render(b) }); avg != 0 {
				t.Errorf("%.2f allocations per frame, want 0", avg)
			}
		})
	}
}
