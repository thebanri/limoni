package main

import (
	"math"
	"testing"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func setup(t *testing.T) (*ui, *fakeSource, *buffer.Buffer) {
	t.Helper()
	f := newFake()
	u := newUI(f)
	b := buffer.NewBuffer(cell.Rect{Width: 140, Height: 46})
	settle(u, b)
	if !u.haveDeck {
		t.Fatal("no deck drawn at 140×46")
	}
	return u, f, b
}

// cellAt is the screen cell at record units (uu, vv).
func cellAt(dv deckView, uu, vv float64) (uint16, uint16) {
	return uint16(dv.cx + uu*dv.R), uint16(dv.cy + vv*dv.R/2)
}

func press(u *ui, x, y uint16) {
	u.mouse(limoni.MouseEvent{Button: limoni.MouseLeft, X: x, Y: y})
}
func drag(u *ui, x, y uint16) {
	u.mouse(limoni.MouseEvent{Button: limoni.MouseLeft, X: x, Y: y, Drag: true})
}
func release(u *ui, x, y uint16) {
	u.mouse(limoni.MouseEvent{Button: limoni.MouseRelease, X: x, Y: y})
}

func TestDraggingTheRecordScrubs(t *testing.T) {
	u, f, _ := setup(t)
	// A quarter turn clockwise, along a circle of radius 0.7 from the top
	// to the right (y grows downwards, so clockwise is increasing angle).
	x, y := cellAt(u.dv, 0, -0.7)
	press(u, x, y)
	if !f.grabbed {
		t.Fatal("pressing on the record did not take hold of it")
	}
	for i := 1; i <= 10; i++ {
		a := -math.Pi/2 + float64(i)/10*math.Pi/2
		x, y = cellAt(u.dv, 0.7*math.Cos(a), 0.7*math.Sin(a))
		drag(u, x, y)
	}
	release(u, x, y)
	if f.grabbed {
		t.Fatal("letting go did not let go")
	}
	if math.Abs(f.scrubs-0.25) > 0.04 {
		t.Fatalf("a quarter turn forward scrubbed %.3f turns", f.scrubs)
	}

	// Up the left side is clockwise too, across the angle where atan2
	// wraps from π to -π: a small step forward, not most of a turn back.
	f.scrubs = 0
	x, y = cellAt(u.dv, -0.7, 0.1)
	press(u, x, y)
	x, y = cellAt(u.dv, -0.7, -0.1)
	drag(u, x, y)
	release(u, x, y)
	if f.scrubs <= 0 || f.scrubs > 0.1 {
		t.Fatalf("a small turn forward across the wrap scrubbed %.3f turns", f.scrubs)
	}
	// And the same way down is backwards.
	f.scrubs = 0
	press(u, x, y)
	x, y = cellAt(u.dv, -0.7, 0.1)
	drag(u, x, y)
	release(u, x, y)
	if f.scrubs >= 0 || f.scrubs < -0.1 {
		t.Fatalf("a small turn back across the wrap scrubbed %.3f turns", f.scrubs)
	}
}

func TestDraggingTheArmMovesTheNeedle(t *testing.T) {
	u, f, b := setup(t)
	// Take the arm by its middle and put the needle at the outer groove:
	// the start of the song.
	nu, nv := pivotU+math.Cos(u.armA)*armLen*0.6, pivotV+math.Sin(u.armA)*armLen*0.6
	x, y := cellAt(u.dv, nu, nv)
	press(u, x, y)
	if !u.armDrag {
		t.Fatalf("pressing on the arm at %d,%d did not take it", x, y)
	}
	if f.grabbed {
		t.Fatal("taking the arm took the record too")
	}
	ox, oy := needleAt(grooveOut)
	x, y = cellAt(u.dv, ox, oy)
	drag(u, x, y)
	release(u, x, y)
	if len(f.seeks) != 1 || f.seeks[0] > 0.1*f.np.length {
		t.Fatalf("the needle put down at the outer groove seeked to %v of %.0f s", f.seeks, f.np.length)
	}
	// The arm swings there with the song.
	settle(u, b)
	want := armAngle(needleAt(grooveOut))
	if math.Abs(u.armA-want) > 0.02 {
		t.Fatalf("arm at %.3f, want %.3f", u.armA, want)
	}
}

func TestProgressBarAndButtons(t *testing.T) {
	u, f, _ := setup(t)
	bar := u.bar
	press(u, uint16(bar.x+bar.w-1), uint16(bar.y))
	release(u, uint16(bar.x+bar.w-1), uint16(bar.y))
	if len(f.seeks) != 1 || math.Abs(f.seeks[0]-f.np.length) > 0.01 {
		t.Fatalf("clicking the end of the bar seeked to %v", f.seeks)
	}
	for i, want := range []*int{nil, &f.prevs, &f.toggles, &f.nexts, nil} {
		if want == nil {
			continue
		}
		b := u.buttons[i]
		before := *want
		press(u, uint16(b.x+b.w/2), uint16(b.y))
		release(u, uint16(b.x+b.w/2), uint16(b.y))
		if *want != before+1 {
			t.Fatalf("button %d did nothing", i)
		}
	}
}

func TestKeys(t *testing.T) {
	u, f, b := setup(t)
	pos := f.np.pos
	u.key(limoni.KeyEvent{Type: limoni.KeyRight})
	u.key(limoni.KeyEvent{Type: limoni.KeyLeft, Shift: true})
	// The second press counts from the first, though no frame came between.
	if len(f.seeks) != 2 || f.seeks[0] != pos+5 || f.seeks[1] != pos+5-30 {
		t.Fatalf("→ then shift+← from %.0f seeked %v", pos, f.seeks)
	}
	u.key(limoni.KeyEvent{Type: limoni.KeySpace})
	u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: ']'})
	u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: '['})
	if f.toggles != 1 || f.nexts != 1 || f.prevs != 1 {
		t.Fatalf("space, ], [: %d %d %d", f.toggles, f.nexts, f.prevs)
	}
	// The list: down, enter chooses the second entry.
	u.key(limoni.KeyEvent{Type: limoni.KeyDown})
	u.render(b)
	u.key(limoni.KeyEvent{Type: limoni.KeyEnter})
	if f.chosen != 1 {
		t.Fatalf("↓ Enter chose %d", f.chosen)
	}
	u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'q'})
	if !u.quit {
		t.Fatal("q did not quit")
	}
}

