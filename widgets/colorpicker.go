package widgets

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// DefaultPalette contains curated modern RGB colors.
var DefaultPalette = []cell.Color{
	cell.NewColorRGB(255, 75, 75),   // Coral Red
	cell.NewColorRGB(255, 140, 0),   // Orange
	cell.NewColorRGB(255, 215, 0),   // Gold Yellow
	cell.NewColorRGB(46, 204, 113),  // Emerald Green
	cell.NewColorRGB(26, 188, 156),  // Turquoise
	cell.NewColorRGB(52, 152, 219),  // Sky Blue
	cell.NewColorRGB(155, 89, 182),  // Amethyst Purple
	cell.NewColorRGB(233, 30, 99),   // Magenta Pink
	cell.NewColorRGB(240, 240, 245), // Pure White
	cell.NewColorRGB(127, 140, 141), // Silver Gray
	cell.NewColorRGB(44, 62, 80),    // Midnight Blue
	cell.NewColorRGB(20, 20, 25),    // Deep Obsidian
}

// ColorPickerState holds the mutable color values, HSV coordinates, and modes.
type ColorPickerState struct {
	Red          uint8
	Green        uint8
	Blue         uint8
	Hue          float64 // 0.0 to 360.0 degrees
	Sat          float64 // 0.0 to 1.0
	Val          float64 // 0.0 to 1.0
	PaletteIndex int
	ActiveMode   int // 0: 2D Sat/Val, 1: Hue Bar, 2: RGB Sliders, 3: Hex Input
	ActiveSlider int // 0: Red, 1: Green, 2: Blue
	HexInput     string
	HexEditing   bool
}

// NewColorPickerState creates a state initialized with the given RGB color.
func NewColorPickerState(r, g, b uint8) *ColorPickerState {
	s := &ColorPickerState{}
	s.SetRGB(r, g, b)
	return s
}

// Color returns the current RGB color as cell.Color.
func (s *ColorPickerState) Color() cell.Color {
	if s == nil {
		return cell.NewColorRGB(255, 255, 255)
	}
	return cell.NewColorRGB(s.Red, s.Green, s.Blue)
}

// SetRGB sets the color channels, recalculates HSV, and updates the hex string.
func (s *ColorPickerState) SetRGB(r, g, b uint8) {
	if s == nil {
		return
	}
	s.Red = r
	s.Green = g
	s.Blue = b
	s.Hue, s.Sat, s.Val = rgbToHSV(r, g, b)
	s.syncHex()
}

// SetHSV updates the HSV coordinates, recalculates RGB, and updates the hex string.
func (s *ColorPickerState) SetHSV(h, sat, v float64) {
	if s == nil {
		return
	}
	if h < 0 {
		h = 0
	} else if h > 360 {
		h = 360
	}
	if sat < 0 {
		sat = 0
	} else if sat > 1 {
		sat = 1
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	s.Hue = h
	s.Sat = sat
	s.Val = v
	s.Red, s.Green, s.Blue = hsvToRGB(h, sat, v)
	s.syncHex()
}

// SetHex parses a hex color string and updates RGB + HSV.
func (s *ColorPickerState) SetHex(hexStr string) bool {
	if s == nil {
		return false
	}
	hexStr = strings.TrimPrefix(strings.TrimSpace(hexStr), "#")
	if len(hexStr) != 6 {
		return false
	}
	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return false
	}
	r := uint8((val >> 16) & 0xFF)
	g := uint8((val >> 8) & 0xFF)
	b := uint8(val & 0xFF)
	s.SetRGB(r, g, b)
	return true
}

func (s *ColorPickerState) syncHex() {
	s.HexInput = fmt.Sprintf("%02X%02X%02X", s.Red, s.Green, s.Blue)
}

