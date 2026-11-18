package main

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// tempControlDir keeps each test's sockets out of the real user's runtime
// directory, on every platform.
func tempControlDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
}

func TestControlSocketReload(t *testing.T) {
	tempControlDir(t)
	reload := make(chan struct{}, 1)
	live := make(chan liveConn, 1)
	stop, err := listenControl(reload, live)
	if err != nil {
		t.Fatalf("listenControl: %v", err)
	}
	defer stop()

	path := controlSocket()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no socket at %s: %v", path, err)
	}
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", path, err)
	}
	if _, err := conn.Write([]byte("reload\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.Close()

	select {
	case <-reload:
	case <-time.After(2 * time.Second):
		t.Fatal("the reload request did not arrive")
	}
	if n := reloadRunning(); n != 1 {
		t.Fatalf("reloadRunning reached %d wrappers, want 1", n)
	}
	stop()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stopping left the socket behind: %v", err)
	}
}

func TestControlSocketLive(t *testing.T) {
	tempControlDir(t)
	reload := make(chan struct{}, 1)
	live := make(chan liveConn, 1)
	stop, err := listenControl(reload, live)
	if err != nil {
		t.Fatalf("listenControl: %v", err)
	}
	defer stop()

	conn, err := net.DialTimeout("unix", controlSocket(), time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if _, err := io.WriteString(conn, "live colour 42\nframes\n"); err != nil {
		t.Fatalf("write: %v", err)
	}

	select {
	case lc := <-live:
		if lc.id != "colour" || lc.pid != 42 {
			t.Fatalf("live id=%q pid=%d, want colour and 42", lc.id, lc.pid)
		}
		buf := make([]byte, 6)
		if _, err := io.ReadFull(lc.rw, buf); err != nil || string(buf) != "frames" {
			t.Fatalf("frames %q, err %v", buf, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the live connection did not arrive")
	}
}

func TestReloadRunningRemovesStaleSocket(t *testing.T) {
	tempControlDir(t)
	if err := os.MkdirAll(controlDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(controlDir(), "12345.sock")
	if err := os.WriteFile(stale, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if n := reloadRunning(); n != 0 {
		t.Fatalf("reloadRunning reached %d wrappers, want 0", n)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a stale socket was left behind: %v", err)
	}
}
