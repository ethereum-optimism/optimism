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
	"regexp"
	"strings"
	"text/template"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/ast"
	"github.com/bobanetwork/v3-anchorage/boba-bindings/foundry"
)

type flags struct {
	ForgeArtifacts string
	Contracts      string
	SourceMaps     string
	OutDir         string
	Package        string
	MonorepoBase   string
	GeneratorDir   string
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
	flag.StringVar(&f.MonorepoBase, "monorepo-base", "", "Base of the monorepo")
	// flag.StringVar(&f.GeneratorDir, "generator-dir", "", "Directory containing the generator for the bindings")
	flag.Parse()

	if f.MonorepoBase == "" {
		log.Fatal("must provide -monorepo-base")
	}
	// if f.GeneratorDir == "" {
	// log.Fatal("must provide -generator-dir")
	// }

	log.Printf("Using monorepo base %s\n", f.MonorepoBase)

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
	dir, err := os.MkdirTemp("", "boba-bindings")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Using package %s\n", f.Package)

	defer os.RemoveAll(dir)
	log.Printf("created temp dir %s\n", dir)

	// If some contracts have the same name then the path to their
	// artifact depends on their full import path. Scan over all artifacts
	// and hold a mapping from the contract name to the contract path.
	// Walk walks the directory deterministically, so the later instance
	// of the contract with the same name will be used
	re := regexp.MustCompile(`\.\d+\.\d+\.\d+`)
	artifactPaths := make(map[string]string)
	if err := filepath.Walk(f.ForgeArtifacts,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(path, ".json") {
				base := filepath.Base(path)
				name := strings.TrimSuffix(base, ".json")

				// remove the compiler version from the name
				sanitized := re.ReplaceAllString(name, "")
				if _, ok := artifactPaths[sanitized]; !ok {
					artifactPaths[sanitized] = path
				}
			}
			return nil
		}); err != nil {
		log.Fatal(err)
	}

	for _, name := range contracts {
		log.Printf("generating code for %s\n", name)

		artifactPath := path.Join(f.ForgeArtifacts, name+".sol", name+".json")
		forgeArtifactData, err := os.ReadFile(artifactPath)
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("cannot find forge-artifact for %s at standard path %s, trying %s\n", name, artifactPath, artifactPaths[name])
			artifactPath = artifactPaths[name]
			forgeArtifactData, err = os.ReadFile(artifactPath)
			if errors.Is(err, os.ErrNotExist) {
				log.Fatalf("cannot find forge-artifact of %q\n", name)
			}
		}

		log.Printf("using forge-artifact %s\n", artifactPath)
		var artifact foundry.Artifact
		if err := json.Unmarshal(forgeArtifactData, &artifact); err != nil {
			log.Fatalf("failed to parse forge artifact of %q: %v\n", name, err)
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

		cmd := exec.Command("go", "run", "github.com/ledgerwatch/erigon/cmd/abigen", "--abi", abiFile, "--bin", bytecodeFile, "--pkg", f.Package, "--type", name, "--out", outFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Fatalf("error running abigen: %v\n", err)
		}

		if !strings.Contains(outFile, "boba.go") {
			bFile, err := os.ReadFile(outFile)
			if err != nil {
				log.Fatalf("error reading file: %v\n", err)
			}
			sFile := string(bFile)
			sFile = strings.Replace(sFile, `
// ERC20VotesCheckpoint is an auto generated low-level Go binding around an user-defined struct.
type ERC20VotesCheckpoint struct {
	FromBlock uint32
	Votes     *big.Int
}
`, "", -1)
			if err := os.WriteFile(outFile, []byte(sFile), 0o600); err != nil {
				log.Fatalf("error writing file: %v\n", err)
			}
		}

		storage := artifact.StorageLayout
		canonicalStorage := ast.CanonicalizeASTIDs(&storage, f.MonorepoBase)
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

	"github.com/bobanetwork/v3-anchorage/boba-bindings/solc"
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
