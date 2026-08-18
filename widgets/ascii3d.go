package widgets

import (
	"math"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// Standard ASCII luminance and typography ramps
const (
	RampCanvasUI   = " .,'~:;!+*1czunxrjftvCYUJX0OZmwqkpdbhao#%&8B@$"
	RampTypography = " .,'`^~:;!+*?1zZcvunxrjftCYUJX0OZmwqkpdbhao%#8&B@$MW"
	RampDuck       = " ._~:;!+*?1zZpY3ygo0QPB#@"
	RampStandard   = " .:-=+*#%@"
	RampBlocks     = " ░▒▓█"
	RampBinary     = " 01"
	RampMinimal    = " .:+*#"
)

// Ascii3DMode defines the visual rendering technique used for 3D models.
type Ascii3DMode int

const (
	// ModeASCII renders using rich ASCII character typography ramps (CanvasUI style).
	ModeASCII Ascii3DMode = iota

	// ModeBlock renders using 2x vertical sub-cell Half-Block resolution (▀/▄ with dual TrueColor).
	ModeBlock

	// ModeDithered renders using retro Bayer 4x4 ordered dithering shading.
	ModeDithered

	// ModeBraille renders using 8x sub-pixel Unicode Braille dot patterns (⠀ to ⣿).
	ModeBraille
)

// Ascii3D (or AsciiObject) is a high-performance 3D vector-to-ASCII terminal renderer.
// It renders 3D models directly into terminal cells with dynamic lighting, Blinn-Phong
// specular highlights, contrast/exposure tone mapping, perspective projection,
// bounds-safe auto-clamping, and multiple rendering modes (ASCII, Block, Dithered, Braille).
type Ascii3D struct {
	// 3D Model geometry or file path
	Model graphics.Model3D
	Src   string // Path to .glb, .gltf, .obj, .stl, or .ply file

	// Rendering Mode:
	// - ModeASCII: Typography character ramps (default)
	// - ModeBlock: 2x vertical sub-cell Half-Block (▀/▄)
	// - ModeDithered: Retro Bayer 4x4 ordered dithering
	// - ModeBraille: 8x sub-pixel Unicode Braille dot matrix
	Mode Ascii3DMode

	// Transform & Animation
	Scale             float64 // Object size multiplier on screen (default: 4.2)
	XOffset           float64 // 3D X translation offset (default: 0.0)
	YOffset           float64 // 3D Y translation offset (default: 0.0)
	FloatIntensity    float64 // Floating hover amplitude (default: 0.0)
	FloatSpeed        float64 // Floating hover frequency (default: 2.0)
	RotationIntensity float64 // Floating angular wobble intensity (default: 1.0)
	AutoRotate        bool    // Continuously rotate around Y axis
	AutoRotateSpeed   float64 // Auto-rotation speed in degrees/sec (default: 30.0)
	Time              float64 // Elapsed time in seconds for animations

	// Manual Rotation Offsets (degrees)
	RotX float64
	RotY float64
	RotZ float64

	// Camera & Projection
	FOV            float64 // Field of view in degrees (default: 65.0)
	CameraDistance float64 // Distance along Z axis (default: 4.2)
	CellAspect     float64 // Terminal character height/width aspect ratio (default: 0.50)

	// Shading & Optics
	Contrast             float64 // Tone curve contrast exponent (default: 1.2)
	EdgeContrast         float64 // Silhouette / edge boost factor (default: 3.0)
	Exposure             float64 // Overall lighting exposure multiplier (default: 1.0)
	EnvironmentIntensity float64 // Ambient/environment lighting level (default: 1.0)
	Roughness            float64 // Surface roughness: lower = sharper specular highlight (default: 0.15)
	LightDirection       graphics.Vector3D // Primary directional light source

	// Rendering Mode & Palette
	Ascii     bool       // True for ASCII character ramps, False for solid HalfBlocks
	SubCell   bool       // True for 2x vertical Half-Block sub-pixel mode (legacy toggle)
	Braille   bool       // True for 8x Braille dot mode (legacy toggle)
	Colored   bool       // True for 24-bit TrueColor RGB, False for monochrome
	Invert    bool       // Invert luminance character ramp
	Color     cell.Color // Primary surface/diffuse fallback color (default: Yellow #ffd700)
	Highlight cell.Color // Specular highlight color (default: #066aff Electric Blue)
	Ramp      string     // Custom character luminance ramp (default: RampCanvasUI)
}

// AsciiObject is an alias for Ascii3D to match the declarative React/Three.js naming.
type AsciiObject = Ascii3D

// Draw implements the widgets.Widget interface, rendering the 3D model into the buffer.
func (a Ascii3D) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width < 2 || area.Height < 2 {
		return
	}

	model := a.Model
	if len(model.Vertices) == 0 {
		if a.Src != "" {
			if loaded, err := graphics.LoadModel(a.Src); err == nil {
				model = loaded
			}
		}
		if len(model.Vertices) == 0 {
			model = graphics.NewDuck()
		}
	}

	w := int(area.Width)
	h := int(area.Height)

	// Determine active mode
	effectiveMode := a.Mode
	if effectiveMode == ModeASCII && (a.SubCell || a.Braille) {
		if a.Braille {
			effectiveMode = ModeBraille
		} else if a.SubCell {
			effectiveMode = ModeBlock
		}
	}

	// Sub-pixel resolution multipliers
	xMultiplier := 1
	yMultiplier := 1
	if effectiveMode == ModeBlock {
		yMultiplier = 2
	} else if effectiveMode == ModeBraille {
		xMultiplier = 2
		yMultiplier = 4
	}

	subW := w * xMultiplier
	subH := h * yMultiplier
	totalCells := subW * subH

	// Parameters with sensible defaults
	scale := a.Scale
	if scale <= 0 {
		scale = 4.2
	}
	cellAspect := a.CellAspect
	if cellAspect <= 0 {
		cellAspect = 0.50
	}
	fov := a.FOV
	if fov <= 5.0 {
		fov = 65.0
	}
	cameraDist := a.CameraDistance
	if cameraDist <= 0.5 {
		cameraDist = 4.2
	}
	contrast := a.Contrast
	if contrast <= 0 {
		contrast = 1.2
	}
	exposure := a.Exposure
	if exposure <= 0 {
		exposure = 1.0
	}
	envIntensity := a.EnvironmentIntensity
	if envIntensity <= 0 {
		envIntensity = 1.0
	}
	roughness := a.Roughness
	if roughness < 0.01 {
		roughness = 0.01
	} else if roughness > 1.0 {
		roughness = 1.0
	}

	ramp := a.Ramp
	if ramp == "" {
		ramp = RampCanvasUI
	}
	rampRunes := []rune(ramp)
	rampLen := len(rampRunes)

	baseFallbackColor := a.Color
	if baseFallbackColor == 0 {
		baseFallbackColor = cell.NewColorRGB(255, 220, 20) // Default Duck Yellow
	}
	highlightColor := a.Highlight
	if highlightColor == 0 {
		highlightColor = cell.NewColorRGB(6, 106, 255) // #066aff Electric Blue
	}

	// Two-Light Studio Setup
	keyLightDir := a.LightDirection
	if keyLightDir.X == 0 && keyLightDir.Y == 0 && keyLightDir.Z == 0 {
		keyLightDir = graphics.Vector3D{X: -0.45, Y: 0.80, Z: 0.65}
	}
	keyLightDir = keyLightDir.Normalize()
	fillLightDir := graphics.Vector3D{X: 0.55, Y: 0.25, Z: 0.50}.Normalize()

	// Animation calculations
	time := a.Time
	floatY := 0.0
	wobbleX, wobbleZ := 0.0, 0.0
	if a.FloatIntensity > 0 {
		speed := a.FloatSpeed
		if speed <= 0 {
			speed = 2.0
		}
		floatY = math.Sin(time*speed) * a.FloatIntensity * 0.04
		rotInt := a.RotationIntensity
		if rotInt <= 0 {
			rotInt = 1.0
		}
		wobbleX = math.Sin(time*speed*0.7) * rotInt * 2.0
		wobbleZ = math.Cos(time*speed*0.5) * rotInt * 1.5
	}

	autoRotY := 0.0
	if a.AutoRotate {
		autoSpeed := a.AutoRotateSpeed
		if autoSpeed == 0 {
			autoSpeed = 30.0
		}
		autoRotY = time * autoSpeed
	}

	totRotX := a.RotX + wobbleX
	totRotY := a.RotY + autoRotY
	totRotZ := a.RotZ + wobbleZ

	// Pre-transform & project all vertices
	type projVert struct {
		x, y, z float64
		visible bool
	}
	projected := make([]projVert, len(model.Vertices))
	rotatedVerts := make([]graphics.Vertex3D, len(model.Vertices))

	fovRad := (fov * math.Pi / 180.0) / 2.0
	baseFocal := (float64(subH) * 0.5) / math.Tan(fovRad)

	wCenter := float64(subW) / 2.0
	hCenter := float64(subH) / 2.0

	// First pass: compute maximum extent of the rotated 3D model to prevent overflow
	maxExtentX := 0.0
	maxExtentY := 0.0
	for i, v := range model.Vertices {
		vr := v.RotateY(totRotY).RotateX(totRotX).RotateZ(totRotZ)
		rotatedVerts[i] = vr

		viewX := vr.X + a.XOffset
		viewY := vr.Y + floatY + a.YOffset
		viewZ := cameraDist - vr.Z
		if viewZ <= 0.1 {
			viewZ = 0.1
		}

		projX := math.Abs((viewX * baseFocal) / viewZ)
		projY := math.Abs((viewY * baseFocal) / viewZ) * (cellAspect * float64(yMultiplier) / float64(xMultiplier))

		if projX > maxExtentX {
			maxExtentX = projX
		}
		if projY > maxExtentY {
			maxExtentY = projY
		}
	}

	// Calculate safe scale multiplier that fits 92% of the viewport bounds
	userScale := scale / 2.0
	maxAllowedScale := 100.0
	if maxExtentX > 1e-4 {
		safeX := (wCenter * 0.92) / maxExtentX
		if safeX < maxAllowedScale {
			maxAllowedScale = safeX
		}
	}
	if maxExtentY > 1e-4 {
		safeY := (hCenter * 0.92) / maxExtentY
		if safeY < maxAllowedScale {
			maxAllowedScale = safeY
		}
	}

	effectiveScale := userScale
	if effectiveScale > maxAllowedScale {
		effectiveScale = maxAllowedScale
	}
	if effectiveScale < 0.2 {
		effectiveScale = 0.2
	}

	focalLength := baseFocal * effectiveScale
	aspectFactor := cellAspect * float64(yMultiplier) / float64(xMultiplier)

	for i, vr := range rotatedVerts {
		viewX := vr.X + a.XOffset
		viewY := vr.Y + floatY + a.YOffset
		viewZ := cameraDist - vr.Z

		if viewZ <= 0.1 {
			projected[i] = projVert{visible: false}
			continue
		}

		px := wCenter + (viewX*focalLength)/viewZ
		py := hCenter - ((viewY*focalLength)/viewZ)*aspectFactor

		projected[i] = projVert{
			x:       px,
			y:       py,
			z:       viewZ,
			visible: true,
		}
	}

	// Raster buffers for this frame
	depthBuf := make([]float64, totalCells)
	for i := range depthBuf {
		depthBuf[i] = math.Inf(1)
	}
	intensityBuf := make([]float64, totalCells)
	specularBuf := make([]float64, totalCells)
	matColorBuf := make([]cell.Color, totalCells)

	// Rasterize faces
	for faceIdx, face := range model.Faces {
		if len(face) < 3 {
			continue
		}

		faceColor := baseFallbackColor
		if faceIdx < len(model.FaceColors) && model.FaceColors[faceIdx] != 0 {
			faceColor = model.FaceColors[faceIdx]
		}

		triCount := len(face) - 2
		for t := 0; t < triCount; t++ {
			idx0, idx1, idx2 := face[0], face[t+1], face[t+2]
			p0, p1, p2 := projected[idx0], projected[idx1], projected[idx2]

			if !p0.visible || !p1.visible || !p2.visible {
				continue
			}

			v0, v1, v2 := rotatedVerts[idx0], rotatedVerts[idx1], rotatedVerts[idx2]
			normal := graphics.CalculateNormal(v0, v1, v2)

			// 3D View-space backface culling (normal.Z > 0 for front-facing surfaces)
			if normal.Z <= 0.0 {
				continue
			}

			denom := (p1.y-p2.y)*(p0.x-p2.x) + (p2.x-p1.x)*(p0.y-p2.y)
			if math.Abs(denom) < 1e-6 {
				continue
			}

			// Key light diffuse
			diffKey := normal.Dot(keyLightDir)
			if diffKey < 0 {
				diffKey = 0
			}

			// Fill light diffuse
			diffFill := normal.Dot(fillLightDir)
			if diffFill < 0 {
				diffFill = 0
			}

			// Specular (Blinn-Phong)
			viewDir := graphics.Vector3D{X: 0, Y: 0, Z: 1}
			halfDir := graphics.Vector3D{
				X: keyLightDir.X + viewDir.X,
				Y: keyLightDir.Y + viewDir.Y,
				Z: keyLightDir.Z + viewDir.Z,
			}.Normalize()

			specDot := normal.Dot(halfDir)
			if specDot < 0 {
				specDot = 0
			}
			specPower := math.Pow(specDot, (1.0-roughness)*40.0)

			ambient := 0.35 * envIntensity
			diffuseTotal := (diffKey*0.70 + diffFill*0.30) * envIntensity

			// Triangle bounding box in sub-cell space
			minX := int(math.Max(0, math.Floor(math.Min(p0.x, math.Min(p1.x, p2.x)))))
			maxX := int(math.Min(float64(subW-1), math.Ceil(math.Max(p0.x, math.Max(p1.x, p2.x)))))
			minY := int(math.Max(0, math.Floor(math.Min(p0.y, math.Min(p1.y, p2.y)))))
			maxY := int(math.Min(float64(subH-1), math.Ceil(math.Max(p0.y, math.Max(p1.y, p2.y)))))

			invDenom := 1.0 / denom

			for y := minY; y <= maxY; y++ {
				fy := float64(y) + 0.5
				rowIdx := y * subW
				for x := minX; x <= maxX; x++ {
					fx := float64(x) + 0.5
					l1 := ((p1.y-p2.y)*(fx-p2.x) + (p2.x-p1.x)*(fy-p2.y)) * invDenom
					l2 := ((p2.y-p0.y)*(fx-p2.x) + (p0.x-p2.x)*(fy-p2.y)) * invDenom

					if l1 >= -0.002 && l2 >= -0.002 && (l1+l2) <= 1.002 {
						l3 := 1.0 - l1 - l2
						z := l1*p0.z + l2*p1.z + l3*p2.z
						cellIdx := rowIdx + x

						if z < depthBuf[cellIdx] {
							depthBuf[cellIdx] = z
							intensityBuf[cellIdx] = ambient + diffuseTotal
							specularBuf[cellIdx] = specPower
							matColorBuf[cellIdx] = faceColor
						}
					}
				}
			}
		}
	}

	calcPixelColor := func(cellIdx int) (cell.Color, float64) {
		rawIntensity := intensityBuf[cellIdx]
		specIntensity := specularBuf[cellIdx]
		totIntensity := (rawIntensity + specIntensity*0.80) * exposure
		if totIntensity < 0 {
			totIntensity = 0
		}
		mapped := math.Pow(totIntensity, contrast)
		if mapped > 1.0 {
			mapped = 1.0
		} else if mapped < 0.0 {
			mapped = 0.0
		}
		if a.Invert {
			mapped = 1.0 - mapped
		}

		var fg cell.Color
		if a.Colored {
			base := matColorBuf[cellIdx]
			rB, gB, bB := base.RGB()
			rH, gH, bH := highlightColor.RGB()
			diffFactor := math.Min(1.2, math.Max(0.28, rawIntensity*exposure))
			fR := float64(rB) * diffFactor
			fG := float64(gB) * diffFactor
			fB := float64(bB) * diffFactor
			if specIntensity > 0.05 {
				fR += float64(rH) * specIntensity * 1.3
				fG += float64(gH) * specIntensity * 1.3
				fB += float64(bH) * specIntensity * 1.3
			}
			fR = math.Min(255, math.Max(0, fR))
			fG = math.Min(255, math.Max(0, fG))
			fB = math.Min(255, math.Max(0, fB))
			fg = cell.NewColorRGB(uint8(fR), uint8(fG), uint8(fB))
		} else {
			fg = graphics.ApplyShade(baseFallbackColor, mapped)
		}
		return fg, mapped
	}

	// 1. Half-Block Mode (ModeBlock)
	if effectiveMode == ModeBlock {
		for y := 0; y < h; y++ {
			screenY := area.Y + uint16(y)
			topY := y * 2
			botY := y*2 + 1
			topRow := topY * subW
			botRow := botY * subW

			for x := 0; x < w; x++ {
				screenX := area.X + uint16(x)
				topIdx := topRow + x
				botIdx := botRow + x

				hasTop := !math.IsInf(depthBuf[topIdx], 1)
				hasBot := !math.IsInf(depthBuf[botIdx], 1)

				if !hasTop && !hasBot {
					continue
				}

				var topCol, botCol cell.Color
				if hasTop {
					topCol, _ = calcPixelColor(topIdx)
				}
				if hasBot {
					botCol, _ = calcPixelColor(botIdx)
				}

				if hasTop && hasBot {
					buf.SetCell(screenX, screenY, cell.Cell{
						Content: '▀',
						Style:   cell.Style{Fg: topCol, Bg: botCol},
					})
				} else if hasTop {
					buf.SetCell(screenX, screenY, cell.Cell{
						Content: '▀',
						Style:   cell.Style{Fg: topCol},
					})
				} else {
					buf.SetCell(screenX, screenY, cell.Cell{
						Content: '▄',
						Style:   cell.Style{Fg: botCol},
					})
				}
			}
		}
		return
	}

	// 2. Braille 8x Mode (ModeBraille)
	if effectiveMode == ModeBraille {
		brailleMap := [4][2]rune{
			{0x01, 0x08},
			{0x02, 0x10},
			{0x04, 0x20},
			{0x40, 0x80},
		}

		for y := 0; y < h; y++ {
			screenY := area.Y + uint16(y)
			for x := 0; x < w; x++ {
				screenX := area.X + uint16(x)
				var mask rune = 0
				var avgR, avgG, avgB float64
				count := 0

				for dy := 0; dy < 4; dy++ {
					subY := y*4 + dy
					rowIdx := subY * subW
					for dx := 0; dx < 2; dx++ {
						subX := x*2 + dx
						cellIdx := rowIdx + subX
						if !math.IsInf(depthBuf[cellIdx], 1) {
							mask |= brailleMap[dy][dx]
							col, _ := calcPixelColor(cellIdx)
							r, g, b := col.RGB()
							avgR += float64(r)
							avgG += float64(g)
							avgB += float64(b)
							count++
						}
					}
				}

				if count > 0 {
					avgCol := cell.NewColorRGB(uint8(avgR/float64(count)), uint8(avgG/float64(count)), uint8(avgB/float64(count)))
					buf.SetCell(screenX, screenY, cell.Cell{
						Content: 0x2800 + mask,
						Style:   cell.Style{Fg: avgCol},
					})
				}
			}
		}
		return
	}

	// 3. ModeDithered, ModeASCII & Solid Block
	for y := 0; y < h; y++ {
		rowIdx := y * w
		screenY := area.Y + uint16(y)

		for x := 0; x < w; x++ {
			cellIdx := rowIdx + x
			z := depthBuf[cellIdx]
			if math.IsInf(z, 1) {
				continue
			}

			fg, mapped := calcPixelColor(cellIdx)

			var displayRune rune
			switch effectiveMode {
			case ModeDithered:
				ditherShift := (bayer4x4[y%4][x%4] - 0.5) * 0.32
				dVal := mapped + ditherShift
				if dVal < 0 {
					dVal = 0
				} else if dVal > 1 {
					dVal = 1
				}
				ditherRamp := []rune(" ░▒▓█")
				if a.Ramp != "" && a.Ramp != RampCanvasUI {
					ditherRamp = rampRunes
				}
				dIdx := int(dVal * float64(len(ditherRamp)-1))
				if dIdx < 0 {
					dIdx = 0
				} else if dIdx >= len(ditherRamp) {
					dIdx = len(ditherRamp) - 1
				}
				displayRune = ditherRamp[dIdx]
			default:
				if !a.Ascii && effectiveMode == ModeASCII {
					displayRune = '█'
				} else {
					rampIndex := int(mapped * float64(rampLen-1))
					if rampIndex < 0 {
						rampIndex = 0
					} else if rampIndex >= rampLen {
						rampIndex = rampLen - 1
					}
					displayRune = rampRunes[rampIndex]
				}
			}

			if displayRune == ' ' {
				continue
			}

			screenX := area.X + uint16(x)
			buf.SetCell(screenX, screenY, cell.Cell{
				Content: displayRune,
				Style:   cell.Style{Fg: fg},
			})
		}
	}
}

// SizeHint implements the widgets.Widget interface.
func (a Ascii3D) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}
