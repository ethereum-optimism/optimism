package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemovedAltDAEnvVarPrefixes(t *testing.T) {
	tests := []struct {
		name   string
		chains []uint64
		want   []string
	}{
		{
			name: "global only",
			want: []string{"OP_SUPERNODE_VN_ALL_ALTDA_"},
		},
		{
			name:   "global and configured chains",
			chains: []uint64{10, 8453},
			want: []string{
				"OP_SUPERNODE_VN_ALL_ALTDA_",
				"OP_SUPERNODE_VN_10_ALTDA_",
				"OP_SUPERNODE_VN_8453_ALTDA_",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, removedAltDAEnvVarPrefixes(tt.chains))
		})
	}
}

func TestRejectRemovedAltDAEnvVars(t *testing.T) {
	tests := []struct {
		name   string
		chains []uint64
		envVar string
	}{
		{
			name:   "global",
			envVar: "OP_SUPERNODE_VN_ALL_ALTDA_ENABLED",
		},
		{
			name:   "configured chain",
			chains: []uint64{10},
			envVar: "OP_SUPERNODE_VN_10_ALTDA_ENABLED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, "false")
			err := rejectRemovedAltDAEnvVars(tt.chains)
			require.ErrorContains(t, err, tt.envVar)
		})
	}
}
