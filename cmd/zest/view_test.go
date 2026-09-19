package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thebanri/limoni/widgets"
)

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// Filtering finds every match, in order, across scan batches and while lines
// keep arriving, and a filter change mid-scan does not mix old results in.
func TestViewFiltersWhileLinesArrive(t *testing.T) {
	var s store
	var wakes atomic.Int32
	v := &view{src: &s, wake: func() { wakes.Add(1) }}
	add := func(from, to int) {
		for i := from; i < to; i++ {
			lvl := "info"
			if i%10 == 0 {
				lvl = "error"
			}
			s.add([]byte(`{"level":"` + lvl + `","msg":"request ` + strconv.Itoa(i) + `"}`))
		}
		v.kick()
	}
	add(0, 450_000) // more than two scan batches
	v.setFilter("", widgets.LevelError)
	add(450_000, 500_000)
	waitUntil(t, func() bool { _, _, done := v.progress(); return done && v.Len() == 50_000 })
	for i := 0; i < v.Len(); i += 997 {
		if want := "request " + strconv.Itoa(i*10) + `"`; !strings.Contains(v.Line(i), want) {
			t.Fatalf("match %d is %q, want it to contain %s", i, v.Line(i), want)
		}
	}
	if wakes.Load() == 0 {
		t.Fatal("scanning never woke the UI")
	}

	v.setFilter("REQUEST 4999", widgets.LevelUnknown) // case-insensitive
	waitUntil(t, func() bool { _, _, done := v.progress(); return done })
	if n := v.Len(); n != 111 { // 4999, 49990..49999 and 499900..499999
		t.Fatalf("query matched %d lines, want 111", n)
	}

	v.setFilter("", widgets.LevelUnknown)
	if v.Len() != s.Len() {
		t.Fatalf("clearing the filter shows %d of %d lines", v.Len(), s.Len())
	}
}

func TestReaderHandlesLongLinesCRLFAndAPartialLastLine(t *testing.T) {
	long := strings.Repeat("z", 600<<10) // longer than the reader's buffer
	in := "first\r\n" + long + "\nlast without newline"
	var s store
	if err := readLines(context.Background(), strings.NewReader(in), &s, func() {}, false); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 3 || s.Line(0) != "first" || s.Line(1) != long || s.Line(2) != "last without newline" {
		t.Fatalf("got %d lines: %.20q / %d bytes / %q", s.Len(), s.Line(0), len(s.Line(1)), s.Line(s.Len()-1))
	}
}

// Following a file picks up appended lines, and a truncated file (log
// rotation by copytruncate) is read again from the start.
func TestReaderFollowsAndSurvivesTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var s store
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go readLines(ctx, f, &s, func() {}, true)
	waitUntil(t, func() bool { return s.Len() == 2 })

	w, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	w.WriteString("thr")
	time.Sleep(3 * pollInterval)
	if s.Len() != 2 {
		t.Fatalf("a partial line was added before its newline: %d lines", s.Len())
	}
	w.WriteString("ee\n")
	w.Close()
	waitUntil(t, func() bool { return s.Len() == 3 })
	if s.Line(2) != "three" {
		t.Fatalf("appended line %q", s.Line(2))
	}

	if err := os.WriteFile(path, []byte("rotated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool { return s.Len() == 4 })
	if s.Line(3) != "rotated" {
		t.Fatalf("after truncation got %q", s.Line(3))
	}
}
