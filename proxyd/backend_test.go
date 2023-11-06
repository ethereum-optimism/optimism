package proxyd

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStripXFF(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{"1.2.3, 4.5.6, 7.8.9", "1.2.3"},
		{"1.2.3,4.5.6", "1.2.3"},
		{" 1.2.3 , 4.5.6 ", "1.2.3"},
	}

	for _, test := range tests {
		actual := stripXFF(test.in)
		assert.Equal(t, test.out, actual)
	}
}

func TestCreateBackendGroup(t *testing.T) {
	unweightedOne := &Backend{
		Name:   "one",
		weight: 0,
	}

	unweightedTwo := &Backend{
		Name:   "two",
		weight: 0,
	}

	weightedOne := &Backend{
		Name:   "one",
		weight: 1,
	}

	weightedTwo := &Backend{
		Name:   "two",
		weight: 1,
	}

	tests := []struct {
		name            string
		backends        []*Backend
		weightedRouting bool
		expectError     bool
	}{
		{
			name:            "weighting disabled",
			backends:        []*Backend{unweightedOne, unweightedTwo},
			weightedRouting: false,
			expectError:     false,
		},
		{
			name:            "weighting enabled -- all nodes have weight",
			backends:        []*Backend{weightedOne, weightedTwo},
			weightedRouting: true,
			expectError:     false,
		},
		{
			name:            "weighting enabled -- some nodes have weight",
			backends:        []*Backend{weightedOne, unweightedTwo},
			weightedRouting: true,
			expectError:     false,
		},
		{
			name:            "weighting enabled -- no nodes have weight",
			backends:        []*Backend{unweightedOne, unweightedTwo},
			weightedRouting: true,
			expectError:     true,
		},
	}

	for _, test := range tests {
		result, err := NewBackendGroup(test.name, test.backends, test.weightedRouting)

		if test.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, test.name, result.Name)
			assert.Equal(t, test.backends, test.backends)
			assert.Equal(t, test.weightedRouting, test.weightedRouting)
		}
	}

}

func TestMoveIndexToStart(t *testing.T) {
	one := &Backend{
		Name: "one",
	}

	two := &Backend{
		Name: "two",
	}

	three := &Backend{
		Name: "three",
	}

	tests := []struct {
		choice *Backend
		input  []*Backend
		output []*Backend
	}{
		{
			choice: one,
			input:  []*Backend{one, two, three},
			output: []*Backend{one, two, three},
		},
		{
			choice: two,
			input:  []*Backend{one, two, three},
			output: []*Backend{two, one, three},
		},
		{
			choice: three,
			input:  []*Backend{one, two},
			output: []*Backend{one, two},
		},
		{
			choice: one,
			input:  []*Backend{one},
			output: []*Backend{one},
		},
	}

	for _, test := range tests {
		result := moveBackendToStart(test.choice, test.input)
		assert.Equal(t, test.output, result)
	}
}
