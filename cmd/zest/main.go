// Command zest is a terminal log viewer: it opens files or reads a pipe,
// follows them as they grow, colours lines by level, and filters millions of
// lines without freezing — built on Limoni as its flagship application.
//
//	zest app.log                      # open and follow a file
//	kubectl logs -f pod | zest        # read a pipe; keys still work
//	zest -level warn -filter timeout app.log
//	zest -demo 1000000                # a generated log, for trying it out
//
// Keys: / filter · 1–6 minimum level · Enter details · f follow · ↑↓ PgUp PgDn
// Home End scroll · ←→ scroll sideways · ? help · q quit.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "zest:", err)
		os.Exit(1)
	}
}

func run() error {
	follow := flag.Bool("f", true, "keep reading as the input grows, like tail -f")
	level := flag.String("level", "", "show only lines at this level or above: trace, debug, info, warn, error, fatal")
	filter := flag.String("filter", "", "show only lines containing this text (case-insensitive)")
	demo := flag.Int("demo", 0, "view a generated log of this many lines, still growing, instead of a file")
	socket := flag.String("socket", "", "serve the UI to agents and tests on this Unix socket (needs a -tags limoni_debug build)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: zest [flags] [file]\n\nWith no file, zest reads standard input.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	minLevel, err := parseLevel(*level)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	s := &store{}
	v := &view{src: s}

	// Input: a file, a demo, or standard input. When standard input is a pipe
	// the keyboard is the controlling terminal, opened separately.
	name := "stdin"
	keyboard := os.Stdin
	var input io.Reader
	switch {
	case *demo > 0:
		name = fmt.Sprintf("demo (%d lines)", *demo)
	case flag.NArg() > 0:
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			return err
		}
		defer f.Close()
		input, name = f, filepath.Base(flag.Arg(0))
	default:
		if isTerminal(os.Stdin) {
			flag.Usage()
			return errors.New("give a file, pipe something in, or try -demo 1000000")
		}
		input = os.Stdin
		tty, err := openTTY()
		if err != nil {
			return fmt.Errorf("stdin is a pipe and the terminal cannot be opened for keys: %w", err)
		}
		defer tty.Close()
		keyboard = tty
	}

	b := driver.NewBackend(keyboard, os.Stdout)
	if err := b.Setup(); err != nil {
		return err
	}
	term, err := terminal.New(b)
	if err != nil {
		_ = b.Close()
		return err
	}
	defer term.Close()

	opts := []limoni.AppOption{limoni.WithoutDefaultQuitKeys()}
	if *socket != "" {
		opts = append(opts, limoni.WithAutomation(*socket, limoni.AutomationPolicy{AllowInput: true, ExposeScreen: true, ExposeInputValues: true}))
	}
	app := limoni.NewApp(term, opts...)
	v.wake = app.Wakeup
	notify := func() {
		v.kick()
		app.Wakeup()
	}

	readErr := make(chan error, 1)
	if *demo > 0 {
		go func() { readErr <- generateDemo(ctx, s, *demo, notify) }()
	} else {
		go func() { readErr <- readLines(ctx, input, s, notify, *follow) }()
	}

	ui := newUI(name, s, v)
	ui.filter.SetValue(*filter)
	ui.minLevel = minLevel
	v.setFilter(*filter, minLevel)

	// Redraw now and then while following, so the rate and counts move even
	// when the new lines are filtered out.
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				app.Wakeup()
			}
		}
	}()

	err = app.Run(ctx, ui.frame)
	select {
	case rerr := <-readErr:
		if rerr != nil && !errors.Is(rerr, context.Canceled) && err == nil {
			err = rerr
		}
	default:
	}
	return err
}

func parseLevel(s string) (widgets.LogLevel, error) {
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

// ui is the application's state between frames.
type ui struct {
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

func newUI(name string, s *store, v *view) *ui {
	st := widgets.NewLogViewState()
	u := &ui{name: name, src: s, view: v, state: st, filter: widgets.NewTextInputState(), started: time.Now(), rateAt: time.Now()}
	u.log = &widgets.LogView{ID: "log", Label: "Log", Source: v, State: st, LineNumbers: true}
	return u
}

func (u *ui) frame(f *limoni.Frame, ev *limoni.Event) bool {
	if ev != nil && ev.Type == limoni.EventKey && !u.key(ev.Key) {
		return false
	}
	u.log.Highlight = u.filter.Value()

	area := f.Area()
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
	return true
}

// key handles one key press; false quits.
func (u *ui) key(k driver.KeyEvent) bool {
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
func (u *ui) clearFilter() {
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

func (u *ui) drawHeader(f *limoni.Frame, area limoni.Rect) {
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

func (u *ui) drawFooter(f *limoni.Frame, area limoni.Rect) {
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

func (u *ui) drawDetails(f *limoni.Frame, area limoni.Rect) {
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

func (u *ui) drawHelp(f *limoni.Frame, area limoni.Rect) {
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

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
