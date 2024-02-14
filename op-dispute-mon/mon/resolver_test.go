package mon

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

func TestResolver_Resolve(t *testing.T) {
	t.Run("NoClaims", func(t *testing.T) {
		status, err := Resolve([]faultTypes.Claim{})
		require.NoError(t, err)
		require.Equal(t, types.GameStatusDefenderWon, status)
	})

	t.Run("SingleClaim", func(t *testing.T) {
		status, err := Resolve(createDeepClaimList()[:1])
		require.NoError(t, err)
		require.Equal(t, types.GameStatusDefenderWon, status)
	})

	t.Run("MultipleClaims", func(t *testing.T) {
		status, err := Resolve(createDeepClaimList()[:2])
		require.NoError(t, err)
		require.Equal(t, types.GameStatusChallengerWon, status)
	})

	t.Run("MultipleClaimsAndChildren", func(t *testing.T) {
		status, err := Resolve(createDeepClaimList())
		require.NoError(t, err)
		require.Equal(t, types.GameStatusDefenderWon, status)
	})
}

func TestResolver_CreateBidirectionalTree(t *testing.T) {
	t.Run("SingleClaim", func(t *testing.T) {
		claims := createDeepClaimList()[:1]
		claims[0].CounteredBy = common.Address{}
		tree, err := createBidirectionalTree(claims)
		require.NoError(t, err)
		require.Len(t, tree, 1)
		require.Equal(t, claims[0], *tree[0].Claim)
		require.Empty(t, tree[0].Children)
	})

	t.Run("MultipleClaims", func(t *testing.T) {
		claims := createDeepClaimList()[:2]
		claims[1].CounteredBy = common.Address{}
		tree, err := createBidirectionalTree(claims)
		require.NoError(t, err)
		require.Len(t, tree, 2)
		require.Equal(t, claims[0], *tree[0].Claim)
		require.Len(t, tree[0].Children, 1)
		require.Equal(t, claims[1], *tree[0].Children[0].Claim)
		require.Equal(t, claims[1], *tree[1].Claim)
		require.Empty(t, tree[1].Children)
	})

	t.Run("MultipleClaimsAndChildren", func(t *testing.T) {
		claims := createDeepClaimList()
		tree, err := createBidirectionalTree(claims)
		require.NoError(t, err)
		require.Len(t, tree, 3)
		require.Equal(t, claims[0], *tree[0].Claim)
		require.Len(t, tree[0].Children, 1)
		require.Equal(t, tree[0].Children[0], tree[1])
		require.Equal(t, claims[1], *tree[1].Claim)
		require.Len(t, tree[1].Children, 1)
		require.Equal(t, tree[1].Children[0], tree[2])
		require.Equal(t, claims[2], *tree[2].Claim)
		require.Empty(t, tree[2].Children)
	})
}

func TestResolver_ResolveTree(t *testing.T) {
	t.Run("NoClaims", func(t *testing.T) {
		status := resolveTree([]*BidirectionalClaim{})
		require.Equal(t, types.GameStatusDefenderWon, status)
	})

	t.Run("SingleRootClaim", func(t *testing.T) {
		list := createBidirectionalClaimList()[:1]
		list[0].Claim.CounteredBy = common.Address{}
		status := resolveTree(list)
		require.Equal(t, types.GameStatusDefenderWon, status)
	})

	t.Run("ChallengerWon", func(t *testing.T) {
		list := createBidirectionalClaimList()[:2]
		list[1].Claim.CounteredBy = common.Address{}
		list[1].Children = make([]*BidirectionalClaim, 0)
		status := resolveTree(list)
		require.Equal(t, types.GameStatusChallengerWon, status)
	})

	t.Run("DefenderWon", func(t *testing.T) {
		status := resolveTree(createBidirectionalClaimList())
		require.Equal(t, types.GameStatusDefenderWon, status)
	})
}

func createBidirectionalClaimList() []*BidirectionalClaim {
	claimList := createDeepClaimList()
	bidirectionalClaimList := make([]*BidirectionalClaim, len(claimList))
	bidirectionalClaimList[2] = &BidirectionalClaim{
		Claim:    &claimList[2],
		Children: make([]*BidirectionalClaim, 0),
	}
	bidirectionalClaimList[1] = &BidirectionalClaim{
		Claim:    &claimList[1],
		Children: []*BidirectionalClaim{bidirectionalClaimList[2]},
	}
	bidirectionalClaimList[0] = &BidirectionalClaim{
		Claim:    &claimList[0],
		Children: []*BidirectionalClaim{bidirectionalClaimList[1]},
	}
	return bidirectionalClaimList
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
