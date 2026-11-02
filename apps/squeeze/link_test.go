package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkIgnored(t *testing.T) {
	root, base := newRepo(t)
	write := func(rel, s string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "node_modules/\ntarget/\n")
	write("node_modules/x/index.js", "x")
	write("target/out", "x")
	write("vendor/v.go", "x") // exists but not ignored
	write(".squeeze/worktrees/a", "x")
	if err := excludeSqueeze(root); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(worktreesDir(root), "r1", "claude")
	if err := addWorktree(root, dir, "squeeze/r1/claude", base); err != nil {
		t.Fatal(err)
	}
	if err := linkIgnored(root, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "x", "index.js")); err != nil {
		t.Errorf("ignored node_modules not linked: %v", err)
	}
	for _, n := range []string{"vendor", "target", ".squeeze"} {
		if _, err := os.Lstat(filepath.Join(dir, n)); err == nil {
			t.Errorf("%s must not be linked", n)
		}
	}
	// The link must not count as a change.
	ch, err := changes(dir, base)
	if err != nil || len(ch) != 0 {
		t.Errorf("changes = %v, %v; want none", ch, err)
	}
}
