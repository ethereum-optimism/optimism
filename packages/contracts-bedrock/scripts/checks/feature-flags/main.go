package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	registryPath        = "feature-flags.yaml"
	devFeaturesSolPath  = "src/libraries/DevFeatures.sol"
	devFeaturesGoPath   = "../../op-core/devfeatures/devfeatures.go"
	featuresSolPath     = "src/libraries/Features.sol"
	configSolPath       = "scripts/libraries/Config.sol"
	featureFlagsSolPath = "test/setup/FeatureFlags.sol"

	devEnvPrefix = "DEV_FEATURE__"
	sysEnvPrefix = "SYS_FEATURE__"

	typeDev = "dev"
	typeSys = "sys"

	lifecycleActive      = "active"
	lifecycleHardcodedOn = "hardcoded-on"
	lifecycleLegacy      = "legacy"
)

// sysSetupConsumerFiles are setup files that may consume and activate sys feature readers.
var sysSetupConsumerFiles = []string{
	"scripts/deploy/Deploy.s.sol",
	"scripts/L2Genesis.s.sol",
	"test/setup/CommonTest.sol",
	"test/setup/Setup.sol",
}

// Registry mirrors feature-flags.yaml.
type Registry struct {
	Version      int          `yaml:"version"`
	Features     []Feature    `yaml:"features"`
	Combinations Combinations `yaml:"combinations"`
}

type Feature struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Lifecycle string `yaml:"lifecycle"`
}

type Combinations struct {
	Matrix   [][]string          `yaml:"matrix"`
	Requires map[string][]string `yaml:"requires"`
	Excludes [][]string          `yaml:"excludes"`
}

// DevFeatureConst is a bytes32 constant from DevFeatures.sol.
type DevFeatureConst struct {
	Name string
	Hex  string // 64 character lowercase hex without the 0x prefix
}

// GoDevFeatureConst is a common.Hash variable from op-core/devfeatures/devfeatures.go.
type GoDevFeatureConst struct {
	Name string
	Hex  string // 64 character lowercase hex without the 0x prefix
}

// SysFeatureConst is a bytes32 constant from Features.sol.
type SysFeatureConst struct {
	Name    string
	Literal string
}

// ConfigReader is a feature environment reader from Config.sol.
type ConfigReader struct {
	FuncName string
	EnvVar   string
}

// FeatureFlagsSol captures the two facts we extract from test/setup/FeatureFlags.sol.
type FeatureFlagsSol struct {
	// ResolveDev maps Config reader names to DevFeatures constants.
	ResolveDev map[string]string
	// NameMap maps feature constants to environment variable strings.
	NameMap map[string]string
}

// SysFeatureSetup captures setup branches that read and activate sys features.
type SysFeatureSetup struct {
	// Readers records Config reader functions used as setup branch guards.
	Readers map[string]bool
	// Activations maps Config reader functions to the sys features activated inside their branch.
	Activations map[string]map[string]bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	registry, err := loadRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}

	devConsts, solHardcoded, err := readDevFeaturesSol(devFeaturesSolPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}
	goConsts, goHardcoded, err := readDevFeaturesGo(devFeaturesGoPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}
	sysConsts, err := readFeaturesSol(featuresSolPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}
	readers, err := readConfigSol(configSolPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}
	ff, err := readFeatureFlagsSol(featureFlagsSolPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}
	sysSetup, err := scanSysFeatureSetup(sysSetupConsumerFiles)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}

	var errs []string
	errs = append(errs, validateRegistry(registry)...)
	errs = append(errs, validateDefinitions(registry, devConsts, sysConsts)...)
	errs = append(errs, validateGoParity(registry, devConsts, goConsts, solHardcoded, goHardcoded)...)
	errs = append(errs, validateWiring(registry, readers, ff, sysSetup)...)

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "feature-flags: "+e)
		}
		return fmt.Errorf("feature-flags check failed: %d error(s)", len(errs))
	}
	return nil
}

// File loading and scanning

