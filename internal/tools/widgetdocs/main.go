// Command widgetdocs, widgets paketindeki kaynak koddan docs/widget-gallery.md dosyasını üretir.
//
// Kullanım:
//
//	go run ./internal/tools/widgetdocs -src widgets -out docs/widget-gallery.md
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func firstSentence(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i+1]
	}
	return s
}

func main() {
	src := flag.String("src", "widgets", "widgets paketinin dizini")
	out := flag.String("out", filepath.Join("docs", "widget-gallery.md"), "üretilecek markdown dosyası")
	flag.Parse()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *src, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	pkg := pkgs["widgets"]

	type field struct{ name, typ, doc string }
	type widget struct {
		name, doc, recv string
		fields          []field
	}
	structs := map[string]*widget{}
	drawRecv := map[string]string{}

	for _, f := range pkg.Files {
		for _, d := range f.Decls {
			switch decl := d.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					doc := ""
					if decl.Doc != nil {
						doc = decl.Doc.Text()
					} else if ts.Doc != nil {
						doc = ts.Doc.Text()
					}
					w := &widget{name: ts.Name.Name, doc: firstSentence(doc)}
					for _, fl := range st.Fields.List {
						var buf strings.Builder
						printType(&buf, fl.Type)
						fdoc := ""
						if fl.Doc != nil {
							fdoc = firstSentence(fl.Doc.Text())
						}
						for _, n := range fl.Names {
							if n.IsExported() {
								w.fields = append(w.fields, field{n.Name, buf.String(), fdoc})
							}
						}
					}
					structs[ts.Name.Name] = w
				}
			case *ast.FuncDecl:
				if decl.Name.Name != "Draw" || decl.Recv == nil || len(decl.Recv.List) == 0 {
					continue
				}
				var buf strings.Builder
				printType(&buf, decl.Recv.List[0].Type)
				t := buf.String()
				base := strings.TrimPrefix(t, "*")
				drawRecv[base] = t
			}
		}
	}

	names := make([]string, 0, len(drawRecv))
	for n := range drawRecv {
		if _, ok := structs[n]; ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	var sb strings.Builder
	sb.WriteString("# Widget Galerisi ve API Referansı\n\n")
	sb.WriteString("Bu dosya `widgets` paketindeki kaynak koddan üretilmiştir; her bileşen `Widget`\n")
	sb.WriteString("arayüzünü (`Draw(cell.Context, *buffer.Buffer)`) uygular.\n\n")
	sb.WriteString("| Widget | Alıcı tipi | Alan sayısı |\n| --- | --- | --- |\n")
	for _, n := range names {
		sb.WriteString(fmt.Sprintf("| [%s](#%s) | `%s` | %d |\n", n, strings.ToLower(n), drawRecv[n], len(structs[n].fields)))
	}
	sb.WriteString("\n")
	for _, n := range names {
		w := structs[n]
		sb.WriteString("## " + n + "\n\n")
		if w.doc != "" {
			sb.WriteString(w.doc + "\n\n")
		}
		sb.WriteString(fmt.Sprintf("`RenderWidget` çağrısında kullanılacak tip: `%s`\n\n", drawRecv[n]))
		if len(w.fields) == 0 {
			sb.WriteString("_Alanı yok._\n\n")
			continue
		}
		sb.WriteString("| Alan | Tip | Açıklama |\n| --- | --- | --- |\n")
		for _, f := range w.fields {
			doc := strings.ReplaceAll(f.doc, "|", "\\|")
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", f.name, f.typ, doc))
		}
		sb.WriteString("\n")
	}
	if err := os.WriteFile(*out, []byte(sb.String()), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("%s: %d widget yazıldı\n", *out, len(names))
}

func printType(sb *strings.Builder, e ast.Expr) {
	switch t := e.(type) {
	case *ast.Ident:
		sb.WriteString(t.Name)
	case *ast.StarExpr:
		sb.WriteString("*")
		printType(sb, t.X)
	case *ast.SelectorExpr:
		printType(sb, t.X)
		sb.WriteString(".")
		sb.WriteString(t.Sel.Name)
	case *ast.ArrayType:
		sb.WriteString("[]")
		printType(sb, t.Elt)
	case *ast.MapType:
		sb.WriteString("map[")
		printType(sb, t.Key)
		sb.WriteString("]")
		printType(sb, t.Value)
	case *ast.FuncType:
		sb.WriteString("func(...)")
	case *ast.InterfaceType:
		sb.WriteString("interface{...}")
	case *ast.Ellipsis:
		sb.WriteString("...")
		printType(sb, t.Elt)
	default:
		sb.WriteString("?")
	}
}
