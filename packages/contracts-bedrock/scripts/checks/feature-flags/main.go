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
	circleCIPath        = "../../.circleci/continue/main.yml"
	checksYamlPath      = "checks.yaml"

	devEnvPrefix = "DEV_FEATURE__"
	sysEnvPrefix = "SYS_FEATURE__"

	typeDev = "dev"
	typeSys = "sys"

	lifecycleActive      = "active"
	lifecycleHardcodedOn = "hardcoded-on"
	lifecycleLegacy      = "legacy"

	featuresMatrixAnchor = "features_matrix"
	ciBaselineRow        = "main"

	workflowContractsFeatureTests     = "contracts-feature-tests"
	checkFastFeatureTestsInstanceName = "contracts-bedrock-checks-fast-feature-tests"
	jobChecksFast                     = "contracts-bedrock-checks-fast"
	jobContractsBedrockTests          = "contracts-bedrock-tests"
	jobContractsBedrockCoverage       = "contracts-bedrock-coverage"
	jobContractsBedrockTestsUpgrade   = "contracts-bedrock-tests-upgrade"
	jobContractsBedrockTestsL2Fork    = "contracts-bedrock-tests-l2-fork"
	jobRequiredContractsCI            = "required-contracts-ci"

	featureFlagsCheckName       = "feature-flags"
	featureFlagsCheckCommand    = "go run ./scripts/checks/feature-flags"
	checksFastCommand           = "just check-fast"
	contractsBedrockWorkingDir  = "packages/contracts-bedrock"
	requireTerminalValue        = "terminal"
	setupFeaturesCommandName    = "setup-features"
	setupFeaturesFeaturesParam  = "features"
	setupFeaturesSystemFeatures = "system_features"
	featuresParameterTemplate   = "<<parameters.features>>"
	matrixFeaturesTemplate      = "<<matrix.features>>"
)

// checksYamlFeaturePhases is the set of phases that may host the feature-flags check.
// Build-gated phases (e.g. source, dev) are not allowed because the check must
// run before the contracts are built.
var checksYamlFeaturePhases = map[string]bool{
	"setup":     true,
	"pre-build": true,
}

// setupFeaturesRequiredJobs lists every CircleCI job that must invoke the
// setup-features command.
var setupFeaturesRequiredJobs = []string{
	jobContractsBedrockTests,
	jobContractsBedrockCoverage,
	jobContractsBedrockTestsUpgrade,
	jobContractsBedrockTestsL2Fork,
}

// setupFeaturesAllowedJobs is derived from setupFeaturesRequiredJobs so the
// allowlist and required-job list cannot drift.
var setupFeaturesAllowedJobs = stringSet(setupFeaturesRequiredJobs)

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

// contractsFeatureTestsMatrixConsumers is the expected set of matrix-driven job
// instances in the contracts-feature-tests workflow.
var contractsFeatureTestsMatrixConsumers = []string{
	"contracts-bedrock-tests-heavy-fuzz-modified",
	"contracts-bedrock-tests",
	"contracts-bedrock-tests-develop",
	"contracts-bedrock-coverage",
	"contracts-bedrock-tests-upgrade op-mainnet",
	"contracts-bedrock-tests-upgrade-develop op-mainnet",
}

// contractsFeatureTestsEmptyDefaultJobs are job definitions whose
// parameters.features.default must be the empty string. tests-l2-fork is
// intentionally excluded.
var contractsFeatureTestsEmptyDefaultJobs = []string{
	jobContractsBedrockTests,
	jobContractsBedrockCoverage,
	jobContractsBedrockTestsUpgrade,
}

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

// CircleCIConfig is the feature subset extracted from .circleci/continue/main.yml.
type CircleCIConfig struct {
	SystemFeaturesDefault []string
	FeaturesMatrix        []string
}

// ChecksConfig is the feature-flags subset extracted from
// packages/contracts-bedrock/checks.yaml.
type ChecksConfig struct {
	FeatureFlagsCommand string
	FeatureFlagsPhase   string
	FeatureFlagsFound   bool
	FeatureFlagsCount   int
}

