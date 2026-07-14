package pipeline

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/scriptbackend"
)

// These build individual OPCM deploy/read scripts backed by the out-of-process Rust
// op-script-engine, for the per-stage L1 routing in l1host.go. They wrap the canonical
// scriptbackend.EngineScriptWithOutput (shared with scriptbackend.NewEngineScripts) so the
// construction stays in one place; the generic wrappers (ABI packing and I/O type validation) are
// identical to the Go host path — only the ForgeScript backend differs. origin is the run()
// tx.origin.

func deployMIPSScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployMIPSScript, error) {
	return scriptbackend.EngineScriptWithOutput[opcm.DeployMIPSInput, opcm.DeployMIPSOutput](eng, fa, origin, "DeployMIPS.s.sol", "DeployMIPS")
}

func deployAlphabetVMScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployAlphabetVMScript, error) {
	return scriptbackend.EngineScriptWithOutput[opcm.DeployAlphabetVMInput, opcm.DeployAlphabetVMOutput](eng, fa, origin, "DeployAlphabetVM.s.sol", "DeployAlphabetVM")
}

func deployAltDAScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.DeployAltDAScript, error) {
	return scriptbackend.EngineScriptWithOutput[opcm.DeployAltDAInput, opcm.DeployAltDAOutput](eng, fa, origin, "DeployAltDA.s.sol", "DeployAltDA")
}

func readImplementationAddressesScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.ReadImplementationAddressesScript, error) {
	return scriptbackend.EngineScriptWithOutput[opcm.ReadImplementationAddressesInput, opcm.ReadImplementationAddressesOutput](eng, fa, origin, "ReadImplementationAddresses.s.sol", "ReadImplementationAddresses")
}

func readSuperchainDeploymentScriptEngine(eng *rustengine.Engine, fa *foundry.ArtifactsFS, origin func() common.Address) (opcm.ReadSuperchainDeploymentScript, error) {
	return scriptbackend.EngineScriptWithOutput[opcm.ReadSuperchainDeploymentInput, opcm.ReadSuperchainDeploymentOutput](eng, fa, origin, "ReadSuperchainDeployment.s.sol", "ReadSuperchainDeployment")
}
