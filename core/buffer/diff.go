package buffer

import (
	"bytes"
	"strconv"
	"unicode/utf8"

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
func Diff(front, back *Buffer, out []byte, trueColor, colors256 bool) ([]byte, error) {
	// Zero-Loop Fast-Path: Return immediately if buffer was not dirtied and dimensions match
	if !front.IsDirty && front.Area.Width == back.Area.Width && front.Area.Height == back.Area.Height {
		return out, nil
	}

	// If dimensions mismatch, resize back buffer and execute full stream redraw
	if front.Area.Width != back.Area.Width || front.Area.Height != back.Area.Height {
		back.Resize(front.Area)
		return DiffFullStream(front, back, out, trueColor, colors256)
	}

	width := front.Area.Width
	height := front.Area.Height
	totalCells := int(width) * int(height)
	if totalCells == 0 {
		return out, nil
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
		return DiffFullStream(front, back, out, trueColor, colors256)
	}

	return DiffSparse(front, back, out, trueColor, colors256)
}

// DiffSparse executes Case A: sparse differential rendering using cursor jumps (CUP)
// for only modified spans within lines. Used when dirtyRatio < 0.45.
func DiffSparse(front, back *Buffer, out []byte, trueColor, colors256 bool) ([]byte, error) {
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

			if cursorX != x || cursorY != y {
				out = appendCursor(out, x, y)
				cursorX = x
				cursorY = y
			}

			if frontCell.Style != currentStyle {
				out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, front.StyleCache)
			}

			w := 1
			if frontCell.Content == ' ' || frontCell.Content == 0 || frontCell.Content < 32 || frontCell.Content == 0x7F {
				out = append(out, ' ')
			} else {
				out = utf8.AppendRune(out, frontCell.Content)
				w = cell.RuneWidth(frontCell.Content)
				if w <= 0 {
					w = 1
				}
			}

			cursorX += uint16(w)
			if cursorX >= width {
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
		out, _ = appendStyle(out, currentStyle, defaultStyle, trueColor, colors256, front.StyleCache)
	}

	front.IsDirty = false
	return out, nil
}

// DiffFullStream executes Case B: continuous stream redraw for high-churn frames (dirtyRatio >= 0.45).
// It bypasses cursor jump calculations entirely, wraps the output in synchronized update mode (?2026),
// moves cursor to home (\x1b[H), sequentially overwrites line-by-line using \r\n,
// maintains lazy SGR color emission, and synchronizes buffers via copy(back.Content, front.Content).
func DiffFullStream(front, back *Buffer, out []byte, trueColor, colors256 bool) ([]byte, error) {
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

			// Lazy SGR style emission: only emit escape sequences when style changes
			if frontCell.Style != currentStyle {
				out, currentStyle = appendStyle(out, currentStyle, frontCell.Style, trueColor, colors256, front.StyleCache)
			}

			// Emit character rune
			if frontCell.Content == ' ' || frontCell.Content == 0 || frontCell.Content < 32 || frontCell.Content == 0x7F {
				out = append(out, ' ')
			} else {
				out = utf8.AppendRune(out, frontCell.Content)
			}
		}
	}

	// Reset style at frame end to prevent terminal style bleeding
	var defaultStyle cell.Style
	defaultStyle.Reset()
	if currentStyle != defaultStyle {
		out, _ = appendStyle(out, currentStyle, defaultStyle, trueColor, colors256, front.StyleCache)
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
func appendCursor(out []byte, x, y uint16) []byte {
	return AppendCursor(out, x, y)
}

func getStyleBytes(target cell.Style, trueColor, colors256 bool, cache map[cell.Style][]byte) []byte {
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

func appendStyle(out []byte, cur, target cell.Style, trueColor, colors256 bool, cache map[cell.Style][]byte) ([]byte, cell.Style) {
	if !trueColor {
		target = target.Downsample(trueColor, colors256)
	}
	if cur == target {
		return out, cur
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
		cur.Reset()
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