// HandleKey handles keyboard navigation (Arrow keys, Tab, Enter, Backspace).
func (s *ColorPickerState) HandleKey(ev backend.KeyEvent, palette []cell.Color) bool {
	if s == nil {
		return false
	}
	if len(palette) == 0 {
		palette = DefaultPalette
	}

	if ev.Type == backend.KeyTab {
		s.ActiveMode = (s.ActiveMode + 1) % 4
		return true
	}

	switch s.ActiveMode {
	case 0: // 2D Sat/Val Plane
		switch ev.Type {
		case backend.KeyArrowRight:
			s.SetHSV(s.Hue, s.Sat+0.05, s.Val)
			return true
		case backend.KeyArrowLeft:
			s.SetHSV(s.Hue, s.Sat-0.05, s.Val)
			return true
		case backend.KeyArrowUp:
			s.SetHSV(s.Hue, s.Sat, s.Val+0.05)
			return true
		case backend.KeyArrowDown:
			s.SetHSV(s.Hue, s.Sat, s.Val-0.05)
			return true
		}

	case 1: // Hue Bar
		switch ev.Type {
		case backend.KeyArrowDown, backend.KeyArrowRight:
			s.SetHSV(s.Hue+10.0, s.Sat, s.Val)
			return true
		case backend.KeyArrowUp, backend.KeyArrowLeft:
			s.SetHSV(s.Hue-10.0, s.Sat, s.Val)
			return true
		}

	case 2: // RGB Sliders
		switch ev.Type {
		case backend.KeyArrowDown:
			s.ActiveSlider = (s.ActiveSlider + 1) % 3
			return true
		case backend.KeyArrowUp:
			s.ActiveSlider = (s.ActiveSlider - 1 + 3) % 3
			return true
		case backend.KeyArrowRight:
			s.adjustSlider(5)
			return true
		case backend.KeyArrowLeft:
			s.adjustSlider(-5)
			return true
		}

	case 3: // Hex Input
		if ev.Type == backend.KeyBackspace {
			if len(s.HexInput) > 0 {
				s.HexInput = s.HexInput[:len(s.HexInput)-1]
				return true
			}
		} else if ev.Type == backend.KeyEnter {
			s.SetHex(s.HexInput)
			return true
		} else if ev.Type == backend.KeyRune {
			r := ev.Ch
			if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
				if len(s.HexInput) < 6 {
					s.HexInput += string(r)
					if len(s.HexInput) == 6 {
						s.SetHex(s.HexInput)
					}
					return true
				}
			}
		}
	}
	return false
}

func (s *ColorPickerState) adjustSlider(delta int) {
	r, g, b := int(s.Red), int(s.Green), int(s.Blue)
	switch s.ActiveSlider {
	case 0:
		r = clampInt(r+delta, 0, 255)
	case 1:
		g = clampInt(g+delta, 0, 255)
	case 2:
		b = clampInt(b+delta, 0, 255)
	}
	s.SetRGB(uint8(r), uint8(g), uint8(b))
}

func clampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// ColorPicker is a rich, KDE / desktop-style 2D HSV graphical color picker.
type ColorPicker struct {
	ID          string
	State       *ColorPickerState
	Palette     []cell.Color
	ShowPreview bool
	Style       cell.Style
}

