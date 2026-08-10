package cell

import (
	"fmt"
	"strconv"
	"strings"
)

// StyleResolver resolves custom/class style names or theme roles.
type StyleResolver func(tag string) Style

// ParseRichText parses a tag-formatted string (e.g. "Hello <fg=red,bold>World</>!") into a slice of Cell.
func ParseRichText(text string, baseStyle Style, resolver StyleResolver) []Cell {
	if text == "" {
		return nil
	}

	cells := make([]Cell, 0, len(text))
	styleStack := []Style{baseStyle}

	i := 0
	n := len(text)

	for i < n {
		ch := rune(text[i])
		if ch == '\\' && i+1 < n && text[i+1] == '<' {
			// Escaped '<'
			cells = append(cells, Cell{
				Content: '<',
				Style:   styleStack[len(styleStack)-1],
			})
			i += 2
			continue
		}

		if ch == '<' {
			// Parse tag
			closeIndex := strings.IndexByte(text[i:], '>')
			if closeIndex != -1 {
				tagContent := text[i+1 : i+closeIndex]
				i += closeIndex + 1

				if strings.HasPrefix(tagContent, "/") {
					// Closing tag (e.g. </fg>, </bold> or </>)
					if len(styleStack) > 1 {
						styleStack = styleStack[:len(styleStack)-1]
					}
					continue
				}

				// Opening tag (e.g. <fg=red,bold> or <success>)
				currentStyle := styleStack[len(styleStack)-1]
				newStyle := parseTagStyle(tagContent, currentStyle, resolver)
				styleStack = append(styleStack, newStyle)
				continue
			}
		}

		// Plain text character
		cells = append(cells, Cell{
			Content: ch,
			Style:   styleStack[len(styleStack)-1],
		})
		i++
	}

	return cells
}

func parseTagStyle(tagContent string, parentStyle Style, resolver StyleResolver) Style {
	style := parentStyle
	parts := strings.Split(tagContent, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Key-Value style attributes (e.g., fg=red, bg=black, role=success)
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])

			switch key {
			case "fg":
				style.Fg = parseColor(val)
			case "bg":
				style.Bg = parseColor(val)
			case "role", "class":
				if resolver != nil {
					style = style.Merge(resolver(val))
				}
			}
			continue
		}

		// Modifier tags or simple class names
		switch part {
		case "bold":
			style.Modifier |= ModifierBold
		case "dim":
			style.Modifier |= ModifierDim
		case "italic":
			style.Modifier |= ModifierItalic
		case "underline":
			style.Modifier |= ModifierUnderline
		case "blink":
			style.Modifier |= ModifierBlink
		case "reverse":
			style.Modifier |= ModifierReverse
		case "hidden":
			style.Modifier |= ModifierHidden
		case "strikethrough":
			style.Modifier |= ModifierStrikethrough
		default:
			// Fallback: treat as a simple class tag resolved via callback
			if resolver != nil {
				style = style.Merge(resolver(part))
			}
		}
	}

	return style
}

func parseColor(val string) Color {
	if strings.HasPrefix(val, "#") {
		var r, g, b uint8
		if len(val) == 7 {
			fmt.Sscanf(val, "#%02x%02x%02x", &r, &g, &b)
			return NewColorRGB(r, g, b)
		} else if len(val) == 4 {
			fmt.Sscanf(val, "#%1x%1x%1x", &r, &g, &b)
			return NewColorRGB(r*17, g*17, b*17)
		}
	}
	if code, err := strconv.Atoi(val); err == nil && code >= 0 && code <= 255 {
		return NewColorANSI(uint8(code))
	}
	switch strings.ToLower(val) {
	case "black":
		return NewColorANSI(0)
	case "red":
		return NewColorANSI(1)
	case "green":
		return NewColorANSI(2)
	case "yellow":
		return NewColorANSI(3)
	case "blue":
		return NewColorANSI(4)
	case "magenta":
		return NewColorANSI(5)
	case "cyan":
		return NewColorANSI(6)
	case "white":
		return NewColorANSI(7)
	case "brightblack":
		return NewColorANSI(8)
	case "brightred":
		return NewColorANSI(9)
	case "brightgreen":
		return NewColorANSI(10)
	case "brightyellow":
		return NewColorANSI(11)
	case "brightblue":
		return NewColorANSI(12)
	case "brightmagenta":
		return NewColorANSI(13)
	case "brightcyan":
		return NewColorANSI(14)
	case "brightwhite":
		return NewColorANSI(15)
	}
	return NewColorDefault()
}
