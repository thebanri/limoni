package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

type mode int

const (
	setupMode mode = iota
	raceMode
	compareMode
)

const (
	sidebar = 30 // the racers' cards
	cardH   = 4  // rows per card
)

var (
	yellow = limoni.Style{Fg: limoni.Hex("#FFCC00")}
	dim    = limoni.Style{Modifier: limoni.ModifierDim}
	cyan   = limoni.Style{Fg: limoni.Hex("#7FD4E8")}
	bold   = limoni.Style{Modifier: limoni.ModifierBold}
	green  = limoni.Style{Fg: limoni.Hex("#98C379")}
	red    = limoni.Style{Fg: limoni.Hex("#E06C75")}
	picked = limoni.Style{Bg: limoni.Hex("#3A3A46")}
)

// ui is the whole application: the task, the racers, and which screen is
// up. It is only touched by the draw loop; racers report through wake.
type ui struct {
	root, base, raceID string
	check              string // -check: run in each worktree when its agent stops
	notes              bool   // -notes: racers on a job share notes (notes.go)
	wake               func()
	notify             func(title, body string)
	copy               func(text string) // to the user's clipboard (OSC 52)
	marked             selection         // text selected in an agent's screen

	mode   mode
	prompt string
	input  *widgets.TextAreaState
	agents []agentDef
	chosen []int // how many of each agent race

	racers   []*racer
	racersMu sync.Mutex   // racers grows in a race; watch reads it from its own goroutine
	others   []agentDef   // every agent squeeze knows, offered on the setup screen
	inst     installs     // those being installed from it
	pending  agentDef     // the one the install question is about
	jobs     []string     // the tasks under way; a racer works on one
	cardTop  int          // the first racer whose card is shown
	hits     []hit        // where the last frame drew cards and subagents, for clicks
	viewR    *racer       // whose subagent is shown instead of its screen
	viewI    int          // which of its subagents
	subOff   int          // lines scrolled up from the end of what it shows
	open     func(string) // opens a web address
	sel      int
	attached bool
	paneW    int
	paneH    int

	// compare
	files   []fileChange
	fileSel int
	diff    *widgets.DiffViewState
	diffFor string
	output  bool // the check's output is shown instead of the diff
	outOff  int  // lines scrolled up from its end
	listY   int  // where the drawn file list starts on screen, for clicks
	listTop int  // the index of its first row
	listN   int  // and how many rows it has

	confirm    string    // a question waiting for y or n, in the modal: "quit", "merge", "install"
	modalAt    time.Time // when the modal opened, for its animation
	stoppingAt time.Time // when quitting stopped the agents
	closingAt  time.Time // when the modal began to shut, once they had
	status     string
	statusAt   time.Time // status is shown for a while, then the help again
	buttons    []button  // where the last frame drew its buttons, for clicks
	quit       bool
}

func newUI(root, base, prompt string, agents []agentDef) *ui {
	u := &ui{root: root, base: base, prompt: prompt, agents: agents, input: widgets.NewTextAreaState(),
		raceID: time.Now().Format("20060102-150405")}
	u.chosen = make([]int, len(agents))
	for i := range u.chosen {
		u.chosen[i] = 1
	}
	if prompt == "" {
		u.mode = setupMode
	} else {
		u.mode = raceMode
		u.jobs = []string{prompt}
	}
	return u
}

// minWidth and minHeight are the smallest window squeeze draws in: the
// sidebar and a pane, the header, the frame and the footer.
const minWidth, minHeight = sidebar + 20, 10

// frame draws one frame for ev and reports whether to keep running.
func (u *ui) frame(f *limoni.Frame, ev *limoni.Event) bool {
	if ev != nil {
		switch ev.Type {
		case limoni.EventKey:
			u.key(ev.Key)
		case limoni.EventMouse:
			u.mouse(ev.Mouse)
		case driver.EventPaste:
			if u.attached {
				u.racers[u.sel].emu.Paste(ev.Paste.Text)
				u.racers[u.sel].userInput(false)
				u.racers[u.sel].scroll = 0
			} else if u.mode == setupMode {
				u.input.SetValue(u.input.Value() + ev.Paste.Text)
			}
		}
	}
	if u.quit {
		return false
	}
	u.buttons = u.buttons[:0]
	if a := f.Area(); a.Width < minWidth || a.Height < minHeight {
		// Smaller, and the agents' panes would have no room, or less than none.
		f.RenderWidget(&limoni.Paragraph{Text: fmt.Sprintf("squeeze needs %d×%d; make the window bigger", minWidth, minHeight),
			Style: yellow, Wrap: true}, a)
		return true
	}
	switch u.mode {
	case setupMode:
		u.drawSetup(f)
	case raceMode:
		u.drawRace(f)
	case compareMode:
		u.drawCompare(f)
	}
	u.closing()
	if u.confirm != "" || !u.stoppingAt.IsZero() {
		u.drawModal(f)
	}
	return true
}

func (u *ui) key(k limoni.KeyEvent) {
	plain := !k.Ctrl && !k.Alt && !k.Shift
	if !u.stoppingAt.IsZero() { // on the way out
		return
	}
	if u.attached {
		if isDetach(k) {
			u.attached = false
			return
		}
		r := u.racers[u.sel]
		seq := xtermKey(k)
		if seq != "" {
			r.emu.SendText(seq)
		} else if ev, ok := toEmulator(k); ok {
			r.emu.SendKey(ev)
		} else {
			return
		}
		r.userInput(k.Type == limoni.KeyEnter && seq == "") // a new line is not sending the prompt
		r.scroll = 0
		return
	}
	if k.Ctrl && (k.Ch == 'c' || k.Ch == 'C') {
		u.askQuit()
		return
	}
	if u.confirm != "" {
		u.answer(plain && (k.Ch == 'y' || k.Type == limoni.KeyEnter))
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
	case k.Type == limoni.KeyEsc && u.racers != nil:
		u.mode = raceMode
	case k.Type == limoni.KeyEsc:
		u.quit = true
	case k.Type == limoni.KeyEnter && plain:
		u.begin()
	case k.Type >= limoni.KeyF1 && k.Type <= limoni.KeyF9:
		if i := int(k.Type - limoni.KeyF1); i < len(u.chosen) {
			u.pick(i)
		}
	default:
		u.input.HandleKey(k)
	}
}

// pick races one more of agent i.
// One not installed is offered to be installed instead.
func (u *ui) pick(i int) {
	if def := u.choices()[i]; !onPath(def.Bin) {
		u.askInstall(def)
		return
	}
	u.chosen[i]++
}

