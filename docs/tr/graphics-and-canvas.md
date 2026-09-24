# 🎨 2D & 3D Grafik, Canvas ve Resim Motoru

Standart metin tabanlı TUI kütüphanelerinin ötesine geçen Limoni; **yüksek çözünürlüklü 2D Braille Canvas**, **3D Mesh & Yazılımsal Shader Motoru** ve **doğal terminal resim protokolü sürücüleri (Kitty, Sixel, iTerm2)** sunar.

---

## 1. 2D Yüksek Çözünürlüklü Braille Canvas (`widgets.Canvas`)

Unicode Braille karakterleri (`⠀` - `⣿`), tek bir terminal karakter hücresi içinde $2 \times 4$ noktalı alt-piksel ızgarası oluşturur. $80 \times 24$ boyutundaki bir terminal penceresinde $160 \times 96$ piksel çözünürlüğünde grafik çizilebilir.

```go
import (
	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

// Canvas oluşturma
canvas := widgets.NewCanvas(width, height)

// Çizgi çizimi
canvas.DrawLine(x0, y0, x1, y1, limoni.Fg(limoni.RGB(0, 255, 200)))

// Çember çizimi
canvas.DrawCircle(centerX, centerY, radius, limoni.Fg(limoni.RGB(255, 215, 0)))

// Derinlik tamponlu (Z-Buffer) üçgen çizimi
canvas.DrawFilledTriangleDepth(v0, v1, v2, z0, z1, z2, style)

// Z-Buffer ve afin UV doku eşlemeli üçgen çizimi
canvas.DrawTexturedTriangleDepth(v0, v1, v2, z0, z1, z2, uv0, uv1, uv2, dokuResmi)
```

### Marker'lar

Çizim her zaman hücre başına 2×4 noktayla yapılır; noktaların nasıl gösterileceğini `canvas.Marker` belirler, bu yüzden her çizim çağrısı her marker'la çalışır:

| Marker | Hücre başına nokta | Karakterler | Kullanım |
| :--- | :--- | :--- | :--- |
| `MarkerBraille` (varsayılan) | 2×4 | `U+2800–U+28FF` | en ince ayrıntı |
| `MarkerSextant` | 2×3 | `U+1FB00–U+1FB3B` | dolu alanlar, Braille'i olmayan fontlar |
| `MarkerQuadrant` | 2×2 | `▘▝▀▖▌▞▛▗▚▐▜▄▙▟█` | blok marker'lar içinde en geniş font desteği |
| `MarkerHalfBlock` | 1×2 | `▀▄█` | |
| `MarkerBlock` | 1×1 | `█` | |

`LineChart`, `PieChart` ve `Viewer3D` aynı `Marker` alanına sahiptir.

---

## 2. 3D Mesh Yükleme & Çizim Hattı (`graphics` & `Viewer3D`)

Limoni popüler 3D dosya formatlarını doğal olarak ayrıştırır:
- **Wavefront OBJ** (`.obj`): `graphics.LoadOBJ(path)` / `graphics.ParseOBJ(reader)`
- **Stereolithography STL** (`.stl`): `graphics.LoadSTL(path)` / `graphics.ParseSTL(bytes)`
- **Stanford PLY** (`.ply`): `graphics.LoadPLY(path)` / `graphics.ParsePLY(reader)`
- **glTF binary** (`.glb`): `graphics.LoadGLB(path)`
- **Dahili Geometrik Primitifler**: `graphics.NewCube(size)`, `graphics.NewPyramid(base, height)`, `graphics.NewSphere(radius, rings, sectors)`, `graphics.NewTorus(r1, r2, radial, tubular)`

`Model3D.UVs` görüntü uzayındadır — V aşağı doğru büyür, `(0,0)` sol üst texel'dir — glTF'deki gibi; `LoadOBJ`, OBJ'nin OpenGL tarzı V'sini yüklerken çevirir.

### Yüksek Seviye `Viewer3D` Widget'ı
Tek bir bildirimsel widget ile 3D modelleri döndürün, ışıklandırın ve render edin:

```go
mesh, _ := graphics.LoadOBJ("assets/model.obj")
_ = mesh.LoadTexture("assets/texture.png")

viewer := &widgets.Viewer3D{
	Model:     mesh,
	RotX:      rotX,
	RotY:      rotY,
	Shading:   widgets.ShadingTexture, // ShadingWireframe, ShadingFlat, ShadingLambert, ShadingGouraud
	Wireframe: true,                   // gölgelendirmenin üstüne kenarlar
	Pixels:    true,                   // terminalde varsa kitty/iTerm2/Sixel üzerinden resim
}
f.RenderWidget(viewer, f.Area())
```

