package limoni

import (
	"os"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

// Re-exported Core Types
type (
	Terminal        = terminal.Terminal
	Frame           = terminal.Frame
	Rect            = cell.Rect
	Point           = cell.Point
	Style           = cell.Style
	Color           = cell.Color
	ColorType       = cell.ColorType
	Modifier        = cell.Modifier
	Context         = cell.Context
	Cell            = cell.Cell
	Buffer          = buffer.Buffer
	Backend         = backend.Backend
	Event           = backend.Event
	EventType       = backend.EventType
	KeyEvent        = backend.KeyEvent
	MouseEvent      = backend.MouseEvent
	KeyType         = backend.KeyType
	MouseButton     = backend.MouseButton
	Layout          = layout.FlexLayout
	FlexLayout      = layout.FlexLayout
	Constraint      = layout.Constraint
	Direction       = layout.Direction
	Widget          = widgets.Widget
	Alignment       = widgets.Alignment
	BorderSymbols   = widgets.BorderSymbols
	Insets          = widgets.Insets
	Block           = widgets.Block
	Paragraph       = widgets.Paragraph
	List            = widgets.List
	ListState       = widgets.ListState
	Table           = widgets.Table
	TableRow        = widgets.TableRow
	TableCell       = widgets.TableCell
	TableState      = widgets.TableState
	TableConstraint = widgets.TableConstraint
	TextInput       = widgets.TextInput
	TextInputState  = widgets.TextInputState
	TextArea        = widgets.TextArea
	TextAreaState   = widgets.TextAreaState
	Markdown        = widgets.Markdown
	Text            = widgets.Text
	Line            = widgets.Line
	Span            = widgets.Span
	TextAlignment   = widgets.TextAlignment
	Checkbox        = widgets.Checkbox
	RadioButton     = widgets.RadioButton
	ProgressBar     = widgets.ProgressBar
	Slider          = widgets.Slider
	SliderState     = widgets.SliderState
	Select          = widgets.Select
	SelectState     = widgets.SelectState
	Dialog          = widgets.Dialog
)

// Re-exported Constants
const (
	// Direction
	Horizontal = layout.Horizontal
	Vertical   = layout.Vertical

	// Alignment
	AlignLeft   = widgets.AlignLeft
	AlignCenter = widgets.AlignCenter
	AlignRight  = widgets.AlignRight

	// TextAlignment
	AlignTextLeft   = widgets.AlignTextLeft
	AlignTextCenter = widgets.AlignTextCenter
	AlignTextRight  = widgets.AlignTextRight

	// Borders
	BorderNone   = widgets.BorderNone
	BorderTop    = widgets.BorderTop
	BorderBottom = widgets.BorderBottom
	BorderLeft   = widgets.BorderLeft
	BorderRight  = widgets.BorderRight
	BorderAll    = widgets.BorderAll

	// Modifiers
	ModifierReset     = cell.ModifierReset
	ModifierBold      = cell.ModifierBold
	ModifierDim       = cell.ModifierDim
	ModifierItalic    = cell.ModifierItalic
	ModifierUnderline = cell.ModifierUnderline
	ModifierBlink     = cell.ModifierBlink
	ModifierReverse   = cell.ModifierReverse

	// Event Types
	EventKey    = backend.EventKey
	EventMouse  = backend.EventMouse
	EventResize = backend.EventResize

	// Key Types
	KeyRune      = backend.KeyRune
	KeySpace     = backend.KeySpace
	KeyEnter     = backend.KeyEnter
	KeyBackspace = backend.KeyBackspace
	KeyDelete    = backend.KeyDelete
	KeyTab       = backend.KeyTab
	KeyEsc       = backend.KeyEsc
	KeyUp        = backend.KeyArrowUp
	KeyDown      = backend.KeyArrowDown
	KeyLeft      = backend.KeyArrowLeft
	KeyRight     = backend.KeyArrowRight
	KeyHome      = backend.KeyHome
	KeyEnd       = backend.KeyEnd
	KeyPageUp    = backend.KeyPageUp
	KeyPageDown  = backend.KeyPageDown

	// Mouse Buttons
	MouseLeft       = backend.MouseLeft
	MouseMiddle     = backend.MouseMiddle
	MouseRight      = backend.MouseRight
	MouseRelease    = backend.MouseRelease
	MouseScrollUp   = backend.MouseScrollUp
	MouseScrollDown = backend.MouseScrollDown
)

var (
	SymbolsSingle  = widgets.SymbolsSingle
	SymbolsDouble  = widgets.SymbolsDouble
	SymbolsThick   = widgets.SymbolsThick
	SymbolsRounded = widgets.SymbolsRounded
	SymbolsBlock   = widgets.SymbolsBlock
)

// New initializes standard OS input/output, switches to raw mode, enables
// mouse and TrueColor tracking, and returns a fully ready-to-use Terminal.
// The caller should defer term.Close() to restore the terminal state.
func New() (*Terminal, error) {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		return nil, err
	}
	term, err := terminal.New(b)
	if err != nil {
		_ = b.Close()
		return nil, err
	}
	return term, nil
}

// NewRect creates a new Rect with specified x, y, width, and height.
func NewRect(x, y, w, h uint16) Rect {
	return cell.NewRect(x, y, w, h)
}

// StringWidth returns the terminal display column width of UTF-8 text.
func StringWidth(text string) int {
	return cell.StringWidth(text)
}

// RuneWidth returns the terminal display column width of a single rune.
func RuneWidth(r rune) int {
	return cell.RuneWidth(r)
}
