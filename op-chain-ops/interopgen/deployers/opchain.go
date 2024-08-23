package deployers

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

type DeployOPChainInput struct {
	OpChainProxyAdminOwner common.Address
	SystemConfigOwner      common.Address
	Batcher                common.Address
	UnsafeBlockSigner      common.Address
	Proposer               common.Address
	Challenger             common.Address

	BasefeeScalar     uint32
	BlobBaseFeeScalar uint32
	L2ChainId         *big.Int
	Opsm              common.Address
}

func (input *DeployOPChainInput) InputSet() bool {
	return true
}

type DeployOPChainOutput struct {
	OpChainProxyAdmin                 common.Address
	AddressManager                    common.Address
	L1ERC721BridgeProxy               common.Address
	SystemConfigProxy                 common.Address
	OptimismMintableERC20FactoryProxy common.Address
	L1StandardBridgeProxy             common.Address
	L1CrossDomainMessengerProxy       common.Address
	// Fault proof contracts below.
	OptimismPortalProxy                common.Address
	DisputeGameFactoryProxy            common.Address
	DisputeGameFactoryImpl             common.Address
	AnchorStateRegistryProxy           common.Address
	AnchorStateRegistryImpl            common.Address
	FaultDisputeGame                   common.Address
	PermissionedDisputeGame            common.Address
	DelayedWETHPermissionedGameProxy   common.Address
	DelayedWETHPermissionlessGameProxy common.Address
}

func (output *DeployOPChainOutput) CheckOutput() error {
	return nil
}

type DeployOPChainScript struct {
	Run func(input, output common.Address) error
}

func DeployOPChain(l1Host *script.Host, input *DeployOPChainInput) (*DeployOPChainOutput, error) {
	output := &DeployOPChainOutput{}
	inputAddr := l1Host.NewScriptAddress()
	outputAddr := l1Host.NewScriptAddress()

	cleanupInput, err := script.WithPrecompileAtAddress[*DeployOPChainInput](l1Host, inputAddr, input)
	if err != nil {
		return nil, fmt.Errorf("failed to insert DeployOPChainInput precompile: %w", err)
	}
	defer cleanupInput()

	cleanupOutput, err := script.WithPrecompileAtAddress[*DeployOPChainOutput](l1Host, outputAddr, output,
		script.WithFieldSetter[*DeployOPChainOutput])
	if err != nil {
		return nil, fmt.Errorf("failed to insert DeployOPChainOutput precompile: %w", err)
	}
	defer cleanupOutput()

	deployScript, cleanupDeploy, err := script.WithScript[DeployOPChainScript](l1Host, "DeployOPChain.s.sol", "DeployOPChain")
	if err != nil {
		return nil, fmt.Errorf("failed to load DeployOPChain script: %w", err)
	}
	defer cleanupDeploy()

	if err := deployScript.Run(inputAddr, outputAddr); err != nil {
		return nil, fmt.Errorf("failed to run DeployOPChain script: %w", err)
	}

	return output, nil
}
