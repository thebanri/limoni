//go:build linux

package terminal

import (
	"fmt"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// Terminal, TUI motorunun ana kontrolcüsüdür.
// Çift tampon yönetimini (Front/Back Buffer), ekran boyutu değişikliklerini,
// senkron ekran yenileme protokolünü (?2026) ve fare olaylarının doğru hedeflere yönlendirilmesini koordine eder.
type Terminal struct {
	// backend, düşük seviyeli TTY Raw Mode ve I/O işlemlerini yöneten katmandır.
	backend *backend.Backend

	// front, mevcut çizim karesinde üzerine yazılan aktif tampondur.
	front *buffer.Buffer

	// back, ekranda o an çizili olan hücreleri tutan yedek tampondur (diff alma amacıyla kullanılır).
	back *buffer.Buffer

	// frame, çizim döngüsü sırasında widget'lara sunulan çizim ve tıklama alanı kayıt bağlamıdır.
	frame *Frame

	// writeBuf, diff çıktısı olan ANSI kaçış kodlarının heap allocation yapmadan yazılması için
	// her karede yeniden kullanılan byte dilimi tamponudur.
	writeBuf []byte

	// lastImageCount, bir önceki render karesinde çizilen resim sayısını saklar.
	lastImageCount int

	// lastDrawnImages, bir önceki render karesinde çizilen resimlerin listesini saklar.
	lastDrawnImages []ImageRegion

	// Dither geçiş durumları
	transitionActive   bool
	transitionProgress float64
	transitionOldBuf   *buffer.Buffer

	// Hata ayıklama (Debug / Layout Inspector) durumu
	debugMode bool

	// mouseCaptureHandler, o an aktif olan fare sürükleme (capture) olay yöneticisidir.
	mouseCaptureHandler func(ev backend.MouseEvent)

	// lastLayersHash, bir önceki karedeki katmanların (modal/layers) durum özetidir.
	lastLayersHash string
}

// New, belirtilen Backend'i kullanarak yeni bir Terminal yöneticisi oluşturur ve ilk tamponları tahsis eder.
func New(b *backend.Backend) (*Terminal, error) {
	// Terminalin başlangıç satır ve sütun boyutunu al
	w, h, err := b.Size()
	if err != nil {
		return nil, err
	}

	area := cell.NewRect(0, 0, w, h)
	front := buffer.NewBuffer(area)
	back := buffer.NewBuffer(area)

	focusMgr := NewFocusManager()

	return &Terminal{
		backend:  b,
		front:    front,
		back:     back,
		frame:    NewFrame(front, focusMgr),
		writeBuf: make([]byte, 0, 8192), // Başlangıçta 8 KB'lık yazma tamponu tahsis et
	}, nil
}

// Draw, çizim döngüsünü başlatır. Boyut değişimlerini algılar, güncel tamponu temizler,
// çizim callback fonksiyonunu (fn) çalıştırır, diff hesaplamasını yapar ve tek bir senkron I/O çağrısıyla
// değişen kısımları terminale yazar.
//
// Performans: Sıfır-Tahsisat (Zero-Allocation) tasarımı sayesinde bu fonksiyon düzenli çalışmada heap bellek harcamaz.
func (t *Terminal) Draw(fn func(f *Frame)) error {
	// Güncel ekran boyutunu sorgula
	w, h, err := t.backend.Size()
	if err != nil {
		return err
	}

	// Eğer pencere boyutu değiştiyse sadece front tamponunu yeniden boyutlandır.
	// back tamponunun boyutu değiştirilmez, böylece buffer.Diff boyut değişimini tespit edebilir,
	// ekranı temizleyebilir ve tüm kareyi temizlenmiş ekrana yeniden basabilir.
	if w != t.front.Area.Width || h != t.front.Area.Height {
		t.front.Resize(cell.NewRect(0, 0, w, h))
	}

	// Aktif çizim tamponunu temizle
	t.front.Clear()
	// Tıklama bölgeleri kaydını sıfırla
	t.frame.Reset()
	if t.frame.FocusManager != nil {
		t.frame.FocusManager.Clear()
	}

	// Geliştiriciye çizim karesi bağlamını (Frame) sunarak bileşenleri çizdir
	if fn != nil {
		fn(t.frame)
	}

	// Eğer dither geçişi aktifse, önce görüntü tamponunu harmanla.
	// Debug HUD bundan sonra çizilir; böylece debug çizgileri ve etiketleri
	// geçiş efekti tarafından soluklaştırılmaz veya bozulmaz.
	if t.transitionActive && t.transitionOldBuf != nil {
		animation.ApplyDitherFade(t.front, t.transitionOldBuf, t.transitionProgress)
	}

	// Hata ayıklama modu aktifse, geçişin üzerine yerleşim sınırlarını çiz.
	if t.debugMode {
		// Tüm widget'ların debug bölgelerini göster. Bölgeler kendi z-index ve
		// çizim sıralarına göre birbirini örter; hiçbir widget hariç tutulmaz.
		t.drawDebugOverlay()
	}

	// Katman veya modal yapısının değiştiğini tespit et
	currentLayersHash := t.layersHash()
	layersChanged := (currentLayersHash != t.lastLayersHash)
	t.lastLayersHash = currentLayersHash

	// ── 1. ADIM: Kitty/Sixel resimlerini ÖNCE çiz (en arka piksel katmanı) ──
	proto := graphics.DetectProtocol()
	if len(t.frame.ImageRegions) > 0 {
		cellW, cellH, _ := t.backend.CellPixelSize()

		imagesChanged := layersChanged
		if !imagesChanged {
			if len(t.frame.ImageRegions) != len(t.lastDrawnImages) {
				imagesChanged = true
			} else {
				for i, reg := range t.frame.ImageRegions {
					prev := t.lastDrawnImages[i]
					if reg.Img != prev.Img || reg.Area != prev.Area || reg.ZIndex != prev.ZIndex {
						imagesChanged = true
						break
					}
				}
			}
		}

		if imagesChanged {
			if proto == graphics.ProtocolKitty {
				t.backend.Write([]byte("\x1b_Ga=d,d=A\x1b\\"))
			}

			for _, reg := range t.frame.ImageRegions {
				escSeq := graphics.GetCachedEscapeSequence(reg.Img, reg.Area.Width, reg.Area.Height, cellW, cellH, proto, reg.ZIndex)
				if escSeq != "" {
					moveCursor := fmt.Sprintf("\x1b[%d;%dH", reg.Area.Y+1, reg.Area.X+1)
					t.backend.Write([]byte(moveCursor + escSeq))
				}
			}

			t.lastDrawnImages = make([]ImageRegion, len(t.frame.ImageRegions))
			copy(t.lastDrawnImages, t.frame.ImageRegions)
		}
		t.lastImageCount = len(t.frame.ImageRegions)
	} else {
		if t.lastImageCount > 0 {
			if proto == graphics.ProtocolKitty {
				t.backend.Write([]byte("\x1b_Ga=d,d=A\x1b\\"))
			}
			t.lastImageCount = 0
			t.lastDrawnImages = nil
		}
	}

	// ── 2. ADIM: ASCII buffer'ı SYNC ile çiz (piksel katmanının ÜZERİNE) ──
	// Bu, dialog/modal gibi ASCII widget'ların resmin önünde görünmesini sağlar.
	t.backend.StartSyncUpdate()
	t.writeBuf = t.writeBuf[:0]
	var diffErr error
	t.writeBuf, diffErr = buffer.Diff(t.front, t.back, t.writeBuf)
	if diffErr != nil {
		t.backend.EndSyncUpdate()
		return diffErr
	}

	if len(t.writeBuf) > 0 {
		if _, err := t.backend.Write(t.writeBuf); err != nil {
			t.backend.EndSyncUpdate()
			return err
		}
	}
	t.backend.EndSyncUpdate()

	return nil
}

// SetTransitionProgress, dither-fade geçiş ilerlemesini (0.0 - 1.0) ayarlar.
func (t *Terminal) SetTransitionProgress(p float64) {
	t.transitionProgress = p
}

// SetTransitionActive, dither-fade geçiş durumunu açar veya kapatır.
func (t *Terminal) SetTransitionActive(active bool) {
	if !active {
		// Geçişi yalnızca pasifleştirme; eski frame'i de bırak.
		// Aksi halde modal veya sonraki frame eski görüntüyü yeniden kullanabilir.
		t.transitionActive = false
		t.transitionProgress = 1.0
		t.transitionOldBuf = nil
		return
	}
	if t.transitionActive {
		return
	}
	t.transitionActive = true
	w := t.back.Area.Width
	h := t.back.Area.Height
	if t.transitionOldBuf == nil || t.transitionOldBuf.Area.Width != w || t.transitionOldBuf.Area.Height != h {
		t.transitionOldBuf = buffer.NewBuffer(cell.NewRect(0, 0, w, h))
	}
	// back tamponunun içeriğini oldBuf'a kopyala
	if len(t.transitionOldBuf.Content) == len(t.back.Content) {
		copy(t.transitionOldBuf.Content, t.back.Content)
	}
}

// IsTransitionActive, dither-fade geçişinin aktif olup olmadığını döner.
func (t *Terminal) IsTransitionActive() bool {
	return t.transitionActive
}

// RouteMouseEvent, terminalden gelen bir fare tıklama/sürükleme/tekerlek olayını,
// en son çizilen karedeki kayıtlı tıklama bölgeleriyle karşılaştırarak ilgili callback'e yönlendirir.
// Katmanlı render sistemi: En üstteki katmandaki bölgeler önceliklidir.
// Olay bir bölgeyle eşleşip tetiklendiyse `true`, eşleşmediyse `false` döner.
func (t *Terminal) RouteMouseEvent(ev backend.MouseEvent) bool {
	// 0. Fare yakalama (mouse capture) kontrolü
	if t.mouseCaptureHandler != nil {
		t.mouseCaptureHandler(ev)
		if ev.Button == backend.MouseRelease {
			t.mouseCaptureHandler = nil
		}
		return true
	}

	// MouseRelease capture tarafından yukarıda tüketilir. Normal click bölgeleri
	// yalnızca sol tuş basışını, mouse bölgeleri ise hover (MouseNone) olaylarını alır.
	if ev.Button != backend.MouseLeft && ev.Button != backend.MouseNone {
		return false
	}
	if ev.Button == backend.MouseLeft && ev.Drag {
		return false
	}

	// Normal yönlendirme öncesi frame capture isteklerini sıfırla
	t.frame.mouseCaptureRequest = nil

	// 1. Katman sistemi: En üstteki katmandan başlayarak aşağı doğru ara
	if len(t.frame.Layers) > 0 {
		topLayer := t.frame.TopLayer()
		if topLayer != nil {
			if topLayer.Area.Contains(ev.X, ev.Y) {
				// Tıklama en üst katmanın içinde: Sadece o katmanın bölgelerini kontrol et
				for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
					reg := t.frame.ClickRegions[i]
					if reg.LayerID == topLayer.ID && reg.Area.Contains(ev.X, ev.Y) && (ev.Button == backend.MouseNone && reg.MouseOnly || ev.Button == backend.MouseLeft) {
						reg.Handler(ev)
						if t.frame.mouseCaptureRequest != nil {
							t.mouseCaptureHandler = t.frame.mouseCaptureRequest
							t.frame.mouseCaptureRequest = nil
						}
						return true
					}
				}
				// En üst katman içinde ama o katmana ait tıklama alanı yok.
				// Geriye dönük uyumluluk: ActiveModal (eski RegisterModal API'si) varsa onu da dene.
				if t.frame.ActiveModal != nil && t.frame.ActiveModal.ID == topLayer.ID {
					// ActiveModal path'e devam et (aşağıdaki blokta ele alınacak)
				} else {
					return true // Katman içinde ama eşleşen alan yok → olayı yut
				}
			} else {
				// En üst katmanın dışına tıklandı → ClickOutside tetikle (sadece sol tıklama basınçlarında)
				if ev.Button == backend.MouseLeft && !ev.Drag && topLayer.ClickOutside != nil {
					topLayer.ClickOutside()
				}
				return true // Tıklamayı yut
			}
		}
	}

	// 2. Geriye dönük uyumluluk: ActiveModal (eski RegisterModal API'si ile ayarlanmış olabilir)
	if t.frame.ActiveModal != nil {
		modal := t.frame.ActiveModal
		if modal.Area.Contains(ev.X, ev.Y) {
			// Modal içinde: LayerID'si boş olan (kök) veya modal ile aynı ID olan bölgeleri ara
			for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
				reg := t.frame.ClickRegions[i]
				if (reg.LayerID == "" || reg.LayerID == modal.ID) && reg.Area.Contains(ev.X, ev.Y) && (ev.Button == backend.MouseNone && reg.MouseOnly || ev.Button == backend.MouseLeft) {
					reg.Handler(ev)
					if t.frame.mouseCaptureRequest != nil {
						t.mouseCaptureHandler = t.frame.mouseCaptureRequest
						t.frame.mouseCaptureRequest = nil
					}
					return true
				}
			}
			return true // Modal içinde ama boşluğa tıklandı, olayı yut
		} else {
			// Modal dışı tıklama (sadece sol tıklama basınçlarında)
			if ev.Button == backend.MouseLeft && !ev.Drag && modal.ClickOutside != nil {
				modal.ClickOutside()
			}
			return true
		}
	}

	// 3. Normal (katmansız) tıklama yönlendirme döngüsü
	for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
		reg := t.frame.ClickRegions[i]
		if reg.LayerID == "" && reg.Area.Contains(ev.X, ev.Y) && (ev.Button == backend.MouseNone && reg.MouseOnly || ev.Button == backend.MouseLeft) {
			reg.Handler(ev)
			if t.frame.mouseCaptureRequest != nil {
				t.mouseCaptureHandler = t.frame.mouseCaptureRequest
				t.frame.mouseCaptureRequest = nil
			}
			return true
		}
	}
	return false
}

