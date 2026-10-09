package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thebanri/limoni/uitest"
)

func newTestBrowser(t *testing.T) *browser {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{
		"main.go":            "package main\n\nfunc main() {}\n",
		"docs/readme.md":     "# Readme\n",
		"sub/deep/notes.txt": "hello notes\n",
		".git/HEAD":          "ref: refs/heads/main\n",
		"blob.bin":           "\x00\x01\x02",
		".env":               "SECRET=1\n",
		"big.txt":            strings.Repeat("0123456789abcde\n", maxPreview/16+1),
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "docs"), filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	b, err := newBrowser(root)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func expectScreen(t *testing.T, page *uitest.Page, want ...string) {
	t.Helper()
	screen := page.Screen()
	for _, w := range want {
		if !strings.Contains(screen, w) {
			t.Fatalf("%q missing from the screen:\n%s", w, screen)
		}
	}
}

func TestBrowsePreviewAndJump(t *testing.T) {
	b := newTestBrowser(t)
	page := uitest.Run(t, 100, 24, b.draw)
	expectScreen(t, page, "Files", "docs/", "sub/", "main.go", "Tab pane")

	// Moving onto a file previews it, highlighted, in the code view.
	for range len(b.picker.Entries) {
		if e, _ := b.picker.SelectedEntry(); e.Name == "main.go" {
			break
		}
		page.Press("down")
	}
	expectScreen(t, page, " main.go ", "package main", "3 lines")

	page.Press("up") // big.txt and blob.bin sort before main.go
	expectScreen(t, page, " blob.bin ", "binary file, not shown")
	page.Press("up")
	expectScreen(t, page, " big.txt ", "lines (first 256 KB)")

	// The jump box finds files in subdirectories, but nothing under .git.
	page.Press("/")
	expectScreen(t, page, "Go to file", "Esc close")
	for _, hidden := range []string{".git/HEAD", ".env", "linked"} {
		page.Type(hidden)
		if screen := page.Screen(); strings.Contains(screen, "  "+hidden) {
			t.Fatalf("the jump list offers %s:\n%s", hidden, screen)
		}
		for range hidden {
			page.Press("backspace")
		}
	}
	for _, f := range b.files {
		if f == ".env" || strings.HasPrefix(f, "linked") {
			t.Fatalf("jump list holds %q: %v", f, b.files)
		}
	}
	page.Type("notes")
	expectScreen(t, page, "sub/deep/notes.txt")
	page.Press("enter")
	expectScreen(t, page, " notes.txt ", "hello notes", "deep")
	if b.jumping || b.focus != "code" {
		t.Fatalf("after opening, jumping=%v focus=%q; want the code view focused", b.jumping, b.focus)
	}

	page.Press("tab")
	if b.focus != "files" {
		t.Fatalf("Tab from the code view left focus on %q", b.focus)
	}
	page.Press("ctrl+p")
	expectScreen(t, page, "Go to file")
	page.Press("esc")
	if b.jumping {
		t.Fatal("Esc left the jump box open")
	}
	if b.focus != "files" {
		t.Fatalf("Tab from the code view left focus on %q", b.focus)
	}

	// [ and ] move the divider.
	before := b.split.Ratio
	page.Press("]")
	if b.split.Ratio <= before {
		t.Fatalf("] left the divider at %v (was %v)", b.split.Ratio, before)
	}

	page.Press("q")
	page.ExpectExit()
}
