package globe

import (
	"sort"
	"strings"

	"github.com/thebanri/limoni/apps/globe/globe/geo"
)

// index is the searchable form of the place table, folded once at startup so
// that typing a letter does not refold seven thousand names.
type index struct {
	places []geo.Place
	keys   []string // folded "name|local|code|region", one per place
}

func newIndex(places []geo.Place) *index {
	ix := &index{places: places, keys: make([]string, len(places))}
	for i, p := range places {
		ix.keys[i] = fold(p.Name + "\x00" + p.Local + "\x00" + p.Code + "\x00" + p.Region)
	}
	return ix
}

// match is a place that matched a query, with the rank that put it there.
type match struct {
	Place geo.Place
	Index int
	rank  int
}

// Search returns the places matching a query, best first, at most limit of
// them. An empty query returns the countries in their own order, which is
// what makes the list useful before anything has been typed.
//
// Ranking, in order: a name that starts with the query beats one that merely
// contains it; a country beats a city; a larger city beats a smaller one.
// So "tur" offers Turkey before Turkmenistan's capital, and "ist" offers
// Istanbul before Istria.
func (ix *index) Search(query string, limit int) []match {
	q := fold(strings.TrimSpace(query))
	out := make([]match, 0, limit)

	if q == "" {
		for i, p := range ix.places {
			if p.Kind != geo.Country {
				continue
			}
			if len(out) == limit {
				break
			}
			out = append(out, match{Place: p, Index: i})
		}
		return out
	}

	for i, key := range ix.keys {
		at := strings.Index(key, q)
		if at < 0 {
			continue
		}
		p := ix.places[i]
		rank := 0
		if at == 0 || key[at-1] == 0x00 || key[at-1] == ' ' {
			rank += 100 // a word in the name starts with the query
		}
		if p.Kind == geo.Country {
			rank += 40
		}
		if p.Population > 0 {
			rank += min(30, p.Population/500_000)
		}
		if fold(p.Code) == q {
			rank += 200 // "TR" means Turkey and nothing else
		}
		out = append(out, match{Place: p, Index: i, rank: rank})
	}

	sort.SliceStable(out, func(a, b int) bool {
		if out[a].rank != out[b].rank {
			return out[a].rank > out[b].rank
		}
		return len(out[a].Place.Name) < len(out[b].Place.Name)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
