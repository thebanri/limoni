package main

// scaffoldFile describes one generated project file.
type scaffoldFile struct {
	Name     string
	Template string
}

// scaffoldFiles is the file set `limoni init` and `limoni new` write.
// {{.Module}}, {{.Name}} and {{.GoVersion}} are filled in when rendering.
var scaffoldFiles = []scaffoldFile{
	{Name: "go.mod", Template: goModTemplate},
	{Name: "main.go", Template: mainTemplate},
	{Name: ".gitignore", Template: gitignoreTemplate},
	{Name: "README.md", Template: readmeTemplate},
}

// goModTemplate deliberately leaves the limoni requirement out: `go mod tidy`
// (or `go get github.com/thebanri/limoni`) resolves the latest release.
//
// The go directive must never be newer than the one in Limoni's own go.mod —
// a newer one forces every user onto that exact toolchain. The test
// TestGeneratedGoDirectiveMatchesModule keeps the two in step.
const goModTemplate = `module {{.Module}}

go {{.GoVersion}}
`

const mainTemplate = `// {{.Name}} is a terminal application built with Limoni's declarative
// (Elm architecture) runtime: Init, Update and View on a model.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"
)

type model struct {
	presses int
}

func (m *model) Init() []limoni.Cmd { return nil }

func (m *model) Update(msg limoni.Msg) limoni.UpdateResult {
	switch msg := msg.(type) {
	case limoni.KeyPressMsg:
		if msg.Key.Type == limoni.KeyEsc || (msg.Key.Type == limoni.KeyRune && msg.Key.Ch == 'q') {
			return limoni.UpdateResult{Quit: true}
		}
		m.presses++
		return limoni.UpdateResult{Redraw: true}
	case limoni.ResizeMsg:
		return limoni.UpdateResult{Redraw: true}
	}
	return limoni.UpdateResult{}
}

func (m *model) View(f *limoni.Frame) {
	f.SetTheme(widgets.DarkTheme())
	f.RenderWidget(limoni.Block{
		Title:         " {{.Name}} ",
		Borders:       limoni.BorderAll,
		BorderSymbols: limoni.SymbolsRounded,
		Padding:       widgets.UniformInsets(1),
		Child: &limoni.Paragraph{
			Text: fmt.Sprintf("Hello from Limoni!\n\nKeys pressed: %d\n\nPress q or Esc to quit.", m.presses),
			Wrap: true,
		},
	}, f.Area())
}

func main() {
	if err := limoni.RunProgram(context.Background(), &model{}, limoni.WithProgramFPS(60)); err != nil {
		fmt.Fprintln(os.Stderr, "{{.Name}}:", err)
		os.Exit(1)
	}
}
`

const gitignoreTemplate = `{{.Name}}
*.test
*.out
`

const readmeTemplate = `# {{.Name}}

A terminal application built with [Limoni](https://github.com/thebanri/limoni).

## Run

` + "```bash" + `
go mod tidy
go run .
` + "```" + `

## Layout

- ` + "`main.go`" + ` — a model with ` + "`Init`" + `, ` + "`Update`" + ` and ` + "`View`" + `, run by ` + "`limoni.RunProgram`" + `.
- Keys: ` + "`q`" + ` or ` + "`Esc`" + ` quits.

## Next steps

- Use widgets from the ` + "`widgets`" + ` package — Table, List, TextInput, Tabs, Canvas — inside ` + "`View`" + `.
- Test it like a web page with [uitest](https://pkg.go.dev/github.com/thebanri/limoni/uitest):
  ` + "`uitest.Program(t, 80, 24, &model{})`" + `, then find widgets by role and label.
- Browse the widget gallery: https://github.com/thebanri/limoni/blob/main/docs/widget-gallery.md
`
