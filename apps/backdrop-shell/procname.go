package main

import (
	"path/filepath"
	"strings"
	"sync"
)

// Terminals that name a tab after the program running in it — Konsole's
// default title is "%d : %n", the foreground process's directory and name —
// find the wrapper there, not the shell, and the tab reads "backdrop-shell"
// whatever runs inside. So the wrapper takes on the name of the program in the
// foreground of its own terminal: bash at the prompt, htop while htop runs.
//
// How the name is read and written is the platform's business: on Unix it is
// /proc, which Windows does not have (platform_unix.go, platform_windows.go).
// Where the trick does not apply, nothing changes.
type procName struct {
	mu   sync.Mutex
	pgrp int    // the group last seen, 0 for none yet
	name string // the name last given, "" for none
}

// set renames the wrapper, if the name is new.
func (p *procName) set(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.setLocked(name)
}

// withOwnDir returns env with the directory of this binary at the end of
// PATH, if it is not in PATH already. install.sh puts the binary in
// $(go env GOPATH)/bin, which many systems — Ubuntu and Kubuntu among them —
// leave out of PATH: the start-up hook names the binary by its full path, so
// the scene appears, but "backdrop-shell opacity 0.3" typed at the prompt
// would be "command not found".
func withOwnDir(env []string, exe string) []string {
	dir := filepath.Dir(exe)
	if exe == "" || !filepath.IsAbs(dir) {
		return env
	}
	for i, kv := range env {
		path, ok := strings.CutPrefix(kv, "PATH=")
		if !ok {
			continue
		}
		for _, d := range filepath.SplitList(path) {
			if filepath.Clean(d) == dir {
				return env
			}
		}
		out := append([]string(nil), env...)
		if path == "" {
			out[i] = "PATH=" + dir
		} else {
			out[i] = kv + string(filepath.ListSeparator) + dir
		}
		return out
	}
	return append(env, "PATH="+dir)
}
