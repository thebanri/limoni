package main

import (
	"strconv"
	"testing"

	"github.com/thebanri/limoni/uitest"
)

// The whole viewer, driven the way an agent or a user would: through its
// semantic tree, by role and label, in process.
func TestViewerEndToEnd(t *testing.T) {
	s := &store{}
	for i := 1; i <= 5000; i++ {
		level := "info"
		switch {
		case i%500 == 0:
			level = "error"
		case i%50 == 0:
			level = "warn"
		}
		s.add([]byte(`{"level":"` + level + `","msg":"job ` + strconv.Itoa(i) + ` done","worker":"w` + strconv.Itoa(i%4) + `"}`))
	}
	v := &view{src: s}
	u := newUI("test.log", s, v)
	page := uitest.Run(t, 120, 20, u.frame)
	v.wake = func() {}

	log := page.GetByID("log")
	page.Expect(log).ToHaveCount(1)
	page.Expect(page.GetByRole("list-item", `{"level":"error","msg":"job 5000 done","worker":"w0"}`)).ToBeVisible()

	// Errors only: 5000/500 = 10 lines.
	page.Press("5")
	page.Expect(page.GetByRole("list-item", "").Within(log)).ToHaveCount(10)

	// Then text on top of the level.
	page.Press("/")
	page.GetByID("filter").Type("job 2500 ")
	page.Press("enter")
	rows := page.GetByRole("list-item", "").Within(log)
	page.Expect(rows).ToHaveCount(1)

	// Details of that line, pretty-printed.
	rows.Nth(0).Select()
	page.Press("enter")
	page.Expect(page.GetByID("details")).ToHaveValue("level: error\nmsg: job 2500 done\nworker: w0")

	// Esc clears the filter, then closes details, then quits.
	page.Press("esc")
	page.Press("1")
	page.Expect(page.GetByRole("list-item", "").Within(log)).ToHaveCount(18) // the pane's height
	page.Press("esc")
	page.Press("esc")
	if !page.Exited() {
		t.Fatal("the viewer did not quit")
	}
}
