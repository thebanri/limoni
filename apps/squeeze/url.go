package main

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"
)

// urlRE finds web addresses printed as text: agy underlines them but sends
// no OSC 8, and the terminal cannot find them itself under squeeze, which
// takes the mouse.
var urlRE = regexp.MustCompile("https?://[^\\s<>\"'`]+")

// linkAt is the web address under at in the racer's output: its OSC 8 link,
// or a URL printed as text, read across the rows it wraps over (a row that
// runs to the last column goes on in the next).
func (r *racer) linkAt(at spot) string {
	if c := r.cellAt(at.line, at.col); c != nil && c.Link.URL != "" {
		if strings.HasPrefix(c.Link.URL, "https://") || strings.HasPrefix(c.Link.URL, "http://") {
			return c.Link.URL
		}
		return ""
	}
	w := r.emu.Width()
	lines := r.emu.ScrollbackLen() + r.emu.Height()
	full := func(line int) bool {
		c := r.cellAt(line, w-1)
		return c != nil && c.Content != "" && c.Content != " "
	}
	first := at.line
	for first > 0 && at.line-first < 4 && full(first-1) {
		first--
	}
	var text []rune
	click := -1
	for line := first; line < lines && line <= at.line+4; line++ {
		row := make([]rune, w)
		for x := range w {
			row[x] = ' '
			if c := r.cellAt(line, x); c != nil && c.Content != "" {
				row[x], _ = utf8.DecodeRuneInString(c.Content)
			}
		}
		skip := 0
		if line > first { // a wrapped row's indent is not part of the address
			for skip < w && row[skip] == ' ' {
				skip++
			}
		}
		if line == at.line {
			if at.col < skip {
				return ""
			}
			click = len(text) + at.col - skip
		}
		text = append(text, row[skip:]...)
		if line >= at.line && !full(line) {
			break
		}
	}
	s := string(text)
	for _, m := range urlRE.FindAllStringIndex(s, -1) {
		url := trimURL(s[m[0]:m[1]])
		start := utf8.RuneCountInString(s[:m[0]])
		if click >= start && click < start+utf8.RuneCountInString(url) {
			return url
		}
	}
	return ""
}

// trimURL drops the punctuation a sentence puts after an address, and a
// closing bracket the address did not open.
func trimURL(url string) string {
	for url != "" {
		last := url[len(url)-1]
		switch {
		case strings.IndexByte(".,;:!?", last) >= 0:
		case last == ')' && strings.Count(url, "(") < strings.Count(url, ")"):
		case last == ']' && strings.Count(url, "[") < strings.Count(url, "]"):
		default:
			return url
		}
		url = url[:len(url)-1]
	}
	return url
}

// openURL opens a web address in the user's browser.
func openURL(url string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	cmd := exec.Command(name, url) // its output goes nowhere, not onto squeeze's screen
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
