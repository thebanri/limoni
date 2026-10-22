//go:build linux

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
)

// A terminal cannot lock the mouse, so on Wayland the game asks the
// compositor the way every game does: it opens a window of its own —
// full screen, every pixel transparent, so the terminal shows through —
// and locks the pointer to it (zwp_pointer_constraints_v1). The pointer's
// movement then comes as relative motion (zwp_relative_pointer_v1), and
// the keyboard, which the window holds while it is up, as the compositor's
// own presses and releases: a key let go is always known to be let go.
// No permission, no setup; the window goes when the run is paused.
//
// The wire is Wayland's: a header of object id (u32) and size<<16|opcode
// (u32), arguments 4-byte aligned in the host's byte order, strings as a
// length counting the NUL then the padded bytes, file descriptors beside
// the bytes (SCM_RIGHTS). Opcodes are from wayland.xml, xdg-shell.xml,
// pointer-constraints-unstable-v1.xml and relative-pointer-unstable-v1.xml.

type wlEvent struct {
	kind wlKind
	code uint32 // an evdev key or button code
	down bool
}

type wlKind uint8

const (
	wlKey wlKind = iota
	wlButton
	wlEnter // the window has the keyboard: presses of the keys held follow
	wlLeave // the window lost the keyboard, or was asked to close
)

type wlClient struct {
	conn *net.UnixConn

	mu      sync.Mutex // everything below, and the writing
	out     []byte
	starts  []int // where each buffered request begins
	fds     []int // received, waiting for the message they came with
	next    uint32
	types   map[uint32]string
	globals map[string][2]uint32 // interface → name, version

	registry, compositor, shm, wm, seat, constraints, relative uint32
	pointer, keyboard, relPointer                              uint32
	surface, xdgSurface, toplevel, lock, pool, buffer          uint32
	w, h, bufW, bufH                                           int32
	file                                                       *os.File

	dx, dy atomic.Int64 // motion since the last drain, in 1/256 of a pixel
	locked atomic.Bool
	events chan wlEvent
}

var errNoWayland = errors.New("not a Wayland session")

// dialWayland connects to the compositor and binds what the lock needs.
func dialWayland() (*wlClient, error) {
	name := os.Getenv("WAYLAND_DISPLAY")
	if name == "" {
		return nil, errNoWayland
	}
	if !filepath.IsAbs(name) {
		name = filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), name)
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: name, Net: "unix"})
	if err != nil {
		return nil, err
	}
	c := newWLClient(conn)
	if err := c.setUp(); err != nil {
		conn.Close()
		return nil, err
	}
	go c.run(nil)
	return c, nil
}

func newWLClient(conn *net.UnixConn) *wlClient {
	return &wlClient{conn: conn, next: 2, types: map[uint32]string{1: "wl_display"},
		globals: map[string][2]uint32{}, events: make(chan wlEvent, 256)}
}

// wlNeeds is what the lock is made of.
var wlNeeds = []string{"wl_compositor", "wl_shm", "xdg_wm_base", "wl_seat", "zwp_pointer_constraints_v1", "zwp_relative_pointer_manager_v1"}

// setUp reads the globals and binds them; the seat hands over its pointer
// and keyboard once the events are being read.
func (c *wlClient) setUp() error {
	c.mu.Lock()
	c.registry = c.newID("wl_registry")
	c.begin(1, 1) // wl_display.get_registry
	c.u32(c.registry)
	done := c.newID("wl_callback")
	c.begin(1, 0) // wl_display.sync: answered once every global is sent
	c.u32(done)
	err := c.send()
	c.mu.Unlock()
	if err != nil {
		return err
	}
	if err := c.run(func() bool { return c.types[done] != "" }); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, need := range wlNeeds {
		if _, ok := c.globals[need]; !ok {
			return errors.New("the compositor has no " + need)
		}
	}
	c.compositor = c.bind("wl_compositor", 4)
	c.shm = c.bind("wl_shm", 1)
	c.wm = c.bind("xdg_wm_base", 1)
	c.seat = c.bind("wl_seat", 5)
	c.constraints = c.bind("zwp_pointer_constraints_v1", 1)
	c.relative = c.bind("zwp_relative_pointer_manager_v1", 1)
	return c.send()
}

