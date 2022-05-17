package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

type l1MockInfo struct {
	hash           common.Hash
	parentHash     common.Hash
	root           common.Hash
	num            uint64
	time           uint64
	mixDigest      [32]byte
	baseFee        *big.Int
	receiptRoot    common.Hash
	sequenceNumber uint64
}

func (l *l1MockInfo) Hash() common.Hash {
	return l.hash
}

func (l *l1MockInfo) ParentHash() common.Hash {
	return l.parentHash
}

func (l *l1MockInfo) Root() common.Hash {
	return l.root
}

func (l *l1MockInfo) NumberU64() uint64 {
	return l.num
}

func (l *l1MockInfo) Time() uint64 {
	return l.time
}

func (l *l1MockInfo) MixDigest() common.Hash {
	return l.mixDigest
}

func (l *l1MockInfo) BaseFee() *big.Int {
	return l.baseFee
}

func (l *l1MockInfo) ReceiptHash() common.Hash {
	return l.receiptRoot
}

func (l *l1MockInfo) ID() eth.BlockID {
	return eth.BlockID{Hash: l.hash, Number: l.num}
}

func (l *l1MockInfo) BlockRef() eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       l.hash,
		Number:     l.num,
		ParentHash: l.parentHash,
		Time:       l.time,
	}
}

func randomHash(rng *rand.Rand) (out common.Hash) {
	rng.Read(out[:])
	return
}

func randomL1Info(rng *rand.Rand) *l1MockInfo {
	return &l1MockInfo{
		parentHash:     randomHash(rng),
		num:            rng.Uint64(),
		time:           rng.Uint64(),
		hash:           randomHash(rng),
		baseFee:        big.NewInt(rng.Int63n(1000_000 * 1e9)), // a million GWEI
		receiptRoot:    types.EmptyRootHash,
		root:           randomHash(rng),
		sequenceNumber: rng.Uint64(),
	}
}

func makeInfo(fn func(l *l1MockInfo)) func(rng *rand.Rand) *l1MockInfo {
	return func(rng *rand.Rand) *l1MockInfo {
		l := randomL1Info(rng)
		if fn != nil {
			fn(l)
		}
		return l
	}
}

var _ L1Info = (*l1MockInfo)(nil)

type infoTest struct {
	name   string
	mkInfo func(rng *rand.Rand) *l1MockInfo
}

var MockDepositContractAddr = common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeef00000000")

func TestParseL1InfoDepositTxData(t *testing.T) {
	// Go 1.18 will have native fuzzing for us to use, until then, we cover just the below cases
	cases := []infoTest{
		{"random", makeInfo(nil)},
		{"zero basefee", makeInfo(func(l *l1MockInfo) {
			l.baseFee = new(big.Int)
		})},
		{"zero time", makeInfo(func(l *l1MockInfo) {
			l.time = 0
		})},
		{"zero num", makeInfo(func(l *l1MockInfo) {
			l.num = 0
		})},
		{"zero seq", makeInfo(func(l *l1MockInfo) {
			l.sequenceNumber = 0
		})},
		{"all zero", func(rng *rand.Rand) *l1MockInfo {
			return &l1MockInfo{baseFee: new(big.Int)}
		}},
	}
	for i, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			info := testCase.mkInfo(rand.New(rand.NewSource(int64(1234 + i))))
			depTx, err := L1InfoDeposit(123, info)
			require.NoError(t, err)
			res, err := L1InfoDepositTxData(depTx.Data)
			require.NoError(t, err, "expected valid deposit info")
			assert.Equal(t, res.Number, info.num)
			assert.Equal(t, res.Time, info.time)
			assert.True(t, res.BaseFee.Sign() >= 0)
			assert.Equal(t, res.BaseFee.Bytes(), info.baseFee.Bytes())
			assert.Equal(t, res.BlockHash, info.hash)
		})
	}
	t.Run("no data", func(t *testing.T) {
		_, err := L1InfoDepositTxData(nil)
		assert.Error(t, err)
	})
	t.Run("not enough data", func(t *testing.T) {
		_, err := L1InfoDepositTxData([]byte{1, 2, 3, 4})
		assert.Error(t, err)
	})
	t.Run("too much data", func(t *testing.T) {
		_, err := L1InfoDepositTxData(make([]byte, 4+32+32+32+32+32+1))
		assert.Error(t, err)
	})
}
