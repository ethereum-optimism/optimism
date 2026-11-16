package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// Use is one fork-API use: a package of this module that does not type-check against upstream
// go-ethereum, and what it needs from the fork.
type Use struct {
	// Package is the import path relative to the module root ("." for the root package), with a
	// " (tests)" suffix for uses in its test files.
	Package string
	// Need names what is missing upstream, e.g. "missing go-ethereum/core/types.NewIsthmusSigner".
	Need string
}

func (u Use) String() string { return u.Package + " | " + u.Need }

var (
	undefinedQualified = regexp.MustCompile(`^undefined: \w+\.(\w+)$`)
	missingMember      = regexp.MustCompile(`^.+ undefined \(type .+ has no field or method (\w+)`)
	unknownField       = regexp.MustCompile(`^unknown field (\w+) in struct literal`)
	missingMethod      = regexp.MustCompile(`does not implement .+ \(missing method (\w+)\)$`)
	wrongArgCount      = regexp.MustCompile(`^(?:too many|not enough) arguments in call to (.+)$`)
	wrongArgType       = regexp.MustCompile(`^cannot use .+ in argument to (.+?)(?:: .*)?$`)
	wrongResultCount   = regexp.MustCompile(`^assignment mismatch: .+ but (.+) returns \d+ values?$`)
	wrongAssignType    = regexp.MustCompile(`^cannot use .+ as .+ value in assignment`)
	cannotIndirect     = regexp.MustCompile(`^invalid operation: cannot indirect `)
	mismatchedTypes    = regexp.MustCompile(`^invalid operation: .+ \(mismatched types .+ and .+\)$`)
	missingPackage     = regexp.MustCompile(`(?:does not contain package|no required module provides package) ([^\s;:]+)`)
)

// collectUses type-checks every package (including tests) matching patterns in dir and returns
// the fork-API uses of the packages in module modPath.
//
// Every package of the module is type-checked from source, and go/types continues past errors,
// so each package reports only its own uses: a dependency that fails to type-check does not mask
// the errors of its importers, which `go build` and `go vet` would.
func collectUses(ctx context.Context, dir, modPath, gethPath string, buildFlags, patterns []string) (map[Use]bool, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Context:    ctx,
		Dir:        dir,
		Tests:      true,
		BuildFlags: buildFlags,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	c := classifier{gethPath: gethPath}
	uses := make(map[Use]bool)
	var unexpected []string
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		if pkg.Module != nil && pkg.Module.Path == modPath {
			return
		}
		// A dependency that fails, e.g. on a module whose download failed, would leave its
		// importers type-checked against an incomplete API, hiding uses.
		for _, e := range pkg.Errors {
			if !c.expectedListError(e) {
				unexpected = append(unexpected, fmt.Sprintf("%s: %s", pkg.ID, e))
			}
		}
	})
	for _, pkg := range pkgs {
		if pkg.Module == nil || pkg.Module.Path != modPath {
			continue
		}
		rel := relPackage(pkg, modPath)
		for _, e := range pkg.Errors {
			if e.Kind == packages.TypeError {
				continue // classified from pkg.TypeErrors, which carry exact positions
			}
			if !c.expectedListError(e) {
				unexpected = append(unexpected, fmt.Sprintf("%s: %s", pkg.ID, e))
			}
		}
		for _, te := range pkg.TypeErrors {
			need, ok := c.classify(pkg, te)
			if !ok {
				unexpected = append(unexpected, fmt.Sprintf("%s: %s", pkg.ID, te))
				continue
			}
			files := []string{pkg.Fset.File(te.Pos).Name()}
			if short, ok := strings.CutPrefix(need, "missing package go-ethereum"); ok {
				// The type checker reports a failed import once per package, not per file.
				if importers := importingFiles(pkg, gethPath+short); len(importers) > 0 {
					files = importers
				}
			}
			for _, file := range files {
				u := Use{Package: rel, Need: need}
				if strings.HasSuffix(file, "_test.go") {
					// Kept apart so that a use in tests doesn't admit the same use in production.
					u.Package += " (tests)"
				}
				uses[u] = true
			}
		}
	}
	if len(unexpected) > 0 {
		return nil, fmt.Errorf("unexpected errors (a dependency failing to load can be a failed module download):\n  %s", strings.Join(unexpected, "\n  "))
	}
	return uses, nil
}

type classifier struct {
	gethPath string
}

// expectedListError reports whether a non-type error of a package follows from the switch to
// upstream go-ethereum, with the use itself reported by the package's type errors.
func (c classifier) expectedListError(e packages.Error) bool {
	msg, _, _ := strings.Cut(e.Msg, "\n")
	if m := missingPackage.FindStringSubmatch(msg); m != nil {
		return c.isGeth(m[1])
	}
	// Compiler output from building export data: the package is type-checked from source too.
	return e.Kind == packages.ListError && strings.HasPrefix(msg, "# ")
}

