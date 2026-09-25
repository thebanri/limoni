// Package scores is the scoreboard's /scores on Vercel, as a function: the
// board's handler over the Postgres in DATABASE_URL, which Vercel's Neon
// integration sets. vercel.json sends /scores here.
package scores

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

// Handler is what Vercel calls.
func Handler(w http.ResponseWriter, r *http.Request) {
	h, err := serve()
	if err != nil {
		log.Printf("opening the database: %v", err)
		http.Error(w, `{"error":"the board is not available"}`, http.StatusServiceUnavailable)
		return
	}
	h.ServeHTTP(w, r)
}
