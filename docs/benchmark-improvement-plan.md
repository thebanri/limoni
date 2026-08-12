# Limoni Benchmark Sonuçları ve Uygulama Planı

> Bu belge, benchmark sonuçlarını inceleyen bir yapay zekâ/kod ajanının doğrudan uygulayabileceği görev planıdır. Ajan her faza başlamadan önce mevcut kodu ve artifact'ları okumalı, değişiklikleri küçük adımlara bölmeli ve her adımın sonunda test/benchmark çalıştırmalıdır.

## 1. Amaç ve uygulama kuralları

Amaçlar:

1. Benchmark sonuçlarının aynı koşul ve workload manifestiyle üretildiğini garanti etmek.
2. No-op frame ve tek hücre güncelleme yollarını hızlandırmak.
3. Virtual list/table viewport yolunu optimize etmek.
4. Full redraw ve resize maliyetlerini ayrıştırıp azaltmak.
5. Büyük workload'larda allocation ve GC baskısını düşürmek.

Kurallar:

- Varsayım yapma; önce ilgili dosyaları, testleri ve mevcut sonuçları oku.
- Mevcut bağımlılıklar dışında kütüphane ekleme.
- Her optimizasyon için önce/sonra p50, p95, p99, bytes/frame ve allocation sonucunu kaydet.
- Eşdeğer olmayan workload'larda doğrudan performans iddiasında bulunma.
- Ölçümün kendisinin sonucu bozmadığını kontrol et: snapshot kopyaları, output buffer allocation'ları ve warmup davranışı açıkça belirtilmelidir.

## 2. Mevcut sonuçların özeti

Kaynaklar:

- `/home/thebanri/Projects/Limoni/benchmark-results/limoni.json`
- `/home/thebanri/Projects/Limoni/benchmark-results/bubbletea.json`
- `/home/thebanri/Projects/Limoni/benchmark-results/ratatui.json`
- `/home/thebanri/Projects/Limoni/benchmark-results/dashboard.html`
- `/home/thebanri/Projects/Limoni/benchmarks/`

| Workload | Limoni p50 | Bubble Tea p50 | Ratatui p50 | Yorum |
|---|---:|---:|---:|---|
| `empty-frame` | 9.87 µs | 50 ns | 36.05 µs | Limoni Ratatui'den iyi; Bubble Tea aynı işi yapıyor mu doğrulanmalı |
| `full-redraw-120x40` | 42.87 µs | 2.03 µs | 108.35 µs | Limoni Ratatui'den iyi; Bubble Tea path'i farklı olabilir |
| `single-cell-update` | 6.80 µs | 80 ns | 36.38 µs | Dirty update yolu daha da daraltılmalı |
| `text-heavy-120x40` | **16.64 µs** | 84.12 µs | 105.08 µs | Limoni güçlü |
| `unicode-emoji` | **7.30 µs** | 33.78 µs | 44.58 µs | Limoni güçlü |
| `table-10000` | **90.36 µs** | 549.78 µs | 3.41 ms | Limoni açık ara güçlü |
| `virtual-1000000` | 34.46 µs | **7.22 µs** | 196.97 µs | En önemli Limoni performans açığı |
| `mouse-hit-test` | 7.89 µs | 10.81 µs | 37.99 µs | İyi; implementation-specific olarak raporlanmalı |
| `hundred-layers` | 14.64 µs | 15.34 µs | 43.07 µs | Limoni ve Bubble Tea benzer |
| `resize` | 16.72 µs | 160 ns | 63.07 µs | Message-only ve gerçek resize ayrılmalı |
| `async-update-burst` | **190 ns** | 250 ns | 36.05 µs | Limoni güçlü; end-to-end latency ayrıca ölçülmeli |
| `native-image-capability` | 12.33 µs | 30.23 µs | 37.97 µs | Gerçek image encoding eşdeğerliği doğrulanmalı |

Ana çıkarımlar:

- Limoni text, Unicode ve `table-10000` iş yüklerinde güçlüdür.
- `virtual-1000000` p50 değeri Bubble Tea'dan yaklaşık 4.8 kat yavaştır; ilk üretim kodu hedefi budur.
- `empty-frame` ile `single-cell-update` arasındaki fark yalnızca yaklaşık 3 µs'tir; incremental path tam olarak daralmıyor olabilir.
- `full-redraw` ve `resize` karşılaştırmaları framework'ler arasında aynı miktarda işi ölçmeyebilir.
- Runner allocation ölçümleri eşdeğer değildir; Ratatui/Bubble Tea değeri Limoni ile aynı metodolojiyle ölçülmeden karşılaştırılamaz.
- JSON artifact'larında görünen iteration/frame sayısı ile dashboard'daki gösterim aynı üretimden doğrulanmalıdır.

