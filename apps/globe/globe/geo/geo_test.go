package geo

import (
	"runtime"
	"strings"
	"testing"
)

// Known points, picked to be far from any coastline so that the 1:110m source
// and the grid resolution cannot make the answer a matter of opinion.
func TestIsLandAtKnownPoints(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
		land     bool
	}{
		{"central Anatolia", 39.0, 33.0, true},
		{"the Sahara", 23.0, 13.0, true},
		{"the Amazon", -5.0, -60.0, true},
		{"Siberia", 62.0, 100.0, true},
		{"central Australia", -24.0, 134.0, true},
		{"Antarctica", -80.0, 0.0, true},
		{"the middle of the Pacific", 0.0, -140.0, false},
		{"the South Atlantic", -30.0, -20.0, false},
		{"the Indian Ocean", -20.0, 80.0, false},
		{"the North Pole", 89.5, 0.0, false},
		{"the Mediterranean", 34.5, 18.0, false},
	}
	for _, c := range cases {
		if got := IsLand(c.lat, c.lon); got != c.land {
			t.Errorf("%s (%.1f, %.1f): land = %v, want %v", c.name, c.lat, c.lon, got, c.land)
		}
	}
}

// Longitude wraps, so a caller may add a rotation without normalising it.
func TestLongitudeWraps(t *testing.T) {
	for _, lon := range []float64{13, 13 + 360, 13 - 360, 13 + 720} {
		if !IsLand(23, lon) {
			t.Errorf("the Sahara at longitude %.0f reads as sea", lon)
		}
	}
	if IsLand(0, -140) != IsLand(0, 220) {
		t.Error("-140 and 220 are the same meridian")
	}
}

// Latitudes past the poles are clamped rather than wrapping to the other
// hemisphere: a globe rotated too far must not turn the Arctic into Antarctica.
func TestLatitudeClampsAtThePoles(t *testing.T) {
	if IsLand(120, 0) != IsLand(90, 0) {
		t.Error("beyond the north pole should clamp to the north pole")
	}
	if IsLand(-120, 0) != IsLand(-90, 0) {
		t.Error("beyond the south pole should clamp to the south pole")
	}
}

// Every pixel of every frame calls this, so it may not allocate.
func TestIsLandDoesNotAllocate(t *testing.T) {
	IsLand(0, 0) // pay for the one-time decompression first
	if n := testing.AllocsPerRun(200, func() {
		IsLand(39.0, 33.0)
		runtime.GC()
	}); n != 0 {
		t.Errorf("IsLand allocated %v times per call", n)
	}
}

func TestPlacesTable(t *testing.T) {
	all := All()
	if len(all) < 400 {
		t.Fatalf("%d places, want every country and city", len(all))
	}

	var turkey, istanbul *Place
	for i := range all {
		switch {
		case all[i].Kind == Country && all[i].Name == "Turkey":
			turkey = &all[i]
		case all[i].Kind == City && all[i].Name == "Istanbul":
			istanbul = &all[i]
		}
	}
	if turkey == nil || istanbul == nil {
		t.Fatal("Turkey or Istanbul is missing from the table")
	}
	if turkey.Local != "Türkiye" {
		t.Errorf("Turkey's local name is %q", turkey.Local)
	}
	if turkey.Code != "TR" || turkey.Region != "Asia" {
		t.Errorf("Turkey: code %q, region %q", turkey.Code, turkey.Region)
	}
	if got := turkey.Display(); !strings.Contains(got, "Türkiye") || !strings.Contains(got, "Turkey") {
		t.Errorf("Display() = %q, want both names", got)
	}
	if istanbul.Region != "Turkey" || istanbul.Population < 1_000_000 {
		t.Errorf("Istanbul: region %q, population %d", istanbul.Region, istanbul.Population)
	}

	// A place's own point must be on land, or aiming the globe at it puts the
	// marker in the sea. Small island states are the exception the mask
	// cannot represent at 1:110m, so a handful are allowed to miss.
	misses := 0
	for _, p := range all {
		if p.Lat < -60 { // Antarctic research stations
			continue
		}
		if !IsLand(p.Lat, p.Lon) {
			misses++
		}
	}
	if misses > len(all)/10 {
		t.Errorf("%d of %d places are not on land in the mask", misses, len(all))
	}

	// Countries come first, and Display falls back to the plain name.
	if all[0].Kind != Country {
		t.Error("All should list countries first")
	}
	plain := Place{Name: "Chad"}
	if plain.Display() != "Chad" {
		t.Errorf("Display without a local name = %q", plain.Display())
	}
	if Country.String() != "country" || City.String() != "city" {
		t.Error("Kind.String")
	}
}
