package interop

// This file is part of the Dafny model checkers (dafny_check_*.go files):
// test/debug-only helpers that check predicates from op-supernode/dafny-models/
// against the real Go types. Production code paths must not call them.

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// sortedChainIDs returns the key set of a chain-keyed map in ascending order
// so violation reports are deterministic.
func sortedChainIDs[V any](m map[eth.ChainID]V) []eth.ChainID {
	return slices.SortedFunc(maps.Keys(m), eth.ChainID.Cmp)
}

// sameChainIDKeys reports whether two chain-keyed maps have equal key sets
// (model expression `a.Keys == b.Keys`).
func sameChainIDKeys[V1, V2 any](a map[eth.ChainID]V1, b map[eth.ChainID]V2) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// rewindPlanEqual reports model equality of two RewindPlan values: Option
// fields compare as nil pointers (None) vs pointed-to values (Some),
// targetHeads as map equality (nil and empty are both the empty model map).
func rewindPlanEqual(a, b RewindPlan) bool {
	if a.RewindAtOrAfter != b.RewindAtOrAfter {
		return false
	}
	if (a.ResetAllChainsTo == nil) != (b.ResetAllChainsTo == nil) {
		return false
	}
	if a.ResetAllChainsTo != nil && *a.ResetAllChainsTo != *b.ResetAllChainsTo {
		return false
	}
	return maps.Equal(a.TargetHeads, b.TargetHeads)
}

// checkAdvancesVerifiedDB is the body of AdvancesVerifiedDB(ts, blocksAtTS);
// callers have already established the model's `requires verifiedDB.Valid()`.
func checkAdvancesVerifiedDB(i *Interop, ts uint64, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy AdvancesVerifiedDB"
	p := modelParamsFromInterop(i)

	lastTS, initialized := i.verifiedDB.LastTimestamp()
	if !initialized {
		if ts != p.ActivationTimestamp {
			return violation(pred, "N1",
				"verifiedDB is empty but ts %d != ACTIVATION_TIMESTAMP %d", ts, p.ActivationTimestamp)
		}
		return nil
	}

	lastResult, err := i.verifiedDB.Get(lastTS)
	if err != nil {
		return violation(pred, "0", "verifiedDB.Get(%d) failed: %v", lastTS, err)
	}

	var errs []error
	if ts != lastTS+1 {
		errs = append(errs, violation(pred, "S1", "ts %d != lastTS + 1 == %d", ts, lastTS+1))
	}
	if !sameChainIDKeys(blocks, lastResult.L2Heads) {
		errs = append(errs, violation(pred, "S2",
			"blocksAtTS.Keys %v != lastVerifiedResult.l2Heads.Keys %v",
			sortedChainIDs(blocks), sortedChainIDs(lastResult.L2Heads)))
	}
	for _, cid := range sortedChainIDs(blocks) {
		lastBlock, ok := lastResult.L2Heads[cid]
		if !ok {
			continue // missing chain already reported by (S2)
		}
		if n := blocks[cid].Number; n < lastBlock.Number || n-lastBlock.Number > 1 {
			errs = append(errs, violation(pred, "S3",
				"chain %s: new head number %d not in [%d, %d]",
				cid, n, lastBlock.Number, lastBlock.Number+1))
		}
	}
	return errors.Join(errs...)
}

// CheckAdvancesVerifiedDB mirrors AdvancesVerifiedDB(ts, blocksAtTS) in
// op-supernode/dafny-models/Interop.dfy. The model's `requires
// verifiedDB.Valid()` is checked first via CheckVerifiedDBValid; on failure
// the predicate body is skipped. Conjuncts, matching on
// verifiedDB.LastTimestamp():
//
//	(0) i is non-nil and DB reads succeed (mapping requirement)
//	None case:
//	  (N1) ts == ACTIVATION_TIMESTAMP
//	Some(lastTS) case:
//	  (S1) ts == lastTS + 1
//	  (S2) blocksAtTS.Keys == verifiedDB.Get(lastTS).l2Heads.Keys
//	  (S3) forall chainID in blocksAtTS.Keys:
//	      lastBlock.number <= blocksAtTS[chainID].number <= lastBlock.number + 1
func CheckAdvancesVerifiedDB(i *Interop, ts uint64, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy AdvancesVerifiedDB"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckVerifiedDBValid(i.verifiedDB.concrete()); err != nil {
		return fmt.Errorf("%s requires verifiedDB.Valid(): %w", pred, err)
	}
	return checkAdvancesVerifiedDB(i, ts, blocks)
}

