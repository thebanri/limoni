---
name: limoni_development
description: Guidelines and principles for developing the Limoni TUI library in Go and teaching Go to the user.
---

# Limoni TUI Geliştirme, Mimari ve Handover Kılavuzu

Bu uzmanlık dosyası, **Limoni** projesinin vizyonunu, mimari prensiplerini, mevcut gelişim durumunu, çözdüğü TUI kısıtlamalarını ve gelecek yol haritasını tanımlar. Seyahat sonrası veya yeni bir chat oturumunda projeyi devralacak yapay zekalar (AI Agents) için eksiksiz bir sistem el kitabı niteliğindedir.

---

## 1. Proje Vizyonu ve Çıkış Noktası

Limoni; Rust ekosistemindeki **Ratatui** kütüphanesinin çizim hızı ve anlık render (Immediate Mode) yeteneklerini, Go (Golang) dilinin yerel eşzamanlılık (Goroutine, Channel) gücüyle birleştiren yeni nesil bir **TUI (Terminal User Interface) Motorudur**.

### Rakip Kütüphanelerin Aşılması Hedeflenen Eksiklikleri ve Çözümlerimiz:
1. **Otomatik Fare Tıklaması Yönlendirme (Mouse Event Hit-Testing)**:
   - *Sorun*: Ratatui ve Bubble Tea'de fare tıklamalarının hangi widget'a isabet ettiğini geliştirici elle koordinat hesaplayarak bulur.
   - *Çözüm*: Limoni'de `Frame.RegisterClickHandler(area, callback)` API'si ile çizim anında interaktif bölgeler kaydedilir. `Terminal.RouteMouseEvent(ev)` mekanizması tıklamaları otomatik doğru callback'e yönlendirir.
2. **Paket Döngüsel Bağımlılığı Olmadan İnteraktiflik Köprüsü**:
   - *Sorun*: Alt seviye `cell` paketinin üst seviye `terminal` veya `backend` olaylarına bağımlı olması döngüsel paket bağımlılığı (circular dependency) yaratır.
   - *Çözüm*: `cell.Context` yapısına `RegisterClick func(area Rect, handler func())` fonksiyon imza alanı eklenmiştir. `Frame.RenderWidget` çizimi başlatırken bu köprüyü kendi yönlendiricisine bağlar. Böylece `widgets` paketi bağımsız kalır.
3. **Düzen Pazarlığı (Layout Negotiation)**:
   - *Sorun*: Ratatui'de bir widget kendi boyut ihtiyacını layout'a bildiremez.
   - *Çözüm*: `Widget` interface'ine `SizeHint` API'si eklenmiştir. Bileşenler (örn. `Block`, `Paragraph`, `List`) kendi kenarlık, başlık ve içeriklerine göre esnek yerleşim motoruyla (`layout`) pazarlık yapabilir.
4. **Stil Mirası (Cascading Styles)**:
   - *Sorun*: Ratatui'de her alt bileşene elle stil geçilmesi gerekir.
   - *Çözüm*: `cell.Context` üzerinden üst bileşenin stili otomatik olarak miras kalır. `Style.Merge(other)` ile alt bileşen sadece kendi değiştirmek istediği özellikleri ezer.

---

## 2. Mevcut Mimari Yapı ve Paket Dağılımı

Proje, tek bir Go modülü (`github.com/thebanri/limoni`) içinde modüler paketler halinde tasarlanmıştır:

```
limoni/
├── go.mod
├── .agents/skills/limoni_development/SKILL.md  # Bu el kitabı
├── core/
│   ├── cell/
│   │   ├── cell.go       # Cell (16 byte), Style (12 byte), Color (uint32)
│   │   ├── rect.go       # Ekran alanı geometrisi (Rect)
│   │   └── context.go    # Stack-allocated cascading Context & Merge
│   ├── buffer/
│   │   ├── buffer.go     # Flat 1D slice hücre matrisi (Buffer), Resize
│   │   └── diff.go       # Sıfır tahsisatlı double-buffered diff algoritması
│   ├── backend/
│   │   ├── types.go      # Düz (flat) tipli olay yapıları (Event, KeyEvent vb.)
│   │   ├── termios_linux # Unix ioctl çağrıları ile Pure Go Raw Mode kontrolü
│   │   ├── parser.go     # Klavye/mouse/focus/hover ANSI dizi çözümleyicisi
│   │   └── backend.go    # SIGWINCH ve 25ms ESC timeout asenkron Event Loop
│   └── terminal/
│       ├── frame.go      # Kare çizim bağlamı, Click handler kaydı
│       └── terminal.go   # Terminal yöneticisi, Draw döngüsü, Mouse Router
├── layout/
│   └── layout.go         # Flexbox yerleşim motoru (Fixed, Percentage, Ratio, Min, Max, Fill)
├── widgets/
│   ├── widget.go         # Core Widget arayüzü (Draw ve SizeHint)
│   ├── block.go          # Kenarlıklı, başlıklı, Padding'li Blok kapsayıcısı
│   ├── paragraph.go      # Kelime kaydırmalı (Word wrap) çok satırlı metin widget'ı
│   └── list.go           # Seçilebilir, otomatik kaydırılabilir (scrolling) interaktif liste
└── examples/
    └── demo/main.go      # Fare tıklamasıyla liste seçimi ve hover ile çıkış destekleyen demo
```

