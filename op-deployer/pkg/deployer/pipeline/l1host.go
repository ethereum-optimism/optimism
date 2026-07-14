package pipeline

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

// This file centralizes the L1-host operations that are not driven through env.Scripts, so the
// individual pipeline stages call one method. Every operation runs on the out-of-process Rust
// op-script-engine (L1Engine), whether it backs the non-forked DeploymentTargetGenesis L1 deploy or
// a forked Live/Calldata/Noop target running in fork mode.

// l1Origin resolves the tx.origin for engine-backed L1 script runs: the deployer.
func (env *Env) l1Origin() func() common.Address {
	d := env.Deployer
	return func() common.Address { return d }
}

// ReadImplementationAddresses runs the ReadImplementationAddresses script on the L1 engine.
func (env *Env) ReadImplementationAddresses(in opcm.ReadImplementationAddressesInput) (opcm.ReadImplementationAddressesOutput, error) {
	scr, err := readImplementationAddressesScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	if err != nil {
		return opcm.ReadImplementationAddressesOutput{}, fmt.Errorf("failed to load ReadImplementationAddresses script: %w", err)
	}
	return scr.Run(in)
}

// ReadSuperchainDeployment runs the ReadSuperchainDeployment script on the L1 engine.
func (env *Env) ReadSuperchainDeployment(in opcm.ReadSuperchainDeploymentInput) (opcm.ReadSuperchainDeploymentOutput, error) {
	scr, err := readSuperchainDeploymentScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	if err != nil {
		return opcm.ReadSuperchainDeploymentOutput{}, fmt.Errorf("failed to load ReadSuperchainDeployment script: %w", err)
	}
	return scr.Run(in)
}

// DeployAlphabetVM runs the DeployAlphabetVM script on the L1 engine.
func (env *Env) DeployAlphabetVM(in opcm.DeployAlphabetVMInput) (opcm.DeployAlphabetVMOutput, error) {
	scr, err := deployAlphabetVMScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	if err != nil {
		return opcm.DeployAlphabetVMOutput{}, fmt.Errorf("failed to load DeployAlphabetVM script: %w", err)
	}
	return scr.Run(in)
}

// DeployMIPS runs the DeployMIPS script on the L1 engine.
func (env *Env) DeployMIPS(in opcm.DeployMIPSInput) (opcm.DeployMIPSOutput, error) {
	scr, err := deployMIPSScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	if err != nil {
		return opcm.DeployMIPSOutput{}, fmt.Errorf("failed to load DeployMIPS script: %w", err)
	}
	return scr.Run(in)
}

// DeployAltDA runs the DeployAltDA script on the L1 engine.
func (env *Env) DeployAltDA(in opcm.DeployAltDAInput) (opcm.DeployAltDAOutput, error) {
	scr, err := deployAltDAScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	if err != nil {
		return opcm.DeployAltDAOutput{}, fmt.Errorf("failed to load DeployAltDA script: %w", err)
	}
	return scr.Run(in)
}

// SetDisputeGameImpl runs the SetDisputeGameImpl OPCM void script on the L1 engine.
func (env *Env) SetDisputeGameImpl(in opcm.SetDisputeGameImplInput) error {
	return rustengine.RunScriptVoid[opcm.SetDisputeGameImplInput](
		env.L1Engine, in, "SetDisputeGameImpl.s.sol", "SetDisputeGameImpl", env.Deployer)
}

// InsertPreinstallsL1 runs SetPreinstalls.setPreinstalls() on the L1 engine.
func (env *Env) InsertPreinstallsL1() error {
	artifact, err := env.L1Artifacts.ReadArtifact("SetPreinstalls.s.sol", "SetPreinstalls")
	if err != nil {
		return fmt.Errorf("failed to load SetPreinstalls script: %w", err)
	}
	calldata, err := artifact.ABI.Pack("setPreinstalls")
	if err != nil {
		return fmt.Errorf("failed to pack setPreinstalls() call: %w", err)
	}
	if _, err := env.L1Engine.RunScript("SetPreinstalls.s.sol", "SetPreinstalls", calldata, env.Deployer); err != nil {
		return fmt.Errorf("failed to set preinstalls: %w", err)
	}
	return nil
}

// WipeL1 clears an account's code/nonce/balance on the L1 engine.
func (env *Env) WipeL1(addr common.Address) error {
	return env.L1Engine.Wipe(addr)
}

// SetBalanceL1 sets an account's balance on the L1 engine.
func (env *Env) SetBalanceL1(addr common.Address, bal *uint256.Int) error {
	return env.L1Engine.SetBalance(addr, bal)
}

// StateDumpL1 dumps the committed L1 state into forge-allocs form.
func (env *Env) StateDumpL1() (*foundry.ForgeAllocs, error) {
	return env.L1Engine.StateDump()
}
