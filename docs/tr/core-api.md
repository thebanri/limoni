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

## 5. `core/runtime` — The Elm Architecture (TEA)

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

## 6. `core/backend` — Çapraz Platform VT & Ham Terminal Motoru

İşletim sistemi terminal sürücüsüyle doğrudan iletişim kurar:

- **Linux / macOS**: `termios` ile ham moda geçer, alternatif ekran tamponunu (`\x1b[?1049h`), fare takibini (`\x1b[?1006h`) ve parantezli yapıştırmayı (bracketed paste) açar.
- **Windows**: `ENABLE_VIRTUAL_TERMINAL_PROCESSING` ile yerel Win32 Console API'sini (`GetConsoleMode`, `SetConsoleMode`) ve yerel olay döngüsü girdi çözücüsünü kullanır.
- **Sinyal Yönetimi**: Unix'te `SIGWINCH`, Windows'ta konsol yeniden boyutlandırma olaylarını dinleyerek ekranın anında yeniden hesaplanmasını sağlar.
