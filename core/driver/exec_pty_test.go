//go:build linux

package driver

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"testing"
	"time"
)

// Before Start there is no process: reads and writes fail, and stopping is a
// no-op rather than a nil dereference.
func TestExecPTYAdapterBeforeStart(t *testing.T) {
	p := NewExecPTYAdapter("cat")
	if _, err := p.Read(make([]byte, 1)); err == nil {
		t.Error("Read before Start succeeded")
	}
	if _, err := p.Write([]byte("x")); err == nil {
		t.Error("Write before Start succeeded")
	}
	if err := p.Stop(); err != nil {
		t.Errorf("Stop before Start: %v", err)
	}
}

func TestExecPTYAdapterStartFailsForAMissingProgram(t *testing.T) {
	if err := NewExecPTYAdapter("/nonexistent/limoni-test").Start(); err == nil {
		t.Fatal("Start of a missing program succeeded")
	}
}

// A line written to the child comes back from it, the size is remembered even
// though resizing a plain process is left to the platform adapters, and after
// Stop the stream ends.
func TestExecPTYAdapterRunsAProcess(t *testing.T) {
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("no cat")
	}
	p := NewExecPTYAdapter("cat")
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Stop() })

	if _, err := p.Write([]byte("hello, limoni\n")); err != nil {
		t.Fatal(err)
	}
	line := make(chan string, 1)
	go func() {
		s, _ := bufio.NewReader(p).ReadString('\n')
		line <- s
	}()
	select {
	case got := <-line:
		if got != "hello, limoni\n" {
			t.Fatalf("read back %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cat never echoed the line")
	}

	if err := p.Resize(80, 24); err == nil {
		t.Error("Resize claimed to resize a plain process")
	}
	if w, h, err := p.Size(); err != nil || w != 80 || h != 24 {
		t.Fatalf("Size after Resize = %d, %d, %v; want 80, 24", w, h, err)
	}

	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	// Stop once only killed the child, so a write could still land in the pipe.
	if _, err := p.Write([]byte("x")); err == nil {
		t.Error("Write after Stop succeeded")
	}
	if err := p.Stop(); err != nil {
		t.Errorf("second Stop: %v", err)
	}
	ended := make(chan error, 1)
	go func() {
		_, err := p.Read(make([]byte, 64))
		ended <- err
	}()
	select {
	case err := <-ended:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("read after Stop: %v, want EOF", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stream still open after Stop")
	}
}
