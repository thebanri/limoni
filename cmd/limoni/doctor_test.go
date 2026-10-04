package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
)

// The report says what the terminal answered and what Limoni does with it,
// marking what the answers changed — the lines a bug report needs.
func TestDoctorReport(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("LIMONI_GRAPHICS", "")
	report := driver.TerminalReport{
		Answered: true, Name: "Konsole", Version: "26.08.1",
		SyncOutput: driver.ModeSet, GraphemeClusters: driver.ModeUnsupported,
		KittyKeyboard: true, KittyFlags: 1, Sixel: true,
		Repeat: driver.Yes, ClusterWidth: 2,
	}
	detected := terminal.CapabilityProfile{GraphicsProto: graphics.ProtocolHalfBlock}
	var out bytes.Buffer
	printDoctor(&out, report, 3*time.Millisecond, detected, detected.WithReport(report))
	got := out.String()
	for _, want := range []string{
		"name              Konsole 26.08.1",
		"answered          yes, in 3ms",
		"mode 2026 sync    supported, on",
		"mode 2027         not supported",
		"kitty keyboard    yes (flags 1)",
		"sixel             yes",
		"REP measured      works",
		"family emoji      2 columns (clusters as units)",
		"TERM              xterm-256color",
		"images                  half blocks  →  sixel  (changed)",
		"kitty keyboard          no       →  yes  (changed)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the report lacks %q:\n%s", want, got)
		}
	}

	// A terminal that said nothing.
	out.Reset()
	printDoctor(&out, driver.TerminalReport{}, doctorTimeout, detected, detected.WithReport(driver.TerminalReport{}))
	for _, want := range []string{"did not answer XTVERSION", "no reply to DA1", "REP measured      no cursor report", "family emoji      no cursor report"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the silent report lacks %q:\n%s", want, out.String())
		}
	}
}

// Off a terminal, doctor says so instead of hanging on questions nobody answers.
func TestDoctorNeedsATerminal(t *testing.T) {
	if isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		t.Skip("run from a terminal: doctor would probe it")
	}
	if err := run([]string{"doctor"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "needs a terminal") {
		t.Errorf("got %v", err)
	}
}

func TestDescribersCoverEveryValue(t *testing.T) {
	for _, m := range []driver.ModeState{driver.ModeUnsupported, driver.ModeSet, driver.ModeReset, driver.ModePermanentlySet, driver.ModePermanentlyReset, driver.ModeUnknown} {
		if describeMode(m) == "" {
			t.Errorf("mode %v has no description", m)
		}
	}
	for _, p := range []terminal.NotifyProtocol{terminal.NotifyNone, terminal.NotifyOSC9, terminal.NotifyOSC777, terminal.NotifyOSC99} {
		if notifyName(p) == "" {
			t.Errorf("notify %v has no name", p)
		}
	}
	for _, p := range []graphics.Protocol{graphics.ProtocolAuto, graphics.ProtocolKitty, graphics.ProtocolSixel, graphics.ProtocolIterm2, graphics.ProtocolHalfBlock} {
		if protocolName(p) == "" {
			t.Errorf("protocol %v has no name", p)
		}
	}
}
