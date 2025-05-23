package sysext

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/controller/surface"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type ControlPlane struct {
	o *Orchestrator
}

func (c *ControlPlane) setLifecycleState(svcID string, mode stack.ControlAction) {
	ctx := c.o.P().Ctx()
	require := c.o.P().Require()

	ctl, err := c.o.env.Control()
	require.NoError(err, "Error getting control plane")
	lc, ok := ctl.(surface.ServiceLifecycleSurface)
	require.True(ok, "Control plane does not support service lifecycle management")

	switch mode {
	case stack.Start:
		require.NoError(lc.StartService(ctx, svcID), "Error starting service")
	case stack.Stop:
		require.NoError(lc.StopService(ctx, svcID), "Error stopping service")
	}
}

func (c *ControlPlane) SupervisorState(id stack.SupervisorID, mode stack.ControlAction) {
	c.setLifecycleState(string(id), mode)
}

func (c *ControlPlane) L2CLNodeState(id stack.L2CLNodeID, mode stack.ControlAction) {
	c.setLifecycleState(id.Key, mode)
}

func (c *ControlPlane) FakePoSState(id stack.L1CLNodeID, mode stack.ControlAction) {
	panic("not implemented: plug in kurtosis wrapper, or gate for the test that uses this method to not run in kurtosis")
}

var _ stack.ControlPlane = (*ControlPlane)(nil)
