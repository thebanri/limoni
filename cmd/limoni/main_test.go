package main

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldWritesCompilableTemplate(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := scaffold(dir, "example.com/acme/dashboard", false, &out); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	for _, file := range scaffoldFiles {
		if _, err := os.Stat(filepath.Join(dir, file.Name)); err != nil {
			t.Fatalf("expected %s to be created: %v", file.Name, err)
		}
	}

	mainPath := filepath.Join(dir, "main.go")
	if _, err := parser.ParseFile(token.NewFileSet(), mainPath, nil, parser.AllErrors); err != nil {
		t.Fatalf("generated main.go does not parse: %v", err)
	}

	body, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `" dashboard "`) {
		t.Fatalf("project name was not rendered into main.go:\n%s", body)
	}

	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gomod), "module example.com/acme/dashboard") {
		t.Fatalf("module path was not rendered into go.mod:\n%s", gomod)
	}
}

func TestScaffoldRefusesExistingFilesWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := scaffold(dir, "demo", false, &out); err == nil {
		t.Fatal("expected scaffold to fail when main.go already exists")
	}
	if err := scaffold(dir, "demo", true, &out); err != nil {
		t.Fatalf("scaffold with force failed: %v", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"frobnicate"}, &out); err == nil {
		t.Fatal("expected unknown command error")
	}
	if err := run([]string{"version"}, &out); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.Contains(out.String(), "limoni scaffold") {
		t.Fatalf("version output missing: %q", out.String())
	}
}

func TestProjectName(t *testing.T) {
	cases := map[string]string{
		"github.com/user/app": "app",
		"app":                 "app",
		"app/":                "app",
		"":                    "",
	}
	for in, want := range cases {
		if got := projectName(in); got != want {
			t.Fatalf("projectName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestGeneratedGoDirectiveMatchesModule guards the trap that once made the
// scaffold emit "go 1.26.5": a generated project must not demand a newer Go
// than Limoni itself does.
func TestGeneratedGoDirectiveMatchesModule(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	var declared string
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
			declared = strings.TrimSpace(v)
			break
		}
	}
	if declared == "" {
		t.Fatal("no go directive in Limoni's go.mod")
	}
	if goVersion != strings.TrimSuffix(declared, ".0") {
		t.Fatalf("scaffold writes go %s, Limoni's go.mod declares go %s", goVersion, declared)
	}
}

// TestScaffoldBuilds compiles the generated application against this
// checkout. Parsing is not enough: the template once imported packages
// through paths that had since moved.
func TestScaffoldBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a module; skipped in -short")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := scaffold(dir, "example.com/scaffoldcheck", false, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	gomod := filepath.Join(dir, "go.mod")
	f, err := os.OpenFile(gomod, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("\nrequire github.com/thebanri/limoni v0.0.0\nreplace github.com/thebanri/limoni => " + root + "\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "-o", os.DevNull, "."}, {"vet", "."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}
