package terminal

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/widgets"
)

// ClickRegion, ekranda tıklanabilir (interaktif) bir bölgeyi ve bu bölgeye tıklandığında
// çalıştırılacak olan fare olay yöneticisi (callback) fonksiyonunu tanımlar.
type ClickRegion struct {
	// Area, tıklanabilir bölgenin ekran koordinatları ve boyut sınırlarıdır.
	Area cell.Rect
	// Handler, bu alana tıklandığında tetiklenecek olan olay yöneticisi fonksiyonudur.
	Handler func(ev backend.MouseEvent)
	// LayerID, bu tıklama bölgesinin hangi katmana ait olduğunu belirtir.
	// Boş string ise kök (root) katmanına aittir.
	LayerID string
	// MouseOnly, MouseNone hareket/hover olaylarının da bu bölgeye yönlendirilmesini sağlar.
	MouseOnly bool
}

// ImageRegion, ekranda grafik olarak çizdirilmek istenen bir resmi ve bu resmin
// çizileceği hedef hücre koordinatlarını tanımlar.
type ImageRegion struct {
	// Area, resmin çizileceği hedef hücre koordinatları ve satır/sütun boyutlarıdır.
	Area cell.Rect
	// Img, çizilecek olan ham resim verisidir.
	Img image.Image
	// ZIndex, resmin dikey katman sıralama parametresidir.
	ZIndex int
	// Transparent, resmin şeffaf piksellerinin korunup korunmayacağını belirtir.
	Transparent bool
}

// Frame, tek bir çizim karesinin (render pass) bağlamını temsil eder.
// Çizim işlemi sırasında hem ham tampona yazmayı hem de interaktif tıklama, resim çizim alanları ve odak yönetimini kaydetmeyi yönetir.
type Frame struct {
	// Buffer, bu karede üzerine çizim yapılan aktif terminal hücre matrisidir.
	Buffer *buffer.Buffer

	// ClickRegions, bu karede widget'lar tarafından kaydedilen tıklanabilir bölgeler listesidir.
	ClickRegions []ClickRegion
	EventRegions []eventRegion

	// ImageRegions, bu karede widget'lar tarafından kaydedilen resim çizim alanları listesidir.
	ImageRegions []ImageRegion

	// FocusManager, bu karedeki odak durumunu ve sekmeli geçiş sırasını yönetir.
	FocusManager *FocusManager

	// ActiveModal, bu çizim karesinde etkin olan en üst modal katman bilgisidir.
	ActiveModal *Modal

	// Layers, bu çizim karesinde aktif olan katmanların z-index sırasına göre listesidir.
	// En yüksek z-index en sonda (en üstte) yer alır.
	Layers []Layer

	// activeLayerID, çizim sırasında mevcut katmanın ID'sini tutar.
	activeLayerID string

	// DebugRegions, bu çizim karesinde çizilen widget'ların yerleşim alanlarını saklar.
	DebugRegions []DebugRegion

	// mouseCaptureRequest, çizim sırasında bir widget tarafından talep edilen fare yakalama callback'idir.
	mouseCaptureRequest func(ev backend.MouseEvent)
	hoveredRegionID     string
	lastClickID         string
	lastClickAt         time.Time

	Theme    widgets.Theme
	ThemeSet bool

	// WidgetStats, bu çizim karesinde çizilen widget'ların render sürelerini saklar.
	WidgetStats   []WidgetStat
	Accessibility []accessibility.AccessibilityNode
}

// DispatchClick dispatches a click to the topmost enabled target region and
// reports ClickCount 2 when the same region is clicked twice within 500ms.
// The timestamp is supplied by the caller to keep tests deterministic.
func (f *Frame) DispatchClick(ev backend.MouseEvent, at time.Time) bool {
	if f == nil {
		return false
	}
	var target *eventRegion
	for i := len(f.EventRegions) - 1; i >= 0; i-- {
		region := &f.EventRegions[i]
		if region.Phase == TargetPhase && !region.Disabled && region.Area.Contains(ev.X, ev.Y) {
			target = region
			break
		}
	}
	if target == nil {
		return false
	}
	clickCount := 1
	if f.lastClickID == target.ID && !f.lastClickAt.IsZero() && at.Sub(f.lastClickAt) >= 0 && at.Sub(f.lastClickAt) <= 500*time.Millisecond {
		clickCount = 2
	}
	f.lastClickID = target.ID
	f.lastClickAt = at
	ctx := &backend.EventContext{
		Mouse: ev, Phase: TargetPhase, RegionID: target.ID,
		LayerID: target.LayerID, ZIndex: target.ZIndex,
		ClickCount: clickCount, EventTime: at,
	}
	target.Handler(ctx)
	return true
}

