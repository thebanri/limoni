package zestapp

import "github.com/thebanri/limoni/widgets"

// detectLevel finds a line's severity without allocating. It understands the
// shapes logs usually come in:
//
//	{"level":"warn",...}   {"lvl":"ERROR"}   {"severity":"info"}   (JSON)
//	level=warn  lvl=error                                         (logfmt)
//	2026-09-19 12:00:01 ERROR disk full   [WARN] ...   W0919 ...  (plain)
//
// Only the first 200 bytes are examined: a level appears near the start, and
// a long stack trace or payload should not cost a full scan per line.
func detectLevel(line []byte) widgets.LogLevel {
	if len(line) > 200 {
		line = line[:200]
	}
	// Keyed forms first: they are unambiguous.
	for _, key := range [...]string{`"level":`, `"lvl":`, `"severity":`, `level=`, `lvl=`, `severity=`} {
		if i := indexASCII(line, key); i >= 0 {
			v := line[i+len(key):]
			for len(v) > 0 && (v[0] == ' ' || v[0] == '"') {
				v = v[1:]
			}
			if lv := levelWord(v); lv != widgets.LevelUnknown {
				return lv
			}
		}
	}
	// Otherwise the first word that is a level name, standing alone.
	for i := 0; i < len(line); i++ {
		if !isLetter(line[i]) || (i > 0 && isLetter(line[i-1])) {
			continue
		}
		if lv := levelWord(line[i:]); lv != widgets.LevelUnknown {
			return lv
		}
	}
	return widgets.LevelUnknown
}

// levelWord reads a level name at the start of b, if the word ends there.
func levelWord(b []byte) widgets.LogLevel {
	n := 0
	for n < len(b) && isLetter(b[n]) {
		n++
	}
	if n == 0 || n > 8 {
		return widgets.LevelUnknown
	}
	var w [8]byte
	for i := 0; i < n; i++ {
		c := b[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		w[i] = c
	}
	switch string(w[:n]) { // does not allocate: the conversion is only compared
	case "trace", "trc":
		return widgets.LevelTrace
	case "debug", "dbg":
		return widgets.LevelDebug
	case "info", "inf", "notice":
		return widgets.LevelInfo
	case "warn", "warning", "wrn":
		return widgets.LevelWarn
	case "error", "err", "eror":
		return widgets.LevelError
	case "fatal", "panic", "crit", "critical", "emerg", "alert":
		return widgets.LevelFatal
	}
	return widgets.LevelUnknown
}

func isLetter(c byte) bool { return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') }

// indexASCII finds sub in b, case-insensitively, without allocating.
func indexASCII(b []byte, sub string) int {
outer:
	for i := 0; i+len(sub) <= len(b); i++ {
		for j := 0; j < len(sub); j++ {
			c := b[i+j]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != sub[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}
