package resolution

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/transform"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

func TestResolver_Resolve(t *testing.T) {
	t.Run("NoClaims", func(t *testing.T) {
		tree := transform.CreateBidirectionalTree([]faultTypes.Claim{})
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})

	t.Run("SingleRootClaim", func(t *testing.T) {
		builder := test.NewAlphabetClaimBuilder(t, big.NewInt(10), 4).GameBuilder(true)
		tree := transform.CreateBidirectionalTree(builder.Game.Claims())
		tree.Claims[0].Claim.CounteredBy = common.Address{}
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})

	t.Run("ManyClaims_ChallengerWon", func(t *testing.T) {
		builder := test.NewAlphabetClaimBuilder(t, big.NewInt(10), 5).GameBuilder(true)
		builder.Seq(). // Defender winning
				AttackCorrect(). // Challenger winning
				AttackCorrect(). // Defender winning
				DefendCorrect(). // Challenger winning
				DefendCorrect(). // Defender winning
				AttackCorrect()  // Challenger winning
		tree := transform.CreateBidirectionalTree(builder.Game.Claims())
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusChallengerWon, status)
	})

	t.Run("ManyClaims_DefenderWon", func(t *testing.T) {
		builder := test.NewAlphabetClaimBuilder(t, big.NewInt(10), 5).GameBuilder(true)
		builder.Seq(). // Defender winning
				AttackCorrect(). // Challenger winning
				AttackCorrect(). // Defender winning
				DefendCorrect(). // Challenger winning
				DefendCorrect()  // Defender winning
		tree := transform.CreateBidirectionalTree(builder.Game.Claims())
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})

	t.Run("MultipleBranches_ChallengerWon", func(t *testing.T) {
		builder := test.NewAlphabetClaimBuilder(t, big.NewInt(10), 5).GameBuilder(true)
		forkPoint := builder.Seq(). // Defender winning
						AttackCorrect(). // Challenger winning
						AttackCorrect()  // Defender winning
		forkPoint.
			DefendCorrect(). // Challenger winning
			DefendCorrect(). // Defender winning
			AttackCorrect()  // Challenger winning
		forkPoint.Defend(common.Hash{0xbb}). // Challenger winning
							DefendCorrect(). // Defender winning
							AttackCorrect(). // Challenger winning
							Step()           // Defender winning
		forkPoint.Defend(common.Hash{0xcc}). // Challenger winning
							DefendCorrect() // Defender winning
		tree := transform.CreateBidirectionalTree(builder.Game.Claims())
		status := Resolve(tree)
		// First fork has an uncountered claim with challenger winning so that invalidates the parent and wins the game
		require.Equal(t, gameTypes.GameStatusChallengerWon, status)
	})

	t.Run("MultipleBranches_DefenderWon", func(t *testing.T) {
		builder := test.NewAlphabetClaimBuilder(t, big.NewInt(10), 5).GameBuilder(true)
		forkPoint := builder.Seq(). // Defender winning
						AttackCorrect(). // Challenger winning
						AttackCorrect()  // Defender winning
		forkPoint.
			DefendCorrect(). // Challenger winning
			DefendCorrect()  // Defender winning
		forkPoint.Defend(common.Hash{0xbb}). // Challenger winning
							DefendCorrect(). // Defender winning
							AttackCorrect(). // Challenger winning
							Step()           // Defender winning
		forkPoint.Defend(common.Hash{0xcc}). // Challenger winning
							DefendCorrect() // Defender winning
		tree := transform.CreateBidirectionalTree(builder.Game.Claims())
		status := Resolve(tree)
		// Defender won all forks
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})

	t.Run("SteppedClaimed_ChallengerWon", func(t *testing.T) {
		builder := test.NewAlphabetClaimBuilder(t, big.NewInt(10), 4).GameBuilder(true)
		builder.Seq(). // Defender winning
				AttackCorrect(). // Challenger winning
				AttackCorrect(). // Defender winning
				DefendCorrect(). // Challenger winning
				DefendCorrect(). // Defender winning
				Step()           // Challenger winning
		claims := builder.Game.Claims()
		// Successful step so mark as countered
		claims[len(claims)-1].CounteredBy = common.Address{0xaa}
		tree := transform.CreateBidirectionalTree(claims)
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusChallengerWon, status)
	})

	t.Run("SteppedClaimed_DefenderWon", func(t *testing.T) {
		builder := test.NewAlphabetClaimBuilder(t, big.NewInt(10), 5).GameBuilder(true)
		builder.Seq(). // Defender winning
				AttackCorrect(). // Challenger winning
				AttackCorrect(). // Defender winning
				DefendCorrect(). // Challenger winning
				DefendCorrect(). // Defender winning
				AttackCorrect(). // Challenger winning
				Step()           // Defender winning
		claims := builder.Game.Claims()
		// Successful step so mark as countered
		claims[len(claims)-1].CounteredBy = common.Address{0xaa}
		tree := transform.CreateBidirectionalTree(claims)
		status := Resolve(tree)
		require.Equal(t, gameTypes.GameStatusDefenderWon, status)
	})
}
