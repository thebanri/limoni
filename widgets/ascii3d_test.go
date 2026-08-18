package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

func TestDuckCanvasUIFrame(t *testing.T) {
	rect := cell.NewRect(0, 0, 70, 32)
	duck := graphics.NewDuck()

	buf := buffer.NewBuffer(rect)
	ctx := cell.Context{Area: rect}
	w := Ascii3D{
		Model:          duck,
		Scale:          4.2,
		RotX:           10.0,
		RotY:           180.0,
		FOV:            65.0,
		CameraDistance: 4.2,
		Contrast:       1.2,
		Exposure:       1.0,
		Ascii:          true,
		Colored:        false,
		Ramp:           RampCanvasUI,
	}
	w.Draw(ctx, buf)

	var out strings.Builder
	for y := uint16(0); y < 32; y++ {
		for x := uint16(0); x < 70; x++ {
			ch := buf.Get(x, y).Content
			if ch == 0 {
				ch = ' '
			}
			out.WriteRune(ch)
		}
		out.WriteByte('\n')
	}
	t.Logf("=== Radiant Lit Duck at RotY: 180 ===\n%s", out.String())
}

func TestBackOfDuckNoBeak(t *testing.T) {
	rect := cell.NewRect(0, 0, 60, 30)
	duck := graphics.NewDuck()

	buf := buffer.NewBuffer(rect)
	ctx := cell.Context{Area: rect}
	w := Ascii3D{
		Model:          duck,
		Scale:          4.0,
		RotX:           0.0,
		RotY:           0.0, // Looking from behind (back of head facing camera)
		FOV:            65.0,
		CameraDistance: 4.2,
		Ascii:          true,
		Colored:        true,
	}
	w.Draw(ctx, buf)

	orangeCount := 0
	yellowCount := 0
	for y := uint16(0); y < 30; y++ {
		for x := uint16(0); x < 60; x++ {
			c := buf.Get(x, y)
			if c.Content != ' ' && c.Content != 0 {
				r, g, _ := c.Style.Fg.RGB()
				if r > 180 && g > 140 {
					yellowCount++
				}
				if r > 180 && g < 140 {
					orangeCount++
				}
			}
		}
	}

	t.Logf("Back of Duck (RotY=0): Yellow body pixels=%d, Orange beak pixels=%d", yellowCount, orangeCount)
	if orangeCount > 0 {
		t.Errorf("expected 0 orange beak pixels visible from behind the duck, got %d", orangeCount)
	}
	if yellowCount == 0 {
		t.Errorf("expected yellow body visible from behind")
	}
}

func TestAscii3DModes(t *testing.T) {
	rect := cell.NewRect(0, 0, 40, 20)
	duck := graphics.NewDuck()

	modes := []struct {
		name string
		mode Ascii3DMode
	}{
		{"ModeASCII", ModeASCII},
		{"ModeBlock", ModeBlock},
		{"ModeDithered", ModeDithered},
		{"ModeBraille", ModeBraille},
	}

	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			buf := buffer.NewBuffer(rect)
			ctx := cell.Context{Area: rect}
			w := Ascii3D{
				Model:          duck,
				Mode:           m.mode,
				Scale:          4.0,
				RotX:           10.0,
				RotY:           180.0,
				CameraDistance: 4.2,
				Colored:        true,
			}
			w.Draw(ctx, buf)

			nonEmpty := 0
			for y := uint16(0); y < 20; y++ {
				for x := uint16(0); x < 40; x++ {
					c := buf.Get(x, y)
					if c.Content != ' ' && c.Content != 0 {
						nonEmpty++
					}
				}
			}
			if nonEmpty == 0 {
				t.Fatalf("expected rendered pixels in mode %s, got 0", m.name)
			}
			t.Logf("Mode %s: %d non-empty cells rendered", m.name, nonEmpty)
		})
	}
}

func TestAscii3DOptions(t *testing.T) {
	rect := cell.NewRect(0, 0, 30, 15)
	buf := buffer.NewBuffer(rect)
	ctx := cell.Context{
		Area: rect,
	}

	cube := graphics.NewCube(2.0)
	obj := AsciiObject{
		Model:   cube,
		Scale:   2.0,
		Ascii:   false, // solid block
		Colored: false,
		Invert:  true,
		Color:   cell.NewColorRGB(255, 200, 0),
		Ramp:    RampBlocks,
	}

	obj.Draw(ctx, buf)

	drawn := false
	for y := uint16(0); y < 15; y++ {
		for x := uint16(0); x < 30; x++ {
			c := buf.Get(x, y)
			if c.Content == '█' {
				drawn = true
				break
			}
		}
		if drawn {
			break
		}
	}

	if !drawn {
		t.Fatalf("expected solid block '█' to be rendered with Ascii: false")
	}
}
