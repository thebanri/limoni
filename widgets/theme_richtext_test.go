package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestThemeRoleStyle(t *testing.T) {
	theme := DarkTheme()
	if theme.RoleStyle("success").Fg != theme.Colors.Success {
		t.Fatal("success role should resolve to success color")
	}
	if theme.RoleStyle("unknown").Fg != theme.Base.Fg {
		t.Fatal("unknown role should fall back to base style")
	}
}

func TestThemePresets(t *testing.T) {
	dark := DarkTheme()
	light := LightTheme()
	if dark.Colors.Background == light.Colors.Background {
		t.Fatal("dark and light backgrounds should differ")
	}
	if dark.Colors.Primary == 0 || light.Colors.Primary == 0 {
		t.Fatal("theme primary colors should be set")
	}
}

func TestRichTextDrawAndSizeHint(t *testing.T) {
	text := Text{Lines: []Line{
		NewLine(Span{Text: "Status: "}, Span{Text: "OK", Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}}),
	}}
	area := cell.NewRect(0, 0, 20, 2)
	buf := buffer.NewBuffer(area)
	text.Draw(cell.NewContext(area, cell.Style{}), buf)
	if buf.Get(0, 0).Content != 'S' || buf.Get(8, 0).Content != 'O' {
		t.Fatal("rich text spans were not drawn")
	}
	width, height := text.SizeHint(area)
	if width != 10 || height != 1 {
		t.Fatalf("size hint = %dx%d; want 10x1", width, height)
	}
}
