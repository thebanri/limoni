package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func waitCheck(t *testing.T, c *check) (checkState, string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		st, out, _ := c.result()
		if st != checking {
			return st, string(out)
		}
		if time.Now().After(deadline) {
			t.Fatal("the check never finished")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A check passes on exit 0 and fails otherwise, in the worktree, with its
// output; it is not run again on a tree it has seen unless forced, and is
// run again once the tree changes.
func TestCheckRunsInTheWorktree(t *testing.T) {
	root, base := newRepo(t)
	count := filepath.Join(t.TempDir(), "runs") // outside the tree, or counting would change it
	cmd := "echo >> " + count + `; grep -q 'func b' main.go && echo ok || { echo "no b in $PWD"; exit 1; }`
	var c check
	runs := func() string {
		data, _ := os.ReadFile(count)
		return strconv.Itoa(strings.Count(string(data), "\n"))
	}

	c.run(cmd, root, base, false, func() {})
	if st, out := waitCheck(t, &c); st != failed || !strings.Contains(out, "no b in "+root) {
		t.Fatalf("before the fix: %v %q, want failed with the worktree's path", st, out)
	}
	c.run(cmd, root, base, false, func() {})
	waitCheck(t, &c)
	if runs() != "1" {
		t.Errorf("runs = %s on an unchanged tree, want 1", runs())
	}
	c.run(cmd, root, base, true, func() {})
	waitCheck(t, &c)
	if runs() != "2" {
		t.Errorf("runs = %s after a forced run, want 2", runs())
	}

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc b() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.run(cmd, root, base, false, func() {})
	if st, out := waitCheck(t, &c); st != passed || strings.TrimSpace(out) != "ok" || runs() != "3" {
		t.Errorf("after the fix: %v %q runs=%s, want passed \"ok\" 3", st, out, runs())
	}
}

// Stopping kills the check and what it started, and no new run begins.
func TestCheckStopKillsIt(t *testing.T) {
	root, base := newRepo(t)
	var c check
	c.run("sleep 30 & wait", root, base, false, func() {})
	time.Sleep(300 * time.Millisecond)
	start := time.Now()
	c.stop()
	if st, _ := waitCheck(t, &c); st != failed || time.Since(start) > 2*time.Second {
		t.Errorf("stopped check: %v after %v", st, time.Since(start))
	}
	c.run("true", root, base, true, func() {})
	if st, _, _ := c.result(); st == checking {
		t.Error("a stopped check started again")
	}
}

// The race screen runs the check when an agent stops working, and marks it
// in the sidebar.
func TestTheRaceChecksAnAgentThatStops(t *testing.T) {
	root, base := newRepo(t)
	u := newUI(root, base, "task", nil)
	u.check = "echo checked"
	u.notify = func(string, string) {}
	u.wake = func() {}
	r := fake(t, `printf 'x\007'`)
	r.dir = root
	u.racers = []*racer{r}
	deadline := time.Now().Add(5 * time.Second)
	for r.state(time.Now()) == working {
		if time.Now().After(deadline) {
			t.Fatal("the fake agent never stopped")
		}
		time.Sleep(10 * time.Millisecond)
	}
	u.stopped(time.Now())
	if st, out := waitCheck(t, &r.chk); st != passed || strings.TrimSpace(out) != "checked" {
		t.Errorf("check = %v %q", st, out)
	}
	if mark, _ := checkMark(&r.chk); mark != "✓" {
		t.Errorf("sidebar mark = %q", mark)
	}
}

// A second squeeze in a repository is refused while the first runs, and
// allowed once it has gone, or when the pid left behind is dead.
func TestOneSqueezePerRepository(t *testing.T) {
	root, _ := newRepo(t)
	unlock, err := lock(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock(root); err == nil || !strings.Contains(err.Error(), "another squeeze") {
		t.Errorf("second lock: %v", err)
	}
	unlock()
	if status, _ := git(root, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("the lock shows in git status: %q", status)
	}
	if err := os.WriteFile(filepath.Join(root, ".squeeze", "pid"), []byte("999999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := lock(root)
	if err != nil {
		t.Fatalf("a dead pid's lock: %v", err)
	}
	again()
}
