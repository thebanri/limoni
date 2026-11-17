package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/testkit"
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

// On the setup screen F1 adds a copy of an agent, with no limit.
func TestTheSetupScreenRacesCopies(t *testing.T) {
	u := newUI(t.TempDir(), "", "", []agentDef{{Name: "claude", Bin: "sh"}, {Name: "agy", Bin: "sh"}})
	u.input.SetValue("task")
	u.key(limoni.KeyEvent{Type: limoni.KeyF1})
	u.begin()
	if got := strings.Join(racerNames(u.agents), ","); got != "claude,claude-2,agy" {
		t.Errorf("racers = %s", got)
	}
	// No limit: twelve picks race twelve.
	u = newUI(t.TempDir(), "", "", []agentDef{{Name: "claude", Bin: "sh"}})
	for range 11 {
		u.pick(0)
	}
	if u.chosen[0] != 12 {
		t.Errorf("after 11 more picks, %d copies, want 12", u.chosen[0])
	}
}

// Every screen draws at every window size, a tiny one included, where the
// panes would have had a negative width and x/vt panicked resizing to it.
func TestEverySizeDraws(t *testing.T) {
	u := newUI(t.TempDir(), "", "task", nil)
	u.notify, u.wake = func(string, string) {}, func() {}
	u.racers = []*racer{fake(t, `seq 1 100; sleep 5`)}
	for _, mode := range []mode{setupMode, raceMode, compareMode} {
		u.mode = mode
		for _, sz := range [][2]uint16{{1, 1}, {12, 5}, {minWidth - 1, 30}, {minWidth, minHeight}, {200, 60}, {80, 24}} {
			term := testkit.NewTerminal(sz[0], sz[1])
			term.Draw(func(f *limoni.Frame) { u.frame(f, nil) })
		}
	}
	term := testkit.NewTerminal(20, 5)
	term.Draw(func(f *limoni.Frame) { u.frame(f, nil) })
	if !strings.Contains(term.Snapshot(), "bigger") {
		t.Errorf("a tiny window shows\n%s", term.Snapshot())
	}
}

// More racers than the sidebar holds: the cards follow the selection, and a
// click picks the card it lands on.
func TestTheCardsFollowTheSelection(t *testing.T) {
	u := newUI(t.TempDir(), "", "task", nil)
	u.notify, u.wake = func(string, string) {}, func() {}
	for range 6 {
		r := fake(t, `sleep 5`)
		r.name = fmt.Sprintf("r%d", len(u.racers))
		u.racers = append(u.racers, r)
	}
	u.sel = 5
	f := frameOf(t, u) // 16 rows: three cards
	if !strings.Contains(strings.Join(f, "\n"), " r5 ") || strings.Contains(strings.Join(f, "\n"), " r0 ") {
		t.Fatalf("the selected card is not shown:\n%s", strings.Join(f, "\n"))
	}
	u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: 3, Y: 1 + 1})
	if u.sel != 3 {
		t.Errorf("a click on the first card shown picked %d, want 3", u.sel)
	}
}

// An agent not installed is offered on the setup screen with an install
// button: it asks first, runs the command, and then the agent can be picked.
// A failed install says why and offers to try again.
func TestInstallingAnAgentFromTheSetupScreen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	good := agentDef{Name: "fakeagent", Bin: "fakeagent", Desc: "a test agent",
		Install: `printf '#!/bin/sh\n' > "` + dir + `/fakeagent" && chmod +x "` + dir + `/fakeagent"`}
	bad := agentDef{Name: "broken", Bin: "brokenagent", Desc: "never installs", Install: `echo "no network" >&2; exit 1`}
	u := newUI(t.TempDir(), "", "", nil)
	u.wake = func() {}
	u.offer([]agentDef{good, bad})
	if u.chosen[0] != 0 {
		t.Fatal("an agent not installed starts picked")
	}
	press := func(label string) {
		t.Helper()
		f := frameOf(t, u)
		for _, b := range u.buttons {
			if row := []rune(f[b.y]); b.x+b.w <= len(row) && strings.Contains(string(row[b.x:b.x+b.w]), label) {
				u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: uint16(b.x + 1), Y: uint16(b.y)})
				return
			}
		}
		t.Fatalf("no %q button on\n%s", label, strings.Join(f, "\n"))
	}
	press("install")
	if u.confirm != "install" || u.pending.Name != "fakeagent" {
		t.Fatalf("installing did not ask first: %q %q", u.confirm, u.pending.Name)
	}
	time.Sleep(modalOpen) // it grows open first
	if f := strings.Join(frameOf(t, u), "\n"); !strings.Contains(f, "fakeagent\" && chmod") {
		t.Errorf("the modal does not show the command:\n%s", f)
	}
	press("Install")
	u.key(limoni.KeyEvent{Type: limoni.KeyF2})
	u.answer(true) // the second agent's install
	deadline := time.Now().Add(5 * time.Second)
	for (u.inst.get("fakeagent") == "installing" || u.inst.get("broken") == "installing") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if st := u.inst.get("fakeagent"); st != "installed" {
		t.Fatalf("install: %q", st)
	}
	u.key(limoni.KeyEvent{Type: limoni.KeyF1})
	if u.chosen[0] != 1 {
		t.Error("the installed agent cannot be picked")
	}
	f := strings.Join(frameOf(t, u), "\n")
	if !strings.Contains(f, "no network") || !strings.Contains(f, "retry") {
		t.Errorf("the failed install does not say why:\n%s", f)
	}
}
