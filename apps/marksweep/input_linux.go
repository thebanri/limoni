//go:build linux

package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// A terminal cannot lock the pointer, so on Linux the game locks it itself,
// below the terminal and the compositor: it reads the mice from their evdev
// devices and, while a run is on, grabs them (EVIOCGRAB), so their movement
// reaches the game alone. The pointer stays where it is — it cannot wander
// onto another monitor or click another window — and the game gets the
// mouse's raw movement, as a browser's pointer lock gives. Cards, menus, the
// pause and the terminal losing focus let it go.
//
// It needs read access to the mice's /dev/input/event* devices; without it
// the game falls back to the terminal's mouse reports (game.mouse). One udev
// rule gives the person at the seat the mice and nothing else:
//
//	SUBSYSTEM=="input", KERNEL=="event*", ENV{ID_INPUT_MOUSE}=="1", TAG+="uaccess"
type pointer struct {
	mice    []*os.File
	dx      atomic.Int64
	buttons chan pointerEvent
	grabbed atomic.Bool
	why     string // why there is no lock, for the title screen
}

type pointerEvent struct {
	button int
	down   bool
}

// rawLook is the turn a count of raw mouse movement gives, as a share of
// the browser's per-pixel sensitivity: a mouse counts finer than pixels
// move, and with no acceleration.
const rawLook = 0.3

const (
	evKey     = 1
	evRel     = 2
	relX      = 0
	btnLeft   = 0x110
	btnRight  = 0x111
	evioCGrab = 0x40044590 // _IOW('E', 0x90, int)
)

// eventSize is a struct input_event: a timeval, then type, code and value.
var eventSize = int(unsafe.Sizeof(syscall.Timeval{})) + 8

func newPointer() *pointer {
	p := &pointer{buttons: make(chan pointerEvent, 64)}
	paths := micePaths()
	if len(paths) == 0 {
		p.why = "no mouse found in /proc/bus/input/devices"
		return p
	}
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			p.why = "no access to " + path + ": the mouse cannot be locked (README: Mouse lock)"
			continue
		}
		p.mice = append(p.mice, f)
		go p.read(f)
	}
	if len(p.mice) > 0 {
		p.why = ""
	}
	return p
}

// micePaths finds the event devices of everything the kernel counts as a
// mouse (a mouseN handler), touchpads included.
func micePaths() []string {
	f, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return nil
	}
	defer f.Close()
	return micePathsIn(f)
}

func micePathsIn(r io.Reader) []string {
	var paths []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "H: Handlers=") {
			continue
		}
		var event string
		mouse := false
		for _, h := range strings.Fields(strings.TrimPrefix(line, "H: Handlers=")) {
			switch {
			case strings.HasPrefix(h, "mouse"):
				mouse = true
			case strings.HasPrefix(h, "event"):
				event = h
			}
		}
		if mouse && event != "" {
			paths = append(paths, "/dev/input/"+event)
		}
	}
	return paths
}

// read turns a device's events into movement and buttons, for as long as
// the device is open. Nothing is kept while the mouse is not grabbed: then
// the terminal reports it.
func (p *pointer) read(f *os.File) {
	buf := make([]byte, eventSize*64)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return
		}
		p.parse(buf[:n])
	}
}

func (p *pointer) parse(b []byte) {
	for ; len(b) >= eventSize; b = b[eventSize:] {
		ev := b[eventSize-8:]
		typ, code := binary.NativeEndian.Uint16(ev), binary.NativeEndian.Uint16(ev[2:])
		value := int32(binary.NativeEndian.Uint32(ev[4:]))
		if !p.grabbed.Load() {
			continue
		}
		switch {
		case typ == evRel && code == relX:
			p.dx.Add(int64(value))
		case typ == evKey && (code == btnLeft || code == btnRight) && value != 2:
			button := 0
			if code == btnRight {
				button = 2
			}
			select {
			case p.buttons <- pointerEvent{button, value == 1}:
			default:
			}
		}
	}
}

func (p *pointer) drain(g *game) {
	g.locked = p.grabbed.Load()
	g.lookDX += float64(p.dx.Swap(0)) * rawLook
	for {
		select {
		case b := <-p.buttons:
			g.pointerButton(b.button, b.down)
		default:
			return
		}
	}
}

// want grabs the mice during a run and lets them go otherwise.
func (p *pointer) want(w bool) {
	if w == p.grabbed.Load() || len(p.mice) == 0 {
		return
	}
	grabbed := false
	for _, f := range p.mice {
		if grab(f, w) && w {
			grabbed = true
		}
	}
	p.grabbed.Store(w && grabbed)
	if !w {
		p.dx.Store(0)
	}
}

func grab(f *os.File, on bool) bool {
	c, err := f.SyscallConn()
	if err != nil {
		return false
	}
	var arg uintptr
	if on {
		arg = 1
	}
	ok := false
	c.Control(func(fd uintptr) {
		_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, evioCGrab, arg)
		ok = e == 0
	})
	return ok
}

// close lets the mice go; the kernel would on exit anyway.
func (p *pointer) close() {
	p.want(false)
	for _, f := range p.mice {
		f.Close()
	}
}

// problem is why the mouse cannot be locked, or nothing.
func (p *pointer) problem() string { return p.why }
