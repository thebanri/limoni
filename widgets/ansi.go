package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

const esc = 0x1b

// drawANSI draws line as a program printed it: SGR sequences (colours, bold,
// underline…) change the style, every other escape sequence is skipped, and
// a reset returns to base rather than to the terminal's default. hoff columns
// are scrolled off on the left. It does not allocate.
func drawANSI(buf *buffer.Buffer, x, y, width uint16, line string, hoff int, base cell.Style) {
	col := -hoff
	style := base
	for len(line) > 0 && col < int(width) {
		if line[0] == esc {
			n, st := ansiEscape(line, style, base)
			line, style = line[n:], st
			continue
		}
		end := strings.IndexByte(line, esc)
		if end < 0 {
			end = len(line)
		}
		run := line[:end]
		line = line[end:]
		if col < 0 {
			var skipped int
			run, skipped = skipColumns(run, -col)
			col += skipped
			if run == "" || col < 0 {
				continue
			}
		}
		col += int(buf.SetStringWithin(x+uint16(col), y, run, style, width-uint16(col)))
	}
}

// stripANSI returns line without its escape sequences. It allocates only when
// there is one to remove.
func stripANSI(line string) string {
	if strings.IndexByte(line, esc) < 0 {
		return line
	}
	var b strings.Builder
	b.Grow(len(line))
	for len(line) > 0 {
		if line[0] == esc {
			n, _ := ansiEscape(line, cell.Style{}, cell.Style{})
			line = line[n:]
			continue
		}
		end := strings.IndexByte(line, esc)
		if end < 0 {
			end = len(line)
		}
		b.WriteString(line[:end])
		line = line[end:]
	}
	return b.String()
}

// ansiEscape measures the escape sequence at the start of s (s[0] is ESC) and
// applies it to style if it is SGR. An unterminated sequence takes the rest of
// the line.
func ansiEscape(s string, style, base cell.Style) (int, cell.Style) {
	if len(s) < 2 {
		return len(s), style
	}
	switch s[1] {
	case '[': // CSI: parameters 0x30–0x3F, intermediates 0x20–0x2F, a final byte 0x40–0x7E
		i := 2
		for i < len(s) && s[i] >= 0x30 && s[i] <= 0x3F {
			i++
		}
		params := s[2:i]
		for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2F {
			i++
		}
		if i >= len(s) {
			return len(s), style
		}
		if s[i] == 'm' && i == 2+len(params) && (params == "" || params[0] < '<') {
			style = applySGR(params, style, base)
		}
		return i + 1, style
	case ']', 'P', 'X', '^', '_': // OSC, DCS, SOS, PM, APC: up to BEL or ST
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 && s[1] == ']' {
				return i + 1, style
			}
			if s[i] == esc && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2, style
			}
		}
		return len(s), style
	}
	// Any other escape: intermediates, then one final byte (ESC ( B, ESC 7…).
	i := 1
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2F {
		i++
	}
	if i < len(s) {
		i++
	}
	return i, style
}

// sgrGroup is one ';'-separated SGR parameter with its ':' sub-parameters.
type sgrGroup struct {
	v [6]int
	n int
}

