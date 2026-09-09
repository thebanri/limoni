# 📐 Yerleşim Düzeni Rehberi (Layout Engine Guide): Flexbox & CSS Grid

Limoni, modern CSS Flexbox, CSS Grid ve Ratatui'nin kısıt tabanlı (constraint-based) geometri sisteminden ilham alan, sıfır-tahsisatlı (zero-allocation) güçlü bir yerleşim motoru içerir.

---

## 1. Yüksek Seviyeli Hızlı Bölücüler (Splitters)

Çoğu terminal arayüzünde kullanılan yaygın kalıplar için Limoni pratik bölücüler sunar:

```go
import "github.com/thebanri/limoni"

// 1. Dikey Bölme (Örn: Başlık, Ana Gövde, Durum Çubuğu)
rows := limoni.SplitVertical(area,
    limoni.Fixed(3), // 3 satır başlık
    limoni.Fill(),   // Kalan tüm dikey alanı doldurur
    limoni.Fixed(1), // 1 satır alt çubuk
)
headerArea := rows[0]
bodyArea   := rows[1]
footerArea := rows[2]

// 2. Yatay Bölme (Örn: Yan Menü ve Ana İçerik)
cols := limoni.SplitHorizontal(bodyArea,
    limoni.Percentage(25), // %25 genişlik
    limoni.Percentage(75), // %75 genişlik
)
sidebarArea := cols[0]
contentArea := cols[1]

// 3. Orantılı / Ağırlıklı Bölme (Ratio)
cards := limoni.SplitHorizontal(contentArea,
    limoni.Ratio(1),
    limoni.Ratio(2), // Ratio(1)'in iki katı kadar alan alır
)
```

---

## 2. Flexbox Motoru (`layout.FlexLayout`)

Öğeler arasındaki boşluklar (gap), kenar boşlukları ve yön üzerinde hassas kontrol gerektiğinde `FlexLayout` kullanılır:

```go
import "github.com/thebanri/limoni/layout"

chunks := layout.NewFlexLayout(
    layout.Vertical, // veya layout.Horizontal
    1,               // Hücre cinsinden boşluk (gap)
    layout.Fixed(4),
    layout.Fill(),
    layout.Fixed(2),
).Split(area)
```

### Kısıt (Constraint) Tipleri

| Kısıt | Açıklama | Örnek |
| :--- | :--- | :--- |
| **`Fixed(N)`** | Tam olarak $N$ adet terminal hücresi ayırır. | `limoni.Fixed(3)` (Başlık çubuğu) |
| **`Percentage(P)`** | Kullanılabilir alanın $\%P$ kadarını ayırır (0–100). | `limoni.Percentage(30)` (Yan panel) |
| **`Ratio(R)`** | Kalan serbest alanı ağırlıklarına göre orantılı dağıtır. | `limoni.Ratio(2)`, `limoni.Ratio(1)` |
| **`Fill()`** | Kalan tüm alanı doldurur (`Ratio(1)` kısayoludur). | `limoni.Fill()` (Ana çalışma alanı) |
| **`Min(N)`** | En az $N$ hücre tahsis edilmesini garanti eder. | `limoni.Min(15)` |
| **`Max(N)`** | Ayrılacak alanı en fazla $N$ hücre ile sınırlar. | `limoni.Max(40)` |
| **`FitContent()`** | İçerideki widget'ın `SizeHint` ölçümüne göre tam gereken alanı ayırır. | `limoni.FitContent()` |

---

## 3. CSS Grid Motoru (`layout.GridLayout`)

Limoni, satır ve sütunların 2 boyutlu ızgara üzerinde hassas parçalara bölünebildiği bir CSS Grid motoruna sahiptir:

```go
grid := layout.NewGridLayout(
    area,
    // Sütunlar: 20 hücre, kalanı doldur, toplamın %25'i
    []layout.Constraint{layout.Fixed(20), layout.Fill(), layout.Percentage(25)},
    // Satırlar: 3 satır, kalanı doldur, 5 satır
    []layout.Constraint{layout.Fixed(3), layout.Fill(), layout.Fixed(5)},
)

// Hücrelere koordinatla erişim: (satır, sütun)
solUst  := grid.Cell(0, 0)
merkez  := grid.Cell(1, 1)
sagAlt  := grid.Cell(2, 2)

// Birden fazla satır veya sütunu birleştirme (Span)
bannerAlani := grid.Area(0, 0, 1, 3) // Satır 0, Sütun 0, 1 Satır Yüksek, 3 Sütun Geniş
```

