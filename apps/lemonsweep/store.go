package main

import "encoding/json"

// No leaderboard: the records stay where the game is played, in the
// browser's localStorage or a file in the config directory.

type records struct {
	bests
	endless bests // /dev/zero's, kept apart: its runs are meant to go on
	maps    [nStages]mapBest
	volume  int
	sens    int // 0: not saved
	ret     reticle
	hasRet  bool // a crosshair was saved
}

// mapBest is one of the three maps' records.
type mapBest struct {
	runs    int // times it was played to an end: swept, or died on
	swept   int // times its warden was swept
	fastest int // the fewest seconds its warden stood, if swept
	kills   int // the most swept on it in one run
}

// add counts a run on the map ending: swept after the warden stood for
// secs, or not, with kills swept on it.
func (m *mapBest) add(swept bool, secs, kills int) {
	m.runs++
	m.kills = max(m.kills, kills)
	if swept {
		if m.swept == 0 || secs < m.fastest {
			m.fastest = secs
		}
		m.swept++
	}
}

type bests struct {
	runs   int
	secs   int
	level  int
	kills  int
	peak   int
	bigHit int
}

type runRecord struct {
	secs, level, kills, peak, bigHit int
}

// add counts a run and keeps what it beat; it reports whether it beat the
// longest run.
func (r *bests) add(run runRecord) bool {
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
	Sens   int  `json:"sensitivity,omitempty"`

	Endless *savedBests `json:"endless,omitempty"`
	Maps    []savedMap  `json:"maps,omitempty"`

	Crosshair *savedReticle `json:"crosshair,omitempty"`
}

type savedBests struct {
	Runs   int `json:"runs"`
	Secs   int `json:"best_seconds"`
	Level  int `json:"best_level"`
	Kills  int `json:"best_kills"`
	Peak   int `json:"most_shots"`
	BigHit int `json:"biggest_hit"`
}

type savedMap struct {
	Runs    int `json:"runs"`
	Swept   int `json:"swept"`
	Fastest int `json:"fastest_warden_seconds"`
	Kills   int `json:"best_kills"`
}

type savedReticle struct {
	Style   int  `json:"style"`
	Color   int  `json:"color"`
	Size    int  `json:"size"`
	Outline bool `json:"outline"`
}

func encodeRecords(r records) []byte {
	v := r.volume
	s := saved{Runs: r.runs, Secs: r.secs, Level: r.level, Kills: r.kills, Peak: r.peak, BigHit: r.bigHit, Volume: &v, Sens: r.sens}
	if e := r.endless; e.runs > 0 {
		s.Endless = &savedBests{e.runs, e.secs, e.level, e.kills, e.peak, e.bigHit}
	}
	if r.maps != ([nStages]mapBest{}) {
		for _, m := range r.maps {
			s.Maps = append(s.Maps, savedMap{m.runs, m.swept, m.fastest, m.kills})
		}
	}
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
	r := records{bests: bests{s.Runs, s.Secs, s.Level, s.Kills, s.Peak, s.BigHit}, volume: -1}
	if e := s.Endless; e != nil {
		r.endless = bests{e.Runs, e.Secs, e.Level, e.Kills, e.Peak, e.BigHit}
	}
	for i, m := range s.Maps {
		if i < nStages && m.Swept >= 0 && m.Runs >= m.Swept {
			r.maps[i] = mapBest{m.Runs, m.Swept, m.Fastest, m.Kills}
		}
	}
	if s.Volume != nil && *s.Volume >= 0 && *s.Volume <= 10 {
		r.volume = *s.Volume
	}
	if s.Sens >= 1 && s.Sens <= 10 {
		r.sens = s.Sens
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

// persist saves the records with the current volume, sensitivity and crosshair.
func (g *game) persist() {
	if g.store == nil {
		return
	}
	g.best.volume, g.best.sens = g.volume, g.sens
	g.best.ret, g.best.hasRet = g.ret, true
	g.store.save(g.best)
}
