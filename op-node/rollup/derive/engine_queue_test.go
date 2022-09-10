package derive

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestEngineQueue_Finalize(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)

	rng := rand.New(rand.NewSource(1234))
	// create a short test L2 chain:
	//
	// L2:
	//	A0: genesis
	//	A1: finalized, incl in B
	//  B0: safe, incl in C
	//  B1: not yet included in L1, becomes safe later
	//  C0: not yet included in L1, becomes safe later
	//  C1-E0: all unsafe intermediate blocks, but with canonical L1 origins
	//  E1: head, not included in L1 yet
	//
	// L1:
	//  A: genesis
	//  B: finalized, incl A1
	//  C: safe, incl B0
	//  D: unsafe, not yet referenced by L2

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
	t.Log("refA", refA.Hash)
	t.Log("refB", refB.Hash)
	t.Log("refC", refC.Hash)
	t.Log("refD", refD.Hash)
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

	metrics := &TestMetrics{}
	eng := &testutils.MockEngine{}
	// we find the common point to initialize to by comparing the L1 origins in the L2 chain with the L1 chain
	l1F := &testutils.MockL1Source{}

	eng.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	eng.ExpectL2BlockRefByLabel(eth.Safe, refB0, nil)
	eng.ExpectL2BlockRefByLabel(eth.Unsafe, refE1, nil)

	l1F.ExpectL1BlockRefByNumber(refE.Number, refE, nil)     // fetch L1 origin of head, it's canon
	eng.ExpectL2BlockRefByHash(refE1.ParentHash, refE0, nil) // traverse L2 chain, find safe head B0
	eng.ExpectL2BlockRefByHash(refE0.ParentHash, refD1, nil) // traverse back full seq window
	l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil)
	eng.ExpectL2BlockRefByHash(refD1.ParentHash, refD0, nil)
	eng.ExpectL2BlockRefByHash(refD0.ParentHash, refC1, nil)
	l1F.ExpectL1BlockRefByNumber(refC.Number, refC, nil)
	eng.ExpectL2BlockRefByHash(refC1.ParentHash, refC0, nil)
	eng.ExpectL2BlockRefByHash(refC0.ParentHash, refB1, nil)
	eng.ExpectL2BlockRefByHash(refB1.ParentHash, refB0, nil)
	l1F.ExpectL1BlockRefByNumber(refB.Number, refB, nil)
	l1F.ExpectL1BlockRefByHash(refB.Hash, refB, nil)

	eq := NewEngineQueue(logger, cfg, eng, metrics)
	require.NoError(t, RepeatResetStep(t, eq.ResetStep, l1F, 20))

	require.Equal(t, refB0, eq.SafeL2Head(), "L2 reset should go back to sequence window ago: blocks with origin D and C are not safe until we reconcile")
	require.Equal(t, refB, eq.Progress().Origin, "Expecting to be set back derivation L1 progress to B")
	require.Equal(t, refA1, eq.Finalized(), "A1 is recognized as finalized before we run any steps")

	// now say B1 was included in C and became the new safe head
	eq.progress.Origin = refC
	eq.safeHead = refB1
	eq.postProcessSafeL2()

	// now say C0 was included in D and became the new safe head
	eq.progress.Origin = refD
	eq.safeHead = refC0
	eq.postProcessSafeL2()

	// let's finalize C (current L1), from which we fully derived B1, but not C0
	eq.Finalize(refC.ID())

	// Now a few steps later, without consuming any additional L1 inputs,
	// we should be able to resolve that B1 is now finalized, since it was included in finalized L1 block C
	require.NoError(t, RepeatStep(t, eq.Step, eq.progress, 10))
	require.Equal(t, refB1, eq.Finalized(), "B1 was included in finalized C, and should now be finalized")

	l1F.AssertExpectations(t)
	eng.AssertExpectations(t)
}
