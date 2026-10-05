package main

import (
	"context"
	"math/rand/v2"
)

// localSource plays files through pikap's own deck: the needle is in the
// groove, so the record can be slowed, stopped, pushed and pulled back, and
// the sound does what the platter does.
type localSource struct {
	d      *deck
	out    *output
	stopAu chan struct{}

	tracks  []*track
	order   []int // play order, shuffled or not
	at      int   // index into order
	rec     *pcm
	shuffle bool
	repeat  int // 0 off, 1 all, 2 one

	tags   chan tagResult
	cancel context.CancelFunc
	list   []entry
	dirty  bool
	errMsg string
	hand   float64
	noOut  string
}

func newLocal(tracks []*track, out *output, noOut string) *localSource {
	s := &localSource{
		d: newDeck(), out: out, stopAu: make(chan struct{}), tracks: tracks,
		tags: make(chan tagResult, 64), dirty: true, noOut: noOut,
	}
	for i := range tracks {
		s.order = append(s.order, i)
	}
	var w interface{ Write([]byte) (int, error) }
	if out != nil {
		w = out.w
	}
	go s.d.run(w, s.stopAu)
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go probeAll(ctx, tracks, 0, s.tags)
	s.play(0, true)
	return s
}

func (s *localSource) close() {
	s.cancel()
	close(s.stopAu)
	s.rec.stop()
	if s.out != nil {
		s.out.close()
	}
}

func (s *localSource) cur() *track { return s.tracks[s.order[s.at]] }

// play puts the record at order index i on the platter.
func (s *localSource) play(i int, motor bool) {
	s.rec.stop()
	s.at = i
	t := s.cur()
	if t.demo {
		s.rec = demoRecord()
	} else {
		s.rec = decodeFile(t.path)
	}
	s.d.load(s.rec)
	s.d.motor.Store(motor)
	s.dirty = true
}

func (s *localSource) update(np *nowPlaying) {
	for {
		select {
		case r := <-s.tags:
			r.apply()
			s.dirty = true
			continue
		default:
		}
		break
	}
	if s.d.ended.Load() {
		s.ended()
	}
	t := s.cur()
	*np = nowPlaying{
		title: t.title, artist: t.artist, album: t.album, player: "pikap",
		artKey: t.artKey, length: t.dur, pos: s.d.position(),
		playing: s.d.motor.Load(), canSeek: true, volume: float64(s.d.getVolume()),
		hasPlatter: true, speed: s.d.platterSpeed(),
		shuffle: s.shuffle, repeat: s.repeat,
	}
	if np.length == 0 && s.rec.done.Load() {
		np.length = float64(s.rec.frames.Load()) / sampleRate
	}
	if err := s.rec.Err(); err != nil && s.rec.frames.Load() == 0 {
		np.status = err.Error()
	}
	np.note = s.noOut
	if s.dirty {
		s.rebuildList()
	}
}

// ended moves on when the needle reaches the run-out groove.
func (s *localSource) ended() {
	switch {
	case s.repeat == 2:
		s.play(s.at, true)
	case s.at+1 < len(s.order):
		s.play(s.at+1, true)
	case s.repeat == 1:
		if s.shuffle {
			s.reshuffle()
		}
		s.play(0, true)
	default:
		s.d.motor.Store(false)
		s.d.ended.Store(false)
		s.d.seek(0)
	}
}

func (s *localSource) toggle() {
	s.d.motor.Store(!s.d.motor.Load())
	s.dirty = true
}

func (s *localSource) next() {
	if s.at+1 < len(s.order) {
		s.play(s.at+1, s.d.motor.Load())
	} else {
		s.play(0, s.d.motor.Load())
	}
}

// prev goes back to the start of the record, or to the one before when the
// needle is near the start already — as every player does.
func (s *localSource) prev() {
	if s.d.position() > 3 || s.at == 0 {
		s.d.seek(0)
		return
	}
	s.play(s.at-1, s.d.motor.Load())
}

func (s *localSource) seek(sec float64) {
	s.d.ended.Store(false)
	s.d.seek(sec)
}

func (s *localSource) setVolume(v float64) { s.d.setVolume(float32(v)) }
func (s *localSource) cycleRepeat()        { s.repeat = (s.repeat + 1) % 3 }

func (s *localSource) grab() {
	s.hand = s.d.platterAngle()
	s.d.holdAt(s.hand)
	s.d.hand.Store(true)
}

func (s *localSource) scrub(rev float64) {
	s.hand += rev
	s.d.holdAt(s.hand)
}

func (s *localSource) release() { s.d.hand.Store(false) }

func (s *localSource) entries() []entry  { return s.list }
func (s *localSource) listTitle() string { return "Queue" }

// choose plays the i-th track of the list, which is in play order.
func (s *localSource) choose(i int) {
	if i >= 0 && i < len(s.order) {
		s.play(i, true)
	}
}

func (s *localSource) toggleShuffle() {
	s.shuffle = !s.shuffle
	cur := s.order[s.at]
	if s.shuffle {
		s.reshuffle()
	} else {
		for i := range s.order {
			s.order[i] = i
		}
	}
	for i, v := range s.order {
		if v == cur {
			s.at = i
		}
	}
	s.dirty = true
}

func (s *localSource) reshuffle() {
	rand.Shuffle(len(s.order), func(i, j int) { s.order[i], s.order[j] = s.order[j], s.order[i] })
}

func (s *localSource) rebuildList() {
	s.dirty = false
	s.list = s.list[:0]
	for i, idx := range s.order {
		t := s.tracks[idx]
		s.list = append(s.list, entry{
			num: t.num, label: t.title, detail: t.artist, dur: t.dur,
			current: i == s.at, playing: i == s.at && s.d.motor.Load(),
		})
	}
}

func (s *localSource) artLoader(*nowPlaying) func() *art {
	t := *s.cur()
	return func() *art { return loadArt(&t) }
}
