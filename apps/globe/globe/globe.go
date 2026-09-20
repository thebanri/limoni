package globe

import (
	"math"
	"time"

	"github.com/thebanri/limoni/apps/globe/globe/geo"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Globe draws the Earth as a lit sphere, seen from above a point on it.
//
// It is an orthographic projection worked backwards: for every pixel inside
// the disc the renderer asks which point of the sphere is there, looks that
// point up in the land mask, and shades it. Doing it that way rather than by
// projecting coastline vectors means zooming costs nothing — the sphere is
// sampled at whatever resolution the pane happens to have — and that the draw
// path allocates nothing at all.
//
// A terminal cell is about twice as tall as it is wide, so the renderer works
// in half-cells: each cell holds two square pixels, drawn as an upper half
// block whose foreground is the top pixel and whose background is the bottom
// one. That is what makes the sphere round rather than an ellipse.
type Globe struct {
	ID    string
	Label string

	// Lat and Lon are the point at the centre of the disc, in degrees.
	Lat, Lon float64
	// Zoom is 1 when the whole globe fits the pane, and grows from there.
	Zoom float64

	// Markers are the pinned places. A marker on the far side of the globe is
	// hidden by it, as it should be.
	Markers []Marker
	// Now is the frame's time, which drives the ping animation. The zero time
	// disables it, which is what tests want.
	Now time.Time

	// ASCII draws with shading characters and no colour, for terminals — or
	// screenshots — where that reads better.
	ASCII bool
	// Graticule draws the parallels and meridians every 30°.
	Graticule bool
	// Borders draws where one country meets another. Coasts are not drawn:
	// the water's edge is already a change of colour.
	Borders bool
}

// Marker is a pinned place.
type Marker struct {
	Name     string
	Lat, Lon float64
	// Pinged is when the marker was last pinged; a ring expands from it for
	// pingFor after that. The zero time means no ping.
	Pinged time.Time
	// Label draws the name beside the dot when there is room.
	Label bool
}

const (
	// pingFor is how long the ring takes to expand and go. It is short on
	// purpose: a ping says "here", and something that says "here" for a
	// second and a half is still saying it long after you have looked.
	pingFor    = 650 * time.Millisecond
	pingRadius = 12.0 // half-cells the ring grows to
)

// The light sits above, to the left and in front of the viewer. It is fixed
// in view space rather than in the world, so the sphere stays lit the same
// way as it turns and the shading reads as shape rather than as time of day.
var lightX, lightY, lightZ = normalize(-0.45, 0.55, 0.78)

// asciiRamp is the shading ramp for ASCII mode, darkest first. '+' is
// deliberately not in it: in ASCII mode that character means a border and
// nothing else, so a line between two countries cannot be mistaken for a
// patch of half-lit ground.
const asciiRamp = " .:-*#%@"

// vec3 is a point or direction in the world frame: X towards (0°N, 90°E),
// Y towards the north pole, Z towards (0°N, 0°E).
type vec3 struct{ X, Y, Z float64 }

func normalize(x, y, z float64) (float64, float64, float64) {
	n := math.Sqrt(x*x + y*y + z*z)
	if n == 0 {
		return 0, 0, 0
	}
	return x / n, y / n, z / n
}

// sphere returns the unit-sphere point at a latitude and longitude.
func sphere(lat, lon float64) vec3 {
	la, lo := lat*math.Pi/180, lon*math.Pi/180
	cl := math.Cos(la)
	return vec3{cl * math.Sin(lo), math.Sin(la), cl * math.Cos(lo)}
}

// basis returns the camera's axes for a view centred on a point: right across
// the disc, up it, and forward out of it towards the viewer.
func basis(lat, lon float64) (right, up, fwd vec3) {
	fwd = sphere(lat, lon)
	right = sphere(0, lon+90)
	up = sphere(lat+90, lon)
	return right, up, fwd
}

// project maps a place to the disc. The returned x and y are in half-cells
// from the centre, with y growing downwards as the screen does, and front is
// false when the point is round the back of the globe.
func project(lat, lon float64, right, up, fwd vec3, radius float64) (x, y float64, front bool) {
	p := sphere(lat, lon)
	nx := p.X*right.X + p.Y*right.Y + p.Z*right.Z
	ny := p.X*up.X + p.Y*up.Y + p.Z*up.Z
	nz := p.X*fwd.X + p.Y*fwd.Y + p.Z*fwd.Z
	return nx * radius, -ny * radius, nz > 0
}

// unproject is project run backwards: the point of the sphere under a pixel.
// It reports false outside the disc, where there is no sphere to hit.
func unproject(x, y float64, right, up, fwd vec3, radius float64) (lat, lon float64, hit bool) {
	nx, ny := x/radius, -y/radius
	r2 := nx*nx + ny*ny
	if r2 > 1 {
		return 0, 0, false
	}
	nz := math.Sqrt(1 - r2)
	wx := nx*right.X + ny*up.X + nz*fwd.X
	wy := nx*right.Y + ny*up.Y + nz*fwd.Y
	wz := nx*right.Z + ny*up.Z + nz*fwd.Z
	return math.Asin(clamp(wy, -1, 1)) * 180 / math.Pi, math.Atan2(wx, wz) * 180 / math.Pi, true
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// radius returns the globe's radius in half-cells for an area.
func (g *Globe) radius(area cell.Rect) float64 {
	zoom := g.Zoom
	if zoom <= 0 {
		zoom = 1
	}
	w, h := float64(area.Width), float64(area.Height)*2
	r := math.Min(w, h) / 2
	return math.Max(2, r*0.96*zoom)
}

// Centre returns the place the globe is currently aimed at.
func (g *Globe) Centre() (lat, lon float64) { return g.Lat, wrapLon(g.Lon) }

// At returns the point of the sphere under a cell, for a click. row and col
// are relative to the pane, and half says which half of the cell was hit.
func (g *Globe) At(area cell.Rect, col, row uint16, lower bool) (lat, lon float64, hit bool) {
	right, up, fwd := basis(g.Lat, g.Lon)
	radius := g.radius(area)
	cx, cy := float64(area.Width)/2, float64(area.Height)
	y := float64(row)*2 + 0.5
	if lower {
		y++
	}
	return unproject(float64(col)+0.5-cx, y-cy, right, up, fwd, radius)
}

func (g *Globe) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	right, up, fwd := basis(g.Lat, g.Lon)
	radius := g.radius(area)
	cx, cy := float64(area.Width)/2, float64(area.Height)
	// Degrees covered by one half-cell at the centre of the disc, which is
	// what decides how wide a graticule line has to be drawn to stay visible.
	degPerPixel := 57.29578 / radius

	for row := uint16(0); row < area.Height; row++ {
		for col := uint16(0); col < area.Width; col++ {
			px := float64(col) + 0.5 - cx
			topY := float64(row)*2 + 0.5 - cy

			if g.ASCII {
				buf.SetCellDirect(area.X+col, area.Y+row, g.asciiCell(px, topY+0.5, right, up, fwd, radius, degPerPixel))
				continue
			}

			topC, topHit := g.shade(px, topY, right, up, fwd, radius, degPerPixel)
			botC, botHit := g.shade(px, topY+1, right, up, fwd, radius, degPerPixel)
			if !topHit && !botHit {
				buf.SetCellDirect(area.X+col, area.Y+row, cell.Cell{Content: ' ', Style: cell.Style{Bg: spaceColor}})
				continue
			}
			buf.SetCellDirect(area.X+col, area.Y+row, cell.Cell{
				Content: '▀',
				Style:   cell.Style{Fg: topC, Bg: botC},
			})
		}
	}

	g.drawMarkers(area, buf, right, up, fwd, radius)
}

var spaceColor = cell.NewColorRGB(8, 10, 18)

// shade returns the colour of one pixel of the disc, and false for space.
func (g *Globe) shade(x, y float64, right, up, fwd vec3, radius, degPerPixel float64) (cell.Color, bool) {
	nx, ny := x/radius, -y/radius
	r2 := nx*nx + ny*ny
	if r2 > 1 {
		return spaceColor, false
	}
	nz := math.Sqrt(1 - r2)

	// The view-space normal is the pixel's own direction, so the diffuse term
	// is a dot product with the fixed light and nothing else.
	lum := nx*lightX + ny*lightY + nz*lightZ
	if lum < 0 {
		lum = 0
	}
	lum = 0.18 + 0.82*lum

	wx := nx*right.X + ny*up.X + nz*fwd.X
	wy := nx*right.Y + ny*up.Y + nz*fwd.Y
	wz := nx*right.Z + ny*up.Z + nz*fwd.Z
	lat := math.Asin(clamp(wy, -1, 1)) * 180 / math.Pi
	lon := math.Atan2(wx, wz) * 180 / math.Pi

	var r, gr, b float64
	if geo.IsLand(lat, lon) {
		r, gr, b = landColor(lat)
		// A border is drawn darker than the land it crosses, which reads on
		// desert, grass and ice alike. Over sea it would be a line in open
		// water, so it stops at the coast.
		if g.Borders && geo.IsBorder(lat, lon, degPerPixel) {
			r, gr, b = r*0.42+30*0.58, gr*0.42+24*0.58, b*0.42+20*0.58
		}
	} else {
		r, gr, b = 14, 48, 104
	}

	if g.Graticule && onGraticule(lat, lon, degPerPixel) {
		r, gr, b = r*0.55+120*0.45, gr*0.55+150*0.45, b*0.55+170*0.45
	}

	// The limb is dimmed a little beyond the diffuse term, which is what
	// makes the edge read as curvature rather than as a cut-out circle.
	edge := 1 - 0.35*r2*r2
	lum *= edge

	return cell.NewColorRGB(channel(r*lum), channel(gr*lum), channel(b*lum)), true
}

// landColor is a plausible Earth by latitude: ice at the caps, desert in the
// two dry belts, green in between. The mask says where land is, not what
// grows on it, and guessing from latitude is close enough to read as Earth.
func landColor(lat float64) (r, g, b float64) {
	a := math.Abs(lat)
	switch {
	case a > 68:
		return 228, 236, 245
	case a > 60:
		t := (a - 60) / 8
		return 110 + 118*t, 140 + 96*t, 110 + 135*t
	case a > 33:
		return 88, 128, 74
	case a > 26:
		t := (a - 26) / 7
		return 186 - 98*t, 166 - 38*t, 108 - 34*t
	case a > 15:
		return 186, 166, 108
	case a > 10:
		t := (a - 10) / 5
		return 96 + 90*t, 140 + 26*t, 70 + 38*t
	default:
		return 82, 136, 66
	}
}

func channel(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}

// onGraticule reports whether a point is close enough to a 30° parallel or
// meridian to draw one. Meridians converge at the poles, so the tolerance
// grows with the secant of the latitude or they vanish near the caps.
func onGraticule(lat, lon, degPerPixel float64) bool {
	tol := degPerPixel * 0.6
	if math.Abs(math.Mod(math.Abs(lat)+15, 30)-15) < tol {
		return true
	}
	c := math.Cos(lat * math.Pi / 180)
	if c < 0.02 {
		return false
	}
	return math.Abs(math.Mod(math.Abs(lon)+15, 30)-15) < tol/c
}

// asciiCell renders one cell in ASCII mode, sampling the sphere at its centre.
func (g *Globe) asciiCell(x, y float64, right, up, fwd vec3, radius, degPerPixel float64) cell.Cell {
	nx, ny := x/radius, -y/radius
	r2 := nx*nx + ny*ny
	if r2 > 1 {
		return cell.Cell{Content: ' '}
	}
	nz := math.Sqrt(1 - r2)
	lum := nx*lightX + ny*lightY + nz*lightZ
	if lum < 0 {
		lum = 0
	}

	wx := nx*right.X + ny*up.X + nz*fwd.X
	wy := nx*right.Y + ny*up.Y + nz*fwd.Y
	wz := nx*right.Z + ny*up.Z + nz*fwd.Z
	lat := math.Asin(clamp(wy, -1, 1)) * 180 / math.Pi
	lon := math.Atan2(wx, wz) * 180 / math.Pi

	// Land takes the top half of the ramp and sea the bottom, so the
	// continents stay legible even where the light is flat.
	var idx int
	if geo.IsLand(lat, lon) {
		if g.Borders && geo.IsBorder(lat, lon, degPerPixel) {
			// There is no colour to darken in ASCII, so a border is drawn
			// as itself.
			return cell.Cell{Content: '+'}
		}
		idx = 3 + int(lum*4.99)
	} else {
		idx = int(lum * 2.99)
	}
	if idx >= len(asciiRamp) {
		idx = len(asciiRamp) - 1
	}
	return cell.Cell{Content: rune(asciiRamp[idx])}
}

var (
	markerColor = cell.NewColorRGB(255, 96, 96)
	labelColor  = cell.NewColorRGB(255, 232, 160)
	pingColor   = cell.NewColorRGB(255, 180, 90)
)

func (g *Globe) drawMarkers(area cell.Rect, buf *buffer.Buffer, right, up, fwd vec3, radius float64) {
	cx, cy := float64(area.Width)/2, float64(area.Height)

	for i := range g.Markers {
		m := &g.Markers[i]
		x, y, front := project(m.Lat, m.Lon, right, up, fwd, radius)
		if !front {
			continue
		}
		if !g.Now.IsZero() && !m.Pinged.IsZero() {
			if age := g.Now.Sub(m.Pinged); age >= 0 && age < pingFor {
				g.drawPing(area, buf, x+cx, y+cy, float64(age)/float64(pingFor))
			}
		}
		g.setPixel(area, buf, x+cx, y+cy, markerColor)

		if m.Label && m.Name != "" {
			col := int(x+cx) + 2
			row := int((y + cy) / 2)
			if col >= 0 && row >= 0 && row < int(area.Height) && col < int(area.Width) {
				buf.SetStringWithin(area.X+uint16(col), area.Y+uint16(row), m.Name,
					cell.Style{Fg: labelColor}, area.Width-uint16(col))
			}
		}
	}
}

// drawPing draws the expanding ring that says "here". progress runs 0 to 1.
func (g *Globe) drawPing(area cell.Rect, buf *buffer.Buffer, x, y, progress float64) {
	r := pingRadius * progress
	if r < 0.5 {
		return
	}
	// Squared, so the ring is faint for most of its short life rather than
	// fading evenly to the end: a linear fade left a visible ring at 90% of
	// the way through, which is what made the old ping outstay its welcome.
	fade := (1 - progress) * (1 - progress)
	c := cell.NewColorRGB(
		channel(float64(255)*fade),
		channel(float64(180)*fade),
		channel(float64(90)*fade),
	)
	steps := int(6 * r)
	if steps < 12 {
		steps = 12
	}
	for i := 0; i < steps; i++ {
		a := 2 * math.Pi * float64(i) / float64(steps)
		g.setPixel(area, buf, x+r*math.Cos(a), y+r*math.Sin(a), c)
	}
}

// setPixel colours one half-cell, keeping whatever is in the other half.
func (g *Globe) setPixel(area cell.Rect, buf *buffer.Buffer, x, y float64, col cell.Color) {
	if x < 0 || y < 0 {
		return
	}
	cx, cy := int(x), int(y)
	row := cy / 2
	if cx >= int(area.Width) || row >= int(area.Height) {
		return
	}
	c := buf.Get(area.X+uint16(cx), area.Y+uint16(row))
	if c == nil {
		return
	}
	if g.ASCII {
		c.Content = '◆'
		c.Style.Fg = col
		return
	}
	if c.Content != '▀' {
		// A label or a space cell: make it a half block first so that both
		// halves mean what the renderer thinks they mean.
		c.Content = '▀'
		c.Style.Fg = spaceColor
		c.Style.Bg = spaceColor
	}
	if cy%2 == 0 {
		c.Style.Fg = col
	} else {
		c.Style.Bg = col
	}
}

func (g *Globe) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, maxArea.Height }

