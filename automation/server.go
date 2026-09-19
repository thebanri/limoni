package automation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/driver"
)

// maxSocketPathLen is the shortest sockaddr_un limit across the platforms
// Limoni supports (104 on macOS and the BSDs, 108 on Linux), minus room for the
// terminating NUL. Checked uniformly so a path that works on Linux does not
// surprise a macOS user later.
const maxSocketPathLen = 103

// Snapshot is what one frame published to the server: the semantic tree plus
// the rendered grid and focus, captured after a draw.
type Snapshot struct {
	Tree     []accessibility.AccessibilityNode
	Screen   string
	Focused  string
	Width    uint16
	Height   uint16
	Injector func(driver.Event)
}

// Server answers automation requests over a Unix socket.
//
// The zero value is not usable; call Listen.
type Server struct {
	listener net.Listener
	path     string

	policy Policy

	mu    sync.RWMutex
	state Snapshot

	closeOnce sync.Once
	done      chan struct{}
	wg        sync.WaitGroup

	connMu sync.Mutex
	conns  map[net.Conn]struct{}
}

// Option configures a Server.
type Option func(*Server)

// WithPolicy sets what the server lets leave the process. Without it the
// server exposes structure only: no input values, no screen text, and no
// input synthesis.
func WithPolicy(policy Policy) Option {
	return func(s *Server) { s.policy = policy }
}

// Listen creates the socket and starts accepting connections.
//
// The socket is created with 0600 permissions, and its directory is expected to
// be one only the user can write. There is no TCP equivalent on purpose: this
// interface synthesises input into a live application, and a port would offer
// that to anything that can reach the host.
func Listen(socketPath string, opts ...Option) (*Server, error) {
	if socketPath == "" {
		return nil, fmt.Errorf("automation: empty socket path")
	}
	// sockaddr_un caps the path at 104 bytes on macOS and the BSDs, 108 on
	// Linux. Over the limit, bind fails with a bare EINVAL that says nothing
	// about length, so check it here and say what is actually wrong. A deep
	// project directory or a Go test temp dir will hit this.
	if len(socketPath) > maxSocketPathLen {
		return nil, fmt.Errorf(
			"automation: socket path is %d bytes, over the %d-byte limit the OS allows for a Unix socket: %s",
			len(socketPath), maxSocketPathLen, socketPath)
	}
	if dir := filepath.Dir(socketPath); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("automation: create socket directory: %w", err)
		}
	}
	// A leftover socket from a crashed run would otherwise make every
	// subsequent start fail with "address already in use".
	if info, err := os.Stat(socketPath); err == nil && info.Mode()&os.ModeSocket != 0 {
		_ = os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("automation: listen: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("automation: restrict socket permissions: %w", err)
	}

	s := &Server{listener: listener, path: socketPath, done: make(chan struct{}), conns: make(map[net.Conn]struct{})}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	s.wg.Add(1)
	go s.accept()
	return s, nil
}

// Addr returns the socket path the server is listening on.
func (s *Server) Addr() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Publish replaces the snapshot served to clients. The application calls it
// once per frame, after drawing.
func (s *Server) Publish(state Snapshot) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
}

// Close stops the server, disconnects every client and removes the socket.
//
// Clients are disconnected rather than waited for: an attached agent or test
// keeps its connection open indefinitely, and an application must not hang on
// exit because something is watching it.
func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	var err error
	s.closeOnce.Do(func() {
		close(s.done)
		err = s.listener.Close()
		_ = os.Remove(s.path)
		s.connMu.Lock()
		for conn := range s.conns {
			_ = conn.Close()
		}
		s.connMu.Unlock()
	})
	s.wg.Wait()
	return err
}

func (s *Server) accept() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				// A transient accept error should not kill automation for the
				// rest of the process lifetime.
				continue
			}
		}
		s.connMu.Lock()
		select {
		case <-s.done:
			// Close ran between Accept and here and will not see this one.
			s.connMu.Unlock()
			_ = conn.Close()
			return
		default:
		}
		s.conns[conn] = struct{}{}
		s.wg.Add(1)
		s.connMu.Unlock()
		go func() {
			defer s.wg.Done()
			defer func() {
				s.connMu.Lock()
				delete(s.conns, conn)
				s.connMu.Unlock()
			}()
			s.serve(conn)
		}()
	}
}

