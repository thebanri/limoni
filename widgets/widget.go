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

// DrawFocusRing, belirtilen alanın etrafına 1 hücrelik kalın kesikli odak halkası (parlak bir çerçeve) çizer.
func DrawFocusRing(buf *buffer.Buffer, area cell.Rect, style cell.Style) {
	if area.Width < 2 || area.Height < 2 {
		return
	}
	// Yatay kesikli çizgiler
	for col := area.X + 1; col < area.X+area.Width-1; col++ {
		if c := buf.Get(col, area.Y); c != nil {
			c.Content = '╍'
			c.Style = style
		}
		if c := buf.Get(col, area.Y+area.Height-1); c != nil {
			c.Content = '╍'
			c.Style = style
		}
	}
	// Dikey kesikli çizgiler
	for row := area.Y + 1; row < area.Y+area.Height-1; row++ {
		if c := buf.Get(area.X, row); c != nil {
			c.Content = '╏'
			c.Style = style
		}
		if c := buf.Get(area.X+area.Width-1, row); c != nil {
			c.Content = '╏'
			c.Style = style
		}
	}
	// Köşe birleşimleri (kalın köşeler)
	if c := buf.Get(area.X, area.Y); c != nil {
		c.Content = '┏'
		c.Style = style
	}
	if c := buf.Get(area.X+area.Width-1, area.Y); c != nil {
		c.Content = '┓'
		c.Style = style
	}
	if c := buf.Get(area.X, area.Y+area.Height-1); c != nil {
		c.Content = '┗'
		c.Style = style
	}
	if c := buf.Get(area.X+area.Width-1, area.Y+area.Height-1); c != nil {
		c.Content = '┛'
		c.Style = style
	}
}
