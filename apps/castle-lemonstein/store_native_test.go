//go:build !js

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenamedGameKeepsExistingScores(t *testing.T) {
	dir := t.TempDir()
	old := fileStore{filepath.Join(dir, "lemonhunt", "scores.json")}
	old.save("Deniz", []scoreEntry{{Name: "Deniz", Score: 4200}})
	next := fileStore{filepath.Join(dir, "castle-lemonstein", "scores.json")}
	name, board := next.load()
	if name != "Deniz" || len(board) != 1 || board[0].Score != 4200 {
		t.Fatalf("existing save lost: %q, %+v", name, board)
	}
	next.save("Ece", board)
	name, _ = next.load()
	if name != "Ece" {
		t.Fatalf("new save did not take precedence: %q", name)
	}
	name, _ = old.load()
	if name != "Deniz" {
		t.Fatal("legacy save was overwritten")
	}
	// An explicitly empty new save must not resurrect an older profile.
	if err := os.WriteFile(next.path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	name, board = next.load()
	if name != "" || len(board) != 0 {
		t.Fatal("empty new save fell back to the legacy profile")
	}
}
