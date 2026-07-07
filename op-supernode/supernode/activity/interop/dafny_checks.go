package interop

// This file holds the Dafny model checkers: test/debug-only helpers that check
// predicates from op-supernode/dafny-models/ against the real Go types.
// Production code paths must not call them.

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

	bolt "go.etcd.io/bbolt"
)

// === types ===

// dafnyT is the minimal testing surface needed by the Assert* wrappers;
// *testing.T satisfies it.
type dafnyT interface {
	Helper()
	Errorf(format string, args ...any)
	FailNow()
}

// failOnViolation fails t with the full violation list when err is non-nil.
func failOnViolation(t dafnyT, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("dafny model violation:\n%v", err)
		t.FailNow()
	}
}

// violation reports one violated conjunct of a Dafny predicate.
// predicate cites the Dafny file and predicate name, conjunct the label used
// in the corresponding checker's doc comment.
func violation(predicate, conjunct, format string, args ...any) error {
	return fmt.Errorf("%s conjunct (%s): %s", predicate, conjunct, fmt.Sprintf(format, args...))
}

// ModelParams instantiates the ghost constants of Types.dfy.
type ModelParams struct {
	// ActivationTimestamp maps Types.dfy ACTIVATION_TIMESTAMP. Per the SPEC.md
	// model-to-Go mapping it carries the Go first-verifiable timestamp
	// (Interop.firstVerifiableTimestamp), not necessarily the protocol
	// activation timestamp.
	ActivationTimestamp uint64
	// ChainIDs maps Types.dfy CHAIN_IDS: the set of chains the instance runs.
	ChainIDs map[eth.ChainID]struct{}
}

// modelParamsFromInterop derives the model ghost constants from a live
// instance: ChainIDs from the key set of i.chains, ActivationTimestamp from
// i.firstVerifiableTimestamp(), falling back to the protocol activation
// timestamp before initialization completes.
func modelParamsFromInterop(i *Interop) ModelParams {
	p := ModelParams{
		ActivationTimestamp: i.activationTimestamp,
		ChainIDs:            make(map[eth.ChainID]struct{}, len(i.chains)),
	}
	for id := range i.chains {
		p.ChainIDs[id] = struct{}{}
	}
	if first, err := i.firstVerifiableTimestamp(); err == nil {
		p.ActivationTimestamp = first
	}
	return p
}

