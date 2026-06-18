package interop

// This file is part of the Dafny model checkers (dafny_check_*.go files):
// test/debug-only helpers that check predicates from op-supernode/dafny-models/
// against the real Go types. Production code paths must not call them.

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// findSealedOption maps LogsDB.FindSealedBlock's (BlockSeal, error) result to
// the model's Option<BlockSeal>: ErrFuture/ErrSkipped mean None, any other
// error breaks the model mapping.
func findSealedOption(db LogsDB, number uint64) (suptypes.BlockSeal, bool, error) {
	seal, err := db.FindSealedBlock(number)
	switch {
	case err == nil:
		return seal, true, nil
	case errors.Is(err, suptypes.ErrFuture), errors.Is(err, suptypes.ErrSkipped):
		return suptypes.BlockSeal{}, false, nil
	default:
		return suptypes.BlockSeal{}, false, err
	}
}

// CheckLogsDBSealsWellFormed mirrors the function axioms of FirstSealedBlock,
// LatestSealedBlock, and FindSealedBlock in
// op-supernode/dafny-models/LogsDB.dfy. Option mapping: LatestSealedBlock
// ok==false ↔ None; FirstSealedBlock/FindSealedBlock ErrFuture/ErrSkipped ↔
// None; model BlockID ↔ eth.BlockID{Hash: seal.Hash, Number: seal.Number}.
// Conjuncts:
//
//	(0) db is non-nil and every FirstSealedBlock/FindSealedBlock error is a
//	    not-found sentinel (mapping requirement; the remaining conjuncts
//	    assume it)
//	(E1) FirstSealedBlock None <==> LatestSealedBlock None (each axiom's None
//	    case forces FindSealedBlock to None everywhere, contradicting the
//	    other's Some case)
//	(B1) first.number <= latest.number (FindSealedBlock(first.number).Some?
//	    plus LatestSealedBlock's `forall number > latest.number ==> None`)
//	(F1) FindSealedBlock(first.number).Some? && value.id == first
//	    (FirstSealedBlock axiom, Some case)
//	(L1) FindSealedBlock(latest.number).Some? && value.id == latest
//	    (LatestSealedBlock axiom, Some case)
//	(N1) forall n in [first.number, latest.number]:
//	    FindSealedBlock(n).Some? ==> value.id.number == n
//	    (first FindSealedBlock axiom)
//	(T1) found seals' timestamps strictly increase with block number
//	    (second FindSealedBlock axiom, consecutive found pairs cover all
//	    pairs by transitivity of <)
//
// The model does not exclude not-found gaps strictly inside the sealed range,
// so the checker tolerates them. The scan is O(latest.number - first.number)
// FindSealedBlock calls and is intended for test workloads only.
func CheckLogsDBSealsWellFormed(db LogsDB) error {
	const pred = "LogsDB.dfy sealed-block axioms"
	if db == nil {
		return violation(pred, "0", "LogsDB is nil")
	}
	latest, hasLatest := db.LatestSealedBlock()
	first, err := db.FirstSealedBlock()
	hasFirst := err == nil
	if err != nil && !errors.Is(err, suptypes.ErrFuture) && !errors.Is(err, suptypes.ErrSkipped) {
		return violation(pred, "0", "FirstSealedBlock failed: %v", err)
	}
	if hasFirst != hasLatest {
		return violation(pred, "E1",
			"FirstSealedBlock is None: %t but LatestSealedBlock is None: %t", !hasFirst, !hasLatest)
	}
	if !hasLatest {
		return nil
	}

	var errs []error
	if first.Number > latest.Number {
		errs = append(errs, violation(pred, "B1",
			"first sealed number %d > latest sealed number %d", first.Number, latest.Number))
	}

	firstID := eth.BlockID{Hash: first.Hash, Number: first.Number}
	if seal, found, ferr := findSealedOption(db, first.Number); ferr != nil {
		errs = append(errs, violation(pred, "0", "FindSealedBlock(%d) failed: %v", first.Number, ferr))
	} else if !found || seal.ID() != firstID {
		errs = append(errs, violation(pred, "F1",
			"FindSealedBlock(%d) = (%s, found=%t) does not match FirstSealedBlock %s",
			first.Number, seal.ID(), found, firstID))
	}
	if seal, found, ferr := findSealedOption(db, latest.Number); ferr != nil {
		errs = append(errs, violation(pred, "0", "FindSealedBlock(%d) failed: %v", latest.Number, ferr))
	} else if !found || seal.ID() != latest {
		errs = append(errs, violation(pred, "L1",
			"FindSealedBlock(%d) = (%s, found=%t) does not match LatestSealedBlock %s",
			latest.Number, seal.ID(), found, latest))
	}

	var prev suptypes.BlockSeal
	prevFound := false
	for n := first.Number; n <= latest.Number; n++ {
		seal, found, ferr := findSealedOption(db, n)
		if ferr != nil {
			errs = append(errs, violation(pred, "0", "FindSealedBlock(%d) failed: %v", n, ferr))
		} else if found {
			if seal.Number != n {
				errs = append(errs, violation(pred, "N1",
					"FindSealedBlock(%d) returned seal with number %d", n, seal.Number))
			}
			if prevFound && seal.Timestamp <= prev.Timestamp {
				errs = append(errs, violation(pred, "T1",
					"timestamp %d at block %d does not exceed timestamp %d at block %d",
					seal.Timestamp, n, prev.Timestamp, prev.Number))
			}
			prev, prevFound = seal, true
		}
		if n == latest.Number { // guards against uint64 wrap of n++
			break
		}
	}

	return errors.Join(errs...)
}

// AssertLogsDBSealsWellFormed fails t when CheckLogsDBSealsWellFormed reports
// violations.
func AssertLogsDBSealsWellFormed(t dafnyT, db LogsDB) {
	t.Helper()
	failOnViolation(t, CheckLogsDBSealsWellFormed(db))
}

// CheckFetchReceiptsPost mirrors the FetchReceipts postcondition
// `ensures info.id == blockID` in
// op-supernode/dafny-models/ChainContainer.dfy, with model info.id mapped to
// eth.BlockID{Hash: info.Hash(), Number: info.NumberU64()}. Conjuncts:
//
//	(0) info is non-nil (mapping requirement)
//	(1) info.id == blockID
func CheckFetchReceiptsPost(blockID eth.BlockID, info eth.BlockInfo) error {
	const pred = "ChainContainer.dfy FetchReceipts ensures"
	if info == nil {
		return violation(pred, "0", "info is nil")
	}
	got := eth.BlockID{Hash: info.Hash(), Number: info.NumberU64()}
	if got != blockID {
		return violation(pred, "1", "info.id %s != blockID %s", got, blockID)
	}
	return nil
}

// AssertFetchReceiptsPost fails t when CheckFetchReceiptsPost reports
// violations.
func AssertFetchReceiptsPost(t dafnyT, blockID eth.BlockID, info eth.BlockInfo) {
	t.Helper()
	failOnViolation(t, CheckFetchReceiptsPost(blockID, info))
}
