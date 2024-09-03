package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spanBatchTxTest struct {
	name      string
	trials    int
	mkTx      func(rng *rand.Rand, signer types.Signer) *types.Transaction
	protected bool
}

func TestSpanBatchTxConvert(t *testing.T) {
	cases := []spanBatchTxTest{
		{"unprotected legacy tx", 32, testutils.RandomLegacyTx, false},
		{"legacy tx", 32, testutils.RandomLegacyTx, true},
		{"access list tx", 32, testutils.RandomAccessListTx, true},
		{"dynamic fee tx", 32, testutils.RandomDynamicFeeTx, true},
	}

	for i, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(0x1331 + i)))
			chainID := big.NewInt(rng.Int63n(1000))
			signer := types.NewLondonSigner(chainID)
			if !testCase.protected {
				signer = types.HomesteadSigner{}
			}

			for txIdx := 0; txIdx < testCase.trials; txIdx++ {
				tx := testCase.mkTx(rng, signer)

				v, r, s := tx.RawSignatureValues()
				sbtx, err := newSpanBatchTx(tx)
				require.NoError(t, err)

				tx2, err := sbtx.convertToFullTx(tx.Nonce(), tx.Gas(), tx.To(), chainID, v, r, s)
				require.NoError(t, err)

				// compare after marshal because we only need inner field of transaction
				txEncoded, err := tx.MarshalBinary()
				require.NoError(t, err)
				tx2Encoded, err := tx2.MarshalBinary()
				require.NoError(t, err)

				assert.Equal(t, txEncoded, tx2Encoded)
			}
		})
	}
}

func TestSpanBatchTxRoundTrip(t *testing.T) {
	cases := []spanBatchTxTest{
		{"unprotected legacy tx", 32, testutils.RandomLegacyTx, false},
		{"legacy tx", 32, testutils.RandomLegacyTx, true},
		{"access list tx", 32, testutils.RandomAccessListTx, true},
		{"dynamic fee tx", 32, testutils.RandomDynamicFeeTx, true},
	}

	for i, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(0x1332 + i)))
			chainID := big.NewInt(rng.Int63n(1000))
			signer := types.NewLondonSigner(chainID)
			if !testCase.protected {
				signer = types.HomesteadSigner{}
			}

			for txIdx := 0; txIdx < testCase.trials; txIdx++ {
				tx := testCase.mkTx(rng, signer)

				sbtx, err := newSpanBatchTx(tx)
				require.NoError(t, err)

				sbtxEncoded, err := sbtx.MarshalBinary()
				require.NoError(t, err)

				var sbtx2 spanBatchTx
				err = sbtx2.UnmarshalBinary(sbtxEncoded)
				require.NoError(t, err)

				assert.Equal(t, sbtx, &sbtx2)
			}
		})
	}
}

type spanBatchDummyTxData struct{}

func (txData *spanBatchDummyTxData) txType() byte { return types.DepositTxType }

func TestSpanBatchTxInvalidTxType(t *testing.T) {
	// span batch never contain deposit tx
	depositTx := types.NewTx(&types.DepositTx{})
	_, err := newSpanBatchTx(depositTx)
	require.ErrorContains(t, err, "invalid tx type")

	var sbtx spanBatchTx
	sbtx.inner = &spanBatchDummyTxData{}
	_, err = sbtx.convertToFullTx(0, 0, nil, nil, nil, nil, nil)
	require.ErrorContains(t, err, "invalid tx type")
}

func TestSpanBatchTxDecodeInvalid(t *testing.T) {
	var sbtx spanBatchTx
	_, err := sbtx.decodeTyped([]byte{})
	require.ErrorIs(t, err, ErrTypedTxTooShort)

	tx := types.NewTx(&types.LegacyTx{})
	txEncoded, err := tx.MarshalBinary()
	require.NoError(t, err)

	// legacy tx is not typed tx
	_, err = sbtx.decodeTyped(txEncoded)
	require.EqualError(t, err, types.ErrTxTypeNotSupported.Error())

	tx2 := types.NewTx(&types.AccessListTx{})
	tx2Encoded, err := tx2.MarshalBinary()
	require.NoError(t, err)

	tx2Encoded[0] = types.DynamicFeeTxType
	_, err = sbtx.decodeTyped(tx2Encoded)
	require.ErrorContains(t, err, "failed to decode spanBatchDynamicFeeTxData")

	tx3 := types.NewTx(&types.DynamicFeeTx{})
	tx3Encoded, err := tx3.MarshalBinary()
	require.NoError(t, err)

	tx3Encoded[0] = types.AccessListTxType
	_, err = sbtx.decodeTyped(tx3Encoded)
	require.ErrorContains(t, err, "failed to decode spanBatchAccessListTxData")

	invalidLegacyTxDecoded := []byte{0xFF, 0xFF}
	err = sbtx.UnmarshalBinary(invalidLegacyTxDecoded)
	require.ErrorContains(t, err, "failed to decode spanBatchLegacyTxData")
}
