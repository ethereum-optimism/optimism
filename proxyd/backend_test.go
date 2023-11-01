package proxyd

import (
	"github.com/stretchr/testify/assert"
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

func TestMoveIndexToStart(t *testing.T) {
	backends := []*Backend{
		{
			Name: "node1",
		},
		{
			Name: "node1",
		},
		{
			Name: "node1",
		},
	}

	tests := []struct {
		index int
		out   []*Backend
	}{
		{
			index: 0,
			out: []*Backend{
				backends[0],
				backends[1],
				backends[2],
			},
		},
		{
			index: 1,
			out: []*Backend{
				backends[1],
				backends[0],
				backends[2],
			},
		},
		{
			index: 2,
			out: []*Backend{
				backends[2],
				backends[0],
				backends[1],
			},
		},
	}

	for _, test := range tests {
		result := moveIndexToStart(backends, test.index)
		assert.Equal(t, test.out, result)
	}
}
