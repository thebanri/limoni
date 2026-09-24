package buffer

import (
	"bytes"
	"strconv"

	"github.com/thebanri/limoni/core/cell"
)

// AdaptiveDiffThreshold defines the dirty ratio threshold (45%) at which Diff transitions
// from sparse differential rendering with cursor jumps (Case A) to continuous full stream redraw (Case B).
const AdaptiveDiffThreshold = 0.45

// Diff compares the front (currently drawn) and back (currently displayed) buffers.
// It implements an adaptive rendering strategy:
// - Case A (dirtyRatio < 0.45): Sparse differential rendering with minimal cursor jumps (CUP).
// - Case B (dirtyRatio >= 0.45): Continuous full stream redraw with synchronized update mode (?2026).
// Performance: Operates with zero heap allocations (0 B/op) when out has sufficient capacity.
// DiffOptions selects the encodings the diff is allowed to emit.
//
// Emitted bytes, not CPU time, govern responsiveness over a network link, so
// the encoder can compress runs — but only with sequences the terminal
// actually implements. A terminal without REP prints the escape instead of
// obeying it, which corrupts the frame, so these are opt-in per capability
// rather than assumed.
type DiffOptions struct {
	TrueColor bool
	Colors256 bool
	// EraseChar allows ECH (CSI n X) for runs of blanks.
	EraseChar bool
	// RepeatChar allows REP (CSI n b) for runs of one glyph.
	RepeatChar bool
	// SyncOutput wraps the frame in synchronized update mode (?2026) so the
	// terminal presents it atomically instead of tearing.
	SyncOutput bool
	// Hyperlinks allows OSC 8, which makes a styled span clickable. A
	// terminal that does not implement OSC 8 should ignore it, but not all of
	// them do, so it is off unless the terminal is recognised.
	Hyperlinks bool
	// ClusterWidths says the terminal confirmed mode 2027: it advances the
	// cursor by a grapheme cluster's width, as the buffer does. The encoder
	// then trusts the cursor after a cluster instead of re-anchoring it.
	// Leave it false unless the terminal answered DECRQM for 2027 as set.
	ClusterWidths bool
	// ScrollRegions lets the diff move rows that scrolled with DECSTBM and
	// SU/SD, so a list or log that moved by a line costs a few bytes instead
	// of a redraw of every row. Not for inline mode, whose rows are relative
	// to a cursor that the scroll region would move.
	ScrollRegions bool
	// InsertDelete lets the diff shift the rest of a row with ICH or DCH
	// (VT220) when text was inserted or deleted in it, as typing in the
	// middle of a line does, instead of rewriting the tail.
	InsertDelete bool
}

// minEraseRun and minRepeatRun are the lengths at which a control sequence
// becomes shorter than the literal cells it replaces. "CSI n X" is four bytes
// at a single digit, so a run of four blanks breaks even and five wins.
const (
	minEraseRun  = 5
	minRepeatRun = 5
)

// Diff compares the buffers with the default encodings. See DiffWithOptions
// for control over run compression.
func Diff(front, back *Buffer, out []byte, trueColor, colors256 bool) ([]byte, error) {
	return DiffWithOptions(front, back, out, DiffOptions{
		TrueColor: trueColor,
		Colors256: colors256,
		EraseChar: true,
	})
}

// DiffWithOptions compares the buffers and appends the escape sequence stream.
func DiffWithOptions(front, back *Buffer, out []byte, opts DiffOptions) ([]byte, error) {
	return diff(front, back, out, opts)
}

func diff(front, back *Buffer, out []byte, opts DiffOptions) ([]byte, error) {
	// Zero-Loop Fast-Path: Return immediately if buffer was not dirtied and dimensions match
	if !front.IsDirty && front.Area.Width == back.Area.Width && front.Area.Height == back.Area.Height {
		return out, nil
	}

	// If dimensions mismatch, resize back buffer and execute full stream redraw
	if front.Area.Width != back.Area.Width || front.Area.Height != back.Area.Height {
		back.Resize(front.Area)
		return diffFullStream(front, back, out, opts)
	}

	width := front.Area.Width
	height := front.Area.Height
	totalCells := int(width) * int(height)
	if totalCells == 0 {
		return out, nil
	}

	if opts.ScrollRegions {
		if m, ok := findScroll(front, back); ok {
			out = applyScroll(out, back, m)
		}
	}

	// Count modified cells
	dirtyCount := 0
	for i := 0; i < totalCells; i++ {
		if front.Content[i] != back.Content[i] {
			dirtyCount++
		}
	}

	// Fast-Path: completely identical
	if dirtyCount == 0 {
		front.IsDirty = false
		return out, nil
	}

	dirtyRatio := float64(dirtyCount) / float64(totalCells)
	if dirtyRatio >= AdaptiveDiffThreshold {
		return diffFullStream(front, back, out, opts)
	}

	return diffSparse(front, back, out, opts)
}

