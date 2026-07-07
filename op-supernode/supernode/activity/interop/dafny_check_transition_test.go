package interop

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// dafnyBlock builds a BlockID matching a dafnyVerifiedResult head at number.
func dafnyBlock(number uint64) eth.BlockID {
	return eth.BlockID{Hash: common.HexToHash("0xaa"), Number: number}
}

// dafnyHeads builds a chainID->head map from chainID->number pairs, with
// dafnyBlock hashes.
func dafnyHeads(numbers map[uint64]uint64) map[eth.ChainID]eth.BlockID {
	heads := make(map[eth.ChainID]eth.BlockID, len(numbers))
	for c, n := range numbers {
		heads[dafnyChainID(c)] = dafnyBlock(n)
	}
	return heads
}

// dafnySyncedPlan is a rewind plan consistent with dafnySyncedInterop:
// rewind at 1002, reset to 1001, target heads = verified heads at 1001.
func dafnySyncedPlan() RewindPlan {
	return RewindPlan{
		RewindAtOrAfter:  1002,
		ResetAllChainsTo: ptrUint64(1001),
		TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
	}
}

// dafnySyncedAdvance is a pending Advance transition consistent with
// dafnySyncedInterop (next ts 1003, heads advancing by one block).
func dafnySyncedAdvance() PendingTransition {
	return PendingTransition{
		Decision: DecisionAdvance,
		Result: &Result{
			Timestamp: 1003,
			L2Heads:   dafnyHeads(map[uint64]uint64{1: 103, 2: 203}),
		},
	}
}

// dafnyRewoundInterop builds the state right after applyRewindPlan: verified
// results at 1000..1001, logsDBs sealed to the 1001 heads, and the rewind
// pending transition still stored. Returns the instance and the stored plan.
func dafnyRewoundInterop(t *testing.T) (*Interop, RewindPlan) {
	t.Helper()
	i := dafnyTestInterop(t)
	for off := uint64(0); off <= 1; off++ {
		require.NoError(t, i.verifiedDB.Commit(
			dafnyVerifiedResult(1000+off, map[uint64]uint64{1: 100 + off, 2: 200 + off})))
	}
	i.logsDBs[dafnyChainID(1)] = dafnySealedMock(dafnyHeadSeal(100, 1000), dafnyHeadSeal(101, 1001))
	i.logsDBs[dafnyChainID(2)] = dafnySealedMock(dafnyHeadSeal(200, 1000), dafnyHeadSeal(201, 1001))
	plan := RewindPlan{
		RewindAtOrAfter:  1002,
		ResetAllChainsTo: ptrUint64(1001),
		TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
	}
	require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
		Decision: DecisionRewind,
		Rewind:   &plan,
	}))
	return i, plan
}

func TestCheckAdvancesVerifiedDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: empty db at activation timestamp", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckAdvancesVerifiedDB(dafnyTestInterop(t), 1000, nil))
	})

	t.Run("pass: next timestamp with heads advancing by at most one", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 202})))
	})

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.concrete().lastTimestamp = 999
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("conjunct N1: empty db but ts not at activation", func(t *testing.T) {
		t.Parallel()
		err := CheckAdvancesVerifiedDB(dafnyTestInterop(t), 1001, nil)
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: ts not the successor of the last timestamp", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1005, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S2)")
	})

	t.Run("conjunct S2: blocks key set differs from last heads", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 103}))
		require.ErrorContains(t, err, "conjunct (S2)")
	})

	t.Run("conjunct S3: head number jumps by more than one", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 105, 2: 203}))
		require.ErrorContains(t, err, "conjunct (S3)")
		require.ErrorContains(t, err, "not in [102, 103]")
	})

	t.Run("conjunct S3: head number decreases", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesVerifiedDB(i, 1003, dafnyHeads(map[uint64]uint64{1: 101, 2: 203}))
		require.ErrorContains(t, err, "conjunct (S3)")
	})
}

func TestCheckAdvancesLogsDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: empty logsDB at activation timestamp", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckAdvancesLogsDB(dafnyTestInterop(t), 1000, dafnyChainID(1), dafnyBlock(100)))
	})

	t.Run("pass: next block and repeated tip block", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(103)))
		require.NoError(t, CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(102)))
	})

	t.Run("conjunct 0: unknown chain", func(t *testing.T) {
		t.Parallel()
		err := CheckAdvancesLogsDB(dafnySyncedInterop(t), 1003, dafnyChainID(9), dafnyBlock(103))
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "has no logsDB")
	})

	t.Run("conjunct N1: empty logsDB but ts not at activation", func(t *testing.T) {
		t.Parallel()
		err := CheckAdvancesLogsDB(dafnyTestInterop(t), 1234, dafnyChainID(1), dafnyBlock(100))
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: block number out of range", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(104))
		require.ErrorContains(t, err, "conjunct (S1)")
		err = CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), dafnyBlock(101))
		require.ErrorContains(t, err, "conjunct (S1)")
	})

	t.Run("conjunct S2: same-number block differs from latest", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesLogsDB(i, 1003, dafnyChainID(1), eth.BlockID{Hash: common.Hash{0xff}, Number: 102})
		require.ErrorContains(t, err, "conjunct (S2)")
		require.NotContains(t, err.Error(), "conjunct (S1)")
	})
}

func TestCheckAdvancesAllLogsDBs(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckAdvancesAllLogsDBs(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203})))
	})

	t.Run("conjunct 0: key set mismatch", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesAllLogsDBs(i, 1003, dafnyHeads(map[uint64]uint64{1: 103}))
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "requires blocksAtTS.Keys == logsDBs.Keys")
	})

	t.Run("reports only the failing chain", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		err := CheckAdvancesAllLogsDBs(i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 205}))
		require.ErrorContains(t, err, "chain 2:")
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "chain 1:")
	})
}

func TestCheckPlanConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("pass: None plan is vacuous", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckPlanConsistentWithVerified(i, RewindPlan{RewindAtOrAfter: 1000}))
	})

	t.Run("pass: consistent Some plan", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckPlanConsistentWithVerified(dafnySyncedInterop(t), dafnySyncedPlan()))
	})

	t.Run("conjunct S1: reset target not in db", func(t *testing.T) {
		t.Parallel()
		plan := RewindPlan{
			RewindAtOrAfter:  1005,
			ResetAllChainsTo: ptrUint64(1004),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
		}
		err := CheckPlanConsistentWithVerified(dafnySyncedInterop(t), plan)
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S3)")
	})

	t.Run("conjunct S2: surviving timestamp above reset target", func(t *testing.T) {
		t.Parallel()
		plan := RewindPlan{
			RewindAtOrAfter:  1002,
			ResetAllChainsTo: ptrUint64(1000),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 100, 2: 200}),
		}
		err := CheckPlanConsistentWithVerified(dafnySyncedInterop(t), plan)
		require.ErrorContains(t, err, "conjunct (S2)")
		require.ErrorContains(t, err, "committed timestamp 1001")
	})

	t.Run("conjunct S3: target heads diverge from verified result", func(t *testing.T) {
		t.Parallel()
		plan := dafnySyncedPlan()
		plan.TargetHeads = dafnyHeads(map[uint64]uint64{1: 999, 2: 201})
		err := CheckPlanConsistentWithVerified(dafnySyncedInterop(t), plan)
		require.ErrorContains(t, err, "conjunct (S3)")
	})
}

func TestCheckPlanConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("pass: None plan is always consistent", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckPlanConsistentWithLogs(i, RewindPlan{RewindAtOrAfter: 1000}, dafnyChainID(1)))
	})

	t.Run("pass: target heads sealed on both chains", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(1)))
		require.NoError(t, CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(2)))
	})

	t.Run("conjunct 0: unknown chain", func(t *testing.T) {
		t.Parallel()
		err := CheckPlanConsistentWithLogs(dafnySyncedInterop(t), dafnySyncedPlan(), dafnyChainID(9))
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct 0: chain missing from target heads", func(t *testing.T) {
		t.Parallel()
		plan := dafnySyncedPlan()
		delete(plan.TargetHeads, dafnyChainID(1))
		err := CheckPlanConsistentWithLogs(dafnySyncedInterop(t), plan, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (0)")
		require.ErrorContains(t, err, "requires chain")
	})

	t.Run("conjunct S1: target head not sealed", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101)
		err := CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S1)")
	})

	t.Run("conjunct S2: sealed block differs from target head", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		err := CheckPlanConsistentWithLogs(i, dafnySyncedPlan(), dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S2)")
	})
}

func TestCheckRewoundVerifiedDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: rewound state with stored plan", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, CheckRewoundVerifiedDB(i, plan))
	})

	t.Run("pass: full rewind on empty db", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		plan := RewindPlan{RewindAtOrAfter: 1000}
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		require.NoError(t, CheckRewoundVerifiedDB(i, plan))
	})

	t.Run("requires: inconsistent plan short-circuits", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		plan.ResetAllChainsTo = ptrUint64(1005)
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "requires PlanConsistentWithVerified")
	})

	t.Run("conjunct 1: no pending transition", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, i.verifiedDB.ClearPendingTransition())
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (1)")
	})

	t.Run("conjunct 2: stored decision is not Rewind", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1002, L2Heads: dafnyHeads(map[uint64]uint64{1: 102, 2: 202})},
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (2)")
	})

	t.Run("conjunct 3: stored plan differs from given plan", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		stored := plan
		stored.RewindAtOrAfter = 1003
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &stored,
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (3)")
	})

	t.Run("conjunct N1: db not empty after full rewind", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, i.verifiedDB.Commit(dafnyVerifiedResult(1000, map[uint64]uint64{1: 100, 2: 200})))
		plan := RewindPlan{RewindAtOrAfter: 1000}
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: last timestamp beyond reset target", func(t *testing.T) {
		t.Parallel()
		i, _ := dafnyRewoundInterop(t)
		plan := RewindPlan{
			RewindAtOrAfter:  1001,
			ResetAllChainsTo: ptrUint64(1000),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 100, 2: 200}),
		}
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		err := CheckRewoundVerifiedDB(i, plan)
		require.ErrorContains(t, err, "conjunct (S1)")
		require.ErrorContains(t, err, "LastTimestamp() 1001 != resetAllChainsTo 1000")
	})
}

func TestCheckRewoundLogsDB(t *testing.T) {
	t.Parallel()

	t.Run("pass: rewound state on both chains", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		require.NoError(t, CheckRewoundLogsDB(i, plan, dafnyChainID(1)))
		require.NoError(t, CheckRewoundLogsDB(i, plan, dafnyChainID(2)))
	})

	t.Run("pass: cleared logsDB after full rewind", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		require.NoError(t, CheckRewoundLogsDB(i, RewindPlan{RewindAtOrAfter: 1000}, dafnyChainID(1)))
	})

	t.Run("requires: plan inconsistent with logs short-circuits", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		delete(mockLogsDBFor(t, i, 1).seals, 101)
		err := CheckRewoundLogsDB(i, plan, dafnyChainID(1))
		require.ErrorContains(t, err, "requires PlanConsistentWithLogs")
	})

	t.Run("conjunct 0: chain missing from target heads", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		delete(plan.TargetHeads, dafnyChainID(2))
		err := CheckRewoundLogsDB(i, plan, dafnyChainID(2))
		require.ErrorContains(t, err, "conjunct (0)")
	})

	t.Run("conjunct N1: logsDB not cleared after full rewind", func(t *testing.T) {
		t.Parallel()
		i := dafnyTestInterop(t)
		i.logsDBs[dafnyChainID(1)] = dafnySealedMock(dafnyHeadSeal(100, 1000))
		err := CheckRewoundLogsDB(i, RewindPlan{RewindAtOrAfter: 1000}, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (N1)")
	})

	t.Run("conjunct S1: latest sealed block beyond target head", func(t *testing.T) {
		t.Parallel()
		i, plan := dafnyRewoundInterop(t)
		mockLogsDBFor(t, i, 1).latest = dafnyBlock(102)
		err := CheckRewoundLogsDB(i, plan, dafnyChainID(1))
		require.ErrorContains(t, err, "conjunct (S1)")
	})
}