func loadRegistry(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Registry{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parseRegistry(data)
}

func readDevFeaturesSol(path string) ([]DevFeatureConst, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	consts := parseDevFeaturesSol(string(data))
	hardcoded, err := parseHardcodedDevFeaturesSol(string(data))
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return consts, hardcoded, nil
}

func readDevFeaturesGo(path string) ([]GoDevFeatureConst, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	consts, hardcoded, err := parseDevFeaturesGo(string(data))
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return consts, hardcoded, nil
}

func readFeaturesSol(path string) ([]SysFeatureConst, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return parseFeaturesSol(string(data)), nil
}

func readConfigSol(path string) ([]ConfigReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return parseConfigSol(string(data)), nil
}

func readFeatureFlagsSol(path string) (FeatureFlagsSol, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FeatureFlagsSol{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parseFeatureFlagsSol(string(data)), nil
}

func newSysFeatureSetup() SysFeatureSetup {
	return SysFeatureSetup{
		Readers:     map[string]bool{},
		Activations: map[string]map[string]bool{},
	}
}

func (s SysFeatureSetup) addActivation(reader string, feature string) {
	if s.Activations[reader] == nil {
		s.Activations[reader] = map[string]bool{}
	}
	s.Activations[reader][feature] = true
}

func (s SysFeatureSetup) activates(reader string, feature string) bool {
	return s.Activations[reader][feature]
}

func scanSysFeatureSetup(paths []string) (SysFeatureSetup, error) {
	setup := newSysFeatureSetup()
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return SysFeatureSetup{}, fmt.Errorf("read %s: %w", p, err)
		}
		scanSysFeatureSetupContent(string(data), &setup)
	}
	return setup, nil
}

// Parsing

func parseRegistry(data []byte) (Registry, error) {
	var r Registry
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return Registry{}, fmt.Errorf("parse registry yaml: %w", err)
	}
	return r, nil
}

var (
	// DevFeatures.sol constants may use single line or multiline declarations.
	devFeatureConstRe = regexp.MustCompile(`(?s)bytes32\s+public\s+constant\s+(\w+)\s*=\s*bytes32\(\s*(0x[0-9a-fA-F]+)\s*\)\s*;`)

	devFeatureEnabledFuncRe = regexp.MustCompile(`function\s+isDevFeatureEnabled\s*\(`)
	hardcodedSolIfRe        = regexp.MustCompile(`(?s)if\s*\((.*?)\)\s*(?:\{\s*return\s+true\s*;\s*\}|return\s+true\s*;)`)
	solHasFlagFeatureRe     = regexp.MustCompile(`(?s)^hasFlag\s*\(\s*_feature\s*,\s*(?:DevFeatures\.)?(\w+)\s*\)$`)
	solFeatureEqRe          = regexp.MustCompile(`(?s)^(?:_feature\s*==\s*(?:DevFeatures\.)?(\w+)|(?:DevFeatures\.)?(\w+)\s*==\s*_feature)$`)

	// Features.sol constants use an internal bytes32 name and string literal.
	sysFeatureConstRe = regexp.MustCompile(`bytes32\s+internal\s+constant\s+(\w+)\s*=\s*"([^"]+)"\s*;`)

	// Config.sol reader names may differ from the environment variable string.
	configReaderRe = regexp.MustCompile(`(?s)function\s+(\w+)\s*\(\s*\)\s*internal\s+view\s+returns\s*\(\s*bool\s*\)\s*\{\s*return\s+vm\.envOr\(\s*"([^"]+)"\s*,\s*false\s*\)\s*;\s*\}`)

	// FeatureFlags.sol resolveFeaturesFromEnv branches pair Config readers with dev feature constants.
	resolveDevRe = regexp.MustCompile(`(?s)if\s*\(\s*Config\.(\w+)\s*\(\s*\)\s*\)\s*\{[^}]*devFeatureBitmap\s*\|=\s*DevFeatures\.(\w+)\s*;[^}]*\}`)

	// FeatureFlags.sol getFeatureName branches.
	nameMapRe = regexp.MustCompile(`(?s)_feature\s*==\s*(DevFeatures|Features)\.(\w+)\s*\)\s*\{\s*return\s+"([^"]+)"`)

	// Setup sys-feature branches.
	configIfRe          = regexp.MustCompile(`if\s*\(\s*Config\.(\w+)\s*\(\s*\)\s*\)\s*\{`)
	setFeatureTrueRe    = regexp.MustCompile(`setFeature\s*\(\s*Features\.(\w+)\s*,\s*true\s*\)`)
	customGasTokenCfgRe = regexp.MustCompile(`\bsetUseCustomGasToken\s*\(\s*true\s*\)`)
)

func parseDevFeaturesSol(content string) []DevFeatureConst {
	content = stripComments(content)
	var out []DevFeatureConst
	for _, m := range devFeatureConstRe.FindAllStringSubmatch(content, -1) {
		out = append(out, DevFeatureConst{Name: m[1], Hex: normalizeHex(m[2])})
	}
	return out
}

