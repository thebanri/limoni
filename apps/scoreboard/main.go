// Scoreboard is Lemon Hunt's shared leaderboard: a small HTTP server that
// takes each finished run and hands back the best.
//
//	GET  /scores        the best ten, as {"board": [...]}
//	POST /scores        a run, as {"name", "won", "secs", "shots", "hits",
//	                    "kills", "lemons"}; answers {"rank", "board"}
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
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thebanri/limoni/apps/scoreboard/board"
)

func main() {
	var store board.Store
	dir := firstSet(os.Getenv("DATA_DIR"), os.Getenv("RAILWAY_VOLUME_MOUNT_PATH"))
	if url := os.Getenv("DATABASE_URL"); url != "" {
		store = board.NewLazyPG(url) // opened on the first request that needs it
		log.Print("keeping the board in Postgres")
	} else {
		path := ""
		if dir != "" {
			path = filepath.Join(dir, "scores.json")
			log.Printf("keeping the board in %s", path)
		} else {
			log.Print("no DATABASE_URL or DATA_DIR: scores are kept in memory and lost on restart")
		}
		mem, err := board.OpenMem(path)
		if err != nil {
			log.Fatalf("reading the board: %v", err)
		}
		store = mem
	}
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

func firstSet(v ...string) string {
	for _, s := range v {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	return ""
}
