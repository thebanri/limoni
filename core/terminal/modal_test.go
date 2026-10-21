package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
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

	// ContainsRect tests
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

	// Place the modal (40x10, centred)
	modalArea := cell.NewRect(20, 7, 40, 10)
	frame.RegisterModal("test_modal", modalArea, nil)

	// 1. Draw the widget inside the modal (it must be focusable and clickable)
	insideW := dummyWidget{id: "inside"}
	frame.RenderWidget(insideW, cell.NewRect(25, 8, 10, 2))

	// 2. Draw the widget outside the modal (its focus and clicks must be blocked!)
	outsideW := dummyWidget{id: "outside"}
	frame.RenderWidget(outsideW, cell.NewRect(0, 0, 10, 2))

	// Checks
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

	// Register modal
	modalArea := cell.NewRect(20, 7, 40, 10)
	outsideClicked := false
	trm.frame.RegisterModal("test_modal", modalArea, func() {
		outsideClicked = true
	})

	// Click area inside the modal
	insideTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area:    cell.NewRect(25, 8, 5, 2),
		LayerID: "test_modal",
		Handler: func(ev driver.MouseEvent) {
			insideTriggered = true
		},
	})

	// Background click area under the modal (overlapping coordinates)
	backgroundTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area:    cell.NewRect(22, 8, 10, 2),
		LayerID: "",
		Handler: func(ev driver.MouseEvent) {
			backgroundTriggered = true
		},
	})

	// Click area outside the modal
	outsideTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area:    cell.NewRect(5, 5, 5, 2),
		LayerID: "",
		Handler: func(ev driver.MouseEvent) {
			outsideTriggered = true
		},
	})

	// 1. Click on empty space inside the modal (must not leak to the background!)
	handled := trm.RouteMouseEvent(driver.MouseEvent{X: 22, Y: 8, Button: driver.MouseLeft})
	if !handled {
		t.Errorf("Modal içi boşluk tıklaması yutulmalıydı (handled=true)!")
	}
	if backgroundTriggered {
		t.Errorf("Modal altındaki arka plan bölgesi tetiklendi! Sızma engellenmeliydi.")
	}

	// 2. Click the button inside the modal
	trm.RouteMouseEvent(driver.MouseEvent{X: 27, Y: 9, Button: driver.MouseLeft})
	if !insideTriggered {
		t.Errorf("Modal içi tıklama tetiklenmedi!")
	}
	if backgroundTriggered {
		t.Errorf("Modal altındaki arka plan bölgesi tetiklendi!")
	}

	// 3. Click outside the modal (click-outside must fire and the handler outside must be blocked)
	trm.RouteMouseEvent(driver.MouseEvent{X: 6, Y: 6, Button: driver.MouseLeft})
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

	// 1. Draw a widget in the root layer (no ActiveModal, no layer → unrestricted)
	rootWidget := dummyWidget{id: "root_item"}
	frame.RenderWidget(rootWidget, cell.NewRect(5, 5, 10, 2))
	if len(focusMgr.focusable) != 1 || focusMgr.focusable[0] != "root_item" {
		t.Errorf("Kök widget odaklanamadı: %v", focusMgr.focusable)
	}
	// Clean up: clear the first root widget's click areas so they do not affect the next steps
	frame.ClickRegions = frame.ClickRegions[:0]
	frame.FocusManager.Clear()

	// 2. Register a modal layer (with RegisterLayer only)
	modalArea := cell.NewRect(20, 7, 40, 10)
	frame.RegisterLayer("modal1", LayerModal, modalArea, 1000, nil)

	// 3. Draw a widget in the root layer → must be blocked while the modal exists
	outsideWidget := dummyWidget{id: "root_outside"}
	frame.RenderWidget(outsideWidget, cell.NewRect(5, 5, 10, 2))
	if len(focusMgr.focusable) != 0 {
		t.Errorf("Kök widget odaklanmamalıydı (modal aktif): %v", focusMgr.focusable)
	}

	// 4. Draw inside the modal with BeginLayer → must be registered
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

	// 5. Check the click areas
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

	// Layer 1 (low z-index)
	layer1Area := cell.NewRect(0, 0, 40, 12)
	trm.frame.RegisterLayer("layer_low", LayerModal, layer1Area, 100, nil)

	// Layer 2 (high z-index) - above layer1
	layer2Area := cell.NewRect(20, 5, 30, 10)
	layer2Clicked := false
	trm.frame.RegisterLayer("layer_high", LayerModal, layer2Area, 200, func() {
		layer2Clicked = true
	})

	// Click area inside layer 2
	highTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area:    cell.NewRect(25, 6, 10, 2),
		Handler: func(ev driver.MouseEvent) { highTriggered = true },
		LayerID: "layer_high",
	})

	// Click area inside layer 1 (but overlapping layer 2 above it)
	lowTriggered := false
	trm.frame.ClickRegions = append(trm.frame.ClickRegions, ClickRegion{
		Area:    cell.NewRect(25, 6, 10, 2),
		Handler: func(ev driver.MouseEvent) { lowTriggered = true },
		LayerID: "layer_low",
	})

	// Click in the overlap: the topmost layer (layer_high) must catch it
	trm.RouteMouseEvent(driver.MouseEvent{X: 27, Y: 7, Button: driver.MouseLeft})
	if !highTriggered {
		t.Errorf("En üst katmandaki handler tetiklenmedi!")
	}
	if lowTriggered {
		t.Errorf("Alt katmandaki handler tetiklendi! Üst katman engellemeliydi.")
	}

	// Click outside layer_high → layer_high's ClickOutside must fire
	trm.RouteMouseEvent(driver.MouseEvent{X: 5, Y: 5, Button: driver.MouseLeft})
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

	// Remove the popup_b layer
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
	if frame.TopmostModal() != nil {
		t.Errorf("after Reset a modal is still registered")
	}
}

func TestRegisterImageZIndexMapping(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
	frame := NewFrame(buf, NewFocusManager())

	// We render a dummy widget that registers an image.
	// 1. An explicit negative z-index is preserved without a modal.
	frame.RenderWidget(dummyImageWidget{zIndex: -1}, cell.NewRect(10, 5, 20, 10))
	if len(frame.imageRegions) != 1 {
		t.Fatalf("Expected 1 image region, got %d", len(frame.imageRegions))
	}
	if frame.imageRegions[0].ZIndex != -1 {
		t.Errorf("Expected explicit ZIndex -1 when no modal, got %d", frame.imageRegions[0].ZIndex)
	}

	// Reset frame
	frame.Reset()

	// 2. The explicit z-index remains below the modal when it overlaps.
	frame.RegisterModal("modal", cell.NewRect(15, 5, 20, 10), nil)
	frame.RenderWidget(dummyImageWidget{zIndex: -1}, cell.NewRect(10, 5, 20, 10))
	if len(frame.imageRegions) != 1 {
		t.Fatalf("Expected 1 image region, got %d", len(frame.imageRegions))
	}
	if frame.imageRegions[0].ZIndex != -1 {
		t.Errorf("Expected explicit ZIndex -1 due to modal overlap, got %d", frame.imageRegions[0].ZIndex)
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