// CIControlPlane captures the CircleCI wiring that keeps feature-flag checks
// required: the checks-fast gate path, setup-features command usage, and the
// features_matrix anchor and consumers.
type CIControlPlane struct {
	ChecksFastCommand                   string
	ChecksFastWorkingDir                string
	HasCheckFastFeatureTests            bool
	RequiredCIReqsCheckFast             bool
	SetupFeaturesCommandExists          bool
	SetupFeaturesCommandCount           int
	SetupFeaturesHasFeaturesParam       bool
	SetupFeaturesHasSystemFeaturesParam bool
	SetupFeaturesCallers                []SetupFeaturesCaller
	FeaturesMatrixAnchorCount           int
	MatrixConsumers                     []MatrixConsumer
	FeatureDefaults                     map[string]string
}

// SetupFeaturesCaller is one invocation of the setup-features command.
type SetupFeaturesCaller struct {
	Job                    string
	Features               string
	SystemFeatures         string
	SystemFeaturesOverride bool
	Line                   int
}

// MatrixConsumer is one workflow-level job invocation that consumes the
// features matrix.
type MatrixConsumer struct {
	Workflow           string
	JobType            string
	InstanceName       string
	UsesFeaturesAnchor bool
	Line               int
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
	ci, controlPlane, err := readCircleCIConfig(circleCIPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}
	checksCfg, err := readChecksYaml(checksYamlPath)
	if err != nil {
		return fmt.Errorf("feature-flags: %w", err)
	}

	var errs []string
	errs = append(errs, validateRegistry(registry)...)
	errs = append(errs, validateDefinitions(registry, devConsts, sysConsts)...)
	errs = append(errs, validateGoParity(registry, devConsts, goConsts, solHardcoded, goHardcoded)...)
	errs = append(errs, validateWiring(registry, readers, ff, sysSetup)...)
	errs = append(errs, validateCIParity(registry, ci)...)
	errs = append(errs, validateChecksConfigControlPlane(checksCfg)...)
	errs = append(errs, validateCIControlPlane(controlPlane)...)

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

func readCircleCIConfig(path string) (CircleCIConfig, CIControlPlane, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CircleCIConfig{}, CIControlPlane{}, fmt.Errorf("read %s: %w", path, err)
	}
	doc, err := parseCircleCIDoc(data)
	if err != nil {
		return CircleCIConfig{}, CIControlPlane{}, err
	}
	cfg, err := parseCircleCIConfigFromDoc(doc)
	if err != nil {
		return CircleCIConfig{}, CIControlPlane{}, err
	}
	return cfg, parseCIControlPlane(doc), nil
}

