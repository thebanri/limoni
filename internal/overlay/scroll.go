//go:build linux

package overlay

import (
	"image"
	"io"
	"os"
	"slices"
)

// A terminal shows its scrollback by itself (the wheel, Shift+PgUp) and
// tells no program, so pictures would stay where they are over other lines.
// What can be seen is the terminal's window: Anchor reads a few lines of
// pixels in every row of cells while the terminal is at the bottom, and
// Follow reads them again and looks for the number of rows the old ones
// have moved down by. The pictures move down with them; if no number fits
// (scrolled further than a screen, or the window shows something else),
// they are hidden until it does again.
// shortcut: a picture that had scrolled off the top before Anchor is not
// brought back by scrolling up to it.

// sampleLines are where in a row of cells its lines are read, in quarters.
var sampleLines = [...]int{1, 2, 3}

// sample is the terminal's window as Anchor or Follow read it.
type sample struct {
	rows   [][]byte   // per row of cells: its lines, 4 bytes a pixel
	skip   [][][2]int // per row: byte ranges under a picture, not compared
	cursor int        // the row whose content may blink, -1 for none
}

// Anchor reads the window with the terminal at the bottom, as Follow will
// compare it; cursor is the row of cells the cursor is in.
func (o *Overlay) Anchor(cursor int) bool {
	if o.dy != 0 || o.hidden {
		return false
	}
	a, ok := o.read()
	if ok {
		a.cursor = cursor
		o.anchor = &a
	}
	return ok
}

// Follow reads the window again and moves the pictures down by the rows it
// has scrolled since Anchor, or hides them when that cannot be told.
func (o *Overlay) Follow() error {
	if o.anchor == nil || o.cellH == 0 {
		return nil
	}
	b, ok := o.read()
	if !ok || len(b.rows) != len(o.anchor.rows) {
		return nil
	}
	k, found := shiftOf(o.anchor, &b, o.dy/o.cellH)
	if !found {
		return o.hide(true)
	}
	if dy := k * o.cellH; dy != o.dy {
		o.dy = dy
		for _, w := range o.wins {
			o.x.move(w.id, w.x, w.y+dy)
		}
	}
	return o.hide(false)
}

// Moved says whether the pictures are not where Show put them: moved with
// the scrollback, or hidden.
func (o *Overlay) Moved() bool { return o.dy != 0 || o.hidden }

// Hide takes the pictures away until Reset: output came while the terminal
// showed its scrollback, and where they go is not known.
func (o *Overlay) Hide() error {
	o.anchor = nil
	return o.hide(true)
}

// Reset puts the pictures back where Show put them: the terminal is at the
// bottom again (a key was typed).
func (o *Overlay) Reset() error {
	o.anchor = nil
	if o.dy != 0 {
		o.dy = 0
		for _, w := range o.wins {
			o.x.move(w.id, w.x, w.y)
		}
	}
	return o.hide(false)
}

func (o *Overlay) hide(hidden bool) error {
	if hidden == o.hidden {
		return o.x.flush()
	}
	o.hidden = hidden
	for _, w := range o.wins {
		if hidden {
			o.x.unmapWindow(w.id)
		} else {
			o.x.mapWindow(w.id)
		}
	}
	return o.x.flush()
}

// read takes sampleLines of every row of cells from the terminal's window,
// over a connection of its own.
func (o *Overlay) read() (sample, bool) {
	if o.probe == nil {
		p, err := dialX(os.Getenv("DISPLAY"))
		if err != nil {
			return sample{}, false
		}
		o.probe = p
	}
	// Only the cells are read, less the last column: a terminal may draw a
	// scrollbar over it while scrolled (kitty does), or in its padding.
	x0, w := o.padX, o.gridW-o.cellW
	if w <= 0 || o.cellH == 0 || o.gridH < o.cellH {
		return sample{}, false
	}
	n := o.gridH / o.cellH
	s := sample{rows: make([][]byte, n), skip: make([][][2]int, n), cursor: -1}
	// Every line is asked for at once and the replies read after: one wait
	// for the server instead of one a line, so a scroll is followed at once.
	for r := range n {
		for _, q := range sampleLines {
			req := o.probe.u32(o.x.term, 0, 0, 0xffffffff)
			le.PutUint16(req[4:], uint16(x0))
			le.PutUint16(req[6:], uint16(o.padY+r*o.cellH+q*o.cellH/4))
			le.PutUint16(req[8:], uint16(w))
			le.PutUint16(req[10:], 1)
			o.probe.send(opGetImage, imageFormatZPixmap, req)
		}
	}
	if o.probe.flush() != nil {
		return sample{}, false
	}
	ok := true
	for r := range n {
		for i, q := range sampleLines {
			reply, err := o.probe.reply(opGetImage)
			if err != nil || len(reply) < 32+4*w {
				ok = false // the rest are read all the same, to keep the replies in step
				continue
			}
			s.rows[r] = append(s.rows[r], reply[32:32+4*w]...)
			y := o.padY + r*o.cellH + q*o.cellH/4
			for _, win := range o.wins {
				pic := image.Rect(win.x-x0, win.y+o.dy, win.x-x0+win.key.w, win.y+o.dy+win.key.h)
				if y >= pic.Min.Y && y < pic.Max.Y {
					at := 4 * w * i
					s.skip[r] = append(s.skip[r], [2]int{at + 4*max(0, min(pic.Min.X, w)), at + 4*max(0, min(pic.Max.X, w))})
				}
			}
		}
	}
	return s, ok
}

