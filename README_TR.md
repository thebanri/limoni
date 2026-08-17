<p align="center">
  <img src="docs/assets/showcase.png" alt="Limoni Vitrini" width="100%" />
</p>

<h1 align="center">🍋 Limoni</h1>

<p align="center">
  <strong>Go Dili İçin Ultra Hızlı, Sıfır Bellek Tahsisatlı (Zero-Alloc), İş Parçacığı Güvenli Modern TUI Motoru.</strong>
</p>

<p align="center">
  <a href="https://github.com/thebanri/limoni/actions"><img src="https://img.shields.io/github/actions/workflow/status/thebanri/limoni/ci.yml?branch=main&style=flat-square&logo=github" alt="Derleme Durumu"></a>
  <a href="https://pkg.go.dev/github.com/thebanri/limoni"><img src="https://img.shields.io/badge/go.dev-referans-007d9c?style=flat-square&logo=go&logoColor=white" alt="Go.Dev Referans"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-%3E%3D%201.22-blue?style=flat-square&logo=go" alt="Go Sürümü"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/lisans-MIT-emerald?style=flat-square" alt="Lisans"></a>
  <a href="#-performans-ve-kıyaslamalar"><img src="https://img.shields.io/badge/tahsisat-0_B%2Fop-brightgreen?style=flat-square" alt="Sıfır Bellek Tahsisatı"></a>
</p>

<p align="center">
  <strong>Dil / Language:</strong>
  <a href="README.md">English</a> •
  <a href="README_TR.md">Türkçe</a>
</p>

<p align="center">
  <a href="#-neden-limoni">Neden Limoni?</a> •
  <a href="#-temel-özellikler">Özellikler</a> •
  <a href="#-hızlı-başlangıç">Hızlı Başlangıç</a> •
  <a href="#-dokümantasyon">Dokümantasyon</a> •
  <a href="#-zengin-widget-ekosistemi">Widget'lar</a> •
  <a href="#-örnek-uygulamalar">Örnekler</a>
</p>

---

## ⚡ Genel Bakış

**Limoni**, Go dili için sıfırdan tasarlanmış kurumsal düzeyde, yüksek performanslı bir Terminal Kullanıcı Arayüzü (TUI) motorudur. Veri yoğun izleme panelleri, DevOps araçları ve modern CLI uygulamaları için Go'nun geliştirici ergonomisini Rust benzeri ham render hızıyla buluşturur.

**1D düz hücre matrisi**, **sıfır bellek tahsisatlı sıcak yollar (zero-alloc hot-paths)** ve **mikrosaniye altı diferansiyel ANSI motoru** sayesinde Go Garbage Collector'ını tetiklemeden 60+ FPS pürüzsüz çizim sağlar.

---

## 💡 Neden Limoni?

| Özellik / Hedef | 🍋 Limoni (Go) | 🫧 Bubble Tea (Go) | 🐀 Ratatui (Rust) |
| :--- | :--- | :--- | :--- |
| **Dil ve Araçlar** | **Go (Yerel)** | Go (Yerel) | Rust (Yerel) |
| **Render Mimarisi** | **1D Düz Matris + ANSI Diff** | String birleştirme / TEA | Çift Tamponlu Immediate Mode |
| **Kritik Yol Tahsisatı**| **`0 B/op` (Sıfır Alloc)** | Yüksek heap tahsisatı | Stack / RAII |
| **Büyük Veri / Tablolar**| **1M+ Satır Sanallaştırma** | Yüksek GC yükü | Yüksek layout klonlama yükü |
| **3D & Vektör Grafikleri**| **Dahili 3D (OBJ/STL/PLY) & Shaders** | Harici eklenti gerekir | Eklenti gerekir |
| **Erişilebilirlik (A11y)** | **Dahili Semantik Ağaç ve Ekran Okuyucu** | Kısıtlı / Manuel | Deneysel |
| **Eşzamanlılık (Concurrency)** | **Kilit-Serbest Kanallar / İş Parçacığı Güvenli** | Tek iş parçacıklı TEA | Manuel iş parçacığı yönetimi |

---

## ✨ Temel Özellikler

* 🚀 **Mikrosaniye Altı ANSI Diffing**: Sadece değişen hücreleri tespit eder ve terminale en kısa ANSI kaçış dizilerini gönderir.
* 📦 **1D Düz Tampon (Flat Buffer)**: Bellek parçalanmasını önler ve CPU L1/L2 önbellek erişimini maksimize eder.
* 🎨 **24-Bit TrueColor & Otomatik Geri Dönüş**: TrueColor desteği olmayan terminallerde otomatik 256 ve 16 renk dönüşümü.
* 📐 **Esnek Flexbox & Grid Düzeni**: Proportional, Fixed, Min/Max, GridArea ve boyut pazarlığı (negotiation) desteği.
* 🎬 **60 FPS Animasyon & Fizik Motoru**: Yay (spring) fizikleri, renk enterpolasyonu ve akıcı easing eğrileri.
* 🕶️ **Dahili 3D & Vektör Grafik Motoru**: `.obj`, `.stl`, `.ply` 3D modelleri Gouraud/Lambertian gölgelendirme ile doğrudan terminalde işleme.
* ♿ **Dahili Erişilebilirlik (A11y)**: Ekran okuyucular için semantik gezinme ağacı ve satır satır denetim modu.

---

## 🚀 Hızlı Başlangıç

### Kurulum

```bash
go get github.com/thebanri/limoni
```

### Örnek Uygulama:

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

	t, err := terminal.New(b)
	if err != nil {
		panic(err)
	}

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

---

## 📂 Örnek Uygulamalar

| Dizin | Başlık | Çalıştırma |
| :--- | :--- | :--- |
| **[`examples/3d_viewer`](examples/3d_viewer)** | **3D Model & Shader Viewer** | `go run ./examples/3d_viewer` |
| **[`examples/paint`](examples/paint)** | **Noktasal Paint & Çizim Stüdyosu** | `go run ./examples/paint` |
| **[`examples/dashboard`](examples/dashboard)** | **Sistem Telemetri Paneli** | `go run ./examples/dashboard` |
| **[`examples/table_virtual`](examples/table_virtual)** | **1M Satırlı Sanal Tablo** | `go run ./examples/table_virtual` |
| **[`examples/todo`](examples/todo)** | **TEA Todo Uygulaması** | `go run ./examples/todo` |
| **[`examples/demo`](examples/demo)** | **Kapsamlı Vitrin Demosu** | `go run ./examples/demo` |
| **[`examples/forms`](examples/forms)** | **Form & Girdi Kontrolleri** | `go run ./examples/forms` |
| **[`examples/layer_demo`](examples/layer_demo)** | **Katman & Modal Demosu** | `go run ./examples/layer_demo` |

---

## 📚 Türkçe Dokümantasyon

Detaylı Türkçe rehberler için [docs/tr/ dizinine](docs/tr/README.md) göz atabilirsiniz:
- [Hızlı Başlangıç Rehberi](docs/tr/getting-started.md)
- [Çekirdek Motor API Referansı](docs/tr/core-api.md)
- [Widget Kataloğu & Kullanım Kılavuzu](docs/tr/widgets-reference.md)
- [Örnek Uygulamalar Rehberi](docs/tr/examples.md)

---

## 📄 Lisans

Bu proje **MIT Lisansı** altında lisanslanmıştır.
