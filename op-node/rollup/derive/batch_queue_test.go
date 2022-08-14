package derive

import (
	"context"
	"encoding/binary"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// fakeBatchQueueOutput fakes the next stage (receive only) for the batch queue
// It tracks the progress state of the next stage.
// Upon receiving a batch, relevant characteristic of safeL2Head are immediately advanced.
type fakeBatchQueueOutput struct {
	progress   Progress
	batches    []*BatchData
	safeL2Head eth.L2BlockRef
}

var _ BatchQueueOutput = (*fakeBatchQueueOutput)(nil)

func (f *fakeBatchQueueOutput) AddBatch(batch *BatchData) {
	f.batches = append(f.batches, batch)
	if batch.ParentHash != f.safeL2Head.Hash {
		panic("batch has wrong parent hash")
	}
	newEpoch := f.safeL2Head.L1Origin.Hash != batch.EpochHash
	// Advance SafeL2Head
	f.safeL2Head.Time = batch.Timestamp
	f.safeL2Head.L1Origin.Number = uint64(batch.EpochNum)
	f.safeL2Head.L1Origin.Hash = batch.EpochHash
	if newEpoch {
		f.safeL2Head.SequenceNumber = 0
	} else {
		f.safeL2Head.SequenceNumber += 1
	}
	f.safeL2Head.ParentHash = batch.ParentHash
	f.safeL2Head.Hash = mockHash(batch.Timestamp, 2)
}

func (f *fakeBatchQueueOutput) SafeL2Head() eth.L2BlockRef {
	return f.safeL2Head
}

func (f *fakeBatchQueueOutput) Progress() Progress {
	return f.progress
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

func TestBatchQueueEager(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	l1 := L1Chain([]uint64{10, 20, 30})
	next := &fakeBatchQueueOutput{
		safeL2Head: eth.L2BlockRef{
			Hash:           mockHash(10, 2),
			Number:         0,
			ParentHash:     common.Hash{},
			Time:           10,
			L1Origin:       l1[0].ID(),
			SequenceNumber: 0,
		},
		progress: Progress{
			Origin: l1[0],
			Closed: false,
		},
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     30,
	}

	bq := NewBatchQueue(log, cfg, next)
	require.Equal(t, io.EOF, bq.ResetStep(context.Background(), nil), "reset should complete without l1 fetcher, single step")

	// We start with an open L1 origin as progress in the first step
	progress := bq.progress
	require.Equal(t, bq.progress.Closed, false)

	// Add batches
	batches := []*BatchData{b(12, l1[0]), b(14, l1[0])}
	for _, batch := range batches {
		bq.AddBatch(batch)
	}
	// Step
	require.NoError(t, RepeatStep(t, bq.Step, progress, 10))

	// Verify Output
	require.Equal(t, batches, next.batches)
}

func TestBatchQueueFull(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	l1 := L1Chain([]uint64{10, 15, 20})
	next := &fakeBatchQueueOutput{
		safeL2Head: eth.L2BlockRef{
			Hash:           mockHash(10, 2),
			Number:         0,
			ParentHash:     common.Hash{},
			Time:           10,
			L1Origin:       l1[0].ID(),
			SequenceNumber: 0,
		},
		progress: Progress{
			Origin: l1[0],
			Closed: false,
		},
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     2,
	}

	bq := NewBatchQueue(log, cfg, next)
	require.Equal(t, io.EOF, bq.ResetStep(context.Background(), nil), "reset should complete without l1 fetcher, single step")

	// We start with an open L1 origin as progress in the first step
	progress := bq.progress
	require.Equal(t, bq.progress.Closed, false)

	// Add batches
	batches := []*BatchData{b(14, l1[0]), b(16, l1[0]), b(18, l1[1])}
	for _, batch := range batches {
		bq.AddBatch(batch)
	}
	// Missing first batch
	err := bq.Step(context.Background(), progress)
	require.Equal(t, err, io.EOF)

	// Close previous to close bq
	progress.Closed = true
	err = bq.Step(context.Background(), progress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Open previous to open bq with the new inclusion block
	progress.Closed = false
	progress.Origin = l1[1]
	err = bq.Step(context.Background(), progress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Close previous to close bq (for epoch 2)
	progress.Closed = true
	err = bq.Step(context.Background(), progress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Open previous to open bq with the new inclusion block (epoch 2)
	progress.Closed = false
	progress.Origin = l1[2]
	err = bq.Step(context.Background(), progress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Finally add batch
	firstBatch := b(12, l1[0])
	bq.AddBatch(firstBatch)

	// Close the origin
	progress.Closed = true
	err = bq.Step(context.Background(), progress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Step, but should have full epoch now
	require.NoError(t, RepeatStep(t, bq.Step, progress, 10))

	// Verify Output
	var final []*BatchData
	final = append(final, firstBatch)
	final = append(final, batches...)
	require.Equal(t, final, next.batches)
}

func TestBatchQueueMissing(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	l1 := L1Chain([]uint64{10, 15, 20})
	next := &fakeBatchQueueOutput{
		safeL2Head: eth.L2BlockRef{
			Hash:           mockHash(10, 2),
			Number:         0,
			ParentHash:     common.Hash{},
			Time:           10,
			L1Origin:       l1[0].ID(),
			SequenceNumber: 0,
		},
		progress: Progress{
			Origin: l1[0],
			Closed: false,
		},
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 10,
		},
		BlockTime:         2,
		MaxSequencerDrift: 600,
		SeqWindowSize:     2,
	}

	bq := NewBatchQueue(log, cfg, next)
	require.Equal(t, io.EOF, bq.ResetStep(context.Background(), nil), "reset should complete without l1 fetcher, single step")

	// We start with an open L1 origin as progress in the first step
	progress := bq.progress
	require.Equal(t, bq.progress.Closed, false)

	// The batches at 18 and 20 are skipped to stop 22 from being eagerly processed.
	// This test checks that batch timestamp 12 & 14 are created, 16 is used, and 18 is advancing the epoch.
	// Due to the large sequencer time drift 16 is perfectly valid to have epoch 0 as origin.
	batches := []*BatchData{b(16, l1[0]), b(22, l1[1])}
	for _, batch := range batches {
		bq.AddBatch(batch)
	}
	// Missing first batches with timestamp 12 and 14, nothing to do yet.
	err := bq.Step(context.Background(), progress)
	require.Equal(t, err, io.EOF)

	// Close l1[0]
	progress.Closed = true
	require.NoError(t, RepeatStep(t, bq.Step, progress, 10))
	require.Equal(t, bq.progress.Closed, true)

	// Open l1[1]
	progress.Closed = false
	progress.Origin = l1[1]
	require.NoError(t, RepeatStep(t, bq.Step, progress, 10))
	require.Equal(t, bq.progress.Closed, false)
	require.Empty(t, next.batches, "no batches yet, sequence window did not expire, waiting for 12 and 14")

	// Close l1[1]
	progress.Closed = true
	require.NoError(t, RepeatStep(t, bq.Step, progress, 10))
	require.Equal(t, bq.progress.Closed, true)

	// Open l1[2]
	progress.Closed = false
	progress.Origin = l1[2]
	require.NoError(t, RepeatStep(t, bq.Step, progress, 10))
	require.Equal(t, bq.progress.Closed, false)

	// Close l1[2], this is the moment that l1[0] expires and empty batches 12 and 14 can be created,
	// and batch 16 can then be used.
	progress.Closed = true
	require.NoError(t, RepeatStep(t, bq.Step, progress, 10))
	require.Equal(t, bq.progress.Closed, true)
	require.Equal(t, 4, len(next.batches), "expecting empty batches with timestamp 12 and 14 to be created and existing batch 16 to follow")
	require.Equal(t, uint64(12), next.batches[0].Timestamp)
	require.Equal(t, uint64(14), next.batches[1].Timestamp)
	require.Equal(t, batches[0], next.batches[2])
	require.Equal(t, uint64(18), next.batches[3].Timestamp)
	require.Equal(t, rollup.Epoch(1), next.batches[3].EpochNum)
}
