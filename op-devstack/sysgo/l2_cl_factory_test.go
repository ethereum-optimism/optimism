package sysgo

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

type fakeExternalL2CL struct {
	starts int
	stops  int
}

func (n *fakeExternalL2CL) Start() { n.starts++ }
func (n *fakeExternalL2CL) Stop()  { n.stops++ }
func (n *fakeExternalL2CL) UserRPC() string {
	return "http://127.0.0.1:9545"
}
func (n *fakeExternalL2CL) InteropRPC() (string, eth.Bytes32) { return "", eth.Bytes32{} }

func TestL2CLFactorySupportsPerSlotSelectionAndRestart(t *testing.T) {
	dt := devtest.SerialT(t)
	var seen []L2CLLaunchContext
	nodes := make(map[string]*fakeExternalL2CL)
	factory := L2CLFactoryFn(func(_ devtest.T, ctx L2CLLaunchContext) (L2CLNode, bool) {
		seen = append(seen, ctx)
		if ctx.Target.Name == "stock" {
			return nil, false
		}
		node := &fakeExternalL2CL{}
		nodes[ctx.Target.Name] = node
		return node, true
	})

	sequencer, handled := factory.CreateL2CL(dt, L2CLLaunchContext{
		Target: NewComponentTarget("sequencer", eth.ChainIDFromUInt64(901)),
		Role:   L2CLRoleSequencer,
	})
	require.True(t, handled)
	sequencer.Start()
	sequencer.Stop()
	sequencer.Start()

	verifier, handled := factory.CreateL2CL(dt, L2CLLaunchContext{
		Target: NewComponentTarget("late-verifier", eth.ChainIDFromUInt64(901)),
		Role:   L2CLRoleVerifier,
	})
	require.True(t, handled)
	verifier.Start()

	_, handled = factory.CreateL2CL(dt, L2CLLaunchContext{
		Target: NewComponentTarget("stock", eth.ChainIDFromUInt64(901)),
		Role:   L2CLRoleVerifier,
	})
	require.False(t, handled)

	require.Equal(t, []string{"sequencer", "late-verifier", "stock"}, []string{
		seen[0].Target.Name,
		seen[1].Target.Name,
		seen[2].Target.Name,
	})
	require.Equal(t, 2, nodes["sequencer"].starts)
	require.Equal(t, 1, nodes["sequencer"].stops)
	require.Equal(t, 1, nodes["late-verifier"].starts)
}
