// Hyperlinks: text that a terminal can make clickable, with OSC 8.
//
//	go run ./examples/hyperlinks
//
// In kitty, WezTerm, foot, Ghostty, iTerm2, Konsole, GNOME Terminal or
// Windows Terminal the underlined text is clickable — ctrl- or cmd-click,
// depending on the terminal. Elsewhere it is drawn as ordinary text and no
// escape sequence is sent at all. `limoni doctor` says which case you are in.
package main

import (
	"fmt"

	"github.com/thebanri/limoni/widgets"

	"github.com/thebanri/limoni"
)

const doc = `# Hyperlinks

Markdown links are rendered with OSC 8, so the label alone is clickable:
read the [documentation](https://github.com/thebanri/limoni/tree/main/docs),
the [changelog](https://github.com/thebanri/limoni/blob/main/CHANGELOG.md),
or try the [playground](https://thebanri.github.io/limoni/) in a browser.

Where the terminal cannot show a link, the address is written out instead,
so nothing is ever hidden from a reader who cannot click it.
`

func main() {
	md := widgets.NewMarkdown(doc)

	err := limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Type {
			case limoni.KeyEsc:
				return false
			case limoni.KeyRune:
				if ev.Key.Ch == 'q' {
					return false
				}
			}
		}

		area := f.Area()
		f.RenderWidget(md, limoni.NewRect(area.X+2, area.Y+1, area.Width-4, area.Height-4))

		state := "this terminal cannot show hyperlinks: the addresses are printed instead"
		if f.Hyperlinks {
			state = "this terminal shows hyperlinks: ctrl- or cmd-click a label"
		}
		f.Buffer.SetString(2, area.Height-2,
			fmt.Sprintf("%s  ·  q quits", state),
			limoni.NewStyle().Dim())
		return true
	}, limoni.WithTitle("Limoni hyperlinks"))
	if err != nil {
		fmt.Println(err)
	}
}
