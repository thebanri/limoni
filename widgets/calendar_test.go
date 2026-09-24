package widgets

import (
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func drawCalendar(c Calendar) (*buffer.Buffer, []string) {
	area := cell.NewRect(0, 0, 20, 8)
	buf := buffer.NewBuffer(area)
	c.Draw(cell.NewContext(area, cell.Style{}), buf)
	lines := strings.Split(buf.Snapshot(), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return buf, lines
}

func TestCalendarLayout(t *testing.T) {
	// September 2026 starts on a Tuesday.
	_, got := drawCalendar(Calendar{Month: day(2026, time.September, 24), FirstWeekday: time.Monday})
	want := []string{
		"   September 2026",
		"Mo Tu We Th Fr Sa Su",
		"    1  2  3  4  5  6",
		" 7  8  9 10 11 12 13",
		"14 15 16 17 18 19 20",
		"21 22 23 24 25 26 27",
		"28 29 30",
		"",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// Sunday first, Turkish names.
	tr := [12]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}
	trDays := [7]string{"Pz", "Pt", "Sa", "Ça", "Pe", "Cu", "Ct"}
	_, got = drawCalendar(Calendar{Month: day(2026, time.February, 1), MonthNames: &tr, WeekdayNames: &trDays})
	if got[0] != "     Şubat 2026" || got[1] != "Pz Pt Sa Ça Pe Cu Ct" || got[2] != " 1  2  3  4  5  6  7" || got[6] != "" {
		t.Errorf("February 2026, Sunday first:\n%s", strings.Join(got, "\n"))
	}
}

func TestCalendarStyles(t *testing.T) {
	today := cell.Style{Fg: cell.NewColorRGB(0, 255, 0)}
	busy := cell.Style{Fg: cell.NewColorRGB(255, 0, 0)}
	state := &CalendarState{Selected: day(2026, time.September, 10)}
	buf, _ := drawCalendar(Calendar{
		State: state, FirstWeekday: time.Monday, Today: day(2026, time.September, 24), TodayStyle: today,
		DayStyle: func(d time.Time) (cell.Style, bool) { return busy, d.Day() == 3 },
	})
	if got := buf.CellAt(10, 5).Style.Fg; got != today.Fg { // the 24th: row 5, Thursday
		t.Errorf("today's colour %v", got)
	}
	if got := buf.CellAt(10, 2).Style.Fg; got != busy.Fg { // the 3rd, also a Thursday
		t.Errorf("DayStyle colour %v", got)
	}
	if got := buf.CellAt(10, 3).Style; got.Modifier&cell.ModifierReverse == 0 { // the 10th
		t.Errorf("selected day not reversed: %+v", got)
	}
}

func TestCalendarKeys(t *testing.T) {
	s := &CalendarState{Selected: day(2026, time.January, 31)}
	for _, tc := range []struct {
		key  driver.KeyType
		want time.Time
	}{
		{driver.KeyPageDown, day(2026, time.February, 28)}, // clamped, not March 3
		{driver.KeyArrowDown, day(2026, time.March, 7)},
		{driver.KeyArrowLeft, day(2026, time.March, 6)},
		{driver.KeyEnd, day(2026, time.March, 31)},
		{driver.KeyArrowRight, day(2026, time.April, 1)},
		{driver.KeyPageUp, day(2026, time.March, 1)},
		{driver.KeyHome, day(2026, time.March, 1)},
	} {
		if !s.HandleKey(driver.KeyEvent{Type: tc.key}) {
			t.Fatalf("key %v not handled", tc.key)
		}
		if !sameDay(s.Selected, tc.want) {
			t.Fatalf("after key %v: %s, want %s", tc.key, s.Selected.Format(time.DateOnly), tc.want.Format(time.DateOnly))
		}
	}
}

func TestCalendarClickSelectsDay(t *testing.T) {
	state := &CalendarState{Selected: day(2026, time.September, 1)}
	var handler func(driver.MouseEvent)
	area := cell.NewRect(5, 2, 20, 8)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(_ cell.Rect, h func(driver.MouseEvent)) { handler = h }
	Calendar{State: state, FirstWeekday: time.Monday}.Draw(ctx, buffer.NewBuffer(cell.NewRect(0, 0, 30, 12)))

	// The 17th: Thursday (column 3) of the third week (grid row 2), under
	// the header and weekday rows.
	handler(driver.MouseEvent{Button: driver.MouseLeft, X: 5 + 3*3 + 1, Y: 2 + 2 + 2})
	if !sameDay(state.Selected, day(2026, time.September, 17)) {
		t.Errorf("clicked the 17th, selected %s", state.Selected.Format(time.DateOnly))
	}
	// Clicking the blank before the 1st changes nothing.
	handler(driver.MouseEvent{Button: driver.MouseLeft, X: 5, Y: 4})
	if !sameDay(state.Selected, day(2026, time.September, 17)) {
		t.Errorf("a blank cell changed the selection to %s", state.Selected.Format(time.DateOnly))
	}
}

func TestCalendarDoesNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	ctx.RegisterMouse = func(cell.Rect, func(driver.MouseEvent)) {}
	c := Calendar{State: &CalendarState{Selected: day(2026, time.September, 24)}, Today: day(2026, time.September, 24), FirstWeekday: time.Monday}
	c.Draw(ctx, buf)
	if a := testing.AllocsPerRun(50, func() { c.Draw(ctx, buf) }); a != 0 {
		t.Errorf("%.0f allocs per Draw", a)
	}
}

func BenchmarkCalendarDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	c := Calendar{State: &CalendarState{Selected: day(2026, time.September, 24)}, FirstWeekday: time.Monday}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.Draw(ctx, buf)
	}
}
