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

func (c *ControlPlane) L2ELNodeState(id stack.L2ELNodeID, mode stack.ControlAction) {
	s, ok := c.o.l2ELs.Get(id)
	c.o.P().Require().True(ok, "need l2el node to change state")
	control(s, mode)
}

func (c *ControlPlane) FakePoSState(id stack.L1CLNodeID, mode stack.ControlAction) {
	s, ok := c.o.l1CLs.Get(id)
	c.o.P().Require().True(ok, "need l1cl node to change state of fakePoS module")

	control(s.fakepos, mode)
}

func (c *ControlPlane) OPRBuilderNodeState(id stack.OPRBuilderNodeID, mode stack.ControlAction) {
	s, ok := c.o.oprbuilderNodes.Get(id)
	c.o.P().Require().True(ok, "need oprbuilder node to change state")
	control(s, mode)
}

func (c *ControlPlane) RollupBoostNodeState(id stack.RollupBoostNodeID, mode stack.ControlAction) {
	s, ok := c.o.rollupBoosts.Get(id)
	c.o.P().Require().True(ok, "need rollup boost node to change state")
	control(s, mode)
}

var _ stack.ControlPlane = (*ControlPlane)(nil)
