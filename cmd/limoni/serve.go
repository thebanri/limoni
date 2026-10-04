package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

// limoni serve runs a terminal program once per browser tab, in a
// pseudo-terminal, and draws it with xterm.js. It is how a Limoni app (or any
// terminal program) is shown to someone without a terminal: a demo, a
// workshop, a screen share.
//
// Whoever can open the page runs the program as you. So the server listens
// on loopback unless told otherwise, and every request must carry a random
// token printed at start, the way Jupyter does it.

var errNoPTY = errors.New("limoni serve needs a pseudo-terminal, which this build supports on Linux and macOS")

type serveConfig struct {
	addr     string
	token    string
	maxConns int
	command  []string
	stderr   io.Writer
}

func runServe(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(out)
	addr := fs.String("addr", "127.0.0.1:7681", "address to listen on; anything but loopback lets other machines run the program")
	token := fs.String("token", "", "the token every request must carry (default: a random one)")
	maxConns := fs.Int("max", 8, "most programs running at once")
	fs.Usage = func() {
		fmt.Fprint(out, `limoni serve [flags] [--] <program> [args...]

Runs <program> in a pseudo-terminal for each browser tab that opens the
printed URL, and draws it with xterm.js. Build the program first: with
"go run" every tab compiles it again.

  limoni serve ./myapp
  limoni serve -addr :8080 -- ./myapp -theme dark

Flags:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return errors.New("serve: no program given")
	}
	if !ptySupported {
		return errNoPTY
	}
	cfg := serveConfig{addr: *addr, token: *token, maxConns: *maxConns, command: fs.Args(), stderr: os.Stderr}
	if cfg.token == "" {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return err
		}
		cfg.token = hex.EncodeToString(b[:])
	}
	if _, err := exec.LookPath(cfg.command[0]); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	ln, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return err
	}
	handler := newServeHandler(cfg)
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	shown := host
	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		shown = "localhost"
	}
	fmt.Fprintf(out, "Serving %s at\n\n  http://%s/?token=%s\n\n", strings.Join(cfg.command, " "), net.JoinHostPort(shown, port), cfg.token)
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		fmt.Fprintln(out, "Not on loopback: anyone who can reach this address and has the token runs the program as you.")
	}
	fmt.Fprintln(out, "Ctrl+C stops the server and every program it started.")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	err = srv.Serve(ln)
	// Shutdown does not close upgraded connections: hang up their programs.
	handler.stopAll()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

type serveHandler struct {
	cfg   serveConfig
	slots chan struct{}

	mu       sync.Mutex
	sessions map[*exec.Cmd]struct{}
}

func newServeHandler(cfg serveConfig) *serveHandler {
	if cfg.maxConns < 1 {
		cfg.maxConns = 1
	}
	return &serveHandler{cfg: cfg, slots: make(chan struct{}, cfg.maxConns), sessions: map[*exec.Cmd]struct{}{}}
}

func (h *serveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		http.Error(w, "missing or wrong token: open the URL limoni serve printed", http.StatusForbidden)
		return
	}
	switch r.URL.Path {
	case "/":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer") // the token is in the URL
		title := strings.Join(h.cfg.command, " ")
		_, _ = io.WriteString(w, strings.Replace(servePage, "{{TITLE}}", htmlEscape(title), 1))
	case "/ws":
		h.serveTerminal(w, r)
	default:
		http.NotFound(w, r)
	}
}

// authorized checks the token, and for the WebSocket also that the page
// asking is this server's own: a browser sends Origin with every WebSocket,
// and another site's page must not reach a program running as you.
func (h *serveHandler) authorized(r *http.Request) bool {
	got := r.URL.Query().Get("token")
	if subtle.ConstantTimeCompare([]byte(got), []byte(h.cfg.token)) != 1 {
		return false
	}
	if r.URL.Path == "/ws" {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // not a browser
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host {
			return false
		}
	}
	return true
}

func (h *serveHandler) serveTerminal(w http.ResponseWriter, r *http.Request) {
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		http.Error(w, "too many programs running; close a tab or raise -max", http.StatusServiceUnavailable)
		return
	}
	cols, rows := sizeParam(r, "cols", 80), sizeParam(r, "rows", 24)

	ws, err := wsAccept(w, r)
	if err != nil {
		return
	}
	defer ws.conn.Close()

	cmd := exec.Command(h.cfg.command[0], h.cfg.command[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	master, err := startInPTY(cmd, cols, rows)
	if err != nil {
		_ = ws.WriteBinary([]byte("limoni serve: " + err.Error() + "\r\n"))
		_ = ws.Close(1011)
		return
	}
	h.track(cmd, true)
	defer h.track(cmd, false)

	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()

	// The program's output to the browser.
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 32*1024)
		for {
			n, err := master.Read(buf)
			if n > 0 && ws.WriteBinary(buf[:n]) != nil {
				return
			}
			if err != nil {
				return // EIO once nothing holds the terminal open, or the close below
			}
		}
	}()
	// The browser's keys and resizes to the program.
	gone := make(chan struct{})
	go func() {
		defer close(gone)
		for {
			op, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			switch op {
			case wsBinary:
				if _, err := master.Write(msg); err != nil {
					return
				}
			case wsText:
				var ctl struct {
					Type       string
					Cols, Rows uint16
				}
				if json.Unmarshal(msg, &ctl) == nil && ctl.Type == "resize" && ctl.Cols > 0 && ctl.Rows > 0 {
					_ = setPTYSize(master, ctl.Cols, ctl.Rows)
				}
			}
		}
	}()

	select {
	case <-exited:
		// Let the last output through; a child left in the background may
		// hold the terminal open, so do not wait for it.
		select {
		case <-done:
		case <-time.After(300 * time.Millisecond):
		}
	case <-done:
	case <-gone:
	}
	// Hang up the whole session, as closing a terminal window does, and
	// kill what ignores that.
	hangUp(cmd)
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		killSession(cmd)
		<-exited
	}
	master.Close()
	<-done
	_ = ws.Close(1000)
}

// stopAll hangs up every running program: a stopping server does not leave
// them behind.
func (h *serveHandler) stopAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for cmd := range h.sessions {
		hangUp(cmd)
	}
}

// track records running programs so a stopping server can hang them all up.
func (h *serveHandler) track(cmd *exec.Cmd, running bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if running {
		h.sessions[cmd] = struct{}{}
	} else {
		delete(h.sessions, cmd)
	}
}

func sizeParam(r *http.Request, name string, fallback uint16) uint16 {
	n, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || n < 2 || n > 1000 {
		return fallback
	}
	return uint16(n)
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
