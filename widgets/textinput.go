package widgets

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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
func (state *TextInputState) HandleKey(key backend.KeyEvent) bool {
	switch key.Type {
	case backend.KeyRune:
		state.insert(key.Ch)
		return true

	case backend.KeySpace:
		state.insert(' ')
		return true

	case backend.KeyBackspace:
		return state.backspace()

	case backend.KeyDelete:
		return state.delete()

	case backend.KeyArrowLeft:
		if state.Cursor > 0 {
			state.Cursor--
			return true
		}

	case backend.KeyArrowRight:
		if state.Cursor < len(state.Text) {
			state.Cursor++
			return true
		}

	case backend.KeyHome:
		if state.Cursor != 0 {
			state.Cursor = 0
			return true
		}

	case backend.KeyEnd:
		if state.Cursor != len(state.Text) {
			state.Cursor = len(state.Text)
			return true
		}
	}
	return false
}

func (state *TextInputState) insert(r rune) {
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

	isFocused := (ctx.FocusedID == ti.ID)

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
				c.Style = boxStyle
			}
		}
	}

	// Metni veya placeholder'ı çiz
	textStr := ti.State.Value()
	if textStr == "" && ti.Placeholder != "" {
		phStyle := boxStyle.Merge(ti.PlaceholderStyle)
		buf.SetString(ctx.Area.X, ctx.Area.Y, ti.Placeholder, phStyle)
	} else {
		buf.SetString(ctx.Area.X, ctx.Area.Y, textStr, boxStyle)
	}

	// Eğer odaklıysa software cursor (Reverse style) çiz
	if isFocused {
		cursorX := ctx.Area.X + uint16(ti.State.Cursor)
		if cursorX < ctx.Area.X+ctx.Area.Width {
			if c := buf.Get(cursorX, ctx.Area.Y); c != nil {
				c.Style.Modifier |= cell.ModifierReverse
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
