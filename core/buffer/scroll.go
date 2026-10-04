package buffer

import (
	"bytes"
	"strconv"
	"unsafe"

	"github.com/thebanri/limoni/core/cell"
)

// minScrollGain is how many rows a scroll must save from being redrawn to be
// worth its escape sequences: three short CSIs, about 20 bytes, against at
// least a row's width of cells each.
const minScrollGain = 2

// scrollScratch holds the per-row hashes the scroll search compares, kept on
// the front buffer between frames so the search does not allocate.
type scrollScratch struct {
	front, back []uint64
	image       []bool // whether each front row shows part of an image
	changed     []bool // whether each row differs between the buffers
	shifts      []bool // the shifts some changed row's key allows, indexed shift+h-1
}

func (s *scrollScratch) grow(h int) {
	if cap(s.front) < h {
		s.front = make([]uint64, h)
		s.back = make([]uint64, h)
		s.image = make([]bool, h)
		s.changed = make([]bool, h)
		s.shifts = make([]bool, 2*h)
	}
	s.front, s.back, s.image, s.changed, s.shifts = s.front[:h], s.back[:h], s.image[:h], s.changed[:h], s.shifts[:2*h]
}

// keyCells is how many leading cells a row's key covers.
const keyCells = 8

// rowKey hashes a row's first cells: rows with different keys differ, so
// keys rule out most shifts before whole rows are hashed.
func rowKey(row []cell.Cell) uint64 { return rowHash(row[:min(len(row), keyCells)]) }

// cellBytes is a row's cells as their memory. A Cell is 16 bytes with no
// padding (TestCellHasNoPadding), so equal bytes are equal cells and the
// hardware's memory compare and hash can stand in for comparing fields.
func cellBytes(row []cell.Cell) []byte {
	if len(row) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&row[0])), len(row)*int(unsafe.Sizeof(row[0])))
}

func rowsEqual(a, b []cell.Cell) bool { return bytes.Equal(cellBytes(a), cellBytes(b)) }

func rowIsBlank(row []cell.Cell) bool {
	var blank cell.Cell
	blank.Reset()
	for i := range row {
		if row[i] != blank {
			return false
		}
	}
	return true
}

func rowHasImage(row []cell.Cell) bool {
	for i := range row {
		if row[i].Content == cell.RuneImage {
			return true
		}
	}
	return false
}

// scrollMatch is a run of rows that moved: front rows [start, start+n) are
// back rows shifted by shift (positive: the content moved up).
type scrollMatch struct {
	shift, start, n, gain int
}

// findScroll looks for the vertical shift that lets the terminal move the
// most rows itself instead of having them redrawn. It only counts rows that
// are not blank and not already in place, since those cost nothing to leave.
func findScroll(front, back *Buffer) (scrollMatch, bool) {
	w, h := int(front.Area.Width), int(front.Area.Height)
	if h < 3 || w == 0 {
		return scrollMatch{}, false
	}
	// Count the changed rows with a plain compare first: a frame that
	// changed fewer rows than a scroll would save cannot be a scroll, and
	// most frames are such frames.
	s := &front.scroll
	s.grow(h)
	changed := 0
	for y := 0; y < h; y++ {
		s.changed[y] = !rowsEqual(front.Content[y*w:(y+1)*w], back.Content[y*w:(y+1)*w])
		if s.changed[y] {
			changed++
		}
	}
	if changed < minScrollGain {
		return scrollMatch{}, false
	}
	// A scroll needs changed rows that are some other old row moved. Keys
	// of the first cells find out cheaply whether any are: a full redraw
	// or a table whose cells all changed has none, and skips the hashing.
	for y := 0; y < h; y++ {
		s.front[y], s.back[y] = rowKey(front.Content[y*w:(y+1)*w]), rowKey(back.Content[y*w:(y+1)*w])
	}
	// The same pass notes which shifts the matches allow, so the search
	// below tries those rather than every shift the screen could have.
	clear(s.shifts)
	movable := 0
	for y := 0; y < h; y++ {
		if !s.changed[y] {
			continue
		}
		found := false
		for src := 0; src < h; src++ {
			if src != y && s.front[y] == s.back[src] {
				s.shifts[src-y+h-1] = true
				found = true
			}
		}
		if found {
			movable++
		}
	}
	if movable < minScrollGain {
		return scrollMatch{}, false
	}
	for y := 0; y < h; y++ {
		fr := front.Content[y*w : (y+1)*w]
		s.front[y], s.back[y], s.image[y] = rowHash(fr), rowHash(back.Content[y*w:(y+1)*w]), rowHasImage(fr)
	}

	var best scrollMatch
	for shift := -(h - 1); shift <= h-1; shift++ {
		if shift == 0 || !s.shifts[shift+h-1] {
			continue
		}
		start, n, gain := 0, 0, 0
		flush := func() {
			if gain > best.gain {
				best = scrollMatch{shift: shift, start: start, n: n, gain: gain}
			}
			n, gain = 0, 0
		}
		for y := max(0, -shift); y < min(h, h-shift); y++ {
			src := y + shift
			fr := front.Content[y*w : (y+1)*w]
			if s.front[y] == s.back[src] && !s.image[y] && rowsEqual(fr, back.Content[src*w:(src+1)*w]) {
				if n == 0 {
					start = y
				}
				n++
				if !rowIsBlank(fr) && (s.front[y] != s.back[y] || !rowsEqual(fr, back.Content[y*w:(y+1)*w])) {
					gain++
				}
				continue
			}
			flush()
		}
		flush()
	}
	if best.gain < minScrollGain {
		return scrollMatch{}, false
	}
	// The terminal scrolls a region, blank rows and all: nothing in it may be
	// an image, whose pixels the diff does not own.
	top, bottom := best.region()
	for y := top; y <= bottom; y++ {
		if rowHasImage(back.Content[y*w : (y+1)*w]) {
			return scrollMatch{}, false
		}
	}
	if !worthScrolling(front, back, best) {
		return scrollMatch{}, false
	}
	return best, true
}

