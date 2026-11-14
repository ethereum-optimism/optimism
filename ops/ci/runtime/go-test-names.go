//go:build ignore

// List statically registered tests without running package init or TestMain.
package main

import (
	"encoding/json"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() {
	var paths []string
	if err := json.NewDecoder(os.Stdin).Decode(&paths); err != nil {
		panic(err)
	}
	names := map[string]bool{}
	files := make([]*ast.File, 0, len(paths))
	for _, path := range paths {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		files = append(files, file)
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			name := function.Name.Name
			if name == "TestMain" {
				continue
			}
			for _, prefix := range []string{"Test", "Fuzz"} {
				if !strings.HasPrefix(name, prefix) {
					continue
				}
				next, _ := utf8.DecodeRuneInString(strings.TrimPrefix(name, prefix))
				if !unicode.IsLower(next) {
					names[name] = true
				}
			}
		}
	}
	for _, example := range doc.Examples(files...) {
		if example.Output != "" || example.EmptyOutput {
			names["Example"+example.Name] = true
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
