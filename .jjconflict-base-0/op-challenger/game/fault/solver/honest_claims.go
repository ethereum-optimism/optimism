package solver

import "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

type honestClaimTracker struct {
	// agreed tracks the existing claims in the game that the honest actor would make
	// The claims may not yet have been made so are tracked by ClaimID not ContractIndex
	agreed map[types.ClaimID]bool

	// counters tracks the counter claim for a claim by contract index.
	// The counter claim may not yet be part of the game state (ie it may be a move the honest actor is planning to make)
	counters map[types.ClaimID]types.Claim
}

func newHonestClaimTracker() *honestClaimTracker {
	return &honestClaimTracker{
		agreed:   make(map[types.ClaimID]bool),
		counters: make(map[types.ClaimID]types.Claim),
	}
}

func (a *honestClaimTracker) AddHonestClaim(parent types.Claim, claim types.Claim) {
	a.agreed[claim.ID()] = true
	if parent != (types.Claim{}) {
		a.counters[parent.ID()] = claim
	}
}

func (a *honestClaimTracker) IsHonest(claim types.Claim) bool {
	return a.agreed[claim.ID()]
}

func (a *honestClaimTracker) HonestCounter(parent types.Claim) (types.Claim, bool) {
	counter, ok := a.counters[parent.ID()]
	return counter, ok
}
