package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// git runs git in dir and returns its output, or an error that carries
// what git said.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(errOut.String()+" "+err.Error()))
	}
	return out.String(), nil
}

// worktreesDir is where a race's worktrees live: inside the repository, so
// an agent that trusts the project trusts them too, and excluded from git
// in .git/info/exclude, which is not part of the project.
func worktreesDir(root string) string { return filepath.Join(root, ".squeeze", "worktrees") }

func excludeSqueeze(root string) error {
	common, err := git(root, "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	common = strings.TrimSpace(common)
	if !filepath.IsAbs(common) {
		common = filepath.Join(root, common)
	}
	path := filepath.Join(common, "info", "exclude")
	data, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "/.squeeze/" {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return os.WriteFile(path, append(data, "/.squeeze/\n"...), 0o644)
}

// addWorktree makes a worktree at path on a new branch from base.
func addWorktree(root, path, branch, base string) error {
	_, err := git(root, "worktree", "add", "-b", branch, path, base)
	return err
}

// reuseWorktree makes a worktree at path on a new branch from base. Agents
// remember trust per path, so a racer keeps one path across races: when path
// still holds an earlier race's worktree, whatever is uncommitted there is
// committed to its old branch (kept; `squeeze clean` deletes it) before the
// worktree is removed and the new one takes its place.
func reuseWorktree(root, path, branch, base string) error {
	if _, err := os.Stat(path); err == nil {
		if _, err := git(path, "add", "--all"); err != nil {
			return err
		}
		if out, _ := git(path, "status", "--porcelain"); strings.TrimSpace(out) != "" {
			if _, err := git(path, "commit", "--no-verify", "-m", "squeeze: work left in the worktree"); err != nil {
				return err
			}
		}
		if _, err := git(root, "worktree", "remove", "--force", path); err != nil {
			return err
		}
	}
	return addWorktree(root, path, branch, base)
}

// removeWorktree deletes a racer's worktree and its branch, changes and all.
func removeWorktree(root, path, branch string) error {
	_, err1 := git(root, "worktree", "remove", "--force", path)
	_, err2 := git(root, "branch", "-D", branch)
	return errors.Join(err1, err2)
}

// fileChange is one file a racer changed since base.
type fileChange struct {
	Path           string
	Added, Removed int // -1 for a binary file
}

// changes lists what a worktree changed since base: committed or not, new
// files included. Untracked files are marked intent-to-add first, which
// touches only the worktree's own index.
func changes(dir, base string) ([]fileChange, error) {
	if _, err := git(dir, "add", "--intent-to-add", "--all"); err != nil {
		return nil, err
	}
	out, err := git(dir, "diff", "--numstat", "--no-renames", base)
	if err != nil {
		return nil, err
	}
	var files []fileChange
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		a, errA := strconv.Atoi(parts[0])
		r, errR := strconv.Atoi(parts[1])
		if errA != nil || errR != nil {
			a, r = -1, -1
		}
		files = append(files, fileChange{Path: parts[2], Added: a, Removed: r})
	}
	return files, nil
}

// versions is a file at base and as it is in the worktree; either is empty
// where the file does not exist.
func versions(dir, base, path string) (before, after string) {
	before, _ = git(dir, "show", base+":"+path)
	data, _ := os.ReadFile(filepath.Join(dir, path))
	return before, string(data)
}

// merge brings a racer's changes into the main worktree as staged changes,
// for the user to look over and commit: git merge --squash of the racer's
// branch, after committing whatever the agent left uncommitted there.
func merge(root, dir, branch, name string) error {
	if _, err := git(dir, "add", "--all"); err != nil {
		return err
	}
	if out, _ := git(dir, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		if _, err := git(dir, "commit", "--no-verify", "-m", "squeeze: "+name); err != nil {
			return err
		}
	}
	_, err := git(root, "merge", "--squash", branch)
	return err
}

// removeBranches deletes every squeeze/* branch, including those of
// worktrees that a later race replaced.
func removeBranches(root string) {
	out, _ := git(root, "for-each-ref", "--format=%(refname:short)", "refs/heads/squeeze/")
	for _, b := range strings.Fields(out) {
		_, _ = git(root, "branch", "-D", b)
	}
}
