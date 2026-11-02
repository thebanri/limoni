package overlay

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// xconn is the few X11 requests the overlay needs, spoken over the local
// socket. Everything that needs an answer (finding the terminal's window and
// its size) happens in Open, before a goroutine starts reading; from then on
// the server only sends errors, which are dropped, and the terminal window's
// ConfigureNotify events, which keep its size.
type xconn struct {
	c          net.Conn
	buf        []byte
	base, mask uint32
	next       uint32
	root       uint32
	maxReq     int // bytes
	atoms      map[string]uint32
	// deep holds the depths the server stores 32 bits per pixel for.
	deep map[byte]bool
	// term is the terminal's window, the pictures' parent, of depth depth.
	term  uint32
	depth byte
	// alpha is how opaque the terminal draws its background: below 255 in
	// a translucent terminal's 32-bit window.
	alpha byte
	gc    uint32
	// size is term's width<<16 | height, written by read; 0 once it is gone.
	size atomic.Uint32
}

const (
	opCreateWindow           = 1
	opChangeWindowAttributes = 2
	opDestroyWindow          = 4
	opMapWindow              = 8
	opConfigureWindow        = 12
	opGetGeometry            = 14
	opInternAtom             = 16
	opGetProperty            = 20
	opCreatePixmap           = 53
	opFreePixmap             = 54
	opCreateGC               = 55
	opClearArea              = 61
	opPutImage               = 72
	opGetImage               = 73
	cwBackPixmap             = 1
	cwEventMask              = 0x800
	structureNotifyMask      = 1 << 17
	eventConfigureNotify     = 22
	eventDestroyNotify       = 17
	windowClassInputOutput   = 1
	imageFormatZPixmap       = 2
	configureX, configureY   = 1, 2
	anyPropertyType          = 0
	maxPropertyWords         = 1 << 16
)

var le = binary.LittleEndian

// dialX connects to the X server named by display, which must be local
// (":0", ":1.0"): a picture shown on some other machine's screen is no use.
func dialX(display string) (*xconn, error) {
	num, err := displayNumber(display)
	if err != nil {
		return nil, err
	}
	path := "/tmp/.X11-unix/X" + num
	c, err := net.Dial("unix", path)
	if err != nil {
		if c, err = net.Dial("unix", "@"+path); err != nil {
			return nil, err
		}
	}
	name, data := xauthCookie(num)
	x := &xconn{c: c}
	if err := x.setup(name, data); err != nil {
		c.Close()
		return nil, err
	}
	if err := x.intern("_NET_WM_PID", "_NET_CLIENT_LIST", "_NET_ACTIVE_WINDOW"); err != nil {
		c.Close()
		return nil, err
	}
	return x, nil
}

// roundTrip sends a request and returns its reply, skipping any event that
// comes first. Only for use before attach starts the reader.
func (x *xconn) roundTrip(op, detail byte, body []byte) ([]byte, error) {
	x.send(op, detail, body)
	if err := x.flush(); err != nil {
		return nil, err
	}
	head := make([]byte, 32)
	for {
		if _, err := io.ReadFull(x.c, head); err != nil {
			return nil, err
		}
		switch head[0] {
		case 0:
			return nil, fmt.Errorf("overlay: X error %d for request %d", head[1], op)
		case 1:
			more := make([]byte, 4*int(le.Uint32(head[4:])))
			if _, err := io.ReadFull(x.c, more); err != nil {
				return nil, err
			}
			return append(head, more...), nil
		}
	}
}

// geometry returns a window's depth and size.
func (x *xconn) geometry(win uint32) (depth byte, w, h int, err error) {
	r, err := x.roundTrip(opGetGeometry, 0, x.u32(win))
	if err != nil {
		return 0, 0, 0, err
	}
	return r[1], int(le.Uint16(r[16:])), int(le.Uint16(r[18:])), nil
}

