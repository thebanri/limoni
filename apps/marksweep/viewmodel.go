package main

import "math"

// The sweeper in the player's hands has a small 3D engine of its own: a
// mesh built once from boxes, hulls and prisms, posed every frame so its
// barrel points through the crosshair, lit, rasterised with a depth buffer
// into half-block pixels, outlined, and laid over the picture. The muzzle
// flash and the tracer are light added to what is behind them. Nothing in
// it allocates per frame.

type v3 struct{ x, y, z float64 }

func (a v3) add(b v3) v3          { return v3{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a v3) sub(b v3) v3          { return v3{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a v3) mul(k float64) v3     { return v3{a.x * k, a.y * k, a.z * k} }
func (a v3) dot(b v3) float64     { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a v3) cross(b v3) v3        { return v3{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x} }
func (a v3) unit() v3             { return a.mul(1 / math.Max(1e-9, math.Sqrt(a.dot(a)))) }
func lerp3(a, b v3, t float64) v3 { return a.add(b.sub(a).mul(t)) }

// m3 is a rotation, row by row.
type m3 [9]float64

func (m m3) apply(v v3) v3 {
	return v3{m[0]*v.x + m[1]*v.y + m[2]*v.z, m[3]*v.x + m[4]*v.y + m[5]*v.z, m[6]*v.x + m[7]*v.y + m[8]*v.z}
}

func (m m3) mul(n m3) m3 {
	var r m3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i*3+j] = m[i*3]*n[j] + m[i*3+1]*n[3+j] + m[i*3+2]*n[6+j]
		}
	}
	return r
}

// The camera looks down +z, with +x to the right and +y up. A positive yaw
// turns +z towards +x, a positive pitch raises it, a positive roll leans
// the top to the left.
func rotY(a float64) m3 { c, s := math.Cos(a), math.Sin(a); return m3{c, 0, s, 0, 1, 0, -s, 0, c} }
func rotX(a float64) m3 { c, s := math.Cos(a), math.Sin(a); return m3{1, 0, 0, 0, c, s, 0, -s, c} }
func rotZ(a float64) m3 { c, s := math.Cos(a), math.Sin(a); return m3{c, -s, 0, s, c, 0, 0, 0, 1} }

// ── the mesh ─────────────────────────────────────────────────────────────

type material uint8

const (
	mBody   material = iota // gunmetal
	mDark                   // the darker plates
	mTrim                   // bright metal
	mLemon                  // the glowing strips: they flare when it fires
	mGlass                  // the heap's empty glass
	mBytes                  // the heap's glow: as tall as the rounds left
	mGrip                   // rubber
	mGlove                  //
	mSleeve                 //
	nMat
)

// matBase is each material's colour; matGlow is 1 for light it gives off
// rather than takes.
var (
	matBase = [nMat]rgb{hex(0x56627f), hex(0x262c3c), hex(0xaab6d0), hex(0xffd23a), hex(0x10232e), hex(0x5ad1ff), hex(0x201c1a), hex(0x34313d), hex(0x3a2440)}
	matGlow = [nMat]float32{mLemon: 1, mBytes: 1, mGlass: 0.35}
	matSpec = [nMat]float32{mBody: 0.6, mDark: 0.35, mTrim: 0.9, mGlass: 0.8}
)

// A part moves on its own: the heap drops out on a reload, and its glow is
// as tall as the rounds left.
type part uint8

const (
	pGun   part = iota
	pHeap       // the heap's cage
	pFill       // the glow, scaled from the bottom
	pEmpty      // the glass above it, scaled from the top
	nPart
)

type face struct {
	a, b, c uint16
	m       material
}

type mesh struct {
	v  []v3
	vp []part
	f  []face
}

// hull adds a six-sided solid from its eight corners, numbered by bits:
// x is bit 0, y bit 1 and z bit 2.
func (m *mesh) hull(c [8]v3, mat material, p part) {
	base := uint16(len(m.v))
	for _, v := range c {
		m.v = append(m.v, v)
		m.vp = append(m.vp, p)
	}
	for _, q := range [6][4]uint16{{0, 2, 6, 4}, {1, 3, 7, 5}, {0, 1, 5, 4}, {2, 3, 7, 6}, {0, 1, 3, 2}, {4, 5, 7, 6}} {
		m.f = append(m.f, face{base + q[0], base + q[1], base + q[2], mat}, face{base + q[0], base + q[2], base + q[3], mat})
	}
}

