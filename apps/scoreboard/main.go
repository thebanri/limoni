// Scoreboard is Castle Lemonstein's shared leaderboard: a small HTTP server that
// takes each finished run and hands back the best.
//
//	GET  /scores        the best ten, as {"board": [...]}
//	POST /scores        a run, as {"name", "won", "secs", "shots", "hits",
//	                    "kills", "lemons"}; answers {"rank", "board"}
//	GET  /drop/scores   Lemon Drop's best ten; POST a run of it, as
//	                    {"name", "secs", "pieces", "drops", "clears"}
//	GET  /healthz       ok
//
// This is what Vercel runs: its Go preset finds this main.go and runs the
// server, listening on $PORT, as it does on Render, Railway, or a machine of
// one's own. The api/ functions serve the same handler for a project set up
// the older way, as functions.
//
// It keeps the best hundred runs in Postgres when DATABASE_URL names one,
// or else in $DATA_DIR/scores.json, on a disk that outlives the process (a
// Railway volume, which Railway also names in RAILWAY_VOLUME_MOUNT_PATH),
// or else in memory only. Browsers may call it from the origins in
// ALLOWED_ORIGINS (comma-separated; the GitHub Pages playground by
// default). It listens on $PORT.
//
// It is a module of its own like the game, so none of it, and none of its
// Postgres driver, reaches anyone who imports Limoni.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thebanri/limoni/apps/scoreboard/board"
)

func main() {
	store, where, err := storeFor(os.Getenv)
	if err != nil {
		log.Fatalf("reading the board: %v", err)
	}
	log.Print(where)
	origins := board.Origins(os.Getenv("ALLOWED_ORIGINS"))
	port := firstSet(os.Getenv("PORT"), "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           board.New(board.Config{Store: store, Origins: origins}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Printf("scoreboard on :%s, browsers from %v", port, origins)
	log.Fatal(srv.ListenAndServe())
}

// storeFor picks where the board is kept from the environment, and says so.
//
// On Vercel (which sets VERCEL) a board in memory would be a different
// board in each copy of the server, and gone when the copy is: every run
// sent would vanish, and an empty board would look just like a working
// one. So there, with no DATABASE_URL, the board says it has no database
// instead.
func storeFor(getenv func(string) string) (board.Store, string, error) {
	if url := getenv("DATABASE_URL"); url != "" {
		// Opened on the first request that needs it.
		return board.NewLazyPG(url), "keeping the board in Postgres", nil
	}
	if getenv("VERCEL") != "" {
		return noDatabase{}, "no DATABASE_URL on Vercel: connect a Neon database (Storage) and redeploy", nil
	}
	dir := firstSet(getenv("DATA_DIR"), getenv("RAILWAY_VOLUME_MOUNT_PATH"))
	if dir == "" {
		mem, err := board.OpenMem("")
		return mem, "no DATABASE_URL or DATA_DIR: scores are kept in memory and lost on restart", err
	}
	path := filepath.Join(dir, "scores.json")
	mem, err := board.OpenMem(path)
	return mem, "keeping the board in " + path, err
}

// noDatabase is the board on Vercel without a database: every request for
// it fails, and the handler answers that the board is not available.
type noDatabase struct{}

var errNoDatabase = errors.New("DATABASE_URL is not set")

func (noDatabase) Top(context.Context, int) ([]board.Entry, error) { return nil, errNoDatabase }
func (noDatabase) Add(context.Context, board.Entry) (int, error)   { return 0, errNoDatabase }
func (noDatabase) Allow(context.Context, string, time.Time) (bool, error) {
	return false, errNoDatabase
}
func (noDatabase) TopDrop(context.Context, int) ([]board.DropEntry, error) {
	return nil, errNoDatabase
}
func (noDatabase) AddDrop(context.Context, board.DropEntry) (int, error) { return 0, errNoDatabase }

func firstSet(v ...string) string {
	for _, s := range v {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	return ""
}
