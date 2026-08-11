package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestModalCenteringAndContains(t *testing.T) {
	parent := cell.NewRect(0, 0, 80, 24)
	w, h := uint16(40), uint16(10)
	centered := CenterRect(parent, w, h)

	if centered.Width != w || centered.Height != h {
		t.Errorf("CenterRect dimensions = (%d, %d); (%d, %d) bekleniyordu", centered.Width, centered.Height, w, h)
	}

	expectedX := uint16((80 - 40) / 2) // 20
	expectedY := uint16((24 - 10) / 2) // 7
	if centered.X != expectedX || centered.Y != expectedY {
		t.Errorf("CenterRect coordinates = (%d, %d); (%d, %d) bekleniyordu", centered.X, centered.Y, expectedX, expectedY)
	}

	// ContainsRect testleri
	inside := cell.NewRect(25, 8, 10, 5)
	if !ContainsRect(centered, inside) {
		t.Errorf("ContainsRect = false; inside rect should be contained in centered")
	}

	outside := cell.NewRect(10, 5, 20, 10)
	if ContainsRect(centered, outside) {
		t.Errorf("ContainsRect = true; outside rect should not be contained in centered")
	}
}

type dummyWidget struct {
	id string
}

func (dw dummyWidget) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ctx.RegisterFocus != nil {
		ctx.RegisterFocus(dw.id)
	}
	if ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {})
	}
}

func (dw dummyWidget) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}

func TestModalFocusAndClickTrapping(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	focusMgr := NewFocusManager()
	frame := NewFrame(buf, focusMgr)

	// Modal alanını belirle (merkezde 40x10)
	modalArea := cell.NewRect(20, 7, 40, 10)
	frame.RegisterModal("test_modal", modalArea, nil)

	// 1. Modal içerisindeki widget'ı çiz (odaklanabilir/tıklanabilir olmalı)
	insideW := dummyWidget{id: "inside"}
	frame.RenderWidget(insideW, cell.NewRect(25, 8, 10, 2))

	// 2. Modal dışındaki widget'ı çiz (odaklanması ve tıklanması bloklanmalı!)
	outsideW := dummyWidget{id: "outside"}
	frame.RenderWidget(outsideW, cell.NewRect(0, 0, 10, 2))

	// Doğrulama
	if len(focusMgr.focusable) != 1 {
		t.Errorf("Focus list len = %d; 1 bekleniyordu (outside widget engellenmeliydi)", len(focusMgr.focusable))
	}
	if focusMgr.focusable[0] != "inside" {
		t.Errorf("Focus list elements = %v; ['inside'] bekleniyordu", focusMgr.focusable)
	}

	if len(frame.ClickRegions) != 1 {
		t.Errorf("Click regions count = %d; 1 bekleniyordu (outside widget engellenmeliydi)", len(frame.ClickRegions))
	}
}

func TestRouteMouseEventWithModal(t *testing.T) {
	trm := &Terminal{
		frame: NewFrame(nil, NewFocusManager()),
	}

	// Modal kaydet
	modalArea := cell.NewRect(20, 7, 40, 10)
	outsideClicked := false
	trm.frame.RegisterModal("test_modal", modalArea, func() {
		outsideClicked = true
	})

	// Modal içi click area
	insideTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area: cell.NewRect(25, 8, 5, 2),
		Handler: func(ev backend.MouseEvent) {
			insideTriggered = true
		},
	})

	// Modal dışı click area
	outsideTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area: cell.NewRect(5, 5, 5, 2),
		Handler: func(ev backend.MouseEvent) {
			outsideTriggered = true
		},
	})

	// 1. Modal içine tıklama
	trm.RouteMouseEvent(backend.MouseEvent{X: 27, Y: 9, Button: backend.MouseLeft})
	if !insideTriggered {
		t.Errorf("Modal içi tıklama tetiklenmedi!")
	}

	// 2. Modal dışına tıklama (click-outside tetiklenmeli ve dışarıdaki handler engellenmeli)
	trm.RouteMouseEvent(backend.MouseEvent{X: 6, Y: 6, Button: backend.MouseLeft})
	if !outsideClicked {
		t.Errorf("ClickOutside callback tetiklenmedi!")
	}
	if outsideTriggered {
		t.Errorf("Modal dışındaki alt handler tetiklendi! Sızma engellenmeliydi.")
	}
}

