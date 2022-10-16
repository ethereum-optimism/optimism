package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var _ BlockInfo = (&types.Block{})

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

type NumberAndHash interface {
	Hash() common.Hash
	NumberU64() uint64
}

func ToBlockID(b NumberAndHash) BlockID {
	return BlockID{
		Hash:   b.Hash(),
		Number: b.NumberU64(),
	}
}
