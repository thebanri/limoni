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
	drop    http.Handler
	health  = board.Health()
)

// serve builds the handler on the first request and keeps it for the rest
// of this copy's life, with its connections. A failure to reach the
// database is not kept: the next request tries again.
func serve() (scores, dropScores http.Handler, err error) {
	mu.Lock()
	defer mu.Unlock()
	if handler != nil {
		return handler, drop, nil
	}
	pg, err := board.OpenPG(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, nil, err
	}
	c := board.Config{Store: pg, Origins: board.Origins(os.Getenv("ALLOWED_ORIGINS"))}
	handler, drop = board.Scores(c), board.DropScores(c)
	return handler, drop, nil
}

// Scores serves /scores.
func Scores(w http.ResponseWriter, r *http.Request) {
	h, _, err := serve()
	if err != nil {
		unavailable(w, err)
		return
	}
	h.ServeHTTP(w, r)
}

// DropScores serves /drop/scores, Lemon Drop's board.
func DropScores(w http.ResponseWriter, r *http.Request) {
	_, h, err := serve()
	if err != nil {
		unavailable(w, err)
		return
	}
	h.ServeHTTP(w, r)
}

func unavailable(w http.ResponseWriter, err error) {
	log.Printf("opening the database: %v", err)
	http.Error(w, `{"error":"the board is not available"}`, http.StatusServiceUnavailable)
}

// Health serves /healthz. It does not touch the database.
func Health(w http.ResponseWriter, r *http.Request) { health.ServeHTTP(w, r) }
