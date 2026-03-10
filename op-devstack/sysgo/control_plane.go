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

func (c *ControlPlane) SupervisorState(id stack.ComponentID, mode stack.ControlAction) {
	component, ok := c.o.GetSupervisor(id)
	c.o.P().Require().True(ok, "need supervisor to change state")
	control(component, mode)
}

func (c *ControlPlane) L2CLNodeState(id stack.ComponentID, mode stack.ControlAction) {
	component, ok := c.o.GetL2CL(id)
	c.o.P().Require().True(ok, "need l2cl node to change state")
	control(component, mode)
}

func (c *ControlPlane) L2ELNodeState(id stack.ComponentID, mode stack.ControlAction) {
	component, ok := c.o.GetL2EL(id)
	c.o.P().Require().True(ok, "need l2el node to change state")
	control(component, mode)
}

func (c *ControlPlane) FakePoSState(id stack.ComponentID, mode stack.ControlAction) {
	component, ok := c.o.GetL1CL(id)
	c.o.P().Require().True(ok, "need l1cl node to change state of fakePoS module")
	control(component.fakepos, mode)
}

func (c *ControlPlane) OPRBuilderNodeState(id stack.ComponentID, mode stack.ControlAction) {
	component, ok := c.o.GetOPRBuilder(id)
	c.o.P().Require().True(ok, "need oprbuilder node to change state")
	control(component, mode)
}

func (c *ControlPlane) RollupBoostNodeState(id stack.ComponentID, mode stack.ControlAction) {
	component, ok := c.o.GetRollupBoost(id)
	c.o.P().Require().True(ok, "need rollup boost node to change state")
	control(component, mode)
}

var _ stack.ControlPlane = (*ControlPlane)(nil)
