# Limoni `benchmark-results` Klasörü — Uygulama Planı

> Bu belge, `/home/thebanri/Projects/Limoni/benchmark-results` klasöründeki benchmark artifact'larını güvenilir, tekrarlanabilir ve yapay zekâ/kod ajanı tarafından uygulanabilir hale getirmek için hazırlanmıştır.

## 1. Kapsam ve mevcut durum

Bu plan sonuç klasörünü, sonuç üretim akışını, dashboard doğruluğunu ve CI regression kontrolünü kapsar. Üretim kodu optimizasyonları, sonuç artifact'ları güvenilir hale getirildikten sonra yapılmalıdır.

Mevcut dosyalar:

- `README.txt`
- `go-benchmark.txt`
- `limoni.json`
- `bubbletea.json`
- `ratatui.json`
- `dashboard.html`
- `limoni-baseline.json`

### Kritik sorun

`/home/thebanri/Projects/Limoni/benchmark-results/limoni-baseline.json` dosyası 0 byte'tır. Geçerli JSON değildir ve baseline olarak kullanılamaz.

İlk görev bu dosyayı gerçek baseline ile doldurmak veya yanlışlıkla oluşturulduysa silmektir. Boş baseline dosyası CI veya regresyon karşılaştırmasını bozmamalıdır.

## 2. Mevcut sonuçların yorumu

### Güçlü Limoni workload'ları

- `text-heavy-120x40`: yaklaşık 16.6 µs p50
- `unicode-emoji`: yaklaşık 7.3 µs p50
- `table-10000`: yaklaşık 90 µs p50
- `async-update-burst`: yaklaşık 190 ns p50
- `mouse-hit-test` ve `hundred-layers`

### Öncelikli workload'lar

| Workload | Limoni p50 | Öncelik | Yorum |
|---|---:|---:|---|
| `empty-frame` | 2.67 µs | P1 | Bubble Tea'nın 50 ns değeri aynı işi ölçüyor mu doğrulanmalı |
| `single-cell-update` | 8.45 µs | P1 | Dirty-region avantajı yeterince görünmüyor |
| `full-redraw-120x40` | 29.64 µs | P2 | Buffer fill, diff ve encoding ayrılmalı |
| `virtual-1000000` | yaklaşık 34.46 µs | P1 | Bubble Tea'dan yaklaşık 4.8× yavaş |
| `resize` | 21.67 µs | P2 | Message-only ve gerçek resize ayrılmalı |

### Karşılaştırma uyarısı

Üç runner aynı `manifest_hash` kullanıyor olsa da aynı isimli workload'lar aynı kod yolunu çalıştırmıyor olabilir:

- Limoni gerçek buffer/widget/diff yolunu ölçüyor.
- Bubble Tea model/view ve string üretimini ölçüyor.
- Ratatui bazı workload'larda farklı buffer/table işlemleri yapıyor.
- Ratatui allocation değerlerinin bazı sonuçlarda `0` olması, Limoni ile aynı metodolojinin kullanılmadığını gösterir.
- `native-image-capability`, `mouse-hit-test`, `resize` ve `async-update-burst` implementation-specific olarak ayrıca raporlanmalıdır.

Dashboard tek bir koşulsuz `VALID COMPARISON` mesajı yerine workload bazlı status göstermelidir.

## 3. Faz P0 — Artifact temizliği ve baseline standardı

**Amaç:** Klasörde yalnızca geçerli, açıklanabilir ve kullanılabilir sonuçlar bulundurmak.

### Hedef dosyalar

- `/home/thebanri/Projects/Limoni/benchmark-results/limoni-baseline.json`
- `/home/thebanri/Projects/Limoni/benchmark-results/README.txt`
- `/home/thebanri/Projects/Limoni/.github/workflows/benchmarks.yml`

### Görevler

