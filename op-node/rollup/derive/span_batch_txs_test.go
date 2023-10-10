package derive

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestSpanBatchTxsContractCreationBits(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1234567))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	contractCreationBits := rawSpanBatch.txs.contractCreationBits
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.contractCreationBits = contractCreationBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err := sbt.encodeContractCreationBits(&buf)
	assert.NoError(t, err)

	// contractCreationBit field is fixed length: single bit
	contractCreationBitBufferLen := totalBlockTxCount / 8
	if totalBlockTxCount%8 != 0 {
		contractCreationBitBufferLen++
	}
	assert.Equal(t, buf.Len(), int(contractCreationBitBufferLen))

	result := buf.Bytes()
	sbt.contractCreationBits = nil

	r := bytes.NewReader(result)
	err = sbt.decodeContractCreationBits(r)
	assert.NoError(t, err)

	assert.Equal(t, contractCreationBits, sbt.contractCreationBits)
}

func TestSpanBatchTxsContractCreationCount(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1337))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	contractCreationBits := rawSpanBatch.txs.contractCreationBits
	contractCreationCount := rawSpanBatch.txs.contractCreationCount()
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.contractCreationBits = contractCreationBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err := sbt.encodeContractCreationBits(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	sbt.contractCreationBits = nil

	r := bytes.NewReader(result)
	err = sbt.decodeContractCreationBits(r)
	assert.NoError(t, err)

	assert.Equal(t, contractCreationCount, sbt.contractCreationCount())
}

func TestSpanBatchTxsYParityBits(t *testing.T) {
	rng := rand.New(rand.NewSource(0x7331))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	yParityBits := rawSpanBatch.txs.yParityBits
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.yParityBits = yParityBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err := sbt.encodeYParityBits(&buf)
	assert.NoError(t, err)

	// yParityBit field is fixed length: single bit
	yParityBitBufferLen := totalBlockTxCount / 8
	if totalBlockTxCount%8 != 0 {
		yParityBitBufferLen++
	}
	assert.Equal(t, buf.Len(), int(yParityBitBufferLen))

	result := buf.Bytes()
	sbt.yParityBits = nil

	r := bytes.NewReader(result)
	err = sbt.decodeYParityBits(r)
	assert.NoError(t, err)

	assert.Equal(t, yParityBits, sbt.yParityBits)
}

func TestSpanBatchTxsTxSigs(t *testing.T) {
	rng := rand.New(rand.NewSource(0x73311337))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txSigs := rawSpanBatch.txs.txSigs
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount
	sbt.txSigs = txSigs

	var buf bytes.Buffer
	err := sbt.encodeTxSigsRS(&buf)
	assert.NoError(t, err)

	// txSig field is fixed length: 32 byte + 32 byte = 64 byte
	assert.Equal(t, buf.Len(), 64*int(totalBlockTxCount))

	result := buf.Bytes()
	sbt.txSigs = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxSigsRS(r)
	assert.NoError(t, err)

	// v field is not set
	for i := 0; i < int(totalBlockTxCount); i++ {
		assert.Equal(t, txSigs[i].r, sbt.txSigs[i].r)
		assert.Equal(t, txSigs[i].s, sbt.txSigs[i].s)
	}
}