// DiffSparse executes Case A: sparse differential rendering using cursor jumps (CUP)
// for only modified spans within lines. Used when dirtyRatio < 0.45.
func DiffSparse(front, back *Buffer, out []byte, trueColor, colors256 bool) ([]byte, error) {
	return diffSparse(front, back, out, DiffOptions{TrueColor: trueColor, Colors256: colors256, EraseChar: true})
}

func diffSparse(front, back *Buffer, out []byte, opts DiffOptions) ([]byte, error) {
	trueColor, colors256, links := opts.TrueColor, opts.Colors256, opts.Hyperlinks
	width := front.Area.Width
	height := front.Area.Height

	var currentStyle cell.Style
	currentStyle.Reset()

	cursorX := uint16(9999)
	cursorY := uint16(9999)

	for y := uint16(0); y < height; y++ {
		first := int(-1)
		last := int(-1)
		rowOffset := int(y) * int(width)
		for x := 0; x < int(width); x++ {
			idx := rowOffset + x
			if front.Content[idx] != back.Content[idx] {
				start := x
				end := x
				if x > 0 && (front.Content[idx].Content == cell.RuneContinuation || back.Content[idx].Content == cell.RuneContinuation) {
					start = x - 1
					back.Content[rowOffset+x-1] = cell.Cell{}
				}
				wFront := cell.RuneWidth(front.Content[idx].Content)
				wBack := cell.RuneWidth(back.Content[idx].Content)
				if (wFront == 2 || wBack == 2) && x+1 < int(width) {
					end = x + 1
				}
				if first == -1 || start < first {
					first = start
				}
				if end > last {
					last = end
				}
			}
		}
		if first == -1 {
			continue // No changes on this line
		}

		if opts.InsertDelete {
			if k, _ := findLineShift(front, back, int(y), first, last); k != 0 {
				var def cell.Style
				def.Reset()
				if currentStyle != def {
					// ICH and DCH open cells in the current background.
					out, currentStyle = appendStyle(out, currentStyle, def, trueColor, colors256, links, front.StyleCache)
				}
				out = applyLineShift(out, back, int(y), first, k)
				cursorX, cursorY = uint16(first), y // neither moves the cursor
			}
		}

		for x := uint16(first); x <= uint16(last); x++ {
			idx := int(y)*int(width) + int(x)
			frontCell := &front.Content[idx]
			backCell := &back.Content[idx]

			if frontCell.Content == backCell.Content && frontCell.Style == backCell.Style {
				continue
			}

			if frontCell.Content == cell.RuneContinuation {
				*backCell = *frontCell
				cursorX = 9999
				cursorY = 9999
				continue
			}

			if frontCell.Content == cell.RuneImage {
				spanEnd := x
				needsErase := false
				for checkX := x; checkX <= uint16(last); checkX++ {
					cIdx := int(y)*int(width) + int(checkX)
					if front.Content[cIdx].Content != cell.RuneImage {
						break
					}
					spanEnd = checkX
					prevContent := back.Content[cIdx].Content
					if prevContent != cell.RuneImage {
						needsErase = true
					}
				}

				count := int(spanEnd - x + 1)
				if needsErase {
					out = appendCursor(out, x, y)
					cursorX = 9999
					cursorY = 9999
					if currentStyle != (cell.Style{}) {
						out = append(out, "\x1b[0m"...)
						currentStyle.Reset()
					}
					out = append(out, "\x1b["...)
					out = strconv.AppendInt(out, int64(count), 10)
					out = append(out, 'X')
				} else {
					cursorX = 9999
					cursorY = 9999
				}

				for setX := x; setX <= spanEnd; setX++ {
					cIdx := int(y)*int(width) + int(setX)
					back.Content[cIdx] = front.Content[cIdx]
				}

				x = spanEnd
				continue
			}

			// Run compression. A row of identical cells is common — padding,
			// cleared regions, rules, fills — and writing it literally is the
			// single largest avoidable cost in the emitted stream.
			if opts.EraseChar || opts.RepeatChar {
				run := uint16(1)
				for nx := x + 1; nx <= uint16(last); nx++ {
					nIdx := int(y)*int(width) + int(nx)
					if front.Content[nIdx] != *frontCell {
						break
					}
					// Only cells that actually need writing may be folded into
					// a run; stopping at a clean cell keeps the diff minimal.
					if front.Content[nIdx] == back.Content[nIdx] {
						break
					}
					run++
				}

				isBlank := frontCell.Content == ' ' || frontCell.Content == 0
				glyphWidth := cell.RuneWidth(frontCell.Content)

				switch {
				case opts.EraseChar && isBlank && run >= minEraseRun:
					if cursorX != x || cursorY != y {
						out = appendCursor(out, x, y)
					}
					if frontCell.Style != currentStyle {
						out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, links, front.StyleCache)
					}
					out = append(out, "\x1b["...)
					out = strconv.AppendInt(out, int64(run), 10)
					out = append(out, 'X')
					// ECH erases in place and leaves the cursor where it was,
					// so the next write has to reposition.
					cursorX, cursorY = 9999, 9999
					for i := uint16(0); i < run; i++ {
						back.Content[int(y)*int(width)+int(x+i)] = *frontCell
					}
					x += run - 1
					continue

				case opts.RepeatChar && !isBlank && !cell.IsCluster(frontCell.Content) && glyphWidth == 1 && run >= minRepeatRun:
					if cursorX != x || cursorY != y {
						out = appendCursor(out, x, y)
					}
					if frontCell.Style != currentStyle {
						out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, links, front.StyleCache)
					}
					out = cell.AppendContent(out, frontCell.Content)
					out = append(out, "\x1b["...)
					out = strconv.AppendInt(out, int64(run-1), 10)
					out = append(out, 'b')
					cursorX, cursorY = x+run, y
					if cursorX >= width {
						cursorX, cursorY = 9999, 9999
					}
					for i := uint16(0); i < run; i++ {
						back.Content[int(y)*int(width)+int(x+i)] = *frontCell
					}
					x += run - 1
					continue
				}
			}

			if cursorX != x || cursorY != y {
				out = appendCursor(out, x, y)
				cursorX = x
				cursorY = y
			}

			if frontCell.Style != currentStyle {
				out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, links, front.StyleCache)
			}

			w := 1
			if frontCell.Content == ' ' || frontCell.Content == 0 || frontCell.Content < 32 || frontCell.Content == 0x7F {
				out = append(out, ' ')
			} else {
				out = cell.AppendContent(out, frontCell.Content)
				w = cell.RuneWidth(frontCell.Content)
				if w <= 0 {
					w = 1
				}
			}

			cursorX += uint16(w)
			if cursorX >= width || (cell.IsCluster(frontCell.Content) && !opts.ClusterWidths) {
				// A terminal without mode 2027 may advance by a different
				// amount for a cluster, so its cursor position is unknown
				// and the next write addresses its cell explicitly.
				cursorX = 9999
				cursorY = 9999
			}

			*backCell = *frontCell

			if w == 2 && x+1 < width {
				back.Content[idx+1] = front.Content[idx+1]
			}
		}
	}

	var defaultStyle cell.Style
	defaultStyle.Reset()
	if currentStyle != defaultStyle {
		out, _ = appendStyle(out, currentStyle, defaultStyle, trueColor, colors256, links, front.StyleCache)
	}

	front.IsDirty = false
	return out, nil
}

