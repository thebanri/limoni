package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// BigTextSize is how many 8×8 font pixels one terminal cell holds.
type BigTextSize uint8

const (
	// BigTextFull draws each pixel as a cell: 8×8 cells per character.
	BigTextFull BigTextSize = iota
	// BigTextHalfHeight draws two pixels a cell, stacked: 8×4 per character,
	// which looks square in most fonts.
	BigTextHalfHeight
	// BigTextQuadrant draws 2×2 pixels a cell: 4×4 per character.
	BigTextQuadrant
)

func (s BigTextSize) cell() (w, h int) {
	switch s {
	case BigTextHalfHeight:
		return 1, 2
	case BigTextQuadrant:
		return 2, 2
	}
	return 1, 1
}

// BigText draws text in large letters from an 8×8 bitmap font, for titles,
// clocks and counters:
//
//	██████   ████
//	  ██    ██  ██
//	  ██    ██  ██ ...
//
// The font covers ASCII and Latin-1; ğ, ş, ı and the other Turkish letters
// outside Latin-1 fall back to their base letter, and anything else is drawn
// as "?". Lines are split on "\n".
type BigText struct {
	ID        string
	Text      string
	Size      BigTextSize
	Alignment Alignment
	Style     cell.Style
}

// glyph returns the font rows for r.
func bigGlyph(r rune) *[8]byte {
	switch r {
	case 'ğ':
		r = 'g'
	case 'Ğ':
		r = 'G'
	case 'ş':
		r = 's'
	case 'Ş':
		r = 'S'
	case 'ı':
		r = 'i'
	case 'İ':
		r = 'I'
	case '\t':
		r = ' '
	}
	switch {
	case r >= 0x20 && r < 0x7F:
		return &font8x8[0][r-0x20]
	case r >= 0xA0 && r <= 0xFF:
		return &font8x8[1][r-0xA0]
	}
	return &font8x8[0]['?'-0x20]
}

// lineWidth is the width in cells of one line of big text.
func (b BigText) lineWidth(line string) int {
	cw, _ := b.Size.cell()
	n := 0
	for range line {
		n++
	}
	return n * 8 / cw
}

// Draw renders the text. It does not allocate.
func (b BigText) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	style := ctx.Style.Merge(b.Style)
	cw, ch := b.Size.cell()
	glyphW, glyphH := 8/cw, 8/ch
	y := int(area.Y)
	text := b.Text
	for {
		line, rest, more := strings.Cut(text, "\n")
		text = rest
		if y+glyphH > int(area.Y+area.Height) {
			return
		}
		x := int(area.X)
		switch w := b.lineWidth(line); b.Alignment {
		case AlignCenter:
			x += max(0, (int(area.Width)-w)/2)
		case AlignRight:
			x += max(0, int(area.Width)-w)
		}
		for _, r := range line {
			if x+glyphW > int(area.X+area.Width) {
				break
			}
			g := bigGlyph(r)
			for gy := 0; gy < glyphH; gy++ {
				for gx := 0; gx < glyphW; gx++ {
					// The pixels this cell covers, as a quadrant bit pattern:
					// 1 top left, 2 top right, 4 bottom left, 8 bottom right.
					var bits uint8
					for py := 0; py < ch; py++ {
						row := g[gy*ch+py]
						for px := 0; px < cw; px++ {
							if row>>(gx*cw+px)&1 != 0 {
								bits |= 1 << (py*2 + px)
							}
						}
					}
					if bits == 0 {
						continue
					}
					switch {
					case b.Size == BigTextFull:
						bits = 15
					case cw == 1: // one pixel wide: it covers the whole width
						bits = map1x2[bits&1|bits>>1&2]
					}
					buf.SetCell(uint16(x+gx), uint16(y+gy), cell.Cell{Content: quadrantRunes[bits], Style: style})
				}
			}
			x += glyphW
		}
		y += glyphH
		if !more {
			return
		}
	}
}

// map1x2 turns a one-column pattern (1 top, 2 bottom) into quadrant bits
// covering the whole cell width: ▀ ▄ █.
var map1x2 = [4]uint8{0, 1 | 2, 4 | 8, 15}

// SizeHint is the size of the text at this Size, clipped to maxArea.
func (b BigText) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	_, ch := b.Size.cell()
	w, lines := 0, 0
	for _, line := range strings.Split(b.Text, "\n") {
		w = max(w, b.lineWidth(line))
		lines++
	}
	return min(uint16(w), maxArea.Width), min(uint16(lines*8/ch), maxArea.Height)
}

// AccessibilityNode reads the text itself, not the blocks it is drawn with.
func (b BigText) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	return accessibility.AccessibilityNode{ID: b.ID, Role: accessibility.RoleGeneric, Label: b.Text, Value: b.Text, State: state, Bounds: bounds}
}
