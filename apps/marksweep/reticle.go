package main

import (
	"math"

	"github.com/thebanri/limoni"
)

// The crosshair, as the settings choose it: a style, a colour, a size, and
// a dark outline or none. Every style but the classic is drawn in pixels;
// the classic is the old one, in characters.

type reticle struct {
	style, color, size int
	outline            bool
}

var reticleStyles = [...]string{"CROSS", "CROSS + DOT", "DOT", "CIRCLE", "CHEVRON", "CLASSIC"}

var reticleColors = [...]struct {
	name string
	col  uint32
}{{"WHITE", 0xf4f6ff}, {"GREEN", 0x6ee07a}, {"CYAN", 0x5ad1ff}, {"YELLOW", 0xffe14a}, {"PINK", 0xff6ad5}, {"RED", 0xff4a5a}}

var reticleSizes = [...]string{"SMALL", "MEDIUM", "LARGE"}

// reticlePx collects a crosshair's pixels before they are drawn, so the
// outline can go under all of them first.
var reticlePx [96][2]int

// drawReticle draws crosshair r centred on pixel (cx, cy), opened by
// spread pixels, in col.
func drawReticle(v *view, cx, cy int, r reticle, spread int, col rgb) {
	if r.style == len(reticleStyles)-1 {
		classicReticle(v, cx, cy/2, spread, col)
		return
	}
	arm := [...]int{2, 3, 5}[r.size]
	gap := [...]int{1, 2, 3}[r.size] + spread
	n := 0
	add := func(x, y int) {
		if n < len(reticlePx) {
			reticlePx[n] = [2]int{x, y}
			n++
		}
	}
	// A dot: a pixel, a square of four, or a rounded square of nine
	// without its corners.
	dot := func() {
		add(cx, cy)
		if r.size >= 1 {
			add(cx+1, cy)
			add(cx, cy+1)
			add(cx+1, cy+1)
		}
		if r.size == 2 {
			add(cx-1, cy)
			add(cx-1, cy+1)
			add(cx, cy-1)
			add(cx+1, cy-1)
			add(cx, cy+2)
			add(cx+1, cy+2)
		}
	}
	switch reticleStyles[r.style] {
	case "CROSS", "CROSS + DOT":
		for k := 0; k < arm; k++ {
			d := gap + k
			add(cx-d, cy)
			add(cx+d, cy)
			add(cx, cy-d)
			add(cx, cy+d)
		}
		if r.style == 1 {
			dot()
		}
	case "DOT":
		dot()
	case "CIRCLE":
		rad := float64(gap + arm - 1)
		steps := int(rad * 7)
		for i := 0; i < steps; i++ {
			a := float64(i) / float64(steps) * 2 * math.Pi
			add(cx+int(math.Round(math.Cos(a)*rad)), cy+int(math.Round(math.Sin(a)*rad)))
		}
		add(cx, cy)
	case "CHEVRON":
		// A V pointing up, its tip just under the point aimed at.
		for k := 0; k <= arm; k++ {
			add(cx-k, cy+gap+k)
			add(cx+k, cy+gap+k)
		}
		add(cx, cy)
	}
	if r.outline {
		ink := hex(0x05060a)
		for _, p := range reticlePx[:n] {
			for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				if !litAt(n, p[0]+d[0], p[1]+d[1]) {
					blendPx(v, p[0]+d[0], p[1]+d[1], ink, 0.75)
				}
			}
		}
	}
	for _, p := range reticlePx[:n] {
		blendPx(v, p[0], p[1], col, 1)
	}
}

func litAt(n, x, y int) bool {
	for _, p := range reticlePx[:n] {
		if p[0] == x && p[1] == y {
			return true
		}
	}
	return false
}

// classicReticle is the first crosshair: a dot between two dashes, in
// characters, the dashes moving out with the kick.
func classicReticle(v *view, x, y, spread int, col rgb) {
	gap := 1 + spread/2
	v.c.fg(x, y, "·", col)
	v.c.fg(x-gap-1, y, "─", col.mul(0.85))
	v.c.fg(x+gap+1, y, "─", col.mul(0.85))
}

// reticleColor is the crosshair's colour now: its own, or a warning one
// over an enemy (white, if its own is the red).
func (g *game) reticleColor() rgb {
	own := hex(reticleColors[g.ret.color].col)
	switch {
	case g.ammo == 0:
		return hudBad.mul(0.8)
	case g.aimOn && reticleColors[g.ret.color].name == "RED":
		return hex(0xffffff)
	case g.aimOn:
		return hex(0xff4a5a)
	}
	return own
}

// ── settings ─────────────────────────────────────────────────────────────

var settingRows = [...]string{"SOUND", "CROSSHAIR", "COLOR", "SIZE", "OUTLINE", "BACK"}

func (g *game) openSettings() {
	g.setFrom, g.phase, g.setSel = g.phase, phSettings, 0
	g.sound(sfxMenu)
}

func (g *game) closeSettings() {
	g.phase = g.setFrom
	g.sound(sfxMenu)
}

// adjust moves setting row by dir, round its choices, and saves it.
func (g *game) adjust(row, dir int) {
	wrap := func(v, n int) int { return ((v+dir)%n + n) % n }
	switch settingRows[row] {
	case "SOUND":
		g.setVolume(g.volume + dir)
	case "CROSSHAIR":
		g.ret.style = wrap(g.ret.style, len(reticleStyles))
	case "COLOR":
		g.ret.color = wrap(g.ret.color, len(reticleColors))
	case "SIZE":
		g.ret.size = wrap(g.ret.size, len(reticleSizes))
	case "OUTLINE":
		g.ret.outline = !g.ret.outline
	default:
		return
	}
	g.sound(sfxMenu)
	g.persist()
}

func (g *game) settingsKey(k limoni.KeyEvent, isRune bool, ch rune) {
	n := len(settingRows)
	switch {
	case isRune && ch == 'w' || k.Type == limoni.KeyUp:
		g.setSel = (g.setSel + n - 1) % n
		g.sound(sfxMenu)
	case isRune && ch == 's' || k.Type == limoni.KeyDown:
		g.setSel = (g.setSel + 1) % n
		g.sound(sfxMenu)
	case isRune && ch == 'a' || k.Type == limoni.KeyLeft:
		g.adjust(g.setSel, -1)
	case isRune && ch == 'd' || k.Type == limoni.KeyRight:
		g.adjust(g.setSel, 1)
	case k.Type == limoni.KeyEnter || k.Type == limoni.KeySpace:
		if g.setSel == n-1 {
			g.closeSettings()
		} else {
			g.adjust(g.setSel, 1)
		}
	case k.Type == limoni.KeyEsc:
		g.closeSettings()
	}
}
