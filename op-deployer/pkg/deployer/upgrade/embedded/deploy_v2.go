package embedded

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

type DeployOPChainV2Input struct {
	Opcm         common.Address `json:"opcm"`
	FullConfigV2 FullConfigV2   `evm:"-" json:"fullConfig"`
}

type FullConfigV2 struct {
	SaltMixer                 string              `json:"saltMixer"`
	SuperchainConfig          common.Address      `json:"superchainConfig"`
	ProxyAdminOwner           common.Address      `json:"proxyAdminOwner"`
	SystemConfigOwner         common.Address      `json:"systemConfigOwner"`
	UnsafeBlockSigner         common.Address      `json:"unsafeBlockSigner"`
	Batcher                   common.Address      `json:"batcher"`
	StartingAnchorRoot        Proposal            `json:"startingAnchorRoot"`
	StartingRespectedGameType uint32              `json:"startingRespectedGameType"`
	BasefeeScalar             uint32              `json:"basefeeScalar"`
	BlobBasefeeScalar         uint32              `json:"blobBasefeeScalar"`
	GasLimit                  uint64              `json:"gasLimit"`
	L2ChainId                 *big.Int            `json:"l2ChainId"`
	ResourceConfig            ResourceConfig      `json:"resourceConfig"`
	DisputeGameConfigs        []DisputeGameConfig `json:"disputeGameConfigs"`
}

type Proposal struct {
	Root             [32]byte `json:"root"`
	L2SequenceNumber uint64   `json:"l2SequenceNumber"`
}

type ResourceConfig struct {
	MaxResourceLimit            uint32   `json:"maxResourceLimit"`
	ElasticityMultiplier        uint8    `json:"elasticityMultiplier"`
	BaseFeeMaxChangeDenominator uint8    `json:"baseFeeMaxChangeDenominator"`
	MinimumBaseFee              uint32   `json:"minimumBaseFee"`
	SystemTxMaxGas              uint32   `json:"systemTxMaxGas"`
	MaximumBaseFee              *big.Int `json:"maximumBaseFee"`
}

type DeployOPChainV2Output struct {
	ChainContractsV2 ChainContracts `evm:"-" json:"chainContracts"`
}

func (d *DeployOPChainV2Output) ChainContracts() ([]byte, error) {
	return chainContractsEncoder.EncodeArgs(&d.ChainContractsV2)
}

func (d *DeployOPChainV2Output) SetChainContracts(data []byte) error {
	return chainContractsEncoder.DecodeReturns(data, &d.ChainContractsV2)
}

type ChainContracts struct {
	SystemConfig                     common.Address `json:"systemConfig"`
	ProxyAdmin                       common.Address `json:"proxyAdmin"`
	AddressManager                   common.Address `json:"addressManager"`
	L1CrossDomainMessenger           common.Address `json:"l1CrossDomainMessenger"`
	L1ERC721Bridge                   common.Address `json:"l1ERC721Bridge"`
	L1StandardBridge                 common.Address `json:"l1StandardBridge"`
	OptimismPortal                   common.Address `json:"optimismPortal"`
	EthLockbox                       common.Address `json:"ethLockbox"`
	OptimismMintableERC20Factory     common.Address `json:"optimismMintableERC20Factory"`
	DisputeGameFactory               common.Address `json:"disputeGameFactory"`
	AnchorStateRegistry              common.Address `json:"anchorStateRegistry"`
	DelayedWETH                      common.Address `json:"delayedWETH"`
}

var fullConfigEncoder = w3.MustNewFunc(
	"dummy((string saltMixer,address superchainConfig,address proxyAdminOwner,address systemConfigOwner,address unsafeBlockSigner,address batcher,(bytes32 root,uint256 l2SequenceNumber) startingAnchorRoot,uint32 startingRespectedGameType,uint32 basefeeScalar,uint32 blobBasefeeScalar,uint64 gasLimit,uint256 l2ChainId,(uint32 maxResourceLimit,uint8 elasticityMultiplier,uint8 baseFeeMaxChangeDenominator,uint32 minimumBaseFee,uint32 systemTxMaxGas,uint128 maximumBaseFee) resourceConfig,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs))",
	"",
)

var chainContractsEncoder = w3.MustNewFunc(
	"dummy()",
	"(address systemConfig,address proxyAdmin,address addressManager,address l1CrossDomainMessenger,address l1ERC721Bridge,address l1StandardBridge,address optimismPortal,address ethLockbox,address optimismMintableERC20Factory,address disputeGameFactory,address anchorStateRegistry,address delayedWETH)",
)

func (d *DeployOPChainV2Input) FullConfig() ([]byte, error) {
	data, err := fullConfigEncoder.EncodeArgs(&d.FullConfigV2)
	if err != nil {
		return nil, fmt.Errorf("failed to encode full config: %w", err)
	}

	// Strip the 4-byte function selector
	return data[4:], nil
}

func DeployOPChainV2(host *script.Host, input DeployOPChainV2Input) (*DeployOPChainV2Output, error) {
	output, err := opcm.RunScriptSingle[DeployOPChainV2Input, DeployOPChainV2Output](
		host,
		input,
		"DeployOPChainV2.s.sol",
		"DeployOPChainV2",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy OP chain V2: %w", err)
	}
	return &output, nil
}
