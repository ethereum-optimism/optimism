package solver

import "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

type agreedClaimTracker struct {
	agreed map[int]bool
}

func newAgreedClaimTracker() *agreedClaimTracker {
	return &agreedClaimTracker{
		agreed: make(map[int]bool),
	}
}

func (a *agreedClaimTracker) MarkAgreed(claim types.Claim) {
	a.agreed[claim.ContractIndex] = true
}

func (a *agreedClaimTracker) IsAgreed(claim types.Claim) bool {
	return a.agreed[claim.ContractIndex]
}
