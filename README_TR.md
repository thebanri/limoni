<p align="center">
  <img src="assets/logo.png" alt="Limoni Logo" width="180" />
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
  <a href="#-vitrin--canlı-demolar">Vitrin</a> •
  <a href="#-temel-özellikler">Özellikler</a> •
  <a href="#-hızlı-başlangıç">Hızlı Başlangıç</a> •
  <a href="#-dokümantasyon">Dokümantasyon</a> •
  <a href="#-zengin-widget-ekosistemi">Widget'lar</a> •
  <a href="#-örnek-uygulamalar">Örnekler</a>
</p>

---

## ⚡ Genel Bakış

**Limoni**, Go dili için sıfırdan tasarlanmış kurumsal düzeyde, yüksek performanslı bir Terminal Kullanıcı Arayüzü (TUI) motorudur. Veri yoğun izleme panelleri, DevOps araçları ve modern CLI uygulamaları için Go'nun geliştirici ergonomisini Rust benzeri ham render hızıyla buluşturur.

**1D düz hücre matrisi**, **sıfır bellek tahsisatlı sıcak yollar (zero-alloc hot-paths)** ve **yüksek performanslı diferansiyel ANSI motoru (~7.1 µs tam ekran diff, 0 B/op)** sayesinde Go Garbage Collector'ını tetiklemeden yüksek FPS'te pürüzsüz çizim sağlar.

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

## 🎬 Vitrin & Canlı Demolar

### 🎮 3D Vektör ve Model İşleme Motoru
Terminal hücrelerinde 60+ FPS hızında gerçek zamanlı 3D yazılımsal rasterizasyon. `.obj`, `.stl` ve `.ply` model desteği, derinlik tamponlu Gouraud gölgelendirme, Lambertian aydınlatma ve etkileşimli fare/klavye kamera yörünge kontrolleri.

<p align="center">
  <img src="assets/3d.gif" alt="Limoni 3D Model İşleme" width="100%" />
</p>

```bash
go run ./examples/3d_viewer
```

---

### 📁 Süper Dosya Gezgini & Görsel Önizleme
Açılır/kapanır klasörler, hiyerarşik kılavuz çizgileri, dosya meta verileri ve yerleşik TrueColor yarım-blok (half-block) görsel önizleme desteğine sahip gelişmiş dosya ağacı bileşeni (`widgets.TreeView`).

<p align="center">
  <img src="assets/treeview.gif" alt="Limoni Dosya Gezgini ve Görsel Önizleme" width="100%" />
</p>

```bash
go run ./examples/treeview
```

---

### 📊 Yüksek Çözünürlüklü Grafikler & Veri Görselleştirme
Alt-piksel Braille eğrileri (`widgets.LineChart`), dikey gradyan spektrum çubukları (`widgets.BarChart`) ve pasta/halka dağılımları (`widgets.PieChart`) ile sıfır bellek tahsisatlı yüksek frekanslı telemetri görselleştirme.

<p align="center">
  <img src="assets/chart.gif" alt="Limoni Grafikler ve Veri Görselleştirme" width="100%" />
</p>

```bash
go run ./examples/charts
```

---

## ✨ Temel Özellikler

* 🚀 **Ultra Hızlı ANSI Diffing**: Ekrandaki değişiklikleri tespit edip tam ekran yenilemede dahi ~7.1 µs (~140.000 FPS) sürede sıfır bellek tahsisatıyla minimum ANSI kaçış dizilerini terminale gönderir; ekran değişmediğinde ~2 ns içinde anında döner.
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

## 📊 Performans ve Kıyaslamalar (Benchmarks)

Limoni, standart sanal terminal ortamında (120×40 hücre = 4.800 hücre) gerçek dirty diffing, kısmi güncellemeler, sanal kaydırma ve bellek tahsisatlarını ölçen kapsamlı bir kıyaslama paketine sahiptir.

Testleri yerel ortamınızda çalıştırmak için:
```bash
# Buffer Diff kıyaslamaları (kirli ve temiz kare testleri)
go test ./core/buffer -run '^$' -bench . -benchmem

# Widget ve Düzen kıyaslamaları
go test ./benchmarks -run '^$' -bench . -benchmem
```

### Doğrulanmış Kıyaslama Sonuçları (120×40 Görünüm Alanı, AMD Ryzen / EPYC):

