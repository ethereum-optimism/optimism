package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// Scripts contains all the deployment scripts for ease of passing them around
type Scripts struct {
	DeployAlphabetVM      DeployAlphabetVMScript
	DeployAltDA           DeployAltDA2Script
	DeployAsterisc        DeployAsteriscScript
	DeployDisputeGame     DeployDisputeGameScript
	DeployImplementations DeployImplementations2Script
	DeployMIPS            DeployMIPSScript
	DeployPreimageOracle  DeployPreimageOracleScript
	DeployProxy           DeployProxyScript
	DeploySuperchain      DeploySuperchainScript
}

// NewScripts collects all the deployment scripts, raising exceptions if any of them
// are not found or if the Go types don't match the ABI
func NewScripts(host *script.Host) (*Scripts, error) {
	deployImplementations, err := NewDeployImplementationsScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployImplementations script: %w", err)
	}

	deploySuperchain, err := NewDeploySuperchainScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeploySuperchain script: %w", err)
	}

	deployAlphabetVM, err := NewDeployAlphabetVMScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployAlphabetVM script: %w", err)
	}

	deployAltDA, err := NewDeployAltDAScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployAltDA script: %w", err)
	}

	deployAsterisc, err := NewDeployAsteriscScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployAsterisc script: %w", err)
	}

	deployDisputeGame, err := NewDeployDisputeGameScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployDisputeGame script: %w", err)
	}

	deployMIPSScript, err := NewDeployMIPSScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployMIPSScript script: %w", err)
	}

	deployPreimageOracle, err := NewDeployPreimageOracleScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployPreimageOracle script: %w", err)
	}

	deployProxy, err := NewDeployProxyScript(host)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployProxy script: %w", err)
	}

	return &Scripts{
		DeployAlphabetVM:      deployAlphabetVM,
		DeployAltDA:           deployAltDA,
		DeployAsterisc:        deployAsterisc,
		DeployDisputeGame:     deployDisputeGame,
		DeployMIPS:            deployMIPSScript,
		DeployPreimageOracle:  deployPreimageOracle,
		DeployProxy:           deployProxy,
		DeployImplementations: deployImplementations,
		DeploySuperchain:      deploySuperchain,
	}, nil
}
