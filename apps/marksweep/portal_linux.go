//go:build linux

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
)

// No terminal can lock the mouse pointer, so the game asks the desktop:
// the InputCapture portal (KDE Plasma 6.1+, GNOME 45+) is how Input Leap
// and Barrier take the mouse, and it needs no root and no setup — the
// desktop asks once whether to allow it, and the answer is remembered.
// Capture starts when the pointer crosses a barrier the game lays along the
// outer edges of the screens (the portal offers no other way to start it),
// so a flick to any edge takes the mouse; from then on the pointer is the
// compositor's to hide and keep, and its movement reaches the game alone,
// through libei (ei_linux.go). The pause and the terminal losing focus
// give it back.

const (
	portalDest = "org.freedesktop.portal.Desktop"
	portalPath = "/org/freedesktop/portal/desktop"
	portalIC   = "org.freedesktop.portal.InputCapture"
	capPointer = 2
)

type capture struct {
	bus     *dbus.Conn
	session dbus.ObjectPath
	ei      *eiClient
	signals chan *dbus.Signal

	mu         sync.Mutex
	enabled    bool
	activation uint32
	active     atomic.Bool
	tokens     int
}

// startCapture sets a capture session up, asking the desktop's permission
// if it was never given. It blocks while the desktop asks.
func startCapture() (*capture, error) {
	bus, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, err
	}
	if err := bus.Auth(nil); err != nil {
		bus.Close()
		return nil, err
	}
	if err := bus.Hello(); err != nil {
		bus.Close()
		return nil, err
	}
	c := &capture{bus: bus, signals: make(chan *dbus.Signal, 32)}
	bus.Signal(c.signals)
	if err := c.setUp(); err != nil {
		bus.Close()
		return nil, err
	}
	go c.watch()
	return c, nil
}

func (c *capture) portal() dbus.BusObject { return c.bus.Object(portalDest, portalPath) }

// token is a fresh handle token for a request.
func (c *capture) token() string {
	c.tokens++
	return fmt.Sprintf("marksweep%d_%d", os.Getpid(), c.tokens)
}

// request calls a portal method that answers through a Request object, and
// waits for the answer.
func (c *capture) request(method string, build func(token string) []any) (map[string]dbus.Variant, error) {
	token := c.token()
	sender := strings.ReplaceAll(strings.TrimPrefix(c.bus.Names()[0], ":"), ".", "_")
	path := dbus.ObjectPath(portalPath + "/request/" + sender + "/" + token)
	resp := make(chan *dbus.Signal, 1)
	c.bus.Signal(resp)
	defer c.bus.RemoveSignal(resp)
	if err := c.bus.AddMatchSignal(dbus.WithMatchObjectPath(path), dbus.WithMatchInterface("org.freedesktop.portal.Request")); err != nil {
		return nil, err
	}
	if err := c.portal().Call(portalIC+"."+method, 0, build(token)...).Err; err != nil {
		return nil, err
	}
	timeout := time.After(waitTimeout)
	for {
		select {
		case s := <-resp:
			if s.Path != path || s.Name != "org.freedesktop.portal.Request.Response" || len(s.Body) < 2 {
				continue
			}
			code, _ := s.Body[0].(uint32)
			results, _ := s.Body[1].(map[string]dbus.Variant)
			if code != 0 {
				return nil, fmt.Errorf("%s: refused (%d)", method, code)
			}
			return results, nil
		case <-timeout:
			return nil, errors.New(method + ": no answer")
		}
	}
}

func (c *capture) setUp() error {
	var created map[string]dbus.Variant
	err := c.portal().Call(portalIC+".CreateSession2", 0, map[string]dbus.Variant{
		"session_handle_token": dbus.MakeVariant(c.token()),
	}).Store(&created)
	if err != nil {
		return err
	}
	switch s := created["session_handle"].Value().(type) {
	case dbus.ObjectPath:
		c.session = s
	case string:
		c.session = dbus.ObjectPath(s)
	default:
		return errors.New("CreateSession2: no session")
	}
	opts := map[string]dbus.Variant{"capabilities": dbus.MakeVariant(uint32(capPointer)), "persist_mode": dbus.MakeVariant(uint32(2))}
	if t := loadToken(); t != "" {
		opts["restore_token"] = dbus.MakeVariant(t)
	}
	started, err := c.request("Start", func(token string) []any {
		opts["handle_token"] = dbus.MakeVariant(token)
		return []any{c.session, "", opts}
	})
	if err != nil {
		return err
	}
	if t, ok := started["restore_token"].Value().(string); ok {
		saveToken(t)
	}
	if err := c.barriers(); err != nil {
		return err
	}
	if err := c.bus.AddMatchSignal(dbus.WithMatchInterface(portalIC)); err != nil {
		return err
	}
	var fd dbus.UnixFD
	if err := c.portal().Call(portalIC+".ConnectToEIS", 0, c.session, map[string]dbus.Variant{}).Store(&fd); err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), "eis")
	conn, err := net.FileConn(f)
	f.Close()
	if err != nil {
		return err
	}
	c.ei = newEIClient(conn)
	go func() {
		c.ei.run()
		conn.Close()
	}()
	return nil
}

