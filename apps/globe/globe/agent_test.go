package globe

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/apps/globe/globe/geo"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/uitest"
)

// page starts the viewer the way a person or an agent meets it, with flights
// arriving at once so the test is about what happens, not how long it takes.
func page(t *testing.T, w, h uint16) (*uitest.Page, *Viewer) {
	t.Helper()
	v := New()
	v.SetFlightDuration(0)
	v.SetSpinning(false)
	return uitest.Run(t, w, h, v.Frame), v
}

// The whole point of the application, driven the way an agent drives it:
// find Turkey on the world map and show it. Nothing here touches a pixel or
// a key binding — it is the semantic tree from end to end.
func TestAnAgentCanFindTurkey(t *testing.T) {
	p, v := page(t, 120, 40)

	// The search box is in the tree, and clicking it gives it the keyboard.
	p.Expect(p.GetByID("search")).ToBeVisible()
	p.GetByID("search").Click()
	p.GetByID("search").Type("Turkey")

	// The results are list items, so they can be read and chosen by name.
	row := p.GetByRole("list-item", "Turkey  Asia")
	p.Expect(row).ToBeVisible()
	row.Click()

	// And the globe says where it is pointed, so the agent can confirm that
	// it worked without looking at the screen.
	p.Expect(p.GetByID("globe")).ToContainValue("39.3°N")
	p.Expect(p.GetByID("globe")).ToContainValue("34.5°E")

	if lat, lon := v.Globe().Centre(); lat < 38 || lat > 41 || lon < 33 || lon > 36 {
		t.Errorf("the globe is centred on (%.1f, %.1f), not on Turkey", lat, lon)
	}

	// The place is pinned, and the pin is on the side of the globe facing us.
	p.Expect(p.GetByRole("list-item", "Turkey  39.3°N 34.5°E")).ToBeVisible()
	if !strings.Contains(p.Screen(), "Turkey") {
		t.Error("the pin's label is not on screen")
	}
}

// The name is typed in Turkish and shown in English: the local name is what
// someone searches with, the English one is what the map is written in.
func TestTurkishNamesAreSearchable(t *testing.T) {
	for _, query := range []string{"Türkiye", "turkiye", "TÜRKİYE", "tr"} {
		t.Run(query, func(t *testing.T) {
			p, _ := page(t, 120, 40)
			p.GetByID("search").Click()
			p.GetByID("search").Type(query)
			p.Expect(p.GetByRole("list-item", "Turkey  Asia")).ToBeVisible()
		})
	}
}

// The city table is Natural Earth's 1:10m one, not the 243 cities of 1:110m:
// a provincial city is there, and so is one whose name starts with a dotted
// capital İ, which lowercases to something no one types.
func TestMidSizedCitiesAreFound(t *testing.T) {
	ix := newIndex(geo.All())
	for query, want := range map[string]string{
		"bursa":    "Bursa",
		"izmir":    "İzmir",
		"lyon":     "Lyon",
		"eskisehi": "Eskişehir",
		"porto al": "Porto Alegre",
	} {
		hits := ix.Search(query, 5)
		if len(hits) == 0 || hits[0].Place.Name != want {
			t.Errorf("%q: first hit is %v, want %s", query, hits, want)
		}
	}
}

// A city is findable too, and flying to one comes in closer than a country.
func TestFindingACityZoomsFurtherIn(t *testing.T) {
	p, v := page(t, 120, 40)
	p.GetByID("search").Click()
	p.GetByID("search").Type("Istanbul")
	p.GetByRole("list-item", "Istanbul  Turkey · 10061k").Click()

	p.Expect(p.GetByID("globe")).ToContainValue("41.1°N")
	if v.Globe().Zoom < 4 {
		t.Errorf("flying to a city stopped at zoom %.1f", v.Globe().Zoom)
	}
}

// The far side of the globe is hidden, so a pin there is not in the tree —
// an agent reading it sees what is visible, which is the point of the tree.
func TestAPinOnTheFarSideIsNotReported(t *testing.T) {
	p, v := page(t, 120, 40)
	v.Look(39, 35, 2) // over Turkey
	v.Pin("Fiji", -17.8, 178.0, time.Time{})

	p.Expect(p.GetByID("globe")).ToContainValue("39.0°N")
	tree := p.Tree()
	if findLabel(tree, "Fiji") {
		t.Error("a marker on the far side of the globe is being reported as visible")
	}

	v.Look(-17.8, 178, 2) // now over Fiji
	p.Expect(p.GetByRole("list-item", "Fiji")).ToBeVisible()
}

func findLabel(nodes []accessibility.AccessibilityNode, label string) bool {
	for _, n := range nodes {
		if n.Label == label {
			return true
		}
		if findLabel(n.Children, label) {
			return true
		}
	}
	return false
}

// Searching for something that does not exist says so with an empty list
// rather than by offering the wrong place.
func TestAnUnknownPlaceOffersNothing(t *testing.T) {
	p, _ := page(t, 120, 40)
	p.GetByID("search").Click()
	p.GetByID("search").Type("zzzzznowhere")
	p.Expect(p.GetByRole("list-item", "").Within(p.GetByID("results"))).ToHaveCount(0)
}

