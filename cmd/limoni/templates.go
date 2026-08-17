package main

// scaffoldFile, üretilecek tek bir proje dosyasını tanımlar.
type scaffoldFile struct {
	Name     string
	Template string
}

// scaffoldFiles, `limoni init` ve `limoni new` tarafından üretilen dosya kümesidir.
// Şablonlardaki {{.Module}} ve {{.Name}} yer tutucuları render sırasında değiştirilir.
var scaffoldFiles = []scaffoldFile{
	{Name: "go.mod", Template: goModTemplate},
	{Name: "main.go", Template: mainTemplate},
	{Name: ".gitignore", Template: gitignoreTemplate},
	{Name: "README.md", Template: readmeTemplate},
}

// goModTemplate, limoni bağımlılığını bilinçli olarak sabitlemez.
// Sürüm, `go mod tidy` (veya `go get github.com/thebanri/limoni`) ile çözülür.
const goModTemplate = `module {{.Module}}

go 1.26.5
`

const mainTemplate = `// {{.Name}}, Limoni Init/Update/View runtime'ı ile yazılmış bir terminal uygulamasıdır.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

type model struct {
	presses int
}

func (m *model) Init() []runtime.Cmd { return nil }

func (m *model) Update(msg runtime.Msg) runtime.UpdateResult {
	switch ev := msg.(type) {
	case runtime.KeyPressMsg:
		if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
			return runtime.UpdateResult{Quit: true}
		}
		m.presses++
		return runtime.UpdateResult{Redraw: true}
	case runtime.ResizeMsg:
		return runtime.UpdateResult{Redraw: true}
	}
	return runtime.UpdateResult{}
}

func (m *model) View(f *terminal.Frame) {
	f.SetTheme(widgets.DarkTheme())
	f.RenderWidget(widgets.Block{
		Title:         " {{.Name}} ",
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		Padding:       widgets.UniformInsets(1),
		Child: &widgets.Paragraph{
			Text: fmt.Sprintf("Merhaba Limoni!\n\nTuş basışı: %d\n\nÇıkmak için q veya Esc.", m.presses),
			Wrap: true,
		},
	}, f.Buffer.Area)
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	term, err := terminal.New(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, "limoni:", err)
		os.Exit(1)
	}

	program := runtime.New(runtime.WithModel(&model{}), runtime.WithFPS(60))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := program.RunTerminal(ctx, term, b); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "limoni:", err)
		os.Exit(1)
	}
}
`

const gitignoreTemplate = `{{.Name}}
*.test
*.out
`

const readmeTemplate = `# {{.Name}}

[Limoni](https://github.com/thebanri/limoni) ile oluşturulmuş terminal uygulaması.

## Çalıştırma

` + "```bash" + `
go get github.com/thebanri/limoni
go mod tidy
go run .
` + "```" + `

## Yapı

- ` + "`main.go`" + ` — ` + "`runtime.Model`" + ` arayüzünü uygulayan Init/Update/View döngüsü.
- Klavye: ` + "`q`" + ` veya ` + "`Esc`" + ` çıkış.

## Sonraki adımlar

- ` + "`widgets`" + ` paketindeki Block, Table, List, TextInput, Canvas gibi bileşenleri ` + "`View`" + ` içinde kullanın.
- Deterministik testler için ` + "`testkit.NewTerminal`" + ` ile anlık görüntü (snapshot) testleri yazın.
`
