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

func TestDetectionBatch_Update(t *testing.T) {
	statusExpectations := []struct {
		status types.GameStatus
		create func(int, bool) DetectionBatch
	}{
		{
			status: types.GameStatusInProgress,
			create: func(inProgress int, _ bool) DetectionBatch {
				return DetectionBatch{inProgress, 0, 0, 0, 0}
			},
		},
		{
			status: types.GameStatusDefenderWon,
			create: func(defenderWon int, agree bool) DetectionBatch {
				if agree {
					return DetectionBatch{0, defenderWon, 0, 0, 0}
				}
				return DetectionBatch{0, 0, defenderWon, 0, 0}
			},
		},
		{
			status: types.GameStatusChallengerWon,
			create: func(challengerWon int, agree bool) DetectionBatch {
				if agree {
					return DetectionBatch{0, 0, 0, challengerWon, 0}
				}
				return DetectionBatch{0, 0, 0, 0, challengerWon}
			},
		},
	}

	type test struct {
		name        string
		status      types.GameStatus
		agree       bool
		invocations int
		expected    DetectionBatch
	}

	var tests []test
	for i := 0; i < 100; i++ {
		for _, exp := range statusExpectations {
			agree := i%2 == 0
			tests = append(tests, test{
				name:        fmt.Sprintf("Invocation-%d", i),
				status:      exp.status,
				agree:       agree,
				invocations: i,
				expected:    exp.create(i, agree),
			})
		}
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			d := DetectionBatch{}
			for i := 0; i < test.invocations; i++ {
				d.Update(test.status, test.agree)
			}
			require.Equal(t, test.expected, d)
		})
	}
}

func TestDetectionBatch_Merge(t *testing.T) {
	type test struct {
		name     string
		merge    DetectionBatch
		expected DetectionBatch
	}

	tests := []test{
		{
			name:     "Empty",
			merge:    DetectionBatch{},
			expected: DetectionBatch{},
		},
		{
			name:     "InProgress",
			merge:    DetectionBatch{1, 0, 0, 0, 0},
			expected: DetectionBatch{1, 0, 0, 0, 0},
		},
		{
			name:     "AgreeDefenderWins",
			merge:    DetectionBatch{0, 1, 0, 0, 0},
			expected: DetectionBatch{0, 1, 0, 0, 0},
		},
		{
			name:     "DisagreeDefenderWins",
			merge:    DetectionBatch{0, 0, 1, 0, 0},
			expected: DetectionBatch{0, 0, 1, 0, 0},
		},
		{
			name:     "AgreeChallengerWins",
			merge:    DetectionBatch{0, 0, 0, 1, 0},
			expected: DetectionBatch{0, 0, 0, 1, 0},
		},
		{
			name:     "DisagreeChallengerWins",
			merge:    DetectionBatch{0, 0, 0, 0, 1},
			expected: DetectionBatch{0, 0, 0, 0, 1},
		},
		{
			name:     "All",
			merge:    DetectionBatch{1, 1, 1, 1, 1},
			expected: DetectionBatch{1, 1, 1, 1, 1},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			d := DetectionBatch{}
			d.Merge(test.merge)
			require.Equal(t, test.expected, d)
		})
	}
}
