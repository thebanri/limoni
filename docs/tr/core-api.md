# ⚙️ Çekirdek Motor Mimarisi ve API Referansı (Core API)

Limoni, çizim sıcak yolunda (hot path) **sıfır yığın tahsisatı (zero heap allocations)**, donanım imleç senkronizasyonu ve deterministik olay işleme hedefleriyle tasarlanmış modüler ve katmanlı bir mimariye sahiptir.

---

## 1. Birleşik Kök Arayüz (`package limoni`)

Uygulamaların %95'inde yalnızca kök paketi içe aktarmanız yeterlidir:

```go
import "github.com/thebanri/limoni"
```

Kök paket şunları doğrudan dışa aktarır:
- **Temel Tipler**: `Frame`, `Terminal`, `Rect`, `Style`, `Color`, `Event`, `KeyEvent`, `MouseEvent`.
- **Yüksek Seviyeli Çalıştırıcılar**: `limoni.Start()`, `limoni.Run()`, `limoni.NewProgram()`, `limoni.RunProgram()`.
- **Composable Lego Blokları**: `limoni.VStack()`, `limoni.HStack()`, `limoni.Pad()`, `limoni.PadAll()`, `limoni.PadAxis()`, `limoni.Border()`, `limoni.Center()`, `limoni.AlignComponent()`, `limoni.Flex()`, `limoni.FixedSize()`, `limoni.Label()`, `limoni.AsComponent()`.
- **Yerleşim Bölücüleri**: `limoni.SplitVertical()`, `limoni.SplitHorizontal()`, `limoni.Fixed()`, `limoni.Percentage()`, `limoni.Fill()`, `limoni.Ratio()`.
- **Akıcı Yapıcılar (Fluent Builders)**: `limoni.NewBlock()`, `limoni.NewParagraph()`, `limoni.NewTable()`, `limoni.NewList()`, `limoni.NewTextInput()`, `limoni.NewMarkdown()`.
- **Renk ve Stil Yardımcıları**: `limoni.RGB()`, `limoni.Hex()`, `limoni.ANSI()`, `limoni.Fg()`, `limoni.Bg()`, `limoni.Bold()`, `limoni.Italic()`.

---

## 2. `core/cell` — Atomik Hücreler ve Geometrik Matematik

Terminal ızgarasının en temel yapı taşını tanımlar: karakter hücreleri, TrueColor RGB renkler, metin biçimlendiriciler ve sınırlayıcı dikdörtgenler (`Rect`).

### `cell.Color`
24-bit TrueColor RGB, 8-bit ANSI veya varsayılan terminal renklerini tek bir 32-bit tamsayıda saklar:

```go
colRGB  := cell.NewColorRGB(0, 255, 180)
colANSI := cell.NewColorANSI(196)
colDef  := cell.NewColorDefault()
```

### `cell.Style`
Ön plan, arka plan ve bitmask modifikatörlerini içeren 12-byte sıkıştırılmış struct:

```go
style := cell.Style{
    Fg: cell.NewColorRGB(0, 210, 255),
    Bg: cell.NewColorRGB(18, 20, 24),
    Modifier: cell.ModifierBold | cell.ModifierUnderline,
}

// Stilleri birleştirme (üzerine yazma):
birlestirilmis := anaStil.Merge(ekStil)
```

### `cell.Rect`
Sıfır yığın tahsisatlı 2B sınırlayıcı kutu matematiği:

```go
alan := cell.NewRect(0, 0, 80, 24)
icindeMi := alan.Contains(10, 5)
kesisim := alan.Intersection(digerAlan)
```

