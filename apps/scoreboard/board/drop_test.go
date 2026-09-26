package board

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type dropAnswer struct {
	Rank  int         `json:"rank"`
	Board []DropEntry `json:"board"`
	Error string      `json:"error"`
}

func (f *fixture) drop(method, body string, header ...string) (int, dropAnswer) {
	f.t.Helper()
	w, _ := f.doRaw(method, "/drop/scores", body, header...)
	var a dropAnswer
	if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(w.Body.Bytes(), &a); err != nil {
			f.t.Fatalf("%s: %v", w.Body.String(), err)
		}
	}
	return w.Code, a
}

// run is a run of Lemon Drop as the game would send it.
func run(name string, secs, pieces, drops int, clears ...[2]int) string {
	cs, _ := json.Marshal(clears)
	if clears == nil {
		cs = []byte("[]")
	}
	return fmt.Sprintf(`{"name":%q,"secs":%d,"pieces":%d,"drops":%d,"clears":%s}`, name, secs, pieces, drops, cs)
}

func TestDropRunsAreScoredByTheServerAndRanked(t *testing.T) {
	forEachStore(t, func(t *testing.T, open func() Store) {
		f := newFixture(t, open(), epoch)
		// Five clears: the fourth and fifth are at level 2, the fifth a chain.
		body := run("Ada", 120, 40, 50, [2]int{100, 1}, [2]int{120, 1}, [2]int{100, 1}, [2]int{110, 1}, [2]int{100, 2})
		code, a := f.drop("POST", body)
		want := 50 + 100 + 120 + 100 + 110*2 + 100*2*2
		if code != 200 || a.Rank != 1 || len(a.Board) != 1 {
			t.Fatalf("%d %+v", code, a)
		}
		if e := a.Board[0]; e.Score != want || e.Clears != 5 || e.Level != 2 || e.Secs != 120 || e.Name != "Ada" {
			t.Fatalf("scored %+v, want %d points, 5 clears, level 2", e, want)
		}
		f.now = f.now.Add(1)
		if _, a := f.drop("POST", run("Bo", 60, 20, 10)); a.Rank != 2 {
			t.Errorf("a lower run ranked %d", a.Rank)
		}
		if _, a := f.drop("POST", run("Cy", 300, 100, 3000, [2]int{1800, 1}, [2]int{1800, 2})); a.Rank != 1 {
			t.Errorf("a higher run ranked %d", a.Rank)
		}
		if _, a := f.drop("GET", ""); len(a.Board) != 3 || a.Board[0].Name != "Cy" || a.Board[2].Name != "Bo" {
			t.Errorf("the board: %+v", a.Board)
		}
		// Castle Lemonstein's board is its own.
		if _, a := f.do("GET", "/scores", ""); len(a.Board) != 0 {
			t.Errorf("Lemon Drop's runs reached Castle Lemonstein's board: %+v", a.Board)
		}
	})
}

func TestDropRunsTheGameCannotProduceAreRefused(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	for _, c := range []struct{ why, body string }{
		{"no name", run(" ", 60, 10, 0)},
		{"no pieces", run("A", 60, 0, 0)},
		{"pieces faster than anyone places them", run("A", 10, 200, 0)},
		{"more drop points than drops give", run("A", 60, 10, 10*73)},
		{"a clear narrower than the board", run("A", 60, 10, 0, [2]int{10, 1})},
		{"a clear larger than the board", run("A", 60, 100, 0, [2]int{1801, 1})},
		{"a chain that skips", run("A", 60, 10, 0, [2]int{100, 1}, [2]int{100, 3})},
		{"a chain that starts at 2", run("A", 60, 10, 0, [2]int{100, 2})},
		{"more sand cleared than pieces brought", run("A", 60, 3, 0, [2]int{100, 1}, [2]int{100, 1})},
		{"a field the game does not send", `{"name":"A","secs":60,"pieces":10,"drops":0,"clears":[],"score":999999}`},
	} {
		if code, a := f.drop("POST", c.body); code/100 != 4 || a.Error == "" {
			t.Errorf("%s: %d %+v", c.why, code, a)
		}
	}
	if _, a := f.drop("GET", ""); len(a.Board) != 0 {
		t.Errorf("a refused run reached the board: %+v", a.Board)
	}
}

// One address may send six runs a minute, of both games together.
func TestTheLimitCoversBothGames(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	from := []string{"X-Forwarded-For", "10.9.9.9"}
	for i := 0; i < 3; i++ {
		if w, _ := f.do("POST", "/scores", `{"name":"A","won":false,"secs":30}`, from...); w.Code != 200 {
			t.Fatalf("Castle Lemonstein run %d: %d", i, w.Code)
		}
		if code, _ := f.drop("POST", run("A", 60, 10, 0), from...); code != 200 {
			t.Fatalf("Lemon Drop run %d: %d", i, code)
		}
	}
	if code, _ := f.drop("POST", run("A", 60, 10, 0), from...); code != http.StatusTooManyRequests {
		t.Errorf("a seventh run in a minute: %d", code)
	}
}

func TestTheDropBoardSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.json")
	m, _ := OpenMem(path)
	f := newFixture(t, m, epoch)
	if code, _ := f.drop("POST", run("Ada", 60, 10, 5)); code != 200 {
		t.Fatal(code)
	}
	// With no Castle Lemonstein runs at all, so no scores.json beside it.
	again, err := OpenMem(path)
	if err != nil {
		t.Fatal(err)
	}
	top, _ := again.TopDrop(context.Background(), Shown)
	if len(top) != 1 || top[0].Name != "Ada" {
		t.Fatalf("after a restart: %+v", top)
	}
}

func TestTheDropBoardAnswersTheBrowserAndApiPaths(t *testing.T) {
	m, _ := OpenMem("")
	f := newFixture(t, m, epoch)
	w, _ := f.doRaw("OPTIONS", "/api/drop/scores", "", "Origin", "https://thebanri.github.io")
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "https://thebanri.github.io" {
		t.Errorf("OPTIONS from the playground: %d %v", w.Code, w.Header())
	}
	w, _ = f.doRaw("GET", "/api/drop/scores", "", "Origin", "https://elsewhere.example")
	if w.Code != 200 || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("GET from another page: %d %v", w.Code, w.Header())
	}
}
