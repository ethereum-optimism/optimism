package solver

import (
	"context"
	"encoding/hex"
	"testing"

	faulttest "github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type actionMaker func(game types.Game) Action

func TestCalculateNextActions(t *testing.T) {
	maxDepth := 4
	claimBuilder := faulttest.NewAlphabetClaimBuilder(t, maxDepth)

	attackClaim := func(parentIdx int) actionMaker {
		return func(game types.Game) Action {
			parentClaim := game.Claims()[parentIdx]
			return Action{
				Type:      ActionTypeMove,
				ParentIdx: parentIdx,
				IsAttack:  true,
				Value:     claimBuilder.CorrectClaimAtPosition(parentClaim.Position.Attack()),
			}
		}
	}
	defendClaim := func(parentIdx int) actionMaker {
		return func(game types.Game) Action {
			parentClaim := game.Claims()[parentIdx]
			return Action{
				Type:      ActionTypeMove,
				ParentIdx: parentIdx,
				IsAttack:  false,
				Value:     claimBuilder.CorrectClaimAtPosition(parentClaim.Position.Defend()),
			}
		}
	}
	stepAttack := func(parentIdx int) actionMaker {
		return func(game types.Game) Action {
			parentClaim := game.Claims()[parentIdx]
			traceIdx := parentClaim.Position.TraceIndex(maxDepth)
			return Action{
				Type:       ActionTypeStep,
				ParentIdx:  parentIdx,
				IsAttack:   true,
				PreState:   claimBuilder.CorrectPreState(traceIdx),
				ProofData:  claimBuilder.CorrectProofData(traceIdx),
				OracleData: claimBuilder.CorrectOracleData(traceIdx),
			}
		}
	}
	stepDefend := func(parentIdx int) actionMaker {
		return func(game types.Game) Action {
			parentClaim := game.Claims()[parentIdx]
			traceIdx := parentClaim.Position.TraceIndex(maxDepth) + 1
			return Action{
				Type:       ActionTypeStep,
				ParentIdx:  parentIdx,
				IsAttack:   false,
				PreState:   claimBuilder.CorrectPreState(traceIdx),
				ProofData:  claimBuilder.CorrectProofData(traceIdx),
				OracleData: claimBuilder.CorrectOracleData(traceIdx),
			}
		}
	}

	tests := []struct {
		name                string
		agreeWithOutputRoot bool
		rootClaimCorrect    bool
		setupGame           func(builder *faulttest.GameBuilder)
		expectedActions     []actionMaker
	}{
		{
			name:                "AttackRootClaim",
			agreeWithOutputRoot: true,
			setupGame:           func(builder *faulttest.GameBuilder) {},
			expectedActions: []actionMaker{
				attackClaim(0),
			},
		},
		{
			name:                "DoNotAttackRootClaimWhenDisagreeWithOutputRoot",
			agreeWithOutputRoot: false,
			setupGame:           func(builder *faulttest.GameBuilder) {},
			expectedActions:     nil,
		},
		{
			// Note: The fault dispute game contract should prevent a correct root claim from actually being posted
			// But for completeness, test we ignore it so we don't get sucked into playing an unwinnable game.
			name:                "DoNotAttackCorrectRootClaim_AgreeWithOutputRoot",
			agreeWithOutputRoot: true,
			rootClaimCorrect:    true,
			setupGame:           func(builder *faulttest.GameBuilder) {},
			expectedActions:     nil,
		},
		{
			// Note: The fault dispute game contract should prevent a correct root claim from actually being posted
			// But for completeness, test we ignore it so we don't get sucked into playing an unwinnable game.
			name:                "DoNotAttackCorrectRootClaim_DisagreeWithOutputRoot",
			agreeWithOutputRoot: false,
			rootClaimCorrect:    true,
			setupGame:           func(builder *faulttest.GameBuilder) {},
			expectedActions:     nil,
		},

		{
			name:                "DoNotPerformDuplicateMoves",
			agreeWithOutputRoot: true,
			setupGame: func(builder *faulttest.GameBuilder) {
				// Expected move has already been made.
				builder.Seq().AttackCorrect()
			},
			expectedActions: nil,
		},

		{
			name:                "RespondToAllClaimsAtDisagreeingLevel",
			agreeWithOutputRoot: true,
			setupGame: func(builder *faulttest.GameBuilder) {
				honestClaim := builder.Seq().AttackCorrect() // 1
				honestClaim.AttackCorrect()                  // 2
				honestClaim.DefendCorrect()                  // 3
				honestClaim.Attack(common.Hash{0xaa})        // 4
				honestClaim.Attack(common.Hash{0xbb})        // 5
				honestClaim.Defend(common.Hash{0xcc})        // 6
				honestClaim.Defend(common.Hash{0xdd})        // 7
			},
			expectedActions: []actionMaker{
				// Defend the correct claims
				defendClaim(2),
				defendClaim(3),

				// Attack the incorrect claims
				attackClaim(4),
				attackClaim(5),
				attackClaim(6),
				attackClaim(7),
			},
		},

		{
			name:                "StepAtMaxDepth",
			agreeWithOutputRoot: true,
			setupGame: func(builder *faulttest.GameBuilder) {
				lastHonestClaim := builder.Seq().
					AttackCorrect(). // 1 - Honest
					AttackCorrect(). // 2 - Dishonest
					DefendCorrect()  // 3 - Honest
				lastHonestClaim.AttackCorrect()           // 4 - Dishonest
				lastHonestClaim.Attack(common.Hash{0xdd}) // 5 - Dishonest
			},
			expectedActions: []actionMaker{
				stepDefend(4),
				stepAttack(5),
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			builder := claimBuilder.GameBuilder(test.agreeWithOutputRoot, test.rootClaimCorrect)
			test.setupGame(builder)
			game := builder.Game
			for i, claim := range game.Claims() {
				t.Logf("Claim %v: Pos: %v ParentIdx: %v, Countered: %v, Value: %v", i, claim.Position.ToGIndex(), claim.ParentContractIndex, claim.Countered, claim.Value)
			}

			solver := NewGameSolver(maxDepth, claimBuilder.CorrectTraceProvider())
			actions, err := solver.CalculateNextActions(context.Background(), game)
			require.NoError(t, err)
			for i, action := range actions {
				t.Logf("Move %v: Type: %v, ParentIdx: %v, Attack: %v, Value: %v, PreState: %v, ProofData: %v",
					i, action.Type, action.ParentIdx, action.IsAttack, action.Value, hex.EncodeToString(action.PreState), hex.EncodeToString(action.ProofData))
			}
			require.Len(t, actions, len(test.expectedActions))
			for i, action := range test.expectedActions {
				require.Containsf(t, actions, action(game), "Expected claim %v missing", i)
			}
		})
	}
}