func TestCheckTransitionConsistentWithVerified(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid pending transition short-circuits", func(t *testing.T) {
		t.Parallel()
		err := CheckTransitionConsistentWithVerified(dafnySyncedInterop(t),
			PendingTransition{Decision: DecisionWait})
		require.ErrorContains(t, err, "requires ValidPendingTransition")
	})

	t.Run("requires: invalid verifiedDB short-circuits", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.verifiedDB.concrete().lastTimestamp = 999
		err := CheckTransitionConsistentWithVerified(i, dafnySyncedAdvance())
		require.ErrorContains(t, err, "requires verifiedDB.Valid()")
	})

	t.Run("pass: consistent rewind, invalidate, and advance", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, CheckTransitionConsistentWithVerified(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan}))
		require.NoError(t, CheckTransitionConsistentWithVerified(i, PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		require.NoError(t, CheckTransitionConsistentWithVerified(i, dafnySyncedAdvance()))
	})

	t.Run("conjunct R1: rewind plan inconsistent with verifiedDB", func(t *testing.T) {
		t.Parallel()
		plan := RewindPlan{
			RewindAtOrAfter:  1005,
			ResetAllChainsTo: ptrUint64(1004),
			TargetHeads:      dafnyHeads(map[uint64]uint64{1: 101, 2: 201}),
		}
		err := CheckTransitionConsistentWithVerified(dafnySyncedInterop(t),
			PendingTransition{Decision: DecisionRewind, Rewind: &plan})
		require.ErrorContains(t, err, "conjunct (R1)")
	})

	t.Run("conjunct A1: advance result does not advance the db", func(t *testing.T) {
		t.Parallel()
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		err := CheckTransitionConsistentWithVerified(dafnySyncedInterop(t), pending)
		require.ErrorContains(t, err, "conjunct (A1)")
	})
}

func TestCheckTransitionConsistentWithLogs(t *testing.T) {
	t.Parallel()

	t.Run("requires: invalid pending transition short-circuits", func(t *testing.T) {
		t.Parallel()
		err := CheckTransitionConsistentWithLogs(dafnySyncedInterop(t),
			PendingTransition{Decision: DecisionWait})
		require.ErrorContains(t, err, "requires ValidPendingTransition")
	})

	t.Run("pass: consistent rewind, invalidate, and advance", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, CheckTransitionConsistentWithLogs(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan}))
		require.NoError(t, CheckTransitionConsistentWithLogs(i, PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		require.NoError(t, CheckTransitionConsistentWithLogs(i, dafnySyncedAdvance()))
	})

	t.Run("conjunct R1: logsDB chain missing from target heads", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		i.logsDBs[dafnyChainID(3)] = dafnySealedMock(dafnyHeadSeal(300, 1000))
		plan := dafnySyncedPlan()
		err := CheckTransitionConsistentWithLogs(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan})
		require.ErrorContains(t, err, "conjunct (R1)")
		require.ErrorContains(t, err, "chain 3 not in plan.targetHeads")
	})

	t.Run("conjunct R2: rewind plan inconsistent with one chain's logs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 1).seals[101] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 101, Timestamp: 1001,
		}
		plan := dafnySyncedPlan()
		err := CheckTransitionConsistentWithLogs(i,
			PendingTransition{Decision: DecisionRewind, Rewind: &plan})
		require.ErrorContains(t, err, "conjunct (R2)")
		require.ErrorContains(t, err, "chain 1")
		require.NotContains(t, err.Error(), "chain 2")
	})

	t.Run("conjunct A1: result heads key set differs from logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckTransitionConsistentWithLogs(i, dafnySyncedAdvance())
		require.ErrorContains(t, err, "conjunct (A1)")
		require.NotContains(t, err.Error(), "conjunct (A2)")
	})

	t.Run("conjunct A2: advance result does not advance the logsDBs", func(t *testing.T) {
		t.Parallel()
		pending := dafnySyncedAdvance()
		pending.Result.L2Heads = dafnyHeads(map[uint64]uint64{1: 105, 2: 203})
		err := CheckTransitionConsistentWithLogs(dafnySyncedInterop(t), pending)
		require.ErrorContains(t, err, "conjunct (A2)")
		require.ErrorContains(t, err, "chain 1")
	})
}

