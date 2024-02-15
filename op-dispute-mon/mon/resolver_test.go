package mon

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
)

func TestResolver_Resolve(t *testing.T) {
	t.Run("NoClaims", func(t *testing.T) {
		tree := &monTypes.BidirectionalTree{Claims: []*monTypes.BidirectionalClaim{}}
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})

	t.Run("SingleRootClaim", func(t *testing.T) {
		tree := createBidirectionalTree(1)
		tree.Claims[0].Claim.CounteredBy = common.Address{}
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})

	t.Run("ManyClaims_ChallengerWon", func(t *testing.T) {
		tree := createBidirectionalTree(2)
		tree.Claims[1].Claim.CounteredBy = common.Address{}
		tree.Claims[1].Children = make([]*monTypes.BidirectionalClaim, 0)
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusChallengerWon, status)
	})

	t.Run("ManyClaims_DefenderWon", func(t *testing.T) {
		status := Resolve(createBidirectionalTree(3))
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})
}

func createBidirectionalTree(claimCount uint64) *monTypes.BidirectionalTree {
	claimList := createDeepClaimList()
	bidirectionalClaimList := make([]*monTypes.BidirectionalClaim, len(claimList))
	bidirectionalClaimList[2] = &monTypes.BidirectionalClaim{
		Claim:    &claimList[2],
		Children: make([]*monTypes.BidirectionalClaim, 0),
	}
	bidirectionalClaimList[1] = &monTypes.BidirectionalClaim{
		Claim:    &claimList[1],
		Children: []*monTypes.BidirectionalClaim{bidirectionalClaimList[2]},
	}
	bidirectionalClaimList[0] = &monTypes.BidirectionalClaim{
		Claim:    &claimList[0],
		Children: []*monTypes.BidirectionalClaim{bidirectionalClaimList[1]},
	}
	return &monTypes.BidirectionalTree{Claims: bidirectionalClaimList[:claimCount]}
}

func createDeepClaimList() []faultTypes.Claim {
	return []faultTypes.Claim{
		{
			ClaimData: faultTypes.ClaimData{
				Position: faultTypes.NewPosition(0, big.NewInt(0)),
			},
			ContractIndex:       0,
			CounteredBy:         common.HexToAddress("0x222222"),
			ParentContractIndex: math.MaxInt64,
			Claimant:            common.HexToAddress("0x111111"),
		},
		{
			ClaimData: faultTypes.ClaimData{
				Position: faultTypes.NewPosition(1, big.NewInt(0)),
			},
			CounteredBy:         common.HexToAddress("0x111111"),
			ContractIndex:       1,
			ParentContractIndex: 0,
			Claimant:            common.HexToAddress("0x222222"),
		},
		{
			ClaimData: faultTypes.ClaimData{
				Position: faultTypes.NewPosition(2, big.NewInt(0)),
			},
			ContractIndex:       2,
			ParentContractIndex: 1,
			Claimant:            common.HexToAddress("0x111111"),
		},
	}
}
