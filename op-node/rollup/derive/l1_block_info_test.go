package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ eth.L1Info = (*testutils.MockL1Info)(nil)

type infoTest struct {
	name   string
	mkInfo func(rng *rand.Rand) *testutils.MockL1Info
}

var MockDepositContractAddr = common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeef00000000")

func TestParseL1InfoDepositTxData(t *testing.T) {
	// Go 1.18 will have native fuzzing for us to use, until then, we cover just the below cases
	cases := []infoTest{
		{"random", testutils.MakeL1Info(nil)},
		{"zero basefee", testutils.MakeL1Info(func(l *testutils.MockL1Info) {
			l.InfoBaseFee = new(big.Int)
		})},
		{"zero time", testutils.MakeL1Info(func(l *testutils.MockL1Info) {
			l.InfoTime = 0
		})},
		{"zero num", testutils.MakeL1Info(func(l *testutils.MockL1Info) {
			l.InfoNum = 0
		})},
		{"zero seq", testutils.MakeL1Info(func(l *testutils.MockL1Info) {
			l.InfoSequenceNumber = 0
		})},
		{"all zero", func(rng *rand.Rand) *testutils.MockL1Info {
			return &testutils.MockL1Info{InfoBaseFee: new(big.Int)}
		}},
	}
	for i, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			info := testCase.mkInfo(rand.New(rand.NewSource(int64(1234 + i))))
			depTx, err := L1InfoDeposit(info.SequenceNumber(), info)
			require.NoError(t, err)
			res, err := L1InfoDepositTxData(depTx.Data)
			require.NoError(t, err, "expected valid deposit info")
			assert.Equal(t, res.Number, info.NumberU64())
			assert.Equal(t, res.Time, info.Time())
			assert.True(t, res.BaseFee.Sign() >= 0)
			assert.Equal(t, res.BaseFee.Bytes(), info.BaseFee().Bytes())
			assert.Equal(t, res.BlockHash, info.Hash())
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
