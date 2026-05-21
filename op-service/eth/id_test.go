package eth

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestL2BlockRefAdvances(t *testing.T) {
	current := L2BlockRef{Hash: common.Hash{0x01}, Number: 1}
	genesis := L2BlockRef{Hash: common.Hash{0x02}, Number: 0}
	next := L2BlockRef{Hash: common.Hash{0x03}, Number: 2}

	tests := []struct {
		name      string
		candidate L2BlockRef
		current   L2BlockRef
		expected  bool
	}{
		{
			name:      "zero candidate does not advance zero current",
			candidate: L2BlockRef{},
			current:   L2BlockRef{},
			expected:  false,
		},
		{
			name:      "zero candidate does not advance initialized current",
			candidate: L2BlockRef{},
			current:   current,
			expected:  false,
		},
		{
			name:      "real genesis advances zero current",
			candidate: genesis,
			current:   L2BlockRef{},
			expected:  true,
		},
		{
			name:      "higher number advances initialized current",
			candidate: next,
			current:   current,
			expected:  true,
		},
		{
			name:      "same number does not advance initialized current",
			candidate: L2BlockRef{Hash: common.Hash{0x04}, Number: current.Number},
			current:   current,
			expected:  false,
		},
		{
			name:      "lower number does not advance initialized current",
			candidate: genesis,
			current:   current,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, L2BlockRefAdvances(tt.current, tt.candidate))
		})
	}
}