// choices is the agents the setup screen offers: those found or named
// before the race, every one installed during it.
func (u *ui) choices() []agentDef {
	if len(u.others) > 0 {
		return u.others
	}
	return u.agents
}

// offer puts every agent squeeze knows on the setup screen, installed or
// not; before the race, those installed start picked.
func (u *ui) offer(defs []agentDef) {
	u.others = defs
	u.chosen = make([]int, len(defs))
	if u.racers == nil {
		for i, d := range defs {
			if onPath(d.Bin) {
				u.chosen[i] = 1
			}
		}
	}
}

// begin starts what the setup screen says: the race, or during it a new
// job, or more agents on a job under way when its task is given again.
func (u *ui) begin() {
	task := strings.TrimSpace(u.input.Value())
	var picked []agentDef
	for i, a := range u.choices() {
		for range u.chosen[i] {
			picked = append(picked, a)
		}
	}
	switch {
	case task == "":
		u.say("Write the task first.")
		return
	case len(picked) == 0:
		u.say("Pick at least one agent (F1…F9 or a click).")
		return
	}
	u.mode = raceMode
	if u.racers == nil { // the race starts with the first frame
		u.prompt, u.jobs, u.agents = task, []string{task}, picked
		return
	}
	job := slices.Index(u.jobs, task)
	if job < 0 {
		job = len(u.jobs)
		u.jobs = append(u.jobs, task)
	}
	for _, def := range picked {
		u.add(def, job)
	}
}

// newJob opens the setup screen during the race, on the task of the racer
// shown: keep it to add agents to its job, or write another for a new one.
func (u *ui) newJob() {
	u.mode, u.attached, u.viewR = setupMode, false, nil
	u.input.SetValue(u.racers[u.sel].task)
	u.chosen = make([]int, len(u.choices()))
}

// startStyle is the button that starts what the setup screen says.
var startStyle = limoni.Style{Fg: ink, Bg: limoni.Hex("#FFCC00"), Modifier: limoni.ModifierBold}

// drawSetup draws the task and the agents to race on it: the whole screen
// before the race, a panel beside the cards during it, for a new job.
func (u *ui) drawSetup(f *limoni.Frame) {
	a := f.Area()
	inRace := u.racers != nil
	title, about := " New race ", "One task, several agents, each in its own worktree. Compare what they did; keep the best."
	cancel := btn{"Esc", "Quit", func() { u.quit = true }}
	brand(f)
	var box limoni.Rect
	if inRace {
		u.drawCards(f, time.Now())
		box = limoni.NewRect(a.X+sidebar, a.Y+1, min(a.Width-sidebar, 100), a.Height-2)
		title = " New job "
		about = "A new task starts a new job; the task of a job under way adds agents to it."
		cancel = btn{"Esc", "Cancel", func() { u.mode = raceMode }}
	} else {
		w := min(a.Width-2, 100)
		box = limoni.NewRect(a.X+(a.Width-w)/2, a.Y+1, w, a.Height-2)
	}
	// As tall as what it holds: the line about it, the task, the agents,
	// the jobs under way, the buttons; no taller than the window.
	need := 1 + 1 + 6 + 1 + 1 + u.agentCardsHeight(box.Width-4) + 1 + 1 + 2
	if inRace {
		need += 2 + len(u.jobs)
	}
	box.Height = min(box.Height, uint16(need))
	f.RenderWidget(&limoni.Block{Title: title, TitleStyle: startStyle, Borders: limoni.BorderAll,
		BorderSymbols: limoni.SymbolsRounded, BorderStyle: yellow}, box)
	x, w := box.X+2, box.Width-4
	y, bottom := box.Y+1, box.Y+box.Height-2 // the last row inside holds the buttons
	line := func(text string, st limoni.Style, ix, iw uint16) {
		if y < bottom {
			text, _ = cell.Truncate(text, int(iw))
			f.RenderWidget(&limoni.Paragraph{Text: text, Style: st}, limoni.NewRect(ix, y, iw, 1))
		}
	}
	if int(bottom)-int(y) >= 2+3+1+1+u.agentCardsHeight(w) { // in a short window the cards come first
		line(about, dim, x, w)
		y += 2
	}
	// The task box gives way to the agents' rows in a short window.
	taskH := uint16(max(min(int(bottom)-int(y)-2-u.agentCardsHeight(w), 6), 3))
	if y+taskH < bottom {
		f.RenderWidget(&limoni.Block{
			Title: " Task ", TitleStyle: bold, Borders: limoni.BorderAll, BorderSymbols: limoni.SymbolsRounded, BorderStyle: dim,
			Child: &widgets.TextArea{ID: "task", State: u.input, Focused: u.confirm == "",
				Placeholder: "What should they do? Shift+Enter for a new line"},
		}, limoni.NewRect(x, y, w, taskH))
	}
	y += taskH + 1

	line("Agents", bold, x, 7)
	line("a click on a card or F1… adds one; − takes one away", dim, x+8, w-8)
	y++
	if y+uint16(u.agentCardsHeight(w)) <= bottom {
		u.drawAgentCards(f, x, y, w)
	}
	y += uint16(u.agentCardsHeight(w))
	if inRace && y+2 < bottom {
		y++
		line("Jobs under way", bold, x, 15)
		line("a click puts its task above", dim, x+16, w-16)
		y++
		for j, task := range u.jobs {
			if y >= bottom {
				break
			}
			line(fmt.Sprintf("#%d", j+1), yellow, x+1, 3)
			line(strings.ReplaceAll(task, "\n", " "), limoni.Style{}, x+5, w-5)
			u.buttons = append(u.buttons, button{x: int(x), y: int(y), w: int(w), act: func() { u.input.SetValue(task) }})
			y++
		}
	}
	// Start and the way out, at the panel's bottom right.
	start := " Start ▸ "
	sw := cell.StringWidth(start)
	bx := int(box.X+box.Width) - 2 - sw
	f.RenderWidget(&limoni.Paragraph{Text: start, Style: startStyle}, limoni.NewRect(uint16(bx), bottom, uint16(sw), 1))
	u.buttons = append(u.buttons, button{x: bx, y: int(bottom), w: sw, act: u.begin})
	bx -= keyButtonWidth(cancel) + 3
	u.keyButton(f, bx, bottom, cancel, nil)
	if inRace {
		u.footer(f, "Enter starts · F1… picks agents · Esc cancels")
	} else {
		u.footer(f, "Enter starts the race · F1… picks agents · Esc quits")
	}
}

// agentCardH is the height of an agent's card on the setup screen.
const agentCardH = 6

