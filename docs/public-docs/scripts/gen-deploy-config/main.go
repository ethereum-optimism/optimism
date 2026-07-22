// Command gen-deploy-config generates the MDX schema reference for the OP
// Stack DeployConfig — the flat JSON deployment configuration defined by the
// DeployConfig struct tree in op-chain-ops/genesis — so the docs schema
// reference is derived from the Go struct tags and doc comments instead of
// hand-transcribed.
//
// The tool parses the Go source of the genesis package (go/ast, no build
// required, so it can run against the source checked out at a release tag),
// walks the DeployConfig struct and its embedded config structs in
// declaration order, and emits one snippet:
//
//	docs/public-docs/snippets/generated/deploy-config-schema.mdx
//
// Each embedded config struct becomes a section; each field becomes a table
// row with its JSON key (from the json:"..." struct tag), a friendly type,
// and its Go doc comment. The snippet carries a do-not-edit header and a
// provenance line naming the release tag recorded in manifest.json.
//
// Run from the monorepo root:
//
//	go run ./docs/public-docs/scripts/gen-deploy-config
//
// Regeneration is keyed to finalized op-deployer releases (op-deployer is the
// released tool that derives and emits the DeployConfig): the snippet is
// generated from the genesis package source at the op-deployer release tag
// recorded in manifest.json — see README.md in this directory for the
// procedure, including how to point -source at the tag's source when develop
// has moved past it. Pass -check to verify the committed snippet matches the
// SHA-256 recorded in manifest.json (hand-edit detection) without writing.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// rootStruct is the struct the walk starts from.
const rootStruct = "DeployConfig"

// componentKey is the manifest.json key for this schema's provenance entry.
const componentKey = "deploy-config"

// groupTitles maps each struct that contributes fields to the schema to the
// section title it renders under. The generator fails on a struct missing
// from this table, so a new embedded config struct upstream forces a
// conscious title choice here instead of silently inheriting a Go type name.
var groupTitles = map[string]string{
	"DevDeployConfig":             "Development accounts",
	"L2GenesisBlockDeployConfig":  "L2 genesis block",
	"OwnershipDeployConfig":       "Ownership",
	"L2VaultsDeployConfig":        "Fee vaults",
	"GovernanceDeployConfig":      "Governance token",
	"GasPriceOracleDeployConfig":  "Gas price oracle",
	"GasTokenDeployConfig":        "Custom gas token",
	"OperatorDeployConfig":        "Operator addresses",
	"EIP1559DeployConfig":         "EIP-1559 fee market",
	"UpgradeScheduleDeployConfig": "Network upgrade (hardfork) activations",
	"L2CoreDeployConfig":          "Core protocol parameters",
	"FeeMarketConfig":             "Fee market limits",
	"AltDADeployConfig":           "Alt-DA mode",
	"DevL1DeployConfig":           "Development L1 genesis",
	"DeployConfig":                "L1 starting block",
	"SuperchainL1DeployConfig":    "Superchain configuration",
	"OutputOracleDeployConfig":    "Legacy output oracle",
	"FaultProofDeployConfig":      "Fault proofs",
	"L1DependenciesConfig":        "L1 dependency addresses",
	"LegacyDeployConfig":          "Legacy fields",
}

// typeLabels maps the Go source type of a field to the label rendered in the
// Type column. The generator fails on a type missing from this table, so a
// new field type upstream forces a conscious, human-readable label instead of
// leaking a Go type name into the docs.
//
// The WithdrawalNetwork semantics mirror its Valid()/ToUint8() methods in
// op-chain-ops/genesis/withdrawal_network.go: "remote" (0) withdraws to L1,
// "local" (1) withdraws to L2, and the legacy uint8 values are accepted when
// unmarshalling.
var typeLabels = map[string]string{
	"bool":                             "boolean",
	"string":                           "string",
	"int":                              "number",
	"int64":                            "number",
	"uint16":                           "number",
	"uint32":                           "number",
	"uint64":                           "number",
	"hexutil.Uint64":                   "number (hex-encoded)",
	"*hexutil.Uint64":                  "number (hex-encoded), nullable",
	"*hexutil.Big":                     "number (hex-encoded big integer)",
	"common.Hash":                      "32-byte hash",
	"common.Address":                   "address",
	"WithdrawalNetwork":                `string: "remote" (withdraw to L1) or "local" (withdraw to L2); legacy 0/1 accepted`,
	"*params.BlobScheduleConfig":       "object (go-ethereum blob schedule)",
	"*MarshalableRPCBlockNumberOrHash": "L1 block number, tag, or hash",
}

