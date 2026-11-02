//go:build unix

package main

import (
	"bytes"
	"image"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
	"golang.org/x/sys/unix"

	"github.com/thebanri/limoni/internal/overlay"
)

// session sits between the program in the pty and the real terminal. The
// program's output goes on unchanged, but for its sixel pictures, which are
// shown in the overlay (half blocks without one), and for the questions the
// real terminal cannot answer about sixel, which are answered here. The same
// output feeds a shadow screen that says where the cursor is and where the
// pictures went (pictures.go).
type session struct {
	mu    sync.Mutex
	out   *os.File // the real terminal
	ptmx  *os.File
	emu   *vt.Emulator
	pics  pictures
	ov    *overlay.Overlay // nil: half blocks
	parse parser
	cols  int
	rows  int
	cellW int
	cellH int
	sdm   bool // DECSDM: pictures go to the top left, the cursor stays
	kitty kitty
	buf   []byte // what goes to the real terminal from this write
	// dirty is set when the screen may have changed: text went to it or a
	// picture was placed. A read that only adds to a picture still arriving
	// (a video frame comes in hundreds) leaves it alone.
	dirty  bool
	shown  []overlay.Picture
	pixels bool        // the cell size is measured, not guessed
	timer  *time.Timer // the next animation frame
	iterm  struct {    // an iTerm2 file arriving in parts
		args  string
		parts []byte
	}
	// margins is set while a scroll region may be in force, lrmm while left
	// and right margins may: lines then scroll through part of the screen.
	margins, lrmm bool
	scroll        scrollback
}

func newSession(out, ptmx *os.File) *session {
	s := &session{out: out, ptmx: ptmx}
	if ov, err := overlay.Open(); err == nil {
		// Every sixel picture is decoded into a new value, so a kept copy
		// would never be shown again.
		ov.SetCache(0)
		s.ov = ov
	}
	s.cols, s.rows = 80, 24
	s.emu = vt.NewEmulator(s.cols, s.rows)
	// Only the screen is looked at; the scrollback counts the lines a write
	// scrolls off it (pictures.update), and is emptied after every look.
	// shortcut: a write that scrolls more lines than this between two looks
	// leaves the pictures above the screen too low; raise it if one shows.
	s.emu.SetScrollbackSize(1024)
	// The shadow screen answers queries too; the real terminal already has.
	go io.Copy(io.Discard, s.emu)
	s.pics = pictures{emu: s.emu, byID: map[int]*picture{}}
	s.resize()
	return s
}

