package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// L2ToL2CrossDomainMessenger binds the message expiry functions of the L2ToL2CrossDomainMessenger
// predeploy.
type L2ToL2CrossDomainMessenger struct {
	// Read-only functions
	SentMessageTimestamps func(messageHash [32]byte) TypedCall[*big.Int] `sol:"sentMessageTimestamps"`
	ExpiredMessages       func(messageHash [32]byte) TypedCall[bool]     `sol:"expiredMessages"`
	ExpiryHub             func() TypedCall[common.Address]               `sol:"expiryHub"`
	ProxyAdminOwner       func() TypedCall[common.Address]               `sol:"proxyAdminOwner"`

	// Write functions
	SetExpiryHub             func(expiryHub common.Address) TypedCall[any]                                                                                                     `sol:"setExpiryHub"`
	ExportUndeliveredMessage func(source eth.ChainID, nonce *big.Int, sender common.Address, target common.Address, message []byte, minGasLimit uint32) TypedCall[eth.Bytes32] `sol:"exportUndeliveredMessage"`
}
