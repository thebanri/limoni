package driver

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ProbeQueries asks the terminal what it is and what it supports:
//
//   - XTVERSION (CSI > 0 q): its name and version, even inside tmux, over SSH
//     and where TERM and TERM_PROGRAM say nothing useful.
//   - DECRQM for mode 2026 (synchronized output) and 2027 (grapheme clusters).
//   - The Kitty keyboard flags (CSI ? u).
//   - Two measurements, made by writing and asking where the cursor went
//     (CSI 6 n), because names and modes do not say everything: whether REP
//     repeats a glyph, and how wide the terminal really draws a ZWJ family
//     emoji. Many terminals draw clusters as units without implementing mode
//     2027; others advance per code point. The cells are erased and the
//     cursor restored before anything else is drawn.
//   - DA1 (CSI c), last. Every terminal answers DA1 and answers in order, so
//     its reply means every query before it has been answered or never will be.
//
// A terminal that does not know a query ignores it. The replies arrive as
// input and are consumed by the backend's event loop (see TerminalReport); they
// never reach the application as key presses.
const ProbeQueries = "\x1b[>0q" + "\x1b[?2026$p" + "\x1b[?2027$p" + "\x1b[?u" +
	"\x1b7" + // save the cursor
	"\r \x1b[1b\x1b[6n" + // a space, REP once: column 3 if REP works, 2 if not
	"\r" + probeCluster + "\x1b[6n" + // column 3 if the cluster is drawn two wide
	"\r\x1b[8X" + // erase what was written
	"\x1b8" + // restore the cursor
	"\x1b[c"

// probeCluster is the cluster measured: man, ZWJ, woman, ZWJ, girl. A
// terminal that measures code points draws it six columns wide.
const probeCluster = "\U0001F468\u200D\U0001F469\u200D\U0001F467"

// probeEnabled reports whether setup should send ProbeQueries. LIMONI_PROBE=0
// turns the handshake off, for a terminal that misbehaves on one of them.
func probeEnabled() bool { return os.Getenv("LIMONI_PROBE") != "0" }

// withProbe appends ProbeQueries to a setup sequence, unless the handshake is
// off, and records that they were sent.
func (c *replyCollector) withProbe(setup string) string {
	if !probeEnabled() {
		return setup
	}
	c.markSent()
	return setup + ProbeQueries
}

// probeDrainTimeout bounds how long Close waits for replies still in flight.
// A reply that arrives after the terminal is restored lands in the shell as
// garbage like "^[[?62;22c", so an application that exits immediately after
// starting waits this long at most for the DA1 sentinel.
const probeDrainTimeout = 150 * time.Millisecond

// ModeState is a DECRPM answer about one mode.
type ModeState uint8

const (
	// ModeUnknown: the terminal has not answered (yet, or at all).
	ModeUnknown ModeState = iota
	// ModeUnsupported: the terminal answered that it does not know the mode.
	ModeUnsupported
	ModeSet
	ModeReset
	ModePermanentlySet
	ModePermanentlyReset
)

// Recognized reports whether the terminal knows the mode at all.
func (m ModeState) Recognized() bool { return m >= ModeSet }

// Enabled reports whether the mode is on right now.
func (m ModeState) Enabled() bool { return m == ModeSet || m == ModePermanentlySet }

func modeState(setting int) ModeState {
	switch setting {
	case 0:
		return ModeUnsupported
	case 1:
		return ModeSet
	case 2:
		return ModeReset
	case 3:
		return ModePermanentlySet
	case 4:
		return ModePermanentlyReset
	}
	return ModeUnknown
}

// TerminalReport is what the terminal said about itself in answer to
// ProbeQueries. Fields stay at their zero value until the matching reply arrives.
type TerminalReport struct {
	// Answered is true once the DA1 sentinel arrived: every other field is
	// final from then on.
	Answered bool
	// Name and Version come from XTVERSION, split at the first '(' or space:
	// "kitty(0.39.1)" is kitty / 0.39.1, "tmux 3.5a" is tmux / 3.5a.
	Name, Version string
	// SyncOutput and GraphemeClusters are the DECRPM answers for modes 2026
	// and 2027.
	SyncOutput, GraphemeClusters ModeState
	// KittyKeyboard is true if the terminal answered the Kitty keyboard query,
	// with its current flags in KittyFlags.
	KittyKeyboard bool
	KittyFlags    int
	// Sixel is DA1 attribute 4.
	Sixel bool
	// Repeat is the measured answer to whether REP works.
	Repeat Answer
	// ClusterWidth is the measured width of a ZWJ family emoji: 2 where the
	// terminal draws grapheme clusters as units, more where it measures code
	// points, 0 if the terminal did not report the cursor.
	ClusterWidth int
}

