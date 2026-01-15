package optimismcustom

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("optimismcustom", New)
}

type MySettings struct {
	One   string    `json:"one"`
	Two   []Element `json:"two"`
	Three Element   `json:"three"`
}

type Element struct {
	Name string `json:"name"`
}

type PluginExample struct {
	settings MySettings
}

func New(settings any) (register.LinterPlugin, error) {
	fmt.Println("New optimismcustom plugin")
	// The configuration type will be map[string]any or []interface, it depends on your configuration.
	// You can use https://github.com/go-viper/mapstructure to convert map to struct.

	s, err := register.DecodeSettings[MySettings](settings)
	if err != nil {
		fmt.Println("Returning error")
		return nil, err
	}

	fmt.Println("Created plugin")
	return &PluginExample{settings: s}, nil
}
func (f *PluginExample) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	fmt.Println("Building analyzers")
	return []*analysis.Analyzer{
		{
			Name: "optimismcustom",
			Doc:  "find todos without an author",
			Run:  f.run,
		},
	}, nil
}

func (f *PluginExample) GetLoadMode() string {
	fmt.Println("GetLoadMode")
	// NOTE: the mode can be `register.LoadModeSyntax` or `register.LoadModeTypesInfo`.
	// - `register.LoadModeSyntax`: if the linter doesn't use types information.
	// - `register.LoadModeTypesInfo`: if the linter uses types information.

	return register.LoadModeSyntax
}

func (f *PluginExample) run(pass *analysis.Pass) (any, error) {
	fmt.Println("Run optimismcustom plugin")
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if comment, ok := n.(*ast.Comment); ok {
				if strings.HasPrefix(comment.Text, "// TODO:") || strings.HasPrefix(comment.Text, "// TODO():") {
					pass.Report(analysis.Diagnostic{
						Pos:            comment.Pos(),
						End:            0,
						Category:       "todo",
						Message:        "TODO comment has no author",
						SuggestedFixes: nil,
					})
				}
			}

			return true
		})
	}

	return nil, nil
}
