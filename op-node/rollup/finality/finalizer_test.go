package finality

import (
	"context"
	"errors"
	"math/rand" // nosemgrep
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type fakeEngine struct {
	finalized eth.L2BlockRef
}

func (f *fakeEngine) Finalized() eth.L2BlockRef {
	return f.finalized
}

func (f *fakeEngine) SetFinalizedHead(ref eth.L2BlockRef) {
	f.finalized = ref
}

var _ FinalizerEngine = (*fakeEngine)(nil)

func TestEngineQueue_Finalize(t *testing.T) {
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
	refG := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refF.Number + 1,
		ParentHash: refF.Hash,
		Time:       refF.Time + l1Time,
	}
	refH := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refG.Number + 1,
		ParentHash: refG.Hash,
		Time:       refG.Time + l1Time,
	}
	refI := eth.L1BlockRef{
		Hash:       testutils.RandomHash(rng),
		Number:     refH.Number + 1,
		ParentHash: refH.Hash,
		Time:       refH.Time + l1Time,
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
	_ = refF1

	// We expect the L1 block that the finalized L2 data was derived from to be checked,
	// to be sure it is part of the canonical chain, after the finalization signal.
	t.Run("basic", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil)
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil)

		ec := &fakeEngine{}
		ec.SetFinalizedHead(refA1)

		fi := NewFinalizer(logger, &rollup.Config{}, l1F, ec)

		// now say C1 was included in D and became the new safe head
		fi.PostProcessSafeL2(refC1, refD)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refD))

		// now say D0 was included in E and became the new safe head
		fi.PostProcessSafeL2(refD0, refE)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refE))

		// let's finalize D from which we fully derived C1, but not D0
		fi.Finalize(context.Background(), refD)
		require.Equal(t, refC1, ec.Finalized(), "C1 was included in finalized D, and should now be finalized, as finality signal is instantly picked up")
	})

	// Finality signal is received, but couldn't immediately be checked
	t.Run("retry", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, errors.New("fake error"))
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil) // to check finality signal
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil) // to check what was derived from (same in this case)

		ec := &fakeEngine{}
		ec.SetFinalizedHead(refA1)

		fi := NewFinalizer(logger, &rollup.Config{}, l1F, ec)

		// now say C1 was included in D and became the new safe head
		fi.PostProcessSafeL2(refC1, refD)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refD))

		// now say D0 was included in E and became the new safe head
		fi.PostProcessSafeL2(refD0, refE)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refE))

		// let's finalize D from which we fully derived C1, but not D0
		fi.Finalize(context.Background(), refD)
		require.Equal(t, refA1, ec.Finalized(), "C1 was included in finalized D, but finality could not be verified yet, due to temporary test error")

		require.NoError(t, fi.OnDerivationL1End(context.Background(), refF))
		require.Equal(t, refC1, ec.Finalized(), "C1 was included in finalized D, and should now be finalized, as check can succeed when revisited")
	})

	// Test that finality progression can repeat a few times.
	t.Run("repeat", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)

		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil)
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil)
		l1F.ExpectL1BlockRefByNumber(refE.Number, refE, nil)
		l1F.ExpectL1BlockRefByNumber(refE.Number, refE, nil)
		l1F.ExpectL1BlockRefByNumber(refH.Number, refH, nil)
		l1F.ExpectL1BlockRefByNumber(refH.Number, refH, nil)

		ec := &fakeEngine{}
		ec.SetFinalizedHead(refA1)

		fi := NewFinalizer(logger, &rollup.Config{}, l1F, ec)

		fi.PostProcessSafeL2(refC1, refD)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refD))

		fi.PostProcessSafeL2(refD0, refE)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refE))

		fi.Finalize(context.Background(), refD)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refF))
		require.Equal(t, refC1, ec.Finalized(), "C1 was included in D, and should be finalized now")

		fi.Finalize(context.Background(), refE)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refG))
		require.Equal(t, refD0, ec.Finalized(), "D0 was included in E, and should be finalized now")

		fi.PostProcessSafeL2(refD1, refH)
		fi.PostProcessSafeL2(refE0, refH)
		fi.PostProcessSafeL2(refE1, refH)
		fi.PostProcessSafeL2(refF0, refH)
		fi.PostProcessSafeL2(refF1, refH)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refH))
		require.Equal(t, refD0, ec.Finalized(), "D1-F1 were included in L1 blocks that have not been finalized yet")

		fi.Finalize(context.Background(), refH)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refI))
		require.Equal(t, refF1, ec.Finalized(), "F1 should be finalized now")
	})

	// In this test the finality signal is for a block more than
	// 1 L1 block later than what the L2 data was included in.
	t.Run("older-data", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil) // check the signal
		l1F.ExpectL1BlockRefByNumber(refC.Number, refC, nil) // check what we derived the L2 block from

		ec := &fakeEngine{}
		ec.SetFinalizedHead(refA1)

		fi := NewFinalizer(logger, &rollup.Config{}, l1F, ec)

		// now say B1 was included in C and became the new safe head
		fi.PostProcessSafeL2(refB1, refC)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refC))

		// now say C0 was included in E and became the new safe head
		fi.PostProcessSafeL2(refC0, refE)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refE))

		// let's finalize D, from which we fully derived B1, but not C0 (referenced L1 origin in L2 block != inclusion of L2 block in L1 chain)
		fi.Finalize(context.Background(), refD)
		require.Equal(t, refB1, ec.Finalized(), "B1 was included in finalized D, and should now be finalized")
	})

	// Test that reorg race condition is handled.
	t.Run("reorg-safe", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelInfo)
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectL1BlockRefByNumber(refF.Number, refF, nil) // check signal
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil) // shows reorg to Finalize attempt
		l1F.ExpectL1BlockRefByNumber(refF.Number, refF, nil) // check signal
		l1F.ExpectL1BlockRefByNumber(refD.Number, refD, nil) // shows reorg to OnDerivationL1End attempt
		l1F.ExpectL1BlockRefByNumber(refF.Number, refF, nil) // check signal
		l1F.ExpectL1BlockRefByNumber(refE.Number, refE, nil) // post-reorg

		ec := &fakeEngine{}
		ec.SetFinalizedHead(refA1)

		fi := NewFinalizer(logger, &rollup.Config{}, l1F, ec)

		// now say B1 was included in C and became the new safe head
		fi.PostProcessSafeL2(refB1, refC)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refC))

		// temporary fork of the L1, and derived safe L2 blocks from.
		refC0Alt := eth.L2BlockRef{
			Hash:           testutils.RandomHash(rng),
			Number:         refB1.Number + 1,
			ParentHash:     refB1.Hash,
			Time:           refB1.Time + cfg.BlockTime,
			L1Origin:       refC.ID(),
			SequenceNumber: 0,
		}
		refC1Alt := eth.L2BlockRef{
			Hash:           testutils.RandomHash(rng),
			Number:         refC0Alt.Number + 1,
			ParentHash:     refC0Alt.Hash,
			Time:           refC0Alt.Time + cfg.BlockTime,
			L1Origin:       refC.ID(),
			SequenceNumber: 1,
		}
		refDAlt := eth.L1BlockRef{
			Hash:       testutils.RandomHash(rng),
			Number:     refC.Number + 1,
			ParentHash: refC.Hash,
			Time:       refC.Time + l1Time,
		}
		fi.PostProcessSafeL2(refC0Alt, refDAlt)
		fi.PostProcessSafeL2(refC1Alt, refDAlt)

		// We get an early finality signal for F, of the chain that did not include refC0Alt and refC1Alt,
		// as L1 block F does not build on DAlt.
		// The finality signal was for a new chain, while derivation is on an old stale chain.
		// It should be detected that C0Alt and C1Alt cannot actually be finalized,
		// even though they are older than the latest finality signal.
		fi.Finalize(context.Background(), refF)
		require.Equal(t, refA1, ec.Finalized(), "cannot verify refC0Alt and refC1Alt, and refB1 is older and not checked")
		// And process DAlt, still stuck on old chain.
		require.ErrorIs(t, derive.ErrReset, fi.OnDerivationL1End(context.Background(), refDAlt))
		require.Equal(t, refA1, ec.Finalized(), "no new finalized L2 blocks after early finality signal with stale chain")
		require.Equal(t, refF, fi.FinalizedL1(), "remember the new finality signal for later however")
		// Now reset, because of the reset error
		fi.Reset()

		// And process the canonical chain, with empty block D (no post-processing of canonical C0 blocks yet)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refD))

		// Include C0 in E
		fi.PostProcessSafeL2(refC0, refE)
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refE))
		// Due to the "finalityDelay" we don't repeat finality checks shortly after one another.
		require.Equal(t, refA1, ec.Finalized())
		// if we reset the attempt, then we can finalize however.
		fi.triedFinalizeAt = 0
		require.NoError(t, fi.OnDerivationL1End(context.Background(), refE))
		require.Equal(t, refC0, ec.Finalized())
	})
}
