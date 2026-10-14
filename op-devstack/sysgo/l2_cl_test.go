package sysgo

import (
	"testing"

	"github.com/stretchr/testify/require"

	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestL2CLVerifierSyncMode(t *testing.T) {
	cfg := DefaultL2CLConfig()
	L2CLVerifierSyncMode(nodeSync.ELSync).Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, nodeSync.ELSync, cfg.VerifierSyncMode)
	require.Equal(t, nodeSync.CLSync, cfg.SequencerSyncMode, "the sequencer keeps its own sync mode")
}
