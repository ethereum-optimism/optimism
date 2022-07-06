package derive

import (
	"context"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum"
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
	// Advance SafeL2Head
	f.safeL2Head.Time = batch.Timestamp
	f.safeL2Head.L1Origin.Number = uint64(batch.EpochNum)
}

func (f *fakeBatchQueueOutput) SafeL2Head() eth.L2BlockRef {
	return f.safeL2Head
}

func (f *fakeBatchQueueOutput) Progress() Progress {
	return f.progress
}

func b(timestamp uint64, epoch eth.L1BlockRef) *BatchData {
	rng := rand.New(rand.NewSource(int64(timestamp)))
	data := testutils.RandomData(rng, 20)
	return &BatchData{BatchV1{
		Timestamp:    timestamp,
		EpochNum:     rollup.Epoch(epoch.Number),
		EpochHash:    epoch.Hash,
		Transactions: []hexutil.Bytes{data},
	}}
}

func L1Chain(l1Times []uint64) []eth.L1BlockRef {
	var out []eth.L1BlockRef
	var parentHash [32]byte
	for i, time := range l1Times {
		hash := [32]byte{byte(i)}
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

type fakeL1Fetcher struct {
	l1 []eth.L1BlockRef
}

func (f *fakeL1Fetcher) L1BlockRefByNumber(_ context.Context, n uint64) (eth.L1BlockRef, error) {
	if n >= uint64(len(f.l1)) {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return f.l1[int(n)], nil
}

func TestBatchQueueEager(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	next := &fakeBatchQueueOutput{
		safeL2Head: eth.L2BlockRef{
			Number:   0,
			Time:     10,
			L1Origin: eth.BlockID{Number: 0},
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

	l1 := L1Chain([]uint64{10, 20, 30})

	fetcher := fakeL1Fetcher{l1: l1}
	bq := NewBatchQueue(log, cfg, &fetcher, next)

	prevProgress := Progress{
		Origin: l1[0],
		Closed: false,
	}

	// Setup progress
	bq.progress.Closed = true
	err := bq.Step(context.Background(), prevProgress)
	require.Nil(t, err)

	// Add batches
	batches := []*BatchData{b(12, l1[0]), b(14, l1[0])}
	for _, batch := range batches {
		err := bq.AddBatch(batch)
		require.Nil(t, err)
	}
	// Step
	for {
		if err := bq.Step(context.Background(), prevProgress); err == io.EOF {
			break
		} else {
			require.Nil(t, err)
		}
	}
	// Verify Output
	require.Equal(t, batches, next.batches)
}

func TestBatchQueueFull(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	next := &fakeBatchQueueOutput{
		safeL2Head: eth.L2BlockRef{
			Number:   0,
			Time:     10,
			L1Origin: eth.BlockID{Number: 0},
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

	l1 := L1Chain([]uint64{10, 15, 20})

	fetcher := fakeL1Fetcher{l1: l1}
	bq := NewBatchQueue(log, cfg, &fetcher, next)

	// Start with open previous & closed self.
	// Then this stage is opened at the first step.
	bq.progress.Closed = true
	prevProgress := Progress{
		Origin: l1[0],
		Closed: false,
	}

	// Do the bq open
	err := bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Add batches
	batches := []*BatchData{b(14, l1[0]), b(16, l1[0]), b(18, l1[1])}
	for _, batch := range batches {
		err := bq.AddBatch(batch)
		require.Nil(t, err)
	}
	// Missing first batch
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, io.EOF)

	// Close previous to close bq
	prevProgress.Closed = true
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Open previous to open bq with the new inclusion block
	prevProgress.Closed = false
	prevProgress.Origin = l1[1]
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Close previous to close bq (for epoch 2)
	prevProgress.Closed = true
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Open previous to open bq with the new inclusion block (epoch 2)
	prevProgress.Closed = false
	prevProgress.Origin = l1[2]
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Finally add batch
	firstBatch := b(12, l1[0])
	err = bq.AddBatch(firstBatch)
	require.Equal(t, err, nil)

	// Close the origin
	prevProgress.Closed = true
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Step, but should have full epoch now
	for {
		if err := bq.Step(context.Background(), prevProgress); err == io.EOF {
			break
		} else {
			require.Nil(t, err)
		}
	}
	// Verify Output
	var final []*BatchData
	final = append(final, firstBatch)
	final = append(final, batches...)
	require.Equal(t, final, next.batches)
}

func TestBatchQueueMissing(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	next := &fakeBatchQueueOutput{
		safeL2Head: eth.L2BlockRef{
			Number:   0,
			Time:     10,
			L1Origin: eth.BlockID{Number: 0},
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

	l1 := L1Chain([]uint64{10, 15, 20})

	fetcher := fakeL1Fetcher{l1: l1}
	bq := NewBatchQueue(log, cfg, &fetcher, next)

	// Start with open previous & closed self.
	// Then this stage is opened at the first step.
	bq.progress.Closed = true
	prevProgress := Progress{
		Origin: l1[0],
		Closed: false,
	}

	// Do the bq open
	err := bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Add batches
	// NB: The batch at 18 is skipped to skip over the ability to
	// do eager batch processing for that batch. This test checks
	// that batch timestamp 12 & 14 is created & 16 is used.
	batches := []*BatchData{b(16, l1[0]), b(20, l1[1])}
	for _, batch := range batches {
		err := bq.AddBatch(batch)
		require.Nil(t, err)
	}
	// Missing first batch
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, io.EOF)

	// Close previous to close bq
	prevProgress.Closed = true
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Open previous to open bq with the new inclusion block
	prevProgress.Closed = false
	prevProgress.Origin = l1[1]
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Close previous to close bq (for epoch 2)
	prevProgress.Closed = true
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Open previous to open bq with the new inclusion block (epoch 2)
	prevProgress.Closed = false
	prevProgress.Origin = l1[2]
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, false)

	// Close the origin
	prevProgress.Closed = true
	err = bq.Step(context.Background(), prevProgress)
	require.Equal(t, err, nil)
	require.Equal(t, bq.progress.Closed, true)

	// Step, but should have full epoch now + fill missing
	for {
		if err := bq.Step(context.Background(), prevProgress); err == io.EOF {
			break
		} else {
			require.Nil(t, err)
		}
	}
	// TODO: Maybe check actuall batch validity better
	require.Equal(t, 3, len(next.batches))
}