// isBlankCell reports whether a cell renders as a space, which is what makes
// it a candidate for erasure rather than a literal write.
func isBlankCell(c *cell.Cell) bool {
	return c.Content == ' ' || c.Content == 0 || c.Content < 32 || c.Content == 0x7F
}

// DiffFullStream executes Case B: continuous stream redraw for high-churn frames (dirtyRatio >= 0.45).
// It bypasses cursor jump calculations entirely, wraps the output in synchronized update mode (?2026),
// moves cursor to home (\x1b[H), sequentially overwrites line-by-line using \r\n,
// maintains lazy SGR color emission, and synchronizes buffers via copy(back.Content, front.Content).
func DiffFullStream(front, back *Buffer, out []byte, trueColor, colors256 bool) ([]byte, error) {
	return diffFullStream(front, back, out, DiffOptions{TrueColor: trueColor, Colors256: colors256, EraseChar: true})
}

func diffFullStream(front, back *Buffer, out []byte, opts DiffOptions) ([]byte, error) {
	trueColor, colors256, links := opts.TrueColor, opts.Colors256, opts.Hyperlinks
	if front.Area.Width != back.Area.Width || front.Area.Height != back.Area.Height {
		back.Resize(front.Area)
	}

	width := front.Area.Width
	height := front.Area.Height

	// Check if synchronized update was already opened by caller in batch buffer
	syncWrapped := false
	if !bytes.HasPrefix(out, []byte("\x1b[?2026h")) {
		out = append(out, "\x1b[?2026h"...)
		syncWrapped = true
	}

	// Move cursor to home (1, 1) without clearing screen to prevent flicker
	out = append(out, "\x1b[H"...)

	var currentStyle cell.Style
	currentStyle.Reset()

	for y := uint16(0); y < height; y++ {
		if y > 0 {
			out = append(out, "\r\n"...)
		}
		rowOffset := int(y) * int(width)
		for x := uint16(0); x < width; x++ {
			idx := rowOffset + int(x)
			frontCell := &front.Content[idx]

			// Skip continuation cells (second column of wide character)
			if frontCell.Content == cell.RuneContinuation {
				continue
			}

			// Handle native image cells
			if frontCell.Content == cell.RuneImage {
				spanEnd := x
				needsErase := false
				for checkX := x; checkX < width; checkX++ {
					cIdx := rowOffset + int(checkX)
					if front.Content[cIdx].Content != cell.RuneImage {
						break
					}
					spanEnd = checkX
					if back.Content[cIdx].Content != cell.RuneImage {
						needsErase = true
					}
				}

				count := int(spanEnd - x + 1)
				if needsErase {
					if currentStyle != (cell.Style{}) {
						out = append(out, "\x1b[0m"...)
						currentStyle.Reset()
					}
					out = append(out, "\x1b["...)
					out = strconv.AppendInt(out, int64(count), 10)
					out = append(out, 'X')
				}

				out = append(out, "\x1b["...)
				out = strconv.AppendInt(out, int64(count), 10)
				out = append(out, 'C')
				x = spanEnd
				continue
			}

			// Erase to end of line. Most of a typical frame is padding, and
			// three bytes replace the whole tail of the row. EL erases with
			// the current background, so it is only safe when every remaining
			// cell shares one style.
			if opts.EraseChar && isBlankCell(frontCell) {
				tailStyle := frontCell.Style
				tailBlank := true
				for checkX := x + 1; checkX < width; checkX++ {
					c := &front.Content[rowOffset+int(checkX)]
					if !isBlankCell(c) || c.Style != tailStyle {
						tailBlank = false
						break
					}
				}
				if tailBlank && width-x >= minEraseRun {
					if tailStyle != currentStyle {
						out, currentStyle = appendStyle(out, currentStyle, tailStyle, trueColor, colors256, links, front.StyleCache)
					}
					out = append(out, "\x1b[K"...)
					break
				}
			}

			// Repeat a run of one glyph. REP advances the cursor exactly as
			// writing the glyph that many times would, so the sequential
			// stream stays aligned.
			if opts.RepeatChar && !isBlankCell(frontCell) && !cell.IsCluster(frontCell.Content) && cell.RuneWidth(frontCell.Content) == 1 {
				run := uint16(1)
				for nx := x + 1; nx < width; nx++ {
					if front.Content[rowOffset+int(nx)] != *frontCell {
						break
					}
					run++
				}
				if run >= minRepeatRun {
					if frontCell.Style != currentStyle {
						out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, links, front.StyleCache)
					}
					out = cell.AppendContent(out, frontCell.Content)
					out = append(out, "\x1b["...)
					out = strconv.AppendInt(out, int64(run-1), 10)
					out = append(out, 'b')
					x += run - 1
					continue
				}
			}

			// Lazy SGR style emission: only emit escape sequences when style changes
			if frontCell.Style != currentStyle {
				out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, links, front.StyleCache)
			}

			// Emit character rune
			if isBlankCell(frontCell) {
				out = append(out, ' ')
			} else {
				out = cell.AppendContent(out, frontCell.Content)
				if cell.IsCluster(frontCell.Content) && !opts.ClusterWidths {
					out = appendClusterResync(out, frontCell.Content, x, width)
				}
			}
		}
	}

	// Reset style at frame end to prevent terminal style bleeding
	var defaultStyle cell.Style
	defaultStyle.Reset()
	if currentStyle != defaultStyle {
		out, _ = appendStyle(out, currentStyle, defaultStyle, trueColor, colors256, links, front.StyleCache)
	}

	// Close synchronized update if this function opened it
	if syncWrapped {
		out = append(out, "\x1b[?2026l"...)
	}

	// Synchronize buffers
	copy(back.Content, front.Content)
	front.IsDirty = false

	return out, nil
}

