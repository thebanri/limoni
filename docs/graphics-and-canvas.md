# 🎨 2D & 3D Grafik, Canvas ve Resim Motoru (Graphics & Canvas)

Limoni, standart metin tabanlı TUI kütüphanelerinin ötesine geçerek terminal içinde **Braille yüksek çözünürlüklü 2D Canvas**, **3D Mesh & Shader Motoru** ve **Protokol Seviyesinde Resim Sürücüleri (Kitty, Sixel, iTerm2)** sunar.

---

## 1. 2D Yüksek Çözünürlüklü Braille Canvas (`widgets.Canvas`)

Braille Unicode karakterleri (`⠀` - `⣿`), tek bir terminal hücresinde $2 \times 4$ piksellik bir ızgara oluşturur. Böylece $80 \times 24$ boyutundaki bir terminalde $160 \times 96$ piksel çözünürlük elde edilir.

```go
// Canvas oluştur
canvas := widgets.NewCanvas(width, height)

// Çizgi çiz
canvas.DrawLine(x0, y0, x1, y1, cell.Style{Fg: cell.NewColorRGB(0, 255, 200)})

// Daire çiz
canvas.DrawCircle(centerX, centerY, radius, cell.Style{Fg: cell.NewColorRGB(255, 215, 0)})

// Dolu Üçgen çiz (Derinlik Z-Buffer destekli)
canvas.DrawFilledTriangleDepth(v0, v1, v2, z0, z1, z2, style)
```

---

## 2. 3D Mesh Yükleme & Render Motoru (`graphics`)

Limoni aşağıdaki 3D dosya formatlarını dahili olarak ayrıştırabilir:
- **Wavefront OBJ** (`.obj`): `graphics.LoadOBJ(path)` / `graphics.ParseOBJ(reader)`
- **Stereolithography STL** (`.stl`): `graphics.LoadSTL(path)` / `graphics.ParseSTL(bytes)`
- **Stanford PLY** (`.ply`): `graphics.LoadPLY(path)` / `graphics.ParsePLY(reader)`

### 3D Render Stilleri & Shader'lar

1. **Tel Kafes (Wireframe)**: Modelin kenar çizgilerini çizer.
2. **Dolu Renkli (Solid Prismatic)**: Yüzeyleri tek renk veya poligon paletleriyle doldurur.
3. **Lambertian Diffuse Gölgelendirme**: Yüzey normallerini (`graphics.CalculateNormal`) hesaplayarak yönsel ışık kaynağına (`graphics.Light`) göre gerçekçi gölgeler oluşturur (`canvas.DrawLambertTriangleDepth`).
4. **Gouraud Shading**: Üçgen köşeleri arasında barycentric enterpolasyonla pürüzsüz renk geçişleri sağlar (`canvas.DrawGouraudTriangleDepth`).
5. **Doku Kaplama (Texture Mapping)**: UV koordinatları ile PNG/prosedürel doku resimlerini poligonların üzerine giydirir (`canvas.DrawTexturedTriangle`).

---

## 3. Resim Sürücüleri (Image Drivers)

Limoni, terminalin yeteneklerine göre en yüksek kaliteli resim render protokolünü otomatik seçer:
- **Kitty Graphics Protocol**: Modern terminallerde tam RGB piksel resim aktarımı.
- **Sixel Graphics Protocol**: Klasik DEC VT terminalleri ve xterm için piksel grafikleri.
- **iTerm2 Inline Images Protocol**: macOS iTerm2 için base64 resim aktarımı.
- **Half-Block ANSI Fallback**: Piksel grafik desteği olmayan terminallerde $1 \times 2$ ANSI yarım blok (`▀`, `▄`) ile geriye dönük tam uyumluluk.
