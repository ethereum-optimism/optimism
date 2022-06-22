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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// fakeBatcheQueueOutput fakes the next stage (receive only) for the batch queue
// It tracks the Open/Close state (received from the batch queue) as wells as
// saving every batch (in order) that is sent to it.
// Upon receiving a batch, the SafeL2Head is immediately advanced (only relevant characteristics).
type fakeBatcheQueueOutput struct {
	originOpen bool
	origin     eth.L1BlockRef
	batches    []*BatchData
	safeL2Head eth.L2BlockRef
}

func (f *fakeBatcheQueueOutput) OpenOrigin(origin eth.L1BlockRef) {
	f.originOpen = true
	f.origin = origin
}

func (f *fakeBatcheQueueOutput) CloseOrigin() {
	f.originOpen = false
	f.origin = eth.L1BlockRef{}
}

func (f *fakeBatcheQueueOutput) AddBatch(batch *BatchData) {
	f.batches = append(f.batches, batch)
	// Advance SafeL2Head
	f.safeL2Head.Number = batch.BlockNumber
	f.safeL2Head.Time = batch.Timestamp
	f.safeL2Head.L1Origin.Number = uint64(batch.Epoch)

}

func (f *fakeBatcheQueueOutput) SafeL2Head() eth.L2BlockRef {
	return f.safeL2Head
}

func b(number, timestamp, epoch uint64) *BatchData {
	rng := rand.New(rand.NewSource(1234))
	data := testutils.RandomData(rng, int(number))
	return &BatchData{BatchV1{
		BlockNumber:  number,
		Timestamp:    timestamp,
		Epoch:        rollup.Epoch(epoch),
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

func TestBatcheQueueEager(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	next := &fakeBatcheQueueOutput{
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
	bq := NewBatchQueue(log, cfg, next)

	l1 := L1Chain([]uint64{10, 20, 30})

	// Open
	bq.OpenOrigin(l1[0])
	// Add batches
	batches := []*BatchData{b(1, 12, 0), b(2, 14, 0)}
	for _, batch := range batches {
		err := bq.AddBatch(batch)
		require.Nil(t, err)
	}
	// Step
	for {
		if err := bq.Step(context.Background()); err == io.EOF {
			break
		} else {
			require.Nil(t, err)
		}
	}
	// Verify Output
	require.Equal(t, batches, next.batches)
}

func TestBatcheQueueFull(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	next := &fakeBatcheQueueOutput{
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
	bq := NewBatchQueue(log, cfg, next)

	l1 := L1Chain([]uint64{10, 15, 20})

	// Open
	bq.OpenOrigin(l1[0])
	// Add batches
	batches := []*BatchData{b(2, 14, 0), b(3, 16, 1), b(4, 18, 1)}
	for _, batch := range batches {
		err := bq.AddBatch(batch)
		require.Nil(t, err)
	}
	// Missing first batch
	err := bq.Step(context.Background())
	require.Equal(t, err, io.EOF)
	bq.CloseOrigin()
	bq.OpenOrigin(l1[1])
	// Still missing first batch
	err = bq.Step(context.Background())
	require.Equal(t, err, io.EOF)
	bq.CloseOrigin()
	// Open up origin that completes the seq window
	bq.OpenOrigin(l1[2])
	firstBatch := b(1, 12, 0)
	err = bq.AddBatch(firstBatch)
	require.Nil(t, err)
	// Step
	for {
		if err := bq.Step(context.Background()); err == io.EOF {
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

func TestBatcheQueueMissing(t *testing.T) {
	log := testlog.Logger(t, log.LvlTrace)
	next := &fakeBatcheQueueOutput{
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
	bq := NewBatchQueue(log, cfg, next)

	l1 := L1Chain([]uint64{10, 16, 20})

	// Open
	bq.OpenOrigin(l1[0])

	// Missing first batch
	err := bq.Step(context.Background())
	require.Equal(t, err, io.EOF)
	bq.CloseOrigin()
	bq.OpenOrigin(l1[1])
	// No Seq window yet
	err = bq.Step(context.Background())
	require.Equal(t, err, io.EOF)
	bq.CloseOrigin()
	// Open up origin that completes the seq window
	bq.OpenOrigin(l1[2])
	// Step
	for {
		if err := bq.Step(context.Background()); err == io.EOF {
			break
		} else {
			require.Nil(t, err)
		}
	}
	// Verify Output
	require.Equal(t, 2, len(next.batches))
}