func readChecksYaml(path string) (ChecksConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ChecksConfig{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parseChecksYaml(data)
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

func parseCircleCIDoc(data []byte) (*yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse circleci yaml: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parse circleci yaml: root must be a mapping")
	}
	return root.Content[0], nil
}

func parseCircleCIConfig(data []byte) (CircleCIConfig, error) {
	doc, err := parseCircleCIDoc(data)
	if err != nil {
		return CircleCIConfig{}, err
	}
	return parseCircleCIConfigFromDoc(doc)
}

func parseCircleCIConfigFromDoc(doc *yaml.Node) (CircleCIConfig, error) {
	sysDefault, err := readCISystemFeaturesDefault(doc)
	if err != nil {
		return CircleCIConfig{}, err
	}
	matrix, err := readCIFeaturesMatrix(doc)
	if err != nil {
		return CircleCIConfig{}, err
	}
	return CircleCIConfig{
		SystemFeaturesDefault: sysDefault,
		FeaturesMatrix:        matrix,
	}, nil
}

func parseChecksYaml(data []byte) (ChecksConfig, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return ChecksConfig{}, fmt.Errorf("parse checks yaml: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return ChecksConfig{}, fmt.Errorf("parse checks yaml: root must be a mapping")
	}
	doc := root.Content[0]

	phases := yamlMapNodeAt(doc, "phases")
	if phases == nil || phases.Kind != yaml.SequenceNode {
		return ChecksConfig{}, nil
	}
	var cfg ChecksConfig
	for _, phase := range phases.Content {
		if phase.Kind != yaml.MappingNode {
			continue
		}
		phaseName := ""
		if n := yamlMapNodeAt(phase, "name"); n != nil && n.Kind == yaml.ScalarNode {
			phaseName = strings.TrimSpace(n.Value)
		}
		checks := yamlMapNodeAt(phase, "checks")
		if checks == nil || checks.Kind != yaml.SequenceNode {
			continue
		}
		for _, check := range checks.Content {
			if check.Kind != yaml.MappingNode {
				continue
			}
			nameNode := yamlMapNodeAt(check, "name")
			if nameNode == nil || nameNode.Kind != yaml.ScalarNode || nameNode.Value != featureFlagsCheckName {
				continue
			}
			cfg.FeatureFlagsCount++
			if cfg.FeatureFlagsFound {
				continue
			}
			cmd := ""
			if c := yamlMapNodeAt(check, "command"); c != nil && c.Kind == yaml.ScalarNode {
				cmd = strings.TrimSpace(c.Value)
			}
			cfg.FeatureFlagsCommand = cmd
			cfg.FeatureFlagsPhase = phaseName
			cfg.FeatureFlagsFound = true
		}
	}
	return cfg, nil
}

func parseCIControlPlane(doc *yaml.Node) CIControlPlane {
	cp := CIControlPlane{FeatureDefaults: map[string]string{}}
	cp.ChecksFastCommand, cp.ChecksFastWorkingDir = extractChecksFastRunStep(doc)
	cp.SetupFeaturesCommandExists, cp.SetupFeaturesHasFeaturesParam, cp.SetupFeaturesHasSystemFeaturesParam = extractSetupFeaturesCommand(doc)
	cp.SetupFeaturesCommandCount = countMappingKey(yamlMapNodeAt(doc, "commands"), setupFeaturesCommandName)
	cp.SetupFeaturesCallers = collectSetupFeaturesCallers(doc)
	cp.FeaturesMatrixAnchorCount = countAnchors(doc, featuresMatrixAnchor)
	cp.MatrixConsumers = collectMatrixConsumers(doc, workflowContractsFeatureTests)
	cp.HasCheckFastFeatureTests = hasCheckFastFeatureTestsInvocation(doc, workflowContractsFeatureTests)
	cp.RequiredCIReqsCheckFast = requiredContractsCITerminalRequiresCheckFast(doc, workflowContractsFeatureTests)
	for _, job := range contractsFeatureTestsEmptyDefaultJobs {
		if v, ok := readJobFeatureDefault(doc, job); ok {
			cp.FeatureDefaults[job] = v
		}
	}
	return cp
}

func extractChecksFastRunStep(doc *yaml.Node) (command, workingDir string) {
	steps := yamlMapNodeAt(doc, "jobs", jobChecksFast, "steps")
	if steps == nil || steps.Kind != yaml.SequenceNode {
		return "", ""
	}
	var namedFallbackCommand, namedFallbackWorkingDir string
	var namedFallbackFound bool
	var substringFallbackCommand, substringFallbackWorkingDir string
	for _, step := range steps.Content {
		if step.Kind != yaml.MappingNode || len(step.Content) < 2 {
			continue
		}
		if step.Content[0].Value != "run" {
			continue
		}
		runVal := step.Content[1]
		if runVal.Kind != yaml.MappingNode {
			continue
		}
		var stepName, stepCmd, stepDir string
		for i := 0; i+1 < len(runVal.Content); i += 2 {
			switch runVal.Content[i].Value {
			case "name":
				stepName = runVal.Content[i+1].Value
			case "command":
				stepCmd = runVal.Content[i+1].Value
			case "working_directory":
				stepDir = runVal.Content[i+1].Value
			}
		}
		trimmedStepCmd := strings.TrimSpace(stepCmd)
		if trimmedStepCmd == checksFastCommand {
			return trimmedStepCmd, stepDir
		}
		// Keep likely stale check steps so errors print the actual command.
		if stepName == "Run checks" && !namedFallbackFound {
			namedFallbackFound = true
			namedFallbackCommand = stepCmd
			namedFallbackWorkingDir = stepDir
		} else if substringFallbackCommand == "" && strings.Contains(trimmedStepCmd, "check-fast") {
			substringFallbackCommand = stepCmd
			substringFallbackWorkingDir = stepDir
		}
	}
	if namedFallbackFound {
		return namedFallbackCommand, namedFallbackWorkingDir
	}
	return substringFallbackCommand, substringFallbackWorkingDir
}

func extractSetupFeaturesCommand(doc *yaml.Node) (exists, hasFeatures, hasSystemFeatures bool) {
	cmd := yamlMapNodeAt(doc, "commands", setupFeaturesCommandName)
	if cmd == nil || cmd.Kind != yaml.MappingNode {
		return false, false, false
	}
	exists = true
	params := yamlMapNodeAt(cmd, "parameters")
	if params == nil || params.Kind != yaml.MappingNode {
		return exists, false, false
	}
	for i := 0; i+1 < len(params.Content); i += 2 {
		switch params.Content[i].Value {
		case setupFeaturesFeaturesParam:
			hasFeatures = true
		case setupFeaturesSystemFeatures:
			hasSystemFeatures = true
		}
	}
	return exists, hasFeatures, hasSystemFeatures
}

func collectSetupFeaturesCallers(doc *yaml.Node) []SetupFeaturesCaller {
	var out []SetupFeaturesCaller
	jobs := yamlMapNodeAt(doc, "jobs")
	if jobs == nil || jobs.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i+1 < len(jobs.Content); i += 2 {
		jobName := jobs.Content[i].Value
		steps := yamlMapNodeAt(jobs.Content[i+1], "steps")
		if steps == nil || steps.Kind != yaml.SequenceNode {
			continue
		}
		for _, step := range steps.Content {
			if step.Kind == yaml.ScalarNode && step.Value == setupFeaturesCommandName {
				out = append(out, SetupFeaturesCaller{Job: jobName, Line: step.Line})
				continue
			}
			if step.Kind != yaml.MappingNode || len(step.Content) < 2 {
				continue
			}
			keyNode := step.Content[0]
			if keyNode.Value != setupFeaturesCommandName {
				continue
			}
			caller := SetupFeaturesCaller{Job: jobName, Line: keyNode.Line}
			valNode := step.Content[1]
			if valNode.Kind == yaml.MappingNode {
				for j := 0; j+1 < len(valNode.Content); j += 2 {
					switch valNode.Content[j].Value {
					case setupFeaturesFeaturesParam:
						caller.Features = strings.TrimSpace(valNode.Content[j+1].Value)
					case setupFeaturesSystemFeatures:
						caller.SystemFeaturesOverride = true
						caller.SystemFeatures = valNode.Content[j+1].Value
					}
				}
			}
			out = append(out, caller)
		}
	}
	return out
}

func countAnchors(node *yaml.Node, anchor string) int {
	if node == nil {
		return 0
	}
	n := 0
	if node.Anchor == anchor {
		n++
	}
	for _, c := range node.Content {
		n += countAnchors(c, anchor)
	}
	return n
}

func countMappingKey(node *yaml.Node, key string) int {
	if node == nil || node.Kind != yaml.MappingNode {
		return 0
	}
	n := 0
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			n++
		}
	}
	return n
}

