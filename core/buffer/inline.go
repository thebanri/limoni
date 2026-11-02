package buffer

import (
	"strconv"

	"github.com/thebanri/limoni/core/cell"
)

// DiffInline renders the frame in place in the normal screen buffer, without
// the alternate screen.
//
// This is the mode `gum`, CI progress renderers and shell prompts are built on:
// the application occupies a few rows where the cursor already is, the
// scrollback above it is untouched, and what it drew is still on screen after
// it exits.
//
// It cannot use the cursor-addressed sparse path. Absolute positioning needs to
// know which terminal row the application starts on, and that row moves
// whenever the terminal scrolls — the shell printing a line, another process
// writing to the same tty. So every frame is emitted relative to the cursor:
// the caller guarantees the cursor sits at the application's first row and
// column zero, and this leaves it there again.
//
// Each row is cleared with EL rather than padded with spaces, so a row that
// shrank does not leave the previous frame's tail behind it.
func DiffInline(front, back *Buffer, out []byte, opts DiffOptions) ([]byte, error) {
	return diffInline(front, back, out, opts, false)
}

// DiffInlineParked is DiffInline that leaves the cursor at the end of the
// band's last row instead of on its first. A terminal that reflows on a width
// change moves a cursor sitting at column zero unpredictably, but keeps one
// that follows the end of the text; the caller moves up again with
// InlineVisualRows before the next frame. See InlineExtents.
func DiffInlineParked(front, back *Buffer, out []byte, opts DiffOptions) ([]byte, error) {
	return diffInline(front, back, out, opts, true)
}

func diffInline(front, back *Buffer, out []byte, opts DiffOptions, park bool) ([]byte, error) {
	if front.Area.Width != back.Area.Width || front.Area.Height != back.Area.Height {
		back.Resize(front.Area)
	}

	width := front.Area.Width
	height := front.Area.Height
	if width == 0 || height == 0 {
		return out, nil
	}

	syncWrapped := false
	if opts.SyncOutput {
		out = append(out, "\x1b[?2026h"...)
		syncWrapped = true
	}

	if opts.InlineRows > 0 && opts.InlineRows < height {
		height = opts.InlineRows
	}

	var currentStyle cell.Style
	currentStyle.Reset()

	// Column zero of the first row. The caller positioned the cursor on the
	// right row; \r makes the column unambiguous.
	out = append(out, '\r')

	for y := uint16(0); y < height; y++ {
		if y > 0 {
			out = append(out, "\r\n"...)
		}
		out, currentStyle = appendInlineRow(out, front, y, currentStyle, opts)
	}

	// Return to the first row so the next frame starts where this one did.
	if park {
		// Stay at the end of the last row (see DiffInlineParked).
	} else if height > 1 {
		out = append(out, "\x1b["...)
		out = strconv.AppendInt(out, int64(height-1), 10)
		out = append(out, 'A')
	}
	if !park {
		out = append(out, '\r')
	}

	if syncWrapped {
		out = append(out, "\x1b[?2026l"...)
	}

	copy(back.Content, front.Content)
	front.IsDirty = false
	return out, nil
}

// AppendRows writes every row of b at the cursor, each followed by a newline,
// so the rows scroll up into the scrollback as a program's output would. It
// is how terminal.Terminal.Commit prints above an inline band.
func AppendRows(out []byte, b *Buffer, opts DiffOptions) []byte {
	var currentStyle cell.Style
	currentStyle.Reset()
	out = append(out, '\r')
	for y := uint16(0); y < b.Area.Height; y++ {
		out, currentStyle = appendInlineRow(out, b, y, currentStyle, opts)
		out = append(out, "\r\n"...)
	}
	return out
}

