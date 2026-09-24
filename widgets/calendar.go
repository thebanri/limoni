package widgets

import (
	"strconv"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// CalendarState is the selected day. The calendar shows the month the
// selection is in, so moving the selection past the end of a month turns
// the page.
type CalendarState struct {
	Selected time.Time

	// Geometry of the last frame, for clicks.
	origin      cell.Rect
	gridTop     uint16
	first       time.Time
	firstColumn int
	onMouse     func(driver.MouseEvent)
}

// HandleKey moves the selection: arrows by a day or a week, Page Up and
// Page Down by a month, Home and End to the ends of the month.
func (s *CalendarState) HandleKey(ev driver.KeyEvent) bool {
	if s == nil {
		return false
	}
	d := s.Selected
	switch ev.Type {
	case driver.KeyArrowLeft:
		d = d.AddDate(0, 0, -1)
	case driver.KeyArrowRight:
		d = d.AddDate(0, 0, 1)
	case driver.KeyArrowUp:
		d = d.AddDate(0, 0, -7)
	case driver.KeyArrowDown:
		d = d.AddDate(0, 0, 7)
	case driver.KeyPageUp:
		d = addMonthsClamped(d, -1)
	case driver.KeyPageDown:
		d = addMonthsClamped(d, 1)
	case driver.KeyHome:
		d = time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
	case driver.KeyEnd:
		d = time.Date(d.Year(), d.Month(), daysIn(d.Year(), d.Month()), 0, 0, 0, 0, d.Location())
	default:
		return false
	}
	s.Selected = d
	return true
}

// addMonthsClamped moves by whole months and keeps the day inside the
// target month: January 31 plus one month is February 28, not March 3.
func addMonthsClamped(d time.Time, months int) time.Time {
	first := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location()).AddDate(0, months, 0)
	day := d.Day()
	if n := daysIn(first.Year(), first.Month()); day > n {
		day = n
	}
	return time.Date(first.Year(), first.Month(), day, d.Hour(), d.Minute(), d.Second(), d.Nanosecond(), d.Location())
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

var (
	defaultMonthNames = [12]string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	defaultWeekdayNames = [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
)

// Calendar shows one month as a grid of days, seven columns of three cells:
//
//	 September 2026
//	Mo Tu We Th Fr Sa Su
//	    1  2  3  4  5  6
//	 7  8  9 10 11 12 13
//	...
//
// It is 20 columns wide and at most 8 rows tall.
type Calendar struct {
	ID string

	// State holds the selected day; the calendar shows its month. Without a
	// State, Month decides which month is shown and nothing is selected.
	State *CalendarState
	Month time.Time

	// Today, if set, is drawn in TodayStyle. The calendar does not read the
	// clock itself, so it draws the same thing in a test as in a terminal.
	Today time.Time

	// FirstWeekday is the column the week starts in; the zero value is
	// Sunday. Set time.Monday for ISO weeks.
	FirstWeekday time.Weekday

	// MonthNames and WeekdayNames replace the English ones, Sunday first.
	// Weekday names are cut to two columns.
	MonthNames   *[12]string
	WeekdayNames *[7]string

	// HideHeader drops the month and year line; HideWeekdays the day names.
	HideHeader   bool
	HideWeekdays bool

	// DayStyle, if set, is asked about every day shown; returning ok merges
	// the style over the day, for marking days with events.
	DayStyle func(day time.Time) (style cell.Style, ok bool)

	Style         cell.Style
	HeaderStyle   cell.Style
	WeekdayStyle  cell.Style
	TodayStyle    cell.Style
	SelectedStyle cell.Style
}

func (c Calendar) shownMonth() time.Time {
	d := c.Month
	if c.State != nil && !c.State.Selected.IsZero() {
		d = c.State.Selected
	}
	if d.IsZero() {
		d = c.Today
	}
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
}

// Draw renders the month. It does not allocate unless DayStyle does.
func (c Calendar) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 {
		return
	}
	if c.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(c.ID)
	}
	base := ctx.Style.Merge(c.Style)
	first := c.shownMonth()
	y := area.Y
	bottom := area.Y + area.Height

	if !c.HideHeader && y < bottom {
		names := &defaultMonthNames
		if c.MonthNames != nil {
			names = c.MonthNames
		}
		name := names[first.Month()-1]
		var tmp [8]byte
		year := strconv.AppendInt(tmp[:0], int64(first.Year()), 10)
		w := cell.StringWidth(name) + 1 + len(year)
		x := area.X
		if int(area.Width) > w {
			x += uint16((min(int(area.Width), 20) - w) / 2)
		}
		style := base.Merge(c.HeaderStyle)
		x += buf.SetStringWithin(x, y, name, style, area.X+area.Width-x)
		if x < area.X+area.Width {
			buf.SetCell(x, y, cell.Cell{Content: ' ', Style: style})
			x++
		}
		for _, ch := range year {
			if x >= area.X+area.Width {
				break
			}
			buf.SetCell(x, y, cell.Cell{Content: rune(ch), Style: style})
			x++
		}
		y++
	}

	if !c.HideWeekdays && y < bottom {
		names := &defaultWeekdayNames
		if c.WeekdayNames != nil {
			names = c.WeekdayNames
		}
		style := base.Merge(c.WeekdayStyle)
		for col := 0; col < 7; col++ {
			x := area.X + uint16(col*3)
			if x >= area.X+area.Width {
				break
			}
			name := names[(int(c.FirstWeekday)+col)%7]
			setClipped(buf, x, y, name, style, min(2, int(area.X+area.Width-x)))
		}
		y++
	}

	firstColumn := (int(first.Weekday()) - int(c.FirstWeekday) + 7) % 7
	days := daysIn(first.Year(), first.Month())
	if c.State != nil {
		c.State.origin, c.State.gridTop = area, y
		c.State.first, c.State.firstColumn = first, firstColumn
	}
	for day := 1; day <= days; day++ {
		slot := firstColumn + day - 1
		row, col := uint16(slot/7), slot%7
		if y+row >= bottom {
			break
		}
		x := area.X + uint16(col*3)
		if x+2 > area.X+area.Width {
			continue
		}
		date := first.AddDate(0, 0, day-1)
		style := base
		if c.DayStyle != nil {
			if st, ok := c.DayStyle(date); ok {
				style = style.Merge(st)
			}
		}
		if !c.Today.IsZero() && sameDay(date, c.Today) {
			style = style.Merge(c.TodayStyle)
		}
		if c.State != nil && sameDay(date, c.State.Selected) {
			sel := c.SelectedStyle
			if sel == (cell.Style{}) {
				sel = cell.Style{}.Reverse()
			}
			style = style.Merge(sel)
		}
		tens := ' '
		if day >= 10 {
			tens = rune('0' + day/10)
		}
		buf.SetCell(x, y+row, cell.Cell{Content: tens, Style: style})
		buf.SetCell(x+1, y+row, cell.Cell{Content: rune('0' + day%10), Style: style})
	}

	if c.State != nil && ctx.RegisterMouse != nil {
		ctx.RegisterMouse(area, c.State.handler())
	}
}

