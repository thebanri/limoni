package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// StatusItem is one entry of a StatusBar: an optional key, drawn in the
// bar's KeyStyle, followed by text.
//
//	StatusItem{Key: "^Q", Text: "quit"}  →  ^Q quit
type StatusItem struct {
	Key   string
	Text  string
	Style cell.Style // merged over the bar's Style for Text
}

func (it StatusItem) width() int {
	w := cell.StringWidth(it.Text)
	if it.Key != "" {
		w += cell.StringWidth(it.Key)
		if it.Text != "" {
			w++
		}
	}
	return w
}

// StatusBar is a one-row bar with items on the left, centre and right, the
// usual last line of a full-screen application.
//
// When they do not all fit, the left group keeps its place, the right group
// comes next and is cut from its left edge, and the centre group is left out
// rather than overlap either of them.
type StatusBar struct {
	// ID names the bar in the semantic tree. With one, the node's value is
	// the bar's text, so a screen reader or an agent can read the key hints;
	// building it costs one allocation per frame. Without one, the node has
	// no value and drawing allocates nothing.
	ID string

	Left, Center, Right []StatusItem

	// Style fills the whole row; KeyStyle is merged over it for item keys.
	Style    cell.Style
	KeyStyle cell.Style
	// Separator goes between items in a group. Empty is two spaces.
	Separator string
}

func (s StatusBar) separator() string {
	if s.Separator == "" {
		return "  "
	}
	return s.Separator
}

func (s StatusBar) groupWidth(items []StatusItem) int {
	w := 0
	for i, it := range items {
		if i > 0 {
			w += cell.StringWidth(s.separator())
		}
		w += it.width()
	}
	return w
}

// drawGroup draws items from column x, clipped to [x, limit), and returns
// the column after the last one written.
func (s StatusBar) drawGroup(buf *buffer.Buffer, x, y, limit uint16, items []StatusItem, base, key cell.Style) uint16 {
	put := func(text string, st cell.Style) {
		if x < limit && text != "" {
			x += buf.SetStringWithin(x, y, text, st, limit-x)
		}
	}
	for i, it := range items {
		if i > 0 {
			put(s.separator(), base)
		}
		if it.Key != "" {
			put(it.Key, key)
			if it.Text != "" {
				put(" ", base)
			}
		}
		put(it.Text, base.Merge(it.Style))
	}
	return x
}

// Draw renders the bar on the first row of its area. It does not allocate.
func (s StatusBar) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	base := ctx.Style.Merge(s.Style)
	key := base.Merge(s.KeyStyle)
	y, end := area.Y, area.X+area.Width
	for x := area.X; x < end; x++ {
		buf.SetCell(x, y, cell.Cell{Content: ' ', Style: base})
	}

	leftEnd := s.drawGroup(buf, area.X, y, end, s.Left, base, key)
	if len(s.Left) > 0 && leftEnd < end {
		leftEnd++ // keep a gap after the left group
	}

	rightStart := end
	if rw := s.groupWidth(s.Right); rw > 0 && leftEnd < end {
		room := int(end - leftEnd)
		if rw <= room {
			rightStart = end - uint16(rw)
			s.drawGroup(buf, rightStart, y, end, s.Right, base, key)
		} else {
			// Too wide: keep its right end, which is usually the more
			// specific part (a clock, a position), and cut its left.
			rightStart = leftEnd
			s.drawRightClipped(buf, leftEnd, y, end, rw-room, base, key)
		}
	}

	cw := s.groupWidth(s.Center)
	if cw == 0 {
		return
	}
	cx := int(area.X) + (int(area.Width)-cw)/2
	gapStart, gapEnd := int(leftEnd), int(rightStart)
	if rightStart < end {
		gapEnd-- // and a gap before the right group
	}
	if cx < gapStart {
		cx = gapStart
	}
	if cx+cw > gapEnd {
		return
	}
	s.drawGroup(buf, uint16(cx), y, uint16(cx+cw), s.Center, base, key)
}

// drawRightClipped draws the right group with its first skip columns cut
// off, starting at x.
func (s StatusBar) drawRightClipped(buf *buffer.Buffer, x, y, limit uint16, skip int, base, key cell.Style) {
	put := func(text string, st cell.Style) {
		for rest := text; rest != "" && x < limit; {
			var cluster string
			var w int
			cluster, w, rest = cell.NextCluster(rest)
			if skip > 0 {
				skip -= w
				// A wide cluster cut in half: blank its visible half.
				for ; skip < 0 && x < limit; skip++ {
					buf.SetCell(x, y, cell.Cell{Content: ' ', Style: st})
					x++
				}
				continue
			}
			x += buf.SetStringWithin(x, y, cluster, st, limit-x)
		}
	}
	for i, it := range s.Right {
		if i > 0 {
			put(s.separator(), base)
		}
		if it.Key != "" {
			put(it.Key, key)
			if it.Text != "" {
				put(" ", base)
			}
		}
		put(it.Text, base.Merge(it.Style))
	}
}

// SizeHint takes the width offered and one row.
func (s StatusBar) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }

// AccessibilityNode describes the bar: with an ID, its items as text, "key
// text" each, groups separated by the separator.
func (s StatusBar) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	node := accessibility.AccessibilityNode{ID: s.ID, Role: accessibility.RoleGeneric, Label: "Status bar", Bounds: bounds}
	if focused {
		node.State |= accessibility.StateFocused
	}
	if s.ID == "" {
		return node
	}
	groups := [3][]StatusItem{s.Left, s.Center, s.Right}
	size := 0
	for _, group := range groups {
		for _, it := range group {
			size += len(s.separator()) + len(it.Key) + 1 + len(it.Text)
		}
	}
	var b strings.Builder
	b.Grow(size)
	for _, group := range groups {
		for _, it := range group {
			if b.Len() > 0 {
				b.WriteString(s.separator())
			}
			b.WriteString(it.Key)
			if it.Key != "" && it.Text != "" {
				b.WriteByte(' ')
			}
			b.WriteString(it.Text)
		}
	}
	node.Value = b.String()
	return node
}
