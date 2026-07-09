package bindings

import (
	"math/big"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type ABIIdentifier struct {
	Origin      common.Address
	BlockNumber *big.Int
	LogIndex    *big.Int
	Timestamp   *big.Int
	ChainId     *big.Int
}

type CrossL2Inbox struct {
	ValidateMessage func(identifier messages.Identifier, msgHash eth.Bytes32) TypedCall[any] `sol:"validateMessage"`
}
