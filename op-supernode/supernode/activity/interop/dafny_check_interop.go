package interop

// This file is part of the Dafny model checkers (dafny_check_*.go files):
// test/debug-only helpers that check predicates from op-supernode/dafny-models/
// against the real Go types. Production code paths must not call them.

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// sameLogsDB reports whether two LogsDB interface values are the same model
// object; uncomparable dynamic types are treated as distinct.
func sameLogsDB(a, b LogsDB) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if !va.Comparable() || !vb.Comparable() {
		return false
	}
	return va.Equal(vb)
}

// sortedLogsDBChainIDs returns the key set of i.logsDBs in ascending order so
// violation reports are deterministic.
func sortedLogsDBChainIDs(i *Interop) []eth.ChainID {
	return slices.SortedFunc(maps.Keys(i.logsDBs), eth.ChainID.Cmp)
}

// CheckInteropValid mirrors the class invariant Interop.Valid() in
// op-supernode/dafny-models/Interop.dfy, with the model ghost constants
// instantiated by modelParamsFromInterop. The model conjunct
// `activationTimestamp == ACTIVATION_TIMESTAMP` is definitional under that
// mapping (ModelParams.ActivationTimestamp is derived from the instance) and
// has no separate check. Conjuncts:
//
//	(0) i is non-nil and the model state is reachable (mapping requirement)
//	(1) chains.Keys == CHAIN_IDS (definitional under modelParamsFromInterop,
//	    checked for explicitness)
//	(2) logsDBs.Keys == CHAIN_IDS
//	(3) all logsDBs are distinct
//	(4) verifiedDB.Valid(), via the embedded CheckVerifiedDBValid; conjuncts
//	    (5)-(7) read the model db and are skipped when it fails
//	(5) lastTimestamp.Some? ==> ACTIVATION_TIMESTAMP in verifiedDB.db
//	(6) forall ts in verifiedDB.db: ACTIVATION_TIMESTAMP <= ts
//	(7) forall ts in verifiedDB.db: db[ts].l2Heads.Keys == CHAIN_IDS
//	(8) pendingTransition.Some? ==>
//	    ValidPendingTransition(GetPendingTransition().value)
func CheckInteropValid(i *Interop) error {
	const pred = "Interop.dfy Valid()"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	p := modelParamsFromInterop(i)
	var errs []error

	if err := checkChainIDCoverage(pred, "1", "chains", i.chains, p.ChainIDs); err != nil {
		errs = append(errs, err)
	}
	if err := checkChainIDCoverage(pred, "2", "logsDBs", i.logsDBs, p.ChainIDs); err != nil {
		errs = append(errs, err)
	}

	ids := sortedLogsDBChainIDs(i)
	for a := 0; a < len(ids); a++ {
		for b := a + 1; b < len(ids); b++ {
			if sameLogsDB(i.logsDBs[ids[a]], i.logsDBs[ids[b]]) {
				errs = append(errs, violation(pred, "3",
					"logsDBs for chains %s and %s are the same instance", ids[a], ids[b]))
			}
		}
	}

	// Conjuncts (4)-(8) all read the verifiedDB store.
	if i.verifiedDB == nil || i.verifiedDB.db == nil {
		errs = append(errs, violation(pred, "4", "VerifiedDB has no underlying store"))
		return errors.Join(errs...)
	}

	if err := CheckVerifiedDBValid(i.verifiedDB); err != nil {
		errs = append(errs, fmt.Errorf("%s conjunct (4): %w", pred, err))
	} else if db, dbErr := i.verifiedDB.allVerified(); dbErr != nil {
		errs = append(errs, violation(pred, "4", "enumerate verified bucket: %v", dbErr))
	} else {
		if _, initialized := i.verifiedDB.LastTimestamp(); initialized {
			if _, ok := db[p.ActivationTimestamp]; !ok {
				errs = append(errs, violation(pred, "5",
					"lastTimestamp is Some but ACTIVATION_TIMESTAMP %d not in db", p.ActivationTimestamp))
			}
		}
		for _, ts := range slices.Sorted(maps.Keys(db)) {
			if ts < p.ActivationTimestamp {
				errs = append(errs, violation(pred, "6",
					"committed timestamp %d below ACTIVATION_TIMESTAMP %d", ts, p.ActivationTimestamp))
			}
			if err := checkChainIDCoverage(pred, "7",
				fmt.Sprintf("db[%d].l2Heads", ts), db[ts].L2Heads, p.ChainIDs); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if pending, err := i.verifiedDB.GetPendingTransition(); err != nil {
		errs = append(errs, violation(pred, "8", "GetPendingTransition failed: %v", err))
	} else if pending != nil {
		if err := CheckValidPendingTransition(p, *pending); err != nil {
			errs = append(errs, fmt.Errorf("%s conjunct (8): stored pending transition invalid: %w", pred, err))
		}
	}

	return errors.Join(errs...)
}

// AssertInteropValid fails t when CheckInteropValid reports violations.
func AssertInteropValid(t dafnyT, i *Interop) {
	t.Helper()
	failOnViolation(t, CheckInteropValid(i))
}

// CheckDBsInSyncUpTo mirrors DBsInSyncUpTo(chainID, upperTS) in
// op-supernode/dafny-models/Interop.dfy: for every timestamp t in
// [ACTIVATION_TIMESTAMP, upper] the verified entry exists, covers chainID, and
// its head for chainID is sealed in the chain's logsDB with the same id
// (SealedBlockForVerifiedAtTimestamp). The scan is O(upper -
// ACTIVATION_TIMESTAMP) and intended for test workloads only (SPEC.md).
// Conjuncts, each quantified over t:
//
//	(0) i is non-nil, the verifiedDB store is reachable, chainID in
//	    logsDBs.Keys, and DB reads only fail with not-found sentinels
//	    (mapping requirement and the model's requires clause)
//	(1) verifiedDB.Has(t)
//	(2) chainID in verifiedDB.Get(t).l2Heads
//	(3) SealedBlockForVerifiedAtTimestamp(chainID, t).Some?, i.e.
//	    FindSealedBlock(l2Heads[chainID].number) finds a seal
//	(4) the found seal's id == verifiedDB.Get(t).l2Heads[chainID]
func CheckDBsInSyncUpTo(i *Interop, chainID eth.ChainID, upper uint64) error {
	const pred = "Interop.dfy DBsInSyncUpTo"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if i.verifiedDB == nil || i.verifiedDB.db == nil {
		return violation(pred, "0", "VerifiedDB has no underlying store")
	}
	db, ok := i.logsDBs[chainID]
	if !ok || db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}
	p := modelParamsFromInterop(i)

	var errs []error
	for t := p.ActivationTimestamp; t <= upper; t++ {
		result, err := i.verifiedDB.Get(t)
		switch {
		case errors.Is(err, ErrNotFound):
			errs = append(errs, violation(pred, "1", "verifiedDB.Has(%d) is false", t))
		case err != nil:
			errs = append(errs, violation(pred, "0", "verifiedDB.Get(%d) failed: %v", t, err))
		default:
			head, inHeads := result.L2Heads[chainID]
			if !inHeads {
				errs = append(errs, violation(pred, "2",
					"chain %s not in verifiedDB.Get(%d).l2Heads", chainID, t))
				break
			}
			seal, found, ferr := findSealedOption(db, head.Number)
			if ferr != nil {
				errs = append(errs, violation(pred, "0",
					"chain %s FindSealedBlock(%d) failed: %v", chainID, head.Number, ferr))
			} else if !found {
				errs = append(errs, violation(pred, "3",
					"chain %s: no sealed block %d for verified head at ts %d", chainID, head.Number, t))
			} else if seal.ID() != head {
				errs = append(errs, violation(pred, "4",
					"chain %s: sealed block %s != verified head %s at ts %d", chainID, seal.ID(), head, t))
			}
		}
		if t == upper { // guards against uint64 wrap of t++
			break
		}
	}
	return errors.Join(errs...)
}

// AssertDBsInSyncUpTo fails t when CheckDBsInSyncUpTo reports violations.
func AssertDBsInSyncUpTo(t dafnyT, i *Interop, chainID eth.ChainID, upper uint64) {
	t.Helper()
	failOnViolation(t, CheckDBsInSyncUpTo(i, chainID, upper))
}

// checkDBsInSync is the body of DBsInSync(chainID); callers have already
// established the model's `requires verifiedDB.Valid()`.
func checkDBsInSync(i *Interop, chainID eth.ChainID) error {
	const pred = "Interop.dfy DBsInSync"
	db, ok := i.logsDBs[chainID]
	if !ok || db == nil {
		return violation(pred, "0", "chain %s has no logsDB", chainID)
	}

	lastTS, initialized := i.verifiedDB.LastTimestamp()
	if !initialized {
		if latest, has := db.LatestSealedBlock(); has {
			return violation(pred, "N1",
				"verifiedDB is empty but chain %s LatestSealedBlock is Some(%s)", chainID, latest)
		}
		return nil
	}

	result, err := i.verifiedDB.Get(lastTS)
	if err != nil {
		return violation(pred, "0", "verifiedDB.Get(%d) failed: %v", lastTS, err)
	}

	var errs []error
	if head, inHeads := result.L2Heads[chainID]; !inHeads {
		errs = append(errs, violation(pred, "S1",
			"chain %s not in verifiedDB.Get(%d).l2Heads", chainID, lastTS))
	} else if latest, has := db.LatestSealedBlock(); !has {
		errs = append(errs, violation(pred, "S2",
			"chain %s LatestSealedBlock is None but last verified head is %s", chainID, head))
	} else if latest != head {
		errs = append(errs, violation(pred, "S2",
			"chain %s LatestSealedBlock %s != last verified head %s at ts %d", chainID, latest, head, lastTS))
	}
	if err := CheckDBsInSyncUpTo(i, chainID, lastTS); err != nil {
		errs = append(errs, fmt.Errorf("%s conjunct (S3): %w", pred, err))
	}
	return errors.Join(errs...)
}

// CheckDBsInSync mirrors DBsInSync(chainID) in
// op-supernode/dafny-models/Interop.dfy. The model's `requires
// verifiedDB.Valid()` is checked first via CheckVerifiedDBValid; on failure
// the predicate body is skipped (composite checkers short-circuit dependent
// conjuncts, SPEC.md). Conjuncts, matching on verifiedDB.LastTimestamp():
//
//	(0) i is non-nil and chainID in logsDBs.Keys (mapping requirement and the
//	    model's requires clause)
//	None case:
//	  (N1) logsDBs[chainID].LatestSealedBlock() == None
//	Some(ts) case:
//	  (S1) chainID in verifiedDB.Get(ts).l2Heads
//	  (S2) logsDBs[chainID].LatestSealedBlock() == Some(l2Heads[chainID])
//	  (S3) DBsInSyncUpTo(chainID, ts)
func CheckDBsInSync(i *Interop, chainID eth.ChainID) error {
	const pred = "Interop.dfy DBsInSync"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckVerifiedDBValid(i.verifiedDB); err != nil {
		return fmt.Errorf("%s requires verifiedDB.Valid(): %w", pred, err)
	}
	return checkDBsInSync(i, chainID)
}

// AssertDBsInSync fails t when CheckDBsInSync reports violations.
func AssertDBsInSync(t dafnyT, i *Interop, chainID eth.ChainID) {
	t.Helper()
	failOnViolation(t, CheckDBsInSync(i, chainID))
}

// CheckAllDBsInSyncUpTo mirrors AllDBsInSyncUpTo(upper) in
// op-supernode/dafny-models/Interop.dfy: DBsInSyncUpTo(k, upper) for every k
// in logsDBs.Keys. Violations carry the failing chain's ID; conjunct labels
// are those of CheckDBsInSyncUpTo.
func CheckAllDBsInSyncUpTo(i *Interop, upper uint64) error {
	const pred = "Interop.dfy AllDBsInSyncUpTo"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	var errs []error
	for _, k := range sortedLogsDBChainIDs(i) {
		if err := CheckDBsInSyncUpTo(i, k, upper); err != nil {
			errs = append(errs, fmt.Errorf("%s: chain %s: %w", pred, k, err))
		}
	}
	return errors.Join(errs...)
}

// AssertAllDBsInSyncUpTo fails t when CheckAllDBsInSyncUpTo reports
// violations.
func AssertAllDBsInSyncUpTo(t dafnyT, i *Interop, upper uint64) {
	t.Helper()
	failOnViolation(t, CheckAllDBsInSyncUpTo(i, upper))
}

// CheckAllDBsInSync mirrors AllDBsInSync() in
// op-supernode/dafny-models/Interop.dfy: DBsInSync(k) for every k in
// logsDBs.Keys. The model's `requires verifiedDB.Valid()` is checked once via
// CheckVerifiedDBValid; on failure the per-chain bodies are skipped.
// Violations carry the failing chain's ID; conjunct labels are those of
// CheckDBsInSync.
func CheckAllDBsInSync(i *Interop) error {
	const pred = "Interop.dfy AllDBsInSync"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckVerifiedDBValid(i.verifiedDB); err != nil {
		return fmt.Errorf("%s requires verifiedDB.Valid(): %w", pred, err)
	}
	var errs []error
	for _, k := range sortedLogsDBChainIDs(i) {
		if err := checkDBsInSync(i, k); err != nil {
			errs = append(errs, fmt.Errorf("%s: chain %s: %w", pred, k, err))
		}
	}
	return errors.Join(errs...)
}

// AssertAllDBsInSync fails t when CheckAllDBsInSync reports violations.
func AssertAllDBsInSync(t dafnyT, i *Interop) {
	t.Helper()
	failOnViolation(t, CheckAllDBsInSync(i))
}
