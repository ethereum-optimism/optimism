package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type ControlPlane struct {
	o *Orchestrator
}

func control(lifecycle stack.Lifecycle, mode stack.ControlAction) {
	switch mode {
	case stack.Start:
		lifecycle.Start()
	case stack.Stop:
		lifecycle.Stop()
	}
}

func (c *ControlPlane) SupervisorState(id stack.SupervisorID, mode stack.ControlAction) {
	s, ok := c.o.supervisors.Get(id)
	c.o.P().Require().True(ok, "need supervisor to change state")
	control(s, mode)
}

func (c *ControlPlane) L2CLNodeState(id stack.L2CLNodeID, mode stack.ControlAction) {
	s, ok := c.o.l2CLs.Get(id)
	c.o.P().Require().True(ok, "need l2cl node to change state")
	control(s, mode)
}

func (c *ControlPlane) FakePoSState(id stack.L1CLNodeID, mode stack.ControlAction) {
	s, ok := c.o.l1CLs.Get(id)
	c.o.P().Require().True(ok, "need l1cl node to change state of fakePoS module")

	control(s.fakepos, mode)
}

var _ stack.ControlPlane = (*ControlPlane)(nil)
