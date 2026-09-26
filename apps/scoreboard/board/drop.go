package board

import (
	"errors"
	"time"
)

// Lemon Drop's board (apps/lemondrop) sits beside Castle Lemonstein's, at
// /drop/scores, with its own runs and the same limit on how often one
// address may send one.
//
// The game scores a clear as its base, the grains it held in tens of
// points a block's worth, times the level and the chain; the level is one
// more for every four clears. Dropping pieces scores a little too. The
// client sends what the run did, clear by clear, and the score is worked
// out here from that:
//
//	score  drops + Σ base × level × chain
//	level  1 + (clears so far, this one included) ÷ 4
//
// As with Castle Lemonstein's board, anyone can still send a run they did not
// play; the checks below only make sure it is one the game could have
// produced.
const (
	dropCols      = 10 // the board's width, in blocks
	dropRows      = 18 // and height
	dropMinB      = 2  // grains to a block edge, in the smallest window
	dropMaxB      = 6  // and the largest
	dropMaxSecs   = 24 * 60 * 60
	dropMaxClears = 5000
	dropMaxChain  = 50 // clears in a row from one piece's sand

	// A piece dropped at once from the top scores 2 a row of blocks, so
	// 36 at most; let down with ↓, 1 a row, with a point for each press.
	dropMaxDropPts = 2 * dropRows * 2
	// Nobody places pieces faster than this, a second.
	dropMaxPiecesPerSec = 10

	// DropBody is how large a run may be: a long game sends a clear list.
	DropBody = 64 << 10
)

// DropRun is what the game sends when a run of Lemon Drop ends.
type DropRun struct {
	Name   string `json:"name"`
	Secs   int    `json:"secs"`
	Pieces int    `json:"pieces"`
	Drops  int    `json:"drops"` // points from dropping pieces
	// Clears is each clear in turn, as [base, chain].
	Clears [][2]int `json:"clears"`
}

// DropEntry is a run on Lemon Drop's board.
type DropEntry struct {
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Clears int    `json:"clears"`
	Level  int    `json:"level"`
	Secs   int    `json:"secs"`
	At     int64  `json:"at"` // unix seconds
}

// Check cleans the run's name and makes sure its numbers are ones the game
// could have produced.
func (r *DropRun) Check() error {
	r.Name = CleanName(r.Name)
	switch {
	case r.Name == "":
		return errors.New("a name is needed")
	case r.Secs < 1 || r.Secs > dropMaxSecs:
		return errBadRun
	case r.Pieces < 1 || r.Pieces > r.Secs*dropMaxPiecesPerSec:
		return errBadRun
	case r.Drops < 0 || r.Drops > r.Pieces*dropMaxDropPts:
		return errBadRun
	case len(r.Clears) > dropMaxClears:
		return errBadRun
	}
	// A clear reaches from wall to wall, so it holds at least a row of
	// grains, ten blocks' worth divided by the grains a block has on a
	// side; and at most the whole board. Its base is that in tens of
	// points: between 100/6 and 1800.
	const minBase, maxBase = dropCols * 10 / dropMaxB, dropCols * dropRows * 10
	// Sand is only what pieces brought: four blocks each, 40 in bases, with
	// half a point of rounding a clear.
	sum, chain := 0, 0
	for _, c := range r.Clears {
		base, ch := c[0], c[1]
		if base < minBase || base > maxBase {
			return errBadRun
		}
		// A chain is 1, or one more than the clear before it: it counts
		// clears since the last piece landed.
		if ch != 1 && ch != chain+1 || ch > dropMaxChain {
			return errBadRun
		}
		chain = ch
		sum += base
	}
	if sum > r.Pieces*40+len(r.Clears) {
		return errBadRun
	}
	return nil
}

// Entry scores the run, as made at now.
func (r *DropRun) Entry(now time.Time) DropEntry {
	e := DropEntry{Name: r.Name, Secs: r.Secs, Clears: len(r.Clears), Level: 1 + len(r.Clears)/4, At: now.Unix()}
	e.Score = r.Drops
	for i, c := range r.Clears {
		e.Score += c[0] * (1 + (i+1)/4) * c[1]
	}
	return e
}

// dropBefore reports whether a ranks above b: the higher score, and in a
// tie, the run made first.
func dropBefore(a, b DropEntry) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.At < b.At
}