// shiftOf finds k, the rows a's lines have moved down by in b: every row of
// a that is still on screen matches row k below it in b, where neither is
// under a picture. prefer, the shift already in force, is kept if it still
// fits; another needs two rows that are not blank to fit, and to be the only
// one that does, as blank rows fit anywhere.
func shiftOf(a, b *sample, prefer int) (int, bool) {
	var found []int
	for k := range len(a.rows) {
		fits, informative := true, 0
		for i := 0; i+k < len(b.rows) && fits; i++ {
			if i == a.cursor {
				continue
			}
			var info bool
			fits, info = sameRow(a.rows[i], b.rows[i+k], a.skip[i], b.skip[i+k])
			if info {
				informative++
			}
		}
		if fits && k == prefer {
			return k, true
		}
		if fits && informative >= 2 {
			found = append(found, k)
		}
	}
	if len(found) == 1 {
		return found[0], true
	}
	return 0, false
}

// sameRow compares two rows outside both rows' skipped ranges, and says
// whether a has anything there but one colour.
func sameRow(a, b []byte, skipA, skipB [][2]int) (same, info bool) {
	skip := slices.Concat(skipA, skipB)
	slices.SortFunc(skip, func(p, q [2]int) int { return p[0] - q[0] })
	var first []byte
	at := 0
	segment := func(from, to int) bool {
		if from >= to {
			return true
		}
		if !near(a[from:to], b[from:to]) {
			return false
		}
		for p := from; p+4 <= to && !info; p += 4 {
			if first == nil {
				first = a[p : p+4]
			} else if !near(first, a[p:p+4]) {
				info = true
			}
		}
		return true
	}
	for _, r := range skip {
		if !segment(at, min(r[0], len(a))) {
			return false, info
		}
		at = max(at, r[1])
	}
	return segment(at, len(a)), info
}

// near compares pixels a channel at a time, allowing for the rounding of a
// GPU that draws the same text a level apart on another row (kitty does).
func near(a, b []byte) bool {
	for i := range a {
		if d := int(a[i]) - int(b[i]); d > 2 || d < -2 {
			return false
		}
	}
	return true
}

// Redrawn returns a channel that receives when the terminal draws its
// window again (the DAMAGE extension), so a scroll is followed as it
// happens rather than at the next look; nil without the extension. The
// channel is closed with the overlay.
func (o *Overlay) Redrawn() <-chan struct{} {
	x, err := dialX(os.Getenv("DISPLAY"))
	if err != nil {
		return nil
	}
	name := "DAMAGE"
	q := make([]byte, 4+pad4(len(name)))
	le.PutUint16(q, uint16(len(name)))
	copy(q[4:], name)
	r, err := x.roundTrip(opQueryExtension, 0, q)
	if err != nil || r[8] == 0 {
		x.c.Close()
		return nil
	}
	major, event := r[9], r[10]
	if _, err := x.roundTrip(major, damageQueryVersion, x.u32(1, 1)); err != nil {
		x.c.Close()
		return nil
	}
	// NonEmpty: one event, then none until the damage is subtracted.
	dmg := x.id()
	x.send(major, damageCreate, append(x.u32(dmg, o.x.term), damageNonEmpty, 0, 0, 0))
	if x.flush() != nil {
		x.c.Close()
		return nil
	}
	o.damage = x
	ch := make(chan struct{}, 1)
	go func() {
		defer close(ch)
		b := make([]byte, 32)
		for {
			if _, err := io.ReadFull(x.c, b); err != nil {
				return
			}
			if b[0]&0x7f != event {
				continue
			}
			x.send(major, damageSubtract, x.u32(dmg, 0, 0))
			if x.flush() != nil {
				return
			}
			select {
			case ch <- struct{}{}:
			default: // one waiting is enough
			}
		}
	}()
	return ch
}
