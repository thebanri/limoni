//go:build ignore

// gen.go builds the two data files this package embeds, from Natural Earth's
// public-domain 1:110m vectors:
//
//	land.bin.gz     a land/sea bitmask on an equirectangular grid
//	borders.bin.gz  where one country meets another, at six resolutions
//	places.go       countries and cities, with the point to centre on
//
// Run it from this directory when the data needs rebuilding:
//
//	go run gen.go
//
// It downloads from the natural-earth-vector repository, so it needs network
// access; nothing else in the tree does.
package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
)

// The grid the land mask is rasterised onto. 2048x1024 is a little under a
// fifth of a degree per cell: finer than the 1:110m source can justify at the
// equator, and it is what keeps coastlines from turning into staircases when
// the globe is zoomed in. It gzips to a few tens of kilobytes because most of
// the planet is uniform ocean.
const (
	maskW = 2048
	maskH = 1024
)

const base = "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/"

type featureCollection struct {
	Features []struct {
		Properties map[string]any  `json:"properties"`
		Geometry   json.RawMessage `json:"geometry"`
	} `json:"features"`
}

type geometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run() error {
	land, err := fetch("ne_110m_land.geojson")
	if err != nil {
		return err
	}
	if err := writeMask(land); err != nil {
		return err
	}

	countries, err := fetch("ne_110m_admin_0_countries.geojson")
	if err != nil {
		return err
	}
	if err := writeBorders(countries); err != nil {
		return err
	}
	cities, err := fetch("ne_110m_populated_places_simple.geojson")
	if err != nil {
		return err
	}
	return writePlaces(countries, cities)
}