// agentCardCols is how many agent cards go in a row w columns wide: all of
// them when they fit at 15 columns or more each.
func agentCardCols(w uint16, n int) int { return max(1, min(n, (int(w)+1)/16)) }

// agentCardsHeight is the rows the agent cards take in a panel w wide.
func (u *ui) agentCardsHeight(w uint16) int {
	n := len(u.choices())
	if n == 0 {
		return 0
	}
	cols := agentCardCols(w, n)
	rows := (n + cols - 1) / cols
	return rows*agentCardH + rows - 1
}

// symbolsDashed is the border of an agent not installed yet.
var symbolsDashed = widgets.BorderSymbols{Horizontal: '╌', Vertical: '╎', TopLeft: '╭', TopRight: '╮', BottomLeft: '╰', BottomRight: '╯'}

// tint is c faded towards the terminal's dark background, for the inside
// of a picked card.
func tint(hex string) limoni.Color {
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return picked.Bg
	}
	mix := func(v, bg int) uint8 { return uint8(bg + (v-bg)*18/100) }
	return limoni.RGB(mix(r, 0x1E), mix(g, 0x1E), mix(b, 0x1E))
}

// drawAgentCards draws a card per agent on the setup screen: its name and
// what it is, and at the bottom, centred, how many of it race, or a way to
// install it. A click on a card adds one; picked cards are drawn heavy, in
// the agent's colour, those not installed dashed.
func (u *ui) drawAgentCards(f *limoni.Frame, x, y, w uint16) {
	agents := u.choices()
	cols := agentCardCols(w, len(agents))
	cw := (int(w) - (cols - 1)) / cols
	for i, ag := range agents {
		cx := x + uint16(i%cols*(cw+1))
		cy := y + uint16(i/cols*(agentCardH+1))
		inner := uint16(cw - 2)
		c := limoni.Hex(ag.Color)
		ready := onPath(ag.Bin)
		n := u.chosen[i]
		state := u.inst.get(ag.Name)

		syms, border, fill := limoni.SymbolsRounded, limoni.Style{Fg: c, Modifier: limoni.ModifierDim}, limoni.Style{}
		switch {
		case n > 0:
			syms, border, fill = limoni.SymbolsThick, limoni.Style{Fg: c}, limoni.Style{Bg: tint(ag.Color)}
			border.Bg = fill.Bg
		case !ready:
			syms, border = symbolsDashed, dim
		}
		on := func(st limoni.Style) limoni.Style { st.Bg = fill.Bg; return st }
		f.RenderWidget(&limoni.Block{Borders: limoni.BorderAll, BorderSymbols: syms, BorderStyle: border},
			limoni.NewRect(cx, cy, uint16(cw), agentCardH))
		if n > 0 { // after the block, which clears its inside
			for r := uint16(1); r < agentCardH-1; r++ {
				f.RenderWidget(&limoni.Paragraph{Text: " ", Style: fill}, limoni.NewRect(cx+1, cy+r, inner, 1))
			}
		}
		key := fmt.Sprintf(" F%d ", i+1)
		f.RenderWidget(&limoni.Paragraph{Text: key, Style: on(dim)}, limoni.NewRect(cx+uint16(cw)-1-uint16(len(key)), cy, uint16(len(key)), 1))

		name := limoni.Style{Fg: c, Modifier: limoni.ModifierBold}
		lines := [2]string{ag.Blurb[0], ag.Blurb[1]}
		lineStyle := [2]limoni.Style{dim, dim}
		if !ready {
			name = limoni.Style{Fg: c}
			lines = [2]string{"not installed", ag.Blurb[0]}
		}
		if strings.HasPrefix(state, "failed") {
			lines[1], lineStyle[1] = strings.TrimPrefix(state, "failed: "), red
		}
		f.RenderWidget(&limoni.Paragraph{Text: ag.Name, Style: on(name)}, limoni.NewRect(cx+2, cy+1, inner-1, 1))
		if n > 0 {
			count := fmt.Sprintf("×%d", n)
			f.RenderWidget(&limoni.Paragraph{Text: count, Style: on(bold)}, limoni.NewRect(cx+uint16(cw)-2-uint16(cell.StringWidth(count)), cy+1, uint16(cell.StringWidth(count)), 1))
		}
		for l, text := range lines {
			text, _ = cell.Truncate(text, int(inner)-1)
			f.RenderWidget(&limoni.Paragraph{Text: text, Style: on(lineStyle[l])}, limoni.NewRect(cx+2, cy+2+uint16(l), inner-1, 1))
		}

		// The control, centred on the card's last row inside.
		ctl := func(text string, st limoni.Style) uint16 {
			tw := uint16(cell.StringWidth(text))
			at := cx + 1 + (inner-tw)/2
			f.RenderWidget(&limoni.Paragraph{Text: text, Style: on(st)}, limoni.NewRect(at, cy+4, tw, 1))
			return at
		}
		switch {
		case ready && n > 0:
			text := "−   " + fmt.Sprint(n) + "   +"
			at := ctl(text, bold)
			plus := at + uint16(cell.StringWidth(text)) - 1
			f.RenderWidget(&limoni.Paragraph{Text: "−", Style: on(yellow.Merge(bold))}, limoni.NewRect(at, cy+4, 1, 1))
			f.RenderWidget(&limoni.Paragraph{Text: "+", Style: on(yellow.Merge(bold))}, limoni.NewRect(plus, cy+4, 1, 1))
			u.buttons = append(u.buttons,
				button{x: int(at) - 1, y: int(cy) + 4, w: 3, act: func() { u.chosen[i] = max(u.chosen[i]-1, 0) }},
				button{x: int(plus) - 1, y: int(cy) + 4, w: 3, act: func() { u.chosen[i]++ }})
		case ready:
			at := ctl("+ add", dim)
			u.buttons = append(u.buttons, button{x: int(at), y: int(cy) + 4, w: 5, act: func() { u.pick(i) }})
		case state == "installing":
			ctl("installing…", yellow)
		case strings.HasPrefix(state, "failed"):
			at := ctl("✗ retry", red.Merge(limoni.Style{Modifier: limoni.ModifierUnderline}))
			u.buttons = append(u.buttons, button{x: int(at), y: int(cy) + 4, w: 7, act: func() { u.askInstall(ag) }})
		case ag.Install != "":
			at := ctl("↓ install", yellow.Merge(limoni.Style{Modifier: limoni.ModifierUnderline}))
			u.buttons = append(u.buttons, button{x: int(at), y: int(cy) + 4, w: 9, act: func() { u.askInstall(ag) }})
		default:
			ctl("see its site", dim)
		}
		// The rest of the card adds one, or offers the install.
		u.buttons = append(u.buttons, button{x: int(cx), y: int(cy), w: cw, h: agentCardH, act: func() { u.pick(i) }})
	}
}

