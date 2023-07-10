package devnet

import (
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-devnet/flags"
	"github.com/urfave/cli/v2"
)

type paths struct {
	monorepoDir          string
	devnetDir            string
	contractsBedrock     string
	deploymentDir        string
	deploymentConfigDir  string
	opNodeDir            string
	opsBedrockDir        string
	genesisL1Path        string
	genesisL2Path        string
	addressesJsonPath    string
	sdkAddressesJsonPath string
	rollupConfigPath     string
}

// NewPaths returns the paths to used directories.
func NewPaths(ctx *cli.Context) (paths, error) {
	monorepoDir := ctx.String(flags.MonorepoDir.Name)
	if monorepoDir == "" {
		monoDir, err := os.Getwd()
		if err != nil {
			return paths{}, err
		}
		monorepoDir = monoDir
	}
	devnetDir := filepath.Join(monorepoDir, ".devnet")
	contractsBedrockDir := filepath.Join(monorepoDir, "packages", "contracts-bedrock")
	deploymentDir := filepath.Join(devnetDir, "deployments", "devnetL1")
	opNodeDir := filepath.Join(monorepoDir, "op-node")
	opsBedrockDir := filepath.Join(monorepoDir, "ops-bedrock")

	return paths{
		monorepoDir:          monorepoDir,
		devnetDir:            devnetDir,
		contractsBedrock:     contractsBedrockDir,
		deploymentDir:        deploymentDir,
		deploymentConfigDir:  filepath.Join(contractsBedrockDir, "deploy-config"),
		opNodeDir:            opNodeDir,
		opsBedrockDir:        opsBedrockDir,
		genesisL1Path:        filepath.Join(devnetDir, "genesis-l1.json"),
		genesisL2Path:        filepath.Join(devnetDir, "genesis-l2.json"),
		addressesJsonPath:    filepath.Join(devnetDir, "addresses.json"),
		sdkAddressesJsonPath: filepath.Join(devnetDir, "sdk-addresses.json"),
		rollupConfigPath:     filepath.Join(devnetDir, "rollup-config.json"),
	}, nil
}
