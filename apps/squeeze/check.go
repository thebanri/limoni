package main

import (
	"context"
	"crypto/sha256"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// checkOutputMax is how much of a check's output is kept: the end of it,
// where test runners put the summary and the failure that stopped them.
const checkOutputMax = 64 << 10

type checkState int

const (
	unchecked checkState = iota
	checking
	passed
	failed
)

// check is the user's -check command, run in a racer's worktree whenever the
// agent stops working, so the racers are compared by whether their code
// passes rather than by eye.
type check struct {
	mu      sync.Mutex
	running bool
	state   checkState // of the last run that finished
	output  []byte
	took    time.Duration
	tree    [32]byte // what the worktree held at that run
	cancel  context.CancelFunc
	stopped bool
}

// run runs cmd with sh in dir, unless the worktree is as it was at the last
// run and force is unset. It returns at once; wake is called when it ends.
func (c *check) run(cmd, dir, base string, force bool, wake func()) {
	c.mu.Lock()
	if c.running || c.stopped {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()
	go func() {
		tree := treeHash(dir, base)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		c.mu.Lock()
		if c.stopped || !force && c.state != unchecked && tree == c.tree {
			c.running = false
			c.mu.Unlock()
			return
		}
		c.cancel = cancel
		c.mu.Unlock()
		wake()

		sh := exec.CommandContext(ctx, "sh", "-c", cmd)
		sh.Dir = dir
		// Its own process group, so stopping it stops what it started too.
		sh.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		sh.Cancel = func() error { return syscall.Kill(-sh.Process.Pid, syscall.SIGKILL) }
		start := time.Now()
		out, err := sh.CombinedOutput()

		c.mu.Lock()
		c.running = false
		c.output = out[max(len(out)-checkOutputMax, 0):]
		c.took = time.Since(start)
		c.tree = tree
		c.state = passed
		if err != nil {
			c.state = failed
			if len(out) == 0 {
				c.output = []byte(err.Error())
			}
		}
		c.mu.Unlock()
		wake()
	}()
}

// result is the check's state (checking while a run is under way), and the
// output and duration of the last run that finished.
func (c *check) result() (checkState, []byte, time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return checking, c.output, c.took
	}
	return c.state, c.output, c.took
}

// stop kills a run under way and refuses new ones.
func (c *check) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopped = true
	if c.cancel != nil {
		c.cancel()
	}
}

// treeHash identifies what a worktree holds, committed or not, new files
// included: the hash of its diff from base.
func treeHash(dir, base string) [32]byte {
	_, _ = git(dir, "add", "--intent-to-add", "--all")
	out, _ := git(dir, "diff", "--binary", base)
	return sha256.Sum256([]byte(out))
}
