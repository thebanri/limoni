//go:build !js

package main

import (
	"os"
	"path/filepath"
)

// fileStore keeps the name and the leaderboard in the user's config
// directory: ~/.config/lemonhunt/scores.json on Linux.
type fileStore struct{ path string }

// newStore returns the file store, or nil where there is no config
// directory to put it in; the game then keeps its scores for the session.
func newStore() store {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	return fileStore{filepath.Join(dir, "lemonhunt", "scores.json")}
}

func (s fileStore) load() (string, []scoreEntry) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return "", nil
	}
	return decodeSaved(b)
}

// save writes a temporary file and renames it over the old one, so a crash
// halfway through never leaves the board truncated.
func (s fileStore) save(name string, board []scoreEntry) {
	if os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, encodeSaved(name, board), 0o644) != nil {
		return
	}
	_ = os.Rename(tmp, s.path)
}
