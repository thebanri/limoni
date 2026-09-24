package widgets

// Marker picks the characters a Canvas is drawn with. Every marker draws the
// same 2×4 dots per cell; the coarser ones merge dots when the canvas is
// drawn, so anything that draws on a Canvas works with all of them.
//
// Braille has the finest resolution but needs a font with the Braille block,
// which not every terminal font carries. Sextants (Unicode 13, Symbols for
// Legacy Computing) and quadrants are block characters most modern monospace
// fonts draw edge to edge, so filled areas look solid instead of dotted.
type Marker uint8

const (
	// MarkerBraille draws 2×4 dots per cell with U+2800–U+28FF. The default.
	MarkerBraille Marker = iota
	// MarkerSextant draws 2×3 blocks per cell with U+1FB00–U+1FB3B.
	MarkerSextant
	// MarkerQuadrant draws 2×2 blocks per cell with ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█.
	MarkerQuadrant
	// MarkerHalfBlock draws 1×2 blocks per cell with ▀▄█.
	MarkerHalfBlock
	// MarkerBlock fills a cell with █ when any of its dots is set.
	MarkerBlock
)

// Dot bits as Canvas stores them, by row (y) and column (x); see brailleOffset.
const (
	dotRow0 = 0x01 | 0x08
	dotRow1 = 0x02 | 0x10
	dotRow2 = 0x04 | 0x20
	dotRow3 = 0x40 | 0x80
	dotLeft = 0x01 | 0x02 | 0x04 | 0x40
)

var quadrantRunes = [16]rune{' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛', '▗', '▚', '▐', '▜', '▄', '▙', '▟', '█'}

// markerRune is the character for one cell's dot mask under marker m.
func markerRune(m Marker, mask byte) rune {
	if mask == 0 {
		return ' '
	}
	switch m {
	case MarkerSextant:
		// Four dot rows into three: the middle sextant row takes the two
		// middle dot rows, so a shape keeps its symmetry top to bottom.
		return sextantRune(
			sub(mask, dotRow0, 1, 2) |
				sub(mask, dotRow1|dotRow2, 4, 8) |
				sub(mask, dotRow3, 16, 32))
	case MarkerQuadrant:
		return quadrantRunes[sub(mask, dotRow0|dotRow1, 1, 2)|sub(mask, dotRow2|dotRow3, 4, 8)]
	case MarkerHalfBlock:
		top, bottom := mask&(dotRow0|dotRow1) != 0, mask&(dotRow2|dotRow3) != 0
		switch {
		case top && bottom:
			return '█'
		case top:
			return '▀'
		}
		return '▄'
	case MarkerBlock:
		return '█'
	}
	return rune(0x2800 + int(mask))
}

// sub returns left and/or right when the dots of mask within rows are set in
// the left or right column.
func sub(mask, rows byte, left, right uint8) uint8 {
	var bits uint8
	if mask&rows&dotLeft != 0 {
		bits |= left
	}
	if mask&rows&^dotLeft != 0 {
		bits |= right
	}
	return bits
}

// sextantRune maps a sextant bit pattern (1 top-left, 2 top-right, 4 middle
// left, 8 middle right, 16 bottom left, 32 bottom right) to its character.
// U+1FB00 lists the patterns in order but leaves out the four that already
// existed: empty, the left half ▌, the right half ▐ and the full block █.
func sextantRune(bits uint8) rune {
	switch bits {
	case 0:
		return ' '
	case 21:
		return '▌'
	case 42:
		return '▐'
	case 63:
		return '█'
	}
	idx := int(bits) - 1
	if bits > 21 {
		idx--
	}
	if bits > 42 {
		idx--
	}
	return rune(0x1FB00 + idx)
}
