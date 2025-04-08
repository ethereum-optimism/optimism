package presets

import (
	"errors"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/sysgo"
)

func contractPaths() (sysgo.ContractPaths, error) {
	contractsBedrockPath := "packages/contracts-bedrock"
	root, err := findMonorepoRoot(contractsBedrockPath)
	if err != nil {
		return sysgo.ContractPaths{}, err
	}
	return sysgo.ContractPaths{
		FoundryArtifacts: root + contractsBedrockPath + "/forge-artifacts",
		SourceMap:        root + contractsBedrockPath,
	}, nil
}

// findMonorepoRoot finds the relative path to the monorepo root
// Different tests might be nested in subdirectories of the op-e2e dir.
func findMonorepoRoot(testPath string) (string, error) {
	path := "./"
	// Only search up 5 directories
	// Avoids infinite recursion if the root isn't found for some reason
	for i := 0; i < 5; i++ {
		_, err := os.Stat(path + testPath)
		if errors.Is(err, os.ErrNotExist) {
			path = path + "../"
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to stat %v even though it existed: %w", path, err)
		}
		return path, nil
	}
	return "", fmt.Errorf("failed to find monorepo root using %v as the relative test path", testPath)
}
