package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
)

var _ eth.BlockInfo = (*testutils.MockBlockInfo)(nil)

type infoTest struct {
	name   string
	mkInfo func(rng *rand.Rand) *testutils.MockBlockInfo
	seqNr  func(rng *rand.Rand) uint64
}

var MockDepositContractAddr = common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeef00000000")

func TestParseL1InfoDepositTxData(t *testing.T) {
	randomSeqNr := func(rng *rand.Rand) uint64 {
		return rng.Uint64()
	}
	// Go 1.18 will have native fuzzing for us to use, until then, we cover just the below cases
	cases := []infoTest{
		{"random", testutils.MakeBlockInfo(nil), randomSeqNr},
		{"zero basefee", testutils.MakeBlockInfo(func(l *testutils.MockBlockInfo) {
			l.InfoBaseFee = new(big.Int)
		}), randomSeqNr},
		{"zero time", testutils.MakeBlockInfo(func(l *testutils.MockBlockInfo) {
			l.InfoTime = 0
		}), randomSeqNr},
		{"zero num", testutils.MakeBlockInfo(func(l *testutils.MockBlockInfo) {
			l.InfoNum = 0
		}), randomSeqNr},
		{"zero seq", testutils.MakeBlockInfo(nil), func(rng *rand.Rand) uint64 {
			return 0
		}},
		{"all zero", func(rng *rand.Rand) *testutils.MockBlockInfo {
			return &testutils.MockBlockInfo{InfoBaseFee: new(big.Int)}
		}, func(rng *rand.Rand) uint64 {
			return 0
		}},
	}
	for i, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(1234 + i)))
			info := testCase.mkInfo(rng)
			seqNr := testCase.seqNr(rng)
			depTx, err := L1InfoDeposit(seqNr, info)
			require.NoError(t, err)
			res, err := L1InfoDepositTxData(depTx.Data)
			require.NoError(t, err, "expected valid deposit info")
			assert.Equal(t, res.Number, info.NumberU64())
			assert.Equal(t, res.Time, info.Time())
			assert.True(t, res.BaseFee.Sign() >= 0)
			assert.Equal(t, res.BaseFee.Bytes(), info.BaseFee().Bytes())
			assert.Equal(t, res.BlockHash, info.Hash())
			assert.Equal(t, res.SequenceNumber, seqNr)
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
