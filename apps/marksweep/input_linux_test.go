//go:build linux

package main

import (
	"encoding/binary"
	"os"
	"slices"
	"strings"
	"testing"
)

func inputEvent(typ, code uint16, value int32) []byte {
	b := make([]byte, eventSize)
	e := b[eventSize-8:]
	binary.NativeEndian.PutUint16(e, typ)
	binary.NativeEndian.PutUint16(e[2:], code)
	binary.NativeEndian.PutUint32(e[4:], uint32(value))
	return b
}

// Grabbed, the mouse's raw movement turns the view and its buttons fire and
// collect; its vertical movement and a held button's repeats do nothing.
func TestTheLockedMouseLooksAndFires(t *testing.T) {
	p := &pointer{buttons: make(chan pointerEvent, 64)}
	p.grabbed.Store(true)
	g := playing()
	g.ammo = 3
	p.parse(slices.Concat(
		inputEvent(evRel, relX, 10), inputEvent(evRel, relX, -3), inputEvent(evRel, 1, 50),
		inputEvent(evKey, btnLeft, 1), inputEvent(evKey, btnLeft, 2)))
	p.drain(g)
	if !g.locked || g.lookDX != 7*rawLook || !g.held[actFire] {
		t.Fatalf("locked %v, look %.2f, firing %v; want true, %.2f, true", g.locked, g.lookDX, g.held[actFire], 7*rawLook)
	}
	p.parse(slices.Concat(inputEvent(evKey, btnLeft, 0), inputEvent(evKey, btnRight, 1)))
	p.drain(g)
	if g.held[actFire] || g.reloadT <= 0 {
		t.Fatal("the button let go still fires, or the right button did not collect")
	}

	// Not grabbed, the compositor has the mouse and the terminal reports it.
	p.grabbed.Store(false)
	g.lookDX = 0
	p.parse(inputEvent(evRel, relX, 40))
	p.drain(g)
	if g.locked || g.lookDX != 0 {
		t.Fatal("an ungrabbed mouse moved the view")
	}
}

// A device that cannot be grabbed is not counted as locked: the game goes on
// with the terminal's mouse.
func TestAFailedGrabIsNoLock(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	p := &pointer{buttons: make(chan pointerEvent, 1), mice: []*os.File{r}}
	p.want(true)
	if p.grabbed.Load() {
		t.Fatal("a pipe was grabbed")
	}
}

func TestMiceAreFoundByTheirMouseHandler(t *testing.T) {
	devices := `I: Bus=0003 Vendor=1ea7 Product=0064
N: Name="Rawmtech USB Receiver 4K"
H: Handlers=event5 mouse0
B: EV=17

N: Name="AT Translated Set 2 keyboard"
H: Handlers=sysrq kbd event3 leds

N: Name="Touchpad"
H: Handlers=mouse1 event11
`
	got := micePathsIn(strings.NewReader(devices))
	if want := []string{"/dev/input/event5", "/dev/input/event11"}; !slices.Equal(got, want) {
		t.Fatalf("mice %q, want %q", got, want)
	}
}
