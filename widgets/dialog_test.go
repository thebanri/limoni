package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func scaleRect(base cell.Rect, progress float64) cell.Rect {
	if progress <= 0 {
		return cell.NewRect(base.X+base.Width/2, base.Y+base.Height/2, 0, 0)
	}
	if progress >= 1.0 {
		return base
	}
	w := uint16(float64(base.Width) * progress)
	h := uint16(float64(base.Height) * progress)
	if w%2 != 0 && w < base.Width {
		w++
	}
	if h%2 != 0 && h < base.Height {
		h++
	}
	x := base.X + (base.Width-w)/2
	y := base.Y + (base.Height-h)/2
	return cell.NewRect(x, y, w, h)
}

func TestDialogDrawBasic(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 50, 20))
	dialog := Dialog{
		ID:         "test_dialog",
		Title:      " ⚠️ SYSTEM EXIT ",
		Message:    "Are you sure you want to exit the application?",
		SubMessage: "The session and all unsaved state will be terminated.",
		Buttons: []DialogButton{
			{Text: "Yes"},
			{Text: "No"},
		},
		Shadow: true,
	}

	dialogArea := cell.NewRect(2, 2, 46, 9)
	var clickCount int
	ctx := cell.NewContext(dialogArea, cell.Style{})
	ctx.RegisterClick = func(area cell.Rect, handler func()) {
		clickCount++
	}

	dialog.Draw(ctx, buf)

	if clickCount != 2 {
		t.Errorf("expected 2 button click registrations, got %d", clickCount)
	}

	// Verify top-left corner
	topLeft := buf.Get(2, 2)
	if topLeft == nil || topLeft.Content != SymbolsRounded.TopLeft {
		t.Errorf("expected top-left border symbol at (2,2), got %v", topLeft)
	}

	// Verify bottom-right corner
	bottomRight := buf.Get(2+46-1, 2+9-1)
	if bottomRight == nil || bottomRight.Content != SymbolsRounded.BottomRight {
		t.Errorf("expected bottom-right border symbol at (47,10), got %v", bottomRight)
	}
}

func TestDialogNoOverflowOnScaling(t *testing.T) {
	bufW, bufH := uint16(80), uint16(30)
	dialog := Dialog{
		ID:         "exit_dialog",
		Title:      " ⚠️ SYSTEM EXIT ",
		Message:    "Are you sure you want to exit the application?",
		SubMessage: "The session and all unsaved state will be terminated.",
		Buttons: []DialogButton{
			{Text: "Yes"},
			{Text: "No"},
		},
		Shadow: false, // test strict bounding without shadow
	}

	baseArea := cell.NewRect(15, 10, 46, 9)

	for progress := 0.0; progress <= 1.0; progress += 0.05 {
		scaledArea := scaleRect(baseArea, progress)

		// Create buffer filled with sentinel character
		buf := buffer.NewBuffer(cell.NewRect(0, 0, bufW, bufH))
		for y := uint16(0); y < bufH; y++ {
			for x := uint16(0); x < bufW; x++ {
				if c := buf.Get(x, y); c != nil {
					c.Content = '.'
					c.Style = cell.Style{Fg: cell.NewColorRGB(100, 100, 100)}
				}
			}
		}

		ctx := cell.NewContext(scaledArea, cell.Style{})
		dialog.Draw(ctx, buf)

		// Assert NO cell outside scaledArea was modified!
		for y := uint16(0); y < bufH; y++ {
			for x := uint16(0); x < bufW; x++ {
				isInside := x >= scaledArea.X && x < scaledArea.X+scaledArea.Width &&
					y >= scaledArea.Y && y < scaledArea.Y+scaledArea.Height

				c := buf.Get(x, y)
				if !isInside {
					if c.Content != '.' {
						t.Fatalf("at progress %f, scaledArea %v, cell outside bounds at (%d,%d) was overwritten with '%c'!",
							progress, scaledArea, x, y, c.Content)
					}
				}
			}
		}
	}
}

func TestDialogSmallDimensions(t *testing.T) {
	dialog := Dialog{
		ID:      "small_dialog",
		Title:   "Short",
		Message: "Hello",
		Buttons: []DialogButton{{Text: "OK"}},
	}

	// Dimensions from 0 to 15 width, 0 to 8 height
	for w := uint16(0); w <= 15; w++ {
		for h := uint16(0); h <= 8; h++ {
			buf := buffer.NewBuffer(cell.NewRect(0, 0, 20, 10))
			ctx := cell.NewContext(cell.NewRect(2, 2, w, h), cell.Style{})
			// Must not panic, crash, or underflow
			dialog.Draw(ctx, buf)
		}
	}
}
