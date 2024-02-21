package transform

import (
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
)

// CreateBidirectionalTree walks backwards through the list of claims and creates a bidirectional
// tree of claims. The root claim must be at index 0. The tree is returned as a flat array so it
// can be easily traversed following the resolution process.
func CreateBidirectionalTree(claims []types.Claim) *monTypes.BidirectionalTree {
	claimMap := make(map[int]*monTypes.BidirectionalClaim)
	res := make([]*monTypes.BidirectionalClaim, 0, len(claims))
	for _, claim := range claims {
		claim := claim
		bidirectionalClaim := &monTypes.BidirectionalClaim{
			Claim: &claim,
		}
		claimMap[claim.ContractIndex] = bidirectionalClaim
		if !claim.IsRoot() {
			// SAFETY: the parent must exist in the list prior to the current claim.
			parent := claimMap[claim.ParentContractIndex]
			parent.Children = append(parent.Children, bidirectionalClaim)
		}
		res = append(res, bidirectionalClaim)
	}
	return &monTypes.BidirectionalTree{Claims: res}
}
