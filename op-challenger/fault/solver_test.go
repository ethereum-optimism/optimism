package fault

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func alphabetClaim(index uint64, letter string) common.Hash {
	return crypto.Keccak256Hash(BuildAlphabetPreimage(index, letter))
}

// TestSolver_NextMove_Opponent tests the [Solver] NextMove function
// with an [fault.AlphabetProvider] as the [TraceProvider].
func TestSolver_NextMove_Opponent(t *testing.T) {
	// Construct the solver.
	maxDepth := 3
	canonicalProvider := NewAlphabetProvider("abcdefgh", uint64(maxDepth))
	solver := NewSolver(maxDepth, canonicalProvider)

	// The following claims are created using the state: "abcdexyz".
	// The responses are the responses we expect from the solver.
	indices := []struct {
		claim    types.Claim
		response types.ClaimData
	}{
		{
			types.Claim{
				ClaimData: types.ClaimData{
					Value:    alphabetClaim(7, "z"),
					Position: types.NewPosition(0, 0),
				},
				// Root claim has no parent
			},
			types.ClaimData{
				Value:    alphabetClaim(3, "d"),
				Position: types.NewPosition(1, 0),
			},
		},
		{
			types.Claim{
				ClaimData: types.ClaimData{
					Value:    alphabetClaim(3, "d"),
					Position: types.NewPosition(1, 0),
				},
				Parent: types.ClaimData{
					Value:    alphabetClaim(7, "h"),
					Position: types.NewPosition(0, 0),
				},
			},
			types.ClaimData{
				Value:    alphabetClaim(5, "f"),
				Position: types.NewPosition(2, 2),
			},
		},
		{
			types.Claim{
				ClaimData: types.ClaimData{
					Value:    alphabetClaim(5, "x"),
					Position: types.NewPosition(2, 2),
				},
				Parent: types.ClaimData{
					Value:    alphabetClaim(7, "h"),
					Position: types.NewPosition(1, 1),
				},
			},
			types.ClaimData{
				Value:    alphabetClaim(4, "e"),
				Position: types.NewPosition(3, 4),
			},
		},
	}

	for _, test := range indices {
		res, err := solver.NextMove(test.claim, false)
		require.NoError(t, err)
		require.Equal(t, test.response, res.ClaimData)
	}
}

func TestNoMoveAgainstOwnLevel(t *testing.T) {
	maxDepth := 3
	mallory := NewAlphabetProvider("abcdepqr", uint64(maxDepth))
	solver := NewSolver(maxDepth, mallory)

	claim := types.Claim{
		ClaimData: types.ClaimData{
			Value:    alphabetClaim(7, "z"),
			Position: types.NewPosition(0, 0),
		},
		// Root claim has no parent
	}

	move, err := solver.NextMove(claim, true)
	require.Nil(t, move)
	require.Nil(t, err)
}

func TestAttemptStep(t *testing.T) {
	maxDepth := 3
	canonicalProvider := &alphabetWithProofProvider{NewAlphabetProvider("abcdefgh", uint64(maxDepth))}
	solver := NewSolver(maxDepth, canonicalProvider)
	_, _, middle, bottom := createTestClaims()

	zero := types.Claim{
		ClaimData: types.ClaimData{
			// Zero value is a purposely disagree with claim value "a"
			Position: types.NewPosition(3, 0),
		},
	}

	step, err := solver.AttemptStep(bottom, false)
	require.NoError(t, err)
	require.Equal(t, bottom, step.LeafClaim)
	require.True(t, step.IsAttack)
	require.Equal(t, step.PreState, BuildAlphabetPreimage(3, "d"))
	require.Equal(t, step.ProofData, []byte{3})

	_, err = solver.AttemptStep(middle, false)
	require.Error(t, err)

	step, err = solver.AttemptStep(zero, false)
	require.NoError(t, err)
	require.Equal(t, zero, step.LeafClaim)
	require.True(t, step.IsAttack)
	require.Equal(t, canonicalProvider.AbsolutePreState(), step.PreState)
}

func TestAttempStep_AgreeWithClaimLevel_Fails(t *testing.T) {
	maxDepth := 3
	canonicalProvider := NewAlphabetProvider("abcdefgh", uint64(maxDepth))
	solver := NewSolver(maxDepth, canonicalProvider)
	_, _, middle, _ := createTestClaims()

	step, err := solver.AttemptStep(middle, true)
	require.Error(t, err)
	require.Equal(t, StepData{}, step)
}

type alphabetWithProofProvider struct {
	*AlphabetProvider
}

func (a *alphabetWithProofProvider) GetPreimage(i uint64) ([]byte, []byte, error) {
	preimage, _, err := a.AlphabetProvider.GetPreimage(i)
	if err != nil {
		return nil, nil, err
	}
	return preimage, []byte{byte(i)}, nil
}

func createTestClaims() (types.Claim, types.Claim, types.Claim, types.Claim) {
	// root & middle are from the trace "abcdexyz"
	// top & bottom are from the trace  "abcdefgh"
	root := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000077a"),
			Position: types.NewPosition(0, 0),
		},
		// Root claim has no parent
	}
	top := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
			Position: types.NewPosition(1, 0),
		},
		Parent: root.ClaimData,
	}
	middle := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000578"),
			Position: types.NewPosition(2, 2),
		},
		Parent: top.ClaimData,
	}

	bottom := types.Claim{
		ClaimData: types.ClaimData{
			Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
			Position: types.NewPosition(3, 4),
		},
		Parent: middle.ClaimData,
	}

	return root, top, middle, bottom
}
