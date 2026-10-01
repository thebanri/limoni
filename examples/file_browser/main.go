// file_browser puts the file widgets together: a FilePicker and a CodeView
// side by side in a SplitPane, a StatusBar with the keys, and an
// Autocomplete to jump to any file by name.
//
//	go run ./examples/file_browser [dir]
//
// Keys: ↑/↓ move through the files and preview them, Enter or → opens a
// directory (or moves to the code of a file), ← or Backspace goes up, Tab
// switches between the panes, / or Ctrl+P jumps to a file, [ and ] move the
// divider (so does dragging it), q or Esc quits.
//
// main_test.go drives the same application with uitest.
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

const (
	maxJumpFiles = 5000      // files offered by the jump list
	maxPreview   = 256 << 10 // bytes of a file shown in the code view
)

var (
	accent = cell.NewColorRGB(0, 220, 170)
	muted  = cell.Style{Fg: cell.NewColorRGB(120, 135, 160)}
)

// browser is the application state. Its draw method is what limoni.Run
// calls, which is also what lets main_test.go drive it with uitest.
type browser struct {
	root  string
	files []string // paths under root, for the jump list

	picker *widgets.FilePickerState
	code   *widgets.CodeViewState
	split  *widgets.SplitState
	jump   *widgets.AutocompleteState

	focus   string // "files" or "code"
	jumping bool
	shown   string // the file in the code view
	cut     bool   // the file is longer than maxPreview
	message string
}

func newBrowser(root string) (*browser, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	b := &browser{
		root:   root,
		picker: widgets.NewFilePickerState(root),
		code:   widgets.NewCodeViewState("Move onto a file to preview it.", nil),
		split:  &widgets.SplitState{},
		focus:  "files",
	}
	if b.picker.Err != nil {
		return nil, b.picker.Err
	}
	b.split.SetRatio(0.35)
	b.files = listFiles(root)
	b.preview()
	return b, nil
}

// listFiles walks root for the jump list, skipping dot directories.
func listFiles(root string) []string {
	var files []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// Only regular files the picker would show too: WalkDir does not
		// follow symlinks, and the picker hides dot files.
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if rel, err := filepath.Rel(root, path); err == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		if len(files) >= maxJumpFiles {
			return filepath.SkipAll
		}
		return nil
	})
	return files
}

// preview shows the file under the picker's cursor.
func (b *browser) preview() {
	if e, ok := b.picker.SelectedEntry(); ok && !e.IsDir {
		b.show(filepath.Join(b.picker.Dir, e.Name))
	}
}

// show loads path into the code view.
func (b *browser) show(path string) {
	if path == b.shown {
		return
	}
	b.shown = path
	b.cut = false
	b.code.Offset, b.code.HOffset = 0, 0

	f, err := os.Open(path)
	if err != nil {
		b.code.SetSource(err.Error(), nil)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxPreview))
	switch {
	case err != nil:
		b.code.SetSource(err.Error(), nil)
	case bytes.IndexByte(data, 0) >= 0:
		b.code.SetSource("(binary file, not shown)", nil)
	default:
		b.cut = len(data) == maxPreview
		b.code.SetSource(string(data), widgets.LanguageForFile(path))
	}
}

// open shows a file from the jump list and moves the picker to it.
func (b *browser) open(rel string) {
	path := filepath.Join(b.root, filepath.FromSlash(rel))
	if err := b.picker.Load(filepath.Dir(path)); err != nil {
		b.message = err.Error()
		return
	}
	for i, e := range b.picker.Entries {
		if e.Name == filepath.Base(path) {
			b.picker.Selected = i
		}
	}
	b.show(path)
	b.focus = "code"
}

func (b *browser) isFile(rel string) bool {
	for _, f := range b.files {
		if f == rel {
			return true
		}
	}
	return false
}

// handleJump feeds a key to the jump box. Enter on a known file opens it.
func (b *browser) handleJump(key limoni.KeyEvent) {
	handled := b.jump.HandleKey(key, b.files)
	switch key.Type {
	case limoni.KeyEsc:
		if !handled { // the first Esc closes the list, the second the box
			b.jumping = false
		}
	case limoni.KeyEnter:
		if value := b.jump.Input.Value(); b.isFile(value) {
			b.jumping = false
			b.open(value)
		} else if !handled {
			b.message = fmt.Sprintf("no file %q", value)
		}
	}
}

