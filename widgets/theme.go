package widgets

import (
	"math"

	"github.com/thebanri/limoni/core/cell"
)

// ThemeColors contains semantic colors shared by widgets.
type ThemeColors struct {
	Primary    cell.Color
	Secondary  cell.Color
	Background cell.Color
	Surface    cell.Color
	Border     cell.Color
	Text       cell.Color
	Muted      cell.Color
	Success    cell.Color
	Warning    cell.Color
	Error      cell.Color
}

// Theme is the semantic style palette for an application.
type Theme struct {
	Colors  ThemeColors
	Base    cell.Style
	Border  cell.Style
	Focus   cell.Style
	Classes map[string]cell.Style
}

// DarkTheme returns a high-contrast dark terminal theme.
func DarkTheme() Theme {
	return Theme{
		Colors: ThemeColors{
			Primary: cell.NewColorRGB(100, 200, 255), Secondary: cell.NewColorRGB(180, 120, 255),
			Background: cell.NewColorRGB(12, 14, 18), Surface: cell.NewColorRGB(25, 28, 36),
			Border: cell.NewColorRGB(70, 75, 90), Text: cell.NewColorRGB(220, 225, 235),
			Muted: cell.NewColorRGB(125, 130, 145), Success: cell.NewColorRGB(80, 220, 140),
			Warning: cell.NewColorRGB(255, 210, 80), Error: cell.NewColorRGB(255, 90, 90),
		},
		Base:   cell.Style{Fg: cell.NewColorRGB(220, 225, 235), Bg: cell.NewColorRGB(12, 14, 18)},
		Border: cell.Style{Fg: cell.NewColorRGB(70, 75, 90)},
		Focus:  cell.Style{Fg: cell.NewColorRGB(100, 200, 255), Modifier: cell.ModifierBold},
	}
}

// RoleStyle resolves a semantic role to a foreground style.
func (t Theme) RoleStyle(role string) cell.Style {
	if t.Classes != nil {
		if style, ok := t.Classes[role]; ok {
			return style
		}
	}
	switch role {
	case "primary":
		return cell.Style{Fg: t.Colors.Primary}
	case "secondary":
		return cell.Style{Fg: t.Colors.Secondary}
	case "background":
		return cell.Style{Fg: t.Colors.Text, Bg: t.Colors.Background}
	case "surface":
		return cell.Style{Fg: t.Colors.Text, Bg: t.Colors.Surface}
	case "border":
		return cell.Style{Fg: t.Colors.Border}
	case "text":
		return cell.Style{Fg: t.Colors.Text}
	case "muted":
		return cell.Style{Fg: t.Colors.Muted}
	case "success":
		return cell.Style{Fg: t.Colors.Success}
	case "warning":
		return cell.Style{Fg: t.Colors.Warning}
	case "error":
		return cell.Style{Fg: t.Colors.Error}
	case "focus":
		return t.Focus
	default:
		return t.Base
	}
}

// AddClass adds a custom class-based style mapping to the theme.
func (t Theme) AddClass(name string, style cell.Style) Theme {
	if t.Classes == nil {
		t.Classes = make(map[string]cell.Style)
	}
	t.Classes[name] = style
	return t
}

// HighContrastTheme returns a black/white accessibility-focused theme.
func HighContrastTheme() Theme {
	theme := DarkTheme()
	theme.Colors.Background = cell.NewColorRGB(0, 0, 0)
	theme.Colors.Surface = cell.NewColorRGB(0, 0, 0)
	theme.Colors.Text = cell.NewColorRGB(255, 255, 255)
	theme.Colors.Muted = cell.NewColorRGB(220, 220, 220)
	theme.Colors.Border = cell.NewColorRGB(255, 255, 255)
	theme.Colors.Primary = cell.NewColorRGB(255, 255, 0)
	theme.Colors.Success = cell.NewColorRGB(0, 255, 0)
	theme.Colors.Warning = cell.NewColorRGB(255, 255, 0)
	theme.Colors.Error = cell.NewColorRGB(255, 80, 80)
	theme.Base = cell.Style{Fg: theme.Colors.Text, Bg: theme.Colors.Background}
	theme.Border = cell.Style{Fg: theme.Colors.Border}
	theme.Focus = cell.Style{Fg: theme.Colors.Primary, Modifier: cell.ModifierBold}
	return theme
}

// ContrastRatio returns the WCAG-style contrast ratio of two terminal colors.
func ContrastRatio(foreground, background cell.Color) float64 {
	fr, fg, fb := foreground.RGB()
	br, bg, bb := background.RGB()
	luminance := func(r, g, b uint8) float64 {
		linear := func(value uint8) float64 {
			v := float64(value) / 255
			if v <= 0.03928 {
				return v / 12.92
			}
			return math.Pow((v+0.055)/1.055, 2.4)
		}
		return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
	}
	l1, l2 := luminance(fr, fg, fb), luminance(br, bg, bb)
	if l2 > l1 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// ValidateContrast returns semantic color pairs below the requested ratio.
func (t Theme) ValidateContrast(minimum float64) []string {
	pairs := []struct {
		name   string
		fg, bg cell.Color
	}{
		{"text/background", t.Colors.Text, t.Colors.Background},
		{"muted/background", t.Colors.Muted, t.Colors.Background},
		{"primary/surface", t.Colors.Primary, t.Colors.Surface},
		{"success/surface", t.Colors.Success, t.Colors.Surface},
		{"warning/surface", t.Colors.Warning, t.Colors.Surface},
		{"error/surface", t.Colors.Error, t.Colors.Surface},
	}
	var failures []string
	for _, pair := range pairs {
		if ContrastRatio(pair.fg, pair.bg) < minimum {
			failures = append(failures, pair.name)
		}
	}
	return failures
}

// LightTheme returns a readable light terminal theme.
func LightTheme() Theme {
	return Theme{
		Colors: ThemeColors{
			Primary: cell.NewColorRGB(30, 90, 180), Secondary: cell.NewColorRGB(100, 50, 160),
			Background: cell.NewColorRGB(245, 245, 245), Surface: cell.NewColorRGB(230, 232, 238),
			Border: cell.NewColorRGB(100, 105, 115), Text: cell.NewColorRGB(25, 28, 35),
			Muted: cell.NewColorRGB(95, 100, 110), Success: cell.NewColorRGB(20, 140, 75),
			Warning: cell.NewColorRGB(180, 120, 0), Error: cell.NewColorRGB(190, 30, 35),
		},
		Base:   cell.Style{Fg: cell.NewColorRGB(25, 28, 35), Bg: cell.NewColorRGB(245, 245, 245)},
		Border: cell.Style{Fg: cell.NewColorRGB(100, 105, 115)},
		Focus:  cell.Style{Fg: cell.NewColorRGB(30, 90, 180), Modifier: cell.ModifierBold},
	}
}
