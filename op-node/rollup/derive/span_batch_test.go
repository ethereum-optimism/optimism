package derive

import (
	"bytes"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// initializedSpanBatch creates a new SpanBatch with given SingularBatches.
// It is used *only* in tests to create a SpanBatch with given SingularBatches as a convenience.
// It will also ignore any errors that occur during AppendSingularBatch.
// Tests should manually set the first bit of the originBits if needed using SetFirstOriginChangedBit
func initializedSpanBatch(singularBatches []*SingularBatch, genesisTimestamp uint64, chainID *big.Int) *SpanBatch {
	spanBatch := NewSpanBatch(genesisTimestamp, chainID)
	if len(singularBatches) == 0 {
		return spanBatch
	}
	for i := 0; i < len(singularBatches); i++ {
		if err := spanBatch.AppendSingularBatch(singularBatches[i], uint64(i)); err != nil {
			continue
		}
	}
	return spanBatch
}

// setFirstOriginChangedBit sets the first bit of the originBits to the given value
// used for testing when a Span Batch is made with InitializedSpanBatch, which doesn't have a sequence number
func (b *SpanBatch) setFirstOriginChangedBit(bit uint) {
	b.originBits.SetBit(b.originBits, 0, bit)
}

func TestSpanBatchForBatchInterface(t *testing.T) {
	rng := rand.New(rand.NewSource(0x5432177))
	chainID := big.NewInt(rng.Int63n(1000))

	singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
	blockCount := len(singularBatches)
	safeL2Head := testutils.RandomL2BlockRef(rng)
	safeL2Head.Hash = common.BytesToHash(singularBatches[0].ParentHash[:])

	spanBatch := initializedSpanBatch(singularBatches, uint64(0), chainID)

	// check interface method implementations except logging
	require.Equal(t, SpanBatchType, spanBatch.GetBatchType())
	require.Equal(t, singularBatches[0].Timestamp, spanBatch.GetTimestamp())
	require.Equal(t, singularBatches[0].EpochNum, spanBatch.GetStartEpochNum())
	require.True(t, spanBatch.CheckOriginHash(singularBatches[blockCount-1].EpochHash))
	require.True(t, spanBatch.CheckParentHash(singularBatches[0].ParentHash))
}

func TestEmptySpanBatch(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556691))
	chainID := big.NewInt(rng.Int63n(1000))
	spanTxs, err := newSpanBatchTxs(nil, chainID)
	require.NoError(t, err)

	rawSpanBatch := RawSpanBatch{
		spanBatchPrefix: spanBatchPrefix{
			relTimestamp:  uint64(rng.Uint32()),
			l1OriginNum:   rng.Uint64(),
			parentCheck:   *(*[20]byte)(testutils.RandomData(rng, 20)),
			l1OriginCheck: *(*[20]byte)(testutils.RandomData(rng, 20)),
		},
		spanBatchPayload: spanBatchPayload{
			blockCount:    0,
			originBits:    big.NewInt(0),
			blockTxCounts: []uint64{},
			txs:           spanTxs,
		},
	}

	var buf bytes.Buffer
	err = rawSpanBatch.encodeBlockCount(&buf)
	assert.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	err = sb.decodeBlockCount(r)
	require.ErrorIs(t, err, ErrEmptySpanBatch)
}

func TestSpanBatchOriginBits(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77665544))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	blockCount := rawSpanBatch.blockCount

	var buf bytes.Buffer
	err := rawSpanBatch.encodeOriginBits(&buf)
	require.NoError(t, err)

	// originBit field is fixed length: single bit
	originBitBufferLen := blockCount / 8
	if blockCount%8 != 0 {
		originBitBufferLen++
	}
	require.Equal(t, buf.Len(), int(originBitBufferLen))

	result := buf.Bytes()
	var sb RawSpanBatch
	sb.blockCount = blockCount
	r := bytes.NewReader(result)
	err = sb.decodeOriginBits(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.originBits, sb.originBits)
}

