package derive

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum-optimism/optimism/op-node/testutils/fuzzerutils"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
)

// FuzzParseL1InfoDepositTxDataValid is a fuzz test built from TestParseL1InfoDepositTxData, which constructs random
// L1 deposit tx info and derives a tx from it, then derives the info back from the tx, to ensure round-trip
// derivation is upheld. This generates "valid" data and ensures it is always derived back to original values.
func FuzzParseL1InfoDepositTxDataValid(f *testing.F) {
	f.Fuzz(func(t *testing.T, fuzzedData []byte) {
		// Create our fuzzer wrapper to generate complex values
		typeProvider := fuzz.NewFromGoFuzz(fuzzedData).NilChance(0).MaxDepth(10000).NumElements(0, 0x100)
		fuzzerutils.AddFuzzerFunctions(typeProvider)

		var l1Info testutils.MockBlockInfo
		typeProvider.Fuzz(&l1Info)
		var seqNr uint64
		typeProvider.Fuzz(&seqNr)
		var sysCfg eth.SystemConfig
		typeProvider.Fuzz(&sysCfg)

		// Create our deposit tx from our info
		depTx, err := L1InfoDeposit(seqNr, &l1Info, sysCfg)
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
		require.Equal(t, res.BatcherAddr, sysCfg.BatcherAddr)

		require.Equal(t, res.L1FeeOverhead, sysCfg.Overhead)
		require.Equal(t, res.L1FeeScalar, sysCfg.Scalar)
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
