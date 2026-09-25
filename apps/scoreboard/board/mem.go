package board

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Mem keeps the board in the process, and, when it has a path, in a JSON
// file there too, rewritten on every run: for a server with a disk that
// outlives it, such as a Railway volume. It suits one long-running process,
// not functions that come and go.
type Mem struct {
	mu      sync.Mutex
	entries []Entry
	path    string
	posts   map[string][]time.Time // recent runs sent, by address
}

// OpenMem reads the board from path, if there is one there. An empty path
// keeps the board in memory only.
func OpenMem(path string) (*Mem, error) {
	m := &Mem{path: path, posts: map[string][]time.Time{}}
	if path == "" {
		return m, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &m.entries); err != nil {
		return nil, err
	}
	m.sort()
	return m, nil
}

func (m *Mem) sort() {
	sort.SliceStable(m.entries, func(i, j int) bool { return before(m.entries[i], m.entries[j]) })
	if len(m.entries) > Keep {
		m.entries = m.entries[:Keep]
	}
}

func (m *Mem) Top(_ context.Context, n int) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n = min(n, len(m.entries))
	out := make([]Entry, n)
	copy(out, m.entries[:n])
	return out, nil
}

func (m *Mem) Add(_ context.Context, e Entry) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, e)
	m.sort()
	rank := 0
	for i := range m.entries {
		if m.entries[i] == e {
			rank = i + 1
			break
		}
	}
	return rank, m.save()
}

func (m *Mem) Allow(_ context.Context, addr string, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	recent := m.posts[addr][:0]
	for _, t := range m.posts[addr] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= PostsPerMinute {
		m.posts[addr] = recent
		return false, nil
	}
	m.posts[addr] = append(recent, now)
	if len(m.posts) > 10000 { // forget everyone rather than grow without end
		m.posts = map[string][]time.Time{addr: m.posts[addr]}
	}
	return true, nil
}

// save writes a temporary file and renames it over the old one, so a crash
// halfway through never leaves the board cut short.
func (m *Mem) save() error {
	if m.path == "" {
		return nil
	}
	data, err := json.Marshal(m.entries)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}