// Draw renders the graphical 2D HSV gradient matrix, vertical Hue bar, preview, and RGB/Hex controls.
func (cp ColorPicker) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 28 || area.Height < 7 {
		return
	}

	if cp.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(cp.ID)
	}

	state := cp.State
	if state == nil {
		state = NewColorPickerState(255, 59, 48)
	}
	palette := cp.Palette
	if len(palette) == 0 {
		palette = DefaultPalette
	}

	baseStyle := ctx.Style.Merge(cp.Style)
	tabStyle := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(140, 150, 165)})
	activeTabStyle := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Modifier: cell.ModifierBold})

	// Header Mode Tabs: [1: 2D Field] [2: Hue] [3: RGB] [4: Hex]
	modes := []string{"[1: 2D Field]", "[2: Hue]", "[3: RGB]", "[4: Hex]"}
	tabX := area.X
	for i, m := range modes {
		st := tabStyle
		if state.ActiveMode == i {
			st = activeTabStyle
		}
		buf.SetString(tabX, area.Y, m, st)
		tabX += uint16(len(m) + 1)
	}

	contentY := area.Y + 2
	availableH := int(area.Height) - 4
	if availableH < 5 {
		availableH = 5
	}
	if availableH > 8 {
		availableH = 8
	}

	// 1. 2D SATURATION / VALUE GRADIENT FIELD (Left)
	fieldW := 22
	if int(area.Width) < 48 {
		fieldW = int(area.Width) - 26
		if fieldW < 12 {
			fieldW = 12
		}
	}
	fieldH := availableH

	crosshairX := int(math.Round(state.Sat * float64(fieldW-1)))
	crosshairY := int(math.Round((1.0 - state.Val) * float64(fieldH-1)))

	for fy := 0; fy < fieldH; fy++ {
		val := 1.0 - (float64(fy) / float64(fieldH-1))
		py := contentY + uint16(fy)

		for fx := 0; fx < fieldW; fx++ {
			sat := float64(fx) / float64(fieldW-1)
			px := area.X + uint16(fx)

			r, g, b := hsvToRGB(state.Hue, sat, val)
			cellCol := cell.NewColorRGB(r, g, b)

			cellStyle := cell.Style{Bg: cellCol}
			symbol := ' '

			if fx == crosshairX && fy == crosshairY {
				symbol = '◎'
				if val < 0.5 || (sat > 0.6 && state.Hue > 200 && state.Hue < 280) {
					cellStyle.Fg = cell.NewColorRGB(255, 255, 255)
				} else {
					cellStyle.Fg = cell.NewColorRGB(0, 0, 0)
				}
				cellStyle.Modifier = cell.ModifierBold
			}

			buf.SetCell(px, py, cell.Cell{Content: symbol, Style: cellStyle})
		}
	}

	// Register 2D Field Mouse Drag & Click
	if ctx.RegisterMouse != nil && cp.State != nil {
		st := cp.State
		fieldArea := cell.Rect{X: area.X, Y: contentY, Width: uint16(fieldW), Height: uint16(fieldH)}
		ctx.RegisterMouse(fieldArea, func(ev backend.MouseEvent) {
			if ev.Button == backend.MouseLeft || ev.Drag {
				relX := int(ev.X) - int(fieldArea.X)
				relY := int(ev.Y) - int(fieldArea.Y)
				if relX < 0 {
					relX = 0
				} else if relX >= fieldW {
					relX = fieldW - 1
				}
				if relY < 0 {
					relY = 0
				} else if relY >= fieldH {
					relY = fieldH - 1
				}
				newSat := float64(relX) / float64(fieldW-1)
				newVal := 1.0 - (float64(relY) / float64(fieldH-1))
				st.SetHSV(st.Hue, newSat, newVal)
				st.ActiveMode = 0
			}
		})
	}

	// 2. VERTICAL HUE RAINBOW BAR (Middle)
	hueBarX := area.X + uint16(fieldW) + 1
	hueRow := int(math.Round((state.Hue / 360.0) * float64(fieldH-1)))

	for hy := 0; hy < fieldH; hy++ {
		hDeg := (float64(hy) / float64(fieldH-1)) * 360.0
		hr, hg, hb := hsvToRGB(hDeg, 1.0, 1.0)
		hueCol := cell.NewColorRGB(hr, hg, hb)
		py := contentY + uint16(hy)

		hueStyle := cell.Style{Bg: hueCol}
		buf.SetCell(hueBarX, py, cell.Cell{Content: ' ', Style: hueStyle})
		buf.SetCell(hueBarX+1, py, cell.Cell{Content: ' ', Style: hueStyle})

		// Hue Pointer arrow on side
		if hy == hueRow {
			pointerStyle := cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold}
			buf.SetCell(hueBarX+2, py, cell.Cell{Content: '◄', Style: pointerStyle})
		}
	}

	// Register Hue Bar Mouse Drag & Click
	if ctx.RegisterMouse != nil && cp.State != nil {
		st := cp.State
		hueArea := cell.Rect{X: hueBarX, Y: contentY, Width: 3, Height: uint16(fieldH)}
		ctx.RegisterMouse(hueArea, func(ev backend.MouseEvent) {
			if ev.Button == backend.MouseLeft || ev.Drag {
				relY := int(ev.Y) - int(hueArea.Y)
				if relY < 0 {
					relY = 0
				} else if relY >= fieldH {
					relY = fieldH - 1
				}
				newHue := (float64(relY) / float64(fieldH-1)) * 360.0
				st.SetHSV(newHue, st.Sat, st.Val)
				st.ActiveMode = 1
			}
		})
	}

	// 3. RIGHT DETAILS & PREVIEW PANEL
	infoX := hueBarX + 4
	currCol := state.Color()

	// Live Color Preview Box
	previewStyle := cell.Style{Bg: currCol}
	buf.SetString(infoX, contentY, "Preview:", tabStyle)
	for py := uint16(1); py <= 2; py++ {
		for px := uint16(0); px < 8; px++ {
			buf.SetCell(infoX+px, contentY+py, cell.Cell{Content: ' ', Style: previewStyle})
		}
	}

	// Hex Code
	hexY := contentY + 3
	hexPrefixStyle := cell.Style{Fg: cell.NewColorRGB(180, 190, 205), Modifier: cell.ModifierBold}
	buf.SetString(infoX, hexY, "HEX: #", hexPrefixStyle)
	hexValStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Modifier: cell.ModifierBold}
	if state.ActiveMode == 3 {
		hexValStyle.Modifier |= cell.ModifierUnderline
	}
	buf.SetString(infoX+6, hexY, state.HexInput, hexValStyle)

	// RGB Values
	rgbY := contentY + 4
	buf.SetString(infoX, rgbY, fmt.Sprintf("RGB: %3d %3d %3d", state.Red, state.Green, state.Blue), baseStyle)

	// HSV Values
	hsvY := contentY + 5
	buf.SetString(infoX, hsvY, fmt.Sprintf("HSV: %3.0f° %2.0f%% %2.0f%%", state.Hue, state.Sat*100, state.Val*100), tabStyle)

	// 4. BOTTOM PRESET PALETTE SWATCHES
	swatchY := contentY + uint16(fieldH) + 1
	if swatchY < area.Y+area.Height {
		buf.SetString(area.X, swatchY, "Presets: ", tabStyle)
		swX := area.X + 9

		for i, pCol := range palette {
			if swX+2 > area.X+area.Width {
				break
			}
			swStyle := cell.Style{Fg: pCol}
			symbol := '●'
			if i == state.PaletteIndex {
				symbol = '◉'
			}
			buf.SetCell(swX, swatchY, cell.Cell{Content: symbol, Style: swStyle})

			if ctx.RegisterClick != nil && cp.State != nil {
				palIdx := i
				st := cp.State
				targetCol := pCol
				itemRect := cell.Rect{X: swX, Y: swatchY, Width: 1, Height: 1}
				ctx.RegisterClick(itemRect, func() {
					st.PaletteIndex = palIdx
					r, g, b := targetCol.RGB()
					st.SetRGB(r, g, b)
				})
			}
			swX += 2
		}
	}
}

