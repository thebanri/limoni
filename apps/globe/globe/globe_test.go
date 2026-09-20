package globe

import (
	"math"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/apps/globe/globe/geo"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// The projection and its inverse have to agree, or every pixel of the globe
// is drawn from the wrong part of the world — which looks plausible enough
// on screen that only a test catches it.
func TestProjectionRoundTrips(t *testing.T) {
	views := [][2]float64{{0, 0}, {39, 35}, {-34, 151}, {70, -40}, {0, 179}, {-80, 10}}
	points := [][2]float64{{0, 0}, {39.9, 32.9}, {51.5, -0.1}, {-23, -46}, {35, 139}, {89, 0}, {-89, 0}}

	for _, v := range views {
		right, up, fwd := basis(v[0], v[1])
		const radius = 100
		for _, p := range points {
			x, y, front := project(p[0], p[1], right, up, fwd, radius)
			if !front {
				continue // the far side has no pixel to check
			}
			lat, lon, hit := unproject(x, y, right, up, fwd, radius)
			if !hit {
				t.Errorf("view %v: %v projects to (%.1f, %.1f), which is off the disc", v, p, x, y)
				continue
			}
			if math.Abs(lat-p[0]) > 0.01 || math.Abs(angleDiff(lon, p[1])) > 0.01 {
				t.Errorf("view %v: %v round-tripped to (%.3f, %.3f)", v, p, lat, lon)
			}
		}
	}
}

func angleDiff(a, b float64) float64 { return wrapLon(a - b) }

// The centre of the disc is the point the globe is aimed at. If this drifts,
// "fly to Turkey" puts Turkey somewhere off to one side.
func TestCentreOfTheDiscIsTheViewPoint(t *testing.T) {
	for _, v := range [][2]float64{{0, 0}, {39.35, 34.51}, {-33.9, 18.4}, {64, -21}} {
		right, up, fwd := basis(v[0], v[1])
		lat, lon, hit := unproject(0, 0, right, up, fwd, 50)
		if !hit {
			t.Fatalf("the centre of the disc missed the globe")
		}
		if math.Abs(lat-v[0]) > 0.001 || math.Abs(angleDiff(lon, v[1])) > 0.001 {
			t.Errorf("aimed at %v, centre reads (%.3f, %.3f)", v, lat, lon)
		}
	}
}

// Half the world is behind the globe and must not be drawn on it.
func TestTheFarSideIsHidden(t *testing.T) {
	right, up, fwd := basis(0, 0) // looking at the Gulf of Guinea
	cases := []struct {
		name     string
		lat, lon float64
		front    bool
	}{
		{"the point itself", 0, 0, true},
		{"90° east, on the limb", 0, 89.9, true},
		{"the far side", 0, 180, false},
		{"just past the limb", 0, 91, false},
		{"the north pole", 89, 0, true},
	}
	for _, c := range cases {
		if _, _, front := project(c.lat, c.lon, right, up, fwd, 100); front != c.front {
			t.Errorf("%s: front = %v, want %v", c.name, front, c.front)
		}
	}
}

// Pixels outside the disc are space, and the disc is round: the same radius
// in every direction, in half-cells.
func TestTheDiscIsRound(t *testing.T) {
	right, up, fwd := basis(20, 0)
	const radius = 40
	for deg := 0; deg < 360; deg += 15 {
		a := float64(deg) * math.Pi / 180
		if _, _, hit := unproject(radius*0.99*math.Cos(a), radius*0.99*math.Sin(a), right, up, fwd, radius); !hit {
			t.Errorf("%d°: just inside the edge missed the globe", deg)
		}
		if _, _, hit := unproject(radius*1.01*math.Cos(a), radius*1.01*math.Sin(a), right, up, fwd, radius); hit {
			t.Errorf("%d°: just outside the edge hit the globe", deg)
		}
	}
}

// renderLand draws the globe and reports, for each cell, whether the point
// under it is land — read back through the same projection the renderer uses,
// so the map and the drawing cannot disagree.
func renderLand(g *Globe, w, h uint16) []string {
	right, up, fwd := basis(g.Lat, g.Lon)
	area := cell.NewRect(0, 0, w, h)
	radius := g.radius(area)
	cx, cy := float64(w)/2, float64(h)
	rows := make([]string, 0, h)
	for row := uint16(0); row < h; row++ {
		var b strings.Builder
		for col := uint16(0); col < w; col++ {
			lat, lon, hit := unproject(float64(col)+0.5-cx, float64(row)*2+1-cy, right, up, fwd, radius)
			switch {
			case !hit:
				b.WriteByte(' ')
			case geo.IsLand(lat, lon):
				b.WriteByte('#')
			default:
				b.WriteByte('.')
			}
		}
		rows = append(rows, b.String())
	}
	return rows
}

// The world drawn on the sphere has to be the world. Rather than guess at a
// number, this compares the rendered disc against the same question asked of
// the sphere directly: in an orthographic projection a patch of surface at
// angle θ from the centre covers cos θ of the disc area it would cover head
// on, so the fraction of land pixels must match the cos-weighted fraction of
// land over the visible hemisphere. Two independent routes to one number.
func TestTheDiscMatchesTheHemisphereItShows(t *testing.T) {
	for _, v := range [][2]float64{{5, 20}, {0, -160}, {45, 70}, {-85, 0}, {39, 35}} {
		g := &Globe{Lat: v[0], Lon: v[1], Zoom: 1}
		drawn := landFraction(renderLand(g, 120, 60))
		direct := hemisphereLandFraction(v[0], v[1])
		if math.Abs(drawn-direct) > 0.04 {
			t.Errorf("looking at (%.0f, %.0f): the disc is %.1f%% land, the hemisphere %.1f%%",
				v[0], v[1], drawn*100, direct*100)
		}
	}
}

func landFraction(rows []string) float64 {
	land, total := 0, 0
	for _, r := range rows {
		land += strings.Count(r, "#")
		total += strings.Count(r, "#") + strings.Count(r, ".")
	}
	if total == 0 {
		return 0
	}
	return float64(land) / float64(total)
}

// hemisphereLandFraction integrates the land mask over the hemisphere facing
// a point, weighting each patch by cos of its angle from the centre — the
// orthographic foreshortening — and by cos of its latitude, which is the
// area of the patch itself.
func hemisphereLandFraction(lat0, lon0 float64) float64 {
	var land, total float64
	for i := 0; i < 900; i++ {
		lat := -89.9 + 179.8*float64(i)/899
		w := math.Cos(lat * math.Pi / 180)
		for j := 0; j < 1800; j++ {
			lon := -180 + 360*float64(j)/1800
			cosTheta := math.Cos(angularDistance(lat0, lon0, lat, lon) * math.Pi / 180)
			if cosTheta <= 0 {
				continue
			}
			weight := w * cosTheta
			total += weight
			if geo.IsLand(lat, lon) {
				land += weight
			}
		}
	}
	return land / total
}

// The centre pixel is the plainest check there is: aim at a place, and the
// middle of the disc is that place.
func TestTheCentrePixelIsTheRightPlace(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
		land     bool
	}{
		{"central Anatolia", 39.0, 34.0, true},
		{"the middle of the Pacific", 0, -150, false},
		{"the Sahara", 23, 13, true},
		{"the South Atlantic", -35, -20, false},
		{"Antarctica", -82, 0, true},
	}
	for _, c := range cases {
		rows := renderLand(&Globe{Lat: c.lat, Lon: c.lon, Zoom: 1}, 81, 41)
		got := rows[len(rows)/2][len(rows[0])/2]
		want := byte('.')
		if c.land {
			want = '#'
		}
		if got != want {
			t.Errorf("aimed at %s: the centre pixel is %q, want %q", c.name, got, want)
		}
	}
}

