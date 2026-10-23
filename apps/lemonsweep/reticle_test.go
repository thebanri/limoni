package main

import (
	"testing"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type memStore struct{ saved []byte }

func (m *memStore) load() records  { return decodeRecords(m.saved) }
func (m *memStore) save(r records) { m.saved = encodeRecords(r) }

// The settings open from the title, change the crosshair with the keys,
// go back with Esc, and the choice is saved and read back next time.
func TestTheCrosshairIsChosenAndKept(t *testing.T) {
	st := &memStore{}
	g := newGame(1)
	g.store = st
	key := func(k limoni.KeyEvent) { g.key(k) }
	for range 3 { // past the two modes and RECIPES
		key(limoni.KeyEvent{Type: limoni.KeyDown})
	}
	key(limoni.KeyEvent{Type: limoni.KeyEnter})
	if g.phase != phSettings {
		t.Fatalf("SETTINGS opened phase %v", g.phase)
	}
	key(limoni.KeyEvent{Type: limoni.KeyDown}) // CROSSHAIR
	key(limoni.KeyEvent{Type: limoni.KeyRight})
	key(limoni.KeyEvent{Type: limoni.KeyRight})
	key(limoni.KeyEvent{Type: limoni.KeyDown}) // COLOR
	key(limoni.KeyEvent{Type: limoni.KeyLeft})
	key(limoni.KeyEvent{Type: limoni.KeyEsc})
	want := reticle{style: 2, color: len(reticleColors) - 1, size: 1, outline: true}
	if g.phase != phTitle || g.ret != want {
		t.Fatalf("back on %v with %+v, want the title with %+v", g.phase, g.ret, want)
	}
	if got := st.load(); !got.hasRet || got.ret != want {
		t.Fatalf("saved %+v (%v)", got.ret, got.hasRet)
	}
	if got := decodeRecords([]byte(`{"crosshair":{"style":99}}`)); got.hasRet {
		t.Fatal("a crosshair style that does not exist was read")
	}
}

// Every style draws a crosshair of its own round the middle, and the
// outline darkens what is round it.
func TestEveryCrosshairDrawsItsOwn(t *testing.T) {
	seen := map[string]string{}
	for s := range reticleStyles {
		for _, outline := range []bool{false, true} {
			b := buffer.NewBuffer(cell.Rect{Width: 40, Height: 20})
			c := canvas{b, 40, 20}
			c.fill(0, 0, 40, 20, '▀', style(hex(0x808080), hex(0x808080)))
			v := view{c: c, w: 40, h: 20}
			drawReticle(&v, 20, 21, reticle{style: s, size: 1, outline: outline}, 0, hex(0xff0000))
			pic := ""
			red, dark := 0, 0
			for i, cl := range b.Content {
				fg, bg := fromColor(cl.Style.Fg), fromColor(cl.Style.Bg)
				for _, px := range []rgb{fg, bg} {
					switch {
					case px.r > 0.9 && px.g < 0.1:
						red++
					case px.r < 0.4:
						dark++
					}
				}
				if fg.r > 0.9 && fg.g < 0.1 || cl.Content != '▀' {
					pic += "t" + string(rune('0'+i%40)) + string(rune('0'+i/40))
				}
				if bg.r > 0.9 && bg.g < 0.1 {
					pic += "b" + string(rune('0'+i%40)) + string(rune('0'+i/40))
				}
			}
			name := reticleStyles[s]
			if red == 0 {
				t.Errorf("%s draws nothing", name)
			}
			if outline && dark == 0 && name != "CLASSIC" {
				t.Errorf("%s has no outline", name)
			}
			if other, dup := seen[pic]; dup && !outline {
				t.Errorf("%s draws the same as %s", name, other)
			}
			seen[pic] = name
		}
	}
}

// On the title the buttons follow the mouse, and a click on SETTINGS
// opens the settings, not the run.
func TestAClickPicksTheButtonUnderIt(t *testing.T) {
	g := newGame(1)
	b := buffer.NewBuffer(cell.Rect{Width: 140, Height: 44})
	g.render(b)
	g.mouse(limoni.MouseEvent{X: 70, Y: uint16(g.menuY + 3), Button: limoni.MouseLeft})
	if g.phase != phSettings || g.setFrom != phTitle {
		t.Fatalf("a click on SETTINGS gave phase %v", g.phase)
	}
	g.render(b)
	g.mouse(limoni.MouseEvent{X: 70, Y: uint16(g.setY + len(settingRows) - 1), Button: limoni.MouseLeft})
	if g.phase != phTitle {
		t.Fatalf("a click on BACK gave phase %v", g.phase)
	}
}
