// X-Ray: a night city that shows what the renderer actually sends.
//
//	go run ./examples/xray          # interactive
//	go run ./examples/xray -film    # a 20-second loop with no UI hints, for recording
//
// The scene looks like it is all moving — stars, a train, lit windows, water —
// yet most of the screen is identical from one frame to the next. Press x and
// an X-ray sweeps across: every cell that changed since the previous frame
// glows, everything else goes dark, and a panel reports how many bytes went to
// the terminal for this frame against a full repaint of the same picture.
//
// The numbers are measured, not estimated. The scene is drawn into its own
// buffer pair each frame and encoded with Limoni's diff (buffer.DiffWithOptions,
// with the same options this terminal gets) and with a full-screen stream
// (the same encoder against a back buffer of another size, which is what a
// resize triggers) for comparison. They cover the city only, not the
// panel drawn over it. Press space to freeze the city: nothing changes, so
// nothing is sent.
//
// Keys: x  toggle the X-ray · space  freeze · q / Esc  quit
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

const fps = 30

func main() {
	film := flag.Bool("film", false, "run a scripted loop with no key hints, for recording")
	flag.Parse()

	term, err := limoni.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer term.Close()

	caps := term.Capabilities()
	s := newShow(*film, buffer.DiffOptions{
		TrueColor:     caps.TrueColor,
		Colors256:     caps.Colors256,
		EraseChar:     caps.EraseChar,
		RepeatChar:    caps.RepeatChar,
		ScrollRegions: caps.ScrollRegions,
		InsertDelete:  caps.ScrollRegions,
	})

	app := limoni.NewApp(term, limoni.WithFPS(fps), limoni.WithTitle("Limoni · X-Ray"))
	if err := app.Run(context.Background(), s.frame); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ── colours ──────────────────────────────────────────────────────────────

type rgb struct{ r, g, b float64 }

func (c rgb) color() cell.Color {
	return cell.NewColorRGB(clamp8(c.r), clamp8(c.g), clamp8(c.b))
}

func (c rgb) mix(o rgb, t float64) rgb {
	return rgb{c.r + (o.r-c.r)*t, c.g + (o.g-c.g)*t, c.b + (o.b-c.b)*t}
}

func (c rgb) scale(k float64) rgb { return rgb{c.r * k, c.g * k, c.b * k} }

func clamp8(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v)
}

func toRGB(c cell.Color) rgb {
	if c.Type() != cell.ColorRGB {
		return rgb{}
	}
	r, g, b := c.RGB()
	return rgb{float64(r), float64(g), float64(b)}
}

var (
	skyTop     = rgb{6, 8, 24}
	skyHorizon = rgb{52, 26, 70}
	moonLight  = rgb{246, 236, 196}
	windowWarm = rgb{255, 206, 120}
	windowCool = rgb{170, 220, 255}
	windowOff  = rgb{34, 34, 54}
	railSteel  = rgb{70, 76, 96}
	trainBody  = rgb{44, 50, 70}
	waterDeep  = rgb{4, 8, 22}

	xrayCold = rgb{16, 40, 54}
	xrayBg   = rgb{2, 5, 9}
	xrayHot  = rgb{160, 255, 250}
	xrayGlow = rgb{0, 92, 110}
	accent   = rgb{80, 230, 220}
)

// craters on the moon: centre and radius, in moon radii.
var craters = [][3]float64{
	{-0.35, -0.25, 0.24}, {0.32, 0.2, 0.2}, {0.05, -0.55, 0.13},
	{-0.25, 0.5, 0.16}, {0.55, -0.3, 0.1},
}

// ── the city ─────────────────────────────────────────────────────────────

type star struct {
	x, y   int
	period int // frames between twinkles
	offset int
}

type window struct {
	x, y int
	lit  bool
	cool bool
}

type building struct {
	x0, x1, top int
	body        rgb
}

type city struct {
	w, h, horizon int

	stars     []star
	buildings []building
	windows   []window
	antennaX  int
	antennaY  int
	moonX     float64
	moonY     float64
	moonR     float64

	// shooting star
	shootOn    bool
	shootX     float64
	shootY     float64
	shootAge   int
	shootTimer int

	trainX    float64
	trainLen  int
	trainWait int

	glints []int // water cells glinting this frame, as indices
	glintT []int // frames left for each glint
}

