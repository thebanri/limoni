package main

import (
	"math"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// The deck is drawn as a picture first — the record, the tonearm, the
// panel behind them — sampled a few times in every cell, and only then
// turned into characters: ASCII by shape (glyphs.go) or half blocks, two
// pixels to a cell. The picture lives in record units: the record's radius
// is 1, its centre is the origin, and y grows downwards. A cell is about
// twice as tall as it is wide, which the sampling accounts for.
//
// Colours are linear light while they are mixed, and sRGB when they reach
// a cell. Nothing here allocates once the textures are made for a size.

type rgb struct{ r, g, b float64 }

func (c rgb) add(o rgb) rgb       { return rgb{c.r + o.r, c.g + o.g, c.b + o.b} }
func (c rgb) scale(k float64) rgb { return rgb{c.r * k, c.g * k, c.b * k} }
func (c rgb) mix(o rgb, t float64) rgb {
	return rgb{c.r + (o.r-c.r)*t, c.g + (o.g-c.g)*t, c.b + (o.b-c.b)*t}
}
func (c rgb) lum() float64  { return 0.2126*c.r + 0.7152*c.g + 0.0722*c.b }
func (c rgb) mul(o rgb) rgb { return rgb{c.r * o.r, c.g * o.g, c.b * o.b} }

// lin is a colour given as sRGB hex, in linear light.
func lin(v uint32) rgb {
	return rgb{
		srgbToLinear(float64(v>>16&0xff) / 255),
		srgbToLinear(float64(v>>8&0xff) / 255),
		srgbToLinear(float64(v&0xff) / 255),
	}
}

// toSRGB8 is linearToSRGB as a table: it runs three times a cell.
var srgbLUT = func() (t [4097]uint8) {
	for i := range t {
		t[i] = uint8(math.Round(linearToSRGB(float64(i)/4096) * 255))
	}
	return
}()

func to8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return srgbLUT[int(v*4096)]
}

func (c rgb) color() cell.Color { return cell.NewColorRGB(to8(c.r), to8(c.g), to8(c.b)) }

// perceived is how light a colour looks, 0…1: sRGB-encoded luminance.
func perceived(c rgb) float64 { return float64(to8(c.lum())) / 255 }

type discStyle uint8

const (
	pictureDisc discStyle = iota // the cover over the whole record, as Spotify draws it
	blackVinyl                   // grooves, with the cover on the label
)

type glyphStyle uint8

const (
	asciiGlyphs glyphStyle = iota
	halfBlocks
)

// Where the tonearm is: its pivot, its length, and the radii the needle
// runs between.
const (
	pivotU, pivotV = 1.22, -0.9
	armLen         = 1.5
	grooveOut      = 0.94
)

func grooveIn(s discStyle) float64 {
	if s == blackVinyl {
		return 0.42
	}
	return 0.3
}

// needleAt is where the needle is when it plays radius r: where the circle
// the needle sweeps crosses the groove, on the side the arm hangs to.
func needleAt(r float64) (u, v float64) {
	d := math.Hypot(pivotU, pivotV)
	a := (r*r - armLen*armLen + d*d) / (2 * d)
	h := math.Sqrt(max(0, r*r-a*a))
	ex, ey := pivotU/d, pivotV/d
	// Of the two crossings, the one below the line to the pivot.
	return ex*a - ey*h, ey*a + ex*h
}

// armAngle is the angle of the arm about its pivot for a needle at (u, v).
func armAngle(u, v float64) float64 { return math.Atan2(v-pivotV, u-pivotU) }

// restAngle is the arm on its rest, off the record.
const restAngle = math.Pi/2 - 0.08

type sceneParams struct {
	angle  float64 // platter, revolutions
	arm    float64 // arm angle about the pivot, radians
	lifted float64 // 0 in the groove … 1 lifted
	style  discStyle
	tube   float64 // the arm's half-width; thinner in ASCII, where a line is a glyph
}

type scene struct {
	tex, label []rgb // the cover, resampled for the record and for its label
	texN, lblN int
	texArt     *art
	texR       int
	texStyle   discStyle

	// Set per frame from sceneParams.
	cosA, sinA float64
	armDX      float64
	armDY      float64
	shellX     float64 // the headshell's direction
	shellY     float64
	armBox     [4]float64 // u0, v0, u1, v1: nothing of the arm lies outside
	// The headshell is a few pixels: drawn point by point like the tube it
	// would be speckled, so it is one colour, the opposite of what is
	// around it, and changes side only past a margin, or it would flicker
	// as the record turns under it.
	shellBody rgb
	shellDark bool
	shellInit bool
	aa        float64
	p         sceneParams
}

