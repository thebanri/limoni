package widgets

import (
	"image"
	"image/color"
	"sync"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/layout"
)

var (
	solidImageCache   = make(map[cell.Color]image.Image)
	solidImageCacheMu sync.Mutex
)

func getSolidImage(c cell.Color) image.Image {
	solidImageCacheMu.Lock()
	defer solidImageCacheMu.Unlock()

	if img, ok := solidImageCache[c]; ok {
		return img
	}

	r, g, b := c.RGB()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: r, G: g, B: b, A: 255})
	if len(solidImageCache) > 256 {
		clear(solidImageCache)
	}
	solidImageCache[c] = img
	return img
}

// Border direction masks (bitmask).
const (
	BorderNone   uint8 = 0
	BorderLeft   uint8 = 1 << 0
	BorderRight  uint8 = 1 << 1
	BorderTop    uint8 = 1 << 2
	BorderBottom uint8 = 1 << 3
	BorderAll    uint8 = BorderLeft | BorderRight | BorderTop | BorderBottom
)

// BorderSymbols is the set of glyphs (runes) used to draw a border.
type BorderSymbols struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
}

var (
	// SymbolsSingle is the thin, standard border.
	SymbolsSingle = BorderSymbols{
		Horizontal:  '─',
		Vertical:    '│',
		TopLeft:     '┌',
		TopRight:    '┐',
		BottomLeft:  '└',
		BottomRight: '┘',
	}
	// SymbolsDouble is the double-line border.
	SymbolsDouble = BorderSymbols{
		Horizontal:  '═',
		Vertical:    '║',
		TopLeft:     '╔',
		TopRight:    '╗',
		BottomLeft:  '╚',
		BottomRight: '╝',
	}
	// SymbolsThick is the thick border.
	SymbolsThick = BorderSymbols{
		Horizontal:  '━',
		Vertical:    '┃',
		TopLeft:     '┏',
		TopRight:    '┓',
		BottomLeft:  '┗',
		BottomRight: '┛',
	}
	// SymbolsRounded is the thin border with rounded corners.
	SymbolsRounded = BorderSymbols{
		Horizontal:  '─',
		Vertical:    '│',
		TopLeft:     '╭',
		TopRight:    '╮',
		BottomLeft:  '╰',
		BottomRight: '╯',
	}
	// SymbolsBlock is the thick border made of full block elements.
	SymbolsBlock = BorderSymbols{
		Horizontal:  '█',
		Vertical:    '█',
		TopLeft:     '█',
		TopRight:    '█',
		BottomLeft:  '█',
		BottomRight: '█',
	}
	// SymbolsOuterHalfBlock uses outer half-block elements for a
	// smooth, anti-aliased appearance on supported terminals.
	SymbolsOuterHalfBlock = BorderSymbols{
		Horizontal:  '▀', // top: ▀, bottom: ▄ (drawn via Vertical fallback)
		Vertical:    '▐',
		TopLeft:     '▛',
		TopRight:    '▜',
		BottomLeft:  '▙',
		BottomRight: '▟',
	}
	// SymbolsInnerHalfBlock uses inner half-block elements for a
	// thinner, inset border appearance on supported terminals.
	SymbolsInnerHalfBlock = BorderSymbols{
		Horizontal:  '▄', // top: ▄, bottom: ▀ (inverted from outer)
		Vertical:    '▌',
		TopLeft:     '▗',
		TopRight:    '▖',
		BottomLeft:  '▝',
		BottomRight: '▘',
	}
)

// Alignment is the alignment of a title or text.
type Alignment uint8

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// Insets describes CSS-like outer or inner spacing in terminal cells.
type Insets struct {
	Top    uint16
	Right  uint16
	Bottom uint16
	Left   uint16
}

func UniformInsets(value uint16) Insets {
	return Insets{Top: value, Right: value, Bottom: value, Left: value}
}

