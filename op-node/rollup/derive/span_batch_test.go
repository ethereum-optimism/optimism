package derive

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

func TestSpanBatchForBatchInterface(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5432177))
	chainID := big.NewInt(rng.Int63n(1000))

	singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
	blockCount := len(singularBatches)
	safeL2Head := testutils.RandomL2BlockRef(rng)
	safeL2Head.Hash = common.BytesToHash(singularBatches[0].ParentHash[:])

	spanBatch := NewSpanBatch(singularBatches)

	// check interface method implementations except logging
	assert.Equal(t, SpanBatchType, spanBatch.GetBatchType())
	assert.Equal(t, singularBatches[0].Timestamp, spanBatch.GetTimestamp())
	assert.Equal(t, singularBatches[0].EpochNum, spanBatch.GetStartEpochNum())
	assert.True(t, spanBatch.CheckOriginHash(singularBatches[blockCount-1].EpochHash))
	assert.True(t, spanBatch.CheckParentHash(singularBatches[0].ParentHash))
}

func TestSpanBatchOriginBits(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77665544))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	blockCount := rawSpanBatch.blockCount

	var buf bytes.Buffer
	err := rawSpanBatch.encodeOriginBits(&buf)
	assert.NoError(t, err)

	// originBit field is fixed length: single bit
	originBitBufferLen := blockCount / 8
	if blockCount%8 != 0 {
		originBitBufferLen++
	}
	assert.Equal(t, buf.Len(), int(originBitBufferLen))

	result := buf.Bytes()
	var sb RawSpanBatch
	sb.blockCount = blockCount
	r := bytes.NewReader(result)
	err = sb.decodeOriginBits(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch.originBits, sb.originBits)
}

func TestSpanBatchPrefix(t *testing.T) {
	rng := rand.New(rand.NewSource(0x44775566))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	// only compare prefix
	rawSpanBatch.spanBatchPayload = spanBatchPayload{}

	var buf bytes.Buffer
	err := rawSpanBatch.encodePrefix(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodePrefix(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch, &sb)
}

func TestSpanBatchRelTimestamp(t *testing.T) {
	rng := rand.New(rand.NewSource(0x44775566))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeRelTimestamp(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeRelTimestamp(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch.relTimestamp, sb.relTimestamp)
}

func TestSpanBatchL1OriginNum(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556688))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeL1OriginNum(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeL1OriginNum(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch.l1OriginNum, sb.l1OriginNum)
}

func TestSpanBatchParentCheck(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556689))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeParentCheck(&buf)
	assert.NoError(t, err)

	// parent check field is fixed length: 20 bytes
	assert.Equal(t, buf.Len(), 20)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeParentCheck(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch.parentCheck, sb.parentCheck)
}

func TestSpanBatchL1OriginCheck(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556690))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeL1OriginCheck(&buf)
	assert.NoError(t, err)

	// l1 origin check field is fixed length: 20 bytes
	assert.Equal(t, buf.Len(), 20)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeL1OriginCheck(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch.l1OriginCheck, sb.l1OriginCheck)
}

func TestSpanBatchPayload(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556691))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodePayload(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	err = sb.decodePayload(r)
	assert.NoError(t, err)

	sb.txs.recoverV(chainID)

	assert.Equal(t, rawSpanBatch.spanBatchPayload, sb.spanBatchPayload)
}

func TestSpanBatchBlockCount(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556691))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeBlockCount(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	err = sb.decodeBlockCount(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch.blockCount, sb.blockCount)
}

func TestSpanBatchBlockTxCounts(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556692))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeBlockTxCounts(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	sb.blockCount = rawSpanBatch.blockCount
	err = sb.decodeBlockTxCounts(r)
	assert.NoError(t, err)

	assert.Equal(t, rawSpanBatch.blockTxCounts, sb.blockTxCounts)
}

func TestSpanBatchTxs(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556693))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeTxs(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	sb.blockTxCounts = rawSpanBatch.blockTxCounts
	err = sb.decodeTxs(r)
	assert.NoError(t, err)

	sb.txs.recoverV(chainID)

	assert.Equal(t, rawSpanBatch.txs, sb.txs)
}

func TestSpanBatchRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556694))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	result, err := rawSpanBatch.encodeBytes()
	assert.NoError(t, err)

	var sb RawSpanBatch
	err = sb.decodeBytes(result)
	assert.NoError(t, err)

	sb.txs.recoverV(chainID)

	assert.Equal(t, rawSpanBatch, &sb)
}

