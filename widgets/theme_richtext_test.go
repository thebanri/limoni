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

func TestThemeContrastValidation(t *testing.T) {
	if ratio := ContrastRatio(cell.NewColorRGB(255, 255, 255), cell.NewColorRGB(0, 0, 0)); ratio < 20 {
		t.Fatalf("black/white contrast = %f; want high contrast", ratio)
	}
	if failures := HighContrastTheme().ValidateContrast(4.5); len(failures) != 0 {
		t.Fatalf("high contrast failures = %v", failures)
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

func TestRichTextWrappingAndAlignment(t *testing.T) {
	text := Text{Wrap: true, Alignment: AlignTextCenter, Lines: []Line{NewLine(Span{Text: "abcdefghij"})}}
	width, height := text.SizeHint(cell.NewRect(0, 0, 5, 4))
	if width != 5 || height != 2 {
		t.Fatalf("wrapped size = %dx%d; want 5x2", width, height)
	}
}

func TestRichTextSemanticRoleAndClick(t *testing.T) {
	clicked := false
	text := Text{Lines: []Line{NewLine(Span{Text: "Open", Role: "success", OnClick: func() { clicked = true }})}}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 10, 1))
	ctx := cell.NewContext(buf.Area, cell.Style{})
	ctx.ThemeStyle = DarkTheme().RoleStyle
	var handler func()
	ctx.RegisterClick = func(_ cell.Rect, callback func()) { handler = callback }
	text.Draw(ctx, buf)
	if handler == nil {
		t.Fatal("clickable span did not register a handler")
	}
	handler()
	if !clicked {
		t.Fatal("clickable span handler did not run")
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

func TestThemeCustomClasses(t *testing.T) {
	theme := DarkTheme()
	customStyle := cell.Style{Fg: cell.NewColorRGB(123, 45, 67), Modifier: cell.ModifierUnderline}
	theme = theme.AddClass("myclass", customStyle)

	resolved := theme.RoleStyle("myclass")
	if resolved != customStyle {
		t.Fatalf("Expected custom class style %+v, got %+v", customStyle, resolved)
	}
}

func TestTextFromRichText(t *testing.T) {
	theme := DarkTheme()
	customStyle := cell.Style{Fg: cell.NewColorRGB(200, 100, 50)}
	theme = theme.AddClass("highlight", customStyle)

	baseStyle := cell.Style{Fg: cell.NewColorRGB(255, 255, 255)}
	textWidget := TextFromRichText("Hello <highlight>World</>\nNew <bold>Line</>", baseStyle, theme)

	if len(textWidget.Lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(textWidget.Lines))
	}

	// Line 0: "Hello " (base style), "World" (highlight style)
	line0 := textWidget.Lines[0]
	if len(line0.Spans) != 2 {
		t.Fatalf("Expected 2 spans in line 0, got %d", len(line0.Spans))
	}

	if line0.Spans[0].Text != "Hello " || line0.Spans[0].Style != baseStyle {
		t.Errorf("Span 0 style mismatch: got %q / %+v", line0.Spans[0].Text, line0.Spans[0].Style)
	}

	if line0.Spans[1].Text != "World" || line0.Spans[1].Style != customStyle {
		t.Errorf("Span 1 style mismatch: got %q / %+v", line0.Spans[1].Text, line0.Spans[1].Style)
	}

	// Line 1: "New " (base style), "Line" (base + bold style)
	line1 := textWidget.Lines[1]
	if len(line1.Spans) != 2 {
		t.Fatalf("Expected 2 spans in line 1, got %d", len(line1.Spans))
	}

	if line1.Spans[0].Text != "New " || line1.Spans[0].Style != baseStyle {
		t.Errorf("Span 0 line 1 style mismatch: got %q / %+v", line1.Spans[0].Text, line1.Spans[0].Style)
	}

	expectedBoldStyle := baseStyle
	expectedBoldStyle.Modifier = cell.ModifierBold
	if line1.Spans[1].Text != "Line" || line1.Spans[1].Style != expectedBoldStyle {
		t.Errorf("Span 1 line 1 style mismatch: got %q / %+v", line1.Spans[1].Text, line1.Spans[1].Style)
	}
}