func TestSpanBatchPrefix(t *testing.T) {
	rng := rand.New(rand.NewSource(0x44775566))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	// only compare prefix
	rawSpanBatch.spanBatchPayload = spanBatchPayload{}

	var buf bytes.Buffer
	err := rawSpanBatch.encodePrefix(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodePrefix(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch, &sb)
}

func TestSpanBatchRelTimestamp(t *testing.T) {
	rng := rand.New(rand.NewSource(0x44775566))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeRelTimestamp(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeRelTimestamp(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.relTimestamp, sb.relTimestamp)
}

func TestSpanBatchL1OriginNum(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556688))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeL1OriginNum(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeL1OriginNum(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.l1OriginNum, sb.l1OriginNum)
}

func TestSpanBatchParentCheck(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556689))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeParentCheck(&buf)
	require.NoError(t, err)

	// parent check field is fixed length: 20 bytes
	require.Equal(t, buf.Len(), 20)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeParentCheck(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.parentCheck, sb.parentCheck)
}

func TestSpanBatchL1OriginCheck(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556690))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeL1OriginCheck(&buf)
	require.NoError(t, err)

	// l1 origin check field is fixed length: 20 bytes
	require.Equal(t, buf.Len(), 20)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeL1OriginCheck(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.l1OriginCheck, sb.l1OriginCheck)
}

func TestSpanBatchPayload(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556691))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodePayload(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	err = sb.decodePayload(r)
	require.NoError(t, err)

	err = sb.txs.recoverV(chainID)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.spanBatchPayload, sb.spanBatchPayload)
}

func TestSpanBatchBlockCount(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556691))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeBlockCount(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	err = sb.decodeBlockCount(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.blockCount, sb.blockCount)
}

func TestSpanBatchBlockTxCounts(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556692))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeBlockTxCounts(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	sb.blockCount = rawSpanBatch.blockCount
	err = sb.decodeBlockTxCounts(r)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.blockTxCounts, sb.blockTxCounts)
}

func TestSpanBatchTxs(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556693))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var buf bytes.Buffer
	err := rawSpanBatch.encodeTxs(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch

	sb.blockTxCounts = rawSpanBatch.blockTxCounts
	err = sb.decodeTxs(r)
	require.NoError(t, err)

	err = sb.txs.recoverV(chainID)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch.txs, sb.txs)
}

func TestSpanBatchRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556694))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)

	var result bytes.Buffer
	err := rawSpanBatch.encode(&result)
	require.NoError(t, err)

	var sb RawSpanBatch
	err = sb.decode(bytes.NewReader(result.Bytes()))
	require.NoError(t, err)

	err = sb.txs.recoverV(chainID)
	require.NoError(t, err)

	require.Equal(t, rawSpanBatch, &sb)
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

		spanBatch := initializedSpanBatch(singularBatches, genesisTimeStamp, chainID)
		// set originChangedBit to match the original test implementation
		spanBatch.setFirstOriginChangedBit(uint(originChangedBit))
		rawSpanBatch, err := spanBatch.ToRawSpanBatch()
		require.NoError(t, err)

		spanBatchDerived, err := rawSpanBatch.derive(l2BlockTime, genesisTimeStamp, chainID)
		require.NoError(t, err)

		blockCount := len(singularBatches)
		require.Equal(t, safeL2Head.Hash.Bytes()[:20], spanBatchDerived.ParentCheck[:])
		require.Equal(t, singularBatches[blockCount-1].Epoch().Hash.Bytes()[:20], spanBatchDerived.L1OriginCheck[:])
		require.Equal(t, len(singularBatches), int(rawSpanBatch.blockCount))

		for i := 1; i < len(singularBatches); i++ {
			require.Equal(t, spanBatchDerived.Batches[i].Timestamp, spanBatchDerived.Batches[i-1].Timestamp+l2BlockTime)
		}

		for i := 0; i < len(singularBatches); i++ {
			require.Equal(t, singularBatches[i].EpochNum, spanBatchDerived.Batches[i].EpochNum)
			require.Equal(t, singularBatches[i].Timestamp, spanBatchDerived.Batches[i].Timestamp)
			require.Equal(t, singularBatches[i].Transactions, spanBatchDerived.Batches[i].Transactions)
		}
	}
}

