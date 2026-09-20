package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// doctorTimeout bounds the wait for the terminal's answers. A local terminal
// answers in a millisecond or two; SSH adds a round trip.
const doctorTimeout = 2 * time.Second

// runDoctor asks the terminal what it is and prints what Limoni makes of it:
// the replies to the capability handshake, the environment it would otherwise
// have guessed from, and the profile an application would render with. It is
// what a bug report about rendering should include.
func runDoctor(out io.Writer) error {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return errors.New("doctor needs a terminal on stdin and stdout")
	}

	detected := terminal.DetectCapabilities()

	// One inline row: no alternate screen, nothing left behind but a blank line.
	b := driver.NewBackend(os.Stdin, os.Stdout)
	b.SetInline(1)
	if err := b.Setup(); err != nil {
		return err
	}
	b.StartEventLoop()
	start := time.Now()
	var report driver.TerminalReport
	for time.Since(start) < doctorTimeout {
		if report, _ = b.TerminalReport(); report.Answered {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	elapsed := time.Since(start)
	if err := b.Close(); err != nil {
		return err
	}

	final := detected.WithReport(report)

	fmt.Fprintln(out, "Terminal")
	switch {
	case report.Name != "":
		fmt.Fprintf(out, "  name              %s %s\n", report.Name, report.Version)
	default:
		fmt.Fprintln(out, "  name              (did not answer XTVERSION)")
	}
	if report.Answered {
		fmt.Fprintf(out, "  answered          yes, in %s\n", elapsed.Round(time.Millisecond))
	} else {
		fmt.Fprintf(out, "  answered          no reply to DA1 within %s\n", doctorTimeout)
	}
	fmt.Fprintf(out, "  mode 2026 sync    %s\n", describeMode(report.SyncOutput))
	fmt.Fprintf(out, "  mode 2027         %s\n", describeMode(report.GraphemeClusters))
	if report.KittyKeyboard {
		fmt.Fprintf(out, "  kitty keyboard    yes (flags %d)\n", report.KittyFlags)
	} else {
		fmt.Fprintln(out, "  kitty keyboard    no")
	}
	fmt.Fprintf(out, "  sixel             %s\n", yesNo(report.Sixel))
	switch report.Repeat {
	case driver.Yes:
		fmt.Fprintln(out, "  REP measured      works")
	case driver.No:
		fmt.Fprintln(out, "  REP measured      does not work")
	default:
		fmt.Fprintln(out, "  REP measured      no cursor report")
	}
	if report.ClusterWidth > 0 {
		fmt.Fprintf(out, "  family emoji      %d columns (%s)\n", report.ClusterWidth, map[bool]string{true: "clusters as units", false: "per code point"}[report.ClusterWidth == 2])
	} else {
		fmt.Fprintln(out, "  family emoji      no cursor report")
	}

	fmt.Fprintln(out, "\nEnvironment")
	for _, name := range []string{"TERM", "COLORTERM", "TERM_PROGRAM", "TMUX", "SSH_TTY", "LIMONI_REP", "LIMONI_HYPERLINKS", "LIMONI_NO_SYNC", "LIMONI_PROBE", "LIMONI_GRAPHEME"} {
		if v, ok := os.LookupEnv(name); ok {
			fmt.Fprintf(out, "  %-16s  %s\n", name, v)
		}
	}

	fmt.Fprintln(out, "\nLimoni will use          guessed  →  after handshake")
	row := func(name string, guess, got bool) {
		mark := ""
		if guess != got {
			mark = "  (changed)"
		}
		fmt.Fprintf(out, "  %-22s  %-7s  →  %s%s\n", name, yesNo(guess), yesNo(got), mark)
	}
	row("truecolor", detected.TrueColor, final.TrueColor)
	row("256 colours", detected.Colors256, final.Colors256)
	row("synchronized output", detected.SyncOutput, final.SyncOutput)
	row("REP (repeat glyph)", detected.RepeatChar, final.RepeatChar)
	row("OSC 8 hyperlinks", detected.Hyperlinks, final.Hyperlinks)
	row("cluster widths (2027)", detected.ClusterWidths, final.ClusterWidths)
	return nil
}

func describeMode(m driver.ModeState) string {
	switch m {
	case driver.ModeUnsupported:
		return "not supported"
	case driver.ModeSet:
		return "supported, on"
	case driver.ModeReset:
		return "supported, off"
	case driver.ModePermanentlySet:
		return "always on"
	case driver.ModePermanentlyReset:
		return "always off"
	}
	return "no answer"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// isTerminal reports whether f is a character device, which is what a
// terminal is on every platform Limoni runs on.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
