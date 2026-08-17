# Stratejik İyileştirme ve Mimari Geliştirme Planı (Limoni PATH Execution)

## Context
Limoni'nin Ratatui ve Bubble Tea ile yapılan kapsamlı benchmark analizleri (`benchmark-results/*.json`) sonucunda;
- `full-redraw-120x40` (22.5µs), `virtual-1000000` (22.3µs) ve `native-image-capability` (6.3µs) gibi iş yüklerinde sektör lideri hızlar yakalanmıştır.
- Ancak `hundred-layers`, `mouse-hit-test`, `table-10000` ve `empty-frame` iş yüklerinde gereksiz heap bellek tahsisleri ve CPU döngüleri tespit edilmiştir.
- Bu plan, tespit edilen 4 ana mimari darboğazı çözmeyi ve projeyi `v0.1.0` prodüksiyon kalitesine taşımayı hedefler.

---

## 1. Mimari İyileştirmeler (Architectural Optimizations)

### A. `hundred-layers` ve `mouse-hit-test` İçin Sıfır-Tahsisat (Zero-Alloc Frame Pooling)
- **Problem**: `Frame.RegisterLayer`, `RegisterClickHandler`, `EventRegions` ve `ClickRegions` her karede dinamik slice büyümesi (`append`) ve heap allocation yapmaktadır (`922 KB` ve `120 KB`).
- **Çözüm**: 
  - `core/terminal/frame.go` içerisindeki `Layer`, `ClickRegion`, `eventRegion` ve `DebugRegion` dilimleri `Reset()` anında kapasiteleri korunacak şekilde dilimlenmeli (`slice = slice[:0]`).
  - `NewFrame` başlatılırken önceden yeterli kapasiteyle (örn. 64-128 eleman) pre-allocate edilmeli.
  - `RegisterLayer` içindeki `fmt.Sprintf` veya string kopyalamaları yerine referans veya inline identifier desteği sağlanmalı.

### B. `table-10000` İçin Sıfır-Tahsisat Görünür Satır Gezgini (Zero-Alloc Table Rendering)
- **Problem**: `widgets/table.go` `Draw` metodu içerisinde `filtered` ve `rows` üzerinde gereksiz slice tahsisleri veya `make` çağrıları gerçekleşmektedir.
- **Çözüm**:
  - `tableDrawScratch` havuzuna (`tableDrawScratchPool`) lazy iterator mantığı bağlanarak, 10.000 satır olsa bile yalnızca `startRow` ve `endRow` arasındaki görünür satırlar işlenecek.
  - Sahiplik matrisi (`owner` & `cellsMap`) boyutları sabit tutulup her karede map yeniden tahsis edilmeden `clear(m)` ile sıfırlanacak.

### C. `empty-frame` İçin Hızlı Yol (Fast-Path Dirty Bypass)
- **Problem**: `empty-frame` iş yükünde 1.7 µs harcanmaktadır.
- **Çözüm**:
  - `Buffer` üzerinde `Dirty` bayrağı kontrol edilerek hiçbir widget çizilmediğinde veya hücre değişmediğinde `buffer.Diff` hızlı dönüş (`return nil, nil` / `0ns`) yapacaktır.

### D. Cross-Platform Native Termios (`darwin` / macOS Desteği)
- **Problem**: macOS üzerinde `backend_nonlinux.go` stub olarak çalışmaktadır.
- **Çözüm**:
  - `core/backend/termios_darwin.go` dosyası eklenerek `golang.org/x/sys/unix` üzerinden macOS'a özgü `TIOCGETA` ve `TIOCSETA` ioctl çağrılarıyla CGO'suz saf Go ham mod (raw mode) desteği sağlanacaktır.

---

## 2. Değiştirilecek / Eklenecek Kritik Dosyalar

1. `core/terminal/frame.go` — Frame slice pooling, event and click region zero-alloc reset
2. `core/terminal/terminal.go` — Fast-path check for empty/untouched frame
3. `widgets/table.go` — Zero-alloc visible rows slicing and scratch pool reuse
4. `core/backend/termios_darwin.go` — Native macOS Termios implementation
5. `core/backend/backend_darwin.go` — Native macOS event loop and raw mode support
6. `benchmarks/runners/limoni/main.go` — Updated benchmark runner reflecting the zero-alloc hot paths

---

## 3. Doğrulama ve Test Adımları (Verification Plan)

1. **Birim Testleri ve Race Detector**:
   ```bash
   go test -v -race ./...
   go vet ./...
   ```
2. **Benchmark Sıfır-Tahsisat Doğrulaması**:
   ```bash
   go run ./benchmarks/runners/limoni -output benchmark-results/limoni.json
   go test ./benchmarks -run '^$' -bench . -benchmem
   ```
   - `hundred-layers`: Hedef `< 10 KB / 0 allocs in hot loop`
   - `mouse-hit-test`: Hedef `0 B/op`
   - `empty-frame`: Hedef `< 100 ns`
   - `table-10000`: Hedef `< 500 KB`
3. **Cross-Platform Derleme Testi**:
   ```bash
   GOOS=darwin GOARCH=arm64 go build ./...
   GOOS=windows GOARCH=amd64 go build ./...
   GOOS=linux GOARCH=amd64 go build ./...
   ```
