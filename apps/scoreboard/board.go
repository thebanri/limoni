package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// The score is worked out here from what the run did, with the same rules
// as the game (apps/lemonhunt/score.go), rather than taken from the client:
//
//	aim     5000 × hits ÷ squirts
//	time    10 for every second under ten minutes, for a win only
//	rats    50 each
//	lemons  100 each
//
// Anyone can still send a run they did not play; the server can only make
// sure it is a run the game could have produced.
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

	keep = 100 // runs kept; the board shows the top of them
)

// run is what a client sends when a game ends.
type run struct {
	Name   string `json:"name"`
	Won    bool   `json:"won"`
	Secs   int    `json:"secs"`
	Shots  int    `json:"shots"`
	Hits   int    `json:"hits"`
	Kills  int    `json:"kills"`
	Lemons int    `json:"lemons"`
}

// entry is a run on the board, in the shape the game stores its own.
type entry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Secs  int    `json:"secs"`
	Aim   int    `json:"aim"` // percent
	Won   bool   `json:"won"`
	At    int64  `json:"at"` // unix seconds
}

var errBadRun = errors.New("not a run the game could have produced")

// check cleans a run's name and makes sure its numbers are ones the game
// could have produced.
func (r *run) check() error {
	r.Name = cleanName(r.Name)
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

func (r *run) entry(now time.Time) entry {
	e := entry{Name: r.Name, Secs: r.Secs, Won: r.Won, At: now.Unix()}
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

// board holds the best runs, best first, in memory, and has a keeper write
// each one down so that they survive a restart.
type board struct {
	mu      sync.Mutex
	entries []entry
	keeper  keeper
}

// keeper is where the board is kept: a file on a disk, or a Postgres
// database (keeper_pg.go); nil keeps it in memory only.
type keeper interface {
	load() ([]entry, error)
	// added writes down e, which has just been put on the board; all is the
	// whole board now, best first.
	added(e entry, all []entry) error
}

func openBoard(k keeper) (*board, error) {
	b := &board{keeper: k}
	if k == nil {
		return b, nil
	}
	entries, err := k.load()
	if err != nil {
		return nil, err
	}
	b.entries = entries
	b.sort()
	return b, nil
}

// sort orders the board best first; a tie goes to the run made earlier.
func (b *board) sort() {
	sort.SliceStable(b.entries, func(i, j int) bool {
		if b.entries[i].Score != b.entries[j].Score {
			return b.entries[i].Score > b.entries[j].Score
		}
		return b.entries[i].At < b.entries[j].At
	})
	if len(b.entries) > keep {
		b.entries = b.entries[:keep]
	}
}

// add puts e on the board and returns its place, 1-based, or 0 when it did
// not make the runs kept.
func (b *board) add(e entry) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append(b.entries, e)
	b.sort()
	rank := 0
	for i := range b.entries {
		if b.entries[i] == e {
			rank = i + 1
			break
		}
	}
	if b.keeper == nil {
		return rank, nil
	}
	return rank, b.keeper.added(e, b.entries)
}

// top returns the best n runs.
func (b *board) top(n int) []entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	n = min(n, len(b.entries))
	out := make([]entry, n)
	copy(out, b.entries[:n])
	return out
}

// fileKeeper keeps the board in a JSON file: on a Railway volume, or
// anywhere with a disk that outlives the process.
type fileKeeper struct{ path string }

func (f fileKeeper) load() ([]entry, error) {
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []entry
	return entries, json.Unmarshal(data, &entries)
}

// added writes the whole board to a temporary file and renames it over the
// old one, so a crash halfway through never leaves the board cut short.
func (f fileKeeper) added(_ entry, all []entry) error {
	data, err := json.Marshal(all)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}