// ── race ──────────────────────────────────────────────────────────────

// start opens a worktree for each agent and starts it, on a terminal the
// size of the pane.
func (u *ui) start() {
	if err := excludeSqueeze(u.root); err != nil {
		u.say(err.Error())
	}
	if err := writeAgyHooks(filepath.Dir(worktreesDir(u.root))); err != nil {
		u.say("agy's hooks: " + err.Error())
	}
	if err := writeOpencodePlugin(filepath.Dir(worktreesDir(u.root))); err != nil {
		u.say("opencode's plugin: " + err.Error())
	}
	for i, name := range racerNames(u.agents) {
		u.launch(u.agents[i], name, 0)
	}
	go u.watch()
}

// launch starts def on job as the racer called name, in a worktree of its
// own, after the job's other racers. It returns the racer's index, or -1.
func (u *ui) launch(def agentDef, name string, job int) int {
	dir := filepath.Join(worktreesDir(u.root), name) // the same path every race: agents remember trust per path
	branch := "squeeze/" + u.raceID + "/" + name
	if err := reuseWorktree(u.root, dir, branch, u.base); err != nil {
		u.say(err.Error())
		return -1
	}
	if err := linkIgnored(u.root, dir); err != nil {
		u.say(name + ": " + err.Error())
	}
	task := u.jobs[job]
	if u.notes {
		if err := os.MkdirAll(filepath.Join(dir, notesDir), 0o755); err != nil {
			u.say(name + ": " + err.Error())
		}
		task += notesPrompt(name)
	}
	r, err := startRacer(def, name, task, dir, branch, u.paneW, u.paneH, u.wake)
	if err != nil {
		u.say(name + ": " + err.Error())
		return -1
	}
	r.task, r.job = u.jobs[job], job
	at := len(u.racers)
	for i, o := range u.racers {
		if o.job <= job {
			at = i + 1
		}
	}
	u.racersMu.Lock()
	u.racers = slices.Insert(u.racers, at, r)
	u.racersMu.Unlock()
	if at <= u.sel && len(u.racers) > 1 {
		u.sel++ // the racer shown stays shown
	}
	return at
}

// add joins def to job, during the race, and shows it.
func (u *ui) add(def agentDef, job int) {
	u.agents = append(u.agents, def)
	names := racerNames(u.agents)
	if i := u.launch(def, names[len(names)-1], job); i >= 0 {
		u.sel, u.attached, u.viewR = i, false, nil
	}
}

// watch keeps each racer's list of changes current.
func (u *ui) watch() {
	for {
		u.racersMu.Lock()
		racers := slices.Clone(u.racers)
		u.racersMu.Unlock()
		if u.notes {
			shareNotes(racers)
		}
		for _, r := range racers {
			r.refresh(u.base)
		}
		u.wake()
		time.Sleep(2 * time.Second)
	}
}

func (u *ui) raceKey(k limoni.KeyEvent, plain bool) {
	switch {
	case plain && (k.Ch == 'a' || k.Ch == 'n') && len(u.racers) > 0:
		u.newJob()
	case k.Type == limoni.KeyEsc && u.viewing():
		u.viewR = nil
	case k.Type == limoni.KeyPageUp && u.viewing():
		u.subOff += 10
	case k.Type == limoni.KeyPageDown && u.viewing():
		u.subOff = max(u.subOff-10, 0)
	case k.Type == limoni.KeyUp || (plain && k.Ch == 'k'):
		u.sel, u.viewR = max(u.sel-1, 0), nil
	case k.Type == limoni.KeyDown || (plain && k.Ch == 'j'):
		u.sel, u.viewR = min(u.sel+1, max(len(u.racers)-1, 0)), nil // no racers yet: stay at 0, never -1
	case k.Type == limoni.KeyEnter && len(u.racers) > 0:
		u.attached, u.viewR = true, nil
	case plain && k.Ch == 'c' && len(u.racers) > 0:
		u.compare()
	case plain && k.Ch == 't' && len(u.racers) > 0 && u.check != "":
		u.runCheck(u.racers[u.sel], true)
	case plain && k.Ch == 'q':
		u.askQuit()
	}
}

func (u *ui) drawRace(f *limoni.Frame) {
	a := f.Area()
	u.paneW, u.paneH = int(a.Width)-sidebar-2, int(a.Height)-4 // inside the frame
	if u.racers == nil {
		u.racers = []*racer{}
		u.start()
	}
	for _, r := range u.racers {
		r.resize(u.paneW, u.paneH)
	}
	now := time.Now()
	u.stopped(now)
	u.header(f)
	u.drawCards(f, now)
	if u.viewing() {
		u.drawSubagent(f)
	} else if len(u.racers) > 0 {
		r := u.racers[u.sel]
		note := ""
		if u.attached {
			note = " typing · F12 leaves "
		}
		inner := u.frameFor(f, r, "", note, u.attached)
		f.RenderWidget(pane{r: r, cursor: u.attached, sel: &u.marked}, inner)
	}
	switch {
	case u.viewing():
		u.footer(f, "A subagent of "+u.viewR.name+" · the wheel or PgUp/PgDn scrolls · Esc goes back")
	case len(u.racers) > 0 && u.racers[u.sel].scroll > 0:
		u.footer(f, fmt.Sprintf("%d lines up · the wheel scrolls back down, a key goes back to the bottom", u.racers[u.sel].scroll))
	case u.attached:
		u.footer(f, "Typing into "+u.racers[u.sel].name+" · F12, Ctrl+] or a click on the list gives the keys back")
	default:
		u.footer(f, "↑↓ pick · Enter types into it · n new job · c compare"+u.checkHelp()+" · q quit")
	}
}

// hit is a card or a subagent's row in the sidebar, for clicks: racer i's,
// and its subagent sub, or -1 for the card.
type hit struct{ y0, y1, i, sub int }

// subRows is how many of its subagents a card lists at most.
const subRows = 3

// shownSubs is which of a racer's subagents its card lists: those at work,
// then those done, the latest first; and how many more there are.
func shownSubs(subs []subagent) (idx []int, more int) {
	for _, running := range []bool{true, false} {
		for i := len(subs) - 1; i >= 0; i-- {
			if subs[i].running() == running {
				idx = append(idx, i)
			}
		}
	}
	if len(idx) > subRows {
		return idx[:subRows], len(idx) - subRows
	}
	return idx, 0
}