func TestPausedPlatterCoastsToAStop(t *testing.T) {
	u, f, b := setup(t)
	if u.speed < 0.95 {
		t.Fatalf("a playing record turns at %.2f", u.speed)
	}
	f.np.playing = false
	settle(u, b) // three seconds
	if u.speed != 0 {
		t.Fatalf("paused for three seconds, still turning at %.3f", u.speed)
	}
	a := u.angle
	settle(u, b)
	if u.angle != a {
		t.Fatal("a stopped platter turned")
	}
}

func TestSmallAndOddSizes(t *testing.T) {
	f := newFake()
	u := newUI(f)
	for _, sz := range [][2]uint16{{1, 1}, {10, 3}, {40, 12}, {80, 24}, {300, 90}} {
		b := buffer.NewBuffer(cell.Rect{Width: sz[0], Height: sz[1]})
		settle(u, b)
		u.mouse(limoni.MouseEvent{Button: limoni.MouseLeft, X: sz[0] / 2, Y: sz[1] / 2})
		u.mouse(limoni.MouseEvent{Button: limoni.MouseRelease, X: sz[0] / 2, Y: sz[1] / 2})
	}
	// Nothing playing: the status shows instead of a title.
	f.np = nowPlaying{status: "Nothing is playing.", volume: -1}
	b := buffer.NewBuffer(cell.Rect{Width: 100, Height: 30})
	settle(u, b)
	if s := b.Snapshot(); !contains(s, "Nothing is playing.") {
		t.Fatal("the status is not on screen")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestHalfBlocksByDefault(t *testing.T) {
	u, _, b := setup(t)
	if u.glyph != halfBlocks {
		t.Fatal("the default is not half blocks")
	}
	x, y := cellAt(u.dv, 0.5, 0.5)
	if r := b.CellAt(x, y).Content; r != '▀' {
		t.Fatalf("the record is drawn with %q, not half blocks", r)
	}
	u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'v'})
	u.render(b)
	if r := b.CellAt(x, y).Content; r == '▀' {
		t.Fatal("V did not switch to ASCII")
	}
}

func TestRPMIsOnlyALook(t *testing.T) {
	u, f, b := setup(t)
	key := func(ch rune) {
		u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: ch})
		settle(u, b)
	}
	shows := func(want string) {
		t.Helper()
		if !contains(b.Snapshot(), want) {
			t.Fatalf("the screen does not show %q", want)
		}
	}
	// turns is how far the record is shown turning in a second.
	turns := func() float64 {
		a := u.angle
		settle(u, b) // 90 frames of 33 ms
		return (u.angle - a) / 2.97
	}
	before := *f
	shows("33⅓ rpm")
	if got := turns(); math.Abs(got-rev33) > 0.01 {
		t.Fatalf("at 33⅓ the record turns %.3f times a second, want %.3f", got, rev33)
	}
	key('<')
	shows("33.0 rpm")
	key('<')
	shows("32.0 rpm")
	key('1')
	shows("16⅔ rpm")
	if got := turns(); math.Abs(got-rev33/2) > 0.01 {
		t.Fatalf("at 16⅔ the record turns %.3f times a second, want %.3f", got, rev33/2)
	}
	for range 20 {
		u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: '<'})
	}
	settle(u, b)
	shows("8.3 rpm")
	key('>')
	shows("9.0 rpm")
	key('7')
	shows("78 rpm")
	key('4')
	shows("45 rpm")

	// None of it reached the player: no seek, no pause, and the song goes
	// on at its own speed.
	if len(f.seeks) != 0 || f.toggles != before.toggles || f.np.playing != before.np.playing || f.np.pos != before.np.pos {
		t.Fatalf("the rpm keys touched the player: %+v", f)
	}
}

