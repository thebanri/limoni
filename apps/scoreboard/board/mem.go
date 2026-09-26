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
	drops   []DropEntry // Lemon Drop's board, in drop.json beside it
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
	if err := readFile(path, &m.entries); err != nil {
		return nil, err
	}
	m.sort()
	if err := readFile(m.dropPath(), &m.drops); err != nil {
		return nil, err
	}
	m.sortDrops()
	return m, nil
}

// readFile reads JSON from path into v; a file not there leaves v as it is.
func readFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (m *Mem) dropPath() string { return filepath.Join(filepath.Dir(m.path), "drop.json") }

func (m *Mem) sortDrops() {
	sort.SliceStable(m.drops, func(i, j int) bool { return dropBefore(m.drops[i], m.drops[j]) })
	if len(m.drops) > Keep {
		m.drops = m.drops[:Keep]
	}
}

// TopDrop is Top for Lemon Drop's board.
func (m *Mem) TopDrop(_ context.Context, n int) ([]DropEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n = min(n, len(m.drops))
	out := make([]DropEntry, n)
	copy(out, m.drops[:n])
	return out, nil
}

// AddDrop is Add for Lemon Drop's board.
func (m *Mem) AddDrop(_ context.Context, e DropEntry) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drops = append(m.drops, e)
	m.sortDrops()
	rank := 0
	for i := range m.drops {
		if m.drops[i] == e {
			rank = i + 1
			break
		}
	}
	if m.path == "" {
		return rank, nil
	}
	return rank, writeFile(m.dropPath(), m.drops)
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
	return writeFile(m.path, m.entries)
}

func writeFile(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