type WidgetStat struct {
	Type     string
	Duration time.Duration
}

type DebugRegion struct {
	Area       cell.Rect
	WidgetType string
	ZIndex     int
}

// NewFrame, belirtilen buffer ve odak yöneticisi üzerinde çizim yapacak yeni bir Frame örneği oluşturur.
func NewFrame(buf *buffer.Buffer, focusMgr *FocusManager) *Frame {
	return &Frame{
		Buffer:       buf,
		ClickRegions: make([]ClickRegion, 0, 32),
		EventRegions: make([]eventRegion, 0, 16),
		ImageRegions: make([]ImageRegion, 0, 8),
		FocusManager: focusMgr,
		ActiveModal:  nil,
		Layers:       make([]Layer, 0, 4),
		DebugRegions: make([]DebugRegion, 0, 32),
	}
}

// Reset, çizim karesinin durumunu (kaydedilmiş tıklama, resim alanları, modal ve katmanları) sıfırlar.
// Bellek Optimizasyonu: Slice kapasitesini koruyarak sıfır tahsisatla listeyi temizler (slice[:0]).
// BeginFocusScope restricts keyboard navigation to widgets rendered in the scope.
func (f *Frame) BeginFocusScope(id string) {
	if f.FocusManager != nil {
		f.FocusManager.BeginScope(id)
	}
}

func (f *Frame) EndFocusScope() {
	if f.FocusManager != nil {
		f.FocusManager.EndScope()
	}
}

// IsFocused reports whether a widget owns focus in this frame.
func (f *Frame) IsFocused(id string) bool {
	return f.FocusManager != nil && f.FocusManager.IsFocused(id)
}

// SetTheme sets the semantic theme inherited by widgets rendered in this frame.
func (f *Frame) SetTheme(theme widgets.Theme) {
	f.Theme = theme
	f.ThemeSet = true
}

func (f *Frame) Reset() {
	f.ClickRegions = f.ClickRegions[:0]
	f.EventRegions = f.EventRegions[:0]
	f.ImageRegions = f.ImageRegions[:0]
	f.ActiveModal = nil
	f.Layers = f.Layers[:0]
	f.activeLayerID = ""
	f.DebugRegions = f.DebugRegions[:0]
	f.mouseCaptureRequest = nil
	f.hoveredRegionID = ""
	f.lastClickID = ""
	f.lastClickAt = time.Time{}
	f.WidgetStats = f.WidgetStats[:0]
	f.Accessibility = f.Accessibility[:0]
}

// RegisterAccessibility adds a semantic node to the current frame tree.
func (f *Frame) RegisterAccessibility(node accessibility.AccessibilityNode) {
	if f != nil {
		f.Accessibility = append(f.Accessibility, node)
	}
}

// AccessibilityTree returns the nodes registered during the current frame.
func (f *Frame) AccessibilityTree() []accessibility.AccessibilityNode {
	if f == nil {
		return nil
	}
	result := make([]accessibility.AccessibilityNode, len(f.Accessibility))
	copy(result, f.Accessibility)
	return result
}

// RegisterModal, bu karede çizilen aktif bir modal katmanı kaydeder.
// Sadece tek bir aktif modal desteklenir ve en son kaydedilen (en üstteki) modal geçerli olur.
// Geriye dönük uyumluluk: Eski API'yi korumak için ActiveModal'a da yazar.
func (f *Frame) RegisterModal(id string, area cell.Rect, onClickOutside func()) {
	f.ActiveModal = &Modal{
		ID:           id,
		Area:         area,
		ClickOutside: onClickOutside,
	}
	// Yeni katman sistemine de ekle
	f.Layers = append(f.Layers, Layer{
		ID:           id,
		Type:         LayerModal,
		Area:         area,
		ClickOutside: onClickOutside,
		ZIndex:       1000,
	})
}

