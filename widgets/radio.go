package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)


// RadioButton, çoklu seçenek gruplarında tekil seçim yapmayı sağlayan radyo butonudur.
type RadioButton struct {
	ID           string
	Selected     *string
	Value        string
	Label        string
	Style        cell.Style
	FocusedStyle cell.Style
}

// Draw, radyo butonunu ( ) veya (*) formatında çizer ve tıklanıldığında odağı alıp seçili grup değerini günceller.
func (rb RadioButton) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if rb.ID == "" || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	// Odaklanabilir olarak kaydet
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(rb.ID)
	}

	isFocused := (ctx.FocusedID == rb.ID)

	// Tıklama olayında odağı al ve seçimi güncelle
	if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(rb.ID)
			}
			if rb.Selected != nil {
				*rb.Selected = rb.Value
			}
		})
	}

	// Stil birleştirme
	textStyle := ctx.Style.Merge(rb.Style)
	if isFocused {
		textStyle = textStyle.Merge(rb.FocusedStyle)
	}

	// ( ) veya (*) durum metnini hazırla
	prefix := "( ) "
	if rb.Selected != nil && *rb.Selected == rb.Value {
		prefix = "(*) "
	}

	buf.SetString(ctx.Area.X, ctx.Area.Y, prefix+rb.Label, textStyle)
}

// SizeHint, radyo butonunun kaplayacağı tek satırlık alanı ve en boy ihtiyacını döner.
func (rb RadioButton) SizeHint(maxArea cell.Rect) (width, height uint16) {
	neededW := uint16(utf8.RuneCountInString(rb.Label) + 4)
	if neededW > maxArea.Width {
		neededW = maxArea.Width
	}
	return neededW, 1
}
