package terminal

import (
	"errors"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

// ErrNotInline is what Commit returns on a full-screen terminal, which has
// no scrollback of its own to print into.
var ErrNotInline = errors.New("terminal: Commit needs inline mode")

// Commit prints w, height rows tall and as wide as the terminal, above an
// inline application's band; a height of zero is as tall as w needs, from
// its Rows method (widgets.Markdown) or else its SizeHint. The rows become ordinary output: they scroll
// into the terminal's scrollback, stay there after the application exits,
// and are never drawn again. The band moves down beneath them.
//
// This is how a chat or an agent keeps a long conversation: finished
// messages are committed, and only what is still changing — the reply being
// streamed, the input — is drawn in the band each frame.
//
// Called while a frame is drawn (Frame.Commit), the rows go out with that
// frame; called elsewhere, at once. Either way it must run on the goroutine
// that draws. It allocates a buffer for the rows: it is meant for a message,
// not for every frame.
func (t *Terminal) Commit(height uint16, w widgets.Widget) error {
	if t.inline == 0 {
		return ErrNotInline
	}
	if w == nil {
		return nil
	}
	width, _, err := t.driver.Size()
	if err != nil {
		return err
	}
	if width == 0 {
		width = t.front.Area.Width
	}
	if height == 0 {
		if r, ok := w.(interface {
			Rows(width uint16, links bool) int
		}); ok {
			height = uint16(min(r.Rows(width, t.caps.Hyperlinks), 0xFFFF))
		} else {
			_, height = w.SizeHint(cell.NewRect(0, 0, width, 0xFFFF))
		}
		if height == 0 {
			return nil
		}
	}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, width, height))
	var style cell.Style
	style.Reset()
	ctx := cell.NewContext(buf.Area, style)
	ctx.Hyperlinks = t.caps.Hyperlinks
	w.Draw(ctx, buf)
	t.commits = buffer.AppendRows(t.commits, buf, t.diffOptions())
	if t.drawing {
		return nil
	}
	return t.present(time.Now())
}

// Commit prints w above an inline band with the frame; see Terminal.Commit.
// It returns ErrNotInline where there is no band.
func (f *Frame) Commit(height uint16, w widgets.Widget) error {
	if f.commit == nil {
		return ErrNotInline
	}
	return f.commit(height, w)
}

// SetCommitHandler replaces where Commit sends rows, for a frame drawn
// without a Terminal: testkit records them instead of printing them.
func (f *Frame) SetCommitHandler(fn func(height uint16, w widgets.Widget) error) { f.commit = fn }

// SetInlineFit makes an inline band as tall as the rows a frame draws in,
// up to the height given to SetInline, instead of always that tall. A band
// that holds an input under a reply that comes and goes then sits directly
// under the last committed row rather than above empty rows.
func (t *Terminal) SetInlineFit(on bool) { t.inlineFit = on }

// usedRows is the number of rows down to the last one with anything drawn
// in it, at least one and at most limit.
func usedRows(b *buffer.Buffer, limit uint16) uint16 {
	w := int(b.Area.Width)
	for y := min(int(b.Area.Height), int(limit)) - 1; y > 0; y-- {
		for _, c := range b.Content[y*w : (y+1)*w] {
			if (c.Content != ' ' && c.Content != 0) || c.Style != (cell.Style{}) {
				return uint16(y + 1)
			}
		}
	}
	return 1
}

func (t *Terminal) diffOptions() buffer.DiffOptions {
	return buffer.DiffOptions{
		TrueColor:  t.caps.TrueColor,
		Colors256:  t.caps.Colors256,
		EraseChar:  t.caps.EraseChar,
		RepeatChar: t.caps.RepeatChar,
		Hyperlinks: t.caps.Hyperlinks,
		// A terminal that confirmed mode 2027 needs no cursor re-anchoring
		// after each grapheme cluster.
		ClusterWidths: t.caps.ClusterWidths,
		ScrollRegions: t.caps.ScrollRegions,
		InsertDelete:  t.caps.ScrollRegions,
		// Draw already wrapped the frame in ?2026; wrapping again inside the
		// encoder would nest the sequence.
		SyncOutput: false,
	}
}

// unparkInline moves the cursor from the end of the band to its first row,
// where the driver expects it when it leaves the screen.
func (t *Terminal) unparkInline() {
	if t.inline == 0 || !t.inlineParked {
		return
	}
	t.inlineParked = false
	w, _, _ := t.driver.Size()
	if w == 0 {
		w = t.front.Area.Width
	}
	_, _ = t.driver.Write(buffer.InlineUp(nil, buffer.InlineVisualRows(t.inlineExt, w)))
}
