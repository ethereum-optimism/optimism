package pipeline

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
)

// This file centralizes the non-forked L1-host operations that are not driven through env.Scripts,
// so the individual pipeline stages call one method and this file owns the single go-vs-engine
// branch per operation. When L1Engine is set (non-forked DeploymentTargetGenesis on the rust
// engine) the operation runs on the out-of-process op-script-engine; otherwise it runs on the
// in-process Go L1ScriptHost, byte-for-byte the prior behavior.

// usingL1Engine reports whether the non-forked L1 deploy stages should run on the Rust engine.
func (env *Env) usingL1Engine() bool {
	return env.L1Engine != nil
}

// l1Origin resolves the tx.origin for engine-backed L1 script runs. For the non-forked genesis
// path this is the deployer, matching env.DefaultScriptHost's scriptCtx.Origin = deployer.
func (env *Env) l1Origin() func() common.Address {
	d := env.Deployer
	return func() common.Address { return d }
}

// ReadImplementationAddresses runs the ReadImplementationAddresses script on the L1 host.
func (env *Env) ReadImplementationAddresses(in opcm.ReadImplementationAddressesInput) (opcm.ReadImplementationAddressesOutput, error) {
	var scr opcm.ReadImplementationAddressesScript
	var err error
	if env.usingL1Engine() {
		scr, err = readImplementationAddressesScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	} else {
		scr, err = opcm.NewReadImplementationAddressesScript(env.L1ScriptHost)
	}
	if err != nil {
		return opcm.ReadImplementationAddressesOutput{}, fmt.Errorf("failed to load ReadImplementationAddresses script: %w", err)
	}
	return scr.Run(in)
}

// ReadSuperchainDeployment runs the ReadSuperchainDeployment script on the L1 host.
func (env *Env) ReadSuperchainDeployment(in opcm.ReadSuperchainDeploymentInput) (opcm.ReadSuperchainDeploymentOutput, error) {
	var scr opcm.ReadSuperchainDeploymentScript
	var err error
	if env.usingL1Engine() {
		scr, err = readSuperchainDeploymentScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	} else {
		scr, err = opcm.NewReadSuperchainDeploymentScript(env.L1ScriptHost)
	}
	if err != nil {
		return opcm.ReadSuperchainDeploymentOutput{}, fmt.Errorf("failed to load ReadSuperchainDeployment script: %w", err)
	}
	return scr.Run(in)
}

// DeployAlphabetVM runs the DeployAlphabetVM script on the L1 host.
func (env *Env) DeployAlphabetVM(in opcm.DeployAlphabetVMInput) (opcm.DeployAlphabetVMOutput, error) {
	var scr opcm.DeployAlphabetVMScript
	var err error
	if env.usingL1Engine() {
		scr, err = deployAlphabetVMScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	} else {
		scr, err = opcm.NewDeployAlphabetVMScript(env.L1ScriptHost)
	}
	if err != nil {
		return opcm.DeployAlphabetVMOutput{}, fmt.Errorf("failed to load DeployAlphabetVM script: %w", err)
	}
	return scr.Run(in)
}

// DeployMIPS runs the DeployMIPS script on the L1 host.
func (env *Env) DeployMIPS(in opcm.DeployMIPSInput) (opcm.DeployMIPSOutput, error) {
	var scr opcm.DeployMIPSScript
	var err error
	if env.usingL1Engine() {
		scr, err = deployMIPSScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	} else {
		scr, err = opcm.NewDeployMIPSScript(env.L1ScriptHost)
	}
	if err != nil {
		return opcm.DeployMIPSOutput{}, fmt.Errorf("failed to load DeployMIPS script: %w", err)
	}
	return scr.Run(in)
}

// DeployAltDA runs the DeployAltDA script on the L1 host.
func (env *Env) DeployAltDA(in opcm.DeployAltDAInput) (opcm.DeployAltDAOutput, error) {
	var scr opcm.DeployAltDAScript
	var err error
	if env.usingL1Engine() {
		scr, err = deployAltDAScriptEngine(env.L1Engine, env.L1Artifacts, env.l1Origin())
	} else {
		scr, err = opcm.NewDeployAltDAScript(env.L1ScriptHost)
	}
	if err != nil {
		return opcm.DeployAltDAOutput{}, fmt.Errorf("failed to load DeployAltDA script: %w", err)
	}
	return scr.Run(in)
}

// SetDisputeGameImpl runs the SetDisputeGameImpl OPCM void script on the L1 host.
func (env *Env) SetDisputeGameImpl(in opcm.SetDisputeGameImplInput) error {
	if env.usingL1Engine() {
		return rustengine.RunScriptVoid[opcm.SetDisputeGameImplInput](
			env.L1Engine, in, "SetDisputeGameImpl.s.sol", "SetDisputeGameImpl", env.Deployer)
	}
	return opcm.SetDisputeGameImpl(env.L1ScriptHost, in)
}

// InsertPreinstallsL1 runs SetPreinstalls.setPreinstalls() on the L1 host.
func (env *Env) InsertPreinstallsL1() error {
	if env.usingL1Engine() {
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
	return opcm.InsertPreinstalls(env.L1ScriptHost)
}

// WipeL1 clears an account's code/nonce/balance on the L1 host.
func (env *Env) WipeL1(addr common.Address) error {
	if env.usingL1Engine() {
		return env.L1Engine.Wipe(addr)
	}
	env.L1ScriptHost.Wipe(addr)
	return nil
}

// SetBalanceL1 sets an account's balance on the L1 host.
func (env *Env) SetBalanceL1(addr common.Address, bal *uint256.Int) error {
	if env.usingL1Engine() {
		return env.L1Engine.SetBalance(addr, bal)
	}
	env.L1ScriptHost.SetBalance(addr, bal)
	return nil
}

// StateDumpL1 dumps the committed L1 state into forge-allocs form.
func (env *Env) StateDumpL1() (*foundry.ForgeAllocs, error) {
	if env.usingL1Engine() {
		return env.L1Engine.StateDump()
	}
	return env.L1ScriptHost.StateDump()
}
