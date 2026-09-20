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

	var currentStyle cell.Style
	currentStyle.Reset()

	// Column zero of the first row. The caller positioned the cursor on the
	// right row; \r makes the column unambiguous.
	out = append(out, '\r')

	for y := uint16(0); y < height; y++ {
		if y > 0 {
			out = append(out, "\r\n"...)
		}
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
					out = appendClusterResync(out, c.Content, x, width)
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
		out = append(out, "\x1b[K"...)
	}

	// Return to the first row so the next frame starts where this one did.
	if height > 1 {
		out = append(out, "\x1b["...)
		out = strconv.AppendInt(out, int64(height-1), 10)
		out = append(out, 'A')
	}
	out = append(out, '\r')

	if syncWrapped {
		out = append(out, "\x1b[?2026l"...)
	}

	copy(back.Content, front.Content)
	front.IsDirty = false
	return out, nil
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
