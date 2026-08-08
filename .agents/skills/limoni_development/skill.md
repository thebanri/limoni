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

## 4. Tamamlanan Aşamalar (Faz 1 - Faz 7)

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
- **Faz 6: Braille Canvas ve Vektör Çizim (High-Resolution Graphics) [TAMAMLANDI]**
  - Hücre başına 2x4 piksel çözünürlük sunan Braille Canvas widget'ı ve vektörel çizim fonksiyonları (Çizgi, daire, dikdörtgen, bezier eğrileri çizme) tamamlandı.
- **Faz 7: Medya ve Görüntü Gösterim Katmanı (graphics) [TAMAMLANDI]**
  - Kitty Graphics Protocol, Sixel, iTerm2 ve Ghostty yerel resim kodlayıcıları (terminalde gerçek PNG/JPG çizimi) tamamlandı. Kitty protokolündeki 4096 byte tekil veri paketi sınırı için base64 chunking desteği eklendi.
  - Alacritty gibi yerel protokol desteği bulunmayan terminaller için evrensel **Half-Block (`▄` U+2584)** 1x2 çözünürlüklü hücre-tamponu üstüne çizim desteği ve `Image` widget'ı tamamlandı.
  - `Block` gibi konteyner widget'ların alt bileşenlerine `RegisterClick` ve `RegisterImage` olay köprülerini (callback) doğru şekilde aktaramaması hatası düzeltilerek konteyner içi resim çizdirme ve tıklama desteği kararlı hale getirildi.

- **Faz 8: Animasyon Motoru [TAMAMLANDI]**
  - Zaman tabanlı interpolasyon, yumuşak geçiş efektleri (transitions) ve 16 adet standart ivmelenme (easing - Linear, Quad, Cubic, Sine, Expo, Bounce) fonksiyonu içeren `animation` paketi geliştirildi.
  - Sayısal değerler için `Float` ve 24-bit TrueColor RGB renk geçişleri için `Color` animasyon yapıları oluşturuldu.
  - FPS izleme ve time.Ticker ile event loop'u tıkamadan 30 FPS çizim yeteneğini gösteren animasyonlu bir demo uygulaması (`examples/animation`) eklendi.

- **Faz 9 (Zengin Form ve Girdi Kontrolleri):**
  - Çizim sırasına göre dinamik sekmeli odak geçişini yöneten `FocusManager` yapısı geliştirildi.
  - Tek satırlı interaktif `TextInput` (durum ve imleç yönetimiyle), boolean onay kutusu `Checkbox` ve tekli seçim aracı `RadioButton` bileşenleri kütüphaneye kazandırıldı.
  - Birim testleri (`widgets/form_test.go`) ve ana demoda dinamik renk teması geçişiyle entegre form gösterimi yapıldı.

- **Faz 10: Katmanlı Render, Modal Pencere ve Açılır Menü (Popup) [TAMAMLANDI]**
  - `Layer` yapısı (ID, Type, Area, ZIndex, ClickOutside) ile birden fazla üst üste binen katman desteği eklendi.
  - `Frame.BeginLayer(id)` / `Frame.EndLayer()` API'si ile widget'ların hangi katmana ait olduğunu çizim anında bildirmesi sağlandı.
  - `Frame.RegisterLayer()` ve `Frame.RemoveLayer()` ile dinamik katman ekleme/kaldırma desteği.
  - `RouteMouseEvent()` çoklu katman için z-index bazlı olay yönlendirmesi: En üstteki katman önce yakalar, click-outside tetikler.
  - `Popup` widget'ı: Başlangıç butonu + açılır liste, hover seçimi, disabled öğe desteği, border çizimi.
  - `FocusManager` ile modal içinde odak kilitlenmesi (focus trapping): Modal dışındaki widget'ların odak/tıklama kayıtları otomatik olarak bloke edilir.
  - `ClickRegion` yapısına `LayerID` alanı eklenerek tıklama bölgelerinin hangi katmana ait olduğu takip edilir.
  - Geriye dönük uyumluluk: Eski `RegisterModal()` API'si korundu; hem `ActiveModal` hem `Layers` listesini günceller.
  - Mevcut `RegisterImage` callback imzası hatası (2 parametre → 3 parametre) düzeltildi.
  - `layer_demo` örnek uygulaması ile multi-layer, popup menüler ve modal etkileşimi gösterildi.
  - 8 adet yeni birim testi eklendi: `TestLayerSystemBasic`, `TestMultiLayerZOrdering`, `TestRemoveLayer`, `TestTopLayer`, `TestResetClearsLayers` vb.

