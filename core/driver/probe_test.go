package driver

import (
	"strings"
	"testing"
	"time"
)

// Each reply to ProbeQueries, as real terminals send them.
func TestParseTerminalReplies(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  ReplyEvent
	}{
		{"DA1 xterm with sixel", "\x1b[?64;1;2;4;6;9;15;18;21;22c",
			ReplyEvent{Kind: ReplyPrimaryDA, Attributes: 1<<1 | 1<<2 | 1<<4 | 1<<6 | 1<<9 | 1<<15 | 1<<18 | 1<<21 | 1<<22}},
		{"DA1 minimal", "\x1b[?1;2c", ReplyEvent{Kind: ReplyPrimaryDA, Attributes: 1 << 2}},
		{"DECRPM 2026 reset", "\x1b[?2026;2$y", ReplyEvent{Kind: ReplyMode, Mode: 2026, Setting: 2}},
		{"DECRPM 2027 set", "\x1b[?2027;1$y", ReplyEvent{Kind: ReplyMode, Mode: 2027, Setting: 1}},
		{"DECRPM unknown mode", "\x1b[?2027;0$y", ReplyEvent{Kind: ReplyMode, Mode: 2027, Setting: 0}},
		{"kitty keyboard flags", "\x1b[?0u", ReplyEvent{Kind: ReplyKittyKeyboard, Flags: 0}},
		{"kitty keyboard flags set", "\x1b[?31u", ReplyEvent{Kind: ReplyKittyKeyboard, Flags: 31}},
		{"XTVERSION ST", "\x1bP>|kitty(0.39.1)\x1b\\", ReplyEvent{Kind: ReplyVersion, Version: "kitty(0.39.1)"}},
		{"XTVERSION BEL", "\x1bP>|tmux 3.5a\x07", ReplyEvent{Kind: ReplyVersion, Version: "tmux 3.5a"}},
		{"cursor position", "\x1b[12;40R", ReplyEvent{Kind: ReplyCursor, Row: 12, Col: 40}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ev, n := ParseEvent([]byte(tc.input))
			if n != len(tc.input) {
				t.Fatalf("consumed %d of %d bytes", n, len(tc.input))
			}
			if ev.Type != EventReply || ev.Reply != tc.want {
				t.Fatalf("got %+v, want reply %+v", ev, tc.want)
			}
		})
	}
}

// A reply must never be mistaken for typing. Before the parser knew the '?'
// marker, "CSI ? 97 u" — Kitty flags, had they been 97 — came out as the key 'a'.
func TestRepliesAreNotKeys(t *testing.T) {
	for _, in := range []string{
		"\x1b[?97u",           // kitty flags that happen to be a printable code point
		"\x1b[>1;4000;15c",    // DA2, never queried but sometimes volunteered
		"\x1b[?1;2;3;4;5$y",   // malformed DECRPM
		"\x1bP>|\x1b\\",       // empty XTVERSION
		"\x1bP1$r0m\x1b\\",    // DECRQSS reply
		"\x1b]11;rgb:0/0/0\a", // OSC 11 background colour
	} {
		ev, n := ParseEvent([]byte(in))
		if n != len(in) {
			t.Errorf("%q: consumed %d of %d", in, n, len(in))
		}
		if ev.Type == EventKey || ev.Type == EventMouse {
			t.Errorf("%q parsed as input: %+v", in, ev)
		}
	}
}

// A long DA1 reply split across reads must wait for its final byte rather than
// be discarded at the old 32-byte limit and leak its tail as keystrokes.
func TestLongReplySplitAcrossReads(t *testing.T) {
	full := "\x1b[?65;1;2;3;4;6;7;8;9;15;16;17;18;21;22;28;29c"
	if len(full) <= 32 {
		t.Fatalf("test reply is only %d bytes; it must exceed the old limit", len(full))
	}
	for cut := 3; cut < len(full); cut++ {
		if _, n := ParseEvent([]byte(full[:cut])); n != 0 {
			t.Fatalf("partial reply of %d bytes consumed %d; want to wait", cut, n)
		}
	}
	ev, n := ParseEvent([]byte(full))
	if n != len(full) || ev.Type != EventReply || ev.Reply.Kind != ReplyPrimaryDA {
		t.Fatalf("got %+v after %d bytes", ev, n)
	}
}

// Parsing a key, a mouse report or a reply must not allocate: mouse motion
// alone arrives hundreds of times a second.
func TestParsingInputDoesNotAllocate(t *testing.T) {
	for _, in := range []string{"\x1b[1;5A", "\x1b[<35;120;40M", "\x1b[3~", "\x1b[27;5;13~", "\x1b[?2026;1$y", "x"} {
		buf := []byte(in)
		if a := testing.AllocsPerRun(100, func() { ParseEvent(buf) }); a != 0 {
			t.Errorf("ParseEvent(%q) allocates %.0f times", in, a)
		}
	}
}

func TestSplitVersion(t *testing.T) {
	for in, want := range map[string][2]string{
		"kitty(0.39.1)":           {"kitty", "0.39.1"},
		"tmux 3.5a":               {"tmux", "3.5a"},
		"XTerm(398)":              {"XTerm", "398"},
		"WezTerm 20240203-110809": {"WezTerm", "20240203-110809"},
		"foot":                    {"foot", ""},
	} {
		if name, version := splitVersion(in); name != want[0] || version != want[1] {
			t.Errorf("splitVersion(%q) = %q, %q; want %q, %q", in, name, version, want[0], want[1])
		}
	}
}