// handler is built once per state: a click on a day selects it.
func (s *CalendarState) handler() func(driver.MouseEvent) {
	if s.onMouse == nil {
		s.onMouse = func(ev driver.MouseEvent) {
			if ev.Button != driver.MouseLeft || ev.Drag || ev.Y < s.gridTop || ev.X < s.origin.X {
				return
			}
			col := int(ev.X-s.origin.X) / 3
			if col > 6 {
				return
			}
			day := int(ev.Y-s.gridTop)*7 + col - s.firstColumn + 1
			if day >= 1 && day <= daysIn(s.first.Year(), s.first.Month()) {
				s.Selected = s.first.AddDate(0, 0, day-1)
			}
		}
	}
	return s.onMouse
}

// SizeHint is 20×8, or less when the header or weekday row is hidden.
func (c Calendar) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	h := uint16(6)
	if !c.HideHeader {
		h++
	}
	if !c.HideWeekdays {
		h++
	}
	return min(20, maxArea.Width), min(h, maxArea.Height)
}

// AccessibilityNode reports the selected date, as YYYY-MM-DD.
func (c Calendar) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	value := ""
	if c.State != nil && !c.State.Selected.IsZero() {
		value = c.State.Selected.Format(time.DateOnly)
	}
	return accessibility.AccessibilityNode{
		ID:     c.ID,
		Role:   accessibility.RoleTable,
		Label:  "Calendar",
		Value:  value,
		State:  state,
		Bounds: bounds,
	}
}
