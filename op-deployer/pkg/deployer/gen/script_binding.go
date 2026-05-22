package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployerv2/schema"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const (
	artifactFlagName = "artifact"
	baseDirFlagName  = "base-dir"
	configFlagName   = "config"
	functionFlagName = "function"
	packageFlagName  = "package"
	outFlagName      = "out"
	filenameFlagName = "filename"
)

// Commands contains source-generation commands for op-deployer.
var Commands = []*cli.Command{
	{
		Name:    "bindings",
		Aliases: []string{"script-bindings"},
		Usage:   "generates Go bindings declared in a manifest",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     configFlagName,
				Usage:    "path to the script bindings manifest YAML",
				Required: true,
			},
			&cli.StringFlag{
				Name:  baseDirFlagName,
				Usage: "base directory used to resolve relative manifest paths",
				Value: ".",
			},
		},
		Action: ScriptBindingsCLI,
	},
	{
		Name:  "script-binding",
		Usage: "generates Go types from a Foundry script function ABI",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     artifactFlagName,
				Usage:    "path to the Foundry artifact JSON",
				Required: true,
			},
			&cli.StringFlag{
				Name:  functionFlagName,
				Usage: "script function to generate bindings for",
				Value: "run",
			},
			&cli.StringFlag{
				Name:     packageFlagName,
				Usage:    "Go package name for the generated file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     outFlagName,
				Usage:    "output directory for the generated file",
				Required: true,
			},
			&cli.StringFlag{
				Name:  filenameFlagName,
				Usage: "generated file name",
				Value: "script_binding.gen.go",
			},
		},
		Action: ScriptBindingCLI,
	},
}

// ScriptBindingsManifest declares all script bindings that should be generated.
type ScriptBindingsManifest struct {
	ArtifactBase   string                  `yaml:"artifact_base"`
	OutputBase     string                  `yaml:"output_base"`
	Manifest       SchemaManifestConfig    `yaml:"manifest"`
	ScriptBindings []ScriptBindingManifest `yaml:"script_bindings"`
}

// SchemaManifestConfig declares an optional generated schema manifest file.
type SchemaManifestConfig struct {
	Package  string `yaml:"package"`
	Out      string `yaml:"out"`
	Filename string `yaml:"filename"`
}

// ScriptBindingManifest declares one generated script binding.
type ScriptBindingManifest struct {
	Name     string `yaml:"name"`
	Artifact string `yaml:"artifact"`
	Function string `yaml:"function"`
	Package  string `yaml:"package"`
	Out      string `yaml:"out"`
	Filename string `yaml:"filename"`
}

// ScriptBindingsConfig configures manifest-driven script binding generation.
type ScriptBindingsConfig struct {
	ConfigPath string
	BaseDir    string
}

// ScriptBindingConfig configures script-binding generation.
type ScriptBindingConfig struct {
	ArtifactPath string
	FunctionName string
	PackageName  string
	OutputDir    string
	Filename     string
}

// ScriptBindingsCLI generates all script bindings declared by a manifest.
func ScriptBindingsCLI(ctx *cli.Context) error {
	outPaths, err := GenerateScriptBindings(ScriptBindingsConfig{
		ConfigPath: ctx.String(configFlagName),
		BaseDir:    ctx.String(baseDirFlagName),
	})
	if err != nil {
		return err
	}
	for _, outPath := range outPaths {
		if _, err := fmt.Fprintf(ctx.App.Writer, "generated %s\n", outPath); err != nil {
			return err
		}
	}
	return nil
}

