package solver

import (
	"context"
	"encoding/hex"
	"testing"

	faulttest "github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateNextActions(t *testing.T) {
	maxDepth := 4
	claimBuilder := faulttest.NewAlphabetClaimBuilder(t, maxDepth)

	tests := []struct {
		name             string
		rootClaimCorrect bool
		setupGame        func(builder *faulttest.GameBuilder)
	}{
		{
			name: "AttackRootClaim",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().ExpectAttack()
			},
		},
		{
			// Note: The fault dispute game contract should prevent a correct root claim from actually being posted
			// But for completeness, test we ignore it so we don't get sucked into playing an unwinnable game.
			name:             "DoNotAttackCorrectRootClaim_AgreeWithOutputRoot",
			rootClaimCorrect: true,
			setupGame:        func(builder *faulttest.GameBuilder) {},
		},
		{
			name: "DoNotPerformDuplicateMoves",
			setupGame: func(builder *faulttest.GameBuilder) {
				// Expected move has already been made.
				builder.Seq().AttackCorrect()
			},
		},
		{
			name: "RespondToAllClaimsAtDisagreeingLevel",
			setupGame: func(builder *faulttest.GameBuilder) {
				honestClaim := builder.Seq().AttackCorrect()
				honestClaim.AttackCorrect().ExpectDefend()
				honestClaim.DefendCorrect().ExpectDefend()
				honestClaim.Attack(common.Hash{0xaa}).ExpectAttack()
				honestClaim.Attack(common.Hash{0xbb}).ExpectAttack()
				honestClaim.Defend(common.Hash{0xcc}).ExpectAttack()
				honestClaim.Defend(common.Hash{0xdd}).ExpectAttack()
			},
		},
		{
			name: "StepAtMaxDepth",
			setupGame: func(builder *faulttest.GameBuilder) {
				lastHonestClaim := builder.Seq().
					AttackCorrect().
					AttackCorrect().
					DefendCorrect()
				lastHonestClaim.AttackCorrect().ExpectStepDefend()
				lastHonestClaim.Attack(common.Hash{0xdd}).ExpectStepAttack()
			},
		},
		{
			name: "PoisonedPreState",
			setupGame: func(builder *faulttest.GameBuilder) {
				// A claim hash that has no pre-image
				maliciousStateHash := common.Hash{0x01, 0xaa}

				// Dishonest actor counters their own claims to set up a situation with an invalid prestate
				// The honest actor should ignore path created by the dishonest actor, only supporting its own attack on the root claim
				honestMove := builder.Seq().AttackCorrect() // This expected action is the winning move.
				dishonestMove := honestMove.Attack(maliciousStateHash)
				// The expected action by the honest actor
				dishonestMove.ExpectAttack()
				// The honest actor will ignore this poisoned path
				dishonestMove.
					Defend(maliciousStateHash).
					Attack(maliciousStateHash)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			builder := claimBuilder.GameBuilder(test.rootClaimCorrect)
			test.setupGame(builder)
			game := builder.Game
			for i, claim := range game.Claims() {
				t.Logf("Claim %v: Pos: %v TraceIdx: %v ParentIdx: %v, Countered: %v, Value: %v",
					i, claim.Position.ToGIndex(), claim.Position.TraceIndex(maxDepth), claim.ParentContractIndex, claim.Countered, claim.Value)
			}

			solver := NewGameSolver(maxDepth, trace.NewSimpleTraceAccessor(claimBuilder.CorrectTraceProvider()))
			actions, err := solver.CalculateNextActions(context.Background(), game)
			require.NoError(t, err)
			for i, action := range actions {
				t.Logf("Move %v: Type: %v, ParentIdx: %v, Attack: %v, Value: %v, PreState: %v, ProofData: %v",
					i, action.Type, action.ParentIdx, action.IsAttack, action.Value, hex.EncodeToString(action.PreState), hex.EncodeToString(action.ProofData))
				// Check that every move the solver returns meets the generic validation rules
				require.NoError(t, checkRules(game, action), "Attempting to perform invalid action")
			}
			for i, action := range builder.ExpectedActions {
				t.Logf("Expect %v: Type: %v, ParentIdx: %v, Attack: %v, Value: %v, PreState: %v, ProofData: %v",
					i, action.Type, action.ParentIdx, action.IsAttack, action.Value, hex.EncodeToString(action.PreState), hex.EncodeToString(action.ProofData))
				require.Containsf(t, actions, action, "Expected claim %v missing", i)
			}
			require.Len(t, actions, len(builder.ExpectedActions), "Incorrect number of actions")
		})
	}
}
