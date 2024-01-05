package derive

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type fakeBatchQueueInput struct {
	i       int
	batches []Batch
	errors  []error
	origin  eth.L1BlockRef
}

func (f *fakeBatchQueueInput) Origin() eth.L1BlockRef {
	return f.origin
}

func (f *fakeBatchQueueInput) NextBatch(ctx context.Context) (Batch, error) {
	if f.i >= len(f.batches) {
		return nil, io.EOF
	}
	b := f.batches[f.i]
	e := f.errors[f.i]
	f.i += 1
	return b, e
}

func mockHash(time uint64, layer uint8) common.Hash {
	hash := common.Hash{31: layer} // indicate L1 or L2
	binary.LittleEndian.PutUint64(hash[:], time)
	return hash
}

func b(chainId *big.Int, timestamp uint64, epoch eth.L1BlockRef) *SingularBatch {
	rng := rand.New(rand.NewSource(int64(timestamp)))
	signer := types.NewLondonSigner(chainId)
	tx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
	txData, _ := tx.MarshalBinary()
	return &SingularBatch{
		ParentHash:   mockHash(timestamp-2, 2),
		Timestamp:    timestamp,
		EpochNum:     rollup.Epoch(epoch.Number),
		EpochHash:    epoch.Hash,
		Transactions: []hexutil.Bytes{txData},
	}
}

func buildSpanBatches(t *testing.T, parent *eth.L2BlockRef, singularBatches []*SingularBatch, blockCounts []int, chainId *big.Int) []Batch {
	var spanBatches []Batch
	idx := 0
	for _, count := range blockCounts {
		span := NewSpanBatch(singularBatches[idx : idx+count])
		spanBatches = append(spanBatches, span)
		idx += count
	}
	return spanBatches
}

func getDeltaTime(batchType int) *uint64 {
	minTs := uint64(0)
	if batchType == SpanBatchType {
		return &minTs
	}
	return nil
}

func l1InfoDepositTx(t *testing.T, l1BlockNum uint64) hexutil.Bytes {
	l1Info := L1BlockInfo{
		Number:  l1BlockNum,
		BaseFee: big.NewInt(0),
	}
	infoData, err := l1Info.MarshalBinary()
	require.NoError(t, err)
	depositTx := &types.DepositTx{
		Data: infoData,
	}
	txData, err := types.NewTx(depositTx).MarshalBinary()
	require.NoError(t, err)
	return txData
}

func singularBatchToPayload(t *testing.T, batch *SingularBatch, blockNumber uint64) eth.ExecutionPayload {
	txs := []hexutil.Bytes{l1InfoDepositTx(t, uint64(batch.EpochNum))}
	txs = append(txs, batch.Transactions...)
	return eth.ExecutionPayload{
		BlockHash:    mockHash(batch.Timestamp, 2),
		ParentHash:   batch.ParentHash,
		BlockNumber:  hexutil.Uint64(blockNumber),
		Timestamp:    hexutil.Uint64(batch.Timestamp),
		Transactions: txs,
	}
}

func singularBatchToBlockRef(t *testing.T, batch *SingularBatch, blockNumber uint64) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:       mockHash(batch.Timestamp, 2),
		Number:     blockNumber,
		ParentHash: batch.ParentHash,
		Time:       batch.Timestamp,
		L1Origin:   eth.BlockID{Hash: batch.EpochHash, Number: uint64(batch.EpochNum)},
	}
}

func L1Chain(l1Times []uint64) []eth.L1BlockRef {
	var out []eth.L1BlockRef
	var parentHash common.Hash
	for i, time := range l1Times {
		hash := mockHash(time, 1)
		out = append(out, eth.L1BlockRef{
			Hash:       hash,
			Number:     uint64(i),
			ParentHash: parentHash,
			Time:       time,
		})
		parentHash = hash
	}
	return out
}

