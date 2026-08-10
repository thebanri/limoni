package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func loadProfileImage() image.Image {
	for _, path := range []string{"examples/demo/profile.png", "examples/demo/profile.jpg", "profile.png", "profile.jpg"} {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		img, _, decodeErr := image.Decode(file)
		file.Close()
		if decodeErr == nil {
			return img
		}
	}
	return nil
}

func navigateDemoTab(state *AppState, focus *terminal.FocusManager, delta int) {
	tabs := []string{"Giriş", "Ayarlar", "Grafik", "Playground", "Referans"}
	current := 0
	for i, tab := range tabs {
		if tab == state.ActiveTab {
			current = i
			break
		}
	}
	current = (current + delta + len(tabs)) % len(tabs)
	state.ActiveTab = tabs[current]
	state.IsTransitioning = false
	focus.SetFocused("tab_" + state.ActiveTab)
}

func themeForSelection(selection string) widgets.Theme {
	theme := widgets.DarkTheme()
	switch selection {
	case "Açık":
		theme = widgets.LightTheme()
	case "Renkli":
		theme.Colors.Primary = cell.NewColorRGB(255, 165, 0)
		theme.Colors.Secondary = cell.NewColorRGB(255, 0, 255)
		theme.Colors.Success = cell.NewColorRGB(255, 0, 255)
		theme.Colors.Surface = cell.NewColorRGB(35, 25, 45)
		theme.Base = cell.Style{Fg: theme.Colors.Text, Bg: theme.Colors.Background}
		theme.Focus = cell.Style{Fg: theme.Colors.Secondary, Modifier: cell.ModifierBold}
	case "Yüksek Kontrast":
		theme = widgets.HighContrastTheme()
	}
	return theme
}

func registerTargetClick(f *terminal.Frame, area cell.Rect, handler func(backend.MouseEvent)) {
	f.RegisterEventHandler(area, terminal.TargetPhase, func(event *backend.EventContext) {
		if event.Mouse.Button != backend.MouseLeft || event.Mouse.Drag {
			return
		}
		event.PreventDefault()
		handler(event.Mouse)
	})
}
