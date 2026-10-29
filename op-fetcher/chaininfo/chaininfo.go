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

// Info is the flat result of a fetch. A RespectedGameType of math.MaxUint32 means the chain
// has no fault proofs.
type Info struct {
	Addresses
	addresses.OpChainRoles
	FaultProofStatus
}

// Config is the structured form of an Info, as the superchain-registry stores it.
type Config struct {
	Addresses        Addresses              `json:"addresses"`
	Roles            addresses.OpChainRoles `json:"roles"`
	FaultProofStatus *FaultProofStatus      `json:"faultProofs,omitempty" toml:"fault_proofs,omitempty"`
}

// Config returns the structured form of i, without fault proofs if the chain has none.
func (i Info) Config() Config {
	cfg := Config{
		Addresses: i.Addresses,
		Roles:     i.OpChainRoles,
	}
	if i.RespectedGameType != math.MaxUint32 {
		cfg.FaultProofStatus = &i.FaultProofStatus
	}
	return cfg
}
