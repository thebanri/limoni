package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// fakeBoard stands in for apps/scoreboard: it answers with a fixed board,
// and a fixed place for a run sent, and remembers what it was sent.
type fakeBoard struct {
	mu   sync.Mutex
	sent []map[string]any
	rank int
}

func (f *fakeBoard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/scores" {
		http.NotFound(w, r)
		return
	}
	board := []scoreEntry{
		{Name: "Zeynep", Score: 12000, Secs: 150, Aim: 91, Won: true},
		{Name: "Mert", Score: 9000, Secs: 240, Aim: 70, Won: true},
	}
	answer := map[string]any{"board": board}
	if r.Method == "POST" {
		body, _ := io.ReadAll(r.Body)
		var run map[string]any
		_ = json.Unmarshal(body, &run)
		f.mu.Lock()
		f.sent = append(f.sent, run)
		f.mu.Unlock()
		answer["rank"] = f.rank
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answer)
}

// waitFor steps the game until the world board settles, or fails.
func waitFor(t *testing.T, g *game, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatalf("no answer from the board: state %d, rank %d", g.worldState, g.worldRank)
		}
		time.Sleep(5 * time.Millisecond)
		g.step(tick)
	}
}

func TestTheWorldBoardIsFetchedAndShown(t *testing.T) {
	srv := httptest.NewServer(&fakeBoard{})
	defer srv.Close()
	g := newGame()
	g.name = "Can"
	g.remote = newRemote(srv.URL + "/") // a trailing slash is forgiven
	g.worldState = worldLoading
	g.remote.fetch()
	waitFor(t, g, func() bool { return g.worldState == worldOK })
	if g.nWorld != 2 || g.world[0].Name != "Zeynep" {
		t.Fatalf("world board %+v", g.world[:g.nWorld])
	}
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.render(b)
	if s := screen(b); !strings.Contains(s, "WORLD BEST") || !strings.Contains(s, "Zeynep") || !strings.Contains(s, "12000") {
		t.Errorf("the title does not show the world's board:\n%s", s)
	}
}

func TestARunIsSentAndPlaced(t *testing.T) {
	fake := &fakeBoard{rank: 3}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	g := playing()
	g.name = "Deniz"
	g.remote = newRemote(srv.URL)
	g.worldState = worldLoading
	g.shots, g.hits, g.kills, g.lemons = 30, 20, 9, 10
	g.now, g.finished, g.phase = 180.4, 180.4, phWon
	g.record(true)
	if g.worldRank != -1 {
		t.Fatalf("while sending, rank %d, want -1", g.worldRank)
	}
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.now += 2
	g.render(b)
	if s := screen(b); !strings.Contains(s, "Sending the score") {
		t.Errorf("the end screen does not say the score is on its way:\n%s", s)
	}
	waitFor(t, g, func() bool { return g.worldRank != -1 })
	if g.worldRank != 3 || g.worldState != worldOK {
		t.Fatalf("placed %d, state %d", g.worldRank, g.worldState)
	}
	fake.mu.Lock()
	run := fake.sent[0]
	fake.mu.Unlock()
	want := map[string]any{"name": "Deniz", "won": true, "secs": 180.0, "shots": 30.0, "hits": 20.0, "kills": 9.0, "lemons": 10.0}
	for k, v := range want {
		if run[k] != v {
			t.Errorf("sent %s = %v, want %v", k, run[k], v)
		}
	}
	if _, ok := run["score"]; ok {
		t.Error("the client sent a score; the server works it out")
	}
	g.render(b)
	if s := screen(b); !strings.Contains(s, "Number 3 in the world") || !strings.Contains(s, "WORLD BEST") {
		t.Errorf("the end screen after the answer:\n%s", s)
	}
}

func TestAnUnreachableBoardFallsBackToTheLocalOne(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens there now
	g := playing()
	g.name = "Ece"
	g.remote = newRemote(url)
	g.worldState = worldLoading
	g.shots, g.hits = 10, 5
	g.now, g.finished, g.phase = 60, 60, phDead
	g.record(false)
	waitFor(t, g, func() bool { return g.worldState == worldOffline })
	entries, mark, world := g.shownBoard()
	if world || len(entries) != 1 || mark != 1 || g.worldRank != 0 {
		t.Fatalf("offline: world %v, %d entries, mark %d, rank %d", world, len(entries), mark, g.worldRank)
	}
	g.now += 2
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.render(b)
	if s := screen(b); !strings.Contains(s, "BEST SCORES") || !strings.Contains(s, "Ece") {
		t.Errorf("offline end screen:\n%s", s)
	}
}

func TestNoBoardMeansNoRemote(t *testing.T) {
	if newRemote("  ") != nil {
		t.Error("an empty address made a remote")
	}
	if boardURL("off") != "" {
		t.Error("-board off still names a board")
	}
	t.Setenv("CASTLE_LEMONSTEIN_BOARD", "https://example.test")
	if got := boardURL(""); got != "https://example.test" {
		t.Errorf("CASTLE_LEMONSTEIN_BOARD gave %q", got)
	}
	if got := boardURL("https://flag.test"); got != "https://flag.test" {
		t.Errorf("the flag gave %q", got)
	}
}

func TestFramesWithAWorldBoardAllocateNothing(t *testing.T) {
	srv := httptest.NewServer(&fakeBoard{rank: 1})
	defer srv.Close()
	g := playing()
	g.remote = newRemote(srv.URL)
	g.remote.fetch()
	waitFor(t, g, func() bool { return g.worldState == worldOK })
	g.phase = phTitle
	b := buffer.NewBuffer(cell.Rect{Width: 140, Height: 44})
	g.render(b)
	if avg := testing.AllocsPerRun(50, func() { g.step(tick); g.render(b) }); avg != 0 {
		t.Errorf("%.2f allocations per frame on the title with a world board, want 0", avg)
	}
}
