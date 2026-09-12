package widgets

import (
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/layout"
)

// TextInputState, metin giriş kutusunun veri durumunu (text ve imleç konumu) saklar.
type TextInputState struct {
	Text   []rune
	Cursor int
}

// NewTextInputState, yeni bir TextInputState örneği oluşturur.
func NewTextInputState() *TextInputState {
	return &TextInputState{
		Text:   make([]rune, 0, 64),
		Cursor: 0,
	}
}

// Value, metin kutusundaki veriyi string olarak döner.
func (state *TextInputState) Value() string {
	return string(state.Text)
}

// SetValue, metin kutusunun içeriğini günceller ve imleci sona taşır.
func (state *TextInputState) SetValue(s string) {
	state.Text = []rune(s)
	state.Cursor = len(state.Text)
}

// HandleKey, basılan tuşu metin kutusuna uygular. Değer veya imleç değiştiyse true döner.
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
			state.Cursor--
			return true
		}

	case driver.KeyArrowRight:
		if state.Cursor < len(state.Text) {
			state.Cursor++
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
	// Araya karakter ekleme
	state.Text = append(state.Text, 0)
	copy(state.Text[state.Cursor+1:], state.Text[state.Cursor:])
	state.Text[state.Cursor] = r
	state.Cursor++
}

func (state *TextInputState) backspace() bool {
	if state.Cursor == 0 {
		return false
	}
	// İmlecin solundaki karakteri sil
	state.Text = append(state.Text[:state.Cursor-1], state.Text[state.Cursor:]...)
	state.Cursor--
	return true
}

func (state *TextInputState) delete() bool {
	if state.Cursor >= len(state.Text) {
		return false
	}
	// İmlecin altındaki karakteri sil
	state.Text = append(state.Text[:state.Cursor], state.Text[state.Cursor+1:]...)
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

// Draw, metin kutusunu çizer, tıklandığında odak almasını sağlar ve aktif odaklıysa software cursor gösterir.
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

	// Metni veya placeholder'ı çiz
	textStr := ti.State.Value()
	if textStr == "" && ti.Placeholder != "" {
		phStyle := boxStyle.Merge(ti.PlaceholderStyle)
		phRunes := []rune(ti.Placeholder)
		if len(phRunes) > visibleWidth {
			phRunes = phRunes[:visibleWidth]
		}
		buf.SetString(ctx.Area.X, ctx.Area.Y, string(phRunes), phStyle)

		// Boşken ilk karakter üzerinde software cursor çiz
		if isFocused {
			if c := buf.Get(ctx.Area.X, ctx.Area.Y); c != nil {
				c.Style.Modifier |= cell.ModifierReverse
				c.Style.Bg = cell.NewColorRGB(255, 255, 255)
				c.Style.Fg = cell.NewColorRGB(0, 0, 0)
				c.Style.Modifier |= cell.ModifierBold
			}
		}
	} else {
		var displayText strings.Builder
		cursorVisualCol := 0
		for idx, r := range ti.State.Text {
			if idx == ti.State.Cursor {
				cursorVisualCol = len([]rune(displayText.String()))
			}
			if r == '\n' {
				displayText.WriteString(" ↵ ")
			} else {
				displayText.WriteRune(r)
			}
		}
		if ti.State.Cursor >= len(ti.State.Text) {
			cursorVisualCol = len([]rune(displayText.String()))
		}

		runes := []rune(displayText.String())

		// Horizontal scrolling / Viewport calculation
		startOffset := 0
		if cursorVisualCol >= visibleWidth {
			startOffset = cursorVisualCol - visibleWidth + 1
		}
		if startOffset > len(runes) {
			startOffset = len(runes)
		}
		visibleRunes := runes[startOffset:]
		if len(visibleRunes) > visibleWidth {
			visibleRunes = visibleRunes[:visibleWidth]
		}

		s := ti.SelectionStart
		e := ti.SelectionEnd
		hasSelection := (s != -1 && e != -1 && s != e)
		if s > e {
			s, e = e, s
		}

		for i, r := range visibleRunes {
			charIdx := startOffset + i
			st := boxStyle
			if hasSelection && charIdx >= s && charIdx < e {
				if ti.SelectionStyle.Bg != 0 || ti.SelectionStyle.Fg != 0 || ti.SelectionStyle.Modifier != 0 {
					st = st.Merge(ti.SelectionStyle)
				} else {
					st.Bg = cell.NewColorRGB(255, 255, 255)
					st.Fg = cell.NewColorRGB(0, 0, 0)
					st.Modifier |= cell.ModifierBold
				}
			}
			colX := ctx.Area.X + uint16(i)
			if colX < ctx.Area.X+ctx.Area.Width {
				buf.SetCell(colX, ctx.Area.Y, cell.Cell{
					Content: r,
					Style:   st,
				})
			}
		}

		// Eğer odaklıysa software cursor çiz: harfin üzeri beyaz, yazı siyah
		if isFocused {
			relCursor := cursorVisualCol - startOffset
			if relCursor >= 0 && relCursor < visibleWidth {
				cursorX := ctx.Area.X + uint16(relCursor)
				if c := buf.Get(cursorX, ctx.Area.Y); c != nil {
					c.Style.Modifier |= cell.ModifierReverse
					c.Style.Bg = cell.NewColorRGB(255, 255, 255)
					c.Style.Fg = cell.NewColorRGB(0, 0, 0)
					c.Style.Modifier |= cell.ModifierBold
				}
			}
		}
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
	if ti.State != nil {
		val = string(ti.State.Text)
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
