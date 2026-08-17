package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type text struct {
	value string
	style cell.Style
}

func (t text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	lines := strings.Split(t.value, "\n")
	for i, line := range lines {
		if uint16(i) >= ctx.Area.Height {
			break
		}
		buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(i), line, ctx.Style.Merge(t.style))
	}
}

func (t text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	lines := strings.Split(t.value, "\n")
	maxW := 0
	for _, l := range lines {
		w := len([]rune(l))
		if w > maxW {
			maxW = w
		}
	}
	return uint16(maxW), uint16(len(lines))
}

type ToolType int

const (
	ToolBrush ToolType = iota
	ToolEraser
	ToolLine
	ToolCircle
	ToolRect
)

var toolNames = []string{"Pen (Brush)", "Eraser", "Line", "Circle", "Rect"}

// 10 Core Preset Swatches
type ColorSwatch struct {
	Name  string
	Color cell.Color
	Hex   string
	Key   rune
}

var defaultSwatches = []ColorSwatch{
	{Name: "White", Color: cell.NewColorRGB(255, 255, 255), Hex: "#FFFFFF", Key: '1'},
	{Name: "Red", Color: cell.NewColorRGB(255, 59, 48), Hex: "#FF3B30", Key: '2'},
	{Name: "Orange", Color: cell.NewColorRGB(255, 149, 0), Hex: "#FF9500", Key: '3'},
	{Name: "Yellow", Color: cell.NewColorRGB(255, 204, 0), Hex: "#FFCC00", Key: '4'},
	{Name: "Green", Color: cell.NewColorRGB(52, 199, 89), Hex: "#34C759", Key: '5'},
	{Name: "Cyan", Color: cell.NewColorRGB(0, 199, 190), Hex: "#00C7BE", Key: '6'},
	{Name: "Blue", Color: cell.NewColorRGB(0, 122, 255), Hex: "#007AFF", Key: '7'},
	{Name: "Purple", Color: cell.NewColorRGB(175, 82, 222), Hex: "#AF52DE", Key: '8'},
	{Name: "Pink", Color: cell.NewColorRGB(255, 45, 85), Hex: "#FF2D55", Key: '9'},
	{Name: "Dark Gray", Color: cell.NewColorRGB(100, 105, 120), Hex: "#646978", Key: '0'},
}

// 24-Color Rainbow Matrix for Custom Modal
var modalPaletteGrid = [][]cell.Color{
	{
		cell.NewColorRGB(255, 255, 255), cell.NewColorRGB(210, 210, 215), cell.NewColorRGB(142, 142, 147),
		cell.NewColorRGB(99, 99, 102), cell.NewColorRGB(58, 58, 60), cell.NewColorRGB(0, 0, 0),
	},
	{
		cell.NewColorRGB(255, 69, 58), cell.NewColorRGB(255, 159, 10), cell.NewColorRGB(255, 214, 10),
		cell.NewColorRGB(48, 209, 88), cell.NewColorRGB(100, 210, 255), cell.NewColorRGB(10, 132, 255),
	},
	{
		cell.NewColorRGB(191, 90, 242), cell.NewColorRGB(255, 55, 95), cell.NewColorRGB(255, 100, 130),
		cell.NewColorRGB(162, 132, 94), cell.NewColorRGB(0, 245, 160), cell.NewColorRGB(0, 255, 255),
	},
	{
		cell.NewColorRGB(255, 0, 85), cell.NewColorRGB(255, 85, 0), cell.NewColorRGB(255, 255, 0),
		cell.NewColorRGB(0, 255, 0), cell.NewColorRGB(0, 170, 255), cell.NewColorRGB(170, 0, 255),
	},
}

var brailleDotMask = [4][2]byte{
	{0x01, 0x08}, // y=0: x=0 -> Dot 1, x=1 -> Dot 4
	{0x02, 0x10}, // y=1: x=0 -> Dot 2, x=1 -> Dot 5
	{0x04, 0x20}, // y=2: x=0 -> Dot 3, x=1 -> Dot 6
	{0x40, 0x80}, // y=3: x=0 -> Dot 7, x=1 -> Dot 8
}

// BraillePointCanvas provides 2x4 sub-pixel Braille dot matrix drawing
type BraillePointCanvas struct {
	Width      int // cell width
	Height     int // cell height
	VirtWidth  int // Width * 2
	VirtHeight int // Height * 4
	Grid       []byte
	Colors     []cell.Color
	UndoGrids  [][]byte
	UndoColors [][]cell.Color
	ActiveDots int
}

