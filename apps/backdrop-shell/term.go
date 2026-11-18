package main

import (
	"bytes"
	"os"
	"strconv"
	"time"

	"github.com/thebanri/limoni/core/cell"
)

// terminalColors asks the terminal for its default foreground and
// background (OSC 10 and 11), so the scene can fade into the real background
// and the shell can be told the truth when it asks. DA1 goes last: every
// terminal answers it, so the wait ends there instead of at the timeout in a
// terminal that does not answer the colour queries.
//
// It must run with the terminal in raw mode, before anything else reads the
// input. Whatever else arrived in the meantime — keys typed while the
// wrapper started — is returned, to go to the shell.
func terminalColors(in *os.File, out *os.File) (fg, bg cell.Color, rest []byte) {
	if _, err := out.WriteString("\x1b]10;?\x1b\\\x1b]11;?\x1b\\\x1b[c"); err != nil {
		return 0, 0, nil
	}
	var got []byte
	buf := make([]byte, 256)
	deadline := time.Now().Add(300 * time.Millisecond)
	for {
		wait := time.Until(deadline)
		if wait <= 0 {
			break
		}
		if !waitReadable(in, wait) {
			break
		}
		r, err := in.Read(buf)
		if err != nil || r <= 0 {
			break
		}
		got = append(got, buf[:r]...)
		if da1End(got) >= 0 {
			break
		}
	}
	fg, got = takeColor(got, "10")
	bg, got = takeColor(got, "11")
	if end := da1End(got); end >= 0 {
		start := bytes.LastIndex(got[:end], []byte("\x1b[?"))
		got = append(got[:start], got[end+1:]...)
	}
	return fg, bg, got
}

// da1End is the index of the final 'c' of a DA1 answer in b, or -1.
func da1End(b []byte) int {
	i := bytes.Index(b, []byte("\x1b[?"))
	for i >= 0 {
		for j := i + 3; j < len(b); j++ {
			c := b[j]
			if c == 'c' {
				return j
			}
			if (c < '0' || c > '9') && c != ';' {
				break
			}
		}
		next := bytes.Index(b[i+1:], []byte("\x1b[?"))
		if next < 0 {
			return -1
		}
		i += 1 + next
	}
	return -1
}

// takeColor finds the answer to OSC <num> in b — "ESC ] num ; rgb:RRRR/GGGG/BBBB"
// ended by BEL or ST — parses it, and returns b without it.
func takeColor(b []byte, num string) (cell.Color, []byte) {
	prefix := []byte("\x1b]" + num + ";rgb:")
	i := bytes.Index(b, prefix)
	if i < 0 {
		return 0, b
	}
	body := b[i+len(prefix):]
	end := bytes.IndexAny(body, "\x07\x1b")
	if end < 0 {
		return 0, b
	}
	parts := bytes.Split(body[:end], []byte("/"))
	cut := i + len(prefix) + end + 1
	if body[end] == 0x1b && cut < len(b) && b[cut] == '\\' {
		cut++
	}
	rest := append(b[:i:i], b[cut:]...)
	if len(parts) != 3 {
		return 0, rest
	}
	var ch [3]uint8
	for k, p := range parts {
		v, err := strconv.ParseUint(string(p), 16, 32)
		if err != nil || len(p) == 0 || len(p) > 4 {
			return 0, rest
		}
		// Scale 1 to 4 hex digits to 8 bits.
		bits := uint(len(p) * 4)
		ch[k] = uint8(v * 255 / (1<<bits - 1))
	}
	return cell.NewColorRGB(ch[0], ch[1], ch[2]), rest
}
