package main

import (
	"fmt"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// How long the modal takes to open, to wait for the agents, and to close.
const (
	modalOpen   = 140 * time.Millisecond
	stopWait    = 3 * time.Second
	stopShown   = 600 * time.Millisecond // the agents' list stays up at least this long
	modalClose  = 260 * time.Millisecond
	modalWidth  = 64
	modalMargin = 2 // columns of padding inside the frame
)

var modalBg = limoni.Hex("#232331")

// dimmer greys out what is already drawn under a modal.
type dimmer struct{}

func (dimmer) Draw(ctx cell.Context, buf *buffer.Buffer) {
	a := ctx.Area
	for y := a.Y; y < a.Y+a.Height; y++ {
		for x := a.X; x < a.X+a.Width; x++ {
			if c := buf.Get(x, y); c != nil {
				c.Style.Modifier |= cell.ModifierDim
			}
		}
	}
}

func (dimmer) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, maxArea.Height }

// animate redraws often for d: the app otherwise draws twice a second.
func (u *ui) animate(d time.Duration) {
	if u.wake == nil {
		return
	}
	go func() {
		for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(16 * time.Millisecond) {
			u.wake()
		}
		u.wake()
	}()
}

// ask opens the modal with question q (quit, merge, install).
func (u *ui) ask(q string) {
	u.confirm, u.modalAt, u.attached = q, time.Now(), false
	u.animate(modalOpen)
}

// ease is the eased progress of an animation that began at start and runs
// for d, from 0 to 1.
func ease(start time.Time, d time.Duration) float64 {
	p := min(float64(time.Since(start))/float64(d), 1)
	return 1 - (1-p)*(1-p)*(1-p)
}

// modalLine is a line of the modal's text.
type modalLine struct {
	text  string
	style limoni.Style
}

// drawModal draws the open question, or the agents stopping on the way out,
// in a box over the dimmed screen: it grows open, and shrinks shut.
func (u *ui) drawModal(f *limoni.Frame) {
	a := f.Area()
	u.buttons = u.buttons[:0] // only the modal's buttons can be clicked
	f.RenderWidget(dimmer{}, a)

	title, accent, lines, yes, no := u.modalContent()
	w := min(modalWidth, int(a.Width)-4)
	var body []modalLine
	for _, l := range lines {
		for _, part := range wrap(l.text, w-2*modalMargin-2) {
			body = append(body, modalLine{part, l.style})
		}
	}
	h := len(body) + 4 // frame, a blank line, the buttons
	scale := ease(u.modalAt, modalOpen)
	if !u.closingAt.IsZero() {
		scale = 1 - ease(u.closingAt, modalClose)
		if scale <= 0 {
			u.quit = true
			if u.wake != nil {
				u.wake()
			}
			return
		}
	}
	bw := max(int(float64(w)*(0.6+0.4*scale)), 8)
	bh := max(int(float64(h)*scale+0.5), 2)
	box := limoni.NewRect(a.X+uint16((int(a.Width)-bw)/2), a.Y+uint16((int(a.Height)-bh)/2), uint16(bw), uint16(bh))

	bg := func(st limoni.Style) limoni.Style { st.Bg = modalBg; return st }
	for y := box.Y; y < box.Y+box.Height; y++ {
		f.RenderWidget(&limoni.Paragraph{Text: " ", Style: bg(limoni.Style{})}, limoni.NewRect(box.X, y, box.Width, 1))
	}
	f.RenderWidget(&limoni.Block{Title: " " + title + " ", TitleStyle: limoni.Style{Fg: ink, Bg: accent.Fg, Modifier: limoni.ModifierBold},
		Borders: limoni.BorderAll, BorderSymbols: limoni.SymbolsRounded, BorderStyle: bg(accent)}, box)
	if scale < 0.85 { // too small yet for its text
		return
	}
	x, tw := box.X+1+modalMargin, box.Width-2-2*modalMargin
	for i, l := range body {
		if y := box.Y + 1 + uint16(i); y < box.Y+box.Height-1 {
			f.RenderWidget(&limoni.Paragraph{Text: l.text, Style: bg(l.style)}, limoni.NewRect(x, y, tw, 1))
		}
	}
	if yes.label == "" { // closing: no buttons
		return
	}
	by := box.Y + box.Height - 2
	bx := int(box.X+box.Width) - 1 - modalMargin
	for _, b := range []btn{no, yes} {
		bx -= keyButtonWidth(b) + 3
		u.keyButton(f, bx, by, b, bg)
	}
}

