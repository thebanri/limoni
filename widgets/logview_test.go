package widgets

import (
	"strconv"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// testLog is a LogSource over a slice, with a level per line.
type testLog struct {
	lines  []string
	levels []LogLevel
}

func (l *testLog) Len() int             { return len(l.lines) }
func (l *testLog) Line(i int) string    { return l.lines[i] }
func (l *testLog) Level(i int) LogLevel { return l.levels[i] }
func (l *testLog) add(line string, lv LogLevel) {
	l.lines = append(l.lines, line)
	l.levels = append(l.levels, lv)
}

func numberedLog(n int) *testLog {
	l := &testLog{}
	for i := 1; i <= n; i++ {
		l.add("line "+strconv.Itoa(i), LevelInfo)
	}
	return l
}

func drawLog(lv *LogView, w, h uint16) *buffer.Buffer {
	area := cell.NewRect(0, 0, w, h)
	buf := buffer.NewBuffer(area)
	lv.Draw(cell.NewContext(area, cell.Style{}), buf)
	return buf
}

func rowAt(buf *buffer.Buffer, y uint16) string {
	var sb strings.Builder
	for x := uint16(0); x < buf.Area.Width; x++ {
		if c := buf.Get(x, y); c.Content != cell.RuneContinuation {
			sb.WriteString(cell.ClusterText(c.Content))
		}
	}
	return strings.TrimRight(sb.String(), " ")
}

// Following shows the newest lines as they arrive, like tail -f.
func TestLogViewFollowsNewLines(t *testing.T) {
	src := numberedLog(10)
	lv := &LogView{Source: src, State: NewLogViewState()}
	buf := drawLog(lv, 20, 3)
	if got := rowAt(buf, 2); got != "line 10" {
		t.Fatalf("last row %q, want line 10", got)
	}
	src.add("line 11", LevelInfo)
	buf = drawLog(lv, 20, 3)
	if got := rowAt(buf, 2); got != "line 11" {
		t.Fatalf("after a new line the last row is %q", got)
	}
}

// Scrolling up — by key or by mouse wheel — stops following, so new lines do
// not yank the reader away from what they are reading. End resumes.
func TestLogViewScrollingUpStopsFollowing(t *testing.T) {
	src := numberedLog(10)
	st := NewLogViewState()
	lv := &LogView{Source: src, State: st}
	drawLog(lv, 20, 3)

	st.HandleKey(driver.KeyEvent{Type: driver.KeyArrowUp}, src.Len())
	src.add("line 11", LevelInfo)
	buf := drawLog(lv, 20, 3)
	if st.Follow || rowAt(buf, 2) == "line 11" {
		t.Fatalf("still following after Up: follow=%v last=%q", st.Follow, rowAt(buf, 2))
	}

	st.HandleKey(driver.KeyEvent{Type: driver.KeyEnd}, src.Len())
	buf = drawLog(lv, 20, 3)
	if !st.Follow || rowAt(buf, 2) != "line 11" {
		t.Fatalf("End did not resume following: %q", rowAt(buf, 2))
	}

	st.Offset-- // what the mouse wheel does
	buf = drawLog(lv, 20, 3)
	if st.Follow {
		t.Fatalf("the wheel did not stop following")
	}
	if got := rowAt(buf, 0); got != "line 8" {
		t.Fatalf("after one notch up the first row is %q, want line 8", got)
	}
}

func TestLogViewSelectionPaging(t *testing.T) {
	src := numberedLog(100)
	st := NewLogViewState()
	lv := &LogView{Source: src, State: st}
	drawLog(lv, 20, 10)
	st.HandleKey(driver.KeyEvent{Type: driver.KeyHome}, src.Len())
	st.HandleKey(driver.KeyEvent{Type: driver.KeyPageDown}, src.Len())
	st.HandleKey(driver.KeyEvent{Type: driver.KeyPageDown}, src.Len())
	if st.Selected != 18 {
		t.Fatalf("two pages down from the top selected %d, want 18", st.Selected)
	}
	buf := drawLog(lv, 20, 10)
	if c := buf.Get(0, uint16(18-st.Offset)); c.Style.Modifier&cell.ModifierReverse == 0 {
		t.Fatalf("selected line is not drawn selected")
	}
	st.Select(90)
	drawLog(lv, 20, 10)
	if st.Offset > 90 || st.Offset+10 <= 90 {
		t.Fatalf("Select(90) left offset %d", st.Offset)
	}
}

// Line numbers, level colours, and highlighted search hits, including a hit
// after a wide character and one scrolled half off the left edge.
func TestLogViewDecorations(t *testing.T) {
	src := &testLog{}
	src.add("boot ok", LevelInfo)
	src.add("日本 ERROR disk full", LevelError)
	lv := &LogView{Source: src, State: NewLogViewState(), LineNumbers: true, Highlight: "error"}
	buf := drawLog(lv, 30, 2)
	if got := rowAt(buf, 1); got != "2 日本 ERROR disk full" {
		t.Fatalf("row 1 %q", got)
	}
	if c := buf.Get(2, 1); c.Style.Fg != defaultLevelStyles[LevelError].Fg {
		t.Errorf("error line not coloured")
	}
	// "ERROR" starts after the gutter (2 columns) and 日本 plus a space (5).
	for x := uint16(7); x < 12; x++ {
		if c := buf.Get(x, 1); c.Style.Bg != cell.NewColorRGB(250, 210, 60) {
			t.Fatalf("column %d of the hit not highlighted", x)
		}
	}
	if c := buf.Get(12, 1); c.Style.Bg == cell.NewColorRGB(250, 210, 60) {
		t.Fatalf("highlight runs past the hit")
	}
}

func TestIndexFold(t *testing.T) {
	for _, tc := range []struct {
		s, needle string
		want      int
	}{
		{"Disk ERROR", "error", 5}, {"abc", "ABC", 0}, {"abc", "x", -1}, {"日本error", "ERROR", 6}, {"", "a", -1},
	} {
		if got := indexFold(tc.s, tc.needle); got != tc.want {
			t.Errorf("indexFold(%q, %q) = %d, want %d", tc.s, tc.needle, got, tc.want)
		}
	}
}

// The visible lines are list items in the semantic tree, so an agent can read
// the log and select a line by its text.
func TestLogViewSemanticTree(t *testing.T) {
	src := numberedLog(50)
	st := NewLogViewState()
	lv := &LogView{ID: "log", Source: src, State: st}
	drawLog(lv, 20, 4)
	node := lv.AccessibilityNode(cell.NewRect(0, 0, 20, 4), false)
	if node.Role != accessibility.RoleList || node.SetSize != 50 || len(node.Children) != 4 {
		t.Fatalf("node %+v", node)
	}
	if node.Children[3].Label != "line 50" || node.State&accessibility.StateBusy == 0 {
		t.Fatalf("last visible row %q, following=%v", node.Children[3].Label, node.State&accessibility.StateBusy != 0)
	}
}

func TestLogViewDrawDoesNotAllocate(t *testing.T) {
	src := numberedLog(1000)
	src.lines[995] = "日本 error \U0001F468\u200D\U0001F469\u200D\U0001F467"
	lv := &LogView{ID: "log", Source: src, State: NewLogViewState(), LineNumbers: true, Highlight: "error"}
	area := cell.NewRect(0, 0, 60, 10)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	lv.Draw(ctx, buf)
	if a := testing.AllocsPerRun(100, func() {
		lv.Draw(ctx, buf)
		_ = lv.AccessibilityNode(area, false)
	}); a != 0 {
		t.Fatalf("Draw allocates %.0f times", a)
	}
}
