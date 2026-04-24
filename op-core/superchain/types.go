package superchain

import (
	"github.com/ethereum/go-ethereum/common"
)

type ChainConfig struct {
	Name                 string       `toml:"name"`
	PublicRPC            string       `toml:"public_rpc"`
	SequencerRPC         string       `toml:"sequencer_rpc"`
	Explorer             string       `toml:"explorer"`
	SuperchainLevel      int          `toml:"superchain_level"`
	GovernedByOptimism   bool         `toml:"governed_by_optimism"`
	SuperchainTime       *uint64      `toml:"superchain_time"`
	DataAvailabilityType string       `toml:"data_availability_type"`
	DeploymentTxHash     *common.Hash `toml:"deployment_tx_hash"`

	ChainID           uint64          `toml:"chain_id"`
	BatchInboxAddr    common.Address  `toml:"batch_inbox_addr"`
	BlockTime         uint64          `toml:"block_time"`
	SeqWindowSize     uint64          `toml:"seq_window_size"`
	MaxSequencerDrift uint64          `toml:"max_sequencer_drift"`
	GasPayingToken    *common.Address `toml:"gas_paying_token"`
	Hardforks         HardforkConfig  `toml:"hardforks"`
	Interop           *Interop        `toml:"interop,omitempty"`
	Optimism          *OptimismConfig `toml:"optimism,omitempty"`

	AltDA *AltDAConfig `toml:"alt_da,omitempty"`

	Genesis GenesisConfig `toml:"genesis"`

	Roles RolesConfig `toml:"roles"`

	Addresses AddressesConfig `toml:"addresses"`
}

type Dependency struct{}

type Interop struct {
	Dependencies map[string]Dependency `json:"dependencies" toml:"dependencies"`
}

type HardforkConfig struct {
	CanyonTime   *uint64 `toml:"canyon_time"`
	DeltaTime    *uint64 `toml:"delta_time"`
	EcotoneTime  *uint64 `toml:"ecotone_time"`
	FjordTime    *uint64 `toml:"fjord_time"`
	GraniteTime  *uint64 `toml:"granite_time"`
	HoloceneTime *uint64 `toml:"holocene_time"`
	IsthmusTime  *uint64 `toml:"isthmus_time"`
	JovianTime   *uint64 `toml:"jovian_time"`
	KarstTime    *uint64 `toml:"karst_time"`
	InteropTime  *uint64 `toml:"interop_time"`
	// Optional Forks
	PectraBlobScheduleTime *uint64 `toml:"pectra_blob_schedule_time,omitempty"`
}

type OptimismConfig struct {
	EIP1559Elasticity        uint64  `toml:"eip1559_elasticity"`
	EIP1559Denominator       uint64  `toml:"eip1559_denominator"`
	EIP1559DenominatorCanyon *uint64 `toml:"eip1559_denominator_canyon"`
}

type AltDAConfig struct {
	DaChallengeContractAddress common.Address `toml:"da_challenge_contract_address"`
	DaChallengeWindow          uint64         `toml:"da_challenge_window"`
	DaResolveWindow            uint64         `toml:"da_resolve_window"`
	DaCommitmentType           string         `toml:"da_commitment_type"`
}

type GenesisConfig struct {
	L2Time       uint64       `toml:"l2_time"`
	L1           GenesisRef   `toml:"l1"`
	L2           GenesisRef   `toml:"l2"`
	SystemConfig SystemConfig `toml:"system_config"`
}

type GenesisRef struct {
	Hash   common.Hash `toml:"hash"`
	Number uint64      `toml:"number"`
}