func parseHardcodedDevFeaturesSol(content string) ([]string, error) {
	content = stripComments(content)
	funcMatch := devFeatureEnabledFuncRe.FindStringIndex(content)
	if funcMatch == nil {
		return nil, fmt.Errorf("missing isDevFeatureEnabled function")
	}
	openBrace := strings.IndexByte(content[funcMatch[1]:], '{')
	if openBrace < 0 {
		return nil, fmt.Errorf("isDevFeatureEnabled missing function body")
	}
	body, ok := extractBraceBody(content, funcMatch[1]+openBrace)
	if !ok {
		return nil, fmt.Errorf("could not parse isDevFeatureEnabled function body")
	}

	var out []string
	for _, m := range hardcodedSolIfRe.FindAllStringSubmatch(body, -1) {
		feature, ok := solHardcodedFeatureFromCondition(m[1])
		if ok {
			out = append(out, feature)
		}
	}
	return out, nil
}

func solHardcodedFeatureFromCondition(condition string) (string, bool) {
	condition = strings.TrimSpace(condition)
	if m := solHasFlagFeatureRe.FindStringSubmatch(condition); m != nil {
		return m[1], true
	}
	if m := solFeatureEqRe.FindStringSubmatch(condition); m != nil {
		if m[1] != "" {
			return m[1], true
		}
		return m[2], true
	}
	return "", false
}

func parseDevFeaturesGo(content string) ([]GoDevFeatureConst, []string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "devfeatures.go", content, 0)
	if err != nil {
		return nil, nil, err
	}
	consts, err := parseDevFeaturesGoConsts(fset, file)
	if err != nil {
		return nil, nil, err
	}
	hardcoded, err := parseHardcodedDevFeaturesGo(fset, file)
	if err != nil {
		return nil, nil, err
	}
	return consts, hardcoded, nil
}

func parseDevFeaturesGoConsts(fset *token.FileSet, file *ast.File) ([]GoDevFeatureConst, error) {
	var out []GoDevFeatureConst
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				hex, ok, err := goHexToHashLiteral(valueSpec.Values[i])
				if err != nil {
					return nil, fmt.Errorf("%s: %w", fset.Position(valueSpec.Values[i].Pos()), err)
				}
				if !ok {
					continue
				}
				out = append(out, GoDevFeatureConst{Name: name.Name, Hex: normalizeHex(hex)})
			}
		}
	}
	return out, nil
}

func goHexToHashLiteral(expr ast.Expr) (string, bool, error) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false, nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "HexToHash" {
		return "", false, nil
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "common" {
		return "", false, nil
	}
	if len(call.Args) != 1 {
		return "", true, fmt.Errorf("common.HexToHash must have exactly one argument")
	}
	arg, ok := call.Args[0].(*ast.BasicLit)
	if !ok || arg.Kind != token.STRING {
		return "", true, fmt.Errorf("common.HexToHash argument must be a string literal")
	}
	hex, err := strconv.Unquote(arg.Value)
	if err != nil {
		return "", true, fmt.Errorf("parse common.HexToHash string literal: %w", err)
	}
	if err := validateGoHexLiteral(hex); err != nil {
		return "", true, err
	}
	return hex, true, nil
}

func validateGoHexLiteral(hex string) error {
	raw := strings.TrimPrefix(strings.ToLower(hex), "0x")
	if raw == "" {
		return fmt.Errorf("common.HexToHash literal %q is empty", hex)
	}
	if len(raw) > 64 {
		return fmt.Errorf("common.HexToHash literal %q is longer than 32 bytes", hex)
	}
	for _, r := range raw {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return fmt.Errorf("common.HexToHash literal %q is not valid hex", hex)
	}
	return nil
}

func parseHardcodedDevFeaturesGo(fset *token.FileSet, file *ast.File) ([]string, error) {
	var out []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "IsDevFeatureEnabled" || fn.Body == nil {
			continue
		}
		featureParam, err := secondParamName(fn)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fset.Position(fn.Pos()), err)
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			ifStmt, ok := node.(*ast.IfStmt)
			if !ok {
				return true
			}
			if !ifBodyReturnsTrue(ifStmt.Body) {
				return true
			}
			feature, ok := goHardcodedFeatureFromCondition(ifStmt.Cond, featureParam)
			if ok {
				out = append(out, feature)
			}
			return true
		})
		return out, nil
	}
	return nil, fmt.Errorf("missing IsDevFeatureEnabled function")
}

func secondParamName(fn *ast.FuncDecl) (string, error) {
	if fn.Type.Params == nil {
		return "", fmt.Errorf("IsDevFeatureEnabled missing second parameter")
	}
	paramIndex := 0
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			paramIndex++
			if paramIndex == 2 {
				return name.Name, nil
			}
		}
	}
	return "", fmt.Errorf("IsDevFeatureEnabled missing second parameter")
}