// Block is the most basic container widget: it can draw a border, fill a background
// and carry a Title on top.
// It hands its Child the shrunken inner area and passes the inherited style on.
type Block struct {
	// Title is the text shown on the block's top border.
	Title string
	// TitleAlignment sets where the title sits on the border (Left, Center, Right).
	TitleAlignment Alignment
	// TitleStyle sets the title's colour and style.
	TitleStyle cell.Style

	// Borders is the mask of which sides to draw (e.g. BorderAll or BorderTop|BorderBottom).
	Borders uint8
	// BorderSymbols are the glyphs the border is drawn with (e.g. SymbolsRounded).
	BorderSymbols BorderSymbols
	// MergeBorders joins this block's border with light box-drawing characters
	// already in the buffer, so two adjacent blocks share one edge and meet in
	// a junction (├ ┤ ┬ ┴ ┼) instead of one line overwriting the other.
	//
	// Off by default: it costs a read per border cell, and a block drawn over
	// unrelated line art would otherwise fuse with it.
	MergeBorders bool
	// BorderStyle sets the colour and style of the border lines.
	BorderStyle cell.Style

	// Margin is CSS-like space outside the block. The margin is not drawn.
	Margin Insets

	// Padding is CSS-like space between the content and the border.
	// The old separate PaddingLeft/Right/Top/Bottom fields are still supported.
	Padding       Insets
	PaddingLeft   uint16
	PaddingRight  uint16
	PaddingTop    uint16
	PaddingBottom uint16

	// Style sets the block's background fill and default style.
	Style cell.Style

	// Child is the component drawn inside the block.
	Child Widget
	// Opaque, when true, adds a solid-colour image layer behind the block so native images do not show through it.
	Opaque bool
}

// NewBlock returns a Block with BorderAll and SymbolsRounded.
func NewBlock() *Block {
	return &Block{
		Borders:       BorderAll,
		BorderSymbols: SymbolsRounded,
	}
}

// WithTitle sets the title text.
func (b *Block) WithTitle(title string) *Block {
	b.Title = title
	return b
}

// WithTitleAlign sets the title's alignment.
func (b *Block) WithTitleAlign(align Alignment) *Block {
	b.TitleAlignment = align
	return b
}

// WithTitleStyle sets the title's style.
func (b *Block) WithTitleStyle(style cell.Style) *Block {
	b.TitleStyle = style
	return b
}

// WithBorders sets which borders are drawn.
func (b *Block) WithBorders(borders uint8) *Block {
	b.Borders = borders
	return b
}

// Rounded selects the rounded-corner border symbols.
func (b *Block) Rounded() *Block {
	b.BorderSymbols = SymbolsRounded
	return b
}

// Single selects the standard thin border symbols.
func (b *Block) Single() *Block {
	b.BorderSymbols = SymbolsSingle
	return b
}

// Double selects the double-line border symbols.
func (b *Block) Double() *Block {
	b.BorderSymbols = SymbolsDouble
	return b
}

// Thick selects the thick border symbols.
func (b *Block) Thick() *Block {
	b.BorderSymbols = SymbolsThick
	return b
}

// BlockBorder selects the full-block border symbols.
func (b *Block) BlockBorder() *Block {
	b.BorderSymbols = SymbolsBlock
	return b
}

// WithBorderStyle sets the border line style.
func (b *Block) WithBorderStyle(style cell.Style) *Block {
	b.BorderStyle = style
	return b
}

// WithStyle sets the block's overall background style.
func (b *Block) WithStyle(style cell.Style) *Block {
	b.Style = style
	return b
}

// WithPadding sets the block's CSS-like inner spacing.
func (b *Block) WithPadding(top, right, bottom, left uint16) *Block {
	b.Padding = Insets{Top: top, Right: right, Bottom: bottom, Left: left}
	return b
}

