package solver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/solver"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/test"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/stretchr/testify/require"
)

func TestNextMove(t *testing.T) {
	maxDepth := 4
	builder := test.NewAlphabetClaimBuilder(t, maxDepth)
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
			expectedErr: types.ErrGameDepthReached,
		},
		{
			name:        "ErrorWhenClaimIsLeaf_Incorrect",
			claim:       builder.CreateLeafClaim(6, false),
			expectedErr: types.ErrGameDepthReached,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			solver := solver.NewSolver(maxDepth, builder.CorrectTraceProvider())
			move, err := solver.NextMove(context.Background(), test.claim, test.agreeWithLevel)
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
	builder := test.NewAlphabetClaimBuilder(t, maxDepth)

	// Last accessible leaf is the second last trace index
	// The root node is used for the last trace index and can only be attacked.
	lastLeafTraceIndex := uint64(1<<maxDepth - 2)

	errProvider := errors.New("provider error")

	ctx := context.Background()

	preState, err := builder.CorrectTraceProvider().AbsolutePreState(ctx)
	require.NoError(t, err)

	tests := []struct {
		name               string
		claim              types.Claim
		agreeWithLevel     bool
		expectedErr        error
		expectAttack       bool
		expectPreState     []byte
		expectProofData    []byte
		expectedLocal      bool
		expectedOracleKey  []byte
		expectedOracleData []byte
	}{
		{
			name:               "AttackFirstTraceIndex",
			claim:              builder.CreateLeafClaim(0, false),
			expectAttack:       true,
			expectPreState:     preState,
			expectProofData:    nil,
			expectedOracleKey:  []byte{byte(0)},
			expectedOracleData: []byte{byte(0)},
		},
		{
			name:               "DefendFirstTraceIndex",
			claim:              builder.CreateLeafClaim(0, true),
			expectAttack:       false,
			expectPreState:     builder.CorrectPreState(0),
			expectProofData:    builder.CorrectProofData(0),
			expectedOracleKey:  []byte{byte(0)},
			expectedOracleData: []byte{byte(0)},
		},
		{
			name:               "AttackMiddleTraceIndex",
			claim:              builder.CreateLeafClaim(4, false),
			expectAttack:       true,
			expectPreState:     builder.CorrectPreState(3),
			expectProofData:    builder.CorrectProofData(3),
			expectedOracleKey:  []byte{byte(3)},
			expectedOracleData: []byte{byte(3)},
		},
		{
			name:               "DefendMiddleTraceIndex",
			claim:              builder.CreateLeafClaim(4, true),
			expectAttack:       false,
			expectPreState:     builder.CorrectPreState(4),
			expectProofData:    builder.CorrectProofData(4),
			expectedOracleKey:  []byte{byte(4)},
			expectedOracleData: []byte{byte(4)},
		},
		{
			name:               "AttackLastTraceIndex",
			claim:              builder.CreateLeafClaim(lastLeafTraceIndex, false),
			expectAttack:       true,
			expectPreState:     builder.CorrectPreState(lastLeafTraceIndex - 1),
			expectProofData:    builder.CorrectProofData(lastLeafTraceIndex - 1),
			expectedOracleKey:  []byte{byte(5)},
			expectedOracleData: []byte{byte(5)},
		},
		{
			name:               "DefendLastTraceIndex",
			claim:              builder.CreateLeafClaim(lastLeafTraceIndex, true),
			expectAttack:       false,
			expectPreState:     builder.CorrectPreState(lastLeafTraceIndex),
			expectProofData:    builder.CorrectProofData(lastLeafTraceIndex),
			expectedOracleKey:  []byte{byte(6)},
			expectedOracleData: []byte{byte(6)},
		},
		{
			name:        "CannotStepNonLeaf",
			claim:       builder.Seq(false).Attack(false).Get(),
			expectedErr: solver.ErrStepNonLeafNode,
		},
		{
			name:           "CannotStepAgreedNode",
			claim:          builder.Seq(false).Attack(false).Get(),
			agreeWithLevel: true,
			expectedErr:    solver.ErrStepNonLeafNode,
		},
		{
			name:           "CannotStepAgreedNode",
			claim:          builder.Seq(false).Attack(false).Get(),
			agreeWithLevel: true,
			expectedErr:    solver.ErrStepNonLeafNode,
		},
		{
			name:               "AttackLocalOracleData",
			claim:              builder.Seq(false).Attack(false).Attack(true).Defend(false).Get(),
			expectAttack:       true,
			agreeWithLevel:     false,
			expectPreState:     builder.CorrectPreState(1),
			expectProofData:    builder.CorrectProofData(1),
			expectedLocal:      true,
			expectedOracleKey:  []byte{0x01},
			expectedOracleData: []byte{0x01},
			expectedErr:        nil,
		},
		{
			name:           "AttackStepOracleError",
			claim:          builder.Seq(false).Attack(false).Attack(false).Attack(false).Get(),
			agreeWithLevel: false,
			expectedErr:    errProvider,
		},
	}

	for _, tableTest := range tests {
		tableTest := tableTest
		t.Run(tableTest.name, func(t *testing.T) {
			alphabetProvider := test.NewAlphabetWithProofProvider(t, maxDepth, nil)
			if errors.Is(tableTest.expectedErr, errProvider) {
				alphabetProvider = test.NewAlphabetWithProofProvider(t, maxDepth, errProvider)
			}
			builder = test.NewClaimBuilder(t, maxDepth, alphabetProvider)
			alphabetSolver := solver.NewSolver(maxDepth, builder.CorrectTraceProvider())
			step, err := alphabetSolver.AttemptStep(ctx, tableTest.claim, tableTest.agreeWithLevel)
			if tableTest.expectedErr == nil {
				require.NoError(t, err)
				require.Equal(t, tableTest.claim, step.LeafClaim)
				require.Equal(t, tableTest.expectAttack, step.IsAttack)
				require.Equal(t, tableTest.expectPreState, step.PreState)
				require.Equal(t, tableTest.expectProofData, step.ProofData)
				require.Equal(t, tableTest.expectedLocal, step.OracleData.IsLocal)
				require.Equal(t, tableTest.expectedOracleKey, step.OracleData.OracleKey)
				require.Equal(t, tableTest.expectedOracleData, step.OracleData.OracleData)
			} else {
				require.ErrorIs(t, err, tableTest.expectedErr)
				require.Equal(t, solver.StepData{}, step)
			}
		})
	}
}
