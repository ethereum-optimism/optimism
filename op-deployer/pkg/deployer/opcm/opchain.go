package opcm

import (
	_ "embed"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

// PermissionedGameStartingAnchorRoot is a root of bytes32(hex"dead") for the permissioned game at block 0,
// and no root for the permissionless game.
var PermissionedGameStartingAnchorRoot = []byte{
	0xde, 0xad, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
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
	Opcm              common.Address
	SaltMixer         string
	GasLimit          uint64

	DisputeGameType              uint32
	DisputeAbsolutePrestate      common.Hash
	DisputeMaxGameDepth          *big.Int
	DisputeSplitDepth            *big.Int
	DisputeClockExtension        uint64
	DisputeMaxClockDuration      uint64
	AllowCustomDisputeParameters bool

	OperatorFeeScalar   uint32
	OperatorFeeConstant uint64
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
	EthLockboxProxy                    common.Address `evm:"ethLockboxProxy"`
	DisputeGameFactoryProxy            common.Address
	AnchorStateRegistryProxy           common.Address
	FaultDisputeGame                   common.Address
	PermissionedDisputeGame            common.Address
	DelayedWETHPermissionedGameProxy   common.Address
	DelayedWETHPermissionlessGameProxy common.Address
}

type DeployOPChainScript script.DeployScriptWithOutput[DeployOPChainInput, DeployOPChainOutput]

// NewDeployOPChainScript loads and validates the DeployOPChain script contract
func NewDeployOPChainScript(host *script.Host) (DeployOPChainScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployOPChainInput, DeployOPChainOutput](host, "DeployOPChain.s.sol", "DeployOPChain")
}

func NewDeployOPChainForgeCaller(client *forge.Client) forge.ScriptCaller[DeployOPChainInput, DeployOPChainOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployOPChain.s.sol:DeployOPChain",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployOPChainInput]{TypeName: "DeployOPChainInput"},
		&forge.BytesScriptDecoder[DeployOPChainOutput]{TypeName: "DeployOPChainOutput"},
	)
}

type ReadImplementationAddressesInput struct {
	AddressManager                    common.Address
	L1ERC721BridgeProxy               common.Address
	SystemConfigProxy                 common.Address
	OptimismMintableERC20FactoryProxy common.Address
	L1StandardBridgeProxy             common.Address
	OptimismPortalProxy               common.Address
	DisputeGameFactoryProxy           common.Address
	DelayedWETHPermissionedGameProxy  common.Address
	Opcm                              common.Address
}

type ReadImplementationAddressesOutput struct {
	DelayedWETH                  common.Address
	OptimismPortal               common.Address
	OptimismPortalInterop        common.Address
	EthLockbox                   common.Address `evm:"ethLockbox"`
	SystemConfig                 common.Address
	L1CrossDomainMessenger       common.Address
	L1ERC721Bridge               common.Address
	L1StandardBridge             common.Address
	OptimismMintableERC20Factory common.Address
	DisputeGameFactory           common.Address
	MipsSingleton                common.Address
	PreimageOracleSingleton      common.Address
}

type ReadImplementationAddressesScript script.DeployScriptWithOutput[ReadImplementationAddressesInput, ReadImplementationAddressesOutput]

// NewReadImplementationAddressesScript loads and validates the ReadImplementationAddresses script contract
func NewReadImplementationAddressesScript(host *script.Host) (ReadImplementationAddressesScript, error) {
	return script.NewDeployScriptWithOutputFromFile[ReadImplementationAddressesInput, ReadImplementationAddressesOutput](host, "ReadImplementationAddresses.s.sol", "ReadImplementationAddresses")
}

func NewReadImplementationAddressesForgeCaller(client *forge.Client) forge.ScriptCaller[ReadImplementationAddressesInput, ReadImplementationAddressesOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/ReadImplementationAddresses.s.sol:ReadImplementationAddresses",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[ReadImplementationAddressesInput]{TypeName: "ReadImplementationAddressesInput"},
		&forge.BytesScriptDecoder[ReadImplementationAddressesOutput]{TypeName: "ReadImplementationAddressesOutput"},
	)
}
