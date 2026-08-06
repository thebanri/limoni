package layout

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestFlexLayoutHorizontalRatios(t *testing.T) {
	// 100 genişlikli alan, yatayda 1:2:2 oranında bölünür
	area := cell.NewRect(0, 0, 100, 10)
	lay := NewFlexLayout(Horizontal, 0, Ratio(1), Ratio(2), Ratio(2))

	rects := lay.Split(area)
	if len(rects) != 3 {
		t.Fatalf("Bölünen parça sayısı hatalı. Beklenen: 3, Alınan: %d", len(rects))
	}

	// Oranlar: 1/5 (%20), 2/5 (%40), 2/5 (%40)
	// Beklenen genişlikler: 20, 40, 40
	expectedWidths := []uint16{20, 40, 40}
	expectedX := []uint16{0, 20, 60}

	for i, r := range rects {
		if r.Width != expectedWidths[i] {
			t.Errorf("İndeks %d genişliği hatalı. Beklenen: %d, Alınan: %d", i, expectedWidths[i], r.Width)
		}
		if r.X != expectedX[i] {
			t.Errorf("İndeks %d X konumu hatalı. Beklenen: %d, Alınan: %d", i, expectedX[i], r.X)
		}
		if r.Height != 10 {
			t.Errorf("Yükseklik değişmemeliydi")
		}
	}
}

func TestFlexLayoutVerticalMixed(t *testing.T) {
	// 50 yükseklikli alan, dikeyde Fixed(10), Percentage(20) (%20 of 50 = 10) ve Fill() (geriye kalan: 30)
	area := cell.NewRect(0, 0, 80, 50)
	lay := NewFlexLayout(Vertical, 0, Fixed(10), Percentage(20), Fill())

	rects := lay.Split(area)
	if len(rects) != 3 {
		t.Fatalf("Bölünen parça sayısı hatalı")
	}

	expectedHeights := []uint16{10, 10, 30}
	expectedY := []uint16{0, 10, 20}

	for i, r := range rects {
		if r.Height != expectedHeights[i] {
			t.Errorf("İndeks %d yüksekliği hatalı. Beklenen: %d, Alınan: %d", i, expectedHeights[i], r.Height)
		}
		if r.Y != expectedY[i] {
			t.Errorf("İndeks %d Y konumu hatalı. Beklenen: %d, Alınan: %d", i, expectedY[i], r.Y)
		}
		if r.Width != 80 {
			t.Errorf("Genişlik değişmemeliydi")
		}
	}
}

func TestFlexLayoutGap(t *testing.T) {
	// 20 genişlikli alan, yatayda 3 adet Fixed(5) ve aralarında 2 hücre boşluk (gap = 2) ile bölünür.
	// Toplam gap: 2 * 2 = 4. Kullanılabilir net alan: 20 - 4 = 16.
	// Sabit alanlar: 5 + 5 + 5 = 15. Kalan 1 hücre orantısal paylaştırılmaz çünkü hepsi sabit.
	area := cell.NewRect(0, 0, 20, 5)
	lay := NewFlexLayout(Horizontal, 2, Fixed(5), Fixed(5), Fixed(5))

	rects := lay.Split(area)

	expectedX := []uint16{0, 7, 14} // (0 + 5 + 2 = 7), (7 + 5 + 2 = 14)
	for i, r := range rects {
		if r.Width != 5 {
			t.Errorf("Genişlik 5 olmalıydı, alınan: %d", r.Width)
		}
		if r.X != expectedX[i] {
			t.Errorf("İndeks %d X konumu hatalı. Beklenen: %d, Alınan: %d", i, expectedX[i], r.X)
		}
	}
}

func TestFlexLayoutExceededScaling(t *testing.T) {
	// 10 genişlikli küçük bir alana toplamı 20 olan kısıtlamalar uyguluyoruz.
	// Oranlanarak küçültülmeli ve alan aşılmamalıdır.
	area := cell.NewRect(0, 0, 10, 5)
	lay := NewFlexLayout(Horizontal, 0, Fixed(10), Fixed(10)) // Toplam 20, alan 10

	rects := lay.Split(area)
	// Beklenen: Her biri 5 genişliğe düşürülmeli
	if rects[0].Width != 5 || rects[1].Width != 5 {
		t.Errorf("Aşım oranlaması başarısız. Alınan genişlikler: %d ve %d", rects[0].Width, rects[1].Width)
	}
}
