package widgets

import "github.com/thebanri/limoni/core/cell"

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
	Colors ThemeColors
	Base   cell.Style
	Border cell.Style
	Focus  cell.Style
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
