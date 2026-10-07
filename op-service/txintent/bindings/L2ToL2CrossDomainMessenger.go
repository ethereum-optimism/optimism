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
	ExpiredMessages func(messageHash [32]byte) TypedCall[bool] `sol:"expiredMessages"`

	// Write functions
	ExportUndeliveredMessage func(sourceMessenger common.Address, source eth.ChainID, nonce *big.Int, sender common.Address, target common.Address, message []byte, minGasLimit uint32) TypedCall[eth.Bytes32] `sol:"exportUndeliveredMessage"`
}
