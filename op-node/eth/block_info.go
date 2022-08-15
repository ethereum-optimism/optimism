package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type BlockInfo interface {
	Hash() common.Hash
	ParentHash() common.Hash
	Coinbase() common.Address
	Root() common.Hash // state-root
	NumberU64() uint64
	Time() uint64
	// MixDigest field, reused for randomness after The Merge (Bellatrix hardfork)
	MixDigest() common.Hash
	BaseFee() *big.Int
	ID() BlockID
	ReceiptHash() common.Hash
}

func InfoToL1BlockRef(info BlockInfo) L1BlockRef {
	return L1BlockRef{
		Hash:       info.Hash(),
		Number:     info.NumberU64(),
		ParentHash: info.ParentHash(),
		Time:       info.Time(),
	}
}
