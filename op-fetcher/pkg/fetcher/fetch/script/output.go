package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
)

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
	PermissionedWethProxy             common.Address `toml:"PermissionedWethProxy,omitempty" json:"PermissionedWethProxy,omitempty"`
	PermissionlessWethProxy           common.Address `toml:"PermissionlessWethProxy,omitempty" json:"PermissionlessWethProxy,omitempty"`
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
	RespectedGameType        uint32
}

func (output *FetchChainInfoOutput) CheckOutput(input common.Address) error {
	return nil
}

func WriteChainConfigToFile(output FetchChainInfoOutput, chainName string, chainId uint64) error {
	fileData := ChainConfig{
		ChainName: chainName,
		ChainId:   chainId,
		Addresses: output.Addresses,
		Roles:     output.Roles,
		FaultProofStatus: FaultProofStatus{
			Permissioned:      output.FaultProofPermissioned,
			Permissionless:    output.FaultProofPermissionless,
			RespectedGameType: output.RespectedGameType,
		},
	}

	json, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	if err := os.MkdirAll("./.fetcher", 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile := filepath.Join("./.fetcher", fmt.Sprintf("%d.json", chainId))
	err = os.WriteFile(outputFile, json, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output to file: %w", err)
	}

	return nil
}