func NewBraillePointCanvas(w, h int) *BraillePointCanvas {
	if w <= 0 {
		w = 10
	}
	if h <= 0 {
		h = 10
	}
	cells := w * h
	return &BraillePointCanvas{
		Width:      w,
		Height:     h,
		VirtWidth:  w * 2,
		VirtHeight: h * 4,
		Grid:       make([]byte, cells),
		Colors:     make([]cell.Color, cells),
		UndoGrids:  make([][]byte, 0, 25),
		UndoColors: make([][]cell.Color, 0, 25),
	}
}

func (c *BraillePointCanvas) Resize(newW, newH int) {
	if newW <= 0 || newH <= 0 || (newW == c.Width && newH == c.Height) {
		return
	}
	newCells := newW * newH
	newGrid := make([]byte, newCells)
	newColors := make([]cell.Color, newCells)

	for y := 0; y < c.Height && y < newH; y++ {
		for x := 0; x < c.Width && x < newW; x++ {
			oldIdx := y*c.Width + x
			newIdx := y*newW + x
			newGrid[newIdx] = c.Grid[oldIdx]
			newColors[newIdx] = c.Colors[oldIdx]
		}
	}

	c.Width = newW
	c.Height = newH
	c.VirtWidth = newW * 2
	c.VirtHeight = newH * 4
	c.Grid = newGrid
	c.Colors = newColors
}

func (c *BraillePointCanvas) SaveUndo() {
	gridSnap := make([]byte, len(c.Grid))
	copy(gridSnap, c.Grid)
	colSnap := make([]cell.Color, len(c.Colors))
	copy(colSnap, c.Colors)

	if len(c.UndoGrids) >= 25 {
		c.UndoGrids = c.UndoGrids[1:]
		c.UndoColors = c.UndoColors[1:]
	}
	c.UndoGrids = append(c.UndoGrids, gridSnap)
	c.UndoColors = append(c.UndoColors, colSnap)
}

func (c *BraillePointCanvas) Undo() {
	if len(c.UndoGrids) == 0 {
		return
	}
	lastGrid := c.UndoGrids[len(c.UndoGrids)-1]
	lastCols := c.UndoColors[len(c.UndoColors)-1]
	c.UndoGrids = c.UndoGrids[:len(c.UndoGrids)-1]
	c.UndoColors = c.UndoColors[:len(c.UndoColors)-1]

	copy(c.Grid, lastGrid)
	copy(c.Colors, lastCols)
	c.recountDots()
}

func (c *BraillePointCanvas) Clear() {
	c.SaveUndo()
	for i := range c.Grid {
		c.Grid[i] = 0
		c.Colors[i] = cell.NewColorDefault()
	}
	c.ActiveDots = 0
}

func (c *BraillePointCanvas) recountDots() {
	count := 0
	for _, b := range c.Grid {
		for i := 0; i < 8; i++ {
			if (b & (1 << i)) != 0 {
				count++
			}
		}
	}
	c.ActiveDots = count
}

func (c *BraillePointCanvas) SetDot(vx, vy int, col cell.Color) {
	if vx < 0 || vx >= c.VirtWidth || vy < 0 || vy >= c.VirtHeight {
		return
	}
	cx := vx / 2
	cy := vy / 4
	subX := vx % 2
	subY := vy % 4

	cellIdx := cy*c.Width + cx
	if cellIdx < 0 || cellIdx >= len(c.Grid) {
		return
	}

	mask := brailleDotMask[subY][subX]
	if (c.Grid[cellIdx] & mask) == 0 {
		c.ActiveDots++
	}
	c.Grid[cellIdx] |= mask
	c.Colors[cellIdx] = col
}

func (c *BraillePointCanvas) EraseDot(vx, vy int) {
	if vx < 0 || vx >= c.VirtWidth || vy < 0 || vy >= c.VirtHeight {
		return
	}
	cx := vx / 2
	cy := vy / 4
	subX := vx % 2
	subY := vy % 4

	cellIdx := cy*c.Width + cx
	if cellIdx < 0 || cellIdx >= len(c.Grid) {
		return
	}

	mask := brailleDotMask[subY][subX]
	if (c.Grid[cellIdx] & mask) != 0 {
		c.ActiveDots--
	}
	c.Grid[cellIdx] &= ^mask
}

