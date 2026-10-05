package main

// A source is what the turntable shows and steers: either the media some
// other program is playing (mpris.go), or files pikap plays itself
// (local.go). The interface goroutine calls every method; a source does
// its slow work elsewhere and never blocks here.
type source interface {
	// update refreshes np for this frame. It must not allocate: it runs
	// every frame.
	update(np *nowPlaying)
	toggle()
	next()
	prev()
	seek(sec float64) // to an absolute position
	setVolume(v float64)
	toggleShuffle()
	cycleRepeat() // off, all, one

	// The record under a hand. scrub moves it by rev revolutions.
	grab()
	scrub(rev float64)
	release()

	// The list beside the deck: the queue, or the players to follow.
	entries() []entry
	choose(i int)
	listTitle() string

	// artLoader returns what loads the cover named by np.artKey. It is
	// called on the interface goroutine, when the key changes, and what it
	// returns runs on another — so it takes copies, not the source's state.
	artLoader(np *nowPlaying) func() *art
	close()
}

type nowPlaying struct {
	title, artist, album string
	player               string // who is playing it
	artKey               string // changes when the cover does
	length               float64
	pos                  float64
	playing              bool
	canSeek              bool
	volume               float64 // 0…1, or -1 when unknown

	// The platter's speed, when the source knows it (local playback runs
	// real physics); otherwise the interface spins one itself. 1 is the
	// music's own speed, whatever the record is shown turning at.
	hasPlatter bool
	speed      float64

	shuffle bool
	repeat  int // 0 off, 1 all, 2 one

	status string // a line to show when there is nothing to play
	note   string // a line to show under the controls
}

type entry struct {
	num     int
	label   string
	detail  string
	dur     float64
	current bool
	playing bool
}
