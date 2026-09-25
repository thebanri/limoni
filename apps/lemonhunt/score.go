package main

import (
	"encoding/json"
	"strings"
)

// A run is scored on how well the player aimed and how fast they were:
//
//	aim     5000 × hits ÷ squirts
//	time    10 for every second under ten minutes, for a win only
//	rats    50 each
//	lemons  100 each
//
// so a quick, careful win beats a slow one that sprayed the walls, and a
// loss still scores what it got done.
const (
	aimPoints   = 5000
	timeLimit   = 600 // seconds
	timePoints  = 10  // a second saved
	ratPoints   = 50
	lemonPoints = 100
)

// scoreEntry is one run on the leaderboard.
type scoreEntry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Secs  int    `json:"secs"`
	Aim   int    `json:"aim"` // percent
	Won   bool   `json:"won"`
}

// result is the run that just ended, taken apart for the end screen.
type result struct {
	aim, time, rats, lemons int // points
	entry                   scoreEntry
	rank                    int // 1-based place on the board, 0 if off it
}

// store keeps the player's name and the leaderboard between runs: a file
// in the user's config directory, or the browser's localStorage.
type store interface {
	load() (name string, board []scoreEntry)
	save(name string, board []scoreEntry)
}

// score works out the points for the run so far.
func (g *game) score(won bool) result {
	var r result
	aim := 0
	if g.shots > 0 {
		aim = g.hits * 100 / g.shots
		r.aim = g.hits * aimPoints / g.shots
	}
	secs := int(g.finished)
	if won {
		r.time = max(0, timeLimit-secs) * timePoints
	}
	r.rats = g.kills * ratPoints
	r.lemons = min(g.lemons, lemonsNeeded) * lemonPoints
	r.entry = scoreEntry{
		Name:  g.playerName(),
		Score: r.aim + r.time + r.rats + r.lemons,
		Secs:  secs,
		Aim:   aim,
		Won:   won,
	}
	return r
}

func (g *game) playerName() string {
	if g.name == "" {
		return "PLAYER"
	}
	return g.name
}

// record scores the run that just ended, puts it on the board if it earns a
// place, and saves the board. It runs once a run, not once a frame.
func (g *game) record(won bool) {
	g.last = g.score(won)
	g.last.rank = g.insert(g.last.entry)
	g.persist()
}

// insert places e on the board, best first, and returns its place (1-based),
// or 0 when it did not make the board. A tie goes below the scores already
// there.
func (g *game) insert(e scoreEntry) int {
	i := 0
	for i < g.nBoard && g.board[i].Score >= e.Score {
		i++
	}
	if i >= boardSize {
		return 0
	}
	if g.nBoard < boardSize {
		g.nBoard++
	}
	copy(g.board[i+1:g.nBoard], g.board[i:g.nBoard-1])
	g.board[i] = e
	return i + 1
}

func (g *game) persist() {
	if g.store != nil {
		g.store.save(g.name, g.board[:g.nBoard])
	}
}

// restore loads the name and the board from the store.
func (g *game) restore() {
	if g.store == nil {
		return
	}
	name, board := g.store.load()
	g.name = cleanName(name)
	g.nBoard = 0
	for _, e := range board {
		e.Name = cleanName(e.Name)
		if e.Name == "" {
			continue
		}
		g.insert(e)
	}
}

// nameRune reports whether r may be part of a name: what the HUD font shows
// in one column.
func nameRune(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' ||
		r == ' ' || r == '-' || r == '_' || r == '.' ||
		strings.ContainsRune("ÇĞİÖŞÜçğıöşü", r)
}

// cleanName keeps what a name may hold, trimmed and at most nameMax long,
// whatever a stored file or a browser's storage handed back.
func cleanName(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n == nameMax {
			break
		}
		if nameRune(r) {
			b.WriteRune(r)
			n++
		}
	}
	return strings.TrimSpace(b.String())
}

// saved is what the store writes: one JSON object.
type saved struct {
	Name  string       `json:"name"`
	Board []scoreEntry `json:"board"`
}

func encodeSaved(name string, board []scoreEntry) []byte {
	b, _ := json.Marshal(saved{Name: name, Board: board})
	return b
}

func decodeSaved(b []byte) (string, []scoreEntry) {
	var s saved
	if json.Unmarshal(b, &s) != nil {
		return "", nil
	}
	return s.Name, s.Board
}