func TestLayerSystemBasic(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	focusMgr := NewFocusManager()
	frame := NewFrame(buf, focusMgr)

	// 1. Kök katmanda bir widget çiz (ActiveModal yok, layer yok → serbest)
	rootWidget := dummyWidget{id: "root_item"}
	frame.RenderWidget(rootWidget, cell.NewRect(5, 5, 10, 2))
	if len(focusMgr.focusable) != 1 || focusMgr.focusable[0] != "root_item" {
		t.Errorf("Kök widget odaklanamadı: %v", focusMgr.focusable)
	}
	// Temizle: İlk kök widget'ın tıklama alanlarını temizle ki sonraki testleri etkilemesin
	frame.ClickRegions = frame.ClickRegions[:0]
	frame.FocusManager.Clear()

	// 2. Bir modal katman kaydet (sadece RegisterLayer ile)
	modalArea := cell.NewRect(20, 7, 40, 10)
	frame.RegisterLayer("modal1", LayerModal, modalArea, 1000, nil)

	// 3. Kök katmanda widget çiz → modal varken engellenmeli
	outsideWidget := dummyWidget{id: "root_outside"}
	frame.RenderWidget(outsideWidget, cell.NewRect(5, 5, 10, 2))
	if len(focusMgr.focusable) != 0 {
		t.Errorf("Kök widget odaklanmamalıydı (modal aktif): %v", focusMgr.focusable)
	}

	// 4. BeginLayer ile modal içinde çiz → kaydedilmeli
	insideWidget := dummyWidget{id: "modal_item"}
	frame.BeginLayer("modal1")
	frame.RenderWidget(insideWidget, cell.NewRect(25, 8, 10, 2))
	frame.EndLayer()

	if len(focusMgr.focusable) != 1 {
		t.Errorf("Focus list len = %d; 1 bekleniyordu (modal içindeki widget)", len(focusMgr.focusable))
	}
	if focusMgr.focusable[0] != "modal_item" {
		t.Errorf("Focus = %q; 'modal_item' bekleniyordu", focusMgr.focusable[0])
	}

	// 5. Tıklama alanları kontrolü
	if len(frame.ClickRegions) != 1 {
		t.Errorf("Click regions = %d; 1 bekleniyordu", len(frame.ClickRegions))
	}
	if frame.ClickRegions[0].LayerID != "modal1" {
		t.Errorf("Click region LayerID = %q; 'modal1' bekleniyordu", frame.ClickRegions[0].LayerID)
	}
}

func TestMultiLayerZOrdering(t *testing.T) {
	trm := &Terminal{
		frame: NewFrame(nil, NewFocusManager()),
	}

	// Katman 1 (düşük z-index)
	layer1Area := cell.NewRect(0, 0, 40, 12)
	trm.frame.RegisterLayer("layer_low", LayerModal, layer1Area, 100, nil)

	// Katman 2 (yüksek z-index) - layer1'in üstünde
	layer2Area := cell.NewRect(20, 5, 30, 10)
	layer2Clicked := false
	trm.frame.RegisterLayer("layer_high", LayerModal, layer2Area, 200, func() {
		layer2Clicked = true
	})

	// Katman 2 içindeki click area
	highTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area:    cell.NewRect(25, 6, 10, 2),
		Handler: func(ev backend.MouseEvent) { highTriggered = true },
		LayerID: "layer_high",
	})

	// Katman 1 içindeki click area (ama üstteki katman 2 ile kesişiyor)
	lowTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area:    cell.NewRect(25, 6, 10, 2),
		Handler: func(ev backend.MouseEvent) { lowTriggered = true },
		LayerID: "layer_low",
	})

	// Kesişim alanına tıklama: En üstteki katman (layer_high) yakalamalı
	trm.RouteMouseEvent(backend.MouseEvent{X: 27, Y: 7, Button: backend.MouseLeft})
	if !highTriggered {
		t.Errorf("En üst katmandaki handler tetiklenmedi!")
	}
	if lowTriggered {
		t.Errorf("Alt katmandaki handler tetiklendi! Üst katman engellemeliydi.")
	}

	// layer_high dışına tıklama → layer_high'ın ClickOutside tetiklenmeli
	trm.RouteMouseEvent(backend.MouseEvent{X: 5, Y: 5, Button: backend.MouseLeft})
	if !layer2Clicked {
		t.Errorf("En üst katmanın ClickOutside tetiklenmedi!")
	}
}