func TestSpanBatchDerive(t *testing.T) {
	rng := rand.New(rand.NewSource(0xbab0bab0))

	chainID := new(big.Int).SetUint64(rng.Uint64())
	l2BlockTime := uint64(2)

	for originChangedBit := 0; originChangedBit < 2; originChangedBit++ {
		singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
		safeL2Head := testutils.RandomL2BlockRef(rng)
		safeL2Head.Hash = common.BytesToHash(singularBatches[0].ParentHash[:])
		genesisTimeStamp := 1 + singularBatches[0].Timestamp - 128

		spanBatch := NewSpanBatch(singularBatches)
		originChangedBit := uint(originChangedBit)
		rawSpanBatch, err := spanBatch.ToRawSpanBatch(originChangedBit, genesisTimeStamp, chainID)
		assert.NoError(t, err)

		spanBatchDerived, err := rawSpanBatch.derive(l2BlockTime, genesisTimeStamp, chainID)
		assert.NoError(t, err)

		blockCount := len(singularBatches)
		assert.Equal(t, safeL2Head.Hash.Bytes()[:20], spanBatchDerived.parentCheck)
		assert.Equal(t, singularBatches[blockCount-1].Epoch().Hash.Bytes()[:20], spanBatchDerived.l1OriginCheck)
		assert.Equal(t, len(singularBatches), int(rawSpanBatch.blockCount))

		for i := 1; i < len(singularBatches); i++ {
			assert.Equal(t, spanBatchDerived.batches[i].Timestamp, spanBatchDerived.batches[i-1].Timestamp+l2BlockTime)
		}

		for i := 0; i < len(singularBatches); i++ {
			assert.Equal(t, singularBatches[i].EpochNum, spanBatchDerived.batches[i].EpochNum)
			assert.Equal(t, singularBatches[i].Timestamp, spanBatchDerived.batches[i].Timestamp)
			assert.Equal(t, singularBatches[i].Transactions, spanBatchDerived.batches[i].Transactions)
		}
	}
}

func TestSpanBatchAppend(t *testing.T) {
	rng := rand.New(rand.NewSource(0x44337711))

	chainID := new(big.Int).SetUint64(rng.Uint64())

	singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
	// initialize empty span batch
	spanBatch := NewSpanBatch([]*SingularBatch{})

	L := 2
	for i := 0; i < L; i++ {
		spanBatch.AppendSingularBatch(singularBatches[i])
	}
	// initialize with two singular batches
	spanBatch2 := NewSpanBatch(singularBatches[:L])

	assert.Equal(t, spanBatch, spanBatch2)
}

func TestSpanBatchMerge(t *testing.T) {
	rng := rand.New(rand.NewSource(0x73314433))

	genesisTimeStamp := rng.Uint64()
	chainID := new(big.Int).SetUint64(rng.Uint64())

	for originChangedBit := 0; originChangedBit < 2; originChangedBit++ {
		singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
		blockCount := len(singularBatches)

		spanBatch := NewSpanBatch(singularBatches)
		originChangedBit := uint(originChangedBit)
		rawSpanBatch, err := spanBatch.ToRawSpanBatch(originChangedBit, genesisTimeStamp, chainID)
		assert.NoError(t, err)

		// check span batch prefix
		assert.Equal(t, rawSpanBatch.relTimestamp, singularBatches[0].Timestamp-genesisTimeStamp, "invalid relative timestamp")
		assert.Equal(t, rollup.Epoch(rawSpanBatch.l1OriginNum), singularBatches[blockCount-1].EpochNum)
		assert.Equal(t, rawSpanBatch.parentCheck, singularBatches[0].ParentHash.Bytes()[:20], "invalid parent check")
		assert.Equal(t, rawSpanBatch.l1OriginCheck, singularBatches[blockCount-1].EpochHash.Bytes()[:20], "invalid l1 origin check")

		// check span batch payload
		assert.Equal(t, int(rawSpanBatch.blockCount), len(singularBatches))
		assert.Equal(t, rawSpanBatch.originBits.Bit(0), originChangedBit)
		for i := 1; i < blockCount; i++ {
			if rawSpanBatch.originBits.Bit(i) == 1 {
				assert.Equal(t, singularBatches[i].EpochNum, singularBatches[i-1].EpochNum+1)
			} else {
				assert.Equal(t, singularBatches[i].EpochNum, singularBatches[i-1].EpochNum)
			}
		}
		for i := 0; i < len(singularBatches); i++ {
			txCount := len(singularBatches[i].Transactions)
			assert.Equal(t, txCount, int(rawSpanBatch.blockTxCounts[i]))
		}

		// check invariants
		endEpochNum := rawSpanBatch.l1OriginNum
		assert.Equal(t, endEpochNum, uint64(singularBatches[blockCount-1].EpochNum))

		// we do not check txs field because it has to be derived to be compared
	}
}

func TestSpanBatchToSingularBatch(t *testing.T) {
	rng := rand.New(rand.NewSource(0xbab0bab1))
	chainID := new(big.Int).SetUint64(rng.Uint64())

	for originChangedBit := 0; originChangedBit < 2; originChangedBit++ {
		singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
		safeL2Head := testutils.RandomL2BlockRef(rng)
		safeL2Head.Hash = common.BytesToHash(singularBatches[0].ParentHash[:])
		safeL2Head.Time = singularBatches[0].Timestamp - 2
		genesisTimeStamp := 1 + singularBatches[0].Timestamp - 128

		spanBatch := NewSpanBatch(singularBatches)
		originChangedBit := uint(originChangedBit)
		rawSpanBatch, err := spanBatch.ToRawSpanBatch(originChangedBit, genesisTimeStamp, chainID)
		assert.NoError(t, err)

		l1Origins := mockL1Origin(rng, rawSpanBatch, singularBatches)

		singularBatches2, err := spanBatch.GetSingularBatches(l1Origins, safeL2Head)
		assert.NoError(t, err)

		// GetSingularBatches does not fill in parent hash of singular batches
		// empty out parent hash for comparison
		for i := 0; i < len(singularBatches); i++ {
			singularBatches[i].ParentHash = common.Hash{}
		}
		// check parent hash is empty
		for i := 0; i < len(singularBatches2); i++ {
			assert.Equal(t, singularBatches2[i].ParentHash, common.Hash{})
		}

		assert.Equal(t, singularBatches, singularBatches2)
	}
}