type field struct {
	jsonKey  string
	typeName string
	doc      string
}

type group struct {
	structName string
	doc        string
	fields     []field
}

// structDecl is one struct type collected from the parsed package source.
type structDecl struct {
	typ *ast.StructType
	doc string
}

func main() {
	docsDir := flag.String("docs-dir", "docs/public-docs", "path to the docs root (run from the monorepo root)")
	sourceDir := flag.String("source", "op-chain-ops/genesis", "path to the genesis package source to parse (point at a release-tag checkout to generate at the tag)")
	check := flag.Bool("check", false, "verify the committed snippet against the sha256 recorded in manifest.json instead of writing; exit nonzero on any mismatch (current-source parity is reported informationally)")
	flag.Parse()

	docsRoot, err := filepath.Abs(filepath.Clean(*docsDir))
	if err != nil {
		fatalf("resolving docs dir %s: %v", *docsDir, err)
	}

	manifestPath, err := pathUnder(docsRoot, "scripts", "gen-deploy-config", "manifest.json")
	if err != nil {
		fatalf("%v", err)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		fatalf("reading %s (run from the monorepo root): %v", manifestPath, err)
	}
	var m map[string]*manifestEntry
	if err := json.Unmarshal(raw, &m); err != nil {
		fatalf("parsing %s: %v", manifestPath, err)
	}
	entry, ok := m[componentKey]
	if !ok || entry.Tag == "" {
		fatalf("no release tag for %q in %s", componentKey, manifestPath)
	}

	structs, err := parsePackage(*sourceDir)
	if err != nil {
		fatalf("parsing %s: %v", *sourceDir, err)
	}
	groups, err := walk(structs, rootStruct)
	if err != nil {
		fatalf("walking %s: %v", rootStruct, err)
	}
	content, err := render(groups, entry.Tag)
	if err != nil {
		fatalf("rendering: %v", err)
	}

	outPath, err := pathUnder(docsRoot, "snippets", "generated", "deploy-config-schema.mdx")
	if err != nil {
		fatalf("%v", err)
	}

	if *check {
		committed, err := os.ReadFile(outPath)
		if err != nil {
			fatalf("reading committed snippet %s: %v", outPath, err)
		}
		if entry.Sha256 == "" {
			fatalf("no sha256 for %q in %s; regenerate per README.md", componentKey, manifestPath)
		}
		if got := hashOf(committed); got != entry.Sha256 {
			fatalf("%s does not match the sha256 recorded in manifest.json (tag %s): the snippet was hand-edited or the manifest was not updated with it; regenerate per README.md", outPath, entry.Tag)
		}
		if string(committed) == content {
			fmt.Printf("ok: %s matches manifest sha256 and current-source output (%s)\n", outPath, entry.Tag)
		} else {
			// Not a failure: the snippet documents the release tag, and the
			// genesis package on this tree may have moved past it. The snippet
			// is refreshed when the next finalized release is published.
			fmt.Printf("ok: %s matches manifest sha256 (%s); note: this tree's schema source has moved past the tag, snippet stays pinned to the release\n", outPath, entry.Tag)
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		fatalf("creating %s: %v", filepath.Dir(outPath), err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o640); err != nil {
		fatalf("writing %s: %v", outPath, err)
	}
	entry.Sha256 = hashOf([]byte(content))
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fatalf("encoding %s: %v", manifestPath, err)
	}
	if err := os.WriteFile(manifestPath, append(out, '\n'), 0o640); err != nil {
		fatalf("writing %s: %v", manifestPath, err)
	}
	nFields := 0
	for _, g := range groups {
		nFields += len(g.fields)
	}
	fmt.Printf("wrote %s (%d values in %d groups)\n", outPath, nFields, len(groups))
}

type manifestEntry struct {
	Tag    string `json:"tag"`
	Sha256 string `json:"sha256,omitempty"`
}

// parsePackage parses every non-test .go file in dir and returns the struct
// type declarations by name, with their doc comments.
func parsePackage(dir string) (map[string]structDecl, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	structs := make(map[string]structDecl)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				doc := ts.Doc
				if doc == nil {
					doc = gd.Doc
				}
				structs[ts.Name.Name] = structDecl{typ: st, doc: doc.Text()}
			}
		}
	}
	if len(structs) == 0 {
		return nil, fmt.Errorf("no struct declarations found in %s", dir)
	}
	return structs, nil
}

