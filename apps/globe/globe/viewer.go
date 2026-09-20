// Package globeapp is a rotating world you can search, zoom into and pin —
// and, because everything it shows is in the semantic tree, one an agent can
// drive as well as a person: "find Turkey" is a search and a click.
package globe

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/apps/globe/globe/geo"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

// The widget identifiers the keyboard and the semantic tree both use.
const (
	globeID  = "globe"
	searchID = "search"
)

const (
	sidebarWidth = 34
	// spinRate is how fast the world turns when nothing is holding it, in
	// degrees a second. A full turn takes a little under a minute, which is
	// slow enough to read a coastline and fast enough to feel alive.
	spinRate = 6.0
	// defaultFlyFor is how long it takes to swing round to a selected place.
	defaultFlyFor = 900 * time.Millisecond

	minZoom = 0.6
	maxZoom = 40.0
)

// Viewer is the whole application: the globe, the search beside it, and the
// pins that have been dropped on it.
type Viewer struct {
	globe *Globe
	ix    *index

	query   *widgets.TextInputState
	results *widgets.ListState
	pins    *widgets.ListState
	hits    []match
	rows    []string // the result rows, rebuilt only when the query changes
	pinRows []string

	searching bool
	spinning  bool
	help      bool
	status    string

	fly   *flight
	last  time.Time
	wake  func()
	clock func() time.Time
	// flyFor is how long a flight takes. A test sets it to zero to arrive at
	// once rather than waiting for an animation it is not testing.
	flyFor time.Duration

	// The click handler and the area it reads are kept on the viewer so that
	// registering it every frame costs nothing: a closure built during Draw
	// is a heap allocation per frame.
	globeArea  limoni.Rect
	clickGlobe func(driver.MouseEvent)
	footer     string

	// The lists' areas and their click handlers, built once. Registering a
	// closure made during Draw would allocate on every frame.
	resultsArea limoni.Rect
	pinsArea    limoni.Rect
	clickResult func(driver.MouseEvent)
	clickPin    func(driver.MouseEvent)

	// wantFocus is where the keyboard should go on the next frame. A click
	// handler runs outside the draw, where there is no frame to ask, and
	// choosing a place has to hand the keyboard back to the globe — or the
	// next key zooms nothing and types into the search box instead.
	wantFocus string
}

// flight is a swing from one view of the globe to another.
type flight struct {
	fromLat, fromLon, fromZoom float64
	toLat, toLon, toZoom       float64
	started                    time.Time
	name                       string
}

// New returns a viewer showing the whole globe, turning.
func New() *Viewer {
	v := &Viewer{
		globe: &Globe{
			ID:        globeID,
			Label:     "Globe",
			Lat:       20,
			Lon:       20,
			Zoom:      1,
			Graticule: true,
		},
		ix:       newIndex(geo.All()),
		query:    widgets.NewTextInputState(),
		results:  widgets.NewListState(),
		pins:     widgets.NewListState(),
		spinning: true,
		clock:    time.Now,
		flyFor:   defaultFlyFor,
		last:     time.Now(),
	}
	v.clickGlobe = v.clickAt
	v.clickResult = v.chooseResult
	v.clickPin = v.choosePin
	v.refresh()
	return v
}

// SetWake installs the redraw request the animation needs: the globe turns
// between key presses, so something has to ask for the next frame.
func (v *Viewer) SetWake(wake func()) { v.wake = wake }

// Globe exposes the globe for a caller that wants to set the opening view.
func (v *Viewer) Globe() *Globe { return v.globe }

// Look aims the globe at a place immediately, without the animation.
func (v *Viewer) Look(lat, lon, zoom float64) {
	v.globe.Lat, v.globe.Lon, v.globe.Zoom = clampLat(lat), wrapLon(lon), clampZoom(zoom)
	v.fly = nil
}

// Pin drops a marker and pings it. Pinning the same place twice pings it
// again rather than stacking two markers on one spot.
func (v *Viewer) Pin(name string, lat, lon float64, now time.Time) {
	for i := range v.globe.Markers {
		m := &v.globe.Markers[i]
		if m.Name == name && near(m.Lat, lat) && near(m.Lon, lon) {
			m.Pinged = now
			return
		}
	}
	v.globe.Markers = append(v.globe.Markers, Marker{
		Name: name, Lat: lat, Lon: lon, Pinged: now, Label: true,
	})
	v.pinRows = append(v.pinRows, name+"  "+formatCoord(lat, lon))
	v.pins.Selected = len(v.pinRows) - 1
}

func near(a, b float64) bool { return a-b < 0.05 && b-a < 0.05 }