func (s *Server) serve(conn net.Conn) {
	defer conn.Close()
	if err := s.verifyPeer(conn); err != nil {
		// One explanatory line, then the connection is gone. A client from the
		// wrong user learns only that it was refused.
		_ = json.NewEncoder(conn).Encode(Response{Err: err.Error()})
		// Read what the client already sent before closing. Closing a socket
		// with unread data makes Windows reset the connection, and the client
		// then reads a reset instead of this explanation. Bounded, so a
		// refused client cannot hold the connection open.
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		_, _ = io.Copy(io.Discard, io.LimitReader(conn, 64<<10))
		return
	}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		select {
		case <-s.done:
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = encoder.Encode(Response{Err: fmt.Sprintf("automation: malformed request: %v", err)})
			continue
		}
		if err := encoder.Encode(s.handle(req)); err != nil {
			return
		}
	}
}

func (s *Server) handle(req Request) Response {
	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()

	// Everything below works on the redacted tree, including selector
	// resolution. Resolving against the raw tree would turn a withheld value
	// into an oracle: a client could ask for value="hunter2" and learn the
	// password from whether anything matched.
	tree := s.policy.redactTree(state.Tree)

	switch req.Op {
	case OpHello:
		return Response{
			OK: true, Version: Version, Width: state.Width, Height: state.Height,
			AllowInput:        s.policy.AllowInput,
			ExposeScreen:      s.policy.ExposeScreen,
			ExposeInputValues: s.policy.ExposeInputValues,
		}

	case OpTree:
		return Response{OK: true, Nodes: tree}

	case OpSnapshot:
		if !s.policy.ExposeScreen {
			return Response{Err: "automation: screen snapshots are disabled by policy (Policy.ExposeScreen)"}
		}
		return Response{OK: true, Snapshot: state.Screen}

	case OpFocused:
		return Response{OK: true, Focused: state.Focused}

	case OpFind:
		if req.Selector.IsEmpty() {
			return Response{OK: true, Nodes: tree}
		}
		return Response{OK: true, Nodes: Resolve(tree, req.Selector)}

	case OpClick:
		if err := s.checkInput(state); err != nil {
			return Response{Err: err.Error()}
		}
		node, err := ResolveOne(tree, req.Selector)
		if err != nil {
			return Response{Err: err.Error()}
		}
		if node.Bounds.Width == 0 || node.Bounds.Height == 0 {
			return Response{Err: fmt.Sprintf("automation: %s has empty bounds and cannot be clicked", req.Selector)}
		}
		// Centre of the node, which is inside it for any non-empty rect.
		state.Injector(driver.Event{
			Type: driver.EventMouse,
			Mouse: driver.MouseEvent{
				Button: driver.MouseLeft,
				X:      node.Bounds.X + node.Bounds.Width/2,
				Y:      node.Bounds.Y + node.Bounds.Height/2,
			},
		})
		return Response{OK: true, Nodes: []accessibility.AccessibilityNode{node}}

	case OpKey:
		if err := s.checkInput(state); err != nil {
			return Response{Err: err.Error()}
		}
		key, err := parseKey(req.Key)
		if err != nil {
			return Response{Err: err.Error()}
		}
		key.Ctrl, key.Alt, key.Shift = req.Ctrl, req.Alt, req.Shift
		state.Injector(driver.Event{Type: driver.EventKey, Key: key})
		return Response{OK: true}

	case OpText:
		if err := s.checkInput(state); err != nil {
			return Response{Err: err.Error()}
		}
		for _, r := range req.Text {
			state.Injector(driver.Event{
				Type: driver.EventKey,
				Key:  driver.KeyEvent{Type: driver.KeyRune, Ch: r},
			})
		}
		return Response{OK: true}

	default:
		return Response{Err: fmt.Sprintf("automation: unknown op %q", req.Op)}
	}
}