// modalContent is what the modal says for the question open, or while
// squeeze closes.
func (u *ui) modalContent() (title string, accent limoni.Style, lines []modalLine, yes, no btn) {
	plain := limoni.Style{}
	cancel := btn{"n", "Cancel", func() { u.answer(false) }}
	switch {
	case !u.closingAt.IsZero() || !u.stoppingAt.IsZero():
		lines = append(lines, modalLine{"Stopping the agents…", dim}, modalLine{"", plain})
		for _, r := range u.racers {
			mark, st, what := "●", yellow, "stopping"
			if r.ended.Load() != 0 {
				mark, st, what = "✓", green, "stopped"
			}
			lines = append(lines, modalLine{fmt.Sprintf("%s %-12s %s", mark, r.name, what), st})
		}
		return "Closing squeeze", red, lines, btn{}, btn{}
	case u.confirm == "quit":
		lines = []modalLine{
			{fmt.Sprintf("%d agents stop.", len(u.racers)), plain},
			{"Their worktrees stay in .squeeze/worktrees.", dim},
			{"squeeze clean removes them.", dim},
		}
		if len(u.racers) == 1 {
			lines[0].text = "The agent stops."
		}
		return "Quit squeeze?", red, lines, btn{"y", "Quit", func() { u.answer(true) }}, btn{"n", "Stay", func() { u.answer(false) }}
	case u.confirm == "merge":
		r := u.racers[u.sel]
		lines = []modalLine{{r.name + "'s changes go into your working tree, staged, not committed.", plain}}
		if st, _, _ := r.chk.result(); st == failed {
			lines = append(lines, modalLine{"", plain}, modalLine{"✗ It fails the check.", red})
		}
		return "Merge " + r.name + "?", yellow, lines, btn{"y", "Merge", func() { u.answer(true) }}, cancel
	case u.confirm == "install":
		return "Install " + u.pending.Name + "?", yellow, []modalLine{
			{"It runs:", plain},
			{u.pending.Install, cyan},
			{"", plain},
			{"For you alone, under ~/.local; nothing needs root.", dim},
		}, btn{"y", "Install", func() { u.answer(true) }}, cancel
	}
	return "", yellow, nil, btn{}, btn{}
}

// quitNow stops every agent and closes the modal once they have gone, or
// after stopWait for one that does not.
func (u *ui) quitNow() {
	u.stoppingAt = time.Now()
	for _, r := range u.racers {
		r.stop()
	}
	u.animate(stopWait + modalClose)
}

// closing advances the way out: when the agents have stopped, the modal
// starts to shut.
func (u *ui) closing() {
	if u.stoppingAt.IsZero() || !u.closingAt.IsZero() || time.Since(u.stoppingAt) < stopShown {
		return
	}
	for _, r := range u.racers {
		if r.ended.Load() == 0 && time.Since(u.stoppingAt) < stopWait {
			return
		}
	}
	u.closingAt = time.Now()
}

// keyButtonWidth is the width keyButton draws b in.
func keyButtonWidth(b btn) int { return cell.StringWidth(b.key) + 1 + cell.StringWidth(b.label) }

// keyButton draws b at (x, y) as its key, in yellow, and its label, and
// makes it clickable; style adds a background.
func (u *ui) keyButton(f *limoni.Frame, x int, y uint16, b btn, style func(limoni.Style) limoni.Style) {
	if style == nil {
		style = func(st limoni.Style) limoni.Style { return st }
	}
	kw := cell.StringWidth(b.key)
	f.RenderWidget(&limoni.Paragraph{Text: b.key, Style: style(yellow.Merge(bold))}, limoni.NewRect(uint16(x), y, uint16(kw), 1))
	f.RenderWidget(&limoni.Paragraph{Text: " " + b.label, Style: style(limoni.Style{})},
		limoni.NewRect(uint16(x+kw), y, uint16(1+cell.StringWidth(b.label)), 1))
	u.buttons = append(u.buttons, button{x: x, y: int(y), w: keyButtonWidth(b), act: b.act})
}
