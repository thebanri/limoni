package terminal

import (
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
	// Handler, bu alana tıklandığında tetiklenecek olan olay yöneticisi fonksiyondur.
	Handler func(ev backend.MouseEvent)
}

// Frame, tek bir çizim karesinin (render pass) bağlamını temsil eder.
// Çizim işlemi sırasında hem ham tampona yazmayı hem de interaktif tıklama bölgelerini kaydetmeyi yönetir.
type Frame struct {
	// Buffer, bu karede üzerine çizim yapılan aktif terminal hücre matrisidir.
	Buffer *buffer.Buffer

	// ClickRegions, bu karede widget'lar tarafından kaydedilen tıklanabilir bölgeler listesidir.
	ClickRegions []ClickRegion
}

// NewFrame, belirtilen buffer üzerinde çizim yapacak yeni bir Frame örneği oluşturur.
func NewFrame(buf *buffer.Buffer) *Frame {
	return &Frame{
		Buffer:       buf,
		ClickRegions: make([]ClickRegion, 0, 32),
	}
}

// Reset, çizim karesinin durumunu (kaydedilmiş tıklama alanlarını) sıfırlar.
// Bellek Optimizasyonu: Slice kapasitesini koruyarak sıfır tahsisatla listeyi temizler (slice[:0]).
func (f *Frame) Reset() {
	f.ClickRegions = f.ClickRegions[:0]
}

// RegisterClickHandler, belirtilen alan (rect) üzerine fare tıklaması yapıldığında
// çalıştırılacak bir callback kaydeder. Otomatik fare yönlendirme sistemi (Mouse Event Router) bu kaydı kullanır.
//
// Parametreler:
//   - area: Tıklanabilir ekran bölgesi.
//   - handler: Fare tıklama olayı geldiğinde tetiklenecek fonksiyon.
func (f *Frame) RegisterClickHandler(area cell.Rect, handler func(ev backend.MouseEvent)) {
	if handler == nil {
		return
	}
	f.ClickRegions = append(f.ClickRegions, ClickRegion{
		Area:    area,
		Handler: handler,
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
	var defStyle cell.Style
	defStyle.Reset()

	// Temiz stil ve sınırlandırılmış alan ile çizim bağlamı oluştur
	ctx := cell.NewContext(area, defStyle)

	// Tıklama kaydını Frame'in RegisterClickHandler metoduna köprüle
	ctx.RegisterClick = func(clickArea cell.Rect, handler func()) {
		f.RegisterClickHandler(clickArea, func(ev backend.MouseEvent) {
			handler()
		})
	}

	w.Draw(ctx, f.Buffer)
}
