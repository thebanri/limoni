//go:build windows

package main

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestWatchResizeNoticesSizeChanges checks the Windows side of the resize
// seam: a console gives no signal, so the watcher has to poll for a new size,
// report only real changes, and stop when the session is done with it.
func TestWatchResizeNoticesSizeChanges(t *testing.T) {
	var step atomic.Int32
	sizes := [][2]int{{80, 24}, {100, 30}, {100, 24}}
	first := make(chan struct{}, 1)
	size := func(int) (int, int) {
		i := int(step.Load())
		if i >= len(sizes) {
			i = len(sizes) - 1
		}
		if i == 1 { // the watcher's baseline, taken before it polls
			select {
			case first <- struct{}{}:
			default:
			}
		}
		return sizes[i][0], sizes[i][1]
	}

	// Establish the baseline before the watcher starts, so changing to the
	// next size is a real change and not just its first sample.
	step.Store(1)
	changed := make(chan struct{}, 4)
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		watchResizeWith(0, size, 5*time.Millisecond, changed, done)
	}()
	select {
	case <-first:
	case <-time.After(time.Second):
		t.Fatal("the watcher never took a baseline")
	}

	for i := 2; i < len(sizes); i++ {
		step.Store(int32(i))
		select {
		case <-changed:
		case <-time.After(time.Second):
			t.Fatalf("no report of size %v", sizes[i])
		}
	}
	select {
	case <-changed:
		t.Fatal("a size that did not change was reported")
	case <-time.After(30 * time.Millisecond):
	}

	close(done)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("the watcher did not stop when done was closed")
	}
}
