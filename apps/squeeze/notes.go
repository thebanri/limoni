package main

import (
	"bytes"
	"os"
	"path/filepath"
)

// notesDir is where, with -notes, a racer writes its notes and finds the
// other racers'. Each racer writes only <its name>.md there, inside its own
// worktree (outside it, agents ask before writing), and shareNotes copies
// it to the others: a symlinked shared file would be replaced, not written
// through, by an editor that saves to a temporary file and renames it.
const notesDir = ".squeeze-notes"

// notesPrompt is what a racer is told about the notes, after its task.
func notesPrompt(name string) string {
	return "\n\nOther agents are working on this same task at the same time, each in its own copy of the repository. " +
		"Write what you learn — what you tried, what worked, dead ends — to " + notesDir + "/" + name + ".md as you go, " +
		"and read the other files in " + notesDir + "/ now and then: they are the other agents' notes. " +
		"Only your own file reaches them, and nothing in " + notesDir + "/ is part of your change."
}

// shareNotes copies each racer's notes into the notes directory of every
// other racer on its job.
func shareNotes(racers []*racer) {
	for _, from := range racers {
		data, err := os.ReadFile(filepath.Join(from.dir, notesDir, from.name+".md"))
		if err != nil {
			continue
		}
		for _, to := range racers {
			if to == from || to.job != from.job {
				continue
			}
			path := filepath.Join(to.dir, notesDir, from.name+".md")
			if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
				continue
			}
			_ = os.MkdirAll(filepath.Dir(path), 0o755)
			_ = os.WriteFile(path, data, 0o644)
		}
	}
}
