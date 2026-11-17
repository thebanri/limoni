package main

import (
	"strings"
	"time"
)

// Agents ask for the user with a terminal notification: a bare BEL, OSC 9
// (iTerm2/Ghostty/WezTerm/foot), OSC 777;notify (VTE, urxvt), OSC 99 (kitty),
// or, in Gemini CLI, a window title carrying "✋" (Action Required). A BEL
// that ends an OSC sequence is not one, so the output is scanned as a stream
// that survives being split across reads.

type scanState uint8

const (
	scGround scanState = iota
	scEsc              // after ESC
	scCSI              // inside CSI, until its final byte
	scOSC              // inside OSC, collecting the payload
	scOSCEsc           // ESC inside OSC: ST ends it
	scStr              // DCS/APC/PM/SOS, which BEL does not end
	scStrEsc
)

const maxOSC = 1024 // longer payloads keep their first bytes; only the head is read

// scan feeds output to the signal detector, and returns it for the emulator
// without the non-ASCII bytes of OSC payloads: x/vt (x/ansi v0.11.9 too)
// takes 0x9C inside one as ST, so a UTF-8 title such as Claude Code's
// "✳ …" (E2 9C B3) ends early and the rest is printed onto the screen.
// The emulator does nothing with those bytes; the signals were read here.
// Called from the read loop only; the result is valid until the next call.
func (r *racer) scan(p []byte) []byte {
	out := r.clean[:0]
	for _, c := range p {
		if r.sc != scOSC || c < 0x80 {
			out = append(out, c)
		}
		r.step(c)
	}
	r.clean = out
	return out
}

func (r *racer) step(c byte) {
	switch r.sc {
	case scGround:
		switch {
		case c == 0x07:
			r.signal("")
		case c == 0x1b:
			r.sc = scEsc
		case c > 0x20 && c != 0x7f:
			r.drew = true
		}
	case scCSI:
		switch {
		case c >= 0x40 && c <= 0x7e:
			r.sc = scGround
			if c == 'h' || c == 'l' {
				r.decMode(c == 'h')
			}
		case c == 0x18, c == 0x1a:
			r.sc = scGround
		case c == 0x1b:
			r.sc = scEsc
		default:
			if len(r.csi) < 32 {
				r.csi = append(r.csi, c)
			}
		}
	case scEsc:
		switch c {
		case ']':
			r.sc, r.osc = scOSC, r.osc[:0]
		case '[':
			r.sc, r.csi = scCSI, r.csi[:0]
		case 'P', '_', '^', 'X':
			r.sc = scStr
		case 0x1b:
		default:
			r.sc = scGround
		}
	case scOSC:
		switch c {
		case 0x07:
			r.endOSC()
		case 0x1b:
			r.sc = scOSCEsc
		case 0x18, 0x1a: // CAN, SUB abort the sequence
			r.sc = scGround
		default:
			if len(r.osc) < maxOSC {
				r.osc = append(r.osc, c)
			}
		}
	case scOSCEsc:
		if c == '\\' {
			r.endOSC()
		} else { // ESC aborted the OSC and starts something else
			r.sc = scEsc
			r.step(c)
		}
	case scStr:
		if c == 0x1b {
			r.sc = scStrEsc
		}
	case scStrEsc:
		if c == '\\' {
			r.sc = scGround
		} else {
			r.sc = scStr
		}
	}
}

// mouseModes are the DEC modes that turn mouse reporting on: X10, normal,
// button-event and any-event tracking. x/vt keeps them to itself, so they
// are followed here, to know whether the wheel belongs to the agent.
var mouseModes = map[string]uint32{"9": 1, "1000": 2, "1002": 4, "1003": 8}

// decMode notes a CSI ? … h or l that sets or resets mouse modes.
func (r *racer) decMode(set bool) {
	params, ok := strings.CutPrefix(string(r.csi), "?")
	if !ok {
		return
	}
	m := r.mouse.Load()
	for _, p := range strings.Split(params, ";") {
		if bit := mouseModes[p]; set {
			m |= bit
		} else {
			m &^= bit
		}
	}
	r.mouse.Store(m)
}

func (r *racer) endOSC() {
	r.sc = scGround
	code, rest, _ := strings.Cut(string(r.osc), ";")
	switch code {
	case "9":
		switch {
		case rest == workingMark:
			r.working()
		case rest == promptMark:
			r.working()
			r.promptSent()
		case strings.HasPrefix(rest, subMark):
			r.subEvent(strings.TrimPrefix(rest, subMark))
		case !strings.HasPrefix(rest, "4;"): // 9;4 is a progress bar, not a notification
			r.signal(rest)
		}
	case "777":
		if body, ok := strings.CutPrefix(rest, "notify;"); ok {
			title, text, _ := strings.Cut(body, ";")
			if text != "" {
				text = title + ": " + text
			} else {
				text = title
			}
			r.signal(text)
		}
	case "99": // kitty: metadata ; payload
		_, text, _ := strings.Cut(rest, ";")
		r.signal(text)
	case "0", "2":
		if strings.Contains(rest, "✋") {
			r.signal("")
		}
	}
}

// signal marks the agent as needing the user until userInput. text is the
// agent's own notification, if it sent one.
func (r *racer) signal(text string) {
	text = strings.TrimSpace(text)
	if len(text) > 120 {
		text = strings.ToValidUTF8(text[:120], "") + "…"
	}
	r.askText.Store(&text)
	r.asked.Store(true)
}

// echo is how long after a key the agent's output counts as the key's echo
// rather than work: typing into a prompt redraws it, and is not the agent
// working. A var so tests can change it.
var echo = 300 * time.Millisecond

// userInput records that the user sent the agent a key, a paste or a click.
// Submitting (Enter) to an agent without hooks answers whatever it asked
// and sets it working; anything else is typing, whose echo is not work. A
// hooked agent says itself when it works.
func (r *racer) userInput(submit bool) {
	now := time.Now().UnixNano()
	if !submit || r.hooked {
		r.typed.Store(now)
		return
	}
	r.typed.Store(0)
	r.asked.Store(false)
	r.lastOut.Store(now)
}

// printed records output that printed something: activity, unless it is
// the echo of a key. It does not answer an ask: agents redraw while they
// wait (Claude Code's suggestions, agy's status line).
func (r *racer) printed() {
	if now := time.Now().UnixNano(); now-r.typed.Load() >= int64(echo) {
		r.lastOut.Store(now)
	}
}

// workingMark is what squeeze's hooks send, as an OSC 9, when the agent
// starts working again (see claudeHooks, writeAgyHooks).
const workingMark = "squeeze:working"

// working records the agent's own word that it is working.
func (r *racer) working() {
	r.asked.Store(false)
	r.lastOut.Store(time.Now().UnixNano())
}

// ask is the agent's own words for what it wants, or "".
func (r *racer) ask() string {
	if t := r.askText.Load(); t != nil && r.asked.Load() {
		return *t
	}
	return ""
}
