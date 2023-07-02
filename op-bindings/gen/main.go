package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ethereum-optimism/optimism/op-bindings/ast"
	"github.com/ethereum-optimism/optimism/op-bindings/foundry"
)

type flags struct {
	ForgeArtifacts string
	Contracts      string
	SourceMaps     string
	OutDir         string
	Package        string
}

type data struct {
	Name              string
	StorageLayout     string
	DeployedBin       string
	Package           string
	DeployedSourceMap string
}

func main() {
	var f flags
	flag.StringVar(&f.ForgeArtifacts, "forge-artifacts", "", "Forge artifacts directory, to load sourcemaps from, if available")
	flag.StringVar(&f.OutDir, "out", "", "Output directory to put code in")
	flag.StringVar(&f.Contracts, "contracts", "artifacts.json", "Path to file containing list of contracts to generate bindings for")
	flag.StringVar(&f.SourceMaps, "source-maps", "", "Comma-separated list of contracts to generate source-maps for")
	flag.StringVar(&f.Package, "package", "artifacts", "Go package name")
	flag.Parse()

	contractData, err := os.ReadFile(f.Contracts)
	if err != nil {
		log.Fatal("error reading contract list: %w\n", err)
	}
	contracts := []string{}
	if err := json.Unmarshal(contractData, &contracts); err != nil {
		log.Fatal("error parsing contract list: %w\n", err)
	}

	sourceMaps := strings.Split(f.SourceMaps, ",")
	sourceMapsSet := make(map[string]struct{})
	for _, k := range sourceMaps {
		sourceMapsSet[k] = struct{}{}
	}

	if len(contracts) == 0 {
		log.Fatalf("must define a list of contracts")
	}

	t := template.Must(template.New("artifact").Parse(tmpl))

	// Make a temp dir to hold all the inputs for abigen
	dir, err := os.MkdirTemp("", "op-bindings")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Using package %s\n", f.Package)

	defer os.RemoveAll(dir)
	log.Printf("created temp dir %s\n", dir)

	for _, name := range contracts {
		log.Printf("generating code for %s\n", name)

		forgeArtifactData, err := os.ReadFile(path.Join(f.ForgeArtifacts, name+".sol", name+".json"))
		if errors.Is(err, os.ErrNotExist) {
			log.Fatalf("cannot find forge-artifact of %q\n", name)
		}

		var artifact foundry.Artifact
		if err := json.Unmarshal(forgeArtifactData, &artifact); err != nil {
			log.Fatalf("failed to parse forge artifact of %q: %v\n", name, err)
		}
		if err != nil {
			log.Fatalf("error reading storage layout %s: %v\n", name, err)
		}

		rawAbi := artifact.Abi
		if err != nil {
			log.Fatalf("error marshaling abi: %v\n", err)
		}
		abiFile := path.Join(dir, name+".abi")
		if err := os.WriteFile(abiFile, rawAbi, 0o600); err != nil {
			log.Fatalf("error writing file: %v\n", err)
		}
		rawBytecode := artifact.Bytecode.Object.String()
		if err != nil {
			log.Fatalf("error marshaling bytecode: %v\n", err)
		}
		bytecodeFile := path.Join(dir, name+".bin")
		if err := os.WriteFile(bytecodeFile, []byte(rawBytecode), 0o600); err != nil {
			log.Fatalf("error writing file: %v\n", err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("error getting cwd: %v\n", err)
		}

		lowerName := strings.ToLower(name)
		outFile := path.Join(cwd, f.Package, lowerName+".go")

		cmd := exec.Command("abigen", "--abi", abiFile, "--bin", bytecodeFile, "--pkg", f.Package, "--type", name, "--out", outFile)
		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err != nil {
			log.Fatalf("error running abigen: %v\n", err)
		}

		storage := artifact.StorageLayout
		canonicalStorage := ast.CanonicalizeASTIDs(&storage)
		ser, err := json.Marshal(canonicalStorage)
		if err != nil {
			log.Fatalf("error marshaling storage: %v\n", err)
		}
		serStr := strings.Replace(string(ser), "\"", "\\\"", -1)

		deployedSourceMap := ""
		if _, ok := sourceMapsSet[name]; ok {
			deployedSourceMap = artifact.DeployedBytecode.SourceMap
		}

		d := data{
			Name:              name,
			StorageLayout:     serStr,
			DeployedBin:       artifact.DeployedBytecode.Object.String(),
			Package:           f.Package,
			DeployedSourceMap: deployedSourceMap,
		}

		fname := filepath.Join(f.OutDir, strings.ToLower(name)+"_more.go")
		outfile, err := os.OpenFile(
			fname,
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			0o600,
		)
		if err != nil {
			log.Fatalf("error opening %s: %v\n", fname, err)
		}

		if err := t.Execute(outfile, d); err != nil {
			log.Fatalf("error writing template %s: %v", outfile.Name(), err)
		}
		outfile.Close()
		log.Printf("wrote file %s\n", outfile.Name())
	}
}

var tmpl = `// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package {{.Package}}

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const {{.Name}}StorageLayoutJSON = "{{.StorageLayout}}"

var {{.Name}}StorageLayout = new(solc.StorageLayout)

var {{.Name}}DeployedBin = "{{.DeployedBin}}"
{{if .DeployedSourceMap}}
var {{.Name}}DeployedSourceMap = "{{.DeployedSourceMap}}"
{{end}}
func init() {
	if err := json.Unmarshal([]byte({{.Name}}StorageLayoutJSON), {{.Name}}StorageLayout); err != nil {
		panic(err)
	}

	layouts["{{.Name}}"] = {{.Name}}StorageLayout
	deployedBytecodes["{{.Name}}"] = {{.Name}}DeployedBin
}
`