func newCity(w, h int, rng *rand.Rand) *city {
	c := &city{w: w, h: h}
	water := max(3, h/4)
	c.horizon = h - water

	c.moonR = math.Max(2, float64(h)/9)
	c.moonX = float64(w) * 0.8
	c.moonY = float64(h) * 0.2

	for i := 0; i < w*c.horizon/28; i++ {
		c.stars = append(c.stars, star{
			x:      rng.Intn(w),
			y:      rng.Intn(max(1, c.horizon*2/3)),
			period: 40 + rng.Intn(120),
			offset: rng.Intn(160),
		})
	}

	// Skyline: buildings stand on the row above the water.
	tallest, tallestX := 0, 0
	for x := 0; x < w; {
		bw := 4 + rng.Intn(8)
		minH := c.horizon / 4
		bh := minH + rng.Intn(max(1, c.horizon*3/5-minH))
		if rng.Intn(6) == 0 {
			bh = c.horizon * 3 / 4 // an occasional tower
		}
		top := c.horizon - bh
		shade := 16 + rng.Float64()*14
		b := building{x0: x, x1: min(w, x+bw), top: top, body: rgb{shade, shade + 2, shade + 16}}
		c.buildings = append(c.buildings, b)
		if bh > tallest {
			tallest, tallestX = bh, x+bw/2
		}
		for wy := top + 1; wy < c.horizon-3; wy += 2 {
			for wx := x + 1; wx < b.x1-1; wx += 2 {
				c.windows = append(c.windows, window{
					x: wx, y: wy,
					lit:  rng.Intn(3) != 0,
					cool: rng.Intn(5) == 0,
				})
			}
		}
		x += bw + rng.Intn(2)
	}
	c.antennaX, c.antennaY = tallestX, c.horizon-tallest-3

	c.trainLen = max(18, w/3)
	c.trainX = float64(w + 4)
	c.shootTimer = 60
	return c
}

// step advances the city by one frame. Randomness comes from rng, so a run
// with the same seed and size replays exactly.
func (c *city) step(frame int, rng *rand.Rand) {
	// A window or two changes every second.
	if len(c.windows) > 0 && rng.Intn(fps/2) == 0 {
		i := rng.Intn(len(c.windows))
		c.windows[i].lit = !c.windows[i].lit
	}

	// The train crosses right to left, then waits.
	if c.trainWait > 0 {
		c.trainWait--
	} else {
		c.trainX -= 0.9
		if c.trainX < float64(-c.trainLen-6) {
			c.trainX = float64(c.w + 4)
			c.trainWait = fps * 2
		}
	}

	// A shooting star now and then.
	if c.shootOn {
		c.shootX += 1.6
		c.shootY += 0.55
		c.shootAge++
		if c.shootAge > 26 || int(c.shootY) >= c.horizon-2 {
			c.shootOn = false
			c.shootTimer = fps*4 + rng.Intn(fps*4)
		}
	} else if c.shootTimer--; c.shootTimer <= 0 {
		c.shootOn = true
		c.shootAge = 0
		c.shootX = float64(rng.Intn(max(1, c.w/2)))
		c.shootY = float64(rng.Intn(max(1, c.horizon/4)))
	}

	// Glints on the water: a few cells at a time, each lasting a few frames.
	water := (c.h - c.horizon) * c.w
	for i := 0; i < len(c.glints); {
		c.glintT[i]--
		if c.glintT[i] <= 0 {
			last := len(c.glints) - 1
			c.glints[i], c.glintT[i] = c.glints[last], c.glintT[last]
			c.glints, c.glintT = c.glints[:last], c.glintT[:last]
			continue
		}
		i++
	}
	for n := 0; n < 2 && water > 0; n++ {
		c.glints = append(c.glints, rng.Intn(water))
		c.glintT = append(c.glintT, 4+rng.Intn(8))
	}
	_ = frame
}