---

## 3. Elde Edilen Performans ve Standartlar

Yeni geliştirilecek modüllerde bu teknik kararların ve performans kriterlerinin korunması **ZORUNLUDUR**:

- **Bellek Hizalaması (Memory Alignment)**:
  - `Style` boyutu **12 byte** (Fg: 4, Bg: 4, Modifier: 2 + 2 byte padding) olarak sabitlenmiştir.
  - `Cell` boyutu **16 byte** (Content: 4 + Style: 12) olarak sabitlenmiştir. L2 Cache dostu hizalama korunmalıdır.
- **Sıfır Heap Tahsisatı (Zero-Allocation on Draw)**:
  - Çizim ve diff döngüsünde dinamik bellek ayrımı yapılmaz.
  - `Terminal` içindeki `writeBuf` byte dilimi tekrar kullanılır. ANSI dönüşümlerinde `strconv.AppendInt` kullanılır.
  - 120x40 ekran (4800 hücre) güncelleme süresi **18.3 μs**'dir (Tek çekirdekte **54.000+ FPS**).
- **Stack-Allocated Context**:
  - `cell.Context` yapısı çizim sırasında alt widget'lara değer kopyalaması (by-copy) ile aktarılır. Bu, heap escape'i engeller.

---

## 4. Tamamlanan Aşamalar (Faz 1 - Faz 5)

- **Faz 1: Çekirdek Tampon ve Diff Motoru [TAMAMLANDI]**
  - Matris hizalaması, double-buffer diff ve sıfır tahsisatlı ANSI kodlayıcı tamamlandı.
- **Faz 2: Backend ve OS Terminal Kontrolü [TAMAMLANDI]**
  - CGO olmadan Linux ioctl termios kontrolü ve 25ms ESC süzgeçli Event Loop yazıldı.
- **Faz 3: Esnek Layout Motoru [TAMAMLANDI]**
  - Flexbox yerleşim motoru, oran, yüzde ve sabit bölünme matematiklerine göre yazıldı.
- **Faz 4: Terminal, Frame ve Block Widget'ı [TAMAMLANDI]**
  - Çizim Frame'i, Block kenarlık (rounded, single vb.) çizimleri ve fare tıklama yönlendirmesi kuruldu.
- **Faz 5: Zengin Widget Seti (Paragraph & List) [TAMAMLANDI]**
  - `Paragraph` (otomatik word-wrap) ve `List` (seçim durumu ve auto-viewport scroll) widget'ları interaktif tıklama özellikleri ile tamamlandı.

---

## 5. Gelecek Yol Haritası (Faz 6 - Faz 8)

Projeyi devralan ajanın sırasıyla gerçekleştirmesi beklenen sonraki aşamalar:

1. **Faz 6: Braille Canvas ve Vektör Çizim (High-Resolution Graphics)**:
   - Hücre başına 2x4 piksel çözünürlük sunan Braille Canvas widget'ı.
   - Vektörel çizim fonksiyonları (Çizgi, daire, dikdörtgen, bezier eğrileri çizme).
2. **Faz 7: Medya ve Görüntü Gösterim Katmanı (graphics)**:
   - Kitty Graphics Protocol, Sixel, iTerm2 ve Ghostty native resim kodlayıcıları (terminalde gerçek PNG/JPG çizimi).
3. **Faz 8: Animasyon Motoru**:
   - Zaman tabanlı interpolasyon, geçiş (transition) ve ivmelenme (easing) fonksiyonları.
