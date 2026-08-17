# 🍋 Limoni Örnek Uygulamalar Rehberi

Limoni deposu, `examples/` dizini altında yer alan 12 adet tam teşekküllü, bağımsız ve canlı çalışan örnek uygulama içerir.

---

## 📂 Örnek Uygulamalar Listesi

| Dizin | Başlık | Açıklama ve Özellikler | Çalıştırma |
| :--- | :--- | :--- | :--- |
| **[`examples/3d_viewer`](../../examples/3d_viewer)** | **3D Yazılım Rasterizer** | `.obj`, `.stl`, `.ply` 3D model yükleme, derinlik tamponlu Gouraud/Lambert gölgelendirme, serbest fare yörünge kontrolü. | `go run ./examples/3d_viewer` |
| **[`examples/paint`](../../examples/paint)** | **Noktasal Paint Stüdyosu** | $2 \times 4$ Braille alt-piksel tuval, 10 temel renk, özel Hex modalı, canlı tutuş önizlemesi (rubber-band), çizgi/daire/kutu/silgi ve 25 adımlı geri alma. | `go run ./examples/paint` |
| **[`examples/dashboard`](../../examples/dashboard)** | **Sistem Telemetri Paneli** | 4 motorlu grafik görselleştirici (Braille çizgi, tam boy ekolayzır spektrumu, 2x2 sparkline gridi, dolu alan), gerçek Linux `/proc` telemetrisi ve interaktif süreç tablosu. | `go run ./examples/dashboard` |
| **[`examples/table_virtual`](../../examples/table_virtual)** | **1M Satırlı Sanal Tablo** | 1.000.000 log kaydını sıfır bellek tahsisatı ve 60 FPS akıcılıkla sanal kaydırma (viewport virtualization) ile görüntüleme. | `go run ./examples/table_virtual` |
| **[`examples/todo`](../../examples/todo)** | **TEA Görev Yöneticisi** | The Elm Architecture (`Init`, `Update`, `View`), bulanık arama (fuzzy search), durum değiştirme ve klavye gezinimi. | `go run ./examples/todo` |
| **[`examples/demo`](../../examples/demo)** | **Kapsamlı Vitrin Demosu** | 3D modeller, sistem göstergeleri, Matrix yağmuru, form kontrolleri ve komut paleti içeren çok sekmeli ana demo. | `go run ./examples/demo` |
| **[`examples/forms`](../../examples/forms)** | **Form & Girdi Kontrolleri** | `TextInput`, `TextArea`, `Checkbox`, `RadioGroup`, `Select`, `Slider` ve odak yönetimi. | `go run ./examples/forms` |
| **[`examples/layer_demo`](../../examples/layer_demo)** | **Katman & Modal Sistemi** | Z-Index tabanlı katman yönetimi, üst üste binen pencereler (`LayerModal`) ve dışa tıklama ile kapanma. | `go run ./examples/layer_demo` |
| **[`examples/animation`](../../examples/animation)** | **Animasyon & Fizik** | Fizik tabanlı yaylar, renk enterpolasyonu ve 60 FPS akıcı easing eğrileri (`animation.Float`, `animation.Color`). | `go run ./examples/animation` |
| **[`examples/custom_widget`](../../examples/custom_widget)** | **Özel Widget Geliştirme** | `widgets.Widget` arayüzü (`Draw`, `SizeHint`) uygulanarak geliştirilmiş analog hız göstergesi. | `go run ./examples/custom_widget` |
| **[`examples/ssh_server`](../../examples/ssh_server)** | **Çok Kullanıcılı SSH Sunucusu** | Ağ soketi üzerinden bağlanan kullanıcılara izole 60 FPS ANSI diff akışı sunan uzak terminal sunucusu. | `go run ./examples/ssh_server` |
| **[`examples/wasm`](../../examples/wasm)** | **WebAssembly Tarayıcı Demosu** | WebAssembly olarak derlenip xterm.js üzerinden web tarayıcısında çalışan Limoni uygulaması. | `go run ./examples/wasm` |