func (c *city) draw(b *buffer.Buffer, frame int) {
	w, h := c.w, c.h
	set := func(x, y int, r rune, fg, bg rgb) {
		if x < 0 || y < 0 || x >= w || y >= h {
			return
		}
		b.Content[y*w+x] = cell.Cell{Content: r, Style: cell.Style{Fg: fg.color(), Bg: bg.color()}}
	}
	skyAt := func(y int) rgb {
		t := float64(y) / float64(max(1, c.horizon-1))
		return skyTop.mix(skyHorizon, t*t)
	}

	// Sky, with a halo around the moon.
	for y := 0; y < c.horizon; y++ {
		base := skyAt(y)
		for x := 0; x < w; x++ {
			dx := (float64(x) - c.moonX) / 2
			dy := float64(y) - c.moonY
			d := math.Sqrt(dx*dx+dy*dy) / (c.moonR * 3.2)
			col := base
			if d < 1 {
				col = base.mix(rgb{60, 58, 84}, (1-d)*(1-d)*0.8)
			}
			set(x, y, ' ', col, col)
		}
	}

	// Stars twinkle one at a time.
	for _, s := range c.stars {
		bg := toRGB(b.Content[s.y*w+s.x].Style.Bg)
		phase := (frame + s.offset) % s.period
		switch {
		case phase < 3:
			set(s.x, s.y, '✦', rgb{255, 250, 220}, bg)
		case phase < 6:
			set(s.x, s.y, '+', rgb{200, 200, 230}, bg)
		default:
			set(s.x, s.y, '·', rgb{130, 130, 170}, bg)
		}
	}

	// The moon, at twice the vertical resolution with half blocks.
	for y := int(c.moonY - c.moonR - 1); y <= int(c.moonY+c.moonR+1); y++ {
		for x := int(c.moonX - c.moonR*2 - 2); x <= int(c.moonX+c.moonR*2+2); x++ {
			if x < 0 || y < 0 || x >= w || y >= c.horizon {
				continue
			}
			// Position in moon radii, for the upper and lower half of the cell.
			at := func(sub float64) (mx, my float64) {
				return (float64(x) - c.moonX) / 2 / c.moonR, (float64(y) + sub - c.moonY) / c.moonR
			}
			in := func(sub float64) bool {
				mx, my := at(sub)
				return mx*mx+my*my <= 1
			}
			upper, lower := in(0.25), in(0.75)
			if !upper && !lower {
				continue
			}
			shade := func(sub float64) rgb {
				mx, my := at(sub)
				k := 1 - 0.2*(mx*mx+my*my)
				for _, cr := range craters {
					dx, dy := mx-cr[0], my-cr[1]
					if dx*dx+dy*dy < cr[2]*cr[2] {
						k -= 0.13
					}
				}
				return moonLight.scale(k)
			}
			bg := b.Content[y*w+x].Style.Bg
			switch {
			case upper && lower:
				set(x, y, '▀', shade(0.25), shade(0.75))
			case upper:
				set(x, y, '▀', shade(0.25), toRGB(bg))
			default:
				set(x, y, '▄', shade(0.75), toRGB(bg))
			}
		}
	}

	// Shooting star and its trail.
	if c.shootOn {
		for i := 0; i < 7; i++ {
			x := int(c.shootX - float64(i)*1.6)
			y := int(c.shootY - float64(i)*0.55)
			if y < 0 || y >= c.horizon {
				continue
			}
			k := 1 - float64(i)/7
			r := '━'
			if i == 0 {
				r = '✦'
			}
			set(x, y, r, rgb{255, 245, 210}.scale(0.35+0.65*k), skyAt(y))
		}
	}

	// Buildings and windows.
	for _, bd := range c.buildings {
		for y := max(0, bd.top); y < c.horizon; y++ {
			for x := bd.x0; x < bd.x1; x++ {
				set(x, y, ' ', bd.body, bd.body)
			}
		}
		// Roof edge catches a little moonlight.
		for x := bd.x0; x < bd.x1; x++ {
			if bd.top >= 0 {
				set(x, bd.top, '▁', bd.body.mix(moonLight, 0.25), skyAt(bd.top))
			}
		}
	}
	for _, wn := range c.windows {
		body := toRGB(b.Content[wn.y*w+wn.x].Style.Bg)
		switch {
		case wn.lit && wn.cool:
			set(wn.x, wn.y, '▪', windowCool, body)
		case wn.lit:
			set(wn.x, wn.y, '▪', windowWarm, body)
		default:
			set(wn.x, wn.y, '▪', windowOff, body)
		}
	}

	// Antenna with a slow red beacon.
	if c.antennaY >= 0 {
		for y := c.antennaY + 1; y < c.antennaY+3; y++ {
			set(c.antennaX, y, '│', rgb{90, 90, 110}, skyAt(y))
		}
		if (frame/fps)%2 == 0 {
			set(c.antennaX, c.antennaY, '●', rgb{255, 60, 70}, skyAt(c.antennaY))
		} else {
			set(c.antennaX, c.antennaY, '●', rgb{90, 30, 40}, skyAt(c.antennaY))
		}
	}

	// Elevated rail and the train on it.
	rail := c.horizon - 1
	for x := 0; x < w; x++ {
		set(x, rail, '▀', railSteel, toRGB(b.Content[rail*w+x].Style.Bg))
	}
	tx := int(math.Round(c.trainX))
	for i := 0; i < c.trainLen; i++ {
		x := tx + i
		if x < 0 || x >= w {
			continue
		}
		car := i % 12
		roof, body := rail-3, rail-2
		switch {
		case i == 0:
			set(x, roof, '▗', trainBody, toRGB(b.Content[roof*w+x].Style.Bg))
			set(x, body, '█', trainBody, trainBody)
		case car == 11:
			// the gap between two cars
		default:
			set(x, roof, '▄', trainBody, toRGB(b.Content[roof*w+x].Style.Bg))
			if car%3 == 1 {
				set(x, body, '█', windowCool.mix(windowWarm, 0.3), trainBody)
			} else {
				set(x, body, ' ', trainBody, trainBody)
			}
		}
		set(x, rail, '▀', railSteel, trainBody.scale(0.7))
	}
	// Headlight.
	if tx > 0 && tx <= w {
		for i := 1; i <= 4; i++ {
			x := tx - i
			if x >= 0 && x < w {
				bg := toRGB(b.Content[(rail-2)*w+x].Style.Bg)
				set(x, rail-2, ' ', bg, bg.mix(rgb{255, 240, 180}, 0.5/float64(i)))
			}
		}
	}

	// Water: the city mirrored, darkened and blued, with glints.
	for y := c.horizon; y < h; y++ {
		src := c.horizon - 1 - (y - c.horizon)
		depth := float64(y-c.horizon+1) / float64(h-c.horizon+1)
		for x := 0; x < w; x++ {
			s := b.Content[src*w+x]
			fg := toRGB(s.Style.Fg).mix(waterDeep, 0.45+0.35*depth)
			bg := toRGB(s.Style.Bg).mix(waterDeep, 0.5+0.4*depth)
			r := s.Content
			switch r {
			case '▀':
				r = '▄'
			case '▄':
				r = '▀'
			case '▗':
				r = '▝'
			case '▁':
				r = '▔'
			}
			set(x, y, r, fg, bg)
		}
	}
	// Ripples break up the reflection on fixed columns.
	for y := c.horizon; y < h; y++ {
		for x := (y * 7) % 11; x < w; x += 11 {
			bg := toRGB(b.Content[y*w+x].Style.Bg)
			set(x, y, '─', bg.mix(rgb{120, 140, 200}, 0.25), bg)
		}
	}
	for _, g := range c.glints {
		x, y := g%w, c.horizon+g/w
		bg := toRGB(b.Content[y*w+x].Style.Bg)
		set(x, y, '━', bg.mix(moonLight, 0.55), bg)
	}
}