func collectMatrixConsumers(doc *yaml.Node, workflow string) []MatrixConsumer {
	var out []MatrixConsumer
	jobs := yamlMapNodeAt(doc, "workflows", workflow, "jobs")
	if jobs == nil || jobs.Kind != yaml.SequenceNode {
		return out
	}
	for _, entry := range jobs.Content {
		if entry.Kind != yaml.MappingNode || len(entry.Content) < 2 {
			continue
		}
		keyNode := entry.Content[0]
		valNode := entry.Content[1]
		if valNode.Kind != yaml.MappingNode {
			continue
		}
		var name, features string
		var matrixFeaturesNode *yaml.Node
		for i := 0; i+1 < len(valNode.Content); i += 2 {
			k := valNode.Content[i].Value
			v := valNode.Content[i+1]
			switch k {
			case "name":
				name = v.Value
			case setupFeaturesFeaturesParam:
				features = strings.TrimSpace(v.Value)
			case "matrix":
				matrixFeaturesNode = yamlMapNodeAt(v, "parameters", setupFeaturesFeaturesParam)
			}
		}
		if features != matrixFeaturesTemplate {
			continue
		}
		usesAnchor := false
		if matrixFeaturesNode != nil {
			if matrixFeaturesNode.Anchor == featuresMatrixAnchor {
				usesAnchor = true
			}
			if matrixFeaturesNode.Kind == yaml.AliasNode && matrixFeaturesNode.Alias != nil && matrixFeaturesNode.Alias.Anchor == featuresMatrixAnchor {
				usesAnchor = true
			}
		}
		out = append(out, MatrixConsumer{
			Workflow:           workflow,
			JobType:            keyNode.Value,
			InstanceName:       stripMatrixFeaturesTemplate(name),
			UsesFeaturesAnchor: usesAnchor,
			Line:               keyNode.Line,
		})
	}
	return out
}

