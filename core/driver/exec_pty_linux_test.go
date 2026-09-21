//go:build linux

package driver

import (
	"io"
	"testing"
	"time"
)

// ExecPTYAdapter is a process-backed stream, not a kernel PTY. Drive cat:
// write a line, read it back, record a size, stop.
func TestExecPTYAdapterCatRoundTrip(t *testing.T) {
	p := NewExecPTYAdapter("cat")
	if err := p.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop() })

	const line = "hello limoni\n"
	if n, err := p.Write([]byte(line)); err != nil || n != len(line) {
		t.Fatalf("write: n=%d err=%v", n, err)
	}

	buf := make([]byte, 64)
	done := make(chan struct {
		n   int
		err error
	}, 1)
	go func() {
		n, err := io.ReadAtLeast(p, buf, len(line))
		done <- struct {
			n   int
			err error
		}{n, err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("read: %v", r.err)
		}
		if got := string(buf[:r.n]); got != line {
			t.Fatalf("read %q, want %q", got, line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cat")
	}

	if w, h, err := p.Size(); err != nil || w != 0 || h != 0 {
		t.Fatalf("size before resize: %d×%d err=%v", w, h, err)
	}
	if err := p.Resize(80, 24); err == nil {
		t.Fatal("resize: want platform-adapter error")
	}
	if w, h, err := p.Size(); err != nil || w != 80 || h != 24 {
		t.Fatalf("size after resize: %d×%d err=%v; want 80×24", w, h, err)
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if n, err := p.Write([]byte("x")); err == nil {
		t.Fatalf("write after stop: n=%d err=%v", n, err)
	}
}

func TestExecPTYAdapterReadWriteBeforeStart(t *testing.T) {
	p := NewExecPTYAdapter("cat")
	if n, err := p.Read(make([]byte, 1)); err == nil || n != 0 {
		t.Fatalf("read before start: n=%d err=%v", n, err)
	}
	if n, err := p.Write([]byte("x")); err == nil || n != 0 {
		t.Fatalf("write before start: n=%d err=%v", n, err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("stop before start: %v", err)
	}
}