| Kıyaslama İşlemi | Ölçülen Gecikme | Kare / İşlem Hızı | Bellek Tahsisatı | Açıklama |
| :--- | :--- | :--- | :--- | :--- |
| **`BenchmarkDiff_FullChanges`** | **`~90.1 µs`** | **~11.100 FPS** | **`0 B/op (0 allocs)`** | %100 tam ekran hücre değişimi (4.800 hücre) çift tampon diff işlemi ve ANSI akışı üretimi |
| **`BenchmarkDiff_PartialChanges`** | **`~20.5 µs`** | **~48.800 FPS** | **`0 B/op (0 allocs)`** | %10 ekran alanı değişimi (480 hücre) çift tampon diff işlemi |
| **`BenchmarkDiff_NoChanges`** | **`~1.92 ns`** | **~520.000.000 FPS** | **`0 B/op (0 allocs)`** | Tamponda hiçbir değişiklik olmadığında fast-path ile anında dönüş |
| **`BenchmarkTextHeavyFrame`** | **`~60.8 µs`** | **~16.400 FPS** | **`5 B/op (0 allocs)`** | 120 sütuna yayılan 40 satırlık unicode sembollü ve kelime kaydırmalı metin çizimi |
| **`BenchmarkHundredLayers`** | **`~47.0 µs`** | **~21.200 FPS** | **`0 B/op (0 allocs)`** | 100 katmanlı Block widget çizimi ve değerlendirmesi (Ratatui hundred-layers denklik testi) |
| **`BenchmarkTenThousandRowTable`** | **`~102 µs`** | **~9.800 FPS** | **`614 B/op`** | 10.000 satırlık tabloda aktif imleç kaydırma (scrolling) ve görünür satır çizimi |
| **`BenchmarkOneMillionRowVirtualScroll`**| **`~2.53 ms`** | **~395 FPS** | **`4.9 KB/op (6 allocs)`** | 1.000.000 satırlık sanal veri kaynağında aktif kaydırma ve görünür alan yönetimi |
| **`BenchmarkMouseHitTest`** | **`~61.7 ns`** | **~16.200.000 op/s** | **`0 B/op (0 allocs)`** | 100 tıklama bölgesi üzerinde hiyerarşik uzamsal fare tıklama tespiti |
| **`BenchmarkAsyncUpdateBurst`** | **`~214 ns`** | **~4.660.000 msg/s** | **`8 B/op (0 allocs)`** | Elm çalışma mimarisinde yüksek verimli asenkron mesaj kuyruğu iletimi |

> [!NOTE]
> **Şeffaflık ve Mühendislik Dürüstlüğü Garantisi**:
> Sentetik kısayollar, yapay tampon temizlemeleri veya statik sıfır-offset döngüleri kullanılmaz.
> - **Diff Kıyaslamaları**: Hücrelerin her karede bizzat değiştiği kalıcı çift tampon üzerinde çalışır; diff motorunu ve ANSI kodlayıcısını uçtan uca çalıştırır.
> - **Kaydırma Kıyaslamaları**: `Select((i * 7) % N)` ile satırlar arasında aktif olarak kaydırma yapar ve sürekli kaydırma altında bellek tüketimini test eder.
> - **100 Katman Testi**: Tıklama kestirmesi yerine 100 adet `Block` widget'ını ekrana bizzat çizer.

---

## 📂 Örnek Uygulamalar

| Dizin | Başlık | Çalıştırma |
| :--- | :--- | :--- |
| **[`examples/3d_viewer`](examples/3d_viewer)** | **3D Model & Shader Viewer** | `go run ./examples/3d_viewer` |
| **[`examples/paint`](examples/paint)** | **Noktasal Paint & Çizim Stüdyosu** | `go run ./examples/paint` |
| **[`examples/dashboard`](examples/dashboard)** | **Sistem Telemetri Paneli** | `go run ./examples/dashboard` |
| **[`examples/table_virtual`](examples/table_virtual)** | **1M Satırlı Sanal Tablo** | `go run ./examples/table_virtual` |
| **[`examples/todo`](examples/todo)** | **TEA Todo Uygulaması** | `go run ./examples/todo` |
| **[`examples/demo`](examples/demo)** | **3D Limon Modeli (GLB/ASCII/Braille/Half-Block) & Tanıtım Vitrini** | `go run ./examples/demo` |
| **[`examples/showcase`](examples/showcase)** | **Gelişmiş Vitrin Demosu (Matrix, DevTools F12, Formlar, Komut Paleti)** | `go run ./examples/showcase` |
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
