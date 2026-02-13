package transform

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
)

func TestResolver_CreateBidirectionalTree(t *testing.T) {
	t.Run("SingleClaim", func(t *testing.T) {
		claims := createDeepClaimList()[:1]
		claims[0].CounteredBy = common.Address{}
		tree := CreateBidirectionalTree(claims)
		require.Len(t, tree.Claims, 1)
		require.Equal(t, claims[0].Claim, *tree.Claims[0].Claim)
		require.Empty(t, tree.Claims[0].Children)
	})

	t.Run("MultipleClaims", func(t *testing.T) {
		claims := createDeepClaimList()[:2]
		claims[1].CounteredBy = common.Address{}
		tree := CreateBidirectionalTree(claims)
		require.Len(t, tree.Claims, 2)
		require.Equal(t, claims[0].Claim, *tree.Claims[0].Claim)
		require.Len(t, tree.Claims[0].Children, 1)
		require.Equal(t, claims[1].Claim, *tree.Claims[0].Children[0].Claim)
		require.Equal(t, claims[1].Claim, *tree.Claims[1].Claim)
		require.Empty(t, tree.Claims[1].Children)
	})

	t.Run("MultipleClaimsAndChildren", func(t *testing.T) {
		claims := createDeepClaimList()
		tree := CreateBidirectionalTree(claims)
		require.Len(t, tree.Claims, 3)
		require.Equal(t, claims[0].Claim, *tree.Claims[0].Claim)
		require.Len(t, tree.Claims[0].Children, 1)
		require.Equal(t, tree.Claims[0].Children[0], tree.Claims[1])
		require.Equal(t, claims[1].Claim, *tree.Claims[1].Claim)
		require.Len(t, tree.Claims[1].Children, 1)
		require.Equal(t, tree.Claims[1].Children[0], tree.Claims[2])
		require.Equal(t, claims[2].Claim, *tree.Claims[2].Claim)
		require.Empty(t, tree.Claims[2].Children)
	})
}

func createDeepClaimList() []monTypes.EnrichedClaim {
	return []monTypes.EnrichedClaim{
		{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Position: types.NewPosition(0, big.NewInt(0)),
				},
				ContractIndex:       0,
				CounteredBy:         common.HexToAddress("0x222222"),
				ParentContractIndex: math.MaxInt64,
				Claimant:            common.HexToAddress("0x111111"),
			},
		},
		{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Position: types.NewPosition(1, big.NewInt(0)),
				},
				CounteredBy:         common.HexToAddress("0x111111"),
				ContractIndex:       1,
				ParentContractIndex: 0,
				Claimant:            common.HexToAddress("0x222222"),
			},
		},
		{
			Claim: types.Claim{
				ClaimData: types.ClaimData{
					Position: types.NewPosition(2, big.NewInt(0)),
				},
				ContractIndex:       2,
				ParentContractIndex: 1,
				Claimant:            common.HexToAddress("0x111111"),
			},
		},
	}
}
