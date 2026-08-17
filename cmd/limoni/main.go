// Command limoni, Limoni TUI projeleri için iskelet (scaffold) üretir.
//
// Kullanım:
//
//	limoni init [modül-yolu]   Mevcut dizinde yeni bir Limoni uygulaması oluşturur.
//	limoni new <ad>            <ad> dizinini oluşturup içine uygulama iskeleti yazar.
//	limoni version             Sürüm bilgisini yazar.
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

// limoniVersion, üretilen go.mod dosyasında istenen Limoni sürümüdür.
const limoniVersion = "latest"

// scaffoldData, şablonlara aktarılan render bağlamıdır.
type scaffoldData struct {
	Module        string
	Name          string
	LimoniVersion string
}

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
		return fmt.Errorf("bilinmeyen komut %q", args[0])
	}
}

func usage(out io.Writer) {
	fmt.Fprint(out, `limoni — Limoni TUI proje iskeleti üreticisi

Komutlar:
  limoni init [modül-yolu]  Mevcut dizinde uygulama iskeleti oluşturur
  limoni new <ad>           <ad> dizinini oluşturup iskeleti içine yazar
  limoni version            Sürüm bilgisini yazar

Seçenekler:
  -force                    Var olan dosyaların üzerine yazar
`)
}

func runInit(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(out)
	force := fs.Bool("force", false, "var olan dosyaların üzerine yaz")
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
	force := fs.Bool("force", false, "var olan dosyaların üzerine yaz")
	moduleFlag := fs.String("module", "", "go.mod modül yolu (varsayılan: proje adı)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	target := fs.Arg(0)
	if target == "" {
		return errors.New("kullanım: limoni new <ad>")
	}

	module := *moduleFlag
	if module == "" {
		// Hedef bir dizin yolu olabilir; modül yolu olarak son bileşeni kullan.
		module = projectName(filepath.ToSlash(filepath.Clean(target)))
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	return scaffold(target, module, *force, out)
}

// scaffold, şablon kümesini hedef dizine render eder.
func scaffold(dir, module string, force bool, out io.Writer) error {
	name := projectName(module)
	if name == "" {
		return fmt.Errorf("geçersiz modül yolu %q", module)
	}
	data := scaffoldData{Module: module, Name: name, LimoniVersion: limoniVersion}

	rendered := make(map[string][]byte, len(scaffoldFiles))
	for _, file := range scaffoldFiles {
		path := filepath.Join(dir, file.Name)
		if !force {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s zaten var (üzerine yazmak için -force kullanın)", file.Name)
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

	fmt.Fprintf(out, "Limoni projesi oluşturuldu: %s (modül %s)\n", name, module)
	fmt.Fprintln(out, "Sonraki adımlar:")
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

// projectName, modül yolunun son bileşeninden okunabilir bir proje adı çıkarır.
func projectName(module string) string {
	module = strings.TrimSpace(strings.Trim(module, "/"))
	if module == "" {
		return ""
	}
	parts := strings.Split(module, "/")
	return parts[len(parts)-1]
}