func TestBatchQueue(t *testing.T) {
	tests := []struct {
		name string
		f    func(t *testing.T, batchType int)
	}{
		{"BatchQueueNewOrigin", BatchQueueNewOrigin},
		{"BatchQueueEager", BatchQueueEager},
		{"BatchQueueInvalidInternalAdvance", BatchQueueInvalidInternalAdvance},
		{"BatchQueueMissing", BatchQueueMissing},
		{"BatchQueueAdvancedEpoch", BatchQueueAdvancedEpoch},
		{"BatchQueueShuffle", BatchQueueShuffle},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name+"_SingularBatch", func(t *testing.T) {
			test.f(t, SingularBatchType)
		})
	}

	for _, test := range tests {
		test := test
		t.Run(test.name+"_SpanBatch", func(t *testing.T) {
			test.f(t, SpanBatchType)
		})
	}
}

// BatchQueueNewOrigin tests that the batch queue properly saves the new origin
// when the safehead's origin is ahead of the pipeline's origin (as is after a reset).
// This issue was fixed in https://github.com/ethereum-optimism/optimism/pull/3694
func BatchQueueNewOrigin(t *testing.T, batchType int) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{10, 15, 20, 25})
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(10, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           20,
		L1Origin:       l1[2].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     2,
		DeltaTime:         getDeltaTime(batchType),
	}

	input := &fakeBatchQueueInput{
		batches: []Batch{nil},
		errors:  []error{io.EOF},
		origin:  l1[0],
	}

	bq := NewBatchQueue(log, cfg, input, nil)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})
	require.Equal(t, []eth.L1BlockRef{l1[0]}, bq.l1Blocks)

	// Prev Origin: 0; Safehead Origin: 2; Internal Origin: 0
	// Should return no data but keep the same origin
	data, _, err := bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, data)
	require.Equal(t, io.EOF, err)
	require.Equal(t, []eth.L1BlockRef{l1[0]}, bq.l1Blocks)
	require.Equal(t, l1[0], bq.origin)

	// Prev Origin: 1; Safehead Origin: 2; Internal Origin: 0
	// Should wipe l1blocks + advance internal origin
	input.origin = l1[1]
	data, _, err = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, data)
	require.Equal(t, io.EOF, err)
	require.Empty(t, bq.l1Blocks)
	require.Equal(t, l1[1], bq.origin)

	// Prev Origin: 2; Safehead Origin: 2; Internal Origin: 1
	// Should add to l1Blocks + advance internal origin
	input.origin = l1[2]
	data, _, err = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, data)
	require.Equal(t, io.EOF, err)
	require.Equal(t, []eth.L1BlockRef{l1[2]}, bq.l1Blocks)
	require.Equal(t, l1[2], bq.origin)
}

// BatchQueueEager adds a bunch of contiguous batches and asserts that
// enough calls to `NextBatch` return all of those batches.
func BatchQueueEager(t *testing.T, batchType int) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{10, 20, 30})
	chainId := big.NewInt(1234)
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(10, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           10,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
		DeltaTime:         getDeltaTime(batchType),
		L2ChainID:         chainId,
	}

	// expected output of BatchQueue.NextBatch()
	expectedOutputBatches := []*SingularBatch{
		b(cfg.L2ChainID, 12, l1[0]),
		b(cfg.L2ChainID, 14, l1[0]),
		b(cfg.L2ChainID, 16, l1[0]),
		b(cfg.L2ChainID, 18, l1[0]),
		b(cfg.L2ChainID, 20, l1[0]),
		b(cfg.L2ChainID, 22, l1[0]),
		nil,
	}
	// expected error of BatchQueue.NextBatch()
	expectedOutputErrors := []error{nil, nil, nil, nil, nil, nil, io.EOF}
	// errors will be returned by fakeBatchQueueInput.NextBatch()
	inputErrors := expectedOutputErrors
	// batches will be returned by fakeBatchQueueInput
	var inputBatches []Batch
	if batchType == SpanBatchType {
		spanBlockCounts := []int{1, 2, 3}
		inputErrors = []error{nil, nil, nil, io.EOF}
		inputBatches = buildSpanBatches(t, &safeHead, expectedOutputBatches, spanBlockCounts, chainId)
		inputBatches = append(inputBatches, nil)
	} else {
		for _, singularBatch := range expectedOutputBatches {
			inputBatches = append(inputBatches, singularBatch)
		}
	}

	input := &fakeBatchQueueInput{
		batches: inputBatches,
		errors:  inputErrors,
		origin:  l1[0],
	}

	bq := NewBatchQueue(log, cfg, input, nil)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})
	// Advance the origin
	input.origin = l1[1]

	for i := 0; i < len(expectedOutputBatches); i++ {
		b, _, e := bq.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, e, expectedOutputErrors[i])
		if b == nil {
			require.Nil(t, expectedOutputBatches[i])
		} else {
			require.Equal(t, expectedOutputBatches[i], b)
			safeHead.Number += 1
			safeHead.Time += cfg.BlockTime
			safeHead.Hash = mockHash(b.Timestamp, 2)
			safeHead.L1Origin = b.Epoch()
		}
	}
}