// FocusManager, terminalin odak yöneticisini döndürür.
func (t *Terminal) FocusManager() *FocusManager {
	return t.frame.FocusManager
}

// SetDebugMode, hata ayıklama (Layout Inspector) modunu açar veya kapatır.
func (t *Terminal) SetDebugMode(active bool) {
	t.debugMode = active
}

// DebugMode, hata ayıklama modunun açık olup olmadığını döner.
func (t *Terminal) DebugMode() bool {
	return t.debugMode
}

// drawDebugOverlay, çizilen tüm widget'ların sınırlarını kesikli çizgilerle kaplar
// ve köşelerine widget türünü, boyutlarını ve z-index katmanını belirten etiketler yazar.
// Z-Order Kırpma (Layout Clipping) özelliği sayesinde üstte kalan katmanlar alttakilerin çizgilerini örter.
//
// Debug bölgeleri z-index ve çizim sırasına göre kırpılır. En üstteki
// widget'ın kendi sınırı ve etiketi yine çizilir; hiçbir widget gizlenmez.
func (t *Terminal) drawDebugOverlay() {
	borderStyle := cell.Style{
		Fg: cell.NewColorRGB(255, 0, 255), // Parlak Mor / Magenta
		Bg: cell.NewColorRGB(35, 20, 35),
	}
	textStyle := cell.Style{
		Fg:       cell.NewColorRGB(255, 255, 255),
		Bg:       cell.NewColorRGB(255, 0, 255),
		Modifier: cell.ModifierBold,
	}

	for regionIndex, reg := range t.frame.DebugRegions {
		area := reg.Area
		if area.Width == 0 || area.Height == 0 {
			continue
		}

		// Yatay kesikli çizgiler
		for col := area.X; col < area.X+area.Width; col++ {
			if !isObscured(col, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(col, area.Y); c != nil {
					c.Content = '╌'
					c.Style = borderStyle
				}
			}
			if !isObscured(col, area.Y+area.Height-1, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(col, area.Y+area.Height-1); c != nil {
					c.Content = '╌'
					c.Style = borderStyle
				}
			}
		}
		// Dikey kesikli çizgiler
		for row := area.Y; row < area.Y+area.Height; row++ {
			if !isObscured(area.X, row, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(area.X, row); c != nil {
					c.Content = '╎'
					c.Style = borderStyle
				}
			}
			if !isObscured(area.X+area.Width-1, row, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
				if c := t.front.Get(area.X+area.Width-1, row); c != nil {
					c.Content = '╎'
					c.Style = borderStyle
				}
			}
		}

		// Köşeleri birleştir
		if !isObscured(area.X, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X, area.Y); c != nil {
				c.Content = '┌'
				c.Style = borderStyle
			}
		}
		if !isObscured(area.X+area.Width-1, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X+area.Width-1, area.Y); c != nil {
				c.Content = '┐'
				c.Style = borderStyle
			}
		}
		if !isObscured(area.X, area.Y+area.Height-1, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X, area.Y+area.Height-1); c != nil {
				c.Content = '└'
				c.Style = borderStyle
			}
		}
		if !isObscured(area.X+area.Width-1, area.Y+area.Height-1, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
			if c := t.front.Get(area.X+area.Width-1, area.Y+area.Height-1); c != nil {
				c.Content = '┘'
				c.Style = borderStyle
			}
		}

		// Sol üst köşeye boyut ve tür etiketi bas
		label := fmt.Sprintf(" %s [%dx%d] z:%d ", reg.WidgetType, area.Width, area.Height, reg.ZIndex)
		for idx, r := range label {
			col := area.X + 1 + uint16(idx)
			if col < area.X+area.Width-1 {
				if !isObscured(col, area.Y, reg.ZIndex, regionIndex, t.frame.DebugRegions) {
					if c := t.front.Get(col, area.Y); c != nil {
						c.Content = r
						c.Style = textStyle
					}
				}
			}
		}
	}
}

