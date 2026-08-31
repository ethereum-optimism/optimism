package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRollupNodeMainRejectsRemovedAltDAEnvVars(t *testing.T) {
	t.Setenv("OP_NODE_ALTDA_ENABLED", "false")

	_, err := RollupNodeMain(nil, nil)
	require.ErrorContains(t, err, "OP_NODE_ALTDA_ENABLED")
}
