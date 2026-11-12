package sysgo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

func TestL2CLVerifierSyncMode(t *testing.T) {
	cfg := &L2CLConfig{SequencerSyncMode: nodeSync.CLSync, VerifierSyncMode: nodeSync.CLSync}
	L2CLVerifierSyncMode(nodeSync.ELSync).Apply(nil, ComponentTarget{}, cfg)

	require.Equal(t, nodeSync.ELSync, cfg.VerifierSyncMode)
	require.Equal(t, nodeSync.CLSync, cfg.SequencerSyncMode, "the sequencer keeps its own sync mode")
}

func TestL2CLPostFinalizationELSync(t *testing.T) {
	require.False(t, DefaultL2CLConfig().SupportsPostFinalizationELSync, "op-node skips EL sync past finality by default")
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled=%t", enabled), func(t *testing.T) {
			cfg := &L2CLConfig{SupportsPostFinalizationELSync: !enabled}
			L2CLPostFinalizationELSync(enabled).Apply(nil, ComponentTarget{}, cfg)

			require.Equal(t, enabled, cfg.SupportsPostFinalizationELSync)
		})
	}
}