// BatchQueueInvalidInternalAdvance asserts that we do not miss an epoch when generating batches.
// This is a regression test for CLI-3378.
func BatchQueueInvalidInternalAdvance(t *testing.T, batchType int) {
	log := testlog.Logger(t, log.LvlTrace)
	l1 := L1Chain([]uint64{10, 15, 20, 25, 30})
	chainId := big.NewInt(1234)
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(10, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           10,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     2,
		DeltaTime:         getDeltaTime(batchType),
		L2ChainID:         chainId,
	}

	// expected output of BatchQueue.NextBatch()
	expectedOutputBatches := []*SingularBatch{
		b(cfg.L2ChainID, 12, l1[0]),
		b(cfg.L2ChainID, 14, l1[0]),
		b(cfg.L2ChainID, 16, l1[0]),
		b(cfg.L2ChainID, 18, l1[0]),
		b(cfg.L2ChainID, 20, l1[0]),
		b(cfg.L2ChainID, 22, l1[0]),
		nil,
	}
	// expected error of BatchQueue.NextBatch()
	expectedOutputErrors := []error{nil, nil, nil, nil, nil, nil, io.EOF}
	// errors will be returned by fakeBatchQueueInput.NextBatch()
	inputErrors := expectedOutputErrors
	// batches will be returned by fakeBatchQueueInput
	var inputBatches []Batch
	if batchType == SpanBatchType {
		spanBlockCounts := []int{1, 2, 3}
		inputErrors = []error{nil, nil, nil, io.EOF}
		inputBatches = buildSpanBatches(t, &safeHead, expectedOutputBatches, spanBlockCounts, chainId)
		inputBatches = append(inputBatches, nil)
	} else {
		for _, singularBatch := range expectedOutputBatches {
			inputBatches = append(inputBatches, singularBatch)
		}
	}

	input := &fakeBatchQueueInput{
		batches: inputBatches,
		errors:  inputErrors,
		origin:  l1[0],
	}

	bq := NewBatchQueue(log, cfg, input, nil)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})

	// Load continuous batches for epoch 0
	for i := 0; i < len(expectedOutputBatches); i++ {
		b, _, e := bq.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, e, expectedOutputErrors[i])
		if b == nil {
			require.Nil(t, expectedOutputBatches[i])
		} else {
			require.Equal(t, expectedOutputBatches[i], b)
			safeHead.Number += 1
			safeHead.Time += 2
			safeHead.Hash = mockHash(b.Timestamp, 2)
			safeHead.L1Origin = b.Epoch()
		}
	}

	// Advance to origin 1. No forced batches yet.
	input.origin = l1[1]
	b, _, e := bq.NextBatch(context.Background(), safeHead)
	require.ErrorIs(t, e, io.EOF)
	require.Nil(t, b)

	// Advance to origin 2. No forced batches yet because we are still on epoch 0
	// & have batches for epoch 0.
	input.origin = l1[2]
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.ErrorIs(t, e, io.EOF)
	require.Nil(t, b)

	// Advance to origin 3. Should generate one empty batch.
	input.origin = l1[3]
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.NotNil(t, b)
	require.Equal(t, safeHead.Time+2, b.Timestamp)
	require.Equal(t, rollup.Epoch(1), b.EpochNum)
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)
	safeHead.L1Origin = b.Epoch()
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.ErrorIs(t, e, io.EOF)
	require.Nil(t, b)

	// Advance to origin 4. Should generate one empty batch.
	input.origin = l1[4]
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.NotNil(t, b)
	require.Equal(t, rollup.Epoch(2), b.EpochNum)
	require.Equal(t, safeHead.Time+2, b.Timestamp)
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)
	safeHead.L1Origin = b.Epoch()
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.ErrorIs(t, e, io.EOF)
	require.Nil(t, b)

}

