package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type answer struct {
	Rank  int     `json:"rank"`
	Board []entry `json:"board"`
	Error string  `json:"error"`
}

type fixture struct {
	t    *testing.T
	h    http.Handler
	now  time.Time
	addr int
}

func newFixture(t *testing.T, path string) *fixture {
	b, err := openBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, now: time.Unix(1_800_000_000, 0)}
	f.h = newServer(b, []string{"https://thebanri.github.io"}, func() time.Time { return f.now })
	return f
}

func (f *fixture) do(method, body string, header ...string) (*httptest.ResponseRecorder, answer) {
	f.t.Helper()
	req := httptest.NewRequest(method, "/scores", strings.NewReader(body))
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	if req.Header.Get("X-Forwarded-For") == "" {
		f.addr++ // a different player each time, unless the test says otherwise
		req.Header.Set("X-Forwarded-For", "10.0.0."+string(rune('0'+f.addr%10))+string(rune('0'+f.addr/10%10)))
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
	f := newFixture(t, "")
	// A win in 200 s, 36 of 40: 4500 aim + 4000 time + 500 rats + 1000 lemons.
	w, a := f.do("POST", `{"name":"Ayşe","won":true,"secs":200,"shots":40,"hits":36,"kills":10,"lemons":10}`)
	if w.Code != 200 || a.Rank != 1 || len(a.Board) != 1 || a.Board[0].Score != 10000 || a.Board[0].Aim != 90 {
		t.Fatalf("%d %+v", w.Code, a)
	}
	// A loss: no time points.
	_, a = f.do("POST", `{"name":"Can","won":false,"secs":90,"shots":10,"hits":5,"kills":3,"lemons":4}`)
	if a.Rank != 2 || a.Board[1].Score != 2500+150+400 {
		t.Fatalf("%+v", a)
	}
	// A better win goes to the top.
	_, a = f.do("POST", `{"name":"Deniz","won":true,"secs":150,"shots":30,"hits":30,"kills":12,"lemons":10}`)
	if a.Rank != 1 || a.Board[0].Name != "Deniz" || len(a.Board) != 3 {
		t.Fatalf("%+v", a)
	}
	_, a = f.do("GET", "")
	if len(a.Board) != 3 || a.Board[0].Name != "Deniz" || a.Board[2].Name != "Can" {
		t.Fatalf("%+v", a)
	}
}

func TestTheClientsScoreIsIgnored(t *testing.T) {
	f := newFixture(t, "")
	w, _ := f.do("POST", `{"name":"x","won":false,"secs":10,"shots":1,"hits":1,"kills":0,"lemons":0,"score":999999}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("a run with its own score: %d, want 400", w.Code)
	}
}

func TestRunsTheGameCannotProduceAreRefused(t *testing.T) {
	f := newFixture(t, "")
	for _, body := range []string{
		`{"name":"","won":false,"secs":10,"shots":1,"hits":1}`,          // no name
		`{"name":"<\u001b>{}","won":false,"secs":10,"shots":1,"hits":1}`, // nothing left of it
		`{"name":"x","won":false,"secs":10,"shots":1,"hits":2}`,         // more hits than squirts
		`{"name":"x","won":false,"secs":10,"shots":-1,"hits":0}`,
		`{"name":"x","won":false,"secs":10,"kills":99}`,
		`{"name":"x","won":false,"secs":10,"lemons":11}`,
		`{"name":"x","won":true,"secs":300,"lemons":9}`, // a win needs ten lemons
		`{"name":"x","won":true,"secs":5,"lemons":10}`,  // and more than five seconds
		`{"name":"x","won":false,"secs":0}`,
		`not json`,
		`{"name":"` + strings.Repeat("a", 2000) + `"}`, // too big
	} {
		if w, a := f.do("POST", body); w.Code < 400 || a.Rank != 0 {
			t.Errorf("%.60s: %d %+v", body, w.Code, a)
		}
	}
	if _, a := f.do("GET", ""); len(a.Board) != 0 {
		t.Errorf("refused runs reached the board: %+v", a.Board)
	}
}

func TestNamesAreCleaned(t *testing.T) {
	f := newFixture(t, "")
	_, a := f.do("POST", `{"name":"  <b>Özgür</b> the very long ","won":false,"secs":10}`)
	if a.Board[0].Name != "bÖzgürb the" { // cut at twelve, then trimmed
		t.Errorf("stored %q", a.Board[0].Name)
	}
}

func TestOnePlayerCannotFloodTheBoard(t *testing.T) {
	f := newFixture(t, "")
	run := `{"name":"x","won":false,"secs":10}`
	for i := 0; i < postsPerIP; i++ {
		if w, _ := f.do("POST", run, "X-Forwarded-For", "1.2.3.4, 10.0.0.1"); w.Code != 200 {
			t.Fatalf("post %d: %d", i, w.Code)
		}
	}
	if w, _ := f.do("POST", run, "X-Forwarded-For", "1.2.3.4"); w.Code != http.StatusTooManyRequests {
		t.Errorf("post %d in a minute: %d, want 429", postsPerIP+1, w.Code)
	}
	if w, _ := f.do("POST", run, "X-Forwarded-For", "5.6.7.8"); w.Code != 200 {
		t.Errorf("another player was refused: %d", w.Code)
	}
	f.now = f.now.Add(61 * time.Second)
	if w, _ := f.do("POST", run, "X-Forwarded-For", "1.2.3.4"); w.Code != 200 {
		t.Errorf("a minute later: %d", w.Code)
	}
}

func TestOnlyThePlaygroundMayReadItFromABrowser(t *testing.T) {
	f := newFixture(t, "")
	w, _ := f.do("OPTIONS", "", "Origin", "https://thebanri.github.io", "Access-Control-Request-Method", "POST")
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "https://thebanri.github.io" ||
		!strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Content-Type") {
		t.Errorf("preflight from the playground: %d %v", w.Code, w.Header())
	}
	w, _ = f.do("GET", "", "Origin", "https://evil.example")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("another site was allowed")
	}
}

func TestTheBoardSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "scores.json")
	f := newFixture(t, path)
	for i := 0; i < keep+5; i++ {
		f.now = f.now.Add(time.Second)
		f.do("POST", `{"name":"p","won":false,"secs":10,"kills":`+string(rune('0'+i%10))+`}`)
	}
	g := newFixture(t, path)
	_, a := g.do("GET", "")
	if len(a.Board) != boardLen || a.Board[0].Score != 450 {
		t.Fatalf("after a restart: %+v", a.Board)
	}
	b, _ := openBoard(path)
	if len(b.entries) != keep {
		t.Errorf("%d runs kept, want %d", len(b.entries), keep)
	}
}

func TestATieGoesToTheEarlierRun(t *testing.T) {
	f := newFixture(t, "")
	f.do("POST", `{"name":"first","won":false,"secs":10,"kills":2}`)
	f.now = f.now.Add(time.Second)
	_, a := f.do("POST", `{"name":"second","won":false,"secs":10,"kills":2}`)
	if a.Rank != 2 || a.Board[0].Name != "first" {
		t.Errorf("%+v", a)
	}
}