- [ ] `limoni-baseline.json` dosyasının 0 byte olma nedenini belirle.
- [ ] Geçerli baseline üret veya dosyayı kaldır; boş dosyayı repository'de bırakma.
- [ ] Baseline şemasını güncel runner JSON şemasıyla aynı yap.
- [ ] Baseline metadata'sına commit, manifest hash, timestamp, OS, architecture, CPU, runtime/compiler, runner version, warmup ve iterations ekle.
- [ ] `README.txt` dosyasını klasördeki her artifact'ın amacını açıklayacak şekilde güncelle.
- [ ] Artifact üretim tarihini ve commit bilgisini görünür yap.
- [ ] CI'da boş, geçersiz veya eksik JSON dosyalarını reddet.

### Kabul kriterleri

- `benchmark-results/*.json` içindeki tüm dosyalar geçerli JSON'dır.
- Boş baseline dosyası yoktur.
- Baseline ve güncel sonuçlar aynı şemayı kullanır.
- Her artifact hangi commit, manifest ve host üzerinde üretildiğini gösterir.

## 4. Faz P0 — JSON schema ve klasör validation

**Amaç:** Eksik veya birbirinden farklı sonuçların dashboard'a ulaşmasını engellemek.

### Zorunlu alanlar ve kontroller

- `implementation`, `environment`, `valid` ve `workloads` mevcut olmalı.
- `manifest_hash`, `git_commit` ve `runner_version` mevcut olmalı.
- Workload sayısı 12 olmalı.
- Workload spec alanları bütün runner'larda aynı olmalı.
- `Frames >= Iterations` olmalı.
- `P50 <= P95 <= P99` olmalı.
- Summary metrikleri negatif olmamalı.

### Görevler

- [ ] JSON schema veya eşdeğer Go validation fonksiyonu ekle.
- [ ] Workload adı, boyutu, rows, unicode, full_draw, mouse, async_burst, output mode, color mode ve iterations değerlerini karşılaştır.
- [ ] Manifest hash uyuşmazlığında CI'ı başarısız yap.
- [ ] CPU, OS, architecture, runtime ve build mode farklarını baseline karşılaştırmasında kontrol et.
- [ ] Validation hatalarını dosya ve workload bilgisiyle yazdır.

### Kabul kriterleri

- Eksik metadata CI'ı başarısız yapar.
- Workload sayısı veya sırası farklıysa CI başarısız olur.
- Spec alanlarından biri farklıysa açıklayıcı hata üretilir.
- Geçersiz JSON dashboard üretiminden önce reddedilir.

## 5. Faz P1 — Dashboard karşılaştırma doğruluğu

### Önerilen status değerleri

- `VALID`: equivalent workload
- `WARNING`: implementation-specific workload
- `INVALID`: mismatched workload

### Görevler

- [ ] Genel `VALID COMPARISON` mesajını workload bazlı status ile değiştir.
- [ ] `mouse-hit-test`, `hundred-layers`, `resize`, `async-update-burst` ve `native-image-capability` workload'larını implementation-specific olarak etiketle.
- [ ] Her runner'ın workload başına ölçüm scope'unu rapora ekle.
- [ ] Allocation ölçülmeyen runner'larda `0` yerine `N/A` veya `not_measured` göster.
- [ ] Dashboard'a manifest hash, commit, CPU, runtime, runner version, warmup ve iterations bilgilerini ekle.
- [ ] JSON ve HTML dashboard değerlerinin aynı olduğunu test et.
- [ ] Warning açıklamasını workload satırında göster.

Önerilen scope değerleri: `render-only`, `buffer-write`, `diff-only`, `output-encode`, `model-update`, `layout`, `input-routing`, `runtime`, `allocation`.

### Kabul kriterleri

- Dashboard yalnızca gerçekten eşdeğer workload'ları `VALID` gösterir.
- Implementation-specific workload'lar ayrı bölümde listelenir.
- Ölçülmeyen allocation değeri yanlışlıkla sıfır gibi yorumlanmaz.
- Dashboard metadata'sı kaynak JSON ile birebir eşleşir.

## 6. Faz P1 — Baseline ve regression karşılaştırması

### Hesaplanacak alanlar