func BatchQueueMissing(t *testing.T, batchType int) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{10, 15, 20, 25})
	chainId := big.NewInt(1234)
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(10, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           10,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     2,
		DeltaTime:         getDeltaTime(batchType),
		L2ChainID:         chainId,
	}

	// The inputBatches at 18 and 20 are skipped to stop 22 from being eagerly processed.
	// This test checks that batch timestamp 12 & 14 are created, 16 is used, and 18 is advancing the epoch.
	// Due to the large sequencer time drift 16 is perfectly valid to have epoch 0 as origin.a

	// expected output of BatchQueue.NextBatch()
	expectedOutputBatches := []*SingularBatch{
		b(cfg.L2ChainID, 16, l1[0]),
		b(cfg.L2ChainID, 22, l1[1]),
	}
	// errors will be returned by fakeBatchQueueInput.NextBatch()
	inputErrors := []error{nil, nil}
	// batches will be returned by fakeBatchQueueInput
	var inputBatches []Batch
	if batchType == SpanBatchType {
		spanBlockCounts := []int{1, 1}
		inputErrors = []error{nil, nil, nil, io.EOF}
		inputBatches = buildSpanBatches(t, &safeHead, expectedOutputBatches, spanBlockCounts, chainId)
	} else {
		for _, singularBatch := range expectedOutputBatches {
			inputBatches = append(inputBatches, singularBatch)
		}
	}

	input := &fakeBatchQueueInput{
		batches: inputBatches,
		errors:  inputErrors,
		origin:  l1[0],
	}

	bq := NewBatchQueue(log, cfg, input, nil)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})

	for i := 0; i < len(expectedOutputBatches); i++ {
		b, _, e := bq.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, e, NotEnoughData)
		require.Nil(t, b)
	}

	// advance origin. Underlying stage still has no more inputBatches
	// This is not enough to auto advance yet
	input.origin = l1[1]
	b, _, e := bq.NextBatch(context.Background(), safeHead)
	require.ErrorIs(t, e, io.EOF)
	require.Nil(t, b)

	// Advance the origin. At this point batch timestamps 12 and 14 will be created
	input.origin = l1[2]

	// Check for a generated batch at t = 12
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b.Timestamp, uint64(12))
	require.Empty(t, b.Transactions)
	require.Equal(t, rollup.Epoch(0), b.EpochNum)
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)

	// Check for generated batch at t = 14
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b.Timestamp, uint64(14))
	require.Empty(t, b.Transactions)
	require.Equal(t, rollup.Epoch(0), b.EpochNum)
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)

	// Check for the inputted batch at t = 16
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b, expectedOutputBatches[0])
	require.Equal(t, rollup.Epoch(0), b.EpochNum)
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)

	// Advance the origin. At this point the batch with timestamp 18 will be created
	input.origin = l1[3]

	// Check for the generated batch at t = 18. This batch advances the epoch
	// Note: We need one io.EOF returned from the bq that advances the internal L1 Blocks view
	// before the batch will be auto generated
	_, _, e = bq.NextBatch(context.Background(), safeHead)
	require.Equal(t, e, io.EOF)
	b, _, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b.Timestamp, uint64(18))
	require.Empty(t, b.Transactions)
	require.Equal(t, rollup.Epoch(1), b.EpochNum)
}

