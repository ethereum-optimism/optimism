package derive

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type ValidBatchTestCase struct {
	Name       string
	L1Blocks   []eth.L1BlockRef
	L2SafeHead eth.L2BlockRef
	Batch      BatchWithL1InclusionBlock
	Expected   BatchValidity
}

type SpanBatchHardForkTestCase struct {
	Name          string
	L1Blocks      []eth.L1BlockRef
	L2SafeHead    eth.L2BlockRef
	Batch         BatchWithL1InclusionBlock
	Expected      BatchValidity
	SpanBatchTime uint64
}

var HashA = common.Hash{0x0a}
var HashB = common.Hash{0x0b}

func TestValidSingularBatch(t *testing.T) {
	conf := rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 31, // a genesis time that itself does not align to make it more interesting
		},
		BlockTime:         2,
		SeqWindowSize:     4,
		MaxSequencerDrift: 6,
		// other config fields are ignored and can be left empty.
	}

	rng := rand.New(rand.NewSource(1234))
	l1A := testutils.RandomBlockRef(rng)
	l1B := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1A.Number + 1,
		ParentHash: l1A.Hash,
		Time:       l1A.Time + 7,
	}
	l1C := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1B.Number + 1,
		ParentHash: l1B.Hash,
		Time:       l1B.Time + 7,
	}
	l1D := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1C.Number + 1,
		ParentHash: l1C.Hash,
		Time:       l1C.Time + 7,
	}
	l1E := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1D.Number + 1,
		ParentHash: l1D.Hash,
		Time:       l1D.Time + 7,
	}
	l1F := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1E.Number + 1,
		ParentHash: l1E.Hash,
		Time:       l1E.Time + 7,
	}

	l2A0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         100,
		ParentHash:     testutils.RandomHash(rng),
		Time:           l1A.Time,
		L1Origin:       l1A.ID(),
		SequenceNumber: 0,
	}

	l2A1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A0.Number + 1,
		ParentHash:     l2A0.Hash,
		Time:           l2A0.Time + conf.BlockTime,
		L1Origin:       l1A.ID(),
		SequenceNumber: 1,
	}

	l2A2 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A1.Number + 1,
		ParentHash:     l2A1.Hash,
		Time:           l2A1.Time + conf.BlockTime,
		L1Origin:       l1A.ID(),
		SequenceNumber: 2,
	}

	l2A3 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A2.Number + 1,
		ParentHash:     l2A2.Hash,
		Time:           l2A2.Time + conf.BlockTime,
		L1Origin:       l1A.ID(),
		SequenceNumber: 3,
	}

	l2B0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A3.Number + 1,
		ParentHash:     l2A3.Hash,
		Time:           l2A3.Time + conf.BlockTime, // 8 seconds larger than l1A0, 1 larger than origin
		L1Origin:       l1B.ID(),
		SequenceNumber: 0,
	}

	l1X := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     42,
		ParentHash: testutils.RandomHash(rng),
		Time:       10_000,
	}
	l1Y := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1X.Number + 1,
		ParentHash: l1X.Hash,
		Time:       l1X.Time + 12,
	}
	l1Z := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1Y.Number + 1,
		ParentHash: l1Y.Hash,
		Time:       l1Y.Time + 12,
	}
	l2X0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         1000,
		ParentHash:     testutils.RandomHash(rng),
		Time:           10_000 + 12 + 6 - 1, // add one block, and you get ahead of next l1 block by more than the drift
		L1Origin:       l1X.ID(),
		SequenceNumber: 0,
	}
	l2Y0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2X0.Number + 1,
		ParentHash:     l2X0.Hash,
		Time:           l2X0.Time + conf.BlockTime, // exceeds sequencer time drift, forced to be empty block
		L1Origin:       l1Y.ID(),
		SequenceNumber: 0,
	}

	l2A4 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A3.Number + 1,
		ParentHash:     l2A3.Hash,
		Time:           l2A3.Time + conf.BlockTime, // 4*2 = 8, higher than seq time drift
		L1Origin:       l1A.ID(),
		SequenceNumber: 4,
	}

	l1BLate := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1A.Number + 1,
		ParentHash: l1A.Hash,
		Time:       l2A4.Time + 1, // too late for l2A4 to adopt yet
	}

	testCases := []ValidBatchTestCase{
		{
			Name:       "missing L1 info",
			L1Blocks:   []eth.L1BlockRef{},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash:   l2A1.ParentHash,
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A1.Time,
					Transactions: nil,
				},
			},
			Expected: BatchUndecided,
		},
		{
			Name:       "future timestamp",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash:   l2A1.ParentHash,
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A1.Time + 1, // 1 too high
					Transactions: nil,
				},
			},
			Expected: BatchFuture,
		},
		{
			Name:       "old timestamp",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash:   l2A1.ParentHash,
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A0.Time, // repeating the same time
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "misaligned timestamp",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash:   l2A1.ParentHash,
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A1.Time - 1, // block time is 2, so this is 1 too low
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "invalid parent block hash",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash:   testutils.RandomHash(rng),
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A1.Time,
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequence window expired",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C, l1D, l1E, l1F},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1F, // included in 5th block after epoch of batch, while seq window is 4
				Batch: &SingularBatch{
					ParentHash:   l2A1.ParentHash,
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A1.Time,
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "epoch too old, but good parent hash and timestamp", // repeat of now outdated l2A3 data
			L1Blocks:   []eth.L1BlockRef{l1B, l1C, l1D},
			L2SafeHead: l2B0, // we already moved on to B
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: &SingularBatch{
					ParentHash:   l2B0.Hash,                          // build on top of safe head to continue
					EpochNum:     rollup.Epoch(l2A3.L1Origin.Number), // epoch A is no longer valid
					EpochHash:    l2A3.L1Origin.Hash,
					Timestamp:    l2B0.Time + conf.BlockTime, // pass the timestamp check to get too epoch check
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "insufficient L1 info for eager derivation",
			L1Blocks:   []eth.L1BlockRef{l1A}, // don't know about l1B yet
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: &SingularBatch{
					ParentHash:   l2B0.ParentHash,
					EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
					EpochHash:    l2B0.L1Origin.Hash,
					Timestamp:    l2B0.Time,
					Transactions: nil,
				},
			},
			Expected: BatchUndecided,
		},
		{
			Name:       "epoch too new",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C, l1D},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1D,
				Batch: &SingularBatch{
					ParentHash:   l2B0.ParentHash,
					EpochNum:     rollup.Epoch(l1C.Number), // invalid, we need to adopt epoch B before C
					EpochHash:    l1C.Hash,
					Timestamp:    l2B0.Time,
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "epoch hash wrong",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: &SingularBatch{
					ParentHash:   l2B0.ParentHash,
					EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
					EpochHash:    l1A.Hash, // invalid, epoch hash should be l1B
					Timestamp:    l2B0.Time,
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequencer time drift on same epoch with non-empty txs",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
					ParentHash:   l2A4.ParentHash,
					EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
					EpochHash:    l2A4.L1Origin.Hash,
					Timestamp:    l2A4.Time,
					Transactions: []hexutil.Bytes{[]byte("sequencer should not include this tx")},
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequencer time drift on changing epoch with non-empty txs",
			L1Blocks:   []eth.L1BlockRef{l1X, l1Y, l1Z},
			L2SafeHead: l2X0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1Z,
				Batch: &SingularBatch{
					ParentHash:   l2Y0.ParentHash,
					EpochNum:     rollup.Epoch(l2Y0.L1Origin.Number),
					EpochHash:    l2Y0.L1Origin.Hash,
					Timestamp:    l2Y0.Time, // valid, but more than 6 ahead of l1Y.Time
					Transactions: []hexutil.Bytes{[]byte("sequencer should not include this tx")},
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and late next epoch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1BLate},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1BLate,
				Batch: &SingularBatch{ // l2A4 time < l1BLate time, so we cannot adopt origin B yet
					ParentHash:   l2A4.ParentHash,
					EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
					EpochHash:    l2A4.L1Origin.Hash,
					Timestamp:    l2A4.Time,
					Transactions: nil,
				},
			},
			Expected: BatchAccept, // accepted because empty & preserving L2 time invariant
		},
		{
			Name:       "sequencer time drift on changing epoch with empty txs",
			L1Blocks:   []eth.L1BlockRef{l1X, l1Y, l1Z},
			L2SafeHead: l2X0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1Z,
				Batch: &SingularBatch{
					ParentHash:   l2Y0.ParentHash,
					EpochNum:     rollup.Epoch(l2Y0.L1Origin.Number),
					EpochHash:    l2Y0.L1Origin.Hash,
					Timestamp:    l2Y0.Time, // valid, but more than 6 ahead of l1Y.Time
					Transactions: nil,
				},
			},
			Expected: BatchAccept, // accepted because empty & still advancing epoch
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and no next epoch in sight yet",
			L1Blocks:   []eth.L1BlockRef{l1A},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
					ParentHash:   l2A4.ParentHash,
					EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
					EpochHash:    l2A4.L1Origin.Hash,
					Timestamp:    l2A4.Time,
					Transactions: nil,
				},
			},
			Expected: BatchUndecided, // we have to wait till the next epoch is in sight to check the time
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and but in-sight epoch that invalidates it",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: &SingularBatch{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
					ParentHash:   l2A4.ParentHash,
					EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
					EpochHash:    l2A4.L1Origin.Hash,
					Timestamp:    l2A4.Time,
					Transactions: nil,
				},
			},
			Expected: BatchDrop, // dropped because it could have advanced the epoch to B
		},
		{
			Name:       "empty tx included",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash: l2A1.ParentHash,
					EpochNum:   rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:  l2A1.L1Origin.Hash,
					Timestamp:  l2A1.Time,
					Transactions: []hexutil.Bytes{
						[]byte{}, // empty tx data
					},
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "deposit tx included",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash: l2A1.ParentHash,
					EpochNum:   rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:  l2A1.L1Origin.Hash,
					Timestamp:  l2A1.Time,
					Transactions: []hexutil.Bytes{
						[]byte{types.DepositTxType, 0}, // piece of data alike to a deposit
					},
				},
			},
			Expected: BatchDrop,
		},
		{
			Name:       "valid batch same epoch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash: l2A1.ParentHash,
					EpochNum:   rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:  l2A1.L1Origin.Hash,
					Timestamp:  l2A1.Time,
					Transactions: []hexutil.Bytes{
						[]byte{0x02, 0x42, 0x13, 0x37},
						[]byte{0x02, 0xde, 0xad, 0xbe, 0xef},
					},
				},
			},
			Expected: BatchAccept,
		},
		{
			Name:       "valid batch changing epoch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: &SingularBatch{
					ParentHash: l2B0.ParentHash,
					EpochNum:   rollup.Epoch(l2B0.L1Origin.Number),
					EpochHash:  l2B0.L1Origin.Hash,
					Timestamp:  l2B0.Time,
					Transactions: []hexutil.Bytes{
						[]byte{0x02, 0x42, 0x13, 0x37},
						[]byte{0x02, 0xde, 0xad, 0xbe, 0xef},
					},
				},
			},
			Expected: BatchAccept,
		},
		{
			Name:       "batch with L2 time before L1 time",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{ // we build l2B0', which starts a new epoch too early
					ParentHash:   l2A2.Hash,
					EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
					EpochHash:    l2B0.L1Origin.Hash,
					Timestamp:    l2A2.Time + conf.BlockTime,
					Transactions: nil,
				},
			},
			Expected: BatchDrop,
		},
	}

	// Log level can be increased for debugging purposes
	logger := testlog.Logger(t, log.LvlError)

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			ctx := context.Background()
			validity := CheckBatch(ctx, &conf, logger, testCase.L1Blocks, testCase.L2SafeHead, &testCase.Batch, nil)
			require.Equal(t, testCase.Expected, validity, "batch check must return expected validity level")
		})
	}
}

