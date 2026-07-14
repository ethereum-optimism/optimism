package scriptbackend

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

// This builds the OPCM deploy scripts backed by the Rust engine, the engine-backed counterpart of
// opcm.NewScripts. It is the same construction pipeline.NewEngineScripts performs for the multi-stage
// apply pipeline; it is duplicated here (rather than imported) because pipeline imports upgrade, which
// imports this package — importing pipeline back would form a cycle.

// engineScriptWithOutput builds a typed deploy script bound to the Rust engine, mirroring
// opcm.NewDeployScriptWithOutputFromFile. origin is the run() tx.origin.
func engineScriptWithOutput[I any, O any](
	eng *rustengine.Engine,
	fa *foundry.ArtifactsFS,
	origin func() common.Address,
	file, contract string,
) (script.DeployScriptWithOutput[I, O], error) {
	artifact, err := fa.ReadArtifact(file, contract)
	if err != nil {
		return nil, fmt.Errorf("failed to load script %s from %s: %w", contract, file, err)
	}
	fs := rustengine.NewForgeScript(eng, artifact, file, contract, origin)
	return script.NewDeployScriptWithOutput[I, O](fs, "run")
}

func newEngineScripts(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (*opcm.Scripts, error) {
	deployImplementations, err := engineScriptWithOutput[opcm.DeployImplementationsInput, opcm.DeployImplementationsOutput](eng, fa, origin, "DeployImplementations.s.sol", "DeployImplementations")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployImplementations script: %w", err)
	}
	deploySuperchain, err := engineScriptWithOutput[opcm.DeploySuperchainInput, opcm.DeploySuperchainOutput](eng, fa, origin, "DeploySuperchain.s.sol", "DeploySuperchain")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeploySuperchain script: %w", err)
	}
	deployAlphabetVM, err := engineScriptWithOutput[opcm.DeployAlphabetVMInput, opcm.DeployAlphabetVMOutput](eng, fa, origin, "DeployAlphabetVM.s.sol", "DeployAlphabetVM")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployAlphabetVM script: %w", err)
	}
	deployAltDA, err := engineScriptWithOutput[opcm.DeployAltDAInput, opcm.DeployAltDAOutput](eng, fa, origin, "DeployAltDA.s.sol", "DeployAltDA")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployAltDA script: %w", err)
	}
	deployDisputeGame, err := engineScriptWithOutput[opcm.DeployDisputeGameInput, opcm.DeployDisputeGameOutput](eng, fa, origin, "DeployDisputeGame.s.sol", "DeployDisputeGame")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployDisputeGame script: %w", err)
	}
	deployMIPS, err := engineScriptWithOutput[opcm.DeployMIPSInput, opcm.DeployMIPSOutput](eng, fa, origin, "DeployMIPS.s.sol", "DeployMIPS")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployMIPS script: %w", err)
	}
	deployOPChain, err := engineScriptWithOutput[opcm.DeployOPChainInput, opcm.DeployOPChainOutput](eng, fa, origin, "DeployOPChain.s.sol", "DeployOPChain")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployOPChain script: %w", err)
	}

	return &opcm.Scripts{
		DeployAlphabetVM:      deployAlphabetVM,
		DeployAltDA:           deployAltDA,
		DeployDisputeGame:     deployDisputeGame,
		DeployMIPS:            deployMIPS,
		DeployImplementations: deployImplementations,
		DeploySuperchain:      deploySuperchain,
		DeployOPChain:         deployOPChain,
	}, nil
}