## 3. Faz P0 — Benchmark güvenilirliği ve manifest standardı

**Öncelik:** Kritik  
**Amaç:** Performans optimizasyonundan önce ölçümlerin güvenilir ve tekrarlanabilir olması.

### Hedef dosyalar

- `/home/thebanri/Projects/Limoni/benchmarks/workload.go`
- `/home/thebanri/Projects/Limoni/benchmarks/metrics.go`
- `/home/thebanri/Projects/Limoni/benchmarks/runners/limoni/main.go`
- `/home/thebanri/Projects/Limoni/benchmarks/runners/bubbletea/main.go`
- `/home/thebanri/Projects/Limoni/benchmarks/runners/ratatui/src/main.rs`
- `/home/thebanri/Projects/Limoni/benchmarks/runners/dashboard/main.go`
- `/home/thebanri/Projects/Limoni/.github/workflows/benchmarks.yml`

### Görevler

- [ ] Workload listesini tek bir manifest/şema ile tanımla; üç runner'daki kopyalanmış listeyi drift'e karşı doğrula.
- [ ] Raporlara `manifest_hash`, git commit, runner version, warmup count, timed iterations, build mode ve environment metadata ekle.
- [ ] Tüm spec alanlarını karşılaştır: ad, boyut, rows, unicode, full_draw, mouse, async_burst, output mode, color mode ve iterations.
- [ ] `100 frame` ve `1000 frame` farkının stale artifact mı yoksa farklı konfigürasyon mu olduğunu belirle; tek resmi sonuç üret.
- [ ] Her runner'ın `empty-frame`, `full-redraw`, `single-cell-update`, `resize` ve `virtual-1000000` için gerçekten yaptığı işi dokümante et.
- [ ] `mouse-hit-test`, `async-update-burst` ve `native-image-capability` gibi implementation-specific workload'ları genel render sıralamasından ayır veya açıkça etiketle.
- [ ] Allocation ölçümünün runner'lar arasında eşdeğer olmadığını warning olarak raporla; mümkünse gerçek allocator ölçümü sağla.
- [ ] CI, spec veya metadata uyuşmazlığında başarısız olsun.

### Kabul kriterleri

- Üç runner aynı manifest hash'ini ve aynı workload spec'lerini raporlar.
- Dashboard'daki frame/iteration sayısı JSON ile aynıdır.
- Her raporda host, OS, arch, runtime/compiler ve build mode bulunur.
- Eşdeğer olmayan workload'lar karşılaştırılabilir workload gibi gösterilmez.

## 4. Faz P1 — No-op, single-cell ve dirty-region render yolu

**Öncelik:** Çok yüksek  
**Amaç:** Gereksiz buffer taraması, snapshot kopyası ve output üretimini kaldırmak.

### Hedef dosyalar

- `/home/thebanri/Projects/Limoni/core/buffer/diff.go`
- `/home/thebanri/Projects/Limoni/core/buffer/buffer.go`
- `/home/thebanri/Projects/Limoni/core/buffer/snapshot.go`
- `/home/thebanri/Projects/Limoni/core/terminal/terminal.go`
- `/home/thebanri/Projects/Limoni/testkit/terminal.go`

### Görevler

- [ ] Buffer'da dirty state veya dirty region metadata'sı ekle; hücre değiştiğinde güncelle.
- [ ] Front/back buffer aynıysa `Diff` ve snapshot/output aşamalarını en erken noktada atla.
- [ ] Tek hücre değişiminde tüm ekran yerine değişen hücre veya satır aralığını işle.
- [ ] Her satırın ilk/son dirty kolonunu tutarak contiguous değişiklikleri birleştir.
- [ ] Output byte buffer'ını yeniden kullan; frame başına `[]byte`/`string` kopyasını azalt.
- [ ] Snapshot'ın benchmark için gerekli olduğu ve gerçek terminal output için gerekli olmadığı durumları ayır.
- [ ] Style/color escape üretiminde tekrarları cache'le; cache memory maliyetini ölç.
- [ ] No-op, tek hücre, 10 hücre, %10, %50 ve %100 değişim benchmark'ları ekle.

### Kabul kriterleri

- `single-cell-update` maliyeti değişen alanla orantılı olur.
- No-op frame'de gereksiz output ve allocation oluşmaz.
- Mevcut diff, wide-character ve style transition testleri geçer.

