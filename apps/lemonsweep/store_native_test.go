//go:build !js

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Records kept under the old name are read until the new file exists, and
// saving writes the new one.
func TestRecordsFromBeforeTheRename(t *testing.T) {
	dir := t.TempDir()
	s := fileStore{filepath.Join(dir, "lemonsweep", "records.json"), filepath.Join(dir, "marksweep", "records.json")}
	old := records{bests: bests{runs: 7, secs: 900}, volume: 3}
	if err := os.MkdirAll(filepath.Dir(s.old), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.old, encodeRecords(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.load(); got.runs != 7 || got.secs != 900 {
		t.Fatalf("the old records were not read: %+v", got)
	}
	s.save(records{bests: bests{runs: 8, secs: 950}, volume: 3})
	if _, err := os.Stat(s.path); err != nil {
		t.Fatal("saving did not write the new file")
	}
	if got := s.load(); got.runs != 8 {
		t.Fatalf("the new file did not win: %+v", got)
	}
}
