package layout

import "github.com/thebanri/limoni/core/cell"

// OverflowPolicy describes how content exceeding an allocated rectangle is handled.
type OverflowPolicy uint8

const (
	OverflowClip OverflowPolicy = iota
	OverflowVisible
	OverflowScroll
)

// Measure is the negotiated size contract for a layout child.
type Measure struct {
	MinWidth       uint16
	MinHeight      uint16
	IdealWidth     uint16
	IdealHeight    uint16
	MaxWidth       uint16
	MaxHeight      uint16
	ShrinkPriority int
	GrowPriority   int
	Overflow       OverflowPolicy
}

// Measurable is implemented by widgets that provide explicit negotiation.
type Measurable interface {
	Measure(maxArea cell.Rect) Measure
}

// LegacyMeasurable adapts the existing SizeHint contract without importing widgets.
type LegacyMeasurable interface {
	SizeHint(maxArea cell.Rect) (width, height uint16)
}

// MeasureWidget converts a legacy SizeHint widget into a normalized Measure.
func MeasureWidget(widget LegacyMeasurable, maxArea cell.Rect) Measure {
	if widget == nil {
		return Measure{}
	}
	w, h := widget.SizeHint(maxArea)
	return Measure{
		IdealWidth: w, IdealHeight: h,
		MaxWidth: maxArea.Width, MaxHeight: maxArea.Height,
		Overflow: OverflowClip,
	}
}

// Normalize fills omitted bounds and guarantees min <= ideal <= max where the
// available area permits it. It is deterministic even for impossible bounds.
func (m Measure) Normalize(available cell.Rect) Measure {
	if m.MaxWidth == 0 || m.MaxWidth > available.Width {
		m.MaxWidth = available.Width
	}
	if m.MaxHeight == 0 || m.MaxHeight > available.Height {
		m.MaxHeight = available.Height
	}
	if m.MinWidth > m.MaxWidth {
		m.MinWidth = m.MaxWidth
	}
	if m.MinHeight > m.MaxHeight {
		m.MinHeight = m.MaxHeight
	}
	if m.IdealWidth < m.MinWidth {
		m.IdealWidth = m.MinWidth
	}
	if m.IdealWidth > m.MaxWidth {
		m.IdealWidth = m.MaxWidth
	}
	if m.IdealHeight < m.MinHeight {
		m.IdealHeight = m.MinHeight
	}
	if m.IdealHeight > m.MaxHeight {
		m.IdealHeight = m.MaxHeight
	}
	return m
}

// Arrange resolves a list of measurements into deterministic child rectangles.
// Children receive their ideal size first; remaining space is distributed by
// GrowPriority, while shortages are removed from lowest ShrinkPriority first.
func Arrange(area cell.Rect, measurements []Measure, direction Direction, gap uint16) []cell.Rect {
	result := make([]cell.Rect, len(measurements))
	if len(measurements) == 0 {
		return result
	}
	available := area.Width
	if direction == Vertical {
		available = area.Height
	}
	totalGap := uint16(0)
	if len(measurements) > 1 {
		totalGap = uint16(len(measurements)-1) * gap
	}
	if available > totalGap {
		available -= totalGap
	} else {
		available = 0
	}
	sizes := make([]uint16, len(measurements))
	normalized := make([]Measure, len(measurements))
	for i, measurement := range measurements {
		measurement = measurement.Normalize(area)
		normalized[i] = measurement
		if direction == Horizontal {
			sizes[i] = measurement.IdealWidth
		} else {
			sizes[i] = measurement.IdealHeight
		}
	}
	distributeSizes(sizes, normalized, available, direction)
	pos := area.X
	if direction == Vertical {
		pos = area.Y
	}
	for i, size := range sizes {
		if direction == Horizontal {
			result[i] = cell.NewRect(pos, area.Y, size, area.Height)
		} else {
			result[i] = cell.NewRect(area.X, pos, area.Width, size)
		}
		pos += size + gap
	}
	return result
}

func distributeSizes(sizes []uint16, measurements []Measure, available uint16, direction Direction) {
	total := uint32(0)
	for _, size := range sizes {
		total += uint32(size)
	}
	if total == uint32(available) {
		return
	}
	if total < uint32(available) {
		remaining := uint32(available) - total
		for remaining > 0 {
			weight := 0
			for _, m := range measurements {
				if m.GrowPriority > 0 {
					weight += m.GrowPriority
				}
			}
			if weight == 0 {
				for i := range measurements {
					measurements[i].GrowPriority = 1
				}
				weight = len(measurements)
			}
			changed := false
			for i, m := range measurements {
				if m.GrowPriority <= 0 || remaining == 0 {
					continue
				}
				limit := m.MaxWidth
				if direction == Vertical {
					limit = m.MaxHeight
				}
				if sizes[i] >= limit {
					continue
				}
				add := uint32(m.GrowPriority) * remaining / uint32(weight)
				if add == 0 {
					add = 1
				}
				if uint32(sizes[i])+add > uint32(limit) {
					add = uint32(limit - sizes[i])
				}
				sizes[i] += uint16(add)
				remaining -= add
				changed = true
			}
			if !changed {
				break
			}
		}
	} else {
		shortage := total - uint32(available)
		for shortage > 0 {
			changed := false
			for i, m := range measurements {
				min := m.MinWidth
				if direction == Vertical {
					min = m.MinHeight
				}
				if sizes[i] <= min {
					continue
				}
				sizes[i]--
				shortage--
				changed = true
				if shortage == 0 {
					break
				}
			}
			if !changed {
				break
			}
		}
		// If minimum sizes themselves are impossible, clip deterministically
		// from the last children until the available extent is respected.
		for shortage > 0 {
			changed := false
			for i := len(sizes) - 1; i >= 0 && shortage > 0; i-- {
				if sizes[i] > 0 {
					sizes[i]--
					shortage--
					changed = true
				}
			}
			if !changed {
				break
			}
		}
	}
}
