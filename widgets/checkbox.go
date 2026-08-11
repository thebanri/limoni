package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)


// Checkbox, işaretlenebilir interaktif bir onay kutusudur.
type Checkbox struct {
	ID           string
	Checked      *bool
	Label        string
	Style        cell.Style
	FocusedStyle cell.Style
}

// Draw, onay kutusunu [ ] veya [x] formatında çizer ve tıklanıldığında odağı alıp durumunu tersine çevirir (toggle).
func (cb Checkbox) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if cb.ID == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// Odaklanabilir olarak kaydet
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(cb.ID)
	}

	isFocused := (ctx.FocusedID == cb.ID)

	// Tıklama olayında odağı al ve değeri tersine çevir
	if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(cb.ID)
			}
			if cb.Checked != nil {
				*cb.Checked = !*cb.Checked
			}
		})
	}

	// Stil birleştirme
	textStyle := ctx.Style.Merge(cb.Style)
	if isFocused {
		textStyle = textStyle.Merge(cb.FocusedStyle)
	}

	// [ ] veya [x] durum metnini hazırla
	prefix := "[ ] "
	if cb.Checked != nil && *cb.Checked {
		prefix = "[x] "
	}

	buf.SetString(ctx.Area.X, ctx.Area.Y, prefix+cb.Label, textStyle)
}

// SizeHint, onay kutusunun kaplayacağı tek satırlık alanı ve en boy ihtiyacını döner.
func (cb Checkbox) SizeHint(maxArea cell.Rect) (width, height uint16) {
	neededW := uint16(utf8.RuneCountInString(cb.Label) + 4)
	if neededW > maxArea.Width {
		neededW = maxArea.Width
	}
	return neededW, 1
}

// AccessibilityNode returns the semantic node description for Checkbox.
func (cb Checkbox) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	if cb.Checked != nil && *cb.Checked {
		state |= accessibility.StateChecked
	}
	val := "false"
	if cb.Checked != nil && *cb.Checked {
		val = "true"
	}
	return accessibility.AccessibilityNode{
		ID:     cb.ID,
		Role:   accessibility.RoleCheckbox,
		Label:  cb.Label,
		Value:  val,
		State:  state,
		Bounds: bounds,
	}
}