func stripMatrixFeaturesTemplate(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, matrixFeaturesTemplate, ""))
}

func hasCheckFastFeatureTestsInvocation(doc *yaml.Node, workflow string) bool {
	jobs := yamlMapNodeAt(doc, "workflows", workflow, "jobs")
	if jobs == nil || jobs.Kind != yaml.SequenceNode {
		return false
	}
	for _, entry := range jobs.Content {
		if entry.Kind != yaml.MappingNode || len(entry.Content) < 2 {
			continue
		}
		if entry.Content[0].Value != jobChecksFast {
			continue
		}
		valNode := entry.Content[1]
		if valNode.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(valNode.Content); i += 2 {
			if valNode.Content[i].Value == "name" && valNode.Content[i+1].Value == checkFastFeatureTestsInstanceName {
				return true
			}
		}
	}
	return false
}

func requiredContractsCITerminalRequiresCheckFast(doc *yaml.Node, workflow string) bool {
	jobs := yamlMapNodeAt(doc, "workflows", workflow, "jobs")
	if jobs == nil || jobs.Kind != yaml.SequenceNode {
		return false
	}
	for _, entry := range jobs.Content {
		if entry.Kind != yaml.MappingNode || len(entry.Content) < 2 {
			continue
		}
		if entry.Content[0].Value != jobRequiredContractsCI {
			continue
		}
		requires := yamlMapNodeAt(entry.Content[1], "requires")
		if requires == nil || requires.Kind != yaml.SequenceNode {
			continue
		}
		for _, req := range requires.Content {
			if req.Kind != yaml.MappingNode || len(req.Content) < 2 {
				continue
			}
			if req.Content[0].Value == checkFastFeatureTestsInstanceName && req.Content[1].Value == requireTerminalValue {
				return true
			}
		}
	}
	return false
}

func readJobFeatureDefault(doc *yaml.Node, jobName string) (string, bool) {
	n := yamlMapNodeAt(doc, "jobs", jobName, "parameters", setupFeaturesFeaturesParam, "default")
	if n == nil || n.Kind != yaml.ScalarNode {
		return "", false
	}
	return n.Value, true
}

func readCISystemFeaturesDefault(doc *yaml.Node) ([]string, error) {
	n := yamlMapNodeAt(doc, "commands", "setup-features", "parameters", "system_features", "default")
	if n == nil || n.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("circleci yaml: commands.setup-features.parameters.system_features.default missing or not a scalar")
	}
	return strings.Fields(n.Value), nil
}

