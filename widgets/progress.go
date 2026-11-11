package widgets

import (
	"fmt"
	"strconv"
	"time"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// ProgressBar renders a bounded horizontal progress indicator.
type ProgressBar struct {
	ID    string
	Value float64
	Min   float64
	Max   float64
	// Style is the percentage label, and the bar where the frame gives it
	// no style.
	Style cell.Style
	// FilledStyle and EmptyStyle are the two parts of the bar. When they
	// and Style are unset, they use the frame theme's primary and muted
	// roles, respectively.
	FilledStyle  cell.Style
	EmptyStyle   cell.Style
	FocusedStyle cell.Style
	ShowPercent  bool
}

func (p ProgressBar) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ctx.Area.Width == 0 || ctx.Area.Height == 0 || p.Max <= p.Min {
		return
	}

	if p.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(p.ID)
	}
	// A click focuses the widget; registered as data, so it does not allocate.
	if ctx.RegisterClickAction != nil && p.ID != "" {
		ctx.RegisterClickAction(ctx.Area, cell.ClickAction{Focus: p.ID})
	} else if p.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(p.ID)
			}
		})
	}

	value := p.Value
	if value < p.Min {
		value = p.Min
	}
	if value > p.Max {
		value = p.Max
	}
	ratio := (value - p.Min) / (p.Max - p.Min)
	filled := int(ratio * float64(ctx.Area.Width))
	filledStyle, emptyStyle := p.FilledStyle, p.EmptyStyle
	if p.Style == (cell.Style{}) && ctx.ThemeStyle != nil {
		if filledStyle == (cell.Style{}) {
			filledStyle = ctx.ThemeStyle("primary")
		}
		if emptyStyle == (cell.Style{}) {
			emptyStyle = ctx.ThemeStyle("muted")
		}
	}
	for x := uint16(0); x < ctx.Area.Width; x++ {
		style := ctx.Style.Merge(emptyStyle)
		content := '░'
		if int(x) < filled {
			style = ctx.Style.Merge(filledStyle)
			if ctx.IsFocused(p.ID) {
				style = style.Merge(p.FocusedStyle)
			}
			content = '█'
		}
		if style == (cell.Style{}) {
			style = ctx.Style.Merge(p.Style)
			if ctx.IsFocused(p.ID) {
				style = style.Merge(p.FocusedStyle)
			}
		}
		px := ctx.Area.X + x
		py := ctx.Area.Y
		if c := buf.Get(px, py); c != nil {
			c.Content = content
			c.Style = c.Style.Merge(style)
		} else {
			buf.SetCell(px, py, cell.Cell{Content: content, Style: style})
		}
	}
	if p.ShowPercent && ctx.Area.Width >= 5 {
		// Built on the stack: concatenating the label allocated every frame.
		var tmp [8]byte
		text := append(strconv.AppendInt(tmp[:0], int64(ratio*100), 10), '%')
		start := int(ctx.Area.X) + (int(ctx.Area.Width)-len(text))/2
		buf.SetString(uint16(start), ctx.Area.Y, string(text), ctx.Style.Merge(p.Style))
	}
}

func (p ProgressBar) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }

// AccessibilityNode returns the semantic node description for ProgressBar.
func (p ProgressBar) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.Node {
	state := accessibility.NodeState(0)
	if focused {
		state |= accessibility.StateFocused
	}
	ratio := 0.0
	if p.Max > p.Min {
		ratio = (p.Value - p.Min) / (p.Max - p.Min)
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
	}
	val := fmt.Sprintf("%d%%", int(ratio*100))
	return accessibility.Node{
		ID:     p.ID,
		Role:   accessibility.RoleProgress,
		Label:  "ProgressBar",
		Value:  val,
		State:  state,
		Bounds: bounds,
	}
}

// AnimatableProgressBar is a decorator for ProgressBar that handles time-based easing transitions.
type AnimatableProgressBar struct {
	ProgressBar
	Anim *animation.Float
}

// NewAnimatableProgressBar creates a new AnimatableProgressBar with initial value and default boundaries.
func NewAnimatableProgressBar(id string, initial float64) *AnimatableProgressBar {
	return &AnimatableProgressBar{
		ProgressBar: ProgressBar{
			ID:    id,
			Value: initial,
			Min:   0,
			Max:   100,
		},
		Anim: animation.NewFloat(initial),
	}
}

// AnimateTo initiates a smooth animation towards a target value.
func (ap *AnimatableProgressBar) AnimateTo(target float64, duration time.Duration, easing animation.EasingFunc) {
	ap.Anim.AnimateTo(target, duration, easing)
}

// Update increments the animation progress and returns true if the animation is still running.
func (ap *AnimatableProgressBar) Update(now time.Time) bool {
	animating := ap.Anim.Update(now)
	ap.Value = ap.Anim.Value()
	return animating
}
