package bindings

import (
	"math/big"
)

// ETHLiquidity binds the ETHLiquidity predeploy.
type ETHLiquidity struct {
	// Read-only functions
	Paused             func() TypedCall[bool]     `sol:"paused"`
	PausedMessageNonce func() TypedCall[*big.Int] `sol:"pausedMessageNonce"`
}