// AssertAdvancesVerifiedDB fails t when CheckAdvancesVerifiedDB reports
// violations.
func AssertAdvancesVerifiedDB(t dafnyT, i *Interop, ts uint64, blocks map[eth.ChainID]eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckAdvancesVerifiedDB(i, ts, blocks))
}

// CheckAdvancesLogsDB mirrors AdvancesLogsDB(ts, chainID, newBlock) in
// op-supernode/dafny-models/Interop.dfy. Conjuncts, matching on
// logsDBs[chainID].LatestSealedBlock():
//
//	(0) i is non-nil and chainID in logsDBs (mapping requirement and the
//	    model's requires clause)
//	None case:
//	  (N1) ts == ACTIVATION_TIMESTAMP
//	Some(latestBlock) case (ts is unused, as in the model):
//	  (S1) latestBlock.number <= newBlock.number <= latestBlock.number + 1
//	  (S2) latestBlock.number == newBlock.number ==> latestBlock == newBlock
func CheckAdvancesLogsDB(i *Interop, ts uint64, chainID eth.ChainID, newBlock eth.BlockID) error {
	const pred = "Interop.dfy AdvancesLogsDB"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	db, ok := i.logsDBs[chainID]
	if !ok || db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}

	latest, has := db.LatestSealedBlock()
	if !has {
		p := modelParamsFromInterop(i)
		if ts != p.ActivationTimestamp {
			return violation(pred, "N1",
				"chain %s logsDB is empty but ts %d != ACTIVATION_TIMESTAMP %d",
				chainID, ts, p.ActivationTimestamp)
		}
		return nil
	}

	var errs []error
	if newBlock.Number < latest.Number || newBlock.Number-latest.Number > 1 {
		errs = append(errs, violation(pred, "S1",
			"chain %s: new block number %d not in [%d, %d]",
			chainID, newBlock.Number, latest.Number, latest.Number+1))
	}
	if newBlock.Number == latest.Number && newBlock != latest {
		errs = append(errs, violation(pred, "S2",
			"chain %s: new block %s at the latest number differs from latest sealed block %s",
			chainID, newBlock, latest))
	}
	return errors.Join(errs...)
}

// AssertAdvancesLogsDB fails t when CheckAdvancesLogsDB reports violations.
func AssertAdvancesLogsDB(t dafnyT, i *Interop, ts uint64, chainID eth.ChainID, newBlock eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckAdvancesLogsDB(i, ts, chainID, newBlock))
}

// checkAdvancesAllLogsDBs is the body of AdvancesAllLogsDBs(ts, blocksAtTS);
// callers have already established the model's `requires blocksAtTS.Keys ==
// logsDBs.Keys`.
func checkAdvancesAllLogsDBs(i *Interop, ts uint64, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy AdvancesAllLogsDBs"
	var errs []error
	for _, k := range sortedLogsDBChainIDs(i) {
		if err := CheckAdvancesLogsDB(i, ts, k, blocks[k]); err != nil {
			errs = append(errs, fmt.Errorf("%s: chain %s: %w", pred, k, err))
		}
	}
	return errors.Join(errs...)
}

// CheckAdvancesAllLogsDBs mirrors AdvancesAllLogsDBs(ts, blocksAtTS) in
// op-supernode/dafny-models/Interop.dfy: AdvancesLogsDB(ts, k, blocksAtTS[k])
// for every k in logsDBs.Keys. The model's `requires blocksAtTS.Keys ==
// logsDBs.Keys` is reported as conjunct (0); on failure the per-chain bodies
// are skipped. Violations carry the failing chain's ID; conjunct labels are
// those of CheckAdvancesLogsDB.
func CheckAdvancesAllLogsDBs(i *Interop, ts uint64, blocks map[eth.ChainID]eth.BlockID) error {
	const pred = "Interop.dfy AdvancesAllLogsDBs"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if !sameChainIDKeys(blocks, i.logsDBs) {
		return violation(pred, "0",
			"requires blocksAtTS.Keys == logsDBs.Keys: %v vs %v",
			sortedChainIDs(blocks), sortedLogsDBChainIDs(i))
	}
	return checkAdvancesAllLogsDBs(i, ts, blocks)
}