// BatchQueueAdvancedEpoch tests that batch queue derives consecutive valid batches with advancing epochs.
// Batch queue's l1blocks list should be updated along epochs.
func BatchQueueAdvancedEpoch(t *testing.T, batchType int) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{0, 6, 12, 18, 24}) // L1 block time: 6s
	chainId := big.NewInt(1234)
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(4, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           4,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
		DeltaTime:         getDeltaTime(batchType),
		L2ChainID:         chainId,
	}

	// expected output of BatchQueue.NextBatch()
	expectedOutputBatches := []*SingularBatch{
		// 3 L2 blocks per L1 block
		b(cfg.L2ChainID, 6, l1[1]),
		b(cfg.L2ChainID, 8, l1[1]),
		b(cfg.L2ChainID, 10, l1[1]),
		b(cfg.L2ChainID, 12, l1[2]),
		b(cfg.L2ChainID, 14, l1[2]),
		b(cfg.L2ChainID, 16, l1[2]),
		b(cfg.L2ChainID, 18, l1[3]),
		b(cfg.L2ChainID, 20, l1[3]),
		b(cfg.L2ChainID, 22, l1[3]),
		nil,
	}
	// expected error of BatchQueue.NextBatch()
	expectedOutputErrors := []error{nil, nil, nil, nil, nil, nil, nil, nil, nil, io.EOF}
	// errors will be returned by fakeBatchQueueInput.NextBatch()
	inputErrors := expectedOutputErrors
	// batches will be returned by fakeBatchQueueInput
	var inputBatches []Batch
	if batchType == SpanBatchType {
		spanBlockCounts := []int{2, 2, 2, 3}
		inputErrors = []error{nil, nil, nil, nil, io.EOF}
		inputBatches = buildSpanBatches(t, &safeHead, expectedOutputBatches, spanBlockCounts, chainId)
		inputBatches = append(inputBatches, nil)
	} else {
		for _, singularBatch := range expectedOutputBatches {
			inputBatches = append(inputBatches, singularBatch)
		}
	}

	// ChannelInReader origin number
	inputOriginNumber := 2
	input := &fakeBatchQueueInput{
		batches: inputBatches,
		errors:  inputErrors,
		origin:  l1[inputOriginNumber],
	}

	bq := NewBatchQueue(log, cfg, input, nil)
	_ = bq.Reset(context.Background(), l1[1], eth.SystemConfig{})

	for i := 0; i < len(expectedOutputBatches); i++ {
		expectedOutput := expectedOutputBatches[i]
		if expectedOutput != nil && uint64(expectedOutput.EpochNum) == l1[inputOriginNumber].Number {
			// Advance ChannelInReader origin if needed
			inputOriginNumber += 1
			input.origin = l1[inputOriginNumber]
		}
		b, _, e := bq.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, e, expectedOutputErrors[i])
		if b == nil {
			require.Nil(t, expectedOutput)
		} else {
			require.Equal(t, expectedOutput, b)
			safeHead.Number += 1
			safeHead.Time += cfg.BlockTime
			safeHead.Hash = mockHash(b.Timestamp, 2)
			safeHead.L1Origin = b.Epoch()
		}
	}
}

// BatchQueueShuffle tests batch queue can reorder shuffled valid batches
func BatchQueueShuffle(t *testing.T, batchType int) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{0, 6, 12, 18, 24}) // L1 block time: 6s
	chainId := big.NewInt(1234)
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(4, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           4,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
		DeltaTime:         getDeltaTime(batchType),
		L2ChainID:         chainId,
	}

	// expected output of BatchQueue.NextBatch()
	expectedOutputBatches := []*SingularBatch{
		// 3 L2 blocks per L1 block
		b(cfg.L2ChainID, 6, l1[1]),
		b(cfg.L2ChainID, 8, l1[1]),
		b(cfg.L2ChainID, 10, l1[1]),
		b(cfg.L2ChainID, 12, l1[2]),
		b(cfg.L2ChainID, 14, l1[2]),
		b(cfg.L2ChainID, 16, l1[2]),
		b(cfg.L2ChainID, 18, l1[3]),
		b(cfg.L2ChainID, 20, l1[3]),
		b(cfg.L2ChainID, 22, l1[3]),
	}
	// expected error of BatchQueue.NextBatch()
	expectedOutputErrors := []error{nil, nil, nil, nil, nil, nil, nil, nil, nil, io.EOF}
	// errors will be returned by fakeBatchQueueInput.NextBatch()
	inputErrors := expectedOutputErrors
	// batches will be returned by fakeBatchQueueInput
	var inputBatches []Batch
	if batchType == SpanBatchType {
		spanBlockCounts := []int{2, 2, 2, 3}
		inputErrors = []error{nil, nil, nil, nil, io.EOF}
		inputBatches = buildSpanBatches(t, &safeHead, expectedOutputBatches, spanBlockCounts, chainId)
	} else {
		for _, singularBatch := range expectedOutputBatches {
			inputBatches = append(inputBatches, singularBatch)
		}
	}

	// Shuffle the order of input batches
	rand.Shuffle(len(inputBatches), func(i, j int) {
		inputBatches[i], inputBatches[j] = inputBatches[j], inputBatches[i]
	})
	inputBatches = append(inputBatches, nil)

	// ChannelInReader origin number
	inputOriginNumber := 2
	input := &fakeBatchQueueInput{
		batches: inputBatches,
		errors:  inputErrors,
		origin:  l1[inputOriginNumber],
	}

	bq := NewBatchQueue(log, cfg, input, nil)
	_ = bq.Reset(context.Background(), l1[1], eth.SystemConfig{})

	for i := 0; i < len(expectedOutputBatches); i++ {
		expectedOutput := expectedOutputBatches[i]
		if expectedOutput != nil && uint64(expectedOutput.EpochNum) == l1[inputOriginNumber].Number {
			// Advance ChannelInReader origin if needed
			inputOriginNumber += 1
			input.origin = l1[inputOriginNumber]
		}
		var b *SingularBatch
		var e error
		for j := 0; j < len(expectedOutputBatches); j++ {
			// Multiple NextBatch() executions may be required because the order of input is shuffled
			b, _, e = bq.NextBatch(context.Background(), safeHead)
			if !errors.Is(e, NotEnoughData) {
				break
			}
		}
		require.ErrorIs(t, e, expectedOutputErrors[i])
		if b == nil {
			require.Nil(t, expectedOutput)
		} else {
			require.Equal(t, expectedOutput, b)
			safeHead.Number += 1
			safeHead.Time += cfg.BlockTime
			safeHead.Hash = mockHash(b.Timestamp, 2)
			safeHead.L1Origin = b.Epoch()
		}
	}
}

