package cell

import "testing"

// What a 256-colour terminal is sent instead of 24-bit colour. The cube and
// the grey ramp are the two halves of xterm's 256-colour palette, and mixing
// them up is the difference between a readable dim grey and a brown.
func TestRGBToANSI256(t *testing.T) {
	cases := []struct {
		name    string
		r, g, b uint8
		want    uint8
	}{
		{"pure red is the cube's red corner", 255, 0, 0, 196},
		{"pure green", 0, 255, 0, 46},
		{"pure blue", 0, 0, 255, 21},
		{"white", 255, 255, 255, 231},
		{"black", 0, 0, 0, 16},
		{"mid grey lands on the grey ramp", 128, 128, 128, 244},
		{"near-black grey is the cube's black", 3, 3, 3, 16},
		{"near-white grey is the cube's white", 250, 250, 250, 231},
		{"a tinted grey is not grey enough for the ramp", 128, 128, 160, 103},
	}
	for _, c := range cases {
		if got := RGBToANSI256(c.r, c.g, c.b); got != c.want {
			t.Errorf("%s: RGBToANSI256(%d,%d,%d) = %d, want %d", c.name, c.r, c.g, c.b, got, c.want)
		}
	}
}

// Every grey on the ramp must come back as a grey, never as a cube colour:
// this is the conversion a 256-colour terminal applies to every dim border.
func TestGreysStayOnTheGreyRamp(t *testing.T) {
	for v := 8; v <= 238; v += 10 {
		got := RGBToANSI256(uint8(v), uint8(v), uint8(v))
		if got < 232 || got > 255 {
			t.Errorf("grey %d mapped to %d, outside the 232-255 ramp", v, got)
		}
	}
}

func TestRGBToANSI16PicksTheNearestOfTheSixteen(t *testing.T) {
	cases := []struct {
		r, g, b uint8
		want    uint8
	}{
		{0, 0, 0, 0},
		{255, 0, 0, 9}, // bright red, not dark red
		{130, 0, 0, 1}, // dark red
		{255, 255, 255, 15},
		{200, 200, 200, 7}, // white, which is 192,192,192
		{10, 10, 200, 12},  // bright blue
	}
	for _, c := range cases {
		if got := RGBToANSI16(c.r, c.g, c.b); got != c.want {
			t.Errorf("RGBToANSI16(%d,%d,%d) = %d, want %d", c.r, c.g, c.b, got, c.want)
		}
	}
}

func TestANSI256To16(t *testing.T) {
	for i := 0; i < 16; i++ {
		if got := ANSI256To16(uint8(i)); got != uint8(i) {
			t.Errorf("the first sixteen must pass through: %d became %d", i, got)
		}
	}
	if got := ANSI256To16(196); got != 9 { // cube red
		t.Errorf("196 (cube red) became %d, want 9", got)
	}
	if got := ANSI256To16(255); got != 15 { // brightest grey
		t.Errorf("255 (brightest grey) became %d, want 15", got)
	}
	if got := ANSI256To16(232); got != 0 { // darkest grey
		t.Errorf("232 (darkest grey) became %d, want 0", got)
	}
	for i := 16; i <= 255; i++ {
		if got := ANSI256To16(uint8(i)); got > 15 {
			t.Fatalf("ANSI256To16(%d) = %d, which is not a 16-colour index", i, got)
		}
	}
}

// Downsample is what the diff calls for every style it emits, so what it does
// to each colour type matters more than any single conversion.
func TestDownsampleByCapability(t *testing.T) {
	red := NewColorRGB(255, 0, 0)
	idx := NewColorANSI(196)
	def := NewColorDefault()

	if got := red.Downsample(true, true); got != red {
		t.Error("a truecolor terminal must be sent the colour unchanged")
	}
	if got := def.Downsample(false, false); got != def {
		t.Error("the default colour has nothing to downsample to")
	}
	if got := red.Downsample(false, true); got != NewColorANSI(196) {
		t.Errorf("256-colour: got %v, want index 196", got)
	}
	if got := idx.Downsample(false, true); got != idx {
		t.Error("an index is already 256-colour safe")
	}
	if got := red.Downsample(false, false); got != NewColorANSI(9) {
		t.Errorf("16-colour: got %v, want index 9", got)
	}
	if got := idx.Downsample(false, false); got != NewColorANSI(9) {
		t.Errorf("16-colour from an index: got %v, want index 9", got)
	}
}

func TestStyleDownsampleTouchesBothColoursAndNothingElse(t *testing.T) {
	s := NewStyle().
		WithFg(NewColorRGB(255, 0, 0)).
		WithBg(NewColorRGB(0, 0, 255)).
		Bold().
		WithLink("https://example.com")
	got := s.Downsample(false, false)

	if got.Fg != NewColorANSI(9) || got.Bg != NewColorANSI(12) {
		t.Errorf("colours = %v/%v, want 9/12", got.Fg, got.Bg)
	}
	if !got.HasModifier(ModifierBold) {
		t.Error("downsampling dropped the modifiers")
	}
	if got.Link != s.Link {
		t.Error("downsampling dropped the hyperlink, which has nothing to do with colour depth")
	}
}