// ScriptBindingCLI generates Go source from a Foundry script artifact.
func ScriptBindingCLI(ctx *cli.Context) error {
	cfg := ScriptBindingConfig{
		ArtifactPath: ctx.String(artifactFlagName),
		FunctionName: ctx.String(functionFlagName),
		PackageName:  ctx.String(packageFlagName),
		OutputDir:    ctx.String(outFlagName),
		Filename:     ctx.String(filenameFlagName),
	}
	outPath, err := GenerateScriptBinding(cfg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(ctx.App.Writer, "generated %s\n", outPath)
	return err
}

// GenerateScriptBindings writes all script bindings declared by a manifest.
func GenerateScriptBindings(cfg ScriptBindingsConfig) ([]string, error) {
	if cfg.ConfigPath == "" {
		return nil, fmt.Errorf("%s is required", configFlagName)
	}
	baseDir := cfg.BaseDir
	if baseDir == "" {
		baseDir = "."
	}
	baseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve base directory: %w", err)
	}

	manifestData, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read script bindings manifest: %w", err)
	}
	var manifest ScriptBindingsManifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse script bindings manifest: %w", err)
	}
	if len(manifest.ScriptBindings) == 0 {
		return nil, fmt.Errorf("script bindings manifest must declare at least one binding")
	}

	artifactBase := resolvePath(baseDir, manifest.ArtifactBase)
	outputBase := resolvePath(baseDir, manifest.OutputBase)
	outPaths := make([]string, 0, len(manifest.ScriptBindings))
	for i, binding := range manifest.ScriptBindings {
		genCfg, err := binding.generatorConfig(i, artifactBase, outputBase)
		if err != nil {
			return nil, err
		}
		outPath, err := GenerateScriptBinding(genCfg)
		if err != nil {
			return nil, fmt.Errorf("generate %s: %w", binding.displayName(i), err)
		}
		outPaths = append(outPaths, outPath)
	}
	if manifest.Manifest.Package != "" {
		outPath, err := GenerateSchemaManifest(manifest, baseDir, artifactBase)
		if err != nil {
			return nil, err
		}
		outPaths = append(outPaths, outPath)
	}
	return outPaths, nil
}

// GenerateScriptBinding writes Go source for a script function with a single tuple input and tuple output.
func GenerateScriptBinding(cfg ScriptBindingConfig) (string, error) {
	if cfg.ArtifactPath == "" {
		return "", fmt.Errorf("%s is required", artifactFlagName)
	}
	if cfg.FunctionName == "" {
		return "", fmt.Errorf("%s is required", functionFlagName)
	}
	if cfg.PackageName == "" {
		return "", fmt.Errorf("%s is required", packageFlagName)
	}
	if cfg.OutputDir == "" {
		return "", fmt.Errorf("%s is required", outFlagName)
	}
	if cfg.Filename == "" {
		return "", fmt.Errorf("%s is required", filenameFlagName)
	}

	source, err := BuildScriptBinding(cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	outPath := filepath.Join(cfg.OutputDir, cfg.Filename)
	if err := os.WriteFile(outPath, source, 0o644); err != nil {
		return "", fmt.Errorf("write generated file: %w", err)
	}
	return outPath, nil
}

// GenerateSchemaManifest writes a generated schema manifest for all script bindings.
func GenerateSchemaManifest(manifest ScriptBindingsManifest, baseDir string, artifactBase string) (string, error) {
	if manifest.Manifest.Package == "" {
		return "", fmt.Errorf("manifest package is required")
	}
	outDir := manifest.Manifest.Out
	if outDir == "" {
		outDir = "."
	}
	filename := manifest.Manifest.Filename
	if filename == "" {
		filename = "manifest.gen.go"
	}

	source, err := BuildSchemaManifest(manifest, artifactBase)
	if err != nil {
		return "", err
	}
	outPath := resolvePath(resolvePath(baseDir, outDir), filename)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", fmt.Errorf("create manifest output directory: %w", err)
	}
	if err := os.WriteFile(outPath, source, 0o644); err != nil {
		return "", fmt.Errorf("write schema manifest: %w", err)
	}
	return outPath, nil
}