// WithMargin sets the block's CSS-like outer spacing.
func (b *Block) WithMargin(top, right, bottom, left uint16) *Block {
	b.Margin = Insets{Top: top, Right: right, Bottom: bottom, Left: left}
	return b
}

// WithChild sets the component placed inside the block.
func (b *Block) WithChild(child Widget) *Block {
	b.Child = child
	return b
}

// Draw draws the block and its borders, fills its background and draws its Child.
func (b Block) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := insetRect(ctx.Area, b.Margin)
	if area.Width == 0 || area.Height == 0 {
		return
	}

	// Work out the block's final style (the inherited style merged with this block's)
	blockStyle := ctx.Style.Merge(b.Style)
	if b.Style == (cell.Style{}) && ctx.ThemeStyle != nil {
		blockStyle = blockStyle.Merge(ctx.ThemeStyle("surface"))
	}
	borderStyle := blockStyle.Merge(b.BorderStyle)
	if b.BorderStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
		borderStyle = blockStyle.Merge(ctx.ThemeStyle("border"))
	}

	// Pass 1: fill the block's background
	//
	// When merging borders, cells that will receive a border of this block are
	// left alone: the fill would erase the neighbouring block's glyph before
	// putBorder ever got to merge with it. Those cells are fully restyled by
	// putBorder a moment later, so nothing is left unpainted.
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			if b.MergeBorders && b.coversWithBorder(area, x, y) {
				continue
			}
			buf.SetCellDirect(x, y, cell.Cell{Content: ' ', Style: blockStyle})
		}
	}

	if b.Opaque && blockStyle.Bg.Type() != cell.ColorDefault && ctx.RegisterImage != nil {
		proto := imageProtocol(ctx)
		if proto != graphics.ProtocolHalfBlock {
			solidImg := getSolidImage(blockStyle.Bg)
			ctx.RegisterImage(area, solidImg, -99, false) // Marker for frame.go to map ZIndex
		}
	}

	// Pass 2: draw the borders
	hasL := (b.Borders & BorderLeft) != 0
	hasR := (b.Borders & BorderRight) != 0
	hasT := (b.Borders & BorderTop) != 0
	hasB := (b.Borders & BorderBottom) != 0

	sym := b.BorderSymbols
	// If no border symbols are set, use the default thin line
	if sym.Horizontal == 0 {
		sym = SymbolsSingle
	}

	// Draw the horizontal lines
	if hasT {
		for x := area.X + 1; x < area.X+area.Width-1; x++ {
			b.putBorder(buf, x, area.Y, sym.Horizontal, borderStyle)
		}
	}
	if hasB {
		for x := area.X + 1; x < area.X+area.Width-1; x++ {
			b.putBorder(buf, x, area.Y+area.Height-1, sym.Horizontal, borderStyle)
		}
	}

	// Draw the vertical lines
	if hasL {
		for y := area.Y + 1; y < area.Y+area.Height-1; y++ {
			b.putBorder(buf, area.X, y, sym.Vertical, borderStyle)
		}
	}
	if hasR {
		for y := area.Y + 1; y < area.Y+area.Height-1; y++ {
			b.putBorder(buf, area.X+area.Width-1, y, sym.Vertical, borderStyle)
		}
	}

	// Draw the corner joins
	if hasT && hasL {
		b.putBorder(buf, area.X, area.Y, sym.TopLeft, borderStyle)
	}
	if hasT && hasR {
		b.putBorder(buf, area.X+area.Width-1, area.Y, sym.TopRight, borderStyle)
	}
	if hasB && hasL {
		b.putBorder(buf, area.X, area.Y+area.Height-1, sym.BottomLeft, borderStyle)
	}
	if hasB && hasR {
		b.putBorder(buf, area.X+area.Width-1, area.Y+area.Height-1, sym.BottomRight, borderStyle)
	}

	// Pass 3: draw the title on the top border
	if b.Title != "" && hasT && area.Width > 4 {
		titleStyle := blockStyle.Merge(b.TitleStyle)
		rawTitleWidth := uint16(cell.StringWidth(b.Title))
		maxTitleWidth := area.Width - 4
		var titleWidth uint16
		if rawTitleWidth+2 <= maxTitleWidth {
			titleWidth = rawTitleWidth + 2
		} else {
			titleWidth = maxTitleWidth
		}

		var titleX uint16
		switch b.TitleAlignment {
		case AlignLeft:
			titleX = area.X + 2
		case AlignCenter:
			if area.Width > titleWidth {
				titleX = area.X + (area.Width-titleWidth)/2
			} else {
				titleX = area.X + 2
			}
		case AlignRight:
			if area.Width >= 2+titleWidth {
				titleX = area.X + area.Width - 2 - titleWidth
			} else {
				titleX = area.X + 2
			}
		}

		curX := titleX
		buf.SetCell(curX, area.Y, cell.Cell{Content: ' ', Style: titleStyle})
		curX++
		if titleWidth > 2 {
			written := buf.SetStringWithin(curX, area.Y, b.Title, titleStyle, titleWidth-2)
			curX += written
		}
		closingX := titleX + titleWidth - 1
		for curX < closingX {
			buf.SetCell(curX, area.Y, cell.Cell{Content: ' ', Style: titleStyle})
			curX++
		}
		buf.SetCell(closingX, area.Y, cell.Cell{Content: ' ', Style: titleStyle})
	}

	// Pass 4: draw the Child
	if b.Child != nil {
		var offsetL, offsetR, offsetT, offsetB uint16
		if hasL {
			offsetL = 1
		}
		if hasR {
			offsetR = 1
		}
		if hasT {
			offsetT = 1
		}
		if hasB {
			offsetB = 1
		}

		// Add the border and padding allowances
		left := offsetL + b.Padding.Left + b.PaddingLeft
		right := offsetR + b.Padding.Right + b.PaddingRight
		top := offsetT + b.Padding.Top + b.PaddingTop
		bottom := offsetB + b.Padding.Bottom + b.PaddingBottom

		if area.Width > left+right && area.Height > top+bottom {
			childArea := cell.Rect{
				X:      area.X + left,
				Y:      area.Y + top,
				Width:  area.Width - left - right,
				Height: area.Height - top - bottom,
			}
			// The child gets the whole context with a narrower area and the
			// block's style. Copying fields one by one dropped every field
			// added to Context later — the click actions and wheel scrolling
			// among them.
			childCtx := ctx
			childCtx.Area = childArea
			childCtx.Style = blockStyle
			b.Child.Draw(childCtx, buf)
			if ctx.Describe != nil {
				ctx.Describe(b.Child, childArea)
			}
		}
	}
}

