//go:build unix || windows

package driver

import (
	"io"
	"time"
)

// escTimeout is how long a lone ESC waits for the rest of an escape sequence
// before it is taken as the Esc key. A terminal sends a sequence in one write;
// a gap this long is a person pressing Esc.
const escTimeout = 25 * time.Millisecond

// readChunks reads r on its own goroutine and hands over what each read
// returned, until r fails (the channel is then closed) or done is closed.
func readChunks(r io.Reader, size int, done <-chan struct{}) <-chan []byte {
	out := make(chan []byte, 32)
	go func() {
		defer close(out)
		buf := make([]byte, size)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				select {
				case out <- chunk:
				case <-done:
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return out
}

// parseInput turns the bytes arriving on input into events on b.events: keys,
// mouse reports, pastes and focus changes, with the capability probe's
// replies taken aside. A lone ESC becomes the Esc key only after escTimeout
// without a following byte, so a sequence split across reads still parses as
// one key. When tick is not nil, poll is asked on every tick whether the size
// changed, and a change is sent as an EventResize.
//
// It returns when input is closed or b.done is; every send gives way to
// b.done, so closing the backend never leaves it blocked on a full channel.
// Every platform's event loop runs this one, so one set of tests covers them.
func (b *Backend) parseInput(input <-chan []byte, tick <-chan time.Time, poll func() (w, h uint16, changed bool)) {
	var pending []byte
	var escTimer *time.Timer
	var escFired <-chan time.Time
	defer func() {
		if escTimer != nil {
			escTimer.Stop()
		}
	}()
	send := func(ev Event) bool {
		select {
		case b.events <- ev:
			return true
		case <-b.done:
			return false
		}
	}
	for {
		select {
		case <-b.done:
			return

		case <-tick:
			if w, h, changed := poll(); changed {
				if !send(Event{Type: EventResize, Resize: ResizeEvent{Width: w, Height: h}}) {
					return
				}
			}

		case chunk, ok := <-input:
			if !ok {
				return
			}
			pending = append(pending, chunk...)
			// A byte arrived: whatever ESC was waiting is part of a sequence.
			if escTimer != nil {
				escTimer.Stop()
				escTimer, escFired = nil, nil
			}
			for len(pending) > 0 {
				ev, consumed := ParseBracketedPaste(pending)
				if consumed == 0 {
					ev, consumed = ParseEvent(pending)
				}
				if consumed == 0 {
					break // an incomplete sequence: wait for the rest
				}
				if ev.Type != EventNone && !b.replies.record(ev) && !send(ev) {
					return
				}
				pending = pending[consumed:]
			}
			if len(pending) == 1 && pending[0] == '\x1b' {
				escTimer = time.NewTimer(escTimeout)
				escFired = escTimer.C
			}

		case <-escFired:
			if len(pending) == 1 && pending[0] == '\x1b' {
				if !send(Event{Type: EventKey, Key: KeyEvent{Type: KeyEsc}}) {
					return
				}
				pending = pending[:0]
			}
			escTimer, escFired = nil, nil
		}
	}
}
