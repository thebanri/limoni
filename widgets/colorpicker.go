package widgets

import (
	"fmt"
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

// ColorPickerState holds the mutable color values and active modes for ColorPicker.
type ColorPickerState struct {
	Red          uint8
	Green        uint8
	Blue         uint8
	PaletteIndex int
	ActiveMode   int // 0: Palette, 1: RGB Sliders, 2: Hex
	ActiveSlider int // 0: Red, 1: Green, 2: Blue
	HexInput     string
	HexEditing   bool
}

// NewColorPickerState creates a state initialized with the given RGB color.
func NewColorPickerState(r, g, b uint8) *ColorPickerState {
	s := &ColorPickerState{
		Red:   r,
		Green: g,
		Blue:  b,
	}
	s.syncHex()
	return s
}

// Color returns the current RGB color as cell.Color.
func (s *ColorPickerState) Color() cell.Color {
	if s == nil {
		return cell.NewColorRGB(255, 255, 255)
	}
	return cell.NewColorRGB(s.Red, s.Green, s.Blue)
}

// SetRGB sets the color channels and synchronizes the hex representation.
func (s *ColorPickerState) SetRGB(r, g, b uint8) {
	if s == nil {
		return
	}
	s.Red = r
	s.Green = g
	s.Blue = b
	s.syncHex()
}

// SetHex parses a hex color string (e.g. "#FF5500" or "FF5500") and updates the RGB channels.
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
	s.Red = uint8((val >> 16) & 0xFF)
	s.Green = uint8((val >> 8) & 0xFF)
	s.Blue = uint8(val & 0xFF)
	s.HexInput = hexStr
	return true
}

func (s *ColorPickerState) syncHex() {
	s.HexInput = fmt.Sprintf("%02X%02X%02X", s.Red, s.Green, s.Blue)
}

