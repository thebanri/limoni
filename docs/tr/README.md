# 🍋 Limoni Türkçe Dokümantasyon İndeksi

**Limoni**, Go dili için geliştirilmiş ultra hızlı, sıfır bellek tahsisatlı (zero-allocation), iş parçacığı güvenli (thread-safe) ve kurumsal düzeyde modern bir Terminal Kullanıcı Arayüzü (TUI) motorudur.

---

## 📚 Dokümantasyon Konuları

1. **[Hızlı Başlangıç Rehberi](getting-started.md)**: Kurulum, temel kavramlar, ilk TUI uygulamasını oluşturma ve TEA mimarisi.
2. **[Çekirdek Motor API Referansı](core-api.md)**: `core/terminal`, `core/buffer`, `core/cell`, `core/backend`, `core/runtime` paketleri, katman (Layer) ve odak (Focus) sistemi.
3. **[Zengin Widget Kataloğu](widgets-reference.md)**: Block, Table, VirtualDataView, Canvas, Viewer3D, TextInput, Select, Slider, Tabs ve diğer bileşenler.
4. **[Yerleşim (Layout) Rehberi](layout-guide.md)**: FlexLayout, GridLayout, kısıtlamalar (Fixed, Percentage, Ratio, Fill) ve boyut pazarlığı (negotiate).
5. **[Grafik, Canvas & 3D Motoru](graphics-and-canvas.md)**: $2 \times 4$ Braille alt-piksel çizim tuvali, derinlik tamponlu (Depth Buffer) 3D Gouraud/Lambert rasterizer ve mesh yükleme.
6. **[Örnek Uygulamalar Rehberi](examples.md)**: `examples/` dizinindeki 12 bağımsız örneğin (Dashboard, 3D Viewer, Paint Studio, 1M Virtual Table vb.) tanıtımı ve kısayolları.
7. **[Mimari ve Performans Prensipleri](architecture.md)**: 1D düz hücre matrisi, sıfır GC duraklaması, ANSI diff algoritması ve iş parçacığı güvenliği.

---

## 🚀 Hızlı Başlangıç

```bash
# Limoni'yi projenize ekleyin
go get github.com/thebanri/limoni
```

```go
package main

import (
	"os"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	b.Setup()
	defer b.Close()

	t, _ := terminal.New(b)
	b.StartEventLoop()

	t.Draw(func(f *terminal.Frame) {
		f.RenderWidget(widgets.Block{
			Title:         " 🍋 LIMONI TUI ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 210, 255)},
		}, f.Buffer.Area)
	})

	for ev := range b.Events() {
		if ev.Type == backend.EventKey && ev.Key.Type == backend.KeyEsc {
			return
		}
	}
}
```