// drawCards draws a card per racer down the left side, each with the
// subagents its agent started under it.
func (u *ui) drawCards(f *limoni.Frame, now time.Time) {
	a := f.Area()
	bottom := int(a.Y + a.Height - 1)
	subs := make([][]subagent, len(u.racers))
	heights := make([]int, len(u.racers))
	for i, r := range u.racers {
		subs[i] = r.subagents()
		idx, more := shownSubs(subs[i])
		heights[i] = cardH + len(idx) + min(more, 1)
	}
	// From cardTop, which follows the selection.
	u.cardTop = min(u.cardTop, u.sel)
	for u.cardTop < u.sel {
		used := 0
		for _, h := range heights[u.cardTop : u.sel+1] {
			used += h
		}
		if int(a.Y)+1+used <= bottom {
			break
		}
		u.cardTop++
	}
	u.hits = u.hits[:0]
	y, shown := int(a.Y)+1, 0
	for i := u.cardTop; i < len(u.racers) && y+cardH <= bottom; i++ {
		r := u.racers[i]
		u.drawRacer(f, r, i, a.X, uint16(y), now)
		u.hits = append(u.hits, hit{y, y + cardH, i, -1})
		y += cardH
		idx, more := shownSubs(subs[i])
		for _, j := range idx {
			if y >= bottom {
				break
			}
			u.drawSubRow(f, &subs[i][j], u.viewR == r && u.viewI == j, a.X, uint16(y), now)
			u.hits = append(u.hits, hit{y, y + 1, i, j})
			y++
		}
		if more > 0 && y < bottom {
			f.RenderWidget(&limoni.Paragraph{Text: fmt.Sprintf("   ↳ %d more", more), Style: dim}, limoni.NewRect(a.X, uint16(y), sidebar-1, 1))
			y++
		}
		shown++
	}
	if u.cardTop > 0 || u.cardTop+shown < len(u.racers) {
		more := fmt.Sprintf(" %d–%d of %d ", u.cardTop+1, u.cardTop+shown, len(u.racers))
		w := uint16(cell.StringWidth(more))
		f.RenderWidget(&limoni.Paragraph{Text: more, Style: dim}, limoni.NewRect(a.X+sidebar-1-w, uint16(bottom-1), w, 1))
	}
	if u.mode == raceMode && y+3 <= bottom { // a card of its own, to add one
		f.RenderWidget(&limoni.Block{Borders: limoni.BorderAll, BorderSymbols: limoni.SymbolsRounded, BorderStyle: dim},
			limoni.NewRect(a.X, uint16(y), sidebar-1, 3))
		label := "+  New job"
		f.RenderWidget(&limoni.Paragraph{Text: label, Style: yellow.Merge(bold)},
			limoni.NewRect(a.X+(sidebar-1-uint16(len(label)))/2, uint16(y+1), uint16(len(label)), 1))
		u.buttons = append(u.buttons, button{x: int(a.X), y: y, w: sidebar - 1, h: 3, act: u.newJob})
	}
}

// drawSubRow draws a subagent's row under its racer's card: what it is,
// what it is for, and how long it has run.
func (u *ui) drawSubRow(f *limoni.Frame, s *subagent, on bool, x, y uint16, now time.Time) {
	mark, ms, end := "●", green, now
	if !s.running() {
		mark, ms, end = "✓", dim, s.end
	}
	text := s.desc
	if s.kind != "" && text != "" {
		text = s.kind + ": " + text
	} else if text == "" {
		text = s.kind
	}
	t := clock(end.Sub(s.start).Round(time.Second))
	style := limoni.Style{}
	if on {
		style = picked
	}
	f.RenderWidget(&limoni.Paragraph{Text: strings.Repeat(" ", sidebar-1), Style: style}, limoni.NewRect(x, y, sidebar-1, 1))
	f.RenderWidget(&limoni.Paragraph{Text: " ↳", Style: dim.Merge(style)}, limoni.NewRect(x, y, 2, 1))
	f.RenderWidget(&limoni.Paragraph{Text: mark, Style: ms.Merge(style)}, limoni.NewRect(x+3, y, 1, 1))
	tw := uint16(len(t))
	label, _ := cell.Truncate(text, int(sidebar-8-tw))
	f.RenderWidget(&limoni.Paragraph{Text: label, Style: style}, limoni.NewRect(x+5, y, sidebar-8-tw, 1))
	f.RenderWidget(&limoni.Paragraph{Text: t, Style: dim.Merge(style)}, limoni.NewRect(x+sidebar-2-tw, y, tw, 1))
}

// viewing reports whether a subagent is shown in place of the selected
// racer's screen.
func (u *ui) viewing() bool {
	return u.viewR != nil && len(u.racers) > 0 && u.racers[u.sel] == u.viewR
}

// drawSubagent shows a subagent: what it was asked, what it is doing, from
// its transcript, and what it answered; following the end while it runs.
func (u *ui) drawSubagent(f *limoni.Frame) {
	subs := u.viewR.subagents()
	if u.viewI >= len(subs) {
		u.viewR = nil
		return
	}
	s := subs[u.viewI]
	kind := s.kind
	if kind == "" {
		kind = "subagent"
	}
	note := " ● at work for " + clock(time.Since(s.start).Round(time.Second)) + " "
	if !s.running() {
		note = " ✓ finished in " + clock(s.end.Sub(s.start).Round(time.Second)) + " "
	}
	inner := u.frameFor(f, u.viewR, " › "+kind, note, false)
	w := int(inner.Width)
	var rows []row
	add := func(text string, st limoni.Style) {
		for _, line := range wrap(text, w) {
			rows = append(rows, row{line, st})
		}
	}
	if s.desc != "" {
		add(s.desc, bold)
	}
	if s.prompt != "" {
		add("", limoni.Style{})
		add("Asked", yellow)
		add(s.prompt, limoni.Style{})
	}
	act, answer := activity(s.transcript)
	if answer != "" {
		s.result = answer
	}
	if len(act) > 0 {
		add("", limoni.Style{})
		add("What it does", yellow)
		for _, r := range act {
			add(r.text, r.style)
		}
	}
	if s.result != "" {
		add("", limoni.Style{})
		add("Answer", yellow)
		add(s.result, limoni.Style{})
	}
	drawRows(f, rows, inner, &u.subOff)
}

// drawRows draws the rows that fit in area, the last of them unless off
// lines are scrolled up from the end.
func drawRows(f *limoni.Frame, rows []row, area limoni.Rect, off *int) {
	n := int(area.Height)
	*off = min(*off, max(len(rows)-n, 0))
	end := len(rows) - *off
	for i, r := range rows[max(end-n, 0):end] {
		f.RenderWidget(&limoni.Paragraph{Text: r.text, Style: r.style}, limoni.NewRect(area.X, area.Y+uint16(i), area.Width, 1))
	}
}

