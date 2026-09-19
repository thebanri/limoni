package widgets

import (
	"slices"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/layout"
)

// TextInputState holds the text of an input box and the cursor position.
//
// Text is a slice of code points and Cursor an index into it, but editing
// moves by grapheme cluster: Backspace after a family emoji removes the whole
// family, not just its last person, and the arrow keys never stop inside a
// flag. Code that sets Cursor directly should put it on a cluster boundary;
// one inside a cluster is drawn at the start of that cluster.
type TextInputState struct {
	Text   []rune
	Cursor int

	// text is Text as UTF-8, rebuilt only when Text changes, so drawing and
	// the accessibility node read it without converting on every frame.
	text  string
	built []rune // the Text that text was built from
}

// NewTextInputState creates an empty TextInputState.
func NewTextInputState() *TextInputState {
	return &TextInputState{
		Text:   make([]rune, 0, 64),
		Cursor: 0,
	}
}

// Value returns the text as a string.
func (state *TextInputState) Value() string {
	return state.str()
}

// SetValue replaces the text and moves the cursor to the end.
func (state *TextInputState) SetValue(s string) {
	state.Text = []rune(s)
	state.Cursor = len(state.Text)
}

// str returns Text as a string, converting only when Text has changed since
// the last call. Comparing is cheap next to allocating on every frame.
func (state *TextInputState) str() string {
	if !slices.Equal(state.built, state.Text) {
		state.text = string(state.Text)
		state.built = append(state.built[:0], state.Text...)
	}
	return state.text
}

// clusterAround returns the rune indexes of the grapheme cluster boundaries
// immediately before and after rune index i. At a boundary, before is the
// start of the cluster ending at i and after the end of the one starting there.
func (state *TextInputState) clusterAround(i int) (before, after int) {
	before, after = 0, len(state.Text)
	start := 0
	rest := state.str()
	for rest != "" {
		cluster, _, next := cell.NextCluster(rest)
		end := start + utf8.RuneCountInString(cluster)
		if end < i || (end == i && start < i) {
			before = start
		}
		if start >= i && end > i {
			after = end
			break
		}
		if start < i && end > i { // i is inside this cluster
			before, after = start, end
			break
		}
		start, rest = end, next
	}
	return before, after
}

// HandleKey applies a key press. It reports whether the text or the cursor changed.
func (state *TextInputState) HandleKey(key driver.KeyEvent) bool {
	switch key.Type {
	case driver.KeyRune:
		state.insert(key.Ch)
		return true

	case driver.KeyEnter:
		if key.Shift || key.Alt || key.Ctrl {
			state.insert('\n')
			return true
		}

	case driver.KeySpace:
		state.insert(' ')
		return true

	case driver.KeyBackspace:
		return state.backspace()

	case driver.KeyDelete:
		return state.delete()

	case driver.KeyArrowLeft:
		if state.Cursor > 0 {
			state.Cursor, _ = state.clusterAround(state.Cursor)
			return true
		}

	case driver.KeyArrowRight:
		if state.Cursor < len(state.Text) {
			_, state.Cursor = state.clusterAround(state.Cursor)
			return true
		}

	case driver.KeyHome:
		if state.Cursor != 0 {
			state.Cursor = 0
			return true
		}

	case driver.KeyEnd:
		if state.Cursor != len(state.Text) {
			state.Cursor = len(state.Text)
			return true
		}
	}
	return false
}

func (state *TextInputState) insert(r rune) {
	if r == '\r' {
		return
	}
	state.Cursor = clampInt(state.Cursor, 0, len(state.Text))
	state.Text = append(state.Text, 0)
	copy(state.Text[state.Cursor+1:], state.Text[state.Cursor:])
	state.Text[state.Cursor] = r
	state.Cursor++
}

// backspace removes the whole grapheme cluster before the cursor.
func (state *TextInputState) backspace() bool {
	if state.Cursor <= 0 {
		return false
	}
	state.Cursor = clampInt(state.Cursor, 0, len(state.Text))
	start, _ := state.clusterAround(state.Cursor)
	state.Text = append(state.Text[:start], state.Text[state.Cursor:]...)
	state.Cursor = start
	return true
}

// delete removes the whole grapheme cluster after the cursor.
func (state *TextInputState) delete() bool {
	if state.Cursor >= len(state.Text) {
		return false
	}
	_, end := state.clusterAround(state.Cursor)
	state.Text = append(state.Text[:state.Cursor], state.Text[end:]...)
	return true
}

