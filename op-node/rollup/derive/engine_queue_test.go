package derive

import (
	"context"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

type fakeAttributesQueue struct {
	origin eth.L1BlockRef
}

func (f *fakeAttributesQueue) Origin() eth.L1BlockRef {
	return f.origin
}

func (f *fakeAttributesQueue) NextAttributes(_ context.Context, _ eth.L2BlockRef) (*eth.PayloadAttributes, error) {
	return nil, io.EOF
}

var _ NextAttributesProvider = (*fakeAttributesQueue)(nil)

func TestEngineQueue_Finalize(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)

	rng := rand.New(rand.NewSource(1234))

	l1Time := uint64(2)
	refA := testutils.RandomBlockRef(rng)

	refB := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refA.Number + 1,
		ParentHash: refA.Hash,
		Time:       refA.Time + l1Time,
	}
	refC := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refB.Number + 1,
		ParentHash: refB.Hash,
		Time:       refB.Time + l1Time,
	}
	refD := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refC.Number + 1,
		ParentHash: refC.Hash,
		Time:       refC.Time + l1Time,
	}
	refE := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refD.Number + 1,
		ParentHash: refD.Hash,
		Time:       refD.Time + l1Time,
	}
	refF := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refE.Number + 1,
		ParentHash: refE.Hash,
		Time:       refE.Time + l1Time,
	}

	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}
	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}
	refB0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA1.Number + 1,
		ParentHash:     refA1.Hash,
		Time:           refA1.Time + cfg.BlockTime,
		L1Origin:       refB.ID(),
		SequenceNumber: 0,
	}
	refB1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refB0.Number + 1,
		ParentHash:     refB0.Hash,
		Time:           refB0.Time + cfg.BlockTime,
		L1Origin:       refB.ID(),
		SequenceNumber: 1,
	}
	refC0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refB1.Number + 1,
		ParentHash:     refB1.Hash,
		Time:           refB1.Time + cfg.BlockTime,
		L1Origin:       refC.ID(),
		SequenceNumber: 0,
	}
	refC1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refC0.Number + 1,
		ParentHash:     refC0.Hash,
		Time:           refC0.Time + cfg.BlockTime,
		L1Origin:       refC.ID(),
		SequenceNumber: 1,
	}
	refD0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refC1.Number + 1,
		ParentHash:     refC1.Hash,
		Time:           refC1.Time + cfg.BlockTime,
		L1Origin:       refD.ID(),
		SequenceNumber: 0,
	}
	refD1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refD0.Number + 1,
		ParentHash:     refD0.Hash,
		Time:           refD0.Time + cfg.BlockTime,
		L1Origin:       refD.ID(),
		SequenceNumber: 1,
	}
	refE0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refD1.Number + 1,
		ParentHash:     refD1.Hash,
		Time:           refD1.Time + cfg.BlockTime,
		L1Origin:       refE.ID(),
		SequenceNumber: 0,
	}
	refE1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refE0.Number + 1,
		ParentHash:     refE0.Hash,
		Time:           refE0.Time + cfg.BlockTime,
		L1Origin:       refE.ID(),
		SequenceNumber: 1,
	}
	refF0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refE1.Number + 1,
		ParentHash:     refE1.Hash,
		Time:           refE1.Time + cfg.BlockTime,
		L1Origin:       refF.ID(),
		SequenceNumber: 0,
	}
	refF1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refF0.Number + 1,
		ParentHash:     refF0.Hash,
		Time:           refF0.Time + cfg.BlockTime,
		L1Origin:       refF.ID(),
		SequenceNumber: 1,
	}
	t.Log("refA", refA.Hash)
	t.Log("refB", refB.Hash)
	t.Log("refC", refC.Hash)
	t.Log("refD", refD.Hash)
	t.Log("refE", refE.Hash)
	t.Log("refF", refF.Hash)
	t.Log("refA0", refA0.Hash)
	t.Log("refA1", refA1.Hash)
	t.Log("refB0", refB0.Hash)
	t.Log("refB1", refB1.Hash)
	t.Log("refC0", refC0.Hash)
	t.Log("refC1", refC1.Hash)
	t.Log("refD0", refD0.Hash)
	t.Log("refD1", refD1.Hash)
	t.Log("refE0", refE0.Hash)
	t.Log("refE1", refE1.Hash)
	t.Log("refF0", refF0.Hash)
	t.Log("refF1", refF1.Hash)

	metrics := &testutils.TestDerivationMetrics{}
	eng := &testutils.MockEngine{}
	// we find the common point to initialize to by comparing the L1 origins in the L2 chain with the L1 chain
	l1F := &testutils.MockL1Source{}

	eng.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	eng.ExpectL2BlockRefByLabel(eth.Safe, refE0, nil)
	eng.ExpectL2BlockRefByLabel(eth.Unsafe, refF1, nil)

	// unsafe
	l1F.ExpectL1BlockRefByNumber(refF.Number, refF, nil)
	eng.ExpectL2BlockRefByHash(refF1.ParentHash, refF0, nil)
	eng.ExpectL2BlockRefByHash(refF0.ParentHash, refE1, nil)

	// meet previous safe, counts 1/2
	l1F.ExpectL1BlockRefByNumber(refE.Number, refE, nil)
	eng.ExpectL2BlockRefByHash(refE1.ParentHash, refE0, nil)
	eng.ExpectL2BlockRefByHash(refE0.ParentHash, refD1, nil)

	// now full seq window, inclusive
	l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil)
	eng.ExpectL2BlockRefByHash(refD1.ParentHash, refD0, nil)
	eng.ExpectL2BlockRefByHash(refD0.ParentHash, refC1, nil)

	// now one more L1 origin
	l1F.ExpectL1BlockRefByNumber(refC.Number, refC, nil)
	eng.ExpectL2BlockRefByHash(refC1.ParentHash, refC0, nil)
	// parent of that origin will be considered safe
	eng.ExpectL2BlockRefByHash(refC0.ParentHash, refB1, nil)

	// and we fetch the L1 origin of that as starting point for engine queue
	l1F.ExpectL1BlockRefByHash(refB.Hash, refB, nil)
	l1F.ExpectL1BlockRefByHash(refB.Hash, refB, nil)

	// and mock a L1 config for the last L2 block that references the L1 starting point
	eng.ExpectSystemConfigByL2Hash(refB1.Hash, eth.SystemConfig{
		BatcherAddr: common.Address{42},
		Overhead:    [32]byte{123},
		Scalar:      [32]byte{42},
		GasLimit:    20_000_000,
	}, nil)

	prev := &fakeAttributesQueue{}

	eq := NewEngineQueue(logger, cfg, eng, metrics, prev, l1F)
	require.ErrorIs(t, eq.Reset(context.Background(), eth.L1BlockRef{}, eth.SystemConfig{}), io.EOF)

	require.Equal(t, refB1, eq.SafeL2Head(), "L2 reset should go back to sequence window ago: blocks with origin E and D are not safe until we reconcile, C is extra, and B1 is the end we look for")
	require.Equal(t, refB, eq.Origin(), "Expecting to be set back derivation L1 progress to B")
	require.Equal(t, refA1, eq.Finalized(), "A1 is recognized as finalized before we run any steps")

	// now say C1 was included in D and became the new safe head
	eq.origin = refD
	prev.origin = refD
	eq.safeHead = refC1
	eq.postProcessSafeL2()

	// now say D0 was included in E and became the new safe head
	eq.origin = refE
	prev.origin = refE
	eq.safeHead = refD0
	eq.postProcessSafeL2()

	// let's finalize D (current L1), from which we fully derived C1 (it was safe head), but not D0 (included in E)
	eq.Finalize(refD)

	require.Equal(t, refC1, eq.Finalized(), "C1 was included in finalized D, and should now be finalized")

	l1F.AssertExpectations(t)
	eng.AssertExpectations(t)
}
