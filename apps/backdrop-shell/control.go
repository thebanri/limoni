package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Changing the settings of terminals that are already open.
//
// Each running wrapper listens on a Unix socket of its own, in a directory
// only its user can enter. A command that saves the settings — opacity,
// enable, reset — then asks every one of them to read the settings again,
// so the new opacity or scene shows at once in every open terminal instead
// of in the next one.
//
// A socket rather than a signal: a wrapper that crashed leaves a socket file
// that simply refuses the connection, where a stale process number may by
// now belong to another program, and SIGUSR1 would kill it. Waiting on the
// socket costs nothing while nobody writes to it.

// controlDir is where the wrappers' sockets are: the user's runtime
// directory, or a private one in the temporary directory.
func controlDir() string {
	// A socket's path must fit in about a hundred bytes; a runtime directory
	// too deep for that falls back to the temporary one.
	const room = 100 - len("/limoni-backdrop/4194304.sock")
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" && len(d) <= room {
		return filepath.Join(d, "limoni-backdrop")
	}
	return filepath.Join(os.TempDir(), "limoni-backdrop-"+strconv.Itoa(userID()))
}

// controlSocket is this wrapper's socket.
func controlSocket() string {
	return filepath.Join(controlDir(), strconv.Itoa(os.Getpid())+".sock")
}

// listenControl serves this wrapper's socket until stop is called: a
// "reload" request sends on reload, and "live id pid" hands the connection,
// which then brings a live region's frames, to live (live.go).
func listenControl(reload chan<- struct{}, live chan<- liveConn) (stop func(), err error) {
	dir := controlDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := checkPrivateDir(dir); err != nil {
		return nil, err
	}
	path := controlSocket()
	_ = os.Remove(path) // left by an earlier process with this number
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveControl(conn, reload, live)
		}
	}()
	return func() {
		ln.Close()
		_ = os.Remove(path)
	}, nil
}

func serveControl(conn net.Conn, reload chan<- struct{}, live chan<- liveConn) {
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	br := bufio.NewReaderSize(conn, 64)
	first, _ := br.ReadSlice('\n')
	f := strings.Fields(string(first))
	switch {
	case len(f) == 1 && f[0] == "reload":
		conn.Close()
		select {
		case reload <- struct{}{}:
		default: // one is pending already
		}
	case len(f) == 3 && f[0] == "live":
		pid, _ := strconv.Atoi(f[2])
		_ = conn.SetReadDeadline(time.Time{})
		live <- liveConn{id: f[1], pid: pid, rw: struct {
			io.Reader
			io.Writer
		}{br, conn}, c: conn}
	default:
		conn.Close()
	}
}

// reloadRunning asks every running wrapper to read the settings again and
// returns how many were reached. Sockets nobody answers any more are
// removed.
func reloadRunning() int {
	paths, _ := filepath.Glob(filepath.Join(controlDir(), "*.sock"))
	reached := 0
	for _, p := range paths {
		conn, err := net.DialTimeout("unix", p, 500*time.Millisecond)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "refused") {
				_ = os.Remove(p)
			}
			continue
		}
		_, err = conn.Write([]byte("reload\n"))
		conn.Close()
		if err == nil {
			reached++
		}
	}
	return reached
}

// reload reads the settings again and shows them: a new opacity or
// background is drawn at once.
func (s *session) reload() {
	set, err := loadSettings(settingsPath())
	if err != nil {
		return
	}
	bd, err := chooseScene(set.Scene, set.Image, set.Art, set.SceneCmd, set.Still)
	if err != nil {
		return // keep what is shown rather than show nothing
	}
	closeScene(s.opts.backdrop) // a program drawing the old one stops
	s.opts.scene, s.opts.image, s.opts.art, s.opts.sceneCmd = set.Scene, set.Image, set.Art, set.SceneCmd
	s.opts.opacity, s.opts.fps, s.opts.still = set.Opacity, set.FPS, set.Still
	s.opts.backdrop = bd
	if s.opts.selection != set.Select {
		s.opts.selection = set.Select
		s.syncMouse()
	}
	s.scene = fadeInto(bd, s.termBg, s.opts.opacity)
	s.sceneFrame = -1
	s.draw()
}

// reportReload tells the user what a settings change reached.
func reportReload() {
	switch n := reloadRunning(); n {
	case 0:
		fmt.Println("New terminals will show it.")
	case 1:
		fmt.Println("Applied to the open terminal, and new ones will show it.")
	default:
		fmt.Printf("Applied to %d open terminals, and new ones will show it.\n", n)
	}
}

// cmdOpacity shows or sets how strongly the background shows: a number
// from 0 to 1, or a step up or down such as +0.1 or -0.1.
func cmdOpacity(args []string) error {
	s, err := loadSettings(settingsPath())
	if err != nil {
		return err
	}
	if len(args) == 0 {
		fmt.Printf("opacity %g (0 hides the background, 1 shows it at full strength)\n", s.Opacity)
		return nil
	}
	arg := args[0]
	v, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return fmt.Errorf("%q is not a number; give one from 0 to 1, or a step such as +0.1", arg)
	}
	if strings.HasPrefix(arg, "+") || strings.HasPrefix(arg, "-") {
		v += s.Opacity
	}
	s.Opacity = max(0, min(1, v))
	if err := s.save(settingsPath()); err != nil {
		return err
	}
	fmt.Printf("opacity %g\n", s.Opacity)
	reportReload()
	return nil
}
