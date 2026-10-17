package bindings

import (
	"github.com/ethereum/go-ethereum/common"
)

// SuperchainConfig binds the L1 SuperchainConfig.
type SuperchainConfig struct {
	// Write functions. Only the guardian can call them; the zero identifier pauses every chain.
	Pause   func(identifier common.Address) TypedCall[any] `sol:"pause"`
	Unpause func(identifier common.Address) TypedCall[any] `sol:"unpause"`
}