// BuildSchemaManifest returns formatted Go source for a generated schema manifest.
func BuildSchemaManifest(manifest ScriptBindingsManifest, artifactBase string) ([]byte, error) {
	manifestValue := schema.Manifest{
		Structs: map[string]schema.StructDef{},
	}
	for i, binding := range manifest.ScriptBindings {
		genCfg, err := binding.generatorConfig(i, artifactBase, "")
		if err != nil {
			return nil, err
		}
		structs, err := BuildScriptBindingSchema(genCfg)
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", binding.displayName(i), err)
		}
		for name, def := range structs {
			if _, ok := manifestValue.Structs[name]; ok {
				return nil, fmt.Errorf("schema struct %s generated more than once", name)
			}
			manifestValue.Structs[name] = def
		}
	}

	hashedManifest, err := manifestValue.WithHash()
	if err != nil {
		return nil, err
	}
	return renderSchemaManifest(manifest.Manifest.Package, hashedManifest)
}

// BuildScriptBindingSchema returns schema structs for one script binding.
func BuildScriptBindingSchema(cfg ScriptBindingConfig) (map[string]schema.StructDef, error) {
	artifact, err := foundry.ReadArtifact(cfg.ArtifactPath)
	if err != nil {
		return nil, err
	}
	method, ok := artifact.ABI.Methods[cfg.FunctionName]
	if !ok {
		return nil, fmt.Errorf("function %q not found in artifact %s", cfg.FunctionName, cfg.ArtifactPath)
	}
	if len(method.Inputs) != 1 {
		return nil, fmt.Errorf("function %q must have exactly one input, got %d", cfg.FunctionName, len(method.Inputs))
	}
	if method.Inputs[0].Type.T != abi.TupleTy {
		return nil, fmt.Errorf("function %q input must be a tuple, got %s", cfg.FunctionName, method.Inputs[0].Type.String())
	}
	if len(method.Outputs) > 1 {
		return nil, fmt.Errorf("function %q must have zero or one output, got %d", cfg.FunctionName, len(method.Outputs))
	}
	if len(method.Outputs) == 1 && method.Outputs[0].Type.T != abi.TupleTy {
		return nil, fmt.Errorf("function %q output must be a tuple, got %s", cfg.FunctionName, method.Outputs[0].Type.String())
	}

	_, contractName := compilationTarget(artifact, cfg.ArtifactPath)
	if contractName == "" {
		contractName = strings.TrimSuffix(filepath.Base(cfg.ArtifactPath), filepath.Ext(cfg.ArtifactPath))
	}

	schemaGen := newSchemaGenerator()
	if err := schemaGen.addStruct(contractName+"Input", method.Inputs[0].Type); err != nil {
		return nil, fmt.Errorf("input: %w", err)
	}
	if len(method.Outputs) == 1 {
		if err := schemaGen.addStruct(contractName+"Output", method.Outputs[0].Type); err != nil {
			return nil, fmt.Errorf("output: %w", err)
		}
	}
	return schemaGen.structs, nil
}

func (b ScriptBindingManifest) generatorConfig(index int, artifactBase string, outputBase string) (ScriptBindingConfig, error) {
	if b.Artifact == "" {
		return ScriptBindingConfig{}, fmt.Errorf("script binding %s must declare artifact", b.displayName(index))
	}

	name := b.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(b.Artifact), filepath.Ext(b.Artifact))
	}
	functionName := b.Function
	if functionName == "" {
		functionName = "run"
	}
	packageName := b.Package
	if packageName == "" {
		packageName = packageNameFromBindingName(name)
	}
	outDir := b.Out
	if outDir == "" {
		outDir = packageName
	}
	filename := b.Filename
	if filename == "" {
		filename = snakeCase(name) + ".gen.go"
	}

	return ScriptBindingConfig{
		ArtifactPath: resolvePath(artifactBase, b.Artifact),
		FunctionName: functionName,
		PackageName:  packageName,
		OutputDir:    resolvePath(outputBase, outDir),
		Filename:     filename,
	}, nil
}

