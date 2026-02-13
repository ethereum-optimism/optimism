package bigint

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
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
	shouldSuggestFix := pass.Pkg != nil && pass.Pkg.Path() != "github.com/ethereum-optimism/optimism/op-service/bigs"
	for _, file := range pass.Files {
		addImportEdit := shouldSuggestFix && !fileHasImport(file, "github.com/ethereum-optimism/optimism/op-service/bigs")
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

			var suggestedFixes []analysis.SuggestedFix
			if shouldSuggestFix {
				replacement := buildStrictCall(pass, selector.X)
				if replacement != "" {
					edits := []analysis.TextEdit{
						{
							Pos:     call.Pos(),
							End:     call.End(),
							NewText: []byte(replacement),
						},
					}
					if addImportEdit {
						edits = append(edits, importEdit(pass, file))
						addImportEdit = false
					}
					suggestedFixes = []analysis.SuggestedFix{
						{
							Message:   "Replace with bigs.Uint64Strict",
							TextEdits: edits,
						},
					}
				}
			}

			pass.Report(analysis.Diagnostic{
				Pos:            call.Pos(),
				End:            call.End(),
				Category:       "bigint",
				Message:        "use bigs.Uint64Strict instead of big.Int.Uint64",
				SuggestedFixes: suggestedFixes,
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

func buildStrictCall(pass *analysis.Pass, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, pass.Fset, expr); err != nil {
		return ""
	}
	return "bigs.Uint64Strict(" + buf.String() + ")"
}

func fileHasImport(file *ast.File, path string) bool {
	for _, spec := range file.Imports {
		if spec.Path != nil && spec.Path.Value == `"`+path+`"` {
			return true
		}
	}
	return false
}

func importEdit(pass *analysis.Pass, file *ast.File) analysis.TextEdit {
	insertPos := file.Name.End()
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		insertPos = genDecl.End()
	}
	newText := []byte("\n\nimport \"github.com/ethereum-optimism/optimism/op-service/bigs\"")
	return analysis.TextEdit{
		Pos:     insertPos,
		End:     insertPos,
		NewText: newText,
	}
}