func (m *mesh) box(lo, hi v3, mat material, p part) {
	var c [8]v3
	for i := range c {
		c[i] = v3{pick(i&1 != 0, hi.x, lo.x), pick(i&2 != 0, hi.y, lo.y), pick(i&4 != 0, hi.z, lo.z)}
	}
	m.hull(c, mat, p)
}

func pick(b bool, t, f float64) float64 {
	if b {
		return t
	}
	return f
}

// prism adds a closed prism of n sides around the axis from a to b.
func (m *mesh) prism(a, b v3, r float64, n int, mat material, p part) {
	ax := b.sub(a).unit()
	side := v3{1, 0, 0}
	if math.Abs(ax.x) > 0.9 {
		side = v3{0, 1, 0}
	}
	u := ax.cross(side).unit()
	w := ax.cross(u)
	base := uint16(len(m.v))
	for i := 0; i < n; i++ {
		t := (float64(i) + 0.5) / float64(n) * 2 * math.Pi
		o := u.mul(math.Cos(t) * r).add(w.mul(math.Sin(t) * r))
		m.v = append(m.v, a.add(o), b.add(o))
		m.vp = append(m.vp, p, p)
	}
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		i0, i1, j0, j1 := base+uint16(2*i), base+uint16(2*i+1), base+uint16(2*j), base+uint16(2*j+1)
		m.f = append(m.f, face{i0, j0, j1, mat}, face{i0, j1, i1, mat})
		if i > 0 && i < n-1 {
			m.f = append(m.f, face{base, i0, j0, mat}, face{base + 1, i1, j1, mat})
		}
	}
}

// Where the heap's glow starts and ends, for scaling it.
const heapBot, heapTop = -0.168, -0.058

// sweeper is the gun: a compact receiver with glowing strips, a shrouded
// barrel, a caged canister of glowing bytes under it (the heap, which is
// the magazine), a grip, and the glove and sleeve holding it. Its barrel
// runs along +z; the origin is above the grip.
var sweeper = func() *mesh {
	m := &mesh{}
	// The receiver, a little narrower at the front.
	m.hull([8]v3{
		{-0.042, -0.045, -0.17}, {0.042, -0.045, -0.17}, {-0.04, 0.05, -0.17}, {0.04, 0.05, -0.17},
		{-0.036, -0.04, 0.17}, {0.036, -0.04, 0.17}, {-0.032, 0.042, 0.17}, {0.032, 0.042, 0.17},
	}, mBody, pGun)
	m.box(v3{-0.024, 0.048, -0.14}, v3{0.024, 0.066, 0.13}, mDark, pGun)  // top cover
	m.box(v3{-0.02, 0.066, -0.14}, v3{-0.006, 0.084, -0.11}, mDark, pGun) // rear sight, notched
	m.box(v3{0.006, 0.066, -0.14}, v3{0.02, 0.084, -0.11}, mDark, pGun)
	m.box(v3{-0.006, 0.066, 0.1}, v3{0.006, 0.09, 0.114}, mLemon, pGun)   // front sight
	m.box(v3{-0.047, -0.004, -0.13}, v3{-0.04, 0.01, 0.13}, mLemon, pGun) // strips
	m.box(v3{0.04, -0.004, -0.13}, v3{0.047, 0.01, 0.13}, mLemon, pGun)
	m.box(v3{-0.044, 0.018, -0.15}, v3{0.044, 0.026, 0.15}, mTrim, pGun)    // a trim line
	m.box(v3{-0.034, -0.03, -0.19}, v3{0.034, 0.04, -0.17}, mBody, pGun)    // the back plate
	m.box(v3{-0.014, 0.012, -0.192}, v3{0.014, 0.022, -0.19}, mBytes, pGun) // its status light
	// The barrel: a shroud, the barrel itself, the muzzle.
	m.prism(v3{0, 0.004, 0.16}, v3{0, 0.004, 0.33}, 0.034, 8, mDark, pGun)
	m.prism(v3{0, 0.004, 0.33}, v3{0, 0.004, 0.42}, 0.016, 6, mTrim, pGun)
	m.prism(v3{0, 0.004, 0.41}, v3{0, 0.004, 0.455}, 0.026, 8, mBody, pGun)
	for _, z := range []float64{0.2, 0.245, 0.29} { // cooling rings
		m.prism(v3{0, 0.004, z}, v3{0, 0.004, z + 0.012}, 0.037, 8, mTrim, pGun)
	}
	// The heap: a cage with the glow inside it.
	hz := 0.07
	m.box(v3{-0.036, heapTop - 0.012, hz - 0.036}, v3{0.036, heapTop, hz + 0.036}, mDark, pHeap)
	m.box(v3{-0.036, heapBot - 0.014, hz - 0.036}, v3{0.036, heapBot, hz + 0.036}, mDark, pHeap)
	for _, c := range [4][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		x, z := c[0]*0.03, hz+c[1]*0.03
		m.box(v3{x - 0.006, heapBot, z - 0.006}, v3{x + 0.006, heapTop, z + 0.006}, mTrim, pHeap)
	}
	m.prism(v3{0, heapBot, hz}, v3{0, heapTop, hz}, 0.024, 8, mBytes, pFill)
	m.prism(v3{0, heapBot, hz}, v3{0, heapTop, hz}, 0.024, 8, mGlass, pEmpty)
	// The grip, raked back, its trigger guard, and the hand on it.
	m.hull([8]v3{
		{-0.026, -0.045, -0.15}, {0.026, -0.045, -0.15}, {-0.026, -0.045, -0.08}, {0.026, -0.045, -0.08},
		{-0.026, -0.2, -0.2}, {0.026, -0.2, -0.2}, {-0.026, -0.2, -0.13}, {0.026, -0.2, -0.13},
	}, mGrip, pGun)
	m.box(v3{-0.008, -0.075, -0.08}, v3{0.008, -0.064, 0.0}, mDark, pGun)
	m.hull([8]v3{
		{-0.04, -0.06, -0.19}, {0.04, -0.06, -0.19}, {-0.04, -0.055, -0.06}, {0.04, -0.055, -0.06},
		{-0.04, -0.16, -0.23}, {0.04, -0.16, -0.23}, {-0.04, -0.15, -0.12}, {0.04, -0.15, -0.12},
	}, mGlove, pGun)
	return m
}()

