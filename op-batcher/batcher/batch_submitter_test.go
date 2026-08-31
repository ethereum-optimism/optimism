package batcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainRejectsRemovedAltDAEnvVars(t *testing.T) {
	t.Setenv("OP_BATCHER_ALTDA_ENABLED", "false")

	_, err := Main("")(nil, nil)
	require.ErrorContains(t, err, "OP_BATCHER_ALTDA_ENABLED")
}
