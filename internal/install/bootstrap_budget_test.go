package install

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestBootstrapCLIContextsUsePackageDeadline(t *testing.T) {
	positions := token.NewFileSet()
	fixture, err := parser.ParseFile(positions, "bootstrap_receipt_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(fixture, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "WithTimeout" {
			return true
		}
		name, ok := selector.X.(*ast.Ident)
		if ok && name.Name == "context" {
			t.Errorf("%s: fixed CLI timeout can interrupt placement or rollback; use the package deadline", positions.Position(call.Pos()))
		}
		return true
	})
}
