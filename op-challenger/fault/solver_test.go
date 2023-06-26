package fault

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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
		traceIndex int
		claim      Claim
		response   ClaimData
	}{
		{
			3,
			Claim{
				ClaimData: ClaimData{
					Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000364"),
					Position: NewPosition(1, 0),
				},
				Parent: ClaimData{
					Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
					Position: NewPosition(0, 0),
				},
			},
			ClaimData{
				Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000566"),
				Position: NewPosition(2, 2),
			},
		},
		{
			5,
			Claim{
				ClaimData: ClaimData{
					Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000578"),
					Position: NewPosition(2, 2),
				},
				Parent: ClaimData{
					Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000768"),
					Position: NewPosition(1, 1),
				},
			},
			ClaimData{
				Value:    common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000465"),
				Position: NewPosition(3, 4),
			},
		},
	}

	for _, test := range indices {
		res, err := solver.NextMove(test.claim)
		require.NoError(t, err)
		require.Equal(t, test.response, res.ClaimData)
	}
}