func TestCheckPendingTransitionIsConsistent(t *testing.T) {
	t.Parallel()

	t.Run("requires: Valid() failure short-circuits", func(t *testing.T) {
		t.Parallel()
		err := CheckPendingTransitionIsConsistent(nil)
		require.ErrorContains(t, err, "requires Valid()")
	})

	t.Run("pass: no pending transition with synced DBs", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckPendingTransitionIsConsistent(dafnySyncedInterop(t)))
	})

	t.Run("conjunct N1: no pending transition with diverged DBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		mockLogsDBFor(t, i, 2).seals[201] = suptypes.BlockSeal{
			Hash: common.Hash{0xff}, Number: 201, Timestamp: 1001,
		}
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (N1)")
		require.ErrorContains(t, err, "chain 2")
	})

	t.Run("pass: stored advance, invalidate, and rewind transitions", func(t *testing.T) {
		t.Parallel()
		advance := dafnySyncedInterop(t)
		require.NoError(t, advance.verifiedDB.SetPendingTransition(dafnySyncedAdvance()))
		require.NoError(t, CheckPendingTransitionIsConsistent(advance))

		invalidate := dafnySyncedInterop(t)
		require.NoError(t, invalidate.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		require.NoError(t, CheckPendingTransitionIsConsistent(invalidate))

		rewind := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, rewind.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		require.NoError(t, CheckPendingTransitionIsConsistent(rewind))
	})

	t.Run("conjunct S1: stored advance inconsistent with verifiedDB", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S2)")
	})

	t.Run("conjunct S2: stored advance inconsistent with logsDBs", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.L2Heads = map[eth.ChainID]eth.BlockID{
			dafnyChainID(1): {Hash: common.Hash{0xbb}, Number: 102},
			dafnyChainID(2): dafnyBlock(203),
		}
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S2)")
		require.NotContains(t, err.Error(), "conjunct (S1)")
	})

	t.Run("conjunct S3: stored rewind with desynced prefix", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		plan := dafnySyncedPlan()
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionRewind,
			Rewind:   &plan,
		}))
		delete(mockLogsDBFor(t, i, 1).seals, 100)
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S3)")
		require.NotContains(t, err.Error(), "conjunct (S1)")
		require.NotContains(t, err.Error(), "conjunct (S2)")
	})

	t.Run("conjunct S3: stored invalidate with desynced tip", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		require.NoError(t, i.verifiedDB.SetPendingTransition(PendingTransition{
			Decision: DecisionInvalidate,
			Result:   &Result{Timestamp: 1003, L2Heads: dafnyHeads(map[uint64]uint64{1: 103, 2: 203})},
		}))
		mockLogsDBFor(t, i, 2).latest = dafnyBlock(205)
		err := CheckPendingTransitionIsConsistent(i)
		require.ErrorContains(t, err, "conjunct (S3)")
	})
}

func TestCheckInvariants(t *testing.T) {
	t.Parallel()

	t.Run("pass: fresh and synced instances", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, CheckInvariants(dafnyTestInterop(t)))
		require.NoError(t, CheckInvariants(dafnySyncedInterop(t)))
	})

	t.Run("conjunct 1: Valid() violation", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		delete(i.logsDBs, dafnyChainID(2))
		err := CheckInvariants(i)
		require.ErrorContains(t, err, "conjunct (1)")
		require.ErrorContains(t, err, "Interop.dfy Valid()")
		require.NotContains(t, err.Error(), "PendingTransitionIsConsistent")
	})

	t.Run("conjunct 2: stored pending transition inconsistent", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		err := CheckInvariants(i)
		require.ErrorContains(t, err, "conjunct (2)")
		require.ErrorContains(t, err, "PendingTransitionIsConsistent")
	})
}