// AssertAdvancesAllLogsDBs fails t when CheckAdvancesAllLogsDBs reports
// violations.
func AssertAdvancesAllLogsDBs(t dafnyT, i *Interop, ts uint64, blocks map[eth.ChainID]eth.BlockID) {
	t.Helper()
	failOnViolation(t, CheckAdvancesAllLogsDBs(i, ts, blocks))
}

// checkPlanConsistentWithVerified is the body of PlanConsistentWithVerified;
// callers have already checked i for nil.
func checkPlanConsistentWithVerified(i *Interop, plan RewindPlan) error {
	const pred = "Interop.dfy PlanConsistentWithVerified"
	if i.verifiedDB == nil || i.verifiedDB.concrete() == nil || i.verifiedDB.concrete().db == nil {
		return violation(pred, "0", "VerifiedDB has no underlying store")
	}
	if plan.ResetAllChainsTo == nil {
		return nil
	}
	ts := *plan.ResetAllChainsTo

	db, err := i.verifiedDB.concrete().allVerified()
	if err != nil {
		return violation(pred, "0", "enumerate verified bucket: %v", err)
	}

	var errs []error
	entry, hasTS := db[ts]
	if !hasTS {
		errs = append(errs, violation(pred, "S1",
			"resetAllChainsTo %d not in verifiedDB", ts))
	}
	for _, t := range slices.Sorted(maps.Keys(db)) {
		if t < plan.RewindAtOrAfter && t > ts {
			errs = append(errs, violation(pred, "S2",
				"committed timestamp %d below rewindAtOrAfter %d exceeds resetAllChainsTo %d",
				t, plan.RewindAtOrAfter, ts))
		}
	}
	if hasTS && !maps.Equal(plan.TargetHeads, entry.L2Heads) {
		errs = append(errs, violation(pred, "S3",
			"targetHeads %v != verifiedDB.Get(%d).l2Heads %v", plan.TargetHeads, ts, entry.L2Heads))
	}
	return errors.Join(errs...)
}

// CheckPlanConsistentWithVerified mirrors PlanConsistentWithVerified(plan) in
// op-supernode/dafny-models/Interop.dfy. The None case of
// plan.resetAllChainsTo is vacuously consistent. Conjuncts (Some(ts) case):
//
//	(0) i is non-nil, the store is reachable, and every stored entry maps to
//	    the model db (mapping requirement)
//	(S1) verifiedDB.Has(ts)
//	(S2) forall t < plan.rewindAtOrAfter with verifiedDB.Has(t): t <= ts
//	(S3) plan.targetHeads == verifiedDB.Get(ts).l2Heads; skipped when (S1)
//	     already failed
func CheckPlanConsistentWithVerified(i *Interop, plan RewindPlan) error {
	if i == nil {
		return violation("Interop.dfy PlanConsistentWithVerified", "0", "Interop is nil")
	}
	return checkPlanConsistentWithVerified(i, plan)
}

// AssertPlanConsistentWithVerified fails t when CheckPlanConsistentWithVerified
// reports violations.
func AssertPlanConsistentWithVerified(t dafnyT, i *Interop, plan RewindPlan) {
	t.Helper()
	failOnViolation(t, CheckPlanConsistentWithVerified(i, plan))
}

