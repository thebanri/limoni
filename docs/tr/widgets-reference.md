# 🧩 Zengin Bileşenler Referansı & API Rehberi

Limoni, modern terminal kullanıcı arayüzleri geliştirmek için yüksek performanslı ve zengin bir widget kütüphanesi sunar. Tüm bileşenler `widgets.Widget` arayüzünü (`Draw` ve `SizeHint`) uygular.

Hem kök paket (`github.com/thebanri/limoni`) üzerinden akıcı (fluent) yapıcılarla hem de alt paket (`github.com/thebanri/limoni/widgets`) üzerinden doğrudan struct olarak kullanılabilirler.

---

## 1. Kapsayıcı & Metin Bileşenleri

### `Block`
Köşe yuvarlama (`SymbolsRounded`), çift çizgi, kalın çizgiler, başlıklar, iç boşluk (padding) ve iç içe yerleşim sağlayan temel kapsayıcı.

#### Akıcı (Fluent) Kullanım
```go
block := limoni.NewBlock().
    WithTitle(" 📦 SUNUCU TELEMETRİSİ ").
    WithTitleAlign(limoni.AlignCenter).
    Rounded().                                  // Yuvarlatılmış köşeler (╭ ╮ ╰ ╯)
    WithBorderStyle(limoni.Fg(limoni.Hex("#00FFAA"))).
    WithPadding(1, 2, 1, 2).                    // Üst, Sağ, Alt, Sol
    WithChild(childWidget)

f.RenderWidget(block, area)
inner := block.Inner(area)                      // Kenarlık ve padding sonrası iç çizim alanı
```

#### Kenarlık Seçenekleri
* `Rounded()`: Yuvarlatılmış köşeli (`╭─╮ │ ╰─╯`)
* `Single()`: İnce tek çizgili (`┌─┐ │ └─┘`)
* `Double()`: Çift çizgili (`╔═╗ ║ ╚═╝`)
* `Thick()`: Kalın çizgili (`┏━┓ ┃ ┗━┛`)
* `BlockBorder()`: Dolu blok karakterli (`███ █ ███`)

---

### `Label`
Modal diyaloglar, durum çubukları, etiketler ve metin satırları için geliştirilmiş sıfır ek yüklü, ultra hafif metin bileşeni.

**v0.2.4+** sürümünde `Label`, garantili tam arkaplan dolgusu sunar: Widget'a doğrudan veya üst `cell.Context` üzerinden bir arkaplan rengi verildiğinde, satır sonlarındaki boşluklar ve boş satırlar dahil tüm alan bu renkle doldurulur. Böylece alttaki terminal yazılarının görünmesi engellenir. Tam `SizeHint` ve `Measure` yerleşim müzakeresi desteği sunar.

```go
lbl := widgets.NewLabel("OBJ modelini dışa aktarmak için [Ctrl+S]").
    WithStyle(cell.Style{
        Fg: cell.ColorYellow,
        Bg: cell.NewColorRGB(30, 32, 48),
    })

f.RenderWidget(lbl, area)

// Veya Lego Composable bileşeni olarak:
badge := limoni.HStack(
    limoni.Label("Durum: ", limoni.Fg(limoni.ColorGray)),
    limoni.Label("HAZIR", limoni.Fg(limoni.ColorGreen).Bold()),
)
```

---

### `Paragraph`
Otomatik kelime kaydırma (word wrap), hizalama ve stil desteği sunan metin bloğu. **v0.2.4+** sürümünde arkaplan rengi atandığında tüm sınırlayıcı kutu doldurularak alttaki katmanların aradan sızması önlenir.

```go
p := limoni.NewParagraph("Limoni sıfır bellek tahsisiyle 60-240 FPS akıcı hız sunar.").
    WithWrap(true).
    WithStyle(cell.Style{
        Fg: cell.ColorWhite,
        Bg: cell.NewColorRGB(20, 24, 38),
    }).
    WithAlignment(limoni.AlignCenter)

f.RenderWidget(p, area)
```