func TestFullscreen(t *testing.T) {
	u, _, b := setup(t)
	R := u.dv.R
	if u.listW == 0 {
		t.Fatal("no list at 140×46")
	}
	u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: 'f'})
	settle(u, b)
	if !u.full || u.listW != 0 {
		t.Fatalf("f: full %v, list %d", u.full, u.listW)
	}
	if u.dv.R <= R {
		t.Fatalf("fullscreen did not make the record larger: radius %.0f → %.0f", R, u.dv.R)
	}
	if contains(b.Snapshot(), "Players") || contains(b.Snapshot(), "space play") {
		t.Fatal("the list or the help is still on screen in fullscreen")
	}
	if !contains(b.Snapshot(), "Waters") {
		t.Fatal("the title is gone in fullscreen")
	}
	// Esc leaves fullscreen and does not quit.
	u.key(limoni.KeyEvent{Type: limoni.KeyEsc})
	settle(u, b)
	if u.full || u.quit {
		t.Fatalf("Esc in fullscreen: full %v, quit %v", u.full, u.quit)
	}
	// The corner button goes in and out too.
	press(u, uint16(u.fullBtn.x+1), uint16(u.fullBtn.y))
	release(u, uint16(u.fullBtn.x+1), uint16(u.fullBtn.y))
	settle(u, b)
	if !u.full {
		t.Fatal("the ⤢ button did not go fullscreen")
	}
	press(u, uint16(u.fullBtn.x+1), uint16(u.fullBtn.y))
	release(u, uint16(u.fullBtn.x+1), uint16(u.fullBtn.y))
	settle(u, b)
	if u.full {
		t.Fatal("the ⤡ button did not leave fullscreen")
	}
}

func TestListEdgeDrags(t *testing.T) {
	u, _, b := setup(t)
	W := int(b.Area.Width)
	edge := u.listX
	press(u, uint16(edge), 10)
	if !u.divDrag {
		t.Fatal("pressing the list's edge did not take it")
	}
	drag(u, uint16(edge-20), 10)
	release(u, uint16(edge-20), 10)
	settle(u, b)
	if u.listW != W-(edge-20) {
		t.Fatalf("dragged the edge 20 columns left: list %d wide, want %d", u.listW, W-(edge-20))
	}
	// It keeps the deck and the list their room.
	press(u, uint16(u.listX), 10)
	drag(u, 1, 10)
	release(u, 1, 10)
	settle(u, b)
	if u.listW != W-minMain {
		t.Fatalf("dragged to the far left: list %d wide, want %d", u.listW, W-minMain)
	}
	press(u, uint16(u.listX), 10)
	drag(u, uint16(W-1), 10)
	release(u, uint16(W-1), 10)
	settle(u, b)
	if u.listW != minList {
		t.Fatalf("dragged to the far right: list %d wide, want %d", u.listW, minList)
	}
	// Tab puts it away and brings it back as it was.
	u.key(limoni.KeyEvent{Type: limoni.KeyTab})
	settle(u, b)
	if u.listW != 0 {
		t.Fatal("Tab did not put the list away")
	}
	u.key(limoni.KeyEvent{Type: limoni.KeyTab})
	settle(u, b)
	if u.listW != minList {
		t.Fatalf("Tab brought the list back %d wide, want %d", u.listW, minList)
	}
	// A narrower window keeps the deck its room.
	small := buffer.NewBuffer(cell.Rect{Width: 70, Height: 30})
	u.listWant = 60
	settle(u, small)
	if u.listW != 70-minMain {
		t.Fatalf("in a 70-column window the list is %d wide", u.listW)
	}
}

func TestHelpCardShowsEveryKey(t *testing.T) {
	u, _, b := setup(t)
	if u.listW == 0 {
		t.Fatal("no list at 140×46")
	}
	for _, full := range []bool{false, true} {
		u.full = full
		u.key(limoni.KeyEvent{Type: limoni.KeyRune, Ch: '?'})
		settle(u, b)
		s := b.Snapshot()
		for _, k := range helpKeys {
			if !contains(s, k[0]) || !contains(s, k[1]) {
				t.Fatalf("full %v: %q %q is not on the card", full, k[0], k[1])
			}
		}
		if n := testing.AllocsPerRun(20, func() { u.render(b) }); n != 0 {
			t.Fatalf("full %v: a frame with the card allocates %.0f times", full, n)
		}
		// Esc closes the card first: it neither quits nor leaves fullscreen.
		u.key(limoni.KeyEvent{Type: limoni.KeyEsc})
		settle(u, b)
		if u.help || u.quit || u.full != full {
			t.Fatalf("Esc over the card: help %v, quit %v, full %v", u.help, u.quit, u.full)
		}
		if contains(b.Snapshot(), helpKeys[len(helpKeys)-2][1]) {
			t.Fatalf("full %v: the card is still drawn after Esc", full)
		}
	}
}
