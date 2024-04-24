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

	"github.com/ethereum-optimism/optimism/op-bindings/foundry"
)

type BindGenGeneratorLocal struct {
	BindGenGeneratorBase
	SourceMapsList     string
	ForgeArtifactsPath string
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