// scrollCost is what scrolling a band costs before any cell is written: the
// margins, the scroll and the reset, about twenty bytes.
const scrollCost = 20

// worthScrolling compares the cells the diff would write over the band with
// and without the scroll. Counting matched rows is not enough: a selection bar
// moving down a table matches many rows shifted by one, but in place each of
// them differs in a few cells only, and scrolling the band would make the
// diff rewrite everything the scroll moved under the bar. That sent 339 bytes
// a frame where the plain diff sent 298.
func worthScrolling(front, back *Buffer, m scrollMatch) bool {
	w := int(front.Area.Width)
	top, bottom := m.region()
	k := m.shift
	if k < 0 {
		k = -k
	}
	var blank cell.Cell
	blank.Reset()
	without, with := 0, scrollCost
	for y := top; y <= bottom; y++ {
		fr := front.Content[y*w : (y+1)*w]
		without += cellsDiffer(fr, back.Content[y*w:(y+1)*w])
		// Where row y's text comes from once the band has scrolled.
		src := y + m.shift
		if src < top || src > bottom {
			for i := range fr {
				if fr[i] != blank {
					with++
				}
			}
		} else {
			with += cellsDiffer(fr, back.Content[src*w:(src+1)*w])
		}
		if with >= without+w*(bottom-y) {
			// Even if every row left matched, scrolling cannot win.
			return false
		}
	}
	return with < without
}

// cellsDiffer counts the cells that differ between two rows, comparing each
// 16-byte cell as two machine words (see cellBytes on why that is exact).
func cellsDiffer(a, b []cell.Cell) int {
	if rowsEqual(a, b) {
		return 0
	}
	wa, wb := cellWords(a), cellWords(b)
	n := 0
	for i := 0; i < len(wa); i += 2 {
		if wa[i] != wb[i] || wa[i+1] != wb[i+1] {
			n++
		}
	}
	return n
}

// cellWords is a row's cells as pairs of 64-bit words.
func cellWords(row []cell.Cell) []uint64 {
	if len(row) == 0 {
		return nil
	}
	return unsafe.Slice((*uint64)(unsafe.Pointer(&row[0])), len(row)*2)
}

// region is the band of screen rows the terminal scrolls.
func (m scrollMatch) region() (top, bottom int) {
	if m.shift > 0 {
		return m.start, m.start + m.n - 1 + m.shift
	}
	return m.start + m.shift, m.start + m.n - 1
}

// applyScroll makes the terminal scroll the matched band and moves back's
// rows to where the terminal now has them, so the diff that follows only
// writes what is left. The band is scrolled with DECSTBM and SU or SD; the
// rows it opens are blank in the current background. Every frame ends in the
// default style, so it already is the default here; the reset costs four
// bytes and keeps that true if the frame's preamble ever changes.
func applyScroll(out []byte, back *Buffer, m scrollMatch) []byte {
	w := int(back.Area.Width)
	top, bottom := m.region()
	k := m.shift
	if k < 0 {
		k = -k
	}

	out = append(out, "\x1b[0m\x1b["...)
	out = strconv.AppendInt(out, int64(top+1), 10)
	out = append(out, ';')
	out = strconv.AppendInt(out, int64(bottom+1), 10)
	out = append(out, "r\x1b["...)
	out = strconv.AppendInt(out, int64(k), 10)
	if m.shift > 0 {
		out = append(out, 'S')
	} else {
		out = append(out, 'T')
	}
	// Reset the margins. DECSTBM also homes the cursor; the diff that
	// follows positions it before every write anyway.
	out = append(out, "\x1b[r"...)

	var blank cell.Cell
	blank.Reset()
	rows := back.Content
	if m.shift > 0 {
		copy(rows[top*w:(bottom+1-k)*w], rows[(top+k)*w:(bottom+1)*w])
		for i := (bottom + 1 - k) * w; i < (bottom+1)*w; i++ {
			rows[i] = blank
		}
	} else {
		copy(rows[(top+k)*w:(bottom+1)*w], rows[top*w:(bottom+1-k)*w])
		for i := top * w; i < (top+k)*w; i++ {
			rows[i] = blank
		}
	}
	back.clean = false
	return out
}