func TestAssertInvariants(t *testing.T) {
	t.Parallel()

	t.Run("healthy instance passes on a real t", func(t *testing.T) {
		t.Parallel()
		AssertInvariants(t, dafnySyncedInterop(t))
	})

	t.Run("healthy instance leaves t untouched", func(t *testing.T) {
		t.Parallel()
		ft := &fakeDafnyT{}
		AssertInvariants(ft, dafnySyncedInterop(t))
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	t.Run("stored inconsistent pending transition fails t", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		pending := dafnySyncedAdvance()
		pending.Result.Timestamp = 1005
		require.NoError(t, i.verifiedDB.SetPendingTransition(pending))
		ft := &fakeDafnyT{}
		AssertInvariants(ft, i)
		require.True(t, ft.errorfCalled)
		require.True(t, ft.failNowCalled)
	})
}

func TestTransitionAsserts(t *testing.T) {
	t.Parallel()

	t.Run("pass leaves t untouched", func(t *testing.T) {
		t.Parallel()
		i := dafnySyncedInterop(t)
		r, plan := dafnyRewoundInterop(t)
		ft := &fakeDafnyT{}
		AssertAdvancesVerifiedDB(ft, i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		AssertAdvancesLogsDB(ft, i, 1003, dafnyChainID(1), dafnyBlock(103))
		AssertAdvancesAllLogsDBs(ft, i, 1003, dafnyHeads(map[uint64]uint64{1: 103, 2: 203}))
		AssertPlanConsistentWithVerified(ft, i, dafnySyncedPlan())
		AssertPlanConsistentWithLogs(ft, i, dafnySyncedPlan(), dafnyChainID(1))
		AssertRewoundVerifiedDB(ft, r, plan)
		AssertRewoundLogsDB(ft, r, plan, dafnyChainID(1))
		AssertTransitionConsistentWithVerified(ft, i, dafnySyncedAdvance())
		AssertTransitionConsistentWithLogs(ft, i, dafnySyncedAdvance())
		AssertPendingTransitionIsConsistent(ft, i)
		require.True(t, ft.helperCalled)
		require.False(t, ft.errorfCalled)
		require.False(t, ft.failNowCalled)
	})

	asserts := map[string]func(dafnyT){
		"AssertAdvancesVerifiedDB": func(ft dafnyT) { AssertAdvancesVerifiedDB(ft, nil, 1000, nil) },
		"AssertAdvancesLogsDB":     func(ft dafnyT) { AssertAdvancesLogsDB(ft, nil, 1000, dafnyChainID(1), dafnyBlock(100)) },
		"AssertAdvancesAllLogsDBs": func(ft dafnyT) { AssertAdvancesAllLogsDBs(ft, nil, 1000, nil) },
		"AssertPlanConsistentWithVerified": func(ft dafnyT) {
			AssertPlanConsistentWithVerified(ft, nil, RewindPlan{})
		},
		"AssertPlanConsistentWithLogs": func(ft dafnyT) {
			AssertPlanConsistentWithLogs(ft, nil, RewindPlan{}, dafnyChainID(1))
		},
		"AssertRewoundVerifiedDB": func(ft dafnyT) { AssertRewoundVerifiedDB(ft, nil, RewindPlan{}) },
		"AssertRewoundLogsDB":     func(ft dafnyT) { AssertRewoundLogsDB(ft, nil, RewindPlan{}, dafnyChainID(1)) },
		"AssertTransitionConsistentWithVerified": func(ft dafnyT) {
			AssertTransitionConsistentWithVerified(ft, nil, PendingTransition{})
		},
		"AssertTransitionConsistentWithLogs": func(ft dafnyT) {
			AssertTransitionConsistentWithLogs(ft, nil, PendingTransition{})
		},
		"AssertPendingTransitionIsConsistent": func(ft dafnyT) { AssertPendingTransitionIsConsistent(ft, nil) },
		"AssertInvariants":                    func(ft dafnyT) { AssertInvariants(ft, nil) },
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