// ── the show ─────────────────────────────────────────────────────────────

type show struct {
	film bool
	opts buffer.DiffOptions
	rng  *rand.Rand

	city *city

	// scene is drawn into cur; prev holds what the diff last "sent";
	// fullBack is resized every frame so the encoder repaints in full.
	cur, prev, fullBack *buffer.Buffer
	out                 []byte
	heat                []float32

	frameNo int // frames the city has advanced
	tick    int // frames shown, frozen or not

	frozen bool
	xray   bool
	sweep  float64 // 0 = normal, 1 = full X-ray

	// Measurements for the scene layer of the latest frame.
	changed, total   int
	sent, full       int
	history          [48]int // bytes sent, most recent last
	secSent, secFull [fps]int
}

func newShow(film bool, opts buffer.DiffOptions) *show {
	return &show{
		film:     film,
		opts:     opts,
		cur:      buffer.NewEmptyBuffer(),
		prev:     buffer.NewEmptyBuffer(),
		fullBack: buffer.NewEmptyBuffer(),
		out:      make([]byte, 0, 1<<16),
	}
}

// filmScript drives -film: a 20-second loop.
func (s *show) filmScript() {
	t := float64(s.tick%(20*fps)) / fps
	s.xray = t >= 6 && t < 15
	s.frozen = t >= 11.5 && t < 13
}