Başlangıç hedefi: P0 tamamlandıktan sonra `empty-frame` ve `single-cell-update` p50 değerlerini 2–4 µs aralığına çekmeyi dene. Bu hedef P0 tamamlanmadan regression gate olarak kullanılmamalıdır.

## 5. Faz P1 — Virtual list/table viewport yolu

**Öncelik:** Çok yüksek  
**Amaç:** `virtual-1000000` p50 değerini yaklaşık 34.46 µs'den 15–20 µs aralığına indirmek.

### Hedef dosyalar

- `/home/thebanri/Projects/Limoni/widgets/virtual_data.go`
- `/home/thebanri/Projects/Limoni/widgets/virtual_data_view.go`
- `/home/thebanri/Projects/Limoni/widgets/virtual_data_test.go`
- `/home/thebanri/Projects/Limoni/widgets/virtual_list_test.go`
- `/home/thebanri/Projects/Limoni/benchmarks/runners/limoni/main.go`

### Görevler

- [ ] Yalnızca viewport ve prefetch aralığındaki satırların üretildiğini test ile kanıtla.
- [ ] `RowID` ve provider içindeki `fmt.Sprintf` hot path maliyetini ölç; gerekirse cache veya `strconv.AppendInt` kullan.
- [ ] Aynı viewport tekrar çizildiğinde provider, row creation ve layout hesaplarının tekrarlanmasını engelle.
- [ ] Viewport row cache ekle; invalidation ve cancellation davranışını test et.
- [ ] `Refresh`/provider erişimi ile `Draw` işlemini ayır; render frame'inde gereksiz refresh yapma.
- [ ] Visible row slice'larını kapasiteyi koruyacak şekilde yeniden kullan.
- [ ] Satır ölçümü, kolon genişliği ve text layout sonuçlarını uygun seviyede cache'le.
- [ ] Aynı viewport, 1 satır scroll, 10 satır scroll, hızlı ileri/geri ve rastgele viewport benchmark'ları ekle.

### Kabul kriterleri

- 1.000.000 satır kaynağın tamamı hiçbir viewport frame'inde taranmaz.
- Aynı viewport tekrarında provider çağrısı ve allocation azalır.
- Loading/error/empty/cancellation davranışı korunur.
- Scroll sırasında görüntü doğruluğu ve stable row ID garantisi korunur.
- `virtual-1000000` baseline'dan kötüleşmez; hedef p50 15–20 µs'dir.

## 6. Faz P2 — Full redraw, diff encoder ve resize

**Öncelik:** Yüksek  
**Amaç:** Buffer fill, diff, ANSI encoding ve gerçek resize maliyetlerini ayırmak ve optimize etmek.

### Görevler

- [ ] Full redraw benchmark'ını `buffer fill`, `diff`, `ANSI encode` ve `terminal write` alt ölçümlerine ayır.
- [ ] Style değişimi olmayan ve her hücrede farklı style olan iki full redraw senaryosu ekle.
- [ ] Cursor movement ve ANSI escape üretimini contiguous değişiklik blokları için optimize et.
- [ ] Color/style dönüşümlerini cache'le ve cache allocation maliyetini ölç.
- [ ] Resize benchmark'ını `WindowSizeMsg`, buffer resize ve resize + layout + redraw olarak ayır.
- [ ] Buffer capacity reuse uygula; küçülen ekranlarda gereksiz allocation yapma.
- [ ] Resize sonrası layout invalidation'ı yalnızca gerekli widget'larla sınırla.

### Kabul kriterleri

- Full redraw'un maliyetli alt aşaması profiler ve benchmark ile görülebilir.
- Resize message handling ile gerçek resize/render birbirine karıştırılmaz.
- ANSI, wide-character, style reset ve buffer correctness testleri geçer.
- Adil workload doğrulandıktan sonra full redraw p50 için en az %20 iyileşme hedeflenir.

## 7. Faz P2 — Büyük workload allocation ve GC

**Öncelik:** Orta-yüksek

Güncel yaklaşık baseline:

```text
table-10000:      61 MB / 105.015 allocation
virtual-1000000:  15.5 MB / 125.029 allocation
hundred-layers:   938 KB / 101.016 allocation
```

### Görevler

- [ ] `go test -gcflags=-m` ile hot path escape noktalarını tespit et.
- [ ] Table row/cell, virtual row/provider ve layer registration allocation'larını profile et.
- [ ] `AllocBytes` ve `Allocs` değerlerini frame başına normalize ederek raporla.
- [ ] Heap peak, GC pause ve retained heap için profiling çalıştır.
- [ ] Cache ekleniyorsa cache hit/miss ve memory overhead ölç.

