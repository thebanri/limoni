# 🧩 Zengin Bileşenler Referansı & API Rehberi

Limoni, modern terminal kullanıcı arayüzleri geliştirmek için yüksek performanslı ve zengin bir widget kütüphanesi sunar. Tüm bileşenler `widgets.Widget` arayüzünü (`Draw` ve `SizeHint`) uygular.

---

## 1. Kapsayıcı & Yapısal Bileşenler

### `widgets.Block`
Köşe yuvarlama (`SymbolsRounded`), çift çizgi sınırları, başlıklar, iç boşluk (padding) ve iç içe yerleşim sağlayan temel kapsayıcı.

```go
block := widgets.Block{
    Title:          " 📦 SUNUCU TELEMETRİSİ ",
    TitleAlignment: widgets.AlignCenter,
    Borders:        widgets.BorderAll,
    BorderSymbols:  widgets.SymbolsRounded,
    BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 210, 255)},
    TitleStyle:     cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
    PaddingLeft:    1,
    PaddingRight:   1,
}
frame.RenderWidget(block, area)
```

---

## 2. Gelişmiş Üretkenlik Widget'ları

### `widgets.TreeView`
Klasör simgeli, kılavuz çizgili (`│  ├─  └─`), klavye ve fare tıklaması destekli hiyerarşik ağaç bileşeni.

```go
state := widgets.NewTreeViewState()

tree := widgets.TreeView{
    ID: "project_tree",
    Roots: []widgets.TreeNode{
        {
            ID: "src", Label: "src", Icon: "📁", Expanded: true,
            Children: []widgets.TreeNode{
                {ID: "main.go", Label: "main.go", Icon: "📄"},
                {ID: "config.json", Label: "config.json", Icon: "⚙️"},
            },
        },
    },
    State:      state,
    ShowGuides: true,
}
frame.RenderWidget(tree, area)
```

### `widgets.ColorPicker`
Renk paletleri, RGB kaydırıcıları, Hex girişi ve canlı önizlemeli interaktif renk seçici.

```go
state := widgets.NewColorPickerState(0, 200, 255)

picker := widgets.ColorPicker{
    ID:          "theme_picker",
    State:       state,
    ShowPreview: true,
}
frame.RenderWidget(picker, area)
```

### `widgets.ToastManager`
Bilgi, başarı, uyarı ve hata bildirimlerini zaman ayarlı ve gölgeli olarak ekranda gösteren bildirim yöneticisi.

```go
toastMgr := widgets.NewToastManager(widgets.ToastTopRight)
toastMgr.Success("Veritabanı Bağlandı", "Gecikme: 2ms")

// Render döngüsünde:
toastMgr.Update(time.Now())
toastMgr.Draw(ctx, buf)
```

---

## 3. Zengin Veri Görselleştirme Grafikleri

### `widgets.BarChart`
Dikey ve yatay, otomatik ölçeklenen çubuk grafikler.

```go
chart := widgets.BarChart{
    Data: []widgets.BarData{
        {Label: "Pzt", Value: 35, Color: cell.NewColorRGB(0, 255, 128)},
        {Label: "Sal", Value: 68, Color: cell.NewColorRGB(0, 200, 255)},
        {Label: "Çar", Value: 95, Color: cell.NewColorRGB(255, 100, 50)},
    },
    Direction:  widgets.BarVertical,
    BarWidth:   4,
    BarGap:     2,
    ShowValues: true,
}
frame.RenderWidget(chart, area)
```

### `widgets.LineChart`
Braille alt-pikselli, eksen etiketli ve göstergeli (legend) çoklu seri çizgi grafikler.

```go
lineChart := widgets.LineChart{
    Datasets: []widgets.LineDataset{
        {
            Name:  "Gelen Trafik",
            Data:  []float64{10, 25, 40, 65, 80, 95},
            Color: cell.NewColorRGB(46, 204, 113),
        },
    },
    ShowAxes:   true,
    ShowLegend: true,
    XLabels:    []string{"00:00", "04:00", "08:00", "12:00"},
}
frame.RenderWidget(lineChart, area)
```

### `widgets.PieChart`
Pasta ve halka (donut) grafikler, yüzdelik oran hesaplamaları ve renkli göstergeler.

```go
pie := widgets.PieChart{
    Data: []widgets.PieSlice{
        {Label: "Go", Value: 50, Color: cell.NewColorRGB(0, 200, 255)},
        {Label: "Rust", Value: 30, Color: cell.NewColorRGB(255, 100, 50)},
        {Label: "TS", Value: 20, Color: cell.NewColorRGB(50, 150, 255)},
    },
    DonutHoleRatio:  0.4,
    ShowLegend:      true,
    ShowPercentages: true,
}
frame.RenderWidget(pie, area)
```

---

## 4. Geliştirici Araçları & Canlı Denetleyici

### `widgets.DevTools` (`F12`)
Canlı FPS, render süresi (frametime), heap bellek kullanımı, GC sayaçları ve odak zincirini gösteren F12 hata ayıklama paneli.

```go
devState := widgets.NewDevToolsState()

// Olay dinleyicide:
if ev.Type == backend.KeyF12 {
    devState.Toggle()
}

// Render döngüsünde:
devState.RecordFrame(time.Since(frameStart))
if devState.Enabled {
    widgets.DevTools{State: devState}.Draw(ctx, buf)
}
```