// AppendCursor appends cursor positioning escape sequence (\x1b[row;colH) to out.
func AppendCursor(out []byte, x, y uint16) []byte {
	out = append(out, "\x1b["...)
	out = appendUint16(out, y+1)
	out = append(out, ';')
	out = appendUint16(out, x+1)
	return append(out, 'H')
}

// appendCursor is an internal alias for AppendCursor.
// appendClusterResync moves the cursor to the column after a cluster just
// written at x.
//
// Terminals that implement mode 2027 advance by the cluster's width, as the
// buffer does. Many do not: they advance per code point, so a family emoji
// moves the cursor six columns and a flag four, and in a sequential stream
// every later cell on the row would land shifted. CHA names the column
// outright, which confines the disagreement to the cluster itself. It costs a
// few bytes per cluster and nothing for text without them.
func appendClusterResync(out []byte, content rune, x, width uint16) []byte {
	next := x + uint16(cell.RuneWidth(content))
	if next >= width {
		return out // The row ends here; the next row starts with \r.
	}
	out = append(out, "\x1b["...)
	out = strconv.AppendInt(out, int64(next)+1, 10)
	return append(out, 'G')
}

func appendCursor(out []byte, x, y uint16) []byte {
	return AppendCursor(out, x, y)
}

func getStyleBytes(target cell.Style, trueColor, colors256 bool, cache map[cell.Style][]byte) []byte {
	// Links are emitted by appendStyle, not cached: one entry per style is
	// worth keeping, one per style-and-URL is not.
	target.Link = 0
	if cache == nil {
		var out []byte
		var cur cell.Style
		cur.Reset()
		out, _ = appendStyleRaw(out, cur, target, trueColor, colors256)
		return out
	}
	if bytes, ok := cache[target]; ok {
		return bytes
	}

	// Format style starting from default
	var out []byte
	var cur cell.Style
	cur.Reset()
	out, _ = appendStyleRaw(out, cur, target, trueColor, colors256)

	if len(cache) > 2048 {
		clear(cache)
	}
	cache[target] = out
	return out
}

