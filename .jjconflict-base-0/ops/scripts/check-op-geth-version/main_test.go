package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	tests := map[string]string{
		"valid":                 "",
		"valid-rc":              "",
		"invalid-version":       "invalid op-geth version",
		"invalid-geth-encoding": "invalid op-geth version",
		"wrong-replacement":     "must point to github.com/ethereum-optimism/op-geth",
		"no-replacement":        "no replace directive found",
	}
	for name, errContains := range tests {
		t.Run(name, func(t *testing.T) {
			testDir, err := filepath.Abs(filepath.Join("testdata", name))
			require.NoError(t, err)
			if err = run(testDir); errContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, errContains)
			}
		})
	}
}
