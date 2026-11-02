package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

type mode int

const (
	setupMode mode = iota
	raceMode
	compareMode
)

const sidebar = 28

var (
	yellow = limoni.Style{Fg: limoni.Hex("#FFCC00")}
	dim    = limoni.Style{Modifier: limoni.ModifierDim}
	bold   = limoni.Style{Modifier: limoni.ModifierBold}
	green  = limoni.Style{Fg: limoni.Hex("#98C379")}
	red    = limoni.Style{Fg: limoni.Hex("#E06C75")}
	picked = limoni.Style{Bg: limoni.Hex("#3A3A46")}
)

// ui is the whole application: the task, the racers, and which screen is
// up. It is only touched by the draw loop; racers report through wake.
type ui struct {
	root, base, raceID string
	wake               func()
	notify             func(title, body string)

	mode   mode
	prompt string
	input  *widgets.TextAreaState
	agents []agentDef
	chosen []bool

	racers   []*racer
	sel      int
	attached bool
	paneW    int
	paneH    int

	// compare
	files   []fileChange
	fileSel int
	diff    *widgets.DiffViewState
	diffFor string

	confirm string // a question waiting for y or n: "quit" or "merge"
	status  string
	quit    bool
}

func newUI(root, base, prompt string, agents []agentDef) *ui {
	u := &ui{root: root, base: base, prompt: prompt, agents: agents, input: widgets.NewTextAreaState(),
		raceID: time.Now().Format("20060102-150405")}
	u.chosen = make([]bool, len(agents))
	for i := range u.chosen {
		u.chosen[i] = true
	}
	if prompt == "" {
		u.mode = setupMode
	} else {
		u.mode = raceMode
	}
	return u
}

// frame draws one frame for ev and reports whether to keep running.
func (u *ui) frame(f *limoni.Frame, ev *limoni.Event) bool {
	if ev != nil {
		switch ev.Type {
		case limoni.EventKey:
			u.key(ev.Key)
		case driver.EventPaste:
			if u.attached {
				u.racers[u.sel].emu.Paste(ev.Paste.Text)
				u.racers[u.sel].userInput()
			} else if u.mode == setupMode {
				u.input.SetValue(u.input.Value() + ev.Paste.Text)
			}
		}
	}
	if u.quit {
		return false
	}
	switch u.mode {
	case setupMode:
		u.drawSetup(f)
	case raceMode:
		u.drawRace(f)
	case compareMode:
		u.drawCompare(f)
	}
	return true
}

func (u *ui) key(k limoni.KeyEvent) {
	plain := !k.Ctrl && !k.Alt && !k.Shift
	if u.attached {
		if isDetach(k) {
			u.attached = false
			return
		}
		if ev, ok := toEmulator(k); ok {
			u.racers[u.sel].emu.SendKey(ev)
			u.racers[u.sel].userInput()
		}
		return
	}
	if k.Ctrl && (k.Ch == 'c' || k.Ch == 'C') {
		u.askQuit()
		return
	}
	if u.confirm != "" {
		yes := plain && (k.Ch == 'y' || k.Type == limoni.KeyEnter)
		question := u.confirm
		u.confirm = ""
		u.status = ""
		if yes && question == "quit" {
			u.stop()
		}
		if yes && question == "merge" {
			u.mergeSelected()
		}
		return
	}
	switch u.mode {
	case setupMode:
		u.setupKey(k, plain)
	case raceMode:
		u.raceKey(k, plain)
	case compareMode:
		u.compareKey(k, plain)
	}
}

// ── setup ─────────────────────────────────────────────────────────────

func (u *ui) setupKey(k limoni.KeyEvent, plain bool) {
	switch {
	case k.Type == limoni.KeyEsc:
		u.quit = true
	case k.Type == limoni.KeyEnter && plain:
		u.prompt = strings.TrimSpace(u.input.Value())
		var picked []agentDef
		for i, a := range u.agents {
			if u.chosen[i] {
				picked = append(picked, a)
			}
		}
		switch {
		case u.prompt == "":
			u.status = "Write the task first."
		case len(picked) == 0:
			u.status = "Pick at least one agent (F1…F9)."
		default:
			u.agents = picked
			u.mode = raceMode
		}
	case k.Type >= limoni.KeyF1 && k.Type <= limoni.KeyF9:
		if i := int(k.Type - limoni.KeyF1); i < len(u.chosen) {
			u.chosen[i] = !u.chosen[i]
		}
	default:
		u.input.HandleKey(k)
	}
}