// alphaAt reads the alpha of one pixel of a 32-bit window, 255 if it cannot.
// A window not yet on screen answers GetImage with BadMatch: termimg started
// with its terminal (alacritty -e termimg, or its own Xwayland relaunch)
// asks before the window is mapped, so it asks again for up to a second.
// shortcut: the bottom right corner is taken for background; text drawn
// there would read as opaque, which is how pictures were shown before.
func (x *xconn) alphaAt(win uint32, px, py int) byte {
	b := x.u32(win, 0, 0, 0xffffffff)
	le.PutUint16(b[4:], uint16(px))
	le.PutUint16(b[6:], uint16(py))
	le.PutUint16(b[8:], 1)
	le.PutUint16(b[10:], 1)
	for range 20 {
		r, err := x.roundTrip(opGetImage, imageFormatZPixmap, b)
		if err == nil && len(r) >= 36 {
			return r[35] // BGRA, little-endian
		}
		time.Sleep(50 * time.Millisecond)
	}
	return 0xff
}

// windows reads a property holding 32-bit values (window ids, a pid).
func (x *xconn) windows(win uint32, prop string) []uint32 {
	r, err := x.roundTrip(opGetProperty, 0, x.u32(win, x.atoms[prop], anyPropertyType, 0, maxPropertyWords))
	if err != nil || r[1] != 32 {
		return nil
	}
	n := min(int(le.Uint32(r[16:])), (len(r)-32)/4)
	out := make([]uint32, n)
	for i := range out {
		out[i] = le.Uint32(r[32+4*i:])
	}
	return out
}

// attach makes win the parent of every picture: it learns the window's depth
// and size, asks for its ConfigureNotify events, and starts the reader.
func (x *xconn) attach(win uint32) error {
	depth, w, h, err := x.geometry(win)
	if err != nil {
		return err
	}
	if !x.deep[depth] {
		return fmt.Errorf("overlay: no 32-bit pixel format for depth %d", depth)
	}
	x.term, x.depth, x.alpha = win, depth, 0xff
	if depth == 32 && w > 0 && h > 0 {
		x.alpha = x.alphaAt(win, w-1, h-1)
	}
	x.size.Store(uint32(w)<<16 | uint32(h))
	// Other clients may watch a window's structure too; the terminal keeps
	// its own event mask.
	x.send(opChangeWindowAttributes, 0, x.u32(win, cwEventMask, structureNotifyMask))
	// A GC is made for one depth: make it on a pixmap of the window's.
	pix := x.id()
	x.gc = x.id()
	b := x.u32(pix, x.root, 1<<16|1)
	x.send(opCreatePixmap, depth, b)
	x.send(opCreateGC, 0, x.u32(x.gc, pix, 0))
	x.send(opFreePixmap, 0, x.u32(pix))
	go x.read()
	return x.flush()
}

// read takes everything the server sends until the connection closes:
// errors and events are 32 bytes, replies may carry more.
func (x *xconn) read() {
	b := make([]byte, 32)
	for {
		if _, err := io.ReadFull(x.c, b); err != nil {
			return
		}
		switch code := b[0] & 0x7f; {
		case code == 1 || code == 35:
			if _, err := io.CopyN(io.Discard, x.c, 4*int64(le.Uint32(b[4:]))); err != nil {
				return
			}
		case code == eventConfigureNotify && le.Uint32(b[8:]) == x.term:
			x.size.Store(uint32(le.Uint16(b[20:]))<<16 | uint32(le.Uint16(b[22:])))
		case code == eventDestroyNotify && le.Uint32(b[8:]) == x.term:
			x.size.Store(0)
		}
	}
}

func displayNumber(display string) (string, error) {
	if !strings.HasPrefix(display, ":") {
		return "", fmt.Errorf("overlay: DISPLAY %q is not local", display)
	}
	num, _, _ := strings.Cut(display[1:], ".")
	if num == "" || strings.Trim(num, "0123456789") != "" {
		return "", fmt.Errorf("overlay: bad DISPLAY %q", display)
	}
	return num, nil
}

