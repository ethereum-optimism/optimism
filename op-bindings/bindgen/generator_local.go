package bindgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/ethereum-optimism/optimism/op-bindings/ast"
	"github.com/ethereum-optimism/optimism/op-bindings/foundry"
)

type BindGenGeneratorLocal struct {
	BindGenGeneratorBase
	SourceMapsList     string
	ForgeArtifactsPath string
}

type localContractMetadata struct {
	Name                   string
	StorageLayout          string
	DeployedBin            string
	Package                string
	DeployedSourceMap      string
	HasImmutableReferences bool
}

func (generator *BindGenGeneratorLocal) GenerateBindings() error {
	contracts, err := readContractList(generator.Logger, generator.ContractsListPath)
	if err != nil {
		return fmt.Errorf("error reading contract list %s: %w", generator.ContractsListPath, err)
	}
	if len(contracts.Local) == 0 {
		return fmt.Errorf("no contracts parsed from given contract list: %s", generator.ContractsListPath)
	}

	return generator.processContracts(contracts.Local)
}

func (generator *BindGenGeneratorLocal) processContracts(contracts []string) error {
	tempArtifactsDir, err := mkTempArtifactsDir(generator.Logger)
	if err != nil {
		return err
	}
	defer func() {
		err := os.RemoveAll(tempArtifactsDir)
		if err != nil {
			generator.Logger.Error("Error removing temporary artifact directory", "path", tempArtifactsDir, "err", err.Error())
		} else {
			generator.Logger.Debug("Successfully removed temporary artifact directory")
		}
	}()

	sourceMapsList := strings.Split(generator.SourceMapsList, ",")
	sourceMapsSet := make(map[string]struct{})
	for _, k := range sourceMapsList {
		sourceMapsSet[k] = struct{}{}
	}

	contractArtifactPaths, err := generator.getContractArtifactPaths()
	if err != nil {
		return err
	}

	contractMetadataFileTemplate := template.Must(template.New("localContractMetadata").Parse(localContractMetadataTemplate))

	for _, contractName := range contracts {
		generator.Logger.Info("Generating bindings and metadata for local contract", "contract", contractName)

		forgeArtifact, err := generator.readForgeArtifact(contractName, contractArtifactPaths)
		if err != nil {
			return err
		}

		abiFilePath, bytecodeFilePath, err := writeContractArtifacts(generator.Logger, tempArtifactsDir, contractName, forgeArtifact.Abi, []byte(forgeArtifact.Bytecode.Object.String()))
		if err != nil {
			return err
		}

		err = genContractBindings(generator.Logger, generator.MonorepoBasePath, abiFilePath, bytecodeFilePath, generator.BindingsPackageName, contractName)
		if err != nil {
			return err
		}

		deployedSourceMap, canonicalStorageStr, err := generator.canonicalizeStorageLayout(forgeArtifact, sourceMapsSet, contractName)
		if err != nil {
			return err
		}

		re := regexp.MustCompile(`\s+`)
		immutableRefs, err := json.Marshal(re.ReplaceAllString(string(forgeArtifact.DeployedBytecode.ImmutableReferences), ""))
		if err != nil {
			return fmt.Errorf("error marshaling immutable references: %w", err)
		}

		hasImmutables := string(immutableRefs) != `""`

		contractMetaData := localContractMetadata{
			Name:                   contractName,
			StorageLayout:          canonicalStorageStr,
			DeployedBin:            forgeArtifact.DeployedBytecode.Object.String(),
			Package:                generator.BindingsPackageName,
			DeployedSourceMap:      deployedSourceMap,
			HasImmutableReferences: hasImmutables,
		}

		if err := generator.writeContractMetadata(contractMetaData, contractName, contractMetadataFileTemplate); err != nil {
			return err
		}
	}

	return nil
}