func (c *BraillePointCanvas) DrawDotBrush(vx, vy int, size int, col cell.Color) {
	for dy := -size + 1; dy < size; dy++ {
		for dx := -size + 1; dx < size; dx++ {
			if dx*dx+dy*dy <= size*size {
				c.SetDot(vx+dx, vy+dy, col)
			}
		}
	}
}

func (c *BraillePointCanvas) EraseDotBrush(vx, vy int, size int) {
	for dy := -size + 1; dy < size; dy++ {
		for dx := -size + 1; dx < size; dx++ {
			if dx*dx+dy*dy <= size*size {
				c.EraseDot(vx+dx, vy+dy)
			}
		}
	}
}

func (c *BraillePointCanvas) DrawLine(x0, y0, x1, y1 int, col cell.Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy

	for {
		c.SetDot(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (c *BraillePointCanvas) DrawCircle(cx, cy, r int, col cell.Color) {
	x := 0
	y := r
	d := 3 - 2*r
	drawCirclePoints := func(cx, cy, x, y int) {
		c.SetDot(cx+x, cy+y, col)
		c.SetDot(cx-x, cy+y, col)
		c.SetDot(cx+x, cy-y, col)
		c.SetDot(cx-x, cy-y, col)
		c.SetDot(cx+y, cy+x, col)
		c.SetDot(cx-y, cy+x, col)
		c.SetDot(cx+y, cy-x, col)
		c.SetDot(cx-y, cy-x, col)
	}
	drawCirclePoints(cx, cy, x, y)
	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		drawCirclePoints(cx, cy, x, y)
	}
}

func (c *BraillePointCanvas) DrawRect(x0, y0, x1, y1 int, col cell.Color) {
	minX, maxX := min(x0, x1), max(x0, x1)
	minY, maxY := min(y0, y1), max(y0, y1)
	for x := minX; x <= maxX; x++ {
		c.SetDot(x, minY, col)
		c.SetDot(x, maxY, col)
	}
	for y := minY; y <= maxY; y++ {
		c.SetDot(minX, y, col)
		c.SetDot(maxX, y, col)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func parseHexColor(hexStr string) (cell.Color, bool) {
	hexStr = strings.TrimPrefix(strings.TrimSpace(hexStr), "#")
	if len(hexStr) == 3 {
		hexStr = string([]byte{hexStr[0], hexStr[0], hexStr[1], hexStr[1], hexStr[2], hexStr[2]})
	}
	if len(hexStr) != 6 {
		return cell.NewColorDefault(), false
	}
	r, err1 := strconv.ParseUint(hexStr[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hexStr[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hexStr[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return cell.NewColorDefault(), false
	}
	return cell.NewColorRGB(uint8(r), uint8(g), uint8(b)), true
}

func colorToHex(col cell.Color) string {
	r, g, b := col.RGB()
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

type PaintApp struct {
	Canvas        *BraillePointCanvas
	ActiveTool    ToolType
	ActiveColor   cell.Color
	SelectedIdx   int
	CustomColor   cell.Color
	BrushSize        int
	ShowModal        bool
	ColorPickerState *widgets.ColorPickerState
	CursorX          int
	CursorY          int
	IsMouseDown      bool
	IsDragging       bool
	DragStartX       int
	DragStartY       int
	DragCurrentX     int
	DragCurrentY     int
	CanvasRect       cell.Rect
	PaletteRect      cell.Rect
}

func (app *PaintApp) computePreviewDots() map[int]byte {
	if !app.IsDragging {
		return nil
	}
	overlay := make(map[int]byte)

	setOverlayDot := func(vx, vy int) {
		if vx < 0 || vx >= app.Canvas.VirtWidth || vy < 0 || vy >= app.Canvas.VirtHeight {
			return
		}
		cx := vx / 2
		cy := vy / 4
		subX := vx % 2
		subY := vy % 4

		cellIdx := cy*app.Canvas.Width + cx
		if cellIdx < 0 || cellIdx >= len(app.Canvas.Grid) {
			return
		}
		overlay[cellIdx] |= brailleDotMask[subY][subX]
	}

	x0, y0 := app.DragStartX, app.DragStartY
	x1, y1 := app.DragCurrentX, app.DragCurrentY

	switch app.ActiveTool {
	case ToolLine:
		dx := abs(x1 - x0)
		dy := -abs(y1 - y0)
		sx := 1
		if x0 >= x1 {
			sx = -1
		}
		sy := 1
		if y0 >= y1 {
			sy = -1
		}
		err := dx + dy
		curX, curY := x0, y0
		for {
			setOverlayDot(curX, curY)
			if curX == x1 && curY == y1 {
				break
			}
			e2 := 2 * err
			if e2 >= dy {
				err += dy
				curX += sx
			}
			if e2 <= dx {
				err += dx
				curY += sy
			}
		}

	case ToolCircle:
		dx := x1 - x0
		dy := y1 - y0
		radius := int(math.Sqrt(float64(dx*dx + dy*dy)))
		cx, cy := x0, y0
		r := radius
		x := 0
		y := r
		d := 3 - 2*r
		drawCirclePoints := func(cx, cy, x, y int) {
			setOverlayDot(cx+x, cy+y)
			setOverlayDot(cx-x, cy+y)
			setOverlayDot(cx+x, cy-y)
			setOverlayDot(cx-x, cy-y)
			setOverlayDot(cx+y, cy+x)
			setOverlayDot(cx-y, cy+x)
			setOverlayDot(cx+y, cy-x)
			setOverlayDot(cx-y, cy-x)
		}
		drawCirclePoints(cx, cy, x, y)
		for y >= x {
			x++
			if d > 0 {
				y--
				d = d + 4*(x-y) + 10
			} else {
				d = d + 4*x + 6
			}
			drawCirclePoints(cx, cy, x, y)
		}

	case ToolRect:
		minX, maxX := min(x0, x1), max(x0, x1)
		minY, maxY := min(y0, y1), max(y0, y1)
		for x := minX; x <= maxX; x++ {
			setOverlayDot(x, minY)
			setOverlayDot(x, maxY)
		}
		for y := minY; y <= maxY; y++ {
			setOverlayDot(minX, y)
			setOverlayDot(maxX, y)
		}
	}

	return overlay
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing backend: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing terminal: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	initW, initH, _ := b.Size()
	canvasW := int(initW) - 32
	canvasH := int(initH) - 8
	if canvasW < 20 {
		canvasW = 40
	}
	if canvasH < 10 {
		canvasH = 20
	}

	app := &PaintApp{
		Canvas:           NewBraillePointCanvas(canvasW, canvasH),
		ActiveTool:       ToolBrush,
		ActiveColor:      defaultSwatches[1].Color, // Red default
		SelectedIdx:      1,
		CustomColor:      cell.NewColorRGB(0, 255, 200),
		BrushSize:        1,
		ColorPickerState: widgets.NewColorPickerState(255, 59, 48),
		CursorX:          canvasW,
		CursorY:          canvasH * 2,
	}

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			area := f.Buffer.Area
			f.SetTheme(widgets.DarkTheme())

			accentCol := cell.NewColorRGB(0, 210, 255)

			rootLay := layout.NewFlexLayout(
				layout.Vertical,
				0,
				layout.Fixed(3), // Top Header
				layout.Fixed(3), // 10 Base Swatches Toolbar
				layout.Fill(),   // Dynamic Full-Width Canvas & Right Tools Panel
				layout.Fixed(1), // Footer Shortcuts
			)
			chunks := rootLay.Split(area)

			// 1. Header Bar
			headerTitle := "  LIMONI POINT PAINT — Sub-Pixel Vector Dot Art Studio "
			f.RenderWidget(widgets.Block{
				Title:          " LIMONI POINT PAINT STUDIO ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accentCol, Modifier: cell.ModifierBold},
				Child:          text{value: headerTitle, style: cell.Style{Fg: cell.NewColorRGB(240, 245, 255), Modifier: cell.ModifierBold}},
			}, chunks[0])

			// 2. Palette Swatches Bar (10 Swatches + Custom Option)
			app.PaletteRect = chunks[1]
			f.RenderWidget(widgets.Block{
				Title:         " CORE COLORS & CUSTOM COLOR PICKER ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 130, 150)},
			}, chunks[1])

			pInnerY := chunks[1].Y + 1
			currSwX := chunks[1].X + 2

			for i, sw := range defaultSwatches {
				isSel := (app.SelectedIdx == i)
				swText := fmt.Sprintf("[%c: ● %s]", sw.Key, sw.Name)
				swStyle := cell.Style{Fg: sw.Color, Modifier: cell.ModifierBold}
				if isSel {
					swStyle = cell.Style{Fg: sw.Color, Bg: cell.NewColorRGB(35, 45, 65), Modifier: cell.ModifierBold | cell.ModifierUnderline}
				}
				f.Buffer.SetString(currSwX, pInnerY, swText, swStyle)
				currSwX += uint16(len([]rune(swText))) + 1
			}

			// Custom Color Button at right edge
			customIsSel := (app.SelectedIdx == 10)
			customText := fmt.Sprintf("[C: ● Custom %s]", colorToHex(app.CustomColor))
			customStyle := cell.Style{Fg: app.CustomColor, Modifier: cell.ModifierBold}
			if customIsSel {
				customStyle = cell.Style{Fg: app.CustomColor, Bg: cell.NewColorRGB(45, 35, 65), Modifier: cell.ModifierBold | cell.ModifierUnderline}
			}
			f.Buffer.SetString(currSwX+2, pInnerY, customText, customStyle)

			// 3. Main Workspace Split: Left (Canvas) + Right (Tools & Info Panel)
			bodyLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Fill(),
				layout.Fixed(28),
			)
			bodyChunks := bodyLay.Split(chunks[2])

			// LEFT: Dynamic Full-Area Canvas
			canvasArea := bodyChunks[0]
			app.CanvasRect = canvasArea

			availW := int(canvasArea.Width) - 2
			availH := int(canvasArea.Height) - 2
			if availW > 0 && availH > 0 && (availW != app.Canvas.Width || availH != app.Canvas.Height) {
				app.Canvas.Resize(availW, availH)
			}

			dragStatus := ""
			if app.IsDragging {
				dragStatus = " [LIVE HOLD PREVIEW]"
			}

			f.RenderWidget(widgets.Block{
				Title:         fmt.Sprintf(" POINT CANVAS (%d x %d = %d Dots)%s ", app.Canvas.VirtWidth, app.Canvas.VirtHeight, app.Canvas.VirtWidth*app.Canvas.VirtHeight, dragStatus),
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: accentCol},
			}, canvasArea)

			cInnerX := canvasArea.X + 1
			cInnerY := canvasArea.Y + 1

			overlayDots := app.computePreviewDots()

			for y := 0; y < app.Canvas.Height; y++ {
				for x := 0; x < app.Canvas.Width; x++ {
					cellIdx := y*app.Canvas.Width + x
					dotByte := app.Canvas.Grid[cellIdx]
					dotCol := app.Canvas.Colors[cellIdx]

					if overlayDots != nil {
						if ov, has := overlayDots[cellIdx]; has {
							dotByte |= ov
							dotCol = app.ActiveColor
						}
					}

					var r rune
					if dotByte == 0 {
						r = ' '
					} else {
						r = rune(0x2800 + int(dotByte))
					}

					dotStyle := cell.Style{Fg: dotCol}
					if dotCol.Type() == cell.ColorDefault {
						dotStyle = cell.Style{Fg: cell.NewColorRGB(180, 185, 200)}
					}

					isCursorCell := (x == app.CursorX/2 && y == app.CursorY/4)
					if isCursorCell {
						if dotByte == 0 {
							r = '┼'
							dotStyle = cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold}
						} else {
							dotStyle.Modifier = cell.ModifierBold | cell.ModifierReverse
						}
					}

					f.Buffer.SetCell(cInnerX+uint16(x), cInnerY+uint16(y), cell.Cell{
						Content: r,
						Style:   dotStyle,
					})
				}
			}

			// RIGHT: Tools & Status Panel
			infoArea := bodyChunks[1]
			f.RenderWidget(widgets.Block{
				Title:         " TOOLS & STATUS ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(120, 130, 150)},
			}, infoArea)

			infoInnerX := infoArea.X + 1
			infoInnerY := infoArea.Y + 1

			tools := []struct {
				name string
				key  string
				typ  ToolType
			}{
				{"Pen (Brush)", "[B]", ToolBrush},
				{"Eraser", "[E]", ToolEraser},
				{"Line", "[L]", ToolLine},
				{"Circle", "[O]", ToolCircle},
				{"Rect", "[R]", ToolRect},
			}

			for i, tInfo := range tools {
				tLine := fmt.Sprintf(" %s %-14s", tInfo.key, tInfo.name)
				tStyle := cell.Style{Fg: cell.NewColorRGB(180, 185, 200)}
				if app.ActiveTool == tInfo.typ {
					tStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Bg: cell.NewColorRGB(15, 45, 30), Modifier: cell.ModifierBold}
				}
				f.Buffer.SetString(infoInnerX, infoInnerY+uint16(i), tLine, tStyle)
			}

			// Active Color Preview Box
			prevY := infoInnerY + 6
			f.Buffer.SetString(infoInnerX, prevY, "Active Color:", cell.Style{Fg: cell.NewColorRGB(160, 165, 180)})
			activeHex := colorToHex(app.ActiveColor)
			f.Buffer.SetString(infoInnerX+14, prevY, fmt.Sprintf(" %s ", activeHex), cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: app.ActiveColor, Modifier: cell.ModifierBold})

			swatchLine := "  ██████████████████  "
			f.Buffer.SetString(infoInnerX, prevY+1, swatchLine, cell.Style{Fg: app.ActiveColor})
			f.Buffer.SetString(infoInnerX, prevY+2, swatchLine, cell.Style{Fg: app.ActiveColor})

			coordY := prevY + 4
			modeInfo := "Mode        : Drawing"
			if app.IsDragging {
				modeInfo = "Mode        : Drag & Hold Preview"
			}

			coordInfo := fmt.Sprintf(
				"Point Cursor: X:%3d Y:%3d\nBrush Size  : %d Dots ([/])\nActive Dots : %d Points\nCanvas Res  : %dx%d Dots\n%s\nUndo Buffer : %d Steps",
				app.CursorX, app.CursorY,
				app.BrushSize,
				app.Canvas.ActiveDots,
				app.Canvas.VirtWidth, app.Canvas.VirtHeight,
				modeInfo,
				len(app.Canvas.UndoGrids),
			)
			f.RenderWidget(text{value: coordInfo, style: cell.Style{Fg: cell.NewColorRGB(150, 155, 170)}}, cell.NewRect(infoInnerX, coordY, 24, 7))

			// Quick Action Buttons
			actionY := coordY + 7
			f.Buffer.SetString(infoInnerX, actionY, " [Z] Undo Action   ", cell.Style{Fg: cell.NewColorRGB(255, 200, 100), Bg: cell.NewColorRGB(35, 30, 15)})
			f.Buffer.SetString(infoInnerX, actionY+1, " [K] Clear Canvas  ", cell.Style{Fg: cell.NewColorRGB(255, 80, 80), Bg: cell.NewColorRGB(45, 15, 15)})
			f.Buffer.SetString(infoInnerX, actionY+2, " [C] Custom Color  ", cell.Style{Fg: cell.NewColorRGB(140, 200, 255), Bg: cell.NewColorRGB(15, 30, 45)})

			// 4. Footer
			footerText := "  [Drag & Drop] Draw & Preview   [Space] Paint/Anchor   [1-9,0] Colors   [B] Pen   [L] Line   [O] Circle   [R] Rect   [q] Quit"
			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Style:   cell.Style{Fg: cell.NewColorRGB(140, 145, 160), Bg: cell.NewColorRGB(22, 24, 32)},
				Child:   text{value: footerText, style: cell.Style{Fg: cell.NewColorRGB(140, 145, 160), Modifier: cell.ModifierBold}},
			}, chunks[3])

			// 5. OFFICIAL LIMONI COLOR PICKER MODAL
			if app.ShowModal {
				modalW := uint16(54)
				modalH := uint16(15)
				modalArea := terminal.CenterRect(area, modalW, modalH)

				// Draw Drop Shadow
				widgets.DrawShadow(f.Buffer, modalArea, 2, 1)

				f.RegisterLayer("color_modal", terminal.LayerModal, modalArea, 3000, func() {
					app.ShowModal = false
				})

				f.BeginLayer("color_modal")

				pickerBlock := widgets.Block{
					Title:         " 🎨 COLOR PICKER (Tab: Palette/RGB/Hex | Enter: Apply) ",
					Borders:       widgets.BorderAll,
					BorderSymbols: widgets.SymbolsRounded,
					BorderStyle:   cell.Style{Fg: accentCol, Modifier: cell.ModifierBold},
					Style:         cell.Style{Bg: cell.NewColorRGB(18, 22, 32)},
					PaddingLeft:   2,
					PaddingTop:    1,
				}
				f.RenderWidget(pickerBlock, modalArea)

				inner := pickerBlock.Inner(modalArea)
				f.RenderWidget(widgets.ColorPicker{
					ID:          "paint_color_picker",
					State:       app.ColorPickerState,
					ShowPreview: true,
				}, inner)

				f.EndLayer()
			}
		})
	}

	draw()

	for ev := range b.Events() {
		switch ev.Type {
		case backend.EventKey:
			if app.ShowModal {
				if ev.Key.Type == backend.KeyEsc {
					app.ShowModal = false
					draw()
					break
				}
				if ev.Key.Type == backend.KeyEnter {
					chosen := app.ColorPickerState.Color()
					app.CustomColor = chosen
					app.ActiveColor = chosen
					app.SelectedIdx = 10
					app.ShowModal = false
					draw()
					break
				}

				app.ColorPickerState.HandleKey(ev.Key, nil)
				draw()
				break
			}

			if ev.Key.Type == backend.KeyEsc {
				if app.IsDragging {
					app.IsDragging = false
					draw()
					break
				}
				return
			}
			if ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q' {
				return
			}

			// Number hotkeys for Swatches (1-9, 0)
			if ev.Key.Type == backend.KeyRune && ev.Key.Ch >= '1' && ev.Key.Ch <= '9' {
				idx := int(ev.Key.Ch - '1')
				app.SelectedIdx = idx
				app.ActiveColor = defaultSwatches[idx].Color
				if app.ActiveTool == ToolEraser {
					app.ActiveTool = ToolBrush
				}
				draw()
				break
			}
			if ev.Key.Type == backend.KeyRune && ev.Key.Ch == '0' {
				app.SelectedIdx = 9
				app.ActiveColor = defaultSwatches[9].Color
				if app.ActiveTool == ToolEraser {
					app.ActiveTool = ToolBrush
				}
				draw()
				break
			}

			// Brush Size keys: [ and ]
			if ev.Key.Type == backend.KeyRune && ev.Key.Ch == '[' && app.BrushSize > 1 {
				app.BrushSize--
			}
			if ev.Key.Type == backend.KeyRune && ev.Key.Ch == ']' && app.BrushSize < 8 {
				app.BrushSize++
			}

			// Tool Hotkeys
			if ev.Key.Type == backend.KeyRune {
				switch ev.Key.Ch {
				case 'b', 'B':
					app.ActiveTool = ToolBrush
					app.IsDragging = false
				case 'e', 'E':
					app.ActiveTool = ToolEraser
					app.IsDragging = false
				case 'l', 'L':
					app.ActiveTool = ToolLine
					app.IsDragging = false
				case 'o', 'O':
					app.ActiveTool = ToolCircle
					app.IsDragging = false
				case 'r', 'R':
					app.ActiveTool = ToolRect
					app.IsDragging = false
				case 'c', 'C':
					app.ShowModal = true
					r, g, b := app.ActiveColor.RGB()
					app.ColorPickerState.SetRGB(r, g, b)
				case 'z', 'Z':
					app.Canvas.Undo()
				case 'k', 'K':
					app.Canvas.Clear()
				case ' ':
					switch app.ActiveTool {
					case ToolBrush:
						app.Canvas.SaveUndo()
						if app.BrushSize <= 1 {
							app.Canvas.SetDot(app.CursorX, app.CursorY, app.ActiveColor)
						} else {
							app.Canvas.DrawDotBrush(app.CursorX, app.CursorY, app.BrushSize, app.ActiveColor)
						}
					case ToolEraser:
						app.Canvas.SaveUndo()
						if app.BrushSize <= 1 {
							app.Canvas.EraseDot(app.CursorX, app.CursorY)
						} else {
							app.Canvas.EraseDotBrush(app.CursorX, app.CursorY, app.BrushSize)
						}
					case ToolLine, ToolCircle, ToolRect:
						if !app.IsDragging {
							app.IsDragging = true
							app.DragStartX = app.CursorX
							app.DragStartY = app.CursorY
							app.DragCurrentX = app.CursorX
							app.DragCurrentY = app.CursorY
						} else {
							app.Canvas.SaveUndo()
							switch app.ActiveTool {
							case ToolLine:
								app.Canvas.DrawLine(app.DragStartX, app.DragStartY, app.CursorX, app.CursorY, app.ActiveColor)
							case ToolCircle:
								dx := app.CursorX - app.DragStartX
								dy := app.CursorY - app.DragStartY
								radius := int(math.Sqrt(float64(dx*dx + dy*dy)))
								app.Canvas.DrawCircle(app.DragStartX, app.DragStartY, radius, app.ActiveColor)
							case ToolRect:
								app.Canvas.DrawRect(app.DragStartX, app.DragStartY, app.CursorX, app.CursorY, app.ActiveColor)
							}
							app.IsDragging = false
						}
					}
				}
			}

			// Cursor Navigation
			if ev.Key.Type == backend.KeyArrowUp && app.CursorY > 0 {
				app.CursorY--
				if app.IsDragging {
					app.DragCurrentY = app.CursorY
				}
			} else if ev.Key.Type == backend.KeyArrowDown && app.CursorY < app.Canvas.VirtHeight-1 {
				app.CursorY++
				if app.IsDragging {
					app.DragCurrentY = app.CursorY
				}
			} else if ev.Key.Type == backend.KeyArrowLeft && app.CursorX > 0 {
				app.CursorX--
				if app.IsDragging {
					app.DragCurrentX = app.CursorX
				}
			} else if ev.Key.Type == backend.KeyArrowRight && app.CursorX < app.Canvas.VirtWidth-1 {
				app.CursorX++
				if app.IsDragging {
					app.DragCurrentX = app.CursorX
				}
			}

			draw()

		case backend.EventMouse:
			m := ev.Mouse
			mx, my := int(m.X), int(m.Y)

			if app.ShowModal {
				t.RouteMouseEvent(m)
				draw()
				break
			}

			// Palette Swatch Click Detection
			if my == int(app.PaletteRect.Y)+1 && mx >= int(app.PaletteRect.X)+2 {
				curX := int(app.PaletteRect.X) + 2
				for i, sw := range defaultSwatches {
					swLen := len([]rune(fmt.Sprintf("[%c: ● %s]", sw.Key, sw.Name)))
					if mx >= curX && mx < curX+swLen {
						app.SelectedIdx = i
						app.ActiveColor = sw.Color
						if app.ActiveTool == ToolEraser {
							app.ActiveTool = ToolBrush
						}
						draw()
						break
					}
					curX += swLen + 1
				}
				// Check Custom Button
				if mx >= curX+2 && mx < curX+28 {
					app.ShowModal = true
					r, g, b := app.ActiveColor.RGB()
					app.ColorPickerState.SetRGB(r, g, b)
					draw()
				}
			}

			// Sub-Pixel Canvas Mouse Drawing & Drag Handling
			cLeft := int(app.CanvasRect.X) + 1
			cTop := int(app.CanvasRect.Y) + 1
			cRight := cLeft + app.Canvas.Width
			cBottom := cTop + app.Canvas.Height

			if mx >= cLeft && mx < cRight && my >= cTop && my < cBottom {
				vx := (mx - cLeft) * 2
				vy := (my - cTop) * 4
				app.CursorX = vx
				app.CursorY = vy

				if m.Button == backend.MouseLeft {
					switch app.ActiveTool {
					case ToolBrush:
						if !app.IsMouseDown {
							app.Canvas.SaveUndo()
							app.IsMouseDown = true
						}
						if app.BrushSize <= 1 {
							app.Canvas.SetDot(vx, vy, app.ActiveColor)
						} else {
							app.Canvas.DrawDotBrush(vx, vy, app.BrushSize, app.ActiveColor)
						}

					case ToolEraser:
						if !app.IsMouseDown {
							app.Canvas.SaveUndo()
							app.IsMouseDown = true
						}
						if app.BrushSize <= 1 {
							app.Canvas.EraseDot(vx, vy)
						} else {
							app.Canvas.EraseDotBrush(vx, vy, app.BrushSize)
						}

					case ToolLine, ToolCircle, ToolRect:
						if !app.IsDragging {
							app.IsDragging = true
							app.DragStartX = vx
							app.DragStartY = vy
						}
						app.DragCurrentX = vx
						app.DragCurrentY = vy
					}
					draw()

				} else if m.Button == backend.MouseRelease {
					app.IsMouseDown = false
					if app.IsDragging {
						app.Canvas.SaveUndo()
						switch app.ActiveTool {
						case ToolLine:
							app.Canvas.DrawLine(app.DragStartX, app.DragStartY, app.DragCurrentX, app.DragCurrentY, app.ActiveColor)
						case ToolCircle:
							dx := app.DragCurrentX - app.DragStartX
							dy := app.DragCurrentY - app.DragStartY
							radius := int(math.Sqrt(float64(dx*dx + dy*dy)))
							app.Canvas.DrawCircle(app.DragStartX, app.DragStartY, radius, app.ActiveColor)
						case ToolRect:
							app.Canvas.DrawRect(app.DragStartX, app.DragStartY, app.DragCurrentX, app.DragCurrentY, app.ActiveColor)
						}
						app.IsDragging = false
						draw()
					}
				}
			}

		case backend.EventResize:
			draw()
		}
	}
}