// ── the rasteriser ───────────────────────────────────────────────────────

const maxRows = 200 // the picture's rows the gun can be drawn into

// layer is the gun's picture: a pixel per column and half row. depth holds
// 1/z, 0 where nothing was drawn; a negative depth marks light to be added
// to what is behind rather than drawn over it.
type layer struct {
	w, h           int
	col            []rgb
	depth          []float32
	x0, y0, x1, y1 int // the box drawn in this frame
}

var gl = layer{col: make([]rgb, maxCols*2*maxRows), depth: make([]float32, maxCols*2*maxRows)}

// vmScratch is the mesh's vertices this frame, in the camera's space and on
// the screen.
var vmCam, vmScr = make([]v3, len(sweeper.v)), make([]v3, len(sweeper.v))

func (l *layer) begin(w, h int) {
	l.w, l.h = w, h
	l.x0, l.y0, l.x1, l.y1 = w, h, 0, 0
}

// grow takes the box in pixels into the box to clear and lay over.
func (l *layer) grow(x0, y0, x1, y1 int) {
	l.x0, l.y0 = max(0, min(l.x0, x0)), max(0, min(l.y0, y0))
	l.x1, l.y1 = min(l.w, max(l.x1, x1)), min(l.h, max(l.y1, y1))
}

// clear empties the box, first widened to whole cells: compose reads both
// halves of every cell it touches.
func (l *layer) clear() {
	l.y0 &^= 1
	l.y1 = min(l.h, (l.y1+1)&^1)
	for y := l.y0; y < l.y1; y++ {
		row := y * l.w
		for x := l.x0; x < l.x1; x++ {
			l.depth[row+x] = 0
		}
	}
}