func (u *ui) drawSetup(f *limoni.Frame) {
	a := f.Area()
	w := min(a.Width, 90)
	x := a.X + (a.Width-w)/2
	y := a.Y + 1
	f.RenderWidget(&limoni.Paragraph{Text: "squeeze", Style: yellow.Merge(bold)}, limoni.NewRect(x, y, w, 1))
	f.RenderWidget(&limoni.Paragraph{Text: "Give one task to several agents, each in its own worktree. Compare what they did; keep the best.",
		Style: dim}, limoni.NewRect(x, y+1, w, 2))
	y += 4
	f.RenderWidget(&limoni.Block{
		Title: " Task ", Borders: limoni.BorderAll, BorderSymbols: limoni.SymbolsRounded, BorderStyle: yellow,
		Child: &widgets.TextArea{ID: "task", State: u.input, Focused: true,
			Placeholder: "What should they do? Shift+Enter for a new line"},
	}, limoni.NewRect(x, y, w, 8))
	y += 9
	f.RenderWidget(&limoni.Paragraph{Text: "Agents", Style: bold}, limoni.NewRect(x, y, w, 1))
	y++
	if len(u.agents) == 0 {
		f.RenderWidget(&limoni.Paragraph{Text: "None found on PATH. Install claude, codex, agy or gemini.", Style: red},
			limoni.NewRect(x, y, w, 1))
	}
	for i, ag := range u.agents {
		mark := "[ ]"
		if u.chosen[i] {
			mark = "[x]"
		}
		f.RenderWidget(&limoni.Paragraph{Text: fmt.Sprintf("F%d %s %s", i+1, mark, ag.Name)},
			limoni.NewRect(x, y+uint16(i), w, 1))
	}
	u.footer(f, "Enter starts the race · F1…F9 pick agents · Esc quits")
}

// ── race ──────────────────────────────────────────────────────────────

// start opens a worktree for each agent and starts it, on a terminal the
// size of the pane.
func (u *ui) start() {
	if err := excludeSqueeze(u.root); err != nil {
		u.status = err.Error()
	}
	for i, name := range racerNames(u.agents) {
		dir := filepath.Join(worktreesDir(u.root), name) // the same path every race: agents remember trust per path
		branch := "squeeze/" + u.raceID + "/" + name
		if err := reuseWorktree(u.root, dir, branch, u.base); err != nil {
			u.status = err.Error()
			continue
		}
		if err := linkIgnored(u.root, dir); err != nil {
			u.status = name + ": " + err.Error()
		}
		r, err := startRacer(u.agents[i], name, u.prompt, dir, branch, u.paneW, u.paneH, u.wake)
		if err != nil {
			u.status = name + ": " + err.Error()
			continue
		}
		u.racers = append(u.racers, r)
	}
	go u.watch()
}

// watch keeps each racer's list of changes current.
func (u *ui) watch() {
	for {
		for _, r := range u.racers {
			r.refresh(u.base)
		}
		u.wake()
		time.Sleep(2 * time.Second)
	}
}

func (u *ui) raceKey(k limoni.KeyEvent, plain bool) {
	switch {
	case k.Type == limoni.KeyUp || (plain && k.Ch == 'k'):
		u.sel = max(u.sel-1, 0)
	case k.Type == limoni.KeyDown || (plain && k.Ch == 'j'):
		u.sel = min(u.sel+1, max(len(u.racers)-1, 0)) // no racers yet: stay at 0, never -1
	case k.Type == limoni.KeyEnter && len(u.racers) > 0:
		u.attached = true
	case plain && k.Ch == 'c' && len(u.racers) > 0:
		u.mode = compareMode
		u.diffFor = ""
	case plain && k.Ch == 'q':
		u.askQuit()
	}
}