func insetRect(area cell.Rect, insets Insets) cell.Rect {
	if area.Width <= insets.Left+insets.Right || area.Height <= insets.Top+insets.Bottom {
		return cell.Rect{}
	}
	return cell.Rect{
		X:      area.X + insets.Left,
		Y:      area.Y + insets.Top,
		Width:  area.Width - insets.Left - insets.Right,
		Height: area.Height - insets.Top - insets.Bottom,
	}
}

// Inner returns the usable content area inside the block (accounting for borders, padding, and margins).
func (b Block) Inner(area cell.Rect) cell.Rect {
	area = insetRect(area, b.Margin)
	borders := b.Borders
	var offsetL, offsetR, offsetT, offsetB uint16
	if (borders & BorderLeft) != 0 {
		offsetL = 1
	}
	if (borders & BorderRight) != 0 {
		offsetR = 1
	}
	if (borders & BorderTop) != 0 {
		offsetT = 1
	}
	if (borders & BorderBottom) != 0 {
		offsetB = 1
	}
	left := offsetL + b.Padding.Left + b.PaddingLeft
	right := offsetR + b.Padding.Right + b.PaddingRight
	top := offsetT + b.Padding.Top + b.PaddingTop
	bottom := offsetB + b.Padding.Bottom + b.PaddingBottom

	if area.Width <= left+right || area.Height <= top+bottom {
		return cell.Rect{}
	}
	return cell.Rect{
		X:      area.X + left,
		Y:      area.Y + top,
		Width:  area.Width - left - right,
		Height: area.Height - top - bottom,
	}
}