func TestSpanBatchTxsTxNonces(t *testing.T) {
	rng := rand.New(rand.NewSource(0x123456))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txNonces := rawSpanBatch.txs.txNonces
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount
	sbt.txNonces = txNonces

	var buf bytes.Buffer
	err := sbt.encodeTxNonces(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	sbt.txNonces = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxNonces(r)
	assert.NoError(t, err)

	assert.Equal(t, txNonces, sbt.txNonces)
}

func TestSpanBatchTxsTxGases(t *testing.T) {
	rng := rand.New(rand.NewSource(0x12345))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txGases := rawSpanBatch.txs.txGases
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount
	sbt.txGases = txGases

	var buf bytes.Buffer
	err := sbt.encodeTxGases(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	sbt.txGases = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxGases(r)
	assert.NoError(t, err)

	assert.Equal(t, txGases, sbt.txGases)
}

func TestSpanBatchTxsTxTos(t *testing.T) {
	rng := rand.New(rand.NewSource(0x54321))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txTos := rawSpanBatch.txs.txTos
	contractCreationBits := rawSpanBatch.txs.contractCreationBits
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.txTos = txTos
	// creation bits and block tx count must be se to decode tos
	sbt.contractCreationBits = contractCreationBits
	sbt.totalBlockTxCount = totalBlockTxCount

	var buf bytes.Buffer
	err := sbt.encodeTxTos(&buf)
	assert.NoError(t, err)

	// to field is fixed length: 20 bytes
	assert.Equal(t, buf.Len(), 20*len(txTos))

	result := buf.Bytes()
	sbt.txTos = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxTos(r)
	assert.NoError(t, err)

	assert.Equal(t, txTos, sbt.txTos)
}

func TestSpanBatchTxsTxDatas(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1234))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	txDatas := rawSpanBatch.txs.txDatas
	txTypes := rawSpanBatch.txs.txTypes
	totalBlockTxCount := rawSpanBatch.txs.totalBlockTxCount

	var sbt spanBatchTxs
	sbt.totalBlockTxCount = totalBlockTxCount

	sbt.txDatas = txDatas

	var buf bytes.Buffer
	err := sbt.encodeTxDatas(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	sbt.txDatas = nil
	sbt.txTypes = nil

	r := bytes.NewReader(result)
	err = sbt.decodeTxDatas(r)
	assert.NoError(t, err)

	assert.Equal(t, txDatas, sbt.txDatas)
	assert.Equal(t, txTypes, sbt.txTypes)
}

func TestSpanBatchTxsRecoverV(t *testing.T) {
	rng := rand.New(rand.NewSource(0x123))

	chainID := big.NewInt(rng.Int63n(1000))
	signer := types.NewLondonSigner(chainID)
	totalblockTxCount := rng.Intn(100)

	var spanBatchTxs spanBatchTxs
	var txTypes []int
	var txSigs []spanBatchSignature
	var originalVs []uint64
	yParityBits := new(big.Int)
	for idx := 0; idx < totalblockTxCount; idx++ {
		tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
		txTypes = append(txTypes, int(tx.Type()))
		var txSig spanBatchSignature
		v, r, s := tx.RawSignatureValues()
		// Do not fill in txSig.V
		txSig.r, _ = uint256.FromBig(r)
		txSig.s, _ = uint256.FromBig(s)
		txSigs = append(txSigs, txSig)
		originalVs = append(originalVs, v.Uint64())
		yParityBit := convertVToYParity(v.Uint64(), int(tx.Type()))
		yParityBits.SetBit(yParityBits, idx, yParityBit)
	}

	spanBatchTxs.yParityBits = yParityBits
	spanBatchTxs.txSigs = txSigs
	spanBatchTxs.txTypes = txTypes
	// recover txSig.v
	spanBatchTxs.recoverV(chainID)

	var recoveredVs []uint64
	for _, txSig := range spanBatchTxs.txSigs {
		recoveredVs = append(recoveredVs, txSig.v)
	}

	assert.Equal(t, originalVs, recoveredVs, "recovered v mismatch")
}

func TestSpanBatchTxsRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(0x73311337))
	chainID := big.NewInt(rng.Int63n(1000))

	for i := 0; i < 4; i++ {
		rawSpanBatch := RandomRawSpanBatch(rng, chainID)
		sbt := rawSpanBatch.txs
		totalBlockTxCount := sbt.totalBlockTxCount

		var buf bytes.Buffer
		err := sbt.encode(&buf)
		assert.NoError(t, err)

		result := buf.Bytes()
		r := bytes.NewReader(result)

		var sbt2 spanBatchTxs
		sbt2.totalBlockTxCount = totalBlockTxCount
		err = sbt2.decode(r)
		assert.NoError(t, err)
		sbt2.recoverV(chainID)

		assert.Equal(t, sbt, &sbt2)
	}
}

func TestSpanBatchTxsRoundTripFullTxs(t *testing.T) {
	rng := rand.New(rand.NewSource(0x13377331))
	chainID := big.NewInt(rng.Int63n(1000))
	signer := types.NewLondonSigner(chainID)

	for i := 0; i < 4; i++ {
		totalblockTxCounts := uint64(1 + rng.Int()&0xFF)
		var txs [][]byte
		for i := 0; i < int(totalblockTxCounts); i++ {
			tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
			rawTx, err := tx.MarshalBinary()
			assert.NoError(t, err)
			txs = append(txs, rawTx)
		}
		sbt, err := newSpanBatchTxs(txs, chainID)
		assert.NoError(t, err)

		txs2, err := sbt.fullTxs(chainID)
		assert.NoError(t, err)

		assert.Equal(t, txs, txs2)
	}
}

func TestSpanBatchTxsRecoverVInvalidTxType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	rng := rand.New(rand.NewSource(0x321))
	chainID := big.NewInt(rng.Int63n(1000))

	var sbt spanBatchTxs

	sbt.txTypes = []int{types.DepositTxType}
	sbt.txSigs = []spanBatchSignature{{v: 0, r: nil, s: nil}}
	sbt.yParityBits = new(big.Int)

	// expect panic
	sbt.recoverV(chainID)
}

func TestSpanBatchTxsFullTxNotEnoughTxTos(t *testing.T) {
	rng := rand.New(rand.NewSource(0x13572468))
	chainID := big.NewInt(rng.Int63n(1000))
	signer := types.NewLondonSigner(chainID)

	totalblockTxCounts := uint64(1 + rng.Int()&0xFF)
	var txs [][]byte
	for i := 0; i < int(totalblockTxCounts); i++ {
		tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
		rawTx, err := tx.MarshalBinary()
		assert.NoError(t, err)
		txs = append(txs, rawTx)
	}
	sbt, err := newSpanBatchTxs(txs, chainID)
	assert.NoError(t, err)

	// drop single to field
	sbt.txTos = sbt.txTos[:len(sbt.txTos)-2]

	_, err = sbt.fullTxs(chainID)
	assert.EqualError(t, err, "tx to not enough")
}

func TestSpanBatchTxsMaxContractCreationBitsLength(t *testing.T) {
	var sbt spanBatchTxs
	sbt.totalBlockTxCount = 0xFFFFFFFFFFFFFFFF

	r := bytes.NewReader([]byte{})
	err := sbt.decodeContractCreationBits(r)
	assert.ErrorIs(t, err, ErrTooBigSpanBatchFieldSize)
}

func TestSpanBatchTxsMaxYParityBitsLength(t *testing.T) {
	var sb RawSpanBatch
	sb.blockCount = 0xFFFFFFFFFFFFFFFF

	r := bytes.NewReader([]byte{})
	err := sb.decodeOriginBits(r)
	assert.ErrorIs(t, err, ErrTooBigSpanBatchFieldSize)
}
