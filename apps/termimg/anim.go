//go:build unix

package main

import (
	"image"
	"time"
)

// anim is a picture's frames and where its playback is: kitty's animations
// and iTerm2's GIFs. A still picture is an animation of one frame that never
// runs; kitty may add frames to it later.
type anim struct {
	frames []image.Image
	gaps   []time.Duration // how long each frame shows; <= 0 skips it
	cur    int
	run    bool
	loops  int       // loops left after this one; < 0 forever
	due    time.Time // when cur gives way
}

func still(img image.Image) *anim {
	return &anim{frames: []image.Image{img}, gaps: []time.Duration{defaultGap}, loops: -1}
}

// defaultGap is kitty's gap for a frame that names none.
const defaultGap = 40 * time.Millisecond

// start runs the animation from its current frame.
func (a *anim) start(now time.Time) {
	if !a.run {
		a.run, a.due = true, now.Add(a.gaps[a.cur])
	}
}

// step moves on to the frame due at now.
func (a *anim) step(now time.Time) {
	for a.run && !now.Before(a.due) {
		n := a.next()
		if n < 0 {
			a.run = false
			return
		}
		a.cur = n
		a.due = a.due.Add(a.gaps[n])
	}
}

// next is the frame after cur that shows, -1 at the end of the last loop.
func (a *anim) next() int {
	n := a.cur
	for range a.frames {
		if n++; n == len(a.frames) {
			if a.loops == 0 {
				return -1
			}
			if a.loops > 0 {
				a.loops--
			}
			n = 0
		}
		if a.gaps[n] > 0 {
			return n
		}
	}
	return -1 // every frame is skipped
}

// animate shows every picture at its animation's current frame, and sets
// the timer for the next change.
func (s *session) animate() {
	now := time.Now()
	var next time.Time
	due := func(a *anim) {
		a.step(now)
		if a.run && (next.IsZero() || a.due.Before(next)) {
			next = a.due
		}
	}
	for _, pic := range s.pics.byID {
		if pic.anim == nil {
			continue
		}
		due(pic.anim)
		if f := pic.anim.frames[pic.anim.cur]; f != pic.src {
			pic.src, pic.img, pic.dx, pic.dy = f, pic.render(f), 0, 0
			s.dirty = true
		}
	}
	for _, v := range s.kitty.virtual {
		due(v.anim)
		if v.anim.frames[v.anim.cur] != v.src {
			s.dirty = true
		}
	}
	switch {
	case next.IsZero():
		if s.timer != nil {
			s.timer.Stop()
		}
	case s.timer == nil:
		s.timer = time.AfterFunc(next.Sub(now), s.tick)
	default:
		s.timer.Reset(next.Sub(now))
	}
}

func (s *session) tick() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = false
	s.animate()
	if s.dirty {
		s.buf = s.buf[:0]
		s.show()
		s.flush()
	}
}
