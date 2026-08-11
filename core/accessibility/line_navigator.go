package accessibility

import (
	"strings"
)

// LineNavigator allows sequential line-oriented traversal of screen-reader-safe
// accessibility text. It tracks the current line position.
type LineNavigator struct {
	lines []string
	index int
}

// NewLineNavigator creates a new navigator from accessibility line mode output.
func NewLineNavigator(text string) *LineNavigator {
	if text == "" {
		return &LineNavigator{lines: nil, index: 0}
	}
	parts := strings.Split(text, "\n")
	var lines []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			lines = append(lines, part)
		}
	}
	return &LineNavigator{lines: lines, index: 0}
}

// Current returns the line at the current index. Returns empty string if out of bounds.
func (n *LineNavigator) Current() string {
	if n == nil || n.index < 0 || n.index >= len(n.lines) {
		return ""
	}
	return n.lines[n.index]
}

// Next advances the navigator to the next line and returns it. Returns empty string if out of bounds.
func (n *LineNavigator) Next() string {
	if n == nil || n.index >= len(n.lines)-1 {
		if n != nil {
			n.index = len(n.lines)
		}
		return ""
	}
	n.index++
	return n.lines[n.index]
}

// Previous moves the navigator to the previous line and returns it. Returns empty string if out of bounds.
func (n *LineNavigator) Previous() string {
	if n == nil || n.index <= 0 {
		if n != nil {
			n.index = -1
		}
		return ""
	}
	n.index--
	return n.lines[n.index]
}

// AnnounceCurrent returns a formal screen-reader announcement of the current line.
func (n *LineNavigator) AnnounceCurrent() string {
	curr := n.Current()
	if curr == "" {
		return "End of content"
	}
	return "Screen reader announcement: " + strings.TrimSpace(curr)
}