func (b *browser) draw(f *limoni.Frame, ev *limoni.Event) bool {
	fm := f.FocusManager
	if id := fm.Focused(); id == "files" || id == "code" {
		b.focus = id // a click on a pane focuses it
	}

	if ev != nil && ev.Type == limoni.EventKey {
		key := ev.Key
		b.message = ""
		switch {
		case b.jumping:
			b.handleJump(key)
		case key.Type == limoni.KeyEsc:
			return false
		case key.Type == limoni.KeyTab:
			if b.focus == "files" {
				b.focus = "code"
			} else {
				b.focus = "files"
			}
		case key.Type == limoni.KeyRune && (key.Ch == '/' || key.Ctrl && (key.Ch == 'p' || key.Ch == 'P')):
			b.jumping = true
			b.jump = &widgets.AutocompleteState{}
		case key.Type == limoni.KeyRune && key.Ch == 'q':
			return false
		case key.Type == limoni.KeyRune && key.Ch == '[':
			b.split.SetRatio(b.split.Ratio - 0.05)
		case key.Type == limoni.KeyRune && key.Ch == ']':
			b.split.SetRatio(b.split.Ratio + 0.05)
		case b.focus == "files":
			b.picker.HandleKey(key)
			if b.picker.Chosen != "" {
				b.show(b.picker.Chosen)
				b.picker.Chosen = ""
				b.focus = "code"
			} else {
				b.preview()
			}
		default:
			b.code.HandleKey(key)
		}
	}
	if b.jumping {
		fm.SetFocused("jump")
	} else {
		fm.SetFocused(b.focus)
	}

	area := f.Area()
	rows := layout.VBox(area, layout.Fill(), layout.Fixed(1))

	border := func(id string) cell.Style {
		if b.focus == id && !b.jumping {
			return cell.Style{Fg: accent}
		}
		return muted
	}
	title := " " + filepath.Base(b.shown) + " "
	if b.shown == "" {
		title = " no file "
	}
	f.RenderWidget(widgets.SplitPane{
		ID:    "split",
		State: b.split,
		First: widgets.Block{
			Title: " Files ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: border("files"),
			Child: widgets.FilePicker{ID: "files", State: b.picker, DirStyle: cell.Style{Fg: accent}},
		},
		Second: widgets.Block{
			Title: title, Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: border("code"),
			Child: widgets.CodeView{ID: "code", State: b.code},
		},
	}, rows[0])

	if b.jumping && area.Width > 8 && area.Height > 5 {
		w := min(area.Width-4, 64)
		f.RenderWidget(widgets.Block{
			Title: " Go to file ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
			BorderStyle: cell.Style{Fg: accent}, PaddingLeft: 1, PaddingRight: 1,
			Child: widgets.Autocomplete{
				ID: "jump", Suggestions: b.files, State: b.jump,
				Placeholder: "type part of a path", MaxVisible: 8,
				SelectedStyle: cell.Style{Fg: cell.NewColorRGB(0, 0, 0), Bg: accent},
			},
		}, cell.NewRect(area.X+(area.Width-w)/2, area.Y+2, w, min(area.Height-3, 11)))
	}

	keys := []widgets.StatusItem{{Key: "↑↓", Text: "move"}, {Key: "Tab", Text: "pane"}, {Key: "/", Text: "jump"}, {Key: "[ ]", Text: "divider"}, {Key: "q", Text: "quit"}}
	if b.jumping {
		keys = []widgets.StatusItem{{Key: "↑↓", Text: "choose"}, {Key: "Enter", Text: "open"}, {Key: "Esc", Text: "close"}}
	}
	right := ""
	if b.shown != "" {
		right = fmt.Sprintf("%d lines", b.code.Lines())
		if b.cut {
			right += fmt.Sprintf(" (first %d KB)", maxPreview>>10)
		}
	}
	if b.message != "" {
		right = b.message
	}
	f.RenderWidget(widgets.StatusBar{
		Left:     keys,
		Right:    []widgets.StatusItem{{Text: right}},
		Style:    muted,
		KeyStyle: cell.Style{Fg: accent, Modifier: cell.ModifierBold},
	}, rows[1])
	return true
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	b, err := newBrowser(root)
	if err == nil {
		err = limoni.Run(b.draw, limoni.WithTitle("Limoni file browser"))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "file_browser: %v\n", err)
		os.Exit(1)
	}
}