// checkPlanConsistentWithLogs is the body of PlanConsistentWithLogs(plan,
// chainID); callers have already established the model's requires clauses
// (chainID in logsDBs.Keys, and chainID in plan.targetHeads when
// resetAllChainsTo is Some).
func checkPlanConsistentWithLogs(i *Interop, plan RewindPlan, chainID eth.ChainID) error {
	const pred = "Interop.dfy PlanConsistentWithLogs"
	if plan.ResetAllChainsTo == nil {
		return nil
	}
	db := i.logsDBs[chainID]
	if db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}
	target := plan.TargetHeads[chainID]

	seal, found, ferr := findSealedOption(db, target.Number)
	if ferr != nil {
		return violation(pred, "0",
			"chain %s FindSealedBlock(%d) failed: %v", chainID, target.Number, ferr)
	}
	if !found {
		return violation(pred, "S1",
			"chain %s: target head %s is not sealed in the logsDB", chainID, target)
	}
	if seal.ID() != target {
		return violation(pred, "S2",
			"chain %s: sealed block %s != target head %s", chainID, seal.ID(), target)
	}
	return nil
}

// CheckPlanConsistentWithLogs mirrors PlanConsistentWithLogs(plan, chainID) in
// op-supernode/dafny-models/Interop.dfy. The None case of
// plan.resetAllChainsTo is always consistent (Clear() has no preconditions).
// Conjuncts (Some case):
//
//	(0) i is non-nil, chainID in logsDBs.Keys, chainID in plan.targetHeads,
//	    and FindSealedBlock errors are not-found sentinels (mapping
//	    requirement and the model's requires clauses)
//	(S1) FindSealedBlock(plan.targetHeads[chainID].number).Some?
//	(S2) the found seal's id == plan.targetHeads[chainID]
func CheckPlanConsistentWithLogs(i *Interop, plan RewindPlan, chainID eth.ChainID) error {
	const pred = "Interop.dfy PlanConsistentWithLogs"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if db, ok := i.logsDBs[chainID]; !ok || db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}
	if plan.ResetAllChainsTo != nil {
		if _, ok := plan.TargetHeads[chainID]; !ok {
			return violation(pred, "0", "requires chain %s in plan.targetHeads", chainID)
		}
	}
	return checkPlanConsistentWithLogs(i, plan, chainID)
}

// AssertPlanConsistentWithLogs fails t when CheckPlanConsistentWithLogs
// reports violations.
func AssertPlanConsistentWithLogs(t dafnyT, i *Interop, plan RewindPlan, chainID eth.ChainID) {
	t.Helper()
	failOnViolation(t, CheckPlanConsistentWithLogs(i, plan, chainID))
}

// checkRewoundVerifiedDB is the body of RewoundVerifiedDB(plan); callers have
// already established the model's `requires PlanConsistentWithVerified(plan)`.
func checkRewoundVerifiedDB(i *Interop, plan RewindPlan) error {
	const pred = "Interop.dfy RewoundVerifiedDB"

	var errs []error
	pending, err := i.verifiedDB.GetPendingTransition()
	switch {
	case err != nil:
		errs = append(errs, violation(pred, "0", "GetPendingTransition failed: %v", err))
	case pending == nil:
		errs = append(errs, violation(pred, "1", "pendingTransition is None"))
	default:
		if pending.Decision != DecisionRewind {
			errs = append(errs, violation(pred, "2",
				"pending decision %s != Rewind", pending.Decision))
		}
		if pending.Rewind == nil || !rewindPlanEqual(*pending.Rewind, plan) {
			errs = append(errs, violation(pred, "3",
				"stored pending rewind plan %+v != plan %+v", pending.Rewind, plan))
		}
	}

	if plan.ResetAllChainsTo == nil {
		db, derr := i.verifiedDB.concrete().allVerified()
		switch {
		case derr != nil:
			errs = append(errs, violation(pred, "0", "enumerate verified bucket: %v", derr))
		case len(db) != 0:
			errs = append(errs, violation(pred, "N1",
				"db has %d entries after a full rewind", len(db)))
		}
		return errors.Join(errs...)
	}

	ts := *plan.ResetAllChainsTo
	if last, initialized := i.verifiedDB.LastTimestamp(); !initialized {
		errs = append(errs, violation(pred, "S1",
			"LastTimestamp() is None but resetAllChainsTo is %d", ts))
	} else if last != ts {
		errs = append(errs, violation(pred, "S1",
			"LastTimestamp() %d != resetAllChainsTo %d", last, ts))
	}
	if result, gerr := i.verifiedDB.Get(ts); gerr != nil {
		errs = append(errs, violation(pred, "0", "verifiedDB.Get(%d) failed: %v", ts, gerr))
	} else if !maps.Equal(plan.TargetHeads, result.L2Heads) {
		errs = append(errs, violation(pred, "S2",
			"targetHeads %v != verifiedDB.Get(%d).l2Heads %v", plan.TargetHeads, ts, result.L2Heads))
	}
	return errors.Join(errs...)
}

