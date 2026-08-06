package widgets

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Widget, ekranda kendisini çizebilen durumsuz (stateless) TUI bileşenlerinin uyması gereken temel arayüzdür.
// Limoni kütüphanesindeki tüm görsel bileşenler (Block, Paragraph, Table vb.) bu arayüzü gerçekleştirir.
type Widget interface {
	// Draw, bileşeni kendisine tahsis edilen alan sınırları içerisinde ve miras alınan stil özellikleri
	// doğrultusunda terminal tamponuna (buffer.Buffer) çizer.
	//
	// Parametreler:
	//   - ctx: Üst bileşenden aktarılan stack-allocated çizim bağlamı (alan ve stil mirası).
	//   - buf: Çizimin yapılacağı terminal hücre matrisi tamponu.
	Draw(ctx cell.Context, buf *buffer.Buffer)

	// SizeHint, verilen maksimum alan sınırlarına (maxArea) göre widget'ın tercih ettiği ideal
	// (genişlik, yükseklik) boyutlarını döner. Düzen Pazarlığı (Layout Negotiation) motoru
	// bu değeri kullanarak esnek kutu dağılımı yapar.
	//
	// Parametreler:
	//   - maxArea: Üst bileşenin bu widget için ayırabileceği maksimum alan sınırları.
	SizeHint(maxArea cell.Rect) (width, height uint16)
}
