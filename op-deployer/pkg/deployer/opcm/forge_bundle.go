package opcm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// ForgeScript names a script by its artifact.
type ForgeScript struct {
	File     string
	Contract string
}

var (
	DeploySuperchainForgeScript      = ForgeScript{"DeploySuperchain.s.sol", "DeploySuperchain"}
	DeployImplementationsForgeScript = ForgeScript{"DeployImplementations.s.sol", "DeployImplementations"}

	// L1ForgeScripts are the scripts that a deployment runs on L1.
	L1ForgeScripts = []ForgeScript{
		DeploySuperchainForgeScript,
		DeployImplementationsForgeScript,
		{"DeployOPChain.s.sol", "DeployOPChain"},
		{"ReadImplementationAddresses.s.sol", "ReadImplementationAddresses"},
		{"ReadSuperchainDeployment.s.sol", "ReadSuperchainDeployment"},
		{"DeployAltDA.s.sol", "DeployAltDA"},
		{"DeployDisputeGame.s.sol", "DeployDisputeGame"},
		{"DeployMIPS.s.sol", "DeployMIPS"},
		{"DeployAlphabetVM.s.sol", "DeployAlphabetVM"},
		{"SetDisputeGameImpl.s.sol", "SetDisputeGameImpl"},
	}

	// L2ForgeScripts are the scripts that build the L2 genesis.
	L2ForgeScripts = []ForgeScript{{"L2Genesis.s.sol", "L2Genesis"}}
)

// NewForgeClient returns a forge client for a contracts bundle. It checks that the scripts have the
// runWithBytes entrypoints, which an older contracts release lacks. forge is downloaded to cacheDir
// (or the default cache directory) when the pinned version is not on PATH.
func NewForgeClient(artifactsFS foundry.StatDirFs, cacheDir string, scripts []ForgeScript) (*forge.Client, error) {
	if err := CheckForgeScripts(artifactsFS, scripts); err != nil {
		return nil, err
	}
	var binOpts []forge.StandardBinOpt
	if cacheDir != "" {
		binOpts = append(binOpts, forge.WithCachePather(func() (string, error) { return cacheDir, nil }))
	}
	client, err := forge.NewStandardClient(fmt.Sprintf("%v", artifactsFS), binOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create forge client: %w", err)
	}
	if _, err := os.Stat(filepath.Join(client.Wd, "foundry.toml")); err != nil {
		return nil, fmt.Errorf("the contracts bundle is not a forge project: %w", err)
	}
	// The scripts read the artifacts.
	client.FsPermissions = []forge.FsPermission{
		{Access: "read", Path: "./forge-artifacts/"},
	}
	return client, nil
}

// CheckForgeScripts checks that each script has a runWithBytes entrypoint.
func CheckForgeScripts(artifactsFS foundry.StatDirFs, scripts []ForgeScript) error {
	afs := &foundry.ArtifactsFS{FS: artifactsFS}
	for _, script := range scripts {
		artifact, err := afs.ReadArtifact(script.File, script.Contract)
		if err != nil {
			return fmt.Errorf("the contracts bundle has no %s script: %w", script.Contract, err)
		}
		if _, ok := artifact.ABI.Methods["runWithBytes"]; !ok {
			return fmt.Errorf("the %s script of the contracts bundle has no runWithBytes entrypoint; use the contracts release of this op-deployer release", script.Contract)
		}
	}
	return nil
}