### `cell.Context`
Widget'ların çizim anında aldığı bağlam nesnesi:
- `ctx.Area`: Aktif sınırlayıcı alan.
- `ctx.Style`: Üst kapsayıcıdan veya temadan devralınan stil.
- `ctx.ThemeStyle(role)`: Anlamsal tema renklerini çözer (`"surface"`, `"border"`, `"text"`).
- `ctx.RegisterClick(area, handler)`: Tıklanabilir bölgeleri kaydeder.
- `ctx.RegisterMouse(area, handler)`: Fare gezintisi, sürükleme ve kaydırma bölgelerini kaydeder.
- `ctx.RegisterFocus(id)`: Odaklanabilir öğe sınırlarını kaydeder.

---

## 3. `core/buffer` — 1D Bellek Matrisi ve ANSI Diff Motoru

Diferansiyel ANSI çizimi yapan yüksek performanslı ardışık hücre tamponu.

### Temel Özellikler
- **1D Düz Bellek Dizisi**: `Buffer.Content`, tek bir ardışık `[]cell.Cell` dilimi olarak saklanır; bu sayede CPU önbellek yerelliği maksimize edilir ve işaretçi atlamaları önlenir.
- **Diferansiyel Çizim (`buffer.Diff`)**: `front` ve `back` tamponlarını karşılaştırarak ekranı güncellemek için gereken mutlak minimum ANSI kaçış dizisini hesaplar.
- **Unicode Doğu Asya Genişliği (`W`/`F`) Doğruluğu**: Tablo tabanlı genişlik hesaplamaları sayesinde tek genişlikli semboller (`✓`, `⚠`) 1 sütun, emojiler (`🔴`, `🚀`, `☕`) 2 sütun kaplar ve imleç kayması yaşanmaz.
- **Devam Hücresi (Continuation Cell) Koruması**: Geniş karakterler kısmen örtüldüğünde veya geri yüklendiğinde (örn. modal bir pencere sürüklendiğinde), devam hücreleri (`RuneContinuation`) otomatik olarak geçersiz kılma tetikleyerek kenarlık parçalanmasını ve hayalet karakterleri tamamen engeller.

```go
buf := buffer.NewBuffer(area)
buf.SetCellDirect(x, y, cell.Cell{Content: 'A', Style: style})
buf.SetString(x, y, "🚀 Merhaba Limoni", style)

// ANSI fark akışını üret
writeBuf, err := buffer.Diff(frontBuf, backBuf, writeBuf[:0], true, true)
```

---

## 4. `core/terminal` — Motor, Katmanlar, Modallar ve Odak

Terminal yaşam döngüsünü, çift tamponlamayı, kare üretimini ve girdi yönlendirmesini yönetir.

### Temel Yetenekler
- **60+ FPS Çizim Boru Hattı**: Kareleri bellek tamponunda çizer, önceki kare ile farkını alır ve tamponlu G/Ç ile terminale yazar.
- **Katman Yığınlama (Layer Stacking)**: `f.BeginLayer("overlay")` ve `f.EndLayer()` ile yıkıcı olmayan şeffaf paneller ve menüler açılabilir.
- **Modal İzolasyonu**: `f.RegisterModal("dialog", area, onDismiss)` ile modal açıkken altta kalan widget'ların fare ve klavye olaylarını alması engellenir.
- **Odak Kapsamı (Focus Scoping)**: `f.BeginFocusScope("modal_id")` ile `Tab` / `Shift+Tab` klavye gezintisi yalnızca modal içerisindeki etkileşimli öğelerle sınırlandırılır.

---

## 5. `core/engine` — The Elm Architecture (TEA)

Öngörülebilir, fonksiyonel durum yönetim döngüsü:

```
[Init] ──> Model + Cmd
             │
             ▼
[Mesaj] ──> [Update] ──> Model + Cmd
                            │
                            ▼
                         [View] ──> Frame Çizimi
```

