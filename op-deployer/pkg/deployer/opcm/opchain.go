package opcm

import (
	_ "embed"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// PermissionedGameStartingAnchorRoot is a root of bytes32(hex"dead") for the permissioned game at block 0,
// and no root for the permissionless game.
var PermissionedGameStartingAnchorRoot = []byte{
	0xde, 0xad, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

var deployOPChainSpec = ScriptSpec{
	ScriptFile:      "DeployOPChain.s.sol",
	ContractName:    "DeployOPChain",
	ForgeScriptPath: "scripts/deploy/DeployOPChain.s.sol:DeployOPChain",
}

type DeployOPChainInput = ScriptInput
type DeployOPChainOutput = ScriptOutput

type DeployOPChainScript = ScriptWithOutput

// NewDeployOPChainScript loads and validates the DeployOPChain script contract
func NewDeployOPChainScript(host *script.Host) (DeployOPChainScript, error) {
	return NewScriptWithOutputFromFile(host, deployOPChainSpec)
}

func NewDeployOPChainForgeCaller(client *forge.Client) forge.ScriptCaller[DeployOPChainInput, DeployOPChainOutput] {
	return NewScriptForgeCaller(client, deployOPChainSpec)
}

var readImplementationAddressesSpec = ScriptSpec{
	ScriptFile:      "ReadImplementationAddresses.s.sol",
	ContractName:    "ReadImplementationAddresses",
	ForgeScriptPath: "scripts/deploy/ReadImplementationAddresses.s.sol:ReadImplementationAddresses",
}

type ReadImplementationAddressesInput = ScriptInput
type ReadImplementationAddressesOutput = ScriptOutput

type ReadImplementationAddressesScript = ScriptWithOutput

// NewReadImplementationAddressesScript loads and validates the ReadImplementationAddresses script contract
func NewReadImplementationAddressesScript(host *script.Host) (ReadImplementationAddressesScript, error) {
	return NewScriptWithOutputFromFile(host, readImplementationAddressesSpec)
}

func NewReadImplementationAddressesForgeCaller(client *forge.Client) forge.ScriptCaller[ReadImplementationAddressesInput, ReadImplementationAddressesOutput] {
	return NewScriptForgeCaller(client, readImplementationAddressesSpec)
}

// DeployOPChainViaForge deploys OP Chain contracts using Forge
func DeployOPChainViaForge(env *ForgeEnv, input DeployOPChainInput) (DeployOPChainOutput, error) {
	var output DeployOPChainOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployOPChainForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy OP Chain with Forge: %w", err)
	}
	return output, nil
}

// ReadImplementationAddressesViaForge reads implementation addresses using Forge
func ReadImplementationAddressesViaForge(env *ForgeEnv, input ReadImplementationAddressesInput) (ReadImplementationAddressesOutput, error) {
	var output ReadImplementationAddressesOutput
	if err := env.validate(false); err != nil {
		return output, err
	}
	forgeCaller := NewReadImplementationAddressesForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOptsReadOnly()...)
	if err != nil {
		return output, fmt.Errorf("failed to run ReadImplementationAddresses with Forge: %w", err)
	}
	return output, nil
}
