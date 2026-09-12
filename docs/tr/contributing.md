# Limoni Katkı Rehberi (Contributing Guide)

**Limoni** projesine katkıda bulunmak istediğiniz için teşekkür ederiz! Limoni; Rust ekosistemindeki Ratatui'nin hızı ve immediate-mode mimarisi ile Go'nun yerel eşzamanlılık (concurrency - goroutine & channel) yeteneklerini birleştiren yüksek performanslı bir Terminal Kullanıcı Arayüzü (TUI) motorudur.

Bu belge; katkı süreçlerini, mimari gereksinimleri, model boyut ve bellek sınırlarını ve test adımlarını açıklamaktadır.

---

## İçindekiler

1. [Temel Prensipler](#temel-prensipler)
2. [Geliştirme Ortamı Kurulumu](#geliştirme-ortamı-kurulumu)
3. [Katkı Süreci (Git & PR)](#katkı-süreci-git--pr)
4. [Model Boyut Sınırları ve Bellek Standartları](#model-boyut-sınırları-ve-bellek-standartları)
   - [3B Grafik Modelleri (OBJ, STL, PLY)](#1-3b-grafik-modelleri-obj-stl-ply)
   - [Çekirdek Bellek Modeli ve Struct Boyut Sınırları](#2-çekirdek-bellek-modeli-ve-struct-boyut-sınırları)
   - [Depo (Git Asset) Dosya Sınırları](#3-depo-git-asset-dosya-sınırları)
5. [Sıfır-Tahsisat (Zero-Allocation) Standartları](#sıfır-tahsisat-zero-allocation-standartları)
6. [Testler Nasıl Çalışır?](#testler-nasıl-çalışır)
   - [Birim Testleri (Unit Tests)](#birim-testleri)
   - [Statik Kod Analizi (go vet)](#statik-kod-analizi)
   - [Veri Yarışı Tespiti (Race Detector)](#veri-yarışı-tespiti)
   - [Sıfır-Tahsisat Benchmark Kontrolü](#sıfır-tahsisat-benchmark-kontrolü)
   - [Örneklerin ve WASM Çıktısının Derlenmesi](#örneklerin-ve-wasm-çıktısının-derlenmesi)
   - [TestKit ve Deterministik Ekran Görüntüleri](#testkit-ve-deterministik-ekran-görüntüleri)

---

## Temel Prensipler

- **Çizim Döngüsünde Sıfır Heap Tahsisatı (Zero Allocations)**: `Draw` ve diff algoritmaları çalışma esnasında dinamik heap tahsisatı yapamaz.
- **Saf Go (Pure Go) ve CGO Bağımsızlığı**: Linux ve macOS için doğrudan termios/ioctl sistem çağrıları, Windows için ConPTY VT100 sürücüsü ve tarayıcı için WASM desteği.
- **Döngüsel Paket Bağımlılığı Yasağı**: Widget'lar alt paketlere veya terminal/backend sürücülerine doğrudan bağlanmaz; etkileşim köprüsü `cell.Context` üzerinden kurulur.
- **Deterministik Test Edilebilirlik**: `testkit` ile bellek içi (in-memory) snapshot ve golden file doğrulaması.

---

## Geliştirme Ortamı Kurulumu

- **Go Sürümü**: `1.22` veya üzeri.
- **İşletim Sistemi**: Linux, macOS veya Windows.
- **Terminal**: TrueColor (24-bit ANSI) destekleyen modern bir terminal önerilir (Ghostty, Kitty, WezTerm, Alacritty, iTerm2, Windows Terminal).

```bash
git clone https://github.com/thebanri/limoni.git
cd limoni

go test ./...
go vet ./...
```

---

## Katkı Süreci (Git & PR)

1. Projeyi GitHub üzerinde **Fork** edin.
2. Yeni bir geliştirme dalı açın:
   ```bash
   git checkout -b feat/ozellik-adi
   # veya: fix/hata-adi, perf/hizlandirma, docs/belge-guncellemesi
   ```
3. Standart Go formatına (`gofmt`) ve sade API tasarımına sadık kalın. Widget konfigürasyonlarında akıcı yapıcı (fluent builder, örn: `WithPlaceholder`, `WithFocusedStyle`) metodlarını kullanın.
4. Anlamlı commit mesajları yazın (Conventional Commits):
   - `feat(driver): add Kitty keyboard protocol CSI u parsing`
   - `fix(textinput): support multiline with Shift+Enter`
   - `docs: add contributing guide`
5. Testleri çalıştırın ve tüm kontrollerin geçtiğinden emin olun.
6. GitHub üzerinden **Pull Request** açın.

---

## Model Boyut Sınırları ve Bellek Standartları

Projeye bileşen, 3B grafik modeli veya durum modeli eklerken aşağıdaki sınır ve standartlara uyulmalıdır:

### 1. 3B Grafik Modelleri (OBJ, STL, PLY)

Limoni yerleşik bir 3B vektör motoruna ve yazılımsal Z-buffer derinlik testine sahiptir:

| Parametre | Sınır / Öneri | Gerekçe |
| :--- | :--- | :--- |
| **Ayrıştırıcı (Parser) Üst Sınırı** | **10.000.000 tepe noktası (vertex) ve yüzey (face)** | OOM (bellek tükenmesi) ve kötü niyetli devasa dosya ataklarını engellemek için PLY/OBJ/STL yükleyicilerinde uygulanan güvenlik tavanıdır. |
| **Gerçek Zamanlı Çizim Önerisi** | **5.000 – 50.000 üçgen (triangle)** | Tek bir CPU çekirdeğinde GPU hızlandırma olmaksızın **60 FPS - 240 FPS** akıcı terminal çizimi için tavsiye edilen tepe sayısıdır. |
| **Koordinat Normalizasyonu** | `Model3D.Normalize(size)` | Dışarıdan yüklenen modellerin viewport'a (görünüm alanına) düzgün sığması için koordinatlar normalize edilmelidir. |

### 2. Çekirdek Bellek Modeli ve Struct Boyut Sınırları

Limoni'nin render hızının temeli CPU L1 önbellek hizalamasıdır. Çekirdek struct boyutları korunmalıdır:

- **`cell.Style` kesinlikle 12 byte olmalıdır**:
  `Fg` (4B) + `Bg` (4B) + `Modifier` (2B) + `padding` (2B) = **12 byte**.
- **`cell.Cell` kesinlikle 16 byte olmalıdır**:
  `Content` (4B rune) + `Style` (12B) = **16 byte**.
  *Neden*: 64 byte'lık tek bir CPU L1 cache satırına tam olarak 4 adet hücre sığar. `Cell` struct'ına yeni alan eklenmesi tüm matris tarama hızını düşürür.
- **Stack-Allocated `cell.Context`**:
  `cell.Context` alt widget'lara stack üzerinde değer olarak (by value) aktarılır. Heap'e kaçacak işaretçi kullanımından kaçınılmalıdır.
- **Durum Modelleri (State Models)**:
  `core/engine` (`Model`, `Update`, `View`) ve `compat/bubbletea` içindeki state modellerinde her render tick'inde büyüyen sınırsız slice tahsisatlarından kaçınılmalıdır.

### 3. Depo (Git Asset) Dosya Sınırları

- Git deposu içine büyük binary 3B model dosyaları eklenmemelidir. `examples/` altındaki modeller düşük poligonlu ve maksimum **500 KB** seviyesinde olmalıdır.
- `assets/` klasöründeki GIF ve görseller sıkıştırılmış ve optimize edilmiş olmalıdır.

---

## Sıfır-Tahsisat (Zero-Allocation) Standartları

- `Widget.Draw(ctx, buf)` metotları **0 B/op** ve **0 allocs/op** değerine sahip olmalıdır.
- Çift tamponlu (double-buffered) diff motoru dinamik tahsisat yapmaz.
- Boş kare optimizasyonu: Ekranda değişiklik olmadığında `Terminal.Draw` süresi **< 100 ns** olmalıdır (mevcut değer: ~11 ns).

---

## Testler Nasıl Çalışır?

CI üzerinde çalışan tüm testleri yerel ortamınızda şu komutlarla çalıştırabilirsiniz:

### Birim Testleri
```bash
go test ./...
```

### Statik Kod Analizi
```bash
go vet ./...
```

### Veri Yarışı Tespiti
```bash
go test -race . ./component ./core/engine ./core/terminal ./testkit ./widgets ./layout ./core/accessibility ./core/driver ./compat/bubbletea
```

### Sıfır-Tahsisat Benchmark Kontrolü
```bash
set -euo pipefail
go test ./core/buffer -run '^$' -bench 'BenchmarkDiff_' -benchmem
go test ./widgets -run '^$' -bench 'BenchmarkBlockDraw|BenchmarkParagraphDraw|BenchmarkTableDraw|BenchmarkTableVisibleRows' -benchmem
go test ./benchmarks -run '^$' -bench 'BenchmarkEmptyFrame|BenchmarkMouseHitTest|BenchmarkHundredLayers|BenchmarkTenThousandRowTable' -benchmem
```

### Örneklerin ve WASM Çıktısının Derlenmesi
```bash
go build -o /dev/null ./examples/demo
go build -o /dev/null ./examples/showcase
GOOS=js GOARCH=wasm go build -o /dev/null ./examples/wasm
```

### TestKit ve Deterministik Ekran Görüntüleri
Yeni bir bileşen yazdığınızda görsel çıktıyı `testkit` ile doğrulayın:
```go
term := testkit.NewTerminal(40, 10)
widget := widgets.NewBlock().WithTitle("Başlık")
term.DrawWidget(widget)
term.AssertContains(t, "Başlık")
```