---

### `Markdown`
Başlıklar (`#`, `##`), madde işaretli listeler (`-`, `*`), yatay ayırıcılar (`---`), kalın (`**`), italik (`*`) ve satır içi kod (` ` `) destekleyen interaktif Markdown okuyucu. Fare tekerleği ve sürükleme ile kaydırma destekler.

```go
md := limoni.NewMarkdown(icerik).
    WithID("doc_viewer").
    WithStyle(limoni.Fg(limoni.RGB(220, 225, 235))).
    WithFocusedStyle(limoni.Fg(limoni.Hex("#00E5FF"))).
    WithScrollOffset(&scrollOffset)

f.RenderWidget(md, area)
```

---

## 2. Listeler & Veri Tabloları

### `Table`
Sütun kısıtlamaları, satır seçimi, zebra çizgileri ve ızgara desteği sunan tablo bileşeni.

```go
table := limoni.NewTable().
    WithHeaders("PID", "PROSES", "CPU %", "BELLEK").
    WithRow("1024", "nginx", "4.2%", "42 MB").
    WithRow("2048", "postgres", "12.8%", "256 MB").
    WithConstraints(
        limoni.Fixed(8),
        limoni.Fill(),
        limoni.Fixed(10),
        limoni.Fixed(12),
    ).
    WithGrid(true).
    WithSelected(seciliIndex)

f.RenderWidget(table, area)
```

---

### `VirtualDataView`
Bellek şişmesi olmadan **1.000.000+ satırlık** devasa veri setlerini sıfır GC yüküyle kaydırabilen sanal tablo.

```go
view := widgets.VirtualDataView{
    ID:            "infinite_log_view",
    Source:        logDataSource, // widgets.VirtualDataSource arayüzünü uygular
    Prefetch:      20,
    Offset:        &scrollOffset,
    Style:         cell.Style{Fg: cell.NewColorRGB(190, 195, 205)},
    SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 80, 130)},
}
f.RenderWidget(view, area)
```

---

### `List`
Özelleştirilebilir vurgulama sembolü ve akıcı zincirleme sunan liste bileşeni:

```go
list := limoni.NewList("Genel Bakış", "Metrikler", "Ayarlar", "Kayıtlar").
    WithHighlightSymbol("👉 ").
    WithSelected(seciliIndex).
    WithSelectedStyle(limoni.Fg(limoni.Hex("#00FFAA")).Bold())

