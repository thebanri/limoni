//go:build linux || freebsd || openbsd || netbsd || dragonfly

package main

import (
	"context"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// MPRIS is how media players on Linux and the BSDs say what they play:
// Spotify, Firefox and Chromium (any tab with sound), mpv with its plugin,
// VLC, Rhythmbox, Elisa… Each owns a name org.mpris.MediaPlayer2.* on the
// session bus with the track's metadata, its position and methods to steer
// it. pikap follows one of them — the one playing, or the one chosen — and
// the record turns with it.
//
// Everything that talks to the bus runs on one worker goroutine: it asks
// every player for its state a few times a second and sends what the
// interface asks for. The interface sees a snapshot, and between
// snapshots it moves the position on by the clock.

const mprisPrefix = "org.mpris.MediaPlayer2."

type mprisPlayer struct {
	bus      string
	identity string
	status   string // Playing, Paused, Stopped
	title    string
	artist   string
	album    string
	artURL   string
	trackID  dbus.ObjectPath
	length   float64
	pos      float64
	rate     float64
	volume   float64
	canSeek  bool
	canCtl   bool
	shuffle  bool
	loop     string    // None, Track, Playlist
	seenPlay time.Time // when it was last seen playing
	key      string    // names the cover, for the interface to notice a new one
	// The player stopped saying where it is in a track it had a length
	// for: the length is the one it last gave, and the clock keeps the
	// position.
	noPos bool
}

type mprisCmd struct {
	bus    string
	method string // PlayPause, Next, Previous, SetPosition, Volume
	pos    float64
	track  dbus.ObjectPath
	vol    float64
	on     bool
	loop   string
}

type mprisSource struct {
	conn *dbus.Conn

	mu      sync.Mutex
	shared  []mprisPlayer // the worker's latest look
	version int
	sticky  string // the bus the person chose, if any

	cmds chan mprisCmd
	stop chan struct{}

	// The interface goroutine's own.
	seen      int
	players   []mprisPlayer
	cur       int // index into players, -1 for none
	base      float64
	baseAt    time.Time
	holdUntil time.Time // a seek was just sent: trust it over the player for a moment
	list      []entry
	grabbed   bool
	scrubTo   float64
	lastSent  time.Time
	now       func() time.Time
}

// newMPRIS connects to the session bus; it fails where there is none.
func newMPRIS() (*mprisSource, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	return newMPRISOn(conn), nil
}

func newMPRISOn(conn *dbus.Conn) *mprisSource {
	s := &mprisSource{
		conn: conn, cur: -1, cmds: make(chan mprisCmd, 32), stop: make(chan struct{}),
		now: time.Now,
	}
	go s.work()
	return s
}

func (s *mprisSource) close() {
	close(s.stop)
	_ = s.conn.Close()
}

func (s *mprisSource) work() {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	s.refresh()
	for {
		select {
		case <-s.stop:
			return
		case c := <-s.cmds:
			s.send(c)
			// Look again soon, so the answer shows before the next tick.
			time.Sleep(60 * time.Millisecond)
			s.refresh()
		case <-tick.C:
			s.refresh()
		}
	}
}

func (s *mprisSource) send(c mprisCmd) {
	obj := s.conn.Object(c.bus, "/org/mpris/MediaPlayer2")
	const iface = "org.mpris.MediaPlayer2.Player"
	switch c.method {
	case "SetPosition":
		us := int64(c.pos * 1e6)
		if c.track != "" && c.track.IsValid() {
			if obj.Call(iface+".SetPosition", 0, c.track, us).Err == nil {
				return
			}
		}
		// Players without track ids take a relative Seek instead.
		var cur int64
		if v, err := obj.GetProperty(iface + ".Position"); err == nil {
			cur, _ = asInt64(v.Value())
		}
		obj.Call(iface+".Seek", 0, us-cur)
	case "Volume":
		_ = obj.SetProperty(iface+".Volume", dbus.MakeVariant(c.vol))
	case "Shuffle":
		_ = obj.SetProperty(iface+".Shuffle", dbus.MakeVariant(c.on))
	case "LoopStatus":
		_ = obj.SetProperty(iface+".LoopStatus", dbus.MakeVariant(c.loop))
	default:
		obj.Call(iface+"."+c.method, 0)
	}
}

// refresh asks every player on the bus for its state.
func (s *mprisSource) refresh() {
	var names []string
	if err := s.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return
	}
	s.mu.Lock()
	old := s.shared
	s.mu.Unlock()
	var out []mprisPlayer
	for _, n := range names {
		if !strings.HasPrefix(n, mprisPrefix) {
			continue
		}
		p, ok := s.query(n)
		if !ok {
			continue
		}
		for _, o := range old {
			if o.bus != n {
				continue
			}
			p.seenPlay = o.seenPlay
			// Firefox on YouTube drops a video's length and puts its
			// position at 0 about a second after a seek, and leaves them so
			// while it plays on. A length that vanishes from the same track
			// is not the track turning into a stream.
			if p.length == 0 && o.length > 0 && p.sameTrack(&o) {
				p.length, p.noPos = o.length, true
			}
		}
		if p.status == "Playing" {
			p.seenPlay = time.Now()
		}
		out = append(out, p)
	}
	s.mu.Lock()
	s.shared = out
	s.version++
	s.mu.Unlock()
}