- `current_p50_ns`, `baseline_p50_ns`, `p50_delta_ns`, `p50_delta_percent`
- `current_p95_ns`, `baseline_p95_ns`, `p95_delta_ns`, `p95_delta_percent`
- `current_alloc_bytes`, `baseline_alloc_bytes`, `allocation_delta_percent`

### Görevler

- [ ] Baseline formatını tanımla.
- [ ] Baseline ile güncel sonucu yalnızca aynı manifest hash ve uyumlu environment varsa karşılaştır.
- [ ] CPU veya build mode farklıysa sonucu `not comparable` olarak işaretle.
- [ ] p50, p95, p99 ve allocation farklarını hesapla.
- [ ] Baseline bulunamadığında açık hata üret.
- [ ] Dashboard'a regression değerlerini ve status badge'lerini ekle.

Başlangıç eşikleri:

- p50 regression > 5%: warning
- p95 regression > 10%: failure
- p99 regression > 15%: warning
- allocation/frame regression > 10%: failure
- manifest mismatch, missing baseline veya invalid report: failure

## 7. Faz P2 — `go-benchmark.txt` sonuçlarını ayırma

Klasörde iki farklı benchmark sistemi vardır:

1. `/home/thebanri/Projects/Limoni/benchmark-results/go-benchmark.txt`
2. `limoni.json`, `bubbletea.json`, `ratatui.json`

Go benchmark çıktısı `BenchmarkEmptyFrame`, `BenchmarkTextHeavyFrame`, `BenchmarkTenThousandRowTable`, `BenchmarkMouseHitTest`, `BenchmarkHundredLayers` ve `BenchmarkAsyncUpdateBurst` testlerini içerir.

### Görevler

- [ ] `go-benchmark.txt` için machine-readable JSON veya parse edilebilir ayrı format üret.
- [ ] Go native benchmark sonuçlarını cross-implementation sonuçlarından ayrı sınıflandır.
- [ ] `ns/op`, `B/op` ve `allocs/op` değerlerini ayrı metrikler olarak raporla.
- [ ] CPU ve Go version metadata'sını native benchmark özetine ekle.
- [ ] Dashboard'da native Go benchmark ve cross-implementation bölümlerini ayır.
- [ ] Text parsing başarısız olduğunda CI'ı sessizce başarılı yapma.

### Kabul kriterleri

- Go benchmark sonucu manuel text okumadan işlenebilir.
- Native Go ve cross-implementation sonuçları aynı tabloya karıştırılmaz.
- Her sonuç hangi benchmark sistemine ait olduğunu belirtir.

## 8. Faz P2 — Measurement scope ve runner eşdeğerliği

### Görevler

- [ ] Her runner için workload başına `measurement_scope` alanı ekle.
- [ ] Limoni'nin buffer/diff, Bubble Tea'nın view/string ve Ratatui'nin terminal buffer yolunu dokümante et.
- [ ] Ratatui'de `table_rows.clone()` gibi benchmark içine dahil edilen işlemleri belirt.
- [ ] Native image workload'unun gerçek image encoding yapıp yapmadığını doğrula.
- [ ] Async workload için queue write, update processing ve end-to-end latency'yi ayır.
- [ ] Resize workload'unu message-only, buffer-resize ve resize+layout+redraw olarak ayır.
- [ ] Eşdeğer olmayan workload'ları `framework-specific` olarak işaretle.

### Kabul kriterleri

- Her JSON sonucu hangi kod yolunu ölçtüğünü belirtir.
- Aynı scope'a sahip sonuçlar karşılaştırılır.
- Farklı scope'lar dashboard'da ayrı bölümlerde görünür.
- `VALID` statüsü yalnızca savunulabilir karşılaştırmalarda kullanılır.

## 9. Faz P3 — Artifact arşivleme ve klasör düzeni

Önerilen yapı:

```text
benchmark-results/
├── README.txt
├── latest/
│   ├── limoni.json
│   ├── bubbletea.json
│   ├── ratatui.json
│   ├── dashboard.html
│   └── go-benchmark.txt
├── baseline/
│   ├── limoni.json
│   └── metadata.json
└── history/<git-commit>/
    ├── limoni.json
    ├── bubbletea.json
    ├── ratatui.json
    ├── dashboard.html
    ├── go-benchmark.txt
    └── metadata.json
```