// tri fills a triangle whose points are in pixels, with z their depth.
func (l *layer) tri(a, b, c v3, col rgb) {
	area := (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)
	if math.Abs(area) < 1e-6 {
		return
	}
	x0 := max(l.x0, int(math.Floor(min(a.x, b.x, c.x))))
	x1 := min(l.x1-1, int(math.Ceil(max(a.x, b.x, c.x))))
	y0 := max(l.y0, int(math.Floor(min(a.y, b.y, c.y))))
	y1 := min(l.y1-1, int(math.Ceil(max(a.y, b.y, c.y))))
	ia, ib, ic := 1/a.z, 1/b.z, 1/c.z
	inv := 1 / area
	for y := y0; y <= y1; y++ {
		py := float64(y) + 0.5
		for x := x0; x <= x1; x++ {
			px := float64(x) + 0.5
			w0 := ((b.x-px)*(c.y-py) - (b.y-py)*(c.x-px)) * inv
			w1 := ((c.x-px)*(a.y-py) - (c.y-py)*(a.x-px)) * inv
			w2 := 1 - w0 - w1
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			z := float32(w0*ia + w1*ib + w2*ic)
			k := y*l.w + x
			if z > l.depth[k] {
				l.depth[k], l.col[k] = z, col
			}
		}
	}
}

// glow adds light at pixel (x, y), over the gun or over the picture.
func (l *layer) glow(x, y int, c rgb) {
	if x < l.x0 || y < l.y0 || x >= l.x1 || y >= l.y1 {
		return
	}
	k := y*l.w + x
	if l.depth[k] == 0 {
		l.depth[k], l.col[k] = -1, c
		return
	}
	l.col[k] = l.col[k].add(c)
}

// ── posing and drawing ───────────────────────────────────────────────────

// pose is where the gun is this frame and what its parts are doing.
type pose struct {
	rot      m3
	at       v3
	heapDrop float64 // how far the heap has fallen out, in model units
	fill     float64 // 0…1, the glow's height
	flash    float64 // 0…1, the muzzle flash
	strip    rgb     // the strips' colour
}

// gunPose points the barrel at the crosshair, then adds what the hands
// do: sway behind the aim, bob with the steps, kick on a shot, and the
// reload, where the gun rolls over and the heap drops out and comes back.
func (g *game) gunPose(v *view, cx, cy, fp float64) pose {
	var p pose
	at := v3{0.2, -0.14, 0.78}
	// Bob with the steps, breathe when still.
	at.x += math.Cos(g.bob*0.5) * 0.01
	at.y += math.Abs(math.Sin(g.bob*0.5))*0.012 - 0.006 + math.Sin(g.clock*1.7)*0.002
	// The sweeper leans a little towards where it points.
	mx, my := g.crosshair()
	tx, ty := (float64(mx)+0.5-cx)/fp, -(float64(2*my+1)-cy)/fp
	at.x += tx * 0.05
	at.y += ty * 0.03
	at.x += g.swayX * 0.06
	at.y += g.swayY * 0.04
	// The barrel points at a point on the ray through the crosshair, near
	// enough that it angles in and its side shows. Any such point is drawn
	// on the crosshair, so the barrel's line on the screen meets it there.
	const reach = 2.5
	d := v3{tx * reach, ty * reach, reach}.sub(at)
	yaw := math.Atan2(d.x, d.z) + g.swayX*0.5
	pitch := math.Atan2(d.y, math.Hypot(d.x, d.z)) + g.swayY*0.4
	roll := -g.swayX * 0.6
	if g.dashT > 0 {
		roll += 0.25
	}
	// The kick: back, and up.
	k := math.Max(0, g.recoil)
	at.z -= k * 0.045
	at.y += k * 0.008
	pitch += k * k * 0.16
	p.fill = float64(g.ammo) / float64(max(1, g.st.mag))
	if g.reloadT > 0 && g.st.reload > 0 {
		t := 1 - g.reloadT/g.st.reload
		tilt := math.Sin(math.Pi * t)
		roll += tilt * 0.7
		pitch -= tilt * 0.22
		at.y -= tilt * 0.025
		switch {
		case t < 0.3: // the spent heap falls out
			p.heapDrop = ease(t/0.3) * 0.32
			p.fill = 0
		case t < 0.62:
			p.heapDrop, p.fill = 0.32, 1
		default: // a fresh one comes up
			p.heapDrop, p.fill = 0.32*(1-ease((t-0.62)/0.38)), 1
		}
	}
	p.rot = rotY(yaw).mul(rotX(pitch)).mul(rotZ(roll))
	p.at = at
	p.flash = math.Max(0, (g.recoil-0.55)/0.45)
	p.strip = matBase[mLemon].mul(float32(0.6 + 0.5*k))
	if g.ammo == 0 && g.reloadT <= 0 && int(g.clock*4)%2 == 0 {
		p.strip = hex(0xff3a3a)
	}
	return p
}

