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
```

---

## 2. 3D Mesh Loading & Rendering Pipeline (`graphics` & `Viewer3D`)

Limoni natively parses popular 3D file formats:
- **Wavefront OBJ** (`.obj`): `graphics.LoadOBJ(path)` / `graphics.ParseOBJ(reader)`
- **Stereolithography STL** (`.stl`): `graphics.LoadSTL(path)` / `graphics.ParseSTL(bytes)`
- **Stanford PLY** (`.ply`): `graphics.LoadPLY(path)` / `graphics.ParsePLY(reader)`
- **Built-in Geometric Primitives**: `graphics.NewCube(size)`, `graphics.NewPyramid(base, height)`, `graphics.NewSphere(radius, rings, sectors)`

### High-Level `Viewer3D` Widget
Transform, illuminate, and render 3D models with a single declarative widget:

```go
// Load 3D model and bind to widget
mesh, _ := graphics.LoadOBJ("assets/model.obj")

viewer := widgets.NewViewer3D(mesh).
	WithRotation(rotX, rotY, rotZ).
	WithShading("shaded").      // "textured", "wireframe", "solid", "shaded", "gouraud"
	WithWireframe(true).
	WithTexture("assets/texture.png")

// Render onto current frame
f.RenderWidget(viewer, f.Area())
```

### 3D Render Shading Models

1. **Wireframe**: Renders model polygon edges.
2. **Solid Color**: Fills polygon faces with solid colors or depth-based palettes.
3. **Lambertian Diffuse Shading**: Computes surface normals (`graphics.CalculateNormal`) against directional lights (`graphics.Light`) for realistic illumination (`canvas.DrawLambertTriangleDepth`).
4. **Gouraud Shading**: Interpolates colors across triangle vertices using barycentric coordinates for smooth lighting transitions (`canvas.DrawGouraudTriangleDepth`).
5. **Texture Mapping**: Maps UV coordinates from PNG/JPEG image textures directly onto 3D polygons (`canvas.DrawTexturedTriangle`).

---

## 3. Terminal Image Drivers

Limoni dynamically queries terminal capabilities to select the highest fidelity image rendering protocol:
- **Kitty Graphics Protocol**: Direct 24-bit RGB pixel image streaming on modern terminals.
- **Sixel Graphics Protocol**: Indexed pixel graphics for classic DEC VT terminals and xterm.
- **iTerm2 Inline Images Protocol**: Base64 image payload transmission for macOS iTerm2.
- **Half-Block ANSI Fallback**: $1 \times 2$ ANSI half-block (`▀`, `▄`) fallback ensuring pixel-art representation on any standard terminal emulator.