### Görevler

- [ ] `latest`, `baseline` ve `history` stratejisinden birini seç.
- [ ] CI her çalıştırmayı commit ve manifest hash ile ilişkilendirsin.
- [ ] Full history'yi GitHub Actions artifact olarak saklamayı değerlendir.
- [ ] Repository'de yalnızca güncel özet ve kontrollü baseline tutmayı değerlendir.
- [ ] Baseline güncellemesini kontrollü workflow ile yap.
- [ ] Aynı commit artifact'larının birbirini ezmesini engelle.

## 10. CI uygulama planı

Workflow hedefi: `/home/thebanri/Projects/Limoni/.github/workflows/benchmarks.yml`

### CI adımları

- [ ] Benchmark çalıştırmadan önce output klasörünü temizle.
- [ ] Üç runner'ı ortak manifest ile çalıştır.
- [ ] JSON schema ve metadata validation çalıştır.
- [ ] Workload spec/hash eşleşmesini doğrula.
- [ ] Dashboard üret.
- [ ] Dashboard ve JSON consistency testini çalıştır.
- [ ] Baseline regression karşılaştırmasını çalıştır.
- [ ] JSON, dashboard, native benchmark ve metadata artifact'larını upload et.
- [ ] Invalid report veya eşik aşan regression durumunda warning/failure üret.

## 11. Yapay zekâ/kod ajanı çalışma protokolü

1. Hedef dosyaları ve mevcut testleri oku.
2. `benchmark-results` içindeki güncel artifact'ları kontrol et.
3. Baseline'ın geçerli olduğunu doğrula.
4. Değişiklikten önce ilgili benchmark'ı çalıştır ve baseline kaydet.
5. Tek bir hipotez veya validation hedefi seç.
6. Küçük ve geri alınabilir bir değişiklik yap.
7. İlgili unit testleri çalıştır.
8. İlgili benchmark'ı en az `-count=5` ile tekrar çalıştır.
9. p50, p95, p99, bytes/frame, allocs/frame ve validity farkını karşılaştır.
10. Başarılı değişiklik için regression veya validation test ekle.
11. Sonunda `go test ./...`, `go vet ./...` ve ilgili race testlerini çalıştır.
12. Değiştirilen dosyaları yeniden oku.
13. `git diff --check` ve `git status` ile çalışma ağacını doğrula.

## 12. Genel başarı ölçütleri

- `benchmark-results` içinde boş veya geçersiz JSON bulunmaz.
- Baseline geçerli, metadata içeren ve güncel sonuçla karşılaştırılabilir durumdadır.
- Tüm raporlar aynı manifest hash'ini veya açıkça farklı manifest status'ünü taşır.
- Dashboard tüm sonuçları koşulsuz `VALID COMPARISON` göstermez.
- Implementation-specific workload'lar ayrı etiketlenir.
- Allocation ölçülmeyen runner'lar `0` olarak yorumlanmaz.
- Native Go benchmark sonuçları cross-implementation sonuçlarından ayrıdır.
- Baseline regression otomatik hesaplanır.
- CPU, OS, architecture, runtime ve build mode uyumsuzlukları görünürdür.
- Artifact'ların hangi commit ve host üzerinde üretildiği anlaşılır.

Son doğrulama:

```bash
cd /home/thebanri/Projects/Limoni
go test ./...
go vet ./...
git diff --check
```

## 13. Plan dışı bırakılanlar

- Benchmark kanıtı olmadan genel API redesign.
- Yeni dış bağımlılık eklemek.
- Yalnızca tek p50 sonucu ile performans kararı vermek.
- Eşdeğer olmayan runner workload'larını pazarlama karşılaştırması olarak sunmak.
- Gerçek terminal I/O ölçmeden kullanıcı deneyimi hakkında kesin iddia yapmak.
- Baseline host/CPU uyumsuzken regression sonucu üretmek.