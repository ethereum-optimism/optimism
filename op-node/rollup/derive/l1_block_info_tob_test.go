package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum-optimism/optimism/op-node/testutils/fuzzerutils"
	"github.com/ethereum/go-ethereum/common"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
)

// FuzzParseL1InfoDepositTxDataValid is a fuzz test built from TestParseL1InfoDepositTxData, which constructs random
// L1 deposit tx info and derives a tx from it, then derives the info back from the tx, to ensure round-trip
// derivation is upheld. This generates "valid" data and ensures it is always derived back to original values.
func FuzzParseL1InfoDepositTxDataValid(f *testing.F) {
	f.Fuzz(func(t *testing.T, fuzzedData []byte, seqNr uint64, rngSeed int64) {
		// Create our fuzzer wrapper to generate complex values
		typeProvider := fuzz.NewFromGoFuzz(fuzzedData).NilChance(0).MaxDepth(10000).NumElements(0, 0x100)
		fuzzerutils.AddFuzzerFunctions(typeProvider)

		// Generate our fuzzed value types to construct our L1 info
		var fuzzVars struct {
			InfoBaseFee *big.Int
			InfoTime    uint64
			InfoNum     uint64
			// InfoSequenceNumber uint64
		}
		typeProvider.Fuzz(&fuzzVars)

		// Create an rng provider and construct an L1 info from random + fuzzed data.
		rng := rand.New(rand.NewSource(rngSeed))
		// just go instantiate the struct instead of calling MakeL1Info
		// see what has become of it.
		l1Info := testutils.MakeBlockInfo(func(l *testutils.MockBlockInfo) {
			l.InfoBaseFee = fuzzVars.InfoBaseFee
			l.InfoTime = fuzzVars.InfoTime
			l.InfoNum = fuzzVars.InfoNum
		})(rng)

		// Create our deposit tx from our info
		testSysCfg := eth.SystemConfig{
			BatcherAddr: common.Address{42},
			Overhead:    [32]byte{},
			Scalar:      [32]byte{},
		}
		depTx, err := L1InfoDeposit(seqNr, l1Info, testSysCfg)
		require.NoError(t, err)

		// Get our info from out deposit tx
		res, err := L1InfoDepositTxData(depTx.Data)
		require.NoError(t, err, "expected valid deposit info")

		// Verify all parameters match in our round trip deriving operations
		require.Equal(t, res.Number, l1Info.NumberU64())
		require.Equal(t, res.Time, l1Info.Time())
		require.True(t, res.BaseFee.Sign() >= 0)
		require.Equal(t, res.BaseFee.Bytes(), l1Info.BaseFee().Bytes())
		require.Equal(t, res.BlockHash, l1Info.Hash())
		require.Equal(t, res.SequenceNumber, seqNr)
		l1CfgFetcher := &testutils.MockL2Client{}
		l1CfgFetcher.ExpectSystemConfigByL2Hash(res.BlockHash, testSysCfg, nil)
	})
}

// FuzzParseL1InfoDepositTxDataBadLength is a fuzz test built from TestParseL1InfoDepositTxData, which constructs
// random L1 deposit tx info and derives a tx from it, then derives the info back from the tx, to ensure round-trip
// derivation is upheld. This generates "invalid" data and ensures it always throws an error where expected.
func FuzzParseL1InfoDepositTxDataBadLength(f *testing.F) {
	const expectedDepositTxDataLength = 4 + 32 + 32 + 32 + 32 + 32
	f.Fuzz(func(t *testing.T, fuzzedData []byte) {
		// Derive a transaction from random fuzzed data
		_, err := L1InfoDepositTxData(fuzzedData)

		// If the data is null, or too short or too long, we expect an error
		if fuzzedData == nil || len(fuzzedData) != expectedDepositTxDataLength {
			require.Error(t, err)
		}
	})
}
