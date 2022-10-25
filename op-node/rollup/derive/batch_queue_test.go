package derive

import (
	"context"
	"encoding/binary"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

type fakeBatchQueueInput struct {
	i       int
	batches []*BatchData
	errors  []error
	origin  eth.L1BlockRef
}

func (f *fakeBatchQueueInput) Origin() eth.L1BlockRef {
	return f.origin
}

func (f *fakeBatchQueueInput) NextBatch(ctx context.Context) (*BatchData, error) {
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

func b(timestamp uint64, epoch eth.L1BlockRef) *BatchData {
	rng := rand.New(rand.NewSource(int64(timestamp)))
	data := testutils.RandomData(rng, 20)
	return &BatchData{BatchV1{
		ParentHash:   mockHash(timestamp-2, 2),
		Timestamp:    timestamp,
		EpochNum:     rollup.Epoch(epoch.Number),
		EpochHash:    epoch.Hash,
		Transactions: []hexutil.Bytes{data},
	}}
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

// TestBatchQueueNewOrigin tests that the batch queue properly saves the new origin
// when the safehead's origin is ahead of the pipeline's origin (as is after a reset).
// This issue was fixed in https://github.com/ethereum-optimism/optimism/pull/3694
func TestBatchQueueNewOrigin(t *testing.T) {
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
	}

	input := &fakeBatchQueueInput{
		batches: []*BatchData{nil},
		errors:  []error{io.EOF},
		origin:  l1[0],
	}

	bq := NewBatchQueue(log, cfg, input)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})
	require.Equal(t, []eth.L1BlockRef{l1[0]}, bq.l1Blocks)

	// Prev Origin: 0; Safehead Origin: 2; Internal Origin: 0
	// Should return no data but keep the same origin
	data, err := bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, data)
	require.Equal(t, io.EOF, err)
	require.Equal(t, []eth.L1BlockRef{l1[0]}, bq.l1Blocks)
	require.Equal(t, l1[0], bq.origin)

	// Prev Origin: 1; Safehead Origin: 2; Internal Origin: 0
	// Should wipe l1blocks + advance internal origin
	input.origin = l1[1]
	data, err = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, data)
	require.Equal(t, io.EOF, err)
	require.Empty(t, bq.l1Blocks)
	require.Equal(t, l1[1], bq.origin)

	// Prev Origin: 2; Safehead Origin: 2; Internal Origin: 1
	// Should add to l1Blocks + advance internal origin
	input.origin = l1[2]
	data, err = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, data)
	require.Equal(t, io.EOF, err)
	require.Equal(t, []eth.L1BlockRef{l1[2]}, bq.l1Blocks)
	require.Equal(t, l1[2], bq.origin)
}

// TestBatchQueueEager adds a bunch of contiguous batches and asserts that
// enough calls to `NextBatch` return all of those batches.
func TestBatchQueueEager(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{10, 20, 30})
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
	}

	batches := []*BatchData{b(12, l1[0]), b(14, l1[0]), b(16, l1[0]), b(18, l1[0]), b(20, l1[0]), b(22, l1[0]), b(24, l1[1]), nil}
	errors := []error{nil, nil, nil, nil, nil, nil, nil, io.EOF}

	input := &fakeBatchQueueInput{
		batches: batches,
		errors:  errors,
		origin:  l1[0],
	}

	bq := NewBatchQueue(log, cfg, input)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})
	// Advance the origin
	input.origin = l1[1]

	for i := 0; i < len(batches); i++ {
		b, e := bq.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, e, errors[i])
		require.Equal(t, batches[i], b)

		if b != nil {
			safeHead.Number += 1
			safeHead.Time += 2
			safeHead.Hash = mockHash(b.Timestamp, 2)
			safeHead.L1Origin = b.Epoch()
		}
	}
}

func TestBatchQueueMissing(t *testing.T) {
	log := testlog.Logger(t, log.LvlCrit)
	l1 := L1Chain([]uint64{10, 15, 20})
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
	}

	// The batches at 18 and 20 are skipped to stop 22 from being eagerly processed.
	// This test checks that batch timestamp 12 & 14 are created, 16 is used, and 18 is advancing the epoch.
	// Due to the large sequencer time drift 16 is perfectly valid to have epoch 0 as origin.
	batches := []*BatchData{b(16, l1[0]), b(22, l1[1])}
	errors := []error{nil, nil}

	input := &fakeBatchQueueInput{
		batches: batches,
		errors:  errors,
		origin:  l1[0],
	}

	bq := NewBatchQueue(log, cfg, input)
	_ = bq.Reset(context.Background(), l1[0], eth.SystemConfig{})

	for i := 0; i < len(batches); i++ {
		b, e := bq.NextBatch(context.Background(), safeHead)
		require.ErrorIs(t, e, NotEnoughData)
		require.Nil(t, b)
	}

	// advance origin. Underlying stage still has no more batches
	// This is not enough to auto advance yet
	input.origin = l1[1]
	b, e := bq.NextBatch(context.Background(), safeHead)
	require.ErrorIs(t, e, io.EOF)
	require.Nil(t, b)

	// Advance the origin. At this point batch timestamps 12 and 14 will be created
	input.origin = l1[2]

	// Check for a generated batch at t = 12
	b, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b.Timestamp, uint64(12))
	require.Empty(t, b.BatchV1.Transactions)
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)

	// Check for generated batch at t = 14
	b, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b.Timestamp, uint64(14))
	require.Empty(t, b.BatchV1.Transactions)
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)

	// Check for the inputted batch at t = 16
	b, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b, batches[0])
	safeHead.Number += 1
	safeHead.Time += 2
	safeHead.Hash = mockHash(b.Timestamp, 2)

	// Check for the generated batch at t = 18. This batch advances the epoch
	b, e = bq.NextBatch(context.Background(), safeHead)
	require.Nil(t, e)
	require.Equal(t, b.Timestamp, uint64(18))
	require.Empty(t, b.BatchV1.Transactions)
	require.Equal(t, rollup.Epoch(1), b.EpochNum)
}
