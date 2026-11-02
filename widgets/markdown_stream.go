package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

// codeBatch is how many complete lines of an open code block gather before
// they settle: committing line by line would make a chat commit (and parse) a
// one-row chunk per line, while a batch keeps the live band short.
const codeBatch = 8

// SettledMarkdown splits a reply that is still arriving into the blocks that
// can no longer change and the rest. A block is settled once a blank line
// follows it or its code fence closes and the text after the fence shows
// whether a blank row follows (so the fence alone waits for it); a line is not settled before its
// newline arrives. Blank lines at the start are dropped: Markdown draws a
// run of them as one row, and the settled block before them ended with it.
//
// Inside a code fence that is still open, complete lines settle too, in
// batches of codeBatch and only where the highlighter is not inside a
// multi-line comment or string (its state would not survive the split). Then
// settled ends with a closing fence and rest begins with a line reopening it,
// the same fence and info string, so the code keeps its language. A fence
// line draws no row, so settled and rest draw the rows the whole text would.
// The contract: settled+rest equals the text without its leading blank lines,
// except that settled never ends in the newline after a closing fence (it
// would draw a blank row the whole text does not have), and a split fence adds
// the closing fence to settled and the opening line to rest.
//
// Settled text drawn on its own looks as it would inside the whole reply, so
// a chat can commit it to the scrollback (Terminal.Commit) and redraw only
// the rest while tokens stream in — which also keeps each redraw's parse to
// the block being written instead of the whole reply.
//
//	m.live += token
//	done, m.live = widgets.SettledMarkdown(m.live)
//	if done != "" {
//		result.Commit = append(result.Commit, widgets.NewMarkdown(done))
//	}
func SettledMarkdown(text string) (settled, rest string) {
	for {
		nl := strings.IndexByte(text, '\n')
		if nl < 0 || strings.TrimSpace(text[:nl]) != "" {
			break
		}
		text = text[nl+1:]
	}
	fence, cut := "", 0
	var fenceLine string // the opening line of the open fence, trimmed
	var lang *Language
	open, openKind, openDepth := "", TokenPlain, 0
	pending := 0     // complete code lines since the fence opened or last split
	splitFence := "" // the fence to reopen when the last cut is inside one
	closed := false  // the last cut is just after a closing fence
	closeAt := 0     // end of a closing fence whose next line is not known yet
	for start := 0; start < len(text); {
		nl := strings.IndexByte(text[start:], '\n')
		if nl < 0 {
			break
		}
		end := start + nl + 1
		line := strings.TrimSpace(text[start:end])
		if closeAt > 0 && line != "" {
			cut, closed, closeAt = closeAt, true, 0
		}
		switch {
		case fence != "":
			if strings.HasPrefix(line, fence) && strings.Trim(line, fence[:1]) == "" {
				fence, closeAt, splitFence = "", end, ""
				break
			}
			if lang != nil {
				code := expandTabs(strings.TrimRight(text[start:end], "\r\n"), 4)
				_, open, openKind, openDepth = lexLine(code, lang, open, openKind, openDepth)
			}
			pending++
			if pending >= codeBatch && open == "" && openDepth == 0 {
				cut, splitFence, pending, closed = end, fenceLine, 0, false
			}
		case codeFence(line) != "":
			fence, fenceLine, pending = codeFence(line), line, 0
			lang = LanguageByName(strings.TrimSpace(strings.TrimLeft(line, fence[:1])))
			open, openKind, openDepth = "", TokenPlain, 0
		case line == "":
			cut, closed, closeAt = end, false, 0
		}
		start = end
	}
	if closeAt > 0 && strings.TrimSpace(text[closeAt:]) != "" {
		cut, closed = closeAt, true
	}
	if splitFence != "" && fence != "" {
		return text[:cut] + fence, splitFence + "\n" + text[cut:]
	}
	if closed {
		// The newline after a closing fence would draw a blank row the whole
		// text, with its next line right under the fence, does not have.
		return text[:cut-1], text[cut:]
	}
	return text[:cut], text[cut:]
}

// Rows is how many rows the text takes drawn width columns wide, wrapping
// included. links is whether the terminal shows hyperlinks (Frame.Hyperlinks):
// without them a link's address is written out, which takes room.
// Terminal.Commit asks for it when given a height of zero. It lays the text
// out to count, so it allocates; it is not for every frame of a long text.
func (m *Markdown) Rows(width uint16, links bool) int {
	m.parse(cell.Style{}.Merge(m.Style), links)
	return m.visualLineCount(width)
}
