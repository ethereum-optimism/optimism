package fault

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMSBIndex(t *testing.T) {
	tests := []struct {
		input    uint64
		expected int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{4, 2},
		{8, 3},
		{16, 4},
		{255, 7},
		{1024, 10},
		{18446744073709551615, 63},
	}

	for _, test := range tests {
		result := MSBIndex(test.input)
		if result != test.expected {
			t.Errorf("MSBIndex(%d) expected %d, but got %d", test.input, test.expected, result)
		}
	}

}

func TestGINConvessions(t *testing.T) {
	tests := []struct {
		index    uint64
		position Position
	}{
		{
			index:    1,
			position: Position{Depth: 0, IndexAtDepth: 0},
		},
		{
			index:    0b10,
			position: Position{Depth: 1, IndexAtDepth: 0},
		},
		{
			index:    0b11,
			position: Position{Depth: 1, IndexAtDepth: 1},
		},
		{
			index:    0b100,
			position: Position{Depth: 2, IndexAtDepth: 0},
		},
		{
			index:    0b101,
			position: Position{Depth: 2, IndexAtDepth: 1},
		},
		{
			index:    0b110,
			position: Position{Depth: 2, IndexAtDepth: 2},
		},
		{
			index:    0b111,
			position: Position{Depth: 2, IndexAtDepth: 3},
		},
		{
			index:    0b1010,
			position: Position{Depth: 3, IndexAtDepth: 2},
		},
		{
			index:    0b1100,
			position: Position{Depth: 3, IndexAtDepth: 4},
		},
	}
	for _, test := range tests {
		from := FromGIN(test.index)
		require.Equal(t, test.position, from)
		to := test.position.ToGIN()
		require.Equal(t, test.index, to)
	}
}

func TestTraceIndex(t *testing.T) {
	// Note: Only left children are are valid positions after a move.
	tests := []struct {
		pos Position
		idx int
	}{
		{
			pos: Position{Depth: 0, IndexAtDepth: 0},
			idx: 7,
		},
		{
			pos: Position{Depth: 1, IndexAtDepth: 0},
			idx: 3,
		},
		{
			pos: Position{Depth: 2, IndexAtDepth: 0},
			idx: 1,
		},
		{
			pos: Position{Depth: 2, IndexAtDepth: 2},
			idx: 5,
		},
		{
			pos: Position{Depth: 3, IndexAtDepth: 0},
			idx: 0,
		},
		{
			pos: Position{Depth: 3, IndexAtDepth: 2},
			idx: 2,
		},
		{
			pos: Position{Depth: 3, IndexAtDepth: 4},
			idx: 4,
		},
		{
			pos: Position{Depth: 3, IndexAtDepth: 6},
			idx: 6,
		},
	}
	for _, test := range tests {
		result := test.pos.TraceIndex(3)
		require.Equal(t, test.idx, result)
	}
}
