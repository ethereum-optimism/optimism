package solver

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	faulttest "github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateNextActions(t *testing.T) {
	maxDepth := types.Depth(6)
	startingL2BlockNumber := big.NewInt(0)
	claimBuilder := faulttest.NewAlphabetClaimBuilder(t, startingL2BlockNumber, maxDepth)

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
					DefendCorrect().
					DefendCorrect().
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
		{
			name: "Freeloader-ValidClaimAtInvalidAttackPosition",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().                // Honest response to invalid root
					DefendCorrect().ExpectDefend(). // Defender agrees at this point, we should defend
					AttackCorrect().ExpectDefend()  // Freeloader attacks instead of defends
			},
		},
		{
			name: "Freeloader-InvalidClaimAtInvalidAttackPosition",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().                         // Honest response to invalid root
					DefendCorrect().ExpectDefend().          // Defender agrees at this point, we should defend
					Attack(common.Hash{0xbb}).ExpectAttack() // Freeloader attacks with wrong claim instead of defends
			},
		},
		{
			name: "Freeloader-InvalidClaimAtValidDefensePosition",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().                         // Honest response to invalid root
					DefendCorrect().ExpectDefend().          // Defender agrees at this point, we should defend
					Defend(common.Hash{0xbb}).ExpectAttack() // Freeloader defends with wrong claim, we should attack
			},
		},
		{
			name: "Freeloader-InvalidClaimAtValidAttackPosition",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().                          // Honest response to invalid root
					Defend(common.Hash{0xaa}).ExpectAttack(). // Defender disagrees at this point, we should attack
					Attack(common.Hash{0xbb}).ExpectAttack()  // Freeloader attacks with wrong claim instead of defends
			},
		},
		{
			name: "Freeloader-InvalidClaimAtInvalidDefensePosition",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().                          // Honest response to invalid root
					Defend(common.Hash{0xaa}).ExpectAttack(). // Defender disagrees at this point, we should attack
					Defend(common.Hash{0xbb})                 // Freeloader defends with wrong claim but we must not respond to avoid poisoning
			},
		},
		{
			name: "Freeloader-ValidClaimAtInvalidAttackPosition-RespondingToDishonestButCorrectAttack",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().                // Honest response to invalid root
					AttackCorrect().ExpectDefend(). // Defender attacks with correct value, we should defend
					AttackCorrect().ExpectDefend()  // Freeloader attacks with wrong claim, we should defend
			},
		},
		{
			name: "Freeloader-DoNotCounterOwnClaim",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().                // Honest response to invalid root
					AttackCorrect().ExpectDefend(). // Defender attacks with correct value, we should defend
					AttackCorrect().                // Freeloader attacks instead, we should defend
					DefendCorrect()                 // We do defend and we shouldn't counter our own claim
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
				t.Logf("Claim %v: Pos: %v TraceIdx: %v Depth: %v IndexAtDepth: %v ParentIdx: %v Value: %v",
					i, claim.Position.ToGIndex(), claim.Position.TraceIndex(maxDepth), claim.Position.Depth(), claim.Position.IndexAtDepth(), claim.ParentContractIndex, claim.Value)
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