// drawRacer draws a racer's card, in its agent's colour: its name and
// clock on the border, what it is doing, what it has changed, and the check.
func (u *ui) drawRacer(f *limoni.Frame, r *racer, i int, x, y uint16, now time.Time) {
	c := limoni.Hex(r.color)
	border, title := limoni.Style{Fg: c, Modifier: limoni.ModifierDim}, limoni.Style{Fg: c}
	if i == u.sel {
		border, title = limoni.Style{Fg: c}, limoni.Style{Fg: ink, Bg: c, Modifier: limoni.ModifierBold}
	}
	f.RenderWidget(&limoni.Block{Title: " " + r.name + " ", TitleStyle: title, Borders: limoni.BorderAll,
		BorderSymbols: limoni.SymbolsRounded, BorderStyle: border}, limoni.NewRect(x, y, sidebar-1, cardH))
	t := " " + clock(r.elapsed(now).Round(time.Second)) + " "
	tw := uint16(len(t))
	f.RenderWidget(&limoni.Paragraph{Text: t, Style: border}, limoni.NewRect(x+sidebar-3-tw, y, tw, 1))
	if len(u.jobs) > 1 { // which job, on the bottom border
		job, _ := cell.Truncate(fmt.Sprintf(" #%d %s ", r.job+1, strings.ReplaceAll(r.task, "\n", " ")), sidebar-5)
		f.RenderWidget(&limoni.Paragraph{Text: job, Style: border}, limoni.NewRect(x+2, y+cardH-1, sidebar-5, 1))
	}

	state, ss := "● working", green
	switch r.state(now) {
	case waiting:
		state, ss = "◐ waiting for you", yellow
	case done:
		state, ss = "■ exited", dim
	}
	f.RenderWidget(&limoni.Paragraph{Text: state, Style: ss}, limoni.NewRect(x+2, y+1, sidebar-5, 1))

	files, added, removed := r.changes()
	plus, minus := fmt.Sprintf("+%d", added), fmt.Sprintf("−%d", removed)
	cx := x + 2
	f.RenderWidget(&limoni.Paragraph{Text: plus, Style: green}, limoni.NewRect(cx, y+2, uint16(len(plus)), 1))
	cx += uint16(len(plus)) + 1
	f.RenderWidget(&limoni.Paragraph{Text: minus, Style: red}, limoni.NewRect(cx, y+2, uint16(cell.StringWidth(minus)), 1))
	cx += uint16(cell.StringWidth(minus)) + 1
	f.RenderWidget(&limoni.Paragraph{Text: fmt.Sprintf("· %d files", len(files)), Style: dim}, limoni.NewRect(cx, y+2, x+sidebar-4-cx, 1))
	if mark, ms := checkMark(&r.chk); mark != "" {
		f.RenderWidget(&limoni.Paragraph{Text: mark, Style: ms}, limoni.NewRect(x+sidebar-4, y+2, 1, 1))
	}
}

// ink is the text colour on a coloured title.
var ink = limoni.Hex("#1E1E1E")

// frameFor draws the frame round the right-hand side in r's colour, with
// what is shown as its title and note at its right, thick while the user
// types into the agent, and returns the area inside.
func (u *ui) frameFor(f *limoni.Frame, r *racer, what, note string, thick bool) limoni.Rect {
	a := f.Area()
	c := limoni.Hex(r.color)
	symbols := limoni.SymbolsRounded
	if thick {
		symbols = limoni.SymbolsThick
	}
	box := limoni.NewRect(a.X+sidebar, a.Y+1, a.Width-sidebar, a.Height-2)
	f.RenderWidget(&limoni.Block{Title: " " + r.name + what + " ", TitleStyle: limoni.Style{Fg: ink, Bg: c, Modifier: limoni.ModifierBold},
		Borders: limoni.BorderAll, BorderSymbols: symbols, BorderStyle: limoni.Style{Fg: c}}, box)
	if note != "" {
		nw := uint16(cell.StringWidth(note))
		f.RenderWidget(&limoni.Paragraph{Text: note, Style: limoni.Style{Fg: c, Modifier: limoni.ModifierBold}},
			limoni.NewRect(box.X+box.Width-2-nw, box.Y, nw, 1))
	}
	return limoni.NewRect(box.X+1, box.Y+1, box.Width-2, box.Height-2)
}

// checkMark is the sidebar's sign for a racer's check.
func checkMark(c *check) (string, limoni.Style) {
	st, _, _ := c.result()
	switch st {
	case checking:
		return "…", yellow
	case passed:
		return "✓", green
	case failed:
		return "✗", red
	}
	return "", limoni.Style{}
}

func (u *ui) checkHelp() string {
	if u.check == "" {
		return ""
	}
	return " · t check now"
}

// stopped acts when an agent stops working: it tells the user, once, if
// they are not looking at it, and runs the check on what it has done.
func (u *ui) stopped(now time.Time) {
	if !u.stoppingAt.IsZero() { // quitting stops them; that is no news
		return
	}
	for i, r := range u.racers {
		idle := r.state(now) != working
		if !idle && r.idle {
			r.resume(now)
		}
		if idle && !r.idle {
			r.pause()
			if !(u.attached && i == u.sel) {
				msg := r.name + " is waiting for you"
				if a := r.ask(); a != "" {
					msg = r.name + ": " + a
				}
				u.notify("squeeze", msg)
			}
			u.runCheck(r, false)
		}
		r.idle = idle
	}
}

func (u *ui) runCheck(r *racer, force bool) {
	if u.check != "" {
		r.chk.run(u.check, r.dir, u.base, force, u.wake)
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
		u.fileSel, u.outOff = 0, 0
	case k.Type == limoni.KeyRight || k.Type == limoni.KeyTab:
		u.sel = min(u.sel+1, len(u.racers)-1)
		u.fileSel, u.outOff = 0, 0
	case plain && k.Ch == 'o' && u.check != "":
		u.output, u.outOff = !u.output, 0
	case plain && k.Ch == 't' && u.check != "":
		u.runCheck(u.racers[u.sel], true)
	case k.Type == limoni.KeyPageUp && u.output:
		u.outOff += 10
	case k.Type == limoni.KeyPageDown && u.output:
		u.outOff = max(u.outOff-10, 0)
	case k.Type == limoni.KeyUp:
		u.fileSel = max(u.fileSel-1, 0)
	case k.Type == limoni.KeyDown:
		u.fileSel = min(u.fileSel+1, max(len(u.files)-1, 0))
	case k.Type == limoni.KeyPageDown && u.diff != nil:
		u.diff.Offset += 10
	case k.Type == limoni.KeyPageUp && u.diff != nil:
		u.diff.Offset = max(u.diff.Offset-10, 0)
	case plain && k.Ch == 'm':
		u.askMerge()
	case plain && k.Ch == 'q':
		u.askQuit()
	}
}

