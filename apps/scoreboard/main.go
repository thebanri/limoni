// Scoreboard is Lemon Hunt's shared leaderboard: a small HTTP server that
// takes each finished run and hands back the best.
//
//	GET  /scores        the best ten, as {"board": [...]}
//	POST /scores        a run, as {"name", "won", "secs", "shots", "hits",
//	                    "kills", "lemons"}; answers {"rank", "board"}
//	GET  /healthz       ok
//
// It keeps the best hundred runs in $DATA_DIR/scores.json — on Railway, a
// volume, which Railway also names in RAILWAY_VOLUME_MOUNT_PATH. Browsers
// may call it from the origins in ALLOWED_ORIGINS (comma-separated; the
// GitHub Pages playground by default). It listens on $PORT.
//
// It uses nothing outside the standard library, and is a module of its own
// like the game, so none of it reaches anyone who imports Limoni.
package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	boardLen   = 10
	maxBody    = 1 << 10
	postsPerIP = 6 // in a minute: a run takes longer than ten seconds
)

func main() {
	dir := firstSet(os.Getenv("DATA_DIR"), os.Getenv("RAILWAY_VOLUME_MOUNT_PATH"))
	path := ""
	if dir != "" {
		path = filepath.Join(dir, "scores.json")
	} else {
		log.Print("no DATA_DIR: scores are kept in memory and lost on restart")
	}
	b, err := openBoard(path)
	if err != nil {
		log.Fatalf("reading the board: %v", err)
	}
	origins := strings.Split(firstSet(os.Getenv("ALLOWED_ORIGINS"), "https://thebanri.github.io"), ",")
	port := firstSet(os.Getenv("PORT"), "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           newServer(b, origins, time.Now),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	log.Printf("scoreboard on :%s, %d runs kept, browsers from %v", port, len(b.entries), origins)
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

type server struct {
	board   *board
	origins map[string]bool
	any     bool
	now     func() time.Time

	mu    sync.Mutex
	posts map[string][]time.Time // recent posts, by client address
}

func newServer(b *board, origins []string, now func() time.Time) http.Handler {
	s := &server{board: b, origins: map[string]bool{}, now: now, posts: map[string][]time.Time{}}
	for _, o := range origins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "*" {
			s.any = true
		} else if o != "" {
			s.origins[o] = true
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc("GET /scores", s.list)
	mux.HandleFunc("POST /scores", s.submit)
	mux.HandleFunc("OPTIONS /scores", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return s.cors(mux)
}

// cors lets the playground's page call the server. A request from any other
// page still reaches it — the game in a terminal sends no Origin at all —
// but the browser keeps the answer from that page.
func (s *server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && (s.any || s.origins[o]) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", o)
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			h.Set("Access-Control-Max-Age", "86400")
			h.Add("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"board": s.board.top(boardLen)})
}

func (s *server) submit(w http.ResponseWriter, r *http.Request) {
	if !s.allow(clientAddr(r)) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many runs, too quickly"})
		return
	}
	var run run
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&run); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not a run: " + err.Error()})
		return
	}
	if err := run.check(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	rank, err := s.board.add(run.entry(s.now()))
	if err != nil {
		log.Printf("saving the board: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"rank": rank, "board": s.board.top(boardLen)})
}

// allow limits each client to postsPerIP runs a minute.
func (s *server) allow(addr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	recent := s.posts[addr][:0]
	for _, t := range s.posts[addr] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= postsPerIP {
		s.posts[addr] = recent
		return false
	}
	s.posts[addr] = append(recent, now)
	if len(s.posts) > 10000 { // forget everyone rather than grow without end
		s.posts = map[string][]time.Time{addr: s.posts[addr]}
	}
	return true
}

// clientAddr is the client's address. Behind Railway's proxy that is the
// first address in X-Forwarded-For.
func clientAddr(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return strings.TrimSpace(strings.Split(f, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		log.Printf("writing an answer: %v", err)
	}
}