// CheckRewoundVerifiedDB mirrors RewoundVerifiedDB(plan) in
// op-supernode/dafny-models/Interop.dfy. The model's `requires
// PlanConsistentWithVerified(plan)` is checked first; on failure the predicate
// body is skipped. Conjuncts:
//
//	(0) i is non-nil, the store is reachable, and reads succeed (mapping
//	    requirement)
//	(1) verifiedDB.pendingTransition.Some?
//	(2) the stored pending decision == Rewind; skipped when (1) failed
//	(3) the stored pending rewind == Some(plan); skipped when (1) failed
//	None case (plan.resetAllChainsTo):
//	  (N1) |verifiedDB.db| == 0
//	Some(ts) case:
//	  (S1) verifiedDB.LastTimestamp() == Some(ts)
//	  (S2) plan.targetHeads == verifiedDB.Get(ts).l2Heads
func CheckRewoundVerifiedDB(i *Interop, plan RewindPlan) error {
	const pred = "Interop.dfy RewoundVerifiedDB"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if i.verifiedDB == nil || i.verifiedDB.concrete() == nil || i.verifiedDB.concrete().db == nil {
		return violation(pred, "0", "VerifiedDB has no underlying store")
	}
	if err := checkPlanConsistentWithVerified(i, plan); err != nil {
		return fmt.Errorf("%s requires PlanConsistentWithVerified: %w", pred, err)
	}
	return checkRewoundVerifiedDB(i, plan)
}

// AssertRewoundVerifiedDB fails t when CheckRewoundVerifiedDB reports
// violations.
func AssertRewoundVerifiedDB(t dafnyT, i *Interop, plan RewindPlan) {
	t.Helper()
	failOnViolation(t, CheckRewoundVerifiedDB(i, plan))
}

// checkRewoundLogsDB is the body of RewoundLogsDB(plan, chainID); callers have
// already established the model's requires clauses (chainID in logsDBs.Keys,
// chainID in plan.targetHeads when Some, PlanConsistentWithLogs).
func checkRewoundLogsDB(i *Interop, plan RewindPlan, chainID eth.ChainID) error {
	const pred = "Interop.dfy RewoundLogsDB"
	db := i.logsDBs[chainID]
	if db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}

	latest, has := db.LatestSealedBlock()
	if plan.ResetAllChainsTo == nil {
		if has {
			return violation(pred, "N1",
				"chain %s LatestSealedBlock is Some(%s) after a full rewind", chainID, latest)
		}
		return nil
	}

	target := plan.TargetHeads[chainID]
	if !has {
		return violation(pred, "S1",
			"chain %s LatestSealedBlock is None but target head is %s", chainID, target)
	}
	if latest != target {
		return violation(pred, "S1",
			"chain %s LatestSealedBlock %s != target head %s", chainID, latest, target)
	}
	return nil
}

// CheckRewoundLogsDB mirrors RewoundLogsDB(plan, chainID) in
// op-supernode/dafny-models/Interop.dfy. The model's `requires
// PlanConsistentWithLogs(plan, chainID)` is checked first; on failure the
// predicate body is skipped. Conjuncts, matching on plan.resetAllChainsTo:
//
//	(0) i is non-nil, chainID in logsDBs.Keys, and chainID in plan.targetHeads
//	    when Some (mapping requirement and the model's requires clauses)
//	None case:
//	  (N1) logsDBs[chainID].LatestSealedBlock() == None
//	Some case:
//	  (S1) logsDBs[chainID].LatestSealedBlock() == Some(plan.targetHeads[chainID])
func CheckRewoundLogsDB(i *Interop, plan RewindPlan, chainID eth.ChainID) error {
	const pred = "Interop.dfy RewoundLogsDB"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if db, ok := i.logsDBs[chainID]; !ok || db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}
	if plan.ResetAllChainsTo != nil {
		if _, ok := plan.TargetHeads[chainID]; !ok {
			return violation(pred, "0", "requires chain %s in plan.targetHeads", chainID)
		}
	}
	if err := checkPlanConsistentWithLogs(i, plan, chainID); err != nil {
		return fmt.Errorf("%s requires PlanConsistentWithLogs: %w", pred, err)
	}
	return checkRewoundLogsDB(i, plan, chainID)
}

