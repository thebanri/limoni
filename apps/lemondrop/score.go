package main

import (
	"encoding/json"
	"strings"
)

// A run is scored as it goes (game.go), and also written down clear by
// clear, so that the shared leaderboard (apps/scoreboard, /drop/scores)
// can work the score out again from what the run did rather than take it
// on trust:
//
//	score  drops + Σ base × level × chain
//
// where base is a clear's grains in tens of points a block's worth, level
// is one more for every four clears, and drops are the points for dropping
// pieces. The leaderboard's rules are in apps/scoreboard/board/drop.go.

const (
	nameMax   = 12 // characters in a player's name
	boardSize = 10 // runs on a board
	maxLog    = 5000
)

// scoreEntry is a run on a board, the game's own or the world's.
type scoreEntry struct {
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Clears int    `json:"clears"`
	Level  int    `json:"level"`
	Secs   int    `json:"secs"`
	At     int64  `json:"at,omitempty"`
}

// saved is what the game keeps between runs: the name, and its own board.
type saved struct {
	Name  string       `json:"name"`
	Board []scoreEntry `json:"board"`
}

func decodeSaved(data []byte) (string, []scoreEntry) {
	var s saved
	if json.Unmarshal(data, &s) != nil {
		return "", nil
	}
	return cleanName(s.Name), s.Board
}

func encodeSaved(name string, board []scoreEntry) []byte {
	data, _ := json.Marshal(saved{Name: name, Board: board})
	return data
}

// nameRune is what a name may hold: what the panel's font shows, and what
// the leaderboard keeps.
func nameRune(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' ||
		r == ' ' || r == '-' || r == '_' || r == '.' ||
		strings.ContainsRune("ÇĞİÖŞÜçğıöşü", r)
}

// cleanName keeps what a name may hold, trimmed and at most nameMax long,
// as the leaderboard does.
func cleanName(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if n == nameMax {
			break
		}
		if nameRune(r) && (r != ' ' || n > 0) {
			b.WriteRune(r)
			n++
		}
	}
	return strings.TrimSpace(b.String())
}

// logClear writes a clear down for the leaderboard.
func (g *game) logClear(base, chain int) {
	if g.nlog < maxLog {
		g.clearLog[g.nlog] = [2]int32{int32(base), int32(chain)}
	}
	g.nlog++
}

// addLocal puts e on the game's own board and returns its place, 1-based,
// or 0 when it did not make it.
func (g *game) addLocal(e scoreEntry) int {
	i := 0
	for i < g.nLocal && g.local[i].Score >= e.Score {
		i++
	}
	if i >= boardSize {
		return 0
	}
	if g.nLocal < boardSize {
		g.nLocal++
	}
	copy(g.local[i+1:g.nLocal], g.local[i:g.nLocal-1])
	g.local[i] = e
	return i + 1
}

// restore reads the name and the game's own board from the store.
func (g *game) restore() {
	name, board := g.store.load()
	g.name = name
	g.nLocal = 0
	for _, e := range board {
		if g.nLocal == boardSize {
			break
		}
		e.Name = cleanName(e.Name)
		g.local[g.nLocal] = e
		g.nLocal++
	}
	if g.nLocal > 0 {
		g.best = g.local[0].Score
	}
}

func (g *game) persist() {
	g.store.save(g.name, g.local[:g.nLocal])
}
