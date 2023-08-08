package utils

import (
	"os"
)

// MakeDirAll creates a directory and all parent directories if they do not exist.
func MakeDirAll(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// RemoveTrailingSlash removes the trailing slash from a path.
func RemoveTrailingSlash(path string) string {
	if path[len(path)-1] == '/' {
		return path[:len(path)-1]
	}
	return path
}

// DevnetDirectory returns the devnet directory within the monorepo.
func DevnetDirectory(monorepoDir string) string {
	return RemoveTrailingSlash(monorepoDir) + "/.devnet"
}

// ContractsDirectory returns the contracts directory within the monorepo.
func ContractsDirectory(monorepoDir string) string {
	return RemoveTrailingSlash(monorepoDir) + "/packages/contracts-bedrock"
}

// DeploymentDirectory returns the deployments directory within the monorepo.
func DeploymentDirectory(monorepoDir string) string {
	return ContractsDirectory(RemoveTrailingSlash(monorepoDir)) + "/deployments/devnetL1"
}

// RollupPath returns the rollup path within the monorepo.
func RollupPath(monorepoDir string) string {
	return DevnetDirectory(RemoveTrailingSlash(monorepoDir)) + "/rollup.json"
}

// OpNodeDirectory returns the op-node directory within the monorepo.
func OpNodeDirectory(monorepoDir string) string {
	return RemoveTrailingSlash(monorepoDir) + "/op-node"
}

// OpsDirectory returns the ops directory within the monorepo.
func OpsDirectory(monorepoDir string) string {
	return RemoveTrailingSlash(monorepoDir) + "/ops-bedrock"
}

// DeployConfigDirectory returns the deploy config directory within the monorepo.
func DeployConfigDirectory(monorepoDir string) string {
	return ContractsDirectory(RemoveTrailingSlash(monorepoDir)) + "/deploy-config"
}

// DevnetConfigPath returns the devnet config path within the monorepo.
func DevnetConfigPath(monorepoDir string) string {
	return DeployConfigDirectory(RemoveTrailingSlash(monorepoDir)) + "/devnetL1.json"
}

// DevnetConfigBackup returns the devnet config backup within the monorepo.
func DevnetConfigBackup(monorepoDir string) string {
	return DevnetDirectory(RemoveTrailingSlash(monorepoDir)) + "/devnetL1.json.bak"
}

// L1DeploymentsPath returns the L1 deployments path within the monorepo.
func L1DeploymentsPath(monorepoDir string) string {
	return DeploymentDirectory(RemoveTrailingSlash(monorepoDir)) + "/.deploy"
}

// AddressesJsonPath returns the addresses JSON path within the monorepo.
func AddressesJsonPath(monorepoDir string) string {
	return DevnetDirectory(RemoveTrailingSlash(monorepoDir)) + "/addresses.json"
}

// GenesisJsonPath returns the genesis JSON path within the monorepo.
func GenesisJsonPath(monorepoDir string) string {
	return DevnetDirectory(RemoveTrailingSlash(monorepoDir)) + "/genesis-l1.json"
}

// GenesisL2JsonPath returns the L2 genesis JSON path within the monorepo.
func GenesisL2JsonPath(monorepoDir string) string {
	return DevnetDirectory(RemoveTrailingSlash(monorepoDir)) + "/genesis-l2.json"
}

// AllocsJsonPath returns the allocs JSON path within the monorepo.
func AllocsJsonPath(monorepoDir string) string {
	return DevnetDirectory(RemoveTrailingSlash(monorepoDir)) + "/allocs-l1.json"
}
