package app

import "github.com/thebanri/limoni/widgets"

// Block is the root package's alias.
type Block = widgets.Block

func blocks() []any {
	return []any{
		widgets.Block{Title: " Logs "},                    // want `Block sets Title but not Borders`
		&widgets.Block{BorderStyle: widgets.Style{Fg: 1}}, // want `Block sets BorderStyle but not Borders`
		Block{Title: " via the alias "},                   // want `Block sets Title but not Borders`
		widgets.Block{Title: " framed ", Borders: widgets.BorderAll},
		widgets.Block{Padding: 1}, // spacing only: nothing invisible
		widgets.Block{},
		widgets.NewBlock(),
	}
}
