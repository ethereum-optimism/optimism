package config

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-supernode/supernode/types"
	"github.com/stretchr/testify/require"
)

func TestAppliesTo(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		chainID  types.ChainID
		expected bool
	}{
		{
			name:     "global flag applies to any chain",
			flagName: "vn.all.rollup.config",
			chainID:  types.ChainID(1),
			expected: true,
		},
		{
			name:     "global flag applies to any chain",
			flagName: "vn.all.l1.rpc",
			chainID:  types.ChainID(999),
			expected: true,
		},
		{
			name:     "chain-specific flag applies to matching chain",
			flagName: "vn.1.rollup.config",
			chainID:  types.ChainID(1),
			expected: true,
		},
		{
			name:     "chain-specific flag does not apply to different chain",
			flagName: "vn.1.rollup.config",
			chainID:  types.ChainID(2),
			expected: false,
		},
		{
			name:     "non-vn flag does not apply",
			flagName: "rollup.config",
			chainID:  types.ChainID(1),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appliesTo(tt.flagName, tt.chainID)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestStripVNPrefix(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		chainID  types.ChainID
		expected string
	}{
		{
			name:     "strip global prefix",
			flagName: "vn.all.rollup.config",
			chainID:  types.ChainID(1),
			expected: "rollup.config",
		},
		{
			name:     "strip chain-specific prefix",
			flagName: "vn.1.rollup.config",
			chainID:  types.ChainID(1),
			expected: "rollup.config",
		},
		{
			name:     "chain-specific prefix for different chain returns empty",
			flagName: "vn.1.rollup.config",
			chainID:  types.ChainID(2),
			expected: "",
		},
		{
			name:     "non-vn flag returns empty",
			flagName: "rollup.config",
			chainID:  types.ChainID(1),
			expected: "",
		},
		{
			name:     "strip global prefix with complex flag name",
			flagName: "vn.all.p2p.priv.path",
			chainID:  types.ChainID(999),
			expected: "p2p.priv.path",
		},
		{
			name:     "strip chain-specific prefix for multi-digit chain",
			flagName: "vn.1234.l1.rpc",
			chainID:  types.ChainID(1234),
			expected: "l1.rpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripVNPrefix(tt.flagName, tt.chainID)
			require.Equal(t, tt.expected, result)
		})
	}
}
