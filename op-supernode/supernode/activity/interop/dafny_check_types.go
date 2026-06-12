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