### Güvenlik ve Eşzamanlılık Garantileri
- **Deterministik Komut Sıralaması**: Komutlar arkaplan goroutine'lerinde eşzamanlı çalışır, ancak sonuçları `Update` fonksiyonuna geliş sırasına göre deterministik iletilir.
- **Katı İptal Önceliği**: Context iptal edildiğinde (`ctx.Done()`) veya program kapandığında, bekleyen tüm komut sonuçları ve kuyruktaki mesajlar derhal temizlenir; çıkıştan sonra bellek yarışmaları ve geç mutasyonlar engellenir.
- **Güvenli Panik Yakalama**: `WithPanicHandler` ile kullanıcı kodundaki panikler yakalanır, uygulamanın çökmesi engellenir ve telemetri günlüğü tutulabilir.

---

## 6. `core/driver` — Çapraz Platform VT & Ham Terminal Motoru

İşletim sistemi terminal sürücüsüyle doğrudan iletişim kurar:

- **Linux / macOS**: `termios` ile ham moda geçer, alternatif ekran tamponunu (`\x1b[?1049h`), fare takibini (`\x1b[?1006h`) ve parantezli yapıştırmayı (bracketed paste) açar.
- **Windows**: `ENABLE_VIRTUAL_TERMINAL_PROCESSING` ile yerel Win32 Console API'sini (`GetConsoleMode`, `SetConsoleMode`) ve yerel olay döngüsü girdi çözücüsünü kullanır.
- **Sinyal Yönetimi**: Unix'te `SIGWINCH`, Windows'ta konsol yeniden boyutlandırma olaylarını dinleyerek ekranın anında yeniden hesaplanmasını sağlar.

---

## 7. `component` — Composable Lego Blok Mimarisi

`pony` ve `glyph` gibi modern kütüphanelerden esinlenen `component` paketi; monolitik widget yapılarına alternatif olarak hafif, modüler ve fonksiyonel bir bileşen ağacı sunar. Limoni'nin **çizim sıcak yolunda (hot path) 0 bellek tahsisatı (zero-alloc)** garantisini aynen muhafaza eder.

### Birleşik `Component` Arayüzü
Her composable blok şu arayüzü uygular:
```go
type Component interface {
    Draw(ctx cell.Context, buf *buffer.Buffer)
    LayoutInfo(maxArea cell.Rect) LayoutProps
    SizeHint(maxArea cell.Rect) (width, height uint16)
}
```
`Draw` ve `SizeHint` fonksiyonlarını sağladığı için her `Component` doğrudan `widgets.Widget` ile uyumludur.

### Dekoratörler ve Sarmalayıcılar (Wrappers)
Kenarlık, dolgu ve hizalama gibi özellikler widget'ların içine gömülmek yerine harici dekoratörlerle sarılır:
- `limoni.Border(child, symbols, style)`: Bileşeni dekoratif bir kenarlıkla sarar.
- `limoni.Pad(child, t, r, b, l)`: Bileşene iç dolgu (padding) ekler.
- `limoni.Center(child)` / `limoni.AlignComponent(child, h, v)`: Bileşeni ayrılan alan içinde ortalar veya hizalar.
- `limoni.Flex(weight, child)`: Yığın içinde dinamik oranlı esneme katsayısı tanımlar.
- `limoni.FixedSize(w, h, child)`: Bileşene sabit genişlik ve yükseklik kısıtı atar.
- `limoni.AsComponent(w)`: Var olan monolitik widget'ları (`Table`, `List`, `Block`) composable ağaca bağlar.

### Çizim (Rendering)
```go
view := limoni.VStack(
    limoni.FixedSize(0, 3, limoni.Border(limoni.Center(limoni.Label("Başlık")), widgets.SymbolsRounded, limoni.Fg(limoni.ColorCyan))),
    limoni.Flex(1, limoni.HStack(
        limoni.Flex(1, limoni.AsComponent(sidebar)),
        limoni.Flex(2, limoni.AsComponent(mainContent)),
    )),
    limoni.FixedSize(0, 1, limoni.Label("Durum: Hazır")),
)

f.RenderComponent(view, f.Area())
```

