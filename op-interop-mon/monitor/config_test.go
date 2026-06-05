package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigCheckRequiresDependencySet(t *testing.T) {
	c := &CLIConfig{L2Rpcs: []string{"http://localhost:8545"}}
	require.ErrorContains(t, c.Check(), "dependency-set is required")
}
