package zestapp

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

// ParseLevel reads a level name: trace, debug, info, warn, error or fatal.
func ParseLevel(s string) (widgets.LogLevel, error) {
	if s == "" {
		return widgets.LevelUnknown, nil
	}
	for l := widgets.LevelTrace; l <= widgets.LevelFatal; l++ {
		if strings.EqualFold(s, l.String()) {
			return l, nil
		}
	}
	return 0, fmt.Errorf("unknown level %q: use trace, debug, info, warn, error or fatal", s)
}

// Viewer is the log viewer: its data, its filter and what is on screen.
// Frame draws it full-screen; Draw and Key embed it in another application.
type Viewer struct {
	name string
	src  *store
	view *view

	log     *widgets.LogView
	state   *widgets.LogViewState
	filter  *widgets.TextInputState
	editing bool // the filter input has the keyboard
	details bool
	help    bool

	minLevel   widgets.LogLevel
	detail     detailCache
	paneHeight int // rows of the log pane last frame

	started   time.Time
	rateLines int
	rateAt    time.Time
	rate      float64
}

// NewViewer returns a viewer with an empty log. Feed it with ReadFrom or
// GenerateDemo, and call SetWake so background work can request a redraw.
func NewViewer(name string) *Viewer {
	s := &store{}
	return newViewer(name, s, &view{src: s})
}

func newViewer(name string, s *store, v *view) *Viewer {
	st := widgets.NewLogViewState()
	u := &Viewer{name: name, src: s, view: v, state: st, filter: widgets.NewTextInputState(), started: time.Now(), rateAt: time.Now()}
	u.log = &widgets.LogView{ID: "log", Label: "Log", Source: v, State: st, LineNumbers: true}
	return u
}

// SetWake installs the function background work calls to request a frame:
// new lines read, a filter scan making progress. Usually an App's Wakeup.
func (u *Viewer) SetWake(wake func()) { u.view.wake = wake }

// notify tells the viewer that lines arrived.
func (u *Viewer) notify() {
	u.view.kick()
	if u.view.wake != nil {
		u.view.wake()
	}
}

// ReadFrom reads lines from r until it ends or ctx is cancelled; with follow
// set and r a file, it keeps reading as the file grows, like tail -f.
func (u *Viewer) ReadFrom(ctx context.Context, r io.Reader, follow bool) error {
	return readLines(ctx, r, u.src, u.notify, follow)
}

// GenerateDemo fills the log with n generated lines and keeps adding a few
// dozen a second until ctx is cancelled.
func (u *Viewer) GenerateDemo(ctx context.Context, n int) error {
	return generateDemo(ctx, u.src, n, u.notify)
}

// SetFilter shows only lines containing text, at level min or above.
func (u *Viewer) SetFilter(text string, min widgets.LogLevel) {
	u.filter.SetValue(text)
	u.minLevel = min
	u.view.setFilter(text, min)
}

// Frame draws the viewer on the whole screen and handles ev; false quits.
// It has the shape limoni.Run and App.Run expect.
func (u *Viewer) Frame(f *limoni.Frame, ev *limoni.Event) bool {
	if ev != nil && ev.Type == limoni.EventKey && !u.Key(ev.Key) {
		return false
	}
	u.Draw(f, f.Area())
	return true
}

// Draw draws the viewer in area.
func (u *Viewer) Draw(f *limoni.Frame, area limoni.Rect) {
	u.log.Highlight = u.filter.Value()
	rows := limoni.SplitVertical(area, limoni.Fixed(1), limoni.Fill(), limoni.Fixed(1))
	u.drawHeader(f, rows[0])

	body := rows[1]
	u.paneHeight = int(body.Height)
	if u.details && body.Width >= 60 {
		cols := limoni.SplitHorizontal(body, limoni.Fill(), limoni.Percentage(40))
		f.RenderWidget(u.log, cols[0])
		u.drawDetails(f, cols[1])
	} else {
		f.RenderWidget(u.log, body)
	}
	u.drawFooter(f, rows[2])
	if u.help {
		u.drawHelp(f, area)
	}
}

