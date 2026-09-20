package globe

import (
	"math"
	"strconv"
	"strings"
)

// wrapLon brings a longitude into [-180, 180), which is where a reader and a
// test both expect to find it after the globe has spun a few times round.
func wrapLon(lon float64) float64 {
	lon = math.Mod(lon+180, 360)
	if lon < 0 {
		lon += 360
	}
	return lon - 180
}

// formatCoord writes a coordinate the way an atlas does: "39.3°N 34.5°E".
func formatCoord(lat, lon float64) string {
	var b strings.Builder
	b.Grow(20)
	writeDeg(&b, lat, 'N', 'S')
	b.WriteByte(' ')
	writeDeg(&b, wrapLon(lon), 'E', 'W')
	return b.String()
}

func writeDeg(b *strings.Builder, v float64, pos, neg byte) {
	hemisphere := pos
	if v < 0 {
		hemisphere = neg
		v = -v
	}
	b.WriteString(strconv.FormatFloat(v, 'f', 1, 64))
	b.WriteString("°")
	b.WriteByte(hemisphere)
}

// formatZoom writes the zoom as a multiplier: "1.0×", "12.5×".
func formatZoom(z float64) string {
	if z <= 0 {
		z = 1
	}
	return strconv.FormatFloat(z, 'f', 1, 64) + "×"
}

// fold makes text comparable for searching: lower case, and Turkish letters
// reduced to the ASCII a reader is likely to type. Without it "turkiye" would
// not find "Türkiye", which is precisely the search someone makes first.
func fold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'ı':
			b.WriteByte('i')
		case 'ü':
			b.WriteByte('u')
		case 'ö':
			b.WriteByte('o')
		case 'ş':
			b.WriteByte('s')
		case 'ğ':
			b.WriteByte('g')
		case 'ç':
			b.WriteByte('c')
		case 'â', 'á', 'à', 'ä', 'å', 'ã':
			b.WriteByte('a')
		case 'é', 'è', 'ê', 'ë':
			b.WriteByte('e')
		case 'í', 'ì', 'î', 'ï':
			b.WriteByte('i')
		case 'ó', 'ò', 'ô', 'õ':
			b.WriteByte('o')
		case 'ú', 'ù', 'û':
			b.WriteByte('u')
		case 'ñ':
			b.WriteByte('n')
		case 'ß':
			b.WriteString("ss")
		case '\'', '’', '.', ',':
			// Dropped: "Côte d'Ivoire" should be found by "cote divoire"
			// and by "cote d ivoire" alike.
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