// prepare resamples the cover for a record of radius R cells.
func (s *scene) prepare(a *art, R int, style discStyle) {
	if a == s.texArt && R == s.texR && style == s.texStyle {
		return
	}
	s.texArt, s.texR, s.texStyle = a, R, style
	s.texN = max(16, min(artSize, 4*R))
	s.tex = resample(s.tex, a, s.texN)
	s.lblN = max(8, min(artSize, int(4*float64(R)*0.36*1.3)))
	s.label = resample(s.label, a, s.lblN)
}

func resample(dst []rgb, a *art, n int) []rgb {
	if cap(dst) < n*n {
		dst = make([]rgb, n*n)
	}
	dst = dst[:n*n]
	for y := 0; y < n; y++ {
		sy0, sy1 := y*artSize/n, max(y*artSize/n+1, (y+1)*artSize/n)
		for x := 0; x < n; x++ {
			sx0, sx1 := x*artSize/n, max(x*artSize/n+1, (x+1)*artSize/n)
			var sum rgb
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					sum = sum.add(a.px[sy*artSize+sx])
				}
			}
			dst[y*n+x] = sum.scale(1 / float64((sy1-sy0)*(sx1-sx0)))
		}
	}
	return dst
}

// sampleTex reads a texture bilinearly at (x, y) in 0…1.
func sampleTex(t []rgb, n int, x, y float64) rgb {
	fx := max(0, min(float64(n)-1.001, x*float64(n)-0.5))
	fy := max(0, min(float64(n)-1.001, y*float64(n)-0.5))
	ix, iy := int(fx), int(fy)
	ax, ay := fx-float64(ix), fy-float64(iy)
	i := iy*n + ix
	top := t[i].mix(t[i+1], ax)
	bot := t[i+n].mix(t[i+n+1], ax)
	return top.mix(bot, ay)
}

// cover is how much of a shape a sample sees, from its signed distance
// (negative inside): an edge a sample wide, not a step.
func (s *scene) cover(d float64) float64 {
	return max(0, min(1, 0.5-d/s.aa))
}

// segDist is the distance from (u, v) to the segment from a to b.
func segDist(u, v, ax, ay, bx, by float64) (d, t float64) {
	dx, dy := bx-ax, by-ay
	t = ((u-ax)*dx + (v-ay)*dy) / (dx*dx + dy*dy)
	t = max(0, min(1, t))
	ex, ey := u-ax-t*dx, v-ay-t*dy
	return math.Sqrt(ex*ex + ey*ey), t
}

// pow24 is |x|^24, which the gloss is shaped by, without math.Pow.
func pow24(x float64) float64 {
	x *= x              // 2
	x *= x              // 4
	x2 := x * x         // 8
	return x2 * x2 * x2 // 24
}

var (
	holeCol    = lin(0x050506)
	centreCol  = lin(0x1d1b24)
	ringCol    = lin(0x77778a)
	vinylCol   = lin(0x0e0e11)
	sheenCol   = lin(0xc8ccd8)
	armCol     = lin(0xd2d3d8)
	weightCol  = lin(0x5b5b62)
	baseCol    = lin(0x2c2c32)
	baseTopCol = lin(0x8d8d96)
	bodyCol    = lin(0x07070a)
)

func (s *scene) begin(p sceneParams, R float64) {
	s.p = p
	th := p.angle * 2 * math.Pi
	s.cosA, s.sinA = math.Cos(th), math.Sin(th)
	s.armDX, s.armDY = math.Cos(p.arm), math.Sin(p.arm)
	s.aa = 0.9 / R
	s.shellX = s.armDX*math.Cos(0.35) - s.armDY*math.Sin(0.35)
	s.shellY = s.armDX*math.Sin(0.35) + s.armDY*math.Cos(0.35)
	nx, ny := pivotU+s.armDX*armLen, pivotV+s.armDY*armLen
	wx, wy := pivotU-s.armDX*0.33, pivotV-s.armDY*0.33
	const pad = 0.25 // the base, the weight's thickness, the shadow
	s.armBox = [4]float64{
		min(nx, wx, pivotU) - pad, min(ny, wy, pivotV) - pad,
		max(nx, wx, pivotU) + pad, max(ny, wy, pivotV) + pad,
	}
	s.shadeShell(nx-s.armDX*0.1, ny-s.armDY*0.1)
}

// opposite is the colour across from c, for the arm to be seen on
// whatever is under it: the hue turned round (the sRGB negative) and the
// lightness taken to the far end, very dark over anything light and very
// light over anything dark. A plain negative of a mid grey is the same
// grey; this never is.
func opposite(c rgb) rgb { return oppositeAs(c, perceived(c) >= 0.5) }

