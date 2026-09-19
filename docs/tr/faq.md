# Görsel İşleme İpuçları ve SSS (Rendering Quirks & FAQ)

## 1. Çizgiler veya 3D modeller bazı terminallerde neden ince çizgili (hairline gap) veya delikli görünür?
Standart terminal emülatörlerinde varsayılan monospace yazı tipi satır yüksekliği (`line-height` / hücre dolgusu) genellikle bitişik karakter satırları arasına 1–2 piksellik boşluk ekler. Bitişik Braille alt-piksel matrisleri veya yarım-bloklar (`▄`, `▀`) çizilirken bu boşluk yüzeylerin delikli veya ızgara gibi görünmesine yol açabilir.

### Limoni Bu Sorunu Nasıl Çözer: Alt Yarım-Blok (`▄`, U+2584) Taban Standardı
Geleneksel TUI kütüphaneleri genellikle Üst Yarım-Blok (`▀`, `U+2580`) kullanır. Yazı tipi motorları karakter gliflerini hücrenin **taban çizgisine** (baseline) kilitlediğinden, satır yüksekliği eklendiğinde boşluk hücrenin *üst kısmında* oluşur ve `▀` karakterini üst satırdan ayırır.

Limoni tüm piksel ve 3D çizimlerini **Alt Yarım-Blok (`▄`, `U+2584`)** standardına geçirmiştir:
- **Üst Piksel Rengi:** Hücre arkaplanına (`Cell.Bg`) yazılır.
- **Alt Piksel Rengi:** Hücre önplanına (`Cell.Fg`) yazılır.
- **Karakter:** `▄` (Alt Yarım Blok).

Arka plan rengi karakter hücresinin tamamını kapladığı ve `▄` glifi tam taban çizgisine oturduğu için, gevşek satır yüksekliğine sahip terminallerde dahi pikseller sıfır aralıkla pürüzsüzce birleşir.

## 2. Kusursuz Görsel Deneyim İçin Önerilen Terminal Ayarları
Limoni'nin 3D rasterizasyonunu, grafiklerini ve Braille vektör çizimlerini en yüksek netlikte deneyimlemek için:

* **Satır Yüksekliğini 1.0 Yapın:** Terminalinizin yapılandırma dosyasında `line-height` / `cell-height` değerini `1.0` (veya `%100` / 0 piksel dikey boşluk) olarak ayarlayın.
* **Önerilen Modern Terminaller:**
  - **[Ghostty](https://ghostty.org):** Kutu çizimlerini, Braille ve blok gliflerini sıfır hücre boşluğuyla kusursuz çizen modern GPU terminali.
  - **[Kitty](https://sw.kovidgoyal.net/kitty/):** Yüksek performanslı OpenGL motoru, yerleşik grafik protokolleri ve bitişik glif desteği.
  - **[WezTerm](https://wezfurlong.org/wezterm/):** Mükemmel font fallback ve bitişik kutu glifi desteği.
  - **[Alacritty](https://alacritty.org):** `alacritty.toml` içinde `font.offset.y: 0` ve satır yüksekliği 1.0 kullanın.
* **Önerilen Yazı Tipleri:** [JetBrains Mono](https://www.jetbrains.com/lp/mono/), [Fira Code](https://github.com/tonsky/FiraCode) veya yamalanmış [Nerd Font](https://www.nerdfonts.com/) monospace fontları.

## 3. Limoni Hızlı Animasyonlarda 60+ FPS Performansı Nasıl Korur?
Limoni eşik tabanlı bir **Adaptif Flush Motoruna (Adaptive Flush Engine)** sahiptir:
* **Seyrek Diffing (`dirtyRatio < 0.45`):** Yazma, imleç yanıp sönmesi veya sayaç güncellemeleri gibi seyrek durumlarda sadece değişen hücreleri hesaplayıp hassas imleç sıçramaları (`CUP`) gönderir (**`~14.2 µs`**, 0 B/op).
* **Tam Akış Yenileme (`dirtyRatio >= 0.45`):** 3D model dönüşü veya hızlı kaydırma gibi ekranın %45'inden fazlasının değiştiği durumlarda imleç sıçramaları terk edilir; DEC senkronize güncelleme modu (`\x1b[?2026h`) ve ana konuma dönüş (`\x1b[H`) ile ardışık tam akış gönderilir. Böylece yırtılma ve titreme olmadan **`0 B/op`** hız korunur.

## 4. Emoji, bayraklar ve aksanlı harfler
Limoni her hücrede bir **grapheme cluster** tutar — kaç kod noktasından oluşursa oluşsun, okuyucunun tek karakter olarak gördüğü şey. `🇹🇷` (iki bölgesel gösterge), `👨‍👩‍👧` (ZWJ ile birleşmiş beş kod noktası), `👍🏽` (emoji + ten rengi) ve `e` + U+0301 olarak yazılmış `é` birer hücre kaplar; emojiler iki sütun genişliğindedir. Rune rune yürümek bayrağı iki harf olarak çiziyor, aileyi altı sütun ölçüyor ve birleşik aksanı düşürüyordu.

* **Kurallar:** bölütleme Unicode 17.0 için UAX #29'u izler ve resmî `GraphemeBreakTest.txt`'nin 766 durumunun tamamını geçer. Bir cluster'ın genişliği en geniş kod noktasınınkidir (East Asian Width, emoji sunumu); VS16 iki sütuna, VS15 bire zorlar.
* **Saklama:** hücre hâlâ tek bir `rune` tutar. Çok kod noktalı bir cluster paylaşılan bir tabloya bir kez kaydedilir ve hücre ona bir tutamaç saklar; böylece `Cell` 16 bayt kalır ve tek kod noktaları — metnin neredeyse tamamı — tabloya hiç dokunmaz. Tablo yaklaşık bir milyon farklı cluster ile sınırlıdır; sonrasında yeni cluster'lar belleği büyütmek yerine ilk kod noktalarına düşer.
* **Terminaller:** Limoni mod 2027'yi (`CSI ? 2027 h`) ister; Ghostty, WezTerm, foot ve Contour bunu uygular. Desteklemeyen terminaller imleci kod noktası başına ilerletir ve bir aile emojisini altı sütun çizebilir. Bunun satırın geri kalanını kaydırmasını önlemek için diff her cluster'dan hemen sonra imleci yeniden konumlandırır. Böyle bir terminalde cluster'ın kendisi yine yanlış görünebilir, ama ondan sonraki hiçbir şey yerinden oynamaz.
* **Kapatmak:** `LIMONI_GRAPHEME=0` (ya da `cell.SetGraphemeClusters(false)`) hücre başına bir kod noktasına döner ve mod 2027 isteğini göndermez.
* **Henüz dönüştürülmedi:** `Buffer.SetString` ile çizilen ve `cell.StringWidth` ile ölçülen metin cluster'ları tanır. Metni rune sayısına göre kesen ya da yerleştiren widget'lar — `TextInput`, `TextArea`, tablo ve toast kırpması ve diğerleri — kestikleri yerde bir cluster'ı hâlâ bölebilir.