---

## 4. İç ve Dış Boşluklar (Padding, Margin, Inset)

`Block` gibi kapsayıcı bileşenler CSS benzeri iç ve dış boşlukları destekler:

```go
block := limoni.NewBlock().
    WithTitle(" Kart ").
    Rounded().
    WithPadding(1, 2, 1, 2) // Üst, Sağ, Alt, Sol

// Kenarlıklar ve padding düşüldükten sonraki çizilebilir iç alanı almak:
icAlan := block.Inner(disAlan)
```

---

## 5. İçiçe Yerleşim Örneği (Responsive Dashboard)

Yatay ve dikey bölmelerin bir arada kullanıldığı tam teşekküllü bir gösterge paneli örneği:

```go
func drawDashboard(f *limoni.Frame) {
    screen := f.Area()

    // Kök: Üst çubuk (3 satır), İçerik (Kalan alan), Alt durum çubuğu (1 satır)
    mainRows := limoni.SplitVertical(screen, limoni.Fixed(3), limoni.Fill(), limoni.Fixed(1))

    // Başlık
    f.RenderWidget(limoni.NewBlock().
        WithTitle(" 🚀 BULUT YÖNETİM MERKEZİ ").
        WithTitleAlign(limoni.AlignCenter).
        WithBorderStyle(limoni.Fg(limoni.Hex("#00E5FF"))), mainRows[0])

    // Gövde: Sol Menü (%20), Orta Çalışma Alanı (%55), Sağ Telemetri Paneli (%25)
    bodyCols := limoni.SplitHorizontal(mainRows[1],
        limoni.Percentage(20),
        limoni.Percentage(55),
        limoni.Percentage(25),
    )

    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle(" Servisler "), bodyCols[0])
    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle(" İş Yükleri "), bodyCols[1])
    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle(" Telemetri "), bodyCols[2])

    // Alt bilgi
    f.RenderWidget(limoni.NewParagraph(" [q] Çıkış  [tab] Odak Değiştir  [?] Yardım").
        WithStyle(limoni.Fg(limoni.RGB(140, 150, 165))), mainRows[2])
}
```

---

## 6. Composable Lego Blok Bileşen Mimarisi (VStack, HStack ve Dekoratörler)

Limoni, geleneksel monolitik widget ve manuel alan bölme (Split) mimarisine ek olarak; **modern, fonksiyonel ve composable (Lego blokları gibi birleştirilebilir)** bir bileşen katmanı sunar.

Bu katman, `pony` ve `glyph` gibi modern UI kütüphanelerinden esinlenmiş olup, Limoni'nin sıfır bellek tahsisli (0 allocs/op) 1D hücre tamponu (contiguous cell buffer) mantığını bozmadan çalışır.

### Temel Prensipler

1. **Birleşik Arayüz (`Component`)**:
   Her bileşen yalnızca kendini çizmeyi ve boyut kısıtlarını (`LayoutInfo`) bildirmeyi bilir. Aynı zamanda `widgets.Widget` ile %100 uyumludur.
2. **Dekoratör Sarmalayıcılar**:
   Kenarlık (`Border`), dolgu (`Pad`), hizalama (`Center`, `AlignComponent`) gibi özellikler monolitik widget alanları yerine harici sarmalayıcı (wrapper) olarak uygulanır.
3. **Esnek Yığınlar (`VStack` / `HStack`)**:
   Çocuk bileşenler ağırlıklı esneme (`Flex(n)`) veya sabit boyut (`FixedSize(w, h)`) kurallarıyla otomatik olarak paylaştırılır.
4. **Sıfır Bellek Tahsisi (Hot Path Zero-Alloc)**:
   Yığın yerleşim hesaplamaları stack üzerinde (`[32]uint16`) yürütülür, render anında heap allokasyonu yapılmaz.

### Composable Lego Fonksiyonları