// The backend keeps replies for itself: the application sees the key typed
// between them and nothing else, and TerminalReport holds the answers.
func TestBackendConsumesRepliesIntoReport(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "")
	replies := "\x1bP>|foot(1.20.2)\x1b\\" + "\x1b[?2026;2$y" + "x" + "\x1b[?2027;1$y" + "\x1b[?0u" +
		"\x1b[5;3R" + "\x1b[5;7R" + // REP worked; the family emoji took six columns
		"\x1b[?62;4;22c" +
		"\x1b[9;9R" // a cursor report after DA1 is not a measurement
	io := NewMemoryTerminalIO([]byte(replies), 80, 24)
	b := NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(io.Output()), ProbeQueries) {
		t.Fatalf("setup did not send the probe: %q", io.Output())
	}
	b.StartEventLoop()

	select {
	case ev := <-b.Events():
		if ev.Type != EventKey || ev.Key.Ch != 'x' {
			t.Fatalf("first event %+v; want the key x", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("key never arrived")
	}

	deadline := time.Now().Add(2 * time.Second)
	var r TerminalReport
	for time.Now().Before(deadline) {
		if r, _ = b.TerminalReport(); r.Answered {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	want := TerminalReport{
		Answered: true, Name: "foot", Version: "1.20.2",
		SyncOutput: ModeReset, GraphemeClusters: ModeSet,
		KittyKeyboard: true, Sixel: true,
		Repeat: Yes, ClusterWidth: 6,
	}
	if r != want {
		t.Fatalf("report %+v\nwant   %+v", r, want)
	}
	select {
	case ev := <-b.Events():
		t.Fatalf("reply leaked to the application: %+v", ev)
	default:
	}
	_ = b.Close()
}

// LIMONI_PROBE=0 sends nothing, for a terminal that misbehaves on a query.
func TestProbeCanBeTurnedOff(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := NewMemoryTerminalIO(nil, 80, 24)
	b := NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(io.Output()), "\x1b[c") {
		t.Fatalf("probe sent although disabled: %q", io.Output())
	}
	_ = b.Close()
}

// Close waits for the DA1 sentinel so late replies do not reach the shell,
// but only boundedly: a terminal that never answers must not hang the exit.
func TestCloseWaitsBoundedlyForReplies(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "")
	io := NewMemoryTerminalIO(nil, 80, 24) // never answers
	b := NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	b.StartEventLoop()
	start := time.Now()
	_ = b.Close()
	if d := time.Since(start); d < probeDrainTimeout/2 || d > probeDrainTimeout+time.Second {
		t.Fatalf("Close took %v; want about %v", d, probeDrainTimeout)
	}
}

func TestModeStateRecognized(t *testing.T) {
	// "Supported but off" is recognised; no answer at all is not.
	for _, tc := range []struct {
		m    ModeState
		want bool
	}{
		{ModeUnknown, false},
		{ModeUnsupported, false},
		{ModeSet, true},
		{ModeReset, true},
		{ModePermanentlySet, true},
		{ModePermanentlyReset, true},
	} {
		if got := tc.m.Recognized(); got != tc.want {
			t.Errorf("%v.Recognized() = %v, want %v", tc.m, got, tc.want)
		}
	}
}

func TestModeStateEnabled(t *testing.T) {
	for _, tc := range []struct {
		m    ModeState
		want bool
	}{
		{ModeUnknown, false},
		{ModeUnsupported, false},
		{ModeSet, true},
		{ModeReset, false},
		{ModePermanentlySet, true},
		{ModePermanentlyReset, false},
	} {
		if got := tc.m.Enabled(); got != tc.want {
			t.Errorf("%v.Enabled() = %v, want %v", tc.m, got, tc.want)
		}
	}
}

func TestTerminalReportVersion(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "")
	io := NewMemoryTerminalIO(nil, 80, 24)
	b := NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	if v := b.TerminalReportVersion(); v != 0 {
		t.Fatalf("fresh report version %d, want 0", v)
	}
	da1, _ := ParseEvent([]byte("\x1b[?62;22c"))
	if !b.replies.record(da1) {
		t.Fatal("DA1 was not recorded as a reply")
	}
	if v := b.TerminalReportVersion(); v != 1 {
		t.Fatalf("after DA1 version %d, want 1", v)
	}
	r, n := b.TerminalReport()
	if !r.Answered || n != 1 {
		t.Fatalf("report %+v version %d; want answered with version 1", r, n)
	}
}

// A terminal that ignores CSI 6 n measures nothing. A cursor report arriving
// after DA1 — from the application's own query, say — is not a measurement.
func TestCursorReportAfterSentinelIsNotAMeasurement(t *testing.T) {
	var c replyCollector
	c.markSent()
	for _, in := range []string{"\x1b[?62;22c", "\x1b[1;3R", "\x1b[1;3R"} {
		ev, _ := ParseEvent([]byte(in))
		c.record(ev)
	}
	if r, _ := c.snapshot(); r.Repeat != Unmeasured || r.ClusterWidth != 0 {
		t.Fatalf("late cursor reports taken as measurements: %+v", r)
	}
}