// SizeHint works out the space this block would like, allowing for its border, margin and padding.
func (b Block) SizeHint(maxArea cell.Rect) (width, height uint16) {
	var offsetL, offsetR, offsetT, offsetB uint16
	if (b.Borders & BorderLeft) != 0 {
		offsetL = 1
	}
	if (b.Borders & BorderRight) != 0 {
		offsetR = 1
	}
	if (b.Borders & BorderTop) != 0 {
		offsetT = 1
	}
	if (b.Borders & BorderBottom) != 0 {
		offsetB = 1
	}

	overheadW := offsetL + offsetR + b.Padding.Left + b.Padding.Right + b.PaddingLeft + b.PaddingRight + b.Margin.Left + b.Margin.Right
	overheadH := offsetT + offsetB + b.Padding.Top + b.Padding.Bottom + b.PaddingTop + b.PaddingBottom + b.Margin.Top + b.Margin.Bottom

	// Minimum width for the title to fit
	titleLen := uint16(0)
	if b.Title != "" {
		titleLen = uint16(cell.StringWidth(b.Title)) + 4 // " Title " + corners
	}

	if b.Child != nil {
		var childMaxW, childMaxH uint16
		if maxArea.Width > overheadW {
			childMaxW = maxArea.Width - overheadW
		}
		if maxArea.Height > overheadH {
			childMaxH = maxArea.Height - overheadH
		}

		// Ask the Child for its preferred size
		childW, childH := b.Child.SizeHint(cell.NewRect(maxArea.X, maxArea.Y, childMaxW, childMaxH))
		width = childW + overheadW
		height = childH + overheadH
	} else {
		width = overheadW
		height = overheadH
	}

	// If the title is wider, widen to fit the title
	if width < titleLen {
		width = titleLen
	}

	// Do not go past the bounds
	if width > maxArea.Width {
		width = maxArea.Width
	}
	if height > maxArea.Height {
		height = maxArea.Height
	}

	return width, height
}

// Measure provides explicit size negotiation for Block.
func (b Block) Measure(maxArea cell.Rect) layout.Measure {
	w, h := b.SizeHint(maxArea)
	return layout.Measure{
		IdealWidth:  w,
		IdealHeight: h,
		MaxWidth:    maxArea.Width,
		MaxHeight:   maxArea.Height,
		Overflow:    layout.OverflowClip,
	}
}

// putBorder writes one border glyph, merging it with whatever light
// box-drawing character the cell already holds when MergeBorders is set.
//
// The merge is possible at all because a cell grid still has the previous
// character to consult; a renderer that concatenates strings has already
// overwritten it.
func (b Block) putBorder(buf *buffer.Buffer, x, y uint16, r rune, style cell.Style) {
	if b.MergeBorders {
		if existing := buf.Get(x, y); existing != nil {
			r = cell.MergeBoxDrawing(existing.Content, r)
		}
	}
	buf.SetCellDirect(x, y, cell.Cell{Content: r, Style: style})
}

// coversWithBorder reports whether this block will draw a border glyph at the
// given cell, given which edges are enabled.
func (b Block) coversWithBorder(area cell.Rect, x, y uint16) bool {
	onLeft := x == area.X && b.Borders&BorderLeft != 0
	onRight := x == area.X+area.Width-1 && b.Borders&BorderRight != 0
	onTop := y == area.Y && b.Borders&BorderTop != 0
	onBottom := y == area.Y+area.Height-1 && b.Borders&BorderBottom != 0
	return onLeft || onRight || onTop || onBottom
}
