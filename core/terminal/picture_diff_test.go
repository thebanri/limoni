package terminal

// Tests for placing only the pictures that changed.

import (
	"image"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

// tileFrames draws frames of small pictures, one per two-cell tile, the way
// a tile map does, and returns what each frame sent.
func tileFrames(t *testing.T, term *Terminal, out func() []byte, frames ...[]image.Image) []string {
	t.Helper()
	var sent []string
	for _, tiles := range frames {
		before := len(out())
		err := term.Draw(func(f *Frame) {
			for i, img := range tiles {
				area := cell.NewRect(uint16(2*i), 1, 2, 1)
				for x := area.X; x < area.X+area.Width; x++ {
					f.Buffer.SetCell(x, 1, cell.Cell{Content: cell.RuneImage})
				}
				f.RegisterImageRegion(ImageRegion{Area: area, Img: img})
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		sent = append(sent, string(out()[before:]))
	}
	return sent
}

func TestKittyChangesOnlyTheTilesThatChanged(t *testing.T) {
	term, io := kittyTerminal(t)
	a, b, c := picture(16, 16), picture(16, 16), picture(16, 16)
	c.Pix[0] = 1 // a picture of its own
	sent := tileFrames(t, term, io.Output,
		[]image.Image{a, b, a},
		[]image.Image{a, c, a}, // the middle tile changes
		[]image.Image{a, c},    // the last tile goes
	)

	if n := strings.Count(sent[0], "a=p,"); n != 3 {
		t.Fatalf("first frame made %d placements, want 3", n)
	}
	if strings.Contains(sent[1], "d=a") || strings.Contains(sent[2], "d=a") {
		t.Fatal("a frame deleted every placement")
	}
	if n := strings.Count(sent[1], "a=p,"); n != 1 {
		t.Errorf("changing one tile made %d placements, want 1: %q", n, sent[1])
	}
	if !strings.Contains(sent[1], "a=d,d=i,") {
		t.Errorf("the changed tile's old placement was not deleted: %q", sent[1])
	}
	if n := strings.Count(sent[1], "a=t,"); n != 1 {
		t.Errorf("changing one tile sent %d pictures, want only the new one", n)
	}
	if strings.Contains(sent[2], "a=p,") || !strings.Contains(sent[2], "a=d,d=i,") {
		t.Errorf("removing a tile should only delete its placement: %q", sent[2])
	}
	// b is no longer on screen anywhere, so kitty frees it.
	if !strings.Contains(sent[1], "d=I") {
		t.Errorf("a picture no longer shown was not freed: %q", sent[1])
	}
}

func TestKittyRedrawsEveryTileAfterAFullRedraw(t *testing.T) {
	term, io := kittyTerminal(t)
	a := picture(16, 16)
	tileFrames(t, term, io.Output, []image.Image{a, a})
	term.ForceFullRedraw()
	sent := tileFrames(t, term, io.Output, []image.Image{a, a})
	if n := strings.Count(sent[0], "a=p,"); n != 2 {
		t.Errorf("after a full redraw %d tiles were placed, want 2", n)
	}
}

func TestKittyMovesTilesWithoutSendingThemAgain(t *testing.T) {
	term, io := kittyTerminal(t)
	a := picture(16, 16)
	var sent []string
	for _, off := range []uint16{0, 1} {
		before := len(io.Output())
		err := term.Draw(func(f *Frame) {
			for i := uint16(0); i < 3; i++ {
				area := cell.NewRect(off+2*i, 1, 2, 1)
				for x := area.X; x < area.X+area.Width; x++ {
					f.Buffer.SetCell(x, 1, cell.Cell{Content: cell.RuneImage})
				}
				f.RegisterImageRegion(ImageRegion{Area: area, Img: a})
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		sent = append(sent, string(io.Output()[before:]))
	}

	moved := sent[1]
	if strings.Contains(moved, "a=t,") {
		t.Errorf("moving the tiles sent their picture again: %q", moved)
	}
	if strings.Contains(moved, "d=a") {
		t.Errorf("moving the tiles deleted every placement: %q", moved)
	}
	if n := strings.Count(moved, "a=p,"); n != 3 {
		t.Errorf("moving 3 tiles made %d placements, want 3: %q", n, moved)
	}
	// Each tile moves its own placement in one step, so none is deleted.
	if strings.Contains(moved, "a=d,") {
		t.Errorf("moving the tiles deleted a placement: %q", moved)
	}
	for _, pid := range []string{",p=1,", ",p=2,", ",p=3,"} {
		if !strings.Contains(moved, pid) {
			t.Errorf("placement %s was not moved: %q", pid, moved)
		}
	}
}

func TestKittySwapsTilesWithoutDeletingPlacements(t *testing.T) {
	term, io := kittyTerminal(t)
	a, b := picture(16, 16), picture(16, 16)
	b.Pix[0] = 1 // a picture of its own
	sent := tileFrames(t, term, io.Output,
		[]image.Image{a, b},
		[]image.Image{b, a}, // a map scrolled by one tile
	)
	if n := strings.Count(sent[1], "a=p,"); n != 2 {
		t.Errorf("swapping 2 tiles made %d placements, want 2: %q", n, sent[1])
	}
	if strings.Contains(sent[1], "a=d,") || strings.Contains(sent[1], "a=t,") {
		t.Errorf("swapping tiles should only move their placements: %q", sent[1])
	}
	for _, pid := range []string{",p=1,", ",p=2,"} {
		if !strings.Contains(sent[1], pid) {
			t.Errorf("placement %s was not moved: %q", pid, sent[1])
		}
	}
}

func TestKittyKeepsMovingTilesApart(t *testing.T) {
	term, io := kittyTerminal(t)
	a := picture(16, 16)
	for off := uint16(0); off < 4; off++ {
		before := len(io.Output())
		err := term.Draw(func(f *Frame) {
			for i := uint16(0); i < 3; i++ {
				area := cell.NewRect(off+2*i, 1, 2, 1)
				for x := area.X; x < area.X+area.Width; x++ {
					f.Buffer.SetCell(x, 1, cell.Cell{Content: cell.RuneImage})
				}
				f.RegisterImageRegion(ImageRegion{Area: area, Img: a})
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		sent := string(io.Output()[before:])
		if off > 0 && strings.Contains(sent, "a=d,") {
			t.Errorf("frame %d deleted a placement still in use: %q", off, sent)
		}
		for _, pid := range []string{",p=1,", ",p=2,", ",p=3,"} {
			if off > 0 && strings.Count(sent, pid) != 1 {
				t.Errorf("frame %d placed %s %d times, want once: %q", off, pid, strings.Count(sent, pid), sent)
			}
		}
		if n := len(term.kitty.placed); n != 3 {
			t.Errorf("after frame %d kitty has %d placements on record, want 3", off, n)
		}
	}
}

// A scene drawn over the whole screen clears the pictures placed under it,
// and the terminal takes them down.
func TestClearImageRegionsTakesThePicturesDown(t *testing.T) {
	term, io := kittyTerminal(t)
	img := picture(16, 16)
	draw := func(clear bool) string {
		before := len(io.Output())
		if err := term.Draw(func(f *Frame) {
			f.RegisterImageRegion(ImageRegion{Area: cell.NewRect(0, 1, 2, 1), Img: img})
			if clear {
				f.ClearImageRegions()
			}
		}); err != nil {
			t.Fatal(err)
		}
		return string(io.Output()[before:])
	}
	if got := draw(false); !strings.Contains(got, "a=p,") {
		t.Fatalf("the picture was not placed: %q", got)
	}
	got := draw(true)
	if strings.Contains(got, "a=p,") || !strings.Contains(got, "a=d,") {
		t.Fatalf("cleared, the picture was not taken down: %q", got)
	}
}
