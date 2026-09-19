// Command limoni scaffolds new Limoni applications.
//
// Usage:
//
//	limoni init [module-path]   Create a Limoni application in the current directory.
//	limoni new <name>           Create directory <name> and write an application into it.
//	limoni version              Print version information.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// limoniVersion is the Limoni version the scaffold reports.
const limoniVersion = "latest"

// scaffoldData is what the templates are rendered with.
type scaffoldData struct {
	Module        string
	Name          string
	LimoniVersion string
	GoVersion     string
}

// goVersion is the go directive written into generated go.mod files. It is
// Limoni's own floor: a generated project can never need a newer toolchain
// than the library it depends on.
const goVersion = "1.25"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "limoni:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		usage(out)
		return nil
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], out)
	case "new":
		return runNew(args[1:], out)
	case "version":
		fmt.Fprintf(out, "limoni scaffold (limoni %s)\n", limoniVersion)
		return nil
	case "help", "-h", "--help":
		usage(out)
		return nil
	default:
		usage(out)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage(out io.Writer) {
	fmt.Fprint(out, `limoni — scaffold a new Limoni terminal application

Commands:
  limoni init [module-path]  Create an application in the current directory
  limoni new <name>          Create directory <name> and write an application into it
  limoni version             Print version information

Options:
  -force                     Overwrite existing files
`)
}

func runInit(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(out)
	force := fs.Bool("force", false, "overwrite existing files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	module := fs.Arg(0)
	if module == "" {
		module = filepath.Base(dir)
	}
	return scaffold(dir, module, *force, out)
}

func runNew(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(out)
	force := fs.Bool("force", false, "overwrite existing files")
	moduleFlag := fs.String("module", "", "module path for go.mod (default: the project name)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	target := fs.Arg(0)
	if target == "" {
		return errors.New("usage: limoni new <name>")
	}

	module := *moduleFlag
	if module == "" {
		// The target may be a path; use its last element as the module path.
		module = projectName(filepath.ToSlash(filepath.Clean(target)))
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	return scaffold(target, module, *force, out)
}

// scaffold renders the template set into dir.
func scaffold(dir, module string, force bool, out io.Writer) error {
	name := projectName(module)
	if name == "" {
		return fmt.Errorf("invalid module path %q", module)
	}
	data := scaffoldData{Module: module, Name: name, LimoniVersion: limoniVersion, GoVersion: goVersion}

	rendered := make(map[string][]byte, len(scaffoldFiles))
	for _, file := range scaffoldFiles {
		path := filepath.Join(dir, file.Name)
		if !force {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists (use -force to overwrite)", file.Name)
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		body, err := render(file.Name, file.Template, data)
		if err != nil {
			return err
		}
		rendered[path] = body
	}

	for path, body := range rendered {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return err
		}
	}

	fmt.Fprintf(out, "Created Limoni application %s (module %s)\n", name, module)
	fmt.Fprintln(out, "Next steps:")
	if dir != "." {
		fmt.Fprintf(out, "  cd %s\n", dir)
	}
	fmt.Fprintln(out, "  go mod tidy")
	fmt.Fprintln(out, "  go run .")
	return nil
}

func render(name, body string, data scaffoldData) ([]byte, error) {
	tpl, err := template.New(name).Parse(body)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	if err := tpl.Execute(&sb, data); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}

// projectName derives a readable project name from a module path's last element.
func projectName(module string) string {
	module = strings.TrimSpace(strings.Trim(module, "/"))
	if module == "" {
		return ""
	}
	parts := strings.Split(module, "/")
	return parts[len(parts)-1]
}