// Choosing a place hands the keyboard back to the globe. Without this, the
// next key goes into the search box: pressing + after choosing Istanbul
// typed a "+" instead of zooming, which is what a real agent run found.
func TestChoosingAPlaceReturnsTheKeyboardToTheGlobe(t *testing.T) {
	p, v := page(t, 120, 40)
	p.GetByID("search").Click()
	p.GetByID("search").Type("Istanbul")
	p.GetByRole("list-item", "Istanbul  Turkey · 10061k").Click()

	before := v.Globe().Zoom
	p.Press("+")
	if v.Globe().Zoom <= before {
		t.Errorf("+ after choosing a place did not zoom: %.1f then %.1f", before, v.Globe().Zoom)
	}
	if got := p.GetByID("search").Node().Value; got != "Istanbul" {
		t.Errorf("the key went into the search box: %q", got)
	}
}

// A marker says where on screen it is, so that whatever reads the tree can
// point at it rather than only know that it exists.
func TestAVisibleMarkerCarriesItsPositionOnScreen(t *testing.T) {
	p, v := page(t, 120, 40)
	v.Look(41.1, 29.0, 4)
	v.Pin("Istanbul", 41.1, 29.0, time.Time{})

	node := p.GetByRole("list-item", "Istanbul").Node()
	if node.Bounds.Width == 0 || node.Bounds.Height == 0 {
		t.Fatalf("the marker has no area on screen: %+v", node.Bounds)
	}
	// Centred on the marker, it must be drawn near the middle of the pane.
	if node.Bounds.X < 35 || node.Bounds.X > 50 || node.Bounds.Y < 15 || node.Bounds.Y > 24 {
		t.Errorf("the marker is at %d,%d, which is not the centre of the globe", node.Bounds.X, node.Bounds.Y)
	}
}

// A flight of no duration must arrive rather than divide by zero: 0/0 is NaN,
// and a NaN latitude reached the land mask as an index and panicked.
func TestAnInstantFlightDoesNotProduceNaN(t *testing.T) {
	p, v := page(t, 80, 30)
	v.SetFlightDuration(0)
	p.GetByID("search").Click()
	p.GetByID("search").Type("Fiji")
	p.GetByRole("list-item", "Fiji  Oceania").Click()

	lat, lon := v.Globe().Centre()
	if math.IsNaN(lat) || math.IsNaN(lon) {
		t.Fatalf("the view is at (%v, %v)", lat, lon)
	}
	p.Expect(p.GetByID("globe")).ToContainValue("17.8°S")
}

// p takes the panel off the screen and brings it back. What is not drawn is
// not in the semantic tree either, which is the point: a test or an agent
// should not be able to find a search box nobody can see.
func TestThePanelCanBeHiddenAndBroughtBack(t *testing.T) {
	p, v := page(t, 120, 40)
	p.Expect(p.GetByID("search")).ToBeVisible()
	wide := v.GlobeArea().Width

	p.Press("p")
	p.Expect(p.GetByID("search")).Not().ToBeVisible()
	p.Expect(p.GetByID("results")).Not().ToBeVisible()
	p.Expect(p.GetByID("globe")).ToBeVisible()
	if full := v.GlobeArea().Width; full <= wide {
		t.Errorf("the globe still has %d columns of %d; hiding the panel should give it the rest", full, wide)
	}

	p.Press("p")
	p.Expect(p.GetByID("search")).ToBeVisible()
	if back := v.GlobeArea().Width; back != wide {
		t.Errorf("the globe kept %d columns after the panel came back, want %d", back, wide)
	}
}

// Searching with the panel hidden would type into a box nobody can see, so
// / brings the panel back and takes the keyboard.
func TestSearchingBringsAHiddenPanelBack(t *testing.T) {
	p, _ := page(t, 120, 40)
	p.Press("p")
	p.Expect(p.GetByID("search")).Not().ToBeVisible()

	p.Press("/")
	p.Expect(p.GetByID("search")).ToBeVisible()
	p.GetByID("search").Type("Japan")
	p.Expect(p.GetByRole("list-item", "Japan  Asia")).ToBeVisible()
}

// The globe has the keyboard when the application opens. The focus manager
// gives the first widget that registers the focus, and that is the search
// box — so without claiming it, every key steered a text field instead of
// the world, and p, the arrows and the zoom all did nothing.
func TestTheGlobeHasTheKeyboardAtTheStart(t *testing.T) {
	p, v := page(t, 120, 40)
	before := v.Globe().Zoom
	p.Press("+")
	if v.Globe().Zoom <= before {
		t.Errorf("+ did not zoom on the first key: %.2f then %.2f", before, v.Globe().Zoom)
	}
	if got := p.GetByID("search").Node().Value; got != "" {
		t.Errorf("the key went into the search box: %q", got)
	}
}
