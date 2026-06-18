package interop

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// dafnySyncedObs is the observation of dafnySyncedInterop right before
// verification: last verified 1002, next 1003, frontier heads advancing by
// one block, L1 consistent.
func dafnySyncedObs() RoundObservation {
	return RoundObservation{
		LastVerifiedTS: ptrUint64(1002),
		NextTimestamp:  1003,
		ChainsReady:    true,
		BlocksAtTS:     dafnyHeads(map[uint64]uint64{1: 103, 2: 203}),
		L1Consistent:   true,
	}
}

// dafnyRewindObs is the observation of dafnySyncedInterop after the accepted
// L1 inclusion turned non-canonical (model !l1Consistent, Go L1NeedsRewind).
func dafnyRewindObs() RoundObservation {
	obs := dafnySyncedObs()
	obs.L1Consistent = false
	obs.L1NeedsRewind = true
	return obs
}

// dafnyAdvanceOutput is the advancing step output matching dafnySyncedObs.
func dafnyAdvanceOutput() StepOutput {
	return StepOutput{
		Decision: DecisionAdvance,
		Result:   Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
	}
}

func TestCheckOutputConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.lastTimestamp = 999
		err := CheckOutputConsistentWithVerified(i, StepOutput{Decision: DecisionWait}, dafnySyncedObs())
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("requires: invalid step output short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, obs)
		require.ErrorContains(t, err, "requires ValidStepOutput")
	})

	t.Run("pass: wait output is vacuous", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionWait}, dafnySyncedObs()))
	})

	t.Run("pass: rewind with lastVerifiedTS - 1 present", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, dafnyRewindObs()))
	})

	t.Run("pass: rewind at activation timestamp skips the guarded conjunct", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1000)
		require.NoError(t, CheckOutputConsistentWithVerified(i, StepOutput{Decision: DecisionRewind}, obs))
	})

	t.Run("conjunct R1: rewind with lastVerifiedTS - 1 absent", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1005)
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, obs)
		require.ErrorContains(t, err, "conjunct (R1)")
		require.ErrorContains(t, err, "1004 not in verifiedDB")
	})

	t.Run("pass: advancing output", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			dafnyAdvanceOutput(), dafnySyncedObs()))
	})

	t.Run("conjunct A1: advance result does not advance the db", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Result.Timestamp = 1005
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1005
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t), output, obs)
		require.ErrorContains(t, err, "conjunct (A1)")
	})

	t.Run("pass: invalidate at the next timestamp", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Decision = DecisionInvalidate
		require.NoError(t, CheckOutputConsistentWithVerified(dafnySyncedInterop(t),
			output, dafnySyncedObs()))
	})

	t.Run("conjunct I1: invalidate not at the next timestamp", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Decision = DecisionInvalidate
		output.Result.Timestamp = 1005
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1005
		err := CheckOutputConsistentWithVerified(dafnySyncedInterop(t), output, obs)
		require.ErrorContains(t, err, "conjunct (I1)")
		require.ErrorContains(t, err, "!= NextTimestamp() 1003")
	})
}

func TestCheckOutputConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("requires: Valid() failure short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckOutputConsistentWithLogs(i, StepOutput{Decision: DecisionWait}, dafnySyncedObs())
		require.ErrorContains(t, err, "requires Valid()")
	})

	t.Run("requires: invalid step output short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckOutputConsistentWithLogs(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, obs)
		require.ErrorContains(t, err, "requires ValidStepOutput")
	})

	t.Run("requires: OutputConsistentWithVerified failure short-circuits", func(t *testing.T) {
		t.Parallel()
		output := dafnyAdvanceOutput()
		output.Result.Timestamp = 1005
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1005
		err := CheckOutputConsistentWithLogs(dafnySyncedInterop(t), output, obs)
		require.ErrorContains(t, err, "requires OutputConsistentWithVerified")
	})

	t.Run("pass: wait and invalidate outputs are vacuous", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckOutputConsistentWithLogs(i,
			StepOutput{Decision: DecisionWait}, dafnySyncedObs()))
		output := dafnyAdvanceOutput()
		output.Decision = DecisionInvalidate
		require.NoError(t, CheckOutputConsistentWithLogs(i, output, dafnySyncedObs()))
	})

	t.Run("pass: rewind with target heads sealed on all chains", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithLogs(dafnySyncedInterop(t),
			StepOutput{Decision: DecisionRewind}, dafnyRewindObs()))
	})

	t.Run("conjunct R1: rewind target head not sealed", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101)
		err := CheckOutputConsistentWithLogs(i, StepOutput{Decision: DecisionRewind}, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (R1)")
		require.ErrorContains(t, err, "chain 1")
		require.NotContains(t, err.Error(), "chain 2")
	})

	t.Run("conjunct R2: sealed block differs from verified head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		err := CheckOutputConsistentWithLogs(i, StepOutput{Decision: DecisionRewind}, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (R2)")
	})

	t.Run("pass: advancing output", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckOutputConsistentWithLogs(dafnySyncedInterop(t),
			dafnyAdvanceOutput(), dafnySyncedObs()))
	})

	t.Run("conjunct A1: advance result does not advance the logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).latest = dafnyBlock(105)
		err := CheckOutputConsistentWithLogs(i, dafnyAdvanceOutput(), dafnySyncedObs())
		require.ErrorContains(t, err, "conjunct (A1)")
		require.ErrorContains(t, err, "chain 1")
	})
}

func TestCheckObservationConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.lastTimestamp = 999
		err := CheckObservationConsistentWithVerified(i, dafnySyncedObs())
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("requires: invalid round observation short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "requires ValidRoundObservation")
	})

	t.Run("pass: synced observation", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithVerified(dafnySyncedInterop(t), dafnySyncedObs()))
	})

	t.Run("pass: empty db observation at activation", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.initialized.Store(true)
		i.verificationStartTimestamp = 1000
		obs := RoundObservation{NextTimestamp: 1000}
		require.NoError(t, CheckObservationConsistentWithVerified(i, obs))
	})

	t.Run("conjunct 1: lastVerifiedTS does not mirror the db", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)

		obs := dafnySyncedObs()
		obs.LastVerifiedTS = ptrUint64(1001)
		err := CheckObservationConsistentWithVerified(i, obs)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "1001 != verifiedDB.lastTimestamp 1002")

		obs.LastVerifiedTS = nil
		err = CheckObservationConsistentWithVerified(i, obs)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "obs.lastVerifiedTS is None")
	})

	t.Run("conjunct 2: nextTimestamp does not mirror the db", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1004
		obs.ChainsReady = false // keep conjunct (4) vacuous
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "!= NextTimestamp() 1003")
	})

	t.Run("pass: rewind observation with lastVerifiedTS - 1 present", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithVerified(dafnySyncedInterop(t), dafnyRewindObs()))
	})

	t.Run("pass: rewind observation at activation timestamp skips the guarded conjunct", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1000)
		obs.NextTimestamp = 1001
		require.NoError(t, CheckObservationConsistentWithVerified(i, obs))
	})

	t.Run("conjunct 3: rewind observation with lastVerifiedTS - 1 absent", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = ptrUint64(1005) // also violates conjunct (1); 1004 is not committed
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "1004 not in verifiedDB")
	})

	t.Run("conjunct 4: ready frontier does not advance the db", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.BlocksAtTS = dafnyHeads(map[uint64]uint64{1: 105, 2: 203})
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (4)")
	})

	t.Run("paused: round-flag conjuncts skipped", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.Paused = true
		obs.BlocksAtTS = dafnyHeads(map[uint64]uint64{1: 105, 2: 203}) // would violate (4)
		require.NoError(t, CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs))
	})

	t.Run("paused: db-mirror conjuncts still checked", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.Paused = true
		obs.NextTimestamp = 999
		err := CheckObservationConsistentWithVerified(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "conjunct (2)")
	})
}

func TestCheckObservationConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("requires: Valid() failure short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckObservationConsistentWithLogs(i, dafnySyncedObs())
		require.ErrorContains(t, err, "requires Valid()")
	})

	t.Run("requires: invalid round observation short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnyRewindObs()
		obs.LastVerifiedTS = nil
		err := CheckObservationConsistentWithLogs(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "requires ValidRoundObservation")
	})

	t.Run("requires: ObservationConsistentWithVerified failure short-circuits", func(t *testing.T) {
		t.Parallel()
		obs := dafnySyncedObs()
		obs.NextTimestamp = 1004
		err := CheckObservationConsistentWithLogs(dafnySyncedInterop(t), obs)
		require.ErrorContains(t, err, "requires ObservationConsistentWithVerified")
	})

	t.Run("pass: synced advancing observation", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithLogs(dafnySyncedInterop(t), dafnySyncedObs()))
	})

	t.Run("pass: rewind observation with target heads sealed", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckObservationConsistentWithLogs(dafnySyncedInterop(t), dafnyRewindObs()))
	})

	t.Run("conjunct 1: rewind target head not sealed", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 2).seals, 201)
		err := CheckObservationConsistentWithLogs(i, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "chain 2")
		require.NotContains(t, err.Error(), "chain 1")
	})

	t.Run("conjunct 2: sealed block differs from verified head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 2).seals[201] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 201, Timestamp: 1001,
		}
		err := CheckObservationConsistentWithLogs(i, dafnyRewindObs())
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 3: ready frontier does not advance the logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 2).latest = dafnyBlock(205)
		err := CheckObservationConsistentWithLogs(i, dafnySyncedObs())
		require.ErrorContains(t, err, "conjunct (3)")
		require.ErrorContains(t, err, "chain 2")
	})

	t.Run("paused: all conjuncts skipped", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101) // would violate (1) if not paused
		obs := dafnyRewindObs()
		obs.Paused = true
		require.NoError(t, CheckObservationConsistentWithLogs(i, obs))
	})
}

func TestRoundAsserts(t *testing.T) {
	t.Parallel()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		ft := &fakeDafnyT{}
		AssertOutputConsistentWithVerified(ft, i, dafnyAdvanceOutput(), dafnySyncedObs())
		AssertOutputConsistentWithLogs(ft, i, dafnyAdvanceOutput(), dafnySyncedObs())
		AssertObservationConsistentWithVerified(ft, i, dafnySyncedObs())
		AssertObservationConsistentWithLogs(ft, i, dafnySyncedObs())
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	asserts := map[string]func(dafnyT){
		"AssertOutputConsistentWithVerified": func(ft dafnyT) {
			AssertOutputConsistentWithVerified(ft, nil, StepOutput{}, RoundObservation{})
		},
		"AssertOutputConsistentWithLogs": func(ft dafnyT) {
			AssertOutputConsistentWithLogs(ft, nil, StepOutput{}, RoundObservation{})
		},
		"AssertObservationConsistentWithVerified": func(ft dafnyT) {
			AssertObservationConsistentWithVerified(ft, nil, RoundObservation{})
		},
		"AssertObservationConsistentWithLogs": func(ft dafnyT) {
			AssertObservationConsistentWithLogs(ft, nil, RoundObservation{})
		},
	}
	for name, assert := range asserts {
		t.Run(name+" reports and fails", func(t *testing.T) {
			t.Parallel()
			ft := &fakeDafnyT{}
			assert(ft)
			require.True(t, ft.errorfCalled)
			require.True(t, ft.failNowCalled)
		})
	}
}