// verifyPeer admits a connection only from the user running the application.
//
// File permissions on the socket already say this, but they are one
// misconfigured umask or directory away from failing open. The kernel's record
// of who owns the connecting process is not, so the server checks both.
func (s *Server) verifyPeer(conn net.Conn) error {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("automation: refusing a non-Unix connection")
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return fmt.Errorf("automation: cannot inspect connection: %w", err)
	}
	uid, peerErr := -1, error(nil)
	if ctrlErr := raw.Control(func(fd uintptr) { uid, peerErr = peerUID(fd) }); ctrlErr != nil {
		peerErr = ctrlErr
	}
	return admitPeer(uid, peerErr, os.Getuid(), s.policy.AllowUnverifiedPeers)
}

// admitPeer is the admission decision on its own, so it can be tested without
// a second user account. Fails closed: an unknown peer is refused unless the
// policy says otherwise, and a known peer must be this process's user.
func admitPeer(peer int, peerErr error, self int, allowUnverified bool) error {
	if peerErr != nil {
		if allowUnverified {
			return nil
		}
		return fmt.Errorf("automation: connection refused, the connecting user cannot be verified on this platform (Policy.AllowUnverifiedPeers): %w", peerErr)
	}
	if peer != self {
		return fmt.Errorf("automation: connection refused, peer runs as uid %d, not %d", peer, self)
	}
	return nil
}

// checkInput refuses input synthesis unless the policy allows it and the
// application wired an injector.
func (s *Server) checkInput(state Snapshot) error {
	if !s.policy.AllowInput {
		return fmt.Errorf("automation: input synthesis is disabled by policy (Policy.AllowInput)")
	}
	if state.Injector == nil {
		return fmt.Errorf("automation: application accepts no synthetic input")
	}
	return nil
}

// namedKeys maps protocol key names to driver key types. Single characters are
// handled separately as runes.
var namedKeys = map[string]driver.KeyType{
	"enter":     driver.KeyEnter,
	"tab":       driver.KeyTab,
	"esc":       driver.KeyEsc,
	"escape":    driver.KeyEsc,
	"space":     driver.KeySpace,
	"backspace": driver.KeyBackspace,
	"up":        driver.KeyArrowUp,
	"down":      driver.KeyArrowDown,
	"left":      driver.KeyArrowLeft,
	"right":     driver.KeyArrowRight,
	"home":      driver.KeyHome,
	"end":       driver.KeyEnd,
	"pgup":      driver.KeyPageUp,
	"pgdn":      driver.KeyPageDown,
	"insert":    driver.KeyInsert,
	"delete":    driver.KeyDelete,
	"f1":        driver.KeyF1,
	"f2":        driver.KeyF2,
	"f3":        driver.KeyF3,
	"f4":        driver.KeyF4,
	"f5":        driver.KeyF5,
	"f6":        driver.KeyF6,
	"f7":        driver.KeyF7,
	"f8":        driver.KeyF8,
	"f9":        driver.KeyF9,
	"f10":       driver.KeyF10,
	"f11":       driver.KeyF11,
	"f12":       driver.KeyF12,
}

// ParseKey turns a protocol key name — a single character, or "enter", "tab",
// "esc", "up", "f1" and the rest of the names Request.Key lists — into the key
// event it stands for, without modifiers.
func ParseKey(name string) (driver.KeyEvent, error) { return parseKey(name) }

func parseKey(name string) (driver.KeyEvent, error) {
	if name == "" {
		return driver.KeyEvent{}, fmt.Errorf("automation: empty key")
	}
	if keyType, ok := namedKeys[strings.ToLower(name)]; ok {
		return driver.KeyEvent{Type: keyType}, nil
	}
	if r, size := utf8.DecodeRuneInString(name); size == len(name) && r != utf8.RuneError {
		return driver.KeyEvent{Type: driver.KeyRune, Ch: r}, nil
	}
	return driver.KeyEvent{}, fmt.Errorf("automation: unknown key %q", name)
}