// Turning the globe by a whole turn brings the same world back.
func TestAFullTurnComesBackToTheSameView(t *testing.T) {
	a := renderLand(&Globe{Lat: 10, Lon: 30, Zoom: 1}, 60, 30)
	b := renderLand(&Globe{Lat: 10, Lon: 30 + 360, Zoom: 1}, 60, 30)
	if strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Error("a full turn does not come back to the same view")
	}
}

// Zooming in shows less of the world, not a bigger picture of all of it.
func TestZoomingInNarrowsTheView(t *testing.T) {
	wide := renderLand(&Globe{Lat: 39, Lon: 35, Zoom: 1}, 80, 40)
	close := renderLand(&Globe{Lat: 39, Lon: 35, Zoom: 6}, 80, 40)
	space := func(rows []string) int {
		n := 0
		for _, r := range rows {
			n += strings.Count(r, " ")
		}
		return n
	}
	if space(close) >= space(wide) {
		t.Error("zooming in did not fill more of the pane with globe")
	}

	// At zoom 6 the visible strip is a few tens of degrees across, so the
	// corners of the view are far closer to the centre than at zoom 1.
	right, up, fwd := basis(39, 35)
	radius := (&Globe{Lat: 39, Lon: 35, Zoom: 6}).radius(cell.NewRect(0, 0, 80, 40))
	lat, lon, hit := unproject(39, 0, right, up, fwd, radius) // 39 half-cells right of centre
	if !hit {
		t.Fatal("the edge of the pane is off the globe at zoom 6")
	}
	if d := angularDistance(39, 35, lat, lon); d > 40 {
		t.Errorf("at zoom 6 the edge of the pane is %.0f° away, which is not zoomed in", d)
	}
}