// isObscured, bir debug bölgesinin hücresinin daha üstteki bir bölge tarafından
// örtülüp örtülmediğini denetler. Aynı z-index'te daha sonra çizilen bölge üsttedir.
func isObscured(x, y uint16, zIndex, regionIndex int, regions []DebugRegion) bool {
	for i := regionIndex + 1; i < len(regions); i++ {
		other := regions[i]
		if other.ZIndex < zIndex {
			continue
		}
		if x >= other.Area.X && x < other.Area.X+other.Area.Width &&
			y >= other.Area.Y && y < other.Area.Y+other.Area.Height {
			return true
		}
	}
	return false
}

// layersHash, mevcut katmanların ve modal pencerelerin konum ve boyut özetini döner.
// Bu özet değiştiğinde resimlerin yeniden çizilmesi zorlanır (grafik kirlenmesini önlemek için).
func (t *Terminal) layersHash() string {
	res := ""
	for _, l := range t.frame.Layers {
		res += fmt.Sprintf("%s:%d,%d,%d,%d;", l.ID, l.Area.X, l.Area.Y, l.Area.Width, l.Area.Height)
	}
	if t.frame.ActiveModal != nil {
		res += fmt.Sprintf("modal:%s:%d,%d,%d,%d;", t.frame.ActiveModal.ID, t.frame.ActiveModal.Area.X, t.frame.ActiveModal.Area.Y, t.frame.ActiveModal.Area.Width, t.frame.ActiveModal.Area.Height)
	}
	return res
}

// ForceFullRedraw zorla tüm ekranın temizlenip baştan çizilmesini sağlar.
func (t *Terminal) ForceFullRedraw() {
	if t.back != nil {
		t.back.Area.Width = 0
	}
}