// RegisterLayer, yeni katmanlı render sistemi için bir katman kaydeder.
// ZIndex değeri büyüdükçe katman üst üste biner. Çizim sırasına göre son katman en üsttedir.
func (f *Frame) RegisterLayer(id string, layerType LayerType, area cell.Rect, zIndex int, onClickOutside func()) {
	layer := Layer{
		ID:           id,
		Type:         layerType,
		Area:         area,
		ClickOutside: onClickOutside,
		ZIndex:       zIndex,
	}
	f.Layers = append(f.Layers, layer)

	// Geriye dönük uyumluluk: Modal türündeyse ActiveModal'ı da güncelle
	if layerType == LayerModal {
		f.ActiveModal = &Modal{
			ID:           id,
			Area:         area,
			ClickOutside: onClickOutside,
		}
	}
}

// RemoveLayer, belirtilen ID'ye sahip katmanı listeden kaldırır.
func (f *Frame) RemoveLayer(id string) {
	for i := 0; i < len(f.Layers); i++ {
		if f.Layers[i].ID == id {
			f.Layers = append(f.Layers[:i], f.Layers[i+1:]...)
			i--
		}
	}
	// ActiveModal güncelleme: Sadece modal türündeki katmanlar için
	if f.ActiveModal != nil {
		found := false
		for _, l := range f.Layers {
			if l.ID == f.ActiveModal.ID && l.Type == LayerModal {
				found = true
				break
			}
		}
		if !found {
			f.ActiveModal = nil
		}
	}
}

// TopLayer, en yüksek z-index değerine sahip katmanı döndürür.
// Hiç katman yoksa nil döner.
func (f *Frame) TopLayer() *Layer {
	if len(f.Layers) == 0 {
		return nil
	}
	top := &f.Layers[0]
	for i := 1; i < len(f.Layers); i++ {
		if f.Layers[i].ZIndex > top.ZIndex {
			top = &f.Layers[i]
		}
	}
	return top
}

// TopmostModal, aktif katmanlar arasında en üstteki (en yüksek ZIndex'e sahip) modal katmanı döner.
func (f *Frame) TopmostModal() *Layer {
	var top *Layer
	for i := range f.Layers {
		l := &f.Layers[i]
		if l.Type == LayerModal {
			if top == nil || l.ZIndex >= top.ZIndex {
				top = l
			}
		}
	}
	return top
}

// IsInsideAnyLayer, verilen koordinatın herhangi bir katman alanı içinde olup olmadığını kontrol eder.
func (f *Frame) IsInsideAnyLayer(x, y uint16) bool {
	for i := range f.Layers {
		if f.Layers[i].Area.Contains(x, y) {
			return true
		}
	}
	return false
}

// RegisterClickHandler, belirtilen alan (rect) üzerine fare tıklaması yapıldığında
// çalıştırılacak bir callback kaydeder. Otomatik fare yönlendirme sistemi (Mouse Event Router) bu kaydı kullanır.
// layerID parametresi, bu tıklama bölgesinin hangi katmana ait olduğunu belirtir.
func (f *Frame) RegisterClickHandler(area cell.Rect, handler func(ev backend.MouseEvent)) {
	if handler == nil {
		return
	}
	f.ClickRegions = append(f.ClickRegions, ClickRegion{
		Area:      area,
		Handler:   handler,
		LayerID:   f.activeLayerID,
		MouseOnly: false,
	})
}

func (f *Frame) registerMouseHandler(area cell.Rect, handler func(ev backend.MouseEvent), layerID string) {
	if handler == nil {
		return
	}
	f.ClickRegions = append(f.ClickRegions, ClickRegion{
		Area:      area,
		Handler:   handler,
		LayerID:   layerID,
		MouseOnly: true,
	})
}

// RegisterEventHandler registers an opt-in capture/target/bubble handler.
func (f *Frame) RegisterEventHandler(area cell.Rect, phase EventPhase, handler func(*EventContext)) {
	f.RegisterEventRegion(EventRegion{Area: area, Phase: phase, Handler: handler})
}

