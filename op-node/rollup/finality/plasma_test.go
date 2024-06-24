package finality

import (
	"context"
	"math/rand" // nosemgrep
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type fakePlasmaBackend struct {
	plasmaFn  plasma.HeadSignalFn
	forwardTo plasma.HeadSignalFn
}

func (b *fakePlasmaBackend) Finalize(ref eth.L1BlockRef) {
	b.plasmaFn(ref)
}

func (b *fakePlasmaBackend) OnFinalizedHeadSignal(f plasma.HeadSignalFn) {
	b.forwardTo = f
}

var _ PlasmaBackend = (*fakePlasmaBackend)(nil)

func TestPlasmaFinalityData(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)
	l1F := &testutils.MockL1Source{}

	rng := rand.New(rand.NewSource(1234))

	refA := testutils.RandomBlockRef(rng)
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
			SystemConfig: eth.SystemConfig{
				BatcherAddr: common.Address{42},
				Overhead:    [32]byte{123},
				Scalar:      [32]byte{42},
				GasLimit:    20_000_000,
			},
		},
		BlockTime:     1,
		SeqWindowSize: 2,
	}
	plasmaCfg := &rollup.PlasmaConfig{
		DAChallengeWindow: 90,
		DAResolveWindow:   90,
	}
	// shoud return l1 finality if plasma is not enabled
	require.Equal(t, uint64(defaultFinalityLookback), calcFinalityLookback(cfg))

	cfg.PlasmaConfig = plasmaCfg
	expFinalityLookback := 181
	require.Equal(t, uint64(expFinalityLookback), calcFinalityLookback(cfg))

	refA1 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         refA0.Number + 1,
		ParentHash:     refA0.Hash,
		Time:           refA0.Time + cfg.BlockTime,
		L1Origin:       refA.ID(),
		SequenceNumber: 1,
	}

	// Simulate plasma finality by waiting for the finalized-inclusion
	// of a commitment to turn into undisputed finalized data.
	commitmentInclusionFinalized := eth.L1BlockRef{}
	plasmaBackend := &fakePlasmaBackend{
		plasmaFn: func(ref eth.L1BlockRef) {
			commitmentInclusionFinalized = ref
		},
		forwardTo: nil,
	}

	emitter := &testutils.MockEmitter{}
	fi := NewPlasmaFinalizer(context.Background(), logger, cfg, l1F, emitter, plasmaBackend)
	require.NotNil(t, plasmaBackend.forwardTo, "plasma backend must have access to underlying standard finalizer")

	require.Equal(t, expFinalityLookback, cap(fi.finalityData))

	l1parent := refA
	l2parent := refA1

	// advance over 200 l1 origins each time incrementing new l2 safe heads
	// and post processing.
	for i := uint64(0); i < 200; i++ {
		if i == 10 { // finalize a L1 commitment
			fi.OnEvent(FinalizeL1Event{FinalizedL1: l1parent})
			emitter.AssertExpectations(t) // no events emitted upon L1 finality
			require.Equal(t, l1parent, commitmentInclusionFinalized, "plasma backend received L1 signal")
		}

		previous := l1parent
		l1parent = eth.L1BlockRef{
			Hash:       testutils.RandomHash(rng),
			Number:     previous.Number + 1,
			ParentHash: previous.Hash,
			Time:       previous.Time + 12,
		}

		for j := uint64(0); j < 2; j++ {
			l2parent = eth.L2BlockRef{
				Hash:           testutils.RandomHash(rng),
				Number:         l2parent.Number + 1,
				ParentHash:     l2parent.Hash,
				Time:           l2parent.Time + cfg.BlockTime,
				L1Origin:       previous.ID(), // reference previous origin, not the block the batch was included in
				SequenceNumber: j,
			}
			fi.OnEvent(engine.SafeDerivedEvent{Safe: l2parent, DerivedFrom: l1parent})
			emitter.AssertExpectations(t)
		}
		// might trigger finalization attempt, if expired finality delay
		emitter.ExpectMaybeRun(func(ev rollup.Event) {
			require.IsType(t, TryFinalizeEvent{}, ev)
		})
		fi.OnEvent(derive.DeriverIdleEvent{})
		emitter.AssertExpectations(t)
		// clear expectations
		emitter.Mock.ExpectedCalls = nil

		// no L2 finalize event, as no L1 finality signal has been forwarded by plasma backend yet
		fi.OnEvent(TryFinalizeEvent{})
		emitter.AssertExpectations(t)

		// Pretend to be the plasma backend,
		// send the original finalization signal to the underlying finalizer,
		// now that we are sure the commitment itself is not just finalized,
		// but the referenced data cannot be disputed anymore.
		plasmaFinalization := commitmentInclusionFinalized.Number + cfg.PlasmaConfig.DAChallengeWindow
		if commitmentInclusionFinalized != (eth.L1BlockRef{}) && l1parent.Number == plasmaFinalization {
			// When the signal is forwarded, a finalization attempt will be scheduled
			emitter.ExpectOnce(TryFinalizeEvent{})
			plasmaBackend.forwardTo(commitmentInclusionFinalized)
			emitter.AssertExpectations(t)
			require.Equal(t, commitmentInclusionFinalized, fi.finalizedL1, "finality signal now made its way in regular finalizer")

			// As soon as a finalization attempt is made, after the finality signal was triggered by plasma backend,
			// we should get an attempt to get a finalized L2 block.
			// In this test the L1 origin of the simulated L2 blocks lags 1 behind the block the L2 block is included in on L1.
			// So to check the L2 finality progress, we check if the next L1 block after the L1 origin
			// of the safe block matches that of the finalized L1 block.
			l1F.ExpectL1BlockRefByNumber(commitmentInclusionFinalized.Number, commitmentInclusionFinalized, nil)
			l1F.ExpectL1BlockRefByNumber(commitmentInclusionFinalized.Number, commitmentInclusionFinalized, nil)
			var finalizedL2 eth.L2BlockRef
			emitter.ExpectOnceRun(func(ev rollup.Event) {
				if x, ok := ev.(engine.PromoteFinalizedEvent); ok {
					finalizedL2 = x.Ref
				} else {
					t.Fatalf("expected L2 finalization, but got: %s", ev)
				}
			})
			fi.OnEvent(TryFinalizeEvent{})
			l1F.AssertExpectations(t)
			emitter.AssertExpectations(t)
			require.Equal(t, commitmentInclusionFinalized.Number, finalizedL2.L1Origin.Number+1)
			// Confirm finalization, so there will be no repeats of the PromoteFinalizedEvent
			fi.OnEvent(engine.ForkchoiceUpdateEvent{FinalizedL2Head: finalizedL2})
			emitter.AssertExpectations(t)
		}
	}

	// finality data does not go over challenge + resolve windows + 1 capacity
	// (prunes down to 180 then adds the extra 1 each time)
	require.Equal(t, expFinalityLookback, len(fi.finalityData))
}