// AssertRewoundLogsDB fails t when CheckRewoundLogsDB reports violations.
func AssertRewoundLogsDB(t dafnyT, i *Interop, plan RewindPlan, chainID eth.ChainID) {
	t.Helper()
	failOnViolation(t, CheckRewoundLogsDB(i, plan, chainID))
}

// checkTransitionConsistentWithVerified is the body of
// TransitionConsistentWithVerified(pending); callers have already established
// the model's `requires verifiedDB.Valid() && ValidPendingTransition(pending)`
// (so the Option fields matching the decision are present).
func checkTransitionConsistentWithVerified(i *Interop, pending PendingTransition) error {
	const pred = "Interop.dfy TransitionConsistentWithVerified"
	switch pending.Decision {
	case DecisionRewind:
		if pending.Rewind == nil {
			return violation(pred, "0", "decision is Rewind but rewind is None")
		}
		if err := checkPlanConsistentWithVerified(i, *pending.Rewind); err != nil {
			return fmt.Errorf("%s conjunct (R1): %w", pred, err)
		}
	case DecisionAdvance:
		if pending.Result == nil {
			return violation(pred, "0", "decision is Advance but result is None")
		}
		if err := checkAdvancesVerifiedDB(i, pending.Result.Timestamp, pending.Result.L2Heads); err != nil {
			return fmt.Errorf("%s conjunct (A1): %w", pred, err)
		}
	}
	return nil // Invalidate case: true
}

// CheckTransitionConsistentWithVerified mirrors
// TransitionConsistentWithVerified(pending) in
// op-supernode/dafny-models/Interop.dfy. The model's `requires
// verifiedDB.Valid()` and `requires ValidPendingTransition(pending)` are
// checked first; on failure the predicate body is skipped. Conjuncts, matching
// on pending.decision:
//
//	(0) i is non-nil (mapping requirement)
//	Rewind:     (R1) PlanConsistentWithVerified(pending.rewind.value)
//	Invalidate: true (nothing to check)
//	Advance:    (A1) AdvancesVerifiedDB(result.timestamp, result.l2Heads)
func CheckTransitionConsistentWithVerified(i *Interop, pending PendingTransition) error {
	const pred = "Interop.dfy TransitionConsistentWithVerified"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckVerifiedDBValid(i.verifiedDB.concrete()); err != nil {
		return fmt.Errorf("%s requires verifiedDB.Valid(): %w", pred, err)
	}
	if err := CheckValidPendingTransition(modelParamsFromInterop(i), pending); err != nil {
		return fmt.Errorf("%s requires ValidPendingTransition: %w", pred, err)
	}
	return checkTransitionConsistentWithVerified(i, pending)
}

// AssertTransitionConsistentWithVerified fails t when
// CheckTransitionConsistentWithVerified reports violations.
func AssertTransitionConsistentWithVerified(t dafnyT, i *Interop, pending PendingTransition) {
	t.Helper()
	failOnViolation(t, CheckTransitionConsistentWithVerified(i, pending))
}

