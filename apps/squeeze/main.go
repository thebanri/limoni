// squeeze gives one coding task to several agents at once — Claude Code,
// Codex, Antigravity, Gemini CLI — each in its own git worktree, shows them
// working side by side, and lets you compare what each changed and merge
// the one you like.
//
//	squeeze                          # write the task, pick the agents
//	squeeze "add a --json flag"      # race every installed agent on it
//	squeeze -agents claude,claude,agy "fix the flaky test"
//	squeeze clean                    # remove the worktrees and branches races left
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/thebanri/limoni"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "squeeze:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("squeeze", flag.ContinueOnError)
	agentList := fs.String("agents", "", "agents to race, comma separated (claude, codex, agy, gemini); default: every one installed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := git(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return errors.New("run squeeze inside a git repository: each agent works in a worktree of it")
	}
	root = strings.TrimSpace(root)
	if fs.Arg(0) == "clean" {
		return clean(root)
	}
	base, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		return errors.New("the repository has no commit yet: the agents' worktrees start from HEAD")
	}

	agents := installed()
	if *agentList != "" {
		if agents, err = parseAgents(*agentList); err != nil {
			return err
		}
	}
	u := newUI(root, strings.TrimSpace(base), strings.Join(fs.Args(), " "), agents)
	if dirty, _ := git(root, "status", "--porcelain", "--untracked-files=no"); strings.TrimSpace(dirty) != "" {
		u.status = "Your uncommitted changes are not in the agents' worktrees: they start from HEAD."
	}

	term, err := limoni.New()
	if err != nil {
		return err
	}
	defer term.Close()
	app := limoni.NewApp(term, limoni.WithCatchCtrlC(true), limoni.WithFPS(2), limoni.WithTitle("squeeze"))
	u.wake = app.Wakeup
	u.notify = func(title, body string) { app.Notify(title, body) }
	err = app.Run(context.Background(), u.frame)
	for _, r := range u.racers {
		r.stop()
	}
	if len(u.racers) > 0 {
		fmt.Printf("The agents' worktrees are in %s; `squeeze clean` removes them.\n", worktreesDir(root))
	}
	return err
}

// clean removes every worktree and branch squeeze made in this repository.
func clean(root string) error {
	out, err := git(root, "worktree", "list", "--porcelain")
	if err != nil {
		return err
	}
	prefix := worktreesDir(root) + string(os.PathSeparator)
	n := 0
	for _, line := range strings.Split(out, "\n") {
		path, ok := strings.CutPrefix(line, "worktree ")
		if !ok || !strings.HasPrefix(path, prefix) {
			continue
		}
		branch, _ := git(path, "rev-parse", "--abbrev-ref", "HEAD")
		if err := removeWorktree(root, path, strings.TrimSpace(branch)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		n++
	}
	_ = os.RemoveAll(worktreesDir(root))
	_, _ = git(root, "worktree", "prune")
	removeBranches(root)
	fmt.Printf("Removed %d worktrees.\n", n)
	return nil
}
