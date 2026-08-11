package layout

import "github.com/thebanri/limoni/core/cell"

// Alignment controls placement on the cross axis of an arranged child.
type Alignment uint8

const (
	AlignStart Alignment = iota
	AlignCenter
	AlignEnd
	AlignStretch
	AlignBaseline
)

// ArrangeAligned resolves children like Arrange and then applies cross-axis
// alignment. Arrange remains the compatibility/default API and stretches
// children across the cross axis.
func ArrangeAligned(area cell.Rect, measurements []Measure, direction Direction, gap uint16, alignment Alignment) []cell.Rect {
	result := Arrange(area, measurements, direction, gap)
	if len(result) == 0 || alignment == AlignStretch {
		return result
	}

	normalized := make([]Measure, len(measurements))
	for i, measurement := range measurements {
		normalized[i] = measurement.Normalize(area)
	}

	if direction == Horizontal {
		maxBaseline := uint16(0)
		if alignment == AlignBaseline {
			for _, measurement := range normalized {
				baseline := measurement.Baseline
				if baseline == 0 {
					baseline = measurement.IdealHeight
				}
				if baseline > maxBaseline {
					maxBaseline = baseline
				}
			}
		}
		for i := range result {
			height := normalized[i].IdealHeight
			if height > area.Height {
				height = area.Height
			}
			y := area.Y
			switch alignment {
			case AlignCenter:
				y += (area.Height - height) / 2
			case AlignEnd:
				y += area.Height - height
			case AlignBaseline:
				baseline := normalized[i].Baseline
				if baseline == 0 {
					baseline = normalized[i].IdealHeight
				}
				if baseline <= maxBaseline {
					y += maxBaseline - baseline
				}
			}
			result[i].Y = y
			result[i].Height = height
		}
		return result
	}

	for i := range result {
		width := normalized[i].IdealWidth
		if width > area.Width {
			width = area.Width
		}
		x := area.X
		switch alignment {
		case AlignCenter:
			x += (area.Width - width) / 2
		case AlignEnd:
			x += area.Width - width
		}
		result[i].X = x
		result[i].Width = width
	}
	return result
}
