// Package chaininfo holds the chain configuration that op-fetcher reads from L1, in the
// shapes it is returned and serialized in.
package chaininfo

import (
	"math"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum/go-ethereum/common"
)

type Addresses struct {
	addresses.OpChainContracts
	// Shared singletons
	SuperchainConfigProxy common.Address
	MipsImpl              common.Address
	PreimageOracleImpl    common.Address
}

type FaultProofStatus struct {
	Permissioned      bool   `toml:"permissioned" json:"permissioned"`
	Permissionless    bool   `toml:"permissionless" json:"permissionless"`
	RespectedGameType uint32 `toml:"respected_game_type" json:"respectedGameType"`
}

type FetchChainInfoInput struct {
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
}

// FetchChainInfoOutput is the flat result of a fetch. A RespectedGameType of math.MaxUint32
// means the chain has no fault proofs.
type FetchChainInfoOutput struct {
	Addresses
	addresses.OpChainRoles
	FaultProofStatus
}

type ChainConfig struct {
	Addresses        Addresses              `json:"addresses"`
	Roles            addresses.OpChainRoles `json:"roles"`
	FaultProofStatus *FaultProofStatus      `json:"faultProofs,omitempty" toml:"fault_proofs,omitempty"`
}

// CreateChainConfig creates a nicely structured output from the flat FetchChainInfoOutput
func CreateChainConfig(output FetchChainInfoOutput) ChainConfig {
	chain := ChainConfig{
		Addresses: output.Addresses,
		Roles:     output.OpChainRoles,
	}

	if output.FaultProofStatus.RespectedGameType == math.MaxUint32 {
		chain.FaultProofStatus = nil
	} else {
		chain.FaultProofStatus = &output.FaultProofStatus
	}
	return chain
}