// RegisterEventRegion registers a metadata-rich event region.
func (f *Frame) RegisterEventRegion(region EventRegion) {
	if f == nil || region.Handler == nil {
		return
	}
	f.EventRegions = append(f.EventRegions, eventRegion{
		Area: region.Area, ID: region.ID, LayerID: region.LayerID,
		ZIndex: region.ZIndex, Disabled: region.Disabled,
		Phase: region.Phase, Handler: region.Handler,
		OnEnter: region.OnEnter, OnLeave: region.OnLeave,
	})
}

// HoveredRegionID returns the ID of the target region currently under the
// pointer. It is empty when no registered target region is hovered.
func (f *Frame) HoveredRegionID() string {
	if f == nil {
		return ""
	}
	return f.hoveredRegionID
}

// DispatchPointerMove updates hover state and invokes enter/leave callbacks.
func (f *Frame) DispatchPointerMove(ev backend.MouseEvent) bool {
	if f == nil {
		return false
	}
	var target *eventRegion
	for i := len(f.EventRegions) - 1; i >= 0; i-- {
		region := &f.EventRegions[i]
		if region.Phase == TargetPhase && !region.Disabled && region.Area.Contains(ev.X, ev.Y) {
			target = region
			break
		}
	}
	newID := ""
	if target != nil {
		newID = target.ID
	}
	if newID == f.hoveredRegionID {
		return target != nil
	}
	if f.hoveredRegionID != "" {
		for i := range f.EventRegions {
			region := &f.EventRegions[i]
			if region.ID == f.hoveredRegionID && region.OnLeave != nil {
				ctx := &backend.EventContext{
					Mouse: ev, Phase: TargetPhase, RegionID: region.ID,
					LayerID: region.LayerID, ZIndex: region.ZIndex,
					PointerKind: backend.PointerLeave,
				}
				region.OnLeave(ctx)
			}
		}
	}
	if target != nil && target.OnEnter != nil {
		ctx := &backend.EventContext{
			Mouse: ev, Phase: TargetPhase, RegionID: target.ID,
			LayerID: target.LayerID, ZIndex: target.ZIndex,
			PointerKind: backend.PointerEnter,
		}
		target.OnEnter(ctx)
	}
	f.hoveredRegionID = newID
	return target != nil
}

// DispatchEventRegions dispatches a mouse event through registered capture,
// target, and bubble handlers. It is useful for deterministic event tests and
// custom event loops that do not use Terminal.RouteMouseEvent.
func (f *Frame) DispatchEventRegions(ev backend.MouseEvent) bool {
	if f == nil {
		return false
	}
	ctx := &backend.EventContext{Mouse: ev}
	handled := false
	for _, phase := range []backend.EventPhase{backend.CapturePhase, backend.TargetPhase, backend.BubblePhase} {
		ctx.Phase = phase
		if phase == backend.TargetPhase {
			for i := len(f.EventRegions) - 1; i >= 0; i-- {
				region := f.EventRegions[i]
				if region.Disabled {
					continue
				}
				if region.Phase == phase && region.Area.Contains(ev.X, ev.Y) {
					handled = true
					ctx.RegionID, ctx.LayerID, ctx.ZIndex = region.ID, region.LayerID, region.ZIndex
					region.Handler(ctx)
					break
				}
			}
		} else {
			for i := 0; i < len(f.EventRegions); i++ {
				region := f.EventRegions[i]
				if region.Disabled {
					continue
				}
				if region.Phase == phase && region.Area.Contains(ev.X, ev.Y) {
					handled = true
					ctx.RegionID, ctx.LayerID, ctx.ZIndex = region.ID, region.LayerID, region.ZIndex
					region.Handler(ctx)
					if ctx.IsPropagationStopped() {
						return true
					}
				}
			}
		}
		if ctx.IsPropagationStopped() {
			return true
		}
	}
	return handled || ctx.IsDefaultPrevented()
}