// classify describes the fork API a type error needs, in terms that don't depend on local names
// or import aliases. Errors it has no description for are kept verbatim, marked "unclassified".
// It reports !ok for a failed import of a package outside go-ethereum.
func (c classifier) classify(pkg *packages.Package, te types.Error) (need string, ok bool) {
	msg, _, _ := strings.Cut(te.Msg, "\n")
	if rest, ok := strings.CutPrefix(msg, "could not import "); ok {
		path, _, _ := strings.Cut(rest, " ")
		if !c.isGeth(path) {
			return "", false
		}
		return "missing package " + c.short(path), true
	}

	path := enclosingPath(pkg, te.Pos)
	if m := undefinedQualified.FindStringSubmatch(msg); m != nil && len(path) > 1 {
		if sel, ok := path[1].(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				if pn, ok := pkg.TypesInfo.Uses[x].(*types.PkgName); ok {
					return "missing " + c.short(pn.Imported().Path()) + "." + m[1], true
				}
			}
		}
	}
	if m := missingMember.FindStringSubmatch(msg); m != nil && len(path) > 1 {
		if sel, ok := path[1].(*ast.SelectorExpr); ok {
			return "missing " + c.typeString(pkg.TypesInfo.TypeOf(sel.X)) + "." + m[1], true
		}
	}
	if m := unknownField.FindStringSubmatch(msg); m != nil && len(path) > 2 {
		if lit, ok := path[2].(*ast.CompositeLit); ok {
			return "missing " + c.typeString(pkg.TypesInfo.TypeOf(lit)) + "." + m[1], true
		}
	}
	if m := missingMethod.FindStringSubmatch(msg); m != nil {
		if arg := enclosingArg(path); arg != nil {
			return "missing " + c.typeString(pkg.TypesInfo.TypeOf(arg)) + "." + m[1], true
		}
	}
	for _, re := range []*regexp.Regexp{wrongArgCount, wrongArgType, wrongResultCount} {
		if m := re.FindStringSubmatch(msg); m != nil {
			call := callTo(path, m[1])
			if need := c.changedSignature(pkg.TypesInfo, call); need != "" {
				return need, true
			}
			// f(g()) passes all of g's results, so a changed result count of g fails at f.
			if call != nil && len(call.Args) == 1 {
				if inner, ok := ast.Unparen(call.Args[0]).(*ast.CallExpr); ok {
					if need := c.changedSignature(pkg.TypesInfo, inner); need != "" {
						return need, true
					}
				}
			}
		}
	}
	if wrongAssignType.MatchString(msg) {
		if field := c.gethField(pkg.TypesInfo, assignedTo(path)); field != "" {
			return "changed type of " + field, true
		}
	}
	if cannotIndirect.MatchString(msg) {
		if star, ok := innermost[*ast.StarExpr](path); ok {
			if field := c.gethField(pkg.TypesInfo, star.X); field != "" {
				return "changed type of " + field, true
			}
		}
	}
	if mismatchedTypes.MatchString(msg) {
		if bin := binaryExprIn(path, msg); bin != nil {
			for _, x := range []ast.Expr{bin.X, bin.Y} {
				if need := c.changedOperand(pkg, x); need != "" {
					return need, true
				}
			}
		}
	}
	return "unclassified: " + msg, true
}

// changedOperand names the go-ethereum API whose changed type can make the operand x mismatch:
// a field it selects, a function it calls, or the function whose result it was defined from.
func (c classifier) changedOperand(pkg *packages.Package, x ast.Expr) string {
	if field := c.gethField(pkg.TypesInfo, x); field != "" {
		return "changed type of " + field
	}
	switch x := ast.Unparen(x).(type) {
	case *ast.CallExpr:
		return c.changedSignature(pkg.TypesInfo, x)
	case *ast.Ident:
		v, ok := pkg.TypesInfo.Uses[x].(*types.Var)
		if !ok {
			return ""
		}
		// The variable's declaring identifier sits directly in the left-hand side of its
		// definition, e.g. `ret, left, err := evm.Call(...)`.
		def := enclosingPath(pkg, v.Pos())
		if len(def) < 2 {
			return ""
		}
		if assign, ok := def[1].(*ast.AssignStmt); ok && len(assign.Rhs) == 1 {
			if call, ok := ast.Unparen(assign.Rhs[0]).(*ast.CallExpr); ok {
				return c.changedSignature(pkg.TypesInfo, call)
			}
		}
	}
	return ""
}

// changedSignature names call's function if it is a go-ethereum function, or returns "".
func (c classifier) changedSignature(info *types.Info, call *ast.CallExpr) string {
	if fn := calledFunc(info, call); fn != nil && c.isGethObject(fn) {
		return "changed signature of " + c.funcName(fn)
	}
	return ""
}

