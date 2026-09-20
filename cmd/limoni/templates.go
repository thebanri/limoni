package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// The project templates, one directory each, plus the files every project
// shares. Templates use [[ ]] as delimiters: Go code is full of {{ }} in
// composite literals such as []T{{...}}.
//
//go:embed tmpl
var templateFS embed.FS

// projectTemplate is one kind of project `limoni new -template` can create.
type projectTemplate struct {
	Name    string
	Summary string
	Readme  string // a paragraph for the generated README
}

// projectTemplates lists the templates, the default first.
var projectTemplates = []projectTemplate{
	{Name: "counter", Summary: "the smallest declarative app: a model, Update and View",
		Readme: "A declarative (Elm architecture) application: state lives in `model`, `Update` handles messages, `View` draws. Press keys to count, q to quit."},
	{Name: "dashboard", Summary: "immediate mode with live data: a sparkline and a table fed by a goroutine",
		Readme: "An immediate-mode dashboard: `draw` is called for every event, and `sample` feeds it data from a goroutine, waking the app with `limoni.Wakeup`. Replace `sample` with your own source."},
	{Name: "form", Summary: "a declarative form: text inputs, a checkbox, a button, validation",
		Readme: "A form with two text inputs, a checkbox and a submit button, navigated with Tab. `submit` holds the validation."},
	{Name: "ssh", Summary: "serve an app over SSH, one limoni.App per connection",
		Readme: "Serves an application over SSH: `go run .`, then `ssh -p 2222 localhost`. Each connection gets its own `limoni.App`; your application is in `session.go`. **It accepts anyone and makes a new host key each run** — add authentication and a persistent key before exposing it."},
}

func findTemplate(name string) (projectTemplate, error) {
	for _, t := range projectTemplates {
		if t.Name == name {
			return t, nil
		}
	}
	var names []string
	for _, t := range projectTemplates {
		names = append(names, t.Name)
	}
	return projectTemplate{}, fmt.Errorf("unknown template %q: choose one of %s", name, strings.Join(names, ", "))
}

// scaffoldFile is one file to generate: its name in the project and the
// template it is rendered from.
type scaffoldFile struct {
	Name     string
	Template string
}

// filesFor returns the files a project from template t consists of.
func filesFor(t projectTemplate) ([]scaffoldFile, error) {
	read := func(p string) (string, error) {
		b, err := templateFS.ReadFile(p)
		return string(b), err
	}
	files := []scaffoldFile{}
	for name, src := range map[string]string{"go.mod": "tmpl/go.mod.tmpl", ".gitignore": "tmpl/gitignore.tmpl", "README.md": "tmpl/README.md.tmpl"} {
		body, err := read(src)
		if err != nil {
			return nil, err
		}
		files = append(files, scaffoldFile{Name: name, Template: body})
	}
	entries, err := fs.ReadDir(templateFS, path.Join("tmpl", t.Name))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		body, err := read(path.Join("tmpl", t.Name, e.Name()))
		if err != nil {
			return nil, err
		}
		files = append(files, scaffoldFile{Name: strings.TrimSuffix(e.Name(), ".tmpl"), Template: body})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}
