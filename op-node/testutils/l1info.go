package testutils

import (
	"math/big"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type MockL1Info struct {
	// Prefixed all fields with "Info" to avoid collisions with the interface method names.

	InfoHash           common.Hash
	InfoParentHash     common.Hash
	InfoRoot           common.Hash
	InfoNum            uint64
	InfoTime           uint64
	InfoMixDigest      [32]byte
	InfoBaseFee        *big.Int
	InfoReceiptRoot    common.Hash
	InfoSequenceNumber uint64
}

func (l *MockL1Info) Hash() common.Hash {
	return l.InfoHash
}

func (l *MockL1Info) ParentHash() common.Hash {
	return l.InfoParentHash
}

func (l *MockL1Info) Root() common.Hash {
	return l.InfoRoot
}

func (l *MockL1Info) NumberU64() uint64 {
	return l.InfoNum
}

func (l *MockL1Info) Time() uint64 {
	return l.InfoTime
}

func (l *MockL1Info) MixDigest() common.Hash {
	return l.InfoMixDigest
}

func (l *MockL1Info) BaseFee() *big.Int {
	return l.InfoBaseFee
}

func (l *MockL1Info) ReceiptHash() common.Hash {
	return l.InfoReceiptRoot
}

func (l *MockL1Info) ID() eth.BlockID {
	return eth.BlockID{Hash: l.InfoHash, Number: l.InfoNum}
}

func (l *MockL1Info) BlockRef() eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       l.InfoHash,
		Number:     l.InfoNum,
		ParentHash: l.InfoParentHash,
		Time:       l.InfoTime,
	}
}

func (l *MockL1Info) SequenceNumber() uint64 {
	return l.InfoSequenceNumber
}

func RandomL1Info(rng *rand.Rand) *MockL1Info {
	return &MockL1Info{
		InfoParentHash:     RandomHash(rng),
		InfoNum:            rng.Uint64(),
		InfoTime:           rng.Uint64(),
		InfoHash:           RandomHash(rng),
		InfoBaseFee:        big.NewInt(rng.Int63n(1000_000 * 1e9)), // a million GWEI
		InfoReceiptRoot:    types.EmptyRootHash,
		InfoRoot:           RandomHash(rng),
		InfoSequenceNumber: rng.Uint64(),
	}
}

func MakeL1Info(fn func(l *MockL1Info)) func(rng *rand.Rand) *MockL1Info {
	return func(rng *rand.Rand) *MockL1Info {
		l := RandomL1Info(rng)
		if fn != nil {
			fn(l)
		}
		return l
	}
}
