package main

import (
	"encoding/json"
	"strings"
)

// The shared leaderboard is apps/scoreboard's /drop/scores, beside Lemon
// Hunt's board on the same server. The game talks to it off the frame, on
// a goroutine of its own, and each step picks up the answers from a
// channel without waiting, so a slow or missing server never stalls a
// frame. Without one the game shows its own board. It is the same scheme
// as Lemon Hunt's (apps/lemonhunt/remote.go).

// defaultBoard is the shared leaderboard the game ships with: the
// scoreboard on Vercel, over a Neon database. The browser's is set in
// examples/wasm/index.html; -board off keeps the scores on this machine.
const defaultBoard = "https://limoni-drab.vercel.app"

// remoteRun is what the server is sent: what the run did. The server works
// the score out itself.
type remoteRun struct {
	Name   string     `json:"name"`
	Secs   int        `json:"secs"`
	Pieces int        `json:"pieces"`
	Drops  int        `json:"drops"`
	Clears [][2]int32 `json:"clears"`
}

// remoteAnswer is what comes back: the board, and for a run sent, its place.
type remoteAnswer struct {
	sent  bool // an answer to a run sent, not to a fetch
	board []scoreEntry
	rank  int
	err   error
}

type remote struct {
	url     string // the server's base, without /drop/scores
	answers chan remoteAnswer
}

func newRemote(url string) *remote {
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if url == "" {
		return nil
	}
	return &remote{url: url, answers: make(chan remoteAnswer, 4)}
}

// fetch asks for the best ten.
func (r *remote) fetch() {
	go func() {
		body, err := httpDo("GET", r.url+"/drop/scores", nil)
		r.answer(remoteAnswer{}, body, err)
	}()
}

// submit sends a finished run.
func (r *remote) submit(run remoteRun) {
	go func() {
		data, _ := json.Marshal(run)
		body, err := httpDo("POST", r.url+"/drop/scores", data)
		r.answer(remoteAnswer{sent: true}, body, err)
	}()
}

func (r *remote) answer(a remoteAnswer, body []byte, err error) {
	if err == nil {
		var got struct {
			Rank  int          `json:"rank"`
			Board []scoreEntry `json:"board"`
		}
		if err = json.Unmarshal(body, &got); err == nil {
			a.rank, a.board = got.Rank, got.Board
		}
	}
	a.err = err
	if a.sent && err == nil {
		boardChanged()
	}
	select {
	case r.answers <- a:
	default: // nobody is reading: drop it rather than leak the goroutine
	}
}

// The world board's state, as the screens tell it.
const (
	worldNone    = iota // no server set
	worldLoading        // asked, no answer yet
	worldOK             // g.world holds the server's board
	worldOffline        // the server did not answer; the game's own board is shown
)

// pollRemote takes any answer that has arrived. It runs every step and
// allocates nothing when there is none.
func (g *game) pollRemote() {
	if g.remote == nil {
		return
	}
	select {
	case a := <-g.remote.answers:
		g.takeAnswer(a)
	default:
	}
}

func (g *game) takeAnswer(a remoteAnswer) {
	if a.err != nil {
		g.worldState = worldOffline
		if a.sent {
			g.worldRank = 0
		}
		return
	}
	g.worldState = worldOK
	g.nWorld = 0
	for _, e := range a.board {
		if g.nWorld == boardSize {
			break
		}
		e.Name = cleanName(e.Name)
		g.world[g.nWorld] = e
		g.nWorld++
	}
	if a.sent {
		g.worldRank = a.rank
	}
}

// sendRun sends the run that just ended to the shared board. A run with no
// name, or longer than the log holds, stays on this machine.
func (g *game) sendRun() {
	if g.remote == nil || g.name == "" || g.nlog > maxLog {
		return
	}
	g.worldRank = -1 // sending
	g.remote.submit(remoteRun{
		Name:   g.name,
		Secs:   g.runSecs(),
		Pieces: g.pieces,
		Drops:  g.dropPts,
		Clears: append([][2]int32{}, g.clearLog[:g.nlog]...),
	})
}

// shownBoard is the board the screens show: the world's when the server has
// answered, the game's own otherwise; and the place to pick out on it.
func (g *game) shownBoard() (entries []scoreEntry, mark int, world bool) {
	if g.worldState == worldOK {
		return g.world[:g.nWorld], max(0, g.worldRank), true
	}
	return g.local[:g.nLocal], g.localRank, false
}