// checkTransitionConsistentWithLogs is the body of
// TransitionConsistentWithLogs(pending); callers have already established the
// model's `requires ValidPendingTransition(pending)`.
func checkTransitionConsistentWithLogs(i *Interop, pending PendingTransition) error {
	const pred = "Interop.dfy TransitionConsistentWithLogs"
	switch pending.Decision {
	case DecisionRewind:
		if pending.Rewind == nil {
			return violation(pred, "0", "decision is Rewind but rewind is None")
		}
		plan := *pending.Rewind
		if plan.ResetAllChainsTo == nil {
			return nil
		}
		var errs []error
		for _, k := range sortedLogsDBChainIDs(i) {
			if _, ok := plan.TargetHeads[k]; !ok {
				errs = append(errs, violation(pred, "R1",
					"chain %s not in plan.targetHeads", k))
				continue
			}
			if err := checkPlanConsistentWithLogs(i, plan, k); err != nil {
				errs = append(errs, fmt.Errorf("%s conjunct (R2): chain %s: %w", pred, k, err))
			}
		}
		return errors.Join(errs...)
	case DecisionAdvance:
		if pending.Result == nil {
			return violation(pred, "0", "decision is Advance but result is None")
		}
		if !sameChainIDKeys(pending.Result.L2Heads, i.logsDBs) {
			return violation(pred, "A1",
				"result.l2Heads.Keys %v != logsDBs.Keys %v",
				sortedChainIDs(pending.Result.L2Heads), sortedLogsDBChainIDs(i))
		}
		if err := checkAdvancesAllLogsDBs(i, pending.Result.Timestamp, pending.Result.L2Heads); err != nil {
			return fmt.Errorf("%s conjunct (A2): %w", pred, err)
		}
		return nil
	}
	return nil // Invalidate case: true
}

// CheckTransitionConsistentWithLogs mirrors
// TransitionConsistentWithLogs(pending) in
// op-supernode/dafny-models/Interop.dfy. The model's `requires
// ValidPendingTransition(pending)` is checked first; on failure the predicate
// body is skipped. Conjuncts, matching on pending.decision:
//
//	(0) i is non-nil (mapping requirement)
//	Rewind, when resetAllChainsTo is Some, forall k in logsDBs.Keys:
//	  (R1) k in pending.rewind.value.targetHeads
//	  (R2) PlanConsistentWithLogs(pending.rewind.value, k); skipped per chain
//	       when (R1) failed
//	Invalidate: true (nothing to check)
//	Advance:
//	  (A1) result.l2Heads.Keys == logsDBs.Keys
//	  (A2) AdvancesAllLogsDBs(result.timestamp, result.l2Heads); skipped when
//	       (A1) failed
func CheckTransitionConsistentWithLogs(i *Interop, pending PendingTransition) error {
	const pred = "Interop.dfy TransitionConsistentWithLogs"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckValidPendingTransition(modelParamsFromInterop(i), pending); err != nil {
		return fmt.Errorf("%s requires ValidPendingTransition: %w", pred, err)
	}
	return checkTransitionConsistentWithLogs(i, pending)
}

// AssertTransitionConsistentWithLogs fails t when
// CheckTransitionConsistentWithLogs reports violations.
func AssertTransitionConsistentWithLogs(t dafnyT, i *Interop, pending PendingTransition) {
	t.Helper()
	failOnViolation(t, CheckTransitionConsistentWithLogs(i, pending))
}

// checkAllDBsInSyncBody loops checkDBsInSync over all chains; callers have
// already established `requires verifiedDB.Valid()`. label names the
// enclosing conjunct in violation reports.
func checkAllDBsInSyncBody(i *Interop, pred, label string) error {
	var errs []error
	for _, k := range sortedLogsDBChainIDs(i) {
		if err := checkDBsInSync(i, k); err != nil {
			errs = append(errs, fmt.Errorf("%s conjunct (%s): AllDBsInSync: chain %s: %w", pred, label, k, err))
		}
	}
	return errors.Join(errs...)
}

