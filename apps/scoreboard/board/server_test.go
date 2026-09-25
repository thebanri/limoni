package board

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

// Vercel's Go preset runs the whole server, and its rewrites hand the
// server the rewritten path: /api/healthz for /healthz. That answered 404
// on the first deployment. Both paths answer now.
func TestTheServerAnswersOnApiPathsToo(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	for _, p := range []string{"/healthz", "/api/healthz"} {
		if w, _ := f.do("GET", p, ""); w.Code != 200 || w.Body.String() != "ok\n" {
			t.Errorf("GET %s: %d %q", p, w.Code, w.Body.String())
		}
	}
	if w, a := f.do("POST", "/api/scores", `{"name":"Ece","won":false,"secs":30,"kills":2}`); w.Code != 200 || a.Rank != 1 {
		t.Errorf("POST /api/scores: %d %+v", w.Code, a)
	}
	if _, a := f.do("GET", "/scores", ""); len(a.Board) != 1 || a.Board[0].Name != "Ece" {
		t.Errorf("GET /scores after POST /api/scores: %+v", a)
	}
	if w, _ := f.do("GET", "/", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "leaderboard") {
		t.Errorf("GET /: %d %q", w.Code, w.Body.String())
	}
	if w, _ := f.do("GET", "/nothing", ""); w.Code != http.StatusNotFound {
		t.Errorf("GET /nothing: %d", w.Code)
	}
}

// With the database out of reach the server still starts and answers
// /healthz; the board says it is not available, and the next request
// tries again.
func TestALazyDatabaseOutOfReach(t *testing.T) {
	lazy := NewLazyPG("postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	f := newFixture(t, lazy, epoch)
	if w, _ := f.do("GET", "/api/healthz", ""); w.Code != 200 {
		t.Errorf("healthz without a database: %d", w.Code)
	}
	for i := 0; i < 2; i++ {
		if w, a := f.do("GET", "/scores", ""); w.Code != http.StatusServiceUnavailable || a.Error == "" {
			t.Errorf("scores without a database, try %d: %d %+v", i, w.Code, a)
		}
	}
	if lazy.pg != nil {
		t.Error("a failed open was kept")
	}
}

func TestTheServerOverALazyDatabase(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	first, err := OpenPG(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := first.pool.Exec(context.Background(), `TRUNCATE runs; TRUNCATE posts`); err != nil {
		t.Fatal(err)
	}
	lazy := NewLazyPG(url)
	f := newFixture(t, lazy, epoch)
	if lazy.pg != nil {
		t.Fatal("opened before the first request")
	}
	w, a := f.do("POST", "/api/scores", `{"name":"Arslan","won":true,"secs":245,"shots":60,"hits":41,"kills":14,"lemons":10}`)
	if w.Code != 200 || a.Rank != 1 || a.Board[0].Score != 8666 {
		t.Fatalf("POST: %d %s", w.Code, w.Body.String())
	}
	if _, a := f.do("GET", "/scores", ""); len(a.Board) != 1 || a.Board[0].Name != "Arslan" {
		t.Errorf("GET: %+v", a)
	}
	lazy.pg.Close()
}
