//go:build !tinygo

package buffer

import (
	"hash/maphash"

	"github.com/thebanri/limoni/core/cell"
)

// rowSeed seeds the row hashes; they are only compared within one process.
var rowSeed = maphash.MakeSeed()

// rowHash hashes a row's cells as memory, with the hardware's help where it
// has AES. It was FNV-1a over each field of each cell, four multiplications
// a cell, run over both buffers on every frame. A match is always confirmed
// with rowsEqual, so a collision costs a compare, never a wrong scroll.
func rowHash(row []cell.Cell) uint64 { return maphash.Bytes(rowSeed, cellBytes(row)) }