// Frame draws one frame and handles one event. It is the function to hand to
// limoni.Run.
func (v *Viewer) Frame(f *limoni.Frame, ev *limoni.Event) bool {
	now := v.clock()

	// The search box has the keyboard when the focus manager says so, rather
	// than when the viewer decides it should. That is what makes clicking the
	// input — with a pointer, or through the semantic tree — enough to type
	// into it, which is how anything driving the application from outside
	// will expect it to work.
	v.searching = f.IsFocused(searchID)

	if ev != nil && ev.Type == limoni.EventKey {
		if !v.key(f, ev.Key, now) {
			return false
		}
	}
	if v.wantFocus != "" {
		v.focus(f, v.wantFocus)
		v.wantFocus = ""
	}
	v.advance(now)
	v.globe.Now = now
	v.Draw(f, f.Area())
	return true
}

// chooseResult flies to the row a click landed on. The row is worked out
// here rather than read back from the list, so that clicking the row that is
// already highlighted still counts as choosing it.
func (v *Viewer) chooseResult(ev driver.MouseEvent) {
	row := rowAt(v.resultsArea, ev, v.results.Offset)
	if row < 0 || row >= len(v.hits) {
		return
	}
	v.results.Selected = row
	v.FlyTo(v.hits[row].Place, v.clock())
	v.wantFocus = globeID
	v.Wake()
}

// choosePin flies back to a pin that was clicked, and pings it again.
func (v *Viewer) choosePin(ev driver.MouseEvent) {
	row := rowAt(v.pinsArea, ev, v.pins.Offset)
	if row < 0 || row >= len(v.globe.Markers) {
		return
	}
	now := v.clock()
	m := v.globe.Markers[row]
	v.pins.Selected = row
	v.fly = &flight{
		fromLat: v.globe.Lat, fromLon: v.globe.Lon, fromZoom: v.globe.Zoom,
		toLat: clampLat(m.Lat), toLon: wrapLon(m.Lon), toZoom: clampZoom(math.Max(v.globe.Zoom, 3)),
		started: now, name: m.Name,
	}
	v.globe.Markers[row].Pinged = now.Add(v.flyFor / 2)
	v.spinning = false
	v.status = "→ " + m.Name
	v.wantFocus = globeID
	v.Wake()
}

// rowAt returns the index a click in a list landed on, or -1. offset is the
// list's scroll position, so a click on the third visible row of a list
// scrolled to the fiftieth item chooses the fifty-second place.
func rowAt(area limoni.Rect, ev driver.MouseEvent, offset int) int {
	if area.Height == 0 || ev.Y < area.Y || ev.Y >= area.Y+area.Height ||
		ev.X < area.X || ev.X >= area.X+area.Width {
		return -1
	}
	return int(ev.Y-area.Y) + offset
}

// nearest returns the place closest to a point, if one is close enough that
// the click was plainly meant for it. The threshold shrinks as the globe is
// zoomed in, so a click on a city at high zoom does not snap to its country.
func (v *Viewer) nearest(lat, lon float64) (geo.Place, bool) {
	limit := 7.0 / clamp(v.globe.Zoom, 1, 12)
	best, bestDist := geo.Place{}, limit
	for _, p := range v.ix.places {
		d := angularDistance(lat, lon, p.Lat, p.Lon)
		if d < bestDist {
			best, bestDist = p, d
		}
	}
	return best, bestDist < limit
}

// angularDistance is the great-circle distance between two points, in degrees.
func angularDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const rad = math.Pi / 180
	s := math.Sin(lat1*rad)*math.Sin(lat2*rad) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Cos((lon2-lon1)*rad)
	return math.Acos(clamp(s, -1, 1)) / rad
}

// advance moves the animation on: the flight if one is in the air, the spin
// otherwise.
func (v *Viewer) advance(now time.Time) {
	dt := now.Sub(v.last).Seconds()
	v.last = now
	if dt > 0.5 {
		dt = 0.5 // a suspended application should not spin on resume
	}

	if v.fly != nil {
		// A flight of no duration arrives at once. Timing it instead divides
		// by zero, and 0/0 is NaN — which flows into the latitude, into the
		// land mask, and out as an index far outside the array.
		p := 1.0
		if v.flyFor > 0 {
			p = float64(now.Sub(v.fly.started)) / float64(v.flyFor)
		}
		if p >= 1 || math.IsNaN(p) {
			v.globe.Lat, v.globe.Lon, v.globe.Zoom = v.fly.toLat, v.fly.toLon, v.fly.toZoom
			v.fly = nil
		} else {
			e := ease(p)
			v.globe.Lat = lerp(v.fly.fromLat, v.fly.toLat, e)
			v.globe.Lon = v.fly.fromLon + shortWay(v.fly.fromLon, v.fly.toLon)*e
			v.globe.Zoom = lerp(v.fly.fromZoom, v.fly.toZoom, e)
		}
		return
	}
	if v.spinning {
		v.globe.Lon = wrapLon(v.globe.Lon + spinRate*dt)
	}
}