// edge is a barrier's position, sent as (iiii).
type edge struct{ X1, Y1, X2, Y2 int32 }

// barriers lays a barrier along every edge of every screen; the portal
// keeps those on the outside of the desktop and refuses the rest.
func (c *capture) barriers() error {
	zones, err := c.request("GetZones", func(token string) []any {
		return []any{c.session, map[string]dbus.Variant{"handle_token": dbus.MakeVariant(token)}}
	})
	if err != nil {
		return err
	}
	set, _ := zones["zone_set"].Value().(uint32)
	var list []struct {
		W, H uint32
		X, Y int32
	}
	if err := zones["zones"].Store(&list); err != nil {
		return err
	}
	var barriers []map[string]dbus.Variant
	for _, z := range list {
		r, b := z.X+int32(z.W), z.Y+int32(z.H)
		for _, e := range []edge{{z.X, z.Y, r - 1, z.Y}, {z.X, b, r - 1, b}, {z.X, z.Y, z.X, b - 1}, {r, z.Y, r, b - 1}} {
			barriers = append(barriers, map[string]dbus.Variant{
				"barrier_id": dbus.MakeVariant(uint32(len(barriers) + 1)),
				"position":   dbus.MakeVariant(e),
			})
		}
	}
	_, err = c.request("SetPointerBarriers", func(token string) []any {
		return []any{c.session, map[string]dbus.Variant{"handle_token": dbus.MakeVariant(token)}, barriers, set}
	})
	return err
}

// watch follows the portal's signals: capture starting and ending, and the
// screens changing under it.
func (c *capture) watch() {
	for s := range c.signals {
		if !strings.HasPrefix(s.Name, portalIC+".") || len(s.Body) < 2 {
			continue
		}
		if p, _ := s.Body[0].(dbus.ObjectPath); p != c.session {
			continue
		}
		opts, _ := s.Body[1].(map[string]dbus.Variant)
		switch strings.TrimPrefix(s.Name, portalIC+".") {
		case "Activated":
			id, _ := opts["activation_id"].Value().(uint32)
			c.mu.Lock()
			c.activation = id
			c.mu.Unlock()
			c.active.Store(true)
		case "Deactivated":
			c.active.Store(false)
		case "Disabled":
			c.active.Store(false)
			c.mu.Lock()
			c.enabled = false
			c.mu.Unlock()
		case "ZonesChanged":
			c.mu.Lock()
			on := c.enabled
			c.mu.Unlock()
			if c.barriers() == nil && on {
				c.enable(true)
			}
		}
	}
}

// enable arms the barriers, or disarms them and gives the mouse back.
func (c *capture) enable(on bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if on {
		c.portal().Call(portalIC+".Enable", 0, c.session, map[string]dbus.Variant{})
		c.enabled = true
		return
	}
	if c.active.Load() {
		c.portal().Call(portalIC+".Release", 0, c.session, map[string]dbus.Variant{"activation_id": dbus.MakeVariant(c.activation)})
		c.active.Store(false)
	}
	c.portal().Call(portalIC+".Disable", 0, c.session, map[string]dbus.Variant{})
	c.enabled = false
}

func (c *capture) close() {
	c.enable(false)
	c.bus.Object(portalDest, c.session).Call("org.freedesktop.portal.Session.Close", 0)
	c.bus.Close()
}

// The desktop's permission, remembered between runs.
func tokenPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "marksweep", "input-capture")
}

func loadToken() string {
	b, _ := os.ReadFile(tokenPath())
	return strings.TrimSpace(string(b))
}

func saveToken(t string) {
	if p := tokenPath(); p != "" {
		os.MkdirAll(filepath.Dir(p), 0o700)
		os.WriteFile(p, []byte(t), 0o600)
	}
}

// waitTimeout is how long the desktop's question may stay open.
const waitTimeout = 3 * time.Minute
