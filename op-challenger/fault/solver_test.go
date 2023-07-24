package fault

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/stretchr/testify/require"
)

func TestNextMove(t *testing.T) {
	maxDepth := 4
	builder := NewClaimBuilder(t, maxDepth)
	tests := []struct {
		name           string
		claim          types.Claim
		agreeWithLevel bool
		expectedErr    error
		expectedMove   func(claim types.Claim, correct bool) types.Claim
	}{
		{
			name:           "AgreeWithLevel_CorrectRoot",
			claim:          builder.CreateRootClaim(true),
			agreeWithLevel: true,
		},
		{
			name:           "AgreeWithLevel_IncorrectRoot",
			claim:          builder.CreateRootClaim(false),
			agreeWithLevel: true,
		},
		{
			name:           "AgreeWithLevel_EvenDepth",
			claim:          builder.Seq(false).Attack(false).Get(),
			agreeWithLevel: true,
		},
		{
			name:           "AgreeWithLevel_OddDepth",
			claim:          builder.Seq(false).Attack(false).Defend(false).Get(),
			agreeWithLevel: true,
		},
		{
			name:  "Root_CorrectValue",
			claim: builder.CreateRootClaim(true),
		},
		{
			name:         "Root_IncorrectValue",
			claim:        builder.CreateRootClaim(false),
			expectedMove: builder.AttackClaim,
		},
		{
			name:         "NonRoot_AgreeWithParentAndClaim",
			claim:        builder.Seq(true).Attack(true).Get(),
			expectedMove: builder.DefendClaim,
		},
		{
			name:         "NonRoot_AgreeWithParentDisagreeWithClaim",
			claim:        builder.Seq(true).Attack(false).Get(),
			expectedMove: builder.AttackClaim,
		},
		{
			name:         "NonRoot_DisagreeWithParentAgreeWithClaim",
			claim:        builder.Seq(false).Attack(true).Get(),
			expectedMove: builder.DefendClaim,
		},
		{
			name:         "NonRoot_DisagreeWithParentAndClaim",
			claim:        builder.Seq(false).Attack(false).Get(),
			expectedMove: builder.AttackClaim,
		},
		{
			name:        "ErrorWhenClaimIsLeaf_Correct",
			claim:       builder.CreateLeafClaim(4, true),
			expectedErr: ErrGameDepthReached,
		},
		{
			name:        "ErrorWhenClaimIsLeaf_Incorrect",
			claim:       builder.CreateLeafClaim(6, false),
			expectedErr: ErrGameDepthReached,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			solver := NewSolver(maxDepth, builder.CorrectTraceProvider())
			move, err := solver.NextMove(test.claim, test.agreeWithLevel)
			if test.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, test.expectedErr)
			}
			if test.expectedMove == nil {
				require.Nil(t, move)
			} else {
				expected := test.expectedMove(test.claim, true)
				require.Equal(t, &expected, move)
			}
		})
	}
}

func TestAttemptStep(t *testing.T) {
	maxDepth := 3
	builder := NewClaimBuilder(t, maxDepth)
	solver := NewSolver(maxDepth, builder.CorrectTraceProvider())

	// Last accessible leaf is the second last trace index
	// The root node is used for the last trace index and can only be attacked.
	lastLeafTraceIndex := uint64(1<<maxDepth - 2)

	tests := []struct {
		name            string
		claim           types.Claim
		agreeWithLevel  bool
		expectedErr     error
		expectAttack    bool
		expectPreState  []byte
		expectProofData []byte
	}{
		{
			name:            "AttackFirstTraceIndex",
			claim:           builder.CreateLeafClaim(0, false),
			expectAttack:    true,
			expectPreState:  builder.CorrectTraceProvider().AbsolutePreState(),
			expectProofData: nil,
		},
		{
			name:            "DefendFirstTraceIndex",
			claim:           builder.CreateLeafClaim(0, true),
			expectAttack:    false,
			expectPreState:  builder.CorrectPreState(0),
			expectProofData: builder.CorrectProofData(0),
		},
		{
			name:            "AttackMiddleTraceIndex",
			claim:           builder.CreateLeafClaim(4, false),
			expectAttack:    true,
			expectPreState:  builder.CorrectPreState(3),
			expectProofData: builder.CorrectProofData(3),
		},
		{
			name:            "DefendMiddleTraceIndex",
			claim:           builder.CreateLeafClaim(4, true),
			expectAttack:    false,
			expectPreState:  builder.CorrectPreState(4),
			expectProofData: builder.CorrectProofData(4),
		},
		{
			name:            "AttackLastTraceIndex",
			claim:           builder.CreateLeafClaim(lastLeafTraceIndex, false),
			expectAttack:    true,
			expectPreState:  builder.CorrectPreState(lastLeafTraceIndex - 1),
			expectProofData: builder.CorrectProofData(lastLeafTraceIndex - 1),
		},
		{
			name:            "DefendLastTraceIndex",
			claim:           builder.CreateLeafClaim(lastLeafTraceIndex, true),
			expectAttack:    false,
			expectPreState:  builder.CorrectPreState(lastLeafTraceIndex),
			expectProofData: builder.CorrectProofData(lastLeafTraceIndex),
		},
		{
			name:        "CannotStepNonLeaf",
			claim:       builder.Seq(false).Attack(false).Get(),
			expectedErr: ErrStepNonLeafNode,
		},
		{
			name:           "CannotStepAgreedNode",
			claim:          builder.Seq(false).Attack(false).Get(),
			agreeWithLevel: true,
			expectedErr:    ErrStepNonLeafNode,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			fmt.Printf("%v\n", test.claim.Position.TraceIndex(maxDepth))
			step, err := solver.AttemptStep(test.claim, test.agreeWithLevel)
			if test.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, test.claim, step.LeafClaim)
				require.Equal(t, test.expectAttack, step.IsAttack)
				require.Equal(t, test.expectPreState, step.PreState)
				require.Equal(t, test.expectProofData, step.ProofData)
			} else {
				require.ErrorIs(t, err, test.expectedErr)
				require.Equal(t, StepData{}, step)
			}
		})
	}
}