// The draw path runs for every pixel of every frame, so it may not allocate.
func TestDrawDoesNotAllocate(t *testing.T) {
	g := &Globe{ID: "globe", Lat: 20, Lon: 30, Zoom: 1.4, Graticule: true}
	g.Markers = []Marker{{Name: "Ankara", Lat: 39.9, Lon: 32.9, Label: true}}
	area := cell.NewRect(0, 0, 80, 40)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	g.Draw(ctx, buf) // warm the land mask

	if n := testing.AllocsPerRun(20, func() {
		g.Draw(ctx, buf)
		runtime.GC()
	}); n != 0 {
		t.Errorf("Draw allocated %v times per frame", n)
	}

	g.ASCII = true
	if n := testing.AllocsPerRun(20, func() {
		g.Draw(ctx, buf)
		runtime.GC()
	}); n != 0 {
		t.Errorf("ASCII Draw allocated %v times per frame", n)
	}
}

// Nothing about the globe may change between two frames of the same view, or
// the diff resends the screen and an idle application spends bandwidth.
func TestTheSameViewDrawsTheSameFrame(t *testing.T) {
	g := &Globe{Lat: 12, Lon: -40, Zoom: 2, Graticule: true}
	area := cell.NewRect(0, 0, 60, 24)
	ctx := cell.NewContext(area, cell.Style{})
	a, b := buffer.NewBuffer(area), buffer.NewBuffer(area)
	g.Draw(ctx, a)
	g.Draw(ctx, b)
	for i := range a.Content {
		if a.Content[i] != b.Content[i] {
			t.Fatalf("cell %d differs between two frames of the same view", i)
		}
	}
}

// Pinning a place pings it: a ring expands from the marker for a moment and
// fades. This is the part a person sees and no assertion about coordinates
// would catch, so it is measured directly — the ring is drawn further from
// the marker as the ping ages, and it is gone once the ping is over.
func TestAPingExpandsAndThenStops(t *testing.T) {
	start := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	g := &Globe{
		Lat: 39.3, Lon: 34.5, Zoom: 2.6,
		Markers: []Marker{{Name: "Turkey", Lat: 39.3, Lon: 34.5, Pinged: start}},
	}
	area := cell.NewRect(0, 0, 80, 40)
	ctx := cell.NewContext(area, cell.Style{})

	radii := make([]float64, 0, 3)
	for _, age := range []time.Duration{200 * time.Millisecond, 700 * time.Millisecond, 1400 * time.Millisecond} {
		g.Now = start.Add(age)
		buf := buffer.NewBuffer(area)
		g.Draw(ctx, buf)
		r := ringRadius(buf, area)
		if r == 0 {
			t.Fatalf("no ring %v into the ping", age)
		}
		radii = append(radii, r)
	}
	if !(radii[0] < radii[1] && radii[1] < radii[2]) {
		t.Errorf("the ring did not expand: %.1f, %.1f, %.1f", radii[0], radii[1], radii[2])
	}

	// Once the ping is over the ring is gone, or a pin would glow for ever.
	g.Now = start.Add(pingFor + time.Millisecond)
	buf := buffer.NewBuffer(area)
	g.Draw(ctx, buf)
	if r := ringRadius(buf, area); r != 0 {
		t.Errorf("the ring is still being drawn %v after the ping, at radius %.1f", pingFor, r)
	}
}

// ringRadius returns how far the furthest ping-coloured pixel sits from the
// centre of the disc, in half-cells, or 0 if none is drawn.
func ringRadius(buf *buffer.Buffer, area cell.Rect) float64 {
	cx, cy := float64(area.Width)/2, float64(area.Height)
	furthest := 0.0
	for row := uint16(0); row < area.Height; row++ {
		for col := uint16(0); col < area.Width; col++ {
			c := buf.Get(col, row)
			if c == nil {
				continue
			}
			for half, colour := range [2]cell.Color{c.Style.Fg, c.Style.Bg} {
				if !isPingColour(colour) {
					continue
				}
				x := float64(col) + 0.5 - cx
				y := float64(row)*2 + float64(half) + 0.5 - cy
				if d := math.Hypot(x, y); d > furthest {
					furthest = d
				}
			}
		}
	}
	return furthest
}

// isPingColour recognises the ring's orange, which fades but keeps its hue:
// red highest, then green, then blue, and nothing else on the globe is drawn
// in that proportion.
func isPingColour(c cell.Color) bool {
	if c.Type() != cell.ColorRGB {
		return false
	}
	r, g, b := c.RGB()
	// The ring fades as it grows, so the test looks at the hue rather than
	// the brightness: by the end of the ping it is down to a tenth of its
	// first colour but still the same orange.
	return r > 12 && float64(g) > float64(r)*0.6 && float64(g) < float64(r)*0.78 &&
		float64(b) > float64(r)*0.28 && float64(b) < float64(r)*0.42
}
