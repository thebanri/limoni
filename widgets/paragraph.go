package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
)

// Paragraph shows multi-line text.
// It supports wrapping the text word by word to the area's bounds (word wrap).
type Paragraph struct {
	// ID is the widget's focus ID.
	ID string
	// Text is the text to show. It supports newlines (\n).
	Text string
	// Style sets the text's colour, background and modifiers.
	Style cell.Style
	// FocusedStyle is the style applied when the paragraph is focused.
	FocusedStyle cell.Style
	// Wrap sets whether the text wraps to the next line at the bounds' width.
	Wrap bool

	// Caching fields to avoid heap allocation on draw loops
	lastText    string
	lastWidth   uint16
	lastWrap    bool
	cachedLines []string
	altWidth    uint16
	altLines    []string
}

// NewParagraph creates a new Paragraph widget with wrapping enabled by default.
func NewParagraph(text string) *Paragraph {
	return &Paragraph{
		Text: text,
		Wrap: true,
	}
}

// WithWrap enables or disables word-wrapping.
func (p *Paragraph) WithWrap(wrap bool) *Paragraph {
	p.Wrap = wrap
	return p
}

// WithStyle sets the paragraph text style.
func (p *Paragraph) WithStyle(style cell.Style) *Paragraph {
	p.Style = style
	return p
}

// WithFocusedStyle sets the style applied when focused.
func (p *Paragraph) WithFocusedStyle(style cell.Style) *Paragraph {
	p.FocusedStyle = style
	return p
}

// WithID sets the widget focus and event ID.
func (p *Paragraph) WithID(id string) *Paragraph {
	p.ID = id
	return p
}

// Draw lays out the text, splits it to the line width if needed, and draws it into the terminal buffer.
func (p *Paragraph) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	if p.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(p.ID)
	}
	// A click focuses the widget; registered as data, so it does not allocate.
	if ctx.RegisterClickAction != nil && p.ID != "" {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: p.ID})
	} else if p.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(p.ID)
			}
		})
	}

	mergedStyle := ctx.Style.Merge(p.Style)
	if ctx.IsFocused(p.ID) {
		mergedStyle = mergedStyle.Merge(p.FocusedStyle)
	}

	lines := p.lines(area.Width)

	bg := mergedStyle.Bg
	if bg.Type() == cell.ColorDefault && ctx.Style.Bg.Type() != cell.ColorDefault {
		bg = ctx.Style.Bg
	}

	// Draw line by line without going past the bounds' height
	for i, line := range lines {
		if uint16(i) >= area.Height {
			break
		}
		currY := area.Y + uint16(i)
		written := buf.SetStringWithin(area.X, currY, line, mergedStyle, area.Width)
		if bg.Type() != cell.ColorDefault {
			for x := area.X + written; x < area.X+area.Width; x++ {
				if c := buf.Get(x, currY); c != nil {
					c.Content = ' '
					c.Style.Bg = bg
				}
			}
		}
	}
	if bg.Type() != cell.ColorDefault {
		for i := len(lines); uint16(i) < area.Height; i++ {
			currY := area.Y + uint16(i)
			for x := area.X; x < area.X+area.Width; x++ {
				if c := buf.Get(x, currY); c != nil {
					c.Content = ' '
					c.Style.Bg = bg
				}
			}
		}
	}
}

// SizeHint reports the width and height the text would like.
// Layout negotiation: if Wrap is on, it works out how many lines the text takes at the given width (maxArea.Width).
// lines is the text split into lines at width, from a cache of the last two
// widths. A Viewport measures its child at one width and draws it at another;
// with a single slot, every frame wrapped the text twice.
func (p *Paragraph) lines(width uint16) []string {
	if !p.Wrap {
		width = 0 // unwrapped lines do not depend on the width
	}
	if p.Text != p.lastText || p.Wrap != p.lastWrap {
		p.lastText, p.lastWrap = p.Text, p.Wrap
		p.cachedLines, p.altLines = nil, nil
	}
	if p.cachedLines != nil && p.lastWidth == width {
		return p.cachedLines
	}
	if p.altLines != nil && p.altWidth == width {
		p.cachedLines, p.altLines = p.altLines, p.cachedLines
		p.lastWidth, p.altWidth = p.altWidth, p.lastWidth
		return p.cachedLines
	}
	p.altLines, p.altWidth = p.cachedLines, p.lastWidth
	if width > 0 {
		p.cachedLines = wrapText(p.Text, width)
	} else {
		p.cachedLines = splitLines(p.Text)
	}
	p.lastWidth = width
	return p.cachedLines
}