- **Faz 11: İnteraktif ve Esnek Hücreli Tablo (Table) Bileşeni [TAMAMLANDI]**
  - `widgets/table.go` tablo durum yönetimi, kısıtlama çözücüsü (`SolveWidths`) ve `Table` widget'ı kodlandı.
  - Hücre bazlı kırpma (clipping), ızgara çizgileri, zebra deseni ve dinamik sütun yerleşimi desteği sağlandı.
  - Birim testleri (`widgets/table_test.go`) ve ana demoda interaktif süreç tablosu gösterimi başarıyla yapıldı.
  - Kitty Grafik Protokolü için donanımsal z-index desteği ve hücresel Half-Block (`ForceHalfBlock`) katmanlama özelliği eklendi.

- **Faz 12: Gelişmiş Katmanlı Render ve Animasyonlu Geçişler [TAMAMLANDI]**
  - `ScaleRect` ve `SlideUpRect` yardımcı matris formülleri ile diyalog ve modal açılış/kapanış animasyonları kodlandı.
  - Z-Index öncelikli modal yığını (`TopmostModal` ve sandboxing) mekanizması `frame.go` içerisine entegre edildi.
  - Odaklanan widget'ların etrafında parlayan kalın ve kesikli odak kenarlıkları (`DrawFocusRing`) geliştirildi.
  - `modal_transition_test.go` birim testleri ve ana demoda animasyonlu geçişler/odak halkaları entegre edildi.

- **Faz 13: İnteraktif TUI Oyun Alanı (Playground) ve Dinamik Düzen Kontrolü [TAMAMLANDI]**
  - Ana demoya sol menü üzerinden geçilebilen interaktif bir Oyun Alanı (Playground) sekmesi eklendi.
  - Klavye yön tuşları, `+`/`-` oran değiştiricileri ve fare tıklama olaylarını işleyen canlı kontrol paneli oluşturuldu.

- **Faz 14: CSS Grid Yerleşimi, Markdown Çizici, Retro Dither Geçişleri ve Vektör Renk Karışımı [TAMAMLANDI]**
  - Columns/Rows/Gap parametreleriyle 2D ızgara bölen ve `.Span(rowSpan, colSpan)` birleştirmesini destekleyen `layout/grid.go` CSS Grid motoru kodlandı.
  - `#` başlıklar, `**kalın**`, `*italik*`, `- liste` ve inline kod bloklarını parse eden, UTF-8 rune duyarlı word-wrap özellikli `widgets/markdown.go` bileşeni geliştirildi.
  - Bayer 4x4 matrisi ile ekran sekmeleri arasında pürüzsüz retro karıncalanma (dither-fade) geçişi sağlandı.
  - Donanımsal resim çizimlerinde anti-aliased dairesel kırpma (circle mask) uygulayan avatar filtresi eklendi.
  - Braille Vektör Canvas'ta üst üste gelen çizgiler/grafikler için sub-pixel düzeyinde RGB renk harmanlama (color blending) motoru Set metoduna entegre edildi.

