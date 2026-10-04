// Package invisibleblock reports a widgets.Block that is given a title or a
// border style but no Borders.
//
// Borders is a mask, and its zero value draws no border at all — and with no
// top border there is nowhere to put the title, so the title is not drawn
// either. A literal such as
//
//	widgets.Block{Title: " Logs ", BorderStyle: accent}
//
// therefore draws nothing, which reads as a bug in the renderer. It shipped
// twice: the counter example lost its frame and title, and every scene of the
// browser playground did. Set Borders (widgets.BorderAll), or use
// widgets.NewBlock, which does.
package invisibleblock

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the limonivet invisible-block check.
var Analyzer = &analysis.Analyzer{
	Name:     "limoniinvisibleblock",
	Doc:      "report widgets.Block literals with a title or border style but no Borders, which draw nothing",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const blockType = "github.com/thebanri/limoni/widgets.Block"

// border fields that only mean something when a border is drawn.
var borderOnly = map[string]bool{
	"Title": true, "TitleStyle": true, "TitleAlignment": true,
	"BorderStyle": true, "BorderSymbols": true, "MergeBorders": true,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CompositeLit)(nil)}, func(n ast.Node) {
		lit := n.(*ast.CompositeLit)
		tv, ok := pass.TypesInfo.Types[lit]
		if !ok || !isBlock(tv.Type) {
			return
		}
		var set string
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				return // positional fields: Borders is among them
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if key.Name == "Borders" {
				return
			}
			if borderOnly[key.Name] && set == "" {
				set = key.Name
			}
		}
		if set != "" {
			pass.Reportf(lit.Pos(), "Block sets %s but not Borders, so it draws no border and no title; set Borders: widgets.BorderAll or use widgets.NewBlock", set)
		}
	})
	return nil, nil
}

func isBlock(t types.Type) bool {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path()+"."+named.Obj().Name() == blockType
}