// walk flattens the struct's embedded config structs into an ordered list of
// groups, mirroring how encoding/json flattens the embedded fields into one
// flat JSON object. Direct (non-embedded) fields of a struct form that
// struct's own group, inserted at the position of its first direct field.
func walk(structs map[string]structDecl, name string) ([]group, error) {
	decl, ok := structs[name]
	if !ok {
		return nil, fmt.Errorf("struct %s not found in package source", name)
	}
	var groups []group
	own := -1 // index in groups of this struct's direct-field group
	for _, f := range decl.typ.Fields.List {
		if len(f.Names) == 0 {
			// Embedded struct: resolve locally and recurse. encoding/json
			// flattens an embedded struct only when its tag carries no JSON
			// name: a named embed marshals as a nested object and a "-" embed
			// is dropped entirely, and neither is a flat DeployConfig value.
			if f.Tag != nil {
				tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get("json")
				if jsonName := strings.Split(tag, ",")[0]; jsonName == "-" {
					continue
				} else if jsonName != "" {
					return nil, fmt.Errorf("in %s: an embedded struct has json name %q; nested objects are not supported by this generator", name, jsonName)
				}
			}
			typeName, err := embeddedName(f.Type)
			if err != nil {
				return nil, fmt.Errorf("in %s: %w", name, err)
			}
			sub, err := walk(structs, typeName)
			if err != nil {
				return nil, err
			}
			groups = append(groups, sub...)
			continue
		}
		key, err := jsonKey(f)
		if err != nil {
			return nil, fmt.Errorf("in %s: %w", name, err)
		}
		if key == "" { // json:"-"
			continue
		}
		typeStr, err := typeString(f.Type)
		if err != nil {
			return nil, fmt.Errorf("in %s: %w", name, err)
		}
		label, ok := typeLabels[typeStr]
		if !ok {
			return nil, fmt.Errorf("field %s in %s has type %q with no entry in typeLabels; add a human-readable label", key, name, typeStr)
		}
		if own == -1 {
			groups = append(groups, group{structName: name, doc: decl.doc})
			own = len(groups) - 1
		}
		groups[own].fields = append(groups[own].fields, field{
			jsonKey:  key,
			typeName: label,
			doc:      f.Doc.Text(),
		})
	}
	return groups, nil
}

// embeddedName returns the local type name of an embedded field. Embedded
// types from other packages are rejected: the walk can only resolve structs
// declared in the parsed package, so a cross-package embed must fail loudly
// instead of dropping its fields from the schema.
func embeddedName(expr ast.Expr) (string, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, nil
	default:
		return "", fmt.Errorf("unsupported embedded field type %T (cross-package embeds are not resolvable from source)", expr)
	}
}

// jsonKey extracts the JSON key from a field's struct tag. A field without a
// json tag would be marshalled under its Go name; no DeployConfig field does
// that today, so it is treated as an error to force a conscious decision.
func jsonKey(f *ast.Field) (string, error) {
	if f.Tag == nil {
		return "", fmt.Errorf("field %s has no struct tag", f.Names[0].Name)
	}
	tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get("json")
	if tag == "" {
		return "", fmt.Errorf("field %s has no json tag", f.Names[0].Name)
	}
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return "", nil
	}
	return name, nil
}

