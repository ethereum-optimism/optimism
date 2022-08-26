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
	//  B1: not yet included in L1
	//  C0: head, not included in L1 yet
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

	metrics := &TestMetrics{}
	eng := &testutils.MockEngine{}
	eng.ExpectL2BlockRefByLabel(eth.Finalized, refA1, nil)
	// TODO(Proto): update expectation once we're using safe block label properly for sync starting point
	eng.ExpectL2BlockRefByLabel(eth.Unsafe, refC0, nil)

	// we find the common point to initialize to by comparing the L1 origins in the L2 chain with the L1 chain
	l1F := &testutils.MockL1Source{}
	l1F.ExpectL1BlockRefByLabel(eth.Unsafe, refD, nil)
	l1F.ExpectL1BlockRefByNumber(refC0.L1Origin.Number, refC, nil)
	eng.ExpectL2BlockRefByHash(refC0.ParentHash, refB1, nil)   // good L1 origin
	eng.ExpectL2BlockRefByHash(refB1.ParentHash, refB0, nil)   // need a block with seqnr == 0, don't stop at above
	l1F.ExpectL1BlockRefByHash(refB0.L1Origin.Hash, refB, nil) // the origin of the safe L2 head will be the L1 starting point for derivation.

	eq := NewEngineQueue(logger, cfg, eng, metrics)
	require.NoError(t, RepeatResetStep(t, eq.ResetStep, l1F, 3))

	// TODO(proto): this is changing, needs to be a sequence window ago, but starting traversal back from safe block,
	// safe blocks with canon origin are good, but we go back a full window to ensure they are all included in L1,
	// by forcing them to be consolidated with L1 again.
	require.Equal(t, eq.SafeL2Head(), refB0, "L2 reset should go back to sequence window ago")

	require.Equal(t, refA1, eq.Finalized(), "A1 is recognized as finalized before we run any steps")

	// we are not adding blocks in this test,
	// but we can still trigger post-processing for the already existing safe head,
	// so the engine can prepare to finalize that.
	eq.postProcessSafeL2()
	// let's finalize C, which included B0, but not B1
	eq.Finalize(refC.ID())

	// Now a few steps later, without consuming any additional L1 inputs,
	// we should be able to resolve that B0 is now finalized
	require.NoError(t, RepeatStep(t, eq.Step, eq.progress, 10))
	require.Equal(t, refB0, eq.Finalized(), "B0 was included in finalized C, and should now be finalized")

	l1F.AssertExpectations(t)
	eng.AssertExpectations(t)
}