func (b ScriptBindingManifest) displayName(index int) string {
	if b.Name != "" {
		return b.Name
	}
	return fmt.Sprintf("script_bindings[%d]", index)
}

// BuildScriptBinding returns formatted Go source for a script function with a single tuple input and tuple output.
func BuildScriptBinding(cfg ScriptBindingConfig) ([]byte, error) {
	artifact, err := foundry.ReadArtifact(cfg.ArtifactPath)
	if err != nil {
		return nil, err
	}
	method, ok := artifact.ABI.Methods[cfg.FunctionName]
	if !ok {
		return nil, fmt.Errorf("function %q not found in artifact %s", cfg.FunctionName, cfg.ArtifactPath)
	}
	if len(method.Inputs) != 1 {
		return nil, fmt.Errorf("function %q must have exactly one input, got %d", cfg.FunctionName, len(method.Inputs))
	}
	if method.Inputs[0].Type.T != abi.TupleTy {
		return nil, fmt.Errorf("function %q input must be a tuple, got %s", cfg.FunctionName, method.Inputs[0].Type.String())
	}
	if len(method.Outputs) > 1 {
		return nil, fmt.Errorf("function %q must have zero or one output, got %d", cfg.FunctionName, len(method.Outputs))
	}
	if len(method.Outputs) == 1 && method.Outputs[0].Type.T != abi.TupleTy {
		return nil, fmt.Errorf("function %q output must be a tuple, got %s", cfg.FunctionName, method.Outputs[0].Type.String())
	}

	sourcePath, contractName := compilationTarget(artifact, cfg.ArtifactPath)
	if contractName == "" {
		contractName = strings.TrimSuffix(filepath.Base(cfg.ArtifactPath), filepath.Ext(cfg.ArtifactPath))
	}

	typeGen := newTypeGenerator()
	inputFields, err := typeGen.buildFields(method.Inputs[0].Type, "Input")
	if err != nil {
		return nil, fmt.Errorf("input: %w", err)
	}
	var outputFields []generatedField
	if len(method.Outputs) == 1 {
		outputFields, err = typeGen.buildFields(method.Outputs[0].Type, "Output")
		if err != nil {
			return nil, fmt.Errorf("output: %w", err)
		}
	}

	var buf bytes.Buffer
	fmt.Fprintln(&buf, "// Code generated by op-deployer gen script-binding; DO NOT EDIT.")
	fmt.Fprintln(&buf)
	fmt.Fprintf(&buf, "package %s\n\n", cfg.PackageName)

	if len(typeGen.imports) > 0 {
		importPaths := make([]string, 0, len(typeGen.imports))
		for importPath := range typeGen.imports {
			importPaths = append(importPaths, importPath)
		}
		sort.Strings(importPaths)

		fmt.Fprintln(&buf, "import (")
		for _, importPath := range importPaths {
			fmt.Fprintf(&buf, "\t%q\n", importPath)
		}
		fmt.Fprintln(&buf, ")")
		fmt.Fprintln(&buf)
	}

	fmt.Fprintln(&buf, "const (")
	if sourcePath != "" {
		fmt.Fprintf(&buf, "\tSourcePath = %q\n", sourcePath)
		fmt.Fprintf(&buf, "\tScriptFile = %q\n", path.Base(sourcePath))
	}
	fmt.Fprintf(&buf, "\tContractName = %q\n", contractName)
	if sourcePath != "" && contractName != "" {
		fmt.Fprintf(&buf, "\tForgeScriptPath = %q\n", sourcePath+":"+contractName)
	}
	fmt.Fprintf(&buf, "\tFunctionName = %q\n", cfg.FunctionName)
	if contractName != "" {
		fmt.Fprintf(&buf, "\tInputTypeName = %q\n", contractName+"Input")
		if len(method.Outputs) == 1 {
			fmt.Fprintf(&buf, "\tOutputTypeName = %q\n", contractName+"Output")
		}
	}
	fmt.Fprintln(&buf, "\tRunWithBytesSignature = \"runWithBytes(bytes)\"")
	fmt.Fprintln(&buf, ")")
	fmt.Fprintln(&buf)

	writeStruct(&buf, "Input", inputFields)
	if len(method.Outputs) == 1 {
		fmt.Fprintln(&buf)
		writeStruct(&buf, "Output", outputFields)
	}
	for _, nestedStruct := range typeGen.nestedStructs {
		fmt.Fprintln(&buf)
		writeStruct(&buf, nestedStruct.Name, nestedStruct.Fields)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w", err)
	}
	return formatted, nil
}

