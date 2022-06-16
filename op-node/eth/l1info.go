package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type L1Info interface {
	Hash() common.Hash
	ParentHash() common.Hash
	Root() common.Hash // state-root
	NumberU64() uint64
	Time() uint64
	// MixDigest field, reused for randomness after The Merge (Bellatrix hardfork)
	MixDigest() common.Hash
	BaseFee() *big.Int
	ID() BlockID
	BlockRef() L1BlockRef
	ReceiptHash() common.Hash
}