// CaptureMouse, aktif farenin sürükleme boyunca kayıtlı handler'a yönlendirilmesini sağlar.
// Handler, MouseRelease olayını aldıktan sonra yakalama otomatik olarak bırakılır.
func (f *Frame) CaptureMouse(handler func(ev backend.MouseEvent)) {
	if handler != nil {
		f.mouseCaptureRequest = handler
	}
}

// TakeMouseCapture returns and clears a mouse capture requested while drawing.
// It is primarily useful to deterministic test harnesses and custom event
// loops that dispatch events without owning a Terminal instance.
func (f *Frame) TakeMouseCapture() func(ev backend.MouseEvent) {
	handler := f.mouseCaptureRequest
	f.mouseCaptureRequest = nil
	return handler
}

// RegisterClickHandlerInLayer, belirtilen katman ID'si altında bir tıklama alanı kaydeder.
func (f *Frame) RegisterClickHandlerInLayer(area cell.Rect, handler func(ev backend.MouseEvent), layerID string) {
	if handler == nil {
		return
	}
	f.ClickRegions = append(f.ClickRegions, ClickRegion{
		Area:    area,
		Handler: handler,
		LayerID: layerID,
	})
}

// RenderWidget, verilen widget'ı varsayılan temiz bir stil bağlamıyla tampon üzerine çizer.
// Çizim işlemi, widget'a ait Draw metodu çağrılarak stil mirası zinciri başlatılarak gerçekleştirilir.
//
// Parametreler:
//   - w: Çizilmek istenen durumsuz Widget.
//   - area: Widget'ın kaplayacağı çizim alanı sınırı.
func (f *Frame) RenderWidget(w widgets.Widget, area cell.Rect) {
	if w == nil {
		return
	}

	// Hata ayıklama bölgesi olarak kaydet. Overlay widget'ları çizim için
	// tam ekran alanı kullanabilir; DebugArea ile gerçek görünür sınırlarını
	// ayrıca bildirebilirler.
	wType := fmt.Sprintf("%T", w)
	if idx := strings.Index(wType, "."); idx != -1 {
		wType = wType[idx+1:]
	}
	debugArea := area
	if provider, ok := w.(interface{ DebugArea(cell.Rect) cell.Rect }); ok {
		debugArea = provider.DebugArea(area)
	}
	zIndex := 0
	if f.activeLayerID != "" {
		for _, l := range f.Layers {
			if l.ID == f.activeLayerID {
				zIndex = l.ZIndex
				break
			}
		}
	} else if f.ActiveModal != nil && ContainsRect(f.ActiveModal.Area, area) {
		zIndex = 10
	}
	f.DebugRegions = append(f.DebugRegions, DebugRegion{
		Area:       debugArea,
		WidgetType: wType,
		ZIndex:     zIndex,
	})

	var defStyle cell.Style
	defStyle.Reset()

	// Temiz stil ve sınırlandırılmış alan ile çizim bağlamı oluştur
	ctx := cell.NewContext(area, defStyle)
	if f.ThemeSet {
		ctx.ThemeStyle = func(role string) cell.Style { return f.Theme.RoleStyle(role) }
	}

	// Katman durumunu belirle: Widget, herhangi bir katmanın içinde mi?
	isInsideLayer := f.activeLayerID != ""
	isOutsideModal := false

	// Z-Index / Modal Stack Sandboxing
	topModal := f.TopmostModal()
	if topModal != nil {
		allowed := false
		if ContainsRect(topModal.Area, area) {
			allowed = true
		} else if isInsideLayer {
			var widgetLayerZIndex int
			for _, l := range f.Layers {
				if l.ID == f.activeLayerID {
					widgetLayerZIndex = l.ZIndex
					break
				}
			}
			if widgetLayerZIndex >= topModal.ZIndex {
				allowed = true
			}
		}
		if !allowed {
			isOutsideModal = true
		}
	} else if f.ActiveModal != nil && !ContainsRect(f.ActiveModal.Area, area) {
		// Eski modal sistemi ile geriye dönük uyumluluk
		isOutsideModal = true
	} else if len(f.Layers) > 0 && f.ActiveModal == nil {
		// Kök katmanda çizilen widget'lar, katmanlar varken engellenmeli.
		isOutsideModal = true
	}

	// Tıklama kaydını Frame'in RegisterClickHandler metoduna köprüle
	layerID := f.activeLayerID
	ctx.RegisterClick = func(clickArea cell.Rect, handler func()) {
		if isOutsideModal {
			return // Dışarıdaki tıklamaları yut!
		}
		f.RegisterClickHandlerInLayer(clickArea, func(ev backend.MouseEvent) {
			handler()
		}, layerID)
	}

	ctx.RegisterMouse = func(mouseArea cell.Rect, handler func(ev backend.MouseEvent)) {
		if isOutsideModal {
			return
		}
		f.registerMouseHandler(mouseArea, handler, layerID)
	}
	ctx.RegisterEvent = func(eventArea cell.Rect, phase backend.EventPhase, handler func(*backend.EventContext)) {
		if isOutsideModal {
			return
		}
		f.RegisterEventHandler(eventArea, phase, handler)
	}

	ctx.CaptureMouse = func(handler func(ev backend.MouseEvent)) {
		if isOutsideModal {
			return
		}
		f.mouseCaptureRequest = handler
	}

	// Resim kaydını Frame'in ImageRegions listesine köprüle
	// Not: Resimler pasif çizim elemanlarıdır, olay almazlar.
	// Bu yüzden modal/popup dışında olsalar bile kaydedilmelidirler;
	// aksi halde arka plandaki resimler modal açıldığında kaybolur.
	ctx.RegisterImage = func(imageArea cell.Rect, img image.Image, zIndex int, transparent bool) bool {
		// Z-Index otomatik eşleme mantığı (WezTerm/Ghostty vb. katman çakışmalarını önlemek için):
		// - ZIndex = -99 ise: Block arka plan resmi. Modal/katman için -2, kök için -4 olur.
		// - ZIndex <= 0 ise: Normal görsel. Modal/katman içindeyse -1, arka plandaysa -3 olur.
		topModal := f.TopmostModal()
		if zIndex == -99 {
			if topModal != nil || len(f.Layers) > 0 {
				zIndex = -2
			} else {
				zIndex = -4
			}
		} else if zIndex <= 0 {
			isForeground := false
			if topModal != nil && ContainsRect(topModal.Area, imageArea) {
				isForeground = true
			} else {
				for _, layer := range f.Layers {
					if ContainsRect(layer.Area, imageArea) {
						isForeground = true
						break
					}
				}
			}

			if isForeground {
				zIndex = -1
			} else {
				zIndex = -3
			}
		}

		f.ImageRegions = append(f.ImageRegions, ImageRegion{
			Area:        imageArea,
			Img:         img,
			ZIndex:      zIndex,
			Transparent: transparent,
		})
		return true
	}

	// Odaklanma kaydını FocusManager'a köprüle
	if f.FocusManager != nil {
		ctx.FocusedID = f.FocusManager.Focused()
		ctx.RegisterFocus = func(id string) {
			if isOutsideModal {
				return // Dışarıdaki odaklanma isteklerini engelle!
			}
			f.FocusManager.Register(id)
			f.FocusManager.RegisterBounds(id, area)
		}
		ctx.SetFocus = func(id string) {
			if isOutsideModal {
				return
			}
			f.FocusManager.SetFocused(id)
		}
	}

	t0 := time.Now()
	w.Draw(ctx, f.Buffer)
	if provider, ok := w.(accessibility.Provider); ok {
		node := provider.AccessibilityNode(area, false)
		if f.FocusManager != nil && f.FocusManager.IsFocused(node.ID) {
			node.State |= accessibility.StateFocused
		}
		f.RegisterAccessibility(node)
	}
	dur := time.Since(t0)

	f.WidgetStats = append(f.WidgetStats, WidgetStat{
		Type:     wType,
		Duration: dur,
	})
}

// BeginLayer, bir sonraki çizilecek widget'ların belirli bir katmana ait olduğunu bildirir.
// Widget'lar Draw() sırasında hangi katmana ait olduklarını bu şekilde öğrenir.
func (f *Frame) BeginLayer(id string) {
	f.activeLayerID = id
}

// EndLayer, aktif katman çizimini sonlandırır ve kök katmana geri döner.
func (f *Frame) EndLayer() {
	f.activeLayerID = ""
}
