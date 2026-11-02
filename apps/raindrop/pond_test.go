package main

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// frames steps, draws and diffs a pond at 120×40, a frame per call.
func frames() (*pond, func() int) {
	r := cell.Rect{Width: 120, Height: 40}
	cur, prev, out := buffer.NewBuffer(r), buffer.NewBuffer(r), make([]byte, 0, 1<<17)
	p := newPond(1)
	ns := int64(0)
	return p, func() int {
		ns += 16_666_667
		p.stir(ns, 0, 0)
		p.step()
		p.render(cur)
		out, _ = buffer.DiffWithOptions(cur, prev, out[:0], buffer.DiffOptions{TrueColor: true, EraseChar: true, RepeatChar: true})
		return len(out)
	}
}

func TestFramesAllocateNothing(t *testing.T) {
	p, step := frames()
	frame := func() { step() }
	for range 200 {
		frame()
	}
	if p.drops == 0 {
		t.Fatal("no rain in 200 frames")
	}
	if avg := testing.AllocsPerRun(200, frame); avg != 0 {
		t.Errorf("%.2f allocations per frame, want 0", avg)
	}
}

func TestStirChangesThePool(t *testing.T) {
	a, b := newPond(1), newPond(1)
	if a.pool != b.pool {
		t.Fatal("the same seed gave different pools")
	}
	b.stir(42, 0, 0)
	if a.pool == b.pool {
		t.Fatal("stirring left the pool as it was")
	}
}

func TestADropMakesARipple(t *testing.T) {
	p := newPond(1)
	p.resize(40, 40)
	p.drop(20, 20, 6)
	for range 15 {
		p.step() // a drop of the pool's own may land too; the check is near (20, 20)
	}
	if v := p.cur[20*40+25]; v == 0 {
		t.Error("the wave did not spread 5 pixels in 15 steps")
	}
	if p.cur[0] != 0 {
		t.Error("the border moved")
	}
}

// BenchmarkFrame reports what a frame of steady rain costs on the wire.
func BenchmarkFrame(b *testing.B) {
	_, frame := frames()
	for range 300 {
		frame()
	}
	b.ReportAllocs()
	sent := 0
	for b.Loop() {
		sent += frame()
	}
	b.ReportMetric(float64(sent)/float64(b.N), "bytes/frame")
}

func TestThemesRunFromTheirFirstStopToTheirLast(t *testing.T) {
	for i, th := range themes {
		for _, end := range []struct {
			c    cell.Color
			stop uint32
		}{{palettes[i][0], th.stops[0]}, {palettes[i][39], th.stops[3]}} {
			r, g, b := end.c.RGB()
			if got := uint32(r)<<16 | uint32(g)<<8 | uint32(b); got != end.stop {
				t.Errorf("%s: %06x, want %06x", th.name, got, end.stop)
			}
		}
		if themeNamed(th.name) != i {
			t.Errorf("themeNamed(%q) = %d, want %d", th.name, themeNamed(th.name), i)
		}
	}
}