// appendInlineRow writes row y from column zero and erases what an earlier
// frame left to the right of it. It returns the style in effect afterwards.
func appendInlineRow(out []byte, front *Buffer, y uint16, currentStyle cell.Style, opts DiffOptions) ([]byte, cell.Style) {
	width := front.Area.Width
	rowOffset := int(y) * int(width)

	lastPainted := -1
	for x := uint16(0); x < width; x++ {
		c := &front.Content[rowOffset+int(x)]
		if c.Content == cell.RuneContinuation {
			continue
		}
		if !isBlankCell(c) || c.Style != (cell.Style{}) {
			lastPainted = int(x)
		}
	}

	for x := uint16(0); x <= uint16(lastPainted) && lastPainted >= 0; x++ {
		c := &front.Content[rowOffset+int(x)]
		if c.Content == cell.RuneContinuation {
			continue
		}

		if opts.RepeatChar && !isBlankCell(c) && !cell.IsCluster(c.Content) && cell.RuneWidth(c.Content) == 1 {
			run := uint16(1)
			for nx := x + 1; nx <= uint16(lastPainted); nx++ {
				if front.Content[rowOffset+int(nx)] != *c {
					break
				}
				run++
			}
			if run >= minRepeatRun {
				if c.Style != currentStyle {
					out, currentStyle = appendStyle(out, currentStyle, c.Style, opts.TrueColor, opts.Colors256, opts.Hyperlinks, front.StyleCache)
				}
				out = cell.AppendContent(out, c.Content)
				out = append(out, "\x1b["...)
				out = strconv.AppendInt(out, int64(run-1), 10)
				out = append(out, 'b')
				x += run - 1
				continue
			}
		}

		if c.Style != currentStyle {
			out, currentStyle = appendStyle(out, currentStyle, c.Style, opts.TrueColor, opts.Colors256, opts.Hyperlinks, front.StyleCache)
		}
		if isBlankCell(c) {
			out = append(out, ' ')
		} else {
			out = cell.AppendContent(out, c.Content)
			if cell.IsCluster(c.Content) && !opts.ClusterWidths {
				out = appendLegacyCluster(out, c.Content, x, width)
			}
		}
	}

	// Clear whatever the previous frame left to the right of this row.
	// Styles carry into EL, so reset first or the erase paints a
	// background across the rest of the line.
	var defaultStyle cell.Style
	defaultStyle.Reset()
	if currentStyle != defaultStyle {
		out, currentStyle = appendStyle(out, currentStyle, defaultStyle, opts.TrueColor, opts.Colors256, opts.Hyperlinks, front.StyleCache)
	}
	// A row painted to the last column has nothing to its right, and the
	// cursor waits on that column for the wrap: EL there would erase it.
	if lastPainted < int(width)-1 {
		out = append(out, "\x1b[K"...)
	}
	return out, currentStyle
}

// InlineReserve returns the sequence that makes room for an inline frame of
// the given height and parks the cursor on its first row.
//
// Printing the newlines first is what forces the terminal to scroll if the
// cursor is near the bottom; moving back up then lands on the first row of
// space that is now guaranteed to exist.
func InlineReserve(out []byte, height uint16) []byte {
	if height == 0 {
		return out
	}
	for i := uint16(0); i < height; i++ {
		out = append(out, '\n')
	}
	out = append(out, "\x1b["...)
	out = strconv.AppendInt(out, int64(height), 10)
	out = append(out, 'A')
	return append(out, '\r')
}

// InlineRelease moves the cursor past the frame so a shell prompt lands below
// it, leaving the rendered output in the scrollback.
func InlineRelease(out []byte, height uint16) []byte {
	if height == 0 {
		return append(out, '\r')
	}
	out = append(out, "\x1b["...)
	out = strconv.AppendInt(out, int64(height), 10)
	out = append(out, 'B')
	return append(out, '\r')
}

// InlineExtents appends to dst[:0] how many columns each of the first rows of
// b takes when drawn by DiffInline: the width up to the last painted cell
// (a wide character counts both columns), zero for a blank row.
func InlineExtents(dst []uint16, b *Buffer, rows uint16) []uint16 {
	dst = dst[:0]
	w := int(b.Area.Width)
	for y := 0; y < int(min(rows, b.Area.Height)); y++ {
		last := -1
		for x, c := range b.Content[y*w : (y+1)*w] {
			if c.Content == cell.RuneContinuation || !isBlankCell(&c) || c.Style != (cell.Style{}) {
				last = x
			}
		}
		dst = append(dst, uint16(last+1))
	}
	return dst
}

// InlineVisualRows is how many terminal rows a band drawn with the given
// extents takes once the terminal is width columns wide: a terminal that
// reflows on a resize wraps a row of n columns over ceil(n/width) rows, and
// every row takes at least one.
func InlineVisualRows(extents []uint16, width uint16) int {
	if width == 0 {
		return len(extents)
	}
	n := 0
	for _, e := range extents {
		n += max(1, (int(e)+int(width)-1)/int(width))
	}
	return n
}

// InlineUp is the sequence that takes a cursor parked at the end of a band
// (DiffInlineParked) back to the start of its first row, when the band takes
// visualRows terminal rows.
func InlineUp(out []byte, visualRows int) []byte {
	if visualRows > 1 {
		out = append(out, "\x1b["...)
		out = strconv.AppendInt(out, int64(visualRows-1), 10)
		out = append(out, 'A')
	}
	return append(out, '\r')
}
