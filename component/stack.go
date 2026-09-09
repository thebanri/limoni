package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// JustifyContent defines how extra space is distributed along the primary axis
// when children do not consume all available space.
type JustifyContent uint8

const (
	JustifyStart JustifyContent = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
	JustifySpaceAround
	JustifySpaceEvenly
)

// AlignItems defines cross-axis alignment of children inside a StackLayout.
type AlignItems uint8

const (
	AlignItemsStretch AlignItems = iota
	AlignItemsStart
	AlignItemsCenter
	AlignItemsEnd
)

// StackLayout arranges components linearly along a primary axis.
type StackLayout struct {
	isVertical bool
	gap        uint16
	justify    JustifyContent
	alignItems AlignItems
	children   []Component
}

// VStack creates a vertical stack arranging children top-to-bottom.
func VStack(children ...Component) *StackLayout {
	return &StackLayout{
		isVertical: true,
		gap:        0,
		justify:    JustifyStart,
		alignItems: AlignItemsStretch,
		children:   children,
	}
}

// HStack creates a horizontal stack arranging children left-to-right.
func HStack(children ...Component) *StackLayout {
	return &StackLayout{
		isVertical: false,
		gap:        0,
		justify:    JustifyStart,
		alignItems: AlignItemsStretch,
		children:   children,
	}
}

// WithGap sets the cell gap between adjacent children.
func (s *StackLayout) WithGap(gap uint16) *StackLayout {
	s.gap = gap
	return s
}

// WithJustify sets primary axis distribution (Start, Center, End, SpaceBetween, SpaceAround, SpaceEvenly).
func (s *StackLayout) WithJustify(j JustifyContent) *StackLayout {
	s.justify = j
	return s
}

// WithAlignItems sets cross-axis alignment (Stretch, Start, Center, End).
func (s *StackLayout) WithAlignItems(a AlignItems) *StackLayout {
	s.alignItems = a
	return s
}

