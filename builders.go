package limoni

import (
	"github.com/thebanri/limoni/widgets"
)

// NewBlock creates a new Block widget configured with borders and rounded corners.
func NewBlock() *Block {
	return widgets.NewBlock()
}

// NewParagraph creates a new Paragraph widget with word-wrapping enabled.
func NewParagraph(text string) *Paragraph {
	return widgets.NewParagraph(text)
}

// NewList creates a new List widget with the provided items.
func NewList(items ...string) *List {
	return widgets.NewList(items...)
}

// NewTable creates a new Table widget with grid lines enabled by default.
func NewTable() *Table {
	return widgets.NewTable()
}

// NewRow creates a new TableRow from string cells.
func NewRow(cells ...string) TableRow {
	return widgets.NewRow(cells...)
}

// NewTextInput creates a single-line interactive TextInput widget with the specified ID.
func NewTextInput(id string) *TextInput {
	return widgets.NewTextInput(id)
}

// NewMarkdown creates a new Markdown rendering widget.
func NewMarkdown(content string) *Markdown {
	return widgets.NewMarkdown(content)
}

// NewText creates a new rich Text widget with the provided lines.
func NewText(lines ...Line) *Text {
	return widgets.NewText(lines...)
}

// NewLine creates a rich-text Line from styled Spans.
func NewLine(spans ...Span) Line {
	return widgets.NewLine(spans...)
}

// NewSpan creates a styled Span for rich-text lines.
func NewSpan(text string, style Style) Span {
	return widgets.NewSpan(text, style)
}

// NewTableState creates a new TableState for tracking selection, scrolling, and column sizes.
func NewTableState() *TableState {
	return widgets.NewTableState()
}

// NewListState creates a new ListState for tracking list item selection and scroll offsets.
func NewListState() *ListState {
	return widgets.NewListState()
}

// NewTextInputState creates a new TextInputState for managing typed text and cursor position.
func NewTextInputState() *TextInputState {
	return widgets.NewTextInputState()
}

// NewTextAreaState creates a new TextAreaState for multiline text editing.
func NewTextAreaState() *TextAreaState {
	return widgets.NewTextAreaState()
}

// NewSelectState creates a new SelectState for dropdown selection.
func NewSelectState() *SelectState {
	return widgets.NewSelectState()
}

// NewSliderState creates a new SliderState with the given initial value.
func NewSliderState(value int) *SliderState {
	return widgets.NewSliderState(value)
}