| Fonksiyon | Açıklama |
| :--- | :--- |
| `limoni.VStack(children...)` | Dikey yığın yerleşimi (`.WithGap`, `.WithJustify`, `.WithAlignItems`). |
| `limoni.HStack(children...)` | Yatay yığın yerleşimi (`.WithGap`, `.WithJustify`, `.WithAlignItems`). |
| `limoni.ZStack(children...)` | Derinlik ekseni (Painter's algorithm) katmanlama; arka plandan ön plana tamponsuz doğrudan çizer. |
| `limoni.When(cond, then, [else])` | Bildirimsel koşullu çizim (sıfır ek sarmalayıcı maliyeti). |
| `limoni.Match(val, cases, [default])` | Bildirimsel örüntü eşleştirme (pattern matching) seçicisi. |
| `limoni.WithStyle(style, child)` | Stili alt bileşen ağacına çağrı yığını üzerinden kopyalayarak aktarır. |
| `limoni.WithForeground(fg, child)` | Ön plan rengini alt bileşen ağacına aktarır. |
| `limoni.WithBackground(bg, child)` | Arka plan rengini alt bileşen ağacına aktarır. |
| `limoni.OnClick(child, handler)` | Bileşeni fare tıklama yöneticisiyle sarar. |
| `limoni.Pad(child, top, right, bottom, left)` | Bileşenin etrafına iç dolgu (padding) ekler. |
| `limoni.PadAll(child, p)` | 4 tarafa eşit dolgu uygular. |
| `limoni.PadAxis(child, h, v)` | Yatay ve dikey simetrik dolgu uygular. |
| `limoni.Border(child, sym, style)` | Herhangi bir bileşeni dekoratif kenarlıkla sarar. |
| `limoni.Center(child)` | Bileşeni tahsis edilen alanda hem yatay hem dikey ortalar. |
| `limoni.AlignComponent(child, h, v)` | Bileşeni kurallara göre hizalar (`AlignLeft`, `AlignCenter`, `AlignRight`, `AlignTop`, `AlignMiddle`, `AlignBottom`). |
| `limoni.Flex(weight, child)` | Yığın içinde dinamik genişleme katsayısı belirler (CSS flex-grow). |
| `limoni.FixedSize(w, h, child)` | Bileşene sabit genişlik ve yükseklik atar. |
| `limoni.Label(text, styles...)` | Hafif satır içi metin bileşeni oluşturur. |
| `limoni.AsComponent(widget)` | Var olan monolitik widget'ları (`Table`, `Block`, `List`) anında composable sisteme adapte eder. |

### Bildirimsel (Declarative) Dashboard Örneği

```go
package main

import (
    "github.com/thebanri/limoni"
    "github.com/thebanri/limoni/widgets"
)

func renderComposableUI(f *limoni.Frame, sidebarList, mainTable limoni.Widget) {
    // Lego blokları gibi iç içe kurulmuş bildirimsel görünüm ağacı:
    view := limoni.VStack(
        // Sabit yükseklikte başlık çubuğu
        limoni.FixedSize(0, 3, limoni.Border(
            limoni.Center(limoni.Label("🚀 BULUT İZLEME PANELİ", limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
            widgets.SymbolsRounded,
            limoni.Fg(limoni.Hex("#00FFAA")),
        )),

        // 1:2 oranında esneyen gövde alanı
        limoni.Flex(1, limoni.HStack(
            limoni.Flex(1, limoni.Border(
                limoni.AsComponent(sidebarList),
                widgets.SymbolsSingle,
                limoni.Fg(limoni.Hex("#FFCC00")),
            )),
            limoni.Flex(2, limoni.Border(
                limoni.AsComponent(mainTable),
                widgets.SymbolsDouble,
                limoni.Fg(limoni.Hex("#3399FF")),
            )),
        )),

        // Sabit yükseklikte alt bilgi çubuğu
        limoni.FixedSize(0, 3, limoni.Border(
            limoni.Center(limoni.Label("ESC: Çıkış | 0 B/op Sıfır Bellek Tahsisi", limoni.Fg(limoni.Hex("#888888")))),
            widgets.SymbolsSingle,
            limoni.Fg(limoni.Hex("#666666")),
        )),
    )

    // Kare alanına doğrudan render et:
    f.RenderComponent(view, f.Area())
}
```