func TestSpanBatchReadTxData(t *testing.T) {
	rng := rand.New(rand.NewSource(0x109550))
	chainID := new(big.Int).SetUint64(rng.Uint64())

	txCount := 64

	signer := types.NewLondonSigner(chainID)
	var rawTxs [][]byte
	var txs []*types.Transaction
	m := make(map[byte]int)
	for i := 0; i < txCount; i++ {
		tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
		m[tx.Type()] += 1
		rawTx, err := tx.MarshalBinary()
		assert.NoError(t, err)
		rawTxs = append(rawTxs, rawTx)
		txs = append(txs, tx)
	}

	for i := 0; i < txCount; i++ {
		r := bytes.NewReader(rawTxs[i])
		_, txType, err := ReadTxData(r)
		assert.NoError(t, err)
		assert.Equal(t, int(txs[i].Type()), txType)
	}
	// make sure every tx type is tested
	assert.Positive(t, m[types.LegacyTxType])
	assert.Positive(t, m[types.AccessListTxType])
	assert.Positive(t, m[types.DynamicFeeTxType])
}

func TestSpanBatchReadTxDataInvalid(t *testing.T) {
	dummy, err := rlp.EncodeToBytes("dummy")
	assert.NoError(t, err)

	// test non list rlp decoding
	r := bytes.NewReader(dummy)
	_, _, err = ReadTxData(r)
	assert.ErrorContains(t, err, "tx RLP prefix type must be list")
}

func TestSpanBatchBuilder(t *testing.T) {
	rng := rand.New(rand.NewSource(0xbab1bab1))
	chainID := new(big.Int).SetUint64(rng.Uint64())

	for originChangedBit := 0; originChangedBit < 2; originChangedBit++ {
		singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
		safeL2Head := testutils.RandomL2BlockRef(rng)
		if originChangedBit == 0 {
			safeL2Head.Hash = common.BytesToHash(singularBatches[0].ParentHash[:])
		}
		genesisTimeStamp := 1 + singularBatches[0].Timestamp - 128

		parentEpoch := uint64(singularBatches[0].EpochNum)
		if originChangedBit == 1 {
			parentEpoch -= 1
		}
		spanBatchBuilder := NewSpanBatchBuilder(parentEpoch, genesisTimeStamp, chainID)

		assert.Equal(t, 0, spanBatchBuilder.GetBlockCount())

		for i := 0; i < len(singularBatches); i++ {
			spanBatchBuilder.AppendSingularBatch(singularBatches[i])
			assert.Equal(t, i+1, spanBatchBuilder.GetBlockCount())
			assert.Equal(t, singularBatches[0].ParentHash.Bytes()[:20], spanBatchBuilder.spanBatch.parentCheck)
			assert.Equal(t, singularBatches[i].EpochHash.Bytes()[:20], spanBatchBuilder.spanBatch.l1OriginCheck)
		}

		rawSpanBatch, err := spanBatchBuilder.GetRawSpanBatch()
		assert.NoError(t, err)

		// compare with rawSpanBatch not using spanBatchBuilder
		spanBatch := NewSpanBatch(singularBatches)
		originChangedBit := uint(originChangedBit)
		rawSpanBatch2, err := spanBatch.ToRawSpanBatch(originChangedBit, genesisTimeStamp, chainID)
		assert.NoError(t, err)

		assert.Equal(t, rawSpanBatch2, rawSpanBatch)

		spanBatchBuilder.Reset()
		assert.Equal(t, 0, spanBatchBuilder.GetBlockCount())
	}
}

func TestSpanBatchMaxTxData(t *testing.T) {
	rng := rand.New(rand.NewSource(0x177288))

	invalidTx := types.NewTx(&types.DynamicFeeTx{
		Data: testutils.RandomData(rng, MaxSpanBatchFieldSize+1),
	})

	txEncoded, err := invalidTx.MarshalBinary()
	assert.NoError(t, err)

	r := bytes.NewReader(txEncoded)
	_, _, err = ReadTxData(r)

	assert.ErrorIs(t, err, ErrTooBigSpanBatchFieldSize)
}

func TestSpanBatchMaxOriginBitsLength(t *testing.T) {
	var sb RawSpanBatch
	sb.blockCount = 0xFFFFFFFFFFFFFFFF

	r := bytes.NewReader([]byte{})
	err := sb.decodeOriginBits(r)
	assert.ErrorIs(t, err, ErrTooBigSpanBatchFieldSize)
}
