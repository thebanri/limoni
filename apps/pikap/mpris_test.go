//go:build linux

package main

import (
	"bufio"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

// privateBus starts a dbus-daemon of the test's own, so nothing here
// touches the session the tests run in.
func privateBus(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon")
	}
	cmd := exec.Command(bin, "--session", "--nofork", "--print-address")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skip("dbus-daemon did not start:", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(addr)
}

func connect(t *testing.T, addr string) *dbus.Conn {
	t.Helper()
	c, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// fakePlayer is a media player on the bus, as Spotify would be: it has a
// track, a position, and takes the Player methods.
type fakePlayer struct {
	mu     sync.Mutex
	props  *prop.Properties
	calls  []string
	setPos []int64
}

func (f *fakePlayer) note(s string) {
	f.mu.Lock()
	f.calls = append(f.calls, s)
	f.mu.Unlock()
}

func (f *fakePlayer) PlayPause() *dbus.Error {
	f.note("PlayPause")
	st := f.props.GetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus").(string)
	if st == "Playing" {
		st = "Paused"
	} else {
		st = "Playing"
	}
	f.props.SetMust("org.mpris.MediaPlayer2.Player", "PlaybackStatus", st)
	return nil
}
func (f *fakePlayer) Next() *dbus.Error     { f.note("Next"); return nil }
func (f *fakePlayer) Previous() *dbus.Error { f.note("Previous"); return nil }
func (f *fakePlayer) SetPosition(id dbus.ObjectPath, us int64) *dbus.Error {
	f.note("SetPosition " + string(id))
	f.mu.Lock()
	f.setPos = append(f.setPos, us)
	f.mu.Unlock()
	f.props.SetMust("org.mpris.MediaPlayer2.Player", "Position", us)
	return nil
}
func (f *fakePlayer) seekBy(us int64) *dbus.Error { f.note("Seek"); return nil }

func serveFake(t *testing.T, addr, name, title, status string, pos int64) *fakePlayer {
	t.Helper()
	conn := connect(t, addr)
	t.Cleanup(func() { conn.Close() })
	f := &fakePlayer{}
	path := dbus.ObjectPath("/org/mpris/MediaPlayer2")
	methods := map[string]any{
		"PlayPause": f.PlayPause, "Next": f.Next, "Previous": f.Previous,
		"SetPosition": f.SetPosition, "Seek": f.seekBy,
	}
	if err := conn.ExportMethodTable(methods, path, "org.mpris.MediaPlayer2.Player"); err != nil {
		t.Fatal(err)
	}
	props, err := prop.Export(conn, path, prop.Map{
		"org.mpris.MediaPlayer2": {
			"Identity": {Value: name},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": {Value: status, Writable: true},
			"Position":       {Value: pos},
			"Rate":           {Value: 1.0, Writable: true},
			"Volume":         {Value: 0.5, Writable: true},
			"Shuffle":        {Value: false, Writable: true},
			"LoopStatus":     {Value: "None", Writable: true},
			"CanSeek":        {Value: true},
			"CanControl":     {Value: true},
			"Metadata": {Value: map[string]dbus.Variant{
				"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/track/1")),
				"mpris:length":  dbus.MakeVariant(int64(180e6)),
				"xesam:title":   dbus.MakeVariant(title),
				"xesam:artist":  dbus.MakeVariant([]string{"Feldup"}),
				"xesam:album":   dbus.MakeVariant("Discover Weekly"),
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	f.props = props
	reply, err := conn.RequestName("org.mpris.MediaPlayer2."+strings.ToLower(name), dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatal("could not take the name", err)
	}
	return f
}

// waitFor updates the source until ok says so, or fails after a while.
func waitFor(t *testing.T, s *mprisSource, what string, ok func(np nowPlaying) bool) nowPlaying {
	t.Helper()
	var np nowPlaying
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.update(&np)
		if ok(np) {
			return np
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s; last: %+v", what, np)
	return np
}

func (f *fakePlayer) called(m string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.HasPrefix(c, m) {
			return true
		}
	}
	return false
}

func TestMPRISFollowsAndSteers(t *testing.T) {
	addr := privateBus(t)
	f := serveFake(t, addr, "Spotify", "Waters", "Playing", 61e6)
	s := newMPRISOn(connect(t, addr))
	defer s.close()

	np := waitFor(t, s, "the track", func(np nowPlaying) bool { return np.title == "Waters" })
	if np.artist != "Feldup" || np.album != "Discover Weekly" || np.player != "Spotify" ||
		!np.playing || np.length != 180 || !np.canSeek || np.volume != 0.5 {
		t.Fatalf("read %+v", np)
	}
	if math.Abs(np.pos-61) > 0.5 {
		t.Fatalf("position %.2f, want 61", np.pos)
	}

	// Between looks the clock moves the position on.
	time.Sleep(300 * time.Millisecond)
	s.update(&np)
	if np.pos < 61.2 {
		t.Fatalf("a playing track's position did not move on: %.2f", np.pos)
	}

	s.seek(100)
	waitFor(t, s, "SetPosition", func(nowPlaying) bool { return f.called("SetPosition /track/1") })
	f.mu.Lock()
	got := f.setPos[len(f.setPos)-1]
	f.mu.Unlock()
	if got != 100e6 {
		t.Fatalf("SetPosition got %d µs, want 100e6", got)
	}

	s.toggle()
	np = waitFor(t, s, "pause", func(np nowPlaying) bool { return f.called("PlayPause") && !np.playing })
	// Paused, the position stands still.
	p := np.pos
	time.Sleep(200 * time.Millisecond)
	s.update(&np)
	if np.pos != p {
		t.Fatalf("paused, the position moved from %.2f to %.2f", p, np.pos)
	}

	s.next()
	s.prev()
	waitFor(t, s, "Next and Previous", func(nowPlaying) bool { return f.called("Next") && f.called("Previous") })

	// Scrubbing: two turns forward is twenty seconds, sent as the hand
	// moves and once more when it lets go.
	s.grab()
	for range 20 {
		s.scrub(0.1)
		time.Sleep(10 * time.Millisecond)
	}
	s.release()
	waitFor(t, s, "the scrub to land", func(nowPlaying) bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return len(f.setPos) > 0 && math.Abs(float64(f.setPos[len(f.setPos)-1])/1e6-(p+20)) < 0.5
	})
}

// TestMPRISKeepsTheLengthFirefoxDrops plays out what Firefox does on
// YouTube a second after a seek: the video's length leaves the metadata and
// the position reads 0, while the video plays on from where it was put.
func TestMPRISKeepsTheLengthFirefoxDrops(t *testing.T) {
	addr := privateBus(t)
	f := serveFake(t, addr, "Firefox", "A video", "Playing", 61e6)
	s := newMPRISOn(connect(t, addr))
	defer s.close()
	waitFor(t, s, "the video", func(np nowPlaying) bool { return np.length == 180 })

	s.seek(100)
	waitFor(t, s, "SetPosition", func(nowPlaying) bool { return f.called("SetPosition") })
	// prop stores a map into the one it has, so a key cannot be taken
	// out; a length of 0 reads the same to pikap as none.
	f.props.SetMust("org.mpris.MediaPlayer2.Player", "Metadata", map[string]dbus.Variant{
		"mpris:length": dbus.MakeVariant(int64(0)),
	})
	f.props.SetMust("org.mpris.MediaPlayer2.Player", "Position", int64(0))
	// Past the hold after a seek, and past a few looks at the player.
	time.Sleep(1500 * time.Millisecond)
	var np nowPlaying
	s.update(&np)
	if np.length != 180 {
		t.Fatalf("the length went from 180 to %.0f", np.length)
	}
	if np.pos < 100.5 || np.pos > 103 {
		t.Fatalf("position %.2f, want a little past 100, where the video was put", np.pos)
	}
}

func TestMPRISPrefersThePlayingOne(t *testing.T) {
	addr := privateBus(t)
	serveFake(t, addr, "Firefox", "A video", "Paused", 0)
	serveFake(t, addr, "Spotify", "Waters", "Playing", 0)
	s := newMPRISOn(connect(t, addr))
	defer s.close()
	waitFor(t, s, "Spotify, which plays", func(np nowPlaying) bool { return np.player == "Spotify" })

	// Choosing the other one keeps it, though it is paused.
	var idx int
	for i, e := range s.entries() {
		if e.label == "Firefox" {
			idx = i
		}
	}
	s.choose(idx)
	var np nowPlaying
	for range 10 {
		time.Sleep(60 * time.Millisecond)
		s.update(&np)
		if np.player != "Firefox" {
			t.Fatalf("chose Firefox, following %q", np.player)
		}
	}
}

func TestMPRISWithNobody(t *testing.T) {
	addr := privateBus(t)
	s := newMPRISOn(connect(t, addr))
	defer s.close()
	time.Sleep(100 * time.Millisecond)
	var np nowPlaying
	s.update(&np)
	if np.status == "" || np.title != "" {
		t.Fatalf("with no players: %+v", np)
	}
	// Steering nobody does nothing, and does not panic.
	s.toggle()
	s.seek(10)
	s.grab()
	s.scrub(1)
	s.release()
}

// sessionBus is the bus the tests were started on, before TestMain hides it.
var sessionBus string

func TestMain(m *testing.M) {
	// The tests' buses are their own; make sure nothing reaches for the
	// person's session by accident.
	sessionBus = os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", "disabled:")
	os.Exit(m.Run())
}

// TestServeFakePlayer is not a test but a player to try pikap against, on
// a bus of its own; it plays a 3-minute track in real time until killed:
//
//	dbus-run-session -- sh -c 'PIKAP_FAKE=1 go test -run TestServeFakePlayer -timeout 0 & sleep 1; go run .'
func TestServeFakePlayer(t *testing.T) {
	if os.Getenv("PIKAP_FAKE") == "" || sessionBus == "" {
		t.Skip("set PIKAP_FAKE=1 under dbus-run-session to serve a fake player")
	}
	f := serveFake(t, sessionBus, "Spotify", "Waters", "Playing", 61e6)
	const iface = "org.mpris.MediaPlayer2.Player"
	for {
		time.Sleep(100 * time.Millisecond)
		if f.props.GetMust(iface, "PlaybackStatus").(string) != "Playing" {
			continue
		}
		pos := f.props.GetMust(iface, "Position").(int64) + 100e3
		if pos > 180e6 {
			pos = 0
		}
		f.props.SetMust(iface, "Position", pos)
	}
}
