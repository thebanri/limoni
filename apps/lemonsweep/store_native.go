//go:build !js

package main

import (
	"os"
	"path/filepath"
)

// fileStore keeps the records in the user's config directory:
// ~/.config/lemonsweep/records.json on Linux. The game was Mark & Sweep
// first; old is where it kept them then, read until there is a new file.
type fileStore struct{ path, old string }

func newStore() store {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	return fileStore{filepath.Join(dir, "lemonsweep", "records.json"), filepath.Join(dir, "marksweep", "records.json")}
}

func (s fileStore) load() records {
	b, err := os.ReadFile(s.path)
	if err != nil && s.old != "" {
		b, err = os.ReadFile(s.old)
	}
	if err != nil {
		return records{volume: -1}
	}
	return decodeRecords(b)
}

// save writes a temporary file and renames it over the old one, so a crash
// halfway through never leaves the records truncated.
func (s fileStore) save(r records) {
	if os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, encodeRecords(r), 0o644) != nil {
		return
	}
	_ = os.Rename(tmp, s.path)
}
