package flags

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractVNFlags(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedVNFlags  VNFlagMap
		expectedFiltered []string
	}{
		{
			name: "single dash flags with values",
			args: []string{"app", "-vn.11155420.l1", "http://localhost:8545", "--sample", "test"},
			expectedVNFlags: VNFlagMap{
				"vn.11155420.l1": "http://localhost:8545",
			},
			expectedFiltered: []string{"app", "--sample", "test"},
		},
		{
			name: "double dash flags with equals",
			args: []string{"app", "--vn.all.l1=http://localhost:8545", "--sample", "test"},
			expectedVNFlags: VNFlagMap{
				"vn.all.l1": "http://localhost:8545",
			},
			expectedFiltered: []string{"app", "--sample", "test"},
		},
		{
			name: "mixed format",
			args: []string{"app", "--vn.123.l1=http://l1", "-vn.456.l2", "http://l2", "--chains", "123,456"},
			expectedVNFlags: VNFlagMap{
				"vn.123.l1": "http://l1",
				"vn.456.l2": "http://l2",
			},
			expectedFiltered: []string{"app", "--chains", "123,456"},
		},
		{
			name: "boolean flags",
			args: []string{"app", "--vn.all.flag", "--sample", "test"},
			expectedVNFlags: VNFlagMap{
				"vn.all.flag": "true",
			},
			expectedFiltered: []string{"app", "--sample", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vnFlags, filtered := ExtractVNFlags(tt.args)

			require.Equal(t, tt.expectedVNFlags, vnFlags, "vn flags should match")
			require.Equal(t, tt.expectedFiltered, filtered, "filtered args should match")
		})
	}
}

// TestVNFlagMapCheck tests the VNFlagMap.Check() validation
func TestVNFlagMapCheck(t *testing.T) {
	tests := []struct {
		name        string
		vnFlags     VNFlagMap
		expectedErr bool
		errContains string
	}{
		{
			name: "TC5.1: Valid vn flags pass validation",
			vnFlags: VNFlagMap{
				"vn.all.rollup.config": "/path/to/rollup.json",
				"vn.420.p2p.priv.path": "/path/to/key",
				"vn.10.some-flag":      "value",
			},
			expectedErr: false,
		},
		{
			name: "TC5.2: vn.all.l1 in map returns error",
			vnFlags: VNFlagMap{
				"vn.all.l1": "http://l1:8545",
			},
			expectedErr: true,
			errContains: "global l1 should be set by --l1",
		},
		{
			name: "TC5.3: vn.all.l1.beacon in map returns error",
			vnFlags: VNFlagMap{
				"vn.all.l1.beacon": "http://beacon:5051",
			},
			expectedErr: true,
			errContains: "global l1.beacon should be set by --l1.beacon",
		},
		{
			name: "TC5.4: Chain-specific l1/beacon flags are allowed",
			vnFlags: VNFlagMap{
				"vn.420.l1":        "http://chain-specific-l1:8545",
				"vn.420.l1.beacon": "http://chain-specific-beacon:5051",
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.vnFlags.Check()

			if tt.expectedErr {
				require.Error(t, err, "expected error but got nil")
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains, "error message should contain expected text")
				}
			} else {
				require.NoError(t, err, "unexpected error")
			}
		})
	}
}
