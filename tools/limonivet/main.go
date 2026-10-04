// Command limonivet reports Limoni code that compiles but does not do what it
// says: model methods that break session replay, and Blocks that draw
// nothing.
//
//	go run github.com/thebanri/limoni/tools/limonivet ./...
//
// It is a separate module so that golang.org/x/tools stays out of the Limoni
// module's dependency graph.
package main

import (
	"github.com/thebanri/limoni/tools/limonivet/invisibleblock"
	"github.com/thebanri/limoni/tools/limonivet/nondeterminism"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() { multichecker.Main(nondeterminism.Analyzer, invisibleblock.Analyzer) }