func readCIFeaturesMatrix(doc *yaml.Node) ([]string, error) {
	n := yamlFindAnchorNode(doc, featuresMatrixAnchor)
	if n == nil {
		return nil, fmt.Errorf("circleci yaml: could not find &%s anchor", featuresMatrixAnchor)
	}
	if n.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("circleci yaml: &%s is not a sequence", featuresMatrixAnchor)
	}
	rows := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("circleci yaml: &%s row is not a scalar", featuresMatrixAnchor)
		}
		rows = append(rows, strings.TrimSpace(item.Value))
	}
	return rows, nil
}

func yamlMapNodeAt(node *yaml.Node, keys ...string) *yaml.Node {
	for _, key := range keys {
		if node == nil || node.Kind != yaml.MappingNode {
			return nil
		}
		var next *yaml.Node
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				next = node.Content[i+1]
				break
			}
		}
		node = next
	}
	return node
}

func yamlFindAnchorNode(node *yaml.Node, anchor string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Anchor == anchor {
		return node
	}
	for _, c := range node.Content {
		if got := yamlFindAnchorNode(c, anchor); got != nil {
			return got
		}
	}
	return nil
}

func parseCIMatrixRow(s string) []string {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, ciBaselineRow) {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
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

	sort.Strings(errs)
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
		if !exactlyOneBit(c.Hex) {
			errs = append(errs, fmt.Sprintf("DevFeatures.%s hex value 0x%s must have exactly one bit set", c.Name, c.Hex))
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

	sort.Strings(errs)
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

func renderCIMatrix(r Registry) []string {
	declarationOrder := map[string]int{}
	for i, f := range r.Features {
		declarationOrder[f.Name] = i
	}
	out := make([]string, 0, len(r.Combinations.Matrix))
	for _, row := range r.Combinations.Matrix {
		if len(row) == 0 {
			out = append(out, ciBaselineRow)
			continue
		}
		canon := append([]string(nil), row...)
		sort.SliceStable(canon, func(i, j int) bool {
			return declarationOrder[canon[i]] < declarationOrder[canon[j]]
		})
		out = append(out, strings.Join(canon, ","))
	}
	return out
}

func validateCIParity(r Registry, ci CircleCIConfig) []string {
	var errs []string

	// CircleCI stores system features as a space-separated string.
	var registrySys []string
	for _, f := range r.Features {
		if f.Type == typeSys {
			registrySys = append(registrySys, f.Name)
		}
	}
	sort.Strings(registrySys)
	gotSys := append([]string(nil), ci.SystemFeaturesDefault...)
	sort.Strings(gotSys)
	if !stringSlicesEqual(registrySys, gotSys) {
		errs = append(errs, fmt.Sprintf(".circleci/continue/main.yml setup-features.parameters.system_features.default = %q, expected %q (sorted feature-flags.yaml type: sys features, space-separated)",
			strings.Join(ci.SystemFeaturesDefault, " "),
			strings.Join(registrySys, " ")))
	}

	// Compare matrix rows exactly; registry rows render in declaration order.
	expected := renderCIMatrix(r)
	if !stringSlicesEqual(expected, ci.FeaturesMatrix) {
		errs = append(errs, fmt.Sprintf(".circleci/continue/main.yml features_matrix does not match feature-flags.yaml combinations.matrix (rendered with registry declaration order, main = [])\n  expected: %v\n  actual:   %v",
			expected, ci.FeaturesMatrix))
	}

	// Validate each CI row so errors can point at the bad row directly.
	knownFeature := map[string]Feature{}
	for _, f := range r.Features {
		knownFeature[f.Name] = f
	}
	for i, raw := range ci.FeaturesMatrix {
		row := parseCIMatrixRow(raw)
		if len(row) == 0 {
			continue
		}
		inRow := map[string]bool{}
		for _, name := range row {
			if _, ok := knownFeature[name]; !ok {
				errs = append(errs, fmt.Sprintf(".circleci/continue/main.yml features_matrix[%d] %q references unknown feature %q", i, raw, name))
			}
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
				if f, ok := knownFeature[p]; ok && f.Lifecycle == lifecycleHardcodedOn {
					continue
				}
				errs = append(errs, fmt.Sprintf(".circleci/continue/main.yml features_matrix[%d] %q enables %s but is missing required prerequisite %s", i, raw, key, p))
			}
		}
		for j, excl := range r.Combinations.Excludes {
			if len(excl) == 0 {
				continue
			}
			all := true
			for _, name := range excl {
				if !inRow[name] {
					all = false
					break
				}
			}
			if all {
				errs = append(errs, fmt.Sprintf(".circleci/continue/main.yml features_matrix[%d] %q violates excludes[%d]: %v", i, raw, j, excl))
			}
		}
	}

	sort.Strings(errs)
	return errs
}

func validateChecksConfigControlPlane(cfg ChecksConfig) []string {
	const prefix = "checks.yaml control-plane: "
	var errs []string
	if !cfg.FeatureFlagsFound {
		errs = append(errs, prefix+"feature-flags check is missing")
		return errs
	}
	if cfg.FeatureFlagsCount != 1 {
		errs = append(errs, fmt.Sprintf("%sfeature-flags check appears %d times, expected 1",
			prefix, cfg.FeatureFlagsCount))
	}
	if cfg.FeatureFlagsCommand != featureFlagsCheckCommand {
		errs = append(errs, fmt.Sprintf("%sfeature-flags check command is %q, expected %q",
			prefix, cfg.FeatureFlagsCommand, featureFlagsCheckCommand))
	}
	if !checksYamlFeaturePhases[cfg.FeatureFlagsPhase] {
		errs = append(errs, fmt.Sprintf("%sfeature-flags check phase is %q, must be one of setup, pre-build",
			prefix, cfg.FeatureFlagsPhase))
	}
	return errs
}

func validateCIControlPlane(cp CIControlPlane) []string {
	const prefix = "CircleCI control-plane: "
	var errs []string

	// contracts-bedrock-checks-fast gate path and required-CI wiring.
	if cp.ChecksFastCommand != checksFastCommand {
		errs = append(errs, fmt.Sprintf("%s%s command is %q, expected %q",
			prefix, jobChecksFast, cp.ChecksFastCommand, checksFastCommand))
	}
	if cp.ChecksFastWorkingDir != contractsBedrockWorkingDir {
		errs = append(errs, fmt.Sprintf("%s%s working_directory is %q, expected %q",
			prefix, jobChecksFast, cp.ChecksFastWorkingDir, contractsBedrockWorkingDir))
	}
	if !cp.HasCheckFastFeatureTests {
		errs = append(errs, fmt.Sprintf("%sworkflow %s does not invoke %s with name %s",
			prefix, workflowContractsFeatureTests, jobChecksFast, checkFastFeatureTestsInstanceName))
	}
	if !cp.RequiredCIReqsCheckFast {
		errs = append(errs, fmt.Sprintf("%s%s does not terminal-require %s",
			prefix, jobRequiredContractsCI, checkFastFeatureTestsInstanceName))
	}

	// setup-features command definition and call sites.
	if !cp.SetupFeaturesCommandExists {
		errs = append(errs, fmt.Sprintf("%scommands.%s is missing", prefix, setupFeaturesCommandName))
	} else {
		if cp.SetupFeaturesCommandCount != 1 {
			errs = append(errs, fmt.Sprintf("%scommands.%s appears %d times, expected 1",
				prefix, setupFeaturesCommandName, cp.SetupFeaturesCommandCount))
		}
		if !cp.SetupFeaturesHasFeaturesParam {
			errs = append(errs, fmt.Sprintf("%scommands.%s is missing parameter %q",
				prefix, setupFeaturesCommandName, setupFeaturesFeaturesParam))
		}
		if !cp.SetupFeaturesHasSystemFeaturesParam {
			errs = append(errs, fmt.Sprintf("%scommands.%s is missing parameter %q",
				prefix, setupFeaturesCommandName, setupFeaturesSystemFeatures))
		}
	}
	setupFeaturesSeenJobs := map[string]bool{}
	for _, caller := range cp.SetupFeaturesCallers {
		loc := ""
		if caller.Line > 0 {
			loc = fmt.Sprintf(" at line %d", caller.Line)
		}
		if !setupFeaturesAllowedJobs[caller.Job] {
			errs = append(errs, fmt.Sprintf("%s%s caller in job %s%s is not allowed",
				prefix, setupFeaturesCommandName, caller.Job, loc))
			continue
		}
		setupFeaturesSeenJobs[caller.Job] = true
		if caller.SystemFeaturesOverride || caller.SystemFeatures != "" {
			errs = append(errs, fmt.Sprintf("%s%s caller in job %s%s passes %s override",
				prefix, setupFeaturesCommandName, caller.Job, loc, setupFeaturesSystemFeatures))
		}
		if caller.Features != featuresParameterTemplate {
			errs = append(errs, fmt.Sprintf("%s%s caller in job %s%s passes %s=%q, expected %q",
				prefix, setupFeaturesCommandName, caller.Job, loc,
				setupFeaturesFeaturesParam, caller.Features, featuresParameterTemplate))
		}
	}
	for _, job := range setupFeaturesRequiredJobs {
		if !setupFeaturesSeenJobs[job] {
			errs = append(errs, fmt.Sprintf("%sjob %s does not call %s",
				prefix, job, setupFeaturesCommandName))
		}
	}

	// features_matrix anchor and matrix consumers.
	if cp.FeaturesMatrixAnchorCount != 1 {
		errs = append(errs, fmt.Sprintf("%s&%s anchor count is %d, expected 1",
			prefix, featuresMatrixAnchor, cp.FeaturesMatrixAnchorCount))
	}
	expected := map[string]bool{}
	for _, name := range contractsFeatureTestsMatrixConsumers {
		expected[name] = true
	}
	seen := map[string]bool{}
	for _, mc := range cp.MatrixConsumers {
		loc := ""
		if mc.Line > 0 {
			loc = fmt.Sprintf(" at line %d", mc.Line)
		}
		if !mc.UsesFeaturesAnchor {
			errs = append(errs, fmt.Sprintf("%smatrix consumer %s%s does not source matrix.parameters.features from *%s",
				prefix, mc.InstanceName, loc, featuresMatrixAnchor))
		}
		if !expected[mc.InstanceName] {
			errs = append(errs, fmt.Sprintf("%sunexpected matrix consumer %s%s in workflow %s",
				prefix, mc.InstanceName, loc, mc.Workflow))
		}
		seen[mc.InstanceName] = true
	}
	for _, name := range contractsFeatureTestsMatrixConsumers {
		if !seen[name] {
			errs = append(errs, fmt.Sprintf("%sexpected matrix consumer %s is missing from workflow %s",
				prefix, name, workflowContractsFeatureTests))
		}
	}
	for _, job := range contractsFeatureTestsEmptyDefaultJobs {
		v, ok := cp.FeatureDefaults[job]
		if !ok {
			errs = append(errs, fmt.Sprintf("%s%s parameters.%s.default is missing, expected %q",
				prefix, job, setupFeaturesFeaturesParam, ""))
			continue
		}
		if v != "" {
			errs = append(errs, fmt.Sprintf("%s%s parameters.%s.default = %q, expected %q",
				prefix, job, setupFeaturesFeaturesParam, v, ""))
		}
	}

	sort.Strings(errs)
	return errs
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isZeroHex(h string) bool {
	for _, r := range h {
		if r != '0' {
			return false
		}
	}
	return true
}

// exactlyOneBit reports whether h has exactly one bitmap bit set.
func exactlyOneBit(h string) bool {
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