func (s *mprisSource) query(bus string) (mprisPlayer, bool) {
	obj := s.conn.Object(bus, "/org/mpris/MediaPlayer2")
	p := mprisPlayer{bus: bus, rate: 1, volume: -1}
	var props map[string]dbus.Variant
	if err := obj.Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.mpris.MediaPlayer2.Player").Store(&props); err != nil {
		return p, false
	}
	var root map[string]dbus.Variant
	if obj.Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.mpris.MediaPlayer2").Store(&root) == nil {
		p.identity, _ = root["Identity"].Value().(string)
	}
	if p.identity == "" {
		p.identity = strings.TrimPrefix(bus, mprisPrefix)
		p.identity, _, _ = strings.Cut(p.identity, ".")
	}
	p.status, _ = props["PlaybackStatus"].Value().(string)
	if v, ok := props["Rate"]; ok {
		if r, ok := v.Value().(float64); ok && r > 0 {
			p.rate = r
		}
	}
	if v, ok := props["Volume"]; ok {
		if r, ok := v.Value().(float64); ok {
			p.volume = r
		}
	}
	p.canSeek, _ = props["CanSeek"].Value().(bool)
	p.shuffle, _ = props["Shuffle"].Value().(bool)
	p.loop, _ = props["LoopStatus"].Value().(string)
	p.canCtl, _ = props["CanControl"].Value().(bool)
	if us, ok := asInt64(props["Position"].Value()); ok {
		p.pos = float64(us) / 1e6
	} else if v, err := obj.GetProperty("org.mpris.MediaPlayer2.Player.Position"); err == nil {
		// Some players leave Position out of GetAll, since it changes
		// without a signal.
		us, _ := asInt64(v.Value())
		p.pos = float64(us) / 1e6
	}
	if meta, ok := props["Metadata"].Value().(map[string]dbus.Variant); ok {
		p.title, _ = meta["xesam:title"].Value().(string)
		p.album, _ = meta["xesam:album"].Value().(string)
		p.artURL, _ = meta["mpris:artUrl"].Value().(string)
		switch a := meta["xesam:artist"].Value().(type) {
		case []string:
			p.artist = strings.Join(a, ", ")
		case string:
			p.artist = a
		}
		if p.artist == "" {
			if a, ok := meta["xesam:albumArtist"].Value().([]string); ok {
				p.artist = strings.Join(a, ", ")
			}
		}
		switch id := meta["mpris:trackid"].Value().(type) {
		case dbus.ObjectPath:
			p.trackID = id
		case string:
			p.trackID = dbus.ObjectPath(id)
		}
		if us, ok := asInt64(meta["mpris:length"].Value()); ok {
			p.length = float64(us) / 1e6
		}
		if p.title == "" {
			if u, ok := meta["xesam:url"].Value().(string); ok {
				p.title = titleFromURL(u)
			}
		}
	}
	p.key = p.artURL + "\x00" + p.album + "\x00" + p.artist
	if p.artURL == "" && p.album == "" && p.artist == "" {
		p.key += "\x00" + p.title
	}
	return p, true
}

// sameTrack is whether p and o name the same track of one player.
func (p *mprisPlayer) sameTrack(o *mprisPlayer) bool {
	return p.bus == o.bus && p.trackID == o.trackID && p.title == o.title &&
		p.artist == o.artist && p.album == o.album
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case uint64:
		return int64(n), true
	case int32:
		return int64(n), true
	case uint32:
		return int64(n), true
	case float64:
		return int64(n), true
	}
	return 0, false
}

