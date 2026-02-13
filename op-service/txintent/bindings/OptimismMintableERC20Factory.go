package bindings

import (
	"github.com/ethereum/go-ethereum/common"
)

type OptimismMintableERC20Factory struct {
	// Read-only functions
	Bridge  func() TypedCall[common.Address] `sol:"bridge"`
	Version func() TypedCall[string]         `sol:"version"`

	// Write functions
	CreateOptimismMintableERC20 func(remoteToken common.Address, name string, symbol string) TypedCall[any] `sol:"createOptimismMintableERC20"`
	CreateStandardL2Token       func(remoteToken common.Address, name string, symbol string) TypedCall[any] `sol:"createStandardL2Token"`
}