func (generator *BindGenGeneratorLocal) getContractArtifactPaths() (map[string]string, error) {
	// If some contracts have the same name then the path to their
	// artifact depends on their full import path. Scan over all artifacts
	// and hold a mapping from the contract name to the contract path.
	// Walk walks the directory deterministically, so the earliest instance
	// of the contract with the same name will be used
	artifactPaths := make(map[string]string)
	if err := filepath.Walk(generator.ForgeArtifactsPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(path, ".json") {
				base := filepath.Base(path)
				name := strings.TrimSuffix(base, ".json")

				// remove the compiler version from the name
				re := regexp.MustCompile(`\.\d+\.\d+\.\d+`)
				sanitized := re.ReplaceAllString(name, "")
				_, ok := artifactPaths[sanitized]
				if !ok {
					artifactPaths[sanitized] = path
				} else {
					generator.Logger.Warn("Multiple versions of forge artifacts exist, using lesser version", "contract", sanitized)
				}
			}
			return nil
		}); err != nil {
		return artifactPaths, err
	}

	return artifactPaths, nil
}

func (generator *BindGenGeneratorLocal) readForgeArtifact(contractName string, contractArtifactPaths map[string]string) (foundry.Artifact, error) {
	var forgeArtifact foundry.Artifact

	contractArtifactPath := path.Join(generator.ForgeArtifactsPath, contractName+".sol", contractName+".json")
	forgeArtifactRaw, err := os.ReadFile(contractArtifactPath)
	if errors.Is(err, os.ErrNotExist) {
		generator.Logger.Debug("Cannot find forge-artifact at standard path, trying provided path", "contract", contractName, "standardPath", contractArtifactPath, "providedPath", contractArtifactPaths[contractName])
		contractArtifactPath = contractArtifactPaths[contractName]
		forgeArtifactRaw, err = os.ReadFile(contractArtifactPath)
		if errors.Is(err, os.ErrNotExist) {
			return forgeArtifact, fmt.Errorf("cannot find forge-artifact of %q", contractName)
		}
	}

	generator.Logger.Debug("Using forge-artifact", "path", contractArtifactPath)
	if err := json.Unmarshal(forgeArtifactRaw, &forgeArtifact); err != nil {
		return forgeArtifact, fmt.Errorf("failed to parse forge artifact of %q: %w", contractName, err)
	}

	return forgeArtifact, nil
}

func (generator *BindGenGeneratorLocal) canonicalizeStorageLayout(forgeArtifact foundry.Artifact, sourceMapsSet map[string]struct{}, contractName string) (string, string, error) {
	artifactStorageStruct := forgeArtifact.StorageLayout
	canonicalStorageStruct := ast.CanonicalizeASTIDs(&artifactStorageStruct, generator.MonorepoBasePath)
	canonicalStorageJson, err := json.Marshal(canonicalStorageStruct)
	if err != nil {
		return "", "", fmt.Errorf("error marshaling canonical storage: %w", err)
	}
	canonicalStorageStr := strings.Replace(string(canonicalStorageJson), "\"", "\\\"", -1)

	deployedSourceMap := ""
	if _, ok := sourceMapsSet[contractName]; ok {
		deployedSourceMap = forgeArtifact.DeployedBytecode.SourceMap
	}

	return deployedSourceMap, canonicalStorageStr, nil
}

func (generator *BindGenGeneratorLocal) writeContractMetadata(contractMetaData localContractMetadata, contractName string, fileTemplate *template.Template) error {
	metadataFilePath := filepath.Join(generator.MetadataOut, strings.ToLower(contractName)+"_more.go")
	metadataFile, err := os.OpenFile(
		metadataFilePath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("error opening %s's metadata file at %s: %w", contractName, metadataFilePath, err)
	}
	defer metadataFile.Close()

	if err := fileTemplate.Execute(metadataFile, contractMetaData); err != nil {
		return fmt.Errorf("error writing %s's contract metadata at %s: %w", contractName, metadataFilePath, err)
	}

	generator.Logger.Debug("Successfully wrote contract metadata", "contract", contractName, "path", metadataFilePath)
	return nil
}

// associated with a local Ethereum contract. This template is used to produce
// Go code containing necessary constants and initialization logic for the contract's
// storage layout, deployed bytecode, and optionally its deployed source map.
//
// The template expects the following fields to be provided:
// - Package: The name of the Go package for the generated bindings.
// - Name: The name of the contract.
// - StorageLayout: Canonicalized storage layout of the contract as a JSON string.
// - DeployedBin: The deployed bytecode of the contract.
// - DeployedSourceMap (optional): The source map of the deployed contract.
var localContractMetadataTemplate = `// Code generated - DO NOT EDIT.
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
	immutableReferences["{{.Name}}"] = {{.HasImmutableReferences}}
}
`
