// The shared board and the stores, against a real HTTP server and real
// files: not in a browser.

//go:build !js

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// serverScore is the leaderboard's rule (apps/scoreboard/board/drop.go),
// written out again: the score it works out from what the run sent.
func serverScore(r remoteRun) int {
	s := r.Drops
	for i, c := range r.Clears {
		s += int(c[0]) * (1 + (i+1)/4) * int(c[1])
	}
	return s
}

func (g *game) run() remoteRun {
	return remoteRun{Name: g.name, Secs: g.runSecs(), Pieces: g.pieces, Drops: g.dropPts, Clears: g.clearLog[:g.nlog]}
}

// Whatever a run scores, the leaderboard scores the same from its log:
// drops, clears at every level, and chains.
func TestTheLogScoresAsTheGameDoes(t *testing.T) {
	g := playing()
	y := g.gh - 1
	for i := 0; i < 9; i++ {
		g.hardDrop()
		g.softStep()
		// A clear; every third one followed by a chain.
		stripe(g, y, 0, g.gw, 1)
		g.findClear()
		g.flashT = 0
		if i%3 == 2 {
			stripe(g, y-1, 0, g.gw, 2)
			g.findClear()
			g.flashT = 0
		}
		clear(g.sand[:])
	}
	if g.clears < 12 || g.level < 4 {
		t.Fatalf("the scripted run made %d clears, level %d", g.clears, g.level)
	}
	if got := serverScore(g.run()); got != g.score {
		t.Fatalf("the leaderboard would score %d, the game scored %d", got, g.score)
	}
}

func TestTheNameIsTypedAndKept(t *testing.T) {
	g := newGame(1)
	g.store = store{path: filepath.Join(t.TempDir(), "lemondrop.json")}
	g.askName()
	for _, r := range "Q m-ü!" { // q and m are keys elsewhere; ! is not allowed
		g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: r})
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyBackspace})
	g.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'x'})
	g.key(limoni.KeyEvent{Type: limoni.KeyEnter})
	if g.quit || g.muted {
		t.Fatal("typing a name quit the game or switched the sound off")
	}
	if g.name != "Q m-x" || g.phase != phTitle {
		t.Fatalf("name %q, phase %d", g.name, g.phase)
	}
	again := newGame(2)
	again.store = g.store
	again.restore()
	if again.name != "Q m-x" {
		t.Fatalf("the name after a restart: %q", again.name)
	}
}

func TestRunsGoOnTheGamesOwnBoard(t *testing.T) {
	g := playing()
	g.store = store{path: filepath.Join(t.TempDir(), "lemondrop.json")}
	g.name = "Ada"
	for _, s := range []int{300, 900, 100} {
		g.start()
		g.score = s
		g.gameOver()
	}
	if g.nLocal != 3 || g.local[0].Score != 900 || g.local[2].Score != 100 || g.localRank != 3 {
		t.Fatalf("the board: %+v, the last run's place %d", g.local[:g.nLocal], g.localRank)
	}
	again := newGame(3)
	again.store = g.store
	again.restore()
	if again.nLocal != 3 || again.best != 900 || again.local[1].Name != "Ada" {
		t.Fatalf("after a restart: %+v, best %d", again.local[:again.nLocal], again.best)
	}
}

// The game against a stand-in for the leaderboard: it fetches the board on
// start, sends a run when it ends, and shows the world's board and its
// place, all without a frame waiting.
func TestTheSharedBoard(t *testing.T) {
	var mu sync.Mutex
	var got []remoteRun
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/drop/scores" {
			http.NotFound(w, r)
			return
		}
		board := []scoreEntry{{Name: "Ece", Score: 5000, Clears: 20, Level: 6, Secs: 200}}
		rank := 0
		if r.Method == "POST" {
			var run remoteRun
			if err := json.NewDecoder(r.Body).Decode(&run); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			mu.Lock()
			got = append(got, run)
			mu.Unlock()
			board = append(board, scoreEntry{Name: run.Name, Score: serverScore(run), Clears: len(run.Clears)})
			rank = 2
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"rank": rank, "board": board})
	}))
	defer srv.Close()

	g := playing()
	g.name = "Ada"
	g.remote = newRemote(srv.URL + "/")
	g.worldState = worldLoading
	g.remote.fetch()
	wait := func(what string, done func() bool) {
		t.Helper()
		for end := time.Now().Add(5 * time.Second); !done(); {
			if time.Now().After(end) {
				t.Fatalf("waited for %s", what)
			}
			g.tick()
			time.Sleep(time.Millisecond)
		}
	}
	wait("the board", func() bool { return g.worldState == worldOK })
	if g.nWorld != 1 || g.world[0].Name != "Ece" {
		t.Fatalf("the world's board: %+v", g.world[:g.nWorld])
	}

	stripe(g, g.gh-1, 0, g.gw, 1)
	g.findClear()
	g.hardDrop()
	score := g.score
	g.gameOver()
	if g.worldRank != -1 {
		t.Fatalf("the run was not sent: rank %d", g.worldRank)
	}
	wait("the run's place", func() bool { return g.worldRank > 0 })
	mu.Lock()
	sent := got[0]
	mu.Unlock()
	if sent.Name != "Ada" || len(sent.Clears) != 1 || serverScore(sent) != score {
		t.Fatalf("sent %+v for a run of %d", sent, score)
	}
	if g.worldRank != 2 || g.nWorld != 2 {
		t.Fatalf("place %d on a board of %d", g.worldRank, g.nWorld)
	}

	// The end screen shows it.
	g.overAt = g.now - 5
	b := buffer.NewBuffer(cell.Rect{Width: 100, Height: 40})
	g.render(b)
	s := screen(b)
	for _, want := range []string{"GAME OVER", "#2 world", "WORLD'S BEST", "Ece", "Ada"} {
		if !strings.Contains(s, want) {
			t.Errorf("the end screen has no %q:\n%s", want, s)
		}
	}
}

// A leaderboard that does not answer leaves the game its own board.
func TestAnAbsentBoardFallsBackToTheGamesOwn(t *testing.T) {
	g := playing()
	g.name = "Ada"
	g.remote = newRemote("http://127.0.0.1:1")
	g.worldState = worldLoading
	g.remote.fetch()
	for end := time.Now().Add(5 * time.Second); g.worldState == worldLoading; {
		if time.Now().After(end) {
			t.Fatal("no answer from an address nobody listens on")
		}
		g.tick()
		time.Sleep(time.Millisecond)
	}
	if g.worldState != worldOffline {
		t.Fatalf("state %d", g.worldState)
	}
	g.phase = phTitle
	b := buffer.NewBuffer(cell.Rect{Width: 100, Height: 40})
	g.render(b)
	if s := screen(b); !strings.Contains(s, "offline") {
		t.Errorf("the title does not say the board is offline:\n%s", s)
	}
}
