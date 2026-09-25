// Package fn is the scoreboard as Vercel runs it: the board's handler over
// the Postgres in DATABASE_URL, which Vercel's Neon integration sets. The
// functions in api/ are one line each that call into here, so that api/
// holds nothing but the index.go files Vercel builds: a test file there
// would be taken for a function too.
package fn

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/thebanri/limoni/apps/scoreboard/board"
)

var (
	mu      sync.Mutex
	handler http.Handler
	health  = board.Health()
)

// serve builds the handler on the first request and keeps it for the rest
// of this copy's life, with its connections. A failure to reach the
// database is not kept: the next request tries again.
func serve() (http.Handler, error) {
	mu.Lock()
	defer mu.Unlock()
	if handler != nil {
		return handler, nil
	}
	pg, err := board.OpenPG(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	handler = board.Scores(board.Config{Store: pg, Origins: board.Origins(os.Getenv("ALLOWED_ORIGINS"))})
	return handler, nil
}

// Scores serves /scores.
func Scores(w http.ResponseWriter, r *http.Request) {
	h, err := serve()
	if err != nil {
		log.Printf("opening the database: %v", err)
		http.Error(w, `{"error":"the board is not available"}`, http.StatusServiceUnavailable)
		return
	}
	h.ServeHTTP(w, r)
}

// Health serves /healthz. It does not touch the database.
func Health(w http.ResponseWriter, r *http.Request) { health.ServeHTTP(w, r) }
