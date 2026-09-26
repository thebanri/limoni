package board

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxBody = 1 << 10

// Store keeps the runs. Mem keeps them in the process (and a file); PG in
// Postgres, holding nothing between requests, as a Vercel function must.
type Store interface {
	// Top returns the best n runs, best first.
	Top(ctx context.Context, n int) ([]Entry, error)
	// Add puts e on the board and returns its place, 1-based, or 0 when it
	// did not make the runs kept.
	Add(ctx context.Context, e Entry) (int, error)
	// Allow reports whether addr may send another run at now, and counts
	// this one if so.
	Allow(ctx context.Context, addr string, now time.Time) (bool, error)
}

// DropStore keeps Lemon Drop's runs, beside Lemon Hunt's: every store here
// is both.
type DropStore interface {
	TopDrop(ctx context.Context, n int) ([]DropEntry, error)
	AddDrop(ctx context.Context, e DropEntry) (int, error)
	Allow(ctx context.Context, addr string, now time.Time) (bool, error)
}

// Config is how the handler is set up.
type Config struct {
	Store Store
	// Origins are the pages a browser may call it from; "*" allows any.
	Origins []string
	// Now is the clock; time.Now when nil.
	Now func() time.Time
}

type handler struct {
	store   Store
	origins map[string]bool
	any     bool
	now     func() time.Time
}

// Scores serves GET (the best ten), POST (a run; answers its place and the
// best ten) and the browser's OPTIONS, whatever the path: on Vercel it is
// a function of its own.
func Scores(c Config) http.Handler { return newHandler(c) }

// DropScores serves Lemon Drop's board as Scores does Lemon Hunt's. The
// store must be a DropStore too.
func DropScores(c Config) http.Handler {
	ds, _ := c.Store.(DropStore)
	return &dropHandler{handler: newHandler(c), store: ds}
}

func newHandler(c Config) *handler {
	h := &handler{store: c.Store, origins: map[string]bool{}, now: c.Now}
	if h.now == nil {
		h.now = time.Now
	}
	for _, o := range c.Origins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "*" {
			h.any = true
		} else if o != "" {
			h.origins[o] = true
		}
	}
	return h
}

// Health answers ok.
func Health() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
}

// New serves the whole board as one server: /scores and /healthz, and a
// line of text at /.
//
// On Vercel this is what runs: its Go preset finds main.go and runs the
// server, and vercel.json's rewrites, kept for the api/ functions, hand it
// the rewritten path — /api/healthz for /healthz. So the server answers on
// both.
func New(c Config) http.Handler {
	scores, drop, health := Scores(c), DropScores(c), Health()
	mux := http.NewServeMux()
	mux.Handle("/scores", scores)
	mux.Handle("/api/scores", scores)
	mux.Handle("/drop/scores", drop)
	mux.Handle("/api/drop/scores", drop)
	mux.Handle("/healthz", health)
	mux.Handle("/api/healthz", health)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Lemon Hunt's shared leaderboard. The best ten are at /scores, and Lemon Drop's at /drop/scores.\n"))
	})
	return mux
}

// Origins splits ALLOWED_ORIGINS, or gives the playground's page when it
// is empty.
func Origins(env string) []string {
	if strings.TrimSpace(env) == "" {
		return []string{"https://thebanri.github.io"}
	}
	return strings.Split(env, ",")
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.cors(w, r)
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.submit(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST"})
	}
}

// cors lets the playground's page call the board. A request from any other
// page still reaches it — the game in a terminal sends no Origin at all —
// but the browser keeps the answer from that page.
func (h *handler) cors(w http.ResponseWriter, r *http.Request) {
	o := r.Header.Get("Origin")
	if o == "" || !(h.any || h.origins[o]) {
		return
	}
	hd := w.Header()
	hd.Set("Access-Control-Allow-Origin", o)
	hd.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	hd.Set("Access-Control-Allow-Headers", "Content-Type")
	hd.Set("Access-Control-Max-Age", "86400")
	hd.Add("Vary", "Origin")
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	top, err := h.store.Top(r.Context(), Shown)
	if err != nil {
		h.fail(w, "reading the board", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"board": nonNil(top)})
}

func (h *handler) submit(w http.ResponseWriter, r *http.Request) {
	ok, err := h.store.Allow(r.Context(), clientAddr(r), h.now())
	if err != nil {
		h.fail(w, "counting runs", err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many runs, too quickly"})
		return
	}
	var run Run
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&run); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not a run: " + err.Error()})
		return
	}
	if err := run.Check(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	rank, err := h.store.Add(r.Context(), run.Entry(h.now()))
	if err != nil {
		h.fail(w, "adding the run", err)
		return
	}
	top, err := h.store.Top(r.Context(), Shown)
	if err != nil {
		h.fail(w, "reading the board", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rank": rank, "board": nonNil(top)})
}

// fail logs what went wrong and tells the client only that it did.
func (h *handler) fail(w http.ResponseWriter, doing string, err error) {
	log.Printf("%s: %v", doing, err)
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "the board is not available"})
}

// clientAddr is the client's address: behind the proxy of Vercel, Render or
// Railway, the first address in X-Forwarded-For.
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

func nonNil(e []Entry) []Entry {
	if e == nil {
		return []Entry{}
	}
	return e
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writing an answer: %v", err)
	}
}

// dropHandler is Lemon Drop's board: the same answers in the same shapes,
// over its own runs.
type dropHandler struct {
	*handler
	store DropStore
}

func (h *dropHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.cors(w, r)
	switch {
	case r.Method == http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	case h.store == nil:
		h.fail(w, "serving Lemon Drop", errors.New("the store keeps no Lemon Drop runs"))
	case r.Method == http.MethodGet:
		top, err := h.store.TopDrop(r.Context(), Shown)
		if err != nil {
			h.fail(w, "reading Lemon Drop's board", err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, map[string]any{"board": nonNilDrop(top)})
	case r.Method == http.MethodPost:
		h.submit(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST"})
	}
}

func (h *dropHandler) submit(w http.ResponseWriter, r *http.Request) {
	ok, err := h.store.Allow(r.Context(), clientAddr(r), h.now())
	if err != nil {
		h.fail(w, "counting runs", err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many runs, too quickly"})
		return
	}
	var run DropRun
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, DropBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&run); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not a run: " + err.Error()})
		return
	}
	if err := run.Check(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	rank, err := h.store.AddDrop(r.Context(), run.Entry(h.now()))
	if err != nil {
		h.fail(w, "adding the run", err)
		return
	}
	top, err := h.store.TopDrop(r.Context(), Shown)
	if err != nil {
		h.fail(w, "reading Lemon Drop's board", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rank": rank, "board": nonNilDrop(top)})
}

func nonNilDrop(e []DropEntry) []DropEntry {
	if e == nil {
		return []DropEntry{}
	}
	return e
}