// HandleKey handles keyboard inputs for mode switching, palette selection, and slider tuning.
func (s *ColorPickerState) HandleKey(ev backend.KeyEvent, palette []cell.Color) bool {
	if s == nil {
		return false
	}
	if len(palette) == 0 {
		palette = DefaultPalette
	}

	if ev.Type == backend.KeyTab {
		s.ActiveMode = (s.ActiveMode + 1) % 3
		s.HexEditing = false
		return true
	}

	switch s.ActiveMode {
	case 0: // Palette mode
		switch ev.Type {
		case backend.KeyArrowRight:
			s.PaletteIndex = (s.PaletteIndex + 1) % len(palette)
			s.applyPalette(palette)
			return true
		case backend.KeyArrowLeft:
			s.PaletteIndex = (s.PaletteIndex - 1 + len(palette)) % len(palette)
			s.applyPalette(palette)
			return true
		case backend.KeyArrowDown:
			if s.PaletteIndex+6 < len(palette) {
				s.PaletteIndex += 6
				s.applyPalette(palette)
				return true
			}
		case backend.KeyArrowUp:
			if s.PaletteIndex-6 >= 0 {
				s.PaletteIndex -= 6
				s.applyPalette(palette)
				return true
			}
		}

	case 1: // RGB Sliders mode
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

	case 2: // Hex Edit mode
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

func (s *ColorPickerState) applyPalette(palette []cell.Color) {
	if s.PaletteIndex >= 0 && s.PaletteIndex < len(palette) {
		r, g, b := palette[s.PaletteIndex].RGB()
		s.SetRGB(r, g, b)
	}
}

func (s *ColorPickerState) adjustSlider(delta int) {
	switch s.ActiveSlider {
	case 0:
		newVal := int(s.Red) + delta
		if newVal < 0 {
			newVal = 0
		} else if newVal > 255 {
			newVal = 255
		}
		s.Red = uint8(newVal)
	case 1:
		newVal := int(s.Green) + delta
		if newVal < 0 {
			newVal = 0
		} else if newVal > 255 {
			newVal = 255
		}
		s.Green = uint8(newVal)
	case 2:
		newVal := int(s.Blue) + delta
		if newVal < 0 {
			newVal = 0
		} else if newVal > 255 {
			newVal = 255
		}
		s.Blue = uint8(newVal)
	}
	s.syncHex()
}

// ColorPicker is an interactive color selection widget.
type ColorPicker struct {
	ID          string
	State       *ColorPickerState
	Palette     []cell.Color
	ShowPreview bool
	Style       cell.Style
}

// Draw renders the color picker widget.
func (cp ColorPicker) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 20 || area.Height < 7 {
		return
	}

	if cp.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(cp.ID)
	}

	state := cp.State
	if state == nil {
		state = NewColorPickerState(255, 255, 255)
	}
	palette := cp.Palette
	if len(palette) == 0 {
		palette = DefaultPalette
	}

	baseStyle := ctx.Style.Merge(cp.Style)
	tabStyle := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(140, 150, 165)})
	activeTabStyle := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Modifier: cell.ModifierBold})

	// Header Tabs: [1: Palette] [2: RGB] [3: Hex]
	tabs := []string{"[1: Palette]", "[2: RGB]", "[3: Hex]"}
	tabX := area.X
	for i, t := range tabs {
		st := tabStyle
		if state.ActiveMode == i {
			st = activeTabStyle
		}
		buf.SetString(tabX, area.Y, t, st)
		tabX += uint16(len(t) + 1)
	}

	currColor := state.Color()
	previewStyle := cell.Style{Bg: currColor, Fg: cell.NewColorRGB(0, 0, 0)}

	// Draw Preview swatch on right if space allows
	if cp.ShowPreview && area.Width >= 30 {
		previewX := area.X + area.Width - 10
		buf.SetString(previewX, area.Y, "Preview:", tabStyle)
		for py := uint16(1); py <= 3; py++ {
			for px := previewX; px < previewX+8; px++ {
				buf.SetCell(px, area.Y+py, cell.Cell{Content: ' ', Style: previewStyle})
			}
		}
		buf.SetString(previewX, area.Y+4, "#"+state.HexInput, activeTabStyle)
	}

	contentY := area.Y + 2

	switch state.ActiveMode {
	case 0: // Palette Grid
		cols := 6
		for i, col := range palette {
			row := i / cols
			colIdx := i % cols
			px := area.X + uint16(colIdx*4)
			py := contentY + uint16(row*2)

			swatchStyle := cell.Style{Bg: col}
			isSel := i == state.PaletteIndex

			symbol := ' '
			if isSel {
				symbol = '●'
				swatchStyle.Fg = cell.NewColorRGB(255, 255, 255)
				if r, g, b := col.RGB(); (int(r)+int(g)+int(b))/3 > 150 {
					swatchStyle.Fg = cell.NewColorRGB(0, 0, 0)
				}
			}

			buf.SetCell(px, py, cell.Cell{Content: symbol, Style: swatchStyle})
			buf.SetCell(px+1, py, cell.Cell{Content: symbol, Style: swatchStyle})
			buf.SetCell(px+2, py, cell.Cell{Content: symbol, Style: swatchStyle})

			if ctx.RegisterClick != nil && cp.State != nil {
				palIdx := i
				st := cp.State
				itemRect := cell.Rect{X: px, Y: py, Width: 3, Height: 1}
				ctx.RegisterClick(itemRect, func() {
					st.PaletteIndex = palIdx
					st.applyPalette(palette)
				})
			}
		}

	case 1: // RGB Sliders
		channels := []struct {
			name string
			val  uint8
			col  cell.Color
		}{
			{"R", state.Red, cell.NewColorRGB(255, 80, 80)},
			{"G", state.Green, cell.NewColorRGB(80, 255, 80)},
			{"B", state.Blue, cell.NewColorRGB(80, 160, 255)},
		}

		sliderWidth := int(area.Width) - 15
		if sliderWidth > 20 {
			sliderWidth = 20
		}
		if sliderWidth < 8 {
			sliderWidth = 8
		}

		for idx, ch := range channels {
			sy := contentY + uint16(idx)
			prefixStyle := cell.Style{Fg: ch.col, Modifier: cell.ModifierBold}
			if state.ActiveSlider == idx {
				prefixStyle.Modifier |= cell.ModifierUnderline
			}
			buf.SetString(area.X, sy, ch.name+": ", prefixStyle)

			pos := int(ch.val) * (sliderWidth - 1) / 255
			for sx := 0; sx < sliderWidth; sx++ {
				c := '─'
				st := baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(80, 90, 100)})
				if sx <= pos {
					st = cell.Style{Fg: ch.col}
				}
				if sx == pos {
					c = '●'
					st = cell.Style{Fg: ch.col, Modifier: cell.ModifierBold}
				}
				buf.SetCell(area.X+uint16(3+sx), sy, cell.Cell{Content: c, Style: st})
			}
			valStr := fmt.Sprintf("%3d", ch.val)
			buf.SetString(area.X+uint16(4+sliderWidth), sy, valStr, baseStyle)
		}

	case 2: // Hex Input
		buf.SetString(area.X, contentY, "Hex Color: #", baseStyle)
		inputStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 255), Modifier: cell.ModifierBold}
		buf.SetString(area.X+13, contentY, state.HexInput, inputStyle)
		cursorX := area.X + 13 + uint16(len(state.HexInput))
		buf.SetCell(cursorX, contentY, cell.Cell{Content: '▎', Style: cell.Style{Fg: cell.NewColorRGB(255, 255, 255)}})

		buf.SetString(area.X, contentY+2, "Type 6 hex digits (e.g. FF5500)", tabStyle)
	}
}

// SizeHint returns the preferred dimensions for ColorPicker.
func (cp ColorPicker) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	w := uint16(32)
	h := uint16(8)
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
		Label:  "Color Picker",
		Value:  val,
		State:  st,
		Bounds: bounds,
	}
}