// checkChainIDCoverage reports a violation unless the key set of m equals
// CHAIN_IDS (model expression `<field>.Keys == CHAIN_IDS`).
func checkChainIDCoverage[V any](predicate, conjunct, field string, m map[eth.ChainID]V, chainIDs map[eth.ChainID]struct{}) error {
	var missing, extra []string
	for id := range chainIDs {
		if _, ok := m[id]; !ok {
			missing = append(missing, id.String())
		}
	}
	for id := range m {
		if _, ok := chainIDs[id]; !ok {
			extra = append(extra, id.String())
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return nil
	}
	slices.Sort(missing)
	slices.Sort(extra)
	return violation(predicate, conjunct, "%s.Keys != CHAIN_IDS: missing %v, extra %v", field, missing, extra)
}

// decisionInModel reports whether d maps to a Types.dfy Decision constructor
// (Wait/Advance/Invalidate/Rewind).
func decisionInModel(d Decision) bool {
	switch d {
	case DecisionWait, DecisionAdvance, DecisionInvalidate, DecisionRewind:
		return true
	default:
		return false
	}
}

// CheckValidRewindPlan mirrors ValidRewindPlan(plan) in
// op-supernode/dafny-models/Types.dfy. The match on plan.resetAllChainsTo maps
// to plan.ResetAllChainsTo == nil (None) vs non-nil (Some(ts)). Conjuncts:
//   - None case:
//     (N1) plan.rewindAtOrAfter <= ACTIVATION_TIMESTAMP
//   - Some(ts) case:
//     (S1) ACTIVATION_TIMESTAMP < plan.rewindAtOrAfter
//     (S2) ts == plan.rewindAtOrAfter - 1; evaluated only when
//     plan.rewindAtOrAfter > 0, preserving the model's nat-subtraction guard
//     (S1 already fails whenever the guard does)
//     (S3) plan.targetHeads.Keys == CHAIN_IDS
func CheckValidRewindPlan(p ModelParams, plan RewindPlan) error {
	const pred = "Types.dfy ValidRewindPlan"
	var errs []error
	if plan.ResetAllChainsTo == nil {
		if !(plan.RewindAtOrAfter <= p.ActivationTimestamp) {
			errs = append(errs, violation(pred, "N1",
				"resetAllChainsTo is None but rewindAtOrAfter %d > ACTIVATION_TIMESTAMP %d",
				plan.RewindAtOrAfter, p.ActivationTimestamp))
		}
	} else {
		ts := *plan.ResetAllChainsTo
		if !(p.ActivationTimestamp < plan.RewindAtOrAfter) {
			errs = append(errs, violation(pred, "S1",
				"resetAllChainsTo is Some(%d) but ACTIVATION_TIMESTAMP %d is not < rewindAtOrAfter %d",
				ts, p.ActivationTimestamp, plan.RewindAtOrAfter))
		}
		if plan.RewindAtOrAfter > 0 && ts != plan.RewindAtOrAfter-1 {
			errs = append(errs, violation(pred, "S2",
				"resetAllChainsTo %d != rewindAtOrAfter - 1 == %d",
				ts, plan.RewindAtOrAfter-1))
		}
		if err := checkChainIDCoverage(pred, "S3", "targetHeads", plan.TargetHeads, p.ChainIDs); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// AssertValidRewindPlan fails t when CheckValidRewindPlan reports violations.
func AssertValidRewindPlan(t dafnyT, p ModelParams, plan RewindPlan) {
	t.Helper()
	failOnViolation(t, CheckValidRewindPlan(p, plan))
}

// CheckValidPendingTransition mirrors ValidPendingTransition(pending) in
// op-supernode/dafny-models/Types.dfy. Option fields map to Go nil pointers
// (None) vs non-nil (Some). Conjuncts:
//
//	(0) pending.decision is a Types.dfy Decision constructor (mapping
//	    requirement; the remaining conjuncts assume it)
//	(1) pending.decision != Wait
//	(2) pending.decision == Rewind ==> pending.rewind.Some?
//	(3) pending.decision == Rewind ==> ValidRewindPlan(pending.rewind.value);
//	    skipped when (2) already failed
//	(4) pending.decision == Advance ==> pending.result.Some?
//	(5) pending.decision == Invalidate ==> pending.result.Some?
//	(6) pending.result.Some? ==> pending.result.value.l2Heads.Keys == CHAIN_IDS
func CheckValidPendingTransition(p ModelParams, pending PendingTransition) error {
	const pred = "Types.dfy ValidPendingTransition"
	var errs []error
	if !decisionInModel(pending.Decision) {
		return violation(pred, "0", "decision %s has no Types.dfy Decision constructor", pending.Decision)
	}
	if pending.Decision == DecisionWait {
		errs = append(errs, violation(pred, "1", "decision is Wait"))
	}
	if pending.Decision == DecisionRewind {
		if pending.Rewind == nil {
			errs = append(errs, violation(pred, "2", "decision is Rewind but rewind is None"))
		} else if err := CheckValidRewindPlan(p, *pending.Rewind); err != nil {
			errs = append(errs, fmt.Errorf("%s conjunct (3): rewind plan invalid: %w", pred, err))
		}
	}
	if pending.Decision == DecisionAdvance && pending.Result == nil {
		errs = append(errs, violation(pred, "4", "decision is Advance but result is None"))
	}
	if pending.Decision == DecisionInvalidate && pending.Result == nil {
		errs = append(errs, violation(pred, "5", "decision is Invalidate but result is None"))
	}
	if pending.Result != nil {
		if err := checkChainIDCoverage(pred, "6", "result.l2Heads", pending.Result.L2Heads, p.ChainIDs); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// AssertValidPendingTransition fails t when CheckValidPendingTransition
// reports violations.
func AssertValidPendingTransition(t dafnyT, p ModelParams, pending PendingTransition) {
	t.Helper()
	failOnViolation(t, CheckValidPendingTransition(p, pending))
}

// CheckValidStepOutput mirrors ValidStepOutput(output, obs) in
// op-supernode/dafny-models/Types.dfy, under the StepOutput mapping:
// WaitOutput ↔ DecisionWait, RewindOutput ↔ DecisionRewind (Result ignored),
// AdvanceOutput(result)/InvalidateOutput(result) ↔ DecisionAdvance/
// DecisionInvalidate with output.Result as the constructor argument.
// Conjuncts per constructor:
//
//	(0) output.Decision is a Types.dfy Decision constructor (mapping
//	    requirement)
//	WaitOutput: true (nothing to check)
//	RewindOutput:
//	  (R1) obs.lastVerifiedTS.Some?
//	AdvanceOutput(result) / InvalidateOutput(result):
//	  (1) result.timestamp == obs.nextTimestamp
//	  (2) result.l2Heads == obs.blocksAtTS
//	  (3) result.l2Heads.Keys == CHAIN_IDS
func CheckValidStepOutput(p ModelParams, output StepOutput, obs RoundObservation) error {
	const pred = "Types.dfy ValidStepOutput"
	switch output.Decision {
	case DecisionWait:
		return nil
	case DecisionRewind:
		if obs.LastVerifiedTS == nil {
			return violation(pred, "R1", "output is RewindOutput but obs.lastVerifiedTS is None")
		}
		return nil
	case DecisionAdvance, DecisionInvalidate:
		var errs []error
		if output.Result.Timestamp != obs.NextTimestamp {
			errs = append(errs, violation(pred, "1",
				"result.timestamp %d != obs.nextTimestamp %d",
				output.Result.Timestamp, obs.NextTimestamp))
		}
		if !maps.Equal(output.Result.L2Heads, obs.BlocksAtTS) {
			errs = append(errs, violation(pred, "2",
				"result.l2Heads %v != obs.blocksAtTS %v",
				output.Result.L2Heads, obs.BlocksAtTS))
		}
		if err := checkChainIDCoverage(pred, "3", "result.l2Heads", output.Result.L2Heads, p.ChainIDs); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	default:
		return violation(pred, "0", "decision %s has no Types.dfy StepOutput constructor", output.Decision)
	}
}

// AssertValidStepOutput fails t when CheckValidStepOutput reports violations.
func AssertValidStepOutput(t dafnyT, p ModelParams, output StepOutput, obs RoundObservation) {
	t.Helper()
	failOnViolation(t, CheckValidStepOutput(p, output, obs))
}

// CheckValidRoundObservation mirrors ValidRoundObservation(obs) in
// op-supernode/dafny-models/Types.dfy, under the field mapping: model
// l1Consistent ↔ !obs.L1NeedsRewind (so model !l1Consistent ↔
// obs.L1NeedsRewind), model lastVerifiedTS ↔ obs.LastVerifiedTS. Conjuncts:
//
//	(1) !l1Consistent ==> obs.lastVerifiedTS.Some?
//	(2) obs.chainsReady ==> obs.blocksAtTS.Keys == CHAIN_IDS
//
// Paused is omitted from the model; a paused observeRound returns before
// populating ChainsReady, BlocksAtTS, and L1NeedsRewind, so both conjuncts are
// skipped when obs.Paused is true (SPEC.md model-to-Go mapping).
func CheckValidRoundObservation(p ModelParams, obs RoundObservation) error {
	const pred = "Types.dfy ValidRoundObservation"
	if obs.Paused {
		return nil
	}
	var errs []error
	if obs.L1NeedsRewind && obs.LastVerifiedTS == nil {
		errs = append(errs, violation(pred, "1",
			"l1Consistent is false (L1NeedsRewind) but obs.lastVerifiedTS is None"))
	}
	if obs.ChainsReady {
		if err := checkChainIDCoverage(pred, "2", "blocksAtTS", obs.BlocksAtTS, p.ChainIDs); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// AssertValidRoundObservation fails t when CheckValidRoundObservation reports
// violations.
func AssertValidRoundObservation(t dafnyT, p ModelParams, obs RoundObservation) {
	t.Helper()
	failOnViolation(t, CheckValidRoundObservation(p, obs))
}

// === verifieddb ===

// allVerified snapshots the verified bucket as the model state
// `db: map<nat, VerifiedResult>` of VerifiedDB.dfy, via a read-only bbolt
// View. An entry that does not map to the model (key not an 8-byte
// big-endian timestamp, value not a JSON VerifiedResult) is an error.
func (v *VerifiedDB) allVerified() (map[uint64]VerifiedResult, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	out := make(map[uint64]VerifiedResult)
	err := v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, val []byte) error {
			if len(k) != u64Len {
				return fmt.Errorf("key %x is not an %d-byte big-endian timestamp", k, u64Len)
			}
			var result VerifiedResult
			if err := json.Unmarshal(val, &result); err != nil {
				return fmt.Errorf("value at key %d is not a VerifiedResult: %w", binary.BigEndian.Uint64(k), err)
			}
			out[binary.BigEndian.Uint64(k)] = result
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// checkSequential mirrors Sequential(m) in op-supernode/dafny-models/Utils.dfy
// over the key set of a nat-keyed map: nil iff keys is empty or every key
// strictly below the maximum has an immediate successor. keys must be the
// distinct keys of a map, in any order.
func checkSequential(keys []uint64) error {
	if len(keys) == 0 {
		return nil
	}
	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1]+1 {
			return fmt.Errorf("gap at %d (next committed key is %d)", sorted[i-1]+1, sorted[i])
		}
	}
	return nil
}

// CheckVerifiedDBValid mirrors the class invariant VerifiedDB.Valid() in
// op-supernode/dafny-models/VerifiedDB.dfy, with the model `db` enumerated
// from the bbolt verified bucket (allVerified) rather than the cached
// timestamp fields, and `lastTimestamp: Option<nat>` mapped to the Go
// (lastTimestamp, initialized) pair. Conjuncts:
//
//	(0) every stored entry maps to the model db: map<nat, VerifiedResult>
//	    and the store is reachable (mapping requirement; the remaining
//	    conjuncts assume it)
//	(1) Sequential(db): committed timestamps form a contiguous range
//	(2) forall ts in db: db[ts].timestamp == ts
//	(3) lastTimestamp == (if |db| == 0 then None else Some(MaxKey(db)))
//	(4) forall t1 <= t2 in db, cid in both l2Heads:
//	    db[t1].l2Heads[cid].number <= db[t2].l2Heads[cid].number
func CheckVerifiedDBValid(v *VerifiedDB) error {
	const pred = "VerifiedDB.dfy Valid()"
	if v == nil || v.db == nil {
		return violation(pred, "0", "VerifiedDB has no underlying store")
	}
	db, err := v.allVerified()
	if err != nil {
		return violation(pred, "0", "verified bucket does not map to db: map<nat, VerifiedResult>: %v", err)
	}
	last, initialized := v.LastTimestamp()

	var errs []error
	keys := slices.Sorted(maps.Keys(db))

	if err := checkSequential(keys); err != nil {
		errs = append(errs, violation(pred, "1", "committed timestamps not sequential: %v", err))
	}

	for _, ts := range keys {
		if db[ts].Timestamp != ts {
			errs = append(errs, violation(pred, "2",
				"entry at key %d has timestamp field %d", ts, db[ts].Timestamp))
		}
	}

	switch {
	case len(db) == 0 && initialized:
		errs = append(errs, violation(pred, "3",
			"db is empty but lastTimestamp is Some(%d)", last))
	case len(db) > 0 && !initialized:
		errs = append(errs, violation(pred, "3",
			"db has %d entries but lastTimestamp is None", len(db)))
	case len(db) > 0 && last != keys[len(keys)-1]:
		errs = append(errs, violation(pred, "3",
			"lastTimestamp %d != MaxKey(db) %d", last, keys[len(keys)-1]))
	}

	// Consecutive-occurrence comparisons over ascending timestamps cover all
	// t1 <= t2 pairs of conjunct (4) by transitivity of <=.
	type seen struct {
		number uint64
		ts     uint64
	}
	lastSeen := make(map[eth.ChainID]seen)
	for _, ts := range keys {
		heads := db[ts].L2Heads
		for _, cid := range slices.SortedFunc(maps.Keys(heads), eth.ChainID.Cmp) {
			if prev, ok := lastSeen[cid]; ok && heads[cid].Number < prev.number {
				errs = append(errs, violation(pred, "4",
					"chain %s l2Heads number decreases from %d at ts %d to %d at ts %d",
					cid, prev.number, prev.ts, heads[cid].Number, ts))
			}
			lastSeen[cid] = seen{number: heads[cid].Number, ts: ts}
		}
	}

	return errors.Join(errs...)
}

// AssertVerifiedDBValid fails t when CheckVerifiedDBValid reports violations.
func AssertVerifiedDBValid(t dafnyT, v *VerifiedDB) {
	t.Helper()
	failOnViolation(t, CheckVerifiedDBValid(v))
}

// === logsdb ===

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

// === round ===

// modelNextTimestamp mirrors NextTimestamp() in Interop.dfy: the successor of
// the last committed timestamp, or ACTIVATION_TIMESTAMP when the verifiedDB is
// empty.
func modelNextTimestamp(i *Interop) uint64 {
	if lastTS, initialized := i.verifiedDB.LastTimestamp(); initialized {
		return lastTS + 1
	}
	return modelParamsFromInterop(i).ActivationTimestamp
}

// checkVerifiedHasPrev checks the model expression `ts - 1 in verifiedDB.db`
// (Interop.dfy); pred and label name the calling conjunct in violation
// reports. ts must be positive (the model guards with ACTIVATION_TIMESTAMP <
// ts before subtracting).
func checkVerifiedHasPrev(i *Interop, pred, label string, ts uint64) error {
	if _, err := i.verifiedDB.Get(ts - 1); errors.Is(err, ErrNotFound) {
		return violation(pred, label, "lastVerifiedTS - 1 == %d not in verifiedDB", ts-1)
	} else if err != nil {
		return violation(pred, "0", "verifiedDB.Get(%d) failed: %v", ts-1, err)
	}
	return nil
}

// checkSealedMatchesVerifiedHeads checks, for every chain k in logsDBs.Keys,
// that SealedBlockForVerifiedAtTimestamp(k, ts) (Interop.dfy) is Some
// (reported as conjunct someLabel) and that the seal's id equals
// verifiedDB.Get(ts).l2Heads[k] (reported as conjunct idLabel); pred names the
// calling predicate in violation reports.
func checkSealedMatchesVerifiedHeads(i *Interop, pred, someLabel, idLabel string, ts uint64) error {
	result, err := i.verifiedDB.Get(ts)
	if err != nil {
		return violation(pred, "0", "verifiedDB.Get(%d) failed: %v", ts, err)
	}
	var errs []error
	for _, k := range sortedLogsDBChainIDs(i) {
		db := i.logsDBs[k]
		if db == nil {
			errs = append(errs, violation(pred, "0", "chain %s has no logsDB", k))
			continue
		}
		head, ok := result.L2Heads[k]
		if !ok {
			errs = append(errs, violation(pred, "0",
				"chain %s not in verifiedDB.Get(%d).l2Heads", k, ts))
			continue
		}
		seal, found, ferr := findSealedOption(db, head.Number)
		switch {
		case ferr != nil:
			errs = append(errs, violation(pred, "0",
				"chain %s FindSealedBlock(%d) failed: %v", k, head.Number, ferr))
		case !found:
			errs = append(errs, violation(pred, someLabel,
				"chain %s: no sealed block %d for verified head at ts %d", k, head.Number, ts))
		case seal.ID() != head:
			errs = append(errs, violation(pred, idLabel,
				"chain %s: sealed block %s != verified head %s at ts %d", k, seal.ID(), head, ts))
		}
	}
	return errors.Join(errs...)
}

// checkOutputConsistentWithVerified is the body of
// OutputConsistentWithVerified(output, obs); callers have already established
// the model's `requires verifiedDB.Valid() && ValidStepOutput(output, obs)`.
func checkOutputConsistentWithVerified(i *Interop, output StepOutput, obs RoundObservation) error {
	const pred = "Interop.dfy OutputConsistentWithVerified"
	switch output.Decision {
	case DecisionRewind:
		if obs.LastVerifiedTS == nil {
			return violation(pred, "0", "output is RewindOutput but obs.lastVerifiedTS is None")
		}
		if p := modelParamsFromInterop(i); p.ActivationTimestamp < *obs.LastVerifiedTS {
			return checkVerifiedHasPrev(i, pred, "R1", *obs.LastVerifiedTS)
		}
	case DecisionAdvance:
		if err := checkAdvancesVerifiedDB(i, output.Result.Timestamp, output.Result.L2Heads); err != nil {
			return fmt.Errorf("%s conjunct (A1): %w", pred, err)
		}
	case DecisionInvalidate:
		if next := modelNextTimestamp(i); output.Result.Timestamp != next {
			return violation(pred, "I1",
				"result.timestamp %d != NextTimestamp() %d", output.Result.Timestamp, next)
		}
	}
	return nil // WaitOutput case: true
}

// CheckOutputConsistentWithVerified mirrors OutputConsistentWithVerified(output, obs)
// in op-supernode/dafny-models/Interop.dfy, under the StepOutput and
// RoundObservation mappings of CheckValidStepOutput. The model's `requires
// verifiedDB.Valid()` and `requires ValidStepOutput(output, obs)` are checked
// first; on failure the predicate body is skipped. A paused round always
// yields WaitOutput (checkPreconditions), whose case is vacuous, so no Paused
// skip is needed. Conjuncts, matching on the output constructor:
//
//	(0) i is non-nil and DB reads succeed (mapping requirement)
//	WaitOutput: true (nothing to check)
//	RewindOutput:
//	  (R1) ACTIVATION_TIMESTAMP < obs.lastVerifiedTS.value ==>
//	       obs.lastVerifiedTS.value - 1 in verifiedDB.db
//	AdvanceOutput(result):
//	  (A1) AdvancesVerifiedDB(result.timestamp, result.l2Heads)
//	InvalidateOutput(result):
//	  (I1) result.timestamp == NextTimestamp()
func CheckOutputConsistentWithVerified(i *Interop, output StepOutput, obs RoundObservation) error {
	const pred = "Interop.dfy OutputConsistentWithVerified"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckVerifiedDBValid(i.verifiedDB.concrete()); err != nil {
		return fmt.Errorf("%s requires verifiedDB.Valid(): %w", pred, err)
	}
	if err := CheckValidStepOutput(modelParamsFromInterop(i), output, obs); err != nil {
		return fmt.Errorf("%s requires ValidStepOutput: %w", pred, err)
	}
	return checkOutputConsistentWithVerified(i, output, obs)
}

// AssertOutputConsistentWithVerified fails t when
// CheckOutputConsistentWithVerified reports violations.
func AssertOutputConsistentWithVerified(t dafnyT, i *Interop, output StepOutput, obs RoundObservation) {
	t.Helper()
	failOnViolation(t, CheckOutputConsistentWithVerified(i, output, obs))
}

// checkOutputConsistentWithLogs is the body of OutputConsistentWithLogs(output,
// obs); callers have already established the model's `requires Valid() &&
// ValidStepOutput(output, obs) && OutputConsistentWithVerified(output, obs)`.
func checkOutputConsistentWithLogs(i *Interop, output StepOutput, obs RoundObservation) error {
	const pred = "Interop.dfy OutputConsistentWithLogs"
	switch output.Decision {
	case DecisionRewind:
		if obs.LastVerifiedTS == nil {
			return violation(pred, "0", "output is RewindOutput but obs.lastVerifiedTS is None")
		}
		if p := modelParamsFromInterop(i); p.ActivationTimestamp < *obs.LastVerifiedTS {
			return checkSealedMatchesVerifiedHeads(i, pred, "R1", "R2", *obs.LastVerifiedTS-1)
		}
	case DecisionAdvance:
		if err := CheckAdvancesAllLogsDBs(i, output.Result.Timestamp, output.Result.L2Heads); err != nil {
			return fmt.Errorf("%s conjunct (A1): %w", pred, err)
		}
	}
	return nil // WaitOutput and InvalidateOutput cases: true
}

// CheckOutputConsistentWithLogs mirrors OutputConsistentWithLogs(output, obs)
// in op-supernode/dafny-models/Interop.dfy. The model's `requires Valid()`,
// `requires ValidStepOutput(output, obs)`, and `requires
// OutputConsistentWithVerified(output, obs)` are checked first; on failure the
// predicate body is skipped. A paused round always yields WaitOutput
// (checkPreconditions), whose case is vacuous, so no Paused skip is needed.
// Conjuncts, matching on the output constructor:
//
//	(0) i is non-nil and DB reads succeed (mapping requirement)
//	WaitOutput: true (nothing to check)
//	RewindOutput, forall k in logsDBs.Keys when
//	ACTIVATION_TIMESTAMP < obs.lastVerifiedTS.value:
//	  (R1) SealedBlockForVerifiedAtTimestamp(k, obs.lastVerifiedTS.value - 1).Some?
//	  (R2) the seal's id == verifiedDB.Get(obs.lastVerifiedTS.value - 1).l2Heads[k]
//	AdvanceOutput(result):
//	  (A1) AdvancesAllLogsDBs(result.timestamp, result.l2Heads)
//	InvalidateOutput(_): true (nothing to check)
func CheckOutputConsistentWithLogs(i *Interop, output StepOutput, obs RoundObservation) error {
	const pred = "Interop.dfy OutputConsistentWithLogs"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckInteropValid(i); err != nil {
		return fmt.Errorf("%s requires Valid(): %w", pred, err)
	}
	if err := CheckValidStepOutput(modelParamsFromInterop(i), output, obs); err != nil {
		return fmt.Errorf("%s requires ValidStepOutput: %w", pred, err)
	}
	if err := checkOutputConsistentWithVerified(i, output, obs); err != nil {
		return fmt.Errorf("%s requires OutputConsistentWithVerified: %w", pred, err)
	}
	return checkOutputConsistentWithLogs(i, output, obs)
}

// AssertOutputConsistentWithLogs fails t when CheckOutputConsistentWithLogs
// reports violations.
func AssertOutputConsistentWithLogs(t dafnyT, i *Interop, output StepOutput, obs RoundObservation) {
	t.Helper()
	failOnViolation(t, CheckOutputConsistentWithLogs(i, output, obs))
}

// checkObservationConsistentWithVerified is the body of
// ObservationConsistentWithVerified(obs); callers have already established the
// model's `requires verifiedDB.Valid() && ValidRoundObservation(obs)`.
func checkObservationConsistentWithVerified(i *Interop, obs RoundObservation) error {
	const pred = "Interop.dfy ObservationConsistentWithVerified"
	var errs []error

	lastTS, initialized := i.verifiedDB.LastTimestamp()
	switch {
	case initialized && obs.LastVerifiedTS == nil:
		errs = append(errs, violation(pred, "1",
			"obs.lastVerifiedTS is None but verifiedDB.lastTimestamp is Some(%d)", lastTS))
	case !initialized && obs.LastVerifiedTS != nil:
		errs = append(errs, violation(pred, "1",
			"obs.lastVerifiedTS is Some(%d) but verifiedDB.lastTimestamp is None", *obs.LastVerifiedTS))
	case initialized && *obs.LastVerifiedTS != lastTS:
		errs = append(errs, violation(pred, "1",
			"obs.lastVerifiedTS %d != verifiedDB.lastTimestamp %d", *obs.LastVerifiedTS, lastTS))
	}

	if next := modelNextTimestamp(i); obs.NextTimestamp != next {
		errs = append(errs, violation(pred, "2",
			"obs.nextTimestamp %d != NextTimestamp() %d", obs.NextTimestamp, next))
	}

	if obs.Paused { // conjuncts (3)-(4) are conditional on a non-paused round
		return errors.Join(errs...)
	}

	p := modelParamsFromInterop(i)
	if obs.L1NeedsRewind && obs.LastVerifiedTS != nil && p.ActivationTimestamp < *obs.LastVerifiedTS {
		if err := checkVerifiedHasPrev(i, pred, "3", *obs.LastVerifiedTS); err != nil {
			errs = append(errs, err)
		}
	}
	if obs.ChainsReady && obs.L1Consistent && !obs.L1NeedsRewind {
		if err := checkAdvancesVerifiedDB(i, obs.NextTimestamp, obs.BlocksAtTS); err != nil {
			errs = append(errs, fmt.Errorf("%s conjunct (4): %w", pred, err))
		}
	}
	return errors.Join(errs...)
}

// CheckObservationConsistentWithVerified mirrors
// ObservationConsistentWithVerified(obs) in
// op-supernode/dafny-models/Interop.dfy, under the RoundObservation field
// mapping: model l1Consistent ↔ !obs.L1NeedsRewind, model l2sConsistent ↔
// obs.L1Consistent (SPEC.md model-to-Go mapping). The model's `requires
// verifiedDB.Valid()` and `requires ValidRoundObservation(obs)` are checked
// first; on failure the predicate body is skipped. Go Paused is outside the
// model: a paused observeRound populates only LastVerifiedTS and
// NextTimestamp, so conjuncts (3)-(4), conditional on round flags set after
// the pause check, are skipped when obs.Paused is true; conjuncts (1)-(2) are
// always checked. Conjuncts:
//
//	(0) i is non-nil and DB reads succeed (mapping requirement)
//	(1) obs.lastVerifiedTS == verifiedDB.lastTimestamp
//	(2) obs.nextTimestamp == NextTimestamp()
//	(3) !l1Consistent && ACTIVATION_TIMESTAMP < obs.lastVerifiedTS.value ==>
//	    obs.lastVerifiedTS.value - 1 in verifiedDB.db
//	(4) obs.chainsReady && l2sConsistent && l1Consistent ==>
//	    AdvancesVerifiedDB(obs.nextTimestamp, obs.blocksAtTS)
func CheckObservationConsistentWithVerified(i *Interop, obs RoundObservation) error {
	const pred = "Interop.dfy ObservationConsistentWithVerified"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckVerifiedDBValid(i.verifiedDB.concrete()); err != nil {
		return fmt.Errorf("%s requires verifiedDB.Valid(): %w", pred, err)
	}
	if err := CheckValidRoundObservation(modelParamsFromInterop(i), obs); err != nil {
		return fmt.Errorf("%s requires ValidRoundObservation: %w", pred, err)
	}
	return checkObservationConsistentWithVerified(i, obs)
}

// AssertObservationConsistentWithVerified fails t when
// CheckObservationConsistentWithVerified reports violations.
func AssertObservationConsistentWithVerified(t dafnyT, i *Interop, obs RoundObservation) {
	t.Helper()
	failOnViolation(t, CheckObservationConsistentWithVerified(i, obs))
}

// checkObservationConsistentWithLogs is the body of
// ObservationConsistentWithLogs(obs); callers have already established the
// model's `requires Valid() && ValidRoundObservation(obs) &&
// ObservationConsistentWithVerified(obs)`.
func checkObservationConsistentWithLogs(i *Interop, obs RoundObservation) error {
	const pred = "Interop.dfy ObservationConsistentWithLogs"
	if obs.Paused { // every conjunct is conditional on a non-paused round
		return nil
	}
	p := modelParamsFromInterop(i)
	var errs []error
	if obs.L1NeedsRewind && obs.LastVerifiedTS != nil && *obs.LastVerifiedTS > p.ActivationTimestamp {
		if err := checkSealedMatchesVerifiedHeads(i, pred, "1", "2", *obs.LastVerifiedTS-1); err != nil {
			errs = append(errs, err)
		}
	}
	if obs.ChainsReady && obs.L1Consistent && !obs.L1NeedsRewind {
		if err := CheckAdvancesAllLogsDBs(i, obs.NextTimestamp, obs.BlocksAtTS); err != nil {
			errs = append(errs, fmt.Errorf("%s conjunct (3): %w", pred, err))
		}
	}
	return errors.Join(errs...)
}

// CheckObservationConsistentWithLogs mirrors ObservationConsistentWithLogs(obs)
// in op-supernode/dafny-models/Interop.dfy, under the RoundObservation field
// mapping: model l1Consistent ↔ !obs.L1NeedsRewind, model l2sConsistent ↔
// obs.L1Consistent (SPEC.md model-to-Go mapping). The model's `requires
// Valid()`, `requires ValidRoundObservation(obs)`, and `requires
// ObservationConsistentWithVerified(obs)` are checked first; on failure the
// predicate body is skipped. Go Paused is outside the model: every conjunct is
// conditional on round flags set after observeRound's pause check, so the
// whole body is skipped when obs.Paused is true. Conjuncts:
//
//	(0) i is non-nil and DB reads succeed (mapping requirement)
//	forall k in logsDBs.Keys when !l1Consistent and
//	obs.lastVerifiedTS.value > ACTIVATION_TIMESTAMP:
//	  (1) SealedBlockForVerifiedAtTimestamp(k, obs.lastVerifiedTS.value - 1).Some?
//	  (2) the seal's id == verifiedDB.Get(obs.lastVerifiedTS.value - 1).l2Heads[k]
//	(3) obs.chainsReady && l2sConsistent && l1Consistent ==>
//	    AdvancesAllLogsDBs(obs.nextTimestamp, obs.blocksAtTS)
func CheckObservationConsistentWithLogs(i *Interop, obs RoundObservation) error {
	const pred = "Interop.dfy ObservationConsistentWithLogs"
	if i == nil {
		return violation(pred, "0", "Interop is nil")
	}
	if err := CheckInteropValid(i); err != nil {
		return fmt.Errorf("%s requires Valid(): %w", pred, err)
	}
	if err := CheckValidRoundObservation(modelParamsFromInterop(i), obs); err != nil {
		return fmt.Errorf("%s requires ValidRoundObservation: %w", pred, err)
	}
	if err := checkObservationConsistentWithVerified(i, obs); err != nil {
		return fmt.Errorf("%s requires ObservationConsistentWithVerified: %w", pred, err)
	}
	return checkObservationConsistentWithLogs(i, obs)
}

// AssertObservationConsistentWithLogs fails t when
// CheckObservationConsistentWithLogs reports violations.
func AssertObservationConsistentWithLogs(t dafnyT, i *Interop, obs RoundObservation) {
	t.Helper()
	failOnViolation(t, CheckObservationConsistentWithLogs(i, obs))
}

// === transition ===

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

// === interop ===

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
	if i.verifiedDB == nil || i.verifiedDB.concrete() == nil || i.verifiedDB.concrete().db == nil {
		errs = append(errs, violation(pred, "4", "VerifiedDB has no underlying store"))
		return errors.Join(errs...)
	}

	if err := CheckVerifiedDBValid(i.verifiedDB.concrete()); err != nil {
		errs = append(errs, fmt.Errorf("%s conjunct (4): %w", pred, err))
	} else if db, dbErr := i.verifiedDB.concrete().allVerified(); dbErr != nil {
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
	if i.verifiedDB == nil || i.verifiedDB.concrete() == nil || i.verifiedDB.concrete().db == nil {
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
	if err := CheckVerifiedDBValid(i.verifiedDB.concrete()); err != nil {
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
	if err := CheckVerifiedDBValid(i.verifiedDB.concrete()); err != nil {
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