// maxLineShift is the largest insertion or deletion the diff looks for
// within a row: typing and deleting move text by a character or a word.
const maxLineShift = 8

// maxShiftSkip is how many cells right after a deletion may differ from the
// shifted row and still be worth a DCH: the cursor's cell, typically.
const maxShiftSkip = 2

// minLineShiftGain is how many cells an ICH or DCH must save from being
// rewritten to pay for itself.
const minLineShiftGain = 6

// narrowCell reports whether a cell occupies exactly one column as one code
// point: shifting a row by columns is only safe over such cells.
func narrowCell(c *cell.Cell) bool {
	r := c.Content
	return r != cell.RuneContinuation && r != cell.RuneImage && !cell.IsCluster(r) && cell.RuneWidth(r) == 1
}

// findLineShift checks whether row y of front is row y of back with k cells
// inserted (k > 0) or deleted (k < 0) at column first, the tail of the row
// moving to make room, as ICH and DCH move it. It returns k and how many
// cells that saves, or 0.
func findLineShift(front, back *Buffer, y, first, last int) (k, gain int) {
	w := int(front.Area.Width)
	f := front.Content[y*w : (y+1)*w]
	b := back.Content[y*w : (y+1)*w]
	if last-first+1 < minLineShiftGain {
		return 0, 0 // too few changed cells for any shift to pay for itself
	}
	var blank cell.Cell
	blank.Reset()
	span := last - first + 1
	best, bestGain := 0, 0
	for s := 1; s <= maxLineShift && s < w-first; s++ {
		// Insert s: every cell from first+s on is the old one s to its left.
		ok := true
		for x := first + s; x < w && ok; x++ {
			ok = f[x] == b[x-s]
		}
		// The s inserted cells are still written; everything else is saved.
		if g := span - s; ok && g >= minLineShiftGain && g > bestGain {
			best, bestGain = s, g
		}
		// Delete s: every cell from first on is the old one s to its right.
		// DCH opens s blank columns at the right edge; whatever belongs
		// there instead (text scrolled in, as in a long input) is written
		// by the diff that follows, so it only costs the gain.
		// The first cell or two after the deletion may differ anyway — the
		// cursor stays put and highlights a new character — and are
		// rewritten; the shift only has to hold after them.
		skip := -1
		for j := 0; j <= maxShiftSkip && skip < 0; j++ {
			ok = true
			for x := first + j; x < w-s && ok; x++ {
				ok = f[x] == b[x+s]
			}
			if ok {
				skip = j
			}
		}
		if skip >= 0 {
			g := span - skip
			for x := w - s; x < w; x++ {
				if f[x] != blank {
					g--
				}
			}
			if g >= minLineShiftGain && g > bestGain {
				best, bestGain = -s, g
			}
		}
	}
	// Shifting by columns is only safe over cells one column wide. Checked
	// last: most changed rows match no shift, and the check reads every
	// cell's width.
	if best != 0 {
		for x := first; x < w; x++ {
			if !narrowCell(&f[x]) || !narrowCell(&b[x]) {
				return 0, 0
			}
		}
	}
	return best, bestGain
}

// applyLineShift emits ICH or DCH at (first, y) and shifts back's row the
// same way, so the ordinary diff only writes the inserted cells. The cells
// the terminal opens take the current background, so the style must be the
// default; the caller resets it.
func applyLineShift(out []byte, back *Buffer, y, first, k int) []byte {
	w := int(back.Area.Width)
	row := back.Content[y*w : (y+1)*w]
	var blank cell.Cell
	blank.Reset()
	out = appendCursor(out, uint16(first), uint16(y))
	out = append(out, "\x1b["...)
	if k > 0 {
		out = strconv.AppendInt(out, int64(k), 10)
		out = append(out, '@')
		copy(row[first+k:], row[first:w-k])
		for x := first; x < first+k; x++ {
			row[x] = blank
		}
	} else {
		out = strconv.AppendInt(out, int64(-k), 10)
		out = append(out, 'P')
		copy(row[first:], row[first-k:])
		for x := w + k; x < w; x++ {
			row[x] = blank
		}
	}
	back.clean = false
	return out
}
