package driver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestConfigCheckL1BlockTag(t *testing.T) {
	for _, label := range []eth.BlockLabel{"", eth.Unsafe, eth.Safe, eth.Finalized} {
		require.NoError(t, (&Config{VerifierL1BlockTag: label}).Check(), "label %q", label)
	}
	for _, label := range []eth.BlockLabel{"", eth.Unsafe, eth.Safe} {
		require.NoError(t, (&Config{SequencerL1BlockTag: label}).Check(), "label %q", label)
	}
	require.ErrorContains(t, (&Config{SequencerL1BlockTag: eth.Finalized}).Check(), "max sequencer drift")
	require.ErrorContains(t, (&Config{VerifierL1BlockTag: "pending"}).Check(), "invalid verifier L1 block tag")
	require.ErrorContains(t, (&Config{SequencerL1BlockTag: "Safe"}).Check(), "invalid sequencer L1 block tag")
}

func TestConfigL1Labels(t *testing.T) {
	require.Equal(t, eth.BlockLabel(eth.Unsafe), (&Config{}).VerifierL1Label())
	require.Equal(t, eth.BlockLabel(eth.Unsafe), (&Config{}).SequencerL1Label())
	cfg := &Config{VerifierL1BlockTag: eth.Finalized, SequencerL1BlockTag: eth.Safe}
	require.Equal(t, eth.BlockLabel(eth.Finalized), cfg.VerifierL1Label())
	require.Equal(t, eth.BlockLabel(eth.Safe), cfg.SequencerL1Label())
}