func ifBodyReturnsTrue(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}
	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	result, ok := ret.Results[0].(*ast.Ident)
	return ok && result.Name == "true"
}

func goHardcodedFeatureFromCondition(expr ast.Expr, featureParam string) (string, bool) {
	switch expr := unwrapParenExpr(expr).(type) {
	case *ast.CallExpr:
		if !isIdentNamed(expr.Fun, "hasFlag") || len(expr.Args) != 2 {
			return "", false
		}
		if exprName(expr.Args[0]) != featureParam {
			return "", false
		}
		feature := exprName(expr.Args[1])
		return feature, feature != ""
	case *ast.BinaryExpr:
		if expr.Op != token.EQL {
			return "", false
		}
		left := exprName(expr.X)
		right := exprName(expr.Y)
		switch {
		case left == featureParam && right != "":
			return right, true
		case right == featureParam && left != "":
			return left, true
		default:
			return "", false
		}
	default:
		return "", false
	}
}

func unwrapParenExpr(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}

func isIdentNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func exprName(expr ast.Expr) string {
	switch expr := unwrapParenExpr(expr).(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	default:
		return ""
	}
}

func parseFeaturesSol(content string) []SysFeatureConst {
	content = stripComments(content)
	var out []SysFeatureConst
	for _, m := range sysFeatureConstRe.FindAllStringSubmatch(content, -1) {
		out = append(out, SysFeatureConst{Name: m[1], Literal: m[2]})
	}
	return out
}

func parseConfigSol(content string) []ConfigReader {
	content = stripComments(content)
	var out []ConfigReader
	for _, m := range configReaderRe.FindAllStringSubmatch(content, -1) {
		envVar := m[2]
		if !strings.HasPrefix(envVar, devEnvPrefix) && !strings.HasPrefix(envVar, sysEnvPrefix) {
			continue
		}
		out = append(out, ConfigReader{FuncName: m[1], EnvVar: envVar})
	}
	return out
}

func parseFeatureFlagsSol(content string) FeatureFlagsSol {
	content = stripComments(content)
	ff := FeatureFlagsSol{
		ResolveDev: map[string]string{},
		NameMap:    map[string]string{},
	}
	for _, m := range resolveDevRe.FindAllStringSubmatch(content, -1) {
		ff.ResolveDev[m[1]] = m[2]
	}
	for _, m := range nameMapRe.FindAllStringSubmatch(content, -1) {
		ff.NameMap[m[1]+"."+m[2]] = m[3]
	}
	return ff
}

func scanSysFeatureSetupContent(content string, setup *SysFeatureSetup) {
	content = stripComments(content)
	for _, m := range configIfRe.FindAllStringSubmatchIndex(content, -1) {
		reader := content[m[2]:m[3]]
		setup.Readers[reader] = true

		body, ok := extractBraceBody(content, m[1]-1)
		if !ok {
			continue
		}
		for _, featureMatch := range setFeatureTrueRe.FindAllStringSubmatch(body, -1) {
			setup.addActivation(reader, featureMatch[1])
		}
		if customGasTokenCfgRe.MatchString(body) {
			setup.addActivation(reader, "CUSTOM_GAS_TOKEN")
		}
	}
}

func extractBraceBody(content string, openBrace int) (string, bool) {
	if openBrace < 0 || openBrace >= len(content) || content[openBrace] != '{' {
		return "", false
	}
	depth := 0
	for i := openBrace; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[openBrace+1 : i], true
			}
		}
	}
	return "", false
}

// normalizeHex returns a 64 character lowercase hex string with no 0x prefix.
func normalizeHex(s string) string {
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if len(s) >= 64 {
		return s
	}
	return strings.Repeat("0", 64-len(s)) + s
}

func stripComments(content string) string {
	// Strip block comments before line comments.
	var b strings.Builder
	i := 0
	for i < len(content) {
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '*' {
			end := strings.Index(content[i+2:], "*/")
			if end < 0 {
				break
			}
			i += end + 4
			continue
		}
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '/' {
			end := strings.IndexByte(content[i:], '\n')
			if end < 0 {
				break
			}
			i += end
			continue
		}
		b.WriteByte(content[i])
		i++
	}
	return b.String()
}

// Validation

var upperSnakeRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func validateRegistry(r Registry) []string {
	var errs []string

	if r.Version != 1 {
		errs = append(errs, fmt.Sprintf("feature-flags.yaml version must be 1, got %d", r.Version))
	}

	// Build the feature index and validate feature metadata.
	known := map[string]Feature{}
	declarationOrder := map[string]int{}
	for i, f := range r.Features {
		if !upperSnakeRe.MatchString(f.Name) {
			errs = append(errs, fmt.Sprintf("feature-flags.yaml feature name %q is not UPPER_SNAKE_CASE", f.Name))
		}
		if _, dup := known[f.Name]; dup {
			errs = append(errs, fmt.Sprintf("feature-flags.yaml duplicate feature %q", f.Name))
		}
		known[f.Name] = f
		declarationOrder[f.Name] = i
		switch f.Type {
		case typeDev, typeSys:
		default:
			errs = append(errs, fmt.Sprintf("feature-flags.yaml feature %q has invalid type %q (want dev|sys)", f.Name, f.Type))
		}
		switch f.Lifecycle {
		case lifecycleActive, lifecycleHardcodedOn, lifecycleLegacy:
		default:
			errs = append(errs, fmt.Sprintf("feature-flags.yaml feature %q has invalid lifecycle %q (want active|hardcoded-on|legacy)", f.Name, f.Lifecycle))
		}
	}

	refOK := func(name string) bool { _, ok := known[name]; return ok }

	for key, prereqs := range r.Combinations.Requires {
		if !refOK(key) {
			errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.requires key %q references unknown feature", key))
		}
		for _, p := range prereqs {
			if !refOK(p) {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.requires[%s] references unknown feature %q", key, p))
			}
		}
	}

	for i, excl := range r.Combinations.Excludes {
		for _, name := range excl {
			if !refOK(name) {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.excludes[%d] references unknown feature %q", i, name))
			}
		}
	}

	// Matrix.
	baselineRows := 0
	rowsContaining := map[string]int{}
	for i, row := range r.Combinations.Matrix {
		if len(row) == 0 {
			baselineRows++
			continue
		}
		// Matrix rows use registry declaration order.
		prevIdx := -1
		for _, name := range row {
			if !refOK(name) {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.matrix[%d] references unknown feature %q", i, name))
				continue
			}
			if declarationOrder[name] <= prevIdx {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.matrix[%d] not in declaration order: %v", i, row))
				break
			}
			prevIdx = declarationOrder[name]
		}
		for _, name := range row {
			rowsContaining[name]++
		}

		// Validate requires and excludes for this row.
		inRow := map[string]bool{}
		for _, name := range row {
			inRow[name] = true
		}
		for key, prereqs := range r.Combinations.Requires {
			if !inRow[key] {
				continue
			}
			for _, p := range prereqs {
				if inRow[p] {
					continue
				}
				if f, ok := known[p]; ok && f.Lifecycle == lifecycleHardcodedOn {
					continue // lifecycleHardcodedOn prerequisites are always enabled
				}
				errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.matrix[%d] enables %s but missing required prerequisite %s", i, key, p))
			}
		}
		for j, excl := range r.Combinations.Excludes {
			all := true
			for _, name := range excl {
				if !inRow[name] {
					all = false
					break
				}
			}
			if all && len(excl) > 0 {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.matrix[%d] violates excludes[%d]: %v", i, j, excl))
			}
		}
	}
	if baselineRows != 1 {
		errs = append(errs, fmt.Sprintf("feature-flags.yaml combinations.matrix must contain exactly one baseline [] row, got %d", baselineRows))
	}

	// Check lifecycle constraints across all combinations.
	for _, f := range r.Features {
		switch f.Lifecycle {
		case lifecycleLegacy:
			if rowsContaining[f.Name] > 0 {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml legacy feature %s must not appear in combinations.matrix", f.Name))
			}
			if _, ok := r.Combinations.Requires[f.Name]; ok {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml legacy feature %s must not be a combinations.requires key", f.Name))
			}
			for _, prereqs := range r.Combinations.Requires {
				for _, p := range prereqs {
					if p == f.Name {
						errs = append(errs, fmt.Sprintf("feature-flags.yaml legacy feature %s must not be a combinations.requires prerequisite", f.Name))
					}
				}
			}
			for _, excl := range r.Combinations.Excludes {
				for _, name := range excl {
					if name == f.Name {
						errs = append(errs, fmt.Sprintf("feature-flags.yaml legacy feature %s must not appear in combinations.excludes", f.Name))
					}
				}
			}
		case lifecycleActive:
			if rowsContaining[f.Name] == 0 {
				errs = append(errs, fmt.Sprintf("feature-flags.yaml active feature %s must appear in at least one combinations.matrix row", f.Name))
			}
		}
	}

	return errs
}