// oppositeAs is opposite with the side given: dark, or light.
func oppositeAs(c rgb, dark bool) rgb {
	o := rgb{negLUT[to8(c.r)], negLUT[to8(c.g)], negLUT[to8(c.b)]}
	const darkL, lightL = 0.0137, 0.71 // 0.12 and 0.86 perceived
	if dark {
		if l := o.lum(); l > darkL {
			o = o.scale(darkL / l)
		}
	} else if l := o.lum(); l < lightL {
		o = o.mix(rgb{1, 1, 1}, (lightL-l)/(1-l))
	}
	return o
}

// negLUT is the linear light of the sRGB negative of each 8-bit channel.
var negLUT = func() (t [256]float64) {
	for i := range t {
		t[i] = srgbToLinear(float64(255-i) / 255)
	}
	return
}()

// shadeShell picks the headshell's colour from the record around (cu, cv).
func (s *scene) shadeShell(cu, cv float64) {
	var sum rgb
	for _, o := range [...][2]float64{{0, 0}, {0.06, 0}, {-0.06, 0}, {0, 0.06}, {0, -0.06}} {
		u, v := cu+o[0], cv+o[1]
		c := lin(0x16161a) // off the record, the panel: dark
		if r := math.Sqrt(u*u + v*v); r < 1 {
			c = s.disc(u, v, r)
		}
		sum = sum.add(c)
	}
	avg := sum.scale(1.0 / 5)
	switch l := perceived(avg); {
	case !s.shellInit:
		s.shellDark = l >= 0.5
	case l > 0.56:
		s.shellDark = true
	case l < 0.44:
		s.shellDark = false
	}
	s.shellInit = true
	s.shellBody = oppositeAs(avg, s.shellDark)
}

// at is the picture at (u, v) over the background bg, and how much of it
// is something solid — the record or the arm — rather than the panel.
func (s *scene) at(u, v float64, bg rgb) (rgb, float64) {
	c := bg
	solid := 0.0
	r := math.Sqrt(u*u + v*v)
	if cov := s.cover(r - 1); cov > 0 {
		c = c.mix(s.disc(u, v, r), cov)
		solid = cov
	}
	return s.arm(u, v, c, solid)
}

// disc is the record at (u, v), r from the centre.
func (s *scene) disc(u, v, r float64) rgb {
	// Where (u, v) was before the platter turned.
	lu := u*s.cosA + v*s.sinA
	lv := -u*s.sinA + v*s.cosA
	ir := 1 / max(r, 1e-9)
	var c rgb
	if s.p.style == pictureDisc {
		c = sampleTex(s.tex, s.texN, (lu+1)/2, (lv+1)/2)
		// The rim catches less light.
		if r > 0.95 {
			c = c.scale(1 - (r-0.95)*5)
		}
		// A faint gloss that stays where the light is while the record
		// turns under it.
		// cos(φ+0.7), φ the angle of (u, v).
		g := pow24((u*0.764842-v*0.644218)*ir) * 0.05 * smoothstep(0.25, 0.4, r)
		c = c.mix(sheenCol, g)
		// The centre: a dark disc with a light ring, as Spotify has it.
		if r < 0.235 {
			centre := centreCol
			if ring := math.Abs(r-0.13) - 0.009; ring < s.aa {
				centre = centre.mix(ringCol, s.cover(ring))
			}
			c = c.mix(centre, s.cover(r-0.22))
		}
	} else {
		// Black vinyl. The grooves are rings of slightly different
		// darkness — the same all the way round, so they do not show the
		// turning, as they do not on a real record. The light does:
		// two wedges of sheen, fixed, as from a lamp.
		ring := hashRing(r * 90)
		c = vinylCol.scale(0.85 + 0.35*ring)
		gap := false
		// The gaps between songs, a little brighter.
		for _, g := range [...]float64{0.55, 0.68, 0.81} {
			if math.Abs(r-g) < 0.006 {
				gap = true
			}
		}
		// cos(φ-0.75), squared nine times over.
		cs := (u*0.731689 + v*0.681639) * ir
		cs *= cs
		sheen := cs * cs * cs
		sheen = sheen * sheen * sheen
		sheen *= smoothstep(0.4, 0.55, r) * (1 - smoothstep(0.93, 1, r))
		k := 0.18 * sheen * (0.55 + 0.9*ring)
		if gap {
			k = k*1.8 + 0.03
		}
		c = c.mix(sheenCol, k)
		// The lead-in edge is a lighter bevel.
		if r > 0.975 {
			c = c.mix(sheenCol, 0.08)
		}
		if r < 0.37 {
			lbl := sampleTex(s.label, s.lblN, (lu/0.36+1)/2, (lv/0.36+1)/2)
			c = c.mix(lbl, s.cover(r-0.36))
			// The dead wax between label and grooves.
			if w := math.Abs(r-0.38) - 0.02; w < s.aa {
				c = c.mix(vinylCol.scale(1.4), s.cover(w))
			}
		}
	}
	// The spindle.
	if r < 0.05 {
		c = c.mix(holeCol, s.cover(r-0.03))
	}
	return c
}

