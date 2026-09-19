# Kararlılık ve sürümleme

Limoni henüz **1.0 öncesi** bir sürümde. Bu sayfa bunun pratikte ne anlama
geldiğini anlatıyor. Böylece bağımlılığınızı ne kadar sıkı sabitleyeceğinize siz
karar verebilirsiniz.

## v1.0.0'a kadar geçerli kurallar

- **Yama sürümleri (`v0.x.y`) API'yi bozmaz.** Aynı minor sürüm içinde yükseltmek
  güvenli olmalı. Bu sürümler yalnızca hata düzeltmesi, performans iyileştirmesi,
  yeni widget ve yeni seçenek getirir.
- **Minor sürümler (`v0.x.0`) API'yi bozabilir.** Her kırılma
  [CHANGELOG.md](../../CHANGELOG.md) içinde **Breaking** başlığı altında, yerine
  ne yapılacağıyla birlikte yazılır.
- Her tag'den önce önceki tag'e karşı `gorelease` çalıştırılır. Böylece bir kırılma
  fark edilmeden yama sürümüne giremez.
- Allocation bütçeleri de sözleşmenin parçası. Bugün `0 B/op` olan bir çizim yolu
  bir yama sürümünde allocation yapmaya başlamaz.

## Paketlerin durumu

| Paket | Durum | Ne beklemeli |
| :--- | :--- | :--- |
| `limoni` (kök), `component`, `layout` | **Oturuyor** | Yapının 1.0'a kadar korunması hedefleniyor. İsimler bir minor sürümde yine değişebilir. |
| `widgets`, `core/cell`, `core/buffer`, `testkit` | **Oturuyor** | Aynısı geçerli. Ek olarak widget *state* tipleri dışa açık olmayan alanlar kazanabilir, bu yüzden onları `==` ile karşılaştırmayın. |
| `core/engine`, `core/terminal`, `core/driver`, `core/accessibility` | **Yarı iç** | İleri düzey kullanım için public tutuluyor. Mümkünse kök paketteki karşılıklarını kullanın. |
| `automation`, `uitest`, `session`, `cmd/limoni-mcp` | **Deneysel** | v0.3 ile geldi. Agent'lar ve testler kullandıkça seçiciler ve kablo protokolü değişebilir. |
| `graphics`, `animation`, `core/grapheme` | **Oturuyor** | İşlev olarak kararlı, API yüzeyi küçük. |

## v1.0'a giden yol

v1.0 şu koşullar sağlanınca etiketlenecek:

1. Örnek izolasyonu tamamlanır, yani tek bir süreçte iki Limoni uygulaması
   çalışabilir, ve immediate mode context desteği kazanır (`RunWithContext`).
2. Terminal yetenekleri ortam değişkenlerinden tahmin edilmek yerine terminale
   sorularak öğrenilir.
3. Metin kesen widget'lar grapheme cluster'ları doğru işler.
4. Deneysel paketler bu repo dışında en az bir uygulamada kullanılır ve bu
   kullanımdan gelen geri bildirimle API'leri bir kez gözden geçirilir.

v1.0'dan sonra kırıcı değişiklikler, Go modüllerinin gerektirdiği gibi yeni bir
major sürüm (`/v2`) ister.

## Desteklenen Go sürümleri

Modül derlenebildiği en eski Go sürümünü bildirir (şu an **1.25**). CI bunu o
sürümde `GOTOOLCHAIN=local` ile doğrular. Alt sınırın yükseltilmesi changelog'da
ayrıca belirtilir.
