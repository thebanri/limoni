package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// With -notes, each racer is told where to write, and what one writes
// reaches the others on its job — not a racer on another job — while the
// notes stay out of every diff and merge.
func TestRacersShareNotes(t *testing.T) {
	root, base := newRepo(t)
	u := newUI(root, base, "task", nil)
	u.notify, u.wake = func(string, string) {}, func() {}
	u.notes, u.raceID, u.paneW, u.paneH = true, "t", 40, 10
	u.jobs = []string{"find the leak", "other job"}
	if err := excludeSqueeze(root); err != nil {
		t.Fatal(err)
	}
	told := t.TempDir()
	agent := agentDef{Name: "fake", Bin: "sh", Args: func(p, _ string) []string {
		return []string{"-c", `printf %s "$1" > "$2"; read line`, "sh", p, filepath.Join(told, "prompt")}
	}}
	for name, job := range map[string]int{"claude": 0, "agy": 0, "codex": 1} {
		if u.launch(agent, name, job) < 0 {
			t.Fatalf("%s: %s", name, u.status)
		}
	}
	t.Cleanup(func() {
		for _, r := range u.racers {
			r.stop()
		}
	})
	byName := map[string]*racer{}
	for _, r := range u.racers {
		byName[r.name] = r
	}
	claude, agy, codex := byName["claude"], byName["agy"], byName["codex"]

	deadline := time.Now().Add(5 * time.Second)
	var prompt []byte
	for len(prompt) == 0 && time.Now().Before(deadline) {
		prompt, _ = os.ReadFile(filepath.Join(told, "prompt"))
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(string(prompt), notesDir+"/") {
		t.Errorf("the agent was not told about the notes: %q", prompt)
	}
	if claude.task != "find the leak" {
		t.Errorf("the task shown is %q, the notes prompt included", claude.task)
	}

	if err := os.WriteFile(filepath.Join(claude.dir, notesDir, "claude.md"), []byte("the leak is in cache.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shareNotes(u.racers)
	if got, _ := os.ReadFile(filepath.Join(agy.dir, notesDir, "claude.md")); string(got) != "the leak is in cache.go\n" {
		t.Errorf("agy reads %q", got)
	}
	if _, err := os.Stat(filepath.Join(codex.dir, notesDir, "claude.md")); !os.IsNotExist(err) {
		t.Errorf("claude's notes reached codex, on another job: %v", err)
	}

	for _, r := range []*racer{claude, agy} {
		if files, _ := changes(r.dir, base); len(files) != 0 {
			t.Errorf("%s's notes show as changes: %+v", r.name, files)
		}
	}
	if err := merge(root, agy.dir, agy.branch, "agy"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, notesDir)); !os.IsNotExist(err) {
		t.Errorf("a merge brought the notes into the main tree: %v", err)
	}
}
