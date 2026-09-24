package widgets

import (
	"math"
	"strconv"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// eighthBlocks are the left-anchored partial blocks, index n being n/8 of a cell.
var eighthBlocks = [8]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}

// Gauge fills its whole area in proportion to Ratio, with a label centred
// on it. The edge of the fill moves in eighths of a cell, so a narrow gauge
// still moves smoothly. The part of the label over the fill is drawn in
// reverse, so it reads on either side of the edge.
type Gauge struct {
	ID string

	// Ratio is the filled fraction, clamped to [0, 1].
	Ratio float64

	// Label is drawn in the middle. Empty shows the percentage, "42%".
	Label string
	// HideLabel draws no label at all.
	HideLabel bool

	// Style is the unfilled part and the label.
	Style cell.Style
	// GaugeStyle is the filled part; its Fg is the fill colour.
	GaugeStyle cell.Style

	// WholeCells turns off the eighth-block edge, for fonts that draw the
	// partial blocks badly.
	WholeCells bool
}

func (g Gauge) ratio() float64 {
	if math.IsNaN(g.Ratio) || g.Ratio < 0 {
		return 0
	}
	if g.Ratio > 1 {
		return 1
	}
	return g.Ratio
}

// Draw renders the gauge. It does not allocate.
func (g Gauge) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	if g.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(g.ID)
	}

	base := ctx.Style.Merge(g.Style)
	fill := base.Merge(g.GaugeStyle)

	exact := g.ratio() * float64(area.Width)
	full := int(exact)
	eighths := int((exact - float64(full)) * 8)
	if g.WholeCells {
		full = int(math.Round(exact))
		eighths = 0
	}

	for y := area.Y; y < area.Y+area.Height; y++ {
		for i := 0; i < int(area.Width); i++ {
			c := cell.Cell{Content: ' ', Style: base}
			switch {
			case i < full:
				c = cell.Cell{Content: '█', Style: fill}
			case i == full && eighths > 0:
				// The partial block is fill-coloured on the unfilled background.
				st := base
				st.Fg = fill.Fg
				c = cell.Cell{Content: eighthBlocks[eighths], Style: st}
			}
			buf.SetCell(area.X+uint16(i), y, c)
		}
	}

	if g.HideLabel {
		return
	}
	var tmp [8]byte
	label := g.Label
	var digits []byte
	if label == "" {
		digits = append(strconv.AppendInt(tmp[:0], int64(math.Round(g.ratio()*100)), 10), '%')
	}
	width := len(digits)
	if label != "" {
		width = cell.StringWidth(label)
	}
	if width > int(area.Width) {
		width = int(area.Width)
	}
	x := area.X + (area.Width-uint16(width))/2
	y := area.Y + area.Height/2

	// Over the fill the label is reversed fill colour; elsewhere plain.
	over := base
	over.Fg = fill.Fg
	over = over.Reverse()
	styleAt := func(col uint16) cell.Style {
		if int(col-area.X) < full {
			return over
		}
		return base
	}
	if digits != nil {
		for i, ch := range digits {
			col := x + uint16(i)
			buf.SetCell(col, y, cell.Cell{Content: rune(ch), Style: styleAt(col)})
		}
		return
	}
	end := x + buf.SetStringWithin(x, y, label, base, uint16(width))
	for col := x; col < end; col++ {
		if c := buf.Get(col, y); c != nil && int(col-area.X) < full {
			c.Style = over
		}
	}
}

// SizeHint takes the width offered and one row.
func (g Gauge) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }

// AccessibilityNode reports the gauge as a progress indicator.
func (g Gauge) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	return progressNode(g.ID, "Gauge", g.Label, g.ratio(), bounds, focused)
}

// LineGauge is a one-row gauge: an optional label, then a line drawn heavy up
// to Ratio and light after it.
//
//	Upload 42% ━━━━━━━━━━──────────────
type LineGauge struct {
	ID string

	// Ratio is the filled fraction, clamped to [0, 1].
	Ratio float64

	// Label is drawn before the line, followed by a space. Empty shows the
	// percentage.
	Label string
	// HideLabel draws the line alone.
	HideLabel bool

	// Style is the label; FilledStyle and UnfilledStyle the two parts of
	// the line.
	Style         cell.Style
	FilledStyle   cell.Style
	UnfilledStyle cell.Style

	// FilledRune and UnfilledRune replace ━ and ─.
	FilledRune   rune
	UnfilledRune rune
}

// Draw renders the line gauge on the first row of its area. It does not allocate.
func (g LineGauge) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	if g.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(g.ID)
	}
	ratio := Gauge{Ratio: g.Ratio}.ratio()
	base := ctx.Style.Merge(g.Style)

	x, end := area.X, area.X+area.Width
	if !g.HideLabel {
		if g.Label != "" {
			x += buf.SetStringWithin(x, area.Y, g.Label, base, area.Width)
		} else {
			var tmp [8]byte
			digits := append(strconv.AppendInt(tmp[:0], int64(math.Round(ratio*100)), 10), '%')
			for _, ch := range digits {
				if x >= end {
					break
				}
				buf.SetCell(x, area.Y, cell.Cell{Content: rune(ch), Style: base})
				x++
			}
		}
		if x < end {
			buf.SetCell(x, area.Y, cell.Cell{Content: ' ', Style: base})
			x++
		}
	}

	filledRune, unfilledRune := g.FilledRune, g.UnfilledRune
	if filledRune == 0 {
		filledRune = '━'
	}
	if unfilledRune == 0 {
		unfilledRune = '─'
	}
	filled := x + uint16(math.Round(ratio*float64(end-x)))
	for ; x < end; x++ {
		if x < filled {
			buf.SetCell(x, area.Y, cell.Cell{Content: filledRune, Style: ctx.Style.Merge(g.FilledStyle)})
		} else {
			buf.SetCell(x, area.Y, cell.Cell{Content: unfilledRune, Style: ctx.Style.Merge(g.UnfilledStyle)})
		}
	}
}

// SizeHint takes the width offered and one row.
func (g LineGauge) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }

// AccessibilityNode reports the gauge as a progress indicator.
func (g LineGauge) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	return progressNode(g.ID, "LineGauge", g.Label, Gauge{Ratio: g.Ratio}.ratio(), bounds, focused)
}

func progressNode(id, kind, label string, ratio float64, bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	if label == "" {
		label = kind
	}
	return accessibility.AccessibilityNode{
		ID:     id,
		Role:   accessibility.RoleProgress,
		Label:  label,
		Value:  strconv.Itoa(int(math.Round(ratio*100))) + "%",
		State:  state,
		Bounds: bounds,
	}
}