// Key handles one key press. It returns false when the key asks to quit.
func (u *Viewer) Key(k driver.KeyEvent) bool {
	if u.help {
		u.help = false
		return true
	}
	if u.editing {
		switch k.Type {
		case driver.KeyEnter, driver.KeyEsc:
			u.editing = false
			if k.Type == driver.KeyEsc {
				u.filter.SetValue("")
			}
		default:
			u.filter.HandleKey(k)
		}
		u.view.setFilter(u.filter.Value(), u.minLevel)
		u.state.Follow, u.state.Selected = true, -1
		return true
	}
	if k.Type == driver.KeyRune && k.Ctrl && (k.Ch == 'c' || k.Ch == 'C') {
		return false
	}
	switch {
	case k.Type == driver.KeyEsc && (u.filter.Value() != "" || u.minLevel > widgets.LevelUnknown):
		u.clearFilter()
	case k.Type == driver.KeyEsc && u.details:
		u.details = false
	case k.Type == driver.KeyEsc, k.Type == driver.KeyRune && k.Ch == 'q':
		return false
	case k.Type == driver.KeyRune && k.Ch == '/':
		u.editing = true
	case k.Type == driver.KeyRune && k.Ch >= '1' && k.Ch <= '6':
		u.minLevel = levelKeys[k.Ch-'1']
		u.view.setFilter(u.filter.Value(), u.minLevel)
		u.state.Follow, u.state.Selected = true, -1
	case k.Type == driver.KeyEnter:
		u.details = !u.details
		if u.details && u.state.Selected < 0 && u.view.Len() > 0 {
			u.state.Select(u.view.Len() - 1)
		}
	case k.Type == driver.KeyRune && k.Ch == 'f':
		u.state.Follow = !u.state.Follow
		if u.state.Follow {
			u.state.Selected = -1
		}
	case k.Type == driver.KeyRune && k.Ch == '?':
		u.help = true
	default:
		u.state.HandleKey(k, u.view.Len())
	}
	return true
}

// levelKeys maps keys 1–6 to the minimum level shown: everything, then
// debug, info, warn, error and fatal and above.
var levelKeys = [6]widgets.LogLevel{widgets.LevelUnknown, widgets.LevelDebug, widgets.LevelInfo, widgets.LevelWarn, widgets.LevelError, widgets.LevelFatal}

// clearFilter shows every line again and keeps the reader on the line they
// had found, so its surroundings are right there: find the error with a
// filter, clear it, read what happened around it.
func (u *Viewer) clearFilter() {
	src := -1
	if sel := u.state.Selected; sel >= 0 && sel < u.view.Len() {
		src = u.view.source(sel)
	}
	u.filter.SetValue("")
	u.minLevel = widgets.LevelUnknown
	u.view.setFilter("", widgets.LevelUnknown)
	if src >= 0 {
		// Unfiltered, a line's position is its number. Put it in the middle
		// of the pane, with what came before and after around it.
		u.state.Select(src)
		u.state.Offset = max(0, src-u.paneHeight/2)
	}
}

var (
	headerStyle = cell.Style{Fg: cell.NewColorRGB(20, 20, 20), Bg: cell.NewColorRGB(250, 210, 60), Modifier: cell.ModifierBold}
	barStyle    = cell.Style{Fg: cell.NewColorRGB(200, 205, 215), Bg: cell.NewColorRGB(35, 38, 48)}
	dimStyle    = cell.Style{Fg: cell.NewColorRGB(130, 135, 150), Bg: cell.NewColorRGB(35, 38, 48)}
	warnStyle   = cell.Style{Fg: cell.NewColorRGB(240, 190, 60), Bg: cell.NewColorRGB(35, 38, 48)}
	errStyle    = cell.Style{Fg: cell.NewColorRGB(240, 90, 80), Bg: cell.NewColorRGB(35, 38, 48), Modifier: cell.ModifierBold}
)

