package opsm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// PermissionedGameStartingAnchorRoots is a root of bytes32(hex"dead") for the permissioned game at block 0,
// and no root for the permissionless game.
var PermissionedGameStartingAnchorRoots = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xde, 0xad, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

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
	OpsmProxy         common.Address
}

func (input *DeployOPChainInput) InputSet() bool {
	return true
}

func (input *DeployOPChainInput) StartingAnchorRoots() []byte {
	return PermissionedGameStartingAnchorRoots
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
	AnchorStateRegistryProxy           common.Address
	AnchorStateRegistryImpl            common.Address
	FaultDisputeGame                   common.Address
	PermissionedDisputeGame            common.Address
	DelayedWETHPermissionedGameProxy   common.Address
	DelayedWETHPermissionlessGameProxy common.Address
}

func (output *DeployOPChainOutput) CheckOutput(input common.Address) error {
	return nil
}

type DeployOPChainScript struct {
	Run func(input, output common.Address) error
}

func DeployOPChain(host *script.Host, input DeployOPChainInput) (DeployOPChainOutput, error) {
	var dco DeployOPChainOutput
	inputAddr := host.NewScriptAddress()
	outputAddr := host.NewScriptAddress()

	cleanupInput, err := script.WithPrecompileAtAddress[*DeployOPChainInput](host, inputAddr, &input)
	if err != nil {
		return dco, fmt.Errorf("failed to insert DeployOPChainInput precompile: %w", err)
	}
	defer cleanupInput()
	host.Label(inputAddr, "DeployOPChainInput")

	cleanupOutput, err := script.WithPrecompileAtAddress[*DeployOPChainOutput](host, outputAddr, &dco,
		script.WithFieldSetter[*DeployOPChainOutput])
	if err != nil {
		return dco, fmt.Errorf("failed to insert DeployOPChainOutput precompile: %w", err)
	}
	defer cleanupOutput()
	host.Label(outputAddr, "DeployOPChainOutput")

	deployScript, cleanupDeploy, err := script.WithScript[DeployOPChainScript](host, "DeployOPChain.s.sol", "DeployOPChain")
	if err != nil {
		return dco, fmt.Errorf("failed to load DeployOPChain script: %w", err)
	}
	defer cleanupDeploy()

	if err := deployScript.Run(inputAddr, outputAddr); err != nil {
		return dco, fmt.Errorf("failed to run DeployOPChain script: %w", err)
	}

	return dco, nil
}
