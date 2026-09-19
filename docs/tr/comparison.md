# Neden Limoni?

| Özellik / Hedef | 🍋 Limoni (Go) | 🫧 Bubble Tea **v1** + Lip Gloss v1 (Go) | 🌈 Bubble Tea **v2** + Ultraviolet (Go) | 🐀 Ratatui **0.30** (Rust) |
| :--- | :--- | :--- | :--- | :--- |
| **Dil ve Araçlar** | **Go (Yerel)** | Go (Yerel) | Go (Yerel) | Rust (Yerel) |
| **Render Mimarisi** | **1D Düz Matris + Adaptif ANSI Diff** | String birleştirme / TEA | Hücre tamponu + ncurses tarzı diff | Çift Tamponlu Immediate Mode |
| **Kritik Yol Tahsisatı**| **`0 B/op` (Sıfır Alloc)** | Yüksek heap tahsisatı | Azaltılmış; sıfır-alloc bir tasarım hedefi değil — Ultraviolet her glif için bir `Cell` tahsis ediyor | Stack / RAII |
| **Düzen Paradigması** | **Bildirimsel Flexbox & Yığın Çözücü** | String dilimleme (`JoinHorizontal/Vertical`) | Cassowary kısıt çözücü | Kısıt çözücü (Constraint solver) |
| **Fare Etkileşimi** | **Hücresel Koordinat & Z-Index Yönlendirme** | Yok (manuel koordinat hesabı) | SGR fare olayları; dahili hit-testing yok | Manuel koordinat |
| **Çift Tampon & Diff** | **Mikrosaniye altı diff + Adaptif tam akış** | Yok (tüm string stdout'a dökülür) | Hücre diff + `ECH`/`REP`/`ICH`/`DCH` + kaydırma optimizasyonu | Çift tamponlu diff |
| **Grapheme Cluster** | **UAX #29 cluster'ları, Unicode 17.0, resmî 766 kırılım testinin tamamı geçiyor** + Mod 2027 isteği; desteklemeyen terminaller için her cluster'dan sonra imleç yeniden konumlanır | `uniseg` | `uniseg` + Mod 2027 müzakeresi | `unicode-width` |
| **Yetenek Tespiti** | Yalnızca ortam değişkenleri | Ortam / terminfo | Çalışma anında sorgulama (terminfo'suz) | terminfo / crossterm |
| **Büyük Veri / Tablolar**| **1M satır sanallaştırma (sürekli kaydırma altında ~2,6 ms/kare)** | Yüksek GC yükü | v1'e göre iyileştirilmiş | Her karede tüm satırları yeniden kurar — `Table` satır iterator'ının sahibidir |
| **3D & Vektör Grafikleri**| **Dahili 3D (OBJ/STL/PLY/GLB) & Shaders** | Harici eklenti gerekir | Harici eklenti gerekir | Eklenti gerekir |
| **Erişilebilirlik (A11y)** | **Dahili Semantik Ağaç ve Ekran Okuyucu** | Kısıtlı / Manuel | Kısıtlı / Manuel | Deneysel |
| **Harici Bağımlılık** | **2 (`golang.org/x/sys`, `golang.org/x/crypto`)** | ~15 dolaylı modül | ~15 dolaylı modül | crates.io grafiği |
| **Eşzamanlılık (Concurrency)** | **Kilit-Serbest Kanallar / İş Parçacığı Güvenli** | Tek iş parçacıklı TEA | Tek iş parçacıklı TEA | Manuel iş parçacığı yönetimi |

> **Bubble Tea v2 sütunu hakkında:** bu satırlar Limoni'nin kendi ölçümlerinden değil, üst akış dokümantasyonundan alınmıştır. Charm, render motorunu hücre tabanlı diff yapan [Ultraviolet](https://github.com/charmbracelet/ultraviolet) üzerine yeniden inşa etti; dolayısıyla Limoni'nin **v1**'e karşı açtığı mimari fark **v2** için olduğu gibi geçerli değildir.
>
> Ultraviolet artık burada **ölçülüyor** ve karşılaştırılabilir render iş yüklerinde Limoni 1,9–20 kat daha hızlı, üstelik kare başına belirgin biçimde daha az bayt yayıyor ([§2.4](../benchmark-methodology.md#24-ultraviolet)). Bu bir **Bubble Tea v2 sonucu değildir**: bir v2 programı ayrıca kendi çalışma zamanını, mesaj dağıtımını ve view kurulumunu da öder; bunların hiçbiri burada ölçülmüyor. Bu depodaki hiçbir koşucu Bubble Tea v2'yi link etmiyor, dolayısıyla v2'nin kendisine karşı her performans iddiasını kanıtlanmamış sayın.

## 🍋 Limoni Composable (Lego UI) vs. 🎀 Charm Lip Gloss **v1**

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
| **Render Hattı** | **Çift tamponlu ANSI diffing (`~14 µs` seyrek, `~50 µs` tam ekran)** | Tüm terminale string dökme (ekranda titreme yapar) |
| **Geçiş Köprüsü** | **`compat/bubbletea` akıcı stil oluşturucu** | Charm ekosistemi yerel standardı |

### Neden Sıfır Bellek Tahsisatlı Mimari Önemlidir?
1. **Garbage Collector Donmalarını (GC Stutter) Yok Eder**: Lipgloss her kenarlık, boşluk ve yatay birleştirme için bellekte yeni string nesneleri tahsis eder. 60 FPS çalışan hareketli bir ekranda bu durum saniyede yüz binlerce nesne üreterek Go GC'sini devreye sokar ve arayüzde mikro donmalara (stutter) yol açar. Limoni bileşenleri çağrı yığınında (call stack) çalışır ve doğrudan yeniden kullanılan 1D tampona yazar; **sıfır bellek tahsisatı (`0 B/op`)** garantilenir.
2. **Kutudan Çıkan Fare ve Tıklama Desteği**: Lipgloss sadece düz bir metin ürettiğinden kullanıcının nereye tıkladığını bilemez. Limoni bileşenleri çizildikleri ekran alanını (`cell.Rect`) otomatik kaydeder; tıklama, üzerine gelme (hover), sürükleme ve tekerlek olayları doğrudan ilgili bileşenin callback'ine yönlendirilir.