// screen is where point m of the gun's model is on the screen, in pixels.
func (p *pose) screen(m v3, cx, cy, fp float64) (x, y float64, c v3) {
	c = p.at.add(p.rot.apply(m))
	return cx + c.x/c.z*fp, cy - c.y/c.z*fp, c
}

// muzzleTip is the point of the model the flash and the tracer leave from.
var muzzleTip = v3{0, 0.004, 0.46}

func ease(t float64) float64 { t = math.Max(0, math.Min(1, t)); return t * t * (3 - 2*t) }

// The light on the gun, towards the light in the camera's space: from above,
// to the left and a little behind the viewer.
var (
	lightDir = v3{-0.45, 0.8, -0.4}.unit()
	rimCol   = hex(0x3d5a9a)
	flashCol = hex(0xffc860)
)

// drawGun draws the sweeper into the gun layer and lays it over the
// picture.
func (g *game) drawGun(v *view) {
	if v.h < 10 || v.w < 24 {
		return
	}
	h := min(v.h, maxRows)
	gl.begin(v.w, 2*h)
	cx, cy := float64(v.w)/2, (v.horizon-float64(v.top))*2
	// The gun's own lens: sized by the picture's height as much as its
	// width, so a wide, short window does not make it huge.
	fp := math.Min(v.f, float64(v.h)*2.4)
	p := g.gunPose(v, cx, cy, fp)
	m := sweeper
	sx, sy, muzzle := p.screen(muzzleTip, cx, cy, fp)

	// Pose every vertex, then project it.
	for i, mv := range m.v {
		switch m.vp[i] {
		case pFill:
			mv.y = heapBot + (mv.y-heapBot)*p.fill
		case pEmpty:
			mv.y = heapTop - (heapTop-mv.y)*(1-p.fill)
		}
		if m.vp[i] != pGun {
			mv.y -= p.heapDrop
			mv.z -= p.heapDrop * 0.25
		}
		c := p.at.add(p.rot.apply(mv))
		vmCam[i] = c
		z := math.Max(c.z, 0.02)
		vmScr[i] = v3{cx + c.x/z*fp, cy - c.y/z*fp, z}
		gl.grow(int(vmScr[i].x)-1, int(vmScr[i].y)-1, int(vmScr[i].x)+2, int(vmScr[i].y)+2)
	}
	r := (3 + 7*p.flash) * fp / 110
	if p.flash > 0 {
		gl.grow(int(sx-r*2)-1, int(sy-r*2)-1, int(sx+r*2)+2, int(sy+r*2)+2)
	}
	mx, my := g.crosshair()
	ex, ey := float64(mx)+0.5, float64(2*my+1)
	if g.recoil > 0.7 {
		gl.grow(int(min(sx, ex))-1, int(min(sy, ey))-1, int(max(sx, ex))+2, int(max(sy, ey))+2)
	}
	gl.clear()

	for _, f := range m.f {
		a, b, c := vmCam[f.a], vmCam[f.b], vmCam[f.c]
		if a.z < 0.03 || b.z < 0.03 || c.z < 0.03 {
			continue
		}
		n := b.sub(a).cross(c.sub(a)).unit()
		toEye := a.mul(-1).unit()
		if n.dot(toEye) < 0 {
			n = n.mul(-1) // lit from the side that is seen
		}
		gl.tri(vmScr[f.a], vmScr[f.b], vmScr[f.c], g.shadeFace(f.m, n, toEye, a, muzzle, &p))
	}

	// The muzzle flash: a hot core and four spikes.
	if p.flash > 0 {
		for y := int(sy - r*2); y <= int(sy+r*2); y++ {
			for x := int(sx - r*2); x <= int(sx+r*2); x++ {
				dx, dy := (float64(x)+0.5-sx)/r, (float64(y)+0.5-sy)/r
				d := math.Hypot(dx, dy)
				k := math.Max(0, 1-d) * 1.4
				k += math.Max(0, 1-math.Abs(dy)*4) * math.Max(0, 1-math.Abs(dx)/2)
				k += math.Max(0, 1-math.Abs(dx)*4) * math.Max(0, 1-math.Abs(dy)/2) * 0.7
				if k <= 0.02 {
					continue
				}
				c := flashCol.mul(float32(k * p.flash))
				if d < 0.45 {
					c = c.add(rgb{1, 1, 1}.mul(float32((0.45 - d) * 2 * p.flash)))
				}
				gl.glow(x, y, c)
			}
		}
	}
	// The tracer, from the muzzle to where the shot is going.
	if g.recoil > 0.7 {
		k := float32((g.recoil - 0.7) / 0.3)
		n := int(math.Max(math.Abs(ex-sx), math.Abs(ey-sy)))
		for i := 2; i < n; i++ {
			t := float64(i) / float64(n)
			fade := float32(1-t) * k
			gl.glow(int(sx+(ex-sx)*t), int(sy+(ey-sy)*t), flashCol.mix(rgb{1, 1, 1}, 0.4).mul(fade*0.9))
		}
	}
	gl.compose(v)
}