Gölgelendirme ne olursa olsun her yüz üçgenlere bölünür, kamera yaklaşınca atılmak yerine near plane'de kırpılır, arka yüz elemesi ve derinlik testi yapılır. Geçici tamponlar modelin boyutuna ulaştıktan sonra çizim bellek ayırmaz.

`Pixels` ile model hücre başına 8×16 piksel olarak bir RGBA resme çizilip terminalin görüntü protokolüyle gönderilir; protokol yoksa noktalara döner. Duran model bir kez kodlanır. Hareket eden model her karede kodlanır ve bu protokolün maliyeti kadardır: kitty ile 60×24'lük bir alan için kare başına yaklaşık 7 ms ve 2.7 MB (`BenchmarkViewer3DPixelsKitty` ile ölçüldü).

### 3D Gölgelendirme Modelleri

1. **Tel Çerçeve (Wireframe)**: Modelin çokgen kenarlarını çizer.
2. **Düz Renk (Solid Color)**: Çokgen yüzeylerini tek renk veya derinlik paletiyle boyar.
3. **Lambertian Difüz Gölgelendirme**: Her yüzü dışa bakan normaliyle yönlü ışığa (`graphics.Light`; `Direction` ışığa doğru bakar) göre aydınlatır. `graphics.CalculateNormal`, Limoni'nin ön yüz sarım yönü için *içe* bakan normali döndürür; aydınlatmadan önce işaretini çevirin.
4. **Gouraud Gölgelendirme**: Her köşeyi çevresindeki yüzlerin ortalama normaliyle aydınlatıp üçgen boyunca enterpole eder; eğri yüzeyler yüz yüz değil pürüzsüz gölgelenir.
5. **Doku Haritalama & Z-Buffer**: Resim dokularından alınan UV koordinatlarını Z-Buffer derinlik kontrolüyle doğrudan 3D çokgenlere eşler (`canvas.DrawTexturedTriangleDepth`).

---

## 3. Terminal Resim Protokolleri & Lower Half-Block Standardı

Limoni, en yüksek görsel doğruluğu sunan terminal protokolünü otomatik tespit eder:
- **Kitty Grafik Protokolü**: Modern terminallerde doğrudan 24-bit RGB piksel akışı.
- **Sixel Grafik Protokolü**: Klasik DEC VT terminalleri ve xterm için indekslenmiş piksel grafikleri.
- **iTerm2 Satır İçi Resim Protokolü**: macOS iTerm2 için Base64 resim aktarımı.
- **Lower Half-Block (`▄`) Taban Çizgisi Standardı**: Ters çevrilmiş renk haritalamasıyla $1 \times 2$ alt-piksel gösterimi.

### Neden Lower Half-Block (`▄` / `U+2584`)?
Geleneksel yarım blok çiziciler sıklıkla üst yarım blok (`▀`) karakterini kullanır. Standart dışı satır yüksekliği metriklerine sahip yazı tiplerinde (özellikle macOS Terminal.app ve iTerm2'de), üst yarım bloklar satır sınırlarını aşarak satır atlama, alt çizgilere taşma ve kaydırma titremelerine yol açar.

Limoni, **Lower Half-Block (`▄`)** karakterini standart olarak belirlemiştir:
- Ön plan rengi hücrenin alt yarısını çizer.
- Arka plan rengi hücrenin üst yarısını çizer.
- Karakter yazı tipi taban çizgisine (baseline) sabitlendiği için, dikey bar taşması olmadan tüm işletim sistemlerinde ve emülatörlerde %100 hücre sınırları içinde kalır.

---

## 4. 240 FPS Ultra-Yüksek Kare Hızı Modu

Limoni'nin sıfır bellek tahsisatlı düz 1D tamponu ve diferansiyel ANSI akışı; interaktif animasyonların, parçacık sistemlerinin ve 3D görüntüleyicilerin saniyede **240 kareye** (240 FPS) kadar akıcı çalışmasını sağlar:

```bash
go run github.com/thebanri/limoni/examples/3d_viewer@latest -fps 240
```

240 FPS hızında dahi Limoni, terminal seri çıkışını yalnızca değişen (dirty) hücrelerle sınırlar ve işlemci kullanımını minimumda tutar.