func (u *Viewer) drawHeader(f *limoni.Frame, area limoni.Rect) {
	fill(f, area, barStyle)
	x := area.X
	put := func(text string, st cell.Style) {
		if x < area.X+area.Width {
			x += f.Buffer.SetStringWithin(x, area.Y, text, st, area.X+area.Width-x)
		}
	}
	put(" 🍋 zest ", headerStyle)
	put("  "+u.name, barStyle)

	total := u.src.Len()
	now := time.Now()
	if dt := now.Sub(u.rateAt); dt >= time.Second {
		u.rate = float64(total-u.rateLines) / dt.Seconds()
		u.rateLines, u.rateAt = total, now
	}
	put(fmt.Sprintf("  %s lines · %s", thousands(total), humanBytes(u.src.bytes())), dimStyle)
	if u.rate >= 1 {
		put(fmt.Sprintf(" · %s/s", thousands(int(u.rate))), dimStyle)
	}
	counts := u.src.Counts()
	if n := counts[widgets.LevelError] + counts[widgets.LevelFatal]; n > 0 {
		put(fmt.Sprintf("  ✖ %s", thousands(n)), errStyle)
	}
	if n := counts[widgets.LevelWarn]; n > 0 {
		put(fmt.Sprintf("  ▲ %s", thousands(n)), warnStyle)
	}
	state := "  ⏸ paused"
	if u.state.Follow {
		state = "  ● following"
	}
	put(state, dimStyle)
}

func (u *Viewer) drawFooter(f *limoni.Frame, area limoni.Rect) {
	fill(f, area, barStyle)
	if u.editing {
		f.Buffer.SetString(area.X, area.Y, " / ", headerStyle)
		f.RenderWidget(&widgets.TextInput{ID: "filter", State: u.filter, Placeholder: "filter", Focused: true, Style: barStyle}, limoni.NewRect(area.X+3, area.Y, area.Width-3, 1))
		return
	}
	var parts []string
	if q := u.filter.Value(); q != "" {
		parts = append(parts, fmt.Sprintf("/%s", q))
	}
	if u.minLevel > widgets.LevelUnknown {
		parts = append(parts, "level ≥ "+u.minLevel.String())
	}
	status := ""
	if len(parts) > 0 {
		scanned, total, done := u.view.progress()
		status = fmt.Sprintf(" %s · %s of %s", strings.Join(parts, " · "), thousands(u.view.Len()), thousands(total))
		if !done && total > 0 {
			status += fmt.Sprintf(" · scanning %d%%", scanned*100/total)
		}
		status += "  "
	}
	x := area.X + f.Buffer.SetString(area.X, area.Y, status, warnStyle)
	f.Buffer.SetStringWithin(x, area.Y, " / filter  1-6 level  ⏎ details  f follow  ? help  q quit", dimStyle, area.X+area.Width-x)
}

func (u *Viewer) drawDetails(f *limoni.Frame, area limoni.Rect) {
	text, title := "Select a line.", " details "
	if sel := u.state.Selected; sel >= 0 && sel < u.view.Len() {
		src := u.view.source(sel)
		text = u.detail.get(src, u.src.Line(src))
		title = fmt.Sprintf(" line %s ", thousands(src+1))
	}
	f.RenderWidget(widgets.Block{
		Title: title, Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: cell.NewColorRGB(250, 210, 60)},
		Child:       &widgets.Paragraph{ID: "details", Text: text, Wrap: true},
	}, area)
}

const helpText = `/          filter: show lines containing text (Enter keeps it, Esc clears)
1 … 6      minimum level: all, debug, info, warn, error, fatal
Enter      details: the selected line's fields, pretty-printed
f          follow new lines on and off (scrolling up pauses it)
↑ ↓ j k    move · PgUp PgDn page · Home g top · End G bottom
← →        scroll sideways
Esc        clear filters (staying on the line), close details, quit
q          quit`

func (u *Viewer) drawHelp(f *limoni.Frame, area limoni.Rect) {
	w, h := uint16(74), uint16(12)
	if w > area.Width {
		w = area.Width
	}
	if h > area.Height {
		h = area.Height
	}
	box := limoni.NewRect(area.X+(area.Width-w)/2, area.Y+(area.Height-h)/2, w, h)
	f.RenderWidget(widgets.Block{
		Title: " zest keys ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		Style: cell.Style{Bg: cell.NewColorRGB(25, 27, 35)}, BorderStyle: cell.Style{Fg: cell.NewColorRGB(250, 210, 60)},
		PaddingLeft: 1, Child: &widgets.Paragraph{ID: "help", Text: helpText},
	}, box)
}

func fill(f *limoni.Frame, area limoni.Rect, st cell.Style) {
	for x := area.X; x < area.X+area.Width; x++ {
		f.Buffer.SetCell(x, area.Y, cell.Cell{Content: ' ', Style: st})
	}
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func humanBytes(n int) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
