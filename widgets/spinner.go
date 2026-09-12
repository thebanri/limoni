package widgets

import (
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// SpinnerSet is an animation: a cycle of frames and how long each one is shown.
type SpinnerSet struct {
	// Frames are displayed in order and wrap around.
	Frames []string
	// Interval is how long a single frame is shown.
	Interval time.Duration
}

// Built-in spinner animations.
//
// SpinnerBraille and SpinnerDots need a font with Braille coverage; SpinnerLine
// and SpinnerArrow use ASCII and plain box glyphs, so they are the safe choice
// for constrained terminals, CI logs and SSH sessions with unknown fonts.
var (
	// SpinnerBraille is the smooth eight-phase Braille spinner.
	SpinnerBraille = SpinnerSet{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 80 * time.Millisecond,
	}
	// SpinnerDots cycles a single Braille dot around its cell.
	SpinnerDots = SpinnerSet{
		Frames:   []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		Interval: 80 * time.Millisecond,
	}
	// SpinnerLine is the ASCII spinner that renders everywhere.
	SpinnerLine = SpinnerSet{
		Frames:   []string{"|", "/", "-", "\\"},
		Interval: 120 * time.Millisecond,
	}
	// SpinnerArrow rotates an arrow through eight compass directions.
	SpinnerArrow = SpinnerSet{
		Frames:   []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
		Interval: 100 * time.Millisecond,
	}
	// SpinnerBar grows and shrinks a vertical bar.
	SpinnerBar = SpinnerSet{
		Frames:   []string{"▁", "▃", "▄", "▅", "▆", "▇", "▆", "▅", "▄", "▃"},
		Interval: 90 * time.Millisecond,
	}
	// SpinnerPulse fades a block in and out using shade glyphs.
	SpinnerPulse = SpinnerSet{
		Frames:   []string{"░", "▒", "▓", "█", "▓", "▒"},
		Interval: 110 * time.Millisecond,
	}
	// SpinnerClock walks a clock face through twelve hours.
	SpinnerClock = SpinnerSet{
		Frames:   []string{"🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚", "🕛"},
		Interval: 100 * time.Millisecond,
	}
)

// FrameAt returns the frame shown after elapsed time, wrapping around the cycle.
func (s SpinnerSet) FrameAt(elapsed time.Duration) string {
	if len(s.Frames) == 0 {
		return ""
	}
	interval := s.Interval
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	if elapsed < 0 {
		elapsed = 0
	}
	return s.Frames[int(elapsed/interval)%len(s.Frames)]
}

// Width reports the display width of the widest frame, so a spinner never
// shifts the text next to it as the animation advances.
func (s SpinnerSet) Width() int {
	widest := 0
	for _, frame := range s.Frames {
		if w := cell.StringWidth(frame); w > widest {
			widest = w
		}
	}
	return widest
}

// Spinner renders an animated activity indicator with an optional label.
//
// The widget is stateless. Drive the animation either by clock, which is the
// usual choice under limoni.WithFPS:
//
//	widgets.Spinner{Since: startedAt, Label: "Fetching…"}
//
// or by explicit frame index, which keeps rendering deterministic in tests and
// golden-file snapshots:
//
//	widgets.Spinner{Frame: tick, Label: "Fetching…"}
type Spinner struct {
	// Set selects the animation. The zero value uses SpinnerBraille.
	Set SpinnerSet

	// Since is the time the activity started. When non-zero the frame is
	// derived from the elapsed time and Frame is ignored.
	Since time.Time
	// Frame is an explicit frame index, used when Since is zero.
	Frame int

	// Label is optional text drawn after the spinner.
	Label string

	// Style applies to the spinner glyph, LabelStyle to the label.
	// An unset Style falls back to the theme accent.
	Style      cell.Style
	LabelStyle cell.Style

	// Gap is the number of spaces between the glyph and the label.
	// Defaults to one.
	Gap uint16
	// GapZero places the label immediately after the glyph.
	GapZero bool

	// now overrides the clock in tests.
	now func() time.Time
}

func (s Spinner) set() SpinnerSet {
	if len(s.Set.Frames) == 0 {
		return SpinnerBraille
	}
	return s.Set
}

func (s Spinner) gap() uint16 {
	if s.GapZero {
		return 0
	}
	if s.Gap == 0 {
		return 1
	}
	return s.Gap
}

// CurrentFrame returns the glyph this spinner renders right now.
func (s Spinner) CurrentFrame() string {
	set := s.set()
	if len(set.Frames) == 0 {
		return ""
	}
	if !s.Since.IsZero() {
		now := time.Now
		if s.now != nil {
			now = s.now
		}
		return set.FrameAt(now().Sub(s.Since))
	}
	index := s.Frame % len(set.Frames)
	if index < 0 {
		index += len(set.Frames)
	}
	return set.Frames[index]
}

// Draw renders the spinner glyph and label on the first row of the area.
func (s Spinner) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}

	glyph := s.CurrentFrame()
	if glyph == "" {
		return
	}

	style := ctx.Style.Merge(s.Style)
	if s.Style == (cell.Style{}) && ctx.ThemeStyle != nil {
		style = ctx.Style.Merge(ctx.ThemeStyle("focus"))
	}

	written := buf.SetStringWithin(area.X, area.Y, glyph, style, area.Width)
	if s.Label == "" {
		return
	}

	// Pad to the widest frame so the label never jitters as frames change.
	if widest := uint16(s.set().Width()); written < widest {
		written = widest
	}

	labelX := area.X + written + s.gap()
	if labelX >= area.X+area.Width {
		return
	}

	labelStyle := ctx.Style.Merge(s.LabelStyle)
	buf.SetStringWithin(labelX, area.Y, s.Label, labelStyle, area.X+area.Width-labelX)
}

// SizeHint reports the width of the glyph plus label and a height of one row.
func (s Spinner) SizeHint(maxArea cell.Rect) (width, height uint16) {
	w := s.set().Width()
	if s.Label != "" {
		w += int(s.gap()) + cell.StringWidth(s.Label)
	}
	if w > int(maxArea.Width) {
		w = int(maxArea.Width)
	}
	return uint16(w), 1
}
