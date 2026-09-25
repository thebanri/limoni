package fn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The function as Vercel runs it: DATABASE_URL set, called on its own
// path, twice as two players would, and the second finds the first's run.
func TestTheFunctionServesTheBoard(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	_, _ = conn.Exec(context.Background(), `TRUNCATE runs; TRUNCATE posts`)
	t.Setenv("DATABASE_URL", url)

	post := httptest.NewRequest("POST", "/api/scores", strings.NewReader(
		`{"name":"Arslan","won":true,"secs":245,"shots":60,"hits":41,"kills":14,"lemons":10}`))
	post.Header.Set("Origin", "https://thebanri.github.io")
	w := httptest.NewRecorder()
	Scores(w, post)
	var a struct {
		Rank  int `json:"rank"`
		Board []struct {
			Name  string `json:"name"`
			Score int    `json:"score"`
		} `json:"board"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &a); err != nil || w.Code != 200 || a.Rank != 1 || a.Board[0].Score != 8666 {
		t.Fatalf("POST: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://thebanri.github.io" {
		t.Error("no CORS answer for the playground")
	}
	w = httptest.NewRecorder()
	Scores(w, httptest.NewRequest("GET", "/scores", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"Arslan"`) {
		t.Errorf("GET: %d %s", w.Code, w.Body.String())
	}
}

func TestWithoutADatabaseItSaysSo(t *testing.T) {
	mu.Lock()
	handler = nil
	mu.Unlock()
	t.Setenv("DATABASE_URL", "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	w := httptest.NewRecorder()
	Scores(w, httptest.NewRequest("GET", "/scores", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("no database: %d %s", w.Code, w.Body.String())
	}
}

func TestHealthNeedsNoDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	w := httptest.NewRecorder()
	Health(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 || w.Body.String() != "ok\n" {
		t.Errorf("healthz: %d %q", w.Code, w.Body.String())
	}
}
