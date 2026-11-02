package overlay

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/thebanri/limoni/graphics"
)

// Overlay shows pictures as child windows of the terminal's X11 window, as
// w3m and ueberzug do. Being part of the terminal's window, they move, hide
// and show with it under any window manager, and none animates them. It
// needs the terminal to have an X11 window: any terminal in an X11 session,
// or one started under Xwayland in a Wayland session
// (env -u WAYLAND_DISPLAY alacritty, GDK_BACKEND=x11, QT_QPA_PLATFORM=xcb).
//
// shortcut: a tabbed terminal shows the pictures over every tab; hide them on
// focus-out (mode 1004) if that bites.
type Overlay struct {
	x     *xconn
	wins  []*window
	stamp uint64
	cellW int
	cellH int
	// pix holds every picture sent to the server, at the size it was shown,
	// so a picture shown again (an animation's next loop, a picture scrolled
	// back into view) costs no upload; pixBytes is what they take there.
	pix      map[picKey]*pixmap
	pixBytes int
	maxBytes int
	// bg is what transparent pixels are composed over, premultiplied: the
	// terminal's background, as see-through as the terminal draws it.
	bg color.RGBA
	// padX, padY are where the cell grid starts in the terminal's window and
	// gridW, gridH its size, as Show last placed it.
	padX, padY, gridW, gridH int
	// The terminal's scrollback (scroll.go): the window as it was at the
	// bottom, how far the pictures are moved down to follow it, and whether
	// they are hidden because it shows nothing that matches.
	anchor *sample
	dy     int
	hidden bool
	// probe is a second connection for reading the window: the first one's
	// replies are taken by its reader once attached.
	probe *xconn
}

// maxPixBytes bounds the pictures kept in the X server. Past it the least
// recently shown go, never one on screen.
const maxPixBytes = 128 << 20

type picKey struct {
	img  image.Image
	w, h int
}

type pixmap struct {
	id    uint32
	stamp uint64
}

type window struct {
	id    uint32
	key   picKey
	x, y  int
	stamp uint64
}

// Open connects to the X server and finds the terminal's window.
func Open() (*Overlay, error) {
	x, err := dialX(os.Getenv("DISPLAY"))
	if err != nil {
		return nil, err
	}
	win, err := x.findTerminal(ancestors(os.Getpid()))
	if err == nil {
		err = x.attach(win)
	}
	if err != nil {
		x.c.Close()
		return nil, err
	}
	return &Overlay{x: x, pix: make(map[picKey]*pixmap), maxBytes: maxPixBytes, bg: color.RGBA{A: x.alpha}}, nil
}

// SetBackground sets the colour transparent pixels show, black to start
// with: a window cannot show the terminal through it, so the terminal's own
// background colour is the nearest thing, at the terminal's own opacity.
// Pictures already sent keep theirs.
func (o *Overlay) SetBackground(c color.Color) {
	r, g, b, _ := c.RGBA()
	a := uint32(o.bg.A)
	o.bg = color.RGBA{uint8(r >> 8 * a / 255), uint8(g >> 8 * a / 255), uint8(b >> 8 * a / 255), o.bg.A}
}

// SetCache sets how many bytes of pictures not on screen are kept in the X
// server for showing again (128 MiB to start with). A caller whose pictures
// are new values every time, such as decoded video frames, sets 0: keeping
// them would only hold memory, and the pictures with it.
func (o *Overlay) SetCache(bytes int) { o.maxBytes = bytes }

