package derive

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

func TestSpanBatchTxConvert(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1331))
	chainID := big.NewInt(rng.Int63n(1000))
	signer := types.NewLondonSigner(chainID)

	m := make(map[byte]int)
	for i := 0; i < 32; i++ {
		tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
		m[tx.Type()] += 1
		v, r, s := tx.RawSignatureValues()
		sbtx, err := newSpanBatchTx(*tx)
		assert.NoError(t, err)

		tx2, err := sbtx.convertToFullTx(tx.Nonce(), tx.Gas(), tx.To(), chainID, v, r, s)
		assert.NoError(t, err)

		// compare after marshal because we only need inner field of transaction
		txEncoded, err := tx.MarshalBinary()
		assert.NoError(t, err)
		tx2Encoded, err := tx2.MarshalBinary()
		assert.NoError(t, err)

		assert.Equal(t, txEncoded, tx2Encoded)
	}
	// make sure every tx type is tested
	assert.Positive(t, m[types.LegacyTxType])
	assert.Positive(t, m[types.AccessListTxType])
	assert.Positive(t, m[types.DynamicFeeTxType])
}

func TestSpanBatchTxRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1332))
	chainID := big.NewInt(rng.Int63n(1000))
	signer := types.NewLondonSigner(chainID)

	m := make(map[byte]int)
	for i := 0; i < 32; i++ {
		tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
		m[tx.Type()] += 1
		sbtx, err := newSpanBatchTx(*tx)
		assert.NoError(t, err)

		sbtxEncoded, err := sbtx.MarshalBinary()
		assert.NoError(t, err)

		var sbtx2 spanBatchTx
		err = sbtx2.UnmarshalBinary(sbtxEncoded)
		assert.NoError(t, err)

		assert.Equal(t, sbtx, &sbtx2)
	}
	// make sure every tx type is tested
	assert.Positive(t, m[types.LegacyTxType])
	assert.Positive(t, m[types.AccessListTxType])
	assert.Positive(t, m[types.DynamicFeeTxType])
}

func TestSpanBatchTxRoundTripRLP(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1333))
	chainID := big.NewInt(rng.Int63n(1000))
	signer := types.NewLondonSigner(chainID)

	m := make(map[byte]int)
	for i := 0; i < 32; i++ {
		tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
		m[tx.Type()] += 1
		sbtx, err := newSpanBatchTx(*tx)
		assert.NoError(t, err)

		var buf bytes.Buffer
		err = sbtx.EncodeRLP(&buf)
		assert.NoError(t, err)

		result := buf.Bytes()
		var sbtx2 spanBatchTx
		r := bytes.NewReader(result)
		rlpReader := rlp.NewStream(r, 0)
		err = sbtx2.DecodeRLP(rlpReader)
		assert.NoError(t, err)

		assert.Equal(t, sbtx, &sbtx2)
	}
	// make sure every tx type is tested
	assert.Positive(t, m[types.LegacyTxType])
	assert.Positive(t, m[types.AccessListTxType])
	assert.Positive(t, m[types.DynamicFeeTxType])
}

type spanBatchDummyTxData struct{}

func (txData *spanBatchDummyTxData) txType() byte { return types.DepositTxType }
func TestSpanBatchTxInvalidTxType(t *testing.T) {
	// span batch never contain deposit tx
	depositTx := types.NewTx(&types.DepositTx{})
	_, err := newSpanBatchTx(*depositTx)
	assert.ErrorContains(t, err, "invalid tx type")

	var sbtx spanBatchTx
	sbtx.inner = &spanBatchDummyTxData{}
	_, err = sbtx.convertToFullTx(0, 0, nil, nil, nil, nil, nil)
	assert.ErrorContains(t, err, "invalid tx type")
}

func TestSpanBatchTxDecodeInvalid(t *testing.T) {
	var sbtx spanBatchTx
	_, err := sbtx.decodeTyped([]byte{})
	assert.EqualError(t, err, "typed transaction too short")

	tx := types.NewTx(&types.LegacyTx{})
	txEncoded, err := tx.MarshalBinary()
	assert.NoError(t, err)

	// legacy tx is not typed tx
	_, err = sbtx.decodeTyped(txEncoded)
	assert.EqualError(t, err, types.ErrTxTypeNotSupported.Error())

	tx2 := types.NewTx(&types.AccessListTx{})
	tx2Encoded, err := tx2.MarshalBinary()
	assert.NoError(t, err)

	tx2Encoded[0] = types.DynamicFeeTxType
	_, err = sbtx.decodeTyped(tx2Encoded)
	assert.ErrorContains(t, err, "failed to decode spanBatchDynamicFeeTxData")

	tx3 := types.NewTx(&types.DynamicFeeTx{})
	tx3Encoded, err := tx3.MarshalBinary()
	assert.NoError(t, err)

	tx3Encoded[0] = types.AccessListTxType
	_, err = sbtx.decodeTyped(tx3Encoded)
	assert.ErrorContains(t, err, "failed to decode spanBatchAccessListTxData")

	invalidLegacyTxDecoded := []byte{0xFF, 0xFF}
	err = sbtx.UnmarshalBinary(invalidLegacyTxDecoded)
	assert.ErrorContains(t, err, "failed to decode spanBatchLegacyTxData")
}
