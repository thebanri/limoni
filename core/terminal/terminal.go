//go:build linux

package terminal

import (
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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

	return &Terminal{
		backend:  b,
		front:    front,
		back:     back,
		frame:    NewFrame(front),
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

	// Eğer pencere boyutu değiştiyse tamponları sıfır tahsisatla yeniden boyutlandır
	if w != t.front.Area.Width || h != t.front.Area.Height {
		t.front.Resize(cell.NewRect(0, 0, w, h))
		t.back.Resize(cell.NewRect(0, 0, w, h))
	}

	// Aktif çizim tamponunu temizle
	t.front.Clear()
	// Tıklama bölgeleri kaydını sıfırla
	t.frame.Reset()

	// Geliştiriciye çizim karesi bağlamını (Frame) sunarak bileşenleri çizdir
	if fn != nil {
		fn(t.frame)
	}

	// Modern terminallerde yırtılmayı engellemek için senkron güncellemeyi başlat
	t.backend.StartSyncUpdate()

	// Diff çıktısını yazma tamponuna ekle
	t.writeBuf = t.writeBuf[:0]
	var diffErr error
	t.writeBuf, diffErr = buffer.Diff(t.front, t.back, t.writeBuf)
	if diffErr != nil {
		t.backend.EndSyncUpdate()
		return diffErr
	}

	// Sadece değişiklik varsa terminale yaz
	if len(t.writeBuf) > 0 {
		if _, err := t.backend.Write(t.writeBuf); err != nil {
			t.backend.EndSyncUpdate()
			return err
		}
	}

	// Senkron ekran güncellemesini sonlandır (veriyi terminale bas)
	t.backend.EndSyncUpdate()

	return nil
}

// RouteMouseEvent, terminalden gelen bir fare tıklama/sürükleme/tekerlek olayını,
// en son çizilen karedeki kayıtlı tıklama bölgeleriyle karşılaştırarak ilgili callback'e yönlendirir.
// Olay bir bölgeyle eşleşip tetiklendiyse `true`, eşleşmediyse `false` döner.
func (t *Terminal) RouteMouseEvent(ev backend.MouseEvent) bool {
	// Kayıtlı tıklama bölgelerini tersten gezerek (en son çizilen/en üstte duran önceliklidir) kontrol et
	for i := len(t.frame.ClickRegions) - 1; i >= 0; i-- {
		reg := t.frame.ClickRegions[i]
		if reg.Area.Contains(ev.X, ev.Y) {
			reg.Handler(ev)
			return true
		}
	}
	return false
}