func (u *ui) drawCompare(f *limoni.Frame) {
	now := time.Now()
	u.stopped(now)
	u.header(f)
	u.drawCards(f, now)
	r := u.racers[u.sel]
	u.files, _, _ = r.changes()
	inner := u.frameFor(f, r, " · changes", "", false)
	x, w := inner.X+1, inner.Width-1
	y := inner.Y
	bottom := inner.Y + inner.Height
	if u.check != "" {
		u.drawCheckLine(f, r, x, y, w)
		y += 2
		if u.output {
			u.drawOutput(f, r, limoni.NewRect(x, y, w, bottom-y))
			u.footer(f, "←→ agent · PgUp/PgDn scroll · o diff · t check again · m merge this one · Esc back")
			return
		}
	}
	u.listN = 0
	if len(u.files) == 0 {
		f.RenderWidget(&limoni.Paragraph{Text: r.name + " has not changed anything yet.", Style: dim}, limoni.NewRect(x, y, w, 1))
		u.footer(f, "←→ agent · Esc back")
		return
	}
	u.fileSel = min(u.fileSel, len(u.files)-1)
	listRows := min(len(u.files), 8)
	first := max(0, u.fileSel-listRows+1)
	u.listY, u.listTop, u.listN = int(y), first, listRows
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
	if fc.Added < 0 {
		f.RenderWidget(&limoni.Paragraph{Text: fc.Path + " is a binary file: its contents are not shown.", Style: dim}, limoni.NewRect(x, y, w, 1))
		u.footer(f, "←→ agent · ↑↓ file · m merge this one · Esc back")
		return
	}
	if key := r.name + "\x00" + fc.Path; key != u.diffFor {
		before, after := versions(r.dir, u.base, fc.Path)
		u.diff = widgets.NewDiffViewState(before, after, widgets.LanguageForFile(fc.Path), 3)
		u.diffFor = key
	}
	f.RenderWidget(widgets.DiffView{ID: "diff", State: u.diff}, limoni.NewRect(x, y, w, bottom-y))
	help := "←→ agent · ↑↓ file · PgUp/PgDn scroll · m merge this one · Esc back"
	if u.check != "" {
		help = "←→ agent · ↑↓ file · PgUp/PgDn scroll · o check output · m merge this one · Esc back"
	}
	u.footer(f, help)
}

// drawCheckLine says how the selected racer did on the check.
func (u *ui) drawCheckLine(f *limoni.Frame, r *racer, x, y, w uint16) {
	st, _, took := r.chk.result()
	text, style := "not checked yet: it runs when "+r.name+" stops, or press t", dim
	switch st {
	case checking:
		text, style = "checking…", yellow
	case passed:
		text, style = fmt.Sprintf("✓ passes  %s", took.Round(100*time.Millisecond)), green
	case failed:
		text, style = fmt.Sprintf("✗ fails  %s", took.Round(100*time.Millisecond)), red
	}
	f.RenderWidget(&limoni.Paragraph{Text: " " + u.check + "  ", Style: bold}, limoni.NewRect(x, y, w, 1))
	cw := min(uint16(cell.StringWidth(u.check)+3), w)
	f.RenderWidget(&limoni.Paragraph{Text: text, Style: style}, limoni.NewRect(x+cw, y, w-cw, 1))
}

// drawOutput shows the end of the check's output, scrolled up by outOff.
func (u *ui) drawOutput(f *limoni.Frame, r *racer, area limoni.Rect) {
	_, out, _ := r.chk.result()
	var rows []row
	for _, line := range strings.Split(strings.TrimRight(strings.ReplaceAll(string(out), "\t", "    "), "\n"), "\n") {
		rows = append(rows, row{text: line})
	}
	drawRows(f, rows, area, &u.outOff)
}

// ── mouse ─────────────────────────────────────────────────────────────

// mouse handles a mouse event: the sidebar picks a racer and takes the keys
// back; on the race screen the pane is the agent's (a click types into it,
// and the agent gets the mouse if it asked for it, or the wheel scrolls back
// through what it printed); on the compare screen a click picks a file and
// the wheel scrolls the diff or the check's output.
func (u *ui) mouse(m driver.MouseEvent) {
	x, y := int(m.X), int(m.Y)
	click := m.Button == driver.MouseLeft && !m.Drag
	for _, b := range u.buttons {
		if click && y >= b.y && y < b.y+max(b.h, 1) && x >= b.x && x < b.x+b.w {
			b.act()
			return
		}
	}
	if len(u.racers) == 0 || u.confirm != "" {
		return
	}
	if x < sidebar {
		for _, h := range u.hits {
			if click && y >= h.y0 && y < h.y1 {
				if h.i != u.sel {
					u.fileSel, u.outOff = 0, 0
				}
				u.sel, u.attached, u.viewR = h.i, false, nil
				if u.mode == setupMode { // a card leaves the new job for it
					u.mode = raceMode
				}
				if h.sub >= 0 && u.mode == raceMode {
					u.viewR, u.viewI, u.subOff = u.racers[h.i], h.sub, 0
				}
			}
		}
		return
	}
	if u.mode == setupMode {
		return
	}
	if u.mode == compareMode {
		u.compareMouse(m, y, click)
		return
	}
	px, py := x-sidebar-1, y-2 // inside the frame
	if px < 0 || py < 0 || px >= u.paneW || py >= u.paneH {
		return
	}
	if u.viewing() {
		switch m.Button {
		case driver.MouseScrollUp:
			u.subOff += 3
		case driver.MouseScrollDown:
			u.subOff = max(u.subOff-3, 0)
		}
		return
	}
	r := u.racers[u.sel]
	if click && m.Ctrl && u.openLink(r, r.spotAt(px, py)) { // for agents that take the mouse
		return
	}
	if r.mouse.Load() == 0 || r.scroll > 0 {
		if u.selectMouse(m, r, px, py) {
			return
		}
	}
	switch {
	case r.mouse.Load() != 0 && r.scroll == 0:
		r.emu.SendMouse(toEmulatorMouse(m, px, py))
		r.userInput(false) // a click to start typing is not an answer; one that is, the hooks report
	case isWheel(m.Button) && !r.emu.IsAltScreen():
		if m.Button == driver.MouseScrollUp {
			r.scroll = min(r.scroll+3, r.emu.ScrollbackLen())
		} else {
			r.scroll = max(r.scroll-3, 0)
		}
	}
	if click {
		u.attached = true
	}
}

