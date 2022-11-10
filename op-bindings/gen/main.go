package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ethereum-optimism/optimism/op-bindings/ast"
	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
)

type flags struct {
	ArtifactsDir string
	Contracts    string
	OutDir       string
	Package      string
}

type data struct {
	Name          string
	StorageLayout string
	DeployedBin   string
	Package       string
}

func main() {
	var f flags
	flag.StringVar(&f.ArtifactsDir, "artifacts", "", "Comma-separated list of directories containing artifacts and build info")
	flag.StringVar(&f.OutDir, "out", "", "Output directory to put code in")
	flag.StringVar(&f.Contracts, "contracts", "", "Comma-separated list of contracts to generate code for")
	flag.StringVar(&f.Package, "package", "artifacts", "Go package name")
	flag.Parse()

	artifacts := strings.Split(f.ArtifactsDir, ",")
	contracts := strings.Split(f.Contracts, ",")

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

		d := data{
			Name:          name,
			StorageLayout: serStr,
			DeployedBin:   art.DeployedBytecode.String(),
			Package:       f.Package,
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

func init() {
	if err := json.Unmarshal([]byte({{.Name}}StorageLayoutJSON), {{.Name}}StorageLayout); err != nil {
		panic(err)
	}

	layouts["{{.Name}}"] = {{.Name}}StorageLayout
	deployedBytecodes["{{.Name}}"] = {{.Name}}DeployedBin
}
`
