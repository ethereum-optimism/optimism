package presets

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/stretchr/testify/require"
)

func TestZKProposerMetricsURL(t *testing.T) {
	const metricsAddr = "127.0.0.1:1234"
	sys := &SingleChainInterop{
		T:             devtest.SerialT(t),
		zkMetricsAddr: func() string { return metricsAddr },
	}

	require.Equal(t, "http://"+metricsAddr+"/metrics", sys.ZKProposerMetricsURL())
}
