package zestapp

import (
	"strconv"
	"strings"
	"testing"

	"github.com/thebanri/limoni/widgets"
)

func TestDetectLevel(t *testing.T) {
	for line, want := range map[string]widgets.LogLevel{
		`{"time":"12:00","level":"warn","msg":"slow"}`:      widgets.LevelWarn,
		`{"lvl":"ERROR","msg":"x"}`:                         widgets.LevelError,
		`{"severity": "info"}`:                              widgets.LevelInfo,
		`ts=1 level=debug msg="cache miss"`:                 widgets.LevelDebug,
		`2026-09-19 12:00:01 ERROR disk full`:               widgets.LevelError,
		`[WARN] retrying`:                                   widgets.LevelWarn,
		`panic: runtime error: index out of range`:          widgets.LevelFatal,
		`GET /api/errors 200`:                               widgets.LevelUnknown, // "errors" is not a level word
		`informational text`:                                widgets.LevelUnknown,
		`{"msg":"the level is fine","level":"info"}`:        widgets.LevelInfo,
		strings.Repeat("x", 300) + " ERROR after 300 bytes": widgets.LevelUnknown,
	} {
		if got := detectLevel([]byte(line)); got != want {
			t.Errorf("detectLevel(%.50q) = %v, want %v", line, got, want)
		}
	}
	b := []byte(`2026-09-19 12:00:01 [warning] {"level":"error"}`)
	if a := testing.AllocsPerRun(100, func() { detectLevel(b) }); a != 0 {
		t.Errorf("detectLevel allocates %.0f times", a)
	}
}

// Lines are stored once and handed out without copying, across chunk
// boundaries and for a line bigger than a chunk.
func TestStoreKeepsLinesAcrossChunks(t *testing.T) {
	var s store
	want := []string{}
	for i := 0; i < 50000; i++ {
		line := "line " + strconv.Itoa(i) + " " + strings.Repeat("x", i%97)
		want = append(want, line)
		s.add([]byte(line))
	}
	huge := strings.Repeat("y", chunkSize+10)
	s.add([]byte(huge))
	want = append(want, huge)
	s.add(nil)
	want = append(want, "")

	if s.Len() != len(want) {
		t.Fatalf("Len %d, want %d", s.Len(), len(want))
	}
	for i, w := range want {
		if got := s.Line(i); got != w {
			t.Fatalf("line %d: got %.40q, want %.40q", i, got, w)
		}
	}
	if len(s.chunks) < 3 {
		t.Fatalf("expected the text to span chunks, got %d", len(s.chunks))
	}
	if a := testing.AllocsPerRun(100, func() { _ = s.Line(12345) }); a != 0 {
		t.Errorf("Line allocates %.0f times", a)
	}
}