func (s *show) frame(f *limoni.Frame, ev *limoni.Event) bool {
	if ev != nil && ev.Type == limoni.EventKey {
		switch {
		case ev.Key.Type == limoni.KeyEsc, ev.Key.Type == limoni.KeyRune && ev.Key.Ch == 'q':
			return false
		case ev.Key.Type == limoni.KeyRune && (ev.Key.Ch == 'x' || ev.Key.Ch == 'X'):
			s.xray = !s.xray
		case ev.Key.Type == limoni.KeySpace:
			s.frozen = !s.frozen
		}
	}

	area := f.Area()
	w, h := int(area.Width), int(area.Height)
	if w < 20 || h < 10 {
		f.Buffer.SetString(0, 0, "make the window larger", cell.Style{})
		return true
	}

	// Only timer ticks move the world; a key press redraws the same instant.
	if ev == nil {
		if s.film {
			s.filmScript()
		}
		s.tick++
	}
	advance := ev == nil && !s.frozen

	if s.city == nil || s.city.w != w || s.city.h != h {
		s.rng = rand.New(rand.NewSource(7))
		s.city = newCity(w, h, s.rng)
		s.heat = make([]float32, w*h)
		advance = true
	}
	if advance {
		s.city.step(s.frameNo, s.rng)
		s.frameNo++
	}

	s.measure(w, h)

	target := 0.0
	if s.xray {
		target = 1
	}
	if ev == nil {
		s.sweep += (target - s.sweep) * 0.12
		if math.Abs(target-s.sweep) < 0.002 {
			s.sweep = target
		}
	}

	s.composite(f.Buffer, w, h)
	s.drawPanel(f.Buffer, w, h)
	return true
}

// measure draws the city into its own buffer and encodes it twice: as a
// diff against the previous frame and as a full repaint.
func (s *show) measure(w, h int) {
	rect := cell.NewRect(0, 0, uint16(w), uint16(h))
	if s.cur.Area != rect {
		s.cur.Resize(rect)
	}
	s.cur.Clear()
	s.city.draw(s.cur, s.frameNo)
	s.cur.Invalidate()

	s.changed, s.total = 0, w*h
	if s.prev.Area == rect {
		for i := range s.cur.Content {
			if s.cur.Content[i] != s.prev.Content[i] {
				s.changed++
				s.heat[i] = 1
			} else {
				s.heat[i] *= 0.86
			}
		}
	} else {
		s.changed = s.total
		for i := range s.heat {
			s.heat[i] = 1
		}
	}

	var err error
	s.out, err = buffer.DiffWithOptions(s.cur, s.prev, s.out[:0], s.opts)
	if err == nil {
		s.sent = len(s.out)
	}
	// A back buffer of a different size makes the encoder repaint the whole
	// screen, with the same options as the diff above.
	s.fullBack.Resize(cell.Rect{})
	s.cur.IsDirty = true
	s.out, err = buffer.DiffWithOptions(s.cur, s.fullBack, s.out[:0], s.opts)
	if err == nil {
		s.full = len(s.out)
	}

	copy(s.history[:], s.history[1:])
	s.history[len(s.history)-1] = s.sent
	s.secSent[s.tick%fps] = s.sent
	s.secFull[s.tick%fps] = s.full
}

// composite copies the city into the frame, turning the part the X-ray has
// swept over into a heat map of the cells that changed.
func (s *show) composite(dst *buffer.Buffer, w, h int) {
	// The X-ray front sweeps left to right, slightly slanted.
	edge := s.sweep * float64(w+h+8)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			c := s.cur.Content[i]
			pos := float64(x) + float64(h-y)*0.5
			if pos > edge {
				dst.Content[i] = c
				continue
			}
			heat := float64(s.heat[i])
			// Unchanged cells become a faint blueprint of the scene.
			// Foreground and background share one curve, or the two halves of
			// a half-block cell come apart and the moon turns to stripes.
			fg := blueprint(c.Style.Fg)
			bg := blueprint(c.Style.Bg)
			if heat > 0.03 {
				fg = xrayCold.mix(xrayHot, heat)
				bg = xrayBg.mix(xrayGlow, heat*0.85)
			}
			r := c.Content
			if r == ' ' && heat > 0.03 {
				r = '░'
			}
			// The beam at the sweep front.
			if s.sweep < 0.999 && edge-pos < 1.5 {
				fg, bg = accent, accent.scale(0.5)
			}
			dst.Content[i] = cell.Cell{Content: r, Style: cell.Style{Fg: fg.color(), Bg: bg.color()}}
		}
	}
	dst.Invalidate()
}