### Kabul kriterleri

- Allocation optimizasyonu p50 latency'yi anlamlı biçimde kötüleştirmez.
- Büyük workload allocation'larında baseline'a göre en az %30 azalma hedeflenir.

## 8. Faz P3 — Runtime ve gerçek terminal I/O benchmark'ları

**Öncelik:** Orta

- [ ] `async-update-burst` için Send, queue, Update, redraw scheduling ve end-to-end latency'yi ayrı ölç.
- [ ] Queue depth, coalesced/dropped message count, goroutine count ve cancellation latency raporla.
- [ ] `memory`, `/dev/null`, PTY ve gerçek terminal sink'lerini ayrı benchmark gruplarında çalıştır.
- [ ] truecolor, 256-color, no-color, ASCII ve Unicode output modlarını ayrı raporla.
- [ ] Memory ve terminal I/O sonuçlarını aynı dashboard sıralamasında birleştirme.

Kabul kriterleri:

- Async benchmark yalnızca queue'ya yazma süresini frame latency olarak adlandırmaz.
- Memory ve terminal I/O sonuçları ayrı etiketlenir.
- PTY testleri Linux'ta çalışır ve platform sınırlamaları belgelenir.

## 9. CI regression policy

Workflow hedefi:

- `/home/thebanri/Projects/Limoni/.github/workflows/benchmarks.yml`

Eklenecek kontroller:

- [ ] Benchmark JSON schema validation.
- [ ] Ortak workload manifest/hash validation.
- [ ] Environment/build/commit metadata validation.
- [ ] Base branch ile p50/p95/p99 karşılaştırması.
- [ ] Allocation/frame regression karşılaştırması.
- [ ] JSON, HTML dashboard ve baseline artifact upload'u.
- [ ] Büyük regresyonda CI failure, küçük regresyonda warning.

Başlangıç eşikleri:

```text
p50 latency regression > 5%       warning
p95 latency regression > 10%      failure
allocation/frame regression > 10% failure
workload/spec mismatch            failure
invalid or missing report         failure
```

Eşikler aynı hostta birkaç baseline çalıştırmasından sonra kalibre edilmelidir.

## 10. Ajan çalışma protokolü

Her görev için aşağıdaki sırayı izle:

1. Hedef dosyaları ve mevcut testleri oku.
2. İlgili benchmark'ı çalıştır ve baseline kaydet.
3. Tek bir hipotez seç; örneğin snapshot kopyasının single-cell maliyetini artırdığı.
4. Küçük bir kod değişikliği yap.
5. İlgili unit testleri çalıştır.
6. Benchmark'ı en az `-count=5` ile tekrarla.
7. p50, p95, p99, bytes/frame, allocs/frame ve correctness farkını karşılaştır.
8. İyileşme yoksa değişikliği geri al veya yeni hipotez oluştur.
9. İyileşme varsa regression test/benchmark ekle.
10. En sonda `go test ./...`, gerekiyorsa `go vet ./...` ve race testlerini çalıştır.
11. Değiştirilen dosyaları tekrar oku ve `git diff`/`git status` ile doğrula.

## 11. Genel başarı ölçütleri

- Sonuçlar aynı manifest ve metadata ile tekrarlanabilir.
- Framework-neutral ve implementation-specific workload'lar ayrılmıştır.
- `virtual-1000000` baseline'dan kötüleşmemiş ve 15–20 µs p50 hedefi denenmiştir.
- `single-cell-update` dirty region üzerinden çalışmaktadır.
- Full redraw ve resize alt aşamalara ayrılmıştır.
- Büyük table/virtual/layer workload allocation'larında en az %30 azaltma hedeflenmiştir.
- Her optimizasyonun correctness testi ve önce/sonra benchmark kanıtı vardır.
- Son doğrulama başarılıdır:

```bash
cd /home/thebanri/Projects/Limoni
go test ./...
go vet ./...
```

## 12. Plan dışı bırakılanlar

- Benchmark kanıtı olmadan genel API redesign.
- Yeni dış bağımlılık eklemek.
- Yalnızca tek p50 sonucu ile optimizasyon kararı vermek.
- Eşdeğer olmayan runner workload'larını pazarlama karşılaştırması olarak kullanmak.
- Gerçek terminal I/O ölçmeden kullanıcı deneyimi hakkında kesin performans iddiası yapmak.