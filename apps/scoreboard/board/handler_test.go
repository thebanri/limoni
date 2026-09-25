package board

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Every test runs against each store: in memory, in a file, and in
// Postgres when TEST_DATABASE_URL names one (CI starts one; ci.yml,
// scoreboard-postgres).
func forEachStore(t *testing.T, test func(t *testing.T, open func() Store)) {
	t.Run("memory", func(t *testing.T) {
		m, _ := OpenMem("")
		test(t, func() Store { return m })
	})
	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "data", "scores.json")
		var mu sync.Mutex
		test(t, func() Store {
			mu.Lock()
			defer mu.Unlock()
			m, err := OpenMem(path) // a fresh read each time: a restart
			if err != nil {
				t.Fatal(err)
			}
			return m
		})
	})
	t.Run("postgres", func(t *testing.T) {
		url := os.Getenv("TEST_DATABASE_URL")
		if url == "" {
			t.Skip("TEST_DATABASE_URL is not set")
		}
		first, err := OpenPG(context.Background(), url)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(first.Close)
		if _, err := first.pool.Exec(context.Background(), `TRUNCATE runs; TRUNCATE posts`); err != nil {
			t.Fatal(err)
		}
		test(t, func() Store {
			p, err := OpenPG(context.Background(), url)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(p.Close)
			return p
		})
	})
}

type answer struct {
	Rank  int     `json:"rank"`
	Board []Entry `json:"board"`
	Error string  `json:"error"`
}

type fixture struct {
	t    *testing.T
	h    http.Handler
	now  time.Time
	addr int
}

func newFixture(t *testing.T, s Store, start time.Time) *fixture {
	f := &fixture{t: t, now: start}
	f.h = New(Config{Store: s, Origins: []string{"https://thebanri.github.io"}, Now: func() time.Time { return f.now }})
	return f
}

var epoch = time.Unix(1_800_000_000, 0)

func (f *fixture) do(method, path, body string, header ...string) (*httptest.ResponseRecorder, answer) {
	f.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	if req.Header.Get("X-Forwarded-For") == "" {
		f.addr++ // a different player each time, unless the test says otherwise
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.0.%d.%d", f.addr/250, f.addr%250))
	}
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, req)
	var a answer
	if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(w.Body.Bytes(), &a); err != nil {
			f.t.Fatalf("%s: %v", w.Body.String(), err)
		}
	}
	return w, a
}

func TestRunsAreScoredByTheServerAndRanked(t *testing.T) {
	forEachStore(t, func(t *testing.T, open func() Store) {
		f := newFixture(t, open(), epoch)
		// A win in 200 s, 36 of 40: 4500 aim + 4000 time + 500 rats + 1000 lemons.
		w, a := f.do("POST", "/scores", `{"name":"Ayşe","won":true,"secs":200,"shots":40,"hits":36,"kills":10,"lemons":10}`)
		if w.Code != 200 || a.Rank != 1 || len(a.Board) != 1 || a.Board[0].Score != 10000 || a.Board[0].Aim != 90 {
			t.Fatalf("%d %+v", w.Code, a)
		}
		// A loss: no time points.
		_, a = f.do("POST", "/scores", `{"name":"Can","won":false,"secs":90,"shots":10,"hits":5,"kills":3,"lemons":4}`)
		if a.Rank != 2 || a.Board[1].Score != 2500+150+400 {
			t.Fatalf("%+v", a)
		}
		// A better win goes to the top.
		_, a = f.do("POST", "/scores", `{"name":"Deniz","won":true,"secs":150,"shots":30,"hits":30,"kills":12,"lemons":10}`)
		if a.Rank != 1 || a.Board[0].Name != "Deniz" || len(a.Board) != 3 {
			t.Fatalf("%+v", a)
		}
		_, a = f.do("GET", "/scores", "")
		if len(a.Board) != 3 || a.Board[0].Name != "Deniz" || a.Board[2].Name != "Can" {
			t.Fatalf("%+v", a)
		}
	})
}