- **Faz 15: Terminal Parçacık Yağmuru (Matrix Rain) ve Yüksek Çözünürlüklü Grafik Çizici (Sparkline) [TAMAMLANDI]**
  - 8 farklı dikey blok karakteriyle dikey sığdırma ve normalize hesabı yapıp çok satırlı alan grafikleri çizen `widgets/sparkline.go` geliştirildi.
  - Dikey akan matris parçacıklarını animasyon döngüsünde yürüten Matrix Rain Canvas simülasyonu kodlandı.
  - Ekran boyutu değiştiğinde oluşan karakter kalıntılarını engelleyen "Resize Ghosting Fix" tampon sıfırlayıcı motoru entegre edildi.

- **Faz 16: İnteraktif Fare Sürüklemeli Diyalog ve Kısayol Yardım Paneli [TAMAMLANDI]**
  - `RegisterClickHandler` ve `MouseEvent.Drag` olayları takip edilerek diyalogların başlık çubuklarından (Title bar) sol tıkla basılı tutulup ekranda serbestçe sürüklenebilmesi (TUI Window Dragging) sağlandı.
  - `?` tuşuna basıldığında açılan, CSS Grid ve Markdown yardımı ile dairesel profil avatarını içeren Kısayol Yardım Paneli modalı başarıyla eklendi.

- **Faz 17: Gelişmiş Performans Profili ve Sıfır-Tahsisat (Zero-Allocation) Optimizasyonları [TAMAMLANDI]**
  - `widgets/markdown.go` bileşeni işaretçi alıcılı yapılarak ön bellekleme (AST/Layout caching) mekanizması entegre edildi.
  - Markdown çizim performansı **11.3 kat** arttırıldı ve draw loop sırasında gerçekleşen heap bellek tahsisatı **sıfıra (0 B/op, 0 allocs/op)** düşürüldü.

---

- **Faz 18: TUI Yerleşim Müfettişi ve Hata Ayıklama Katmanı (Layout Inspector / Debug HUD) [TAMAMLANDI]**
  - Çizilen tüm widget'ların türlerini, boyutlarını ve z-index değerlerini otomatik kaydeden `DebugRegions` yapısı kodlandı.
  - `Ctrl+D` kısayoluyla tetiklenen, üst katmanların alt katmanların çizgilerini örtmesini sağlayan pixel-perfect **Z-Order Kırpma (Layout Clipping)** özellikli Debug HUD katmanı entegre edildi.
  - Odaklanmış tablolarda ve butonlarda tuşların yutulmasını önleyen global öncelikli klavye yönlendirme sistemi (Keyboard Focus Fix) uygulandı.

- **Faz 19: İnteraktif Fare ile Pencere Yeniden Boyutlandırma (Resizable Window Modals) [TAMAMLANDI]**
  - Kısayol Yardım penceresinin sağ alt köşesine mor renkli `◢` yeniden boyutlandırma tutamacı çizildi ve click/drag takibi entegre edildi.
  - Kullanıcı fareyle sürükledikçe pencere boyutunun dinamik güncellenmesi sağlandı (min: 40x10, maks: 100x30). İçerideki Flex layout bölmeleri pencere boyutuna göre Markdown ve profil avatarını dinamik/oransal olarak yeniden boyutlandırır.

---

- **Faz 20: Animasyonlu Geçiş Efektli Widget'lar (Animated Widget Transducers) [TAMAMLANDI]**
  - `widgets.Transducer` ve lokal dither sekme geçişleri ile animasyonlu modal geçişleri tamamlandı.
- **Faz 21: 3 Boyutlu Vektör Grafik Motoru (3D Wireframe Graphics Engine) [TAMAMLANDI]**
  - Perspektif projeksiyon (`Project`) ve eksen rotasyon (`RotateX`/`RotateY`/`RotateZ`) fonksiyonları eklendi.
  - Braille Canvas üzerinde otomatik dönen ve sol tık sürüklemeyle yönlendirilebilen 3D Küp entegrasyonu tamamlandı.

---

