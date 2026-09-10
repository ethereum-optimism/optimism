package presets

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/stretchr/testify/require"
)

func TestSingleChainProjectionManagesOnlyStockCLPeerEdges(t *testing.T) {
	primary := &sysgo.SingleChainNodeRuntime{}
	follower := &sysgo.SingleChainNodeRuntime{}
	runtime := &sysgo.SingleChainRuntime{
		Nodes:      map[string]*sysgo.SingleChainNodeRuntime{"sequencer": primary},
		P2PEnabled: true,
	}

	require.True(t, shouldManageSingleChainCLPeer(runtime, follower))
	require.True(t, shouldManageSingleChainCLPeerEdge(primary, follower))
	primary.FactoryHandledCL = true
	require.False(t, shouldManageSingleChainCLPeer(runtime, follower))
	require.False(t, shouldManageSingleChainCLPeerEdge(primary, follower))
	primary.FactoryHandledCL = false
	follower.FactoryHandledCL = true
	require.False(t, shouldManageSingleChainCLPeer(runtime, follower))
	require.False(t, shouldManageSingleChainCLPeerEdge(primary, follower))
	follower.FactoryHandledCL = false
	runtime.P2PEnabled = false
	require.False(t, shouldManageSingleChainCLPeer(runtime, follower))
}
