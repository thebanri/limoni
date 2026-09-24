# 🎨 2D & 3D Graphics, Canvas, and Image Engine

Moving beyond standard text-only TUI libraries, Limoni features a **high-resolution 2D Braille Canvas**, a **3D Mesh & Software Shader Engine**, and **native terminal image protocol drivers (Kitty, Sixel, iTerm2)**.

---

## 1. 2D High-Resolution Braille Canvas (`widgets.Canvas`)

Unicode Braille patterns (`⠀` - `⣿`) form a $2 \times 4$ dot sub-pixel grid within a single terminal cell. An $80 \times 24$ terminal window yields an effective drawing resolution of $160 \times 96$ pixels.

```go
import (
	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

// Create canvas
canvas := widgets.NewCanvas(width, height)

// Draw lines
canvas.DrawLine(x0, y0, x1, y1, limoni.Fg(limoni.RGB(0, 255, 200)))

// Draw circles
canvas.DrawCircle(centerX, centerY, radius, limoni.Fg(limoni.RGB(255, 215, 0)))

// Draw filled depth-tested triangles (Z-Buffer supported)
canvas.DrawFilledTriangleDepth(v0, v1, v2, z0, z1, z2, style)

// Draw depth-tested affine-textured triangles (Z-Buffer + UV texture mapping)
canvas.DrawTexturedTriangleDepth(v0, v1, v2, z0, z1, z2, uv0, uv1, uv2, textureImage)
```

### Markers

Drawing always happens at 2×4 dots per cell; `canvas.Marker` decides how they are shown, so every drawing call works with every marker:

| Marker | Dots per cell | Characters | Use |
| :--- | :--- | :--- | :--- |
| `MarkerBraille` (default) | 2×4 | `U+2800–U+28FF` | finest detail |
| `MarkerSextant` | 2×3 | `U+1FB00–U+1FB3B` | solid fills, fonts without Braille |
| `MarkerQuadrant` | 2×2 | `▘▝▀▖▌▞▛▗▚▐▜▄▙▟█` | the widest font support of the block markers |
| `MarkerHalfBlock` | 1×2 | `▀▄█` | |
| `MarkerBlock` | 1×1 | `█` | |

`LineChart`, `PieChart` and `Viewer3D` have the same `Marker` field.

---

## 2. 3D Mesh Loading & Rendering Pipeline (`graphics` & `Viewer3D`)

Limoni natively parses popular 3D file formats:
- **Wavefront OBJ** (`.obj`): `graphics.LoadOBJ(path)` / `graphics.ParseOBJ(reader)`
- **Stereolithography STL** (`.stl`): `graphics.LoadSTL(path)` / `graphics.ParseSTL(bytes)`
- **Stanford PLY** (`.ply`): `graphics.LoadPLY(path)` / `graphics.ParsePLY(reader)`
- **glTF binary** (`.glb`): `graphics.LoadGLB(path)`
- **Built-in Geometric Primitives**: `graphics.NewCube(size)`, `graphics.NewPyramid(base, height)`, `graphics.NewSphere(radius, rings, sectors)`, `graphics.NewTorus(r1, r2, radial, tubular)`

`Model3D.UVs` are in image space — V grows down, `(0,0)` is the top-left texel — as glTF has them; `LoadOBJ` flips OBJ's OpenGL-style V on the way in.

### High-Level `Viewer3D` Widget
Transform, illuminate, and render 3D models with a single declarative widget:

```go
mesh, _ := graphics.LoadOBJ("assets/model.obj")
_ = mesh.LoadTexture("assets/texture.png")

viewer := &widgets.Viewer3D{
	Model:     mesh,
	RotX:      rotX,
	RotY:      rotY,
	Shading:   widgets.ShadingTexture, // ShadingWireframe, ShadingFlat, ShadingLambert, ShadingGouraud
	Wireframe: true,                   // edges over the shading
	Pixels:    true,                   // a picture over kitty/iTerm2/Sixel where the terminal has one
}
f.RenderWidget(viewer, f.Area())
```

Every face is fanned into triangles, cut at the near plane rather than dropped when the camera gets close, back-face culled and depth tested, whatever the shading. Drawing does not allocate once the scratch buffers have grown to the model.

With `Pixels`, the model is rendered at 8×16 pixels a cell into an RGBA picture and sent with the terminal's image protocol; without one it falls back to dots. A still model is encoded once. A moving one is encoded every frame, which costs what the protocol costs: about 7 ms and 2.7 MB per frame for a 60×24 area with kitty, measured with `BenchmarkViewer3DPixelsKitty`.

### 3D Render Shading Models

1. **Wireframe**: Renders model polygon edges.
2. **Solid Color**: Fills polygon faces with solid colors or depth-based palettes.
3. **Lambertian Diffuse Shading**: Lights each face by its outward normal against a directional light (`graphics.Light`, whose `Direction` points towards the light). Note that `graphics.CalculateNormal` returns the *inward* normal for Limoni's front-face winding; negate it before lighting.
4. **Gouraud Shading**: Lights each vertex with the average normal of the faces around it and interpolates across the triangle, so curved surfaces shade smoothly instead of facet by facet.
5. **Texture Mapping & Z-Buffer**: Maps UV coordinates from image textures directly onto 3D polygons with depth buffering (`canvas.DrawTexturedTriangleDepth`).

---

## 3. Terminal Image Drivers & The Lower Half-Block Standard

Limoni dynamically queries terminal capabilities to select the highest fidelity image rendering protocol:
- **Kitty Graphics Protocol**: Direct 24-bit RGB pixel image streaming on modern terminals.
- **Sixel Graphics Protocol**: Indexed pixel graphics for classic DEC VT terminals and xterm.
- **iTerm2 Inline Images Protocol**: Base64 image payload transmission for macOS iTerm2.
- **Lower Half-Block (`▄`) Baseline Standard**: $1 \times 2$ subpixel representation with inverted color mapping.

### Why Lower Half-Block (`▄` / `U+2584`)?
Traditional half-block renderers often use the upper half-block (`▀`). In fonts with non-standard baseline metrics (notably on macOS Terminal.app and iTerm2), upper half-blocks can exceed line-height bounds, resulting in line-overflow, trailing artifacts, and terminal scrolling jitter. 

Limoni establishes the **Lower Half-Block (`▄`)** as the standard:
- Foreground color renders the lower half of the character cell.
- Background color renders the upper half.
- By pinning the glyph to the font baseline, rendering stays 100% within character cell bounds across all operating systems and terminal emulators without vertical bar overflow.

---

## 4. 240 FPS Ultra-High Framerate Mode

Limoni's zero-allocation 1D flat buffer and differential ANSI streaming allow running interactive animations, particle systems, and 3D viewers at up to **240 frames per second**:

```bash
go run github.com/thebanri/limoni/examples/3d_viewer@latest -fps 240
```

Even at 240 FPS, Limoni limits terminal serial I/O strictly to dirty cells, keeping CPU utilization exceptionally low.