type SystemConfig struct {
	BatcherAddr       common.Address `json:"batcherAddr" toml:"batcherAddress"`
	Overhead          common.Hash    `json:"overhead" toml:"overhead"`
	Scalar            common.Hash    `json:"scalar" toml:"scalar"`
	GasLimit          uint64         `json:"gasLimit" toml:"gasLimit"`
	BaseFeeScalar     *uint64        `json:"baseFeeScalar,omitempty" toml:"baseFeeScalar,omitempty"`
	BlobBaseFeeScalar *uint64        `json:"blobBaseFeeScalar,omitempty" toml:"blobBaseFeeScalar,omitempty"`
}

type RolesConfig struct {
	SystemConfigOwner *common.Address `json:"SystemConfigOwner" toml:"SystemConfigOwner"`
	ProxyAdminOwner   *common.Address `json:"ProxyAdminOwner" toml:"ProxyAdminOwner"`
	Guardian          *common.Address `json:"Guardian" toml:"Guardian"`
	Challenger        *common.Address `json:"Challenger" toml:"Challenger"`
	Proposer          *common.Address `json:"Proposer,omitempty" toml:"Proposer,omitempty"`
	UnsafeBlockSigner *common.Address `json:"UnsafeBlockSigner,omitempty" toml:"UnsafeBlockSigner,omitempty"`
	BatchSubmitter    *common.Address `json:"BatchSubmitter" toml:"BatchSubmitter"`
}

type AddressesConfig struct {
	AddressManager                    *common.Address `toml:"AddressManager,omitempty" json:"AddressManager,omitempty"`
	L1CrossDomainMessengerProxy       *common.Address `toml:"L1CrossDomainMessengerProxy,omitempty" json:"L1CrossDomainMessengerProxy,omitempty"`
	L1ERC721BridgeProxy               *common.Address `toml:"L1ERC721BridgeProxy,omitempty" json:"L1ERC721BridgeProxy,omitempty"`
	L1StandardBridgeProxy             *common.Address `toml:"L1StandardBridgeProxy,omitempty" json:"L1StandardBridgeProxy,omitempty"`
	L2OutputOracleProxy               *common.Address `toml:"L2OutputOracleProxy,omitempty" json:"L2OutputOracleProxy,omitempty"`
	OptimismMintableERC20FactoryProxy *common.Address `toml:"OptimismMintableERC20FactoryProxy,omitempty" json:"OptimismMintableERC20FactoryProxy,omitempty"`
	OptimismPortalProxy               *common.Address `toml:"OptimismPortalProxy,omitempty" json:"OptimismPortalProxy,omitempty"`
	SystemConfigProxy                 *common.Address `toml:"SystemConfigProxy,omitempty" json:"SystemConfigProxy,omitempty"`
	ProxyAdmin                        *common.Address `toml:"ProxyAdmin,omitempty" json:"ProxyAdmin,omitempty"`
	SuperchainConfig                  *common.Address `toml:"SuperchainConfig,omitempty" json:"SuperchainConfig,omitempty"`
	AnchorStateRegistryProxy          *common.Address `toml:"AnchorStateRegistryProxy,omitempty" json:"AnchorStateRegistryProxy,omitempty"`
	DelayedWETHProxy                  *common.Address `toml:"DelayedWETHProxy,omitempty" json:"DelayedWETHProxy,omitempty"`
	DisputeGameFactoryProxy           *common.Address `toml:"DisputeGameFactoryProxy,omitempty" json:"DisputeGameFactoryProxy,omitempty"`
	FaultDisputeGame                  *common.Address `toml:"FaultDisputeGame,omitempty" json:"FaultDisputeGame,omitempty"`
	MIPS                              *common.Address `toml:"MIPS,omitempty" json:"MIPS,omitempty"`
	PermissionedDisputeGame           *common.Address `toml:"PermissionedDisputeGame,omitempty" json:"PermissionedDisputeGame,omitempty"`
	PreimageOracle                    *common.Address `toml:"PreimageOracle,omitempty" json:"PreimageOracle,omitempty"`
	DAChallengeAddress                *common.Address `toml:"DAChallengeAddress,omitempty" json:"DAChallengeAddress,omitempty"`
}