// applySGR applies the parameters of one SGR sequence. "Off" codes and resets
// return to base, so a line's own colour survives a program's ESC[0m.
func applySGR(params string, style, base cell.Style) cell.Style {
	var groups [16]sgrGroup
	count := 0
	cur := &groups[0]
	cur.n = 1
	for i := 0; i < len(params); i++ {
		switch c := params[i]; {
		case c >= '0' && c <= '9':
			if v := &cur.v[cur.n-1]; *v < 1<<16 {
				*v = *v*10 + int(c-'0')
			}
		case c == ':':
			if cur.n < len(cur.v) {
				cur.n++
			}
		case c == ';':
			if count+1 == len(groups) {
				i = len(params) // too many to be a real sequence: ignore the rest
				break
			}
			count++
			cur = &groups[count]
			cur.n = 1
		}
	}
	count++

	mod := func(bits cell.Modifier, on bool) {
		style.Modifier &^= bits
		if on {
			style.Modifier |= bits
		} else {
			style.Modifier |= base.Modifier & bits
		}
	}
	const underlines = cell.ModifierUnderline | cell.ModifierDoubleUnderline | cell.ModifierUndercurl
	for g := 0; g < count; g++ {
		p := groups[g].v[0]
		switch {
		case p == 0:
			style = base
		case p == 1:
			mod(cell.ModifierBold, true)
		case p == 2:
			mod(cell.ModifierDim, true)
		case p == 3:
			mod(cell.ModifierItalic, true)
		case p == 4:
			mod(underlines, false)
			switch sub := groups[g].v[1]; {
			case groups[g].n == 1 || sub == 1:
				mod(cell.ModifierUnderline, true)
			case sub == 2:
				mod(cell.ModifierDoubleUnderline, true)
			case sub >= 3:
				mod(cell.ModifierUndercurl, true)
			}
		case p == 5 || p == 6:
			mod(cell.ModifierBlink, true)
		case p == 7:
			mod(cell.ModifierReverse, true)
		case p == 8:
			mod(cell.ModifierHidden, true)
		case p == 9:
			mod(cell.ModifierStrikethrough, true)
		case p == 21:
			mod(underlines, false)
			mod(cell.ModifierDoubleUnderline, true)
		case p == 22:
			mod(cell.ModifierBold|cell.ModifierDim, false)
		case p == 23:
			mod(cell.ModifierItalic, false)
		case p == 24:
			mod(underlines, false)
		case p == 25:
			mod(cell.ModifierBlink, false)
		case p == 27:
			mod(cell.ModifierReverse, false)
		case p == 28:
			mod(cell.ModifierHidden, false)
		case p == 29:
			mod(cell.ModifierStrikethrough, false)
		case p >= 30 && p <= 37:
			style.Fg = cell.NewColorANSI(uint8(p - 30))
		case p >= 90 && p <= 97:
			style.Fg = cell.NewColorANSI(uint8(p - 90 + 8))
		case p == 39:
			style.Fg = base.Fg
		case p >= 40 && p <= 47:
			style.Bg = cell.NewColorANSI(uint8(p - 40))
		case p >= 100 && p <= 107:
			style.Bg = cell.NewColorANSI(uint8(p - 100 + 8))
		case p == 49:
			style.Bg = base.Bg
		case p == 38 || p == 48:
			c, used, ok := sgrColor(groups[g:count])
			g += used
			if !ok {
				continue
			}
			if p == 38 {
				style.Fg = c
			} else {
				style.Bg = c
			}
		}
	}
	return style
}

// sgrColor reads the colour after a 38 or 48 in groups[0], in either the
// ':' form (38:5:n, 38:2::r:g:b, 38:2:r:g:b) or the ';' form (38;5;n,
// 38;2;r;g;b). used is the number of further groups the ';' form took.
func sgrColor(groups []sgrGroup) (c cell.Color, used int, ok bool) {
	g := groups[0]
	if g.n > 1 { // colon form: everything is in this group
		switch {
		case g.v[1] == 5 && g.n >= 3:
			return cell.NewColorANSI(uint8(min(g.v[2], 255))), 0, true
		case g.v[1] == 2 && g.n >= 6:
			return sgrRGB(g.v[3], g.v[4], g.v[5]), 0, true
		case g.v[1] == 2 && g.n == 5:
			return sgrRGB(g.v[2], g.v[3], g.v[4]), 0, true
		}
		return 0, 0, false
	}
	if len(groups) >= 3 && groups[1].v[0] == 5 {
		return cell.NewColorANSI(uint8(min(groups[2].v[0], 255))), 2, true
	}
	if len(groups) >= 5 && groups[1].v[0] == 2 {
		return sgrRGB(groups[2].v[0], groups[3].v[0], groups[4].v[0]), 4, true
	}
	return 0, len(groups) - 1, false // malformed: the rest cannot be read safely
}

func sgrRGB(r, g, b int) cell.Color {
	return cell.NewColorRGB(uint8(min(r, 255)), uint8(min(g, 255)), uint8(min(b, 255)))
}
