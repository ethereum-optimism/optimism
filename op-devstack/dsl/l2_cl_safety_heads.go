package dsl

import (
	"golang.org/x/sync/errgroup"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// SafetyHeadsAdvancement is the per-level delta and attempt budget for
// asserting unsafe + safe + cross-safe + finalized progress together.
type SafetyHeadsAdvancement struct {
	UnsafeDelta       uint64
	UnsafeAttempts    int
	SafeDelta         uint64
	SafeAttempts      int
	CrossSafeDelta    uint64
	CrossSafeAttempts int
	FinalizedDelta    uint64
	FinalizedAttempts int
}

// DefaultSafetyHeadsAdvancement: unsafe + safe + cross-safe within ~60s,
// finalized within ~720s (in-process L1 has FinalizedDistance=20 blocks @ 6s).
func DefaultSafetyHeadsAdvancement() SafetyHeadsAdvancement {
	return SafetyHeadsAdvancement{
		UnsafeDelta:       5,
		UnsafeAttempts:    30,
		SafeDelta:         1,
		SafeAttempts:      30,
		CrossSafeDelta:    1,
		CrossSafeAttempts: 30,
		FinalizedDelta:    1,
		FinalizedAttempts: 360,
	}
}

// SafetyHeads is a snapshot of an L2 CL's unsafe/safe/cross-safe/finalized
// block numbers.
type SafetyHeads struct {
	Unsafe    uint64
	Safe      uint64
	CrossSafe uint64
	Finalized uint64
}

func (cl *L2CLNode) SafetyHeads() SafetyHeads {
	return SafetyHeads{
		Unsafe:    cl.HeadBlockRef(safety.LocalUnsafe).Number,
		Safe:      cl.HeadBlockRef(safety.LocalSafe).Number,
		CrossSafe: cl.HeadBlockRef(safety.CrossSafe).Number,
		Finalized: cl.HeadBlockRef(safety.Finalized).Number,
	}
}

// AllHeadsAdvancedFn asserts every safety head advances from the head
// observed at call time, using DefaultSafetyHeadsAdvancement.
func (cl *L2CLNode) AllHeadsAdvancedFn() CheckFunc {
	adv := DefaultSafetyHeadsAdvancement()
	fns := []CheckFunc{
		cl.AdvancedFn(safety.LocalUnsafe, adv.UnsafeDelta, adv.UnsafeAttempts),
		cl.AdvancedFn(safety.LocalSafe, adv.SafeDelta, adv.SafeAttempts),
		cl.AdvancedFn(safety.CrossSafe, adv.CrossSafeDelta, adv.CrossSafeAttempts),
		cl.AdvancedFn(safety.Finalized, adv.FinalizedDelta, adv.FinalizedAttempts),
	}
	return runAllInParallel(fns)
}

// AllHeadsReachedFn asserts every safety head reaches snap + the default
// delta. Used to assert catch-up after a resync.
func (cl *L2CLNode) AllHeadsReachedFn(snap SafetyHeads) CheckFunc {
	adv := DefaultSafetyHeadsAdvancement()
	fns := []CheckFunc{
		cl.ReachedFn(safety.LocalUnsafe, snap.Unsafe+adv.UnsafeDelta, adv.UnsafeAttempts),
		cl.ReachedFn(safety.LocalSafe, snap.Safe+adv.SafeDelta, adv.SafeAttempts),
		cl.ReachedFn(safety.CrossSafe, snap.CrossSafe+adv.CrossSafeDelta, adv.CrossSafeAttempts),
		cl.ReachedFn(safety.Finalized, snap.Finalized+adv.FinalizedDelta, adv.FinalizedAttempts),
	}
	return runAllInParallel(fns)
}

func runAllInParallel(fns []CheckFunc) CheckFunc {
	return func() error {
		var g errgroup.Group
		for _, fn := range fns {
			fn := fn
			g.Go(func() error { return fn() })
		}
		return g.Wait()
	}
}
