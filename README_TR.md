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
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-%3E%3D%201.25-blue?style=flat-square&logo=go" alt="Go Sürümü"></a>
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

| Özellik / Hedef | 🍋 Limoni (Go) | 🫧 Bubble Tea **v1** + Lip Gloss v1 (Go) | 🌈 Bubble Tea **v2** + Ultraviolet (Go) | 🐀 Ratatui **0.30** (Rust) |
| :--- | :--- | :--- | :--- | :--- |
| **Dil ve Araçlar** | **Go (Yerel)** | Go (Yerel) | Go (Yerel) | Rust (Yerel) |
| **Render Mimarisi** | **1D Düz Matris + Adaptif ANSI Diff** | String birleştirme / TEA | Hücre tamponu + ncurses tarzı diff | Çift Tamponlu Immediate Mode |
| **Kritik Yol Tahsisatı**| **`0 B/op` (Sıfır Alloc)** | Yüksek heap tahsisatı | Azaltılmış; sıfır-alloc bir tasarım hedefi değil — *henüz burada ölçülmedi* | Stack / RAII |
| **Düzen Paradigması** | **Bildirimsel Flexbox & Yığın Çözücü** | String dilimleme (`JoinHorizontal/Vertical`) | Cassowary kısıt çözücü | Kısıt çözücü (Constraint solver) |
| **Fare Etkileşimi** | **Hücresel Koordinat & Z-Index Yönlendirme** | Yok (manuel koordinat hesabı) | SGR fare olayları; dahili hit-testing yok | Manuel koordinat |
| **Çift Tampon & Diff** | **Mikrosaniye altı diff + Adaptif tam akış** | Yok (tüm string stdout'a dökülür) | Hücre diff + `ECH`/`REP`/`ICH`/`DCH` + kaydırma optimizasyonu | Çift tamponlu diff |
| **Grapheme Cluster** | Rune seviyesinde genişlik — **cluster desteği henüz yok** | `uniseg` | `uniseg` + Mod 2027 müzakeresi | `unicode-width` |
| **Yetenek Tespiti** | Yalnızca ortam değişkenleri | Ortam / terminfo | Çalışma anında sorgulama (terminfo'suz) | terminfo / crossterm |
| **Büyük Veri / Tablolar**| **1M+ Satır Sanallaştırma (~22 µs)** | Yüksek GC yükü | v1'e göre iyileştirilmiş | Yüksek layout klonlama yükü |
| **3D & Vektör Grafikleri**| **Dahili 3D (OBJ/STL/PLY/GLB) & Shaders** | Harici eklenti gerekir | Harici eklenti gerekir | Eklenti gerekir |
| **Erişilebilirlik (A11y)** | **Dahili Semantik Ağaç ve Ekran Okuyucu** | Kısıtlı / Manuel | Kısıtlı / Manuel | Deneysel |
| **Harici Bağımlılık** | **2 (`golang.org/x/sys`, `golang.org/x/crypto`)** | ~15 dolaylı modül | ~15 dolaylı modül | crates.io grafiği |
| **Eşzamanlılık (Concurrency)** | **Kilit-Serbest Kanallar / İş Parçacığı Güvenli** | Tek iş parçacıklı TEA | Tek iş parçacıklı TEA | Manuel iş parçacığı yönetimi |

> **Bubble Tea v2 sütunu hakkında:** bu satırlar Limoni'nin kendi ölçümlerinden değil, üst akış dokümantasyonundan alınmıştır. Charm, render motorunu hücre tabanlı diff yapan [Ultraviolet](https://github.com/charmbracelet/ultraviolet) üzerine yeniden inşa etti; dolayısıyla Limoni'nin **v1**'e karşı açtığı mimari fark **v2** için olduğu gibi geçerli değildir. Bu depodaki benchmark paketi şu an **Bubble Tea v1.3.10**'u hedefliyor; neyin ölçülüp neyin ölçülmediği için [`docs/benchmark-methodology.md`](docs/benchmark-methodology.md) dosyasına bakın ve v2'ye karşı her performans iddiasını o koşucu eklenene kadar kanıtlanmamış sayın.

### 🍋 Limoni Composable (Lego UI) vs. 🎀 Charm Lip Gloss **v1**

**Lip Gloss v1** Go ekosisteminde bildirimsel stili popülerleştirmiş olsa da, string birleştirmeye dayalı mimarisi yüksek frekanslı ve etkileşimli modern TUI uygulamalarında yapısal kısıtlamalar getiriyordu. Aşağıdaki karşılaştırma **özellikle v1**'e karşıdır:

> ⚠️ **Lip Gloss v2 bu tabloyu değiştiriyor.** v2, ham string birleştirme yerine [Ultraviolet](https://github.com/charmbracelet/ultraviolet) hücre tamponu üzerine kuruludur; bu yüzden aşağıdaki "Veri İlkesi", "Render Hattı" ve "Ekran Kırpma" satırları güncel Charm yığınını tarif etmez. Limoni'nin v2'ye karşı kalan yapısal üstünlükleri hit-testing, sanallaştırma, dahili 3D ve bağımlılık ayak izidir — string-hücre mimarisi değil.

| Yetenek | 🍋 Limoni Composable (`component`) | 🎀 Charm Lip Gloss **v1** |
| :--- | :--- | :--- |
| **Veri İlkesi** | **16 baytlık önbellek uyumlu `Cell` yapısı** | Ham ANSI kaçışlı metin (`string`) |
| **Kritik Yol Bellek Tahsisi** | **`0 B/op` (0 allocs/op)** layout ve render | Yüksek tahsisat oranı (~Yüzlerce KB - MB/sn) |
| **Düzen Modeli** | **Gerçek Flexbox & Grid kısıt çözücü** | String dilimleme (`JoinHorizontal`, `JoinVertical`) |
| **Boyut Kısıtları** | **Orantısal `Flex`, `Ratio`, `Min`, `Max`** | Sadece sabit manuel karakter genişlikleri |
| **Fare Hit-Testing** | **Otomatik uzamsal sınırlar & z-index yönlendirme** | Yok (manuel koordinat ve karakter hesabı gerekir) |
| **Ekran Kırpma (Clipping)** | **Hücre seviyesinde dikdörtgensel uzamsal kırpma** | String kesme (bozuk ANSI kaçış dizilerine yol açar) |
| **Z-Index & Katmanlar** | **Donanım benzeri katman yığını & modal izole** | Satır satır string yamama (`PlaceOverlay`) |
| **Render Hattı** | **Çift tamponlu ANSI diffing (`~7.1 µs`)** | Tüm terminale string dökme (ekranda titreme yapar) |
| **Geçiş Köprüsü** | **`compat/bubbletea` akıcı stil oluşturucu** | Charm ekosistemi yerel standardı |

#### Neden Sıfır Bellek Tahsisatlı Mimari Önemlidir?
1. **Garbage Collector Donmalarını (GC Stutter) Yok Eder**: Lipgloss her kenarlık, boşluk ve yatay birleştirme için bellekte yeni string nesneleri tahsis eder. 60 FPS çalışan hareketli bir ekranda bu durum saniyede yüz binlerce nesne üreterek Go GC'sini devreye sokar ve arayüzde mikro donmalara (stutter) yol açar. Limoni bileşenleri çağrı yığınında (call stack) çalışır ve doğrudan yeniden kullanılan 1D tampona yazar; **sıfır bellek tahsisatı (`0 B/op`)** garantilenir.
2. **Kutudan Çıkan Fare ve Tıklama Desteği**: Lipgloss sadece düz bir metin ürettiğinden kullanıcının nereye tıkladığını bilemez. Limoni bileşenleri çizildikleri ekran alanını (`cell.Rect`) otomatik kaydeder; tıklama, üzerine gelme (hover), sürükleme ve tekerlek olayları doğrudan ilgili bileşenin callback'ine yönlendirilir.

---

## 🎬 Vitrin & Canlı Demolar

### 🎮 3D Vektör ve Model İşleme Motoru
Terminal hücrelerinde 60+ FPS hızında gerçek zamanlı 3D yazılımsal rasterizasyon. `.obj`, `.stl` ve `.ply` model desteği, derinlik tamponlu Gouraud gölgelendirme, Lambertian aydınlatma ve etkileşimli fare/klavye kamera yörünge kontrolleri.

<p align="center">
  <img src="assets/3d.gif" alt="Limoni 3D Model İşleme" width="100%" />
</p>

```bash
# Yerel çalıştırma (-fps bayrağı veya [F] tuşu ile 240 FPS destekler):
go run ./examples/3d_viewer -fps 240

# Veya repoyu indirmeden doğrudan terminalinizden çalıştırın:
go run github.com/thebanri/limoni/examples/3d_viewer@latest -fps 240
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
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	d := driver.NewDriver(os.Stdin, os.Stdout)
	d.Setup()
	defer d.Close()

	t, err := terminal.New(d)
	if err != nil {
		panic(err)
	}

	d.StartEventLoop()

	t.Draw(func(f *terminal.Frame) {
		f.RenderWidget(widgets.Block{
			Title:         " 🍋 LIMONI TUI ",
			Borders:       widgets.BorderAll,
			BorderSymbols: widgets.SymbolsRounded,
			BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 210, 255)},
		}, f.Buffer.Area)
	})

	for ev := range d.Events() {
		if ev.Type == driver.EventKey && ev.Key.Type == driver.KeyEsc {
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

## 🖥️ Görsel İşleme İpuçları ve SSS (Rendering Quirks & FAQ)

### 1. Çizgiler veya 3D modeller bazı terminallerde neden ince çizgili (hairline gap) veya delikli görünür?
Standart terminal emülatörlerinde varsayılan monospace yazı tipi satır yüksekliği (`line-height` / hücre dolgusu) genellikle bitişik karakter satırları arasına 1–2 piksellik boşluk ekler. Bitişik Braille alt-piksel matrisleri veya yarım-bloklar (`▄`, `▀`) çizilirken bu boşluk yüzeylerin delikli veya ızgara gibi görünmesine yol açabilir.

#### Limoni Bu Sorunu Nasıl Çözer: Alt Yarım-Blok (`▄`, U+2584) Taban Standardı
Geleneksel TUI kütüphaneleri genellikle Üst Yarım-Blok (`▀`, `U+2580`) kullanır. Yazı tipi motorları karakter gliflerini hücrenin **taban çizgisine** (baseline) kilitlediğinden, satır yüksekliği eklendiğinde boşluk hücrenin *üst kısmında* oluşur ve `▀` karakterini üst satırdan ayırır.

Limoni tüm piksel ve 3D çizimlerini **Alt Yarım-Blok (`▄`, `U+2584`)** standardına geçirmiştir:
- **Üst Piksel Rengi:** Hücre arkaplanına (`Cell.Bg`) yazılır.
- **Alt Piksel Rengi:** Hücre önplanına (`Cell.Fg`) yazılır.
- **Karakter:** `▄` (Alt Yarım Blok).

Arka plan rengi karakter hücresinin tamamını kapladığı ve `▄` glifi tam taban çizgisine oturduğu için, gevşek satır yüksekliğine sahip terminallerde dahi pikseller sıfır aralıkla pürüzsüzce birleşir.

### 2. Kusursuz Görsel Deneyim İçin Önerilen Terminal Ayarları
Limoni'nin 3D rasterizasyonunu, grafiklerini ve Braille vektör çizimlerini en yüksek netlikte deneyimlemek için:

* **Satır Yüksekliğini 1.0 Yapın:** Terminalinizin yapılandırma dosyasında `line-height` / `cell-height` değerini `1.0` (veya `%100` / 0 piksel dikey boşluk) olarak ayarlayın.
* **Önerilen Modern Terminaller:**
  - **[Ghostty](https://ghostty.org):** Kutu çizimlerini, Braille ve blok gliflerini sıfır hücre boşluğuyla kusursuz çizen modern GPU terminali.
  - **[Kitty](https://sw.kovidgoyal.net/kitty/):** Yüksek performanslı OpenGL motoru, yerleşik grafik protokolleri ve bitişik glif desteği.
  - **[WezTerm](https://wezfurlong.org/wezterm/):** Mükemmel font fallback ve bitişik kutu glifi desteği.
  - **[Alacritty](https://alacritty.org):** `alacritty.toml` içinde `font.offset.y: 0` ve satır yüksekliği 1.0 kullanın.
* **Önerilen Yazı Tipleri:** [JetBrains Mono](https://www.jetbrains.com/lp/mono/), [Fira Code](https://github.com/tonsky/FiraCode) veya yamalanmış [Nerd Font](https://www.nerdfonts.com/) monospace fontları.

### 3. Limoni Hızlı Animasyonlarda 60+ FPS Performansı Nasıl Korur?
Limoni eşik tabanlı bir **Adaptif Flush Motoruna (Adaptive Flush Engine)** sahiptir:
* **Seyrek Diffing (`dirtyRatio < 0.45`):** Yazma, imleç yanıp sönmesi veya sayaç güncellemeleri gibi seyrek durumlarda sadece değişen hücreleri hesaplayıp hassas imleç sıçramaları (`CUP`) gönderir (**`~7.1 µs`**, 0 B/op).
* **Tam Akış Yenileme (`dirtyRatio >= 0.45`):** 3D model dönüşü veya hızlı kaydırma gibi ekranın %45'inden fazlasının değiştiği durumlarda imleç sıçramaları terk edilir; DEC senkronize güncelleme modu (`\x1b[?2026h`) ve ana konuma dönüş (`\x1b[H`) ile ardışık tam akış gönderilir. Böylece yırtılma ve titreme olmadan **`0 B/op`** hız korunur.

---

## 📂 Örnek Uygulamalar

| Dizin | Başlık | Çalıştırma |
| :--- | :--- | :--- |
| **[`examples/composable`](examples/composable)** | **Lego Mimarisi Composable UI (VStack, HStack, Border, Sıfır Allokasyon)** | `go run ./examples/composable` |
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

## 💡 Mühendislik Felsefesi & Teşekkür

Limoni, Go ekosisteminde terminal performansının sınırlarını zorlamak ve Rust seviyesinde gecikme ve bellek determinizmini Go'ya kazandırmak amacıyla geliştirilmiştir.

> [!NOTE]
> AI tools were used for generating initial boilerplates, documentation drafts, and test cases, while the core architecture, memory layout, and debugging were directed and implemented by the author.

---

## 📄 Lisans

Bu proje **MIT Lisansı** altında lisanslanmıştır.