f.RenderWidget(list, area)
```

---

## 3. Göstergeler, Girdi Kontrolleri & Formlar

### `Gauge` & `LineGauge`
`Gauge` alanını `Ratio` oranında doldurur; dolgunun kenarı hücrenin sekizde biri adımlarla ilerler, ortadaki etiket dolgunun üstünde ters renge döner. `LineGauge` tek satırdır: bir etiket, ardından orana kadar kalın, sonrası ince bir çizgi.

```go
f.RenderWidget(widgets.Gauge{Ratio: 0.42}, alan)
f.RenderWidget(widgets.LineGauge{Ratio: 0.42, Label: "Yükleme"}, satir) // Yükleme ━━━━━━━━──────────
```

---

### `Autocomplete`
Yazdıkça daralan, bulanık (fuzzy) eşleşen öneri listeli metin kutusu ("gco" → "git checkout"). Eşleştirme `Draw`'da değil `HandleKey`'de yapılır.

```go
durum := &widgets.AutocompleteState{}
durum.HandleKey(ev.Key, komutlar) // ↑/↓ gezinir, Tab ya da Enter kabul eder, Esc kapatır
f.RenderWidget(widgets.Autocomplete{ID: "komut", Suggestions: komutlar, State: durum}, cell.NewRect(x, y, 40, 7))
```

---

### `FilePicker`
Dizin gezgini: `..`, önce sonunda `/` olan dizinler, sağda boyutlar. Enter dizine girer ya da dosyayı seçer (`State.Chosen`), Backspace bir üst dizine çıkar, Ctrl+H gizli dosyaları gösterir. `Extensions` ve `DirsOnly` ile süzülür.

```go
secici := widgets.NewFilePickerState(".")
f.RenderWidget(widgets.FilePicker{ID: "dosyalar", State: secici}, alan)
```

---

### `Calendar`
Bir ayı ızgara olarak gösterir (20×8 hücre). Oklar seçimi bir gün ya da bir hafta, Page Up/Down bir ay taşır; tıklama seçer. `FirstWeekday`, `MonthNames` ve `WeekdayNames` ile yerelleştirilir; `DayStyle` etkinlikli günleri işaretler. Saati kendisi okumaz: `Today` verin.

```go
aylar := [12]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}
gunler := [7]string{"Pz", "Pt", "Sa", "Ça", "Pe", "Cu", "Ct"}
f.RenderWidget(widgets.Calendar{State: takvim, Today: time.Now(), FirstWeekday: time.Monday, MonthNames: &aylar, WeekdayNames: &gunler}, alan)
```

---

### `TextInput`
İmleç takibi, metin seçimi ve şifre maskeleme desteği sunan tek satırlık metin giriş kutusu.

```go
input := limoni.NewTextInput("api_key_input").
    WithPlaceholder("Gizli token giriniz...").
    WithStyle(limoni.Fg(limoni.RGB(220, 225, 235))).
    WithFocusedStyle(limoni.Fg(limoni.Hex("#00E5FF")).Bold())

f.RenderWidget(input, area)
```

---

### `Checkbox` & `RadioButton`
Onay kutuları ve grup bazlı radyo butonları:

```go
cb := widgets.Checkbox{
    ID:      "telemetry_cb",
    Label:   "Telemetri verilerini gönder",
    Checked: isChecked,
    OnToggle: func(val bool) { isChecked = val },
}
f.RenderWidget(cb, area)
```

---

### `Slider`
Sürükle-bırak destekli ses, parlaklık veya oran kaydırıcısı:

```go
slider := widgets.Slider{
    ID:          "volume_slider",
    Min:         0,
    Max:         100,
    State:       sliderState,
    FilledStyle: cell.Style{Fg: cell.NewColorRGB(80, 220, 140)},
    ThumbStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
}
f.RenderWidget(slider, area)
```

---

## 4. Düzen, Metin & Çerçeve

### `SplitPane`
İki widget'ı yan yana (`SplitHorizontal`) ya da üst üste (`SplitVertical`) gösterir; aradaki bölücü fareyle sürüklenir, ok tuşlarıyla taşınır (`SplitState.HandleKey`). İmleç şeklini değiştirebilen terminallerde bölücünün üstünde yeniden boyutlandırma oku görünür.

```go
f.RenderWidget(widgets.SplitPane{First: agac, Second: onizleme, State: bolme, MinFirst: 20}, alan)
```

---

### `StatusBar`
Tam ekran uygulamaların son satırı: solda, ortada ve sağda öğeler; her biri isteğe bağlı bir tuş ve metin. Yer yetmezse sol grup kalır, sağ grup soldan kırpılır, orta grup çıkarılır.

```go
f.RenderWidget(widgets.StatusBar{
    Left:  []widgets.StatusItem{{Text: "NORMAL"}},
    Right: []widgets.StatusItem{{Key: "^Q", Text: "çık"}, {Text: "12:04"}},
}, altSatir)
```

---

### `CodeView`
Satır numaralı, sözdizimi renklendirmeli kod görünümü: Go, Python, JavaScript/TypeScript, Rust, C ailesi, shell, JSON ve YAML — bağımlılıksız küçük bir lexer. Renklendirme `SetSource`'ta bir kez yapılır; `Draw` yalnızca boyar.

```go
kod := widgets.NewCodeViewState(kaynak, widgets.LanguageForFile("main.go"))
f.RenderWidget(widgets.CodeView{State: kod}, alan)
```

---

### `BigText`
Kamu malı font8x8 bitmap fontuyla büyük harfler: karakter başına 8×8, 8×4 (yarım yükseklik) ya da 4×4 (kadran) hücre. Başlıklar, saatler ve sayaçlar için. ğ, ş, ı gibi Latin-1 dışı Türkçe harfler temel harflerine düşer.

```go
f.RenderWidget(widgets.BigText{Text: "12:04", Size: widgets.BigTextHalfHeight, Alignment: widgets.AlignCenter}, alan)
```

---

## 5. Modallar, Dialoglar ve Katmanlar

### `Dialog`
Işıltılı degrade kenarlıklar, gölge, taşınabilir başlık çubuğu ve klavye odak koruması sunan modern pencereler:

```go
dialog := widgets.Dialog{
    ID:         "exit_dialog",
    Title:      " ⚠️ ÇIKIŞI ONAYLA ",
    Message:    "Uygulamadan çıkmak istediğinize emin misiniz?",
    SubMessage: "Kaydedilmemiş tüm değişiklikler kaybolacaktır.",
    Shadow:     true,
    Buttons: []widgets.DialogButton{
        {
            Text: "Vazgeç",
            Handler: func() { closeDialog() },
        },
        {
            Text: "Çıkış Yap",
            Handler: func() { os.Exit(0) },
        },
    },
    ButtonStyle:        cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(45, 45, 45)},
    ButtonFocusedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(80, 220, 140), Modifier: cell.ModifierBold},
}