// findTerminal returns the window of the terminal this process runs in:
// WINDOWID, which most X11 terminals set, else the window whose _NET_WM_PID
// is the nearest of pids (the active one, if that process has several).
func (x *xconn) findTerminal(pids []int) (uint32, error) {
	if id, err := strconv.ParseUint(os.Getenv("WINDOWID"), 10, 32); err == nil && id != 0 {
		// A terminal started from this one inherits WINDOWID, wrongly; where
		// the window names its process, that must be one of ours.
		pid := x.windows(uint32(id), "_NET_WM_PID")
		if len(pid) == 0 || slices.Contains(pids, int(pid[0])) {
			if _, _, _, err := x.geometry(uint32(id)); err == nil {
				return uint32(id), nil
			}
		}
	}
	list := x.windows(x.root, "_NET_CLIENT_LIST")
	var active uint32
	if a := x.windows(x.root, "_NET_ACTIVE_WINDOW"); len(a) > 0 {
		active = a[0]
	}
	owners := make(map[uint32]int, len(list))
	for _, w := range list {
		if pid := x.windows(w, "_NET_WM_PID"); len(pid) > 0 {
			owners[w] = int(pid[0])
		}
	}
	for _, pid := range pids {
		var found uint32
		for _, w := range list {
			if owners[w] != pid {
				continue
			}
			if w == active {
				return w, nil
			}
			if found == 0 {
				found = w
			}
		}
		if found != 0 {
			return found, nil
		}
	}
	return 0, errors.New("overlay: the terminal has no X11 window (a Wayland-native terminal has none)")
}

// CellSize is the size of a cell in pixels for a terminal of cols×rows: the
// inside of its window divided among the cells. A terminal leaves less than
// a cell of padding, so a size that leaves more is the window's old size
// against the new cell count (its ConfigureNotify is still on the way); the
// last good size stands in, as cells keep their size when a window is resized.
// shortcut: a font zoom shows the old size until the next frame with
// pictures; redraw on the window's ConfigureNotify if that ever shows.
func (o *Overlay) CellSize(cols, rows int) (w, h int) {
	pw, ph := o.windowSize()
	if cols > 0 && rows > 0 {
		if cw, ch := pw/cols, ph/rows; cw > 0 && ch > 0 && pw-cw*cols < cw && ph-ch*rows < ch {
			o.cellW, o.cellH = cw, ch
		}
	}
	return o.cellW, o.cellH
}

func (o *Overlay) windowSize() (w, h int) {
	size := o.x.size.Load()
	return int(size >> 16), int(size & 0xffff)
}

// Show makes the windows on screen exactly pics. A window already showing a
// picture is kept (moved, if it must); one of the same size in the same
// place takes the new picture as its background, which is how an animation's
// frames follow each other without a window coming and going. gridW and
// gridH are the terminal's cells in pixels, centred in its window as a
// terminal with leftover space (dynamic padding) draws them.
// shortcut: a terminal that pads only the top left (Alacritty without
// dynamic_padding) puts pictures up to half a cell off.
func (o *Overlay) Show(pics []Picture, gridW, gridH int) error {
	o.stamp++
	winW, winH := o.windowSize()
	if winW == 0 {
		pics = nil // the terminal's window is gone, and the pictures with it
	}
	padX, padY := max(0, (winW-gridW)/2), max(0, (winH-gridH)/2)
	o.padX, o.padY, o.gridW, o.gridH = padX, padY, gridW, gridH
	for _, p := range pics {
		if p.W <= 0 || p.H <= 0 || p.Img == nil {
			continue
		}
		// The window takes the picture's shape, centred in its cells, so
		// what it does not cover shows the terminal's own background rather
		// than bands of the picture's corner colour, as ResizeImageContain
		// fills them for the protocols that need the whole area.
		fw, fh := graphics.ContainSize(p.Img, p.W, p.H)
		k, x, y := picKey{p.Img, fw, fh}, padX+p.X+(p.W-fw)/2, padY+p.Y+(p.H-fh)/2
		win := o.unused(func(w *window) bool { return w.key == k })
		if win == nil {
			if win = o.unused(func(w *window) bool { return w.key.w == fw && w.key.h == fh && w.x == x && w.y == y }); win != nil {
				o.x.setBackground(win.id, o.pixmap(k))
				win.key = k
			}
		}
		if win == nil {
			win = &window{id: o.x.createWindow(o.pixmap(k), x, y+o.dy, fw, fh), key: k, x: x, y: y}
			o.wins = append(o.wins, win)
			if !o.hidden {
				o.x.mapWindow(win.id)
			}
		} else if win.x != x || win.y != y {
			win.x, win.y = x, y
			o.x.move(win.id, x, y+o.dy)
		}
		win.stamp = o.stamp
		o.pix[k].stamp = o.stamp
	}
	// New windows are mapped before old ones go, so a resized picture is
	// never a hole for a moment.
	kept := o.wins[:0]
	for _, w := range o.wins {
		if w.stamp == o.stamp {
			kept = append(kept, w)
		} else {
			o.x.destroyWindow(w.id)
		}
	}
	clear(o.wins[len(kept):])
	o.wins = kept
	o.evict()
	return o.x.flush()
}

