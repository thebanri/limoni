package main

import (
	"os/exec"
	"strings"
	"sync"
)

// onPath reports whether an agent's command can be run. A var for tests.
var onPath = func(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// installs is what became of the agents installed from the setup screen, by
// name: "installing", "installed", or what went wrong.
type installs struct {
	mu    sync.Mutex
	state map[string]string
}

func (in *installs) get(name string) string {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.state[name]
}

func (in *installs) set(name, state string) {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.state == nil {
		in.state = map[string]string{}
	}
	in.state[name] = state
}

// askInstall asks before installing def: the command runs code from the
// internet, so the user sees it first.
func (u *ui) askInstall(def agentDef) {
	if def.Install == "" {
		u.say(def.Name + " has no installer squeeze knows: " + def.Desc)
		return
	}
	u.pending = def
	u.ask("install")
}

// installAgent runs def's install command in the background.
func (u *ui) installAgent(def agentDef) {
	u.inst.set(def.Name, "installing")
	go func() {
		out, err := exec.Command("sh", "-c", def.Install).CombinedOutput()
		switch {
		case err != nil:
			last := strings.TrimSpace(string(out))
			if i := strings.LastIndexByte(last, '\n'); i >= 0 {
				last = last[i+1:]
			}
			if last == "" {
				last = err.Error()
			}
			u.inst.set(def.Name, "failed: "+last)
		case !onPath(def.Bin):
			u.inst.set(def.Name, "failed: installed, but "+def.Bin+" is not on PATH (add ~/.local/bin)")
		default:
			u.inst.set(def.Name, "installed")
		}
		if u.wake != nil {
			u.wake()
		}
	}()
}
