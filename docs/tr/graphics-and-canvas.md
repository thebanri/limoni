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
canvas.DrawTexturedTriangleDepth(x0, y0, z0, u0, v0, x1, y1, z1, u1, v1, x2, y2, z2, u2, v2, dokuResmi)
```

---

## 2. 3D Mesh Yükleme & Çizim Hattı (`graphics` & `Viewer3D`)

Limoni popüler 3D dosya formatlarını doğal olarak ayrıştırır:
- **Wavefront OBJ** (`.obj`): `graphics.LoadOBJ(path)` / `graphics.ParseOBJ(reader)`
- **Stereolithography STL** (`.stl`): `graphics.LoadSTL(path)` / `graphics.ParseSTL(bytes)`
- **Stanford PLY** (`.ply`): `graphics.LoadPLY(path)` / `graphics.ParsePLY(reader)`
- **Dahili Geometrik Primitifler**: `graphics.NewCube(size)`, `graphics.NewPyramid(base, height)`, `graphics.NewSphere(radius, rings, sectors)`

### Yüksek Seviye `Viewer3D` Widget'ı
Tek bir bildirimsel widget ile 3D modelleri döndürün, ışıklandırın ve render edin:

```go
// 3D modeli yükle ve widget'a bağla
mesh, _ := graphics.LoadOBJ("assets/model.obj")

viewer := widgets.NewViewer3D(mesh).
	WithRotation(rotX, rotY, rotZ).
	WithShading("shaded").      // "textured", "wireframe", "solid", "shaded", "gouraud"
	WithWireframe(true).
	WithTexture("assets/texture.png")

// Mevcut kareye render et
f.RenderWidget(viewer, f.Area())
```

### 3D Gölgelendirme Modelleri

1. **Tel Çerçeve (Wireframe)**: Modelin çokgen kenarlarını çizer.
2. **Düz Renk (Solid Color)**: Çokgen yüzeylerini tek renk veya derinlik paletiyle boyar.
3. **Lambertian Difüz Gölgelendirme**: Gerçekçi aydınlatma için yüzey normalleri (`graphics.CalculateNormal`) ile yönlü ışıkları (`graphics.Light`) hesaplar (`canvas.DrawLambertTriangleDepth`).
4. **Gouraud Gölgelendirme**: Pürüzsüz aydınlatma geçişleri için üçgen köşeleri arasında barisentrik koordinatlarla renk interpolasyonu yapar (`canvas.DrawGouraudTriangleDepth`).
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