func TestBatchQueueOverlappingSpanBatch(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{10, 20, 30})
	chainId := big.NewInt(1234)
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(10, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           10,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
		DeltaTime:         getDeltaTime(SpanBatchType),
		L2ChainID:         chainId,
	}

	// expected output of BatchQueue.NextBatch()
	expectedOutputBatches := []*SingularBatch{
		b(cfg.L2ChainID, 12, l1[0]),
		b(cfg.L2ChainID, 14, l1[0]),
		b(cfg.L2ChainID, 16, l1[0]),
		b(cfg.L2ChainID, 18, l1[0]),
		b(cfg.L2ChainID, 20, l1[0]),
		b(cfg.L2ChainID, 22, l1[0]),
		nil,
	}
	// expected error of BatchQueue.NextBatch()
	expectedOutputErrors := []error{nil, nil, nil, nil, nil, nil, io.EOF}
	// errors will be returned by fakeBatchQueueInput.NextBatch()
	inputErrors := []error{nil, nil, nil, nil, io.EOF}

	// batches will be returned by fakeBatchQueueInput
	var inputBatches []Batch
	batchSize := 3
	for i := 0; i < len(expectedOutputBatches)-batchSize; i++ {
		inputBatches = append(inputBatches, NewSpanBatch(expectedOutputBatches[i:i+batchSize]))
	}
	inputBatches = append(inputBatches, nil)
	// inputBatches:
	// [
	//    [12, 14, 16],  // No overlap
	//    [14, 16, 18],  // overlapped blocks: 14, 16
	//    [16, 18, 20],  // overlapped blocks: 16, 18
	//    [18, 20, 22],  // overlapped blocks: 18, 20
	// ]

	input := &fakeBatchQueueInput{
		batches: inputBatches,
		errors:  inputErrors,
		origin:  l1[0],
	}

	l2Client := testutils.MockL2Client{}
	var nilErr error
	for i, batch := range expectedOutputBatches {
		if batch != nil {
			blockRef := singularBatchToBlockRef(t, batch, uint64(i+1))
			payload := singularBatchToPayload(t, batch, uint64(i+1))
			if i < 3 {
				// In CheckBatch(), "L2BlockRefByNumber" is called when fetching the parent block of overlapped span batch
				// so blocks at 12, 14, 16 should be called.
				// CheckBatch() is called twice for a batch - before pushing to the queue, after popping from the queue
				l2Client.Mock.On("L2BlockRefByNumber", uint64(i+1)).Times(2).Return(blockRef, &nilErr)
			}
			if i == 1 || i == 4 {
				// In CheckBatch(), "PayloadByNumber" is called when fetching the overlapped blocks.
				// blocks at 14, 20 are included in overlapped blocks once.
				// CheckBatch() is called twice for a batch - before adding to the queue, after getting from the queue
				l2Client.Mock.On("PayloadByNumber", uint64(i+1)).Times(2).Return(&payload, &nilErr)
			} else if i == 2 || i == 3 {
				// blocks at 16, 18 are included in overlapped blocks twice.
				l2Client.Mock.On("PayloadByNumber", uint64(i+1)).Times(4).Return(&payload, &nilErr)
			}
		}
	}

	bq := NewBatchQueue(log, cfg, input, &l2Client)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})
	// Advance the origin
	input.origin = l1[1]

	for i := 0; i < len(expectedOutputBatches); i++ {
		b, _, e := bq.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, e, expectedOutputErrors[i])
		if b == nil {
			require.Nil(t, expectedOutputBatches[i])
		} else {
			require.Equal(t, expectedOutputBatches[i], b)
			safeHead.Number += 1
			safeHead.Time += cfg.BlockTime
			safeHead.Hash = mockHash(b.Timestamp, 2)
			safeHead.L1Origin = b.Epoch()
		}
	}

	l2Client.Mock.AssertExpectations(t)
}

