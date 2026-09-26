//go:build !js

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// store keeps the best score in a small file in the user's configuration
// directory. Where there is none, the best lasts the session; so does it
// for the zero store, which the tests use.
type store struct {
	path string
}

func newStore() store {
	dir, err := os.UserConfigDir()
	if err != nil {
		return store{}
	}
	return store{path: filepath.Join(dir, "limoni", "lemondrop-best")}
}

func (s store) load() int {
	if s.path == "" {
		return 0
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// save writes the best score. It runs once a run, at game over, so the
// allocation does not touch a frame.
func (s store) save(best int) {
	if s.path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	_ = os.WriteFile(s.path, []byte(strconv.Itoa(best)+"\n"), 0o644)
}
