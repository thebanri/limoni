package bubbletea

import (
	"context"
	"strings"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
)

// Msg represents any Bubble Tea message.
type Msg = runtime.Msg

// Cmd is an asynchronous operation that returns a message.
type Cmd func() Msg

// Batch combines multiple commands into a single command.
func Batch(cmds ...Cmd) Cmd {
	return func() Msg {
		return BatchMsg(cmds)
	}
}

// BatchMsg holds multiple commands to run.
type BatchMsg []Cmd

// Quit is a command that requests the application to terminate.
func Quit() Msg {
	return QuitMsg{}
}

// QuitMsg is sent when Quit is requested.
type QuitMsg struct{}

// Model defines the Bubble Tea interface: Init, Update, View.
// View returns a string (rendered lines) rather than writing to a Frame directly.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// KeyMsg represents a keyboard event formatted similarly to Bubble Tea.
type KeyMsg struct {
	Type  KeyType
	Runes []rune
	Alt   bool
	Ctrl  bool
	Shift bool
	Paste bool
}

// KeyType describes key category.
type KeyType int

const (
	KeyNull KeyType = iota
	KeyRunes
	KeyEnter
	KeyBackspace
	KeyTab
	KeyEsc
	KeySpace
	KeyUp
	KeyDown
	KeyRight
	KeyLeft
	KeyPgUp
	KeyPgDown
	KeyHome
	KeyEnd
	KeyDelete
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlG
	KeyCtrlH
	KeyCtrlI
	KeyCtrlJ
	KeyCtrlK
	KeyCtrlL
	KeyCtrlM
	KeyCtrlN
	KeyCtrlO
	KeyCtrlP
	KeyCtrlQ
	KeyCtrlR
	KeyCtrlS
	KeyCtrlT
	KeyCtrlU
	KeyCtrlV
	KeyCtrlW
	KeyCtrlX
	KeyCtrlY
	KeyCtrlZ
)

func (k KeyMsg) String() string {
	if k.Ctrl && len(k.Runes) > 0 {
		return "ctrl+" + string(k.Runes)
	}
	if len(k.Runes) > 0 {
		return string(k.Runes)
	}
	switch k.Type {
	case KeyEnter:
		return "enter"
	case KeyBackspace:
		return "backspace"
	case KeyTab:
		return "tab"
	case KeyEsc:
		return "esc"
	case KeySpace:
		return " "
	case KeyUp:
		return "up"
	case KeyDown:
		return "down"
	case KeyRight:
		return "right"
	case KeyLeft:
		return "left"
	case KeyCtrlC:
		return "ctrl+c"
	default:
		return ""
	}
}

// WindowSizeMsg conveys terminal dimensions.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// adapterModel wraps a Bubble Tea Model to work with Limoni's native runtime.Model.
type adapterModel struct {
	inner Model
}

func (a *adapterModel) Init() []runtime.Cmd {
	if a.inner == nil {
		return nil
	}
	cmd := a.inner.Init()
	if cmd == nil {
		return nil
	}
	return []runtime.Cmd{wrapCmd(cmd)}
}

func (a *adapterModel) Update(msg runtime.Msg) runtime.UpdateResult {
	if a.inner == nil {
		return runtime.UpdateResult{}
	}

	// Map Limoni runtime input messages to Bubble Tea messages
	teaMsg := mapMsg(msg)

	nextModel, nextCmd := a.inner.Update(teaMsg)
	a.inner = nextModel

	var cmds []runtime.Cmd
	if nextCmd != nil {
		cmds = append(cmds, wrapCmd(nextCmd))
	}

	isQuit := false
	if _, ok := teaMsg.(QuitMsg); ok {
		isQuit = true
	}

	return runtime.UpdateResult{
		Commands: cmds,
		Redraw:   true,
		Quit:     isQuit,
	}
}

func (a *adapterModel) View(frame *terminal.Frame) {
	if a.inner == nil || frame == nil {
		return
	}
	content := a.inner.View()
	lines := strings.Split(content, "\n")
	w := frame.Buffer.Area.Width
	h := frame.Buffer.Area.Height

	for y := 0; y < int(h) && y < len(lines); y++ {
		lineRunes := []rune(lines[y])
		for x := 0; x < int(w) && x < len(lineRunes); x++ {
			frame.Buffer.SetCell(uint16(x), uint16(y), cell.Cell{
				Content: lineRunes[x],
				Style:   cell.Style{},
			})
		}
	}
}

func wrapCmd(cmd Cmd) runtime.Cmd {
	return func(ctx context.Context) runtime.Msg {
		if cmd == nil {
			return nil
		}
		res := cmd()
		if batch, ok := res.(BatchMsg); ok {
			for _, bCmd := range batch {
				if bCmd != nil {
					_ = bCmd()
				}
			}
			return nil
		}
		return res
	}
}

func mapMsg(msg runtime.Msg) Msg {
	switch m := msg.(type) {
	case runtime.KeyPressMsg:
		if m.Key.Ctrl && m.Key.Ch == 'c' {
			return KeyMsg{Type: KeyCtrlC, Ctrl: true, Runes: []rune{'c'}}
		}
		switch m.Key.Type {
		case backend.KeyRune:
			return KeyMsg{Type: KeyRunes, Runes: []rune{m.Key.Ch}, Alt: m.Key.Alt, Ctrl: m.Key.Ctrl, Shift: m.Key.Shift}
		case backend.KeyEnter:
			return KeyMsg{Type: KeyEnter}
		case backend.KeyBackspace:
			return KeyMsg{Type: KeyBackspace}
		case backend.KeyTab:
			return KeyMsg{Type: KeyTab}
		case backend.KeyEsc:
			return KeyMsg{Type: KeyEsc}
		case backend.KeyArrowUp:
			return KeyMsg{Type: KeyUp}
		case backend.KeyArrowDown:
			return KeyMsg{Type: KeyDown}
		case backend.KeyArrowLeft:
			return KeyMsg{Type: KeyLeft}
		case backend.KeyArrowRight:
			return KeyMsg{Type: KeyRight}
		}
	case runtime.ResizeMsg:
		return WindowSizeMsg{Width: int(m.Width), Height: int(m.Height)}
	}
	return msg
}

// Program wraps Limoni's runtime.Program to run Bubble Tea Models seamlessly.
type Program struct {
	prog    *runtime.Program
	backend *backend.Backend
	term    *terminal.Terminal
}

// NewProgram creates a new Bubble Tea compatible Program running on Limoni.
func NewProgram(m Model) *Program {
	adapted := &adapterModel{inner: m}
	p := runtime.New(runtime.WithModel(adapted))
	return &Program{
		prog: p,
	}
}

// Run executes the Program.
func (p *Program) Run(ctx context.Context) error {
	if p.prog == nil {
		return nil
	}
	return p.prog.Run(ctx)
}