func (s *StackLayout) Draw(ctx cell.Context, buf *buffer.Buffer) {
	n := len(s.children)
	if n == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}

	count := n
	if count > 32 {
		count = 32
	}

	// Zero-allocation scratch arrays for up to 32 items
	var sizes [32]uint16
	var offsets [32]uint16
	var crossSizes [32]uint16
	var crossOffsets [32]uint16
	var props [32]LayoutProps

	availPrimary := ctx.Area.Height
	crossAvail := ctx.Area.Width
	if !s.isVertical {
		availPrimary = ctx.Area.Width
		crossAvail = ctx.Area.Height
	}

	var totalFixed uint16
	var totalFlex uint16

	for i := 0; i < count; i++ {
		props[i] = s.children[i].LayoutInfo(ctx.Area)
		if props[i].Flex > 0 {
			totalFlex += props[i].Flex
		} else {
			fixedSize := props[i].MinHeight
			if !s.isVertical {
				fixedSize = props[i].MinWidth
			}
			sizes[i] = fixedSize
			totalFixed += fixedSize
		}
	}

	totalBaseGaps := uint16(0)
	if count > 1 {
		totalBaseGaps = uint16(count-1) * s.gap
	}

	var netAvailForItems uint16
	if availPrimary > totalBaseGaps {
		netAvailForItems = availPrimary - totalBaseGaps
	}

	// Flex distribution pass
	if totalFlex > 0 && netAvailForItems > totalFixed {
		remainingForFlex := netAvailForItems - totalFixed
		var allocatedFlex uint16
		var flexItemsCount uint16
		for i := 0; i < count; i++ {
			if props[i].Flex > 0 {
				flexItemsCount++
			}
		}
		var flexIdx uint16
		for i := 0; i < count; i++ {
			if props[i].Flex > 0 {
				flexIdx++
				var sz uint16
				if flexIdx == flexItemsCount {
					sz = remainingForFlex - allocatedFlex
				} else {
					sz = (remainingForFlex * props[i].Flex) / totalFlex
					allocatedFlex += sz
				}
				sizes[i] = sz
			}
		}
	}

	// Calculate total used primary space
	var totalUsedPrimary uint16
	for i := 0; i < count; i++ {
		totalUsedPrimary += sizes[i]
	}
	totalUsedPrimary += totalBaseGaps

	var leftoverPrimary uint16
	if availPrimary > totalUsedPrimary {
		leftoverPrimary = availPrimary - totalUsedPrimary
	}

	// Calculate primary axis offsets with JustifyContent
	var startOffset uint16
	var extraGapPerItem uint16
	var extraGapRemainder uint16

	if leftoverPrimary > 0 {
		switch s.justify {
		case JustifyCenter:
			startOffset = leftoverPrimary / 2
		case JustifyEnd:
			startOffset = leftoverPrimary
		case JustifySpaceBetween:
			if count > 1 {
				extraGapPerItem = leftoverPrimary / uint16(count-1)
				extraGapRemainder = leftoverPrimary % uint16(count-1)
			}
		case JustifySpaceAround:
			if count > 0 {
				unit := leftoverPrimary / uint16(count)
				startOffset = unit / 2
				extraGapPerItem = unit
				extraGapRemainder = leftoverPrimary % uint16(count)
			}
		case JustifySpaceEvenly:
			if count > 0 {
				unit := leftoverPrimary / uint16(count+1)
				startOffset = unit
				extraGapPerItem = unit
				extraGapRemainder = leftoverPrimary % uint16(count+1)
			}
		default: // JustifyStart
			startOffset = 0
		}
	}

	currentOffset := startOffset
	for i := 0; i < count; i++ {
		offsets[i] = currentOffset
		gap := s.gap
		if leftoverPrimary > 0 {
			switch s.justify {
			case JustifySpaceBetween, JustifySpaceAround, JustifySpaceEvenly:
				gap += extraGapPerItem
				if extraGapRemainder > 0 {
					gap++
					extraGapRemainder--
				}
			}
		}
		currentOffset += sizes[i] + gap
	}

	// Calculate cross-axis sizes and offsets with AlignItems
	for i := 0; i < count; i++ {
		childPreferredCross := props[i].MinWidth
		if !s.isVertical {
			childPreferredCross = props[i].MinHeight
		}

		if s.alignItems == AlignItemsStretch || childPreferredCross == 0 || childPreferredCross >= crossAvail {
			crossSizes[i] = crossAvail
			crossOffsets[i] = 0
		} else {
			crossSizes[i] = childPreferredCross
			switch s.alignItems {
			case AlignItemsStart:
				crossOffsets[i] = 0
			case AlignItemsCenter:
				crossOffsets[i] = (crossAvail - childPreferredCross) / 2
			case AlignItemsEnd:
				crossOffsets[i] = crossAvail - childPreferredCross
			default:
				crossSizes[i] = crossAvail
				crossOffsets[i] = 0
			}
		}
	}

	// Render children
	for i := 0; i < count; i++ {
		var childArea cell.Rect
		if s.isVertical {
			childH := sizes[i]
			offsetY := offsets[i]
			if offsetY+childH > ctx.Area.Height {
				if offsetY < ctx.Area.Height {
					childH = ctx.Area.Height - offsetY
				} else {
					childH = 0
				}
			}
			childArea = cell.Rect{
				X:      ctx.Area.X + crossOffsets[i],
				Y:      ctx.Area.Y + offsetY,
				Width:  crossSizes[i],
				Height: childH,
			}
		} else {
			childW := sizes[i]
			offsetX := offsets[i]
			if offsetX+childW > ctx.Area.Width {
				if offsetX < ctx.Area.Width {
					childW = ctx.Area.Width - offsetX
				} else {
					childW = 0
				}
			}
			childArea = cell.Rect{
				X:      ctx.Area.X + offsetX,
				Y:      ctx.Area.Y + crossOffsets[i],
				Width:  childW,
				Height: crossSizes[i],
			}
		}

		if childArea.Width > 0 && childArea.Height > 0 {
			childCtx := ctx
			childCtx.Area = childArea
			s.children[i].Draw(childCtx, buf)
		}
	}
}