type generatedField struct {
	Name string
	Type string
	Tag  string
}

type generatedStruct struct {
	Name   string
	Fields []generatedField
}

type typeGenerator struct {
	imports       map[string]struct{}
	usedNames     map[string]int
	nestedStructs []generatedStruct
}

func newTypeGenerator() *typeGenerator {
	return &typeGenerator{
		imports: map[string]struct{}{},
		usedNames: map[string]int{
			"Input":  1,
			"Output": 1,
		},
	}
}

func (g *typeGenerator) buildFields(tuple abi.Type, parentName string) ([]generatedField, error) {
	fields := make([]generatedField, 0, len(tuple.TupleElems))
	for i, elem := range tuple.TupleElems {
		rawName := tuple.TupleRawNames[i]
		fieldName := goFieldName(rawName, i)
		fieldType, err := g.goType(rawName, *elem, parentName)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", rawName, err)
		}
		fields = append(fields, generatedField{
			Name: fieldName,
			Type: fieldType,
			Tag:  fmt.Sprintf("`abi:%q json:%q toml:%q`", rawName, rawName, rawName),
		})
	}
	return fields, nil
}

func (g *typeGenerator) goType(rawName string, typ abi.Type, parentName string) (string, error) {
	switch typ.T {
	case abi.AddressTy:
		g.imports["github.com/ethereum/go-ethereum/common"] = struct{}{}
		return "common.Address", nil
	case abi.BoolTy:
		return "bool", nil
	case abi.StringTy:
		return "string", nil
	case abi.BytesTy:
		return "[]byte", nil
	case abi.FixedBytesTy:
		if typ.Size == 32 {
			g.imports["github.com/ethereum/go-ethereum/common"] = struct{}{}
			return "common.Hash", nil
		}
		return fmt.Sprintf("[%d]byte", typ.Size), nil
	case abi.HashTy:
		g.imports["github.com/ethereum/go-ethereum/common"] = struct{}{}
		return "common.Hash", nil
	case abi.UintTy:
		return g.intType("uint", typ.Size), nil
	case abi.IntTy:
		return g.intType("int", typ.Size), nil
	case abi.SliceTy:
		elem, err := g.goType(singular(rawName), *typ.Elem, parentName)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	case abi.ArrayTy:
		elem, err := g.goType(singular(rawName), *typ.Elem, parentName)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[%d]%s", typ.Size, elem), nil
	case abi.TupleTy:
		structName := g.uniqueName(parentName + goFieldName(rawName, len(g.nestedStructs)))
		fields, err := g.buildFields(typ, structName)
		if err != nil {
			return "", err
		}
		g.nestedStructs = append(g.nestedStructs, generatedStruct{
			Name:   structName,
			Fields: fields,
		})
		return structName, nil
	default:
		return "", fmt.Errorf("unsupported ABI type %s", typ.String())
	}
}

func (g *typeGenerator) intType(prefix string, bits int) string {
	switch bits {
	case 8, 16, 32, 64:
		return fmt.Sprintf("%s%d", prefix, bits)
	default:
		g.imports["math/big"] = struct{}{}
		return "*big.Int"
	}
}