func fetch(name string) (*featureCollection, error) {
	resp, err := http.Get(base + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", name, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var fc featureCollection
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	fmt.Printf("%-44s %6d features  %5d KiB\n", name, len(fc.Features), len(body)/1024)
	return &fc, nil
}

// writeMask rasterises every land polygon onto the grid with a scanline fill.
// Even-odd parity is what makes holes work without treating them specially:
// a lake inside a landmass crosses the boundary twice and stays sea.
func writeMask(fc *featureCollection) error {
	mask := make([]byte, maskW*maskH/8)
	set := func(x, y int) {
		i := y*maskW + x
		mask[i/8] |= 1 << uint(i%8)
	}

	var xs []float64
	for _, f := range fc.Features {
		rings, err := polygons(f.Geometry)
		if err != nil {
			return err
		}
		for _, poly := range rings {
			for y := 0; y < maskH; y++ {
				// The centre of the row, in degrees.
				lat := 90.0 - (float64(y)+0.5)*180.0/maskH
				xs = xs[:0]
				for _, ring := range poly {
					for i := 0; i < len(ring); i++ {
						a, b := ring[i], ring[(i+1)%len(ring)]
						if (a[1] > lat) == (b[1] > lat) {
							continue
						}
						t := (lat - a[1]) / (b[1] - a[1])
						xs = append(xs, a[0]+t*(b[0]-a[0]))
					}
				}
				if len(xs) < 2 {
					continue
				}
				sort.Float64s(xs)
				for i := 0; i+1 < len(xs); i += 2 {
					x0 := int(math.Floor((xs[i] + 180) / 360 * maskW))
					x1 := int(math.Ceil((xs[i+1] + 180) / 360 * maskW))
					if x0 < 0 {
						x0 = 0
					}
					if x1 > maskW {
						x1 = maskW
					}
					for x := x0; x < x1; x++ {
						set(x, y)
					}
				}
			}
		}
	}

	f, err := os.Create("land.bin.gz")
	if err != nil {
		return err
	}
	defer f.Close()
	zw, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	if _, err := zw.Write(mask); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	st, _ := f.Stat()
	land := 0
	for _, b := range mask {
		for i := 0; i < 8; i++ {
			if b&(1<<uint(i)) != 0 {
				land++
			}
		}
	}
	fmt.Printf("land.bin.gz  %dx%d  %d KiB raw  %d KiB gzipped  %.1f%% land\n",
		maskW, maskH, len(mask)/1024, st.Size()/1024, float64(land)*100/float64(maskW*maskH))
	return nil
}

// borderLevels is how many resolutions of the border mask are written. Each
// is half the size of the one before, and a cell is set when any of the four
// it stands for is. That is what lets the globe ask "is there a border
// anywhere inside this pixel?" with one lookup at any zoom: a single-pixel
// line sampled at a coarse zoom would otherwise break into dots.
const borderLevels = 6

// writeBorders finds where one country meets another. It fills every country
// with its own number first and then marks the cells whose neighbour belongs
// to someone else, so the coast — land against sea, which is country against
// nothing — is not a border. The colour change at the water's edge already
// shows that, and drawing it again buries the small countries.
func writeBorders(fc *featureCollection) error {
	owner := make([]uint16, maskW*maskH)
	var xs []float64

	for n, f := range fc.Features {
		rings, err := polygons(f.Geometry)
		if err != nil {
			return err
		}
		id := uint16(n + 1)
		for _, poly := range rings {
			for y := 0; y < maskH; y++ {
				lat := 90.0 - (float64(y)+0.5)*180.0/maskH
				xs = xs[:0]
				for _, ring := range poly {
					for i := 0; i < len(ring); i++ {
						a, b := ring[i], ring[(i+1)%len(ring)]
						if (a[1] > lat) == (b[1] > lat) {
							continue
						}
						t := (lat - a[1]) / (b[1] - a[1])
						xs = append(xs, a[0]+t*(b[0]-a[0]))
					}
				}
				if len(xs) < 2 {
					continue
				}
				sort.Float64s(xs)
				for i := 0; i+1 < len(xs); i += 2 {
					x0 := int(math.Floor((xs[i] + 180) / 360 * maskW))
					x1 := int(math.Ceil((xs[i+1] + 180) / 360 * maskW))
					if x0 < 0 {
						x0 = 0
					}
					if x1 > maskW {
						x1 = maskW
					}
					for x := x0; x < x1; x++ {
						owner[y*maskW+x] = id
					}
				}
			}
		}
	}

	// A cell is a border when the cell to its right or below belongs to a
	// different country. Checking two neighbours rather than four draws each
	// line once, on one side of it.
	level0 := make([]byte, maskW*maskH/8)
	set := func(x, y int) {
		i := y*maskW + x
		level0[i/8] |= 1 << uint(i%8)
	}
	for y := 0; y < maskH; y++ {
		for x := 0; x < maskW; x++ {
			id := owner[y*maskW+x]
			if id == 0 {
				continue
			}
			right := owner[y*maskW+(x+1)%maskW] // the grid wraps at ±180
			if right != 0 && right != id {
				set(x, y)
			}
			if y+1 < maskH {
				if below := owner[(y+1)*maskW+x]; below != 0 && below != id {
					set(x, y)
				}
			}
		}
	}

	// The coarser levels: a cell is set when any of its four children is.
	levels := [][]byte{level0}
	w, h := maskW, maskH
	for k := 1; k < borderLevels; k++ {
		prev := levels[k-1]
		pw := w
		w, h = w/2, h/2
		cur := make([]byte, w*h/8)
		get := func(x, y int) bool {
			i := y*pw + x
			return prev[i/8]&(1<<uint(i%8)) != 0
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if get(2*x, 2*y) || get(2*x+1, 2*y) || get(2*x, 2*y+1) || get(2*x+1, 2*y+1) {
					i := y*w + x
					cur[i/8] |= 1 << uint(i%8)
				}
			}
		}
		levels = append(levels, cur)
	}

	f, err := os.Create("borders.bin.gz")
	if err != nil {
		return err
	}
	defer f.Close()
	zw, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	total := 0
	for _, level := range levels {
		if _, err := zw.Write(level); err != nil {
			return err
		}
		total += len(level)
	}
	if err := zw.Close(); err != nil {
		return err
	}
	st, _ := f.Stat()
	fmt.Printf("borders.bin.gz  %d levels  %d KiB raw  %d KiB gzipped\n",
		len(levels), total/1024, st.Size()/1024)
	return nil
}

