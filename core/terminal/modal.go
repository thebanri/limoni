package terminal

import "github.com/thebanri/limoni/core/cell"

// Modal, katmanlı çizimde (layered rendering) üstte duran ve olayları filtreleyen modal pencere bilgisini saklar.
type Modal struct {
	ID           string
	Area         cell.Rect
	ClickOutside func()
}

// LayerType, katman türünü tanımlar.
type LayerType uint8

const (
	// LayerModal, odak kilitli (focus-trapped) modal pencere katmanıdır.
	LayerModal LayerType = iota
	// LayerPopup, açılır menü (dropdown) gibi hafif katman türüdür. Odak kilidi yoktur.
	LayerPopup
)

// Layer, katmanlı render sisteminde tek bir üst üste binen çizim katmanını temsil eder.
// Her katman, kendi alanı içindeki olayları filtreler ve alt katmanlara sızmasını engeller.
// Z-Index değeri büyüdükçe katman üst üste biner (en büyük z-index en üstte).
type Layer struct {
	// ID, katmanın benzersiz tanımlayıcısıdır.
	ID string
	// Type, katmanın türünü (Modal veya Popup) belirtir.
	Type LayerType
	// Area, katmanın kapladığı ekran alanıdır.
	Area cell.Rect
	// ClickOutside, modal alanı dışına tıklandığında tetiklenen callback fonksiyonudur.
	ClickOutside func()
	// ZIndex, katmanın çizim sırasını belirler. Büyük değer = üstte.
	ZIndex int
}

// ContainsRect, child alanının tamamen parent alanı içinde olup olmadığını sorgular.
func ContainsRect(parent, child cell.Rect) bool {
	return child.X >= parent.X &&
		child.Y >= parent.Y &&
		int(child.X)+int(child.Width) <= int(parent.X)+int(parent.Width) &&
		int(child.Y)+int(child.Height) <= int(parent.Y)+int(parent.Height)
}

// Intersects, iki dikdörtgenin kesişip kesişmediğini denetler.
func Intersects(r1, r2 cell.Rect) bool {
	return r1.X < r2.X+r2.Width &&
		r2.X < r1.X+r1.Width &&
		r1.Y < r2.Y+r2.Height &&
		r2.Y < r1.Y+r1.Height
}

// CenterRect, belirtilen genişlik ve yükseklikte, parent alanının tam ortasında konumlanmış bir dikdörtgen hesaplar.
func CenterRect(parent cell.Rect, w, h uint16) cell.Rect {
	if w > parent.Width {
		w = parent.Width
	}
	if h > parent.Height {
		h = parent.Height
	}

	x := parent.X + (parent.Width-w)/2
	y := parent.Y + (parent.Height-h)/2

	return cell.NewRect(x, y, w, h)
}

// ScaleRect, bir dikdörtgen alanını belirtilen ilerleme yüzdesine (0.0 -> 1.0) göre
// merkez noktasını koruyarak yeniden ölçeklendirir.
func ScaleRect(base cell.Rect, progress float64) cell.Rect {
	if progress <= 0 {
		return cell.NewRect(base.X+base.Width/2, base.Y+base.Height/2, 0, 0)
	}
	if progress >= 1.0 {
		return base
	}

	w := uint16(float64(base.Width) * progress)
	h := uint16(float64(base.Height) * progress)

	// Jitter (titreme) ve sub-pixel hizalama kaymalarını önlemek için w ve h değerlerini çifte yuvarla
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

// SlideUpRect, bir dikdörtgeni alt kenardan (ekranın dışından) hedef dikey koordinata (Y) doğru
// belirtilen ilerleme yüzdesine (0.0 -> 1.0) göre pürüzsüzce kaydırır.
func SlideUpRect(base cell.Rect, parentHeight uint16, progress float64) cell.Rect {
	if progress <= 0 {
		return cell.NewRect(base.X, parentHeight, base.Width, base.Height)
	}
	if progress >= 1.0 {
		return base
	}

	startY := parentHeight
	y := startY - uint16(float64(startY-base.Y)*progress)

	return cell.NewRect(base.X, y, base.Width, base.Height)
}