// shadeFace is a face's colour: its material lit by the light, a rim of the
// room's blue at grazing angles, and the muzzle flash on the faces turned
// to it.
func (g *game) shadeFace(mat material, n, toEye, at, muzzle v3, p *pose) rgb {
	base := matBase[mat]
	if mat == mLemon {
		base = p.strip
	}
	glow := matGlow[mat]
	if glow >= 1 {
		return base
	}
	diff := float32(math.Max(0, n.dot(lightDir)))
	c := base.mul(0.22 + 0.85*diff)
	if s := matSpec[mat]; s > 0 {
		hv := lightDir.add(toEye).unit()
		c = c.add(rgb{1, 1, 1}.mul(s * float32(math.Pow(math.Max(0, n.dot(hv)), 22)) * 0.55))
	}
	rim := float32(math.Pow(1-math.Max(0, n.dot(toEye)), 3))
	c = c.add(rimCol.mul(rim * 0.6))
	if p.flash > 0 {
		to := muzzle.sub(at)
		dist := math.Sqrt(to.dot(to))
		lit := math.Max(0, n.dot(to.mul(1/dist))) / (1 + dist*12)
		c = c.add(flashCol.mul(float32(lit * p.flash * 1.6)))
	}
	if glow > 0 {
		c = c.mix(base, glow)
	}
	return c
}

// compose lays the layer over the cells: a half block for every cell the
// gun reaches, its pixels over the picture's, with a dark line around the
// gun so it reads against anything behind it.
func (l *layer) compose(v *view) {
	c := v.c
	for cy := l.y0 / 2; cy*2 < l.y1; cy++ {
		y := v.top + cy
		for x := l.x0; x < l.x1; x++ {
			kt, kb := 2*cy*l.w+x, (2*cy+1)*l.w+x
			var dt, db float32
			if 2*cy < l.h {
				dt = l.depth[kt]
			}
			if 2*cy+1 < l.h {
				db = l.depth[kb]
			}
			if dt == 0 && db == 0 {
				continue
			}
			p := c.at(x, y)
			if p == nil {
				continue
			}
			top, bot := halves(p)
			top = l.over(x, 2*cy, dt, top)
			bot = l.over(x, 2*cy+1, db, bot)
			setHalves(p, top, bot)
		}
	}
}

// over is the colour of pixel (x, y) given what is behind it.
func (l *layer) over(x, y int, d float32, behind rgb) rgb {
	k := y*l.w + x
	switch {
	case d < 0:
		return behind.add(l.col[k])
	case d == 0:
		return behind
	}
	if l.edge(x-1, y) || l.edge(x+1, y) || l.edge(x, y-1) || l.edge(x, y+1) {
		return l.col[k].mul(0.4)
	}
	return l.col[k]
}

// edge reports whether pixel (x, y) is outside the gun. Off the screen is
// not: the gun goes on there.
func (l *layer) edge(x, y int) bool {
	if x < 0 || y < 0 || x >= l.w || y >= l.h {
		return false
	}
	if x < l.x0 || y < l.y0 || x >= l.x1 || y >= l.y1 {
		return true
	}
	return l.depth[y*l.w+x] <= 0
}
