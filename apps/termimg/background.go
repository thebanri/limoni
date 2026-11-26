//go:build unix

package main

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// queryTerminal asks the terminal for its background colour (OSC 11), the
// colour transparent pixels are shown in, and where its cursor is (CSI 6n).
// DA1 follows them, which every terminal answers, so one without OSC 11
// costs no wait. It runs before the program's output goes anywhere, so
// neither question nor answer meets the program's own. It returns the
// colour, whether the terminal said, the cursor (0-based, the top left if
// the terminal did not say), and what else came in meanwhile (keys typed),
// for the program.
//
// It asks too whether the terminal has kitty's graphics protocol (kitty).
func queryTerminal(in, out *os.File) (bg color.RGBA, ok bool, at image.Point, kitty bool, rest []byte) {
	bg, ok, rest = queryBackground(in, out)
	at, rest = cursorReply(rest)
	kitty, rest = kittyAnswer(rest)
	return bg, ok, at, kitty, rest
}

// queryBackground sends the questions and waits for the DA1 answer.
func queryBackground(in, out *os.File) (color.RGBA, bool, []byte) {
	// The kitty query comes after the cursor's: a terminal that does not
	// know APC prints it, and it is erased.
	if _, err := out.WriteString("\x1b]11;?\x1b\\\x1b[6n\x1b7" + kittyQuery + "\x1b8\x1b[K\x1b[c"); err != nil {
		return color.RGBA{}, false, nil
	}
	var b []byte
	buf := make([]byte, 256)
	deadline := time.Now().Add(time.Second)
	for {
		if c, ok, rest, done := backgroundReply(b); done {
			return c, ok, rest
		}
		wait := time.Until(deadline)
		if wait <= 0 {
			return color.RGBA{}, false, b
		}
		fds := []unix.PollFd{{Fd: int32(in.Fd()), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, int(wait/time.Millisecond)+1)
		switch {
		case err == unix.EINTR || err == nil && n == 0:
			continue
		case err != nil:
			return color.RGBA{}, false, b
		}
		n, err = in.Read(buf)
		if err != nil {
			return color.RGBA{}, false, b
		}
		b = append(b, buf[:n]...)
	}
}

// backgroundReply looks in what the terminal sent for the answers to
// queryBackground: done once the DA1 answer is there, with the colour if an
// OSC 11 answer came before it, and the input without the answers.
func backgroundReply(b []byte) (c color.RGBA, ok bool, rest []byte, done bool) {
	i := bytes.Index(b, []byte("\x1b[?"))
	for i >= 0 {
		j := i + 3
		for j < len(b) && (b[j] >= '0' && b[j] <= '9' || b[j] == ';') {
			j++
		}
		if j < len(b) && b[j] == 'c' {
			rest = append(append(rest, b[:i]...), b[j+1:]...)
			done = true
			break
		}
		if k := bytes.Index(b[j:], []byte("\x1b[?")); k >= 0 {
			i = j + k
		} else {
			i = -1
		}
	}
	if !done {
		return color.RGBA{}, false, nil, false
	}
	start := bytes.Index(rest, []byte("\x1b]11;"))
	if start < 0 {
		return color.RGBA{}, false, rest, true
	}
	body := rest[start+5:]
	end, term := bytes.IndexByte(body, '\a'), 1
	if st := bytes.Index(body, []byte("\x1b\\")); st >= 0 && (end < 0 || st < end) {
		end, term = st, 2
	}
	if end < 0 {
		return color.RGBA{}, false, rest, true
	}
	c, ok = parseXColor(string(body[:end]))
	rest = append(rest[:start:start], body[end+term:]...)
	return c, ok, rest, true
}

// cursorReply takes the answer to CSI 6n, ESC [ row ; col R, out of b.
// shortcut: Shift+F3 in legacy encoding is ESC [ 1 ; 2 R too; typed in the
// moment before the answer, it would be taken for it.
func cursorReply(b []byte) (image.Point, []byte) {
	m := cprRe.FindSubmatchIndex(b)
	if m == nil {
		return image.Point{}, b
	}
	row, _ := strconv.Atoi(string(b[m[2]:m[3]]))
	col, _ := strconv.Atoi(string(b[m[4]:m[5]]))
	return image.Pt(max(col-1, 0), max(row-1, 0)), append(b[:m[0]:m[0]], b[m[1]:]...)
}

var cprRe = regexp.MustCompile(`\x1b\[(\d{1,4});(\d{1,4})R`)

// parseXColor reads rgb:R/G/B, each 1 to 4 hex digits, as terminals answer
// OSC 11.
func parseXColor(s string) (color.RGBA, bool) {
	spec, ok := strings.CutPrefix(s, "rgb:")
	parts := strings.Split(spec, "/")
	if !ok || len(parts) != 3 {
		return color.RGBA{}, false
	}
	var ch [3]uint8
	for i, p := range parts {
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil || len(p) == 0 || len(p) > 4 {
			return color.RGBA{}, false
		}
		ch[i] = uint8(v * 255 / (1<<(4*len(p)) - 1))
	}
	return color.RGBA{ch[0], ch[1], ch[2], 0xff}, true
}