func validateDefinitions(r Registry, devConsts []DevFeatureConst, sysConsts []SysFeatureConst) []string {
	var errs []string

	yamlDev, yamlSys := map[string]bool{}, map[string]bool{}
	for _, f := range r.Features {
		switch f.Type {
		case typeDev:
			yamlDev[f.Name] = true
		case typeSys:
			yamlSys[f.Name] = true
		}
	}

	// Dev features must match DevFeatures.sol.
	solDev := map[string]string{}
	for _, c := range devConsts {
		if _, dup := solDev[c.Name]; dup {
			errs = append(errs, fmt.Sprintf("DevFeatures.sol duplicate constant %s", c.Name))
		}
		solDev[c.Name] = c.Hex
	}
	for name := range yamlDev {
		if _, ok := solDev[name]; !ok {
			errs = append(errs, fmt.Sprintf("DevFeatures.sol missing constant for dev feature %s", name))
		}
	}
	for name := range solDev {
		if !yamlDev[name] {
			errs = append(errs, fmt.Sprintf("DevFeatures.sol declares %s but feature-flags.yaml does not list it as a dev feature", name))
		}
	}

	// Dev feature values must follow the bitmap convention in DevFeatures.sol.
	seenHex := map[string]string{}
	for _, c := range devConsts {
		if !yamlDev[c.Name] {
			continue
		}
		if len(c.Hex) != 64 {
			errs = append(errs, fmt.Sprintf("DevFeatures.%s hex value 0x%s must be 64 hex characters", c.Name, c.Hex))
			continue
		}
		if isZeroHex(c.Hex) {
			errs = append(errs, fmt.Sprintf("DevFeatures.%s hex value is zero", c.Name))
		}
		if !exactlyOneNibble(c.Hex) {
			errs = append(errs, fmt.Sprintf("DevFeatures.%s hex value 0x%s must have exactly one nibble set", c.Name, c.Hex))
		}
		if other, dup := seenHex[c.Hex]; dup {
			errs = append(errs, fmt.Sprintf("DevFeatures.%s hex value 0x%s duplicates %s", c.Name, c.Hex, other))
		}
		seenHex[c.Hex] = c.Name
	}

	// Sys features must match Features.sol.
	solSys := map[string]string{}
	for _, c := range sysConsts {
		if _, dup := solSys[c.Name]; dup {
			errs = append(errs, fmt.Sprintf("Features.sol duplicate constant %s", c.Name))
		}
		solSys[c.Name] = c.Literal
		if c.Literal != c.Name {
			errs = append(errs, fmt.Sprintf("Features.sol constant %s has string literal %q (must equal constant name)", c.Name, c.Literal))
		}
	}
	for name := range yamlSys {
		if _, ok := solSys[name]; !ok {
			errs = append(errs, fmt.Sprintf("Features.sol missing constant for sys feature %s", name))
		}
	}
	for name := range solSys {
		if !yamlSys[name] {
			errs = append(errs, fmt.Sprintf("Features.sol declares %s but feature-flags.yaml does not list it as a sys feature", name))
		}
	}

	return errs
}

func validateGoParity(
	r Registry,
	solConsts []DevFeatureConst,
	goConsts []GoDevFeatureConst,
	solHardcoded []string,
	goHardcoded []string,
) []string {
	var errs []string

	solHexByName := map[string]string{}
	solNameByHex := map[string]string{}
	for _, c := range solConsts {
		solHexByName[c.Name] = c.Hex
		if _, ok := solNameByHex[c.Hex]; !ok {
			solNameByHex[c.Hex] = c.Name
		}
	}

	goHexByName := map[string]string{}
	goNameByHex := map[string]string{}
	for _, c := range goConsts {
		goHexByName[c.Name] = c.Hex
		if other, dup := goNameByHex[c.Hex]; dup {
			errs = append(errs, fmt.Sprintf("devfeatures.go duplicate hex 0x%s for constants %s and %s", c.Hex, other, c.Name))
		}
		goNameByHex[c.Hex] = c.Name
	}

	for _, name := range sortedKeys(solHexByName) {
		hex := solHexByName[name]
		if _, ok := goNameByHex[hex]; !ok {
			errs = append(errs, fmt.Sprintf("devfeatures.go missing constant matching DevFeatures.%s (hex 0x%s)", name, hex))
		}
	}
	for _, hex := range sortedKeys(goNameByHex) {
		goName := goNameByHex[hex]
		if _, ok := solNameByHex[hex]; !ok {
			errs = append(errs, fmt.Sprintf("devfeatures.go has extra constant %s (hex 0x%s) not in Solidity", goName, hex))
		}
	}

	solHardcodedByHex, solHardcodedErrs := resolveHardcodedFeatures(solHardcoded, solHexByName, "DevFeatures.sol isDevFeatureEnabled", "DevFeatures.")
	errs = append(errs, solHardcodedErrs...)
	goHardcodedByHex, goHardcodedErrs := resolveHardcodedFeatures(goHardcoded, goHexByName, "devfeatures.go IsDevFeatureEnabled", "devfeatures.")
	errs = append(errs, goHardcodedErrs...)

	featureByName := map[string]Feature{}
	for _, f := range r.Features {
		featureByName[f.Name] = f
	}
	expectedHardcodedByHex, expectedErrs := expectedHardcodedFeatures(r, solHexByName)
	errs = append(errs, expectedErrs...)
	errs = append(errs, validateExpectedHardcoded("DevFeatures.sol isDevFeatureEnabled", expectedHardcodedByHex, solHardcodedByHex)...)
	errs = append(errs, validateExpectedHardcoded("devfeatures.go IsDevFeatureEnabled", expectedHardcodedByHex, goHardcodedByHex)...)
	errs = append(errs, validateHardcodedLifecycle("DevFeatures.sol isDevFeatureEnabled", "DevFeatures.", solHardcodedByHex, solNameByHex, featureByName)...)
	errs = append(errs, validateHardcodedLifecycle("devfeatures.go IsDevFeatureEnabled", "devfeatures.", goHardcodedByHex, solNameByHex, featureByName)...)

	sort.Strings(errs)
	return errs
}

