// Counter demonstrates Limoni's declarative (Elm architecture) application
// model using only the root limoni package.
//
//	go run ./examples/counter
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/thebanri/limoni"
)

type AppModel struct {
	count int
}

func (m *AppModel) Init() []limoni.Cmd { return nil }

func (m *AppModel) Update(msg limoni.Msg) limoni.UpdateResult {
	press, ok := msg.(limoni.KeyPressMsg)
	if !ok {
		return limoni.UpdateResult{}
	}

	switch press.Key.Type {
	case limoni.KeyEsc:
		return limoni.UpdateResult{Quit: true}
	case limoni.KeyRune:
		switch press.Key.Ch {
		case 'q', 'Q':
			return limoni.UpdateResult{Quit: true}
		case '+', '=':
			m.count++
			return limoni.UpdateResult{Redraw: true}
		case '-', '_':
			m.count--
			return limoni.UpdateResult{Redraw: true}
		}
	}
	return limoni.UpdateResult{}
}

func (m *AppModel) View(frame *limoni.Frame) {
	accent := limoni.Style{Fg: limoni.RGB(0, 255, 200)}
	muted := limoni.Style{Fg: limoni.RGB(100, 110, 120)}

	rows := limoni.SplitVertical(frame.Area(),
		limoni.Fixed(3),
		limoni.Fill(),
		limoni.Fixed(3),
	)

	frame.RenderWidget(limoni.Block{
		Title:       " 🍋 Limoni Counter Application ",
		BorderStyle: accent,
	}, rows[0])

	frame.RenderWidget(&limoni.Paragraph{
		Text:  fmt.Sprintf("Current Counter Value: %d\n\nPress '+' to increment, '-' to decrement.", m.count),
		Style: limoni.Style{Fg: limoni.RGB(0, 255, 200), Modifier: limoni.ModifierBold},
	}, rows[1])

	frame.RenderWidget(limoni.Block{
		Title:       " [+] Increment  [-] Decrement  [Q/Esc] Quit ",
		BorderStyle: muted,
	}, rows[2])
}

func main() {
	if err := limoni.RunProgram(context.Background(), &AppModel{}, limoni.WithProgramFPS(60)); err != nil {
		fmt.Fprintf(os.Stderr, "limoni: %v\n", err)
		os.Exit(1)
	}
}
