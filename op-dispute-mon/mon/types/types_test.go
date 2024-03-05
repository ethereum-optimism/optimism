package types

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/stretchr/testify/require"
)

func TestMaxValue(t *testing.T) {
	require.Equal(t, ResolvedBondAmount.String(), "340282366920938463463374607431768211455")
}

func TestStatusBatch_Add(t *testing.T) {
	statusExpectations := []struct {
		status types.GameStatus
		create func(int) StatusBatch
	}{
		{
			status: types.GameStatusInProgress,
			create: func(inProgress int) StatusBatch {
				return StatusBatch{inProgress, 0, 0}
			},
		},
		{
			status: types.GameStatusDefenderWon,
			create: func(defenderWon int) StatusBatch {
				return StatusBatch{0, defenderWon, 0}
			},
		},
		{
			status: types.GameStatusChallengerWon,
			create: func(challengerWon int) StatusBatch {
				return StatusBatch{0, 0, challengerWon}
			},
		},
	}

	type test struct {
		name        string
		status      types.GameStatus
		invocations int
		expected    StatusBatch
	}

	var tests []test
	for i := 0; i < 100; i++ {
		for _, exp := range statusExpectations {
			tests = append(tests, test{
				name:        fmt.Sprintf("Invocation-%d", i),
				status:      exp.status,
				invocations: i,
				expected:    exp.create(i),
			})
		}
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			s := StatusBatch{}
			for i := 0; i < test.invocations; i++ {
				s.Add(test.status)
			}
			require.Equal(t, test.expected, s)
		})
	}
}
