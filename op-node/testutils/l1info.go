package testutils

import (
	"math/big"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type MockBlockInfo struct {
	// Prefixed all fields with "Info" to avoid collisions with the interface method names.
	InfoBaseFee     *big.Int
	InfoNum         uint64
	InfoTime        uint64
	InfoHash        common.Hash
	InfoParentHash  common.Hash
	InfoRoot        common.Hash
	InfoMixDigest   [32]byte
	InfoReceiptRoot common.Hash
	InfoCoinbase    common.Address
}

func (l *MockBlockInfo) Hash() common.Hash {
	return l.InfoHash
}

func (l *MockBlockInfo) ParentHash() common.Hash {
	return l.InfoParentHash
}

func (l *MockBlockInfo) Coinbase() common.Address {
	return l.InfoCoinbase
}

func (l *MockBlockInfo) Root() common.Hash {
	return l.InfoRoot
}

func (l *MockBlockInfo) NumberU64() uint64 {
	return l.InfoNum
}

func (l *MockBlockInfo) Time() uint64 {
	return l.InfoTime
}

func (l *MockBlockInfo) MixDigest() common.Hash {
	return l.InfoMixDigest
}

func (l *MockBlockInfo) BaseFee() *big.Int {
	return l.InfoBaseFee
}

func (l *MockBlockInfo) ReceiptHash() common.Hash {
	return l.InfoReceiptRoot
}

func (l *MockBlockInfo) ID() eth.BlockID {
	return eth.BlockID{Hash: l.InfoHash, Number: l.InfoNum}
}

func (l *MockBlockInfo) BlockRef() eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       l.InfoHash,
		Number:     l.InfoNum,
		ParentHash: l.InfoParentHash,
		Time:       l.InfoTime,
	}
}

func RandomBlockInfo(rng *rand.Rand) *MockBlockInfo {
	return &MockBlockInfo{
		InfoParentHash:  RandomHash(rng),
		InfoNum:         rng.Uint64(),
		InfoTime:        rng.Uint64(),
		InfoHash:        RandomHash(rng),
		InfoBaseFee:     big.NewInt(rng.Int63n(1000_000 * 1e9)), // a million GWEI
		InfoReceiptRoot: types.EmptyRootHash,
		InfoRoot:        RandomHash(rng),
	}
}

func MakeBlockInfo(fn func(l *MockBlockInfo)) func(rng *rand.Rand) *MockBlockInfo {
	return func(rng *rand.Rand) *MockBlockInfo {
		l := RandomBlockInfo(rng)
		if fn != nil {
			fn(l)
		}
		return l
	}
}