// xauthCookie finds the MIT-MAGIC-COOKIE-1 for the display in the
// Xauthority file. None found, the connection is tried without one: an
// Xwayland that trusts the local user takes it.
func xauthCookie(num string) (name string, data []byte) {
	path := os.Getenv("XAUTHORITY")
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".Xauthority")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	return parseXauth(b, num)
}

func parseXauth(b []byte, num string) (string, []byte) {
	field := func() []byte {
		if len(b) < 2 {
			b = nil
			return nil
		}
		n := int(binary.BigEndian.Uint16(b))
		if len(b) < 2+n {
			b = nil
			return nil
		}
		f := b[2 : 2+n]
		b = b[2+n:]
		return f
	}
	for len(b) >= 2 {
		b = b[2:] // family: the socket is local either way
		field()   // address
		number, name, data := field(), field(), field()
		if b == nil {
			break
		}
		if (len(number) == 0 || string(number) == num) && string(name) == "MIT-MAGIC-COOKIE-1" {
			return string(name), data
		}
	}
	return "", nil
}

func pad4(n int) int { return (n + 3) &^ 3 }

func (x *xconn) setup(name string, data []byte) error {
	req := make([]byte, 12, 12+pad4(len(name))+pad4(len(data)))
	req[0] = 'l'
	le.PutUint16(req[2:], 11)
	le.PutUint16(req[6:], uint16(len(name)))
	le.PutUint16(req[8:], uint16(len(data)))
	req = append(req, name...)
	req = append(req, make([]byte, pad4(len(name))-len(name))...)
	req = append(req, data...)
	req = append(req, make([]byte, pad4(len(data))-len(data))...)
	if _, err := x.c.Write(req); err != nil {
		return err
	}
	head := make([]byte, 8)
	if _, err := io.ReadFull(x.c, head); err != nil {
		return err
	}
	body := make([]byte, 4*int(le.Uint16(head[6:])))
	if _, err := io.ReadFull(x.c, body); err != nil {
		return err
	}
	if head[0] != 1 {
		return fmt.Errorf("overlay: X server refused the connection: %s", bytes.TrimRight(body, "\x00"))
	}
	return x.parseSetup(body)
}

// parseSetup reads the setup reply's body: the resource id range, the
// request size limit, the pixel formats, and the first screen's root window.
func (x *xconn) parseSetup(b []byte) error {
	if len(b) < 32 {
		return errors.New("overlay: short X setup reply")
	}
	x.base, x.mask = le.Uint32(b[4:]), le.Uint32(b[8:])
	vendor, nFormats := int(le.Uint16(b[16:])), int(b[21])
	x.maxReq = 4 * int(le.Uint16(b[18:]))
	if b[22] != 0 {
		return errors.New("overlay: X server wants big-endian images")
	}
	formats := 32 + pad4(vendor)
	screen := formats + 8*nFormats
	if len(b) < screen+40 {
		return errors.New("overlay: short X setup reply")
	}
	x.root = le.Uint32(b[screen:])
	x.deep = make(map[byte]bool)
	for f := formats; f < screen; f += 8 {
		if b[f+1] == 32 {
			x.deep[b[f]] = true
		}
	}
	return nil
}

// intern looks up the atoms before anything else is sent, so the replies
// are the next thing on the wire.
func (x *xconn) intern(names ...string) error {
	for _, n := range names {
		body := make([]byte, 4, 4+pad4(len(n)))
		le.PutUint16(body, uint16(len(n)))
		body = append(body, n...)
		x.send(opInternAtom, 0, body)
	}
	if err := x.flush(); err != nil {
		return err
	}
	x.atoms = make(map[string]uint32, len(names))
	reply := make([]byte, 32)
	for _, n := range names {
		if _, err := io.ReadFull(x.c, reply); err != nil {
			return err
		}
		if reply[0] != 1 {
			return fmt.Errorf("overlay: X error %d interning %s", reply[1], n)
		}
		x.atoms[n] = le.Uint32(reply[8:])
	}
	return nil
}

