package presets

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/stretchr/testify/require"
)

func TestSingleChainInteropZKProposerFacades(t *testing.T) {
	handle := &sysgo.ZKProposerRuntime{}
	starts := 0
	accesses := 0
	sys := &SingleChainInterop{
		T: devtest.SerialT(t),
		startZKProposer: func() *sysgo.ZKProposerRuntime {
			starts++
			return handle
		},
		zkProposer: func() *sysgo.ZKProposerRuntime {
			accesses++
			return handle
		},
	}

	require.NotNil(t, sys.StartZKProposer())
	require.NotNil(t, sys.ZKProposer())
	require.Equal(t, 1, starts)
	require.Equal(t, 1, accesses)
}
