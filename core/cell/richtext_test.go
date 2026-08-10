package cell

import (
	"testing"
)

func TestParseRichTextBasic(t *testing.T) {
	base := Style{Fg: NewColorDefault(), Bg: NewColorDefault()}
	cells := ParseRichText("Hello <bold,fg=red>World</>!", base, nil)

	if len(cells) != 12 {
		t.Fatalf("Expected 12 cells, got %d", len(cells))
	}

	// Verify "Hello " has base style
	for i := 0; i < 6; i++ {
		if cells[i].Style != base {
			t.Errorf("Cell %d expected default style, got %+v", i, cells[i].Style)
		}
	}

	// Verify "World" has red foreground and bold modifier
	expectedStyle := base
	expectedStyle.Fg = NewColorANSI(1) // red
	expectedStyle.Modifier = ModifierBold

	for i := 6; i < 11; i++ {
		if cells[i].Style != expectedStyle {
			t.Errorf("Cell %d expected red bold style, got %+v", i, cells[i].Style)
		}
	}

	// Verify "!" has base style
	if cells[11].Style != base {
		t.Errorf("Cell 11 expected default style, got %+v", cells[11].Style)
	}
}

func TestParseRichTextNested(t *testing.T) {
	base := Style{Fg: NewColorDefault(), Bg: NewColorDefault()}
	cells := ParseRichText("<fg=blue>Blue <bold>BoldBlue</bold> BlueAgain</>", base, nil)

	// "Blue " (5) + "BoldBlue" (8) + " BlueAgain" (10) = 23 characters
	if len(cells) != 23 {
		t.Fatalf("Expected 23 cells, got %d", len(cells))
	}

	blueStyle := base
	blueStyle.Fg = NewColorANSI(4) // blue

	boldBlueStyle := blueStyle
	boldBlueStyle.Modifier = ModifierBold

	// Test "Blue "
	for i := 0; i < 5; i++ {
		if cells[i].Style != blueStyle {
			t.Errorf("Cell %d expected blue style, got %+v", i, cells[i].Style)
		}
	}

	// Test "BoldBlue"
	for i := 5; i < 13; i++ {
		if cells[i].Style != boldBlueStyle {
			t.Errorf("Cell %d expected bold blue style, got %+v", i, cells[i].Style)
		}
	}

	// Test " BlueAgain"
	for i := 13; i < 23; i++ {
		if cells[i].Style != blueStyle {
			t.Errorf("Cell %d expected blue style, got %+v", i, cells[i].Style)
		}
	}
}

func TestParseRichTextEscaping(t *testing.T) {
	base := Style{}
	cells := ParseRichText("Hello \\<World>", base, nil)

	// "Hello <World>" = 13 characters
	if len(cells) != 13 {
		t.Fatalf("Expected 13 cells, got %d", len(cells))
	}

	expectedStr := "Hello <World>"
	for i, c := range cells {
		if c.Content != rune(expectedStr[i]) {
			t.Errorf("Cell %d expected character %c, got %c", i, expectedStr[i], c.Content)
		}
	}
}

func TestParseRichTextResolver(t *testing.T) {
	base := Style{}
	resolver := func(tag string) Style {
		if tag == "danger" {
			return Style{Fg: NewColorRGB(255, 0, 0), Modifier: ModifierBold}
		}
		return Style{}
	}

	cells := ParseRichText("This is <danger>bad</>!", base, resolver)

	// "This is " (8) + "bad" (3) + "!" (1) = 12 characters
	if len(cells) != 12 {
		t.Fatalf("Expected 12 cells, got %d", len(cells))
	}

	dangerStyle := Style{Fg: NewColorRGB(255, 0, 0), Modifier: ModifierBold}

	// Test "bad"
	for i := 8; i < 11; i++ {
		if cells[i].Style != dangerStyle {
			t.Errorf("Cell %d expected danger style, got %+v", i, cells[i].Style)
		}
	}
}
