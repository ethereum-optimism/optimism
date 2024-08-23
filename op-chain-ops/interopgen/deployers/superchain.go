package deployers

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

type DeploySuperchainInput struct {
	ProxyAdminOwner            common.Address // TODO(#11783): also used as interop-dependency-set owner
	ProtocolVersionsOwner      common.Address
	Guardian                   common.Address
	Paused                     bool
	RequiredProtocolVersion    params.ProtocolVersion
	RecommendedProtocolVersion params.ProtocolVersion
}

func (input *DeploySuperchainInput) InputSet() bool {
	return true
}

type DeploySuperchainOutput struct {
	SuperchainProxyAdmin  common.Address
	SuperchainConfigImpl  common.Address
	SuperchainConfigProxy common.Address
	ProtocolVersionsImpl  common.Address
	ProtocolVersionsProxy common.Address
}

func (output *DeploySuperchainOutput) CheckOutput() error {
	return nil
}

type DeploySuperchainScript struct {
	Run func(input, output common.Address) error
}

func DeploySuperchain(l1Host *script.Host, input *DeploySuperchainInput) (*DeploySuperchainOutput, error) {
	output := &DeploySuperchainOutput{}
	inputAddr := l1Host.NewScriptAddress()
	outputAddr := l1Host.NewScriptAddress()

	cleanupInput, err := script.WithPrecompileAtAddress[*DeploySuperchainInput](l1Host, inputAddr, input)
	if err != nil {
		return nil, fmt.Errorf("failed to insert DeploySuperchainInput precompile: %w", err)
	}
	defer cleanupInput()

	cleanupOutput, err := script.WithPrecompileAtAddress[*DeploySuperchainOutput](l1Host, outputAddr, output,
		script.WithFieldSetter[*DeploySuperchainOutput])
	if err != nil {
		return nil, fmt.Errorf("failed to insert DeploySuperchainOutput precompile: %w", err)
	}
	defer cleanupOutput()

	deployScript, cleanupDeploy, err := script.WithScript[DeploySuperchainScript](l1Host, "DeploySuperchain.s.sol", "DeploySuperchain")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeploySuperchain script: %w", err)
	}
	defer cleanupDeploy()

	if err := deployScript.Run(inputAddr, outputAddr); err != nil {
		return nil, fmt.Errorf("failed to run DeploySuperchain script: %w", err)
	}

	return output, nil
}