// TextInput, tek satırlı bir metin girişi kutusudur.
type TextInput struct {
	ID               string
	State            *TextInputState
	Placeholder      string
	Style            cell.Style
	PlaceholderStyle cell.Style
	FocusedStyle     cell.Style
	SelectionStart   int
	SelectionEnd     int
	SelectionStyle   cell.Style
	Focused          bool

	// Secret masks the input for passwords and tokens. Every character is drawn
	// as MaskRune, so the secret never reaches the cell buffer, and the
	// accessibility node carries no value and is marked StateSensitive, so it
	// never reaches a screen reader, the automation socket or a recording.
	Secret bool
	// MaskRune is the glyph drawn in place of each character when Secret is
	// set. Defaults to '•'.
	MaskRune rune
}

// NewTextInput creates a new TextInput widget with the specified ID.
func NewTextInput(id string) *TextInput {
	return &TextInput{
		ID:             id,
		State:          NewTextInputState(),
		SelectionStart: -1,
		SelectionEnd:   -1,
	}
}

// WithState sets the TextInputState.
func (ti *TextInput) WithState(state *TextInputState) *TextInput {
	ti.State = state
	return ti
}

// WithPlaceholder sets the placeholder text.
func (ti *TextInput) WithPlaceholder(ph string) *TextInput {
	ti.Placeholder = ph
	return ti
}

// WithStyle sets the default box style.
func (ti *TextInput) WithStyle(style cell.Style) *TextInput {
	ti.Style = style
	return ti
}

// WithPlaceholderStyle sets the style for the placeholder text.
func (ti *TextInput) WithPlaceholderStyle(style cell.Style) *TextInput {
	ti.PlaceholderStyle = style
	return ti
}

// WithFocusedStyle sets the style when the input is focused.
func (ti *TextInput) WithFocusedStyle(style cell.Style) *TextInput {
	ti.FocusedStyle = style
	return ti
}

// WithSelection sets the selection range [start, end).
func (ti *TextInput) WithSelection(start, end int) *TextInput {
	ti.SelectionStart = start
	ti.SelectionEnd = end
	return ti
}

// WithSelectionStyle sets the style for selected text.
func (ti *TextInput) WithSelectionStyle(style cell.Style) *TextInput {
	ti.SelectionStyle = style
	return ti
}

// WithFocused overrides the focus state explicitly.
func (ti *TextInput) WithFocused(focused bool) *TextInput {
	ti.Focused = focused
	return ti
}