func TestValidSpanBatch(t *testing.T) {
	minTs := uint64(0)
	conf := rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 31, // a genesis time that itself does not align to make it more interesting
		},
		BlockTime:         2,
		SeqWindowSize:     4,
		MaxSequencerDrift: 6,
		SpanBatchTime:     &minTs,
		// other config fields are ignored and can be left empty.
	}

	rng := rand.New(rand.NewSource(1234))
	chainId := new(big.Int).SetUint64(rng.Uint64())
	signer := types.NewLondonSigner(chainId)
	randTx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
	randTxData, _ := randTx.MarshalBinary()
	l1A := testutils.RandomBlockRef(rng)
	l1B := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1A.Number + 1,
		ParentHash: l1A.Hash,
		Time:       l1A.Time + 7,
	}
	l1C := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1B.Number + 1,
		ParentHash: l1B.Hash,
		Time:       l1B.Time + 7,
	}
	l1D := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1C.Number + 1,
		ParentHash: l1C.Hash,
		Time:       l1C.Time + 7,
	}
	l1E := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1D.Number + 1,
		ParentHash: l1D.Hash,
		Time:       l1D.Time + 7,
	}
	l1F := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1E.Number + 1,
		ParentHash: l1E.Hash,
		Time:       l1E.Time + 7,
	}

	l2A0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         100,
		ParentHash:     testutils.RandomHash(rng),
		Time:           l1A.Time,
		L1Origin:       l1A.ID(),
		SequenceNumber: 0,
	}

	l2A1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A0.Number + 1,
		ParentHash:     l2A0.Hash,
		Time:           l2A0.Time + conf.BlockTime,
		L1Origin:       l1A.ID(),
		SequenceNumber: 1,
	}

	l2A2 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A1.Number + 1,
		ParentHash:     l2A1.Hash,
		Time:           l2A1.Time + conf.BlockTime,
		L1Origin:       l1A.ID(),
		SequenceNumber: 2,
	}

	l2A3 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A2.Number + 1,
		ParentHash:     l2A2.Hash,
		Time:           l2A2.Time + conf.BlockTime,
		L1Origin:       l1A.ID(),
		SequenceNumber: 3,
	}

	l2B0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A3.Number + 1,
		ParentHash:     l2A3.Hash,
		Time:           l2A3.Time + conf.BlockTime, // 8 seconds larger than l1A0, 1 larger than origin
		L1Origin:       l1B.ID(),
		SequenceNumber: 0,
	}

	l1X := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     42,
		ParentHash: testutils.RandomHash(rng),
		Time:       10_000,
	}
	l1Y := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1X.Number + 1,
		ParentHash: l1X.Hash,
		Time:       l1X.Time + 12,
	}
	l1Z := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1Y.Number + 1,
		ParentHash: l1Y.Hash,
		Time:       l1Y.Time + 12,
	}
	l2X0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         1000,
		ParentHash:     testutils.RandomHash(rng),
		Time:           10_000 + 12 + 6 - 1, // add one block, and you get ahead of next l1 block by more than the drift
		L1Origin:       l1X.ID(),
		SequenceNumber: 0,
	}
	l2Y0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2X0.Number + 1,
		ParentHash:     l2X0.Hash,
		Time:           l2X0.Time + conf.BlockTime, // exceeds sequencer time drift, forced to be empty block
		L1Origin:       l1Y.ID(),
		SequenceNumber: 0,
	}

	l2A4 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A3.Number + 1,
		ParentHash:     l2A3.Hash,
		Time:           l2A3.Time + conf.BlockTime, // 4*2 = 8, higher than seq time drift
		L1Origin:       l1A.ID(),
		SequenceNumber: 4,
	}

	l1BLate := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1A.Number + 1,
		ParentHash: l1A.Hash,
		Time:       l2A4.Time + 1, // too late for l2A4 to adopt yet
	}

	testCases := []ValidBatchTestCase{
		{
			Name:       "missing L1 info",
			L1Blocks:   []eth.L1BlockRef{},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchUndecided,
		},
		{
			Name:       "future timestamp",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time + 1, // 1 too high
						Transactions: nil,
					},
				}),
			},
			Expected: BatchFuture,
		},
		{
			Name:       "old timestamp",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A0.Time, // repeating the same time
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "misaligned timestamp",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time - 1, // block time is 2, so this is 1 too low
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "invalid parent block hash",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   testutils.RandomHash(rng),
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequence window expired",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C, l1D, l1E, l1F},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1F,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "epoch too old, but good parent hash and timestamp", // repeat of now outdated l2A3 data
			L1Blocks:   []eth.L1BlockRef{l1B, l1C, l1D},
			L2SafeHead: l2B0, // we already moved on to B
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2B0.Hash,                          // build on top of safe head to continue
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number), // epoch A is no longer valid
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2B0.Time + conf.BlockTime, // pass the timestamp check to get too epoch check
						Transactions: nil,
					},
					{
						EpochNum:     rollup.Epoch(l1B.Number),
						EpochHash:    l1B.Hash, // pass the l1 origin check
						Timestamp:    l2B0.Time + conf.BlockTime*2,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "insufficient L1 info for eager derivation",
			L1Blocks:   []eth.L1BlockRef{l1A}, // don't know about l1B yet
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2B0.ParentHash,
						EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
						EpochHash:    l2B0.L1Origin.Hash,
						Timestamp:    l2B0.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchUndecided,
		},
		{
			Name:       "epoch too new",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C, l1D},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1D,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2B0.ParentHash,
						EpochNum:     rollup.Epoch(l1C.Number), // invalid, we need to adopt epoch B before C
						EpochHash:    l1C.Hash,
						Timestamp:    l2B0.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "epoch hash wrong",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2B0.ParentHash,
						EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
						EpochHash:    l1A.Hash, // invalid, epoch hash should be l1B
						Timestamp:    l2B0.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "epoch hash wrong - long span",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // valid batch
						ParentHash:   l2A3.ParentHash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l1A.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2B0.ParentHash,
						EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
						EpochHash:    l1A.Hash, // invalid, epoch hash should be l1B
						Timestamp:    l2B0.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequencer time drift on same epoch with non-empty txs",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
						ParentHash:   l2A4.ParentHash,
						EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
						EpochHash:    l2A4.L1Origin.Hash,
						Timestamp:    l2A4.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequencer time drift on same epoch with non-empty txs - long span",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // valid batch
						ParentHash:   l2A3.ParentHash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
					{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
						ParentHash:   l2A4.ParentHash,
						EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
						EpochHash:    l2A4.L1Origin.Hash,
						Timestamp:    l2A4.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequencer time drift on changing epoch with non-empty txs",
			L1Blocks:   []eth.L1BlockRef{l1X, l1Y, l1Z},
			L2SafeHead: l2X0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1Z,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2Y0.ParentHash,
						EpochNum:     rollup.Epoch(l2Y0.L1Origin.Number),
						EpochHash:    l2Y0.L1Origin.Hash,
						Timestamp:    l2Y0.Time, // valid, but more than 6 ahead of l1Y.Time
						Transactions: []hexutil.Bytes{randTxData},
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and late next epoch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1BLate},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1BLate,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // l2A4 time < l1BLate time, so we cannot adopt origin B yet
						ParentHash:   l2A4.ParentHash,
						EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
						EpochHash:    l2A4.L1Origin.Hash,
						Timestamp:    l2A4.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchAccept, // accepted because empty & preserving L2 time invariant
		},
		{
			Name:       "sequencer time drift on changing epoch with empty txs",
			L1Blocks:   []eth.L1BlockRef{l1X, l1Y, l1Z},
			L2SafeHead: l2X0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1Z,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2Y0.ParentHash,
						EpochNum:     rollup.Epoch(l2Y0.L1Origin.Number),
						EpochHash:    l2Y0.L1Origin.Hash,
						Timestamp:    l2Y0.Time, // valid, but more than 6 ahead of l1Y.Time
						Transactions: nil,
					},
				}),
			},
			Expected: BatchAccept, // accepted because empty & still advancing epoch
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and no next epoch in sight yet",
			L1Blocks:   []eth.L1BlockRef{l1A},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
						ParentHash:   l2A4.ParentHash,
						EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
						EpochHash:    l2A4.L1Origin.Hash,
						Timestamp:    l2A4.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchUndecided, // we have to wait till the next epoch is in sight to check the time
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and no next epoch in sight yet - long span",
			L1Blocks:   []eth.L1BlockRef{l1A},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // valid batch
						ParentHash:   l2A3.ParentHash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
					{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
						ParentHash:   l2A4.ParentHash,
						EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
						EpochHash:    l2A4.L1Origin.Hash,
						Timestamp:    l2A4.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchUndecided, // we have to wait till the next epoch is in sight to check the time
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and but in-sight epoch that invalidates it",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
						ParentHash:   l2A4.ParentHash,
						EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
						EpochHash:    l2A4.L1Origin.Hash,
						Timestamp:    l2A4.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop, // dropped because it could have advanced the epoch to B
		},
		{
			Name:       "sequencer time drift on same epoch with empty txs and but in-sight epoch that invalidates it - long span",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // valid batch
						ParentHash:   l2A3.ParentHash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
					{ // we build l2A4, which has a timestamp of 2*4 = 8 higher than l2A0
						ParentHash:   l2A4.ParentHash,
						EpochNum:     rollup.Epoch(l2A4.L1Origin.Number),
						EpochHash:    l2A4.L1Origin.Hash,
						Timestamp:    l2A4.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop, // dropped because it could have advanced the epoch to B
		},
		{
			Name:       "empty tx included",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash: l2A1.ParentHash,
						EpochNum:   rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:  l2A1.L1Origin.Hash,
						Timestamp:  l2A1.Time,
						Transactions: []hexutil.Bytes{
							[]byte{}, // empty tx data
						},
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "deposit tx included",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash: l2A1.ParentHash,
						EpochNum:   rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:  l2A1.L1Origin.Hash,
						Timestamp:  l2A1.Time,
						Transactions: []hexutil.Bytes{
							[]byte{types.DepositTxType, 0}, // piece of data alike to a deposit
						},
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "valid batch same epoch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
				}),
			},
			Expected: BatchAccept,
		},
		{
			Name:       "valid batch changing epoch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A3,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1C,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2B0.ParentHash,
						EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
						EpochHash:    l2B0.L1Origin.Hash,
						Timestamp:    l2B0.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
				}),
			},
			Expected: BatchAccept,
		},
		{
			Name:       "batch with L2 time before L1 time",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // we build l2B0, which starts a new epoch too early
						ParentHash:   l2A2.Hash,
						EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
						EpochHash:    l2B0.L1Origin.Hash,
						Timestamp:    l2A2.Time + conf.BlockTime,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "batch with L2 time before L1 time - long span",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B, l1C},
			L2SafeHead: l2A1,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{ // valid batch
						ParentHash:   l2A1.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number),
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: nil,
					},
					{ // we build l2B0, which starts a new epoch too early
						ParentHash:   l2A2.Hash,
						EpochNum:     rollup.Epoch(l2B0.L1Origin.Number),
						EpochHash:    l2B0.L1Origin.Hash,
						Timestamp:    l2A2.Time + conf.BlockTime,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "valid overlapping batch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number),
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A2.Hash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchAccept,
		},
		{
			Name:       "longer overlapping batch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A0.Hash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A1.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number),
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A2.Hash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchAccept,
		},
		{
			Name:       "fully overlapping batch",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A0.Hash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A1.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number),
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "overlapping batch with invalid parent hash",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A0.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number),
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A2.Hash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "overlapping batch with invalid origin number",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number) + 1,
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A2.Hash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "overlapping batch with invalid tx",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A2,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number),
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
					{
						ParentHash:   l2A2.Hash,
						EpochNum:     rollup.Epoch(l2A3.L1Origin.Number),
						EpochHash:    l2A3.L1Origin.Hash,
						Timestamp:    l2A3.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchDrop,
		},
		{
			Name:       "overlapping batch l2 fetcher error",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A1,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A0.ParentHash,
						EpochNum:     rollup.Epoch(l2A0.L1Origin.Number),
						EpochHash:    l2A0.L1Origin.Hash,
						Timestamp:    l2A0.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A0.Hash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: nil,
					},
					{
						ParentHash:   l2A1.Hash,
						EpochNum:     rollup.Epoch(l2A2.L1Origin.Number),
						EpochHash:    l2A2.L1Origin.Hash,
						Timestamp:    l2A2.Time,
						Transactions: nil,
					},
				}),
			},
			Expected: BatchUndecided,
		},
	}

	// Log level can be increased for debugging purposes
	logger := testlog.Logger(t, log.LvlError)

	l2Client := testutils.MockL2Client{}
	var nilErr error
	// will be return error for block #99 (parent of l2A0)
	tempErr := errors.New("temp error")
	l2Client.Mock.On("L2BlockRefByNumber", l2A0.Number-1).Times(9999).Return(eth.L2BlockRef{}, &tempErr)
	l2Client.Mock.On("PayloadByNumber", l2A0.Number-1).Times(9999).Return(nil, &tempErr)

	// make payloads for L2 blocks and set as expected return value of MockL2Client
	for _, l2Block := range []eth.L2BlockRef{l2A0, l2A1, l2A2, l2A3, l2A4, l2B0} {
		l2Client.ExpectL2BlockRefByNumber(l2Block.Number, l2Block, nil)
		txData := l1InfoDepositTx(t, l2Block.L1Origin.Number)
		payload := eth.ExecutionPayload{
			ParentHash:   l2Block.ParentHash,
			BlockNumber:  hexutil.Uint64(l2Block.Number),
			Timestamp:    hexutil.Uint64(l2Block.Time),
			BlockHash:    l2Block.Hash,
			Transactions: []hexutil.Bytes{txData},
		}
		l2Client.Mock.On("L2BlockRefByNumber", l2Block.Number).Times(9999).Return(l2Block, &nilErr)
		l2Client.Mock.On("PayloadByNumber", l2Block.Number).Times(9999).Return(&payload, &nilErr)
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			ctx := context.Background()
			validity := CheckBatch(ctx, &conf, logger, testCase.L1Blocks, testCase.L2SafeHead, &testCase.Batch, &l2Client)
			require.Equal(t, testCase.Expected, validity, "batch check must return expected validity level")
		})
	}
}