func TestRemoveLayer(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	focusMgr := NewFocusManager()
	frame := NewFrame(buf, focusMgr)

	frame.RegisterLayer("modal_a", LayerModal, cell.NewRect(10, 5, 30, 10), 100, nil)
	frame.RegisterLayer("popup_b", LayerPopup, cell.NewRect(50, 5, 20, 8), 200, nil)

	if len(frame.Layers) != 2 {
		t.Errorf("Layer count = %d; 2 bekleniyordu", len(frame.Layers))
	}

	// popup_b katmanını kaldır
	frame.RemoveLayer("popup_b")
	if len(frame.Layers) != 1 {
		t.Errorf("Layer count after remove = %d; 1 bekleniyordu", len(frame.Layers))
	}
	if frame.Layers[0].ID != "modal_a" {
		t.Errorf("Remaining layer ID = %q; 'modal_a' bekleniyordu", frame.Layers[0].ID)
	}
}

func TestTopLayer(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	frame := NewFrame(buf, NewFocusManager())

	if frame.TopLayer() != nil {
		t.Errorf("TopLayer() = nil değil; boş katman listesinde nil bekleniyordu")
	}

	frame.RegisterLayer("low", LayerModal, cell.NewRect(0, 0, 40, 10), 100, nil)
	frame.RegisterLayer("high", LayerModal, cell.NewRect(20, 5, 30, 10), 300, nil)

	top := frame.TopLayer()
	if top == nil {
		t.Errorf("TopLayer() = nil; yüksek z-index'li katman bekleniyordu")
	}
	if top.ID != "high" {
		t.Errorf("TopLayer().ID = %q; 'high' bekleniyordu", top.ID)
	}
}

func TestResetClearsLayers(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	frame := NewFrame(buf, NewFocusManager())

	frame.RegisterLayer("a", LayerModal, cell.NewRect(0, 0, 40, 10), 100, nil)
	frame.RegisterLayer("b", LayerPopup, cell.NewRect(20, 5, 30, 10), 200, nil)
	frame.BeginLayer("a")

	frame.Reset()

	if len(frame.Layers) != 0 {
		t.Errorf("After Reset, Layers len = %d; 0 bekleniyordu", len(frame.Layers))
	}
	if frame.activeLayerID != "" {
		t.Errorf("After Reset, activeLayerID = %q; '' bekleniyordu", frame.activeLayerID)
	}
	if frame.ActiveModal != nil {
		t.Errorf("After Reset, ActiveModal = nil değildi")
	}
}

func TestRegisterImageZIndexMapping(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	frame := NewFrame(buf, NewFocusManager())

	// We render a dummy widget that registers an image.
	// 1. An explicit negative z-index is preserved without a modal.
	frame.RenderWidget(dummyImageWidget{zIndex: -1}, cell.NewRect(10, 5, 20, 10))
	if len(frame.ImageRegions) != 1 {
		t.Fatalf("Expected 1 image region, got %d", len(frame.ImageRegions))
	}
	if frame.ImageRegions[0].ZIndex != -1 {
		t.Errorf("Expected explicit ZIndex -1 when no modal, got %d", frame.ImageRegions[0].ZIndex)
	}

	// Reset frame
	frame.Reset()

	// 2. The explicit z-index remains below the modal when it overlaps.
	frame.RegisterModal("modal", cell.NewRect(15, 5, 20, 10), nil)
	frame.RenderWidget(dummyImageWidget{zIndex: -1}, cell.NewRect(10, 5, 20, 10))
	if len(frame.ImageRegions) != 1 {
		t.Fatalf("Expected 1 image region, got %d", len(frame.ImageRegions))
	}
	if frame.ImageRegions[0].ZIndex != -1 {
		t.Errorf("Expected explicit ZIndex -1 due to modal overlap, got %d", frame.ImageRegions[0].ZIndex)
	}
}

type dummyImageWidget struct {
	zIndex int
}

func (d dummyImageWidget) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if ctx.RegisterImage != nil {
		ctx.RegisterImage(ctx.Area, nil, d.zIndex, false)
	}
}

func (d dummyImageWidget) SizeHint(maxArea cell.Rect) (width, height uint16) {
	return maxArea.Width, maxArea.Height
}
