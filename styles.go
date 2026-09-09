package limoni

import (
	"strconv"
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

// RGB creates a 24-bit TrueColor RGB Color.
func RGB(r, g, b uint8) Color {
	return cell.NewColorRGB(r, g, b)
}

// ANSI creates an 8-bit ANSI Color (0-255).
func ANSI(code uint8) Color {
	return cell.NewColorANSI(code)
}

// Hex parses a hex color string (e.g. "#FF5733" or "FF5733" or "#F53") into a TrueColor RGB Color.
func Hex(hexStr string) Color {
	s := strings.TrimPrefix(hexStr, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return cell.NewColorDefault()
	}
	val, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return cell.NewColorDefault()
	}
	return RGB(uint8((val>>16)&0xFF), uint8((val>>8)&0xFF), uint8(val&0xFF))
}

// Color Presets
var (
	ColorDefault = cell.NewColorDefault()
	ColorBlack   = cell.NewColorANSI(0)
	ColorRed     = cell.NewColorANSI(1)
	ColorGreen   = cell.NewColorANSI(2)
	ColorYellow  = cell.NewColorANSI(3)
	ColorBlue    = cell.NewColorANSI(4)
	ColorMagenta = cell.NewColorANSI(5)
	ColorCyan    = cell.NewColorANSI(6)
	ColorWhite   = cell.NewColorANSI(7)
)

// NewStyle returns an empty default Style.
func NewStyle() Style {
	return cell.NewStyle()
}

// Fg returns a Style with the given foreground color.
func Fg(c Color) Style {
	return cell.NewStyle().WithFg(c)
}

// Bg returns a Style with the given background color.
func Bg(c Color) Style {
	return cell.NewStyle().WithBg(c)
}

// Bold returns a bold Style.
func Bold() Style {
	return cell.NewStyle().Bold()
}

// Italic returns an italic Style.
func Italic() Style {
	return cell.NewStyle().Italic()
}

// Underline returns an underline Style.
func Underline() Style {
	return cell.NewStyle().Underline()
}

// Dim returns a dim/faint Style.
func Dim() Style {
	return cell.NewStyle().Dim()
}

// Reverse returns a reverse (inverted foreground/background) Style.
func Reverse() Style {
	return cell.NewStyle().Reverse()
}
