package pipeline

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

// This file builds the OPCM deploy scripts backed by the out-of-process Rust op-script-engine,
// the engine-backed counterparts of opcm.NewScripts / opcm.NewXScript. It lives in the pipeline
// package (which already depends on both opcm and rustengine) so the generic op-chain-ops
// rustengine client stays free of any op-deployer/opcm dependency.

// engineScriptWithOutput builds a typed deploy script bound to the Rust engine, mirroring
// opcm.NewDeployScriptWithOutputFromFile. The generic wrappers (ABI packing and I/O type
// validation) are identical to the Go host path; only the ForgeScript backend differs (engine
// runScript RPC vs in-process host). origin is the run() tx.origin.
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

func deploySuperchainScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeploySuperchainScript, error) {
	return engineScriptWithOutput[opcm.DeploySuperchainInput, opcm.DeploySuperchainOutput](eng, fa, origin, "DeploySuperchain.s.sol", "DeploySuperchain")
}

func deployImplementationsScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployImplementationsScript, error) {
	return engineScriptWithOutput[opcm.DeployImplementationsInput, opcm.DeployImplementationsOutput](eng, fa, origin, "DeployImplementations.s.sol", "DeployImplementations")
}

func deployOPChainScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployOPChainScript, error) {
	return engineScriptWithOutput[opcm.DeployOPChainInput, opcm.DeployOPChainOutput](eng, fa, origin, "DeployOPChain.s.sol", "DeployOPChain")
}

func deployDisputeGameScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployDisputeGameScript, error) {
	return engineScriptWithOutput[opcm.DeployDisputeGameInput, opcm.DeployDisputeGameOutput](eng, fa, origin, "DeployDisputeGame.s.sol", "DeployDisputeGame")
}

func deployMIPSScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployMIPSScript, error) {
	return engineScriptWithOutput[opcm.DeployMIPSInput, opcm.DeployMIPSOutput](eng, fa, origin, "DeployMIPS.s.sol", "DeployMIPS")
}

func deployAlphabetVMScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployAlphabetVMScript, error) {
	return engineScriptWithOutput[opcm.DeployAlphabetVMInput, opcm.DeployAlphabetVMOutput](eng, fa, origin, "DeployAlphabetVM.s.sol", "DeployAlphabetVM")
}

func deployAltDAScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployAltDAScript, error) {
	return engineScriptWithOutput[opcm.DeployAltDAInput, opcm.DeployAltDAOutput](eng, fa, origin, "DeployAltDA.s.sol", "DeployAltDA")
}

func readImplementationAddressesScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.ReadImplementationAddressesScript, error) {
	return engineScriptWithOutput[opcm.ReadImplementationAddressesInput, opcm.ReadImplementationAddressesOutput](eng, fa, origin, "ReadImplementationAddresses.s.sol", "ReadImplementationAddresses")
}

func readSuperchainDeploymentScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.ReadSuperchainDeploymentScript, error) {
	return engineScriptWithOutput[opcm.ReadSuperchainDeploymentInput, opcm.ReadSuperchainDeploymentOutput](eng, fa, origin, "ReadSuperchainDeployment.s.sol", "ReadSuperchainDeployment")
}

// NewEngineScripts builds the opcm.Scripts bundle backed by the Rust engine, the engine-backed
// counterpart of opcm.NewScripts. fa provides the ABIs (Go-side packing/validation); origin
// resolves the tx.origin per script run.
func NewEngineScripts(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (*opcm.Scripts, error) {
	deployImplementations, err := deployImplementationsScriptEngine(eng, fa, origin)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployImplementations script: %w", err)
	}
	deploySuperchain, err := deploySuperchainScriptEngine(eng, fa, origin)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeploySuperchain script: %w", err)
	}
	deployAlphabetVM, err := deployAlphabetVMScriptEngine(eng, fa, origin)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployAlphabetVM script: %w", err)
	}
	deployAltDA, err := deployAltDAScriptEngine(eng, fa, origin)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployAltDA script: %w", err)
	}
	deployDisputeGame, err := deployDisputeGameScriptEngine(eng, fa, origin)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployDisputeGame script: %w", err)
	}
	deployMIPS, err := deployMIPSScriptEngine(eng, fa, origin)
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployMIPS script: %w", err)
	}
	deployOPChain, err := deployOPChainScriptEngine(eng, fa, origin)
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
