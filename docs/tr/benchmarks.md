# Performans ve Kıyaslamalar (Benchmarks)

> 📐 **Önce [`docs/benchmark-methodology.md`](../benchmark-methodology.md) dosyasını okuyun.** Hangi sürümlerin ölçüldüğünü, harness'ın neyi yakalayıp neyi yakalamadığını ve hangi iddiaların henüz kanıtlanmadığını açıklar. Çapraz-framework koşucuları **Ratatui 0.30.2**, **Ultraviolet** (Bubble Tea v2 ve Lip Gloss v2'nin altındaki hücre render motoru) ve **Bubble Tea v1.3.10**'u hedefliyor. **Bubble Tea v2 koşucusu yoktur**: burada hiçbir şey onu link etmiyor, dolayısıyla bu depodaki hiçbir iddia v2'nin kendisi hakkında değildir.

Limoni, standart sanal terminal ortamında (120×40 hücre = 4.800 hücre) gerçek dirty diffing, kısmi güncellemeler, sanal kaydırma ve bellek tahsisatlarını ölçen kapsamlı bir kıyaslama paketine sahiptir.

Testleri yerel ortamınızda çalıştırmak için:
```bash
# Buffer Diff kıyaslamaları (kirli ve temiz kare testleri)
go test ./core/buffer -run '^$' -bench . -benchmem

# Widget ve Düzen kıyaslamaları
go test ./benchmarks -run '^$' -bench . -benchmem

# Çapraz-framework karşılaştırması: her koşucuyu derler, üçünü de üçer kez
# çalıştırır, medyan gecikmeyi koşular arası yayılımla raporlar ve koşucuların
# karşılaştırılamaz işaretlediği her oranı gizler. Ratatui için Rust gerekir.
./benchmarks/compare.sh
```

## Ölçülen sonuçlar

**AMD Ryzen 5 5600 (6Ç/12İ), Linux 6.17, Go 1.27.1, `-count=3`, medyan.** Mutlak
değerler donanıma bağlıdır; anlamlı olan, tek bir makinede commit'ler arasındaki
orandır. Yukarıdaki komutla yeniden üretilebilir.

| Kıyaslama İşlemi | Ölçülen Gecikme | Kare / İşlem Hızı | Bellek Tahsisatı | Açıklama |
| :--- | :--- | :--- | :--- | :--- |
| **`BenchmarkDiff_FullChanges`** | **`~50.5 µs`** | **~19.800 FPS** | **`0 B/op (0 allocs)`** | %100 tam ekran hücre değişimi (4.800 hücre) çift tampon diff işlemi ve ANSI akışı üretimi |
| **`BenchmarkDiff_PartialChanges`** | **`~23.7 µs`** | **~42.200 FPS** | **`0 B/op (0 allocs)`** | %10 ekran alanı değişimi (480 hücre) çift tampon diff işlemi |
| **`BenchmarkDiff_NoChanges`** | **`~1.94 ns`** | **~516.000.000 FPS** | **`0 B/op (0 allocs)`** | Tamponda hiçbir değişiklik olmadığında fast-path ile anında dönüş |
| **`BenchmarkTextHeavyFrame`** | **`~31.7 µs`** | **~31.500 FPS** | **`2 B/op (0 allocs)`** | 120 sütuna yayılan 40 satırlık unicode sembollü ve kelime kaydırmalı metin çizimi |
| **`BenchmarkHundredLayers`** | **`~68.2 µs`** | **~14.700 FPS** | **`1 B/op (0 allocs)`** | 100 katmanlı Block widget çizimi ve değerlendirmesi (Ratatui hundred-layers denklik testi) |
| **`BenchmarkTenThousandRowTable`** | **`~68.5 µs`** | **~14.600 FPS** | **`611 B/op (4 allocs)`** | 10.000 satırlık tabloda aktif imleç kaydırma (scrolling) ve görünür satır çizimi |
| **`BenchmarkOneMillionRowVirtualScroll`**| **`~2.70 ms`** | **~370 FPS** | **`4.9 KB/op (6 allocs)`** | 1.000.000 satırlık sanal veri kaynağında aktif kaydırma ve görünür alan yönetimi |
| **`BenchmarkMouseHitTest`** | **`~63.4 ns`** | **~15.800.000 op/s** | **`0 B/op (0 allocs)`** | 100 tıklama bölgesi üzerinde hiyerarşik uzamsal fare tıklama tespiti |
| **`BenchmarkAsyncUpdateBurst`** | **`~232 ns`** | **~4.310.000 msg/s** | **`7 B/op (0 allocs)`** | Elm çalışma mimarisinde yüksek verimli asenkron mesaj kuyruğu iletimi |

> [!NOTE]
> **Bu değerler iki kez, iki yönde de değişti.** Tablonun önceki hâlinde adı
> geçen donanım sınıfında yeniden üretilemeyen gecikmeler vardı;
> `BenchmarkHundredLayers` 47 µs iddia ediliyor, 159 µs ölçülüyordu. Bunu
> düzeltmek zamanın asıl nerede gittiğini ortaya çıkardı: katmanlı bir karenin
> %39'u `cell.RuneWidth`'te geçiyordu, yazılan her hücre için yirmi aralık
> karşılaştırması dolaşarak. Artık cevabı bir arama tablosundan veriyor ve bu,
> çizim yolunu genel olarak 2–3 kat aşağı çekti. Yani yukarıdaki sayılar hem
> düzeltilmiş hâlden hem de orijinal şişirilmiş iddialardan daha düşük — bu kez
> arkalarında bir profil var. Sıfır-tahsisat garantileri baştan sona korundu.

> [!NOTE]
> **Tablo grapheme cluster desteğinden önceye ait.** Metni bölütlemenin ASCII
> dışı karakterlerde bir maliyeti var; düz ASCII bunu atlayan hızlı yoldan
> geçiyor. Yukarıdaki makinede, değişiklikten önceki commit'e karşı art arda
> ölçüldü (`-count=3`, medyan): `BenchmarkTextHeavyFrame` +%5 (30,7 → 32,3 µs —
> metninde satır başına üç sembol var), `BenchmarkDiff_FullChanges` +%2,
> `BenchmarkHundredLayers` ve `BenchmarkDiff_PartialChanges` değişmedi; hepsi
> hâlâ sıfır tahsisatta. Tablodaki mutlak değerler yeniden ölçülmedi; satırları
> değil oranları karşılaştırın.

> [!NOTE]
> **Şeffaflık ve Mühendislik Dürüstlüğü Garantisi**:
> Sentetik kısayollar, yapay tampon temizlemeleri veya statik sıfır-offset döngüleri kullanılmaz.
> - **Diff Kıyaslamaları**: Hücrelerin her karede bizzat değiştiği kalıcı çift tampon üzerinde çalışır; diff motorunu ve ANSI kodlayıcısını uçtan uca çalıştırır.
> - **Kaydırma Kıyaslamaları**: `Select((i * 7) % N)` ile satırlar arasında aktif olarak kaydırma yapar ve sürekli kaydırma altında bellek tüketimini test eder.
> - **100 Katman Testi**: Tıklama kestirmesi yerine 100 adet `Block` widget'ını ekrana bizzat çizer.
