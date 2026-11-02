package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
)

// newRepo makes a repository with one commit holding main.go.
func newRepo(t *testing.T) (root, base string) {
	t.Helper()
	root = t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
	} {
		if _, err := git(root, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc a() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := git(root, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := git(root, "commit", "-q", "-m", "first"); err != nil {
		t.Fatal(err)
	}
	out, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return root, strings.TrimSpace(out)
}

// What a race does to a repository: a worktree per agent, its changes seen
// (an edit and a new file, neither committed), the winner merged into the
// main tree as staged changes, and the main repository's status untouched
// by the .squeeze directory.
func TestARaceWorktreeIsSeenAndMerged(t *testing.T) {
	root, base := newRepo(t)
	if err := excludeSqueeze(root); err != nil {
		t.Fatal(err)
	}
	if err := excludeSqueeze(root); err != nil { // twice adds it once
		t.Fatal(err)
	}
	dir := filepath.Join(worktreesDir(root), "r1", "claude")
	if err := addWorktree(root, dir, "squeeze/r1/claude", base); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc b() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := changes(dir, base)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]fileChange{}
	for _, f := range files {
		got[f.Path] = f
	}
	if f := got["main.go"]; f.Added != 1 || f.Removed != 1 {
		t.Errorf("main.go = %+v, want +1 -1", f)
	}
	if f := got["new.go"]; f.Added != 1 || f.Removed != 0 {
		t.Errorf("new.go = %+v, want +1 -0 (an untracked file counts)", f)
	}
	if before, after := versions(dir, base, "new.go"); before != "" || after != "package main\n" {
		t.Errorf("versions(new.go) = %q, %q", before, after)
	}
	if status, _ := git(root, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("the main tree shows squeeze's files: %q", status)
	}

	if err := merge(root, dir, "squeeze/r1/claude", "claude"); err != nil {
		t.Fatal(err)
	}
	staged, _ := git(root, "diff", "--staged", "--name-only")
	if strings.Fields(staged)[0] != "main.go" || !strings.Contains(staged, "new.go") {
		t.Errorf("staged after merge: %q", staged)
	}
	if data, _ := os.ReadFile(filepath.Join(root, "main.go")); !strings.Contains(string(data), "func b()") {
		t.Errorf("main.go after merge: %q", data)
	}

	if err := removeWorktree(root, dir, "squeeze/r1/claude"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("worktree still there: %v", err)
	}
}

// Ctrl+] as the terminal sends it — byte 0x1d — must detach.
func TestCtrlBracketDetaches(t *testing.T) {
	ev, _ := driver.ParseEvent([]byte{0x1d})
	if !isDetach(ev.Key) {
		t.Errorf("Ctrl+] parsed as %+v, not taken as detach", ev.Key)
	}
	if ev, _ := driver.ParseEvent([]byte("]")); isDetach(ev.Key) {
		t.Error("a plain ] detaches")
	}
}

// A racer runs its agent on a terminal: the screen shows what it printed,
// it is waiting once it falls quiet, keys reach it, and it is done when it
// exits.
func TestARacerRunsItsAgentOnATerminal(t *testing.T) {
	defer func(q time.Duration) { quiet = q }(quiet)
	quiet = 500 * time.Millisecond
	agent := agentDef{Name: "fake", Bin: "sh", Args: func(p, _ string) []string {
		return []string{"-c", `printf 'task: %s\n' "$1"; read line; printf 'got %s\n' "$line"`, "sh", p}
	}}
	r, err := startRacer(agent, "fake", "make it faster", t.TempDir(), "b", 40, 6, func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer r.stop()
	screen := func() string {
		var b strings.Builder
		for y := range 6 {
			for x := range 40 {
				if c := r.emu.CellAt(x, y); c != nil && c.Content != "" {
					b.WriteString(c.Content)
				} else {
					b.WriteByte(' ')
				}
			}
			b.WriteByte('\n')
		}
		return b.String()
	}
	waitFor := func(what string, ok func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for !ok() {
			if time.Now().After(deadline) {
				t.Fatalf("%s never happened; screen:\n%s", what, screen())
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	waitFor("the task on screen", func() bool { return strings.Contains(screen(), "task: make it faster") })
	waitFor("waiting", func() bool { return r.state(time.Now()) == waiting })

	for _, ch := range "yes" {
		k, _ := toEmulator(driver.KeyForRune(ch))
		r.emu.SendKey(k)
	}
	k, _ := toEmulator(driver.KeyEvent{Type: driver.KeyEnter})
	r.emu.SendKey(k)
	waitFor("the typed line echoed back", func() bool { return strings.Contains(screen(), "got yes") })
	waitFor("done", func() bool { return r.state(time.Now()) == done })
}

func TestRacerNamesNumberRepeats(t *testing.T) {
	got := racerNames([]agentDef{{Name: "claude"}, {Name: "agy"}, {Name: "claude"}})
	if strings.Join(got, ",") != "claude,agy,claude-2" {
		t.Errorf("got %v", got)
	}
}