func TestSpanBatchAppend(t *testing.T) {
	rng := rand.New(rand.NewSource(0x44337711))

	chainID := new(big.Int).SetUint64(rng.Uint64())

	singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
	// initialize empty span batch
	spanBatch := initializedSpanBatch([]*SingularBatch{}, uint64(0), chainID)

	L := 2
	for i := 0; i < L; i++ {
		err := spanBatch.AppendSingularBatch(singularBatches[i], uint64(i))
		require.NoError(t, err)
	}
	// initialize with two singular batches
	spanBatch2 := initializedSpanBatch(singularBatches[:L], uint64(0), chainID)

	require.Equal(t, spanBatch, spanBatch2)
}

func TestSpanBatchMerge(t *testing.T) {
	rng := rand.New(rand.NewSource(0x73314433))

	genesisTimeStamp := rng.Uint64()
	chainID := new(big.Int).SetUint64(rng.Uint64())

	for originChangedBit := 0; originChangedBit < 2; originChangedBit++ {
		singularBatches := RandomValidConsecutiveSingularBatches(rng, chainID)
		blockCount := len(singularBatches)

		spanBatch := initializedSpanBatch(singularBatches, genesisTimeStamp, chainID)
		// set originChangedBit to match the original test implementation
		spanBatch.setFirstOriginChangedBit(uint(originChangedBit))
		rawSpanBatch, err := spanBatch.ToRawSpanBatch()
		require.NoError(t, err)

		// check span batch prefix
		require.Equal(t, rawSpanBatch.relTimestamp, singularBatches[0].Timestamp-genesisTimeStamp, "invalid relative timestamp")
		require.Equal(t, rollup.Epoch(rawSpanBatch.l1OriginNum), singularBatches[blockCount-1].EpochNum)
		require.Equal(t, rawSpanBatch.parentCheck[:], singularBatches[0].ParentHash.Bytes()[:20], "invalid parent check")
		require.Equal(t, rawSpanBatch.l1OriginCheck[:], singularBatches[blockCount-1].EpochHash.Bytes()[:20], "invalid l1 origin check")

		// check span batch payload
		require.Equal(t, int(rawSpanBatch.blockCount), len(singularBatches))
		require.Equal(t, rawSpanBatch.originBits.Bit(0), uint(originChangedBit))
		for i := 1; i < blockCount; i++ {
			if rawSpanBatch.originBits.Bit(i) == 1 {
				require.Equal(t, singularBatches[i].EpochNum, singularBatches[i-1].EpochNum+1)
			} else {
				require.Equal(t, singularBatches[i].EpochNum, singularBatches[i-1].EpochNum)
			}
		}
		for i := 0; i < len(singularBatches); i++ {
			txCount := len(singularBatches[i].Transactions)
			require.Equal(t, txCount, int(rawSpanBatch.blockTxCounts[i]))
		}

		// check invariants
		endEpochNum := rawSpanBatch.l1OriginNum
		require.Equal(t, endEpochNum, uint64(singularBatches[blockCount-1].EpochNum))

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

		spanBatch := initializedSpanBatch(singularBatches, genesisTimeStamp, chainID)
		// set originChangedBit to match the original test implementation
		spanBatch.setFirstOriginChangedBit(uint(originChangedBit))
		rawSpanBatch, err := spanBatch.ToRawSpanBatch()
		require.NoError(t, err)

		l1Origins := mockL1Origin(rng, rawSpanBatch, singularBatches)

		singularBatches2, err := spanBatch.GetSingularBatches(l1Origins, safeL2Head)
		require.NoError(t, err)

		// GetSingularBatches does not fill in parent hash of singular batches
		// empty out parent hash for comparison
		for i := 0; i < len(singularBatches); i++ {
			singularBatches[i].ParentHash = common.Hash{}
		}
		// check parent hash is empty
		for i := 0; i < len(singularBatches2); i++ {
			require.Equal(t, singularBatches2[i].ParentHash, common.Hash{})
		}

		require.Equal(t, singularBatches, singularBatches2)
	}
}

