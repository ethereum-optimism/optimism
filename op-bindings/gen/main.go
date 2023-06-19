package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ethereum-optimism/optimism/op-bindings/ast"
	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
)

type flags struct {
	ArtifactsDir   string
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

type forgeArtifact struct {
	DeployedBytecode struct {
		SourceMap string `json:"sourceMap"`
	} `json:"deployedBytecode"`
}

func main() {
	var f flags
	flag.StringVar(&f.ArtifactsDir, "artifacts", "", "Comma-separated list of directories containing artifacts and build info")
	flag.StringVar(&f.ForgeArtifacts, "forge-artifacts", "", "Forge artifacts directory, to load sourcemaps from, if available")
	flag.StringVar(&f.OutDir, "out", "", "Output directory to put code in")
	flag.StringVar(&f.Contracts, "contracts", "", "Comma-separated list of contracts to generate code for")
	flag.StringVar(&f.SourceMaps, "source-maps", "", "Comma-separated list of contracts to generate source-maps for")
	flag.StringVar(&f.Package, "package", "artifacts", "Go package name")
	flag.Parse()

	artifacts := strings.Split(f.ArtifactsDir, ",")
	contracts := strings.Split(f.Contracts, ",")
	sourceMaps := strings.Split(f.SourceMaps, ",")
	sourceMapsSet := make(map[string]struct{})
	for _, k := range sourceMaps {
		sourceMapsSet[k] = struct{}{}
	}

	if len(artifacts) == 0 {
		log.Fatalf("must define a list of artifacts")
	}

	if len(contracts) == 0 {
		log.Fatalf("must define a list of contracts")
	}

	hh, err := hardhat.New("dummy", artifacts, nil)
	if err != nil {
		log.Fatalln("error reading artifacts:", err)
	}

	t := template.Must(template.New("artifact").Parse(tmpl))

	for _, name := range contracts {
		art, err := hh.GetArtifact(name)
		if err != nil {
			log.Fatalf("error reading artifact %s: %v\n", name, err)
		}

		storage, err := hh.GetStorageLayout(name)
		if err != nil {
			log.Fatalf("error reading storage layout %s: %v\n", name, err)
		}
		canonicalStorage := ast.CanonicalizeASTIDs(storage)

		ser, err := json.Marshal(canonicalStorage)
		if err != nil {
			log.Fatalf("error marshaling storage: %v\n", err)
		}
		serStr := strings.Replace(string(ser), "\"", "\\\"", -1)

		deployedSourceMap := ""
		if _, ok := sourceMapsSet[name]; ok {
			// directory has .sol extension
			forgeArtifactData, err := os.ReadFile(path.Join(f.ForgeArtifacts, name+".sol", name+".json"))
			if errors.Is(err, os.ErrNotExist) {
				log.Printf("cannot find forge-artifact with source-map data of %q\n", name)
			}
			if err == nil {
				var artifact forgeArtifact
				if err := json.Unmarshal(forgeArtifactData, &artifact); err != nil {
					log.Fatalf("failed to parse forge artifact of %q: %v\n", name, err)
				}
				deployedSourceMap = artifact.DeployedBytecode.SourceMap
			}
		}

		d := data{
			Name:              name,
			StorageLayout:     serStr,
			DeployedBin:       art.DeployedBytecode.String(),
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