// bind binds a global at the lower of the version given and the
// compositor's.
func (c *wlClient) bind(iface string, version uint32) uint32 {
	g := c.globals[iface]
	id := c.newID(iface)
	c.begin(c.registry, 0) // wl_registry.bind
	c.u32(g[0])
	c.str(iface)
	c.u32(min(version, g[1]))
	c.u32(id)
	return id
}

func (c *wlClient) newID(iface string) uint32 {
	id := c.next
	c.next++
	c.types[id] = iface
	return id
}

// ── writing; the caller holds mu ─────────────────────────────────────────

func (c *wlClient) begin(object, opcode uint32) {
	c.starts = append(c.starts, len(c.out))
	c.out = binary.NativeEndian.AppendUint32(c.out, object)
	c.out = binary.NativeEndian.AppendUint32(c.out, opcode) // the size joins it in send
}

func (c *wlClient) u32(v uint32) { c.out = binary.NativeEndian.AppendUint32(c.out, v) }
func (c *wlClient) i32(v int32)  { c.u32(uint32(v)) }

func (c *wlClient) str(s string) {
	c.u32(uint32(len(s) + 1))
	c.out = append(c.out, s...)
	c.out = append(c.out, 0)
	for len(c.out)%4 != 0 {
		c.out = append(c.out, 0)
	}
}

// send writes the buffered requests, with file descriptors if some go
// with them.
func (c *wlClient) send(fds ...int) error {
	for i, at := range c.starts {
		end := len(c.out)
		if i+1 < len(c.starts) {
			end = c.starts[i+1]
		}
		word := binary.NativeEndian.Uint32(c.out[at+4:])
		binary.NativeEndian.PutUint32(c.out[at+4:], uint32(end-at)<<16|word)
	}
	var oob []byte
	if len(fds) > 0 {
		oob = syscall.UnixRights(fds...)
	}
	_, _, err := c.conn.WriteMsgUnix(c.out, oob, nil)
	c.out, c.starts = c.out[:0], c.starts[:0]
	return err
}

// ── the window ───────────────────────────────────────────────────────────

// show opens the window and asks for the lock; it holds once the
// compositor gives the window the keyboard and the pointer.
func (c *wlClient) show() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.surface != 0 {
		return
	}
	c.surface = c.newID("wl_surface")
	c.begin(c.compositor, 0) // create_surface
	c.u32(c.surface)
	c.xdgSurface = c.newID("xdg_surface")
	c.begin(c.wm, 2) // get_xdg_surface
	c.u32(c.xdgSurface)
	c.u32(c.surface)
	c.toplevel = c.newID("xdg_toplevel")
	c.begin(c.xdgSurface, 1) // get_toplevel
	c.u32(c.toplevel)
	c.begin(c.toplevel, 2) // set_title
	c.str("Mark & Sweep")
	c.begin(c.toplevel, 3) // set_app_id
	c.str("marksweep")
	c.begin(c.toplevel, 11) // set_fullscreen, on the output the compositor picks
	c.u32(0)
	if c.pointer != 0 {
		c.lockPointer()
	}
	c.begin(c.surface, 6) // commit, with no buffer: the compositor answers with a size
	c.send()
}

func (c *wlClient) lockPointer() {
	c.lock = c.newID("zwp_locked_pointer_v1")
	c.begin(c.constraints, 1) // lock_pointer
	c.u32(c.lock)
	c.u32(c.surface)
	c.u32(c.pointer)
	c.u32(0) // the whole surface
	c.u32(2) // persistent: locked again whenever the window has the pointer back
}

// hide closes the window, and with it the lock and the keyboard.
func (c *wlClient) hide() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.surface == 0 {
		return
	}
	for _, o := range []uint32{c.lock, c.toplevel, c.xdgSurface, c.surface, c.buffer} {
		c.destroy(o, 0)
	}
	c.destroy(c.pool, 1)
	c.send()
	c.lock, c.toplevel, c.xdgSurface, c.surface, c.buffer, c.pool = 0, 0, 0, 0, 0, 0
	c.bufW, c.bufH = 0, 0
	if c.file != nil {
		c.file.Close()
		c.file = nil
	}
	c.locked.Store(false)
}

func (c *wlClient) destroy(object, opcode uint32) {
	if object != 0 {
		c.begin(object, opcode)
		delete(c.types, object)
	}
}

