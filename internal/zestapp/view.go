package zestapp

import (
	"sync"

	"github.com/thebanri/limoni/widgets"
)

// view is what the log pane shows: every line, or the lines that pass the
// filter. Filtering runs in a goroutine, a batch at a time, and calls wake
// after each batch, so a million lines never stall a keystroke; new lines
// arriving while following are filtered the same way.
type view struct {
	src  *store
	wake func()

	mu       sync.Mutex
	query    string
	minLevel widgets.LogLevel
	idx      []int32 // matching source lines, in order
	scanned  int     // source lines checked against the current filter
	running  bool
	gen      uint64 // bumped when the filter changes; a scan of an old filter stops
}

// scanBatch is how many lines one scanning step checks before letting the UI
// draw what it has so far.
const scanBatch = 200_000

func (v *view) filtering() bool { return v.query != "" || v.minLevel > widgets.LevelUnknown }

// Len is the number of lines shown.
func (v *view) Len() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.filtering() {
		return v.src.Len()
	}
	return len(v.idx)
}

// source maps a shown line to its line in the store.
func (v *view) source(i int) int {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.filtering() {
		return i
	}
	return int(v.idx[i])
}

func (v *view) Line(i int) string            { return v.src.Line(v.source(i)) }
func (v *view) LineNumber(i int) int         { return v.source(i) + 1 }
func (v *view) Level(i int) widgets.LogLevel { return v.src.Level(v.source(i)) }

// setFilter shows only lines containing query (case-insensitive) at
// minLevel or above, and starts scanning for them.
func (v *view) setFilter(query string, minLevel widgets.LogLevel) {
	v.mu.Lock()
	if query == v.query && minLevel == v.minLevel {
		v.mu.Unlock()
		return
	}
	v.query, v.minLevel = query, minLevel
	v.idx = v.idx[:0]
	v.scanned = 0
	v.gen++
	v.mu.Unlock()
	v.kick()
}

// progress reports how far the scan has got, for the status line.
func (v *view) progress() (scanned, total int, done bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	total = v.src.Len()
	if !v.filtering() {
		return total, total, true
	}
	return v.scanned, total, v.scanned >= total
}

// kick scans any lines not yet checked against the filter. It is called when
// the filter changes and whenever new lines arrive.
func (v *view) kick() {
	v.mu.Lock()
	if v.running || !v.filtering() {
		v.mu.Unlock()
		return
	}
	v.running = true
	v.mu.Unlock()
	go v.scan()
}

func (v *view) scan() {
	for {
		v.mu.Lock()
		gen, query, minLevel, from := v.gen, v.query, v.minLevel, v.scanned
		v.mu.Unlock()
		to := min(v.src.Len(), from+scanBatch)
		if from >= to {
			v.mu.Lock()
			// Lines may have arrived, or the filter changed, since the check.
			if v.gen == gen && v.scanned >= v.src.Len() {
				v.running = false
				v.mu.Unlock()
				return
			}
			v.mu.Unlock()
			continue
		}

		var found []int32
		v.src.mu.RLock()
		for i := from; i < to; i++ {
			if v.src.lines[i].level < minLevel {
				continue
			}
			if query != "" && !containsFold(v.src.lineLocked(i), query) {
				continue
			}
			found = append(found, int32(i))
		}
		v.src.mu.RUnlock()

		v.mu.Lock()
		if v.gen == gen { // otherwise the filter changed mid-batch; start over
			v.idx = append(v.idx, found...)
			v.scanned = to
		}
		v.mu.Unlock()
		if v.wake != nil {
			v.wake()
		}
	}
}

// containsFold reports whether s contains sub, folding ASCII case.
func containsFold(s, sub string) bool {
outer:
	for i := 0; i+len(sub) <= len(s); i++ {
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				continue outer
			}
		}
		return true
	}
	return false
}