func expectedHardcodedFeatures(r Registry, solHexByName map[string]string) (map[string]string, []string) {
	expectedByHex := map[string]string{}
	var errs []string
	for _, f := range r.Features {
		if f.Lifecycle != lifecycleHardcodedOn {
			continue
		}
		if f.Type != typeDev {
			errs = append(errs, fmt.Sprintf("feature-flags.yaml marks %s as hardcoded-on, but type is %q (expected dev)", f.Name, f.Type))
			continue
		}
		hex, ok := solHexByName[f.Name]
		if !ok {
			errs = append(errs, fmt.Sprintf("feature-flags.yaml marks %s as hardcoded-on, but DevFeatures.sol has no matching constant", f.Name))
			continue
		}
		expectedByHex[hex] = f.Name
	}
	return expectedByHex, errs
}

func validateExpectedHardcoded(source string, expectedByHex map[string]string, actualByHex map[string]string) []string {
	var errs []string
	for _, hex := range sortedKeys(expectedByHex) {
		name := expectedByHex[hex]
		if _, ok := actualByHex[hex]; !ok {
			errs = append(errs, fmt.Sprintf("%s missing hardcoded true branch for feature-flags.yaml hardcoded-on DevFeatures.%s (hex 0x%s)", source, name, hex))
		}
	}
	return errs
}

func resolveHardcodedFeatures(names []string, hexByName map[string]string, source string, namePrefix string) (map[string]string, []string) {
	hardcodedByHex := map[string]string{}
	var errs []string
	for _, name := range names {
		hex, ok := hexByName[name]
		if !ok {
			errs = append(errs, fmt.Sprintf("%s hardcodes unknown constant %s%s", source, namePrefix, name))
			continue
		}
		if _, seen := hardcodedByHex[hex]; !seen {
			hardcodedByHex[hex] = name
		}
	}
	return hardcodedByHex, errs
}