func (p *Paragraph) SizeHint(maxArea cell.Rect) (width, height uint16) {
	if len(p.Text) == 0 {
		return 0, 0
	}

	lines := p.lines(maxArea.Width)

	// Find the width of the longest line
	maxW := 0
	for _, line := range lines {
		if width := cell.StringWidth(line); width > maxW {
			maxW = width
		}
	}

	w := uint16(maxW)
	h := uint16(len(lines))

	// Do not exceed the bounds
	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}

	return w, h
}

// Measure provides explicit size negotiation for Paragraph.
func (p *Paragraph) Measure(maxArea cell.Rect) layout.Measure {
	w, h := p.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}

// wrapText splits a long text at word boundaries into lines no wider than width.
func wrapText(text string, width uint16) []string {
	if width == 0 {
		return nil
	}

	var wrappedLines []string
	rawLines := splitLines(text)

	for _, rawLine := range rawLines {
		if len(rawLine) == 0 {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		words := splitWords(rawLine)
		var currentLine string

		for _, word := range words {
			wordW := cell.StringWidth(word)
			// If a word alone is wider than the line, split it character by character
			if wordW > int(width) {
				chunks := breakWord(word, int(width))
				for _, chunk := range chunks {
					chunkW := cell.StringWidth(chunk)
					if len(currentLine) == 0 {
						currentLine = chunk
					} else if cell.StringWidth(currentLine)+1+chunkW <= int(width) {
						currentLine += " " + chunk
					} else {
						wrappedLines = append(wrappedLines, currentLine)
						currentLine = chunk
					}
				}
				continue
			}

			if len(currentLine) == 0 {
				currentLine = word
			} else if cell.StringWidth(currentLine)+1+wordW <= int(width) {
				currentLine += " " + word
			} else {
				wrappedLines = append(wrappedLines, currentLine)
				currentLine = word
			}
		}
		if len(currentLine) > 0 {
			wrappedLines = append(wrappedLines, currentLine)
		}
	}

	return wrappedLines
}

func breakWord(word string, width int) []string {
	if width <= 0 {
		return nil
	}
	var chunks []string
	var current strings.Builder
	currentW := 0

	for _, r := range word {
		rw := cell.RuneWidth(r)
		if currentW+rw > width && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
			currentW = 0
		}
		current.WriteRune(r)
		currentW += rw
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

// splitLines splits the text into raw lines at newlines (\n) (Windows \r\n is cleaned up too).
func splitLines(text string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			line := text[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start <= len(text) {
		line := text[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}

// splitWords splits a line into words at whitespace.
func splitWords(s string) []string {
	var words []string
	start := -1
	for i, r := range s {
		if r == ' ' {
			if start != -1 {
				words = append(words, s[start:i])
				start = -1
			}
		} else {
			if start == -1 {
				start = i
			}
		}
	}
	if start != -1 {
		words = append(words, s[start:])
	}
	return words
}

// AccessibilityNode returns the semantic node description for Paragraph.
//
// Static text is invisible to a screen reader unless something announces it,
// which is why a plain paragraph carries a node at all.
func (p *Paragraph) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	return accessibility.AccessibilityNode{
		ID:     p.ID,
		Role:   accessibility.RoleGeneric,
		Label:  "Text",
		Value:  p.Text,
		State:  state,
		Bounds: bounds,
	}
}
