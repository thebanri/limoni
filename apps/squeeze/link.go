package main

import (
	"os"
	"path/filepath"
	"strings"
)

// linkedDirs are the dependency directories a worktree gets as symlinks to
// the repository root's copy, when git ignores them there. They are what an
// agent needs to build or run tests without reinstalling, and they are
// installed rather than built, so sharing one is safe enough. Build output
// (target, dist, build, .next) is deliberately absent: agents building at
// once into one shared directory would trample each other.
// shortcut: top level only, nested node_modules (monorepo packages) are not
// linked; upgrade by walking the tree if monorepos matter.
var linkedDirs = []string{"node_modules", ".venv", "venv", "vendor"}

// linkIgnored symlinks the ignored dependency directories of root into the
// worktree at dir. A directory git does not ignore is tracked or untracked
// project content, so it is left alone (a tracked one is already checked out).
func linkIgnored(root, dir string) error {
	exclude, err := git(root, "rev-parse", "--git-path", "info/exclude")
	if err != nil {
		return err
	}
	exclude = strings.TrimSpace(exclude)
	if !filepath.IsAbs(exclude) {
		exclude = filepath.Join(root, exclude)
	}
	for _, name := range linkedDirs {
		if fi, err := os.Stat(filepath.Join(root, name)); err != nil || !fi.IsDir() {
			continue
		}
		if _, err := git(root, "check-ignore", "-q", name); err != nil {
			continue // not ignored (git exits 1)
		}
		if err := os.Symlink(filepath.Join(root, name), filepath.Join(dir, name)); err != nil {
			return err
		}
		// A pattern like "node_modules/" matches directories only, so the
		// symlink would show up as a change; exclude it by name.
		f, err := os.OpenFile(exclude, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			return err
		}
		_, err = f.WriteString("/" + name + "\n")
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
	}
	return nil
}