func appendStyle(out []byte, cur, target cell.Style, trueColor, colors256, links bool, cache map[cell.Style][]byte) ([]byte, cell.Style) {
	if !trueColor {
		target = target.Downsample(trueColor, colors256)
	}
	if cur == target {
		return out, cur
	}

	// A hyperlink is not SGR: `ESC[0m` does not close it, and the cached
	// style bytes below are built without it. So it is emitted here, once per
	// change, and then the two styles agree on the link for the SGR work.
	if cur.Link != target.Link {
		if links {
			out = appendLink(out, target.Link)
		}
		cur.Link = target.Link
		if cur == target {
			return out, cur
		}
	}

	// If we are resetting anyway, we can use the cached target bytes directly!
	if (cur.Modifier & ^target.Modifier) != 0 {
		out = append(out, "\x1b[0m"...)
		cached := getStyleBytes(target, trueColor, colors256, cache)
		out = append(out, cached...)
		return out, target
	}

	// Otherwise, do standard incremental diff
	return appendStyleRaw(out, cur, target, trueColor, colors256)
}

// appendStyleRaw analyzes style changes and emits ANSI codes for modified properties only.
func appendStyleRaw(out []byte, cur, target cell.Style, trueColor, colors256 bool) ([]byte, cell.Style) {
	if cur == target {
		return out, cur
	}

	// 1. Modifiers removed
	if (cur.Modifier & ^target.Modifier) != 0 {
		out = append(out, "\x1b[0m"...)
		link := cur.Link // SGR reset does not close a hyperlink.
		cur.Reset()
		cur.Link = link
	}

	// 2. Foreground color change
	if cur.Fg != target.Fg {
		switch target.Fg.Type() {
		case cell.ColorDefault:
			out = append(out, "\x1b[39m"...)
		case cell.ColorANSI:
			out = append(out, "\x1b[38;5;"...)
			out = appendUint8(out, target.Fg.ANSI())
			out = append(out, 'm')
		case cell.ColorRGB:
			r, g, b := target.Fg.RGB()
			out = append(out, "\x1b[38;2;"...)
			out = appendUint8(out, r)
			out = append(out, ';')
			out = appendUint8(out, g)
			out = append(out, ';')
			out = appendUint8(out, b)
			out = append(out, 'm')
		}
		cur.Fg = target.Fg
	}

	// 3. Background color change
	if cur.Bg != target.Bg {
		switch target.Bg.Type() {
		case cell.ColorDefault:
			out = append(out, "\x1b[49m"...)
		case cell.ColorANSI:
			out = append(out, "\x1b[48;5;"...)
			out = appendUint8(out, target.Bg.ANSI())
			out = append(out, 'm')
		case cell.ColorRGB:
			r, g, b := target.Bg.RGB()
			out = append(out, "\x1b[48;2;"...)
			out = appendUint8(out, r)
			out = append(out, ';')
			out = appendUint8(out, g)
			out = append(out, ';')
			out = appendUint8(out, b)
			out = append(out, 'm')
		}
		cur.Bg = target.Bg
	}

	// 4. Added modifiers
	added := target.Modifier & ^cur.Modifier
	if added != 0 {
		out = append(out, "\x1b["...)
		first := true

		if (added & cell.ModifierBold) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '1')
			first = false
		}
		if (added & cell.ModifierDim) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '2')
			first = false
		}
		if (added & cell.ModifierItalic) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '3')
			first = false
		}
		if (added & cell.ModifierUnderline) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '4')
			first = false
		}
		if (added & cell.ModifierBlink) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '5')
			first = false
		}
		if (added & cell.ModifierReverse) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '7')
			first = false
		}
		if (added & cell.ModifierHidden) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '8')
			first = false
		}
		if (added & cell.ModifierStrikethrough) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, '9')
			first = false
		}
		if (added & cell.ModifierDoubleUnderline) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, "21"...)
			first = false
		}
		if (added & cell.ModifierUndercurl) != 0 {
			if !first {
				out = append(out, ';')
			}
			out = append(out, "4:3"...)
			first = false
		}

		out = append(out, 'm')
		cur.Modifier |= added
	}

	return out, cur
}

