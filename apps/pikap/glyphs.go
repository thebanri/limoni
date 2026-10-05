package main

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// ASCII by shape. A cell is split into six parts, two across and three
// down, and every printable ASCII glyph is described by how much ink it
// puts in each part — measured once, at start-up, from a real bitmap font.
// A cell of the picture is described the same way, by how bright each part
// is, and gets the glyph whose description is nearest. A diagonal edge then
// comes out as '/' or '\' and the rim of the record as '(' and ')', where
// a brightness ramp would give a staircase of '#'.
//
// The answer depends only on the six values, so it is cached, with each
// value in eight steps: 2^18 entries, filled as they are first needed.

// glyphSet is the characters a picture is drawn with: punctuation for
// shape and a few letters and signs for weight. With every letter in, a
// picture reads as text — a Q here, a B there — rather than as a picture.
const glyphSet = ` .,'` + "`" + `:;-_~"^!|/\()<>[]{}=+*?%#&@$ilrvxzcoJLT7Yj`

const (
	regions   = 6
	glyphBits = 3
	glyphLvls = 1 << glyphBits
)

type glyphTable struct {
	runes []rune
	desc  [][regions]float32
	cache []uint8 // index+1 into runes; 0 when not yet worked out
}

func newGlyphTable() *glyphTable {
	g := &glyphTable{cache: make([]uint8, 1<<(glyphBits*regions))}
	face := basicfont.Face7x13
	const w, h = 7, 13
	var peak float32
	for _, r := range glyphSet {
		img := image.NewAlpha(image.Rect(0, 0, w, h))
		d := font.Drawer{Dst: img, Src: image.Opaque, Face: face, Dot: fixed.P(0, face.Ascent)}
		d.DrawString(string(r))
		var desc [regions]float32
		// Each pixel is cut in 4×4 so the regions' borders, which fall
		// inside pixels, are shared out fairly.
		for y := 0; y < h*4; y++ {
			for x := 0; x < w*4; x++ {
				a := float32(img.AlphaAt(x/4, y/4).A) / 255
				if a == 0 {
					continue
				}
				rx := x * 2 / (w * 4)
				ry := y * 3 / (h * 4)
				desc[ry*2+rx] += a
			}
		}
		area := float32(w*4*h*4) / regions
		for i := range desc {
			desc[i] /= area
			peak = max(peak, desc[i])
		}
		g.runes = append(g.runes, r)
		g.desc = append(g.desc, desc)
	}
	// The densest part of the densest glyph is full ink; the rest scale
	// with it.
	for i := range g.desc {
		for j := range g.desc[i] {
			g.desc[i][j] /= peak
		}
	}
	return g
}

// match returns the glyph for six ink values in 0…1.
func (g *glyphTable) match(ink *[regions]float32) rune {
	key := 0
	for _, v := range ink {
		q := int(v*(glyphLvls-1) + 0.5)
		q = max(0, min(glyphLvls-1, q))
		key = key<<glyphBits | q
	}
	if c := g.cache[key]; c != 0 {
		return g.runes[c-1]
	}
	var want [regions]float32
	k := key
	for i := regions - 1; i >= 0; i-- {
		want[i] = float32(k&(glyphLvls-1)) / (glyphLvls - 1)
		k >>= glyphBits
	}
	best, bestD := 0, float32(1e9)
	for i, d := range g.desc {
		var dist float32
		for j := range d {
			e := d[j] - want[j]
			dist += e * e
		}
		if dist < bestD {
			best, bestD = i, dist
		}
	}
	g.cache[key] = uint8(best + 1)
	return g.runes[best]
}