// resize follows the real terminal's size, and gives the pty its size in
// pixels too: programs size their sixel pictures by it. With the overlay
// the cell size is measured from the terminal's window; without, it is what
// the terminal reports, or a guess.
func (s *session) resize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	ws, err := unix.IoctlGetWinsize(int(s.out.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 {
		return
	}
	newWidth := int(ws.Col) != s.cols
	s.cols, s.rows = int(ws.Col), int(ws.Row)
	s.cellW, s.cellH, s.pixels = 10, 20, false
	if s.ov != nil {
		if w, h := s.ov.CellSize(s.cols, s.rows); w > 0 {
			s.cellW, s.cellH, s.pixels = w, h, true
		}
	} else if ws.Xpixel > 0 && ws.Ypixel > 0 {
		s.cellW, s.cellH, s.pixels = int(ws.Xpixel)/s.cols, int(ws.Ypixel)/s.rows, true
	}
	s.keepCursor(s.rows)
	s.emu.Resize(s.cols, s.rows)
	if s.ov != nil && newWidth {
		// The terminal lays its lines out anew at another width; at
		// another height they stay, scrolled as keepCursor counts.
		s.ov.Forget()
	}
	s.resized()
	_ = pty.Setsize(s.ptmx, &pty.Winsize{
		Cols: uint16(s.cols), Rows: uint16(s.rows),
		X: uint16(s.cols * s.cellW), Y: uint16(s.rows * s.cellH),
	})
	s.pics.update()
	s.show()
}

// keepCursor scrolls the shadow screen up as a terminal does when it gets
// shorter than the row the cursor is on: the lines above go to its
// scrollback, and the pictures with them, rather than the bottom being cut.
func (s *session) keepCursor(rows int) {
	at := s.emu.CursorPosition()
	n := at.Y - rows + 1
	if n <= 0 || s.emu.IsAltScreen() {
		return
	}
	b := cup(nil, 0, s.emu.Height()-1)
	b = append(b, bytes.Repeat([]byte{'\n'}, n)...)
	s.emu.Write(cup(b, at.X, at.Y-n))
}

// output handles a read of the program's output.
func (s *session) output(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf, s.dirty = s.buf[:0], false
	s.changed()
	s.parse.feed(s.skipScrolled(b), s)
	if s.dirty {
		s.pics.update()
	}
	s.animate()
	if s.dirty {
		s.show()
	}
	if len(s.buf) == 0 {
		return nil
	}
	_, err := s.out.Write(s.buf)
	return err
}

// skipScrolled spares the shadow screen the lines that would only scroll
// through it: x/vt copies the whole screen for each line it scrolls, which
// made cat of a large file 23 times slower than in Alacritty alone. When b
// starts with a screenful of plain lines (text and SGR) after others, those
// others scroll off whatever was there; they go on to the real terminal,
// and the shadow screen gets their colours and a clear screen instead. It
// returns what is left for the parser.
func (s *session) skipScrolled(b []byte) []byte {
	// A sequence the last read cut in two is finished first.
	for len(b) > 0 && (s.parse.state == stEsc || s.parse.state == stCSI) {
		s.parse.feed(b[:1], s)
		b = b[1:]
	}
	if s.parse.state != stText || s.margins || s.lrmm {
		return b
	}
	end := 0 // the plain start of b
	for end < len(b) {
		c := b[end]
		if c == 0x1b {
			n := sgrLen(b[end:])
			if n == 0 {
				break
			}
			end += n
			continue
		}
		if c < 0x20 && c != '\r' && c != '\n' && c != '\t' || c == 0x7f ||
			c == 0xc2 && end+1 < len(b) && b[end+1] >= 0x80 && b[end+1] <= 0x9f { // C1 controls, which x/vt obeys
			break
		}
		end++
	}
	// The last rows-1 lines stay; the ones before, at least a screenful
	// so the cursor is on the bottom row at the cut, scroll off.
	cut := end
	for range s.rows {
		if cut = bytes.LastIndex(b[:cut], []byte("\r\n")); cut < 0 {
			return b
		}
	}
	cut += 2
	if bytes.Count(b[:cut], []byte("\r\n")) < s.rows {
		return b
	}
	// The rest of a character the last read cut in two comes first: x/vt
	// would take the clear for part of it.
	i := 0
	for i < cut && b[i]&0xc0 == 0x80 {
		i++
	}
	pen := append(b[:i:i], "\x1b[H\x1b[2J"...)
	// colours set before the last reset are gone anyway
	i = max(i, bytes.LastIndex(b[:cut], []byte("\x1b[0m")), bytes.LastIndex(b[:cut], []byte("\x1b[m")))
	for ; i < cut; i++ {
		if n := sgrLen(b[i:]); n > 0 {
			pen = append(pen, b[i:i+n]...)
			i += n - 1
		}
	}
	// What was on the screen scrolls off its top: as many lines as came
	// with the cursor on the bottom row.
	at := s.emu.CursorPosition()
	s.pics.scrollOff(max(0, bytes.Count(b[:cut], []byte("\n"))-(s.rows-1-at.Y)))
	s.emu.Write(pen)
	s.emu.ClearScrollback() // the clear is not a scroll
	s.buf = append(s.buf, b[:cut]...)
	s.dirty = true
	return b[cut:]
}

// sgrLen is the length of the SGR sequence b starts with, or 0.
func sgrLen(b []byte) int {
	if len(b) < 3 || b[0] != 0x1b || b[1] != '[' {
		return 0
	}
	for i := 2; i < len(b); i++ {
		switch c := b[i]; {
		case c == 'm':
			return i + 1
		case c != ';' && c != ':' && (c < '0' || c > '9'):
			return 0
		}
	}
	return 0
}

func (s *session) text(b []byte) {
	s.dirty = true
	s.buf = append(s.buf, b...)
	s.emu.Write(b)
}

// csi answers the sixel questions the real terminal would get wrong or not
// answer, and notes DECSDM; the rest goes on.
func (s *session) csi(seq []byte) bool {
	body, final := seq[2:len(seq)-1], seq[len(seq)-1]
	switch {
	case final == 'S' && bytes.HasPrefix(body, []byte("?")):
		s.xtsmgraphics(body[1:])
		return true
	case final == 't' && s.pixels && string(body) == "14":
		s.reply("\x1b[4;" + strconv.Itoa(s.rows*s.cellH) + ";" + strconv.Itoa(s.cols*s.cellW) + "t")
		return true
	case final == 't' && s.pixels && string(body) == "16":
		s.reply("\x1b[6;" + strconv.Itoa(s.cellH) + ";" + strconv.Itoa(s.cellW) + "t")
		return true
	case len(body) > 0 && len(bytes.TrimLeft(body, "0")) == 0 && bytes.IndexByte([]byte("ABCDEFabe@PLMSTb"), final) >= 0:
		// An explicit 0 means 1 to these, in xterm and every terminal after
		// it; the shadow screen takes it for 0 (kitten icat starts each
		// placeholder row with CSI 0 C), and would lose the cursor.
		s.buf = append(s.buf, seq...)
		s.emu.Write([]byte{0x1b, '[', '1', final})
		s.dirty = true
		return true
	case (final == 'h' || final == 'l') && bytes.HasPrefix(body, []byte("?")):
		for _, m := range bytes.Split(body[1:], []byte(";")) {
			switch string(m) {
			case "80":
				s.sdm = final == 'h'
			case "69": // DECLRMM; x/vt keeps the margins when it is reset
				s.lrmm = s.lrmm || final == 'h'
			}
		}
	case final == 'J' && string(body) == "3": // ED 3: the terminal empties its scrollback
		s.pics.gone = nil
		if s.ov != nil {
			s.ov.Forget()
		}
		s.dirty = true
	case final == 'r' && (len(body) == 0 || body[0] < '<'): // DECSTBM, not a private one
		// shortcut: a reset (RIS, DECSTR) leaves margins set, which only
		// costs speed until the next DECSTBM.
		top, bottom, _ := bytes.Cut(body, []byte(";"))
		t, _ := strconv.Atoi(string(top))
		b, _ := strconv.Atoi(string(bottom))
		s.margins = t > 1 || b > 0 && b < s.rows
	}
	return false
}

// xtsmgraphics answers XTSMGRAPHICS: 256 colour registers, and pictures up
// to the size of the screen. Setting either is accepted and changes nothing.
func (s *session) xtsmgraphics(body []byte) {
	item, _, _ := bytes.Cut(body, []byte(";"))
	switch string(item) {
	case "1":
		s.reply("\x1b[?1;0;256S")
	case "2":
		s.reply("\x1b[?2;0;" + strconv.Itoa(s.cols*s.cellW) + ";" + strconv.Itoa(s.rows*s.cellH) + "S")
	default:
		s.reply("\x1b[?" + string(item) + ";1S") // error: unknown item
	}
}

func (s *session) reply(r string) { _, _ = s.ptmx.WriteString(r) }

// sixel places a picture where a terminal with sixel would: at the cursor,
// which then goes to its last row; with DECSDM at the top left, the cursor
// left where it was.
func (s *session) sixel(data []byte) {
	img := decodeSixel(data)
	b := img.Bounds()
	if b.Empty() {
		return
	}
	cols := (b.Dx() + s.cellW - 1) / s.cellW
	rows := (b.Dy() + s.cellH - 1) / s.cellH
	if s.sdm {
		s.place(img, 0, 0, cols, rows, false)
		return
	}
	s.put(img, cols, rows, moveBelow, false)
}

// How the cursor moves after a picture, as each protocol has it.
const (
	moveBelow = iota // sixel: to its last row, in the same column, as in xterm
	moveAfter        // kitty, iTerm2: past its right edge, on its last row
	moveNone         // kitty with C=1
)

// put shows img, already at its size on screen, over cols×rows cells at the
// cursor, and moves the cursor on, scrolling the screen as the terminal
// would. It returns the picture, nil if none is tracked (half blocks).
func (s *session) put(img image.Image, cols, rows, move int, sticky bool) *picture {
	at := s.emu.CursorPosition()
	top := at.Y
	switch move {
	case moveBelow:
		// xterm's rule, which fastfetch counts on to go back up beside it:
		// text after a picture needs one newline more, or it cuts its last row.
		s.text(bytes.Repeat([]byte{'\n'}, rows-1))
		top = s.emu.CursorPosition().Y - (rows - 1)
	case moveAfter:
		s.text(bytes.Repeat([]byte{'\n'}, rows-1))
		top = s.emu.CursorPosition().Y - (rows - 1)
		s.text([]byte("\x1b[" + strconv.Itoa(cols) + "C"))
	}
	return s.place(img, at.X, top, cols, rows, sticky)
}

// place shows img over cols×rows cells at col, top: in the overlay, or in
// half blocks without one.
func (s *session) place(img image.Image, col, top, cols, rows int, sticky bool) *picture {
	if s.ov == nil {
		s.halfBlocks(img, col, top, cols, rows)
		return nil
	}
	img, dx, dy := opaque(img)
	if img == nil {
		return nil // nothing to see
	}
	s.dirty = true
	pic := s.pics.byID[s.pics.add(img, col, top, cols, rows, sticky)]
	if pic != nil {
		pic.dx, pic.dy = dx, dy
	}
	return pic
}

// opaque cuts off the fully transparent edges of img: programs pad pictures
// to whole cells that way (chafa does), and the overlay can only show the
// terminal's background colour there, where a terminal would show what is
// under them. It returns the rest, where that starts in img, and nil if
// nothing is left.
func opaque(img image.Image) (image.Image, int, int) {
	b := img.Bounds()
	seen := image.Rectangle{}
	var alpha func(x, y int) uint8
	switch m := img.(type) {
	case *image.NRGBA:
		alpha = func(x, y int) uint8 { return m.Pix[m.PixOffset(x, y)+3] }
	case *image.RGBA:
		alpha = func(x, y int) uint8 { return m.Pix[m.PixOffset(x, y)+3] }
	default:
		if o, ok := img.(interface{ Opaque() bool }); ok && o.Opaque() {
			return img, 0, 0
		}
		alpha = func(x, y int) uint8 { _, _, _, a := img.At(x, y).RGBA(); return uint8(a >> 8) }
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if alpha(x, y) != 0 {
				seen = seen.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	switch {
	case seen.Empty():
		return nil, 0, 0
	case seen == b:
		return img, 0, 0
	}
	return subImage(img, seen), seen.Min.X - b.Min.X, seen.Min.Y - b.Min.Y
}

// show puts the overlay's windows where the pictures are now.
func (s *session) show() {
	if s.ov == nil {
		return
	}
	s.shown = s.shown[:0]
	add := func(pic *picture) {
		img := pic.img
		if h := pic.rows*s.cellH - pic.dy - pic.oy; h < img.Bounds().Dy() {
			if h <= 0 {
				return
			}
			img = pic.cropped(h)
		}
		s.shown = append(s.shown, overlay.Picture{
			Img: img,
			X:   pic.col*s.cellW + pic.dx + pic.ox, Y: pic.top*s.cellH + pic.dy + pic.oy,
			W: img.Bounds().Dx(), H: img.Bounds().Dy(),
		})
	}
	alt := s.emu.IsAltScreen()
	for _, pic := range s.pics.byID {
		if pic.alt == alt {
			add(pic)
		}
	}
	if !alt {
		// Above the screen, where the overlay brings them in as the
		// terminal shows its scrollback; X takes positions in 16 bits.
		for _, pic := range s.pics.gone {
			if pic.top*s.cellH > -30000 {
				add(pic)
			}
		}
	}
	if len(s.kitty.virtual) > 0 {
		s.shown = s.placeholders(s.shown)
	}
	_ = s.ov.Show(s.shown, s.cols*s.cellW, s.rows*s.cellH)
}

// halfBlocks draws the picture into the real terminal's cells, without the
// shadow screen knowing: it is ordinary text there, scrolled and written
// over by the terminal itself, so it needs no tracking.
func (s *session) halfBlocks(img image.Image, col, top, cols, rows int) {
	at := s.emu.CursorPosition()
	s.buf = drawBlocks(s.buf, img, col, top, min(cols, s.cols-col), rows, s.rows)
	s.buf = cup(s.buf, at.X, at.Y)
}

func (s *session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
	}
	if s.ov != nil {
		_ = s.ov.Close()
		s.ov = nil // a frame due meanwhile shows nothing
	}
	_ = s.emu.Close()
}
