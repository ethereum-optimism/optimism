package mon

import (
	"fmt"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

type BidirectionalClaim struct {
	Claim    *faultTypes.Claim
	Children []*BidirectionalClaim
}

// Resolve creates the bidirectional tree of claims and then computes the resolved game status.
func Resolve(claims []faultTypes.Claim) (types.GameStatus, error) {
	flatBidireactionalTree, err := createBidirectionalTree(claims)
	if err != nil {
		return 0, fmt.Errorf("failed to create bidirectional tree: %w", err)
	}
	return resolveTree(flatBidireactionalTree), nil
}

// createBidirectionalTree walks backwards through the list of claims and creates a bidirectional
// tree of claims. The root claim must be at index 0. The tree is returned as a flat array so it
// can be easily traversed following the resolution process.
func createBidirectionalTree(claims []faultTypes.Claim) ([]*BidirectionalClaim, error) {
	claimMap := make(map[int]*BidirectionalClaim)
	res := make([]*BidirectionalClaim, 0, len(claims))
	for _, claim := range claims {
		claim := claim
		bidirectionalClaim := &BidirectionalClaim{
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
	return res, nil
}

// resolveTree iterates backwards over the bidirectional tree, iteratively
// checking the leftmost counter of each claim, and updating the claim's counter
// claimant. Once the root claim is reached, the resolution game status is returned.
func resolveTree(tree []*BidirectionalClaim) types.GameStatus {
	for i := len(tree) - 1; i >= 0; i-- {
		claim := tree[i]
		counterClaimant := common.Address{}
		for _, child := range claim.Children {
			if child.Claim.CounteredBy == (common.Address{}) {
				counterClaimant = child.Claim.Claimant
			}
		}
		claim.Claim.CounteredBy = counterClaimant
	}
	if (len(tree) == 0 || tree[0].Claim.CounteredBy == common.Address{}) {
		return types.GameStatusDefenderWon
	} else {
		return types.GameStatusChallengerWon
	}
}
