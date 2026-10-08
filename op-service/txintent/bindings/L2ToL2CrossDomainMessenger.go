package bindings

import (
	"math/big"
)

// L2ToL2CrossDomainMessenger binds the message expiry state of the L2ToL2CrossDomainMessenger
// predeploy.
type L2ToL2CrossDomainMessenger struct {
	// Read-only functions
	ExpiryPeriod          func() TypedCall[*big.Int]                     `sol:"expiryPeriod"`
	ExpiredMessages       func(messageHash [32]byte) TypedCall[bool]     `sol:"expiredMessages"`
	SentMessageTimestamps func(messageHash [32]byte) TypedCall[*big.Int] `sol:"sentMessageTimestamps"`
	Version               func() TypedCall[string]                       `sol:"version"`
}
