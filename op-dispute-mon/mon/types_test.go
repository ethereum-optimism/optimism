package mon

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/stretchr/testify/require"
)

func TestStatusBatch_Add(t *testing.T) {
	statusExpectations := []struct {
		status types.GameStatus
		create func(int) statusBatch
	}{
		{
			status: types.GameStatusInProgress,
			create: func(inProgress int) statusBatch {
				return statusBatch{inProgress, 0, 0}
			},
		},
		{
			status: types.GameStatusDefenderWon,
			create: func(defenderWon int) statusBatch {
				return statusBatch{0, defenderWon, 0}
			},
		},
		{
			status: types.GameStatusChallengerWon,
			create: func(challengerWon int) statusBatch {
				return statusBatch{0, 0, challengerWon}
			},
		},
	}

	type test struct {
		name        string
		status      types.GameStatus
		invocations int
		expected    statusBatch
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
			s := statusBatch{}
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
		create func(int, bool) detectionBatch
	}{
		{
			status: types.GameStatusInProgress,
			create: func(inProgress int, _ bool) detectionBatch {
				return detectionBatch{inProgress, 0, 0, 0, 0}
			},
		},
		{
			status: types.GameStatusDefenderWon,
			create: func(defenderWon int, agree bool) detectionBatch {
				if agree {
					return detectionBatch{0, defenderWon, 0, 0, 0}
				}
				return detectionBatch{0, 0, defenderWon, 0, 0}
			},
		},
		{
			status: types.GameStatusChallengerWon,
			create: func(challengerWon int, agree bool) detectionBatch {
				if agree {
					return detectionBatch{0, 0, 0, challengerWon, 0}
				}
				return detectionBatch{0, 0, 0, 0, challengerWon}
			},
		},
	}

	type test struct {
		name        string
		status      types.GameStatus
		agree       bool
		invocations int
		expected    detectionBatch
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
			d := detectionBatch{}
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
		merge    detectionBatch
		expected detectionBatch
	}

	tests := []test{
		{
			name:     "Empty",
			merge:    detectionBatch{},
			expected: detectionBatch{},
		},
		{
			name:     "InProgress",
			merge:    detectionBatch{1, 0, 0, 0, 0},
			expected: detectionBatch{1, 0, 0, 0, 0},
		},
		{
			name:     "AgreeDefenderWins",
			merge:    detectionBatch{0, 1, 0, 0, 0},
			expected: detectionBatch{0, 1, 0, 0, 0},
		},
		{
			name:     "DisagreeDefenderWins",
			merge:    detectionBatch{0, 0, 1, 0, 0},
			expected: detectionBatch{0, 0, 1, 0, 0},
		},
		{
			name:     "AgreeChallengerWins",
			merge:    detectionBatch{0, 0, 0, 1, 0},
			expected: detectionBatch{0, 0, 0, 1, 0},
		},
		{
			name:     "DisagreeChallengerWins",
			merge:    detectionBatch{0, 0, 0, 0, 1},
			expected: detectionBatch{0, 0, 0, 0, 1},
		},
		{
			name:     "All",
			merge:    detectionBatch{1, 1, 1, 1, 1},
			expected: detectionBatch{1, 1, 1, 1, 1},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			d := detectionBatch{}
			d.Merge(test.merge)
			require.Equal(t, test.expected, d)
		})
	}
}