func TestSpanBatchReadTxData(t *testing.T) {
	cases := []spanBatchTxTest{
		{"unprotected legacy tx", 32, testutils.RandomLegacyTx, false},
		{"legacy tx", 32, testutils.RandomLegacyTx, true},
		{"access list tx", 32, testutils.RandomAccessListTx, true},
		{"dynamic fee tx", 32, testutils.RandomDynamicFeeTx, true},
	}

	for i, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(0x109550 + i)))
			chainID := new(big.Int).SetUint64(rng.Uint64())
			signer := types.NewLondonSigner(chainID)
			if !testCase.protected {
				signer = types.HomesteadSigner{}
			}

			var rawTxs [][]byte
			var txs []*types.Transaction
			for txIdx := 0; txIdx < testCase.trials; txIdx++ {
				tx := testCase.mkTx(rng, signer)
				rawTx, err := tx.MarshalBinary()
				require.NoError(t, err)
				rawTxs = append(rawTxs, rawTx)
				txs = append(txs, tx)
			}

			for txIdx := 0; txIdx < testCase.trials; txIdx++ {
				r := bytes.NewReader(rawTxs[i])
				_, txType, err := ReadTxData(r)
				require.NoError(t, err)
				assert.Equal(t, int(txs[i].Type()), txType)
			}
		})
	}
}

func TestSpanBatchReadTxDataInvalid(t *testing.T) {
	dummy, err := rlp.EncodeToBytes("dummy")
	require.NoError(t, err)

	// test non list rlp decoding
	r := bytes.NewReader(dummy)
	_, _, err = ReadTxData(r)
	require.ErrorContains(t, err, "tx RLP prefix type must be list")
}

func TestSpanBatchMaxTxData(t *testing.T) {
	rng := rand.New(rand.NewSource(0x177288))

	invalidTx := types.NewTx(&types.DynamicFeeTx{
		Data: testutils.RandomData(rng, MaxSpanBatchElementCount+1),
	})

	txEncoded, err := invalidTx.MarshalBinary()
	require.NoError(t, err)

	r := bytes.NewReader(txEncoded)
	_, _, err = ReadTxData(r)

	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}

func TestSpanBatchMaxOriginBitsLength(t *testing.T) {
	var sb RawSpanBatch
	sb.blockCount = math.MaxUint64

	r := bytes.NewReader([]byte{})
	err := sb.decodeOriginBits(r)
	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}

func TestSpanBatchMaxBlockCount(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556691))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	rawSpanBatch.blockCount = math.MaxUint64

	var buf bytes.Buffer
	err := rawSpanBatch.encodeBlockCount(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	err = sb.decodeBlockCount(r)
	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}

func TestSpanBatchMaxBlockTxCount(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556692))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	rawSpanBatch.blockTxCounts[0] = math.MaxUint64

	var buf bytes.Buffer
	err := rawSpanBatch.encodeBlockTxCounts(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	sb.blockCount = rawSpanBatch.blockCount
	err = sb.decodeBlockTxCounts(r)
	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}

func TestSpanBatchTotalBlockTxCountNotOverflow(t *testing.T) {
	rng := rand.New(rand.NewSource(0x77556693))
	chainID := big.NewInt(rng.Int63n(1000))

	rawSpanBatch := RandomRawSpanBatch(rng, chainID)
	rawSpanBatch.blockTxCounts[0] = MaxSpanBatchElementCount - 1
	rawSpanBatch.blockTxCounts[1] = MaxSpanBatchElementCount - 1
	// we are sure that totalBlockTxCount will overflow on uint64

	var buf bytes.Buffer
	err := rawSpanBatch.encodeBlockTxCounts(&buf)
	require.NoError(t, err)

	result := buf.Bytes()
	r := bytes.NewReader(result)
	var sb RawSpanBatch
	sb.blockTxCounts = rawSpanBatch.blockTxCounts
	err = sb.decodeTxs(r)

	require.ErrorIs(t, err, ErrTooBigSpanBatchSize)
}