// hashRing is fixed noise by ring: 0…1.
func hashRing(x float64) float64 {
	i := int(x)
	f := x - float64(i)
	a := hash2(i, 7, 99)
	b := hash2(i+1, 7, 99)
	return a + (b-a)*f*f*(3-2*f)
}

// arm draws the tonearm over c: its base, a counterweight behind the
// pivot, the tube, and the headshell at the needle, with its shadow on the
// record.
func (s *scene) arm(u, v float64, c rgb, solid float64) (rgb, float64) {
	if b := &s.armBox; u < b[0] || v < b[1] || u > b[2] || v > b[3] {
		return c, solid
	}
	// The tube and the headshell are the opposite of what is under them,
	// point by point, so they stand out from any cover — and from both
	// sides at once where the arm lies along the record's edge.
	under := c
	dx, dy := s.armDX, s.armDY
	lift := s.p.lifted
	L := armLen
	nx, ny := pivotU+dx*L, pivotV+dy*L
	// The shadow, cast down and right, further when the arm is lifted.
	so := 0.035 + 0.05*lift
	if d, _ := segDist(u-so, v-so*1.4, pivotU, pivotV, nx-dx*0.05, ny-dy*0.05); d < 0.06 {
		c = c.scale(1 - 0.45*s.cover(d-0.03))
	}
	// Base.
	pd := math.Sqrt((u-pivotU)*(u-pivotU) + (v-pivotV)*(v-pivotV))
	if cov := s.cover(pd - 0.15); cov > 0 {
		base := baseCol.mix(baseTopCol, s.cover(pd-0.085)*0.8)
		c = c.mix(base, cov)
		solid = max(solid, cov)
	}
	// Counterweight.
	if d, _ := segDist(u, v, pivotU-dx*0.13, pivotV-dy*0.13, pivotU-dx*0.33, pivotV-dy*0.33); d < 0.12 {
		cov := s.cover(d - 0.075)
		c = c.mix(weightCol.scale(0.8+0.5*(1-d/0.075)), cov)
		solid = max(solid, cov)
	}
	// Tube: brighter along its middle, as a cylinder is, with a rim on
	// the other side of light around it.
	if d, _ := segDist(u, v, pivotU, pivotV, nx-dx*0.17, ny-dy*0.17); d < 0.08 {
		body := opposite(under)
		rim := opposite(body)
		w := s.p.tube + 1.2*s.aa
		if rc := s.cover(d - w); rc > 0 {
			c = c.mix(rim, rc*0.55)
		}
		cov := s.cover(d - s.p.tube)
		c = c.mix(body.scale(0.6+0.5*max(0, 1-d/s.p.tube)), cov)
		solid = max(solid, cov)
	}
	// Headshell, turned in a little from the tube, as they are, and drawn
	// like it.
	hx, hy := s.shellX, s.shellY
	if d, _ := segDist(u, v, nx-dx*0.2, ny-dy*0.2, nx-hx*0.01, ny-hy*0.01); d < 0.12 {
		shell := s.shellBody
		rim := oppositeAs(shell, !s.shellDark)
		if rc := s.cover(d - 0.05 - 1.2*s.aa); rc > 0 {
			c = c.mix(rim, rc*0.55)
		}
		cov := s.cover(d - 0.05)
		c = c.mix(shell.scale(0.7+0.4*max(0, 1-d/0.05)), cov)
		solid = max(solid, cov)
	}
	// The pivot cap over everything.
	if cov := s.cover(pd - 0.05); cov > 0 {
		c = c.mix(armCol, cov)
	}
	return c, solid
}

// deckView is where on screen the deck is, in cells.
type deckView struct {
	x0, y0, w, h int
	cx, cy       float64 // the record's centre, in cells
	R            float64 // its radius in columns; in rows it is R/2
}

// deckExtent is the picture's extent in record units: the record and the
// arm with its base and weight.
const (
	extL, extR = -1.04, 1.5
	extT, extB = -1.12, 1.04
)