// polygons returns every ring set of a Polygon or MultiPolygon geometry.
func polygons(raw json.RawMessage) ([][][][2]float64, error) {
	var g geometry
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, err
	}
	switch g.Type {
	case "Polygon":
		var poly [][][2]float64
		if err := json.Unmarshal(g.Coordinates, &poly); err != nil {
			return nil, err
		}
		return [][][][2]float64{poly}, nil
	case "MultiPolygon":
		var multi [][][][2]float64
		if err := json.Unmarshal(g.Coordinates, &multi); err != nil {
			return nil, err
		}
		return multi, nil
	default:
		return nil, fmt.Errorf("unexpected geometry %q", g.Type)
	}
}

func writePlaces(countries, cities *featureCollection) error {
	var b strings.Builder
	b.WriteString(`// Code generated by gen.go from Natural Earth 1:110m data. DO NOT EDIT.
//
// Natural Earth is in the public domain: https://www.naturalearthdata.com

package geo

// countries is every country Natural Earth draws at 1:110m, with the label
// point it places the name at — which is inside the country even where the
// centroid would not be, as for Norway or Indonesia.
var countries = []Place{
`)

	type row struct {
		name, tr, iso, region string
		lat, lon              float64
	}
	var rows []row
	for _, f := range countries.Features {
		p := f.Properties
		name, _ := p["NAME"].(string)
		if name == "" {
			continue
		}
		tr, _ := p["NAME_TR"].(string)
		iso, _ := p["ISO_A2"].(string)
		region, _ := p["CONTINENT"].(string)
		lon, ok1 := p["LABEL_X"].(float64)
		lat, ok2 := p["LABEL_Y"].(float64)
		if !ok1 || !ok2 {
			continue
		}
		if iso == "-99" {
			iso = ""
		}
		if tr == name {
			tr = ""
		}
		rows = append(rows, row{name, tr, iso, region, lat, lon})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	for _, r := range rows {
		fmt.Fprintf(&b, "\t{Name: %q, Local: %q, Code: %q, Region: %q, Lat: %.4f, Lon: %.4f, Kind: Country},\n",
			r.name, r.tr, r.iso, r.region, r.lat, r.lon)
	}
	b.WriteString("}\n\n// cities are Natural Earth's 1:110m populated places: capitals and the\n// largest cities, enough to aim the globe at a city rather than a country.\nvar cities = []Place{\n")

	type crow struct {
		name, country string
		lat, lon      float64
		pop           int
	}
	var crows []crow
	for _, f := range cities.Features {
		p := f.Properties
		name, _ := p["name"].(string)
		country, _ := p["adm0name"].(string)
		lat, ok1 := p["latitude"].(float64)
		lon, ok2 := p["longitude"].(float64)
		if name == "" || !ok1 || !ok2 {
			continue
		}
		pop, _ := p["pop_max"].(float64)
		crows = append(crows, crow{name, country, lat, lon, int(pop)})
	}
	sort.Slice(crows, func(i, j int) bool { return crows[i].name < crows[j].name })
	for _, c := range crows {
		fmt.Fprintf(&b, "\t{Name: %q, Region: %q, Lat: %.4f, Lon: %.4f, Population: %d, Kind: City},\n",
			c.name, c.country, c.lat, c.lon, c.pop)
	}
	b.WriteString("}\n")

	if err := os.WriteFile("places.go", []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("places.go    %d countries, %d cities\n", len(rows), len(crows))
	return nil
}