// checkPendingTransitionIsConsistent is the body of
// PendingTransitionIsConsistent(); callers have already established the
// model's `requires Valid()` (which includes ValidPendingTransition of the
// stored pending transition).
func checkPendingTransitionIsConsistent(i *Interop) error {
	const pred = "Interop.dfy PendingTransitionIsConsistent"
	pending, err := i.verifiedDB.GetPendingTransition()
	if err != nil {
		return violation(pred, "0", "GetPendingTransition failed: %v", err)
	}

	if pending == nil {
		return checkAllDBsInSyncBody(i, pred, "N1")
	}

	var errs []error
	if err := checkTransitionConsistentWithVerified(i, *pending); err != nil {
		errs = append(errs, fmt.Errorf("%s conjunct (S1): %w", pred, err))
	}
	if err := checkTransitionConsistentWithLogs(i, *pending); err != nil {
		errs = append(errs, fmt.Errorf("%s conjunct (S2): %w", pred, err))
	}
	switch pending.Decision {
	case DecisionRewind:
		if pending.Rewind != nil && pending.Rewind.ResetAllChainsTo != nil {
			if err := CheckAllDBsInSyncUpTo(i, *pending.Rewind.ResetAllChainsTo); err != nil {
				errs = append(errs, fmt.Errorf("%s conjunct (S3): %w", pred, err))
			}
		}
	case DecisionInvalidate:
		if err := checkAllDBsInSyncBody(i, pred, "S3"); err != nil {
			errs = append(errs, err)
		}
	case DecisionAdvance:
		if lastTS, initialized := i.verifiedDB.LastTimestamp(); initialized {
			if err := CheckAllDBsInSyncUpTo(i, lastTS); err != nil {
				errs = append(errs, fmt.Errorf("%s conjunct (S3): %w", pred, err))
			}
		}
	}
	return errors.Join(errs...)
}

// CheckPendingTransitionIsConsistent mirrors PendingTransitionIsConsistent()
// in op-supernode/dafny-models/Interop.dfy. The model's `requires Valid()` is
// checked first via CheckInteropValid; on failure the predicate body is
// skipped (Valid() already implies ValidPendingTransition of the stored
// pending transition, so it is not re-checked here, as in the model).
// Conjuncts, matching on verifiedDB.GetPendingTransition():
//
//	(0) i is non-nil and reads succeed (mapping requirement)
//	None case:
//	  (N1) AllDBsInSync()
//	Some(p) case:
//	  (S1) TransitionConsistentWithVerified(p)
//	  (S2) TransitionConsistentWithLogs(p)
//	  (S3) matching on p.decision:
//	    Rewind:     p.rewind.value.resetAllChainsTo.Some? ==>
//	                AllDBsInSyncUpTo(resetAllChainsTo.value)
//	    Invalidate: AllDBsInSync()
//	    Advance:    verifiedDB.LastTimestamp().Some? ==>
//	                AllDBsInSyncUpTo(LastTimestamp().value)
func CheckPendingTransitionIsConsistent(i *Interop) error {
	const pred = "Interop.dfy PendingTransitionIsConsistent"
	if err := CheckInteropValid(i); err != nil {
		return fmt.Errorf("%s requires Valid(): %w", pred, err)
	}
	return checkPendingTransitionIsConsistent(i)
}

// AssertPendingTransitionIsConsistent fails t when
// CheckPendingTransitionIsConsistent reports violations.
func AssertPendingTransitionIsConsistent(t dafnyT, i *Interop) {
	t.Helper()
	failOnViolation(t, CheckPendingTransitionIsConsistent(i))
}

// CheckInvariants mirrors the requires/ensures of Interop.ProgressAndRecord in
// op-supernode/dafny-models/Interop.dfy: Valid() &&
// PendingTransitionIsConsistent(). Conjuncts:
//
//	(1) Valid(), via CheckInteropValid
//	(2) PendingTransitionIsConsistent(); skipped when (1) failed (the model
//	    predicate requires Valid())
func CheckInvariants(i *Interop) error {
	const pred = "Interop.dfy ProgressAndRecord requires/ensures"
	if err := CheckInteropValid(i); err != nil {
		return fmt.Errorf("%s conjunct (1): %w", pred, err)
	}
	if err := checkPendingTransitionIsConsistent(i); err != nil {
		return fmt.Errorf("%s conjunct (2): %w", pred, err)
	}
	return nil
}

// AssertInvariants checks Interop.Valid() && PendingTransitionIsConsistent()
// from Interop.dfy on the live Interop instance, mirroring the requires and
// ensures of ProgressAndRecord. Tests call it before and after exercising
// progressAndRecord / applyPendingTransition / applyRewindPlan.
func AssertInvariants(t dafnyT, i *Interop) {
	t.Helper()
	failOnViolation(t, CheckInvariants(i))
}