// fill gives the surface a transparent buffer the size asked for.
func (c *wlClient) fill() error {
	w, h := max(c.w, 1), max(c.h, 1)
	if w != c.bufW || h != c.bufH {
		if c.buffer != 0 {
			c.destroy(c.buffer, 0)
			c.destroy(c.pool, 1)
			c.file.Close()
		}
		// An unlinked file of zeros: transparent black, and pages never
		// written cost nothing.
		f, err := os.CreateTemp(os.Getenv("XDG_RUNTIME_DIR"), "marksweep-*")
		if err != nil {
			return err
		}
		os.Remove(f.Name())
		size := int64(w) * int64(h) * 4
		if err := f.Truncate(size); err != nil {
			f.Close()
			return err
		}
		c.file, c.bufW, c.bufH = f, w, h
		c.pool = c.newID("wl_shm_pool")
		c.begin(c.shm, 0) // create_pool
		c.u32(c.pool)
		c.i32(int32(size))
		if err := c.send(int(f.Fd())); err != nil {
			return err
		}
		c.buffer = c.newID("wl_buffer")
		c.begin(c.pool, 0) // create_buffer
		c.u32(c.buffer)
		c.i32(0)
		c.i32(w)
		c.i32(h)
		c.i32(w * 4)
		c.u32(0) // argb8888
	}
	c.begin(c.surface, 1) // attach
	c.u32(c.buffer)
	c.i32(0)
	c.i32(0)
	c.begin(c.surface, 2) // damage
	c.i32(0)
	c.i32(0)
	c.i32(w)
	c.i32(h)
	c.begin(c.surface, 6) // commit
	return nil
}

// ── reading ──────────────────────────────────────────────────────────────

// run reads and handles events until the connection ends, or until more
// says there is no need for more.
func (c *wlClient) run(more func() bool) error {
	buf := make([]byte, 1<<16)
	oob := make([]byte, syscall.CmsgSpace(28*4))
	have := 0
	for more == nil || more() {
		n, oobn, _, _, err := c.conn.ReadMsgUnix(buf[have:], oob)
		if oobn > 0 {
			if msgs, err := syscall.ParseSocketControlMessage(oob[:oobn]); err == nil {
				for _, m := range msgs {
					if fds, err := syscall.ParseUnixRights(&m); err == nil {
						c.mu.Lock()
						c.fds = append(c.fds, fds...)
						c.mu.Unlock()
					}
				}
			}
		}
		if n == 0 && err == nil {
			err = io.EOF
		}
		if err != nil {
			c.emit(wlEvent{kind: wlLeave})
			return err
		}
		have += n
		at := 0
		for have-at >= 8 {
			object := binary.NativeEndian.Uint32(buf[at:])
			word := binary.NativeEndian.Uint32(buf[at+4:])
			size := int(word >> 16)
			if size < 8 || size%4 != 0 {
				c.emit(wlEvent{kind: wlLeave})
				return errors.New("wayland: a message of impossible size")
			}
			if have-at < size {
				break
			}
			c.mu.Lock()
			err := c.handle(object, word&0xffff, &args{b: buf[at+8 : at+size]})
			c.mu.Unlock()
			if err != nil {
				c.emit(wlEvent{kind: wlLeave})
				return err
			}
			at += size
		}
		have = copy(buf, buf[at:have])
	}
	return nil
}

func (c *wlClient) emit(e wlEvent) {
	select {
	case c.events <- e:
	default:
	}
}

