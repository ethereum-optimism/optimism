package bindings

import (
	"github.com/ethereum/go-ethereum/common"
)

// L1CrossDomainMessenger binds a chain's L1CrossDomainMessenger.
type L1CrossDomainMessenger struct {
	// Write functions
	SyncETHLiquidityPause func() TypedCall[any] `sol:"syncETHLiquidityPause"`

	// Read-only functions
	SuperchainConfig func() TypedCall[common.Address] `sol:"superchainConfig"`
}