// ease is a cosine in and out, so a flight starts and lands gently.
func ease(p float64) float64 {
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p * p * (3 - 2*p)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// shortWay is the signed distance from one longitude to another the short way
// round, so a flight from Tokyo to San Francisco crosses the Pacific rather
// than flying back over Europe.
func shortWay(from, to float64) float64 {
	d := wrapLon(to - from)
	return d
}

func clampLat(lat float64) float64 { return clamp(lat, -89.5, 89.5) }
func clampZoom(z float64) float64  { return clamp(z, minZoom, maxZoom) }

// FlyTo swings the globe round to a place and pins it.
func (v *Viewer) FlyTo(p geo.Place, now time.Time) {
	zoom := 2.6
	if p.Kind == geo.City {
		zoom = 5.0
	}
	v.fly = &flight{
		fromLat: v.globe.Lat, fromLon: v.globe.Lon, fromZoom: v.globe.Zoom,
		toLat: clampLat(p.Lat), toLon: wrapLon(p.Lon), toZoom: zoom,
		started: now, name: p.Name,
	}
	v.Pin(p.Display(), p.Lat, p.Lon, now.Add(v.flyFor/2))
	v.status = "→ " + p.Display()
	v.spinning = false
}

// Select flies to the nth search result, as clicking it does.
func (v *Viewer) Select(n int, now time.Time) bool {
	if n < 0 || n >= len(v.hits) {
		return false
	}
	v.results.Selected = n
	v.FlyTo(v.hits[n].Place, now)
	return true
}

// refresh recomputes the search results. It runs when the query changes, not
// every frame.
func (v *Viewer) refresh() {
	v.hits = v.ix.Search(v.query.Value(), 200)
	v.rows = v.rows[:0]
	for _, m := range v.hits {
		v.rows = append(v.rows, resultRow(m.Place))
	}
	if v.results.Selected >= len(v.rows) {
		v.results.Selected = len(v.rows) - 1
	}
	if v.results.Selected < 0 && len(v.rows) > 0 {
		v.results.Selected = 0
	}
	v.results.Offset = 0
}

// resultRow is the line shown for a place: its name, and what it is.
func resultRow(p geo.Place) string {
	var b strings.Builder
	b.Grow(len(p.Name) + len(p.Local) + len(p.Region) + 8)
	b.WriteString(p.Display())
	b.WriteString("  ")
	if p.Kind == geo.City {
		b.WriteString(p.Region)
		if p.Population > 0 {
			b.WriteString(" · ")
			b.WriteString(strconv.Itoa(p.Population / 1000))
			b.WriteString("k")
		}
	} else {
		b.WriteString(p.Region)
	}
	return b.String()
}

// key handles one key press and reports whether the application keeps running.
func (v *Viewer) key(f *limoni.Frame, k driver.KeyEvent, now time.Time) bool {
	if v.searching {
		return v.searchKey(f, k, now)
	}

	switch k.Type {
	case limoni.KeyLeft:
		v.nudge(0, -5)
	case limoni.KeyRight:
		v.nudge(0, 5)
	case limoni.KeyUp:
		v.nudge(5, 0)
	case limoni.KeyDown:
		v.nudge(-5, 0)
	case limoni.KeyEsc:
		return false
	case limoni.KeyRune:
		switch k.Ch {
		case 'q':
			return false
		case '/':
			v.focus(f, searchID)
			v.spinning = false
			v.status = ""
		case '+', '=':
			v.zoomBy(1.35)
		case '-', '_':
			v.zoomBy(1 / 1.35)
		case ' ':
			v.spinning = !v.spinning
			v.fly = nil
		case 'm':
			lat, lon := v.globe.Centre()
			v.Pin(formatCoord(lat, lon), lat, lon, now)
			v.status = "pinned " + formatCoord(lat, lon)
		case 'c':
			v.globe.Markers = v.globe.Markers[:0]
			v.pinRows = v.pinRows[:0]
			v.pins.Selected = -1
			v.status = "pins cleared"
		case 'a':
			v.globe.ASCII = !v.globe.ASCII
		case 'g':
			v.globe.Graticule = !v.globe.Graticule
		case '?':
			v.help = !v.help
		case 'r':
			v.Look(20, 20, 1)
			v.spinning = true
			v.status = ""
		}
	}
	return true
}

func (v *Viewer) searchKey(f *limoni.Frame, k driver.KeyEvent, now time.Time) bool {
	switch k.Type {
	case limoni.KeyEsc:
		v.focus(f, globeID)
		v.query.SetValue("")
		v.refresh()
	case limoni.KeyEnter:
		if v.Select(v.results.Selected, now) {
			v.focus(f, globeID)
		}
	case limoni.KeyUp:
		if v.results.Selected > 0 {
			v.results.Selected--
		}
	case limoni.KeyDown:
		if v.results.Selected+1 < len(v.rows) {
			v.results.Selected++
		}
	default:
		before := v.query.Value()
		v.query.HandleKey(k)
		if v.query.Value() != before {
			v.refresh()
		}
	}
	return true
}

// focus moves the keyboard to a widget and remembers where it went.
func (v *Viewer) focus(f *limoni.Frame, id string) {
	if f != nil && f.FocusManager != nil {
		f.FocusManager.SetFocused(id)
	}
	v.searching = id == searchID
}

func (v *Viewer) nudge(dLat, dLon float64) {
	v.fly = nil
	v.spinning = false
	// A nudge should feel the same on screen however far the globe is zoomed
	// in, so the step shrinks as the zoom grows.
	scale := 1 / clamp(v.globe.Zoom, 1, maxZoom)
	v.globe.Lat = clampLat(v.globe.Lat + dLat*scale)
	v.globe.Lon = wrapLon(v.globe.Lon + dLon*scale)
}

func (v *Viewer) zoomBy(factor float64) {
	v.fly = nil
	v.globe.Zoom = clampZoom(v.globe.Zoom * factor)
}

// ZoomBy is zoomBy for a caller outside the package, such as a test.
func (v *Viewer) ZoomBy(factor float64) { v.zoomBy(factor) }

// Status returns the line under the globe, for tests.
func (v *Viewer) Status() string { return v.status }

// Results returns the current search results, for tests.
func (v *Viewer) Results() []match { return v.hits }

// Wake asks for a redraw if the host installed one.
func (v *Viewer) Wake() {
	if v.wake != nil {
		v.wake()
	}
}

// SetSpinning starts or stops the rotation.
func (v *Viewer) SetSpinning(on bool) { v.spinning = on }

// Moving reports whether anything is animating and the view therefore needs
// another frame: the rotation, a flight, or a ping still expanding.
func (v *Viewer) Moving() bool {
	if v.spinning || v.fly != nil {
		return true
	}
	now := time.Now()
	for i := range v.globe.Markers {
		if p := v.globe.Markers[i].Pinged; !p.IsZero() && now.Sub(p) < pingFor {
			return true
		}
	}
	return false
}

// Find looks a place up by name, the way the -at flag does: an exact match on
// either name or the ISO code first, then the best search hit.
func (v *Viewer) Find(name string) (geo.Place, bool) {
	q := fold(strings.TrimSpace(name))
	if q == "" {
		return geo.Place{}, false
	}
	for _, p := range v.ix.places {
		if fold(p.Name) == q || fold(p.Local) == q || (p.Code != "" && fold(p.Code) == q) {
			return p, true
		}
	}
	if hits := v.ix.Search(name, 1); len(hits) > 0 {
		return hits[0].Place, true
	}
	return geo.Place{}, false
}

// Suggest returns names close to a query, for an error message that helps.
func (v *Viewer) Suggest(name string, n int) []string {
	q := fold(name)
	out := make([]string, 0, n)
	if len(q) > 2 {
		q = q[:2]
	}
	for _, hit := range v.ix.Search(q, n) {
		out = append(out, hit.Place.Name)
	}
	if len(out) == 0 {
		out = append(out, "Turkey", "Japan", "Brazil")
	}
	return out
}

// PinPlace pins a place and pings it, without flying anywhere.
func (v *Viewer) PinPlace(p geo.Place, now time.Time) {
	v.Pin(p.Display(), p.Lat, p.Lon, now)
}

// SetClock replaces the clock the animation reads, so that a test can decide
// what time it is instead of sleeping.
func (v *Viewer) SetClock(now func() time.Time) {
	if now != nil {
		v.clock = now
	}
}

// SetFlightDuration sets how long flying to a place takes. Zero arrives on
// the next frame, which is what a test that is not testing the animation
// wants.
func (v *Viewer) SetFlightDuration(d time.Duration) {
	if d < 0 {
		d = 0
	}
	v.flyFor = d
}
