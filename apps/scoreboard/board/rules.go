// Package board is Castle Lemonstein's shared leaderboard: the rules a run is
// scored and checked by, the HTTP handler, and the stores that keep the
// runs. The long-running server (the module's main) and the Vercel
// functions (api/) both serve it.
package board

import (
	"errors"
	"strings"
	"time"
)

// The score is worked out here from what the run did, with the same rules
// as the game (apps/castle-lemonstein/score.go), rather than taken from the client:
//
//	aim     5000 × hits ÷ squirts
//	time    10 for every second under ten minutes, for a win only
//	rats    50 each
//	lemons  100 each
//
// Anyone can still send a run they did not play; this can only make sure
// it is a run the game could have produced.
const (
	aimPoints   = 5000
	timeLimit   = 600
	timePoints  = 10
	ratPoints   = 50
	lemonPoints = 100

	lemonsNeeded = 10
	maxKills     = 30 // the level holds 8 rats, and Ratatui calls up to 12 more
	maxShots     = 20000
	minWinSecs   = 20 // nobody finds ten lemons and kills a king faster
	maxSecs      = 24 * 60 * 60
	nameMax      = 12

	// Keep is how many runs are kept; the board shows the top Shown.
	Keep  = 100
	Shown = 10
	// PostsPerMinute is how many runs one address may send in a minute: a
	// run takes longer than ten seconds.
	PostsPerMinute = 6
)

// Run is what a client sends when a game ends.
type Run struct {
	Name   string `json:"name"`
	Won    bool   `json:"won"`
	Secs   int    `json:"secs"`
	Shots  int    `json:"shots"`
	Hits   int    `json:"hits"`
	Kills  int    `json:"kills"`
	Lemons int    `json:"lemons"`
}

// Entry is a run on the board, in the shape the game stores its own.
type Entry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Secs  int    `json:"secs"`
	Aim   int    `json:"aim"` // percent
	Won   bool   `json:"won"`
	At    int64  `json:"at"` // unix seconds
}

var errBadRun = errors.New("not a run the game could have produced")

// Check cleans a run's name and makes sure its numbers are ones the game
// could have produced.
func (r *Run) Check() error {
	r.Name = CleanName(r.Name)
	switch {
	case r.Name == "":
		return errors.New("a name is needed")
	case r.Shots < 0 || r.Shots > maxShots || r.Hits < 0 || r.Hits > r.Shots:
		return errBadRun
	case r.Kills < 0 || r.Kills > maxKills || r.Lemons < 0 || r.Lemons > lemonsNeeded:
		return errBadRun
	case r.Secs < 1 || r.Secs > maxSecs:
		return errBadRun
	case r.Won && (r.Lemons != lemonsNeeded || r.Secs < minWinSecs):
		return errBadRun
	}
	return nil
}

// Entry scores the run, as made at now.
func (r *Run) Entry(now time.Time) Entry {
	e := Entry{Name: r.Name, Secs: r.Secs, Won: r.Won, At: now.Unix()}
	if r.Shots > 0 {
		e.Aim = r.Hits * 100 / r.Shots
		e.Score = r.Hits * aimPoints / r.Shots
	}
	if r.Won {
		e.Score += max(0, timeLimit-r.Secs) * timePoints
	}
	e.Score += r.Kills*ratPoints + r.Lemons*lemonPoints
	return e
}

// nameRune is what a name may hold: what the game's font shows in a column.
func nameRune(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' ||
		r == ' ' || r == '-' || r == '_' || r == '.' ||
		strings.ContainsRune("ÇĞİÖŞÜçğıöşü", r)
}

// CleanName keeps what a name may hold, trimmed and at most 12 long; the
// game cleans names the same way.
func CleanName(s string) string {
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

// before reports whether a ranks above b: the higher score, and in a tie,
// the run made first.
func before(a, b Entry) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.At < b.At
}