func TestSpanBatchHardFork(t *testing.T) {
	minTs := uint64(0)
	conf := rollup.Config{
		Genesis: rollup.Genesis{
			L2Time: 31, // a genesis time that itself does not align to make it more interesting
		},
		BlockTime:         2,
		SeqWindowSize:     4,
		MaxSequencerDrift: 6,
		SpanBatchTime:     &minTs,
		// other config fields are ignored and can be left empty.
	}

	rng := rand.New(rand.NewSource(1234))
	chainId := new(big.Int).SetUint64(rng.Uint64())
	signer := types.NewLondonSigner(chainId)
	randTx := testutils.RandomTx(rng, new(big.Int).SetUint64(rng.Uint64()), signer)
	randTxData, _ := randTx.MarshalBinary()
	l1A := testutils.RandomBlockRef(rng)
	l1B := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     l1A.Number + 1,
		ParentHash: l1A.Hash,
		Time:       l1A.Time + 7,
	}

	l2A0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         100,
		ParentHash:     testutils.RandomHash(rng),
		Time:           l1A.Time,
		L1Origin:       l1A.ID(),
		SequenceNumber: 0,
	}

	l2A1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         l2A0.Number + 1,
		ParentHash:     l2A0.Hash,
		Time:           l2A0.Time + conf.BlockTime,
		L1Origin:       l1A.ID(),
		SequenceNumber: 1,
	}

	testCases := []SpanBatchHardForkTestCase{
		{
			Name:       "singular batch before hard fork",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash:   l2A1.ParentHash,
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A1.Time,
					Transactions: []hexutil.Bytes{randTxData},
				},
			},
			SpanBatchTime: l2A1.Time + 2,
			Expected:      BatchAccept,
		},
		{
			Name:       "span batch before hard fork",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
				}),
			},
			SpanBatchTime: l2A1.Time + 2,
			Expected:      BatchDrop,
		},
		{
			Name:       "singular batch after hard fork",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: &SingularBatch{
					ParentHash:   l2A1.ParentHash,
					EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
					EpochHash:    l2A1.L1Origin.Hash,
					Timestamp:    l2A1.Time,
					Transactions: []hexutil.Bytes{randTxData},
				},
			},
			SpanBatchTime: l2A1.Time - 2,
			Expected:      BatchAccept,
		},
		{
			Name:       "span batch after hard fork",
			L1Blocks:   []eth.L1BlockRef{l1A, l1B},
			L2SafeHead: l2A0,
			Batch: BatchWithL1InclusionBlock{
				L1InclusionBlock: l1B,
				Batch: NewSpanBatch([]*SingularBatch{
					{
						ParentHash:   l2A1.ParentHash,
						EpochNum:     rollup.Epoch(l2A1.L1Origin.Number),
						EpochHash:    l2A1.L1Origin.Hash,
						Timestamp:    l2A1.Time,
						Transactions: []hexutil.Bytes{randTxData},
					},
				}),
			},
			SpanBatchTime: l2A1.Time - 2,
			Expected:      BatchAccept,
		},
	}

	// Log level can be increased for debugging purposes
	logger := testlog.Logger(t, log.LvlInfo)

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			rcfg := conf
			rcfg.SpanBatchTime = &testCase.SpanBatchTime
			ctx := context.Background()
			validity := CheckBatch(ctx, &rcfg, logger, testCase.L1Blocks, testCase.L2SafeHead, &testCase.Batch, nil)
			require.Equal(t, testCase.Expected, validity, "batch check must return expected validity level")
		})
	}
}
