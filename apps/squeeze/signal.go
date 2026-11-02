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
	scOSC              // inside OSC, collecting the payload
	scOSCEsc           // ESC inside OSC: ST ends it
	scStr              // DCS/APC/PM/SOS, which BEL does not end
	scStrEsc
)

const maxOSC = 1024 // longer payloads keep their first bytes; only the head is read

// scan feeds output to the signal detector. Called from the read loop only.
func (r *racer) scan(p []byte) {
	for _, c := range p {
		r.step(c)
	}
}

func (r *racer) step(c byte) {
	switch r.sc {
	case scGround:
		switch c {
		case 0x07:
			r.signal("")
		case 0x1b:
			r.sc = scEsc
		}
	case scEsc:
		switch c {
		case ']':
			r.sc, r.osc = scOSC, r.osc[:0]
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

func (r *racer) endOSC() {
	r.sc = scGround
	code, rest, _ := strings.Cut(string(r.osc), ";")
	switch code {
	case "9":
		if !strings.HasPrefix(rest, "4;") { // 9;4 is a progress bar, not a notification
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

// userInput records that the user sent the agent keys or a paste: whatever it
// asked for is being answered.
func (r *racer) userInput() {
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