func TestBatchQueueComplex(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{0, 6, 12, 18, 24}) // L1 block time: 6s
	chainId := big.NewInt(1234)
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(4, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           4,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
		DeltaTime:         getDeltaTime(SpanBatchType),
		L2ChainID:         chainId,
	}

	// expected output of BatchQueue.NextBatch()
	expectedOutputBatches := []*SingularBatch{
		// 3 L2 blocks per L1 block
		b(cfg.L2ChainID, 6, l1[1]),
		b(cfg.L2ChainID, 8, l1[1]),
		b(cfg.L2ChainID, 10, l1[1]),
		b(cfg.L2ChainID, 12, l1[2]),
		b(cfg.L2ChainID, 14, l1[2]),
		b(cfg.L2ChainID, 16, l1[2]),
		b(cfg.L2ChainID, 18, l1[3]),
		b(cfg.L2ChainID, 20, l1[3]),
		b(cfg.L2ChainID, 22, l1[3]),
	}
	// expected error of BatchQueue.NextBatch()
	expectedOutputErrors := []error{nil, nil, nil, nil, nil, nil, nil, nil, nil, io.EOF}
	// errors will be returned by fakeBatchQueueInput.NextBatch()
	inputErrors := []error{nil, nil, nil, nil, nil, nil, io.EOF}
	// batches will be returned by fakeBatchQueueInput
	inputBatches := []Batch{
		NewSpanBatch(expectedOutputBatches[0:2]), // [6, 8] - no overlap
		expectedOutputBatches[2],                 // [10] - no overlap
		NewSpanBatch(expectedOutputBatches[1:4]), // [8, 10, 12] - overlapped blocks: 8 or 8, 10
		expectedOutputBatches[4],                 // [14] - no overlap
		NewSpanBatch(expectedOutputBatches[4:6]), // [14, 16] - overlapped blocks: nothing or 14
		NewSpanBatch(expectedOutputBatches[6:9]), // [18, 20, 22] - no overlap
	}

	// Shuffle the order of input batches
	rand.Shuffle(len(inputBatches), func(i, j int) {
		inputBatches[i], inputBatches[j] = inputBatches[j], inputBatches[i]
	})

	inputBatches = append(inputBatches, nil)

	// ChannelInReader origin number
	inputOriginNumber := 2
	input := &fakeBatchQueueInput{
		batches: inputBatches,
		errors:  inputErrors,
		origin:  l1[inputOriginNumber],
	}

	l2Client := testutils.MockL2Client{}
	var nilErr error
	for i, batch := range expectedOutputBatches {
		if batch != nil {
			blockRef := singularBatchToBlockRef(t, batch, uint64(i+1))
			payload := singularBatchToPayload(t, batch, uint64(i+1))
			if i == 0 || i == 3 {
				// In CheckBatch(), "L2BlockRefByNumber" is called when fetching the parent block of overlapped span batch
				// so blocks at 6, 8 could be called, depends on the order of batches
				l2Client.Mock.On("L2BlockRefByNumber", uint64(i+1)).Return(blockRef, &nilErr).Maybe()
			}
			if i == 1 || i == 2 || i == 4 {
				// In CheckBatch(), "PayloadByNumber" is called when fetching the overlapped blocks.
				// so blocks at 14, 20 could be called, depends on the order of batches
				l2Client.Mock.On("PayloadByNumber", uint64(i+1)).Return(&payload, &nilErr).Maybe()
			}
		}
	}

	bq := NewBatchQueue(log, cfg, input, &l2Client)
	_ = bq.Reset(context.Background(), l1[1], eth.SystemConfig{})

	for i := 0; i < len(expectedOutputBatches); i++ {
		expectedOutput := expectedOutputBatches[i]
		if expectedOutput != nil && uint64(expectedOutput.EpochNum) == l1[inputOriginNumber].Number {
			// Advance ChannelInReader origin if needed
			inputOriginNumber += 1
			input.origin = l1[inputOriginNumber]
		}
		var b *SingularBatch
		var e error
		for j := 0; j < len(expectedOutputBatches); j++ {
			// Multiple NextBatch() executions may be required because the order of input is shuffled
			b, _, e = bq.NextBatch(context.Background(), safeHead)
			if !errors.Is(e, NotEnoughData) {
				break
			}
		}
		require.ErrorIs(t, e, expectedOutputErrors[i])
		if b == nil {
			require.Nil(t, expectedOutput)
		} else {
			require.Equal(t, expectedOutput, b)
			safeHead.Number += 1
			safeHead.Time += cfg.BlockTime
			safeHead.Hash = mockHash(b.Timestamp, 2)
			safeHead.L1Origin = b.Epoch()
		}
	}

	l2Client.Mock.AssertExpectations(t)
}

