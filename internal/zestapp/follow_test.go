package zestapp

import (
	"strconv"
	"testing"

	"github.com/thebanri/limoni/uitest"
)

// Picking a line while new ones keep arriving: the click must pause following,
// or the chosen line scrolls away and Enter shows the details of another.
func TestSelectingWhileLinesArrive(t *testing.T) {
	s := &store{}
	add := func(n int) {
		for i := 0; i < n; i++ {
			lvl := "info"
			if s.Len()%7 == 0 {
				lvl = "error"
			}
			s.add([]byte(`{"level":"` + lvl + `","msg":"connection refused ` + strconv.Itoa(s.Len()) + `"}`))
		}
	}
	add(5000)
	v := &view{src: s}
	u := newViewer("t", s, v)
	page := uitest.Run(t, 120, 20, u.Frame)
	v.wake = func() {}
	page.Press("5")
	page.Press("/")
	page.GetByID("filter").Type("connection refused")
	page.Press("enter")
	add(50)
	v.kick()
	page.GetByRole("list-item", "").Nth(0).Select()
	add(50)
	v.kick()
	page.Press("enter")
	page.Expect(page.GetByID("details")).ToBeVisible()
	page.Press("q")
	if !page.Exited() {
		t.Fatal("no exit")
	}
}