func (s *StackLayout) LayoutInfo(maxArea cell.Rect) LayoutProps {
	var totalW, totalH uint16
	var maxFlex uint16

	for _, c := range s.children {
		p := c.LayoutInfo(maxArea)
		if s.isVertical {
			totalH += p.MinHeight
			if p.MinWidth > totalW {
				totalW = p.MinWidth
			}
		} else {
			totalW += p.MinWidth
			if p.MinHeight > totalH {
				totalH = p.MinHeight
			}
		}
		if p.Flex > maxFlex {
			maxFlex = p.Flex
		}
	}

	if len(s.children) > 1 {
		gapTotal := uint16(len(s.children)-1) * s.gap
		if s.isVertical {
			totalH += gapTotal
		} else {
			totalW += gapTotal
		}
	}

	return LayoutProps{
		MinWidth:  totalW,
		MinHeight: totalH,
		Flex:      maxFlex,
	}
}

func (s *StackLayout) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	props := s.LayoutInfo(maxArea)
	return props.MinWidth, props.MinHeight
}

// HandleEvent dispatches events to children within their layout boundaries.
func (s *StackLayout) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	n := len(s.children)
	if n == 0 || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return false
	}

	count := n
	if count > 32 {
		count = 32
	}

	var sizes [32]uint16
	var offsets [32]uint16
	var crossSizes [32]uint16
	var crossOffsets [32]uint16
	var props [32]LayoutProps

	availPrimary := ctx.Area.Height
	crossAvail := ctx.Area.Width
	if !s.isVertical {
		availPrimary = ctx.Area.Width
		crossAvail = ctx.Area.Height
	}

	var totalFixed uint16
	var totalFlex uint16

	for i := 0; i < count; i++ {
		props[i] = s.children[i].LayoutInfo(ctx.Area)
		if props[i].Flex > 0 {
			totalFlex += props[i].Flex
		} else {
			fixedSize := props[i].MinHeight
			if !s.isVertical {
				fixedSize = props[i].MinWidth
			}
			sizes[i] = fixedSize
			totalFixed += fixedSize
		}
	}

	totalBaseGaps := uint16(0)
	if count > 1 {
		totalBaseGaps = uint16(count-1) * s.gap
	}

	var netAvailForItems uint16
	if availPrimary > totalBaseGaps {
		netAvailForItems = availPrimary - totalBaseGaps
	}

	if totalFlex > 0 && netAvailForItems > totalFixed {
		remainingForFlex := netAvailForItems - totalFixed
		var allocatedFlex uint16
		var flexItemsCount uint16
		for i := 0; i < count; i++ {
			if props[i].Flex > 0 {
				flexItemsCount++
			}
		}
		var flexIdx uint16
		for i := 0; i < count; i++ {
			if props[i].Flex > 0 {
				flexIdx++
				var sz uint16
				if flexIdx == flexItemsCount {
					sz = remainingForFlex - allocatedFlex
				} else {
					sz = (remainingForFlex * props[i].Flex) / totalFlex
					allocatedFlex += sz
				}
				sizes[i] = sz
			}
		}
	}

	var totalUsedPrimary uint16
	for i := 0; i < count; i++ {
		totalUsedPrimary += sizes[i]
	}
	totalUsedPrimary += totalBaseGaps

	var leftoverPrimary uint16
	if availPrimary > totalUsedPrimary {
		leftoverPrimary = availPrimary - totalUsedPrimary
	}

	var startOffset uint16
	var extraGapPerItem uint16
	var extraGapRemainder uint16

	if leftoverPrimary > 0 {
		switch s.justify {
		case JustifyCenter:
			startOffset = leftoverPrimary / 2
		case JustifyEnd:
			startOffset = leftoverPrimary
		case JustifySpaceBetween:
			if count > 1 {
				extraGapPerItem = leftoverPrimary / uint16(count-1)
				extraGapRemainder = leftoverPrimary % uint16(count-1)
			}
		case JustifySpaceAround:
			if count > 0 {
				unit := leftoverPrimary / uint16(count)
				startOffset = unit / 2
				extraGapPerItem = unit
				extraGapRemainder = leftoverPrimary % uint16(count)
			}
		case JustifySpaceEvenly:
			if count > 0 {
				unit := leftoverPrimary / uint16(count+1)
				startOffset = unit
				extraGapPerItem = unit
				extraGapRemainder = leftoverPrimary % uint16(count+1)
			}
		default:
			startOffset = 0
		}
	}

	currentOffset := startOffset
	for i := 0; i < count; i++ {
		offsets[i] = currentOffset
		gap := s.gap
		if leftoverPrimary > 0 {
			switch s.justify {
			case JustifySpaceBetween, JustifySpaceAround, JustifySpaceEvenly:
				gap += extraGapPerItem
				if extraGapRemainder > 0 {
					gap++
					extraGapRemainder--
				}
			}
		}
		currentOffset += sizes[i] + gap
	}

	for i := 0; i < count; i++ {
		childPreferredCross := props[i].MinWidth
		if !s.isVertical {
			childPreferredCross = props[i].MinHeight
		}

		if s.alignItems == AlignItemsStretch || childPreferredCross == 0 || childPreferredCross >= crossAvail {
			crossSizes[i] = crossAvail
			crossOffsets[i] = 0
		} else {
			crossSizes[i] = childPreferredCross
			switch s.alignItems {
			case AlignItemsStart:
				crossOffsets[i] = 0
			case AlignItemsCenter:
				crossOffsets[i] = (crossAvail - childPreferredCross) / 2
			case AlignItemsEnd:
				crossOffsets[i] = crossAvail - childPreferredCross
			default:
				crossSizes[i] = crossAvail
				crossOffsets[i] = 0
			}
		}
	}

	// For mouse events: hit test by area
	if ev.Type == driver.EventMouse {
		for i := 0; i < count; i++ {
			var childArea cell.Rect
			if s.isVertical {
				childH := sizes[i]
				offsetY := offsets[i]
				if offsetY+childH > ctx.Area.Height {
					if offsetY < ctx.Area.Height {
						childH = ctx.Area.Height - offsetY
					} else {
						childH = 0
					}
				}
				childArea = cell.Rect{
					X:      ctx.Area.X + crossOffsets[i],
					Y:      ctx.Area.Y + offsetY,
					Width:  crossSizes[i],
					Height: childH,
				}
			} else {
				childW := sizes[i]
				offsetX := offsets[i]
				if offsetX+childW > ctx.Area.Width {
					if offsetX < ctx.Area.Width {
						childW = ctx.Area.Width - offsetX
					} else {
						childW = 0
					}
				}
				childArea = cell.Rect{
					X:      ctx.Area.X + offsetX,
					Y:      ctx.Area.Y + crossOffsets[i],
					Width:  childW,
					Height: crossSizes[i],
				}
			}

			if childArea.Contains(ev.Mouse.X, ev.Mouse.Y) {
				childCtx := ctx
				childCtx.Area = childArea
				if DispatchEvent(s.children[i], childCtx, ev) {
					return true
				}
			}
		}
		return false
	}

	// For non-mouse events (Key, Resize, Focus, Paste): dispatch to children sequentially
	for i := 0; i < count; i++ {
		childCtx := ctx
		if DispatchEvent(s.children[i], childCtx, ev) {
			return true
		}
	}

	return false
}
