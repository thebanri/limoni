package widgets

import (
	"slices"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// AutocompleteState is the text typed so far and the suggestions that match
// it. Matching happens in HandleKey, when the text changes, not in Draw.
type AutocompleteState struct {
	Input TextInputState

	// Open is whether the suggestion list shows; Selected indexes Matches.
	Open     bool
	Selected int

	matches []int // indices into the suggestions, best match first
	scores  []int
	query   string
}

// Matches are the indices of the suggestions that match the current text,
// best first, as of the last HandleKey or Refresh.
func (s *AutocompleteState) Matches() []int { return s.matches }

// Refresh matches the current text against suggestions again. HandleKey
// does this itself; call it when the suggestions change under the text.
func (s *AutocompleteState) Refresh(suggestions []string) {
	s.query = s.Input.Value()
	s.matches = s.matches[:0]
	if cap(s.scores) < len(suggestions) {
		s.scores = make([]int, len(suggestions))
	}
	s.scores = s.scores[:len(suggestions)]
	if s.query != "" {
		for i, sug := range suggestions {
			if score, ok := FuzzyMatch(s.query, sug); ok && sug != s.query {
				s.scores[i] = score
				s.matches = append(s.matches, i)
			}
		}
		slices.SortStableFunc(s.matches, func(a, b int) int { return s.scores[b] - s.scores[a] })
	}
	s.Selected = 0
	s.Open = len(s.matches) > 0
}

// HandleKey edits the text and drives the suggestion list:
//
//   - ↑/↓ move through the suggestions, opening the list if it is closed;
//   - Tab, or Enter while the list is open, accepts the selected one;
//   - Esc closes the list;
//   - anything else edits the text, and matching runs again.
//
// Enter with the list closed is not handled, so the application can take it
// as submit.
func (s *AutocompleteState) HandleKey(ev driver.KeyEvent, suggestions []string) bool {
	if s == nil {
		return false
	}
	switch ev.Type {
	case driver.KeyArrowDown, driver.KeyArrowUp:
		if len(s.matches) == 0 {
			return false
		}
		if !s.Open {
			s.Open = true
			return true
		}
		if ev.Type == driver.KeyArrowDown {
			s.Selected = (s.Selected + 1) % len(s.matches)
		} else {
			s.Selected = (s.Selected - 1 + len(s.matches)) % len(s.matches)
		}
		return true
	case driver.KeyTab, driver.KeyEnter:
		if ev.Type == driver.KeyEnter && (ev.Shift || ev.Alt || ev.Ctrl) {
			break
		}
		if !s.Open || s.Selected >= len(s.matches) || s.matches[s.Selected] >= len(suggestions) {
			return false
		}
		s.Input.SetValue(suggestions[s.matches[s.Selected]])
		s.Input.Cursor = len(s.Input.Text)
		s.matches = s.matches[:0]
		s.Open = false
		s.query = s.Input.Value()
		return true
	case driver.KeyEsc:
		if s.Open {
			s.Open = false
			return true
		}
		return false
	}
	if !s.Input.HandleKey(ev) {
		return false
	}
	if s.Input.Value() != s.query {
		s.Refresh(suggestions)
	}
	return true
}

// Autocomplete is a text input with a list of suggestions under it that
// narrows as the user types, matched fuzzily the way the command palette
// matches: "gco" finds "git checkout".
//
// The input takes the first row of the area and the list up to MaxVisible
// rows under it, so give it an area that tall, or draw it last in a layer.
// Like TextInput, it needs an ID: without one the input is not drawn.
type Autocomplete struct {
	ID          string
	Suggestions []string
	State       *AutocompleteState
	Placeholder string
	// MaxVisible caps the rows of suggestions; 0 is 6.
	MaxVisible int

	Style            cell.Style
	PlaceholderStyle cell.Style
	FocusedStyle     cell.Style
	SuggestionStyle  cell.Style
	SelectedStyle    cell.Style
}

// Draw renders the input and, while the state is open, the suggestions. It
// does not allocate.
func (a Autocomplete) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 || a.State == nil {
		return
	}
	input := TextInput{
		ID: a.ID, State: &a.State.Input, Placeholder: a.Placeholder,
		Style: a.Style, PlaceholderStyle: a.PlaceholderStyle, FocusedStyle: a.FocusedStyle,
		SelectionStart: -1, SelectionEnd: -1, Focused: ctx.IsFocused(a.ID),
	}
	row := ctx
	row.Area.Height = 1
	input.Draw(row, buf)

	if !a.State.Open || area.Height < 2 {
		return
	}
	visible := a.MaxVisible
	if visible <= 0 {
		visible = 6
	}
	visible = min(visible, int(area.Height)-1, len(a.State.matches))
	start := 0
	if a.State.Selected >= visible {
		start = a.State.Selected - visible + 1
	}
	base := ctx.Style.Merge(a.SuggestionStyle)
	selected := a.SelectedStyle
	if selected == (cell.Style{}) {
		selected = cell.Style{}.Reverse()
	}
	for i := 0; i < visible; i++ {
		idx := a.State.matches[start+i]
		if idx >= len(a.Suggestions) {
			continue
		}
		style := base
		if start+i == a.State.Selected {
			style = base.Merge(selected)
		}
		y := area.Y + 1 + uint16(i)
		for x := area.X; x < area.X+area.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: ' ', Style: style})
		}
		setClipped(buf, area.X+1, y, a.Suggestions[idx], style, int(area.Width)-2)
	}
}

// SizeHint is one row for the input plus the rows of suggestions showing.
func (a Autocomplete) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	h := 1
	if a.State != nil && a.State.Open {
		visible := a.MaxVisible
		if visible <= 0 {
			visible = 6
		}
		h += min(visible, len(a.State.matches))
	}
	return maxArea.Width, min(uint16(h), maxArea.Height)
}

// AccessibilityNode is the input, marked expanded while suggestions show,
// with the highlighted suggestion as its value when there is one.
func (a Autocomplete) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	value := ""
	position, size := 0, 0
	if s := a.State; s != nil {
		value = s.Input.Value()
		if s.Open && s.Selected < len(s.matches) && s.matches[s.Selected] < len(a.Suggestions) {
			state |= accessibility.StateExpanded
			value = a.Suggestions[s.matches[s.Selected]]
			position, size = s.Selected+1, len(s.matches)
		}
	}
	return accessibility.AccessibilityNode{
		ID:       a.ID,
		Role:     accessibility.RoleInput,
		Label:    a.Placeholder,
		Value:    value,
		State:    state,
		Position: position,
		SetSize:  size,
		Bounds:   bounds,
	}
}
