package solver

import (
	"context"
	"math/big"
	"testing"

	faulttest "github.com/ethereum-optimism/optimism/op-challenger/game/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAttemptStep(t *testing.T) {
	maxDepth := 3
	claimBuilder := faulttest.NewAlphabetClaimBuilder(t, maxDepth)

	// Last accessible leaf is the second last trace index
	// The root node is used for the last trace index and can only be attacked.
	lastLeafTraceIndex := big.NewInt(1<<maxDepth - 2)
	lastLeafTraceIndexPlusOne := big.NewInt(1<<maxDepth - 1)
	ctx := context.Background()

	tests := []struct {
		name                string
		agreeWithOutputRoot bool
		expectedErr         error
		expectAttack        bool
		expectPreState      []byte
		expectProofData     []byte
		expectedOracleData  *types.PreimageOracleData
		setupGame           func(builder *faulttest.GameBuilder)
	}{
		{
			name:               "AttackFirstTraceIndex",
			expectAttack:       true,
			expectPreState:     claimBuilder.CorrectPreState(common.Big0),
			expectProofData:    claimBuilder.CorrectProofData(common.Big0),
			expectedOracleData: claimBuilder.CorrectOracleData(common.Big0),
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					Attack(common.Hash{0xaa}).
					AttackCorrect().
					Attack(common.Hash{0xbb})
			},
		},
		{
			name:               "DefendFirstTraceIndex",
			expectAttack:       false,
			expectPreState:     claimBuilder.CorrectPreState(big.NewInt(1)),
			expectProofData:    claimBuilder.CorrectProofData(big.NewInt(1)),
			expectedOracleData: claimBuilder.CorrectOracleData(big.NewInt(1)),
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					Attack(common.Hash{0xaa}).
					AttackCorrect().
					AttackCorrect()
			},
		},
		{
			name:               "AttackMiddleTraceIndex",
			expectAttack:       true,
			expectPreState:     claimBuilder.CorrectPreState(big.NewInt(4)),
			expectProofData:    claimBuilder.CorrectProofData(big.NewInt(4)),
			expectedOracleData: claimBuilder.CorrectOracleData(big.NewInt(4)),
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().
					DefendCorrect().
					Attack(common.Hash{0xaa})
			},
		},
		{
			name:               "DefendMiddleTraceIndex",
			expectAttack:       false,
			expectPreState:     claimBuilder.CorrectPreState(big.NewInt(5)),
			expectProofData:    claimBuilder.CorrectProofData(big.NewInt(5)),
			expectedOracleData: claimBuilder.CorrectOracleData(big.NewInt(5)),
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().
					DefendCorrect().
					AttackCorrect()
			},
		},
		{
			name:               "AttackLastTraceIndex",
			expectAttack:       true,
			expectPreState:     claimBuilder.CorrectPreState(lastLeafTraceIndex),
			expectProofData:    claimBuilder.CorrectProofData(lastLeafTraceIndex),
			expectedOracleData: claimBuilder.CorrectOracleData(lastLeafTraceIndex),
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().
					DefendCorrect().
					Defend(common.Hash{0xaa})
			},
		},
		{
			name:               "DefendLastTraceIndex",
			expectAttack:       false,
			expectPreState:     claimBuilder.CorrectPreState(lastLeafTraceIndexPlusOne),
			expectProofData:    claimBuilder.CorrectProofData(lastLeafTraceIndexPlusOne),
			expectedOracleData: claimBuilder.CorrectOracleData(lastLeafTraceIndexPlusOne),
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().
					DefendCorrect().
					DefendCorrect()
			},
		},
		{
			name: "CannotStepNonLeaf",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().AttackCorrect().AttackCorrect()
			},
			expectedErr:         ErrStepNonLeafNode,
			agreeWithOutputRoot: true,
		},
		{
			name: "CannotStepAgreedNode",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().
					Attack(common.Hash{0xaa}).
					AttackCorrect()
			},
			expectedErr:         ErrStepIgnoreInvalidPath,
			agreeWithOutputRoot: true,
		},
		{
			name: "CannotStepInvalidPath",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					Attack(common.Hash{0xaa}).
					Attack(common.Hash{0xbb}).
					Attack(common.Hash{0xcc})
			},
			expectedErr:         ErrStepIgnoreInvalidPath,
			agreeWithOutputRoot: true,
		},
		{
			name:               "CannotStepNearlyValidPath",
			expectAttack:       true,
			expectPreState:     claimBuilder.CorrectPreState(big.NewInt(4)),
			expectProofData:    claimBuilder.CorrectProofData(big.NewInt(4)),
			expectedOracleData: claimBuilder.CorrectOracleData(big.NewInt(4)),
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					AttackCorrect().
					DefendCorrect().
					DefendCorrect()
			},
			expectedErr:         ErrStepIgnoreInvalidPath,
			agreeWithOutputRoot: true,
		},
	}

	for _, tableTest := range tests {
		tableTest := tableTest
		t.Run(tableTest.name, func(t *testing.T) {
			builder := claimBuilder.GameBuilder(!tableTest.agreeWithOutputRoot)
			tableTest.setupGame(builder)
			alphabetSolver := newClaimSolver(maxDepth, trace.NewSimpleTraceAccessor(claimBuilder.CorrectTraceProvider()))
			game := builder.Game
			claims := game.Claims()
			lastClaim := claims[len(claims)-1]
			step, err := alphabetSolver.AttemptStep(ctx, game, lastClaim)
			if tableTest.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, lastClaim, step.LeafClaim)
				require.Equal(t, tableTest.expectAttack, step.IsAttack)
				require.Equal(t, tableTest.expectPreState, step.PreState)
				require.Equal(t, tableTest.expectProofData, step.ProofData)
				require.Equal(t, tableTest.expectedOracleData.IsLocal, step.OracleData.IsLocal)
				require.Equal(t, tableTest.expectedOracleData.OracleKey, step.OracleData.OracleKey)
				require.Equal(t, tableTest.expectedOracleData.OracleData, step.OracleData.OracleData)
				require.Equal(t, tableTest.expectedOracleData.OracleOffset, step.OracleData.OracleOffset)
			} else {
				require.ErrorIs(t, err, tableTest.expectedErr)
				require.Equal(t, StepData{}, step)
			}
		})
	}
}
