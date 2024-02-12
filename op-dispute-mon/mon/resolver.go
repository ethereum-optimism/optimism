package mon

import (
	"fmt"
	"math"

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
	for i := len(claims) - 1; i >= 0; i-- {
		claim := claims[i]
		// Update this claim if it exists, otherwise create a new claim.
		var bidirectionalClaim *BidirectionalClaim
		stored, ok := claimMap[claim.ContractIndex]
		if ok {
			// This is where we set the "parent" claim if it exists (i.e. the claim has children).
			stored.Claim = &claim
			bidirectionalClaim = stored
		} else {
			claimMap[claim.ContractIndex] = &BidirectionalClaim{
				Claim:    &claim,
				Children: []*BidirectionalClaim{},
			}
			bidirectionalClaim = claimMap[claim.ContractIndex]
		}
		// Update the parent if it exists, otherwise create a new parent.
		parent, ok := claimMap[claim.ParentContractIndex]
		if !ok {
			// Do not set the claim since this is set when we iterate to the parent.
			parent = &BidirectionalClaim{
				Children: []*BidirectionalClaim{bidirectionalClaim},
			}
			claimMap[claim.ParentContractIndex] = parent
		} else {
			parent.Children = append(parent.Children, bidirectionalClaim)
		}

		// Append the claim to the front of the res array
		res = append([]*BidirectionalClaim{bidirectionalClaim}, res...)
	}
	return res, nil
}

// resolveTree iterates backwards over the bidirectional tree, iteratively
// checking the leftmost counter of each claim, and updating the claim's counter
// claimant. Once the root claim is reached, the resolution game status is returned.
func resolveTree(tree []*BidirectionalClaim) types.GameStatus {
	for i := len(tree) - 1; i >= 0; i-- {
		leftMostCounter := uint64(math.MaxUint64)
		claim := tree[i]
		counterClaimant := common.Address{}
		for _, child := range claim.Children {
			notCountered := child.Claim.CounteredBy == common.Address{}
			moreLeft := child.Claim.Position.ToGIndex().Uint64() < leftMostCounter
			if notCountered && moreLeft {
				leftMostCounter = child.Claim.Position.IndexAtDepth().Uint64()
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
