package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	limoni "github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/testkit"
)

// The compare screen from the keyboard: c opens it on the selected racer's
// files and diff, ←→ move between racers, o shows the check's output, and m
// then y squash-merges the one shown into the main tree as staged changes.
// A racer that changed nothing says so, and n to the question merges nothing.
func TestCompareAndMergeFromTheKeyboard(t *testing.T) {
	root, base := newRepo(t)
	u := newUI(root, base, "task", nil)
	u.notify, u.wake = func(string, string) {}, func() {}
	u.check = "echo all good"
	u.mode = raceMode
	for _, name := range []string{"claude", "agy"} {
		r := fake(t, `sleep 30`)
		r.name, r.branch = name, "squeeze/t/"+name
		r.dir = filepath.Join(worktreesDir(root), "t", name)
		if err := addWorktree(root, r.dir, r.branch, base); err != nil {
			t.Fatal(err)
		}
		u.racers = append(u.racers, r)
	}
	agy := u.racers[1]
	if err := os.WriteFile(filepath.Join(agy.dir, "main.go"), []byte("package main\n\nfunc fromAgy() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	agy.refresh(base) // watch does this every two seconds

	term := testkit.NewTerminal(100, 30)
	draw := func(ev *limoni.Event) string {
		term.Draw(func(f *limoni.Frame) { u.frame(f, ev) })
		return term.Snapshot()
	}
	press := func(k driver.KeyEvent) string { return draw(&limoni.Event{Type: limoni.EventKey, Key: k}) }
	letter := func(c rune) string { return press(driver.KeyEvent{Type: driver.KeyRune, Ch: c}) }

	draw(nil)
	if s := letter('c'); u.mode != compareMode || !strings.Contains(s, "claude has not changed anything yet") {
		t.Fatalf("c: mode %v\n%s", u.mode, s)
	}
	letter('m')
	if letter('n'); u.confirm != "" {
		t.Fatalf("n left the question open: %q", u.confirm)
	}
	if staged, _ := git(root, "diff", "--staged", "--name-only"); strings.TrimSpace(staged) != "" {
		t.Fatalf("n merged anyway: %q", staged)
	}

	s := press(driver.KeyEvent{Type: driver.KeyArrowRight})
	if u.sel != 1 || !strings.Contains(s, "main.go") || !strings.Contains(s, "fromAgy") {
		t.Fatalf("→: sel %d, the screen does not show agy's change\n%s", u.sel, s)
	}

	letter('t') // the check, run in agy's worktree
	if st, out := waitCheck(t, &agy.chk); st != passed || strings.TrimSpace(out) != "all good" {
		t.Fatalf("check = %v %q", st, out)
	}
	if s := letter('o'); !strings.Contains(s, "all good") || !strings.Contains(s, "passes") {
		t.Fatalf("o does not show the check's output\n%s", s)
	}
	letter('o')

	letter('m')
	if u.confirm != "merge" {
		t.Fatalf("m asked %q", u.confirm)
	}
	letter('y')
	staged, _ := git(root, "diff", "--staged", "--name-only")
	if strings.TrimSpace(staged) != "main.go" {
		t.Errorf("staged after merging agy: %q (status: %s)", staged, u.status)
	}
	if data, _ := os.ReadFile(filepath.Join(root, "main.go")); !strings.Contains(string(data), "fromAgy") {
		t.Errorf("main.go after merge: %q", data)
	}
	if !strings.Contains(u.status, "agy's changes are staged") {
		t.Errorf("status = %q", u.status)
	}

	press(driver.KeyEvent{Type: driver.KeyEsc})
	if u.mode != raceMode {
		t.Errorf("Esc left mode %v", u.mode)
	}
}

// -agents: a name twice races two, spaces and empty names are skipped, and
// an unknown or missing agent is refused with its reason.
func TestParseAgents(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	defs, err := parseAgents(" claude, ,claude")
	if err != nil || len(defs) != 2 || defs[0].Name != "claude" || defs[1].Name != "claude" {
		t.Errorf("claude twice = %v, %v", defs, err)
	}
	if _, err := parseAgents("codex"); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Errorf("codex missing: %v", err)
	}
	if _, err := parseAgents("cursor"); err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("unknown agent: %v", err)
	}
}

// Found in a real race: a long task ran on under the header's buttons, a
// binary file's diff printed its bytes, and the merge question broke a word
// in two.
func TestLongTaskBinaryFileAndWrap(t *testing.T) {
	root, base := newRepo(t)
	u := newUI(root, base, strings.Repeat("fix the bugs in stats.py ", 10), nil)
	u.notify, u.wake = func(string, string) {}, func() {}
	r := fake(t, `sleep 30`)
	r.name, r.task, r.branch = "agy", u.prompt, "squeeze/t/agy"
	r.dir = filepath.Join(worktreesDir(root), "t", "agy")
	if err := addWorktree(root, r.dir, r.branch, base); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.dir, "a.pyc"), []byte{0, 1, 2, 'G', 'A', 'R', 'B', 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	r.refresh(base)
	u.racers = []*racer{r}
	u.compare()

	term := testkit.NewTerminal(100, 30)
	term.Draw(func(f *limoni.Frame) { u.frame(f, nil) })
	screen := term.Snapshot()
	top := strings.Split(screen, "\n")[0]
	if !strings.Contains(top, "…  m Merge   Esc Back   q Quit") {
		t.Errorf("header: %q", top)
	}
	if strings.Contains(screen, "GARB") || !strings.Contains(screen, "a.pyc is a binary file") {
		t.Errorf("the binary file's diff:\n%s", screen)
	}

	if got := wrap("agy's changes go into your working tree, staged, not committed.", 58); len(got) != 2 || got[0] != "agy's changes go into your working tree, staged, not" || got[1] != "committed." {
		t.Errorf("wrap = %q", got)
	}
	if got := wrap("abcdefgh", 3); strings.Join(got, "|") != "abc|def|gh" {
		t.Errorf("a word longer than the line: %q", got)
	}
}
