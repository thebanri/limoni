package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	limoni "github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/testkit"
)

// fakeAgent is a racer that puts its terminal in raw mode, says READY and
// logs every byte it receives to log.
func fakeAgent(name, log string) agentDef {
	return agentDef{Name: name, Bin: "sh", Args: func(string, string) []string {
		return []string{"-c", `stty raw -echo; echo READY; exec cat >> "$1"`, "sh", log}
	}}
}

func hex(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		b.WriteString(strings.ToUpper(string("0123456789abcdef"[c>>4])) + string("0123456789abcdef"[c&15]) + " ")
	}
	return strings.TrimSpace(b.String())
}

// Keys sent in a burst right after the race starts — Enter, Down, Enter,
// Ctrl+], Down — must reach the racer that was selected when they were
// typed, and no other.
func TestKeysReachOnlyTheSelectedRacer(t *testing.T) {
	root, base := newRepo(t)
	dir := t.TempDir()
	logA, logB := filepath.Join(dir, "a.log"), filepath.Join(dir, "b.log")
	for _, wait := range []bool{true, false} {
		os.Remove(logA)
		os.Remove(logB)
		u := newUI(root, base, "task", []agentDef{fakeAgent("claude", logA), fakeAgent("agy", logB)})
		u.raceID = time.Now().Format("150405.000")
		u.wake = func() {}
		u.notify = func(string, string) {}
		term := testkit.NewTerminal(100, 30)
		step := func(k driver.KeyEvent) {
			term.Draw(func(f *limoni.Frame) { u.frame(f, &limoni.Event{Type: limoni.EventKey, Key: k}) })
		}
		term.Draw(func(f *limoni.Frame) { u.frame(f, nil) })
		if len(u.racers) != 2 {
			t.Fatalf("racers: %d (%s)", len(u.racers), u.status)
		}
		defer func() {
			for _, r := range u.racers {
				r.stop()
			}
		}()
		if wait {
			time.Sleep(700 * time.Millisecond) // both are in raw mode
		}
		step(driver.KeyEvent{Type: driver.KeyEnter})
		step(driver.KeyEvent{Type: driver.KeyArrowDown})
		step(driver.KeyEvent{Type: driver.KeyEnter})
		step(driver.KeyEvent{Type: driver.KeyRune, Ch: ']', Ctrl: true})
		step(driver.KeyEvent{Type: driver.KeyArrowDown})
		time.Sleep(700 * time.Millisecond)
		a, _ := os.ReadFile(logA)
		b, _ := os.ReadFile(logB)
		if len(b) != 0 {
			t.Errorf("wait=%v: agy (not attached) received % x", wait, b)
		}
		if len(a) == 0 {
			t.Errorf("wait=%v: claude received nothing", wait)
		}
		t.Logf("wait=%v claude got: %s", wait, hex(string(a)))
		if u.sel != 1 || u.attached {
			t.Errorf("sel=%d attached=%v", u.sel, u.attached)
		}
	}
}

// A key typed before the racers exist (the first frame starts them) must
// not push the selection to -1, which panics the next draw.
func TestKeyBeforeRacersExistKeepsSelectionValid(t *testing.T) {
	u := newUI("", "", "task", nil)
	u.raceKey(driver.KeyEvent{Type: driver.KeyArrowDown}, true)
	if u.sel != 0 {
		t.Errorf("sel = %d with no racers", u.sel)
	}
}