// unused returns a window not yet claimed this frame that match accepts.
func (o *Overlay) unused(match func(*window) bool) *window {
	for _, w := range o.wins {
		if w.stamp != o.stamp && match(w) {
			return w
		}
	}
	return nil
}

// pixmap returns the server's copy of a picture, sending it the first time.
func (o *Overlay) pixmap(k picKey) uint32 {
	if p := o.pix[k]; p != nil {
		return p.id
	}
	p := &pixmap{id: o.x.createPixmap(bgrx(k.img, k.w, k.h, o.bg), k.w, k.h)}
	o.pix[k] = p
	o.pixBytes += 4 * k.w * k.h
	return p.id
}

// Trim frees the pictures kept that are not on screen now.
func (o *Overlay) Trim() error {
	keep := o.maxBytes
	o.maxBytes = 0
	o.evict()
	o.maxBytes = keep
	return o.x.flush()
}

// evict frees the least recently shown pictures until they fit maxBytes.
func (o *Overlay) evict() {
	for o.pixBytes > o.maxBytes {
		var oldest picKey
		var found *pixmap
		for k, p := range o.pix {
			if p.stamp != o.stamp && (found == nil || p.stamp < found.stamp) {
				oldest, found = k, p
			}
		}
		if found == nil {
			return // everything left is on screen
		}
		o.x.freePixmap(found.id)
		delete(o.pix, oldest)
		o.pixBytes -= 4 * oldest.w * oldest.h
	}
}

// Close removes every window; the X server would on disconnect anyway, with
// the pictures it holds.
func (o *Overlay) Close() error {
	if o.probe != nil {
		o.probe.c.Close()
	}
	err := o.Show(nil, 0, 0)
	if cerr := o.x.c.Close(); err == nil {
		err = cerr
	}
	return err
}

// bgrx scales img to w×h, its own shape (see ContainSize), composed over bg
// (premultiplied), in the X server's 32-bit little-endian pixel layout. The
// fourth byte is alpha for a 32-bit window and ignored otherwise.
func bgrx(img image.Image, w, h int, bg color.RGBA) []byte {
	// A picture already at its size (a video frame) is only converted:
	// resampling it to itself costs more than the rest of the frame.
	scaled := img
	if b := img.Bounds(); b.Dx() != w || b.Dy() != h {
		scaled = graphics.ResizeImage(img, w, h)
	}
	rgba, ok := scaled.(*image.RGBA)
	if !ok || rgba.Bounds() != image.Rect(0, 0, w, h) {
		rgba = image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(rgba, rgba.Bounds(), scaled, scaled.Bounds().Min, draw.Src)
	}
	px := make([]byte, 4*w*h)
	for y := range h {
		row := rgba.Pix[y*rgba.Stride : y*rgba.Stride+4*w]
		out := px[4*w*y:]
		for i := 0; i < len(row); i += 4 {
			// RGBA is premultiplied: the picture's share is already in it.
			t := 255 - uint32(row[i+3])
			out[i] = row[i+2] + uint8(uint32(bg.B)*t/255)
			out[i+1] = row[i+1] + uint8(uint32(bg.G)*t/255)
			out[i+2] = row[i] + uint8(uint32(bg.R)*t/255)
			out[i+3] = row[i+3] + uint8(uint32(bg.A)*t/255)
		}
	}
	return px
}

// ancestors lists pid and its parents, nearest first: the first that owns a
// window is the terminal.
func ancestors(pid int) []int {
	var out []int
	for pid > 1 {
		out = append(out, pid)
		stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if err != nil {
			break
		}
		// The command name may hold spaces and parentheses; the fields after
		// the last ')' are "state ppid ...".
		f := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
		if len(f) < 2 {
			break
		}
		pid, _ = strconv.Atoi(f[1])
	}
	return out
}
