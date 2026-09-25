package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thebanri/limoni/apps/scoreboard/board"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestWhereTheBoardIsKept(t *testing.T) {
	dir := t.TempDir()
	for _, c := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"postgres", map[string]string{"DATABASE_URL": "postgres://x", "VERCEL": "1"}, "*board.LazyPG"},
		{"vercel without a database", map[string]string{"VERCEL": "1", "DATA_DIR": dir}, "main.noDatabase"},
		{"a disk", map[string]string{"DATA_DIR": dir}, "*board.Mem"},
		{"railway's volume", map[string]string{"RAILWAY_VOLUME_MOUNT_PATH": dir}, "*board.Mem"},
		{"memory", map[string]string{}, "*board.Mem"},
	} {
		s, where, err := storeFor(env(c.env))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := typeName(s); got != c.want {
			t.Errorf("%s: %s (%s), want %s", c.name, got, where, c.want)
		}
	}
	if _, where, _ := storeFor(env(map[string]string{"DATA_DIR": dir})); !strings.Contains(where, filepath.Join(dir, "scores.json")) {
		t.Errorf("a disk: %q", where)
	}
}

// On Vercel with no database the board says so, rather than keeping runs
// in memory, where each copy of the server would have its own and lose it.
func TestVercelWithoutADatabaseSaysSo(t *testing.T) {
	s, _, _ := storeFor(env(map[string]string{"VERCEL": "1"}))
	h := board.New(board.Config{Store: s})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/scores", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("GET /scores: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/scores", strings.NewReader(`{"name":"x","won":false,"secs":10}`)))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("POST /scores: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Errorf("GET /healthz: %d", w.Code)
	}
}

func typeName(v any) string {
	switch v.(type) {
	case *board.LazyPG:
		return "*board.LazyPG"
	case noDatabase:
		return "main.noDatabase"
	case *board.Mem:
		return "*board.Mem"
	}
	return "?"
}
