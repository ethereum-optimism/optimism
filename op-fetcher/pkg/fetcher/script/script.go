package script

import (
	_ "embed"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
)

type FetchChainInfoInput struct {
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
}

// Copied struct from superchain-registry/ops/internal/config/chain.go
type Addresses struct {
	AddressManager                    common.Address `toml:"AddressManager,omitempty" json:"AddressManager,omitempty"`
	L1CrossDomainMessengerProxy       common.Address `toml:"L1CrossDomainMessengerProxy,omitempty" json:"L1CrossDomainMessengerProxy,omitempty"`
	L1ERC721BridgeProxy               common.Address `toml:"L1ERC721BridgeProxy,omitempty" json:"L1ERC721BridgeProxy,omitempty"`
	L1StandardBridgeProxy             common.Address `toml:"L1StandardBridgeProxy,omitempty" json:"L1StandardBridgeProxy,omitempty"`
	L2OutputOracleProxy               common.Address `toml:"L2OutputOracleProxy,omitempty" json:"L2OutputOracleProxy,omitempty"`
	OptimismMintableERC20FactoryProxy common.Address `toml:"OptimismMintableERC20FactoryProxy,omitempty" json:"OptimismMintableERC20FactoryProxy,omitempty"`
	OptimismPortalProxy               common.Address `toml:"OptimismPortalProxy,omitempty" json:"OptimismPortalProxy,omitempty"`
	SystemConfigProxy                 common.Address `toml:"SystemConfigProxy,omitempty" json:"SystemConfigProxy,omitempty"`
	ProxyAdmin                        common.Address `toml:"ProxyAdmin,omitempty" json:"ProxyAdmin,omitempty"`
	SuperchainConfig                  common.Address `toml:"SuperchainConfig,omitempty" json:"SuperchainConfig,omitempty"`
	AnchorStateRegistryProxy          common.Address `toml:"AnchorStateRegistryProxy,omitempty" json:"AnchorStateRegistryProxy,omitempty"`
	DelayedWETHProxy                  common.Address `toml:"DelayedWETHProxy,omitempty" json:"DelayedWETHProxy,omitempty"`
	DisputeGameFactoryProxy           common.Address `toml:"DisputeGameFactoryProxy,omitempty" json:"DisputeGameFactoryProxy,omitempty"`
	FaultDisputeGame                  common.Address `toml:"FaultDisputeGame,omitempty" json:"FaultDisputeGame,omitempty"`
	Mips                              common.Address `toml:"MIPS,omitempty" json:"MIPS,omitempty"`
	PermissionedDisputeGame           common.Address `toml:"PermissionedDisputeGame,omitempty" json:"PermissionedDisputeGame,omitempty"`
	PreimageOracle                    common.Address `toml:"PreimageOracle,omitempty" json:"PreimageOracle,omitempty"`
	DaChallengeAddress                common.Address `toml:"DAChallengeAddress,omitempty" json:"DAChallengeAddress,omitempty"`
}

type Roles struct {
	SystemConfigOwner common.Address `toml:"SystemConfigOwner,omitempty" json:"SystemConfigOwner,omitempty"`
	ProxyAdminOwner   common.Address `toml:"ProxyAdminOwner,omitempty" json:"ProxyAdminOwner,omitempty"`
	Guardian          common.Address `toml:"Guardian,omitempty" json:"Guardian,omitempty"`
	Challenger        common.Address `toml:"Challenger,omitempty" json:"Challenger,omitempty"`
	Proposer          common.Address `toml:"Proposer,omitempty" json:"Proposer,omitempty"`
	UnsafeBlockSigner common.Address `toml:"UnsafeBlockSigner,omitempty" json:"UnsafeBlockSigner,omitempty"`
	BatchSubmitter    common.Address `toml:"BatchSubmitter,omitempty" json:"BatchSubmitter,omitempty"`
}

type FetchChainInfoOutput struct {
	Addresses
	Roles
	FaultProofPermissioned   bool
	FaultProofPermissionless bool
}

type ChainConfig struct {
	Addresses        Addresses        `toml:"addresses" json:"addresses"`
	Roles            Roles            `toml:"roles" json:"roles"`
	FaultProofStatus FaultProofStatus `toml:"fault_proof_status" json:"fault_proof_status"`
}

type FaultProofStatus struct {
	Permissioned   bool
	Permissionless bool
}

func (output *FetchChainInfoOutput) CheckOutput(input common.Address) error {
	return nil
}

func FetchChainInfo(h *script.Host, input FetchChainInfoInput) (FetchChainInfoOutput, error) {
	return opcm.RunScriptSingle[FetchChainInfoInput, FetchChainInfoOutput](h, input, "FetchChainInfo.s.sol", "FetchChainInfo")
}
