//go:build tinygo

package buffer

import "github.com/thebanri/limoni/core/cell"

// rowHash for TinyGo, which lacks the runtime internals hash/maphash needs:
// two machine words at a time, multiplied and folded. Slower than maphash,
// but the browser playground is built with TinyGo and must keep building.
func rowHash(row []cell.Cell) uint64 {
	words := cellWords(row)
	h := uint64(0x9E3779B97F4A7C15)
	for i := 0; i+1 < len(words); i += 2 {
		h = (h ^ words[i]) * 0xBF58476D1CE4E5B9
		h = (h ^ words[i+1]) * 0x94D049BB133111EB
		h ^= h >> 31
	}
	return h
}