func (c classifier) isGeth(path string) bool {
	return path == c.gethPath || strings.HasPrefix(path, c.gethPath+"/")
}

func (c classifier) isGethObject(obj types.Object) bool {
	return obj.Pkg() != nil && c.isGeth(obj.Pkg().Path())
}

// gethField names the go-ethereum struct field that e selects, or returns "" if it selects none.
func (c classifier) gethField(info *types.Info, e ast.Expr) string {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	s := info.Selections[sel]
	if s == nil || s.Kind() != types.FieldVal || !c.isGethObject(s.Obj()) {
		return ""
	}
	return c.typeString(s.Recv()) + "." + sel.Sel.Name
}

// short abbreviates go-ethereum import paths to "go-ethereum/...".
func (c classifier) short(path string) string {
	if c.isGeth(path) {
		return "go-ethereum" + strings.TrimPrefix(path, c.gethPath)
	}
	return path
}

func (c classifier) typeString(t types.Type) string {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	return types.TypeString(t, func(p *types.Package) string { return c.short(p.Path()) })
}

func (c classifier) funcName(fn *types.Func) string {
	if recv := fn.Signature().Recv(); recv != nil {
		return c.typeString(recv.Type()) + "." + fn.Name()
	}
	return c.short(fn.Pkg().Path()) + "." + fn.Name()
}

// enclosingPath returns the AST nodes enclosing pos, innermost first.
func enclosingPath(pkg *packages.Package, pos token.Pos) []ast.Node {
	for _, f := range pkg.Syntax {
		if f.FileStart <= pos && pos <= f.FileEnd {
			path, _ := astutil.PathEnclosingInterval(f, pos, pos)
			return path
		}
	}
	return nil
}

// callTo returns the innermost call enclosing path's node whose function the type checker
// prints as fun. Matching by name matters: the error position may lie in a nested call.
func callTo(path []ast.Node, fun string) *ast.CallExpr {
	for _, n := range path {
		if call, ok := n.(*ast.CallExpr); ok && types.ExprString(call.Fun) == fun {
			return call
		}
	}
	return nil
}

func calledFunc(info *types.Info, call *ast.CallExpr) *types.Func {
	if call == nil {
		return nil
	}
	var id *ast.Ident
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		id = fun
	case *ast.SelectorExpr:
		id = fun.Sel
	default:
		return nil
	}
	fn, _ := info.Uses[id].(*types.Func)
	return fn
}

// binaryExprIn returns the binary expression of path that msg reports. Matching the printed
// expression matters: the error position, the start of the expression, also starts its leftmost
// operand, which may be a binary expression itself.
func binaryExprIn(path []ast.Node, msg string) *ast.BinaryExpr {
	for _, n := range path {
		if bin, ok := n.(*ast.BinaryExpr); ok && strings.HasPrefix(msg, "invalid operation: "+types.ExprString(bin)+" (") {
			return bin
		}
	}
	return nil
}

// innermost returns the innermost node of type T in path.
func innermost[T ast.Node](path []ast.Node) (T, bool) {
	for _, n := range path {
		if t, ok := n.(T); ok {
			return t, true
		}
	}
	var zero T
	return zero, false
}

// assignedTo returns the left-hand side that the innermost node of path is assigned to, or nil.
func assignedTo(path []ast.Node) ast.Expr {
	assign, ok := innermost[*ast.AssignStmt](path)
	if !ok || len(assign.Lhs) != len(assign.Rhs) {
		return nil
	}
	for i, rhs := range assign.Rhs {
		if rhs.Pos() <= path[0].Pos() && path[0].End() <= rhs.End() {
			return assign.Lhs[i]
		}
	}
	return nil
}

// enclosingArg returns the call argument that contains the innermost node of path.
func enclosingArg(path []ast.Node) ast.Expr {
	for _, n := range path {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			continue
		}
		for _, arg := range call.Args {
			if arg.Pos() <= path[0].Pos() && path[0].End() <= arg.End() {
				return arg
			}
		}
	}
	return nil
}

func importingFiles(pkg *packages.Package, path string) []string {
	var files []string
	for _, f := range pkg.Syntax {
		for _, imp := range f.Imports {
			if p, err := strconv.Unquote(imp.Path.Value); err == nil && p == path {
				files = append(files, pkg.Fset.File(f.Pos()).Name())
			}
		}
	}
	return files
}

// relPackage returns the package path relative to the module root, naming an external test
// package after the package it tests.
func relPackage(pkg *packages.Package, modPath string) string {
	path := pkg.PkgPath
	if strings.HasSuffix(pkg.Name, "_test") {
		path = strings.TrimSuffix(path, "_test")
	}
	if path == modPath {
		return "."
	}
	return strings.TrimPrefix(path, modPath+"/")
}