func titleFromURL(u string) string {
	p, err := url.Parse(u)
	if err != nil || p.Path == "" {
		return ""
	}
	base := p.Path[strings.LastIndexByte(p.Path, '/')+1:]
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	return base
}

// update takes the worker's latest look, when there is a new one, and
// carries the position forward by the clock in between.
func (s *mprisSource) update(np *nowPlaying) {
	now := s.now()
	s.mu.Lock()
	if s.version != s.seen {
		s.seen = s.version
		s.players = append(s.players[:0], s.shared...)
		s.pick()
		s.rebuildList()
		s.mu.Unlock()
		if s.cur >= 0 {
			p := &s.players[s.cur]
			pos := p.pos
			// Keep the clock's position while it agrees with the player's,
			// or the needle would twitch at every look.
			if now.Before(s.holdUntil) || s.grabbed {
				pos = s.base
			} else if p.noPos {
				pos = s.predicted(now, p)
			} else if p.status == "Playing" && abs(s.predicted(now, p)-pos) < 0.35 {
				pos = s.predicted(now, p)
			}
			s.base, s.baseAt = pos, now
		}
	} else {
		s.mu.Unlock()
	}
	*np = nowPlaying{volume: -1}
	if s.cur < 0 {
		np.status = "Nothing is playing. Start music in any player — Spotify, a browser tab, mpv, VLC…"
		return
	}
	p := &s.players[s.cur]
	np.title, np.artist, np.album, np.player = p.title, p.artist, p.album, p.identity
	if np.title == "" {
		np.title = p.identity
	}
	np.artKey = p.key
	np.length = p.length
	np.playing = p.status == "Playing"
	np.canSeek = p.canSeek
	np.volume = p.volume
	np.shuffle = p.shuffle
	switch p.loop {
	case "Playlist":
		np.repeat = 1
	case "Track":
		np.repeat = 2
	}
	np.pos = s.base
	if s.grabbed {
		np.pos = s.scrubTo
	} else if np.playing {
		np.pos = s.predicted(now, p)
	}
	if np.length > 0 {
		np.pos = min(np.pos, np.length)
	}
}

