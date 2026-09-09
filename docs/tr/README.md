# 🍋 Limoni Türkçe Dokümantasyon İndeksi

**Limoni**, Go dili için geliştirilmiş ultra hızlı, sıfır bellek tahsisatlı (zero-allocation), iş parçacığı güvenli (thread-safe) ve kurumsal düzeyde modern bir Terminal Kullanıcı Arayüzü (TUI) motorudur.

---

## 📚 Dokümantasyon Konuları

1. **[Hızlı Başlangıç Rehberi](getting-started.md)**: Kurulum, tek import ile uygulama başlatma (`limoni.Run`, `limoni.Start`) ve TEA mimarisi.
2. **[Çekirdek Motor API Referansı](../core-api.md)**: Çekirdek paket yapısı, 1D tampon bellek matrisi, ANSI diff algoritması, Unicode East Asian Width emoji doğruluğu ve donanım imleç senkronizasyonu.
3. **[Zengin Widget Kataloğu](../widgets-reference.md)**: Block, Paragraph, Table, VirtualDataView, Canvas, Viewer3D, TextInput, Markdown, Dialog, Slider ve akıcı (fluent) yapıcılar.
4. **[Yerleşim (Layout) Rehberi](../layout-guide.md)**: FlexLayout, GridLayout, kısıtlamalar (Fixed, Percentage, Ratio, Fill) ve `SplitVertical` / `SplitHorizontal` kısayolları.
5. **[Grafik, Canvas & 3D Motoru](../graphics-and-canvas.md)**: $2 \times 4$ Braille alt-piksel çizim tuvali, yerel resim protokolleri (Kitty, Sixel, iTerm2), derinlik tamponlu (Depth Buffer) 3D Gouraud/Lambert rasterizer ve mesh yükleme.
6. **[Örnek Uygulamalar Rehberi](../examples.md)**: `examples/` dizinindeki showcase, 3D viewer, animasyon, sanal tablo ve form demoları.
7. **[Mimari ve Performans Prensipleri](../architecture.md)**: 1D düz hücre matrisi, sıfır GC duraklaması, ANSI diff algoritması ve iş parçacığı güvenliği.

---

## 🚀 Hızlı Başlangıç

```bash
# Limoni'yi projenize ekleyin
go get github.com/thebanri/limoni
```

```go
package main

import "github.com/thebanri/limoni"

func main() {
	limoni.Start(func(f *limoni.Frame) {
		block := limoni.NewBlock().
			WithTitle(" 🍋 LIMONI TUI ").
			WithTitleAlign(limoni.AlignCenter).
			Rounded().
			WithBorderStyle(limoni.Fg(limoni.Hex("#00FFAA"))).
			WithChild(limoni.NewParagraph("Tek import ile modern ve ultra hızlı TUI!").
				WithStyle(limoni.Fg(limoni.RGB(220, 225, 235)).Bold()))

		f.RenderWidget(block, f.Area())
	})
}
```
