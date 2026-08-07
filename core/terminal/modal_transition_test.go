package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestModalScaleRect(t *testing.T) {
	base := cell.NewRect(10, 20, 100, 50)

	// Progress 0.0 -> Size 0, center coordinates
	r0 := ScaleRect(base, 0.0)
	if r0.Width != 0 || r0.Height != 0 {
		t.Errorf("ScaleRect at 0.0 size = (%d, %d); expected (0, 0)", r0.Width, r0.Height)
	}

	// Progress 0.5 -> Size 50x25, centered
	r5 := ScaleRect(base, 0.5)
	if r5.Width != 50 || r5.Height != 25 {
		t.Errorf("ScaleRect at 0.5 size = (%d, %d); expected (50, 25)", r5.Width, r5.Height)
	}
	expectedX := base.X + (base.Width-50)/2
	expectedY := base.Y + (base.Height-25)/2
	if r5.X != expectedX || r5.Y != expectedY {
		t.Errorf("ScaleRect at 0.5 pos = (%d, %d); expected (%d, %d)", r5.X, r5.Y, expectedX, expectedY)
	}

	// Progress 1.0 -> Same as base
	r10 := ScaleRect(base, 1.0)
	if r10 != base {
		t.Errorf("ScaleRect at 1.0 = %v; expected %v", r10, base)
	}
}

func TestModalSlideUpRect(t *testing.T) {
	base := cell.NewRect(10, 20, 100, 50)
	parentHeight := uint16(100)

	// Progress 0.0 -> Starts offscreen (startY = parentHeight = 100)
	r0 := SlideUpRect(base, parentHeight, 0.0)
	if r0.Y != parentHeight {
		t.Errorf("SlideUpRect at 0.0 Y = %d; expected %d", r0.Y, parentHeight)
	}

	// Progress 0.5 -> Midpoint between 100 and 20
	r5 := SlideUpRect(base, parentHeight, 0.5)
	expectedY := uint16(60) // 100 - (100-20)*0.5 = 60
	if r5.Y != expectedY {
		t.Errorf("SlideUpRect at 0.5 Y = %d; expected %d", r5.Y, expectedY)
	}

	// Progress 1.0 -> Reached target
	r10 := SlideUpRect(base, parentHeight, 1.0)
	if r10.Y != base.Y {
		t.Errorf("SlideUpRect at 1.0 Y = %d; expected %d", r10.Y, base.Y)
	}
}

// DummyWidget is a simple widget for testing sandboxing clicks.
type DummyWidget struct {
	ID string
}

func (d DummyWidget) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(d.ID)
	}
	if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {})
	}
}

func (d DummyWidget) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

func TestModalStackSandboxing(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 120, 40))
	focusMgr := NewFocusManager()
	f := NewFrame(buf, focusMgr)

	// Modal A (ZIndex: 100) ve Modal B (ZIndex: 200) ekle
	areaA := cell.NewRect(10, 10, 20, 10)
	areaB := cell.NewRect(40, 10, 20, 10)

	f.RegisterLayer("modal_a", LayerModal, areaA, 100, func() {})
	f.RegisterLayer("modal_b", LayerModal, areaB, 200, func() {})

	// Test 1: Modal A içindeki bir widget çizilirken odak/tıklama bloke olmalı (çünkü Modal B en üstte)
	f.activeLayerID = "modal_a"
	dummyA := DummyWidget{ID: "widget_a"}
	f.RenderWidget(dummyA, areaA)

	if focusMgr.Focused() == "widget_a" {
		t.Errorf("Modal A'daki widget odaklandı, en üstte Modal B varken bloke olmalıydı")
	}

	// Test 2: Modal B içindeki bir widget çizilirken odak/tıklama onaylanmalı (çünkü kendisi en üstte)
	f.activeLayerID = "modal_b"
	dummyB := DummyWidget{ID: "widget_b"}
	f.RenderWidget(dummyB, areaB)

	if focusMgr.Focused() != "widget_b" {
		t.Errorf("Modal B'deki widget odaklanamadı, en üstte o varken serbest olmalıydı")
	}
}
