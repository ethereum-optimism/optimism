package bigint

import (
	"go/ast"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("bigint", New)
}

type BigIntPlugin struct {
}

func New(settings any) (register.LinterPlugin, error) {
	return &BigIntPlugin{}, nil
}

func (f *BigIntPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		{
			Name: "bigint",
			Doc:  "Validate safe use of big.Ints",
			Run:  f.run,
		},
	}, nil
}

func (f *BigIntPlugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

func (f *BigIntPlugin) run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil || selector.Sel.Name != "Uint64" {
				return true
			}

			selection := pass.TypesInfo.Selections[selector]
			if selection == nil {
				return true
			}

			if !isBigIntType(selection.Recv()) {
				return true
			}

			pass.Report(analysis.Diagnostic{
				Pos:            call.Pos(),
				End:            call.End(),
				Category:       "bigint",
				Message:        "calling Uint64 on big.Int is forbidden",
				SuggestedFixes: nil,
			})

			return true
		})
	}

	return nil, nil
}

func isBigIntType(typ types.Type) bool {
	if typ == nil {
		return false
	}

	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}

	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Pkg().Path() == "math/big" && obj.Name() == "Int"
}
