package sysgo

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type ContractPaths struct {
	// must be absolute paths, without file:// prefix
	FoundryArtifacts string
	SourceMap        string
}

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

// findMonorepoRoot finds the path to the monorepo root.
//
// By default it walks up from the current working directory, which works for tests run from inside
// the monorepo tree. Out-of-tree consumers (e.g. an acceptance suite hosted in another repo that
// depends on this module by rev) can set DEVSTACK_MONOREPO_ROOT to an absolute monorepo checkout
// path, since the cwd-relative walk below can't reach a sibling checkout.
func findMonorepoRoot(testPath string) (string, error) {
	if root := os.Getenv("DEVSTACK_MONOREPO_ROOT"); root != "" {
		if !strings.HasSuffix(root, "/") {
			root += "/"
		}
		if _, err := os.Stat(root + testPath); err != nil {
			return "", fmt.Errorf("DEVSTACK_MONOREPO_ROOT=%q does not contain %v: %w", root, testPath, err)
		}
		return root, nil
	}

	path := "./"
	// Only search up 10 directories
	// Avoids infinite recursion if the root isn't found for some reason.
	// 10 is enough to cover deeply-nested test packages (e.g.
	// op-acceptance-tests/tests/supernode/interop/backfill/happy/ is 6
	// levels deep); 6 used to be enough before those were added.
	for i := 0; i < 10; i++ {
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