func (x *xconn) id() uint32 {
	x.next++
	return x.base | (x.next * (x.mask & -x.mask) & x.mask)
}

func (x *xconn) u32(v ...uint32) []byte {
	b := make([]byte, 4*len(v))
	for i, w := range v {
		le.PutUint32(b[4*i:], w)
	}
	return b
}

func (x *xconn) send(op, detail byte, body []byte) {
	n := 4 + pad4(len(body))
	x.buf = append(x.buf, op, detail, 0, 0)
	le.PutUint16(x.buf[len(x.buf)-2:], uint16(n/4))
	x.buf = append(x.buf, body...)
	x.buf = append(x.buf, make([]byte, n-4-len(body))...)
}

func (x *xconn) flush() error {
	_, err := x.c.Write(x.buf)
	x.buf = x.buf[:0]
	return err
}

// createPixmap sends w×h pixels to the server, in a pixmap of the terminal
// window's depth (32-bit for a translucent terminal), and returns it.
func (x *xconn) createPixmap(px []byte, w, h int) uint32 {
	pix := x.id()
	b := x.u32(pix, x.root, 0)
	le.PutUint16(b[8:], uint16(w))
	le.PutUint16(b[10:], uint16(h))
	x.send(opCreatePixmap, x.depth, b)
	rows := max(1, (x.maxReq-24)/(4*w))
	for y := 0; y < h; y += rows {
		n := min(rows, h-y)
		body := make([]byte, 20, 20+4*w*n)
		le.PutUint32(body, pix)
		le.PutUint32(body[4:], x.gc)
		le.PutUint16(body[8:], uint16(w))
		le.PutUint16(body[10:], uint16(n))
		le.PutUint16(body[14:], uint16(y))
		body[17] = x.depth
		body = append(body, px[4*w*y:4*w*(y+n)]...)
		x.send(opPutImage, imageFormatZPixmap, body)
	}
	return pix
}

// createWindow makes a child of the terminal's window at x, y with pix for
// its background, not yet mapped, so the server repaints it by itself. It
// takes the parent's depth and visual, and asks for no input, so clicks over
// it reach the terminal as before.
func (x *xconn) createWindow(pix uint32, posX, posY, w, h int) uint32 {
	win := x.id()
	// Depth and visual 0: CopyFromParent.
	b := x.u32(win, x.term, 0, 0, windowClassInputOutput<<16, 0, cwBackPixmap, pix)
	le.PutUint16(b[8:], uint16(int16(posX)))
	le.PutUint16(b[10:], uint16(int16(posY)))
	le.PutUint16(b[12:], uint16(w))
	le.PutUint16(b[14:], uint16(h))
	x.send(opCreateWindow, 0, b)
	return win
}

// setBackground shows another pixmap in win: a new background, painted at
// once by clearing the window to it, with nothing in between to flicker.
func (x *xconn) setBackground(win, pix uint32) {
	x.send(opChangeWindowAttributes, 0, x.u32(win, cwBackPixmap, pix))
	x.send(opClearArea, 0, x.u32(win, 0, 0)) // 0×0 at 0,0: the whole window
}

func (x *xconn) freePixmap(pix uint32) { x.send(opFreePixmap, 0, x.u32(pix)) }

func (x *xconn) move(win uint32, posX, posY int) {
	b := x.u32(win, configureX|configureY, uint32(int32(posX)), uint32(int32(posY)))
	x.send(opConfigureWindow, 0, b)
}

func (x *xconn) mapWindow(win uint32)     { x.send(opMapWindow, 0, x.u32(win)) }
func (x *xconn) destroyWindow(win uint32) { x.send(opDestroyWindow, 0, x.u32(win)) }