// Draw draws the input, takes focus when clicked, and shows a software cursor
// while focused. It does not allocate: the text is converted to a string only
// when it changes, and clusters are measured in place.
func (ti TextInput) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ti.ID == "" || ti.State == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// Odak sistemine kaydol
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(ti.ID)
	}

	isFocused := ti.Focused || (ti.ID != "" && ctx.FocusedID == ti.ID)

	// Tıklama olayında odağı üzerine al
	if ctx.RegisterClick != nil && ctx.SetFocus != nil {
		ctx.RegisterClick(ctx.Area, func() {
			ctx.SetFocus(ti.ID)
		})
	}

	// Stil birleştirme
	boxStyle := ctx.Style.Merge(ti.Style)
	if isFocused {
		boxStyle = boxStyle.Merge(ti.FocusedStyle)
	}

	// Arka planı doldur
	for y := ctx.Area.Y; y < ctx.Area.Y+ctx.Area.Height; y++ {
		for x := ctx.Area.X; x < ctx.Area.X+ctx.Area.Width; x++ {
			if c := buf.Get(x, y); c != nil {
				c.Content = ' '
				c.Style = c.Style.Merge(boxStyle)
			}
		}
	}

	visibleWidth := int(ctx.Area.Width)
	if visibleWidth <= 0 {
		return
	}

	// Placeholder when empty.
	textStr := ti.State.str()
	if textStr == "" && ti.Placeholder != "" {
		phStyle := boxStyle.Merge(ti.PlaceholderStyle)
		buf.SetStringWithin(ctx.Area.X, ctx.Area.Y, ti.Placeholder, phStyle, ctx.Area.Width)
		if isFocused {
			paintCursor(buf, ctx.Area.X, ctx.Area.Y)
		}
		return
	}

	mask := ti.MaskRune
	if mask == 0 {
		mask = '•'
	}
	maskWidth := cell.RuneWidth(mask)
	if maskWidth < 1 {
		maskWidth = 1
	}

	// First pass: find the cursor's column, measuring whole grapheme clusters
	// so a wide character or an emoji sequence takes the columns it is drawn in.
	cursorCol, col, runeIdx := -1, 0, 0
	for rest := textStr; rest != ""; {
		cluster, cw, next := cell.NextCluster(rest)
		n := utf8.RuneCountInString(cluster)
		if cursorCol < 0 && runeIdx+n > ti.State.Cursor {
			cursorCol = col
		}
		col += ti.displayWidth(cluster, cw, maskWidth)
		runeIdx += n
		rest = next
	}
	if cursorCol < 0 {
		cursorCol = col
	}

	// Scroll horizontally so the cursor cell stays visible.
	startCol := 0
	if cursorCol >= visibleWidth {
		startCol = cursorCol - visibleWidth + 1
	}

	s, e := ti.SelectionStart, ti.SelectionEnd
	hasSelection := s != -1 && e != -1 && s != e
	if s > e {
		s, e = e, s
	}
	selStyle := boxStyle
	if ti.SelectionStyle.Bg != 0 || ti.SelectionStyle.Fg != 0 || ti.SelectionStyle.Modifier != 0 {
		selStyle = selStyle.Merge(ti.SelectionStyle)
	} else {
		selStyle.Bg = cell.NewColorRGB(255, 255, 255)
		selStyle.Fg = cell.NewColorRGB(0, 0, 0)
		selStyle.Modifier |= cell.ModifierBold
	}

	// Second pass: draw the clusters inside the visible window. A cluster cut
	// by the left edge is left blank rather than drawn in half.
	col, runeIdx = 0, 0
	for rest := textStr; rest != ""; {
		cluster, cw, next := cell.NextCluster(rest)
		n := utf8.RuneCountInString(cluster)
		w := ti.displayWidth(cluster, cw, maskWidth)
		if col-startCol+w > visibleWidth {
			break
		}
		if col >= startCol && w > 0 {
			st := boxStyle
			if hasSelection && runeIdx >= s && runeIdx < e {
				st = selStyle
			}
			x := ctx.Area.X + uint16(col-startCol)
			switch {
			case ti.Secret:
				// One mask per character as the reader sees it: a cluster, not
				// a code point, so the mask does not reveal how emoji are built.
				putGlyph(buf, x, ctx.Area.Y, mask, maskWidth, st)
			case cluster == "\n":
				buf.SetStringWithin(x, ctx.Area.Y, " ↵ ", st, 3)
			default:
				buf.SetStringWithin(x, ctx.Area.Y, cluster, st, uint16(w))
			}
		}
		col += w
		runeIdx += n
		rest = next
	}

	if isFocused {
		if rel := cursorCol - startCol; rel >= 0 && rel < visibleWidth {
			paintCursor(buf, ctx.Area.X+uint16(rel), ctx.Area.Y)
		}
	}
}

// displayWidth is how many columns a cluster takes in the input: the mask's
// width when secret, three for the " ↵ " that stands for a newline, and its
// own width otherwise. Other control characters take none.
func (ti TextInput) displayWidth(cluster string, width, maskWidth int) int {
	switch {
	case ti.Secret:
		return maskWidth
	case cluster == "\n":
		return 3
	}
	return width
}

// putGlyph writes one code point that may be two columns wide.
func putGlyph(buf *buffer.Buffer, x, y uint16, r rune, width int, st cell.Style) {
	buf.SetCell(x, y, cell.Cell{Content: r, Style: st})
	if width == 2 {
		buf.SetCell(x+1, y, cell.Cell{Content: cell.RuneContinuation, Style: st})
	}
}

// paintCursor draws the software cursor: the cell under it inverted.
func paintCursor(buf *buffer.Buffer, x, y uint16) {
	if c := buf.Get(x, y); c != nil {
		c.Style.Modifier |= cell.ModifierReverse | cell.ModifierBold
		c.Style.Bg = cell.NewColorRGB(255, 255, 255)
		c.Style.Fg = cell.NewColorRGB(0, 0, 0)
	}
}

// SizeHint, metin giriş kutusunun tek satırlı olduğunu bildirir.
func (ti TextInput) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, 1
}

// Measure provides explicit size negotiation for TextInput.
func (ti TextInput) Measure(maxArea cell.Rect) layout.Measure {
	w, h := ti.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}

// AccessibilityNode returns the semantic node description for TextInput.
func (ti TextInput) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	val := ""
	if ti.Secret {
		state |= accessibility.StateSensitive
	} else if ti.State != nil {
		val = ti.State.str()
	}
	return accessibility.AccessibilityNode{
		ID:     ti.ID,
		Role:   accessibility.RoleInput,
		Label:  ti.Placeholder,
		Value:  val,
		State:  state,
		Bounds: bounds,
	}
}
