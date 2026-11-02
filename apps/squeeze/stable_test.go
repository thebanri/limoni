package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStablePathReusedAcrossRaces(t *testing.T) {
	root, base := newRepo(t)
	path := filepath.Join(worktreesDir(root), "claude")
	if err := reuseWorktree(root, path, "squeeze/r1/claude", base); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "left.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := reuseWorktree(root, path, "squeeze/r2/claude", base); err != nil {
		t.Fatal(err)
	}
	if b, _ := git(path, "rev-parse", "--abbrev-ref", "HEAD"); strings.TrimSpace(b) != "squeeze/r2/claude" {
		t.Fatalf("branch = %q", b)
	}
	if _, err := os.Stat(filepath.Join(path, "left.txt")); err == nil {
		t.Fatal("new worktree should start from base")
	}
	// The earlier race's work survives on its own branch.
	if out, err := git(root, "show", "squeeze/r1/claude:left.txt"); err != nil || out != "work\n" {
		t.Fatalf("leftover work lost: %q %v", out, err)
	}
}

func TestCleanRemovesOldAndNewLayouts(t *testing.T) {
	root, base := newRepo(t)
	old := filepath.Join(worktreesDir(root), "r0", "agy")
	if err := addWorktree(root, old, "squeeze/r0/agy", base); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(worktreesDir(root), "claude")
	for _, r := range []string{"r1", "r2"} {
		if err := reuseWorktree(root, path, "squeeze/"+r+"/claude", base); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(path, r+".txt"), []byte(r), 0o644)
	}
	if err := clean(root); err != nil {
		t.Fatal(err)
	}
	if out, _ := git(root, "branch", "--list", "squeeze/*"); strings.TrimSpace(out) != "" {
		t.Fatalf("branches left: %s", out)
	}
	if out, _ := git(root, "worktree", "list", "--porcelain"); strings.Count(out, "worktree ") != 1 {
		t.Fatalf("worktrees left: %s", out)
	}
	if _, err := os.Stat(worktreesDir(root)); err == nil {
		t.Fatal(".squeeze/worktrees left")
	}
}