func TestBatchQueueResetSpan(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	chainId := big.NewInt(1234)
	l1 := L1Chain([]uint64{0, 4, 8})
	safeHead := eth.L2BlockRef{
		Hash:           mockHash(0, 2),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           0,
		L1Origin:       l1[0].ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
		DeltaTime:         getDeltaTime(SpanBatchType),
		L2ChainID:         chainId,
	}

	singularBatches := []*SingularBatch{
		b(cfg.L2ChainID, 2, l1[0]),
		b(cfg.L2ChainID, 4, l1[1]),
		b(cfg.L2ChainID, 6, l1[1]),
		b(cfg.L2ChainID, 8, l1[2]),
	}

	input := &fakeBatchQueueInput{
		batches: []Batch{NewSpanBatch(singularBatches)},
		errors:  []error{nil},
		origin:  l1[2],
	}
	l2Client := testutils.MockL2Client{}
	bq := NewBatchQueue(log, cfg, input, &l2Client)
	bq.l1Blocks = l1 // Set enough l1 blocks to derive span batch

	// This NextBatch() will derive the span batch, return the first singular batch and save rest of batches in span.
	nextBatch, _, err := bq.NextBatch(context.Background(), safeHead)
	require.NoError(t, err)
	require.Equal(t, nextBatch, singularBatches[0])
	require.Equal(t, len(bq.nextSpan), len(singularBatches)-1)
	// batch queue's epoch should not be advanced until the entire span batch is returned
	require.Equal(t, bq.l1Blocks[0], l1[0])

	// This NextBatch() will return the second singular batch.
	safeHead.Number += 1
	safeHead.Time += cfg.BlockTime
	safeHead.Hash = mockHash(nextBatch.Timestamp, 2)
	safeHead.L1Origin = nextBatch.Epoch()
	nextBatch, _, err = bq.NextBatch(context.Background(), safeHead)
	require.NoError(t, err)
	require.Equal(t, nextBatch, singularBatches[1])
	require.Equal(t, len(bq.nextSpan), len(singularBatches)-2)
	// batch queue's epoch should not be advanced until the entire span batch is returned
	require.Equal(t, bq.l1Blocks[0], l1[0])

	// Call NextBatch() with stale safeHead. It means the second batch failed to be processed.
	// Batch queue should drop the entire span batch.
	nextBatch, _, err = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, nextBatch)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, len(bq.nextSpan), 0)
}
