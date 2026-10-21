package main

import "encoding/json"

// No leaderboard: the records stay where the game is played, in the
// browser's localStorage or a file in the config directory.

type records struct {
	runs   int
	secs   int
	level  int
	kills  int
	peak   int
	bigHit int
	volume int
}

type runRecord struct {
	secs, level, kills, peak, bigHit int
}

// add counts a run and keeps what it beat; it reports whether it beat the
// longest run.
func (r *records) add(run runRecord) bool {
	r.runs++
	longest := run.secs > r.secs
	r.secs = max(r.secs, run.secs)
	r.level = max(r.level, run.level)
	r.kills = max(r.kills, run.kills)
	r.peak = max(r.peak, run.peak)
	r.bigHit = max(r.bigHit, run.bigHit)
	return longest
}

type saved struct {
	Runs   int  `json:"runs"`
	Secs   int  `json:"best_seconds"`
	Level  int  `json:"best_level"`
	Kills  int  `json:"best_kills"`
	Peak   int  `json:"most_shots"`
	BigHit int  `json:"biggest_hit"`
	Volume *int `json:"volume,omitempty"`
}

func encodeRecords(r records) []byte {
	v := r.volume
	b, _ := json.Marshal(saved{r.runs, r.secs, r.level, r.kills, r.peak, r.bigHit, &v})
	return b
}

// decodeRecords reads what encodeRecords wrote; anything else is no
// records, with the volume left at -1 for "not saved".
func decodeRecords(b []byte) records {
	var s saved
	if json.Unmarshal(b, &s) != nil {
		return records{volume: -1}
	}
	r := records{runs: s.Runs, secs: s.Secs, level: s.Level, kills: s.Kills, peak: s.Peak, bigHit: s.BigHit, volume: -1}
	if s.Volume != nil && *s.Volume >= 0 && *s.Volume <= 10 {
		r.volume = *s.Volume
	}
	return r
}

type store interface {
	load() records
	save(records)
}

// persist saves the records with the current volume.
func (g *game) persist() {
	if g.store == nil {
		return
	}
	g.best.volume = g.volume
	g.store.save(g.best)
}
