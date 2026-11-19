package widgets_test

import (
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func TestCalendarFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	september := func(d int) time.Time { return time.Date(2026, time.September, d, 0, 0, 0, 0, time.UTC) }
	for _, preset := range chromeThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			primary := surface.Merge(preset.theme.RoleStyle("primary"))
			muted := surface.Merge(preset.theme.RoleStyle("muted"))
			focusBar := surface.Merge(preset.theme.Focus).Reverse()
			mine := surface.Merge(custom)
			for _, tc := range []struct {
				name                                       string
				calendar                                   widgets.Calendar
				header, weekday, today, selected, plainDay cell.Style
			}{
				{"default", widgets.Calendar{}, primary, muted, primary, focusBar, surface},
				{"explicit header", widgets.Calendar{HeaderStyle: custom}, mine, muted, primary, focusBar, surface},
				{"explicit weekday", widgets.Calendar{WeekdayStyle: custom}, primary, mine, primary, focusBar, surface},
				{"explicit today", widgets.Calendar{TodayStyle: custom}, primary, muted, mine, focusBar, surface},
				// An explicit SelectedStyle is drawn as it is, not reversed.
				{"explicit selected", widgets.Calendar{SelectedStyle: custom}, primary, muted, primary, mine, surface},
				// Style is the whole calendar's: set, the parts keep to it.
				{"explicit style", widgets.Calendar{Style: custom}, mine, mine, mine, mine.Reverse(), mine},
			} {
				t.Run(tc.name, func(t *testing.T) {
					calendar := tc.calendar
					calendar.State = &widgets.CalendarState{Selected: september(10)}
					calendar.Today = september(3)
					term := testkit.NewTerminal(20, 8)
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(calendar, term.Area()) })
					buf := term.Frame().Buffer
					// September 2026 starts on a Tuesday, so with Sunday
					// first the 1st is in the third column of row 2.
					for _, want := range []struct {
						what  string
						x, y  uint16
						ch    rune
						style cell.Style
					}{
						{"header", 3, 0, 'S', tc.header},
						{"weekday", 0, 1, 'S', tc.weekday},
						{"plain day", 10, 2, '2', tc.plainDay},
						{"today", 13, 2, '3', tc.today},
						{"selected", 12, 3, '1', tc.selected},
					} {
						if got := buf.CellAt(want.x, want.y); got.Content != want.ch || got.Style != want.style {
							t.Errorf("%s cell(%d,%d) = %+v; want %q with %+v", want.what, want.x, want.y, got, want.ch, want.style)
						}
					}
				})
			}
		})
	}
}

func TestAutocompleteFrameTheme(t *testing.T) {
	custom := cell.Style{Fg: cell.NewColorRGB(240, 80, 120), Bg: cell.NewColorRGB(40, 50, 60)}
	for _, preset := range chromeThemes {
		t.Run(preset.name, func(t *testing.T) {
			surface := preset.theme.RoleStyle("surface")
			focusBar := surface.Merge(preset.theme.Focus).Reverse()
			mine := surface.Merge(custom)
			for _, tc := range []struct {
				name                 string
				autocomplete         widgets.Autocomplete
				selected, suggestion cell.Style
			}{
				{"default", widgets.Autocomplete{}, focusBar, surface},
				// An explicit SelectedStyle is drawn as it is, not reversed.
				{"explicit selected", widgets.Autocomplete{SelectedStyle: custom}, mine, surface},
				// SuggestionStyle is the list's: set, the highlight is its
				// reverse, as without a theme.
				{"explicit suggestion", widgets.Autocomplete{SuggestionStyle: custom}, mine.Reverse(), mine},
			} {
				t.Run(tc.name, func(t *testing.T) {
					suggestions := []string{"git commit", "git push"}
					state := &widgets.AutocompleteState{}
					for _, r := range "gi" {
						state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: r}, suggestions)
					}
					if !state.Open || len(state.Matches()) != 2 || state.Selected != 0 {
						t.Fatalf("typing %q: open=%v matches=%v selected=%d", "gi", state.Open, state.Matches(), state.Selected)
					}
					autocomplete := tc.autocomplete
					autocomplete.ID, autocomplete.Suggestions, autocomplete.State = "cmd", suggestions, state
					term := testkit.NewTerminal(16, 3)
					term.Draw(func(f *terminal.Frame) { f.SetTheme(preset.theme); f.RenderWidget(autocomplete, term.Area()) })
					buf := term.Frame().Buffer
					for _, want := range []struct {
						what  string
						y     uint16
						style cell.Style
					}{{"selected suggestion", 1, tc.selected}, {"suggestion", 2, tc.suggestion}} {
						if got := buf.CellAt(1, want.y); got.Content != 'g' || got.Style != want.style {
							t.Errorf("%s cell(1,%d) = %+v; want 'g' with %+v", want.what, want.y, got, want.style)
						}
					}
				})
			}
		})
	}
}

func BenchmarkCalendarDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 20, Height: 8}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	september := time.Date(2026, time.September, 24, 0, 0, 0, 0, time.UTC)
	calendar := widgets.Calendar{State: &widgets.CalendarState{Selected: september}, Today: september}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calendar.Draw(ctx, buf)
	}
}

func BenchmarkAutocompleteDrawTheme(b *testing.B) {
	area := cell.Rect{Width: 40, Height: 7}
	buf := buffer.NewBuffer(area)
	theme := widgets.DarkTheme()
	ctx := cell.NewContext(area, theme.RoleStyle("surface"))
	ctx.ThemeStyle = theme.RoleStyle
	suggestions := []string{"git commit", "git checkout", "git push", "git pull"}
	state := &widgets.AutocompleteState{}
	state.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'g'}, suggestions)
	autocomplete := widgets.Autocomplete{ID: "cmd", Suggestions: suggestions, State: state}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		autocomplete.Draw(ctx, buf)
	}
}
