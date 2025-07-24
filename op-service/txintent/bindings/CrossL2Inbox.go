package bindings

import (
	"math/big"

	"github.com/HashKeyChain/verse/op-service/eth"
	supTypes "github.com/HashKeyChain/verse/op-supervisor/supervisor/types"
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
	ValidateMessage func(identifier supTypes.Identifier, msgHash eth.Bytes32) TypedCall[any] `sol:"validateMessage"`
}