func TestAnEmptyBoardIsAnEmptyList(t *testing.T) {
	forEachStore(t, func(t *testing.T, open func() Store) {
		f := newFixture(t, open(), epoch)
		w, _ := f.do("GET", "/scores", "")
		if body := strings.TrimSpace(w.Body.String()); body != `{"board":[]}` {
			t.Errorf("an empty board is %s", body)
		}
	})
}

func TestTheClientsScoreIsIgnored(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	w, _ := f.do("POST", "/scores", `{"name":"x","won":false,"secs":10,"shots":1,"hits":1,"score":999999}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("a run with its own score: %d, want 400", w.Code)
	}
}

func TestRunsTheGameCannotProduceAreRefused(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	for _, body := range []string{
		`{"name":"","won":false,"secs":10,"shots":1,"hits":1}`,           // no name
		`{"name":"<\u001b>{}","won":false,"secs":10,"shots":1,"hits":1}`, // nothing left of it
		`{"name":"x","won":false,"secs":10,"shots":1,"hits":2}`,          // more hits than squirts
		`{"name":"x","won":false,"secs":10,"shots":-1,"hits":0}`,
		`{"name":"x","won":false,"secs":10,"kills":99}`,
		`{"name":"x","won":false,"secs":10,"lemons":11}`,
		`{"name":"x","won":true,"secs":300,"lemons":9}`, // a win needs ten lemons
		`{"name":"x","won":true,"secs":5,"lemons":10}`,  // and more than five seconds
		`{"name":"x","won":false,"secs":0}`,
		`not json`,
		`{"name":"` + strings.Repeat("a", 2000) + `"}`, // too big
	} {
		if w, a := f.do("POST", "/scores", body); w.Code < 400 || a.Rank != 0 {
			t.Errorf("%.60s: %d %+v", body, w.Code, a)
		}
	}
	if _, a := f.do("GET", "/scores", ""); len(a.Board) != 0 {
		t.Errorf("refused runs reached the board: %+v", a.Board)
	}
}

func TestNamesAreCleaned(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	_, a := f.do("POST", "/scores", `{"name":"  <b>Özgür</b> the very long ","won":false,"secs":10}`)
	if a.Board[0].Name != "bÖzgürb the" { // cut at twelve, then trimmed
		t.Errorf("stored %q", a.Board[0].Name)
	}
}

func TestOnePlayerCannotFloodTheBoard(t *testing.T) {
	forEachStore(t, func(t *testing.T, open func() Store) {
		f := newFixture(t, open(), epoch)
		run := `{"name":"x","won":false,"secs":10}`
		for i := 0; i < PostsPerMinute; i++ {
			if w, _ := f.do("POST", "/scores", run, "X-Forwarded-For", "1.2.3.4, 10.0.0.1"); w.Code != 200 {
				t.Fatalf("post %d: %d", i, w.Code)
			}
		}
		if w, _ := f.do("POST", "/scores", run, "X-Forwarded-For", "1.2.3.4"); w.Code != http.StatusTooManyRequests {
			t.Errorf("post %d in a minute: %d, want 429", PostsPerMinute+1, w.Code)
		}
		if w, _ := f.do("POST", "/scores", run, "X-Forwarded-For", "5.6.7.8"); w.Code != 200 {
			t.Errorf("another player was refused: %d", w.Code)
		}
		f.now = f.now.Add(61 * time.Second)
		if w, _ := f.do("POST", "/scores", run, "X-Forwarded-For", "1.2.3.4"); w.Code != 200 {
			t.Errorf("a minute later: %d", w.Code)
		}
	})
}

// Many runs at once from one address still get only PostsPerMinute
// through: on Vercel they may meet different copies of the function.
func TestTheLimitHoldsForRunsSentAtOnce(t *testing.T) {
	forEachStore(t, func(t *testing.T, open func() Store) {
		stores := []Store{open(), open(), open()}
		var wg sync.WaitGroup
		var mu sync.Mutex
		passed := 0
		for i := 0; i < 15; i++ {
			wg.Add(1)
			go func(s Store) {
				defer wg.Done()
				ok, err := s.Allow(context.Background(), "9.9.9.9", epoch)
				if err != nil {
					t.Error(err)
				}
				if ok {
					mu.Lock()
					passed++
					mu.Unlock()
				}
			}(stores[i%3])
		}
		wg.Wait()
		// Three copies of the file store count apart, as three processes
		// would; only one shared store is the point here.
		want := PostsPerMinute
		if a, b := stores[0], stores[1]; a != b {
			if _, isMem := a.(*Mem); isMem {
				t.Skip("separate processes count apart; the limit is per store")
			}
		}
		if passed != want {
			t.Errorf("%d of 15 runs at once got through, want %d", passed, want)
		}
	})
}

func TestOnlyThePlaygroundMayReadItFromABrowser(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	w, _ := f.do("OPTIONS", "/scores", "", "Origin", "https://thebanri.github.io", "Access-Control-Request-Method", "POST")
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "https://thebanri.github.io" ||
		!strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Content-Type") {
		t.Errorf("preflight from the playground: %d %v", w.Code, w.Header())
	}
	w, _ = f.do("GET", "/scores", "", "Origin", "https://evil.example")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("another site was allowed")
	}
	if w, _ := f.do("DELETE", "/scores", ""); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE: %d", w.Code)
	}
	if w, _ := f.do("GET", "/healthz", ""); w.Code != 200 || w.Body.String() != "ok\n" {
		t.Errorf("healthz: %d %q", w.Code, w.Body.String())
	}
}

// The board survives a restart and keeps only the best hundred, and a tie
// goes to the run made first.
func TestTheBoardSurvivesARestart(t *testing.T) {
	forEachStore(t, func(t *testing.T, open func() Store) {
		if m, ok := open().(*Mem); ok && m.path == "" {
			t.Skip("memory does not outlive the process")
		}
		f := newFixture(t, open(), epoch)
		for i := 0; i < Keep+5; i++ {
			f.now = f.now.Add(time.Second)
			f.do("POST", "/scores", fmt.Sprintf(`{"name":"p","won":false,"secs":10,"kills":%d}`, i%10))
		}
		g := newFixture(t, open(), f.now)
		_, a := g.do("GET", "/scores", "")
		if len(a.Board) != Shown || a.Board[0].Score != 450 {
			t.Fatalf("after a restart: %+v", a.Board)
		}
		// The first of the ten runs with nine kills.
		if a.Board[0].At != epoch.Unix()+10 {
			t.Errorf("a tie after a restart went to the run made at %d", a.Board[0].At)
		}
		all, err := open().Top(context.Background(), 1000)
		if err != nil || len(all) != Keep {
			t.Errorf("%d runs kept (%v), want %d", len(all), err, Keep)
		}
		// A run below all hundred is not placed.
		_, a = g.do("POST", "/scores", `{"name":"low","won":false,"secs":10}`)
		if a.Rank != 0 {
			t.Errorf("a run below the hundred placed %d", a.Rank)
		}
	})
}

func TestATieGoesToTheEarlierRun(t *testing.T) {
	forEachStore(t, func(t *testing.T, open func() Store) {
		f := newFixture(t, open(), epoch)
		f.do("POST", "/scores", `{"name":"first","won":false,"secs":10,"kills":2}`)
		f.now = f.now.Add(time.Second)
		_, a := f.do("POST", "/scores", `{"name":"second","won":false,"secs":10,"kills":2}`)
		if a.Rank != 2 || a.Board[0].Name != "first" {
			t.Errorf("%+v", a)
		}
		// Even within the same second.
		_, a = f.do("POST", "/scores", `{"name":"third","won":false,"secs":10,"kills":2}`)
		if a.Rank != 3 {
			t.Errorf("third in the same second placed %d", a.Rank)
		}
	})
}