func (s *mprisSource) predicted(now time.Time, p *mprisPlayer) float64 {
	if p.status != "Playing" {
		return s.base
	}
	return s.base + now.Sub(s.baseAt).Seconds()*p.rate
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// pick chooses whom to follow: the one chosen while it is there, else
// whoever plays — the most recent to start, if several do — else whoever
// played last. It holds s.mu.
func (s *mprisSource) pick() {
	prev := ""
	if s.cur >= 0 && s.cur < len(s.players) {
		prev = s.players[s.cur].bus
	}
	s.cur = -1
	for i := range s.players {
		if s.players[i].bus == s.sticky {
			s.cur = i
			return
		}
	}
	best := time.Time{}
	for i := range s.players {
		p := &s.players[i]
		if p.status == "Playing" && (s.cur < 0 || p.seenPlay.After(best)) {
			s.cur, best = i, p.seenPlay
		}
	}
	if s.cur >= 0 {
		// Two playing at once: stay with the one followed already.
		for i := range s.players {
			if s.players[i].bus == prev && s.players[i].status == "Playing" {
				s.cur = i
			}
		}
		return
	}
	for i := range s.players {
		if s.players[i].bus == prev {
			s.cur = i
			return
		}
	}
	for i := range s.players {
		if s.cur < 0 || s.players[i].seenPlay.After(best) {
			s.cur, best = i, s.players[i].seenPlay
		}
	}
}

func (s *mprisSource) rebuildList() {
	s.list = s.list[:0]
	for i := range s.players {
		p := &s.players[i]
		detail := p.title
		if p.artist != "" && detail != "" {
			detail = p.artist + " — " + detail
		}
		s.list = append(s.list, entry{
			num: i + 1, label: p.identity, detail: detail, dur: p.length,
			current: i == s.cur, playing: p.status == "Playing",
		})
	}
}

func (s *mprisSource) do(c mprisCmd) {
	if s.cur < 0 {
		return
	}
	c.bus = s.players[s.cur].bus
	select {
	case s.cmds <- c:
	default:
	}
}

func (s *mprisSource) toggle() {
	s.do(mprisCmd{method: "PlayPause"})
	// Show it at once; the player's answer follows.
	if s.cur >= 0 {
		p := &s.players[s.cur]
		now := s.now()
		s.base, s.baseAt = s.predicted(now, p), now
		if p.status == "Playing" {
			p.status = "Paused"
		} else {
			p.status = "Playing"
		}
		s.holdUntil = now.Add(700 * time.Millisecond)
		s.rebuildList()
	}
}

func (s *mprisSource) next() { s.do(mprisCmd{method: "Next"}) }
func (s *mprisSource) prev() { s.do(mprisCmd{method: "Previous"}) }

func (s *mprisSource) seek(sec float64) {
	if s.cur < 0 || !s.players[s.cur].canSeek {
		return
	}
	p := &s.players[s.cur]
	sec = max(0, sec)
	if p.length > 0 {
		sec = min(sec, p.length-0.5)
	}
	s.do(mprisCmd{method: "SetPosition", pos: sec, track: p.trackID})
	s.base, s.baseAt = sec, s.now()
	s.holdUntil = s.baseAt.Add(900 * time.Millisecond)
}

func (s *mprisSource) setVolume(v float64) {
	s.do(mprisCmd{method: "Volume", vol: max(0, min(1, v))})
	if s.cur >= 0 {
		s.players[s.cur].volume = max(0, min(1, v))
	}
}

func (s *mprisSource) toggleShuffle() {
	if s.cur >= 0 {
		s.players[s.cur].shuffle = !s.players[s.cur].shuffle
		s.do(mprisCmd{method: "Shuffle", on: s.players[s.cur].shuffle})
	}
}

func (s *mprisSource) cycleRepeat() {
	if s.cur < 0 {
		return
	}
	p := &s.players[s.cur]
	switch p.loop {
	case "Playlist":
		p.loop = "Track"
	case "Track":
		p.loop = "None"
	default:
		p.loop = "Playlist"
	}
	s.do(mprisCmd{method: "LoopStatus", loop: p.loop})
}

// A turn of the record is ten seconds of someone else's track: a real
// record's 1.8 s would take a long wind to get anywhere, and the player
// cannot be scratched anyway — only moved.
const scrubPerRev = 10.0

func (s *mprisSource) grab() {
	if s.cur < 0 {
		return
	}
	now := s.now()
	s.grabbed = true
	s.scrubTo = s.predicted(now, &s.players[s.cur])
	s.lastSent = time.Time{}
}

func (s *mprisSource) scrub(rev float64) {
	if !s.grabbed || s.cur < 0 {
		return
	}
	s.scrubTo = max(0, s.scrubTo+rev*scrubPerRev)
	if l := s.players[s.cur].length; l > 0 {
		s.scrubTo = min(s.scrubTo, l-0.5)
	}
	// Players take a while over each seek; a few a second is plenty.
	if now := s.now(); now.Sub(s.lastSent) > 150*time.Millisecond {
		s.lastSent = now
		s.seek(s.scrubTo)
	}
}

func (s *mprisSource) release() {
	if !s.grabbed {
		return
	}
	s.grabbed = false
	s.seek(s.scrubTo)
}

func (s *mprisSource) entries() []entry  { return s.list }
func (s *mprisSource) listTitle() string { return "Players" }

func (s *mprisSource) choose(i int) {
	if i < 0 || i >= len(s.players) {
		return
	}
	s.mu.Lock()
	s.sticky = s.players[i].bus
	s.pick()
	s.rebuildList()
	s.mu.Unlock()
	if s.cur >= 0 {
		s.base, s.baseAt = s.players[s.cur].pos, s.now()
	}
}

// art fetches the player's cover: a file, or an address on the web (where
// Spotify keeps them), or one painted from the names when there is none.
func (s *mprisSource) artLoader(np *nowPlaying) func() *art {
	u, _, _ := strings.Cut(np.artKey, "\x00")
	name := np.artist + "\x00" + np.album
	if np.album == "" {
		name += "\x00" + np.title
	}
	return func() *art {
		if img := fetchImage(u); img != nil {
			return fromImage(img)
		}
		return generatedArt(name)
	}
}

func fetchImage(u string) image.Image {
	switch {
	case u == "":
		return nil
	case strings.HasPrefix(u, "file://"):
		p, err := url.Parse(u)
		if err != nil {
			return nil
		}
		f, err := os.Open(p.Path)
		if err != nil {
			return nil
		}
		defer f.Close()
		return decodeImage(f)
	case strings.HasPrefix(u, "http://"), strings.HasPrefix(u, "https://"):
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil
		}
		return decodeImage(io.LimitReader(resp.Body, 16<<20))
	}
	return nil
}
