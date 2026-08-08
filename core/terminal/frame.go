package terminal

import (
	"fmt"
	"image"
	"strings"

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
}

// Frame, tek bir çizim karesinin (render pass) bağlamını temsil eder.
// Çizim işlemi sırasında hem ham tampona yazmayı hem de interaktif tıklama, resim çizim alanları ve odak yönetimini kaydetmeyi yönetir.
type Frame struct {
	// Buffer, bu karede üzerine çizim yapılan aktif terminal hücre matrisidir.
	Buffer *buffer.Buffer

	// ClickRegions, bu karede widget'lar tarafından kaydedilen tıklanabilir bölgeler listesidir.
	ClickRegions []ClickRegion

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
		ImageRegions: make([]ImageRegion, 0, 8),
		FocusManager: focusMgr,
		ActiveModal:  nil,
		Layers:       make([]Layer, 0, 4),
		DebugRegions: make([]DebugRegion, 0, 32),
	}
}

// Reset, çizim karesinin durumunu (kaydedilmiş tıklama, resim alanları, modal ve katmanları) sıfırlar.
// Bellek Optimizasyonu: Slice kapasitesini koruyarak sıfır tahsisatla listeyi temizler (slice[:0]).
func (f *Frame) Reset() {
	f.ClickRegions = f.ClickRegions[:0]
	f.ImageRegions = f.ImageRegions[:0]
	f.ActiveModal = nil
	f.Layers = f.Layers[:0]
	f.activeLayerID = ""
	f.DebugRegions = f.DebugRegions[:0]
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
		Area:    area,
		Handler: handler,
		LayerID: f.activeLayerID,
	})
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

	// Hata ayıklama bölgesi olarak kaydet
	wType := fmt.Sprintf("%T", w)
	if idx := strings.Index(wType, "."); idx != -1 {
		wType = wType[idx+1:]
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
		Area:       area,
		WidgetType: wType,
		ZIndex:     zIndex,
	})

	var defStyle cell.Style
	defStyle.Reset()

	// Temiz stil ve sınırlandırılmış alan ile çizim bağlamı oluştur
	ctx := cell.NewContext(area, defStyle)

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
		// Eski modal API kullanılmıyorsa ama yeni katman sistemi aktifse:
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

	// Resim kaydını Frame'in ImageRegions listesine köprüle
	// Not: Resimler pasif çizim elemanlarıdır, olay almazlar.
	// Bu yüzden modal/popup dışında olsalar bile kaydedilmelidirler;
	// aksi halde arka plandaki resimler modal açıldığında kaybolur.
	ctx.RegisterImage = func(imageArea cell.Rect, img image.Image, zIndex int) {
		f.ImageRegions = append(f.ImageRegions, ImageRegion{
			Area:   imageArea,
			Img:    img,
			ZIndex: zIndex,
		})
	}

	// Odaklanma kaydını FocusManager'a köprüle
	if f.FocusManager != nil {
		ctx.FocusedID = f.FocusManager.Focused()
		ctx.RegisterFocus = func(id string) {
			if isOutsideModal {
				return // Dışarıdaki odaklanma isteklerini engelle!
			}
			f.FocusManager.Register(id)
		}
		ctx.SetFocus = func(id string) {
			if isOutsideModal {
				return
			}
			f.FocusManager.SetFocused(id)
		}
	}

	w.Draw(ctx, f.Buffer)
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
