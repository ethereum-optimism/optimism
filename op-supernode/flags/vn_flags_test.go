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