func (g *typeGenerator) uniqueName(base string) string {
	if g.usedNames[base] == 0 {
		g.usedNames[base] = 1
		return base
	}
	for {
		g.usedNames[base]++
		candidate := fmt.Sprintf("%s%d", base, g.usedNames[base])
		if g.usedNames[candidate] == 0 {
			g.usedNames[candidate] = 1
			return candidate
		}
	}
}

func writeStruct(buf *bytes.Buffer, name string, fields []generatedField) {
	fmt.Fprintf(buf, "type %s struct {\n", name)
	for _, field := range fields {
		fmt.Fprintf(buf, "\t%s %s %s\n", field.Name, field.Type, field.Tag)
	}
	fmt.Fprintln(buf, "}")
}

func compilationTarget(artifact *foundry.Artifact, artifactPath string) (string, string) {
	targets := artifact.Metadata.Settings.CompilationTarget
	if len(targets) == 0 {
		return "", ""
	}
	sourcePaths := make([]string, 0, len(targets))
	for sourcePath := range targets {
		sourcePaths = append(sourcePaths, sourcePath)
	}
	sort.Strings(sourcePaths)

	sourcePath := sourcePaths[0]
	contractName := targets[sourcePath]
	if contractName == "" {
		contractName = strings.TrimSuffix(filepath.Base(artifactPath), filepath.Ext(artifactPath))
	}
	return sourcePath, contractName
}

func resolvePath(base string, target string) string {
	if target == "" {
		return base
	}
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(base, target)
}

func packageNameFromBindingName(name string) string {
	parts := wordBoundary.Split(name, -1)
	var out strings.Builder
	for _, part := range parts {
		out.WriteString(strings.ToLower(part))
	}
	if out.Len() == 0 {
		return "generated"
	}
	return out.String()
}

func snakeCase(name string) string {
	var out strings.Builder
	var prevLowerOrDigit bool
	for _, r := range name {
		switch {
		case unicode.IsUpper(r):
			if prevLowerOrDigit {
				out.WriteRune('_')
			}
			out.WriteRune(unicode.ToLower(r))
			prevLowerOrDigit = false
		case unicode.IsLower(r) || unicode.IsDigit(r):
			out.WriteRune(r)
			prevLowerOrDigit = true
		default:
			if out.Len() > 0 && !strings.HasSuffix(out.String(), "_") {
				out.WriteRune('_')
			}
			prevLowerOrDigit = false
		}
	}
	return strings.Trim(out.String(), "_")
}

var wordBoundary = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func goFieldName(name string, index int) string {
	if name == "" {
		return fmt.Sprintf("Field%d", index)
	}
	parts := wordBoundary.Split(name, -1)
	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		out.WriteString(string(runes))
	}
	fieldName := out.String()
	if fieldName == "" {
		return fmt.Sprintf("Field%d", index)
	}
	if !unicode.IsLetter([]rune(fieldName)[0]) {
		return "Field" + fieldName
	}
	return fieldName
}

type schemaGenerator struct {
	structs   map[string]schema.StructDef
	usedNames map[string]int
}

func newSchemaGenerator() *schemaGenerator {
	return &schemaGenerator{
		structs:   map[string]schema.StructDef{},
		usedNames: map[string]int{},
	}
}

func (g *schemaGenerator) addStruct(name string, tuple abi.Type) error {
	if tuple.T != abi.TupleTy {
		return fmt.Errorf("schema struct %s must be generated from tuple, got %s", name, tuple.String())
	}
	if _, ok := g.structs[name]; ok {
		return fmt.Errorf("schema struct %s generated more than once", name)
	}
	if g.usedNames[name] == 0 {
		g.usedNames[name] = 1
	}
	fields := make([]schema.Field, 0, len(tuple.TupleElems))
	for i, elem := range tuple.TupleElems {
		rawName := tuple.TupleRawNames[i]
		field, err := g.schemaField(name, rawName, *elem)
		if err != nil {
			return fmt.Errorf("field %s: %w", rawName, err)
		}
		fields = append(fields, field)
	}
	g.structs[name] = schema.StructDef{Fields: fields}
	return nil
}

