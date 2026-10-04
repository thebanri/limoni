package terminal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

// A cast is a recording of what the terminal was sent, in asciinema's
// asciicast v2 format: a JSON header line, then one [seconds, "o", text]
// line per frame. asciinema plays it, asciinema.org hosts it, and agg turns
// it into a GIF — the README picture of an application, made by running it.
//
//	LIMONI_CAST=demo.cast ./myapp
//	agg demo.cast demo.gif
//
// Only what Limoni draws is recorded, not the setup and capability queries
// around it.

type castRecorder struct {
	mu     sync.Mutex
	w      *bufio.Writer
	closer io.Closer
	start  time.Time
	width  uint16
	height uint16
	err    error
}

// castOnce lets the first terminal of a process take LIMONI_CAST: an SSH
// server runs one per session, and they cannot share the file.
var castOnce sync.Once

func newCastRecorder(w io.Writer, closer io.Closer, width, height uint16) (*castRecorder, error) {
	c := &castRecorder{w: bufio.NewWriter(w), closer: closer, start: time.Now(), width: width, height: height}
	header := map[string]any{
		"version":   2,
		"width":     width,
		"height":    height,
		"timestamp": c.start.Unix(),
		"env":       map[string]string{"TERM": os.Getenv("TERM"), "SHELL": os.Getenv("SHELL")},
	}
	line, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	if _, err := c.w.Write(append(line, '\n')); err != nil {
		return nil, err
	}
	return c, nil
}

// event appends one event line; code is "o" for output, "r" for a resize.
func (c *castRecorder) event(code string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return
	}
	text, err := json.Marshal(string(data))
	if err != nil {
		c.err = err
		return
	}
	elapsed := strconv.FormatFloat(time.Since(c.start).Seconds(), 'f', 6, 64)
	if _, c.err = fmt.Fprintf(c.w, "[%s, %q, %s]\n", elapsed, code, text); c.err == nil {
		// Each frame reaches the file at once, so an application that is
		// killed rather than closed still leaves a recording that plays.
		c.err = c.w.Flush()
	}
}

func (c *castRecorder) output(p []byte) { c.event("o", p) }

func (c *castRecorder) resize(width, height uint16) {
	if width == c.width && height == c.height {
		return
	}
	c.width, c.height = width, height
	c.event("r", []byte(strconv.Itoa(int(width))+"x"+strconv.Itoa(int(height))))
}

func (c *castRecorder) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.w.Flush()
	if c.closer != nil {
		if cerr := c.closer.Close(); err == nil {
			err = cerr
		}
	}
	if c.err != nil {
		return c.err
	}
	return err
}

// RecordCast writes everything the terminal draws from now on to w as an
// asciicast v2 recording, until Close. It is what LIMONI_CAST=path turns on
// for an application without changing its code.
func (t *Terminal) RecordCast(w io.Writer) error {
	c, err := newCastRecorder(w, nil, t.front.Area.Width, t.front.Area.Height)
	if err != nil {
		return err
	}
	t.cast = c
	return nil
}

// recordCastFromEnv honours LIMONI_CAST for the first terminal a process
// creates.
func (t *Terminal) recordCastFromEnv() error {
	path := os.Getenv("LIMONI_CAST")
	if path == "" {
		return nil
	}
	var err error
	castOnce.Do(func() {
		var f *os.File
		if f, err = os.Create(path); err != nil {
			err = fmt.Errorf("LIMONI_CAST: %w", err)
			return
		}
		var c *castRecorder
		if c, err = newCastRecorder(f, f, t.front.Area.Width, t.front.Area.Height); err != nil {
			f.Close()
			return
		}
		t.cast = c
	})
	return err
}
