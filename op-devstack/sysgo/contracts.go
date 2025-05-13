package sysgo

import (
	"errors"
	"fmt"
	"os"
)

func contractPaths() (ContractPaths, error) {
	contractsBedrockPath := "packages/contracts-bedrock"
	root, err := findMonorepoRoot(contractsBedrockPath)
	if err != nil {
		return ContractPaths{}, err
	}
	return ContractPaths{
		FoundryArtifacts: root + contractsBedrockPath + "/forge-artifacts",
		SourceMap:        root + contractsBedrockPath,
	}, nil
}

func ensureDir(dirPath string) error {
	stat, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("path is not a directory")
	}
	return nil
}

// findMonorepoRoot finds the relative path to the monorepo root
// Different tests might be nested in subdirectories of the op-e2e dir.
func findMonorepoRoot(testPath string) (string, error) {
	path := "./"
	// Only search up 6 directories
	// Avoids infinite recursion if the root isn't found for some reason
	for i := 0; i < 6; i++ {
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