// handle acts on one event; the caller holds mu.
func (c *wlClient) handle(object, opcode uint32, a *args) error {
	switch c.types[object] {
	case "wl_display":
		switch opcode {
		case 0: // error
			obj, code, msg := a.u32(), a.u32(), a.str()
			return fmt.Errorf("wayland: %s %d: error %d: %s", c.types[obj], obj, code, msg)
		case 1: // delete_id
			delete(c.types, a.u32())
		}
	case "wl_registry":
		if opcode == 0 { // global
			name, iface, version := a.u32(), a.str(), a.u32()
			c.globals[iface] = [2]uint32{name, version}
		}
	case "wl_callback":
		delete(c.types, object) // done: callbacks are one shot
	case "xdg_wm_base":
		if opcode == 0 { // ping
			c.begin(c.wm, 3) // pong
			c.u32(a.u32())
			return c.send()
		}
	case "wl_seat":
		if opcode == 0 { // capabilities
			caps := a.u32()
			if caps&1 != 0 && c.pointer == 0 {
				c.pointer = c.newID("wl_pointer")
				c.begin(c.seat, 0) // get_pointer
				c.u32(c.pointer)
				c.relPointer = c.newID("zwp_relative_pointer_v1")
				c.begin(c.relative, 1) // get_relative_pointer
				c.u32(c.relPointer)
				c.u32(c.pointer)
				if c.surface != 0 && c.lock == 0 {
					c.lockPointer()
				}
			}
			if caps&2 != 0 && c.keyboard == 0 {
				c.keyboard = c.newID("wl_keyboard")
				c.begin(c.seat, 1) // get_keyboard
				c.u32(c.keyboard)
			}
			return c.send()
		}
	case "xdg_toplevel":
		switch opcode {
		case 0: // configure
			c.w, c.h = int32(a.u32()), int32(a.u32())
		case 1: // close
			c.emit(wlEvent{kind: wlLeave})
		}
	case "xdg_surface":
		if opcode == 0 { // configure
			c.begin(c.xdgSurface, 4) // ack_configure
			c.u32(a.u32())
			if err := c.fill(); err != nil {
				return err
			}
			return c.send()
		}
	case "zwp_locked_pointer_v1":
		c.locked.Store(opcode == 0) // locked, unlocked
	case "zwp_relative_pointer_v1":
		if opcode == 0 { // relative_motion: time, dx, dy, then the same unaccelerated, which games use
			a.u32()
			a.u32()
			a.u32()
			a.u32()
			dx, dy := int32(a.u32()), int32(a.u32())
			c.dx.Add(int64(dx))
			c.dy.Add(int64(dy))
		}
	case "wl_pointer":
		switch opcode {
		case 0: // enter: no pointer over the window
			c.begin(c.pointer, 0) // set_cursor
			c.u32(a.u32())
			c.u32(0)
			c.i32(0)
			c.i32(0)
			return c.send()
		case 3: // button
			a.u32()
			a.u32()
			button, state := a.u32(), a.u32()
			c.emit(wlEvent{kind: wlButton, code: button, down: state == 1})
		}
	case "wl_keyboard":
		switch opcode {
		case 0: // keymap: the game reads key codes, not symbols
			if len(c.fds) > 0 {
				syscall.Close(c.fds[0])
				c.fds = c.fds[1:]
			}
		case 1: // enter, with the keys already held
			a.u32()
			if a.u32() != c.surface {
				break // a window of before
			}
			c.emit(wlEvent{kind: wlEnter})
			for keys := a.array(); len(keys) >= 4; keys = keys[4:] {
				c.emit(wlEvent{kind: wlKey, code: binary.NativeEndian.Uint32(keys), down: true})
			}
		case 2: // leave
			a.u32()
			if a.u32() == c.surface {
				c.emit(wlEvent{kind: wlLeave})
			}
		case 3: // key
			a.u32()
			a.u32()
			key, state := a.u32(), a.u32()
			c.emit(wlEvent{kind: wlKey, code: key, down: state == 1})
		}
	}
	if a.bad {
		return errors.New("wayland: a message shorter than its arguments")
	}
	return nil
}

// motion takes the motion since the last call, in pixels.
func (c *wlClient) motion() (float64, float64) {
	return float64(c.dx.Swap(0)) / 256, float64(c.dy.Swap(0)) / 256
}

// args reads a message's arguments in order.
type args struct {
	b   []byte
	bad bool
}

func (a *args) u32() uint32 {
	if len(a.b) < 4 {
		a.bad = true
		return 0
	}
	v := binary.NativeEndian.Uint32(a.b)
	a.b = a.b[4:]
	return v
}

// array reads an array or a string's bytes: a length, then the bytes
// padded to 4.
func (a *args) array() []byte {
	n := int(a.u32())
	padded := (n + 3) &^ 3
	if len(a.b) < padded || padded < n {
		a.bad = true
		return nil
	}
	v := a.b[:n]
	a.b = a.b[padded:]
	return v
}

func (a *args) str() string {
	b := a.array()
	if len(b) == 0 {
		return ""
	}
	return string(b[:len(b)-1])
}
