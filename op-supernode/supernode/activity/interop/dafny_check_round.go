package interop

// This file is part of the Dafny model checkers (dafny_check_*.go files):
// test/debug-only helpers that check predicates from op-supernode/dafny-models/
// against the real Go types. Production code paths must not call them.

import (
	"errors"
	"fmt"
)

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
	if err := CheckVerifiedDBValid(i.verifiedDB); err != nil {
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
	if err := CheckVerifiedDBValid(i.verifiedDB); err != nil {
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