// SizeHint returns the preferred dimensions for ColorPicker.
func (cp ColorPicker) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	w := uint16(48)
	h := uint16(12)
	if w > maxArea.Width {
		w = maxArea.Width
	}
	if h > maxArea.Height {
		h = maxArea.Height
	}
	return w, h
}

// AccessibilityNode returns the semantic accessibility node for ColorPicker.
func (cp ColorPicker) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	st := accessibility.NodeState(0)
	if focused {
		st |= accessibility.StateFocused
	}
	val := ""
	if cp.State != nil {
		val = "#" + cp.State.HexInput
	}
	return accessibility.AccessibilityNode{
		ID:     cp.ID,
		Role:   accessibility.RoleInput,
		Label:  "KDE-style Graphical Color Picker",
		Value:  val,
		State:  st,
		Bounds: bounds,
	}
}

// hsvToRGB converts Hue (0-360), Saturation (0-1), and Value (0-1) to RGB (0-255).
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	if s <= 0 {
		val := uint8(math.Round(v * 255))
		return val, val, val
	}
	h = math.Mod(h, 360.0)
	if h < 0 {
		h += 360.0
	}
	h /= 60.0
	i := int(math.Floor(h))
	f := h - float64(i)
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))

	var r, g, b float64
	switch i {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return uint8(math.Round(r * 255)), uint8(math.Round(g * 255)), uint8(math.Round(b * 255))
}

// rgbToHSV converts RGB (0-255) to Hue (0-360), Saturation (0-1), and Value (0-1).
func rgbToHSV(r, g, b uint8) (float64, float64, float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	maxVal := math.Max(rf, math.Max(gf, bf))
	minVal := math.Min(rf, math.Min(gf, bf))
	delta := maxVal - minVal

	var h, s, v float64
	v = maxVal

	if maxVal > 0 {
		s = delta / maxVal
	} else {
		s = 0
	}

	if delta <= 0 {
		h = 0
	} else if maxVal == rf {
		h = 60.0 * math.Mod((gf-bf)/delta, 6.0)
	} else if maxVal == gf {
		h = 60.0 * (((bf-rf)/delta) + 2.0)
	} else {
		h = 60.0 * (((rf-gf)/delta) + 4.0)
	}
	if h < 0 {
		h += 360.0
	}
	return h, s, v
}