func (u *ui) drawRace(f *limoni.Frame) {
	a := f.Area()
	u.paneW, u.paneH = int(a.Width)-sidebar-1, int(a.Height)-2
	if u.racers == nil {
		u.racers = []*racer{}
		u.start()
	}
	for _, r := range u.racers {
		r.resize(u.paneW, u.paneH)
	}
	now := time.Now()
	u.header(f)
	for i, r := range u.racers {
		u.drawRacer(f, r, i, a.X, a.Y+1+uint16(i*3), now)
	}
	for y := a.Y + 1; y < a.Y+a.Height-1; y++ {
		f.RenderWidget(&limoni.Paragraph{Text: "│", Style: dim}, limoni.NewRect(a.X+sidebar, y, 1, 1))
	}
	if len(u.racers) > 0 {
		f.RenderWidget(pane{r: u.racers[u.sel], cursor: u.attached},
			limoni.NewRect(a.X+sidebar+1, a.Y+1, uint16(u.paneW), uint16(u.paneH)))
	}
	switch {
	case u.attached:
		u.footer(f, "Typing into "+u.racers[u.sel].name+" · Ctrl+] gives the keys back to squeeze")
	default:
		u.footer(f, "↑↓ pick · Enter type into it · c compare · q quit")
	}
	u.notifyWaiting(now)
}

func (u *ui) drawRacer(f *limoni.Frame, r *racer, i int, x, y uint16, now time.Time) {
	style := limoni.Style{}
	if i == u.sel {
		style = picked
	}
	dot, dotStyle := "●", green
	switch r.state(now) {
	case waiting:
		dot, dotStyle = "◐", yellow
	case done:
		dot, dotStyle = "✓", dim
	}
	d := r.elapsed(now).Round(time.Second)
	files, added, removed := r.changes()
	f.RenderWidget(&limoni.Paragraph{Text: fmt.Sprintf("  %-17s %6s", r.name, clock(d)), Style: style.Merge(bold)},
		limoni.NewRect(x, y, sidebar, 1))
	f.RenderWidget(&limoni.Paragraph{Text: dot, Style: style.Merge(dotStyle)}, limoni.NewRect(x, y, 1, 1))
	f.RenderWidget(&limoni.Paragraph{Text: fmt.Sprintf("  +%d −%d · %d files", added, removed, len(files)), Style: style.Merge(dim)},
		limoni.NewRect(x, y+1, sidebar, 1))
}

// notifyWaiting tells the user, once, when an agent they are not looking at
// starts waiting for them.
func (u *ui) notifyWaiting(now time.Time) {
	for i, r := range u.racers {
		w := r.state(now) != working
		if w && !r.notified && !(u.attached && i == u.sel) {
			msg := r.name + " is waiting for you"
			if a := r.ask(); a != "" {
				msg = r.name + ": " + a
			}
			u.notify("squeeze", msg)
		}
		r.notified = w
	}
}

