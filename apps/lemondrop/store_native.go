//go:build !js

package main

import (
	"os"
	"path/filepath"
)

// store keeps the name and the game's own board in a small file in the
// user's configuration directory: ~/.config/limoni/lemondrop.json on
// Linux. Where there is none, they last the session; so do they for the
// zero store, which the tests use.
type store struct {
	path string
}

func newStore() store {
	dir, err := os.UserConfigDir()
	if err != nil {
		return store{}
	}
	return store{path: filepath.Join(dir, "limoni", "lemondrop.json")}
}

func (s store) load() (string, []scoreEntry) {
	if s.path == "" {
		return "", nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return "", nil
	}
	return decodeSaved(data)
}

// save writes a temporary file and renames it over the old one, so a crash
// halfway through never leaves the board cut short. It runs when a run
// ends or the name changes, never in a frame.
func (s store) save(name string, board []scoreEntry) {
	if s.path == "" {
		return
	}
	data := encodeSaved(name, board)
	if os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) != nil {
		return
	}
	_ = os.Rename(tmp, s.path)
}