## 5. Gelecek Yol Haritası (Faz 22 - Faz 23)

Projeyi devralan ajanın sırasıyla gerçekleştirmesi beklenen sonraki aşamalar:

1. **Faz 22: Komut Paleti ve Kısayol Yönlendirici (Command Palette & Keybindings)**:
   - `Ctrl+P` ile açılan, tüm sekmeler ve eylemler arasında bulanık arama (fuzzy search) yapıp tetikleyen bir Komut Paleti (Command Palette) widget'ı tasarlamak.
   - Declarative klavye kısayol yöneticisi paketi geliştirip event loop'a bağlamak.
2. **Faz 23: Gelişmiş Tablo Hücre Birleştirme ve Sütun Boyutlandırma (Table Cell Spanning & Column Resizing)**:
   - Tablonun sütun sınırlarının fareyle sürüklenerek genişletilip daraltılabilmesi.
   - `colSpan` ve `rowSpan` özellikleriyle tablo içinde birleşik hücre desteği kazandırmak.

## 6. Dosya Yapısı (Güncel)

```
limoni/
├── go.mod
├── .agents/skills/limoni_development/skill.md  # Bu el kitabı
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
│   │   ├── termios_linux # Unix ioctl çağrıları ile Pure Go Raw Mode kontrolü\ n│   │   ├── parser.go     # Klavye/mouse/focus/hover ANSI dizi çözümleyicisi
│   │   └── backend.go    # SIGWINCH ve 25ms ESC timeout asenkron Event Loop
│   └── terminal/
│       ├── frame.go      # Kare çizim bağlamı, Click handler kaydı, Layer API
│       ├── terminal.go   # Terminal yöneticisi, Draw döngüsü, Multi-layer Mouse Router
│       ├── focus.go      # FocusManager: Tab/Shift-Tab navigasyonu
│       └── modal.go      # Modal, Layer, LayerType yapıları, CenterRect, ContainsRect
├── layout/
│   └── layout.go         # Flexbox yerleşim motoru (Fixed, Percentage, Ratio, Min, Max, Fill)
├── widgets/
│   ├── widget.go         # Core Widget arayüzü (Draw ve SizeHint)
│   ├── block.go          # Kenarlıklı, başlıklı, Padding'li Blok kapsayıcısı
│   ├── paragraph.go      # Kelime kaydırmalı (Word wrap) çok satırlı metin widget'ı
│   ├── list.go           # Seçilebilir, otomatik kaydırılabilir (scrolling) interaktif liste
│   ├── table.go          # Esnek sütunlu tablo, başlık, ızgara çizgileri
│   ├── dialog.go         # 3D gölgeli modal diyalog penceresi
│   ├── textinput.go      # Tek satırlı interaktif metin girdisi
│   ├── checkbox.go       # Onay kutusu [ ]/[x]
│   ├── radio.go          # Tekli seçim aracı ( )/(*)
│   ├── popup.go          # Açılır menü (dropdown) widget'ı
│   ├── canvas.go         # Braille 2x4 alt piksel çözünürlüklü çizim alanı
│   ├── vector.go         # Bresenham çizgi, daire, dikdörtgen, bezier eğri çizimi
│   └── image.go          # Kitty/Sixel/iTerm2/HalfBlock resim gösterimi
├── animation/
│   ├── float.go          # Zaman tabanlı float interpolasyonu
│   ├── color.go          # RGB renk geçiş animasyonu
│   └── easing.go         # 15+ ivmelenme (easing) fonksiyonu
├── graphics/
│   └── graphics.go       # Protokol algılama, Kitty/Sixel/iTerm2 kodlayıcıları
└── examples/
    ├── demo/main.go      # Tam interaktif demo (tüm Faz 1-9 özellikleri)
    ├── animation/main.go # Animasyon gösterisi
    └── layer_demo/main.go # Faz 10: Katmanlı render, modal, popup demo
```