func clock(d time.Duration) string {
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

// ── compare ───────────────────────────────────────────────────────────

func (u *ui) compareKey(k limoni.KeyEvent, plain bool) {
	switch {
	case k.Type == limoni.KeyEsc || (plain && k.Ch == 'c'):
		u.mode = raceMode
	case k.Type == limoni.KeyLeft || k.Type == limoni.KeyTab && k.Shift:
		u.sel = max(u.sel-1, 0)
		u.fileSel = 0
	case k.Type == limoni.KeyRight || k.Type == limoni.KeyTab:
		u.sel = min(u.sel+1, len(u.racers)-1)
		u.fileSel = 0
	case k.Type == limoni.KeyUp:
		u.fileSel = max(u.fileSel-1, 0)
	case k.Type == limoni.KeyDown:
		u.fileSel = min(u.fileSel+1, max(len(u.files)-1, 0))
	case k.Type == limoni.KeyPageDown && u.diff != nil:
		u.diff.Offset += 10
	case k.Type == limoni.KeyPageUp && u.diff != nil:
		u.diff.Offset = max(u.diff.Offset-10, 0)
	case plain && k.Ch == 'm':
		u.confirm = "merge"
		u.status = fmt.Sprintf("Merge %s's changes into your working tree (staged, not committed)? y / n", u.racers[u.sel].name)
	case plain && k.Ch == 'q':
		u.askQuit()
	}
}

func (u *ui) drawCompare(f *limoni.Frame) {
	a := f.Area()
	now := time.Now()
	u.header(f)
	for i, r := range u.racers {
		u.drawRacer(f, r, i, a.X, a.Y+1+uint16(i*3), now)
	}
	r := u.racers[u.sel]
	u.files, _, _ = r.changes()
	x, w := a.X+sidebar+1, a.Width-sidebar-1
	y := a.Y + 1
	if len(u.files) == 0 {
		f.RenderWidget(&limoni.Paragraph{Text: r.name + " has not changed anything yet.", Style: dim}, limoni.NewRect(x, y, w, 1))
		u.footer(f, "←→ agent · Esc back")
		return
	}
	u.fileSel = min(u.fileSel, len(u.files)-1)
	listRows := min(len(u.files), 8)
	first := max(0, u.fileSel-listRows+1)
	for i := 0; i < listRows; i++ {
		fc := u.files[first+i]
		style := limoni.Style{}
		if first+i == u.fileSel {
			style = picked
		}
		stat := "binary"
		if fc.Added >= 0 {
			stat = fmt.Sprintf("+%d −%d", fc.Added, fc.Removed)
		}
		f.RenderWidget(&limoni.Paragraph{Text: fmt.Sprintf(" %-12s %s", stat, fc.Path), Style: style},
			limoni.NewRect(x, y+uint16(i), w, 1))
	}
	y += uint16(listRows) + 1

	fc := u.files[u.fileSel]
	if key := r.name + "\x00" + fc.Path; key != u.diffFor {
		before, after := versions(r.dir, u.base, fc.Path)
		u.diff = widgets.NewDiffViewState(before, after, widgets.LanguageForFile(fc.Path), 3)
		u.diffFor = key
	}
	f.RenderWidget(widgets.DiffView{ID: "diff", State: u.diff}, limoni.NewRect(x, y, w, a.Y+a.Height-1-y))
	u.footer(f, "←→ agent · ↑↓ file · PgUp/PgDn scroll · m merge this one · Esc back")
}

// ── shared ────────────────────────────────────────────────────────────

func (u *ui) header(f *limoni.Frame) {
	a := f.Area()
	task := strings.ReplaceAll(u.prompt, "\n", " ")
	f.RenderWidget(&limoni.Paragraph{Text: " squeeze ", Style: limoni.Style{Fg: limoni.Hex("#1E1E1E"), Bg: limoni.Hex("#FFCC00"), Modifier: limoni.ModifierBold}},
		limoni.NewRect(a.X, a.Y, 9, 1))
	f.RenderWidget(&limoni.Paragraph{Text: " " + task}, limoni.NewRect(a.X+9, a.Y, a.Width-9, 1))
}

func (u *ui) footer(f *limoni.Frame, help string) {
	a := f.Area()
	text, style := help, dim
	if u.status != "" {
		text, style = u.status, yellow
	}
	f.RenderWidget(&limoni.Paragraph{ID: "status", Text: " " + text, Style: style}, limoni.NewRect(a.X, a.Y+a.Height-1, a.Width, 1))
}

func (u *ui) askQuit() {
	if len(u.racers) == 0 {
		u.quit = true
		return
	}
	u.confirm = "quit"
	u.status = "Quit? The agents stop; their worktrees stay in .squeeze/ (squeeze clean removes them). y / n"
}

func (u *ui) stop() {
	for _, r := range u.racers {
		r.stop()
	}
	u.quit = true
}

func (u *ui) mergeSelected() {
	r := u.racers[u.sel]
	if err := merge(u.root, r.dir, r.branch, r.name); err != nil {
		u.status = "Merge failed: " + err.Error()
		return
	}
	u.status = r.name + "'s changes are staged in " + u.root + ". Look them over with git diff --staged, then commit."
}