func (g *schemaGenerator) schemaField(parentName string, rawName string, typ abi.Type) (schema.Field, error) {
	field := schema.Field{
		Name:     rawName,
		Required: true,
	}
	fieldType, nestedStruct, err := g.schemaType(parentName, rawName, typ)
	if err != nil {
		return schema.Field{}, err
	}
	field.Type = fieldType
	field.Struct = nestedStruct
	return field, nil
}

func (g *schemaGenerator) schemaType(parentName string, rawName string, typ abi.Type) (string, string, error) {
	switch typ.T {
	case abi.TupleTy:
		structName := g.uniqueName(parentName + goFieldName(rawName, len(g.structs)))
		if err := g.addStruct(structName, typ); err != nil {
			return "", "", err
		}
		return "tuple", structName, nil
	case abi.SliceTy:
		elemType, nestedStruct, err := g.schemaType(parentName, singular(rawName), *typ.Elem)
		if err != nil {
			return "", "", err
		}
		return elemType + "[]", nestedStruct, nil
	case abi.ArrayTy:
		elemType, nestedStruct, err := g.schemaType(parentName, singular(rawName), *typ.Elem)
		if err != nil {
			return "", "", err
		}
		return fmt.Sprintf("%s[%d]", elemType, typ.Size), nestedStruct, nil
	default:
		return typ.String(), "", nil
	}
}

func (g *schemaGenerator) uniqueName(base string) string {
	if g.usedNames[base] == 0 {
		g.usedNames[base] = 1
		return base
	}
	for {
		g.usedNames[base]++
		candidate := fmt.Sprintf("%s%d", base, g.usedNames[base])
		if g.usedNames[candidate] == 0 {
			g.usedNames[candidate] = 1
			return candidate
		}
	}
}

func renderSchemaManifest(packageName string, manifest schema.Manifest) ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "// Code generated by op-deployer gen bindings; DO NOT EDIT.")
	fmt.Fprintln(&buf)
	fmt.Fprintf(&buf, "package %s\n\n", packageName)
	fmt.Fprintln(&buf, "import \"github.com/ethereum-optimism/optimism/op-deployerv2/schema\"")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "var Manifest = schema.Manifest{")
	fmt.Fprintf(&buf, "\tSchemaHash: %q,\n", manifest.SchemaHash)
	fmt.Fprintln(&buf, "\tStructs: map[string]schema.StructDef{")

	structNames := make([]string, 0, len(manifest.Structs))
	for name := range manifest.Structs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	for _, name := range structNames {
		def := manifest.Structs[name]
		fmt.Fprintf(&buf, "\t\t%q: {\n", name)
		fmt.Fprintln(&buf, "\t\t\tFields: []schema.Field{")
		for _, field := range def.Fields {
			fmt.Fprintf(&buf, "\t\t\t\t{Name: %q, Type: %q, Required: %t", field.Name, field.Type, field.Required)
			if field.Struct != "" {
				fmt.Fprintf(&buf, ", Struct: %q", field.Struct)
			}
			fmt.Fprintln(&buf, "},")
		}
		fmt.Fprintln(&buf, "\t\t\t},")
		fmt.Fprintln(&buf, "\t\t},")
	}

	fmt.Fprintln(&buf, "\t},")
	fmt.Fprintln(&buf, "}")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated schema manifest: %w", err)
	}
	return formatted, nil
}

func singular(name string) string {
	switch {
	case strings.HasSuffix(name, "ies") && len(name) > 3:
		return strings.TrimSuffix(name, "ies") + "y"
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") && len(name) > 1:
		return strings.TrimSuffix(name, "s")
	default:
		return name
	}
}