// selectMouse selects text in an agent's screen with the left button, and
// copies it when the button is let go. It reports whether it took the event.
func (u *ui) selectMouse(m driver.MouseEvent, r *racer, px, py int) bool {
	switch {
	case m.Button == driver.MouseLeft && !m.Drag:
		at := r.spotAt(px, py)
		u.marked = selection{r: r, from: at, to: at, dragging: true}
		u.attached = true // a click still types into it
		return true
	case m.Button == driver.MouseLeft && m.Drag && u.marked.dragging && u.marked.r == r:
		u.marked.to = r.spotAt(px, py)
		return true
	case m.Button == driver.MouseRelease && u.marked.dragging:
		u.marked.dragging = false
		if u.marked.empty() {
			u.openLink(r, u.marked.from)
			return true
		}
		if text := u.marked.text(); text != "" && u.copy != nil {
			u.copy(text)
			u.say(fmt.Sprintf("Copied %d characters.", len([]rune(text))))
		}
		return true
	}
	return false
}

func (u *ui) compareMouse(m driver.MouseEvent, y int, click bool) {
	step := 0
	switch m.Button {
	case driver.MouseScrollUp:
		step = -3
	case driver.MouseScrollDown:
		step = 3
	}
	switch {
	case click && y >= u.listY && y < u.listY+u.listN:
		u.fileSel = u.listTop + y - u.listY
	case step != 0 && u.output:
		u.outOff = max(u.outOff-step, 0)
	case step != 0 && u.diff != nil:
		u.diff.Offset = max(u.diff.Offset+step, 0)
	}
}

// ── shared ────────────────────────────────────────────────────────────

// brand writes squeeze's name at the top left, as plain yellow text: a
// filled badge there sat on the selected card's filled title just below
// it. It returns the column after it.
func brand(f *limoni.Frame) uint16 {
	a := f.Area()
	f.RenderWidget(&limoni.Paragraph{Text: "squeeze", Style: yellow.Merge(bold)}, limoni.NewRect(a.X+1, a.Y, 7, 1))
	return a.X + 9
}

func (u *ui) header(f *limoni.Frame) {
	a := f.Area()
	task := u.prompt
	if len(u.racers) > 0 {
		task = u.racers[u.sel].task
		if len(u.jobs) > 1 {
			task = fmt.Sprintf("#%d %s", u.racers[u.sel].job+1, task)
		}
	}
	task = strings.ReplaceAll(task, "\n", " ")
	x := brand(f)
	quit := btn{"q", "Quit", u.askQuit}
	var left int
	switch {
	case u.mode == compareMode:
		left = u.buttonRow(f, a.Y, btn{"m", "Merge", u.askMerge}, btn{"Esc", "Back", func() { u.mode = raceMode }}, quit)
	case len(u.racers) > 0:
		left = u.buttonRow(f, a.Y, btn{"n", "New job", u.newJob}, btn{"c", "Compare", u.compare}, quit)
	default:
		left = u.buttonRow(f, a.Y, quit)
	}
	tw := left - 2 - int(x+2) // a long task stops short of the buttons
	if tw < 1 {
		return
	}
	if cell.StringWidth(task) > tw {
		task, _ = cell.Truncate(task, tw-1)
		task += "…"
	}
	f.RenderWidget(&limoni.Paragraph{Text: "›", Style: dim}, limoni.NewRect(x, a.Y, 1, 1))
	f.RenderWidget(&limoni.Paragraph{Text: task}, limoni.NewRect(x+2, a.Y, uint16(tw), 1))
}

// btn is a button: the key that does the same, and what it does.
type btn struct {
	key, label string
	act        func()
}

type button struct {
	x, y, w int
	h       int // rows, if more than one
	act     func()
}

var buttonStyle = limoni.Style{Fg: limoni.Hex("#1E1E1E"), Bg: limoni.Hex("#B8B8C8")}

// buttonRow draws buttons at the right end of row y, each its key and
// label, and makes them clickable. It returns the column of the leftmost.
func (u *ui) buttonRow(f *limoni.Frame, y uint16, bs ...btn) int {
	a := f.Area()
	x := int(a.X+a.Width) - 1
	left := x
	for i := len(bs) - 1; i >= 0; i-- {
		x -= keyButtonWidth(bs[i])
		if x < int(a.X) {
			return left
		}
		u.keyButton(f, x, y, bs[i], nil)
		left = x
		x -= 3
	}
	return left
}

// openLink opens the web address at the spot in r's output, if one is
// there, and reports whether it did.
func (u *ui) openLink(r *racer, at spot) bool {
	url := r.linkAt(at)
	if url == "" || u.open == nil {
		return false
	}
	u.open(url)
	return true
}

// say shows msg in the footer for a while.
func (u *ui) say(msg string) {
	u.status, u.statusAt = msg, time.Now()
}

// answer answers the question in the footer.
func (u *ui) answer(yes bool) {
	question := u.confirm
	u.confirm, u.status = "", ""
	switch {
	case yes && question == "quit":
		u.quitNow()
	case yes && question == "merge":
		u.mergeSelected()
	case yes && question == "install":
		u.installAgent(u.pending)
	}
}

func (u *ui) footer(f *limoni.Frame, help string) {
	a := f.Area()
	text, style := help, dim
	if u.status != "" && time.Since(u.statusAt) < 8*time.Second {
		text, style = u.status, yellow
	}
	y := a.Y + a.Height - 1
	f.RenderWidget(&limoni.Paragraph{ID: "status", Text: " " + text, Style: style}, limoni.NewRect(a.X, y, a.Width, 1))
}

func (u *ui) compare() {
	if len(u.racers) > 0 {
		u.mode, u.diffFor, u.attached = compareMode, "", false
	}
}

func (u *ui) askMerge() { u.ask("merge") }

func (u *ui) askQuit() {
	if len(u.racers) == 0 {
		u.quit = true
		return
	}
	u.ask("quit")
}

func (u *ui) mergeSelected() {
	r := u.racers[u.sel]
	if err := merge(u.root, r.dir, r.branch, r.name); err != nil {
		u.say("Merge failed: " + err.Error())
		return
	}
	u.say(r.name + "'s changes are staged in " + u.root + ". Look them over with git diff --staged, then commit.")
}
