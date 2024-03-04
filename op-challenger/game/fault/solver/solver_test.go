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
	maxDepth := types.Depth(3)
	startingL2BlockNumber := big.NewInt(0)
	claimBuilder := faulttest.NewAlphabetClaimBuilder(t, startingL2BlockNumber, maxDepth)

	// Last accessible leaf is the second last trace index
	// The root node is used for the last trace index and can only be attacked.
	lastLeafTraceIndex := big.NewInt(1<<maxDepth - 2)
	lastLeafTraceIndexPlusOne := big.NewInt(1<<maxDepth - 1)
	ctx := context.Background()

	tests := []struct {
		name                string
		agreeWithOutputRoot bool
		expectedErr         error
		expectNoStep        bool
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
					Attack(faulttest.WithValue(common.Hash{0xaa})).
					Attack().
					Attack(faulttest.WithValue(common.Hash{0xbb}))
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
					Attack(faulttest.WithValue(common.Hash{0xaa})).
					Attack().
					Attack()
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
					Attack().
					Defend().
					Attack(faulttest.WithValue(common.Hash{0xaa}))
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
					Attack().
					Defend().
					Attack()
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
					Attack().
					Defend().
					Defend(faulttest.WithValue(common.Hash{0xaa}))
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
					Attack().
					Defend().
					Defend()
			},
		},
		{
			name: "CannotStepNonLeaf",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().Attack().Attack()
			},
			expectedErr:         ErrStepNonLeafNode,
			agreeWithOutputRoot: true,
		},
		{
			name: "CannotStepAgreedNode",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					Attack().
					Attack(faulttest.WithValue(common.Hash{0xaa})).
					Attack()
			},
			expectNoStep:        true,
			agreeWithOutputRoot: true,
		},
		{
			name: "CannotStepInvalidPath",
			setupGame: func(builder *faulttest.GameBuilder) {
				builder.Seq().
					Attack(faulttest.WithValue(common.Hash{0xaa})).
					Attack(faulttest.WithValue(common.Hash{0xbb})).
					Attack(faulttest.WithValue(common.Hash{0xcc}))
			},
			expectNoStep:        true,
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
					Attack().
					Defend().
					Defend()
			},
			expectNoStep:        true,
			agreeWithOutputRoot: true,
		},
	}

	for _, tableTest := range tests {
		tableTest := tableTest
		t.Run(tableTest.name, func(t *testing.T) {
			builder := claimBuilder.GameBuilder(faulttest.WithInvalidValue(tableTest.agreeWithOutputRoot))
			tableTest.setupGame(builder)
			alphabetSolver := newClaimSolver(maxDepth, trace.NewSimpleTraceAccessor(claimBuilder.CorrectTraceProvider()))
			game := builder.Game
			claims := game.Claims()
			lastClaim := claims[len(claims)-1]
			agreedClaims := newHonestClaimTracker()
			if tableTest.agreeWithOutputRoot {
				agreedClaims.AddHonestClaim(types.Claim{}, claims[0])
			}
			if (lastClaim.Depth()%2 == 0) == tableTest.agreeWithOutputRoot {
				parentClaim := claims[lastClaim.ParentContractIndex]
				grandParentClaim := claims[parentClaim.ParentContractIndex]
				agreedClaims.AddHonestClaim(grandParentClaim, parentClaim)
			}
			step, err := alphabetSolver.AttemptStep(ctx, game, lastClaim, agreedClaims)
			require.ErrorIs(t, err, tableTest.expectedErr)
			if !tableTest.expectNoStep && tableTest.expectedErr == nil {
				require.NotNil(t, step)
				require.Equal(t, lastClaim, step.LeafClaim)
				require.Equal(t, tableTest.expectAttack, step.IsAttack)
				require.Equal(t, tableTest.expectPreState, step.PreState)
				require.Equal(t, tableTest.expectProofData, step.ProofData)
				require.Equal(t, tableTest.expectedOracleData.IsLocal, step.OracleData.IsLocal)
				require.Equal(t, tableTest.expectedOracleData.OracleKey, step.OracleData.OracleKey)
				require.Equal(t, tableTest.expectedOracleData.GetPreimageWithSize(), step.OracleData.GetPreimageWithSize())
				require.Equal(t, tableTest.expectedOracleData.OracleOffset, step.OracleData.OracleOffset)
			} else {
				require.Nil(t, step)
			}
		})
	}
}
