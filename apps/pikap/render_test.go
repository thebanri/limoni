package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// fakeSource is a player whose state the test sets.
type fakeSource struct {
	np      nowPlaying
	list    []entry
	cover   *art
	seeks   []float64
	toggles int
	nexts   int
	prevs   int
	grabbed bool
	scrubs  float64
	chosen  int
}

func (f *fakeSource) update(np *nowPlaying) { *np = f.np }
func (f *fakeSource) toggle()               { f.toggles++; f.np.playing = !f.np.playing }
func (f *fakeSource) next()                 { f.nexts++ }
func (f *fakeSource) prev()                 { f.prevs++ }
func (f *fakeSource) seek(s float64)        { f.seeks = append(f.seeks, s); f.np.pos = s }
func (f *fakeSource) setVolume(v float64)   { f.np.volume = v }
func (f *fakeSource) toggleShuffle()        { f.np.shuffle = !f.np.shuffle }
func (f *fakeSource) cycleRepeat()          { f.np.repeat = (f.np.repeat + 1) % 3 }
func (f *fakeSource) grab()                 { f.grabbed = true }
func (f *fakeSource) scrub(rev float64)     { f.scrubs += rev }
func (f *fakeSource) release()              { f.grabbed = false }
func (f *fakeSource) entries() []entry      { return f.list }
func (f *fakeSource) choose(i int)          { f.chosen = i }
func (f *fakeSource) listTitle() string     { return "Players" }
func (f *fakeSource) close()                {}
func (f *fakeSource) artLoader(*nowPlaying) func() *art {
	return func() *art { return f.cover }
}

func newFake() *fakeSource {
	return &fakeSource{
		np: nowPlaying{
			title: "Waters", artist: "Feldup", album: "Discover Weekly", player: "Spotify",
			artKey: "k", length: 177, pos: 61, playing: true, canSeek: true, volume: 0.6,
		},
		list: []entry{
			{num: 1, label: "Spotify", detail: "Feldup — Waters", dur: 177, current: true, playing: true},
			{num: 2, label: "Firefox", detail: "Some video"},
		},
		cover:  generatedArt("Feldup\x00Discover Weekly"),
		chosen: -1,
	}
}

// clock is the tests' time, which only goes forwards.
var clock = time.Unix(1000, 0)

// settle runs the interface for a second of frames, so the cover loads and
// the arm and the colours arrive where they are going.
func settle(u *ui, b *buffer.Buffer) {
	for i := 0; i < 90; i++ {
		clock = clock.Add(33 * time.Millisecond)
		u.tick(clock)
		u.render(b)
		time.Sleep(time.Millisecond)
	}
}

// TestSnapshotPNG draws frames to PNG files for a person to look at:
//
//	PIKAP_PNG=/tmp/pikap go test -run TestSnapshotPNG
func TestSnapshotPNG(t *testing.T) {
	dir := os.Getenv("PIKAP_PNG")
	if dir == "" {
		t.Skip("set PIKAP_PNG to a directory to write snapshots")
	}
	for _, c := range []struct {
		name  string
		w, h  int
		disc  discStyle
		glyph glyphStyle
		full  bool
	}{
		{"ascii-picture", 140, 46, pictureDisc, asciiGlyphs, false},
		{"ascii-vinyl", 140, 46, blackVinyl, asciiGlyphs, false},
		{"blocks-picture", 140, 46, pictureDisc, halfBlocks, false},
		{"small", 80, 24, pictureDisc, asciiGlyphs, false},
		{"full", 140, 46, pictureDisc, halfBlocks, true},
	} {
		f := newFake()
		u := newUI(f)
		u.disc, u.glyph, u.full = c.disc, c.glyph, c.full
		b := buffer.NewBuffer(cell.Rect{Width: uint16(c.w), Height: uint16(c.h)})
		settle(u, b)
		if err := writePNG(b, dir+"/"+c.name+".png"); err != nil {
			t.Fatal(err)
		}
	}
}

func writePNG(b *buffer.Buffer, path string) error {
	const cw, ch = 7, 13
	W, H := int(b.Area.Width), int(b.Area.Height)
	img := image.NewRGBA(image.Rect(0, 0, W*cw, H*ch))
	rgba := func(c cell.Color) color.RGBA {
		r, g, bl := c.RGB()
		return color.RGBA{r, g, bl, 255}
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			c := b.Content[y*W+x]
			bg, fg := rgba(c.Style.Bg), rgba(c.Style.Fg)
			for py := 0; py < ch; py++ {
				for px := 0; px < cw; px++ {
					col := bg
					if c.Content == '▀' && py < ch/2 {
						col = fg
					}
					img.SetRGBA(x*cw+px, y*ch+py, col)
				}
			}
			if c.Content != ' ' && c.Content != '▀' && c.Content != cell.RuneContinuation {
				d := font.Drawer{Dst: img, Src: image.NewUniform(fg), Face: basicfont.Face7x13,
					Dot: fixed.P(x*cw, y*ch+11)}
				d.DrawString(string(c.Content))
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func BenchmarkFrame(b *testing.B) {
	for _, g := range []struct {
		name string
		gs   glyphStyle
	}{{"ascii", asciiGlyphs}, {"blocks", halfBlocks}} {
		b.Run(g.name, func(b *testing.B) {
			f := newFake()
			u := newUI(f)
			u.glyph = g.gs
			buf := buffer.NewBuffer(cell.Rect{Width: 160, Height: 50})
			settle(u, buf)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				clock = clock.Add(33 * time.Millisecond)
				u.tick(clock)
				u.render(buf)
			}
		})
	}
}

// flatArt is a cover of one colour.
func flatArt(c rgb) *art {
	a := &art{px: make([]rgb, artSize*artSize), tint: c}
	for i := range a.px {
		a.px[i] = c
	}
	return a
}

func TestArmStandsOutFromAnyCover(t *testing.T) {
	for _, c := range []struct {
		name  string
		cover rgb
	}{
		{"white", lin(0xffffff)}, {"light grey", lin(0xd8d8d8)}, {"mid grey", lin(0x8a8a8a)},
		{"black", lin(0x050505)}, {"lemon", lin(0xf4d21f)},
	} {
		var s scene
		s.prepare(flatArt(c.cover), 30, pictureDisc)
		// The needle halfway through the song.
		arm := armAngle(needleAt((grooveOut + grooveIn(pictureDisc)) / 2))
		s.begin(sceneParams{arm: arm, style: pictureDisc, tube: 0.024}, 30)
		// A point on the tube over the record, and one beside it.
		dx, dy := math.Cos(arm), math.Sin(arm)
		f := armLen * 0.8
		u, v := pivotU+dx*f, pivotV+dy*f
		if math.Hypot(u, v) > 0.9 {
			t.Fatalf("the test point %.2f,%.2f is not over the record", u, v)
		}
		bg := lin(0x101014)
		on, _ := s.at(u, v, bg)
		off, _ := s.at(u-dy*0.2, v+dx*0.2, bg)
		if d := math.Abs(perceived(on) - perceived(off)); d < 0.35 {
			t.Errorf("over a %s cover the arm is %.2f light and the record beside it %.2f: too alike",
				c.name, perceived(on), perceived(off))
		}
	}
}
