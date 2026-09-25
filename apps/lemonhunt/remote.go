package main

import (
	"encoding/json"
	"strings"
)

// The shared leaderboard is a small server (apps/scoreboard) that takes each
// finished run and hands back the best ten. The game talks to it off the
// frame, on a goroutine of its own, and step picks up the answers from a
// channel without waiting, so a slow or missing server never stalls a
// frame. Without one the game keeps its own board, as before.

// remoteRun is what the server is sent: what the run did. The server works
// the score out itself.
type remoteRun struct {
	Name   string `json:"name"`
	Won    bool   `json:"won"`
	Secs   int    `json:"secs"`
	Shots  int    `json:"shots"`
	Hits   int    `json:"hits"`
	Kills  int    `json:"kills"`
	Lemons int    `json:"lemons"`
}

// remoteAnswer is what comes back: the board, and for a run sent, its place.
type remoteAnswer struct {
	sent  bool // an answer to a run sent, not to a fetch
	board []scoreEntry
	rank  int
	err   error
}

type remote struct {
	url     string // the server's base, without /scores
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
		body, err := httpDo("GET", r.url+"/scores", nil)
		r.answer(remoteAnswer{}, body, err)
	}()
}

// submit sends a finished run.
func (r *remote) submit(run remoteRun) {
	go func() {
		data, _ := json.Marshal(run)
		body, err := httpDo("POST", r.url+"/scores", data)
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

// sendRun sends the run that just ended to the shared board.
func (g *game) sendRun(won bool) {
	if g.remote == nil {
		return
	}
	g.worldRank = -1 // sending
	g.remote.submit(remoteRun{
		Name:   g.playerName(),
		Won:    won,
		Secs:   max(1, int(g.finished)),
		Shots:  g.shots,
		Hits:   g.hits,
		Kills:  g.kills,
		Lemons: min(g.lemons, lemonsNeeded),
	})
}

// shownBoard is the board the screens show: the world's when the server has
// answered, the game's own otherwise; and the place to pick out on it.
func (g *game) shownBoard() (entries []scoreEntry, mark int, world bool) {
	if g.worldState == worldOK {
		return g.world[:g.nWorld], max(0, g.worldRank), true
	}
	return g.board[:g.nBoard], g.last.rank, false
}