// Answer is a measured yes or no, or no measurement.
type Answer uint8

const (
	Unmeasured Answer = iota
	Yes
	No
)

// replyCollector accumulates ReplyEvents into a TerminalReport. It is shared
// by every backend so that none of them passes replies on as input.
type replyCollector struct {
	mu       sync.Mutex
	report   TerminalReport
	cursors  int // cursor reports seen: the first measures REP, the second the cluster
	sent     atomic.Bool
	version  atomic.Uint64
	answered chan struct{}
	once     sync.Once
}

func (c *replyCollector) init() {
	c.once.Do(func() { c.answered = make(chan struct{}) })
}

// markSent records that ProbeQueries went out, which is what makes waiting for
// the answer at Close worthwhile.
func (c *replyCollector) markSent() {
	c.init()
	c.sent.Store(true)
}

// record folds one reply into the report. It returns true for a reply event,
// which the caller then does not forward to the application.
func (c *replyCollector) record(ev Event) bool {
	if ev.Type != EventReply {
		return false
	}
	c.init()
	c.mu.Lock()
	r := &c.report
	switch ev.Reply.Kind {
	case ReplyPrimaryDA:
		r.Sixel = ev.Reply.Attributes&(1<<4) != 0
		if !r.Answered {
			r.Answered = true
			close(c.answered)
		}
	case ReplyMode:
		switch ev.Reply.Mode {
		case 2026:
			r.SyncOutput = modeState(ev.Reply.Setting)
		case 2027:
			r.GraphemeClusters = modeState(ev.Reply.Setting)
		}
	case ReplyVersion:
		r.Name, r.Version = splitVersion(ev.Reply.Version)
	case ReplyKittyKeyboard:
		r.KittyKeyboard = true
		r.KittyFlags = ev.Reply.Flags
	case ReplyCursor:
		// Only the two reports the probe asked for, before DA1, are
		// measurements. Anything later is dropped, as it always was.
		if r.Answered || !c.sent.Load() {
			break
		}
		c.cursors++
		switch c.cursors {
		case 1:
			r.Repeat = No
			if ev.Reply.Col == 3 {
				r.Repeat = Yes
			}
		case 2:
			if ev.Reply.Col > 1 {
				r.ClusterWidth = ev.Reply.Col - 1
			}
		}
	}
	c.version.Add(1)
	c.mu.Unlock()
	return true
}

// snapshot returns the report and a counter that changes whenever it does.
func (c *replyCollector) snapshot() (TerminalReport, uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.report, c.version.Load()
}

// drain waits, bounded, for the DA1 sentinel if queries were sent and the
// event loop that reads the answers is running.
func (c *replyCollector) drain(loopRunning bool) {
	if !c.sent.Load() || !loopRunning {
		return
	}
	c.init()
	select {
	case <-c.answered:
	case <-time.After(probeDrainTimeout):
	}
}

// splitVersion separates an XTVERSION string into a name and a version.
func splitVersion(s string) (name, version string) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "( "); i >= 0 {
		name = s[:i]
		version = strings.Trim(s[i:], "() ")
		return name, version
	}
	return s, ""
}

// TerminalReport returns what the terminal has said about itself so far, and
// a counter that changes whenever the report does, so a caller can check
// cheaply on every frame whether anything new arrived.
func (b *Backend) TerminalReport() (TerminalReport, uint64) {
	return b.replies.snapshot()
}

// TerminalReportVersion returns the counter TerminalReport returns, without
// taking the lock: an atomic load, cheap enough for every frame.
func (b *Backend) TerminalReportVersion() uint64 {
	return b.replies.version.Load()
}

// ProbeAnswered returns a channel that is closed when the DA1 sentinel
// arrives, from which point TerminalReport is final. A loop that draws only
// when something happens selects on it to draw once more: answers that land
// after the first frame change how that frame should have been encoded, and
// without input nothing else would draw it again. If the probe was not sent,
// or the terminal never answers, the channel is never closed.
func (b *Backend) ProbeAnswered() <-chan struct{} {
	b.replies.init()
	return b.replies.answered
}