func typeString(expr ast.Expr) (string, error) {
	var b bytes.Buffer
	if err := printer.Fprint(&b, token.NewFileSet(), expr); err != nil {
		return "", err
	}
	return b.String(), nil
}

func render(groups []group, tag string) (string, error) {
	seen := make(map[string]bool)
	nFields := 0
	for _, g := range groups {
		if _, ok := groupTitles[g.structName]; !ok {
			return "", fmt.Errorf("struct %s has no entry in groupTitles; add a section title", g.structName)
		}
		for _, f := range g.fields {
			if seen[f.jsonKey] {
				return "", fmt.Errorf("duplicate JSON key %q", f.jsonKey)
			}
			seen[f.jsonKey] = true
			nFields++
		}
	}

	tagPath := strings.ReplaceAll(tag, "/", "%2F")
	var b strings.Builder
	fmt.Fprintf(&b, `{/*
  GENERATED FILE — DO NOT EDIT.

  Generated by docs/public-docs/scripts/gen-deploy-config from the
  DeployConfig struct tree in op-chain-ops/genesis at the release tag below.
  Do not hand-edit; regenerate per
  docs/public-docs/scripts/gen-deploy-config/README.md when a new finalized
  op-deployer release is published:

    go run ./docs/public-docs/scripts/gen-deploy-config
*/}

Generated from the [`+"`DeployConfig`"+` struct](https://github.com/ethereum-optimism/optimism/blob/%[1]s/op-chain-ops/genesis/config.go)
in `+"`op-chain-ops/genesis`"+` at [`+"`%[2]s`"+`](https://github.com/ethereum-optimism/optimism/releases/tag/%[1]s):
%[3]d values in %[4]d groups. The JSON key for each value is its `+"`json:\"...\"`"+` struct
tag there; descriptions are the Go doc comments. When this page and the source
disagree, the source wins.

`, tagPath, tag, nFields, len(groups))

	for _, g := range groups {
		fmt.Fprintf(&b, "### %s\n\n", groupTitles[g.structName])
		// The root struct's doc describes the whole DeployConfig, not the
		// small group its direct fields form; skip it as a section intro.
		if doc := strings.TrimSpace(g.doc); doc != "" && g.structName != rootStruct {
			fmt.Fprintf(&b, "%s\n\n", cell(doc))
		}
		b.WriteString("| JSON key | Type | Description |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, f := range g.fields {
			desc := cell(strings.TrimSpace(f.doc))
			if desc == "" {
				desc = "—"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", f.jsonKey, cell(f.typeName), desc)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

func hashOf(b []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

// cell escapes text for use inside an MDX table cell: newlines become
// spaces, column separators (|) are escaped everywhere (GFM tables split on
// them even inside code spans), and characters MDX would treat as JSX or
// expression syntax ({, }, <) are escaped outside inline-code spans only —
// code spans already protect them, and escaping inside one would render the
// backslashes literally. Text with unbalanced backticks is escaped in full
// as a safe fallback.
func cell(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	jsxEscaper := strings.NewReplacer(
		"{", "\\{",
		"}", "\\}",
		"<", "\\<",
	)
	parts := strings.Split(s, "`")
	if len(parts)%2 == 0 { // unbalanced backticks
		return jsxEscaper.Replace(s)
	}
	for i := 0; i < len(parts); i += 2 { // even indices are outside code spans
		parts[i] = jsxEscaper.Replace(parts[i])
	}
	return strings.Join(parts, "`")
}

// pathUnder joins elems onto root, cleans the result, and verifies the
// resolved path still lies under root. Every path this tool writes is built
// from in-repo constants, not user input, but the check keeps a bad docs-dir
// value from escaping the docs tree.
func pathUnder(root string, elems ...string) (string, error) {
	p := filepath.Clean(filepath.Join(append([]string{root}, elems...)...))
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %s escapes docs dir %s", p, root)
	}
	return p, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen-deploy-config: "+format+"\n", args...)
	os.Exit(1)
}