func validateHardcodedLifecycle(
	source string,
	namePrefix string,
	hardcodedByHex map[string]string,
	solNameByHex map[string]string,
	featureByName map[string]Feature,
) []string {
	var errs []string
	for _, hex := range sortedKeys(hardcodedByHex) {
		branchName := hardcodedByHex[hex]
		solName, ok := solNameByHex[hex]
		if !ok {
			errs = append(errs, fmt.Sprintf("%s hardcodes %s%s (hex 0x%s), but DevFeatures.sol does not define that hex", source, namePrefix, branchName, hex))
			continue
		}
		feature, ok := featureByName[solName]
		if !ok {
			errs = append(errs, fmt.Sprintf("%s hardcodes %s%s (hex 0x%s), matching DevFeatures.%s, but feature-flags.yaml does not list it", source, namePrefix, branchName, hex, solName))
			continue
		}
		if feature.Type != typeDev {
			errs = append(errs, fmt.Sprintf("%s hardcodes %s%s (hex 0x%s), matching %s, but feature-flags.yaml type is %q (expected dev)", source, namePrefix, branchName, hex, solName, feature.Type))
			continue
		}
		if feature.Lifecycle != lifecycleHardcodedOn {
			errs = append(errs, fmt.Sprintf("%s hardcodes %s%s (hex 0x%s), matching DevFeatures.%s, but feature-flags.yaml lifecycle is %q (expected hardcoded-on)", source, namePrefix, branchName, hex, solName, feature.Lifecycle))
		}
	}
	return errs
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateWiring(
	r Registry,
	readers []ConfigReader,
	ff FeatureFlagsSol,
	sysSetup SysFeatureSetup,
) []string {
	var errs []string

	// Build the reader index using the environment variable string as the key.
	envToFunc := map[string]string{}
	for _, c := range readers {
		envToFunc[c.EnvVar] = c.FuncName
	}

	for _, f := range r.Features {
		if f.Lifecycle == lifecycleLegacy {
			continue
		}
		var envVar string
		switch f.Type {
		case typeDev:
			envVar = devEnvPrefix + f.Name
		case typeSys:
			envVar = sysEnvPrefix + f.Name
		default:
			continue
		}
		funcName, hasReader := envToFunc[envVar]
		if !hasReader {
			errs = append(errs, fmt.Sprintf("Config.sol missing reader for %s feature %s: expected vm.envOr(%q, false)", f.Type, f.Name, envVar))
		}

		// getFeatureName supports both dev and sys features.
		var nameKey string
		if f.Type == typeDev {
			nameKey = "DevFeatures." + f.Name
		} else {
			nameKey = "Features." + f.Name
		}
		gotEnv, mapped := ff.NameMap[nameKey]
		if !mapped {
			errs = append(errs, fmt.Sprintf("FeatureFlags.sol getFeatureName missing branch for %s: expected %q", nameKey, envVar))
		} else if gotEnv != envVar {
			errs = append(errs, fmt.Sprintf("FeatureFlags.sol getFeatureName: %s maps to %q, expected %q", nameKey, gotEnv, envVar))
		}

		if f.Type == typeDev {
			// resolveFeaturesFromEnv must set the corresponding dev feature bit.
			if hasReader {
				gotConst, ok := ff.ResolveDev[funcName]
				if !ok {
					errs = append(errs, fmt.Sprintf("FeatureFlags.sol resolveFeaturesFromEnv missing branch: if (Config.%s()) ... devFeatureBitmap |= DevFeatures.%s", funcName, f.Name))
				} else if gotConst != f.Name {
					errs = append(errs, fmt.Sprintf("FeatureFlags.sol resolveFeaturesFromEnv: Config.%s pairs with DevFeatures.%s, expected DevFeatures.%s", funcName, gotConst, f.Name))
				}
			}
		} else {
			// Sys feature readers must be consumed and activated by the same setup branch.
			if hasReader && !sysSetup.Readers[funcName] {
				errs = append(errs, fmt.Sprintf("%s reader Config.%s() is not consumed by any setup path (%s), so CI would set an environment variable with no effect", envVar, funcName, strings.Join(sysSetupConsumerFiles, ", ")))
			}
			if hasReader && !sysSetup.activates(funcName, f.Name) {
				errs = append(errs, fmt.Sprintf("%s reader Config.%s() does not activate Features.%s in the same setup branch; expected setFeature(Features.%s, true) or an equivalent config setter", envVar, funcName, f.Name, f.Name))
			}
		}
	}

	// Report extra Config readers for features that are not in the registry.
	known := map[string]bool{}
	for _, f := range r.Features {
		known[f.Name] = true
	}
	for envVar := range envToFunc {
		var name string
		switch {
		case strings.HasPrefix(envVar, devEnvPrefix):
			name = strings.TrimPrefix(envVar, devEnvPrefix)
		case strings.HasPrefix(envVar, sysEnvPrefix):
			name = strings.TrimPrefix(envVar, sysEnvPrefix)
		default:
			continue
		}
		if !known[name] {
			errs = append(errs, fmt.Sprintf("Config.sol has reader for %q but feature-flags.yaml does not list %s", envVar, name))
		}
	}
	// Keep error ordering stable for tests and diffs.
	sort.Strings(errs)
	return errs
}

func isZeroHex(h string) bool {
	for _, r := range h {
		if r != '0' {
			return false
		}
	}
	return true
}

// exactlyOneNibble reports whether h has one nonzero nibble and that nibble is a power of two.
func exactlyOneNibble(h string) bool {
	nonZero := 0
	var val byte
	for i := 0; i < len(h); i++ {
		c := h[i]
		if c == '0' {
			continue
		}
		nonZero++
		val = c
	}
	if nonZero != 1 {
		return false
	}
	switch val {
	case '1', '2', '4', '8':
		return true
	default:
		return false
	}
}
