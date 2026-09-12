// Scroll demonstrates the navigation and feedback widgets working together:
// Tabs for switching panes, Viewport + Scrollbar for scrolling content that is
// taller than the screen, and Spinner for background activity.
//
//	go run ./examples/scroll
//
// Keys: Tab / Shift+Tab switch panes, ↑/↓ and PgUp/PgDn scroll,
// Home/End jump, q or Esc quits. The mouse wheel and the scrollbar work too.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"
)

type pane struct {
	title string
	lines []string
	state *limoni.ViewportState
}

func main() {
	panes := []*pane{
		{title: "Changelog", lines: generate("commit", 120), state: limoni.NewViewportState()},
		{title: "Logs", lines: generate("log line", 400), state: limoni.NewViewportState()},
		{title: "Short", lines: generate("fits on screen", 4), state: limoni.NewViewportState()},
	}

	active := 0
	started := time.Now()

	accent := limoni.Fg(limoni.Hex("#00FFAA"))
	muted := limoni.Fg(limoni.Hex("#6B7280"))

	err := limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		current := panes[active]

		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Type {
			case limoni.KeyEsc:
				return false
			case limoni.KeyTab:
				if ev.Key.Shift {
					active = (active - 1 + len(panes)) % len(panes)
				} else {
					active = (active + 1) % len(panes)
				}
			case limoni.KeyUp:
				current.state.ScrollBy(-1)
			case limoni.KeyDown:
				current.state.ScrollBy(1)
			case limoni.KeyPageUp:
				current.state.ScrollBy(-10)
			case limoni.KeyPageDown:
				current.state.ScrollBy(10)
			case limoni.KeyHome:
				current.state.GotoTop()
			case limoni.KeyEnd:
				current.state.GotoBottom()
			case limoni.KeyRune:
				if ev.Key.Ch == 'q' || ev.Key.Ch == 'Q' {
					return false
				}
			}
			current = panes[active]
		}

		rows := limoni.SplitVertical(f.Area(),
			limoni.Fixed(1), // tab bar
			limoni.Fill(),   // scrolling body
			limoni.Fixed(1), // status line
		)

		titles := make([]string, len(panes))
		for i, p := range panes {
			titles[i] = p.title
		}
		f.RenderWidget(limoni.Tabs{
			Titles:   titles,
			Selected: active,
			OnSelect: func(i int) { active = i },
		}, rows[0])

		f.RenderWidget(limoni.Viewport{
			Child: &limoni.Paragraph{
				Text:  strings.Join(current.lines, "\n"),
				Style: muted,
			},
			State:     current.state,
			Scrollbar: true,
		}, rows[1])

		f.RenderWidget(widgets.Spinner{
			Set:   limoni.SpinnerBraille,
			Since: started,
			Label: fmt.Sprintf("%s — %d/%d  (%.0f%%)  [Tab] switch  [q] quit",
				current.title,
				current.state.OffsetY+1,
				len(current.lines),
				current.state.ScrollPercent()*100,
			),
			Style:      accent,
			LabelStyle: muted,
		}, rows[2])

		return true
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "limoni: %v\n", err)
		os.Exit(1)
	}
}

func generate(prefix string, n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("%04d  %s %d — the quick brown fox jumps over the lazy dog", i+1, prefix, i+1)
	}
	return lines
}