// AccessibilityNode describes what the globe is showing, so that a test or an
// agent can read the view rather than the pixels: where it is pointed, how
// far it is zoomed, and which markers are on the visible side.
func (g *Globe) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	node := accessibility.AccessibilityNode{
		ID:     g.ID,
		Role:   accessibility.RoleImage,
		Label:  g.Label,
		Value:  g.viewValue(),
		Bounds: bounds,
	}
	if focused {
		node.State |= accessibility.StateFocused
	}

	right, up, fwd := basis(g.Lat, g.Lon)
	radius := g.radius(bounds)
	cx, cy := float64(bounds.Width)/2, float64(bounds.Height)
	for i := range g.Markers {
		m := &g.Markers[i]
		x, y, front := project(m.Lat, m.Lon, right, up, fwd, radius)
		if !front {
			continue
		}
		// The cell the marker sits in, so that whatever reads this tree knows
		// where on screen the place is, not merely that it is visible.
		col, row := int(x+cx), int((y+cy)/2)
		cellBounds := cell.Rect{}
		if col >= 0 && row >= 0 && col < int(bounds.Width) && row < int(bounds.Height) {
			cellBounds = cell.NewRect(bounds.X+uint16(col), bounds.Y+uint16(row), 1, 1)
		}
		node.Children = append(node.Children, accessibility.AccessibilityNode{
			Role:   accessibility.RoleListItem,
			Label:  m.Name,
			Value:  formatCoord(m.Lat, m.Lon),
			Bounds: cellBounds,
		})
	}
	node.SetSize = len(node.Children)
	for i := range node.Children {
		node.Children[i].Position = i + 1
	}
	return node
}

func (g *Globe) viewValue() string {
	return formatCoord(g.Lat, wrapLon(g.Lon)) + " · zoom " + formatZoom(g.Zoom)
}
