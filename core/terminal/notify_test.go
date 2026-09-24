package terminal

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/driver"
)

func TestNotifySequences(t *testing.T) {
	for _, tc := range []struct {
		name        string
		p           NotifyProtocol
		title, body string
		want        string
	}{
		{"osc 9", NotifyOSC9, "Build", "done", "\x1b]9;Build: done\a"},
		{"osc 9 body only", NotifyOSC9, "", "done", "\x1b]9;done\a"},
		// "4;..." would be ConEmu's progress bar command.
		{"osc 9 command-like", NotifyOSC9, "", "4;3;50", "\x1b]9; 4;3;50\a"},
		{"osc 777", NotifyOSC777, "a;b", "c;d", "\x1b]777;notify;a,b;c;d\a"},
		{"osc 99", NotifyOSC99, "Build", "done", "\x1b]99;i=limoni:d=0:p=title;Build\x1b\\\x1b]99;i=limoni:d=1:p=body;done\x1b\\"},
		{"osc 99 title only", NotifyOSC99, "Build", "", "\x1b]99;i=limoni:d=1:p=title;Build\x1b\\"},
		{"none", NotifyNone, "a", "b", ""},
		{"empty", NotifyOSC9, "", "", ""},
	} {
		if got := string(notifySequence(tc.p, tc.title, tc.body)); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// Nothing inside the text may end the sequence: not BEL, not ESC \, and not
// U+009C, the C1 string terminator a UTF-8 terminal also accepts.
func TestNotifyCannotBeEscaped(t *testing.T) {
	for _, p := range []NotifyProtocol{NotifyOSC9, NotifyOSC777, NotifyOSC99} {
		seq := string(notifySequence(p, "t\x07\x1b\\\u009c\x9cx", "b\x1b]52;c;cHdu\x07"))
		inner := seq[1 : len(seq)-1] // drop the leading ESC and the final byte
		if p == NotifyOSC99 {
			inner = strings.ReplaceAll(seq, "\x1b\\\x1b]99;", "") // the joint between title and body
			inner = strings.TrimSuffix(strings.TrimPrefix(inner, "\x1b"), "\x1b\\")
		}
		if strings.ContainsAny(inner, "\x07\x1b\u009c") || strings.Contains(inner, "\x9c") {
			t.Errorf("protocol %d leaks a terminator: %q", p, seq)
		}
		// ESC goes; the backslash after it is harmless on its own and stays.
		if !strings.Contains(seq, `t\x`) {
			t.Errorf("protocol %d lost the printable text: %q", p, seq)
		}
	}
}

func TestSanitizeOSCTextKeepsUnicode(t *testing.T) {
	for in, want := range map[string]string{
		"Derleme bitti ✅ \U0001F468\u200D\U0001F469\u200D\U0001F467": "Derleme bitti ✅ \U0001F468\u200D\U0001F469\u200D\U0001F467",
		"a\tb\nc":  "abc",
		"a\u0085b": "ab",
		"a\xffb":   "ab",
	} {
		if got := sanitizeOSCText(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

func TestNotifyAndPointerDetection(t *testing.T) {
	for _, tc := range []struct {
		term, prog string
		notify     NotifyProtocol
		pointer    bool
	}{
		{"xterm-kitty", "", NotifyOSC99, true},
		{"xterm-ghostty", "ghostty", NotifyOSC9, true},
		{"foot", "", NotifyOSC9, true},
		{"xterm-256color", "iTerm.app", NotifyOSC9, false},
		{"xterm-256color", "WezTerm", NotifyOSC9, false},
		{"xterm-256color", "", NotifyNone, false},
		{"alacritty", "", NotifyNone, false},
	} {
		t.Setenv("LIMONI_NOTIFY", "")
		t.Setenv("LIMONI_POINTER", "")
		n, p := notifyAndPointerFromEnv(tc.term, tc.prog)
		if n != tc.notify || p != tc.pointer {
			t.Errorf("TERM=%s TERM_PROGRAM=%s: notify %d pointer %v, want %d %v", tc.term, tc.prog, n, p, tc.notify, tc.pointer)
		}
	}
	t.Setenv("LIMONI_NOTIFY", "777")
	t.Setenv("LIMONI_POINTER", "0")
	if n, p := notifyAndPointerFromEnv("xterm-kitty", ""); n != NotifyOSC777 || p {
		t.Errorf("overrides ignored: %d %v", n, p)
	}
}

func TestWithReportTurnsOnKittyKeyboard(t *testing.T) {
	t.Setenv("LIMONI_KITTY_KEYBOARD", "")
	if p := (CapabilityProfile{}).WithReport(driver.TerminalReport{Answered: true, KittyKeyboard: true}); !p.KittyKeyboard {
		t.Error("a terminal that answered the kitty keyboard query did not get it")
	}
	if p := (CapabilityProfile{}).WithReport(driver.TerminalReport{Answered: true}); p.KittyKeyboard {
		t.Error("kitty keyboard on for a terminal that did not answer")
	}
	t.Setenv("LIMONI_KITTY_KEYBOARD", "0")
	if p := (CapabilityProfile{}).WithReport(driver.TerminalReport{Answered: true, KittyKeyboard: true}); p.KittyKeyboard {
		t.Error("LIMONI_KITTY_KEYBOARD=0 ignored")
	}
	t.Setenv("LIMONI_NOTIFY", "")
	t.Setenv("LIMONI_POINTER", "")
	if p := (CapabilityProfile{}).WithReport(driver.TerminalReport{Answered: true, Name: "ghostty"}); p.Notify != NotifyOSC9 || !p.PointerShape {
		t.Errorf("XTVERSION ghostty: notify %d pointer %v", p.Notify, p.PointerShape)
	}
}