func (s *show) drawPanel(b *buffer.Buffer, w, h int) {
	text := func(x, y int, str string, fg, bg rgb, bold bool) {
		st := cell.Style{Fg: fg.color(), Bg: bg.color()}
		if bold {
			st = st.Bold()
		}
		b.SetString(uint16(x), uint16(y), str, st)
	}

	if s.sweep < 0.5 {
		if !s.film {
			hint := " x  X-ray the renderer   space  freeze   q  quit "
			if len(hint) < w {
				text(w-len(hint)-1, h-1, hint, rgb{150, 160, 190}, rgb{10, 14, 26}, false)
			}
		}
		return
	}

	// Headline.
	head := "ONLY WHAT CHANGED IS SENT"
	if s.frozen {
		head = "NOTHING MOVES · NOTHING IS SENT"
	}
	sub := "Limoni diffs each frame against the last and writes the difference"
	if len(sub)+4 > w {
		sub = "each frame is diffed against the last"
	}
	text((w-len(head))/2, 1, head, accent.mix(rgb{255, 255, 255}, 0.4), xrayBg, true)
	text((w-len(sub))/2, 2, sub, rgb{120, 170, 180}, xrayBg, false)

	// Stats panel, bottom left.
	var sumSent, sumFull int
	for i := range s.secSent {
		sumSent += s.secSent[i]
		sumFull += s.secFull[i]
	}
	saved := 0.0
	if sumFull > 0 {
		saved = 100 * (1 - float64(sumSent)/float64(sumFull))
	}
	lines := []string{
		fmt.Sprintf("cells changed   %6d / %d", s.changed, s.total),
		fmt.Sprintf("sent this frame %8s", bytesStr(s.sent)),
		fmt.Sprintf("full repaint    %8s", bytesStr(s.full)),
		fmt.Sprintf("last second     %8s  vs %s", bytesStr(sumSent), bytesStr(sumFull)),
		fmt.Sprintf("saved           %7.1f%%", saved),
	}
	pw := 40
	ph := len(lines) + 4
	px, py := 2, h-ph-1
	if px+pw > w || py < 4 {
		return
	}
	panelBg := rgb{4, 12, 18}
	border := accent.scale(0.8)
	for y := py; y < py+ph; y++ {
		text(px, y, strings.Repeat(" ", pw), border, panelBg, false)
	}
	text(px, py, "╭"+strings.Repeat("─", pw-2)+"╮", border, panelBg, false)
	text(px, py+ph-1, "╰"+strings.Repeat("─", pw-2)+"╯", border, panelBg, false)
	for y := py + 1; y < py+ph-1; y++ {
		text(px, y, "│", border, panelBg, false)
		text(px+pw-1, y, "│", border, panelBg, false)
	}
	text(px+2, py, " DIFF X-RAY · city layer ", accent, panelBg, true)
	for i, l := range lines {
		fg := rgb{190, 220, 225}
		if i == len(lines)-1 {
			fg = accent.mix(rgb{255, 255, 255}, 0.3)
		}
		text(px+2, py+1+i, l, fg, panelBg, i == len(lines)-1)
	}

	// Sparkline of bytes sent per frame, scaled to the full repaint.
	bars := []rune("▁▂▃▄▅▆▇█")
	scale := float64(max(1, s.full)) / 12 // a full repaint would be off the chart
	y := py + ph - 2
	for i, v := range s.history[len(s.history)-(pw-4):] {
		k := int(float64(v) / scale * float64(len(bars)-1))
		k = min(max(k, 0), len(bars)-1)
		r := bars[k]
		if v == 0 {
			r = ' '
		}
		b.SetString(uint16(px+2+i), uint16(y), string(r), cell.Style{Fg: accent.color(), Bg: panelBg.color()})
	}
}

// blueprint maps a colour to the X-ray's cold palette by brightness.
func blueprint(c cell.Color) rgb {
	v := toRGB(c)
	l := (0.2126*v.r + 0.7152*v.g + 0.0722*v.b) / 255
	return xrayBg.mix(xrayCold.scale(2), l)
}

func bytesStr(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