f.BeginFocusScope("exit_dialog")
f.RenderWidget(dialog, modalArea)
```

---

## 6. Grafik & 3D Rasterizer

### `Canvas`
Hücre başına $2 \times 4$ alt-piksel noktayla vektörel çizim. `Marker` noktaların nasıl çizileceğini seçer: Braille (varsayılan), sekstant ($2 \times 3$), kadran ($2 \times 2$), yarım blok ya da tam blok. `LineChart` ve `PieChart` aynı `Marker` alanını alır.

```go
cv := widgets.NewCanvas(width, height)
cv.DrawLine(0, 0, 100, 50, cell.Style{Fg: cell.NewColorRGB(0, 255, 180)})
cv.DrawCircle(50, 25, 20, cell.Style{Fg: cell.NewColorRGB(255, 200, 0)})
f.RenderWidget(cv, area)
```

---

### `Viewer3D`
Yazılımsal 3D rasterleştirici: OBJ, STL, PLY ve GLB modelleri (`graphics.LoadOBJ` vb.) ya da hazır primitifler. Near-plane kırpma, her modda derinlik tamponu; düz, Lambert, Gouraud (pürüzsüz köşe normalleri) ve doku gölgelendirme.

```go
v3d := &widgets.Viewer3D{
    Model:   graphics.NewTorus(1, 0.4, 32, 16),
    Shading: widgets.ShadingGouraud, // ShadingFlat, ShadingLambert, ShadingTexture, ShadingWireframe
    RotX:    35,
    RotY:    aci,
}
f.RenderWidget(v3d, alan)
```

Varsayılan olarak Braille noktalarıyla çizer; `Marker: widgets.MarkerSextant` ya da `MarkerQuadrant`, Braille içermeyen fontlar için dolu bloklarla çizer. `Pixels: true` modeli bir resim olarak çizip terminalin görüntü protokolüyle (kitty, iTerm2, Sixel) gönderir; protokol yoksa noktalara döner. Duran model bir kez kodlanır; hareket eden model her karede kodlanır — ölçüldüğü makinede kitty ile 60×24'lük bir alan için yaklaşık 7 ms ve 2.7 MB — bu yüzden isteğe bağlıdır.
