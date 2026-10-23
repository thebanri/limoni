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
	ret    reticle
	hasRet bool // a crosshair was saved
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

	Crosshair *savedReticle `json:"crosshair,omitempty"`
}

type savedReticle struct {
	Style   int  `json:"style"`
	Color   int  `json:"color"`
	Size    int  `json:"size"`
	Outline bool `json:"outline"`
}

func encodeRecords(r records) []byte {
	v := r.volume
	s := saved{r.runs, r.secs, r.level, r.kills, r.peak, r.bigHit, &v, nil}
	if r.hasRet {
		s.Crosshair = &savedReticle{r.ret.style, r.ret.color, r.ret.size, r.ret.outline}
	}
	b, _ := json.Marshal(s)
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
	if c := s.Crosshair; c != nil && c.Style >= 0 && c.Style < len(reticleStyles) && c.Color >= 0 && c.Color < len(reticleColors) && c.Size >= 0 && c.Size < len(reticleSizes) {
		r.ret, r.hasRet = reticle{c.Style, c.Color, c.Size, c.Outline}, true
	}
	return r
}

type store interface {
	load() records
	save(records)
}

// persist saves the records with the current volume and crosshair.
func (g *game) persist() {
	if g.store == nil {
		return
	}
	g.best.volume = g.volume
	g.best.ret, g.best.hasRet = g.ret, true
	g.store.save(g.best)
}