// fitDeck places the largest deck that fits in w×h cells at (x, y).
func fitDeck(x, y, w, h int) (deckView, bool) {
	// Columns: R·(extR-extL); rows: R/2·(extB-extT).
	R := min(float64(w)/(extR-extL), float64(h)*2/(extB-extT))
	R = math.Floor(R)
	if R < 6 {
		return deckView{}, false
	}
	dw := int(math.Ceil(R * (extR - extL)))
	dh := int(math.Ceil(R / 2 * (extB - extT)))
	dv := deckView{x0: x + (w-dw)/2, y0: y + (h-dh)/2, w: dw, h: dh, R: R}
	dv.cx = float64(dv.x0) - extL*R
	dv.cy = float64(dv.y0) - extT*R/2
	return dv, true
}

// toUnits turns a screen position (cell units, fractional) into record
// units.
func (dv deckView) toUnits(x, y float64) (u, v float64) {
	return (x - dv.cx) / dv.R, (y - dv.cy) / (dv.R / 2)
}

// draw renders the deck into b. rows holds the panel's colour for each row.
func (s *scene) draw(b *buffer.Buffer, dv deckView, p sceneParams, gs glyphStyle, gt *glyphTable, rows []rgb) {
	W, H := int(b.Area.Width), int(b.Area.Height)
	s.begin(p, dv.R)
	sx, sy := 2, 3
	if gs == halfBlocks {
		sy = 4
	}
	var samples [8]rgb
	var solids [8]float64
	for y := dv.y0; y < dv.y0+dv.h; y++ {
		if y < 0 || y >= H {
			continue
		}
		bg := rows[y]
		for x := dv.x0; x < dv.x0+dv.w; x++ {
			if x < 0 || x >= W {
				continue
			}
			k := 0
			for j := 0; j < sy; j++ {
				for i := 0; i < sx; i++ {
					u, v := dv.toUnits(float64(x)+(float64(i)+0.5)/float64(sx), float64(y)+(float64(j)+0.5)/float64(sy))
					samples[k], solids[k] = s.at(u, v, bg)
					k++
				}
			}
			idx := y*W + x
			if gs == halfBlocks {
				top := samples[0].add(samples[1]).add(samples[2]).add(samples[3]).scale(0.25)
				bot := samples[4].add(samples[5]).add(samples[6]).add(samples[7]).scale(0.25)
				b.Content[idx] = cell.Cell{Content: '▀', Style: cell.Style{Fg: top.color(), Bg: bot.color()}}
				continue
			}
			b.Content[idx] = asciiCell(&samples, &solids, bg, gt)
		}
	}
}

// asciiCell turns six samples into a glyph: the solid parts of the picture
// darken the cell's background, so a black record stays black, and what is
// lighter than that background is ink.
func asciiCell(samples *[8]rgb, solids *[8]float64, bg rgb, gt *glyphTable) cell.Cell {
	solid := 0.0
	for i := 0; i < regions; i++ {
		solid += solids[i]
	}
	solid /= regions
	cellBg := bg.mix(bodyCol, solid*0.92)
	lb := perceived(cellBg)
	var ink [regions]float32
	var fg rgb
	var wsum, peak float64
	for i := 0; i < regions; i++ {
		e := (perceived(samples[i]) - lb) / (1 - lb)
		e = max(0, min(1, e))
		ink[i] = float32(e)
		fg = fg.add(samples[i].scale(e + 0.02))
		wsum += e + 0.02
		peak = max(peak, e)
	}
	fg = fg.scale(1 / wsum)
	// Within the cell, the light parts are made lighter than the dark
	// ones, relative to the lightest: edges choose edge glyphs rather than
	// the grey middle of the set.
	if peak > 0 {
		for i := range ink {
			n := float64(ink[i]) / peak
			ink[i] = float32(n * math.Sqrt(n) * peak)
		}
	}
	r := gt.match(&ink)
	if r == ' ' {
		return cell.Cell{Content: ' ', Style: cell.Style{Fg: cellBg.color(), Bg: cellBg.color()}}
	}
	// A glyph covers only part of its cell, so its colour is lifted to
	// make up some of the light the gaps between its strokes lose.
	if l := fg.lum(); l > 0 {
		lift := min(2.2, 1/math.Sqrt(max(0.2, peak)))
		fg = fg.scale(lift)
		if m := max(fg.r, fg.g, fg.b); m > 1 {
			fg = fg.scale(1 / m)
		}
	}
	return cell.Cell{Content: r, Style: cell.Style{Fg: fg.color(), Bg: cellBg.color()}}
}
