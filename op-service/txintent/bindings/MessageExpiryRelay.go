package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// SentMessageRecordResult is the tuple returned by the MessageExpiryRelay's
// `sentMessageRecords` mapping getter. RecordedAt is a uint96 on chain; it is decoded as a
// *big.Int because every field of the getter is a single static ABI word.
type SentMessageRecordResult struct {
	App         common.Address
	RecordedAt  *big.Int
	Destination *big.Int
}

// MessageExpiryRelay binds the MessageExpiryRelay predeploy
// (predeploys.MessageExpiryRelayAddr, 0x42...2E).
type MessageExpiryRelay struct {
	// Read-only functions
	Hub                func() TypedCall[common.Address]                          `sol:"hub"`
	ExpiryWindow       func() TypedCall[*big.Int]                                `sol:"expiryWindow"`
	SentMessageRecords func(msgHash [32]byte) TypedCall[SentMessageRecordResult] `sol:"sentMessageRecords"`
	Version            func() TypedCall[string]                                  `sol:"version"`

	// Write functions
	Initialize        func(hub common.Address, expiryWindow *big.Int) TypedCall[any]                       `sol:"initialize"`
	AttestUndelivered func(msgHash [32]byte, sourceChainID eth.ChainID, minGasLimit uint32) TypedCall[any] `sol:"attestUndelivered"`
}