// appendLink switches the hyperlink the following cells belong to, with
// OSC 8. The id parameter matters: the diff emits cells in the order it finds
// them, so one link's text can arrive in several pieces and on several rows,
// and terminals use the id to treat those pieces as one link when the pointer
// hovers over them. The handle serves as that id, which is exactly its job.
//
// The zero handle closes the link, as OSC 8 requires, with both fields empty.
func appendLink(out []byte, id cell.LinkID) []byte {
	out = append(out, "\x1b]8;"...)
	if id != 0 {
		out = append(out, "id="...)
		out = appendUint16(out, uint16(id))
	}
	out = append(out, ';')
	if id != 0 {
		out = append(out, cell.LinkURL(id)...)
	}
	// ST rather than BEL: BEL inside OSC 8 is accepted but deprecated, and a
	// terminal that does not know the sequence skips more reliably to ST.
	return append(out, "\x1b\\"...)
}

func appendUint8(out []byte, v uint8) []byte {
	if v >= 100 {
		return append(out, '0'+v/100, '0'+(v/10)%10, '0'+v%10)
	}
	if v >= 10 {
		return append(out, '0'+v/10, '0'+v%10)
	}
	return append(out, '0'+v)
}

func appendUint16(out []byte, v uint16) []byte {
	if v >= 10000 {
		return append(out, '0'+byte(v/10000), '0'+byte((v/1000)%10), '0'+byte((v/100)%10), '0'+byte((v/10)%10), '0'+byte(v%10))
	}
	if v >= 1000 {
		return append(out, '0'+byte(v/1000), '0'+byte((v/100)%10), '0'+byte((v/10)%10), '0'+byte(v%10))
	}
	if v >= 100 {
		return append(out, '0'+byte(v/100), '0'+byte((v/10)%10), '0'+byte(v%10))
	}
	if v >= 10 {
		return append(out, '0'+byte(v/10), '0'+byte(v%10))
	}
	return append(out, '0'+byte(v))
}
